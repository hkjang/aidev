# 과제서 — 2026-09-20 qurio

- 과제: OIDC 콜백의 비조용 제공자 오류(`error=access_denied` 등)를 `auth.oidc.failed` 감사 이벤트로 기록하고, 프로덕션 라우터+실제 PostgreSQL 통합 테스트로 증명(그 안에서 OIDC state store의 return_to·silent 왕복도 함께 검증) (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/httpapi/auth_handlers.go:184` 는 Keycloak 이 `error=...` 로 돌려보낸 상호작용 로그인 실패를 `logger.Warn` 만 남기고 감사 로그(`qurio_audit_logs`)에는 아무것도 쓰지 않는다 — 같은 함수 191행의 `callback_failed` 분기는 기록하는데, 제공자가 거절한 경우(사용자 권한 없음·동의 거부·클라이언트 설정 오류)만 관리자 감사 화면에서 보이지 않는다. 고치면 관리자가 "왜 로그인이 안 되나" 를 로그 파일 없이 감사 이벤트만으로 추적할 수 있고, 덤으로 `store.PutOIDCState/ConsumeOIDCState` 의 `return_to`·`silent` 컬럼(migration 0033)이 실제 DB 에서 왕복되는지 처음으로 통합 테스트에 올라간다.

- 수용 기준:
  1) `GET /api/v1/auth/oidc/callback?error=access_denied&state=<유효한 비조용 state>` 요청 뒤 `qurio_audit_logs` 에 `event='auth.oidc.failed'`, `action='login'`, `resource='authentication'`, `actor_user_id IS NULL`, `details->>'reason'='provider_error'`, `details->>'error'='access_denied'` 인 행이 정확히 1건 생기고 응답은 여전히 `303 → /login?error=oidc`.
  2) `error=login_required&state=<silent state>` 는 전과 같이 `303 → /login?sso=none` 이고 감사 행은 생기지 않는다(조용한 시도의 거절은 정상 응답). 같은 state 를 다시 보내면(일회용 소진) `/login?error=oidc` 로 가고 이번엔 `provider_error` 감사 행이 생긴다. `state` 없이/위조 state 로 `error=...` 만 와도 `provider_error` 감사 행이 생긴다.
  3) 감사 `details` 에는 `reason` 과 `error` 코드만 들어간다 — `error_description`, `state` 원문, `return_to` 는 절대 넣지 않는다(운영자 규칙: 감사 details 에 원문 문자열 금지). `error` 값은 64자 이하로 잘라 넣는다(`store.truncate` 가 이벤트명만 자르므로 핸들러에서 직접).
  4) 통합 테스트가 **프로덕션 배선**으로 증명한다: `httpapi.New(Options{Store: store.New(pool, keyring), OIDC: oidcauth.New(repository), Logger: ...})` 로 만든 서버의 실제 라우터(`server.Handler()` 또는 기존 `mcp_oauth_integration_test.go` 가 쓰는 방법 그대로 — 미확인, 그 파일 75행 이후를 보고 따라갈 것)에 요청을 보내고, state 는 `repository.PutOIDCState(ctx, authn.HashToken(raw), store.OIDCState{Silent:…, ReturnTo:"/workspace/sql", RedirectURI:…, Nonce:…, PKCEVerifier:…, ExpiresAt: now+5m})` 로 실제 테이블에 넣는다. 손으로 만든 가짜 repository 를 감사 증거로 쓰지 말 것.
  5) 같은 통합 테스트에서 `ConsumeOIDCState` 왕복을 직접 확인: silent=true·return_to='/workspace/sql' 로 Put 한 행을 Consume 하면 두 값이 그대로 돌아오고 두 번째 Consume 은 `store.ErrNotFound`; `ExpiresAt` 이 과거인 행은 Consume 시 `ErrNotFound`; `ReturnTo:""` 로 Put 한 행은 `''` 로 돌아온다(NULLIF/COALESCE 경로).
  6) 기존 단위 테스트 `TestOIDCCallbackRoutesSilentRefusalToLoginMarker` 는 **그대로 두면 nil store 로 패닉한다** — 아래 "위험" 참고. `go test ./...` 와 `go test -race -p=1 -tags=integration ./internal/httpapi ./internal/store` 모두 통과.
  7) docs/guides/admin-guide.md 의 OIDC/감사 절에 `auth.oidc.failed` 의 `reason` 두 값(`callback_failed`, `provider_error`)과 `error` 필드 한 줄 추가(문서에 감사 이벤트 표가 없으면 Keycloak 로그인 문제 해결 절에 한 문장으로).

- 건드릴 파일:
  - `internal/httpapi/auth_handlers.go:handleOIDCCallback` (175~187행) — `Abandon` 이 silent 가 아니라고 판정한 뒤, 기존 `logger.Warn` 옆에 `s.auditSecurity(r, nil, "auth.oidc.failed", "login", "authentication", "", map[string]any{"reason": "provider_error", "error": <64자로 자른 providerError>})` 추가. 191행 `callback_failed` 호출과 인자 형태를 똑같이 맞출 것.
  - `internal/httpapi/auth_handlers_test.go:TestOIDCCallbackRoutesSilentRefusalToLoginMarker` — `New(Options{OIDC: …})` 는 `Store` 가 nil 이라 `auditSecurity → s.store.WriteAudit → s.pool.Exec` 에서 nil 역참조 패닉. 이 단위 테스트는 감사에 닿기 전에 돌아오는 첫 케이스("silent refusal" → `/login?sso=none`)만 남기고, 나머지 4 케이스(replayed silent refusal / interactive failure / error without state / forged state)는 아래 통합 테스트로 옮긴다. **`auditSecurity` 에 `if s.store == nil { return }` 같은 보호를 넣어 단위 테스트를 살리는 방법은 쓰지 말 것**(배선 결함을 숨기고, 저장소에 그런 nil 가드 선례가 없음 — `grep "store == nil" internal/httpapi` 0건 확인).
  - `internal/httpapi/oidc_callback_audit_integration_test.go` (새 파일, `//go:build integration`) — `mcp_oauth_integration_test.go` 의 앞부분(29~49행: `QURIO_TEST_POSTGRES_DSN` skip, `pgxpool.New`, `store.RunMigrations(ctx, pool, migrations.FS, ".")`, `cryptox.New([]byte("0123456789abcdef0123456789abcdef"))`, `store.New(pool, keyring)`) 을 그대로 따라 서버를 만들고, 수용 기준 1)·2)·4)·5) 를 검증. 감사 행은 `SELECT count(*), details FROM qurio_audit_logs WHERE event='auth.oidc.failed' AND request_id=$1` 처럼 이 테스트만의 request_id 헤더(미들웨어가 어느 헤더를 읽는지 `internal/httpapi/context.go:requestIDFrom` 과 `middleware.go` 에서 확인 — 미확인) 또는 시각 범위+`details->>'error'` 고유값(예: `access_denied` 대신 스펙 허용 코드 중 테스트 고유 접미가 불가하므로 request_id 방식 권장)으로 골라내고 `t.Cleanup` 에서 `DELETE FROM qurio_audit_logs WHERE …`, `DELETE FROM qurio_oidc_states WHERE state_hash=ANY(...)` 로 정리.
  - `docs/guides/admin-guide.md` — 한 줄.

- 검증 명령:
  - `go vet ./... && go vet -tags=integration ./...`
  - `go test ./internal/httpapi ./internal/oidcauth ./internal/store -count=1` (단위)
  - PostgreSQL 이 필요한 통합: `docker run -d --name qurio-it -e POSTGRES_USER=qurio -e POSTGRES_PASSWORD=qurio-test -e POSTGRES_DB=qurio -p 127.0.0.1:55432:5432 postgres:17-alpine` 뒤
    `QURIO_TEST_POSTGRES_DSN='postgres://qurio:qurio-test@127.0.0.1:55432/qurio?sslmode=disable' QURIO_INTEGRATION_DSN="$QURIO_TEST_POSTGRES_DSN" go test -race -p=1 -tags=integration ./internal/httpapi ./internal/store -count=1` (2회 연속 통과 확인 — 정리 누락 검출). 끝나면 `docker rm -f qurio-it`.
  - `go test ./...` 전체 (약 1~2분, 이 세션에서 `./internal/httpapi` 단위 테스트 0.09s 통과 확인).
  - CI(`.github/workflows/ci.yml`)는 `go test -race -p=1 -tags=integration ./...` 를 postgres:17 서비스로 돌리므로 새 통합 테스트는 CI 에서 실제로 실행된다 — 정리(cleanup)를 빠뜨리면 다른 패키지 테스트를 오염시킬 수 있음.

- 위험과 피할 것:
  - **nil store 패닉**: 위 단위 테스트 축소 없이 핸들러만 바꾸면 `go test ./internal/httpapi` 가 패닉으로 실패한다. 반대로 nil 가드로 덮지 말 것.
  - `auth_handlers.go` 는 auth 보호 경로다 — 리다이렉트 목적지·`Abandon` 호출 순서·`Complete` 경로·쿠키 설정은 한 글자도 바꾸지 말고 `auditSecurity` 한 줄만 더한다. `oidcauth` 패키지는 건드리지 않는다.
  - 감사 details 에 `error_description`·state·return_to 원문을 넣지 않는다(운영자 규칙). `error` 는 제공자 값이라 64자로 자른다.
  - 실제 출력이 바뀌지 않는 수정은 넣지 말 것 — 이 과제는 감사 행 1건이 새로 생기는 관찰 가능한 변화가 있다.
  - migrations/ 는 건드리지 않는다(0033 에 `return_to`·`silent` 이미 있음).
  - 통합 테스트는 공유 DB 를 쓰므로 고유 nonce(`fmt.Sprintf("%x", time.Now().UnixNano())`)로 state 를 만들고 반드시 정리.

- 차선 후보: `gofmt -l` 검사를 `Makefile:lint` 와 `.github/workflows/ci.yml` 에 추가(`test -z "$(gofmt -l cmd internal)"`) — 먼저 `gofmt -l cmd internal` 을 돌려 드리프트가 있으면 그 파일만 `gofmt -w` 로 정리(이 세션에서는 gofmt 실행 권한이 없어 드리프트 유무 미확인). 그다음 순위: `web/src/pages/LoginPage.tsx:81` 로컬 로그인 후 `/workspace` 고정 대신 이미 계산된 `returnTo`(47행)로 이동(`mustChangePassword` 면 `/password-renew` 우선) + `LoginPage.test.tsx` 신설(`useAuth`/`apiClient` 모킹, `MemoryRouter initialEntries=[{pathname:'/login', state:{from:'/workspace/sql'}}]`).
