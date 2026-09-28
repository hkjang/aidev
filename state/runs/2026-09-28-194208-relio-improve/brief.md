- 과제: 끊긴 질의(`context.Canceled`·`context.DeadlineExceeded`)를 "잘못된 요청" 으로 판정하지 않고 REST·MCP 두 문 모두에서 서버 오류로 접기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `serviceError`(`internal/server/server.go:716`)의 분류 사슬은 SQLSTATE(`case isPg`)와 연결 실패(`case database.Unreachable`)까지만 보고, 그 둘 중 어느 것도 아닌 컨텍스트 오류는 아래 영어 substring 어디에도 걸리지 않아 **기본값** 400 `invalid_request` 로 떨어지며 드라이버 문장을 그대로 싣는다. 질의가 끊긴 것은 호출자의 잘못이 아니라 서버 쪽 사건인데 4xx 로 답하면 (a) 클라이언트·로그·지표가 "입력이 틀렸다" 로 읽고 (b) `serveIdempotent` 의 `result.StatusCode < 500` 조건(`server.go:694`)에 걸려 그 답이 24시간 캐시 대상이 된다. MCP 쪽도 같은 오류가 `sanitizeToolError`(`internal/mcp/results.go:67`)의 마지막 줄 `return message` 로 원문 그대로 모델에게 나간다 — 2026-09-26·09-27 세 회차가 세운 "같은 DB 오류는 어느 문으로 들어오든 같은 답" 계약의 마지막 구멍이다.

**이번 정찰이 실제로 재현한 것**(임시 테스트 `internal/server/zz_probe_test.go` 를 만들어 `go test ./internal/server/ -run … -v` 로 한 번 돌리고 지웠음, 작업 트리 깨끗):
```
expired-deadline: type=context.deadlineExceededError msg="context deadline exceeded"
  deadline=true canceled=false isPg=false unreachable=false -> 400 invalid_request
  "담당자 목록을 읽을 수 없습니다: context deadline exceeded"
canceled:         type=*errors.errorString            msg="context canceled"
  deadline=false canceled=true  isPg=false unreachable=false -> 400 invalid_request
  "담당자 목록을 읽을 수 없습니다: context canceled"
exec-expired: (pool.Exec 도 동일) -> 400 invalid_request "… context deadline exceeded"
```
재현 방법은 외부 PostgreSQL 이 전혀 필요 없다: 닫아 둔 포트로 `pgxpool.NewWithConfig` 한 뒤 **이미 만료된/취소된 context** 로 `pool.Query`/`pool.Exec` 를 부르면 pgx 가 커넥션 획득 단계에서 `ctx.Err()` 를 그대로 돌려준다(`ConnectError` 가 아니다 — `unreachable=false` 를 위에서 확인). 기존 `internal/server/service_error_test.go` 의 `unreachableQueryError(t)`(196-220행)가 포트 예약·풀 생성을 이미 하고 있으니 그 헬퍼를 그대로 본떠 context 만 만료시키면 된다.

- 수용 기준:
  1) `errors.Is(err, context.Canceled)` 또는 `errors.Is(err, context.DeadlineExceeded)` 인 오류가 `serviceError` 를 통과하면 **500 `internal_error`** 와 기존 일반 문장 `"서버 오류가 발생했습니다."` 가 나가고(기존 `status >= 500` 블록이 문장 교체와 `s.Log.Error` 를 이미 한다 — 그 블록은 건드리지 말 것), 응답 본문 어디에도 `context deadline exceeded`·`context canceled` 가 남지 않는다. 원문은 로그에만 `requestId` 와 함께 남는다.
  2) 같은 오류가 `sanitizeToolError` 를 통과하면 원문 대신 그 함수가 이미 쓰는 일반 문장(`"데이터 처리 중 오류가 발생했습니다. 요청 ID: " + requestID`)이 나가고, 원문은 `slog.Warn("mcp tool database error", …)` 로 남는다. SQLSTATE→문장 표 7그룹/10코드는 **한 글자도** 바꾸지 않는다.
  3) 판정 술어는 `internal/platform/database` 에 함수 하나로 두어 두 문이 같은 정의를 읽는다(`Unreachable` 바로 아래, 예: `func Interrupted(err error) bool`). `server.go` 와 `mcp/results.go` 는 둘 다 이미 이 패키지를 import 하므로 **새 import 간선이 생기지 않는다** — 확인해 둘 것.
  4) 테스트가 증명할 것: 손으로 만든 대역 없이 **실제 `*pgxpool.Pool` 의 실제 질의 오류**(위 재현법)를 실제 `serviceError`(`httptest`) 와 실제 `sanitizeToolError` 에 넣어 500/일반문장/원문없음을 고정한다. `Canceled` 와 `DeadlineExceeded` 를 각각 별도 케이스로 둘 것. 새 분기를 각각 끄면 해당 테스트만 다시 빨개지는 것을 눈으로 확인하고 되돌린다(직전 3회차가 모두 이 절차를 밟았다).
  5) 기존 회귀 그물이 그대로 통과한다: `internal/server/service_error_test.go` 27건(특히 `TestServiceErrorKeepsExistingClassification`), `internal/server/sqlstate_parity_test.go`, `internal/mcp/arguments_test.go:189` 의 `sanitizeToolError(errors.New("name is required"), "") == "name is required"`(컨텍스트 오류가 아닌 평범한 오류는 여전히 원문 통과), `internal/mcp/results_test.go`. 삭제되는 줄 0.

