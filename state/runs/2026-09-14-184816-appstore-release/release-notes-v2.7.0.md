AppStore v2.7.0

Keycloak에 이미 세션이 있으면 로그인 화면을 거치지 않고 `prompt=none`으로 조용히 들어가는 silent SSO를 더한 minor 릴리스입니다.

- 관리자 → 인증·SSO 화면에 '자동 로그인 (Silent SSO)' 스위치가 생겼습니다. 설정은 `oidc_settings.auto_login` 한 열에 저장되고 기본값은 꺼짐이라, 새로 설치한 곳과 기존 설치는 스위치를 켜기 전까지 아무것도 달라지지 않습니다.
- 스위치를 켜면 `/api/v1/public/config`의 `oidcAutoLogin`이 true가 되고, 브라우저는 세션이 없는 첫 방문에서 탭 세션당 한 번만 `prompt=none`으로 제공자에 다녀옵니다. 제공자에 세션이 있으면 로그인 화면 없이 바로 원래 주소로 돌아오고, 없으면(`login_required`) `/login?sso=none(&returnTo=…)`으로 조용히 돌아와 평범한 로그인 화면을 봅니다. 서버는 설정이 꺼져 있으면 브라우저가 `?prompt=none`을 붙여도 무시하고, silent가 아니었던 거절은 기존처럼 401입니다.
- 루프를 막기 위해 sessionStorage 표시, 로그아웃 직후 억제, 주소의 `sso=none` 표시 세 겹을 두었고, `/login`·`/403`·`/admin/bootstrap`과 `/api`·`/mcp`·`/health` 계열 경로는 시도하지 않습니다. `returnTo`는 `/`로 시작하고 `//`가 아닌 값만 받습니다.
- OpenAPI에 `prompt`·`error` 파라미터와 `oidcAutoLogin` 필드를 적었습니다. 관리자 가이드 3.3에 설정 행과 동작 절, 장애 대응·보안 기본값 항목을 더하고 PDF를 다시 구웠습니다(35쪽).
- Migration 000004가 `oidc_settings.auto_login`과 `oidc_auth_requests.silent`를 기본값 false로 더합니다. 기동 시 자동으로 적용되므로 기존 설치는 image만 교체하면 됩니다. 실제 Keycloak으로 "로그인된 상태에서 화면이 바로 뜬다"는 이 환경에 제공자가 없어 확인하지 못했습니다.
