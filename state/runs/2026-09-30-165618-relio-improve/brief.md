# 과제서 — 2026-09-30-165618-relio-improve

- 과제: 자유 텍스트 검색의 **대소문자 접기 계약**을 불변식 테스트로 묶기 — `searchPattern` 이 소문자로 내려보내는 패턴을 소문자가 아닌 컬럼에 `LIKE` 로 물리면 빨개지게 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/crm/service.go:160 searchPattern` 은 `"%" + likeEscaper.Replace(strings.ToLower(q)) + "%"` 로 **반드시 소문자** 패턴을 만든다. 그래서 이 패턴을 쓰는 비교는 왼쪽이 `lower(...)` 이거나 연산자가 `ILIKE` 여야만 맞는데, 지금 그 결합을 지키는 것은 아무것도 없다 — 누가 `lower(` 를 빼거나 새 검색을 `name LIKE $1 ESCAPE '\'` 로 쓰면 컴파일도 되고 테스트도 전부 green 인 채로 "Acme" 검색이 `%acme%` 가 되어 조용히 0건을 돌려준다(오류가 아니라 빈 목록이라 사용자에게는 "자료 없음" 으로 보인다). 오늘 10개 비교 자리는 전부 맞으므로 이번 변경은 동작을 바꾸지 않고 그 상태를 고정하는 것이 목적이다.

- 수용 기준:
  1) `go test ./internal/crm/` 가 지금 코드에서 green 이고, 기존 3개 테스트(`TestSearchPatternKeepsWildcardsLiteral`, `TestSearchPatternLeavesABlankQueryEmpty`, `TestEveryFreeTextSearchDeclaresItsEscapeCharacter`)의 동작과 이름이 그대로 남아 있다.
  2) 새 불변식이 **대소문자 접기**를 검사한다: `internal/` 전 비-테스트 `.go` 를 훑어 `LIKE` 비교마다 (a) 연산자가 `ILIKE` 이거나 (b) 왼쪽 피연산자 표현식이 `lower(` 로 시작하는지 확인하고, 아니면 `path:line` 과 함께 실패한다.
  3) 스캔이 **실제로 표를 읽고 있음**을 스스로 증명한다: 찾은 `LIKE` 비교 자리 수가 기준치(오늘 확인값 아래 "확인한 현황" 참고) 미만이면 `t.Fatalf` 로 "스캔이 더는 검색 구문에 닿지 않는다" 를 말한다. (SQLSTATE 대조 테스트 `internal/server/sqlstate_parity_test.go` 가 이미 쓰는 관용을 그대로 본뜰 것.)
  4) 섭동 3종을 실제로 눈으로 보고 되돌린다 — ① `internal/crm/resources.go:82` 의 `lower(name) LIKE $1` 을 `name LIKE $1` 로 바꾸면 그 줄만 빨개진다. ② `internal/server/admin_operations.go:99` 의 `ILIKE` 하나를 `LIKE` 로 바꾸면 그 줄만 빨개진다(왼쪽이 `COALESCE(actor_name,'')` 라 `lower(` 로 시작하지 않는다). ③ 기존 ESCAPE 검사도 함께 살아 있음 — 아무 자리에서 `ESCAPE '\'` 를 지우면 기존 메시지로 빨개진다.
  5) 프로덕션 동작 코드 변경 0줄. (주석은 허용 — `searchPattern` 독스트링에 "이 계약은 <새 테스트 이름> 이 지킨다" 한 줄까지.)

- 확인한 현황 (이번 정찰이 실제로 열어 본 것):
  - `internal/crm/service.go:160-168` — `searchPattern` + `likeEscaper`. 빈 질의는 `""`.
  - `internal/crm/service.go:172` — `SearchPattern` 공개 래퍼.
  - **`lower(...) LIKE $n ESCAPE '\'` 자리 (전부 올바름)**: `internal/crm/resources.go:30`(2개: `lower(l.name)`, `lower(COALESCE(l.company,''))`), `internal/crm/resources.go:82`(2개: `lower(name)`, `lower(code)`), `internal/crm/advanced.go:21`(2개: `lower(ct.name)`, `lower(COALESCE(ct.email,''))`), `internal/crm/service.go:206`(3개: `lower(c.name)`, `lower(COALESCE(c.registration_no,''))`, `lower(COALESCE(c.customer_code,''))`), `internal/crm/service.go:556`(2개: `lower(o.name)`, `lower(c.name)`), `internal/relationship/team.go:166`(2개: `lower(u.display_name)`, `lower(u.username)`).
  - **`fmt.Sprintf` 로 조립되는 자리**: `internal/voice/knowledge.go:167` 이 `fmt.Sprintf(`+"`"+`%s LIKE $%d ESCAPE '\'`+"`"+`, col, n)`. `col` 은 :159-165 의 map 키 5개이고 전부 `lower(` 로 시작한다(`lower(v.title)`, `lower(COALESCE((SELECT string_agg(value,' ') …)),'')`, `lower(COALESCE(v.root_cause,''))`, `lower(COALESCE(v.body,''))`, `lower(COALESCE(v.resolution,''))`). **리터럴 한 줄에는 왼쪽 피연산자가 없으므로** 여기는 소스 한 줄만 보는 방식으로는 판정할 수 없다 — 구현자는 이 자리를 (a) 화이트리스트가 아니라 "`%s LIKE` 형태는 map 키 쪽을 함께 검사" 하거나 (b) 정직하게 스캔 범위 밖으로 빼고 그 이유를 테스트 주석에 남기는 것 중 하나를 고를 것. `rows_err_test.go` 가 `internal/intelligence` 를 정직하게 뺀 전례가 있다.
  - **`ILIKE` 자리**: `internal/server/admin_operations.go:99` 에 5개(`COALESCE(actor_name,'')`, `action`, `resource`, `COALESCE(resource_id,'')`, `COALESCE(request_id,'')`). 여기만 `lower(` 없이 맞다 — `ILIKE` 가 대소문자를 무시하기 때문. 새 규칙은 이 5개를 반드시 통과시켜야 한다.
  - 즉 오늘 기준: `lower(...) LIKE` 13개(리터럴) + `ILIKE` 5개 + `fmt.Sprintf` 조립 1개. 기준치는 구현자가 실제 스캔 결과로 다시 세되, **오늘 값보다 낮게 잡지 말 것**.
  - 기존 테스트 `internal/crm/search_test.go:43-65` 는 줄 단위 substring 스캔이다: `internal/` 를 `filepath.Walk` 하고 `_test.go` 를 건너뛰며, 한 줄이 `"LIKE '"`/`"LIKE $"`/`"ILIKE '"`/`"ILIKE $"` 중 하나를 포함하는데 같은 줄에 `ESCAPE` 가 없으면 `t.Errorf`. 새 검사는 이 옆에 **더하는 것**이지 이것을 지우는 것이 아니다.

