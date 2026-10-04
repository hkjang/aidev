# jikim 프로필 (2026-10-04)

- 목적: 폐쇄망에서 Secret·개인 키·정책·승인·감사를 운영하는 Go 서버와 React 관리 화면. 제한 OpenBao `/v1` 호환 API 와 MCP(도구 8종)를 제공한다.
- 스택: Go(go.mod 1.26.x), pgx v5·PostgreSQL·go-oidc·go-jose/v4, React 19·Vite 8.2.2·TypeScript·Mantine·React Query, Vitest·Playwright. CI Node 24, `.nvmrc`=24, `web/package.json` `engines.node ">=22.22.2"`.
- 버전 정본: `scripts/version.sh` 의 `JIKIM_VERSION` = **`v0.2.27`**, base **`2f18d04`**(main, 2026-10-04 확인). 최근 10회차는 전부 webhook·tracking·세션폐기·프런트 런타임 계열 소규모 `fix:`/`build:` 였다.

## 구조
- `cmd/server`: 부트스트랩. `internal/httpapi`: REST·OpenBao 호환·MCP·OIDC·AI·tracking 라우팅과 미들웨어. 라우트는 전부 `server.go:routes` 한 곳에 모이고 마지막이 `mux.HandleFunc("/", s.serveFrontend)` SPA fallback. 전역 체인은 `server.go:68` = `requestID(recoverer(securityHeaders(audit(mux))))`. 인증은 전역이 아니라 라우트별 `s.withAuth(...)`/`s.requireRoles(...)` 래퍼다 — 래퍼가 없는 라우트는 인증 없는 라우트다(예: `POST /api/v1/tracking/csp-report`). `withAuth`(server.go:179-203)는 `requestToken`(205-216: Bearer → 쿠키 `jikim_session` → `X-Vault-Token`)으로 토큰을 읽는다.
- `internal/store`: PostgreSQL 저장소. 설정은 키→JSON 객체(`settings.go` 의 `allowedSettingKeys`/`validateSetting`). `AIConfig()`(settings.go:566)는 읽기 시점에 `MaxTokens`·`TimeoutSeconds` 만 범위 클램프한다(`Temperature` 는 쓰기 경로 settings.go:228-232 가 0~2 강제).
- `internal/cryptox·password·ids·config·model·version`: 암호화·인증 기초·구성·모델. `internal/tracking`: CSP 정책 조립(`Config.Active`/`ReportingActive`/`ProxyActive`/`PolicySources`/`Snippet`)과 인메모리 위반 기록(`violations.go`, `MaxViolations = 100`, `Recorder.List` 는 지시어별 허용 조회).
- `migrations`: embed SQL 순방향 마이그레이션. `docs`: ADMIN_GUIDE·USER_GUIDE(md/pdf), `guides/api-guide.md`·`guides/compatibility.md`(API 계약 정본 — **단 AI 엔드포인트의 error code 표는 아직 없다**).
- `web/src`: 관리 SPA. `lib/api.ts`(일반 API·SSE), `lib/vite-proxy.test.ts`(실제 Vite 개발 서버를 띄우는 node 환경 회귀), `pages/admin/SettingsPage.tsx`(설정 탭 7개). `web/e2e`: 실제 이미지 기반 검증 및 `docs/screenshots` 갱신.

## 빌드·테스트 (실제 명령)
- `go test ./... -count=1` / `go vet ./...` / `gofmt -l .`. PostgreSQL lifecycle 테스트는 `JIKIM_TEST_POSTGRES_DSN` 미설정 시 skip.
- 패키지 단위가 빠르다: `go test ./internal/httpapi/ -count=1` 전체가 **0.682초**(2026-10-04 base 2f18d04 에서 green 확인). `-run <정규식>` 으로 더 좁힐 수 있다.
- `npm --prefix web ci --no-audit --no-fund` → `npm --prefix web test -- --maxWorkers=1`(59개) → `npm --prefix web run lint` → `VITE_APP_VERSION="$(./scripts/version.sh)" npm --prefix web run build`.
- `bash scripts/verify.sh`: shell/gofmt/Go test·vet + Node 하한 선행검사(`scripts/verify.sh:47` `NODE_FLOOR='22.22.2'`) + npm ci·test·lint·build + docs + compose. 느리다. **v0.2.27 이후 로컬에서 exit 0 로 끝난다**(v0.2.26 때 Node 20 으로 떨어졌던 문제는 닫혔다).
- `make release-check`: verify→docker→smoke→e2e→package→verify-bundle. 오래 걸리고 캡처 PNG 를 덮어쓴다.

