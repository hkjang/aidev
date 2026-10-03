- 과제: OpenAPI 가 공표하는 정수 질의 파라미터의 **범위·기본값**을 핸들러의 `httpx.IntQuery`/`ClampQuery` 리터럴과 대조하는 불변식 테스트 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/server/openapi_contract_test.go` 는 질의 파라미터의 **이름**만 양방향으로 대조하고(`TestDocumentedQueryParametersMatchTheHandler`) `sort` 의 enum 만 질의 빌더와 맞춘다. 정수 파라미터의 `minimum`/`maximum`/`default` 는 `internal/api/parameters.go` 의 `number()`/`limit()` 호출과 핸들러의 `httpx.IntQuery(r, key, fallback, min, max)` 두 곳에 각각 손으로 적혀 있고 둘을 묶는 것이 아무것도 없다. 한쪽만 고치면 컴파일·테스트가 전부 green 인 채로 공표 스키마가 거짓말을 한다(생성 클라이언트가 유효한 값을 거부하거나, 서버가 조용히 fallback 으로 바꿔 **다른 수**를 답한다 — `httpx.go:96-101` 이 바로 이 사고를 기록해 둔 자리다).
- 수용 기준:
  1) `internal/server/` 에 새 테스트가 생겨, `api.OpenAPI()` 가 실제로 내보내는 문서의 모든 `type: integer` 질의 파라미터에 대해 그 operation 을 답하는 핸들러가 같은 키를 읽는 `httpx.IntQuery`/`ClampQuery` 호출의 `min`/`max` 리터럴과 문서의 `minimum`/`maximum` 이 **같음**을 확인한다. 어긋나면 `operation: 파라미터 — 문서 n..m, 핸들러 a..b` 꼴로 어느 쪽이 무엇인지 찍는다.
  2) 기본값 규칙도 같이 고정한다 — `parameters.go:69 number()` 는 `fallback` 이 `[low, high]` 안일 때만 `default` 를 넣는다. 테스트는 (a) 범위 안이면 `default == fallback`, (b) 범위 밖이면 `default` 키가 **없음**을 확인한다. `versionFilter()`(fallback -1, 0..1_000_000)와 `year()`(fallback 0, 2000..2200)가 (b) 쪽 산 증인이고 `GET /voices/export` 의 `limit(200, 200)` 이 (a) 쪽 경계다.
  3) 스캔이 표에 닿고 있음을 개수 가드로 증명한다 — 파일 수준 상수(`knownIntQueries`, 이 저장소 관용은 `sqlstate_parity_test.go`·`crm/search_test.go`)보다 적게 읽혔으면 `t.Fatalf` 로 "스캐너가 깨졌다" 고 말한다. **이번 정찰에서 실측한 수는 38개**다(`grep -ro` 로 셈, 아래 검증 명령으로 재확인할 것). 키별 분포도 실측했다 — `limit` 24, `days` 3, `months` 2, `year` 2, `version` 2, `minAgeDays` 2, `expiringDays` 1, `minScore` 1, `minimum` 1 = 38. 파일별 분포는 `crm.go` 9줄, `crm_intelligence.go` 5, `relationship.go` 5, `resources.go` 5, `voice.go` 4, `intelligence.go` 3, `voice_workspace.go` 2, `mail.go` 1, `admin_operations.go` 1 (총 **35줄에 38개 호출** — 줄과 호출 수가 다르다).
  4) 섭동으로 테스트가 실제로 도는 것을 보인다: `internal/api/parameters.go` 의 `limit(fallback, high int)` 를 `number("limit", …, fallback, 1, high+1)` 로 한 글자 바꾸면 `limit` 을 쓰는 operation 들이 red; 되돌리면 green. `internal/server/crm.go:20` 의 `httpx.IntQuery(r, "limit", 10, 1, 50)` 를 `… 10, 1, 100)` 으로 바꾸면 `GET /search` 만 red. 두 섭동 모두 `git checkout --` 로 되돌리고 최종 `git status` 에 테스트 파일만 남길 것.
  5) 기존 테스트 이름·실패 메시지 문구는 한 글자도 바꾸지 말 것. 프로덕션 파일은 0개가 목표다.
- 건드릴 파일:
  - `internal/server/openapi_contract_test.go` — **여기 한 파일로 끝내는 것이 목표**. 이미 필요한 뼈대가 다 있다:
    - `packageFiles(t, ".")`(:37) — 비-테스트 `.go` AST 파싱.
    - `routedOperationList(t)`(:68) — operation ↔ 핸들러 메서드 이름 매핑. `route.handler`/`route.method`/`route.path` 를 그대로 쓸 것.
    - `queryKeysByHandler(t)`(:252) — **이미** `strings.HasSuffix(sel.Sel.Name, "Query") && len(call.Args) >= 2 && receiver.Name == "httpx"` 로 바로 그 호출을 방문하고 `call.Args[1]` 의 키만 꺼내 버린다(:289-293). 남은 `call.Args[2..4]`(fallback/min/max)를 함께 담는 자리가 여기다. 반환형을 `map[string]map[string]bool` 에서 바꾸지 말고 **별도 수집기 함수**를 하나 더 두는 쪽이 기존 세 테스트를 건드리지 않아 안전하다(판단은 구현자 몫).
    - `parameterList(t, operations, method)`(:161)·`documentPaths(t)`(:142) — 문서 쪽 파라미터 읽기.
    - `stringLit`(:59) — 문자열 리터럴. **정수 리터럴용은 없으니 새로 써야 한다.**
  - 새 테스트 함수 이름 제안: `TestDocumentedIntegerBoundsMatchTheHandler`.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/server/ -run 'TestDocumented|TestEveryRouted|TestEveryDocumented|TestProjectionParameter|TestEveryPathTemplate' -v`
  - `go test ./internal/server/` · `go test ./...` · `go vet ./...` · `go build ./...`
  - `gofmt -l internal/server/openapi_contract_test.go` (무출력이어야 함)
  - `git diff --check` (무출력)
  - 개수 재확인: `grep -ro 'httpx\.\(Int\|Clamp\)Query' --include=*.go internal/server/ | grep -v _test.go | wc -l` → **38** (이번 정찰 실측). `-o` 가 핵심이다 — `-c`/`-l` 은 **줄**을 세고 35가 나온다. 한 줄에 둘 이상 있는 자리가 있다: `internal/server/intelligence.go:29` 에 `minimum`·`limit` 두 개, `internal/server/crm.go:227` 에 `days`·`limit` 두 개, `internal/server/crm.go:255` 에 `ClampQuery(expiringDays)`+`IntQuery(limit)` 두 개. 상수에는 **AST 가 실제로 센 호출 수**를 쓸 것.
  - 기존 계약 테스트가 지금 green 인 것은 정찰이 확인했다: `go test ./internal/server/ -run 'TestDocumented|TestEveryRouted|TestEveryDocumented|TestProjectionParameter|TestEveryPathTemplate'` → `ok github.com/hkjang/relio/internal/server 0.152s`.
  - `make test` 는 `npm ci` 로 네트워크를 타고 이 환경에서 ETIMEDOUT 이력이 있다. 이번 변경은 Go 테스트 1파일이라 `go test ./...` 로 충분하다. 돌린다면 선택.
