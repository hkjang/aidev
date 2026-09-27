- 과제: REST `pgErrorVerdict` 와 MCP `sanitizeToolError` 의 SQLSTATE→문장 표가 어긋나면 테스트가 빨개지게 하기 (가치 3 / 위험 1 / 작업량 S)

- 왜: 두 표는 "한쪽을 바꾸면 다른 쪽도 바꾸라" 는 **주석으로만** 묶여 있고(`internal/server/server.go:770-777`, `internal/mcp/results.go:55-63`), 한쪽 문장을 고치거나 한쪽에만 SQLSTATE 를 더해도 지금은 아무 테스트도 실패하지 않는다. 2026-09-26(18c83f2)·2026-09-27(f5064f6) 두 회차가 "같은 DB 오류가 어느 문으로 들어오든 같은 답" 이라는 계약을 만들어 놓고 그 계약만 무방비다. 프로덕션 파일 0개로 그 계약을 고정한다.

- 수용 기준:
  1) 새 테스트가 두 함수의 `switch` 에서 `case "<SQLSTATE>"…:` → 반환 문자열 리터럴 쌍을 뽑아, **양쪽이 다루는 코드 집합이 같고** 각 코드의 문장이 **글자 그대로 같음**을 검사한다. 지금 코드에서는 green (아래 "현재 표" 7그룹/10코드가 이미 일치함 — 이번 정찰이 두 소스를 직접 읽어 확인).
  2) red 를 실제로 본다: 한쪽 표의 문장 하나(예: MCP 의 `22003`)를 한 글자 바꾸면 그 코드만 짚어 실패하고, 한쪽에만 새 `case "40003"` 을 더하면 "한쪽에만 있다" 로 실패한다. 확인 뒤 되돌린다.
  3) 테스트가 **허수가 되지 않는다**: `pgErrorVerdict` 또는 `sanitizeToolError` 를 찾지 못하거나 뽑은 case 가 0개면 `t.Fatalf` 로 실패한다(함수명이 바뀌었는데 조용히 통과하는 것이 이런 AST 테스트의 전형적인 함정이다). 뽑은 코드 개수가 10 미만이면 실패하게 하는 것도 같은 목적.
  4) 두 함수의 **의도적 차이**는 검사하지 않는다 — REST 는 `(status, code, message)` 3값, MCP 는 문자열 1값이며, switch **밖**의 폴백(`return http.StatusInternalServerError, "internal_error", ""` / `"데이터 처리 중 오류가 발생했습니다. 요청 ID: " + requestID`)은 서로 다른 것이 정상이므로 비교 대상에서 빠져야 한다. 숫자로 시작하는 `case` 값만 모으면 자연히 빠진다.
  5) 프로덕션 코드 변경 0줄. `go test ./...` 와 `go test -race ./...` 가 이전과 같이 통과한다.

- 현재 표 (이번 정찰이 `internal/server/server.go:780-797` 와 `internal/mcp/results.go:83-100` 을 직접 읽어 대조 — 전부 일치):
  - `22P02` → "입력 값의 형식이 올바르지 않습니다. ID는 목록·검색 도구가 돌려준 UUID를 그대로 사용하세요."
  - `22007`,`22008` → "날짜 형식이 올바르지 않습니다. YYYY-MM-DD 형식을 사용하세요."
  - `22003` → "숫자가 허용 범위를 벗어났습니다."
  - `23502`,`23514` → "필수 값이 비었거나 허용되지 않는 값입니다."
  - `23505` → "같은 값이 이미 등록되어 있습니다. 기존 데이터를 조회해 수정하세요."
  - `23503` → "연결 대상이 없거나 다른 데이터가 참조하고 있어 처리할 수 없습니다."
  - `40001`,`40P01` → "동시에 처리된 다른 요청과 충돌했습니다. 잠시 후 다시 시도하세요."
  - 주의: REST 는 `22P02`/`22007`/`22008`/`22003`/`23502`/`23514` 를 400, `23505`/`23503`/`40001`/`40P01` 을 409 로 낸다. MCP 는 상태 개념이 없으므로 **상태·code 는 비교 대상이 아니다**(수용 기준 4).

