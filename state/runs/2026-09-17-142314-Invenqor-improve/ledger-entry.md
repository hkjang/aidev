## 2026-09-17
- 선택: 캠페인 mcp-oauth-2026-09 — Keycloak 액세스 토큰으로 `/mcp` 에 들어오게 하는 OAuth 2.1 리소스 서버 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 이 저장소는 사용자별 계정·Keycloak OIDC 로그인(`auth.keycloak` 설정 한 덩어리, `external_identities` 로
  subject 연결)·`ivq_sk_` 개인 키·go-oidc 가 모두 있어 표준을 그대로 적용했다. 고른 것들 — (1) 설정 자리: 표준 키 이름
  `mcp.oauth.enabled/resource/audience/scopes` 를 `settings` 표에 mail.* 과 같은 방식으로 두고, 발급자·Client ID·
  username claim·사설 CA 는 새로 만들지 않고 `OIDCService.Settings()`(auth.keycloak) 를 재사용. 전용 관리 API
  `GET/PATCH /api/v1/admin/settings/mcp-oauth` 로 저장 시점에 검증(리소스 URL 모양·알려진 scope·켤 때 발급자 필수 →
  `MCP_OAUTH_OIDC_REQUIRED`). (2) 리소스 식별자 출처: `mcp.oauth.resource` → Keycloak Redirect URI 의 origin + `/mcp`
  (관리자가 이미 적은 공개 주소; 이 저장소엔 server.public_url 류가 없음) → 요청 Host(마지막 수단). (3) 계정 매핑:
  weekly 와 달리 username 폴백을 두지 않고 `external_identities(provider='keycloak', issuer, subject)` 만 본다 — 웹
  로그인이 로컬 계정과 같은 이름의 Keycloak 사용자를 `ErrOIDCUsernameTaken` 으로 거부하므로 토큰이 그보다 넓으면 안 된다.
  (4) 권한: 원칙 `mcp.oauth.scopes ∩ 계정 권한`, super_admin 도 설정 scope 를 넘지 않고, 기본 역할 중 `mcp.access` 는
  super_admin 만 가지므로 viewer 는 403(키를 만들 때와 같은 문). (5) 대상: `aud` 에 리소스 또는 `aud`/`azp` 가
  `mcp.oauth.audience`; 웹 로그인 client_id 는 자동 허용하지 않음(표준 표 그대로). 거부 메시지에 본 aud/azp 와 고칠 값.
  (6) 발급자 도달 불가는 401 이 아니라 503 `MCP_OAUTH_ISSUER_UNREACHABLE`(도전 헤더 없음) — 로그인 루프 방지.
  구현: `auth/mcp_oauth.go`(issuer+CA 별 provider 캐시 `context.WithoutCancel`, `SubjectUser` 조회), `httpapi/mcp_oauth.go`
  (설정 로드/저장·메타데이터·도전 헤더·토큰 검사: RS/ES/PS 만, `SkipClientIDCheck` 후 직접 대상 검사, typ=ID·cnf·nbf 거부),
  `authenticateAPIKey` 에서 `/mcp`+켜짐+JWT 모양일 때만 토큰 경로로 분기(꺼져 있으면 예전과 글자까지 같은 `INVALID_API_KEY`
  와 `Bearer realm="invenqor-api"`), OAuth 주체도 같은 rate limiter(`oauth:<userID>`), 감사 행위자 `sso:<username>`.
  콘솔 설정 → Keycloak 탭에 "MCP SSO (OAuth)" 카드(스위치·리소스·허용 대상·범위·MCP URL/메타데이터 주소 복사·켜짐인데
  비활성이면 이유), openapi.yaml 4개 operation + 스키마 + `/mcp` 401/503 갱신, ADMIN_GUIDE 16.4(설정 표·Keycloak
  클라이언트/Audience 매퍼 표·curl·거부 메시지별 조치), API_MCP_GUIDE 5.3 "키 없이 SSO 로 연결", README 한 줄.
  검증: 가짜 IdP(TLS httptest, 실제 RSA 키 쌍으로 discovery·JWKS 서빙, RS256 서명)로 httpapi 통합 4개 — 기본 꺼짐(메타데이터
  404·읽어도 행 없음·토큰은 키 전용 때와 동일 거부·켜기 거부·잘못된 값 400·키 그대로), 메타데이터 두 경로 맨 JSON+CORS·
  `/mcp` 401 에만 resource_metadata·REST 401 엔 없음·명시 리소스 우선·감사 2건, 토큰 수용(azp 호환 경로·aud 정식 경로·
  super_admin 도 scope 3개로 제한·tools/call·REST 에선 거부·다른 앱 토큰 메시지에 aud/azp/고칠 값·만료/nbf/다른
  issuer/typ=ID/cnf/sub 없음/HS256/쓰레기 각각 거부·미등록 subject 거부+계정 미생성·viewer 403·비활성 거부·끄면 즉시
  원상), 발급자 다운 503. `go test ./...` SQLite 전 패키지, 실 PostgreSQL(`scripts/test-postgres.sh`, 별도 컨테이너·포트)
  전 패키지, OpenAPI 경로 대조·MCP 문서 대조 통과, `go vet`·`gofmt`, vitest 146개(2개 추가), `npm run build` →
  webui/dist 동기화. 실제 Keycloak 과 실제 MCP 클라이언트(Claude/Cursor)로의 연결은 이 환경에 없어 확인하지 못했다 —
  표준 검증 항목 중 그 하나만 미확인. PDF 는 릴리즈 회차 관례라 재생성하지 않았고 버전 범프·릴리즈 노트도 하지 않았다.
- 보류 아이디어: 실제 Keycloak(또는 ReSSO)·실제 MCP 클라이언트로 URL 만 넣어 연결되는지 확인하고 가이드에 화면을 싣기 — Keycloak 이 있는 환경 필요 (가치 4 / 위험 1 / S) · MCP SSO 카드 화면 캡처를 캡처 스크립트·docs/assets/guide·가이드 그림 세 곳에 추가 — 메일·Keycloak(auto_login)·방문 추적과 함께, headless Chrome 필요 (가치 2 / 위험 1 / S) · cargo audit·govulncheck 를 main 에 대해 매일 cron 으로도 돌리기 — ci.yml 에 schedule 트리거 추가 (가치 3 / 위험 1 / S) · 콘솔 Query DSL 화면이 `total`·`offset` 을 쓰지 않아 첫 페이지만 보여줌 — webui/dist 재빌드 동반 (가치 3 / 위험 2 / M) · 여러 목록 핸들러가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — listAgents·settings·자산 상세 sources/history/relations (가치 3 / 위험 1 / M)
