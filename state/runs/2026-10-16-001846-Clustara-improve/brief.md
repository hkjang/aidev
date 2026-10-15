- 과제: 용량 API의 인벤토리 조회를 분석 대상 Kind로 한정하고 창 포화를 보고하기 (가치 3 / 위험 2 / 작업량 M)
- 왜: `handleK8sCapacity`(`internal/proxy/admin_k8s.go:592`)는 Kind 필터 없이 `Limit: 4000` 으로 조회하고 `ListK8sInventory` 는 `ORDER BY i.updated_at DESC`(`internal/store/k8s.go:580`)이지만, `AnalyzeCapacity` 는 `HorizontalPodAutoscaler`·`Pod`·`Node` 만 읽는다(`internal/analyzer/capacity.go:161-172`, `nodePackingAndGPU:236,252`, `ProjectNodeCapacity:88`). ConfigMap·Secret·Event 처럼 자주 갱신되는 비대상 행이 창을 채우면 Node·Pod 가 보고서에서 조용히 빠져 GPU·node_packing·projections 가 빈 배열이 되고, 포화 사실도 응답에 전혀 나타나지 않는다. 바로 옆 `/admin/k8s/security` 는 이미 같은 결함을 `Kinds` + `budget+1` + `scan{status,resources,reason}` 로 고쳐 놓았다(`admin_k8s.go:614-655`) — 용량만 남았다.
- 수용 기준:
  1) 비대상 Kind 행이 `updated_at` 상 더 최신이어서 창을 채우는 인벤토리에서도 `GET /admin/k8s/capacity` 가 `report.node_packing`·`report.gpu`·`report.hpas` 행을 정상 반환한다.
  2) 대상 Kind 행 수가 예산을 넘으면 응답이 그 사실을 명시한다(보수적 안내 문구). 예산 안에 들어가면 그 키/상태는 "완전" 쪽으로 남고 불필요한 경고가 붙지 않는다.
  3) 테스트가 수정 전 코드에서 실제로 빨강이 되는 것을 확인하고(실패 출력을 요약에 인용), `analyzer.CapacityRelevantKinds()` 가 분석이 읽는 세 Kind를 모두 포함하고 비대상 Kind를 포함하지 않음을 단위 테스트로 고정한다.
- 건드릴 파일 (프로덕션 2 + 테스트 2):
  - `internal/analyzer/capacity.go` — 신규 `CapacityRelevantKinds() []string` 추가. `[]string{"HorizontalPodAutoscaler", "Node", "Pod"}`. `RCARelevantKinds()`(`internal/analyzer/rca.go:52`)의 주석 톤을 그대로 따를 것 — "여기 빠진 Kind는 미조회로 보고되지 않고 findings 가 그냥 사라진다". **기존 분석 함수의 로직은 한 줄도 고치지 말 것.**
  - `internal/proxy/admin_k8s.go:handleK8sCapacity` — `ListK8sInventory` 호출에 `Kinds: analyzer.CapacityRelevantKinds()` 와 `Limit: capacityScanBudget + 1` 을 넣고, `len(items) > capacityScanBudget` 이면 잘라낸 뒤 포화를 응답에 싣는다. 응답 형태는 security 와 같은 결로 맞추되 **기존 `report`·`note` 키는 그대로 둘 것**(UI가 `report.gpu` 를 직접 렌더한다 — 추가 키만 허용). `securityScanBudget`(`admin_k8s.go:660`)처럼 `var capacityScanBudget = 4000` 을 패키지 변수로 두어 테스트가 수천 행을 심지 않고 포화 경로를 돌 수 있게 한다(주석도 같은 이유를 적을 것).
  - `internal/proxy/admin_k8s_capacity_scope_test.go`(신규) — 아래 재현 레시피.
  - `internal/analyzer/capacity_kinds_test.go`(신규, 또는 기존 analyzer 용량 테스트 파일에 추가) — `CapacityRelevantKinds()` 가 `Node`·`Pod`·`HorizontalPodAutoscaler` 를 포함하고 `ConfigMap`·`Secret`·`Service` 를 포함하지 않음. `security_scan_scope_test.go:62 TestSecurityRelevantKindsCoversWhatIsAnalysed` 가 본일 뿐 모델.
