## 2026-09-20
- 선택: RCA 이벤트·리비전 연결과 중복 제거를 클러스터별로 격리 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: RCA와 Deployment/Job의 이벤트 키·probe 중복 제거·최근 리비전 선택에 ClusterID를 포함하고, workload 메시지 fallback 및 배포 후 Warning 연결도 동일 클러스터로 제한했다. 신규 회귀 6개(analyzer 5·proxy 1)가 수정 전 증적 혼합·잘못된 severity 상승·finding 누락·알림 sent=1로 실패함을 확인했으며, 수정 후 analyzer/proxy 테스트·go build ./...·go vet ./...·최종 go test ./... 및 수정 5개 파일 gofmt 검사가 모두 통과했다. 빈 ClusterID의 독립성, 같은 클러스터 반복 이벤트 및 재스캔 알림 중복 억제, lookback·created 제외·배포 이전/Normal 제외를 검증했고 158c620으로 커밋했다(실클러스터·실 PostgreSQL은 미검증; 요청된 회사 스킬 3종과 Skill 도구는 카탈로그 및 로컬 검색에서 미발견).
- 보류 아이디어:
  - RCA 자원 태그의 클러스터별 조인 (가치 3 / 위험 1 / 작업량 S) — AttachFindingResources 별도 수정 필요.
  - NodePressure 영향 Pod의 클러스터별 집계 (가치 3 / 위험 1 / 작업량 S) — 노드 이름 조인은 이번 범위 밖.
  - notify scan Pod Security dedup 키에 Kind 포함 (가치 2 / 위험 1 / 작업량 S) — 기존 6시간 기록과 일회성 불일치 영향 검토.
  - notify scan 인벤토리 Limit 2000·kind 범위 개선 (가치 3 / 위험 2 / 작업량 S) — RCA Node를 보존하는 조회 범위 설계 필요.
- 과제서: 채택 — 공유 키와 직접 이벤트 필터의 클러스터 누락이 현 코드에서 재현되어 지정한 5개 파일만 수정했다.
