# MCP OAuth(Keycloak SSO) 표준

사내 서비스 서른 곳이 `/mcp` 를 열어 두고 있고, 전부 **개인 키** 로만 들어간다. 사람이
앱마다 키 페이지를 찾아 키를 만들고, 클라이언트 설정에 붙여 넣고, 잃어버리면 다시 만든다.
MCP 인가 규격(2025-06-18 이후)은 OAuth 2.1 이다 — 클라이언트에 URL 하나만 주면 클라이언트가
스스로 로그인 화면을 띄우고 토큰을 받아 온다. 이미 Keycloak 에 로그인한 사람이면 화면조차
거의 보지 않는다.

이 표준은 **키 체계를 그대로 둔 채 Keycloak 액세스 토큰으로도 `/mcp` 에 들어오게 하는
방법**을 각 서비스에 같은 모양으로 넣기 위한 것이다. 참조 구현은 **weekly** 다. 코드를
베끼지 말고 그 저장소의 언어와 구조에 맞게 같은 능력을 만든다.

- `weekly/internal/app/mcpoauth.go` — 메타데이터 문서, 401 도전(challenge), 토큰 검증, 계정 매핑. 한 파일 257줄
- `weekly/internal/app/auth.go` 의 `authenticate` — 같은 `Authorization: Bearer` 헤더에서 키와 토큰을 가르는 자리
- `weekly/internal/app/mcpoauth_test.go` — 가짜 IdP 로 실제 JWT 를 서명해 검사하는 테스트
- `weekly/docs/MCP.md` §"SSO(OAuth)로 연결하기", `weekly/docs/ADMIN_GUIDE.md` §4.5 — 사용자·관리자 안내

더 엄격한 검사가 필요할 때 보는 곳은 **postra** 다. `postra/internal/application/mcp_oauth.go`
(ID 토큰 거부, `nbf`, `cnf`, JWKS 재요청 제한) 와 `postra/docs/MCP_OAUTH.md` (Keycloak 클라이언트·
Audience 매퍼 설정 표, 문제 해결 표). 관리자 가이드의 순서도는 `kanpic/docs/ADMIN_GUIDE.md` §4.2
를 참고한다.

## 이 서버는 리소스 서버다 — 인증 서버가 아니다

로그인은 Keycloak 이 한다. 이 서버는 **토큰을 받아 검사하는 쪽**이다. 따라서:

- `/authorize`, `/token`, 동적 클라이언트 등록(RFC 7591) 엔드포인트를 **만들지 않는다.**
  Keycloak 이 이미 다 가지고 있다. 클라이언트 등록은 Keycloak 의 몫이다(사전 등록이 기본,
  필요하면 realm 의 Client registration 정책).
- 인증 서버 메타데이터(RFC 8414)를 대신 서빙하지 않는다. `authorization_servers` 에 Keycloak
  realm 의 issuer 를 적으면 클라이언트가 `/.well-known/openid-configuration` 을 직접 읽는다.
- 토큰을 **저장하지 않는다.** 세션으로 바꾸지도 않는다. 요청마다 검사한다.

## 클라이언트가 스스로 찾아오게 한다

### 1. 보호 리소스 메타데이터 (RFC 9728)

`GET /.well-known/oauth-protected-resource` 와 `GET /.well-known/oauth-protected-resource/mcp`
두 경로 모두에서 **인증 없이**, 제품의 응답 봉투가 아니라 **맨 JSON** 으로 답한다.
`Access-Control-Allow-Origin: *` 를 붙인다(브라우저 안에서 도는 클라이언트가 읽는다).

```json
{
  "resource": "https://<공개 주소>/mcp",
  "authorization_servers": ["https://keycloak/realms/<realm>"],
  "bearer_methods_supported": ["header"],
  "scopes_supported": ["mcp:read", "..."],
  "resource_name": "<앱 이름> MCP"
}
```

꺼져 있으면 `404` 다. 메타데이터가 있는데 토큰을 거부하면 클라이언트는 로그인 루프에 빠진다.

### 2. 401 이 길을 가리킨다

`/mcp` 가 토큰 없이 또는 거부된 토큰으로 불리면 `401` 과 함께

