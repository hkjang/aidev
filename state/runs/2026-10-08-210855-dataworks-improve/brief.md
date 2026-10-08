- 과제: 레거시 factory 상품 목록 GET의 아이디어·대시보드 조회 실패를 500으로 전달 (가치 4 / 위험 1 / 작업량 S)
- 왜: HEAD 39f9d99의 `handleFactoryProducts`는 `ListProductIdeas`·`FactoryDashboard` 오류를 버려 읽기 장애에도 상품 팩토리 화면에 빈 아이디어나 부분 KPI를 전달한다. 이미 수정된 store의 오류 전파를 호출부까지 연결하면 운영자가 장애를 정상 조회로 오인하지 않고 기존 화면의 오류 표시를 받는다.
- 수용 기준:
  1) `GET /admin/factory/products`의 상품·아이디어·대시보드 조회 중 하나라도 실패하면 HTTP 500, `error.type=server_error`, 기존 `error.code=products_failed`를 반환한다. 실패 본문에는 `error`만 있고 `products`, `ideas`, `dashboard`는 없다.
  2) 빈 DB에서는 종전처럼 200 + 빈 products/ideas 배열 + 0 KPI다. 상품 2건(draft/review)과 아이디어가 있는 경우 기존 내용·순서·배열 계약을 유지하며 `?status=%20review%20`는 review 상품만 반환하지만 아이디어와 대시보드는 전체 기준 그대로다. 아이디어 limit=50·status trim·조회 순서는 변경하지 않는다.
  3) 실제 SQLite와 `NewServer(...).Routes()` HTTP 테스트로 아래 세 장애를 각각 재현하고 수정 전 실패 → 수정 후 통과를 증명한다. 매 사례 cleanup에서 스키마 복구 후 전체 정상 본문으로 복귀하는지 확인한다. mock store·소스 문자열 검사만으로 대신하지 않는다.
- 건드릴 파일:
  - `internal/proxy/admin_factory.go:handleFactoryProducts` (96~111행) — 두 `_`를 err 확인으로 바꾸고 바로 앞 ListDataProducts 분기와 같은 500 `products_failed`로 즉시 반환. 프로덕션 변경은 이 파일 한 함수뿐이다.
  - `internal/proxy/admin_dataworks_catalog_read_test.go` — 신규 `TestFactoryProductsRejectsUnavailableSources`; 기존 `TestFactoryDashboardRejectsUnavailableAggregates`(:254), `getAdminJSON`, `dashboardCountOf`를 참고하고 기존 테스트를 약화하지 않는다. 테스트명이 신규임을 유의한다.
  - `docs/OPERATIONS.md` KPI 조회 실패 절(:135) — legacy 목록 경로도 실패 시 `products_failed`를 낸다는 운영 문단 3~5줄.
