# jikim 프로필 (2026-09-26)

- 목적: 폐쇄망에서 Secret·개인 키·정책·승인·감사를 운영하는 Go 서버와 React 관리 화면; 제한 OpenBao `/v1` 호환 API와 MCP(도구 8종)를 제공한다.
- 스택: Go(go.mod 1.26.x), pgx v5·PostgreSQL·go-oidc·go-jose/v4, React 19·Vite 8.2.2·TypeScript·Mantine·React Query, Vitest·Playwright. CI Node 24, 로컬 Node 22.
- 버전 정본: `scripts/version.sh` 의 `JIKIM_VERSION`. 현재 **`v0.2.22`**, HEAD **`6424731`**(main). 최근 5회차는 전부 webhook/Vite 프록시 계열 소규모 fix 였다.

## 구조
- `cmd/server`: 부트스트랩. `internal/httpapi`: REST·OpenBao 호환·MCP·OIDC·AI·tracking 라우팅과 미들웨어. 라우트는 전부 `server.go:routes` 한 곳에 모이고 마지막이 `mux.HandleFunc("/", s.serveFrontend)` SPA fallback. 인증은 전역 미들웨어가 아니라 라우트별 `s.requireRoles(...)` 래퍼다 — 래퍼가 없는 라우트는 인증 없는 라우트다(예: `POST /api/v1/tracking/csp-report`).
- `internal/store`: PostgreSQL 저장소. 설정은 키→JSON 객체(`settings.go` 의 `allowedSettingKeys`/`validateSetting`), 그 외 Secret·사용자·세션·Transit·감사. `AIConfig()` 처럼 읽기 시점에 범위 클램프를 하는 곳이 있다.
- `internal/cryptox·password·ids·config·model·version·tracking`: 암호화·인증 기초·구성·모델·방문 추적.
- `internal/tracking`: CSP 정책 조립(`tracking.go` 의 `Config.Active`/`ProxyActive`/`PolicySources`/`Snippet`)과 인메모리 위반 기록(`violations.go`, `MaxViolations = 100`, LastSeen 기준 LRU 축출).
- `migrations`: embed SQL 순방향 마이그레이션. `docs`: ADMIN_GUIDE·USER_GUIDE(md/pdf), `guides/api-guide.md`·`guides/compatibility.md`(API 계약 정본), 홍보 사이트·`screenshots`.
- `web/src`: 관리 SPA. `lib/api.ts`(일반 API·SSE), `lib/vite-proxy.test.ts`(실제 Vite 개발 서버를 띄우는 node 환경 회귀), `pages/admin/SettingsPage.tsx`(설정 탭 7개: general·approval·oidc·ai·security·notifications·tracking).
- `web/e2e`: 실제 이미지 기반 데스크톱·모바일 검증 및 `docs/screenshots` 갱신. `scripts`·`Makefile`: 검증·Docker·스모크·E2E·오프라인 패키징.

## 빌드·테스트 (실제 명령)
- `go test ./... -count=1` / `go vet ./...` / `gofmt -l .`. PostgreSQL lifecycle 테스트는 `JIKIM_TEST_POSTGRES_DSN` 미설정 시 skip.
- 패키지 단위: `go test ./internal/httpapi/ -run <정규식> -count=1 -v` 가 빠르다(전체 httpapi 0.2초 미만).
- `npm --prefix web ci --no-audit --no-fund` → `npm --prefix web test -- --maxWorkers=1` → `npm --prefix web run lint` → `VITE_APP_VERSION="$(./scripts/version.sh)" npm --prefix web run build`.
- `./scripts/verify.sh`: shell/gofmt/Go test·vet + npm ci·test·lint·build + docs + compose. 성공 시 약 42줄.
- `make release-check`: verify→docker→smoke→e2e→package→verify-bundle. 오래 걸리고 캡처 PNG를 덮어쓴다.

