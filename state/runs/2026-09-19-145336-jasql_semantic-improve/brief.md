# 과제서 (수정 과제) — 2026-09-19

## 먼저: 실패 원인 판정 (정찰이 확인한 사실)
- 마지막 회차의 `error` 는 **`hold: budget`** — 러너의 USD 예산(회차당 $2) 소진으로 릴리즈 단계가 멈춘 것이다. 저장소 쪽 스크립트·테스트가 깨진 것이 **아니다**.
- 이 저장소에는 `.github/workflows` 가 없다(확인). CI 는 없고, "릴리즈 워크플로" 는 러너의 정찰→구현→비평→수리→중재→릴리즈 파이프라인이며 각 단계가 `go test ./...` 를 다시 돈다.
- `go test ./...` 는 main@149edbe 에서 **전부 통과**한다(정찰이 직접 실행). 단 소요 시간이 `docs/development.md:8` 에 적힌 "~17초" 가 아니라 **catalog 57.5s + mcp 8.7s ≈ 67s** 다. 그중 두 골든셋 테스트가 43s 를 차지한다:
  - `internal/catalog/graph_test.go:80 TestEvaluateRetrievalOnGoldenSet` — 29.3s (80문항 × RetrieveContext + SearchSchema)
  - `internal/catalog/eval_test.go:12 TestGoldenEvaluation` — 13.9s (80문항 × SearchSchema)
- 즉 회차 예산을 잡아먹는 저장소 쪽 요인은 **질문 1건당 ~0.17s 걸리는 `SearchSchema` 핫패스**다. 단계마다 60초 넘게 도는 테스트를 기다리며 에이전트 턴이 늘어나고, 시간이 지나 재시도까지 겹치면 예산이 바닥난다. 워크플로(테스트 임계값·게이트)를 느슨하게 하는 것은 금지이므로, 고칠 곳은 **프로덕션 검색 경로의 성능**이다. 이는 MCP `search_schema`/`retrieve_context` 실사용 지연도 같이 줄인다.

## 과제
- 과제: `SearchSchema` 의 테이블×샘플 전수 재계산을 없애 골든셋 테스트와 실사용 검색을 빠르게 (가치 4 / 위험 2 / 작업량 M)
- 왜: `internal/catalog/search.go:285 sampleBoostForTable` 이 **모든 테이블마다 `c.Samples` 전체를 순회**하며 매 (테이블, 샘플) 쌍에서 `strings.ToUpper(sample.TargetTable)`, `strings.Join(...)+ToLower` 를 새로 만든다 — 질문 하나에 O(tables×samples) 문자열 할당이 일어나고 `search.go:111` 의 테이블 루프가 이를 그대로 호출한다. 게다가 `search.go:135 scoreColumns` 는 테이블 점수가 0 이어서 버려질 테이블에 대해서도 항상 돈다. 이걸 사전 인덱스(로드 시 테이블명→샘플 목록, 샘플별 소문자 텍스트 캐시)로 바꾸면 결과·순위는 그대로이고 시간만 줄어든다.
- 수용 기준:
  1) `go test ./internal/catalog` 소요 시간이 지금 57s 에서 **절반 이하**로 내려간다(구현자가 before/after 를 `-v` 로 잰 `TestEvaluateRetrievalOnGoldenSet`, `TestGoldenEvaluation` 시간을 PR 본문에 적는다). 효과가 측정되지 않는 변경은 넣지 않는다.
  2) 골든셋 지표가 **바뀌지 않는다**: 변경 전후 `go run ./cmd/jasql-eval` (또는 `c.RunEvaluation(data/kcb/golden_queries.json, 5)` 요약)의 table/join/column 수치가 동일하다. `eval_test.go` 의 임계값과 `graph_test.go:94` 의 0.5 게이트는 손대지 않는다.
  3) 테스트가 증명할 것: `sampleBoostForTable` 리팩터 전후로 **같은 질문·같은 테이블에 같은 boost 와 reason 문자열**을 돌려준다 — 실제 `loadTestCatalog`(data/kcb) 카탈로그와 골든셋 질문 몇 개로 기존 함수 결과와 새 함수 결과를 비교하는 테스트 1개를 `search_test.go`(없으면 `catalog_test.go`)에 추가. 대역(fake) 카탈로그가 아니라 실제 로드된 카탈로그를 쓴다.
  4) `docs/development.md:8` 의 "~17초" 를 측정값으로 고친다.
