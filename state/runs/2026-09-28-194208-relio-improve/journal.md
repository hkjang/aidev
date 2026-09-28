# 회차 노트 2026-09-28-194208-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:42] base pinned — main@fbf4233
- [러너 19:42] autonomy low-risk — 롤백 PR

## 정찰 노트
- 골랐다: 끊긴 질의(context.Canceled·DeadlineExceeded)를 400 이 아니라 500 으로 — 임시 probe 를 실제로 돌려 현재 동작(400 invalid_request + 원문)을 눈으로 확인했고, 재현에 외부 PostgreSQL 이 필요 없으며(닫힌 포트 풀 + 만료 context), 프로덕션 3파일로 끝나고, 직전 3회차가 모두 머지된 같은 자리·같은 관용이다.
- 제친 후보: net.OpError/EOF 단절(실 DB probe 없이 동적 타입 확인 불가) · substring→센티널(354 호출부라 단독 S 아님, 차선으로 남김) · serveIdempotent 4xx 캐시(docs·web·openapi 에 Idempotency 언급 0건 확인 → 계약 미정이라 가치 3→2 로 낮춤) · Frame 부제 한 줄(단독 가치 1).
- 확신 없는 곳(과제서에 "미확인" 으로 명시): REST 요청 경로에 요청별 데드라인을 거는 생산자가 오늘은 없다 — DeadlineExceeded 는 방어적 고정이고 지배적 생산자는 클라이언트 끊김의 Canceled. "idempotency 캐시가 오염된다" 를 단정하지 말 것(끊긴 요청은 INSERT 도 같은 죽은 context 로 실패한다).
- 구현자가 조심할 것: case 순서(Unreachable 이 먼저 — 다이얼 타임아웃은 ConnectError 가 DeadlineExceeded 를 감싼다) · status>=500 로깅 블록과 SQLSTATE 표는 손대지 말 것(sqlstate_parity_test.go 가 감시) · Canceled 를 500 으로 올리면 클라이언트 끊김마다 Log.Error 가 찍히는 것이 이 과제의 진짜 트레이드오프이며 받아들이기로 결정했다(499·로그레벨 분기 금지).
- 프로필은 4일 전 것이 지금 코드와 어긋나지 않아(마지막 마이그레이션 017, 구조·검증 함정 일치) 새로 쓰지 않았다. base 가 main@fbf4233 로 바뀐 것만 참고. 
- [러너 19:46] scout done — 끊긴 질의(`context.Canceled`·`context.DeadlineExceeded`)를 "잘못된 요청" 으로 판정하지 않고 REST·MCP 두 문 모두에�

