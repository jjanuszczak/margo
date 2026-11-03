use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine as _};
use ring::{
    rand::{SecureRandom, SystemRandom},
    signature::{RsaPublicKeyComponents, RSA_PKCS1_2048_8192_SHA256},
};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::{
    process::{Command, Stdio},
    sync::{Mutex, OnceLock},
    time::{SystemTime, UNIX_EPOCH},
};
use tokio::{
    io::{AsyncReadExt, AsyncWriteExt},
    net::TcpListener,
    time::{timeout, Duration},
};

const SERVICE: &str = "margo-desktop";
const ACCOUNT: &str = "chatgpt-oauth";
const HOST_ACCOUNT: &str = "chatgpt-host-id";
const AUTH_ENDPOINT: &str = "https://auth.openai.com/api/accounts/authorize";
const TOKEN_ENDPOINT: &str = "https://auth.openai.com/api/accounts/oauth/token";
const RESOURCE: &str = "https://api.openai.com/v1";
const CALLBACK_PATH: &str = "/auth/callback";
const SHARING_SCOPE: &str = "chatgpt.tokens.use.direct";

#[derive(Clone, Serialize, Deserialize)]
pub struct Connection {
    client_id: String,
    access_token: String,
    refresh_token: String,
    id_token: String,
    email: Option<String>,
    subject: String,
    scopes: Vec<String>,
    expires_at: u64,
}

static SESSION_CONNECTION: OnceLock<Mutex<Option<Connection>>> = OnceLock::new();
static SESSION_HOST_ID: OnceLock<String> = OnceLock::new();

fn session_slot() -> &'static Mutex<Option<Connection>> {
    SESSION_CONNECTION.get_or_init(|| Mutex::new(None))
}

fn load_session_connection() -> Result<Option<Connection>, String> {
    session_slot()
        .lock()
        .map(|connection| connection.clone())
        .map_err(|_| "Temporary ChatGPT session is unavailable".into())
}

fn save_session_connection(connection: Option<Connection>) -> Result<(), String> {
    *session_slot()
        .lock()
        .map_err(|_| "Temporary ChatGPT session is unavailable")? = connection;
    Ok(())
}

#[derive(Serialize)]
pub struct PublicConnection {
    pub connected: bool,
    pub sharing: bool,
    pub email: Option<String>,
}

#[derive(Deserialize)]
struct TokenResponse {
    access_token: String,
    id_token: Option<String>,
    refresh_token: Option<String>,
    scope: Option<String>,
    expires_in: u64,
}

#[derive(Deserialize)]
struct Jwks {
    keys: Vec<Jwk>,
}

#[derive(Deserialize)]
struct Jwk {
    kid: String,
    kty: String,
    alg: Option<String>,
    n: String,
    e: String,
}

#[derive(Deserialize)]
struct OidcConfiguration {
    issuer: String,
    jwks_uri: String,
    revocation_endpoint: Option<String>,
}

#[derive(Deserialize)]
struct Claims {
    iss: String,
    aud: Audience,
    exp: u64,
    sub: String,
    nonce: Option<String>,
    email: Option<String>,
}

#[derive(Deserialize)]
#[serde(untagged)]
enum Audience {
    One(String),
    Many(Vec<String>),
}

impl Audience {
    fn contains(&self, value: &str) -> bool {
        match self {
            Self::One(item) => item == value,
            Self::Many(items) => items.iter().any(|item| item == value),
        }
    }
}

fn keyring_entry(account: &str) -> Result<keyring::Entry, String> {
    keyring::Entry::new(SERVICE, account).map_err(|error| error.to_string())
}

fn load_connection() -> Result<Option<Connection>, String> {
    match keyring_entry(ACCOUNT)?.get_password() {
        Ok(value) => serde_json::from_str(&value)
            .map(Some)
            .map_err(|_| "Saved ChatGPT credentials could not be read. Sign in again.".into()),
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(error) => Err(error.to_string()),
    }
}

