- 과제: REST `serviceError` 가 PostgreSQL 오류를 원문 그대로 내보내는 것을 MCP 와 같은 어휘로 분류하기 (가치 4 / 위험 2 / 작업량 M)

- 왜: 같은 DB 오류를 두 경로가 다르게 읽는다. MCP 는 `internal/mcp/results.go:58 sanitizeToolError` 가 SQLSTATE 를 보고 "스키마 정보가 클라이언트에 그대로 가면 안 된다"는 이유를 주석에 적고 9개 코드를 한국어 문장으로 바꾸지만, REST 의 `internal/server/server.go:714 serviceError` 는 같은 오류가 `not found`·`permission`·`already` 같은 영어 substring 에 걸리지 않으면 **400 `invalid_request` + `err.Error()` 원문**으로 내보낸다. 그래서 `relation "contacts" does not exist (SQLSTATE 42P01)` 은 400 으로(서버 결함인데), `permission denied for table contacts` (42501) 는 403 + 표 이름을 달고 클라이언트에 도착한다. 고치면 REST 도 MCP 와 같은 문장·같은 판정을 주고 스키마 이름이 응답에서 사라지며 서버 결함이 5xx 로 보인다(354개 `serviceError(` 호출 전부가 이 문을 지난다).

- 수용 기준:
  1) `serviceError` 가 오류 사슬에 `*pgconn.PgError` 가 있거나 메시지에 `SQLSTATE` 가 들어 있으면 원문을 응답에 넣지 않는다 — 응답 `message` 에 `SQLSTATE`·`relation`·표 이름·제약 이름이 한 글자도 남지 않는다.
  2) SQLSTATE 별 판정이 `sanitizeToolError` 와 같은 어휘다:
     - `22P02`,`22007`,`22008`,`22003`,`23502`,`23514` → 400 `invalid_request`
     - `23505`,`23503`,`40001`,`40P01` → 409 `conflict`
     - 그 밖(`42P01`,`42501`,`53300`,`XX000`, 코드 없음) → 500 + 기존 `"서버 오류가 발생했습니다."`
     문장은 `internal/mcp/results.go:78~92` 의 한국어 문장을 그대로 쓴다(두 경로가 같은 입력에 같은 말을 하게).
  3) 500 으로 접는 경우 원문은 `s.Log.Error` 로 `sqlstate`·`requestId` 와 함께 남는다(`sanitizeToolError` 의 `slog.Warn("mcp tool database error", …)` 와 같은 관용). 응답 본문에는 원문이 없다.
  4) 기존에 이미 분류되던 오류는 그대로다 — `pgx.ErrNoRows`/`no rows in result set` → 404 + 기존 한국어 문장, `ErrCustomerCodeTaken`("customer code is already registered") → 409, `permission`/`access denied`/`designated approver` → 403, `another user`/`already`/`pending` → 409. 이 오류들은 사슬에 `PgError` 를 담고 있지 않음을 테스트가 함께 고정한다(`customerCodeConflict`·`deleteGuarded` 는 `%w` 없이 새 오류를 만든다 — 확인함).
  5) 테스트가 증명하는 것: 실제 `&pgconn.PgError{Code:…, Message:…}` 를 실제 `serviceError` 에 `httptest.NewRecorder()`/`httptest.NewRequest()` 로 넣어 상태 코드·`code`·`message` 를 검사하고, 특히 42P01·42501 케이스에서 응답 JSON 전체 문자열에 `SQLSTATE`·`contacts` 가 없음을 검사한다. 새 분기를 지우면 이 테스트가 다시 실패해야 한다(red 확인 필수).

