## 2026-09-22
- 선택: NodePressure 영향 Pod 집계를 클러스터별로 격리 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: analyzeNodeConditions의 Pod 인덱스 생성·조회 모두 기존 nodeKey(ClusterID,nodeName)를 사용하도록 수정해 동명 노드의 타 클러스터 Pod가 영향 수와 증적에 섞이지 않게 했다. AnalyzeRCA 회귀(빈 ID, 역순, 다른 노드·미배치 Pod, True 압박 조건, high severity, 0개·7개 영향/최대 5개 증적)와 실제 SQLite·InventoryFromObject·Server.Routes·HTTP RCA/홈 회귀가 수정 전에 영향 수 6 및 3/3과 외부 Pod 증적으로 실패하고 수정 후 1/2 및 단일 클러스터 결과를 검증했다. go test ./internal/analyzer ./internal/proxy, go build ./..., go vet ./..., go test ./..., gofmt·diff 검사 모두 통과하여 5b98765로 커밋했으며 요청된 technology 스킬 3종·Skill 도구는 카탈로그 및 로컬 검색에서 미발견(전용 절차·반환 형식 미확인), 실 Kubernetes·PostgreSQL·브라우저는 미검증이다.
- 보류 아이디어:
  - notify scan Pod Security dedup 키에 Kind 포함 (가치 2 / 위험 1 / 작업량 S) — 기존 6시간 키·일회성 중복 영향 검토 필요.
  - 알림 quiet_hours 숫자 범위 검증 (가치 3 / 위험 2 / 작업량 S) — 24시 종료 계약 확인 필요.
  - 운영 홈 인벤토리 조회 상한 도달 시 분석 범위 안내 (가치 2 / 위험 1 / 작업량 M) — 조회 상한·UI 계약은 이번 범위 밖.
  - 알림 비활성 상태의 scan이 dedup 창을 소모하지 않도록 수정 (가치 3 / 위험 2 / 작업량 M) — 실제 webhook·활성화 전후 회귀 필요.
- 과제서: 채택 — nodeName 단독 조인이 현 코드에서 확인됐고 지정된 실제 분석기·HTTP 경로에서 교차 클러스터 오염을 재현했다.
