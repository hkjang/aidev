# 회차 노트 2026-09-22-050431-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:04] base pinned — main@8b05683
- [러너 05:04] autonomy release — 

## 정찰 노트
- NodePressure의 nodeName 단독 조인을 확인해 선택: 운영 영향 범위가 바로 바뀌며 공용 분석기 한 곳 수정이라 인증·PSS·알림 상태 변경보다 위험이 낮다.
- RCA 자원 태그는 해결(done); 기존 보류 항목 유지·재평가 및 신규 README 링크/비활성 알림 dedup 후보를 추가했다.
- 신규 실패 재현은 미실행(읽기 전용 정찰); 회사 스킬 3종·Skill 도구와 09-07 반려의 구체 접근은 미확인이다.
- 구현자는 빈 ClusterID를 wildcard로 쓰지 말고 실제 SQLite·변환기·Server.Routes로 RCA/홈 증적과 영향 수를 검증한다.