- 건드릴 파일 (프로덕션 3개):
  - `internal/platform/database/database.go:28` — `Unreachable` 바로 아래에 `Interrupted(err error) bool` 추가: `errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)`. `context` 는 이미 import 되어 있다(1-17행 확인). 주석에 "컨텍스트 술어가 왜 database 패키지에 있나 — `Unreachable` 과 같은 이유로 REST·MCP 두 문이 하나의 정의를 읽게 하려고" 를 적을 것.
  - `internal/server/server.go:716 serviceError` — `case database.Unreachable(err):` **바로 뒤**, 영어 substring case 들 **앞**에 `case database.Interrupted(err):` 를 더해 `status = http.StatusInternalServerError; code = "internal_error"`. 순서가 중요하다: 다이얼 타임아웃은 `*pgconn.ConnectError` 가 `context.DeadlineExceeded` 를 감싸므로 `Unreachable` 이 먼저 잡아 기존 판정(로그의 sqlstate 처리 포함)을 유지해야 한다. `msg` 는 건드리지 말 것 — 아래 `status >= 500` 블록이 교체한다.
  - `internal/mcp/results.go:78` — `if errors.As(err, &pgErr) || strings.Contains(message, "SQLSTATE") || database.Unreachable(err) {` 조건에 `|| database.Interrupted(err)` 를 더한다. 이러면 `code` 가 빈 채로 switch 를 통과해 이미 있는 일반 문장으로 떨어진다(`f5064f6` 가 `Unreachable` 에 쓴 것과 똑같은 수법). switch 본문은 손대지 말 것.
  - 새 테스트: `internal/server/service_error_test.go` 에 케이스 추가(또는 같은 패키지 새 파일), `internal/mcp/results_test.go` 에 케이스 추가.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/server/ ./internal/mcp/ ./internal/platform/database/`  ← 이번 정찰에서 `./internal/server/` 가 도는 것을 직접 확인함
  - `go test ./...` · `go test -race ./...` · `go vet ./...` · `go build ./...`
  - `gofmt -l internal/platform/database/database.go internal/server/server.go internal/mcp/results.go <새 테스트 파일들>` (무출력이어야 함) · `git diff --check`
  - `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh`
  - 프런트는 이번 변경과 무관(Go 전용) — `npm` 단계 불필요.

- 위험과 피할 것:
  - **로그 소음**이 이 과제의 진짜 트레이드오프다. net/http 는 클라이언트가 끊기면 `r.Context()` 를 취소하므로, `Canceled` 를 500 으로 올리면 끊김마다 `s.Log.Error` 가 찍힌다. 이 제품은 에어갭 사내 B2B CRM(저트래픽)이고 끊긴 질의는 볼 가치가 있으므로 **그대로 받아들이고 로깅 블록은 바꾸지 말 것**. 로그 레벨을 나누거나 499 같은 새 상태를 도입하지 말 것 — 499 는 `< 500` 이라 idempotency 캐시 문제를 그대로 남기고, `internal/api/openapi.go` 의 오류 어휘에도 없다.
  - **미확인**: 오늘 REST 경로에서 `DeadlineExceeded` 를 실제로 만드는 생산자는 확인하지 못했다. `grep context.WithTimeout` 결과 요청 핸들러에 요청별 데드라인을 거는 자리가 없고(`cmd/relio/main.go:106` 의 `ReadTimeout`/`WriteTimeout` 은 `r.Context()` 를 취소하지 않는다), 데드라인을 거는 곳은 `internal/mail/service.go:155,167,172` 와 `internal/mcp/server.go:271` 뿐이다. 즉 **오늘의 지배적 생산자는 클라이언트 끊김의 `Canceled`** 이고 `DeadlineExceeded` 는 방어적 고정이다. 구현자는 이 사실을 커밋 메시지·PR 본문에 그대로 적을 것 — "idempotency 캐시가 오염된다" 를 단정하지 말 것. 끊긴 클라이언트의 경우 뒤이은 `s.DB.Exec(r.Context(), INSERT …)`(`server.go:695`)도 같은 죽은 context 를 쓰므로 캐시 쓰기 자체가 실패한다.
  - `serveIdempotent` 는 **읽기만** 할 것. `status < 500` 조건 변경(4xx 캐시 여부)은 사용자 계약 결정이며 별도 회차다.
  - `pgErrorVerdict`·`sanitizeToolError` 의 SQLSTATE 표와 문장은 한 글자도 바꾸지 말 것 — `sqlstate_parity_test.go` 가 두 표를 `go/ast` 로 대조해 빨개진다.
  - 보호 경로 회피: `migrations/`, `internal/auth`·`internal/oidc`, `internal/server/public.go`, `.github/workflows/` 에 손대지 말 것. 이 과제는 셋 다 건드릴 이유가 없다.
  - 임시 probe 파일을 만들었다면 반드시 지우고 `git status` 로 확인할 것(이번 정찰도 그렇게 했다).

- 차선 후보: **감사 목록 API 에 기간 필터(`from`/`to`) 추가** — 아니면 더 안전하게, `serviceError` 의 영어 substring 분류(`server.go:755-763`)를 패키지별 센티널 오류(`errors.Is`)로 **한 패키지만** 옮기기(예: `crm` 의 `not found` 계열). 354개 호출부 전체는 M 을 넘으므로 한 조각만 담고, `TestServiceErrorKeepsExistingClassification` 을 회귀 그물로 쓸 것.
