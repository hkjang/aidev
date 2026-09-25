# 회차 노트 2026-09-26-070048-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:00] base pinned — main@9c5863f
- [러너 07:00] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 골랐다: `serviceError` SQLSTATE 분류. 보류 목록 1순위였던 이 항목의 위험을 3→2 로 내린 근거를 실제로 찾았다 — `invalid_request` 문자열을 읽는 코드가 저장소에 0건이고 `web/src/api.ts:19`→`App.tsx:120` 이 `.message` 만 읽어 상태 코드 변경의 클라이언트 영향이 없다. ClientIP(신뢰 경계 미정)·DealsAtRisk(상한 계약 미정)를 제친 이유는 둘 다 결정이 먼저라 45분에 안 끝나서다.
- 이 과제의 진짜 근거는 "두 파서가 같은 값을 다르게 읽는다" 다 — MCP `sanitizeToolError` 는 SQLSTATE 를 보고 REST `serviceError` 는 영어 substring 을 본다. 그래서 브리프에 두 함수를 **합치지 말라**고 못박았다(반환 계약이 다르고 server→mcp import 라 순환).
- 추측으로 적은 것: (1) `openapi_contract_test.go` 가 오류 응답을 검사하는지 열어보지 못했다 — 구현자가 먼저 볼 것으로 적었다. (2) pgx 5.7.5 의 `pgconn.ConnectError` 존재 여부 미확인이라 연결 실패는 범위 밖으로 빼고 아이디어로만 남겼다. (3) 42501 이 실제로 `permission` substring 에 걸린다는 것은 코드 읽기 기반 추론이고 실행으로 재현하지 않았다.
- 조심할 것: SQLSTATE 분기를 기존 substring 분기 **앞**에 두는 순서가 이 과제의 핵심이자 유일한 회귀 위험이다. `customerCodeConflict`·`deleteGuarded` 가 `%w` 없이 새 오류를 만들어 사슬에 PgError 가 없다는 것은 소스에서 확인했고 수용 기준 4번으로 고정했다.
- 검증은 실제 `&pgconn.PgError` 를 실제 `serviceError` 에 넣는 것으로만 — 가짜 대역 금지, grep 증거 금지. `go test ./internal/server ./internal/mcp ./internal/crm` 는 이번 정찰에서 무변경 통과 확인(base 9c5863f).
- [러너 07:04] scout done — REST `serviceError` 가 PostgreSQL 오류를 원문 그대로 내보내는 것을 MCP 와 같은 어휘로 분류하기 (가치 4 / 위험 2

## 구현 노트
- 무엇을 왜: `serviceError` 에 SQLSTATE 분기(`pgErrorVerdict`)를 `pgx.ErrNoRows` 뒤·영어 substring 앞에 넣어 REST 가 MCP `sanitizeToolError` 와 같은 판정·같은 문장을 주게 했다. 42P01 은 400+`relation "contacts" does not exist` → 500+일반 문장, 42501 은 403+표 이름 → 500 이 되고 원문은 `s.Log.Error` 에 `sqlstate`·`requestId` 로만 남는다. 커밋 18c83f2.
- 확신 없는 곳: (1) 22P02 문장을 MCP 것 그대로 썼기 때문에 웹 UI 에 "ID는 목록·검색 도구가 돌려준 UUID를…" 라는 MCP 어투가 노출된다 — 수용 기준 2가 "그대로 쓴다"고 못박아 따랐지만, 사람이 읽는 화면으로는 어색하다. 바꾸려면 양쪽을 함께 바꿔야 한다. (2) 두 표가 어긋나도 빨개지는 테스트가 없다(주석으로만 묶임) — ideas.json 에 AST 대조 스캔으로 남겼다. (3) 400→500/409 로 바뀌는 오류가 MCP `mcp_request_logs`·idempotency 캐시(`server.go:692` 는 `<500` 만 저장)에 주는 영향은 코드로만 확인했고 실행으로 재현하지 않았다 — 이제 42P01 응답이 캐시되지 않는다(이전엔 400 이라 캐시됐다). 이것은 개선이라고 본다.
- 일부러 하지 않은 것: 두 함수 통합(반환 계약이 다르고 `internal/server`→`internal/mcp` import 방향), 영어 substring 분류를 센티널로 옮기는 일(354개 호출부, 별도 회차), 연결 실패 처리(범위 밖 — 다만 기동 로그에서 `failed SASL auth: … (SQLSTATE 28P01)` 처럼 SQLSTATE 문자열을 싣고 오는 경우를 실제로 보았고 그런 것은 이미 500 으로 접힌다), `web/`·문서·마이그레이션.
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(`go test ./internal/server/ -run TestServiceError`). 정정 하나 — 정찰 노트의 "`openapi_contract_test.go` 미확인" 은 확인했다: 그 파일은 라우트·파라미터 AST 대조만 하고 오류 본문 모양은 검사하지 않는다. 실 DB 재현은 서버를 컨테이너 안에서 돌려야 한다(`/var/lib/relio` 를 호스트 사용자가 못 만든다) 그리고 두 바이너리가 같은 DB 를 쓰려면 `ENCRYPTION_KEY` 를 줘야 한다 — 안 주면 첫 컨테이너가 만든 master.key 가 사라져 다음 기동이 "instance master key recovery required" 로 죽는다. `/healthz` 는 이 트리에서 SPA 미빌드라 500("frontend unavailable") 이므로 기동 대기 조건으로 쓰면 안 된다.
- [러너 07:23] brief accepted — 채택 — 과제서의 근거(`server.go:714` 의 substring 분기, `results.go:58` 의 SQLSTATE 표, `customerCodeConflict`·`deleteGuarded` 가 `%w` 로 PgEr
- [러너 07:23] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인함: main 의 server.go 로 되돌려 새 테스트를 실제로 돌렸다 — 17개 하위 테스트 실패(42501 403→500, 42P01 400→500, 23505/23503/40001/40P01 400→409, 22P02 등 원문 노출), HEAD 에서 전부 통과. 테스트는 진짜로 변경을 검증한다. `go build ./...`·`go vet`·`gofmt -l`·`go test ./...` 모두 깨끗. `customerCodeConflict`·`deleteGuarded` 가 `%w` 없이 새 오류를 만든다는 것, `internal/server`→`internal/mcp` 단방향 import, 두 SQLSTATE 표가 글자까지 동일하다는 것, 클라이언트가 `invalid_request` 를 읽지 않는다는 것을 소스에서 확인했다.
- 못 봤음: 실 DB·실 컨테이너 재현, 웹 빌드·프런트 테스트, 릴리즈 워크플로. 전부 이번 변경 범위 밖이라 의도적으로 생략했다.
- 승인이어도 남는 우려 ①: `server.go:724` 의 `Contains(msg,"SQLSTATE")` 가 새 함정이 되었는데 `internal/crm/cursor_test.go:63` 의 함정 목록에 따라오지 않았다 — 프로필이 경고한 그 자리이고, 새 함정만 유일하게 500 을 만든다. 다음 회차에 한 줄 추가 권장.
- 승인이어도 남는 우려 ②: 22P02·23505·23503·40001 은 이제 응답에도 로그에도 원문이 없다(MCP 는 `results.go:78` 에서 전부 Warn 으로 남긴다). REST 운영자의 사후 추적 수단이 사라졌다. ③ `openapi.go:196` 은 200/400/401/403 만 선언하는데 500 이 새로 도달 가능해졌다(404·409 는 원래도 누락).
- 릴리즈 노트에 넣을 것: INSERT 시 23503 이 409 로, 22001 같은 미등록 사용자 원인 코드가 500 으로 바뀐다. 22P02 의 MCP 어투 문장이 웹 UI 에 그대로 보인다 — 고치려면 두 파일을 함께 고쳐야 한다. 보안·법무 차단 없음(정보 노출은 오히려 줄고, 새 개인정보 수집 없음).
- [러너 07:27] review approved — 리뷰 승인 (risk=low)
- [러너 07:28] pr created — https://github.com/hkjang/relio/pull/34
- [러너 07:31] ci passed — 검사 2개 모두 success
- [러너 07:31] merge done — 18c83f2
- [러너 07:31] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
