## Data Works v0.9.56

### 주요 변경 사항
- **Keycloak 액세스 토큰(OAuth 2.1 리소스 서버)으로 `/mcp`·`/mcp/gateway` 에 키 없이 연결하는 MCP SSO 추가 (v0.9.56)**: 두 MCP 경로는 개인 API 키만 받았음. `mcp.oauth.enabled`(기본 꺼짐)를 켜면 새 `internal/proxy/mcp_oauth.go` 가 (1) `/.well-known/oauth-protected-resource`(+`/mcp`, `/mcp/gateway` 접미)에서 인증 없이 RFC 9728 메타데이터(resource·authorization_servers=[Keycloak issuer]·bearer_methods_supported·scopes_supported·resource_name)를 `Access-Control-Allow-Origin: *` 로 내고(꺼져 있으면 404), (2) 두 MCP 경로의 401 에만 `WWW-Authenticate: Bearer realm="Data Works", resource_metadata="…"`(토큰이 있었으면 `error="invalid_token"`)를 붙여 MCP 클라이언트(Claude·Cursor)가 URL 만으로 기존 Keycloak realm 에 로그인하게 하며, (3) `authenticateProxyContext` 에서 키 조회 실패 뒤·passthrough 앞에 MCP 경로의 JWT 를 Keycloak JWKS 로 서명(RS/PS/ES 만, HS·none 거부)·iss·exp·nbf·typ(`ID` 거부)·cnf(있으면 거부)·sub·대상(`aud` 에 리소스 식별자, 또는 `aud`/`azp` 가 `mcp.oauth.audience` 목록에 있어야 하며 거부 메시지에 본 값과 고칠 값을 실음) 검사함. 서버는 토큰을 발급하지 않고, 계정은 `sub` 링크(`auth_identities`) → `email` 순으로 **이미 등록된 활성** 계정만 열며(없으면 `account_not_registered`), 아무것도 프로비저닝·복구하지 않고 토큰 role 로 승격하지 않음. 범위는 `mcp.oauth.scopes`(기본 `mcp:use`) ∩ 계정 역할 범위 ∩ 토큰 `scope`(이 앱 어휘를 실었을 때)이고 `mcp:use` 가 남지 않으면 `insufficient_scope`; 주체 id 는 `oauth_<user id>` 로 기존 MCP 로그·거버넌스 경로를 그대로 탐. REST·admin·웹은 종전대로 키와 세션만 쓰고 키 동작은 변하지 않음. 설정은 admin settings 레지스트리에 `mcp.oauth.enabled|resource|audience|scopes`(카테고리 `mcp.oauth`, 쓰기 그룹 `security`, resource 는 `…/mcp` 경로 강제·scopes 어휘 검증)로 얹어 런타임 스냅샷·멀티 파드 폴링에 편승하고 Issuer 는 Keycloak SSO 설정을 재사용(Issuer 가 비면 비활성으로 동작하고 이유를 로그·상태에 남김). 리소스 식별자는 `mcp.oauth.resource` → Keycloak Redirect URI 오리진 → 요청 Host 순으로 만들고 출처를 보고함. 공용 JWKS 캐시는 `crypto.PublicKey` 로 일반화해 EC 키를 받고 모르는 kid 재조회를 30초에 한 번으로 제한. `GET /admin/mcp/oauth/status` 와 React `Keycloak SSO` 탭의 "MCP SSO (OAuth 2.1) 연결" 카드(설정 4개·상태·문제·출처·복사 가능한 MCP URL·리소스 식별자·메타데이터 주소)를 추가. 가짜 IdP(discovery + RSA·EC JWKS) 기반 Go 테스트 8건(기본 꺼짐 시 404·거부·헤더 없음 / 메타데이터 3경로·CORS·MCP 401 헤더·REST 401 헤더 없음 / azp·aud 매퍼·ES256·PS256·이메일 매칭으로 `tools/list` 200·승격 없음·계정 수 불변 / 다른 클라이언트·만료·nbf·다른 issuer·typ=ID·HS256·cnf·모르는 kid·sub 없음·정지·미등록·viewer 12건 거부 + `mcp_oauth_denied` 감사 12건 / REST 401 / Issuer 없으면 비활성 / resource 우선·매퍼 없이는 거부 / 설정 검증·권한 그룹)을 추가. `docs/ADMIN_GUIDE.md` 4.3 절 "MCP SSO (OAuth 2.1)" 소절(설정 표·Keycloak 공개 클라이언트/Redirect/Audience 매퍼·계정 규칙·curl 확인·거부 코드별 조치 표·API)과 `docs/USER_GUIDE.md` 15절 "키 없이 SSO 로 연결하기", README 표를 추가하고 PDF 를 다시 생성. `dataworks:v0.9.56` 이미지를 `dataworks-v0.9.56.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.56.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.56.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.56
```
