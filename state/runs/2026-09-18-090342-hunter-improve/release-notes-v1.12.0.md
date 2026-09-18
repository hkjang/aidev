## Hunter v1.12.0

폐쇄망 반입용 서비스 Docker 이미지입니다. PostgreSQL은 사내에 별도 준비합니다.

- 이미지: `hunter:v1.12.0`
- 유일한 첨부 자산: `hunter-v1.12.0.tar.gz`
- 플랫폼: `linux/amd64`
- SHA-256: (release.yml 이 실제 아카이브의 값으로 기록)

### MCP · SSO 연결 — /mcp 를 OAuth 2.1 리소스 서버로

- **키 없이 SSO 로 연결**: 관리자가 **서비스 설정 → MCP · SSO 연결**(`mcp.oauth.enabled/resource/audience/scopes`, 기본 꺼짐)을 켜면 MCP 클라이언트(Claude, Cursor 등)에 `/mcp` 주소 하나만 넣어도 클라이언트가 스스로 Keycloak 로그인을 거쳐 액세스 토큰을 받아 옵니다. 개인 키 페이지의 MCP 카드에 "키 없이 SSO 로 연결하기" 안내가 보이며 개인 키 체계는 그대로입니다.
- **Hunter 는 리소스 서버**: 토큰을 발급하지 않고(`/authorize`·`/token`·동적 클라이언트 등록 없음) `/.well-known/oauth-protected-resource(/mcp)` 메타데이터(RFC 9728, 맨 JSON, CORS `*`, 꺼지면 404)와 `/mcp` 401 의 `WWW-Authenticate: Bearer realm="hunter", resource_metadata="…"` 로 클라이언트를 Keycloak 으로 보냅니다. 이 헤더는 MCP 경로에만 붙고 REST·GraphQL·관리 API 는 지금처럼 키·세션만 받습니다. 리소스 식별자는 비우면 서비스 외부 접근 주소 + `/mcp` 이며 요청 `Host` 헤더는 쓰지 않습니다.
- **토큰 검사와 계정 매핑**: 같은 Bearer 헤더에서 `hnt_` 접두사는 키, JWT 모양은 SSO 토큰으로 가르고, `typ=ID`·`HS*`/`none` 은 서명키 요청 전에 거부한 뒤 서명·`iss`·`exp`·`nbf`·`cnf`·`sub`·대상(`aud` 의 리소스 식별자 또는 `aud`/`azp` 의 허용 대상·웹 Client ID)을 검사합니다. 계정은 웹 로그인과 같은 `iss|sub` 해시로 **이미 웹 로그인한 활성 계정**만 찾으며 계정 생성·사용자명 대체 조회·토큰 role 승격은 없습니다. 권한은 `mcp.oauth.scopes`(기본 읽기 3개) ∩ 역할 권한이고 교집합이 비면 거부합니다.
- **운영 안내**: 켤 때 OIDC 꺼짐·Issuer 없음·빈 범위는 400 으로 거부하고, 켠 뒤 전제가 사라지면 꺼진 것처럼 동작하며 `mcp oauth switched on but inactive` 로그를 남깁니다. 거부는 401 본문의 메시지(다른 대상이면 본 `aud`/`azp` 와 적을 값)와 서버 로그 `mcp sso token rejected cause=…` 로 남기고, SSO 토큰으로 호출한 도구는 감사 기록에 `auth: sso` 로 표시됩니다. 관리자 가이드 §15.6 에 Keycloak 클라이언트·Audience 매퍼 표와 curl 확인 절차가 있습니다.
- **기존 보호 유지**: 다른 서비스로 보내기 허용 목록, OIDC 자동 진입 기본 꺼짐, state·nonce·PKCE·서명 토큰·현재 역할 검사, 방문 추적의 격리 미리보기와 CSP 차단 출처 패널은 v1.11.0과 같습니다.

Hunter 는 introspection 을 하지 않으므로 Keycloak 로그아웃·사용자 비활성화 뒤에도 이미 발급된 토큰은 만료까지 살며, 급하면 Hunter 사용자를 비활성화합니다(비활성 계정 토큰은 즉시 거부). 실제 Keycloak·실제 MCP 클라이언트로 URL 만 넣어 연결하는 검증은 가짜 IdP 가 서명한 Keycloak 26 모양 토큰으로 대체했습니다.

네 환경변수, 일반 PostgreSQL과 서비스 Docker 이미지 하나의 배포 조건을 유지합니다. 최종 게시 커밋의 CI·공개 파일 검증 결과는 실제 완료 후 이 본문에 별도로 기록합니다.

[MCP·SSO 연결 운영 가이드](https://hkjang.github.io/hunter/guides/admin-guide.html) · [사용자 키 없이 연결하기 가이드](https://hkjang.github.io/hunter/guides/user-guide.html) · [릴리즈 노트](https://github.com/hkjang/hunter/blob/main/docs/release-v1.12.0.md) · [검증 기록](https://github.com/hkjang/hunter/blob/main/docs/validation.md)

### 설치

```sh
docker load -i hunter-v1.12.0.tar.gz
docker compose up -d
```

[설치 및 관리자 가이드](https://hkjang.github.io/hunter/guides/admin-guide.html) · [사용자 가이드](https://hkjang.github.io/hunter/guides/user-guide.html)

GitHub가 자동 표시하는 소스 코드 다운로드는 릴리즈 첨부 자산과 별개입니다.
