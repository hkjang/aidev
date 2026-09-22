use serde::{Deserialize, Serialize};
use url::Url;
#[derive(Clone, Default, Serialize, Deserialize)]
pub struct Config {
    pub server: String,
    pub allow_http: bool,
    pub remember_key: bool,
}
pub fn server_url(value: &str, allow_http: bool) -> Result<Url, String> {
    let mut url = Url::parse(value.trim()).map_err(|_| "올바른 서버 주소를 입력하세요.")?;
    if !["http", "https"].contains(&url.scheme())
        || !url.username().is_empty()
        || url.password().is_some()
        || url.query().is_some()
        || url.fragment().is_some()
        || url.path() != "/"
        || url.host_str().is_none()
    {
        return Err("경로·계정 정보가 없는 HTTP(S) 서버 주소만 사용할 수 있습니다.".into());
    }
    if url.scheme() == "http"
        && !allow_http
        && !matches!(url.host_str(), Some("localhost" | "127.0.0.1" | "[::1]"))
    {
        return Err(
            "HTTP는 암호화되지 않습니다. 신뢰된 사내 HTTP 사용을 명시적으로 확인하세요.".into(),
        );
    }
    url.set_path("/");
    Ok(url)
}
pub fn deep_link_path(value: &str) -> Result<String, String> {
    if value.len() > 4096 {
        return Err("딥 링크가 너무 깁니다.".into());
    }
    let url = Url::parse(value).map_err(|_| "잘못된 딥 링크입니다.")?;
    if url.scheme() != "madi"
        || !url.username().is_empty()
        || url.password().is_some()
        || url.port().is_some()
        || url.query().is_some()
        || url.fragment().is_some()
    {
        return Err("허용되지 않는 딥 링크입니다.".into());
    }
    match url.host_str() {
        Some("capture") if matches!(url.path(), "" | "/") => Ok("/app/inbox".into()),
        Some("page") => {
            let id = url.path().trim_start_matches('/');
            uuid::Uuid::parse_str(id).map_err(|_| "문서 UUID를 확인하세요.")?;
            Ok(format!("/app/documents/{id}"))
        }
        _ => Err("madi://capture 또는 madi://page/문서UUID만 지원합니다.".into()),
    }
}
pub fn api_route(method: &str, path: &str) -> Result<(), String> {
    if path.len() > 2048
        || path.contains(['\\', '#'])
        || path.contains("..")
        || !path.starts_with('/')
        || path.starts_with("//")
    {
        return Err("허용되지 않는 API 경로입니다.".into());
    }
    let url =
        Url::parse(&format!("https://madi.invalid{path}")).map_err(|_| "잘못된 API 경로입니다.")?;
    let route = url.path();
    let allowed = match method {
        "GET" => {
            ["/auth/me", "/workspaces", "/documents", "/captures"].contains(&route)
                || (route.starts_with("/documents/") && uuid::Uuid::parse_str(&route[11..]).is_ok())
        }
        "POST" => route == "/captures",
        "PUT" => route.starts_with("/documents/") && uuid::Uuid::parse_str(&route[11..]).is_ok(),
        _ => false,
    };
    if allowed {
        Ok(())
    } else {
        Err("데스크톱의 허용된 문서·빠른 기록 API가 아닙니다.".into())
    }
}
#[cfg(feature = "desktop")]
pub mod desktop;
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn server_origin() {
        assert!(server_url("https://wiki.company.local", false).is_ok());
        assert!(server_url("http://127.0.0.1:8080", false).is_ok());
        assert!(server_url("http://wiki.local", false).is_err());
        assert!(server_url("http://wiki.local", true).is_ok());
        for input in [
            "file:///etc/passwd",
            "https://user:secret@example.org",
            "https://example.org/app",
            "https://example.org?url=http://evil.org",
            "javascript:alert(1)",
        ] {
            assert!(server_url(input, true).is_err(), "{input}");
        }
    }
    #[test]
    fn links_cannot_change_server_or_execute() {
        assert_eq!(deep_link_path("madi://capture").unwrap(), "/app/inbox");
        assert!(deep_link_path("madi://page/1aa2bbaa-9988-4111-8444-000000000001").is_ok());
        for input in [
            "madi://capture?server=https://evil.org",
            "madi://page/../admin",
            "madi://run/shell",
            "file:///etc/passwd",
            "madi://user@capture",
            "madi://page/javascript:alert(1)",
        ] {
            assert!(deep_link_path(input).is_err(), "{input}");
        }
    }
    #[test]
    fn routes_are_allowlisted() {
        assert!(api_route("POST", "/captures").is_ok());
        assert!(api_route("GET", "/documents?workspace_id=abc").is_ok());
        for (method, path) in [
            ("GET", "/admin/settings"),
            ("POST", "/auth/login"),
            ("GET", "//evil.org/api"),
            ("GET", "/documents/../../admin"),
            ("GET", "/documents/abc"),
            ("DELETE", "/documents"),
        ] {
            assert!(api_route(method, path).is_err());
        }
    }
}
