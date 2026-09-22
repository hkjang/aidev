use crate::{api_route, deep_link_path, server_url, Config};
use serde_json::{json, Value};
use std::{fs, sync::Mutex, time::Duration};
use tauri::{Emitter, Manager, WebviewUrl, WebviewWindow, WebviewWindowBuilder};
use tauri_plugin_deep_link::DeepLinkExt;
use tauri_plugin_global_shortcut::{GlobalShortcutExt, ShortcutState};
#[derive(Default)]
struct Session {
    config: Config,
    token: String,
    generation: u64,
    pending_link: Option<String>,
    tray_available: bool,
    shortcut_available: bool,
}
type AppState = Mutex<Session>;
fn local(window: &WebviewWindow) -> Result<(), String> {
    let url = window
        .url()
        .map_err(|_| "기기 화면을 확인할 수 없습니다.")?;
    let origin = url.origin().ascii_serialization();
    let allowed = url.scheme() == "tauri" && url.host_str() == Some("localhost")
        || matches!(
            origin.as_str(),
            "http://tauri.localhost" | "https://tauri.localhost"
        )
        || (cfg!(debug_assertions) && origin == "http://127.0.0.1:1420");
    if window.label() != "main" || !allowed {
        return Err("원격 서비스 화면에는 기기 접근 권한이 없습니다.".into());
    }
    Ok(())
}
fn credential(server: &str) -> Result<keyring::Entry, String> {
    keyring::Entry::new("com.madi.desktop", server)
        .map_err(|_| "운영체제 자격 증명 저장소를 열 수 없습니다.".into())
}
fn state_values(app: &tauri::AppHandle) -> Result<(Config, String), String> {
    let state = app.state::<AppState>();
    let state = state
        .lock()
        .map_err(|_| "연결 상태를 확인할 수 없습니다.")?;
    if state.token.is_empty() {
        return Err("개인 API 키로 연결하세요.".into());
    }
    Ok((state.config.clone(), state.token.clone()))
}
async fn request(
    config: &Config,
    token: &str,
    method: &str,
    path: &str,
    data: Option<Value>,
) -> Result<Value, String> {
    api_route(method, path)?;
    let origin = server_url(&config.server, config.allow_http)?;
    let url = origin
        .join(&format!("api/v1{path}"))
        .map_err(|_| "API 주소 오류")?;
    let client = reqwest::Client::builder()
        .redirect(reqwest::redirect::Policy::none())
        .timeout(Duration::from_secs(45))
        .build()
        .map_err(|_| "보안 연결을 초기화하지 못했습니다.")?;
    let mut req = client
        .request(method.parse().map_err(|_| "API 메서드 오류")?, url)
        .bearer_auth(token)
        .header("X-Madi-Request", "1");
    if let Some(value) = data {
        if value.to_string().len() > 4 << 20 {
            return Err("본문은 4 MB 이하로 작성하세요.".into());
        }
        req = req.json(&value);
    }
    let mut response = req
        .send()
        .await
        .map_err(|_| "서버에 연결하지 못했습니다. 주소·인증서·사내망 연결을 확인하세요.")?;
    let status = response.status();
    if response.content_length().unwrap_or(0) > 8 << 20 {
        return Err("서버 응답 크기 한도를 초과했습니다.".into());
    }
    let mut bytes = Vec::new();
    while let Some(chunk) = response
        .chunk()
        .await
        .map_err(|_| "응답 수신이 중단되었습니다.")?
    {
        if bytes.len() + chunk.len() > 8 << 20 {
            return Err("서버 응답 크기 한도를 초과했습니다.".into());
        }
        bytes.extend_from_slice(&chunk);
    }
    let value: Value = serde_json::from_slice(&bytes)
        .map_err(|_| "서버가 올바른 JSON 응답을 반환하지 않았습니다.")?;
    if !status.is_success() {
        return Err(value
            .get("error")
            .and_then(Value::as_str)
            .unwrap_or("API 요청에 실패했습니다.")
            .into());
    }
    Ok(value)
}
#[tauri::command]
async fn connect(
    window: WebviewWindow,
    app: tauri::AppHandle,
    config: Config,
    token: String,
) -> Result<Value, String> {
    local(&window)?;
    let attempt = {
        let managed = app.state::<AppState>();
        let mut state = managed.lock().map_err(|_| "연결 상태 오류")?;
        state.generation += 1;
        state.generation
    };
    let server = server_url(&config.server, config.allow_http)?
        .origin()
        .ascii_serialization();
    if !token.starts_with("madi_") || token.len() > 300 {
        return Err("올바른 개인 API 키를 입력하세요.".into());
    }
    let config = Config { server, ..config };
    let user = request(&config, &token, "GET", "/auth/me", None).await?;
    let managed = app.state::<AppState>();
    let mut state = managed.lock().map_err(|_| "연결 상태 오류")?;
    if state.generation != attempt {
        return Err("서버 연결이 취소되었습니다.".into());
    }
    if config.remember_key {
        credential(&config.server)?.set_password(&token).map_err(|_|"운영체제 자격 증명 저장소에 저장할 수 없습니다. 세션 연결만 사용하거나 OS 키체인을 먼저 잠금 해제하세요.")?;
    } else if let Ok(entry) = credential(&config.server) {
        let _ = entry.delete_credential();
    }
    let dir = app.path().app_config_dir().map_err(|_| "설정 경로 오류")?;
    fs::create_dir_all(&dir).map_err(|_| "설정 폴더를 만들 수 없습니다.")?;
    fs::write(
        dir.join("connection.json"),
        serde_json::to_vec(&config).unwrap(),
    )
    .map_err(|_| "서버 설정을 저장할 수 없습니다.")?;
    if state.config.server != config.server {
        if let Ok(entry) = credential(&state.config.server) {
            let _ = entry.delete_credential();
        }
        if let Some(remote) = app.get_webview_window("workspace") {
            let _ = remote.close();
        }
    }
    state.config = config;
    state.token = token;
    Ok(user)
}
#[tauri::command]
fn connection_status(window: WebviewWindow, app: tauri::AppHandle) -> Result<Value, String> {
    local(&window)?;
    let state = app.state::<AppState>();
    let mut state = state.lock().map_err(|_| "연결 상태 오류")?;
    let pending_link = state.pending_link.take();
    Ok(
        json!({"config":state.config,"connected":!state.token.is_empty(),"version":env!("CARGO_PKG_VERSION"),"pending_link":pending_link,"tray_available":state.tray_available,"shortcut_available":state.shortcut_available}),
    )
}
#[tauri::command]
async fn api_request(
    window: WebviewWindow,
    app: tauri::AppHandle,
    method: String,
    path: String,
    data: Option<Value>,
) -> Result<Value, String> {
    local(&window)?;
    let (config, token) = state_values(&app)?;
    request(&config, &token, &method, &path, data).await
}
#[tauri::command]
fn disconnect(window: WebviewWindow, app: tauri::AppHandle) -> Result<(), String> {
    local(&window)?;
    let state = app.state::<AppState>();
    let mut state = state.lock().map_err(|_| "연결 상태 오류")?;
    state.generation += 1;
    let remove_key = state.config.remember_key;
    let server = state.config.server.clone();
    state.token.clear();
    state.config.remember_key = false;
    // Persist the disconnected state before touching the OS keyring. A locked
    // keyring must never make the next process silently restore this session.
    let persisted = app.path().app_config_dir().ok().is_some_and(|dir| {
        fs::create_dir_all(&dir).is_ok()
            && fs::write(
                dir.join("connection.json"),
                serde_json::to_vec(&state.config).unwrap(),
            )
            .is_ok()
    });
    if let Some(remote) = app.get_webview_window("workspace") {
        let _ = remote.close();
    }
    app.emit_to("main", "madi-lock", ())
        .map_err(|_| "잠금 알림 오류")?;
    if remove_key {
        let deleted = credential(&server).is_ok_and(|entry| {
            matches!(
                entry.delete_credential(),
                Ok(()) | Err(keyring::Error::NoEntry)
            )
        });
        if !deleted {
            return Err("세션은 해제했지만 키체인의 키를 삭제하지 못했습니다. OS 키체인에서 com.madi.desktop 항목을 삭제하고 개인화에서 해당 키를 폐기하세요.".into());
        }
    }
    if !persisted {
        return Err("현재 세션 키는 삭제했지만 연결 해제 설정을 저장하지 못했습니다. 기기 설정 폴더의 쓰기 권한을 확인하세요.".into());
    }
    Ok(())
}
fn show_main(app: &tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.set_focus();
    }
}
fn open_remote(app: &tauri::AppHandle, path: &str) -> Result<(), String> {
    if !matches!(path, "/app" | "/app/inbox" | "/app/devices")
        && !(path.starts_with("/app/documents/") && uuid::Uuid::parse_str(&path[15..]).is_ok())
    {
        return Err("허용되지 않는 서비스 경로입니다.".into());
    }
    let state = app.state::<AppState>();
    let state = state.lock().map_err(|_| "연결 설정 오류")?;
    let server = server_url(&state.config.server, state.config.allow_http)?;
    let target = server.join(path).map_err(|_| "서비스 주소 오류")?;
    let origin = target.origin();
    let allow_http = state.config.allow_http;
    drop(state);
    if let Some(window) = app.get_webview_window("workspace") {
        window
            .navigate(target)
            .map_err(|_| "서비스 화면 이동 오류")?;
        let _ = window.show();
        let _ = window.set_focus();
        return Ok(());
    }
    let navigation_app = app.clone();
    WebviewWindowBuilder::new(app, "workspace", WebviewUrl::External(target))
        .title("madi · 워크스페이스")
        .inner_size(1440.0, 960.0)
        .on_navigation(move |url| {
            let allowed = url.origin() == origin
                || url.scheme() == "https"
                || (allow_http && url.scheme() == "http");
            if allowed {
                if let Some(window) = navigation_app.get_webview_window("workspace") {
                    let _ =
                        window.set_title(&format!("madi · {}", url.origin().ascii_serialization()));
                }
            }
            allowed
        })
        .build()
        .map_err(|_| "서비스 창을 열 수 없습니다.")?;
    Ok(())
}
#[tauri::command]
fn open_workspace(
    window: WebviewWindow,
    app: tauri::AppHandle,
    path: String,
) -> Result<(), String> {
    local(&window)?;
    open_remote(&app, &path)
}
#[tauri::command]
async fn import_markdown(window: WebviewWindow) -> Result<Option<Value>, String> {
    local(&window)?;
    let Some(file) = rfd::AsyncFileDialog::new()
        .add_filter("Markdown 문서", &["md", "markdown", "txt"])
        .pick_file()
        .await
    else {
        return Ok(None);
    };
    let path = file.path();
    let meta = fs::symlink_metadata(&path).map_err(|_| "파일 정보를 확인할 수 없습니다.")?;
    if !meta.is_file() || meta.file_type().is_symlink() || meta.len() > 4 << 20 {
        return Err("4 MB 이하의 일반 Markdown 파일만 가져올 수 있습니다.".into());
    }
    let text = fs::read_to_string(&path).map_err(|_| "UTF-8 문서 파일을 선택하세요.")?;
    Ok(Some(
        json!({"title":path.file_stem().and_then(|name|name.to_str()).unwrap_or("가져온 문서"),"text":text}),
    ))
}
#[tauri::command]
async fn export_markdown(
    window: WebviewWindow,
    title: String,
    text: String,
) -> Result<bool, String> {
    local(&window)?;
    if text.len() > 4 << 20 {
        return Err("내보내기 한도는 4 MB입니다.".into());
    }
    let name: String = title
        .chars()
        .filter(|ch| ch.is_alphanumeric() || " -_".contains(*ch))
        .take(100)
        .collect();
    let Some(file) = rfd::AsyncFileDialog::new()
        .set_file_name(format!(
            "{}.md",
            if name.is_empty() { "madi" } else { &name }
        ))
        .add_filter("Markdown", &["md"])
        .save_file()
        .await
    else {
        return Ok(false);
    };
    let path = file.path();
    if fs::symlink_metadata(&path).is_ok_and(|meta| meta.file_type().is_symlink()) {
        return Err("심볼릭 링크에는 내보낼 수 없습니다.".into());
    }
    fs::write(path, text).map_err(|_| "선택한 파일에 저장할 수 없습니다.")?;
    Ok(true)
}
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_single_instance::init(|app, _, _| {
            show_main(app)
        }))
        .plugin(tauri_plugin_deep_link::init())
        .plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_handler(|app, _, event| {
                    if event.state() == ShortcutState::Pressed {
                        show_main(app);
                        let _ = app.emit_to("main", "madi-capture", ());
                    }
                })
                .build(),
        )
        .manage(Mutex::new(Session::default()))
        .invoke_handler(tauri::generate_handler![
            connect,
            connection_status,
            api_request,
            disconnect,
            open_workspace,
            import_markdown,
            export_markdown
        ])
        .setup(|app| {
            if let Ok(dir) = app.path().app_config_dir() {
                if let Ok(bytes) = fs::read(dir.join("connection.json")) {
                    if let Ok(config) = serde_json::from_slice::<Config>(&bytes) {
                        if server_url(&config.server, config.allow_http).is_ok() {
                            let token = if config.remember_key {
                                credential(&config.server)
                                    .and_then(|entry| {
                                        entry.get_password().map_err(|_| "키체인 잠김".into())
                                    })
                                    .unwrap_or_default()
                            } else {
                                String::new()
                            };
                            *app.state::<AppState>().lock().unwrap() = Session {
                                config,
                                token,
                                ..Session::default()
                            };
                        }
                    }
                }
            }
            let shortcut_available = app
                .global_shortcut()
                .register("CommandOrControl+Shift+Space")
                .is_ok();
            app.state::<AppState>().lock().unwrap().shortcut_available = shortcut_available;
            use tauri::menu::{Menu, MenuItem};
            let quick = MenuItem::with_id(app, "capture", "빠른 기록", true, None::<&str>)?;
            let workspace =
                MenuItem::with_id(app, "workspace", "워크스페이스 열기", true, None::<&str>)?;
            let quit = MenuItem::with_id(app, "quit", "madi 종료", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&quick, &workspace, &quit])?;
            let mut tray = tauri::tray::TrayIconBuilder::new()
                .menu(&menu)
                .tooltip("madi · 빠른 기록")
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "capture" => {
                        show_main(app);
                        let _ = app.emit_to("main", "madi-capture", ());
                    }
                    "workspace" => {
                        if open_remote(app, "/app").is_err() {
                            show_main(app);
                        }
                    }
                    "quit" => app.exit(0),
                    _ => {}
                });
            if let Some(icon) = app.default_window_icon() {
                tray = tray.icon(icon.clone());
            }
            app.state::<AppState>().lock().unwrap().tray_available = tray.build(app).is_ok();
            let handle = app.handle().clone();
            app.deep_link().on_open_url(move |event| {
                for url in event.urls() {
                    if let Ok(path) = deep_link_path(url.as_str()) {
                        handle.state::<AppState>().lock().unwrap().pending_link =
                            Some(path.clone());
                        show_main(&handle);
                        let _ = handle.emit_to("main", "madi-deep-link", path);
                    }
                }
            });
            if let Ok(Some(urls)) = app.deep_link().get_current() {
                if let Some(path) = urls
                    .iter()
                    .find_map(|url| deep_link_path(url.as_str()).ok())
                {
                    app.state::<AppState>().lock().unwrap().pending_link = Some(path);
                }
            }
            Ok(())
        })
        .on_window_event(|window, event| {
            if window.label() == "main" {
                if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                    if window
                        .app_handle()
                        .state::<AppState>()
                        .lock()
                        .unwrap()
                        .tray_available
                    {
                        api.prevent_close();
                        let _ = window.hide();
                    }
                }
            }
        })
        .run(tauri::generate_context!())
        .expect("madi 데스크톱을 시작하지 못했습니다.");
}
