# 회차 노트 2026-10-16-001846-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:18] base pinned — main@5c5beef
- [러너 00:18] autonomy release — 

## 정찰 노트
- 용량 API의 Kind 미필터 조회를 골랐다: 바로 옆 `/admin/k8s/security` 가 **같은 결함을 이미 고쳐 둔 선례**(`Kinds`+`budget+1`+`scan`)가 있어 구현자가 설계를 발명할 필요가 없고, 틀린 출력(노드·GPU 행 누락)을 고치는 일이다. 알림 경로(quiet 감사·팬아웃 500)는 4회 연속 변경된 영역이라 냉각시켰고, inventory GET 메타데이터·dedup Kind 는 반복 차선 유형이라 제쳤다.
- 확인한 것: `AnalyzeCapacity` 가 읽는 Kind 세 개(HPA·Pod·Node), `ORDER BY i.updated_at DESC`, `Kinds` 필터 동작, `UpsertK8sInventory` 가 `item.UpdatedAt` 을 존중(결정적 재현 가능), 기준선 테스트 통과(analyzer 0.005s / proxy 0.119s).
- 추측으로 적은 것: 4000행 포화를 실제 운영 데이터로 관측하지는 못했다 — 코드 구조와 security 핸들러 주석의 자기 기록에서 추론했다. 포화 임계값(4000 유지)과 응답 키 이름은 구현자 재량.
- 구현자가 조심할 것: `report`·`note` 키와 `report.gpu` 배열 형태는 UI가 직접 렌더하니 추가만 할 것. `podRequestGPU` 는 `cost.go` 와 공유되므로 집계 로직 금지. 같은 함수 안의 메트릭 err 폐기는 차선 후보로 분리했으니 합치지 말 것.
- [러너 00:22] scout done — 용량 API의 인벤토리 조회를 분석 대상 Kind로 한정하고 창 포화를 보고하기 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- 무엇을 왜: `handleK8sCapacity` 의 인벤토리 조회를 `analyzer.CapacityRelevantKinds()`(HPA·Node·Pod)로 좁히고 `capacityScanBudget+1` 로 받아 초과 시 잘라낸 뒤 `scan{status,resources,reason}` 을 응답에 더했다. 비대상 Kind 가 `updated_at DESC` 창을 채워 Node·Pod 가 밀려나면 node_packing·gpu·projections 가 조용히 빈 배열이 되던 것을 고친 것이다. `report`·`note` 키와 집계·금액 로직은 불변(프로덕션 2파일).
- 확신 없는 곳·검증 못 한 것: (1) 4000행 포화를 실 운영 데이터로 관측한 것이 아니다 — 테스트는 `capacityScanBudget` 을 2로 낮춰 같은 구조를 재현했다. (2) 브라우저 UI 에서 새 `scan` 키가 기존 렌더를 깨지 않는지는 코드상 추가-only 라는 근거뿐이고 실제 렌더는 확인하지 않았다. (3) 실 PostgreSQL·실 Kubernetes 미확인(SQLite 경로만). (4) 포화 안내 문구의 한국어 표현은 security 쪽 결을 따랐을 뿐 운영자 검토를 받지 않았다.
- 일부러 하지 않은 것: 같은 함수의 `metrics, _ :=` 오류 폐기(과제서가 차선으로 분리 — 합치면 범위가 넓어진다), `securityScanBudget`·security 핸들러(읽기 참고만), `handleK8sConnectivity` 의 동일 유형 Kind 미필터(별도 회차 후보로 ideas.json 에 기록), 기존 분석 함수 로직.
- 다음 역할이 조심할 것: 신규 proxy 테스트는 실 SQLite(`openTestStore`)와 `httptest` 서버가 떠야 돈다. `capacityScanBudget` 을 패키지 var 로 둔 이유가 테스트의 포화 경로이므로 상수로 바꾸면 `admin_k8s_capacity_scope_test.go` 가 컴파일되지 않는다. 빨강 확인은 var 추출(동작 불변)만 먼저 넣고 Kinds·scan 을 빼 둔 상태에서 했고, 빨강→초록 사이의 차이는 그 한 훅뿐이다.
- [러너 00:29] brief accepted — 채택 — 과제서의 근거(Kind 미필터 `Limit: 4000`, `updated_at DESC`, `AnalyzeCapacity` 의 세 Kind, `UpsertK8sInventory` 가 `item.UpdatedAt` 을 �
- [러너 00:30] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인한 것: 핸들러에서 Kinds·scan 훅만 되돌려 새 테스트의 빨강을 직접 재현(`node_packing = []`, `gpu = []`, `no scan block`) — 원장의 실패 출력과 일치. gofmt/build/vet/`go test ./...` 전부 통과. capacity.go의 Kind 분기(88·169·244·260)가 세 Kind로 전부 덮이는지 확인해 입력 축소 회귀 없음을 확인. `boundedLimit(…,200,10000)`이라 4001 안 잘림, Kinds는 상수 바인드라 주입 없음, authorizeAdmin·report·note 키 불변 → 보안·법무 차단 없음. 판정 approve(risk low).
- 못 본 것: 실 PostgreSQL·실 Kubernetes·브라우저 렌더(코드 읽기만), 4000행 실제 포화 관측.
- 승인이어도 남는 우려 ①: 새 `scan` 을 용량 화면이 렌더하지 않는다 — 보안은 `admin_ui.go:13477` 에서 `scanNotice(data.scan)` 를 붙이지만 `renderK8sCapacity`(admin_ui.go:15276-15293)·노드 관리(11375)는 읽지 않아 포화 경고가 SPA 운영자에게 안 보인다. 다음 회차 1순위 후보.
- 우려 ②: `analyzer/capacity_kinds_test.go` 는 동어반복에 가까워 AnalyzeCapacity 가 새 Kind 를 읽어도 깨지지 않는다(동작 증거는 proxy HTTP 테스트). 포화 시 `updated_at DESC` 동률 타이브레이크 없음은 security 선례와 동일한 기존 성질.
- 릴리즈가 할 일: K8S_OPERATIONS_HUB.md 에 이번 항목 문단이 없다 — 08cfc7f 는 fix 커밋에, 5c5beef 는 release 커밋에 문서를 넣었으니 릴리즈 단계에서 보완할 것.
- [러너 00:34] review approved — 리뷰 승인 (risk=low)
- [러너 00:34] pr created — https://github.com/hkjang/clustara/pull/40
- [러너 00:35] ci passed — 검사 없음 — 정책으로 허용
- [러너 00:35] merge done — de54f71