- 건드릴 파일:
  - `internal/catalog/search.go:285 sampleBoostForTable` — 샘플 전수 순회 대신 로드 시 만든 인덱스를 조회. 매칭 규칙(`strings.Contains(target, t.FQN) || strings.Contains(target, t.Name)`, 토큰 포함 1.2점, 질문 포함 20점, 상한 24)은 **그대로** 유지. 부분 문자열 매칭이라 단순 map 키 조회로 바꾸면 결과가 달라진다 — 인덱스를 만들 때도 같은 Contains 규칙으로 테이블별 샘플 목록을 미리 계산하거나, 최소한 샘플별 `upperTarget`/`lowerText`/`lowerQuestion` 을 한 번만 만들어 재사용.
  - `internal/catalog/catalog.go`(Load) 또는 `search.go` — 인덱스/캐시 필드 추가와 초기화. `c.Samples` 가 런타임에 바뀌는 경로(few-shot 추가·학습 승격, `learn.go` 쪽)가 있는지 grep 해서 있으면 그 자리에서 캐시를 무효화할 것(미확인 — 구현자가 확인).
  - `internal/catalog/search.go:135` — `scoreColumns` 를 `schemaGate`/점수 판정보다 뒤로 미룰 수 있는지 검토. 단 `search.go:139-152` 주석대로 게이트 보고는 컬럼 점수 포함값을 쓰므로 순서를 바꾸면 게이트 리포트가 달라진다. 확신 없으면 이 항목은 건너뛴다.
  - `docs/development.md:8` — 소요 시간 문구.
- 검증 명령:
  - `go build ./... && go vet ./...`
  - `go test ./internal/catalog -run 'TestEvaluateRetrievalOnGoldenSet$|TestGoldenEvaluation$' -v` (전후 시간 비교)
  - `go test ./...` (전체, 현재 ~67s)
  - `go run ./cmd/jasql-eval` 로 지표 전후 동일 확인(옵션·출력 형식은 `docs/evaluation.md` 참조, 정찰 미실행)
- 위험과 피할 것:
  - 순위·점수가 1건이라도 바뀌면 안 된다. 랭킹은 `search.go:178,203` 의 stable sort 로 동점 순서에 민감하니 부동소수 계산 순서도 바꾸지 말 것.
  - 테스트 임계값·`-short` 스킵·골든셋 축소로 시간을 줄이는 것은 **금지**(워크플로 완화).
  - `internal/mcp/admin*.go`, 인증·토큰 경로, `scripts/release-image.sh`, `Dockerfile*` 는 건드리지 않는다.
  - 프로파일: `go tool pprof` 가 이 환경에서 승인 없이 안 돌 수 있다(정찰은 막혔음). `-cpuprofile` 로 파일을 만든 뒤 `go tool pprof -top -cum` 을 시도하고, 안 되면 위 정적 분석대로 진행.
  - 러너 예산 자체는 저장소에서 못 고친다. 이 과제로 단계당 테스트 시간을 줄이는 것까지가 저장소 몫이며, 그래도 `hold: budget` 이 반복되면 러너 설정(회차 예산 또는 단계 수) 문제라고 원장에 적는다.
- 차선 후보: `docs/development.md` 의 잘못된 테스트 시간(~17s → 실측 ~67s)과 테스트 표만 고치는 문서 수정 (가치 2 / 위험 1 / S) — 1순위가 지표를 바꾸지 않고는 성립하지 않을 때.