fn save_connection(connection: &Connection) -> Result<(), String> {
    keyring_entry(ACCOUNT)?
        .set_password(&serde_json::to_string(connection).map_err(|error| error.to_string())?)
        .map_err(|error| error.to_string())
}

fn host_id() -> Result<String, String> {
    let entry = keyring_entry(HOST_ACCOUNT)?;
    match entry.get_password() {
        Ok(value) if !value.trim().is_empty() => Ok(value),
        Ok(_) | Err(keyring::Error::NoEntry) => {
            let value = format!("urn:uuid:{}", uuid::Uuid::new_v4());
            entry
                .set_password(&value)
                .map_err(|error| error.to_string())?;
            Ok(value)
        }
        Err(error) => Err(error.to_string()),
    }
}

pub fn public_connection(session_only: bool) -> Result<PublicConnection, String> {
    let connection = if session_only {
        load_session_connection()?
    } else {
        load_connection()?
    };
    Ok(match connection {
        Some(connection) => PublicConnection {
            connected: true,
            sharing: connection.scopes.iter().any(|scope| scope == SHARING_SCOPE),
            email: connection.email,
        },
        None => PublicConnection {
            connected: false,
            sharing: false,
            email: None,
        },
    })
}

fn random_secret() -> Result<String, String> {
    let mut bytes = [0u8; 32];
    SystemRandom::new()
        .fill(&mut bytes)
        .map_err(|_| "Could not create secure sign-in values")?;
    Ok(URL_SAFE_NO_PAD.encode(bytes))
}

fn launch_browser(url: &str) -> Result<(), String> {
    #[cfg(target_os = "macos")]
    let mut command = Command::new("open");
    #[cfg(target_os = "linux")]
    let mut command = Command::new("xdg-open");
    #[cfg(not(any(target_os = "macos", target_os = "linux")))]
    return Err("ChatGPT sign-in is supported on macOS and Linux.".into());
    command
        .arg(url)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .spawn()
        .map(|_| ())
        .map_err(|error| format!("Could not open the sign-in browser: {error}"))
}

async fn receive_callback(
    listener: TcpListener,
    expected_state: String,
) -> Result<(String, Option<String>), String> {
    timeout(Duration::from_secs(240), async move {
        let (mut stream, _) = listener.accept().await.map_err(|error| error.to_string())?;
        let mut request = Vec::with_capacity(2048);
        let mut chunk = [0u8; 1024];
        while request.len() < 8192 && !request.windows(4).any(|part| part == b"\r\n\r\n") {
            let read = stream.read(&mut chunk).await.map_err(|error| error.to_string())?;
            if read == 0 { break; }
            request.extend_from_slice(&chunk[..read]);
        }
        let request = String::from_utf8_lossy(&request);
        let first_line = request.lines().next().ok_or("Invalid sign-in callback")?;
        let mut parts = first_line.split_whitespace();
        if parts.next() != Some("GET") { return Err("Invalid sign-in callback request".into()); }
        let target = parts.next().ok_or("Invalid sign-in callback request")?;
        let callback = reqwest::Url::parse(&format!("http://127.0.0.1{target}"))
            .map_err(|_| "Invalid sign-in callback URL")?;
        if callback.path() != CALLBACK_PATH { return Err("Unexpected sign-in callback path".into()); }
        let values: std::collections::HashMap<String, String> = callback.query_pairs().into_owned().collect();
        let html = "<!doctype html><title>Margo sign-in</title><p>ChatGPT sign-in returned to Margo. You can close this tab.</p>";
        let response = format!("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{}", html.len(), html);
        let _ = stream.write_all(response.as_bytes()).await;
        if values.get("state") != Some(&expected_state) { return Err("ChatGPT sign-in state did not match. Try again.".into()); }
        if let Some(error) = values.get("error") {
            return Err(if error == "access_denied" { "ChatGPT plan access was not approved.".into() } else { format!("ChatGPT sign-in failed: {error}") });
        }
        let code = values.get("code").cloned().ok_or("ChatGPT did not return an authorization code")?;
        let client_id = values.get("client_id").cloned();
        if client_id.as_deref() == Some("dynamic_agent_client") { return Err("ChatGPT returned an incomplete client registration.".into()); }
        Ok((code, client_id))
    })
    .await
    .map_err(|_| "ChatGPT sign-in timed out. Try again.".to_string())?
}