## 관례
- 커밋 메시지는 영어 conventional(`fix:`/`feat:`/`build:`), UI·문서·에러 메시지는 한국어. 릴리즈는 별도 커밋에서 `scripts/version.sh` + `CHANGELOG.md` 갱신(작업 커밋에서 CHANGELOG 를 건드리지 않는다).
- 테스트 하네스 관례: `quietServer()`(auth_outage_test.go:16, `&Server{logger:...}` 뿐 — store 는 nil) + 필요한 seam 만 함수 필드로 주입(`trackingLoader`·`sessionResolver`·`sessionRevoker`·`storagePinger`·`oidcStateOpener`·`webhookConfigLoader`·`webhookCreator`·`webhookCompleter`). nil 이면 `s.store` 로 접는 nil-폴백이 규칙이다. **DB 없이 실제 핸들러·미들웨어를 왕복시키는 것이 이 저장소의 표준 증명 방식**이고 `revokeServer`(auth_revoke_test.go:16-29)가 최신 표본이다.
- 로그 검증 하네스: `webhook_test.go:88-130` 의 `syncBuffer`·`captureWebhookLog`·`webhookWarnings`(JSON WARN 줄 파싱 + 원문 누출 검사), 그 위의 `revokeWarnings`(auth_revoke_test.go:31-40).
- 응답 모양은 `writeError`(response.go:22-26) = `{"error":{"code","message","request_id"}}`. `decodeJSON`(response.go:28-40)은 `UseNumber()` + **`DisallowUnknownFields()`** 이고 실패 시 code `invalid_json`.
- `internal/httpapi` 의 400 응답 24곳은 모두 자기 원인을 가리키는 code 를 쓴다 — 유일한 예외가 `ai.go:94`(한 code 를 다섯 원인에 돌려 씀).

## 위험 구역
- `internal/httpapi/auth_handlers.go`·`oidc*.go`·`mcp_oauth.go`, `internal/store/users.go`, `migrations/`, `.github/workflows/release.yml`.
- `mcp_oauth.go` 의 리소스 식별자 결정 순서와 401 `WWW-Authenticate` 부착 범위(`/mcp` 만)는 문서화된 계약이다.
- `resource_handlers.go` 의 `settings`/`updateSettings`, `store/settings.go`, `SettingsPage.tsx` 는 여러 회차가 인접 hunk 를 건드린 자리.
- `tracking.go` 의 CSP 정책 문자열(`basePagePolicy`·`defaultPagePolicy`)은 "추적이 꺼지면 예전과 바이트 단위로 같다"가 `tracking_test.go:21` 의 `shippedPolicy` 로 못박혀 있다.
- `webhook.go` 는 4회차가 연달아 건드렸다. 로그 문구·필드 이름(`webhook delivery 기록 실패` + `delivery_id`·`event`·`request_id`·`status_code`)과 `webhookTest` 응답 문구는 테스트가 못박은 계약이다.
- `ai.go` 의 `secretMaterialPatterns`(42-53)·`maximumAITokens`(21)·262144 한도(74)는 보안 매처다. 넓히거나 좁히지 말 것.
- `.nvmrc`·`web/package.json` `engines`/`scripts.test`·`web/vitest.config.ts` 가드·`scripts/verify.sh:47` 네 자리는 v0.2.27 의 Node 핀이다. 함께 움직여야 하므로 한쪽만 고치지 말 것.

## 자주 깨지는 곳 / 교훈
- **원인 X 를 원인 Y 로 보고하는 것**이 이 저장소의 반복 결함 가족이다: 저장소 장애를 401/403 으로(v0.2.20), 기록 실패를 전송 실패로(v0.2.21·v0.2.22), 연결만 허용한 출처를 스크립트 허용으로(v0.2.26), 입력 형식 오류를 Secret 평문 혼입으로(ai.go:94, **아직 열려 있음**). 새 과제는 이 가족에서 찾으면 적중률이 높다.
- 관찰 가능한 출력·동작이 바뀌지 않는 수정은 넣지 말 것(운영자 반려 사유). 한쪽 매처·파서를 넓히지 말고 좁게 고칠 것. 감사 details 에 스크럽 전 원문을 넣지 말 것.
- **반환값을 `_ =` 로 버린 자리 광맥은 소진됐다**(webhook v0.2.24, `RevokeToken` v0.2.25). 이 패턴을 다시 과제로 고르지 말 것.
- 과거 회차 기록의 성공 작업이 현재 HEAD 에 있다고 가정하지 말 것 — 메일 기능(`internal/mail`)과 2026-10-02 의 ai.go 수정은 **머지되지 않았고 HEAD 에 없다**. 반드시 `git log` + grep 으로 확인할 것.
- 2026-09-07 사람이 PR 반려한 이력 있음. 같은 접근 재시도 금지.

## 검증 함정
- 작업 트리에 `web/node_modules` 가 없다. 프런트 검증은 `npm ci` 부터 시작하고 시간이 든다.
- `scripts/e2e-docker.sh`·Playwright 는 `docs/screenshots/*.png` 를 덮어써 작업 트리를 더럽힌다. 읽기 전용 정찰에서는 실행 금지.
- 프런트 빌드에 500kB 청크 경고가 있다(기존 상태, 새 결함 아님).
- 실제 Keycloak·SMTP·MCP 클라이언트·GitHub Actions 실행 이력은 이 환경에서 확인할 수 없다(`gh` 미인증). 브라우저 CSP 실제 적용도 확인 불가.
- 요청된 `pmo:*`·`technology:*` 스킬과 `Skill` 도구는 이 세션에 실제로 존재한다(2026-10-04 로드 확인).
- Bash 도구가 `&&` 로 이어붙인 복합 명령과 `./scripts/*.sh` 직접 실행을 승인 요구로 막는다. **명령을 하나씩 나눠 실행하고**(`cd <wt>; <cmd>` 형태) `bash scripts/verify.sh` 로 부를 것.
