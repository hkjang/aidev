- 과제: FactoryDashboard의 집계 조회 실패를 정상 KPI 0건으로 숨기지 않기 (가치 4 / 위험 1 / 작업량 S)
- 왜: `internal/store/factory.go:477`의 `FactoryDashboard`는 `product_ideas` 외 상태별 상품 수·리스크 검토·대기 PoC·평균 매출 조회 오류를 버려, 실제 서버에서 테이블 조회가 실패해도 대시보드가 200과 0을 반환한다. 이 한 함수가 오류를 반환하도록 고치면 이미 오류 처리가 있는 네 HTTP 경로가 불완전한 KPI 대신 기존 500 오류를 전달한다.
- 수용 기준:
  1) `FactoryDashboard`의 10개 집계 쿼리(product_ideas 1 + 상태별 COUNT 6 + 리스크 1 + PoC 1 + 평균 1) 어느 곳에서든 QueryRowContext/Scan 실패가 나면 non-nil error가 반환된다. `countStatus`를 `(int64, error)`로 바꾸고 호출별 오류를 검사하거나 동등하게 작은 구현을 쓴다. 기존 반환형 `(FactoryDashboard, error)`, SQL 조건, 조회 순서, `s.bind`, 빈 집계의 COALESCE, `int64(avgRevenue + 0.5)` 반올림을 유지한다. 오류 때는 기존 첫 쿼리와 동일한 `return d, err` 방식으로 부분 구조체를 유지해 범위 밖 호출자의 동작을 불필요하게 바꾸지 않는다.
  2) 실제 SQLite + `NewServer(...).Routes()`의 HTTP 회귀 테스트에서 `product_ideas`는 정상으로 둔 채 `data_products`, `product_risk_reviews`, `product_poc_plans`를 각각 일시 rename한다. `/admin/factory/dashboard`는 각 실패에 500 `dashboard_failed`, `/admin/dataworks/funnel`은 500 `funnel_failed`를 반환해야 한다. 리스크·PoC 테이블 장애는 `/admin/dataworks/home`의 500 `dashboard_failed`와 `/admin/dataworks/analytics`의 500 `analytics_failed`까지 확인한다. 모든 오류 응답에 정상 KPI 본문이 없어야 한다.
  3) 평균 쿼리도 독립적으로 실패시킨다: `data_products`를 rename한 뒤 원래 이름으로 `SELECT status FROM <실제 테이블>` 뷰를 만든다. 상태별 COUNT 6개는 성공하고 revenue_score 컬럼이 필요한 마지막 AVG만 실패한다. `/admin/factory/dashboard` 및 `/admin/dataworks/funnel`에서 500을 검증한다. 뷰 설치 후 동일 SQL로 상태별 COUNT가 성공하고 AVG가 실패함을 확인해 레버를 검증한다. 이 레버는 정찰의 실제 서버 재현에서 실행 완료했다.
  4) 정상 빈 DB는 200과 0을 유지한다. 실제 store를 이용하는 정상 집계 테스트에 여섯 상태(draft/review/risk_review/approved/published/archived), HighRiskReviews의 >=70 경계, PendingPOCPlans의 pending 조건, 양수 revenue_score 평균 및 반올림을 넣어 기존 계산 결과가 유지됨을 확인한다. 각 장애 사례 cleanup에서 테이블/뷰를 복구한 뒤 HTTP 200 기준선으로 복귀함도 확인한다. 가짜 store 주입·소스 문자열 검사로 대체하지 않는다.
  5) 새 회귀 테스트를 수정 전 코드에서 먼저 실행해 해당 HTTP 단언이 200 때문에 실패하는 것을 기록한다. 수정 후 새 테스트와 기존 store/proxy 테스트를 통과시키고 전체 Go 검증도 수행한다. 신규 API·오류 코드·스키마·웹 변경은 없다.