- 건드릴 파일 (신규 1개, 프로덕션 0개):
  - `internal/server/sqlstate_parity_test.go` (신규, `package server`) — `go/parser` 로 같은 디렉터리의 `server.go` 와 `../mcp/results.go` 를 파싱해 두 함수의 switch 를 훑고 대조. `internal/server` 는 이미 `internal/mcp` 를 import 하지만, 이 테스트는 **소스를 파싱**하므로 import 방향과 무관하고 `sanitizeToolError` 가 비공개인 것도 문제되지 않는다.
  - 관용은 `internal/platform/database/rows_err_test.go` 를 그대로 본뜰 것 — `go/ast`·`go/parser`·`go/token` 로 트리를 걷고, 실패 메시지에 `path:line` 을 찍고, 상단 주석에 이 불변식이 왜 있는지 적는 형식이 이 저장소의 관용이다(2026-09-23·09-24 두 회차가 이 형식으로 머지됐다).
  - 파싱 요령: `*ast.FuncDecl` 로 함수를 찾고 → 본문의 `*ast.SwitchStmt` → 각 `*ast.CaseClause` 의 `List` 를 `*ast.BasicLit`(문자열)로 읽어 코드 집합을 만들고 → 그 절 안의 `*ast.ReturnStmt` 의 **마지막** `Results` 요소가 `*ast.BasicLit` 문자열이면 문장으로 삼는다(REST 는 3값 중 마지막, MCP 는 1값 중 마지막이라 "마지막" 한 규칙으로 둘 다 읽힌다). `strconv.Unquote` 로 따옴표를 벗길 것.
  - `sanitizeToolError` 안에는 switch 앞에도 `return` 이 여럿 있으므로(`arg.message`, `pgx.ErrNoRows` 문장) 반드시 **switch 안쪽만** 훑을 것. 비-리터럴 반환(문자열 연결 등)은 문장으로 세지 말 것.
  - 두 프로덕션 파일에 "이 불변식은 <새 테스트 파일>이 지킨다" 는 한 줄을 기존 주석에 덧붙이는 것까지는 허용(선택). 그 외 프로덕션 변경 금지.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/server/` (이번 정찰에서 `ok 0.169s` 확인)
  - `go test ./...` · `go test -race ./...` · `go vet ./...` · `go build ./...`
  - `gofmt -l internal/server/sqlstate_parity_test.go` (무출력) · `git diff --check`
  - 프런트·`check-static-assets.sh`·`previous-release-tag-test.sh` 는 이 과제와 무관하므로 불필요(웹·릴리즈 경로 무접촉).

- 위험과 피할 것:
  - **표의 문장을 "개선" 하지 말 것.** 이번 과제는 두 표를 *고정* 하는 것이지 다듬는 것이 아니다. 문장을 바꾸면 2026-09-26 의 `service_error_test.go` 27건이 깨지고 과제 범위를 벗어난다.
  - **두 함수를 합치려 하지 말 것.** 반환 계약이 다르고(3값 vs 문자열) `internal/server` → `internal/mcp` 단방향 import 라 이미 두 회차가 "합치지 않는다" 를 명시적 결정으로 남겼다(`results.go:60-63` 주석). 운영자 지침 "계약이 다른 파서를 통합하려 하지 말 것" 과도 같은 방향.
  - **grep 을 증거로 제출하지 말 것** — AST 로 파싱해 실제 구조에서 뽑고, red 를 눈으로 본 뒤 되돌리는 것까지가 증거다(운영자 지침).
  - 보호 경로 무접촉: `internal/auth`·`internal/oidc`·`migrations/`·`.github/workflows/` 를 건드리지 않는다. `web/` 도 건드리지 않으므로 embed 앵커(`internal/webui/dist/README`) 문제도 발생하지 않는다.
  - `rows_err_test.go` 처럼 경로를 상대경로로 여는 테스트는 **작업 디렉터리 가정**이 깨지기 쉽다. `internal/server` 패키지에서 `../mcp/results.go` 를 열고, 못 열면 `t.Fatalf` 로 죽일 것(수용 기준 3).

- 차선 후보: **연결이 성립한 뒤 끊긴 질의 오류(PostgreSQL 재시작·네트워크 단절)도 원문 없이 500 으로 접기** — `database.Unreachable` 은 `*pgconn.ConnectError` 만 본다(`internal/platform/database/database.go:28`). 핸드셰이크 뒤에 끊기면 ConnectError 도 SQLSTATE 도 없어 기본값 400 + 드라이버 원문으로 나가고 그 원문의 `net.OpError` 가 내부 host:port 를 싣는다. **미확인**: 그 경우 pgx v5 가 정확히 어떤 동적 타입을 돌려주는지 이번 정찰에서 확인하지 못했다(실제 PostgreSQL 없이는 핸드셰이크 뒤 단절을 만들 수 없음). 이 후보를 고른다면 **먼저 폐기용 PostgreSQL 17 컨테이너로 타입을 probe 해 확인한 뒤**(2026-09-27 회차가 `*pgconn.ConnectError` 를 그렇게 확인했다) 문자열 매칭이 아니라 `errors.As` 로 잡을 것. 확인 못 하면 손대지 말 것.
  - 3순위: 보류 목록의 `[1/1/S]` 감사 화면 Frame 부제를 실제 6개 채널값(WEB·API·MCP·ADMIN·LOGIN·SSO)으로 맞추기 — 한 줄이지만 확실하다.
