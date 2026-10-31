#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

#[cfg(target_os = "linux")]
use gtk::gdk::prelude::DisplayExtManual;
#[cfg(target_os = "linux")]
use gtk::prelude::*;
use serde::Deserialize;
use serde_json::{json, Value};
use std::{
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