## 관례
- 커밋 메시지는 영어 conventional(`fix:`/`feat:`/`build:`), UI·문서·에러 메시지는 한국어. 릴리즈는 별도 커밋에서 `scripts/version.sh` + `CHANGELOG.md` 갱신(작업 커밋에서 CHANGELOG를 건드리지 않는다).
- 운영 설정은 관리자 UI → store JSON(키별 validate). 부트스트랩 환경변수는 네 개뿐. `CONTRIBUTING.md`·`docs/guides/compatibility.md` 의 계약을 지킬 것.
- 테스트 하네스 관례: `quietServer()` + 필요한 seam만 함수 필드로 주입(`trackingLoader`·`sessionResolver`·`storagePinger`·webhook 계열 seam). DB 없이 실제 핸들러를 왕복시키는 것이 이 저장소의 표준 증명 방식이고, nil 이면 안전한 기본값으로 접는 nil-폴백 관례가 있다.
- store 호출 seam은 작은 메서드로 모아 `New()` 배선 없이 동작하게 한다(webhook.go 선례).

## 위험 구역
- `internal/httpapi/auth_handlers.go`·`oidc*.go`·`mcp_oauth.go`, `internal/store/users.go`, `migrations/`, `.github/workflows/release.yml`.
- `mcp_oauth.go` 의 리소스 식별자 결정 순서(`mcp.oauth.resource` → `oidc.redirect_url` Origin → `Host`)와 401 `WWW-Authenticate` 부착 범위(`/mcp` 만)는 문서화된 계약이다.
- `resource_handlers.go` 의 `settings`/`updateSettings`, `store/settings.go`, `SettingsPage.tsx` 는 여러 회차가 인접 hunk를 건드린 자리.
- `internal/httpapi/tracking.go` 의 CSP 정책 문자열(`basePagePolicy`·`defaultPagePolicy`)은 "추적이 꺼지면 예전과 바이트 단위로 같다"가 `tracking_test.go:21` 의 `shippedPolicy` 로 못박혀 있다.

## 자주 깨지는 곳 / 교훈
- 저장소 장애를 401/403으로 접지 말고 500과 구분할 것. 기록 실패를 전송 실패로 보고하지 말 것(v0.2.21·v0.2.22 가 이 계열). 감사 details 에 스크럽 전 원문을 넣지 말 것.
- 관찰 가능한 출력·동작이 바뀌지 않는 수정은 넣지 말 것(운영자 반려 사유). 한쪽 매처·파서를 넓히지 말고 좁게 고칠 것.
- 반환값을 `_ =` 로 버린 자리가 반복 결함원이었다. 남은 `_ =` 대부분은 `tx.Rollback`·`json.Unmarshal(폴백 있음)`·타입 단정으로 정당하다.
- 과거 회차 기록의 성공 작업이 현재 HEAD에 있다고 가정하지 말 것 — 메일 기능(`internal/mail`·mail 설정 탭)은 여러 회차째 미머지이며 HEAD에 없다.
- 2026-09-07 사람이 PR 반려한 이력 있음. 같은 접근 재시도 금지.

## 검증 함정
- 작업 트리에 `web/node_modules` 가 없다. 프런트 검증은 `npm ci` 부터 시작하고 시간이 든다. 로컬 Node 22 / CI Node 24 차이 주의.
- `scripts/e2e-docker.sh`·Playwright 는 `docs/screenshots/*.png` 를 덮어써 작업 트리를 더럽힌다. 읽기 전용 정찰에서는 실행 금지.
- 실제 Keycloak·SMTP·MCP 클라이언트·GitHub Actions 실행 이력은 이 환경에서 확인할 수 없다(`gh` 미인증).
- 요청된 `pmo:*`·`technology:*` 스킬과 `Skill` 도구는 **이 세션에 실제로 존재한다.** 과거 "스킬을 찾지 못했다"는 기록을 근거로 건너뛰지 말 것.
- Bash 도구가 `&&` 로 이어붙인 복합 명령을 승인 요구로 막는 경우가 있다. 명령을 하나씩 나눠 실행할 것.
