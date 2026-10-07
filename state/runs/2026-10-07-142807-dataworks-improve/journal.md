# 회차 노트 2026-10-07-142807-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:28] base pinned — main@af463ee
- [러너 14:28] autonomy release — 

## 구현 노트
- 무엇/왜: `internal/proxy/admin_dataworks.go` 의 세 GET 핸들러(`/home`, `/analytics`, `/reviews` 의 risk_review 절반)가 `FactoryDashboard`·`ListDataAssets`·`ListDataProducts` 오류를 `_` 로 버려 읽을 수 없는 테이블을 "KPI 전부 0 / 리스크 검토 대기 0건" 인 200 응답으로 내보냈다. 500 + `dashboard_failed`·`analytics_failed`·(기존)`reviews_failed` 로 바꿨다. 같은 핸들러의 형제 조회가 이미 500 을 내므로 두 경로의 불일치를 없앤 것이다.
- 확신 없는 곳·검증 못 한 것: (1) `docs/OPERATIONS.md` 변경을 **GitHub Pages 하네스로 재빌드하지 않았다** — `{{`·`{%` 0건과 백틱만 추가한 것은 확인했지만 지난 회차처럼 jekyll-build-pages 로 end-to-end 렌더는 돌리지 않았다. (2) PostgreSQL 에서 같은 조회 실패가 나는지는 이 환경에 DB 가 없어 미검증(SQLite 만).
- 소비자 확인(검증됨): web 워크벤치는 이 세 엔드포인트를 호출하지 않는다(grep). 레거시 `internal/proxy/admin_ui.go` 는 `/home`(:14479)·`/analytics`(:15675)를 **`try/catch` 로 감싸 오류 메시지를 렌더**하고, 형제 호출들과 달리 `.catch(() => …)` 로 삼키지 않는다 — 그래서 500 이 0 으로 둔갑하지 않고 화면에 사유가 뜬다. web 변경 불필요.
- 일부러 하지 않은 것: `store.FactoryDashboard`(factory.go:484·493·494·496)가 네 집계 오류를 `_ =` 로 삼키는 것 — `internal/store` 위험 구역이라 분리해 ideas.json 에 신규 등록했다. `admin_dataworks(_ops)` 의 남은 목록 조회 5곳도 엔드포인트가 많아 다음 회차로 미뤘다. 퍼블리시 게이트의 SLA·비용 조회(:1760·:1767)는 게이트 의미 변경이라 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트 `internal/proxy/admin_dataworks_catalog_read_test.go` 는 **같은 DSN 에 두 번째 `sql.Open` 을 열어 ALTER TABLE/CREATE VIEW 로 스키마를 흔든다**(기존 evidence·entitlement 테스트와 같은 기법). 병렬 실행(`t.Parallel()`)을 넣으면 깨진다. `reviews_risk_queue_unreadable` 서브케이스는 `data_products.description` 이 NOT NULL 이고 `ListDataProducts` 가 `COALESCE` 없이 `string` 으로 스캔하는 성질에 의존한다 — 그 쿼리에 `COALESCE(description,'')` 를 붙이면 이 레버가 조용히 무력화되니, 그런 변경을 할 때는 이 테스트를 함께 볼 것.
- [러너 14:40] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인: diff 3파일 전수(`af463ee..HEAD`; 로컬 `main` ref 는 v0.9.53 에 멈춰 있어 쓰지 않았다), 신규 테스트 6 서브케이스 직접 실행 PASS, `go test ./internal/proxy ./internal/store ./internal/dataworks` 전부 ok, `gofmt -l` 무출력, api-surface-audit 550/612 수치 불변·4항목 `[]`. 테스트가 바뀐 경로를 지나는지 교차 확인(`FactoryDashboard` 가 `data_products` 집계 오류를 삼키므로 `home_products`/`analytics_products` 의 500 은 새 `ListDataProducts` 검사에서만 나온다 — 형제 조회의 기존 500 에 올라탄 통과가 아니다).
- 구현자 자기 의심 2건 모두 기각: OPERATIONS.md:135-143 은 liquid 태그 0건·코드펜스 없음·`###` 중첩 정상이라 Pages 렌더 위험 없음; PostgreSQL 미검증은 드라이버 error 를 그대로 전파할 뿐이라 드라이버 의존성 없음. 소비자도 독립 확인(admin_ui.go :14479·:15024·:15675 세 호출 전부 `try/catch` 로 메시지 렌더, `.catch(()=>…)` 삼킴 아님; web/src 는 호출 없음).
- 못 본 것: PostgreSQL 실측(환경에 DB 없음), 브라우저에서 레거시 UI 의 500 렌더 실측, Pages end-to-end 렌더.
- 승인이어도 남는 우려 — **릴리즈 노트 필수**: `/admin/dataworks/home`·`/analytics` 가 DB 장애 시 200→500 으로 바뀐다. 이 두 GET 을 헬스 프로브로 폴링하는 설정이 있으면 장애 중 알람이 새로 뜬다(의도한 방향). 문서 사소한 부정확: :137 이 `/analytics` 도 "자산 조회" 라 적지만 그 핸들러는 `ListDataAssets` 를 호출하지 않는다.
- 다음 회차: `store.FactoryDashboard`(factory.go:483-496)가 네 집계를 여전히 `_ =` 로 삼켜 200 응답의 `poc_pending`·`avg_revenue_score` 는 부분 실패 시 0 으로 조용히 떨어진다(ideas.json 등록 확인). 새 테스트는 같은 DSN 두 번째 `sql.Open` 에 의존하니 `t.Parallel()` 금지, `ListDataProducts` 에 `COALESCE(description,'')` 를 붙이면 `reviews_risk_queue_unreadable` 레버가 무력화된다(다만 :226-233 가드가 먼저 실패한다).
- [러너 14:45] review approved — 리뷰 승인 (risk=low)
- [러너 14:45] pr created — https://github.com/hkjang/dataworks/pull/37
- [러너 14:48] ci passed — 검사 2개 모두 success
- [러너 14:49] merge done — 7a883be
