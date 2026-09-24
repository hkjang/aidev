# 정찰 과제서 (2026-09-24)

- 과제: 보고 단위(일별/월별/분기별/연도별) 시간 표현을 결정적 순서로 내보내 `analyze_question`의 `aggregation_level` 무작위화를 없애기 (가치 4 / 위험 1 / 작업량 S)

- 왜: `internal/catalog/timeparse.go:114`의 `for expr, gran := range map[string]string{"일별":"day","월별":"month","분기별":"quarter","연도별":"year","년도별":"year"}` 는 Go의 무작위 map 순회로 TimeRange를 append하므로, 질문이 보고 단위를 둘 이상 말하면 `time_ranges` 배열 순서가 실행마다 달라진다. `internal/catalog/analyze.go:213-214`가 이 슬라이스를 마지막 값 우선(last-wins)으로 읽어 `aggregation_level`(analyze.go:295)을 정하기 때문에, 같은 질문·같은 카탈로그인데 MCP 도구 `analyze_question`/`prepare_sql_context`의 응답이 회차마다 day/month/year로 바뀐다. 고정 순서를 주면 같은 질문이 항상 같은 그룹 단위를 돌려주고, 골든 평가·재현 테스트·사용자 재질의가 안정된다.
  - 실측(이번 정찰, 임시 테스트를 작성해 실행 후 삭제, 트리 clean): 질문 `"연도별 월별 일별 신규 가입자 수"`를 200회 파싱하면 보고 단위 순서가 3종으로 갈리고(`일별,월별,연도별` 124 / `연도별,일별,월별` 50 / `월별,연도별,일별` 26), 같은 질문의 `AnalyzeQuestion` 200회에서 `aggregation_level`이 `year` 115 / `month` 57 / `day` 28로 나왔다. 빈 `&Catalog{}`로도 재현되므로 픽스처가 필요 없다.

- 수용 기준:
  1) `ParseTimeExpressions`가 보고 단위 TimeRange를 **질문에 나타난 위치 순서**(동률이면 `일별,월별,분기별,연도별,년도별` 고정 순)로 내보낸다. 같은 입력을 여러 번 호출하면 `time_ranges` 전체가 바이트 동일하다.
  2) `AnalyzeQuestion`(→ `analyze_question`, `prepare_sql_context`)의 `aggregation_level`이 같은 질문에 대해 항상 같은 값이 된다. last-wins를 유지하므로 결과는 "질문에서 마지막으로 언급된 보고 단위"이며, 이 규칙을 `timeparse.go`의 해당 루프 주석과 `analyze.go:213` 근처 주석에 한 줄로 적는다. **`analyze.go`의 선택 로직 자체는 바꾸지 않는다.**
  3) 테스트가 증명할 것: (a) 보고 단위 2개 이상을 포함한 질문을 최소 50회 반복 파싱해도 순서가 한 가지뿐임(수정 전에는 실패해야 함 — 먼저 red 확인), (b) 위치 순서 규칙이 맞음(예: `"월별 연도별 …"` → 마지막이 `연도별`, `"연도별 월별 …"` → 마지막이 `월별`), (c) 실제 전송 경로 회귀: `internal/mcp`에서 `newFixtureServer`(internal/mcp/datasets_test.go:15)로 서버를 띄우고 `POST /mcp`에 `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"analyze_question","arguments":{"question":"연도별 월별 일별 신규 가입자 수"}}}`를 20회 이상 보내 `aggregation_level`이 매번 동일함(authhttp_test.go:279의 `doReq` 패턴 참고). 보고 단위가 하나뿐인 기존 질문의 출력은 수정 전후 동일해야 한다.

- 건드릴 파일:
  - `internal/catalog/timeparse.go:114-118` — map 리터럴을 `[]struct{expr, gran string}` 고정 슬라이스로 바꾸고, `strings.Index(q, expr)`로 등장 위치를 구해 위치 오름차순(동률은 슬라이스 순)으로 `add(...)`. `Expression`/`Granularity`/`Note`("GROUP BY 보고 단위") 값은 지금과 동일하게 유지.
  - `internal/catalog/analyze.go:213` 근처 — 주석 한 줄만(마지막 언급 보고 단위를 쓴다는 계약 명시). 로직 변경 금지.
  - `internal/catalog/eval_test.go` 또는 신규 `internal/catalog/timeparse_order_test.go` — 위 (a)(b) 표 테스트.
  - `internal/mcp/` 신규 테스트 파일(예: `analyze_order_test.go`) — 위 (c) end-to-end 회귀.

- 검증 명령:
  - `go test ./internal/catalog -run 'TimeExpressions|TimeOrder|Granularity' -count=1`
  - `go test ./internal/mcp -run 'Analyze' -count=1`
  - `go build ./... && go vet ./... && go test ./...` (catalog 전체는 약 60초, mcp는 약 10초)
  - `git diff --check`

- 위험과 피할 것:
  - `internal/catalog/analyze.go`의 intent/dimension 추론, `aggregation_level` 폴백(analyze.go:251-260)은 손대지 말 것. 범위를 넓히면 골든 평가 임계값(`eval_test.go:25-42`, table 0.85 / column recall 0.8 / metric 1.0 / join 0.85)이 흔들린다.
  - `ParseTimeExpressions`의 다른 분기(오늘/지난달/최근 N개월/YYYY년 M월/반기/raw 날짜)는 이번 과제 밖. 특히 과거 회차에서 성공했던 "raw 숫자 날짜 달력 검증"과 "실행 경로 SQL trailing 정리(TrimStatement)"는 현재 pinned HEAD(e9f3fb2)에 들어 있지 않지만 **같은 접근 재시도 금지**(프로필 지시).
  - 보호 경로(`internal/mcp/auth.go`, `oauth.go`, `authapi.go`, `execguard.go`, `internal/oracle/sqlguard.go`, `internal/meta/pg.go`)는 건드리지 말 것. 이번 과제는 이 경로들과 무관하다.
  - 같은 값을 읽는 경로가 여럿이다: `resolve_time`(timeparse.go:197), `analyze_question`/`prepare_sql_context`(analyze.go:201), `search.go:406`, `clarify.go:288`. 네 곳 모두 `ParseTimeExpressions` 한 곳에서 읽으므로 **파서만** 고치고, 호출부에서 정렬을 다시 하지 말 것. 최소 `analyze_question`은 실제 `/mcp` 전송 경로로 end-to-end 확인할 것(손으로 만든 대역 금지).
  - 테스트에 `data/kcb/*`를 쓰거나 덮어쓰지 말 것. 카탈로그가 필요하면 `t.TempDir` JSON + `catalog.Load`(mcp는 `newFixtureServer`).
  - 반복 횟수를 너무 크게 잡아 테스트를 느리게 만들지 말 것(50~200회면 충분; 위 실측은 200회에서 0.01초).

- 차선 후보: `docs/development.md`의 Go 버전(문서 1.24+ vs go.mod 1.25.0)·"외부 의존성 0/go.sum 없음"(실제 godror·pgx·x/crypto 3개, go.sum 존재)·MCP 도구 개수를 소스 기준으로 갱신 (가치 3 / 위험 1 / 작업량 S). 1순위가 이미 고쳐져 있거나 순서 규칙 합의가 안 되면 이것을 하되, 도구 개수는 `internal/mcp/server.go`의 `tools()` 등록 목록을 직접 세어 쓸 것.
