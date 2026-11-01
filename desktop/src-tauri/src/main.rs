#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod chatgpt_auth;

use futures_util::StreamExt;
#[cfg(target_os = "linux")]
use gtk::gdk::prelude::DisplayExtManual;
#[cfg(target_os = "linux")]
use gtk::prelude::*;
use serde::Deserialize;
use serde_json::{json, Value};
use std::{
    collections::BTreeMap,
    fs::{self, OpenOptions},
    io::{BufRead, BufReader, Write},
    path::PathBuf,
    process::{Child, ChildStdin, ChildStdout, Command, Stdio},
    sync::{Arc, Mutex},
};
use tauri::{Emitter, LogicalPosition, LogicalSize, Manager, WebviewBuilder, WebviewUrl};

#[cfg(target_os = "linux")]
static LINUX_WAYLAND: std::sync::OnceLock<bool> = std::sync::OnceLock::new();

#[cfg(target_os = "linux")]
fn linux_wayland() -> bool {
    *LINUX_WAYLAND.get().unwrap_or(&false)
}

#[cfg(target_os = "linux")]
fn prepare_linux_webview_overlay(app: &tauri::AppHandle) -> tauri::Result<()> {
    let window = app.get_window("main").ok_or(tauri::Error::WindowNotFound)?;
    let editor = app
        .get_webview("main")
        .ok_or(tauri::Error::WebviewNotFound)?;
    let callback_window = window.clone();
    editor.with_webview(move |platform| {
        let container = callback_window
            .default_vbox()
            .expect("Margo's GTK window container is unavailable");
        let editor = platform.inner();
        if let Some(parent) = editor.parent() {
            if let Ok(parent) = parent.downcast::<gtk::Container>() {
                parent.remove(&editor);
            }
        }

        let overlay = gtk::Overlay::new();
        overlay.set_hexpand(true);
        overlay.set_vexpand(true);
        overlay.add(&editor);

        let preview_layer = gtk::Fixed::new();
        preview_layer.set_halign(gtk::Align::Start);
        preview_layer.set_valign(gtk::Align::Start);
        overlay.add_overlay(&preview_layer);

        container.pack_start(&overlay, true, true, 0);
        overlay.show_all();
    })?;
    Ok(())
}

#[cfg(target_os = "linux")]
fn position_linux_preview(
    app: &tauri::AppHandle,
    position: LogicalPosition<f64>,
    size: LogicalSize<f64>,
) -> tauri::Result<()> {
    let window = app.get_window("main").ok_or(tauri::Error::WindowNotFound)?;
    let preview = app
        .get_webview("preview")
        .ok_or(tauri::Error::WebviewNotFound)?;
    preview.with_webview(move |platform| {
        let container = window
            .default_vbox()
            .expect("Margo's GTK window container is unavailable");
        let overlay = container
            .children()
            .into_iter()
            .find_map(|child| child.downcast::<gtk::Overlay>().ok())
            .expect("Margo's GTK preview overlay is unavailable");
        let preview_layer = overlay
            .children()
            .into_iter()
            .find_map(|child| child.downcast::<gtk::Fixed>().ok())
            .expect("Margo's GTK preview layer is unavailable");
        let preview = platform.inner();

        let already_positioned = preview
            .parent()
            .and_then(|parent| parent.downcast::<gtk::Fixed>().ok())
            .is_some_and(|parent| parent == preview_layer);
        let x = position.x.round() as i32;
        let y = position.y.round() as i32;
        if !already_positioned {
            if let Some(parent) = preview.parent() {
                if let Ok(parent) = parent.downcast::<gtk::Container>() {
                    parent.remove(&preview);
                }
            }
            preview_layer.put(&preview, 0, 0);
        }
        let width = size.width.ceil() as i32;
        let height = size.height.ceil() as i32;
        preview_layer.set_margin_start(x);
        preview_layer.set_margin_top(y);
        preview_layer.set_size_request(width, height);
        preview.set_size_request(width, height);
        preview_layer.show();
        preview.show();
    })?;
    Ok(())
}

