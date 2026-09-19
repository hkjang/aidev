# 과제서 2026-09-20 — vibe-coders

- 과제: `executeReadOnlyQuery` 의 statement timeout 을 pgx 뿐 아니라 MySQL·SQLite(기본) 경로에도 적용 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/proxy/text2sql_handler.go:1183` `executeReadOnlyQuery` 는 `TEXT2SQL_STATEMENT_TIMEOUT`(기본 15s, `config.go:556`) 을 `case "pgx"` 에서만 `SET LOCAL statement_timeout` 으로 적용하고, `case "mysql"` 은 read-only 트랜잭션만 열고, `default`(sqlite 등) 는 `db.QueryContext` 만 부르며, 세 경로 모두 함수 첫 줄의 고정 `context.WithTimeout(ctx, 30*time.Second)` 에만 기댄다. 그래서 MySQL·SQLite 배포에서는 관리자가 timeout 을 2초로 줄여도 폭주 쿼리가 30초를 점유하고, 진단 화면(`text2sql_handler.go:1031` `"statement_timeout"`)은 적용 중인 것처럼 값을 내보낸다. 고치면 세 드라이버가 같은 설정을 같게 읽는다(운영자 규칙: "같은 값을 읽는 경로가 여럿이면 모든 경로가 같게 읽는지 end-to-end 로 검증").

- 수용 기준:
  1) `stmtTimeout > 0` 이면 드라이버와 무관하게 쿼리 컨텍스트 데드라인이 `min(30s, stmtTimeout)` 이 된다(pgx 의 `SET LOCAL statement_timeout` 은 그대로 유지 — 서버 측 취소가 더 정확하므로 제거하지 말 것). `stmtTimeout == 0` 이면 지금처럼 30초.
  2) 실제 SQLite(modernc, `sql.Open("sqlite", …)` — `admin_traces_test.go:125` 가 쓰는 방식)에서 오래 걸리는 쿼리(예: `WITH RECURSIVE c(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM c WHERE x < 200000000) SELECT count(*) FROM c`)를 `stmtTimeout=200ms` 로 `executeReadOnlyQuery` 에 넣으면 대략 1초 안에 오류(`context.DeadlineExceeded` 또는 드라이버의 interrupted 오류)로 돌아온다. 같은 쿼리를 짧게(`x < 1000`) 바꾸면 `stmtTimeout=200ms` 에서도 성공하고 `count`·`rowLimit` 동작은 변하지 않는다.
  3) 테스트가 증명할 것: 프로덕션 함수 `executeReadOnlyQuery` 를 진짜 `*sql.DB`(sqlite) 로 통과시켜 (a) 느린 쿼리가 timeout 에 걸림 (b) 빠른 쿼리는 통과 (c) `stmtTimeout=0` 이면 빠른 쿼리 통과 — 대역·소스 문자열 검사 금지. 수정 전 코드에서 (a) 가 실패(30초 대기 → 결과 반환)함을 한 번 확인하고 적을 것. MySQL·pgx 실서버는 이 환경에 없으므로 실행 테스트 불가 — 컨텍스트 데드라인 경로가 드라이버 분기 **앞** 에 있으면 세 경로가 같은 코드를 지나므로 sqlite 테스트가 대표한다고 과제서에 적되, "MySQL 실서버 미확인" 이라고 명시.
  4) `go vet`·gofmt 깨끗, 기존 `go test ./internal/proxy` 전부 통과.

- 건드릴 파일:
  - `internal/proxy/text2sql_handler.go:1183-1213` `executeReadOnlyQuery` — 첫 줄의 `context.WithTimeout(ctx, 30*time.Second)` 를 `stmtTimeout > 0 && stmtTimeout < 30s` 이면 `stmtTimeout` 으로 잡는 헬퍼(예: `queryDeadline(stmtTimeout)`)로 바꿈. `case "mysql"` 에 선택적으로 `tx.ExecContext(ctx, "SET SESSION MAX_EXECUTION_TIME = <ms>")` 를 넣을 수 있지만, 그 세션 변수는 풀에 반환된 연결에 남으므로 **넣지 말고** 컨텍스트 데드라인만으로 통일할 것(go-sql-driver/mysql 은 ctx 만료 시 연결을 끊어 쿼리를 종료함). 함수 상단 주석(1179행)에 세 드라이버에 공통 적용됨을 한 줄 적기.
  - `internal/proxy/text2sql_exec_timeout_test.go`(신규) — 위 수용 기준 3 의 테스트 3개. 임시 파일 경로 `t.TempDir()` + `sql.Open("sqlite", path)`, driver 인자는 `""`(→ `normalizeExecDriver` 가 `"sqlite"`) 와 `"sqlite"` 둘 다 한 번씩.
  - `internal/config/config.go:333` — 필드 주석 `(postgres execute) per-statement timeout` 을 `per-statement timeout (all execute drivers; postgres also sets statement_timeout)` 로 고침(주석만).
  - `docs/ADMIN_GUIDE.md` — `TEXT2SQL_STATEMENT_TIMEOUT` 행이 "postgres 전용" 이라 적혀 있으면 한 줄 수정(미확인: grep 으로 먼저 볼 것. 없으면 손대지 않음).

- 검증 명령:
  - `gofmt -l internal/ && go vet ./internal/proxy/`
  - `go test ./internal/proxy -run 'ReadOnlyQuery|Text2SQL' -count=1`
  - `go test ./... -count=1` (약 1~3분, 12 패키지)
  - `go test -race ./internal/proxy -run 'ReadOnlyQuery' -count=1`
  - `go run ./cmd/api-surface-audit` 이 있으면 FAIL 0 확인(이전 회차들이 썼음; API 표면 변화 없어야 함)
  - 프런트엔드·OpenAPI 는 손대지 않으므로 pnpm 검증 불필요(러너가 자동으로 돌리면 통과해야 함 — `web/node_modules` 없으면 `cd web && corepack pnpm install --frozen-lockfile`).

- 위험과 피할 것:
  - `admin_text2sql.go:1639,1643`(골든 twin 비교)·`text2sql_scheduler.go:70`(예약 리포트) 도 같은 함수를 부르므로 시그니처를 바꾸지 말 것 — 데드라인 계산은 함수 안에서만.
  - pgx 경로의 `SET LOCAL statement_timeout`·`work_mem` 은 그대로 두고 그 위에 데드라인만 겹칠 것. 두 값이 다르게 읽히는 새 경로를 만들지 말 것.
  - `stmtTimeout` 이 30초보다 크면 지금처럼 30초 유지(상한을 넓히지 말 것 — 효과 범위 확장은 이 과제 밖).
  - 느린 쿼리 테스트의 숫자(`x < 200000000`)는 CI 러너에서도 200ms 를 확실히 넘기게 넉넉히 잡되, 성공 경로 테스트가 느려지지 않게 짧은 쿼리는 따로 둘 것. 테스트 전체 상한을 `time.Since` 로 재서 5초 넘으면 실패시키면 회귀(데드라인 미적용 → 30초 대기)를 빨리 잡는다.
  - auth·migrations·workflows 는 건드리지 않음. `.env.example` 수정 금지(이 저장소 규칙).
  - 이 브랜치(master@d67267b) 에는 mail·MCP SSO·가격 prefix·`run-tool.mjs` 커밋이 없다. 러너 verify 가 `pnpm run typecheck --silent` 를 만들면 09-18 회차처럼 실패할 수 있는데, 그 수정(`auto/2026-09-17-2153` 964d2b0) 은 이 과제 밖 — Go 전용 변경이므로 러너가 web 검증을 돌리는지 먼저 verify 로그로 확인하고, 필요하면 09-18 회차처럼 964d2b0 를 cherry-pick 하는 것을 허용(같은 판단이 한 번 통과했음).

- 차선 후보: `audit.InferLanguages.addSignal`(`internal/audit/language.go:159`) 이 더 높은 신뢰도 신호가 오면 기존 `Evidence` 를 덮어써 근거가 유실됨 — 갱신 시 `existing.Evidence + "; " + new` 로 이어 붙이고 회귀 테스트 1개 (가치 2 / 위험 1 / S). 1순위가 sqlite 에서 timeout 을 재현하지 못하는 등 성립하지 않을 때 고를 것.
