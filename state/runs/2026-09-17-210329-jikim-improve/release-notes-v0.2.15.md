### 추가

- `/mcp`가 개인 API 토큰뿐 아니라 **Keycloak 액세스 토큰**으로도 열리는 **MCP SSO(OAuth)** 추가. MCP 인가 명세(2025-06-18+)의 리소스 서버 절반만 구현하며 설정은 `settings` 테이블의 새 키 `mcp`에 `{oauth:{enabled,resource,audience,scopes}}`로 저장합니다(기본값 꺼짐, 타입 검증). 켜지 않은 설치에서는 아무것도 달라지지 않습니다
- `GET /.well-known/oauth-protected-resource`와 `…/mcp`가 RFC 9728 보호 리소스 메타데이터를 인증 없이 봉투 없는 JSON(+CORS `*`)으로 내고, 꺼져 있으면 `404`입니다. `/mcp`의 `401`에만 `WWW-Authenticate: Bearer realm="jikim", resource_metadata="…"`(거부된 자격이면 `error="invalid_token"`)를 붙이고 REST·OpenBao 호환 `401`에는 붙이지 않습니다. 리소스 식별자는 `mcp.oauth.resource` → OIDC Callback URL(`oidc.redirect_url`)의 Origin + `/mcp` → 요청 `Host` 순으로 정합니다
- 같은 `Authorization: Bearer` 헤더에서 점 하나(`hvs.`/`jks.`)면 키, 세 조각이면 JWT로 갈라 `oidc.issuer_url`의 JWKS로 서명(RS/ES/PS만)·`iss`·`exp`·`nbf`를 검사하고 `typ=ID`·`cnf`·빈 `sub`는 거부합니다. 대상은 `aud`에 리소스 식별자가 있거나 `aud`/`azp`가 `mcp.oauth.audience`에 있어야 하며, 거부 메시지에 본 `aud`/`azp`와 적을 값을 넣습니다. `oidc.client_id`는 자동 허용하지 않아 웹 로그인 클라이언트의 토큰이 MCP를 열지 않습니다
- 계정은 웹 로그인이 이미 같은 issuer+subject로 연결한 활성 OIDC 계정만 찾고, 만들거나 username으로 대체하거나 role claim을 읽지 않습니다. SSO 주체는 키와 같은 정책·감사를 타되 `mcp.oauth.scopes` 상한(기본 `mcp:read`=조회 도구, `mcp:transit`=Transit 도구; 토큰 `scope`에 `mcp:*`가 있으면 교집합)을 받고 키는 좁히지 않습니다
- 스위치는 Keycloak OIDC가 꺼져 있거나 Issuer URL이 비면 저장 시 `400`으로 거부하고, 저장 뒤 OIDC가 꺼지면 조용히 잠들며 JWT가 들어올 때만 로그에 이유를 남깁니다. 관리 화면 OIDC 탭에 **MCP SSO(OAuth)** 카드(스위치·리소스 식별자·메타데이터 주소(읽기 전용)·허용 대상·범위)를 추가했습니다

### 변경

- 테스트가 실제 서명한 JWT를 만들면서 `github.com/go-jose/go-jose/v4`가 `go.mod`의 직접 의존성이 되었습니다(버전 변화 없음)

### 문서

- 관리자 가이드 3.3절에 `mcp.oauth` 설정 표, Keycloak 클라이언트·Audience 매퍼 절차, curl 확인과 거부 메시지별 조치 표를, API 및 MCP 가이드와 사용자 가이드에 "키 없이 SSO로 연결하기"를 추가했습니다. 문서 프로파일과 두 가이드 PDF 표지를 v0.2.15로 갱신했으며 화면 캡처는 실제로 찍은 `v0.2.9`를 그대로 가리킵니다

**Full Changelog**: https://github.com/hkjang/jikim/compare/v0.2.14...v0.2.15