```
WWW-Authenticate: Bearer realm="<앱>", resource_metadata="https://<공개 주소>/.well-known/oauth-protected-resource/mcp"
```

를 보낸다. 토큰이 있었는데 거부했으면 `, error="invalid_token"` 을 더한다. **MCP 경로에서만**
붙인다. REST 401 에 이 헤더가 붙으면 브라우저와 다른 클라이언트가 엉뚱한 곳으로 간다.

### 3. 리소스 식별자는 설정에서 나온다

`resource` 는 클라이언트가 실제로 접속하는 **공개 HTTPS 주소 + MCP 경로** 다. 프록시 뒤의
`http://127.0.0.1:8080/mcp` 가 아니다. 이미 공개 주소 설정(`server.public_url` 류)이 있으면
그것으로 만들고, 없으면 `mcp.oauth.resource` 로 적게 한다. 요청의 `Host` 헤더에서 만드는 것은
둘 다 비었을 때의 마지막 수단이다 — 누구나 헤더를 바꿀 수 있다.

## 같은 헤더, 두 종류의 자격

키는 그대로 둔다. 키 없이 못 붙는 클라이언트(폐쇄망, 자동화 스크립트)가 여전히 있다.

`Authorization: Bearer <값>` 하나에서 가른다:

1. 값이 이 앱의 **키 접두사**(`wky_`, `mk_`, `ssak_` …)로 시작하거나 키 표에서 찾히면 → 키.
   지금과 똑같이 동작한다.
2. 아니고 **JWT 모양**(점 두 개, 세 조각이 모두 비어 있지 않음)이면 → OAuth 토큰 검사.
3. 둘 다 아니면 → 지금과 같은 "잘못된 키" 응답. 켜져 있지 않은 설치에서 새로운 말을 흘리지 않는다.

OAuth 토큰은 **`/mcp` 에서만** 받는다. REST·웹소켓·관리 API 는 지금처럼 키와 세션만 받는다.
토큰 하나로 앱 전체가 열리면 이 일은 "MCP 를 편하게" 가 아니라 "API 전체에 새 문" 이 된다.

## 토큰을 어떻게 믿는가

대부분의 저장소에 이미 `github.com/coreos/go-oidc/v3` 가 있다(웹 로그인용). 같은 issuer 의
provider 를 재사용하되 요청 컨텍스트에 묶지 말고(`context.WithoutCancel`) 캐시한다.
go-oidc 가 없는 저장소(hand-rolled Keycloak)는 discovery 의 `jwks_uri` 를 읽어 같은 검사를 한다.

반드시 검사하는 것:

| 항목 | 규칙 |
|---|---|
| 서명 | Keycloak JWKS. 알고리즘은 RS/ES/PS 계열만. `HS*`·`none` 거부 |
| `iss` | `oidc.issuer_url` 과 같아야 한다 |
| `exp` · `nbf` | 만료·아직 유효하지 않음 거부 |
| `typ` | `ID` 면 거부. ID 토큰은 로그인 증거지 API 자격이 아니다 |
| `cnf` | 있으면 거부. 검증할 수 없는 소지자 증명(DPoP·mTLS)이 묶인 토큰이다 |
| `sub` | 비어 있으면 거부 |
| **대상(audience)** | 아래 |

### 대상 검사 — 이 토큰이 이 서버를 위한 것인가

다른 앱에 로그인해 받은 토큰이 이 앱의 `/mcp` 를 열어서는 안 된다. 그래서 **어느 하나는
맞아야 한다**:

- `aud` 에 리소스 식별자(`https://<공개 주소>/mcp`)가 있다 — Keycloak 에 Audience 매퍼를 둔 정식 경로
- `aud` 또는 `azp` 가 관리자가 적은 `mcp.oauth.audience` 목록에 있다 — 매퍼 없이 쓰는 호환 경로.
  실제 Keycloak 26 은 `aud` 에 `account` 만 싣고 클라이언트 ID 는 `azp` 에 담으므로, 관리자가
  MCP 클라이언트 ID 를 여기 적으면 된다

