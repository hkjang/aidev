# 회차 노트 2026-10-08-143828-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:38] base pinned — main@879082e
- [러너 14:38] autonomy release — 

## 정찰 노트
- 선택: 실제 NewRouter/WebSocket/DB를 통과하는 수동 DND·away 보존 테스트. 기존 audience 테스트의 직접 handlers/hub.Register 배선 공백을 메우며, 상태코드·본문 파서 수정 반복을 피한다. 프로덕션 0파일, 테스트 1파일, 40분+5분 예비.
- 후보 15개(기존 12+신규 3)를 재평가했다. 외부 러너 로그 접근은 저장소 범위 밖이라 rejected; 나머지는 pending. 북마크 reorder 주석/구현 불일치는 외부 계약 미확인이라 보류했다.
- 확인: 비DB httpapi/ws 및 소스 크기 검사 통과; 실제 PostgreSQL에서 기존 NewRouter·presence 테스트 3.161s PASS/skip 0. 새 lifecycle 시나리오 자체와 전체 race/웹/릴리즈는 미실행이며 현재 제품 결함을 주장하지 않는다.
- 주의: 비동기 콜백 완료를 ClientCount/sleep으로 추측하지 말고 PUT 이벤트를 소비한 뒤 lifecycle 이벤트를 관찰한다. 회사 스킬 3개 미발견을 brief에 기록. 코드·커밋 변경 없이 초안→최종 brief 및 9일 된 profile을 갱신했다.
- [러너 14:44] scout done — 실제 WebSocket 연결·해제가 수동 DND/away 상태를 보존하는 통합 회귀 테스트 (가치 3 / 위험 1 / 작업량 M)