struct Engine {
    child: Child,
    input: Option<ChildStdin>,
    output: BufReader<ChildStdout>,
    preview_origin: Option<String>,
}
impl Drop for Engine {
    fn drop(&mut self) {
        // EOF makes the Go engine close its listener and remove preview artifacts.
        // Explicit close uses graceful EOF in stop(); this is the failure fallback.
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}
impl Engine {
    fn call(&mut self, request: Value) -> Result<Value, String> {
        let input = self.input.as_mut().ok_or("Engine input closed")?;
        serde_json::to_writer(&mut *input, &request).map_err(|e| e.to_string())?;
        input.write_all(b"\n").map_err(|e| e.to_string())?;
        input.flush().map_err(|e| e.to_string())?;
        let mut line = String::new();
        self.output
            .read_line(&mut line)
            .map_err(|e| e.to_string())?;
        if line.is_empty() {
            return Err("Margo engine stopped. Reopen the app; your source files are safe.".into());
        }
        let response: Value = serde_json::from_str(&line).map_err(|e| e.to_string())?;
        if response["protocol"] != 1 {
            return Err("Incompatible bundled engine protocol".into());
        }
        if let Some(preview) = response["preview"].as_str() {
            let url: tauri::Url = preview.parse().map_err(|_| "Invalid engine preview URL")?;
            // Pin the first engine-generated origin and unguessable session prefix.
            let prefix = format!(
                "{}://{}{}/",
                url.scheme(),
                url.host_str().unwrap_or(""),
                url.port().map(|p| format!(":{p}")).unwrap_or_default()
            );
            if url.scheme() != "http" || url.host_str() != Some("127.0.0.1") {
                return Err("Invalid preview origin".into());
            }
            let token = url
                .path()
                .split('/')
                .nth(1)
                .ok_or("Missing preview token")?;
            self.preview_origin = Some(format!("{prefix}{token}/"));
        }
        Ok(response)
    }
}
#[derive(Clone)]
struct State(Arc<Mutex<Option<Engine>>>);

fn require_editor(webview: &tauri::Webview) -> Result<(), String> {
    // Linux cannot safely give remote iframes a different capability set. There
    // are no frames in the privileged editor; the preview shell is unprivileged.
    // Every custom command also checks the caller label.
    if webview.label() != "main" {
        return Err("Native commands are restricted to the editor".into());
    }
    Ok(())
}

#[tauri::command]
async fn engine_request(
    webview: tauri::Webview,
    state: tauri::State<'_, State>,
    request: Value,
) -> Result<Value, String> {
    require_editor(&webview)?;
    let state = state.0.clone();
    tauri::async_runtime::spawn_blocking(move || {
        let mut guard = state.lock().map_err(|_| "Engine lock failed")?;
        guard.as_mut().ok_or("Engine unavailable")?.call(request)
    })
    .await
    .map_err(|e| e.to_string())?
}

#[derive(Deserialize)]
struct ChatRequest {
    #[serde(default)]
    auth_mode: String,
    #[serde(default)]
    session_only: bool,
    endpoint: String,
    model: String,
    messages: Vec<Value>,
    request_id: String,
    tools_enabled: bool,
}

trait LlmProvider {
    fn stream_completion(
        &self,
        model: &str,
        messages: &[Value],
        tools: &Value,
    ) -> reqwest::RequestBuilder;
}

fn pop_sse_event(pending: &mut String) -> Option<String> {
    let lf = pending.find("\n\n").map(|index| (index, 2));
    let crlf = pending.find("\r\n\r\n").map(|index| (index, 4));
    let (index, delimiter_len) = match (lf, crlf) {
        (Some(left), Some(right)) => {
            if left.0 <= right.0 {
                left
            } else {
                right
            }
        }
        (Some(found), None) | (None, Some(found)) => found,
        (None, None) => return None,
    };
    let event = pending[..index].to_string();
    pending.drain(..index + delimiter_len);
    Some(event)
}

fn response_output_text(response: &Value) -> String {
    if let Some(text) = response["output_text"].as_str() {
        if !text.is_empty() {
            return text.to_string();
        }
    }
    response["output"]
        .as_array()
        .into_iter()
        .flatten()
        .filter(|item| item["type"] == "message")
        .flat_map(|item| item["content"].as_array().into_iter().flatten())
        .filter_map(|part| match part["type"].as_str()? {
            "output_text" => part["text"].as_str(),
            "refusal" => part["refusal"].as_str(),
            _ => None,
        })
        .collect::<Vec<_>>()
        .join("")
}

fn response_output_summary(response: &Value) -> String {
    let outputs = response["output"]
        .as_array()
        .into_iter()
        .flatten()
        .map(|item| {
            let kind = item["type"].as_str().unwrap_or("unknown");
            if kind == "function_call" {
                format!("{kind}:{}", item["name"].as_str().unwrap_or("unnamed"))
            } else if kind == "message" {
                let parts = item["content"]
                    .as_array()
                    .into_iter()
                    .flatten()
                    .filter_map(|part| part["type"].as_str())
                    .collect::<Vec<_>>()
                    .join(",");
                format!("message[{parts}]")
            } else {
                kind.to_string()
            }
        })
        .collect::<Vec<_>>()
        .join(", ");
    format!(
        "status={}, output=[{}]",
        response["status"].as_str().unwrap_or("unknown"),
        if outputs.is_empty() {
            "empty"
        } else {
            &outputs
        }
    )
}

struct OpenAICompatibleProvider {
    client: reqwest::Client,
    endpoint: String,
    key: String,
}

impl LlmProvider for OpenAICompatibleProvider {
    fn stream_completion(
        &self,
        model: &str,
        messages: &[Value],
        tools: &Value,
    ) -> reqwest::RequestBuilder {
        let mut body = json!({"model":model,"messages":messages,"stream":true});
        if !tools.is_null() {
            body["tools"] = tools.clone();
            body["tool_choice"] = json!("auto");
        }
        let mut request = self
            .client
            .post(format!("{}/chat/completions", self.endpoint))
            .json(&body);
        if !self.key.is_empty() {
            request = request.bearer_auth(&self.key);
        }
        request
    }
}

#[tauri::command]
async fn store_ai_key(webview: tauri::Webview, key: String) -> Result<(), String> {
    require_editor(&webview)?;
    let entry =
        keyring::Entry::new("margo-desktop", "openai-compatible").map_err(|e| e.to_string())?;
    if key.trim().is_empty() {
        entry.delete_credential().ok();
    } else {
        entry.set_password(key.trim()).map_err(|e| e.to_string())?;
    }
    Ok(())
}

#[tauri::command]
async fn has_ai_key(webview: tauri::Webview) -> Result<bool, String> {
    require_editor(&webview)?;
    let entry =
        keyring::Entry::new("margo-desktop", "openai-compatible").map_err(|e| e.to_string())?;
    Ok(entry.get_password().is_ok())
}

#[tauri::command]
async fn test_ai_connection(
    webview: tauri::Webview,
    endpoint: String,
    model: String,
) -> Result<Value, String> {
    require_editor(&webview)?;
    let endpoint = endpoint.trim_end_matches('/').to_string();
    let parsed = reqwest::Url::parse(&endpoint).map_err(|_| "Invalid provider endpoint")?;
    let local = matches!(
        parsed.host_str(),
        Some("localhost" | "127.0.0.1" | "::1" | "[::1]")
    );
    if !parsed.username().is_empty() || parsed.password().is_some() {
        return Err("Do not put credentials in the provider URL".into());
    }
    if parsed.query().is_some() || parsed.fragment().is_some() {
        return Err("Remove query parameters and fragments from the provider URL".into());
    }
    if parsed.scheme() != "https" && !(parsed.scheme() == "http" && local) {
        return Err("Use HTTPS, localhost, or 127.0.0.1 for the provider endpoint".into());
    }
    let entry =
        keyring::Entry::new("margo-desktop", "openai-compatible").map_err(|e| e.to_string())?;
    let key = if local {
        String::new()
    } else {
        entry.get_password().unwrap_or_default()
    };
    let mut request = reqwest::Client::new().get(format!("{endpoint}/models"));
    if !key.is_empty() {
        request = request.bearer_auth(key);
    }
    let response = request
        .send()
        .await
        .map_err(|e| format!("Could not reach provider: {e}"))?;
    if !response.status().is_success() {
        return Err(format!(
            "Provider returned {}. Check the endpoint and API key.",
            response.status()
        ));
    }
    let body: Value = response
        .json()
        .await
        .map_err(|e| format!("Provider returned an invalid model list: {e}"))?;
    let models: Vec<String> = body
        .get("data")
        .and_then(Value::as_array)
        .into_iter()
        .flatten()
        .filter_map(|item| item.get("id").and_then(Value::as_str).map(str::to_owned))
        .collect();
    let model_available = models.iter().any(|available| available == model.trim());
    Ok(json!({"models": models, "model_available": model_available}))
}

#[tauri::command]
async fn chat_stream(
    webview: tauri::Webview,
    app: tauri::AppHandle,
    state: tauri::State<'_, State>,
    request: ChatRequest,
) -> Result<(), String> {
    require_editor(&webview)?;
    if request.auth_mode == "chatgpt" {
        return chatgpt_chat_stream(app, state, request).await;
    }
    let endpoint = request.endpoint.trim_end_matches('/').to_string();
    let parsed = reqwest::Url::parse(&endpoint).map_err(|_| "Invalid provider endpoint")?;
    let local = matches!(
        parsed.host_str(),
        Some("localhost" | "127.0.0.1" | "::1" | "[::1]")
    );
    if !parsed.username().is_empty() || parsed.password().is_some() {
        return Err("Do not put credentials in the provider URL".into());
    }
    if parsed.query().is_some() || parsed.fragment().is_some() {
        return Err("Remove query parameters and fragments from the provider URL".into());
    }
    if parsed.scheme() != "https" && !(parsed.scheme() == "http" && local) {
        return Err("Use HTTPS, localhost, or 127.0.0.1 for the provider endpoint".into());
    }
    let entry =
        keyring::Entry::new("margo-desktop", "openai-compatible").map_err(|e| e.to_string())?;
    let key = if local {
        String::new()
    } else {
        entry.get_password().unwrap_or_default()
    };
    let provider = OpenAICompatibleProvider {
        client: reqwest::Client::new(),
        endpoint,
        key,
    };
    let tools = if request.tools_enabled {
        json!([
            {"type":"function","function":{"name":"read_project_file","description":"Read one text file inside the open Margo project, using its project-relative path.","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}}},
            {"type":"function","function":{"name":"list_project_directory","description":"List entries in a project directory using a project-relative path. Use an empty path for the project root.","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}}},
            {"type":"function","function":{"name":"search_project","description":"Search Markdown, config, theme, and source files in the open project.","parameters":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"],"additionalProperties":false}}},
            {"type":"function","function":{"name":"get_file_outline","description":"Return Markdown headings for one file inside the open project.","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}}},
            {"type":"function","function":{"name":"propose_file_edit","description":"Stage a replacement or new file in the project for user review. No disk write occurs until the user applies it in Margo. Use only project config, slide Markdown, or active-theme text files, and create files only inside existing folders.","parameters":{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"],"additionalProperties":false}}}
        ])
    } else {
        Value::Null
    };
    let mut messages = request.messages;
    for round in 0..5 {
        let allowed_tools = if request.tools_enabled && round < 4 {
            &tools
        } else {
            &Value::Null
        };
        let response = provider
            .stream_completion(&request.model, &messages, allowed_tools)
            .send()
            .await
            .map_err(|e| format!("Provider request failed: {e}"))?;
        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(format!(
                "Provider returned {status}: {}",
                body.chars().take(500).collect::<String>()
            ));
        }
        let mut stream = response.bytes_stream();
        let mut pending = String::new();
        let mut calls: BTreeMap<usize, Value> = BTreeMap::new();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.map_err(|e| e.to_string())?;
            pending.push_str(&String::from_utf8_lossy(&chunk));
            while let Some(event) = pop_sse_event(&mut pending) {
                for line in event.lines().filter_map(|line| line.strip_prefix("data:")) {
                    let data = line.trim();
                    if data == "[DONE]" {
                        continue;
                    }
                    if let Ok(value) = serde_json::from_str::<Value>(data) {
                        let delta = &value["choices"][0]["delta"];
                        if let Some(text) = delta["content"].as_str() {
                            let _ = app.emit(
                                "margo-chat-delta",
                                json!({"id":request.request_id,"text":text}),
                            );
                        }
                        if let Some(tool_calls) = delta["tool_calls"].as_array() {
                            for call in tool_calls {
                                let index = call["index"].as_u64().unwrap_or(0) as usize;
                                let current = calls.entry(index).or_insert_with(|| json!({"id":"","type":"function","function":{"name":"","arguments":""}}));
                                if let Some(id) = call["id"].as_str() {
                                    current["id"] = json!(id);
                                }
                                if let Some(name) = call["function"]["name"].as_str() {
                                    current["function"]["name"] = json!(format!(
                                        "{}{}",
                                        current["function"]["name"].as_str().unwrap_or(""),
                                        name
                                    ));
                                }
                                if let Some(args) = call["function"]["arguments"].as_str() {
                                    current["function"]["arguments"] = json!(format!(
                                        "{}{}",
                                        current["function"]["arguments"].as_str().unwrap_or(""),
                                        args
                                    ));
                                }
                            }
                        }
                    }
                }
            }
        }
        if calls.is_empty() {
            break;
        }
        let calls: Vec<Value> = calls.into_values().collect();
        messages.push(json!({"role":"assistant","content":null,"tool_calls":calls}));
        for call in calls {
            let name = call["function"]["name"].as_str().unwrap_or("");
            let args: Value =
                serde_json::from_str(call["function"]["arguments"].as_str().unwrap_or("{}"))
                    .unwrap_or(json!({}));
            let (op, params) = match name {
                "read_project_file" => ("ai-read", json!({"path":args["path"]})),
                "list_project_directory" => ("ai-list", json!({"path":args["path"]})),
                "search_project" => ("ai-search", json!({"query":args["query"]})),
                "get_file_outline" => ("ai-outline", json!({"path":args["path"]})),
                "propose_file_edit" => ("ai-propose-edit", json!({"path":args["path"]})),
                _ => {
                    messages.push(json!({"role":"tool","tool_call_id":call["id"],"content":"Unsupported read-only tool"}));
                    continue;
                }
            };
            let _ = app.emit(
                "margo-chat-tool",
                json!({"id":request.request_id,"name":name,"arguments":args}),
            );
            let payload = json!({"protocol":1,"op":op,"root":"","path":params["path"],"query":params["query"]});
            let engine_state = state.0.clone();
            let result = tauri::async_runtime::spawn_blocking(move || {
                let mut guard = engine_state.lock().map_err(|_| "Engine lock failed")?;
                guard.as_mut().ok_or("Engine unavailable")?.call(payload)
            })
            .await
            .map_err(|e| e.to_string())?
            .unwrap_or_else(|e| json!({"error":e}));
            let output = if name == "propose_file_edit" && result["file"].is_object() {
                if let Some(content) = args["content"].as_str() {
                    if content.len() <= 512 * 1024 && !content.contains('\0') {
                        let _ = app.emit(
                            "margo-chat-proposal",
                            json!({"id":request.request_id,"path":args["path"],"original":result["file"]["content"],"revision":result["file"]["revision"],"content":content}),
                        );
                        "Proposal staged for user review. It has not been applied to disk. Continue helping and do not claim that the file was changed.".to_string()
                    } else {
                        "Proposal rejected: content must be plain text under 512 KiB.".to_string()
                    }
                } else {
                    "Proposal rejected: content must be text.".to_string()
                }
            } else if let Some(content) = result["file"]["content"].as_str() {
                content.chars().take(20000).collect::<String>()
            } else {
                result.to_string()
            };
            let _ = app.emit(
                "margo-chat-tool-result",
                json!({"id":request.request_id,"name":name,"preview":output.chars().take(4000).collect::<String>()}),
            );
            messages.push(json!({"role":"tool","tool_call_id":call["id"],"content":output}));
        }
    }
    let _ = app.emit("margo-chat-done", &request.request_id);
    Ok(())
}

#[tauri::command]
async fn connect_chatgpt(
    webview: tauri::Webview,
    session_only: bool,
) -> Result<chatgpt_auth::PublicConnection, String> {
    require_editor(&webview)?;
    chatgpt_auth::sign_in(session_only).await
}

#[tauri::command]
async fn chatgpt_connection(
    webview: tauri::Webview,
    session_only: bool,
) -> Result<chatgpt_auth::PublicConnection, String> {
    require_editor(&webview)?;
    chatgpt_auth::public_connection(session_only)
}

#[tauri::command]
async fn disconnect_chatgpt(webview: tauri::Webview, session_only: bool) -> Result<(), String> {
    require_editor(&webview)?;
    chatgpt_auth::disconnect(session_only).await
}

#[tauri::command]
async fn chatgpt_models(webview: tauri::Webview, session_only: bool) -> Result<Vec<Value>, String> {
    require_editor(&webview)?;
    Ok(chatgpt_auth::list_models(session_only)
        .await?
        .into_iter()
        .map(|(slug, display_name)| json!({"id":slug,"name":display_name}))
        .collect())
}

async fn chatgpt_chat_stream(
    app: tauri::AppHandle,
    state: tauri::State<'_, State>,
    request: ChatRequest,
) -> Result<(), String> {
    let token = chatgpt_auth::access_token(request.session_only).await?;
    let http = reqwest::Client::new();
    let mut instructions = String::new();
    let mut input = Vec::new();
    for message in &request.messages {
        let role = message["role"].as_str().unwrap_or("");
        if role == "system" || role == "developer" {
            if let Some(content) = message["content"].as_str() {
                if !instructions.is_empty() {
                    instructions.push_str("\n\n");
                }
                instructions.push_str(content);
            }
        } else if matches!(role, "user" | "assistant") {
            input.push(message.clone());
        }
    }
    let tools = if request.tools_enabled {
        json!([
            {"type":"function","name":"read_project_file","description":"Read one text file inside the open Margo project, using its project-relative path.","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}},
            {"type":"function","name":"list_project_directory","description":"List entries in a project directory using a project-relative path. Use an empty path for the project root.","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}},
            {"type":"function","name":"search_project","description":"Search Markdown, config, theme, and source files in the open project.","parameters":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"],"additionalProperties":false}},
            {"type":"function","name":"get_file_outline","description":"Return Markdown headings for one file inside the open project.","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"],"additionalProperties":false}},
            {"type":"function","name":"propose_file_edit","description":"Stage a replacement or new file in the project for user review. No disk write occurs until the user applies it in Margo. Use only project config, slide Markdown, or active-theme text files, and create files only inside existing folders.","parameters":{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"],"additionalProperties":false}}
        ])
    } else {
        Value::Null
    };

    let mut empty_response_retried = false;
    for round in 0..5 {
        let mut body = json!({"model":request.model,"instructions":instructions,"input":input,"store":false,"stream":true});
        if !tools.is_null() && round < 4 {
            body["tools"] = tools.clone();
        }
        let response = http
            .post("https://api.openai.com/v1/responses")
            .bearer_auth(&token)
            .json(&body)
            .send()
            .await
            .map_err(|error| format!("ChatGPT request failed: {error}"))?;
        if !response.status().is_success() {
            let status = response.status();
            let detail = response.text().await.unwrap_or_default();
            return Err(format!(
                "ChatGPT returned {status}: {}",
                detail.chars().take(500).collect::<String>()
            ));
        }
        let mut stream = response.bytes_stream();
        let mut pending = String::new();
        let mut completed: Option<Value> = None;
        let mut failure: Option<String> = None;
        let mut streamed_text = false;
        let mut streamed_items: BTreeMap<usize, Value> = BTreeMap::new();
        while let Some(chunk) = stream.next().await {
            let chunk = chunk.map_err(|error| error.to_string())?;
            pending.push_str(&String::from_utf8_lossy(&chunk));
            while let Some(event) = pop_sse_event(&mut pending) {
                let (event_name, data) =
                    event
                        .lines()
                        .fold((String::new(), String::new()), |mut acc, line| {
                            if let Some(value) = line.strip_prefix("event:") {
                                acc.0 = value.trim().to_string();
                            }
                            if let Some(value) = line.strip_prefix("data:") {
                                acc.1 = value.trim().to_string();
                            }
                            acc
                        });
                if data.is_empty() {
                    continue;
                }
                let Ok(value) = serde_json::from_str::<Value>(&data) else {
                    continue;
                };
                match event_name.as_str() {
                    "response.output_item.added" | "response.output_item.done" => {
                        if let (Some(index), Some(item)) =
                            (value["output_index"].as_u64(), value["item"].as_object())
                        {
                            streamed_items.insert(index as usize, Value::Object(item.clone()));
                        }
                    }
                    "response.function_call_arguments.done" => {
                        if let Some(index) = value["output_index"].as_u64() {
                            let item = streamed_items.entry(index as usize).or_insert_with(|| {
                                json!({"type":"function_call","call_id":value["call_id"],"item_id":value["item_id"],"name":"","arguments":""})
                            });
                            item["arguments"] = value["arguments"].clone();
                        }
                    }
                    "response.output_text.delta" | "response.refusal.delta" => {
                        if let Some(text) = value["delta"].as_str() {
                            streamed_text = true;
                            let _ = app.emit(
                                "margo-chat-delta",
                                json!({"id":request.request_id,"text":text}),
                            );
                        }
                    }
                    "response.completed" => completed = Some(value["response"].clone()),
                    "response.failed" => {
                        failure = Some(
                            value["response"]["error"]["message"]
                                .as_str()
                                .unwrap_or("ChatGPT request failed")
                                .to_string(),
                        )
                    }
                    _ => {}
                }
            }
        }
        if let Some(failure) = failure {
            return Err(failure);
        }
        let mut response =
            completed.ok_or("ChatGPT stream ended before a completed response arrived")?;
        if response["output"]
            .as_array()
            .map_or(true, |items| items.is_empty())
            && !streamed_items.is_empty()
        {
            response["output"] = json!(streamed_items.into_values().collect::<Vec<_>>());
        }
        let output = response["output"].as_array().cloned().unwrap_or_default();
        let calls: Vec<Value> = output
            .iter()
            .filter(|item| item["type"] == "function_call")
            .cloned()
            .collect();
        if calls.is_empty() {
            if !streamed_text {
                let text = response_output_text(&response);
                if !text.is_empty() {
                    let _ = app.emit(
                        "margo-chat-delta",
                        json!({"id":request.request_id,"text":text}),
                    );
                } else {
                    let detail = response_output_summary(&response);
                    if !empty_response_retried && round < 4 {
                        empty_response_retried = true;
                        input.push(json!({"role":"user","content":"Your previous turn completed without returning an answer or tool call. Please answer the original request now with a concise visible text reply. Use project tools only if you need to inspect project files."}));
                        continue;
                    }
                    let _ = app.emit("margo-chat-delta", json!({"id":request.request_id,"text":format!("ChatGPT completed without returning an answer or project tool call ({detail}). Please try again. If this repeats, turn off Project tools and retry.")}));
                }
            }
            break;
        }
        input.extend(output);
        for call in calls {
            let name = call["name"].as_str().unwrap_or("");
            let args: Value = serde_json::from_str(call["arguments"].as_str().unwrap_or("{}"))
                .unwrap_or(json!({}));
            let (op, params) = match name {
                "read_project_file" => ("ai-read", json!({"path":args["path"]})),
                "list_project_directory" => ("ai-list", json!({"path":args["path"]})),
                "search_project" => ("ai-search", json!({"query":args["query"]})),
                "get_file_outline" => ("ai-outline", json!({"path":args["path"]})),
                "propose_file_edit" => ("ai-propose-edit", json!({"path":args["path"]})),
                _ => {
                    input.push(json!({"type":"function_call_output","call_id":call["call_id"],"output":"Unsupported read-only tool"}));
                    continue;
                }
            };
            let _ = app.emit(
                "margo-chat-tool",
                json!({"id":request.request_id,"name":name,"arguments":args}),
            );
            let payload = json!({"protocol":1,"op":op,"root":"","path":params["path"],"query":params["query"]});
            let engine_state = state.0.clone();
            let result = tauri::async_runtime::spawn_blocking(move || {
                let mut guard = engine_state.lock().map_err(|_| "Engine lock failed")?;
                guard.as_mut().ok_or("Engine unavailable")?.call(payload)
            })
            .await
            .map_err(|error| error.to_string())?
            .unwrap_or_else(|error| json!({"error":error}));
            let result_text = if name == "propose_file_edit" && result["file"].is_object() {
                if let Some(content) = args["content"].as_str() {
                    if content.len() <= 512 * 1024 && !content.contains('\0') {
                        let _ = app.emit(
                            "margo-chat-proposal",
                            json!({"id":request.request_id,"path":args["path"],"original":result["file"]["content"],"revision":result["file"]["revision"],"content":content}),
                        );
                        "Proposal staged for user review. It has not been applied to disk. Continue helping and do not claim that the file was changed.".to_string()
                    } else {
                        "Proposal rejected: content must be plain text under 512 KiB.".to_string()
                    }
                } else {
                    "Proposal rejected: content must be text.".to_string()
                }
            } else if let Some(content) = result["file"]["content"].as_str() {
                content.chars().take(20000).collect::<String>()
            } else {
                result.to_string()
            };
            let _ = app.emit("margo-chat-tool-result", json!({"id":request.request_id,"name":name,"preview":result_text.chars().take(4000).collect::<String>()}));
            input.push(json!({"type":"function_call_output","call_id":call["call_id"],"output":result_text}));
        }
    }
    let _ = app.emit("margo-chat-done", &request.request_id);
    Ok(())
}

#[tauri::command]
async fn choose_path(webview: tauri::Webview, kind: String) -> Result<Option<String>, String> {
    require_editor(&webview)?;
    if !matches!(
        kind.as_str(),
        "project"
            | "project-archive"
            | "theme-source-archive"
            | "project-config"
            | "theme-archive"
            | "theme-project"
            | "project-parent"
            | "media"
    ) {
        return Err("Unknown picker".into());
    }
    let result = tauri::async_runtime::spawn_blocking(move || match kind.as_str() {
        "project" => rfd::FileDialog::new()
            .set_title("Open Margo project folder")
            .pick_folder(),
        "project-archive" | "theme-source-archive" => rfd::FileDialog::new()
            .set_title(if kind == "project-archive" {
                "Open Margo project archive"
            } else {
                "Choose a Margo project archive"
            })
            .add_filter("Margo project archive", &["margo"])
            .pick_file(),
        "project-config" => rfd::FileDialog::new()
            .set_title("Open Margo project configuration")
            .add_filter("Margo project configuration", &["yaml"])
            .pick_file(),
        "theme-archive" => rfd::FileDialog::new()
            .set_title("Import Margo theme archive")
            .add_filter("Margo theme archive", &["margot"])
            .pick_file(),
        "theme-project" => rfd::FileDialog::new()
            .set_title("Choose a Margo project folder")
            .pick_folder(),
        "project-parent" => rfd::FileDialog::new()
            .set_title("Choose where to create the Margo project")
            .pick_folder(),
        "media" => rfd::FileDialog::new()
            .add_filter(
                "Media",
                &["png", "jpg", "jpeg", "gif", "svg", "webp", "mp4", "webm"],
            )
            .pick_file(),
        _ => None,
    })
    .await
    .map_err(|error| error.to_string())?;
    Ok(result.map(|path| path.to_string_lossy().into_owned()))
}

#[derive(Deserialize)]
struct PreviewRect {
    x: f64,
    y: f64,
    width: f64,
    height: f64,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct PreviewRectSize {
    width: f64,
    height: f64,
}

#[tauri::command]
async fn preview_layout(
    app: tauri::AppHandle,
    webview: tauri::Webview,
    rect: PreviewRect,
    viewport: PreviewRectSize,
    visible: bool,
    url: Option<String>,
    state: tauri::State<'_, State>,
) -> Result<(), String> {
    require_editor(&webview)?;
    let values = [rect.x, rect.y, rect.width, rect.height];
    if values.iter().any(|v| !v.is_finite() || *v < 0.0) {
        return Err("Invalid preview bounds".into());
    }
    if !viewport.width.is_finite()
        || !viewport.height.is_finite()
        || viewport.width <= 0.0
        || viewport.height <= 0.0
    {
        return Err("Invalid editor viewport".into());
    }
    // DOM coordinates are relative to the editor webview, not the native window.
    // Native bounds also account for macOS titlebar offsets and UI zoom/scaling.
    let scale = webview.window().scale_factor().map_err(|e| e.to_string())?;
    let editor_bounds = webview.bounds().map_err(|e| e.to_string())?;
    let origin = editor_bounds.position.to_logical::<f64>(scale);
    let editor_size = editor_bounds.size.to_logical::<f64>(scale);
    let ratio_x = editor_size.width / viewport.width;
    // WKWebView's bounds include the titlebar inset while DOM coordinates start
    // below it. CSS zoom is uniform, so use the width-derived ratio on both axes.
    let ratio_y = ratio_x;
    let titlebar_inset = if cfg!(target_os = "macos") {
        // WKWebView reports its full native frame (including the titlebar) but
        // innerHeight excludes the titlebar. Position APIs can report no inset
        // for a full-size content view, so derive it from these measured sizes.
        (editor_size.height - viewport.height * ratio_x).clamp(0.0, 100.0)
    } else {
        0.0
    };
    let position = LogicalPosition::new(
        origin.x + rect.x * ratio_x,
        origin.y + titlebar_inset + rect.y * ratio_y,
    );
    let size = LogicalSize::new(
        (rect.width * ratio_x).max(1.0),
        (rect.height * ratio_y).max(1.0),
    );
    let zoom = 1.0; // The unprivileged preview shell scales a fixed slide viewport.
    if visible {
        if let Ok(directory) = app.path().app_log_dir() {
            if let Ok(mut file) = OpenOptions::new()
                .create(true)
                .append(true)
                .open(directory.join("preview-layout.log"))
            {
                let _ = writeln!(
                    file,
                    "{}",
                    json!({"dom":{"x":rect.x,"y":rect.y,"width":rect.width,"height":rect.height},"viewport":{"width":viewport.width,"height":viewport.height},"native_origin":{"x":origin.x,"y":origin.y},"native_size":{"width":editor_size.width,"height":editor_size.height},"titlebar":titlebar_inset,"target":{"x":position.x,"y":position.y,"width":size.width,"height":size.height},"zoom":zoom})
                );
            }
        }
    }

    let parsed = if let Some(ref raw) = url {
        let guard = state.0.lock().map_err(|_| "Engine lock failed")?;
        let prefix = guard
            .as_ref()
            .and_then(|e| e.preview_origin.as_ref())
            .ok_or("No preview session")?;
        if !raw.starts_with(prefix) {
            return Err("Preview URL outside the current engine session".into());
        }
        Some(raw.parse::<tauri::Url>().map_err(|e| e.to_string())?)
    } else {
        None
    };
    if let Some(preview) = app.get_webview("preview") {
        if visible {
            #[cfg(not(target_os = "linux"))]
            preview
                .set_bounds(tauri::Rect {
                    position: position.into(),
                    size: size.into(),
                })
                .map_err(|e| e.to_string())?;
            #[cfg(target_os = "linux")]
            if !linux_wayland() {
                preview
                    .set_bounds(tauri::Rect {
                        position: position.into(),
                        size: size.into(),
                    })
                    .map_err(|e| e.to_string())?;
            }
            preview.set_zoom(zoom).map_err(|e| e.to_string())?;
            if let Some(url) = parsed {
                preview.navigate(url).map_err(|e| e.to_string())?;
            }
            preview.show().map_err(|e| e.to_string())?;
        } else {
            preview.hide().map_err(|e| e.to_string())?;
        }
    } else if visible {
        let url = parsed.ok_or("Preview not rendered yet")?;
        let prefix = state
            .0
            .lock()
            .map_err(|_| "Engine lock failed")?
            .as_ref()
            .and_then(|e| e.preview_origin.clone())
            .ok_or("Engine unavailable")?;
        let preview = app
            .get_window("main")
            .ok_or("Main window missing")?
            .add_child(
                WebviewBuilder::new("preview", WebviewUrl::External(url))
                    .on_navigation(move |url| url.as_str().starts_with(&prefix))
                    .on_new_window(|_, _| tauri::webview::NewWindowResponse::Deny),
                position,
                size,
            )
            .map_err(|e| e.to_string())?;
        preview.set_zoom(zoom).map_err(|e| e.to_string())?;
    }
    #[cfg(target_os = "linux")]
    if visible && linux_wayland() {
        position_linux_preview(&app, position, size).map_err(|e| e.to_string())?;
    }
    Ok(())
}

fn load_recent_projects(app: &tauri::AppHandle) -> Result<Vec<String>, String> {
    let directory = app.path().app_data_dir().map_err(|e| e.to_string())?;
    let file = directory.join("recent-projects.json");
    Ok(fs::read(&file)
        .ok()
        .and_then(|b| serde_json::from_slice(&b).ok())
        .unwrap_or_default())
}

#[tauri::command]
fn recent_projects(
    webview: tauri::Webview,
    app: tauri::AppHandle,
    root: Option<String>,
) -> Result<Vec<String>, String> {
    require_editor(&webview)?;
    let directory = app.path().app_data_dir().map_err(|e| e.to_string())?;
    fs::create_dir_all(&directory).map_err(|e| e.to_string())?;
    let file = directory.join("recent-projects.json");
    let mut recent = load_recent_projects(&app)?;
    if let Some(root) = root {
        recent.retain(|p| p != &root);
        recent.insert(0, root);
        recent.truncate(10);
        fs::write(
            &file,
            serde_json::to_vec(&recent).map_err(|e| e.to_string())?,
        )
        .map_err(|e| e.to_string())?;
    }
    app.set_menu(editor_menu(&app, &recent).map_err(|e| e.to_string())?)
        .map_err(|e| e.to_string())?;
    Ok(recent)
}

#[tauri::command]
fn omarchy_palette(webview: tauri::Webview) -> Result<Value, String> {
    require_editor(&webview)?;
    let Some(home) = std::env::var_os("HOME") else {
        return Ok(json!({"available": false}));
    };
    let file = PathBuf::from(home).join(".local/state/omarchy/current/theme/colors.toml");
    let Ok(contents) = fs::read_to_string(file) else {
        return Ok(json!({"available": false}));
    };

    let mut mode = None;
    let mut colors = serde_json::Map::new();
    for line in contents.lines() {
        let Some((key, raw_value)) = line.split_once('=') else {
            continue;
        };
        let key = key.trim();
        let value = raw_value
            .trim()
            .split(" #")
            .next()
            .unwrap_or("")
            .trim()
            .trim_matches('"');
        if key == "mode" {
            if value == "light" || value == "dark" {
                mode = Some(value);
            }
        } else if matches!(
            key,
            "accent"
                | "selection"
                | "muted"
                | "background"
                | "dark_background"
                | "darker_background"
                | "lighter_background"
                | "foreground"
                | "red"
                | "yellow"
        ) && value.len() == 7
            && value.starts_with('#')
            && value[1..].bytes().all(|byte| byte.is_ascii_hexdigit())
        {
            colors.insert(key.to_string(), json!(value));
        }
    }

    if let Some(mode) = mode {
        Ok(json!({"available": true, "mode": mode, "colors": colors}))
    } else {
        Ok(json!({"available": false}))
    }
}

#[tauri::command]
fn close_app(webview: tauri::Webview, app: tauri::AppHandle) -> Result<(), String> {
    require_editor(&webview)?;
    app.exit(0);
    Ok(())
}

fn engine_path(app: &tauri::AppHandle) -> Result<PathBuf, String> {
    let executable = std::env::current_exe().map_err(|e| e.to_string())?;
    let bundled = executable
        .parent()
        .ok_or("Missing executable directory")?
        .join("margo-engine");
    if bundled.is_file() {
        return Ok(bundled);
    }
    if cfg!(debug_assertions) {
        let target = if cfg!(target_os = "macos") {
            if cfg!(target_arch = "aarch64") {
                "aarch64-apple-darwin"
            } else {
                "x86_64-apple-darwin"
            }
        } else if cfg!(target_arch = "aarch64") {
            "aarch64-unknown-linux-gnu"
        } else {
            "x86_64-unknown-linux-gnu"
        };
        let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("binaries")
            .join(format!("margo-engine-{target}"));
        if path.is_file() {
            return Ok(path);
        }
    }
    let resource = app
        .path()
        .resource_dir()
        .map_err(|e| e.to_string())?
        .join("margo-engine");
    if resource.is_file() {
        return Ok(resource);
    }
    Err("Bundled Margo engine missing. Run npm run engine before starting the app.".into())
}

// WKWebView's native Undo menu does not use Monaco's model history.
// Keep the platform menu, routing history actions to the focused editor instead.
fn editor_menu(
    app: &tauri::AppHandle,
    recent: &[String],
) -> tauri::Result<tauri::menu::Menu<tauri::Wry>> {
    use tauri::menu::{Menu, MenuItem, MenuItemKind, PredefinedMenuItem, Submenu};
    let menu = Menu::default(app)?;
    let new_project = MenuItem::with_id(
        app,
        "margo-new-project",
        "New Project…",
        true,
        Some("CmdOrCtrl+N"),
    )?;
    let open_project = MenuItem::with_id(
        app,
        "margo-open-project",
        "Open Project…",
        true,
        Some("CmdOrCtrl+O"),
    )?;
    let open_archive = MenuItem::with_id(
        app,
        "margo-open-project-archive",
        "Open Project Archive…",
        true,
        None::<&str>,
    )?;
    let open_config = MenuItem::with_id(
        app,
        "margo-open-project-config",
        "Open margo.yaml…",
        true,
        None::<&str>,
    )?;
    let themes = MenuItem::with_id(app, "margo-themes", "Themes…", true, None::<&str>)?;
    let html = MenuItem::with_id(app, "margo-export-html", "HTML", true, None::<&str>)?;
    let pdf = MenuItem::with_id(app, "margo-export-pdf", "PDF", true, None::<&str>)?;
    let pptx = MenuItem::with_id(app, "margo-export-pptx", "PPTX", true, None::<&str>)?;
    let export = Submenu::with_items(app, "Export", true, &[&html, &pdf, &pptx])?;

    let projects = Submenu::new(app, "Projects", true)?;
    projects.append(&new_project)?;
    projects.append(&open_project)?;
    projects.append(&open_archive)?;
    projects.append(&open_config)?;
    projects.append(&themes)?;
    projects.append(&PredefinedMenuItem::separator(app)?)?;
    projects.append(&MenuItem::with_id(
        app,
        "margo-recent-heading",
        "Recent Projects",
        false,
        None::<&str>,
    )?)?;
    if recent.is_empty() {
        projects.append(&MenuItem::with_id(
            app,
            "margo-no-recent-projects",
            "No Recent Projects",
            false,
            None::<&str>,
        )?)?;
    } else {
        for (index, path) in recent.iter().enumerate() {
            let label = format!("{}  —  {}", index + 1, path);
            projects.append(&MenuItem::with_id(
                app,
                format!("margo-recent-{index}"),
                label,
                true,
                None::<&str>,
            )?)?;
        }
    }
    projects.append(&PredefinedMenuItem::separator(app)?)?;
    projects.append(&export)?;

    let mut project_position = 0;
    for (index, item) in menu.items()?.into_iter().enumerate() {
        if let MenuItemKind::Submenu(edit) = item {
            if edit.text()? == "Edit" {
                project_position = index;
                edit.remove_at(0)?;
                edit.remove_at(0)?;
                edit.insert(
                    &MenuItem::with_id(app, "margo-undo", "Undo", true, Some("CmdOrCtrl+Z"))?,
                    0,
                )?;
                edit.insert(
                    &MenuItem::with_id(app, "margo-redo", "Redo", true, Some("CmdOrCtrl+Shift+Z"))?,
                    1,
                )?;
                break;
            }
        }
    }
    menu.insert(&projects, project_position)?;
    Ok(menu)
}

fn main() {
    let state = State(Arc::new(Mutex::new(None)));
    let exit_state = state.clone();
    let app = tauri::Builder::default()
        .manage(state)
        .setup(|app| {
            #[cfg(target_os = "linux")]
            {
                let wayland = gtk::gdk::Display::default()
                    .is_some_and(|display| display.backend().is_wayland());
                let _ = LINUX_WAYLAND.set(wayland);
                if wayland {
                    prepare_linux_webview_overlay(app.handle())?;
                }
            }
            let recent = load_recent_projects(app.handle()).map_err(std::io::Error::other)?;
            app.set_menu(editor_menu(app.handle(), &recent)?)?;
            let path = engine_path(app.handle()).map_err(std::io::Error::other)?;
            let directory = app.path().app_log_dir()?;
            fs::create_dir_all(&directory)?;
            let log = OpenOptions::new()
                .create(true)
                .append(true)
                .open(directory.join("engine.log"))?;
            let mut child = Command::new(path)
                .arg("desktop-engine")
                .stdin(Stdio::piped())
                .stdout(Stdio::piped())
                .stderr(log)
                .spawn()?;
            let input = child.stdin.take().ok_or("Missing engine stdin")?;
            let output = BufReader::new(child.stdout.take().ok_or("Missing engine stdout")?);
            let mut engine = Engine {
                child,
                input: Some(input),
                output,
                preview_origin: None,
            };
            engine
                .call(json!({"protocol":1,"op":"hello"}))
                .map_err(std::io::Error::other)?;
            *app.state::<State>().0.lock().unwrap() = Some(engine);
            Ok(())
        })
        .on_menu_event(|app, event| {
            let id = event.id().as_ref();
            let action = match id {
                "margo-undo" | "margo-redo" => {
                    let history_action = if id == "margo-undo" { "undo" } else { "redo" };
                    let _ = app.emit_to("main", "margo-editor-history", history_action);
                    return;
                }
                "margo-new-project" => json!({"action":"new-project"}),
                "margo-open-project" => json!({"action":"open-project"}),
                "margo-open-project-archive" => json!({"action":"open-project-archive"}),
                "margo-open-project-config" => json!({"action":"open-project-config"}),
                "margo-themes" => json!({"action":"themes"}),
                "margo-export-html" => json!({"action":"export","format":"html"}),
                "margo-export-pdf" => json!({"action":"export","format":"pdf"}),
                "margo-export-pptx" => json!({"action":"export","format":"pptx"}),
                _ => {
                    if let Some(index) = id
                        .strip_prefix("margo-recent-")
                        .and_then(|i| i.parse::<usize>().ok())
                    {
                        if let Ok(paths) = load_recent_projects(app) {
                            if let Some(root) = paths.get(index) {
                                let _ = app.emit_to(
                                    "main",
                                    "margo-native-menu",
                                    json!({"action":"open-project","root":root}),
                                );
                            }
                        }
                    }
                    return;
                }
            };
            let _ = app.emit_to("main", "margo-native-menu", action);
        })
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                api.prevent_close();
                let _ = window.emit_to("main", "margo-close-requested", ());
            }
        })
        .invoke_handler(tauri::generate_handler![
            engine_request,
            store_ai_key,
            has_ai_key,
            test_ai_connection,
            chat_stream,
            connect_chatgpt,
            chatgpt_connection,
            disconnect_chatgpt,
            chatgpt_models,
            choose_path,
            preview_layout,
            recent_projects,
            omarchy_palette,
            close_app
        ])
        .build(tauri::generate_context!())
        .expect("Could not start Margo desktop");
    app.run(move |_, event| {
        if let tauri::RunEvent::Exit = event {
            if let Ok(mut guard) = exit_state.0.lock() {
                if let Some(mut engine) = guard.take() {
                    // Replace and drop the pipe to deliver EOF; allow cleanup before Drop's kill fallback.
                    drop(engine.input.take());
                    let _ = engine.child.wait();
                }
            }
        }
    });
}