## 구현 노트
- 무엇을 왜: 끊긴 질의(`context.Canceled`·`DeadlineExceeded`)가 `serviceError` 의 사슬을 모두 빠져나가 기본값 400 + 드라이버 원문으로, MCP 에서는 `sanitizeToolError` 의 `return message` 로 원문 그대로 나가던 구멍을 막았다. 새 `database.Interrupted` 하나를 `Unreachable` 바로 아래 두어 두 문이 같은 정의를 읽는다(프로덕션 3파일, SQLSTATE 표·로깅 블록 무변경, 새 import 간선 0).
- 확신 없는 곳: (a) case 순서 — 다이얼 타임아웃이 `*pgconn.ConnectError` 안에 `DeadlineExceeded` 를 감싸므로 `Unreachable` 이 먼저 잡아야 sqlstate 가 로그에 남는다고 주석에 적었지만, **그 순서를 따로 테스트하지 않았다**. 둘 다 500 `internal_error` 라 응답으로는 구분되지 않고 차이는 로그의 `sqlstate` 필드뿐이라, 재현에 블랙홀 주소가 필요한 것에 비해 얻는 게 없다고 판단했다. (b) REST 경로에 요청별 데드라인을 거는 생산자가 오늘은 없다 — `DeadlineExceeded` 쪽은 방어적 고정이고 실제 지배적 생산자는 클라이언트 끊김의 `Canceled` 다. (c) 실제 PostgreSQL·실제 끊긴 HTTP 클라이언트로 end-to-end 재현은 하지 않았다(닫힌 포트 + 죽은 context 로만 재현).
- 일부러 하지 않은 것: `serveIdempotent` 는 읽기만 했다(`< 500` 조건은 사용자 계약 결정). 499 나 로그 레벨 분기를 만들지 않았다 — 끊김마다 `Log.Error` 가 찍히는 것은 받아들이기로 한 트레이드오프다(저트래픽 에어갭 B2B CRM). 두 술어를 하나로 합치지 않았다(위 (a) 의 순서가 사라진다).
- 다음 역할이 조심할 것: 새 테스트는 DB 가 필요 없다 — 닫아 둔 포트에 풀을 만들고 이미 죽은 context 로 질의한다. 다만 포트 예약(`net.Listen` → 즉시 close)에 기대므로 극히 드물게 그 포트를 누가 가로채면 `errors.Is` 확인 줄이 "this test no longer reproduces an interrupted query" 로 먼저 `t.Fatal` 한다(오진이 아니라 재현 실패를 알리도록 일부러 그렇게 두었다). 헬퍼 이름 `deadEndPool`·`interruptedQueryError` 가 두 패키지에 같은 이름으로 중복되지만 서로 다른 패키지이고, 기존 `unreachableQueryError` 를 건드리지 않으려고 일부러 복제했다.
- [러너 19:54] brief accepted — 채택 — 과제서가 지목한 자리와 근거(`server.go:716` 의 사슬과 case 순서, `results.go:78` 의 조건, `database.go:28` 의 `Unreachable`, `u
- [러너 19:54] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인했다: 원장에 `- 실패 재현:` 이 없어 직접 재현 — 프로덕션 3파일만 main 으로 되돌리고 새 테스트 4개를 돌려 전부 실패(REST `400 invalid_request` + `"…: context canceled"`, MCP 동일 원문), 복원 후 전부 통과. 테스트가 실제 `*pgxpool.Pool` 의 에러를 쓰고 그것이 ConnectError 가 아님을 `%T` 로 확인했으니 `Unreachable` 로 새는 위통과가 아니다. `go test ./...`·`-race`(server·mcp·database)·`go vet`·`gofmt -l` 모두 깨끗.
- 못 본 것: 실제 PostgreSQL·실제 끊긴 HTTP 클라이언트 end-to-end, 프런트·docker 릴리즈 검증(이번 변경과 무관).
- 승인이어도 남는 우려 ①: `internal/mcp/interrupted_query_test.go:98-100` 주석이 "이 패키지에는 deadline 생산자가 있다" 고 하는데 `internal/mcp/server.go:271` 의 `logCtx` 는 버려지는 `mcp_request_logs` INSERT 전용이라 `sanitizeToolError` 에 닿지 않는다 — MCP 도 REST 와 같이 방어적 고정이다. 주석 한 줄 수정 대상(코드 무변경).
- 승인이어도 남는 우려 ②(릴리즈 노트): 끊김마다 `Log.Error` + 500 이 남으므로 5xx·Log.Error 알림이 클라이언트 끊김에 반응하기 시작한다. 또 `deadEndPool` 의 포트 예약이 두 패키지에 새 flake 경로를 만든다(빨개지는 방향).
- case 순서 미검증은 무해함을 확인했다: 다이얼 타임아웃의 ConnectError 는 PgError 를 감싸지 않아 `sqlstate` 가 어느 case 를 타도 비고, 둘 다 500 `internal_error` 다. 추가 테스트 불필요.
- [러너 19:58] review approved — 리뷰 승인 (risk=low)
- [러너 19:58] pr created — https://github.com/hkjang/relio/pull/37
- [러너 20:02] ci passed — 검사 2개 모두 success
- [러너 20:02] merge done — 2e2f68d
- [러너 20:02] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