- 건드릴 파일:
  - `internal/server/server.go:714 serviceError` — `pgx.ErrNoRows` 분기 **바로 뒤, 기존 substring 분기 앞**에 SQLSTATE 분기를 넣는다. 앞에 두는 이유: 42501 `permission denied for table …` 이 지금 `permission` substring 에 걸려 표 이름째로 403 이 되는 자리가 바로 여기다. 이 순서면 substring 분기가 못 보게 되는 오류는 "지금 원문을 흘리고 있는 오류" 뿐이다(4번 기준으로 고정).
  - `internal/server/server.go` import 에 `github.com/jackc/pgx/v5/pgconn` 추가(`internal/server/admin_crud.go:16` 이 이미 쓰는 모듈, go.mod `pgx/v5 v5.7.5`).
  - `internal/server/server_test.go` 또는 새 `internal/server/service_error_test.go` — 위 5번 테스트. `internal/mcp/arguments_test.go:171` 이 이미 실제 `&pgconn.PgError` 를 만들어 쓰는 관용을 따른다.
  - `internal/mcp/results.go:58` 와 `internal/server/server.go:714` 양쪽 주석에 서로를 가리키는 한 줄(“같은 표가 저쪽에도 있다”). **두 함수를 하나로 합치지 말 것** — 반환 계약이 다르다(MCP 는 문자열 하나, REST 는 status+code+message). 게다가 `internal/server` 가 `internal/mcp` 를 import 하므로 반대 방향 import 는 순환이다.

- 검증 명령:
  - `go test ./internal/server/` (먼저 red 확인 → green)
  - `go test ./...` · `go test -race ./...` · `go vet ./...` · `go build ./...`
  - `gofmt -l internal/server/server.go internal/server/service_error_test.go internal/mcp/results.go` (출력 없어야 함)
  - `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh`
  - (선택, 시간 있으면) 정적 바이너리 + throwaway PostgreSQL 17 로 `ALTER TABLE contacts RENAME TO contacts_x` 뒤 목록 API 를 쳐서 고치기 전 400+`SQLSTATE 42P01`, 고친 뒤 500+일반 문장을 눈으로 확인. 2026-09-24 회차에서 이 환경의 docker 가 도는 것을 확인했다.

- 위험과 피할 것:
  - 상태 코드가 400 → 409/500 으로 바뀌는 오류가 있다. 클라이언트 영향은 확인함: `web/src/api.ts:19` 는 `status`·`code`·`message` 를 `APIError` 에 담기만 하고, 유일한 소비자 `web/src/App.tsx:120 errorMessage` 는 `.message` 만 읽는다. `invalid_request` 문자열을 읽는 곳은 저장소에 없다(`internal/api/openapi.go` 에도 없음). **그래도 구현자는 이 두 파일을 직접 열어 재확인할 것.**
  - `internal/server/openapi_contract_test.go` 가 오류 응답 모양을 검사하는지 먼저 볼 것. 봉투(`error.{code,message,requestId}`)는 `httpx.ErrorJSON` 그대로라 바뀌지 않지만 확인하고 시작할 것.
  - 보호 경로를 건드리지 않는다: `internal/auth`·`internal/oidc`·`internal/server/public.go`·`migrations/`·`.github/workflows/`. 이 과제는 그 어느 것도 필요 없다.
  - `web/` 는 손대지 않는다(변경 불필요). `AdminPages.tsx` 는 한 줄이 극단적으로 길어 포매터 금지 — 이번엔 열 일이 없다.
  - 문자열 grep 을 증거로 제출하지 말 것. 증명은 실제 `pgconn.PgError` 를 실제 `serviceError` 에 통과시킨 응답이다.
  - 연결 실패(`failed to connect to host=…`)는 `PgError` 가 아니라 이번 분기에 안 잡힌다. **미확인**: 이 저장소의 pgx 5.7.5 에 `pgconn.ConnectError` 가 있는지 이번 정찰에서 확인하지 못했다. 범위 밖으로 두고 아이디어로만 남길 것.

- 차선 후보: `DealsAtRisk`·`Coaching` 의 열린 딜 200건 상한에 커서 페이징 (`internal/intelligence/health.go:585`, `forecast.go:253` 의 `Limit:200`) — 결과 상한 계약을 먼저 정해야 하므로 1순위가 성립하지 않을 때만. 그보다 더 작은 안전판이 필요하면 감사 화면 Frame 부제를 실제 6개 채널값(WEB·API·MCP·ADMIN·LOGIN·SSO)으로 맞추는 한 줄 (가치 1 / 위험 1 / S).
