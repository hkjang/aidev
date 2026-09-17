## MCP 클라이언트에 URL 하나만 주면, 개인 키 없이 Keycloak 로그인으로 `/mcp/gateway` 에 붙습니다

MCP 인가 규격(2025-06-18 이후)은 OAuth 2.1 입니다 — 클라이언트에 서버 URL 하나만 주면 클라이언트가 **스스로** 로그인 화면을 띄우고 토큰을 받아 와야 합니다. 그런데 Clustara 의 `/mcp`·`/mcp/gateway` 는 지금까지 개인 proxy key 만 받아, 사용자마다 관리자 UI 에서 키를 만들어 Claude·Cursor 설정에 붙여 넣어야 했습니다. 이 저장소에는 사용자별 계정(`auth_users`), Keycloak OIDC 웹 로그인(`auth_identities` 에 `sub` 연결), 개인 proxy key 가 모두 이미 있었으므로 표준을 그대로 얹을 수 있었습니다. 이번 릴리즈는 **OAuth 2.1 리소스 서버 절반만** 더합니다 — 로그인·토큰 발급은 Keycloak 이 하고, Clustara 는 받은 토큰을 요청마다 검사할 뿐 authorize·token·동적 클라이언트 등록 경로를 제공하지 않습니다. **기본값은 꺼짐이고, 개인 키는 그대로 동작합니다.**

### ① 메타데이터와 401 도전이 클라이언트에게 길을 알려 줍니다

`GET /.well-known/oauth-protected-resource[/mcp[/gateway]]` 가 RFC 9728 문서(`resource`, `authorization_servers`=[SSO 발급자], `bearer_methods_supported`, `scopes_supported`)를 인증 없이 CORS `*` 로 냅니다 — 세 경로 모두 같은 문서이고, 리소스 식별자 하나가 `/mcp` 와 그 아래 `/mcp/gateway` 를 함께 덮습니다(python SDK 의 계층 검사와 맞음). MCP 경로의 401 은 `WWW-Authenticate: Bearer realm="Clustara", resource_metadata="…"` 를 실어(토큰이 거부됐으면 `error="invalid_token"` 추가) 클라이언트가 메타데이터 → Keycloak discovery → 로그인으로 이어 가게 합니다. REST 401 에는 이 헤더가 붙지 않습니다.

### ② 꺼져 있으면 아무것도 달라지지 않고, 켜져 있어도 토큰은 MCP 두 경로에서만 받습니다

키와 토큰을 가르는 자리는 기존 `authenticateProxyContext` 입니다 — 키 해시 조회가 빗나간 뒤, passthrough 분기 앞. 다만 **MCP 두 경로 + JWT 모양의 bearer + `mcp.oauth.enabled`** 일 때만 그 분기를 탑니다. 꺼져 있으면 분기 자체가 실행되지 않아 메타데이터는 404, 완벽하게 서명된 토큰도 이전과 똑같은 `invalid_api_key` 입니다(테스트로 고정). 켜져 있어도 REST·WebSocket·관리 API 는 토큰을 거부합니다.

### ③ 토큰 검사는 서명·발급자·만료 위에 대상·유형·바인딩·주체 층을 더합니다

서명은 기존 `keycloakVerifyJWT`(JWKS, RS256 전용 — `HS*`·`none` 은 alg 검사에서 거부)를 재사용하고, 그 위에 `iss`(SSO Issuer URL 과 동일), `exp`·`nbf`, `typ=ID` 거부(ID 토큰은 로그인 증거이지 API 자격이 아님), `cnf` 거부(검증할 수 없는 DPoP/mTLS 바인딩), `sub` 필수, 그리고 **대상**을 봅니다. 대상은 `aud` 에 리소스 식별자(또는 그 아래 `/mcp/gateway`)가 있거나, `aud`/`azp` 가 `mcp.oauth.audience` 에 있어야 합니다 — 실제 Keycloak 26 은 `aud` 에 `account` 만 싣고 클라이언트 ID 는 `azp` 에 담으므로, 매퍼 없이 쓰려면 MCP 클라이언트 ID 를 `mcp.oauth.audience` 에 적으면 됩니다. 웹 로그인 client_id 는 자동으로 허용하지 않습니다(표준의 Keycloak 절대로 MCP 클라이언트는 다른 클라이언트여야 함). 거부 메시지는 **본 값과 고칠 값**을 적고(`aud=[…], azp="…"`), `error.code` 는 감사 이벤트 `mcp_oauth_denied` 에 기록됩니다.