async fn oidc_configuration(client: &reqwest::Client) -> Result<OidcConfiguration, String> {
    client
        .get("https://auth.openai.com/.well-known/openid-configuration")
        .send()
        .await
        .map_err(|error| error.to_string())?
        .error_for_status()
        .map_err(|error| error.to_string())?
        .json()
        .await
        .map_err(|error| format!("Could not load ChatGPT sign-in configuration: {error}"))
}

async fn verify_id_token(
    client: &reqwest::Client,
    token: &str,
    client_id: &str,
    nonce: &str,
) -> Result<Claims, String> {
    let parts: Vec<&str> = token.split('.').collect();
    if parts.len() != 3 {
        return Err("ChatGPT returned an invalid identity token.".into());
    }
    let header: Value = serde_json::from_slice(
        &URL_SAFE_NO_PAD
            .decode(parts[0])
            .map_err(|_| "Invalid identity token")?,
    )
    .map_err(|_| "Invalid identity token")?;
    if header["alg"].as_str() != Some("RS256") {
        return Err("ChatGPT used an unsupported identity-token signature.".into());
    }
    let kid = header["kid"]
        .as_str()
        .ok_or("ChatGPT identity token has no signing key")?;
    let config = oidc_configuration(client).await?;
    let jwks: Jwks = client
        .get(&config.jwks_uri)
        .send()
        .await
        .map_err(|error| error.to_string())?
        .error_for_status()
        .map_err(|error| error.to_string())?
        .json()
        .await
        .map_err(|error| error.to_string())?;
    let jwk = jwks
        .keys
        .iter()
        .find(|key| {
            key.kid == kid && key.kty == "RSA" && key.alg.as_deref().unwrap_or("RS256") == "RS256"
        })
        .ok_or("Could not find ChatGPT's identity signing key")?;
    let n = URL_SAFE_NO_PAD
        .decode(&jwk.n)
        .map_err(|_| "Invalid ChatGPT signing key")?;
    let e = URL_SAFE_NO_PAD
        .decode(&jwk.e)
        .map_err(|_| "Invalid ChatGPT signing key")?;
    let signature = URL_SAFE_NO_PAD
        .decode(parts[2])
        .map_err(|_| "Invalid identity-token signature")?;
    let signing_input = format!("{}.{}", parts[0], parts[1]);
    RsaPublicKeyComponents { n: &n, e: &e }
        .verify(
            &RSA_PKCS1_2048_8192_SHA256,
            signing_input.as_bytes(),
            &signature,
        )
        .map_err(|_| "ChatGPT identity-token signature did not validate")?;
    let claims: Claims = serde_json::from_slice(
        &URL_SAFE_NO_PAD
            .decode(parts[1])
            .map_err(|_| "Invalid identity token")?,
    )
    .map_err(|_| "Invalid identity token claims")?;
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|error| error.to_string())?
        .as_secs();
    if claims.iss != config.issuer
        || !claims.aud.contains(client_id)
        || claims.exp <= now
        || claims.nonce.as_deref() != Some(nonce)
    {
        return Err("ChatGPT identity or sign-in approval did not validate.".into());
    }
    Ok(claims)
}