- 검증 명령 (저장소 루트):
  - `go test ./internal/proxy -run '^TestFactoryProductsRejectsUnavailableSources$' -count=1 -v` — 구현자가 추가할 신규 테스트; 테스트만 추가한 단계는 세 장애 실패를 예상, 수정 후 PASS 필요.
  - `go test ./internal/proxy -run '^TestFactoryDashboardRejectsUnavailableAggregates$' -count=1 -v` — 정찰 실행 PASS, 0.270초.
  - `go test ./internal/store ./internal/proxy -count=1` — 정찰 PASS: store 14.812초, proxy 39.645초.
  - `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `go run ./cmd/api-surface-audit`, `git diff --check` — 실제 CI 정의에 있는 Go 검증(마지막은 변경 검사). 정찰에서는 전체 CI 미실행; 구현 완료 후 수행한다.
- 위험과 피할 것: auth/session, store SQL·마이그레이션, .github/workflows, web, npm 런처·lockfile, 상품 상세 GET, graph, publish gate를 같이 수정하지 않는다. 프로덕션 1파일로 완료하며 API 경로·성공 응답 키·SQL·조회 순서·새 오류 코드를 추가하지 않는다. 정상 빈 배열을 오류로 취급하지 말 것. 장애 SQL은 테스트 전용 격리 DB에서만 실행하고 t.Cleanup을 rename 직후 등록한다. SQL 에러 원문 정책은 기존 products_failed 분기를 그대로 따르고 새 감사 details에 조회 데이터·원문을 넣지 않는다.
- 차선 후보: `buildDataWorksGraph`의 feedback/outcomes/relationships 조회 실패 처리 (가치 3 / 위험 2 / 작업량 M) — 1순위가 이미 해결됐거나 실행 근거가 무효일 때만 선택. `admin_dataworks_ops.go:863`의 세 읽기로 제한하고 `handleDataWorksPortfolioGraph`의 기존 `graph_failed`와 asset lineage의 `lineage_failed` 양쪽에서 SQLite+HTTP 재현을 먼저 확보한다. 이번 정찰에서 graph HTTP 장애는 미확인이다.

범위와 근거

기준 HEAD는 39f9d99이며 c835720(store 집계 오류 처리)이 이미 포함됐다. 같은 수정을 다시 가져오지 않는다. `server.go:413`이 목록 핸들러를 등록하고 `admin_ui.go:renderFactoryHome`(:15885~15900)이 실제로 `/admin/factory/products`를 부른다. 화면은 성공 응답의 `ideas || []`로 빈 목록을 만들고, API 호출 예외에는 기존 catch에서 오류를 표시한다. 브라우저 실행은 미확인이고 HTTP 최종 응답은 아래와 같이 실측했다.

정찰 재현: `go build -o /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-210855-dataworks-improve/assets/dataworks-scout ./cmd/dataworks` 성공 후 독립 SQLite·127.0.0.1 임의 포트의 실제 cmd/dataworks 프로세스를 실행했다. 프로세스는 종료했고 DB 스키마는 복구했다. 원문은 같은 실행 디렉터리의 `assets/factory-reproduction.json`, 실행 절차는 `assets/reproduce_factory.py`다(정찰용 스크립트는 현재 결함의 200을 확인하는 하네스이므로 수정 후 검증은 위 Go 회귀 테스트를 사용).

| 사례 | 장애 레버·대조군 | 현재 관찰 | 수정 후 |
| --- | --- | --- | --- |
| 아이디어만 실패 | product_ideas를 다른 이름으로 rename 후 `CREATE VIEW product_ideas AS SELECT id FROM <백업테이블>`; COUNT는 정상이고 ListProductIdeas SELECT의 title 등이 없어 실패. `/admin/factory/dashboard`는 200·ideas_total=5 | 목록 200, 상품 2건, ideas=null, dashboard 정상 | 목록 500 products_failed |
| 집계 리스크 실패 | product_risk_reviews rename. 아이디어·상품 테이블은 유지. 전용 dashboard는 500 dashboard_failed | 목록 200, 상품 2건·아이디어 5건·부분 dashboard | 목록 500 products_failed |
| 집계 PoC 실패 | product_poc_plans rename. 전용 dashboard는 500 dashboard_failed | 목록 200, 상품 2건·아이디어 5건·부분 dashboard | 목록 500 products_failed |

각 복구 뒤 원래의 전체 응답과 동일함을 실행 확인했다. 빈 DB 기준선은 products=[]/ideas=[]/모든 KPI=0이며, 데이터 기준선은 draft=1/review=1/ideas_total=5다. `POST /admin/dataworks/products`로 상품을 만들고 `POST /admin/factory/ideas/generate`에 industry/market_need를 보내 아이디어를 만들었다. 이번 입력 count=1에도 기존 생성기가 5개를 생성했으므로 회귀 테스트는 생성 응답에서 실제 아이디어 수를 얻어 비교하고 count 동작은 수정하지 않는다.

테스트 설계 주의: product_ideas 통째 rename만 사용하면 ListProductIdeas와 FactoryDashboard가 동시에 실패해 아이디어 오류 분기를 빠뜨린 구현도 통과한다. 위 id 전용 뷰 사례가 반드시 필요하다. data_products 전체 rename이나 status 전용 뷰는 목록의 첫 ListDataProducts가 먼저 실패하므로 새 집계 분기의 증거로 쓰지 않는다. 이 레버들은 운영 스키마 변경 제안이 아니다.

실행 순서와 체크포인트 (사람 승인 없이 자체 확인)

1. [미착수] 위 테스트 파일에 신규 HTTP 회귀 테스트와 정상 빈/비어 있지 않은 기준선을 추가한다. `go test ./internal/proxy -run '^TestFactoryProductsRejectsUnavailableSources$' -count=1 -v`에서 세 장애만 예상한 200 때문에 실패하고 복구 단언은 통과하는지 확인한다. 테스트 파일만 추가하므로 제품 빌드는 유지된다. 다른 원인으로 실패하면 레버를 수정하고 과제서의 달라진 사실을 기록한다.
2. [미착수] handleFactoryProducts의 두 err만 처리한다. 같은 신규 테스트 및 기존 FactoryDashboard 테스트가 통과하는지 확인한다. 정상 응답을 바꾸는 공통 헬퍼·리팩터는 불필요하다.
3. [미착수] OPERATIONS 문단을 추가하고 위 전체 Go 검증과 diff 검사를 실행한다. 변경 목록은 프로덕션 1 + 테스트 1 + 문서 1이어야 한다. 웹 변경이 없으므로 이 과제를 위해 npm 의존성을 설치하거나 릴리즈 경로를 고치지 않는다. CI 자체의 웹 검증은 그대로 둔다.

대안 비교와 선택 근거

- 호출부의 두 오류를 기존 500 계약으로 처리: 가장 작은 변경이며 현재 화면의 오류 경로와 맞는다. 채택.
- 부분 응답 + warnings/별도 상태를 도입: 일부 상품을 계속 보여줄 수 있으나 응답·UI 계약을 새로 설계해야 한다. 이 회차 45분·소폭 수정 범위에서는 보류.
- store에서 오류 대신 정상 기본값을 반환하거나 현상 유지: 변경비용은 작지만 이미 합의·머지된 집계 오류 전파를 무력화하거나 입증된 거짓 성공을 남긴다. 기각.
- graph·상세 GET의 무시된 오류를 한꺼번에 처리: 비슷한 결함이나 다수 출력과 선택적 데이터 의미를 확인해야 한다. 별도 아이디어로 분리.
가장 중요한 전제는 이 목록이 세 소스가 모두 정상인 성공 응답을 제공해야 한다는 것이다. 형제 products 조회의 기존 500 계약, 전용 dashboard의 500, renderFactoryHome의 catch 및 실행 재현으로 뒷받침된다. 실제 사용자 트래픽 빈도와 PostgreSQL 장애·브라우저 결과는 미확인이다.

작업량 산정

방법은 bottom-up이며 범위는 테스트/구현/문서/검증까지다: 테스트·실패 확인 12~15분, 두 오류 분기 3~5분, 문서 2~3분, 전체 검증·검토 8~12분으로 기본 25~35분을 추정한다. 알려진 변동(Go 컴파일·SQLite cleanup·CI 검증 시간)에 예비 5~10분을 별도 배정해 총 30~45분, 신뢰도 중간의 판단 범위로 둔다(통계적 80% 확률 추정은 아님). 관리 예비는 배정하지 않으며 새 범위는 다음 회차로 넘긴다. 과거 catalog/evidence 수정도 프로덕션 1 + 테스트 1 + 문서 1이었으므로 유사 작업의 범위로 교차 확인했지만 실제 소요 분 자료가 없어 속도 비교는 미확인이다. 첫 red 테스트가 복잡해지거나 범위를 바꿔야 하면 추정과 계획을 다시 쓰며 45분에 맞추려고 검증을 생략하지 않는다.

적용 스킬: 로컬 `pmo:estimating-and-contingency`(범위·가정·기본 추정과 예비 분리), `technology:implementation-planning`(파일·명령·자체 체크포인트), `technology:solution-exploration`(선택지와 기각 이유). 전용 Skill 도구가 없어 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/`의 해당 SKILL.md를 직접 읽었다. estimating 스킬의 references/sources.md도 확인했으며 외부 비용률·통계 신뢰수준을 가져오지 않고 저장소 실측과 명시한 판단에 근거했다.

스킬 원문 위치: [estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md).
최종 확인: 정찰은 저장소 코드·커밋 무변경이며 `git status --short` 빈 출력, `git diff --check` exit 0. 스킬 4파일과 저장소 약 24파일의 필요한 구간만 읽었다. CLAUDE.md·AGENTS.md 및 별도 ROADMAP/TODO 파일은 목록 검색에서 없었고, 검토 파일의 TODO/FIXME는 0건이었다. 초안을 먼저 저장한 뒤 실측 결과로 이 문서를 덮어썼다.
