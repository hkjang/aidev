# 과제서 (2026-09-27, base main@39fe52c)

- 과제: PostgreSQL 에 **닿지 못한** 오류(`failed to connect to \`user=… database=…\``)를 REST·MCP 두 문 모두에서 원문 없이 서버 오류로 접기 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/server/server.go:714 serviceError` 는 SQLSTATE 가 실린 오류만 `pgErrorVerdict` 로 분류하므로, SQLSTATE 가 없는 순수 연결 실패(`*pgconn.ConnectError` — 이름 해석 실패·connection refused·dial timeout)는 영어 substring 어디에도 걸리지 않아 **기본값 400 `invalid_request` + 드라이버 원문**으로 클라이언트에 나간다. 그 원문은 `pgconn/errors.go:68` 이 만든 ``failed to connect to `user=relio database=relio`: 10.0.0.2:5432 (db): dial error …`` 로 DB 계정명·DB 이름·내부 호스트:포트를 싣는다. 같은 오류를 MCP 쪽 `internal/mcp/results.go:64 sanitizeToolError` 는 마지막 줄 `return message`(98행)로 **원문 그대로** 모델에게 준다. DB 가 잠깐 죽었다 살아나는 흔한 상황이 (1) 내부 접속 정보 유출과 (2) "클라이언트 잘못(400)" 이라는 틀린 판정을 동시에 만든다. 500 으로 접히면 모니터링이 5xx 로 보고, 멱등성 캐시(`server.go:694` 의 `result.StatusCode < 500` 조건)도 이 응답을 저장하지 않는다.

- 수용 기준:
  1) 실제로 아무도 듣지 않는 포트를 가리키는 **진짜 `*pgxpool.Pool`** 의 질의 오류를 `serviceError` 에 넣으면 500 `internal_error` + `"서버 오류가 발생했습니다."` 가 나오고, **응답 본문 전체**에 `failed to connect`·`user=`·`database=`·포트 번호가 하나도 없다.
  2) 같은 오류를 `sanitizeToolError` 에 넣으면 원문이 아니라 그 함수가 이미 쓰는 일반 문장(`"데이터 처리 중 오류가 발생했습니다. 요청 ID: …"`)이 나오고, 원문은 `slog.Warn("mcp tool database error", …)` 로만 남는다.
  3) 원문은 REST 에서도 응답이 아니라 `s.Log.Error("service error", …)` 에만 남는다(기존 `status >= 500` 블록이 이미 한다 — 테스트가 `logs` 에 `failed to connect` 가 있음을 확인).
  4) 기존 분류가 그대로다: `internal/server/service_error_test.go` 의 27건(pgx.ErrNoRows→404, 22P02→400, 23505→409, not found/permission/already substring 3건 등)과 `go test ./...` 전부 무변경 통과.
  5) 새 분기를 지우면 테스트가 다시 빨개진다(red 확인 후 green).

- 건드릴 파일 (프로덕션 3개 + 테스트 2개):
  - `internal/platform/database/database.go` — 새 함수 `Unreachable(err error) bool`: `var connErr *pgconn.ConnectError; return errors.As(err, &connErr)`. 주석에 "PostgreSQL 이 거절한 것이 아니라 드라이버가 닿지 못한 것. `Error()` 가 DSN 조각을 싣기 때문에 클라이언트에 그대로 가면 안 된다" 를 적을 것. (이 패키지는 이미 `internal/server`·`cmd/relio` 가 import 한다. `internal/mcp` → `internal/platform/database` 는 새 간선이지만 `database` 는 `mcp` 를 import 하지 않으므로 순환 없음. `go build ./...` 로 확인.)
  - `internal/server/server.go:serviceError` — `case isPg:` **뒤**, 영어 substring case **앞**에 `case database.Unreachable(err): status = http.StatusInternalServerError; code = "internal_error"` 한 case 추가. 순서가 중요: SASL 인증 실패처럼 `ConnectError` 가 `PgError`(28P01)를 감싸는 경우는 `isPg` 가 먼저 잡아 로그에 `sqlstate=28P01` 이 남는다(둘 다 결과는 500). `msg` 는 손대지 않는다 — 기존 `if status >= 500` 블록이 로그를 남기고 문장을 `"서버 오류가 발생했습니다."` 로 바꾼다.
  - `internal/mcp/results.go:sanitizeToolError` — 74행 조건을 `if errors.As(err, &pgErr) || strings.Contains(message, "SQLSTATE") || database.Unreachable(err) {` 로 넓히기만 한다. `code` 가 빈 문자열이라 switch 를 그대로 통과해 96행의 일반 문장으로 떨어지고, 79행이 원문을 로그에 남긴다. 문장을 새로 만들지 말 것 — 53~63행 주석이 두 표를 "글자 그대로 같게" 유지하라고 못박고 있고, 이 과제는 SQLSTATE 표를 건드리지 않는다.
  - `internal/server/service_error_test.go` — 기존 헬퍼 `runServiceError(t, err)`(33행, 실제 `(*Server).serviceError` + `httptest`) 를 그대로 쓰고 케이스 1개 추가.
  - `internal/mcp/results_test.go`(신규, 또는 기존 `internal/mcp/server_test.go` 에 추가) — 같은 오류로 2)를 고정.
  - 두 테스트가 쓸 **진짜 오류 만드는 법**: `internal/intelligence/health_lookup_test.go:16 unreachablePool(t)` 를 그대로 복사한다 — `net.Listen("tcp","127.0.0.1:0")` 로 포트를 잡았다 닫고, 그 주소로 `pgxpool.ParseConfig("postgres://relio:relio@"+addr+"/relio?connect_timeout=2&sslmode=disable")` → `NewWithConfig` → `t.Cleanup(pool.Close)`. 그 풀로 `pool.Query(ctx, "SELECT 1")` 을 부른 오류가 입력이다(손으로 만든 대역 금지 — `pgconn.ConnectError` 는 `err` 필드가 비공개라 애초에 외부에서 만들 수 없다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/server/ ./internal/mcp/ ./internal/platform/database/`
  - `go build ./...` · `go vet ./...` · `go test ./...` · `go test -race ./...`
  - `gofmt -l internal/server/server.go internal/mcp/results.go internal/platform/database/database.go <테스트 파일>` (무출력이어야 함) · `git diff --check`
  - 프런트·마이그레이션 무변경이므로 web 빌드는 불필요. 시간이 남으면 `./scripts/check-env-contract.sh`.

- 위험과 피할 것:
  - **먼저 확인할 것(정찰 미확인)**: 실제 오류의 동적 타입이 `*pgconn.ConnectError` 인지. 소스로는 `pgconn/pgconn.go:154·160` 이 dial/이름해석 실패에 `&ConnectError{…}` 를 돌려주고 `pgxpool/pool.go:237-240` 이 그것을 감싸지 않고 올려보내는 것까지 읽었지만, 실제 오류 객체로는 확인하지 않았다. 첫 테스트에서 `t.Logf("%T %v", err, err)` 로 한 번 찍어 보고, 만약 `errors.As` 가 잡지 못하면 그 출력에 맞춰 판정을 고칠 것(문자열 `strings.Contains(msg,"failed to connect")` 폴백은 최후 수단이며, 쓰더라도 `errors.As` 를 먼저 둘 것).
  - 영어 substring 분류(`not found`/`permission`/`already`)를 센티널 오류로 바꾸는 작업으로 번지지 말 것 — 354개 호출부에 걸친 별건 M 이다.
  - `context.DeadlineExceeded`·`net.Error`·`*pgconn.ParseConfigError` 는 이번 범위 밖(ParseConfigError 는 기동 경로에서만 나고 요청 응답에 실리지 않는다). 아이디어로만 남길 것.
  - `pgErrorVerdict` 의 SQLSTATE 표와 `sanitizeToolError` 의 문장은 한 글자도 고치지 말 것 — 두 함수가 서로를 가리키는 주석으로만 묶여 있어 한쪽을 고치면 조용히 어긋난다.
  - 보호 경로(`internal/auth`·`internal/oidc`·`migrations/`·`.github/workflows/`)는 건드리지 않는다. 이 과제는 그럴 이유가 없다.
  - `internal/mcp/results.go` 는 한 줄이 긴 스타일이니 파일 전체 포매팅 금지.

- 차선 후보: **`pgErrorVerdict`(server.go:766) 와 `sanitizeToolError`(results.go:64) 의 SQLSTATE→문장 표가 어긋나는 것을 잡는 테스트** — 두 표는 주석으로만 묶여 있어 한쪽 문장을 바꿔도 아무 테스트도 빨개지지 않는다. `internal/server` 가 `internal/mcp` 를 import 하는 방향 때문에 함수를 노출하지 않고 대조하려면 `internal/platform/database/rows_err_test.go` 가 쓰는 관용대로 `go/ast` 로 두 switch 의 `case 코드 → 반환 문자열` 쌍을 뽑아 비교한다(가치 2 / 위험 1 / S, 프로덕션 파일 0개).