- 재현 레시피 (확인된 배선만 사용):
  - 하네스는 `admin_k8s_capacity_gpu_test.go:18` 과 동일: `db := openTestStore(t)` → `kube.InventoryFromObject(kind, "v1", map[string]any{...})` → `item.ClusterID`/`item.ID` 지정 → `db.UpsertK8sInventory` → `httptest.NewServer(server.Routes())` 로 `GET /admin/k8s/capacity?cluster_id=...`.
  - **`updated_at` 을 직접 지정할 수 있다**: `UpsertK8sInventory`(`store/k8s.go:439-446`)는 `item.UpdatedAt` 이 비어 있을 때만 `nowString()` 을 넣는다. 따라서 ConfigMap 류에 `UpdatedAt: "2026-01-02T00:00:00Z"`, Node/Pod 에 `"2026-01-01T00:00:00Z"` 를 주면 `updated_at DESC` 창이 비대상으로 먼저 채워지는 것이 **시간 동률 없이 결정적으로** 재현된다. `capacityScanBudget` 을 2~3 으로 낮춰 수천 행 없이 돌릴 것(`security_scan_scope_test.go:23-25` 의 `t.Cleanup` 복원 패턴 그대로).
  - 수정 전 기대 실패: GPU/node_packing 이 빈 배열. 수정 후 노드 행이 나온다. 포화 테스트와 비포화("아무것도 안 잘렸으면 경고가 붙지 않는다") 테스트를 **둘 다** 둘 것 — security 쪽도 그 쌍으로 되어 있다.
- 검증 명령 (이번 정찰에서 기준선 실행 확인: analyzer 0.005s / proxy 0.119s 통과):
  - `go test ./internal/analyzer ./internal/proxy -run 'Capacity' -count=1`
  - 추가 회귀: `go test ./internal/analyzer ./internal/proxy -run 'Capacity|Security|NodePacking|GPU|Cost' -count=1`
  - 게이트: `gofmt -l <수정 Go 파일>`(출력 없음) → `go build ./...` → `go vet ./...` → `go test ./... -count=1` (proxy 약 66~98s, store 약 17~27s 소요)
- 위험과 피할 것:
  - `analyzer/capacity.go:podRequestGPU` 는 `cost.go:BuildCostForecast/EstimateCost` 가 공유한다. 이번 과제는 **조회 범위와 응답 메타데이터만** 바꾼다 — 집계·금액 로직 금지.
  - 용량 Idle 은 `allocatable-requested` 로 음수가 될 수 있고 노드 모니터링 Available 은 0 clamp 다. 두 계약을 통합하지 말 것(과거 지시).
  - `report`·`note` 키 이름·의미와 `report.gpu` 의 배열 형태를 바꾸면 UI 가 깨진다. 키는 더하기만.
  - 메트릭 조회(`ListK8sMetricSamples`, `admin_k8s.go:597`)의 버려지는 err 은 **이번 범위 밖**(차선 후보). 같이 손대서 파일/의미를 넓히지 말 것.
  - auth(`authorizeAdmin`)·migrations·DDL·workflows 경로 금지. `securityScanBudget` 과 security 핸들러는 읽기 참고만 하고 수정하지 말 것.
  - 미확인: 실제 PostgreSQL·실 Kubernetes·브라우저 UI 렌더는 확인하지 않았다. 4000행 포화를 실제 운영 데이터로 관측한 것이 아니라 코드상 `updated_at DESC` + Kind 미필터로부터 추론한 것이다(security 핸들러 주석이 같은 결함을 명시적으로 기록하고 있다는 점이 근거).
- 차선 후보: 용량 API 메트릭 조회 오류를 정상 무표본 응답과 구별해 표시 (3/2/S) — `admin_k8s.go:597` 의 `metrics, _ :=` 가 오류를 버려 DB 고장이 "표본 없음" 과 같은 200 을 낸다. 1순위가 성립하지 않으면 이것을 하되, 두 개를 한 회차에 합치지 말 것.
