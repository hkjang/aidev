## 2026-10-07 — 수정 과제
- 선택: 요청 진행 알림 API 시험의 시각 기반 메일 판정을 새 메일 ID로 전환(가치 5 / 위험 2 / S).
- 상태: 정찰 완료, 구현 대기. 코드 수정·커밋·릴리즈 없음.
- 근거: 직전 verify.txt의 tests/api-bulk.spec.ts:20 메일 없음, 118 passed / 1 failed / 3 did not run. 실제 helper+mock HTTP에서 5초 이른 수신 시각으로 같은 거짓 음성 및 ID 방식 성공을 진단했다. 당시 원인이 시계였는지는 미확인.
- 현재 검증: npm test --silent exit 1(@playwright/test 의존성 누락); 업무 API 재현과 수정 후 전체 통과는 구현자가 기록해야 한다. 성공으로 기록하지 않는다.