pub async fn sign_in(session_only: bool) -> Result<PublicConnection, String> {
    let listener = TcpListener::bind(("127.0.0.1", 0))
        .await
        .map_err(|error| error.to_string())?;
    let port = listener
        .local_addr()
        .map_err(|error| error.to_string())?
        .port();
    let redirect_uri = format!("http://127.0.0.1:{port}{CALLBACK_PATH}");
    let verifier = random_secret()?;
    let challenge = URL_SAFE_NO_PAD.encode(Sha256::digest(verifier.as_bytes()));
    let nonce = random_secret()?;
    let state = random_secret()?;
    let existing = if session_only {
        load_session_connection()?
    } else {
        load_connection()?
    };
    let mut authorize = reqwest::Url::parse(AUTH_ENDPOINT).map_err(|error| error.to_string())?;
    {
        let mut query = authorize.query_pairs_mut();
        query.append_pair(
            "client_id",
            existing
                .as_ref()
                .map(|item| item.client_id.as_str())
                .unwrap_or("dynamic_agent_client"),
        );
        if existing.is_none() {
            query.append_pair("agent_name_hint", "Margo");
        }
        let host = if session_only {
            SESSION_HOST_ID
                .get_or_init(|| format!("urn:uuid:{}", uuid::Uuid::new_v4()))
                .clone()
        } else {
            host_id()?
        };
        query.append_pair("ext_agent_host_id", &host);
        if let Some(connection) = &existing {
            query.append_pair("id_token_hint", &connection.id_token);
        }
        query.append_pair("response_type", "code");
        query.append_pair("redirect_uri", &redirect_uri);
        query.append_pair(
            "scope",
            "openid profile email offline_access resource.invoke chatgpt.tokens.use.direct",
        );
        query.append_pair("resource", RESOURCE);
        query.append_pair("state", &state);
        query.append_pair("nonce", &nonce);
        query.append_pair("code_challenge_method", "S256");
        query.append_pair("code_challenge", &challenge);
    }
    launch_browser(authorize.as_str())?;
    let (code, callback_client_id) = receive_callback(listener, state).await?;
    let client_id = callback_client_id
        .or_else(|| existing.as_ref().map(|item| item.client_id.clone()))
        .ok_or("ChatGPT did not register Margo. Try signing in again.")?;
    if existing
        .as_ref()
        .is_some_and(|connection| connection.client_id != client_id)
    {
        return Err("ChatGPT returned a different account registration. Your saved connection was not changed.".into());
    }
    let http = reqwest::Client::new();
    let response: TokenResponse = http
        .post(TOKEN_ENDPOINT)
        .form(&[
            ("grant_type", "authorization_code"),
            ("client_id", client_id.as_str()),
            ("code", code.as_str()),
            ("code_verifier", verifier.as_str()),
            ("redirect_uri", redirect_uri.as_str()),
            ("resource", RESOURCE),
        ])
        .send()
        .await
        .map_err(|error| format!("ChatGPT token exchange failed: {error}"))?
        .error_for_status()
        .map_err(|error| format!("ChatGPT token exchange failed: {error}"))?
        .json()
        .await
        .map_err(|error| format!("ChatGPT returned an invalid token response: {error}"))?;
    let scopes: Vec<String> = response
        .scope
        .unwrap_or_default()
        .split_whitespace()
        .map(str::to_owned)
        .collect();
    let id_token = response
        .id_token
        .ok_or("ChatGPT did not return an identity token")?;
    let claims = verify_id_token(&http, &id_token, &client_id, &nonce).await?;
    if let Some(previous) = &existing {
        if previous.subject != claims.sub {
            return Err("ChatGPT account did not match the saved connection.".into());
        }
    }
    let sharing = scopes.iter().any(|scope| scope == SHARING_SCOPE);
    let refresh_token = response.refresh_token.unwrap_or_default();
    if sharing && refresh_token.is_empty() {
        return Err("ChatGPT plan access was approved, but no renewable session was returned. Try signing in again.".into());
    }
    let connection = Connection {
        client_id,
        access_token: response.access_token,
        refresh_token,
        id_token,
        email: claims.email,
        subject: claims.sub,
        scopes,
        expires_at: SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map_err(|error| error.to_string())?
            .as_secs()
            + response.expires_in,
    };
    if session_only {
        save_session_connection(Some(connection))?;
    } else {
        save_connection(&connection)?;
    }
    public_connection(session_only)
}