- 건드릴 파일 (총 1~2개):
  - `internal/crm/search_test.go` — 새 테스트 함수 1개 추가(예: `TestEveryFreeTextSearchComparesAgainstFoldedText`). 기존 3개 함수는 손대지 말 것.
  - (선택) `internal/crm/service.go:155-159` — `searchPattern` 주석에 새 테스트 이름을 가리키는 한 줄.

- 검증 명령 (이 저장소에서 실제로 도는 것, 정찰이 확인함):
  - `go test ./internal/crm/ -run 'TestSearchPattern|TestEveryFreeText' -v` — 오늘 3건 PASS, 0.009s.
  - `go test ./...` / `go test -race ./...` / `go vet ./...` / `go build ./...`
  - `gofmt -l internal/crm/search_test.go internal/crm/service.go` (무출력)
  - `git diff --check` (무출력)
  - 섭동 되돌리기는 `git checkout -- <path>` 로 하고 **최종 `git diff` 에 남지 않았음을 눈으로 확인**할 것.

- 위험과 피할 것:
  - **프로덕션 SQL 을 "고치지" 말 것.** 오늘 18개 비교는 전부 맞다. 이번 회차는 불변식만 추가한다. 검색 동작을 바꾸면 실 DB 없이 증명할 수 없고 회차 범위를 벗어난다.
  - `ILIKE` 를 `lower(...) LIKE` 로 통일하고 싶어질 수 있다 — 하지 말 것. `admin_operations.go` 는 감사 경로이고 인덱스 계획이 바뀐다.
  - `searchPattern` 의 `strings.ToLower` 를 걷어내는 방향(패턴을 원문 그대로 두고 SQL 쪽에서만 접기)은 **범위 밖**이다. 13개 호출부와 실 DB 검증이 필요하다.
  - 보호 경로(`internal/auth`·`internal/oidc`·`migrations/`·`.github/workflows/`)는 이번 과제에 전혀 등장하지 않는다. 들어갔다면 범위를 벗어난 것이다.
  - 운영자 규칙: **소스 문자열 검사를 "이 경로가 돈다" 는 증거로 제출하지 말 것.** 이 테스트는 "코드가 이렇게 쓰여 있다" 는 불변식이지 런타임 증명이 아니다 — 커밋 메시지와 테스트 주석에 그렇게 정직하게 적을 것. 실제 SQL 동작 증명은 실 PostgreSQL 이 필요하고 이번 범위가 아니다.
  - 과거 교훈: 릴리즈/빌드 경로는 건드리지 않는다. 이번 변경은 테스트 파일 하나라 릴리즈 영향이 없다.

- 차선 후보: **감사 화면 Frame 부제를 실제 채널값에 맞추기** (가치 1 / 위험 1 / S). `web/src/pages/AdminPages.tsx:619` 의 부제는 `"Web, API, MCP, Admin, Login과 Key 작업을 …"` 인데, 같은 화면 `:620` 의 채널 드롭다운은 `['WEB','API','MCP','ADMIN','LOGIN','SSO']` 이다 — **`Key` 는 존재하지 않는 채널이고 `SSO` 가 빠져 있다.** 부제 한 문자열만 고치고 드롭다운 배열을 정본으로 삼을 것. 검증: `npm --prefix web run typecheck` · `npm --prefix web test` · `npm --prefix web run build`. (주의: `AdminPages.tsx` 는 한 줄이 매우 길다 — 대규모 재포매팅 금지, 문자열 하나만 바꿀 것. 정찰 환경에서 `npm ci` 가 ETIMEDOUT 이었으므로 구현자는 먼저 의존성 설치가 되는지 확인할 것.)
