# 회차 노트 2026-09-20-185404-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:54] base pinned — main@10621cc
- [러너 18:54] autonomy release — 

## 정찰 노트
- RCA 이벤트·리비전 클러스터 격리를 선택: 장애 누락·잘못된 severity 상승을 한 과제로 해결하며 인증/PSS/스키마 변경이 없다. notify kind 충돌보다 영향이 크다.
- rcaKey는 workload.go도 공유한다. TestEnrichWithConfigChanges의 누락 ClusterID는 fixture에서 보정하고 빈 클러스터 wildcard로 회피하지 말 것.
- 실클러스터 재현·회사 스킬 절차는 미확인; 코드 근거와 예상 35분+예비 10분을 brief에 기록. 자원 태그/NodePressure 조인은 별도 후보로 범위를 제한했다.
- 프로필의 DB 설명을 기본 SQLite·PostgreSQL 지원으로 정정; 18개 아이디어를 유지/재평가했고 신규 3개 및 이전 해결 1개를 기록했다.