pub async fn access_token(session_only: bool) -> Result<String, String> {
    let mut connection = if session_only {
        load_session_connection()?
    } else {
        load_connection()?
    }
    .ok_or("Sign in with ChatGPT first.")?;
    if !connection.scopes.iter().any(|scope| scope == SHARING_SCOPE) {
        return Err("ChatGPT plan access is not enabled for this connection.".into());
    }
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map_err(|error| error.to_string())?
        .as_secs();
    if connection.expires_at > now + 60 {
        return Ok(connection.access_token);
    }
    let response: TokenResponse = reqwest::Client::new()
        .post(TOKEN_ENDPOINT)
        .form(&[
            ("grant_type", "refresh_token"),
            ("client_id", connection.client_id.as_str()),
            ("refresh_token", connection.refresh_token.as_str()),
            ("resource", RESOURCE),
        ])
        .send()
        .await
        .map_err(|error| format!("ChatGPT token refresh failed: {error}"))?
        .error_for_status()
        .map_err(|error| format!("ChatGPT token refresh failed: {error}"))?
        .json()
        .await
        .map_err(|error| format!("ChatGPT returned an invalid refresh response: {error}"))?;
    connection.access_token = response.access_token;
    if let Some(refresh_token) = response.refresh_token {
        connection.refresh_token = refresh_token;
    }
    if let Some(scope) = response.scope {
        connection.scopes = scope.split_whitespace().map(str::to_owned).collect();
    }
    connection.expires_at = now + response.expires_in;
    if session_only {
        save_session_connection(Some(connection.clone()))?;
    } else {
        save_connection(&connection)?;
    }
    Ok(connection.access_token)
}

pub async fn disconnect(session_only: bool) -> Result<(), String> {
    let connection = if session_only {
        load_session_connection()?
    } else {
        load_connection()?
    };
    if let Some(connection) = connection {
        if !connection.refresh_token.is_empty() {
            if let Ok(config) = oidc_configuration(&reqwest::Client::new()).await {
                if let Some(endpoint) = config.revocation_endpoint {
                    let _ = reqwest::Client::new()
                        .post(endpoint)
                        .form(&[
                            ("token", connection.refresh_token.as_str()),
                            ("token_type_hint", "refresh_token"),
                            ("client_id", connection.client_id.as_str()),
                        ])
                        .send()
                        .await;
                }
            }
        }
    }
    if session_only {
        return save_session_connection(None);
    }
    keyring_entry(ACCOUNT)?
        .delete_credential()
        .or_else(|error| {
            if matches!(error, keyring::Error::NoEntry) {
                Ok(())
            } else {
                Err(error)
            }
        })
        .map_err(|error| error.to_string())
}

pub async fn list_models(session_only: bool) -> Result<Vec<(String, String)>, String> {
    let token = access_token(session_only).await?;
    let response: Value = reqwest::Client::new()
        .get("https://api.openai.com/v1/models")
        .bearer_auth(token)
        .send()
        .await
        .map_err(|error| error.to_string())?
        .error_for_status()
        .map_err(|error| format!("Could not list ChatGPT models: {error}"))?
        .json()
        .await
        .map_err(|error| error.to_string())?;
    let models = response["models"]
        .as_array()
        .ok_or("ChatGPT returned an invalid model catalog")?;
    Ok(models
        .iter()
        .filter(|model| model["visibility"] == "list")
        .filter_map(|model| {
            Some((
                model["slug"].as_str()?.to_owned(),
                model["display_name"].as_str()?.to_owned(),
            ))
        })
        .collect())
}