### ④ 주체는 이미 등록된 활성 계정만, 권한은 DB 의 역할만

토큰의 `sub` 로 연결된 SSO 신원(웹 로그인이 남긴 것) → 없으면 토큰의 `email` 과 같은 계정 순으로 **찾기만** 합니다. 계정을 만들지 않고, 비활성·비밀번호 변경 강제 계정은 거부하며, 토큰의 realm role 이 admin 이어도 역할은 올라가지 않습니다(테스트로 고정). 범위는 `mcp.oauth.scopes` ∩ 계정 역할의 범위 ∩ 토큰 `scope` 의 Clustara 어휘입니다. 주체는 그 사용자가 개인 키로 들어왔을 때와 같은 문(역할·범위·`mcp:use` 게이트·정책·쿼터)을 지나고, 감사 로그에는 `sso:<user id>` 로 남습니다.

### ⑤ 설정은 런타임 설정 레지스트리에, 발급자·클라이언트는 SSO 설정을 재사용합니다

| 런타임 설정 키 | 환경변수 | 기본값 | 뜻 |
|---|---|---|---|
| `mcp.oauth.enabled` | `MCP_OAUTH_ENABLED` | `false` | SSO 액세스 토큰으로 `/mcp` 접속 허용 |
| `mcp.oauth.resource` | `MCP_OAUTH_RESOURCE` | 빈 값 | 리소스 식별자(RFC 8707). 비우면 SSO Redirect URI 의 origin + `/mcp`, 그것도 없으면 요청 Host |
| `mcp.oauth.audience` | `MCP_OAUTH_AUDIENCE` | 빈 값 | 공백 구분 허용 대상. 토큰의 `aud` 또는 `azp` 와 비교 |
| `mcp.oauth.scopes` | `MCP_OAUTH_SCOPES` | `mcp:use` | SSO 토큰 주체에게 주는 범위 — 계정 역할의 범위와 교집합만 적용 |

카테고리 `mcp.oauth`(권한 그룹 security)라 파드 재시작 없이 모든 파드에 적용됩니다. 관리 화면은 **설정 → SSO (Keycloak) → MCP SSO (OAuth) 카드**(스위치·식별자·허용 대상·범위, MCP URL·메타데이터 URL·Audience 매퍼 값 복사). ADMIN_GUIDE 에 설정 표·Keycloak 클라이언트/Audience 매퍼 절차·curl 확인·거부 메시지별 조치 표가, USER_GUIDE §3.13 에 "키 없이 SSO 로 연결하기" 가 추가됐습니다.

### 검증

가짜 IdP(실제 RSA 키, discovery·JWKS 서빙)로 서명한 JWT 를 쓰는 신규 테스트 5개가 표준의 테스트 목록 전 항목을 덮습니다: 꺼짐 기본값(메타데이터 404, 유효 토큰도 `invalid_api_key`), 401 헤더와 REST 미부착, 정식 `aud` 통과, `azp` 호환 통과, 다른 앱 토큰 거부 메시지에 본 값과 고칠 값, 만료·다른 issuer·잘못된 키·HS256·`typ=ID`·`cnf`·`nbf`·`sub` 없음 각각 거부, 미등록/비활성 계정 거부 + 계정 미생성, 유효 토큰의 REST 거부, 키는 그대로. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(20 패키지, 릴리즈 게이트 포함).

**알아 둘 것**: 서명은 RS256 만 받으므로 realm 의 기본 서명 알고리즘이 PS256/ES256 이면 거부됩니다(거부 메시지가 안내). Clustara 는 introspection 을 하지 않으므로 Keycloak 에서 로그아웃해도 이미 발급된 토큰은 만료까지 삽니다 — 액세스 토큰 수명은 짧게 두세요. 실제 Keycloak·실제 MCP 클라이언트로의 종단 확인은 이번 릴리즈 범위 밖입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.283.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.283.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.283.md | 오프라인 배포 가이드 |

```bash
sha256sum -c clustara-v0.9.283.tar.gz.sha256
gunzip -c clustara-v0.9.283.tar.gz | docker load   # Loaded image: clustara:v0.9.283
```