- 건드릴 파일:
  - `internal/store/factory.go:FactoryDashboard`(477-498) — 오류를 버리는 `countStatus`와 세 Scan 오류를 반환. **프로덕션 1파일**.
  - `internal/store/factory_test.go:TestFactoryRoundtrip`(90-165) 인근 — `TestFactoryDashboard...` 정상 집계·빈 DB 테스트 추가. 기존 fixture의 `Open`, `Migrate`, 실제 삽입 메서드를 활용한다.
  - `internal/proxy/admin_dataworks_catalog_read_test.go:TestDataWorksCatalogReadsRejectUnavailableTables`(71) 인근 — 독립 `TestFactoryDashboardRejectsUnavailableAggregates` 추가 권장. 기존 `getAdminJSON`, `dashboardCountOf`, `errorCodeOf`, 실제 SQLite 연결/Routes/rename 패턴을 재사용한다. 이전 회귀 사례는 유지한다.
  - `docs/OPERATIONS.md`의 기존 KPI 조회 오류 설명 — 집계 내부 실패도 기존 오류 코드로 드러남을 짧게 추가. 정본 PDF는 없으며 다른 가이드/PDF는 범위 밖.
- 검증 명령(저장소 루트):
  - 새 테스트 작성 후 전후 비교: `go test ./internal/proxy -run '^TestFactoryDashboardRejectsUnavailableAggregates$' -count=1 -v`
  - 정상 집계: `go test ./internal/store -run '^TestFactory(Dashboard.*|Roundtrip)$' -count=1 -v`
  - 기존 회귀 포함: `go test ./internal/store ./internal/proxy -count=1`
  - 마무리: `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `go run ./cmd/api-surface-audit`, `git diff --check`.
  - 정찰에서 실제 실행: 기존 `go test ./internal/store ./internal/proxy -count=1` PASS(store 18.387s, proxy 48.600s); `go build -o <회차 폴더>/scout-server ./cmd/dataworks` PASS. 위 신규 테스트는 구현자가 작성할 이름이며 아직 존재하지 않는다. 이번 정찰에서 전체 build/vet/test/API 감사와 웹 검증은 미실행이다.
- 위험과 피할 것: 스키마/마이그레이션·auth/session·CI/workflows·웹·의존성·lock·릴리즈 런처 변경 금지. 집계 SQL를 GROUP BY/트랜잭션으로 재구성하거나 시간/상태 파서를 통합하지 않는다. 기존 `main`은 baa3415이므로 비교 기준은 **HEAD a528bcd(v0.9.70)**. 테이블 rename 테스트는 격리된 임시 DB에서만, 병렬 실행 금지, rename 성공 직후 cleanup 등록. data_products 전체 장애로 home/analytics만 검사하면 이미 ListDataProducts에서 500을 내므로 수정 전에도 통과하는 거짓 회귀 테스트가 된다; factory/dashboard와 funnel이 핵심이다. 평균 전용 뷰는 모든 상품 컬럼을 읽는 home/analytics에 쓰지 않는다. 실제 운영 DB·설정은 건드리지 않는다. `handleFactoryProducts`(admin_factory.go:110)는 호출부에서도 오류를 버리므로 이 경로까지 해결됐다고 주장하지 않는다; 별도 보류 후보다.
- 차선 후보: 유실된 web JS/MJS 린트 규칙 복구 — `web/eslint.config.js`의 **/*.{js,mjs} 블록 및 실제 ESLint 회귀 테스트로 제한. 5ed6876은 현재 HEAD 조상이 아니고 현재 config에는 TS/TSX 블록만 있다. 현재 1순위는 실행 재현됐으므로 전제 변화가 없으면 차선으로 옮기지 않는다. 전체 커밋 무조건 cherry-pick·lock 재생성·루트 린트 배선 확대는 금지.

근거와 재현(2026-10-08, HEAD a528bcd):
- 소스를 수정하지 않고 빌드한 `cmd/dataworks` 바이너리를 회차 폴더의 별도 SQLite DB와 loopback 임시 포트로 실행했다. DB/로그도 회차 폴더에만 썼고, 외부 서비스 자격증명은 전달하지 않았다. 별도 Python sqlite3 연결로 장애를 만들고 프로덕션 HTTP 라우트를 호출한 결과는 같은 폴더 `scout-reproduction.json`에 저장했다. 서버는 종료하고 DB는 복구했다.
- 정상 빈 DB: factory/dashboard, dataworks/home, analytics, funnel 모두 200.
- data_products rename: factory/dashboard=200, funnel=200; home/analytics=500(상위 별도 조회가 실패를 잡음).
- product_risk_reviews 또는 product_poc_plans rename: 위 네 경로 모두 200.
- 상태 컬럼만 제공하는 data_products 뷰: factory/dashboard와 funnel 모두 200(평균 조회 실패를 숨김).
- 모든 테이블 복구 후 네 경로 모두 다시 200. 이 결과는 오류 노출 결함의 실행 증거이며, 정상 비어 있지 않은 KPI/평균 반올림의 신규 테스트 결과는 아직 미확인이다.

대안과 선택 근거:
- 선택안은 기존 오류 반환 계약을 집계 안까지 연결한다. 새 구성요소가 없고 기존 핸들러 네 곳을 그대로 사용한다.
- 응답에 부분 성공/경고 필드를 추가하는 대안은 사용할 수 있지만 API·UI 계약 변경이 필요하므로 이번 45분 과제에는 부적합하다.
- 집계 일괄 SQL/스냅샷 트랜잭션은 성능·일관성 요구가 생겼을 때 가능하나 현재 결함에는 필요 없고 두 DB 방언 검증 비용이 늘어난다.
- 현상 유지+운영 로그만 추가하면 HTTP 200 오판은 남는다. web 린트 복구는 타당한 차선이지만, 이번 결함은 실제 사용자 응답의 오류이며 설치 없이 재현돼 우선한다.
- 핵심 전제: 선택한 네 경로는 집계 하나라도 실패하면 기존 err 분기로 요청 전체를 실패시키는 계약이다. 코드와 실행으로 확인했다. PostgreSQL에서 동일 장애 레버 실행은 미확인이나 제품 SQL 자체를 변경하지 않는다.

구현 순서·추정·체크포인트:
1. [미착수] 위 HTTP 회귀 테스트 작성, 수정 전 200 실패 기록(10–13분). proof: 신규 proxy 테스트 명령. 자동 체크포인트: 예상한 실패가 아니면 레버/계획을 고친 후 진행한다. 사람 승인 대기 없음.
2. [미착수] factory.go 한 함수 수정 + 정상 store 집계 테스트(10–13분). proof: 신규 proxy/store 테스트 명령. 자동 체크포인트: 신규 사례 모두 통과하고 기존 정상 계산 유지; 실패하면 범위 확대 없이 원인을 수정한다.
3. [미착수] 운영 문단, 기존 회귀와 전체 Go 검증(8–11분). proof: 위 마무리 명령과 diff 확인. 자동 체크포인트: 오류/회귀가 없을 때 완료. 웹을 안 바꾸므로 npm 설치는 이 과제의 조건이 아니다.
- bottom-up 기본 28–37분 + 알려진 불확실성(테이블 복구/뷰 레버·환경별 테스트 시간) 대응 예비 4–6분 = 계획 범위 32–43분. 신뢰도는 중간이며 통계적 P80 같은 보장값이 아니다. 관리 예비는 배정하지 않으며 새 API/스키마 요구가 생기면 별도 회차로 분리한다.
- 유사 작업 근거는 10-07의 실제 SQLStore/Routes/rename 기반 KPI 오류 수정이다. 다만 당시 소요시간 기록이 없으므로 유사 사례를 별도 숫자 추정으로 꾸미지 않고 난이도/재사용성 교차 확인에만 썼다. 현재 패키지 테스트 실행 시간을 포함해 산정했으며 새 테스트의 실제 red/green 확인 뒤 재추정한다.
- 적용 스킬: Skill 도구는 제공 목록에 없어 로컬 원본 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`의 SKILL.md를 읽고 적용했다(`/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/<부서>/skills/<스킬>/SKILL.md`). estimating의 references/sources.md도 확인했다. 외부 비용 모델이나 검증되지 않은 신뢰수준 수치는 사용하지 않았다.
