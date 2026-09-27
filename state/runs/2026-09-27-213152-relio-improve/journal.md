# 회차 노트 2026-09-27-213152-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:31] base pinned — main@997b9c2
- [러너 21:31] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 고른 이유: 최근 두 머지(18c83f2·f5064f6)가 "같은 DB 오류는 어느 문으로 들어오든 같은 답" 계약을 세웠는데 그 계약만 주석으로 묶여 있다. 프로덕션 0파일·위험 1 이고 `rows_err_test.go` 라는 go/ast 선례가 이 저장소에 이미 있다.
- 제친 후보: 보류 1순위였던 `context.DeadlineExceeded` 는 기각했다 — 요청 컨텍스트에 데드라인을 거는 자리가 아예 없어(main.go:106 은 Read/WriteTimeout 뿐, internal/server 에 WithTimeout 0건) 남는 경로가 클라이언트 취소뿐이고 그때는 읽을 상대가 없다. serveIdempotent 동시 경쟁(신규·가치 4)은 진짜 결함이지만 마이그레이션 018 과 실 DB 증명이 필요해 자율 회차 범위 밖.
- 추측으로 적은 것(구현자 주의): 차선 후보의 "핸드셰이크 뒤 단절" 이 어떤 동적 타입인지는 **미확인** — 실 PostgreSQL 없이 재현 못 했다. 그 후보를 고르면 probe 가 선행 조건이다.
- 조심할 것: 두 SQLSTATE 표의 문장을 다듬지 말 것(기존 27건이 깨진다). AST 테스트가 함수를 못 찾고 조용히 통과하는 허수가 되지 않게 `t.Fatalf` 를 둘 것.
- 프로필은 3일 전 것이 지금 코드와 맞아(마지막 마이그레이션 017 재확인, 구조 동일) 다시 쓰지 않았다. `go test ./internal/server/ ./internal/mcp/ ./internal/platform/database/` 전부 ok.
- [러너 21:37] scout done — REST `pgErrorVerdict` 와 MCP `sanitizeToolError` 의 SQLSTATE→문장 표가 어긋나면 테스트가 빨개지게 하기 (가치 3 / 위

## 구현 노트
- 과제서 그대로 채택. 새 `internal/server/sqlstate_parity_test.go` 가 `go/parser` 로 `server.go` 와 `../mcp/results.go` 를 파싱해 두 switch 의 `case "<SQLSTATE>"`→반환 문자열 리터럴 쌍을 뽑고, 코드 집합과 문장이 글자 그대로 같은지 대조한다. 프로덕션 코드 0줄 — 새 테스트를 가리키는 주석 3줄만 더했다(과제서가 허용한 선택 항목). 커밋 c81324e.
- **확신 없는 곳**: (1) 파싱 규칙 "케이스 안 마지막 `ReturnStmt` 의 마지막 `Results` 요소가 문장" 은 현재 두 함수(각 케이스에 `return` 하나)에서만 검증했다. 누군가 케이스 안에 조건 분기와 `return` 여러 개를 넣으면 마지막 것만 비교하고 앞의 것은 놓친다 — 비교가 조용히 약해지는 방향이다. (2) 숫자로 시작하지 않는 case 값은 의도적으로 건너뛰므로(과제서 수용 기준 4), 누가 `case "unknown":` 같은 비-SQLSTATE 어휘를 한쪽에만 더해도 잡히지 않는다. 둘 다 아는 한계이고 지금 표에는 해당 사례가 없다.
- **검증 못 한 것**: 실제 PostgreSQL 은 띄우지 않았다 — 이 회차는 소스 대조 테스트라 DB 가 필요 없고, 두 함수의 런타임 동작은 기존 `service_error_test.go` 27건이 이미 덮는다. 프런트·`check-static-assets.sh`·`previous-release-tag-test.sh` 는 web·릴리즈 경로 무접촉이라 실행하지 않았다(과제서 지시).
- **일부러 하지 않은 것**: 두 표의 문장을 다듬지 않았다(고정이 목적이지 개선이 아니며, 건드리면 기존 27건이 깨진다). 두 함수를 합치지 않았다(반환 계약이 다르고 import 가 단방향 — 앞선 두 회차의 명시적 결정). 보류 3순위였던 감사 화면 Frame 부제도 범위 밖이라 두었다.
- **다음 역할이 조심할 것**: 이 테스트는 **상대경로** `../mcp/results.go` 를 연다 — `internal/server` 패키지에서 도는 한 문제없고, 못 열거나 함수를 못 찾으면 `t.Fatalf` 로 죽는다(허수 방지). 두 함수의 이름을 바꾸거나 파일을 옮기면 이 테스트가 먼저 빨개지니, 그때는 테스트의 파일·함수명 상수를 같이 고칠 것. 뽑은 코드가 10개 미만이어도 `t.Fatalf` 다.
- red 는 네 섭동으로 직접 확인하고 전부 되돌렸다(문장 한 글자 / REST 에만 코드 추가 / MCP 에만 추가 / 함수명 변경·스캔 축소). 최종 `git diff` 에 섭동 흔적 없음. `go test ./...`·`go test -race ./...` 둘 다 exit 0, FAIL·DATA RACE 0.
- [러너 21:42] brief accepted — 채택 — 과제서가 지목한 두 표(`server.go:780-797`, `results.go:83-100`)와 7그룹/10코드 목록, "마지막 Results 요소가 문장" 이라는 �
- [러너 21:42] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 구현자의 red 주장을 믿지 않고 섭동 5개를 직접 재현했다: MCP 문장 한 글자(→ 문장 불일치 빨강), REST 에만 코드 추가(→ 빨강), MCP 에만 추가(→ 빨강), MCP 에서 코드 1개 삭제(→ 빨강), 함수명+호출부 전부 개명해 빌드되는 상태로 개명(→ `t.Fatalf` 빨강). 모두 되돌린 뒤 `git status` 깨끗, `go test ./...` exit 0, `go build ./...`·`go vet`·`gofmt -l` 무소음. 승인.
- 구현자가 의심한 곳 (1) 을 재현해 **확인했다**: case 안에 `if requestID == "" { return "전혀 다른 문장" }` 를 넣고 기존 `return` 을 남기면 테스트는 **초록**이다(마지막 `ReturnStmt` 만 비교). 오늘 두 표에 그런 형태는 없고 조용히 약해지는 방향이라 차단하지 않았으나, 케이스 안에 분기를 넣는 변경이 오면 이 테스트를 신뢰하지 말 것.
- 남는 우려(다음 회차): `len(table) < 10` 이 **현재 표 크기를 하한으로 못박았다**. 한쪽에서 코드가 하나 빠지면 개수 `Fatalf` 가 더 친절한 불일치 비교보다 **먼저** 터져 "the scan is no longer reaching the table" 라는 **엉뚱한 진단**을 읽게 된다(섭동 4 의 실제 출력). 양쪽에서 코드를 정당하게 줄이는 변경도 이 상수를 같이 고쳐야 한다 — 빨강 방향이라 안전하지만 메시지가 거짓말을 한다.
- 보지 않은 것: 프런트·`check-static-assets.sh`·`previous-release-tag-test.sh`·실 PostgreSQL·`go test -race`(구현자 보고만 믿음). web·릴리즈·DB 경로 무접촉이라 범위 밖으로 판단했다. 두 함수의 런타임 동작은 이번 diff 가 건드리지 않았다(프로덕션 변경은 주석 3줄).
- 보안·법무 소견 없음(차단 0): 테스트 전용, 표준 라이브러리만, 신규 경로·식별자·비밀값·개인정보 없음. 오히려 DSN·스키마 유출을 막는 기존 통제를 고정하는 방향이라 보안에 이롭다.
- [러너 21:45] review approved — 리뷰 승인 (risk=low)
- [러너 21:45] pr created — https://github.com/hkjang/relio/pull/36
- [러너 21:48] ci passed — 검사 2개 모두 success
- [러너 21:49] merge done — c81324e
- [러너 21:49] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