아무것도 맞지 않으면 거부하되, **무엇을 봤고 무엇을 적으면 되는지** 말한다("토큰의 aud=[account],
azp=claude-mcp — 허용 대상에 claude-mcp 를 더하거나 Audience 매퍼에 https://…/mcp 를 넣으세요").
weekly 의 거부 메시지가 그 예다. 운영자는 이 메시지 하나로 설정을 끝낸다.

`SkipClientIDCheck` 로 라이브러리의 대상 검사를 끄고 직접 하는 이유가 이것이다. 라이브러리는
`aud` 만 보고 `azp` 를 모른다.

### 계정은 만들지 않는다

토큰의 `sub`(없으면 `oidc.username_claim`, 기본 `preferred_username`) 로 **이미 등록된, 활성**
계정을 찾는다. 없으면 거부하고 "웹으로 먼저 한 번 로그인하세요" 라고 말한다.

웹으로 로그인하는 순간이 등록이다. 프로그램이 토큰을 내미는 순간은 누군가를 등록할 자리가
아니다 — 정지된 계정이 MCP 로 되살아나거나, 토큰의 role claim 으로 관리자가 되는 일이 생긴다.
토큰의 role 을 권한으로 옮기지 않는다.

### 권한은 키보다 넓지 않다

OAuth 로 들어온 주체는 **그 사용자가 키를 만들어 들어왔을 때와 같은 문** 을 지난다. 키에 범위
검사·경로 표가 있으면 같은 표를 탄다. 범위는 토큰의 `scope` 가 아니라 관리자 설정
`mcp.oauth.scopes` 가 정한다 — Keycloak 에 이 앱의 범위 어휘를 가르치지 않아도 되게 하기
위해서다. 앱에 범위 어휘가 있고 토큰에 그 어휘가 실려 오면 **교집합** 만 준다.

## 설정

이름을 weekly 와 같게 쓴다. 앱마다 다르면 운영자가 서른 번 다르게 배운다.

| 키 | 기본값 | 뜻 |
|---|---|---|
| `mcp.oauth.enabled` | `false` | **꺼짐이 기본.** 관리자가 켠다 |
| `mcp.oauth.resource` | 빈 값 | 리소스 식별자. 비면 공개 주소 설정 + MCP 경로로 만든다 |
| `mcp.oauth.audience` | 빈 값 | 공백 구분 허용 대상. `aud` 또는 `azp` 와 비교 |
| `mcp.oauth.scopes` | 앱의 읽기 범위 | 공백 구분. SSO 토큰 주체에게 주는 범위 |
| (재사용) `oidc.issuer_url` · `oidc.client_id` · `oidc.username_claim` | 웹 로그인 설정 | 새로 만들지 않는다 |

켜는 조건은 세 가지가 다 있을 때다 — OIDC 가 구성돼 있고(issuer 있음), 리소스 식별자를 만들 수
있고, MCP 가 켜져 있다. 하나라도 없으면 켜 두어도 조용히 꺼진 것처럼 동작하고 이유를 로그에
남긴다. 설정 저장 시점에 거부할 수 있으면(postra 처럼) 그것이 더 낫다.

기존 OIDC 설정이 파일·환경 변수라면 같은 자리에 같은 이름(`MCP_OAUTH_ENABLED` 등)으로 둔다.
관리 화면 설정이라면 같은 화면의 카드 하나로 둔다.

## 화면과 문서

- 관리 화면의 OIDC(또는 MCP) 카드에 **켜기 스위치·리소스 식별자·메타데이터 주소·허용 대상·범위**
  를 두고, 연결에 필요한 값(MCP URL, 메타데이터 URL)을 복사할 수 있게 한다.
- 개인 키 페이지(또는 MCP 연결 안내)에 "키 없이 SSO 로 연결하기" 절을 더한다 — URL 하나 주면 된다는 것.
- `docs/ADMIN_GUIDE.md` 에 절을 더한다: 설정 표, Keycloak 쪽 할 일(아래), `curl` 로 메타데이터와
  401 헤더를 확인하는 방법, 거부 메시지별 조치. `docs/MCP.md` 가 있으면 사용자 쪽 절을 더한다.

Keycloak 쪽 할 일(관리자 가이드에 그대로 적는다):

1. MCP 클라이언트용 **공개(public) 클라이언트** 를 만든다. Standard Flow 켬, PKCE `S256`,
   Direct Access Grants·Implicit·Service accounts 끔. 웹 로그인 클라이언트와 **다른** 클라이언트다.
2. Valid Redirect URIs 에 쓰는 MCP 클라이언트의 콜백을 정확히 적는다(Claude 는
   `https://claude.ai/api/mcp/auth_callback`, 로컬 클라이언트는 `http://127.0.0.1:*/callback` 류).
   `*` 하나로 다 여는 것은 금지.
3. 정식 경로: 그 클라이언트(또는 전용 client scope)에 **Audience 매퍼** — Included Custom Audience
   = 리소스 식별자, Add to access token 켬, Add to ID token 끔. 호환 경로: 매퍼 없이 이 앱의
   `mcp.oauth.audience` 에 클라이언트 ID 를 적는다.
4. 액세스 토큰 수명은 짧게(5분 안팎). 이 서버는 introspection 을 하지 않으므로 Keycloak 에서
   로그아웃해도 이미 발급된 토큰은 만료까지 산다 — 이 한계를 가이드에 적는다.

## 하지 말아야 할 것

- 전송 방식(POST 전용이든 SSE 든)을 이 일에서 바꾸지 않는다. 인증만 더한다.
- MCP 경로 밖에서 OAuth 토큰을 받지 않는다.
- 토큰으로 계정을 만들거나, 정지된 계정을 열거나, role claim 으로 권한을 올리지 않는다.
- 이 앱이 인증 서버 노릇을 하지 않는다(`/token`, 등록 프록시 금지).
- Keycloak 이 프레임·CORS 를 허용하는지에 기대지 않는다. 메타데이터만 CORS 로 연다.
- **이 저장소에 사용자별 계정과 OIDC 로그인이 없으면**(공유 정적 토큰 하나로 MCP 를 여는 곳)
  억지로 만들지 않는다. 원장에 그 사실을 적고 변경 없이 끝낸다.

## 테스트

가짜 IdP 로 실제 키 쌍을 만들어 JWT 를 서명하고 discovery·JWKS 를 서빙하는 테스트 서버를 둔다
(`weekly/internal/app/mcpoauth_test.go`, `kanpic/internal/auth/token_test.go` 참고).

- 꺼져 있으면 메타데이터는 404 이고, 토큰을 내밀어도 키 전용 때와 똑같이 거부된다.
- 토큰 없이 `/mcp` 를 부르면 401 에 `resource_metadata` 가 붙고, REST 401 에는 붙지 않는다.
- 이 서버를 대상으로 발급된 토큰은 등록된 계정으로 `tools/list` 를 연다.
- 다른 대상의 토큰은 거부되고, 메시지에 본 `aud`/`azp` 와 고칠 값이 들어 있다.
- 관리자가 `mcp.oauth.audience` 에 `azp` 를 적으면 매퍼 없이 통과한다.
- 만료·다른 issuer·`typ=ID`·`HS256` 서명·`cnf` 있는 토큰은 각각 거부된다.
- 등록되지 않았거나 비활성인 계정의 토큰은 거부되고 계정은 생기지 않는다.
- 유효한 토큰으로 REST 경로를 부르면 거부된다.
- 키는 전과 똑같이 동작한다(기존 키 테스트가 그대로 통과한다).

## 검증

- [ ] 기본값이 꺼짐이고, 새로 설치한 곳은 아무것도 달라지지 않는다.
- [ ] 실제 Keycloak(또는 ReSSO) 으로 토큰을 받아 `/mcp` 가 열리는 것을 확인했다 — 가능하면
      실제 MCP 클라이언트(Claude·Cursor)로 URL 만 넣어 연결했다.
- [ ] 다른 앱용 토큰으로는 열리지 않는다.
- [ ] 기존 키 흐름이 그대로다.
- [ ] 관리자 가이드에 설정 표·Keycloak 설정·확인 방법·거부 메시지별 조치를 적었다.
- [ ] 무엇을 골랐고(설정 자리, 리소스 식별자 출처) 왜 골랐는지 원장에 적었다.