- 위험과 피할 것:
  - **파싱 함정 2개** — ① `versionFilter()` 의 fallback `-1` 은 `*ast.BasicLit` 이 아니라 `*ast.UnaryExpr{Op: token.SUB}` 다. ② `1_000_000` 은 `strconv.Atoi` 가 거부한다. `strconv.ParseInt(lit.Value, 0, 64)` 는 base 0 일 때 Go 정수 리터럴의 밑줄을 받으므로 그쪽을 쓸 것. 문서 쪽 값은 `any` 에 담긴 Go `int` 이므로 `int` 로 타입 단정(assert)한다(`parameters.go:70` 이 `min: low, max: high` 로 `int` 를 그대로 넣는다).
  - **오늘 드리프트는 없다** — 정찰이 38자리 전부를 `parameters.go` 와 눈으로 대조했고 어긋난 자리를 못 찾았다(`limit`·`days`·`months`·`year`·`version`·`minScore`·`minAgeDays`·`expiringDays`·`minimum` 전부 일치). 즉 이것은 **그물이지 버그 수정이 아니다**. 사전 red 는 "새 테스트가 아직 표에 닿지 못하는" 모양이나 위 섭동으로만 나온다 — 2026-09-30 회차와 같은 상황이고, 그때처럼 섭동을 눈으로 보고 기록하라.
  - `internal/api/parameters.go` 를 **고치지 마라**. 오늘 두 표가 일치하므로 고칠 것이 없다. 섭동으로 잠시 바꾸는 것만 허용하고 반드시 되돌릴 것.
  - 보호 경로(`internal/auth`·`internal/oidc`·`internal/server/public.go`·`migrations/`·`.github/workflows/`)는 0줄. 이 과제는 그 어느 것도 필요 없다.
  - `GET /auth/oidc/start`·`callback` 은 정수 파라미터가 없으니 자연히 범위 밖이다 — 일부러 제외하지 말고 "정수 파라미터만" 이라는 조건으로 자연히 빠지게 둘 것.
  - 문서 쪽에 `$ref`(`#/components/parameters/fields`)인 항목이 섞여 있다(`openapi_contract_test.go:333` 이 이미 건너뛴다). 새 테스트도 `parameter["$ref"] != nil` 이면 건너뛸 것. `fields` 는 `type: string` 이라 정수 조건으로도 빠지지만 두 겹으로 막는 편이 낫다.
  - `sel.Sel.Name` 접미사 매칭(`HasSuffix(…, "Query")`)은 `r.URL.Query()` 와 겹칠 수 있다 — 기존 코드가 `receiver.Name == "httpx"` 로 막고 있으니 그 조건을 떨어뜨리지 말 것.
- 차선 후보: `internal/api/parameters.go` 의 `choice()` enum 을 서비스가 실제로 받는 값과 대조하기 — `TestDocumentedSortValuesMatchTheQueryBuilder`(:436)가 `sort` 에 대해서만 하고 있는 일을 `status`·`severity`·`forecastCategory`·`voiceType`·`knowledgeStatus`·`causeEvidence` 로 넓히는 것. 받는 쪽 정본이 Go 상수인지 `migrations/` 의 CHECK 제약인지 자리마다 다를 수 있어(미확인) 1순위보다 범위가 흔들린다 — 1순위가 성립하지 않을 때만, 그리고 한 번에 한두 키만.
