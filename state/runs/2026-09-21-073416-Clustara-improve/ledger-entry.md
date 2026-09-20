## 2026-09-21
- 선택: 운영 홈 RCA 자원 태그를 클러스터별로 정확히 연결 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: AttachFindingResources의 인덱스 생성과 조회에 기존 rcaKey로 ClusterID를 포함해 동명 Pod의 CPU·메모리 태그가 다른 클러스터 값으로 덮이던 결함을 수정하고 fixture와 부정확한 owner fallback 주석을 정정했다. 신규 analyzer 회귀(입력 역순·빈 ID·namespace/kind/name 구분·nil 유지·Deployment template)와 실제 SQLite·Server.Routes·HTTP 홈 회귀가 수정 전 잘못된 동일 태그로 실패하고 수정 후 통과했으며, go test ./internal/analyzer ./internal/proxy → go build ./... → go vet ./... → go test ./... 및 수정 파일 gofmt·diff 검사를 통과하여 7a66433으로 커밋했다. 회사 technology 스킬 3종과 Skill 도구는 카탈로그·로컬 검색에서 미발견하여 전용 반환 형식은 미확인이며 로컬 superpowers의 systematic-debugging/test-driven-development를 참고했다; 실 Kubernetes·PostgreSQL·브라우저 렌더링은 미검증이다.
- 보류 아이디어:
  - NodePressure 영향 Pod의 클러스터별 집계 (가치 3 / 위험 1 / 작업량 S) — 별도 차선 과제로 유지.
  - notify scan Pod Security dedup 키에 Kind 포함 (가치 2 / 위험 1 / 작업량 S) — 기존 6시간 기록 영향 검토 필요.
  - 알림 quiet_hours 숫자 범위 검증 (가치 3 / 위험 2 / 작업량 S) — 24시 종료 계약과 입력·실행 경로 검토 필요.
  - 운영 홈 인벤토리 조회 상한 도달 시 분석 범위 안내 (가치 2 / 위험 1 / 작업량 M) — 상한 안내 및 UI 계약 검토 필요.
- 과제서: 채택 — 현재 코드의 ClusterID 누락과 홈 API의 잘못된 자원 태그를 실제 회귀로 확인하여 지정된 3개 파일만 수정했다.
