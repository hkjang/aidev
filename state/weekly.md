## 2026-09-02
- 선택: 주 시작 요일 변경 후 이미 제출한 팀원에게 작성 권고 메일이 가는 버그 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공 — PR https://github.com/hkjang/weekly/pull/1 (예산 $3 소진으로 에이전트가 원장을 못 남겨 러너가 대신 기록)
- 요약: queueTeamReminders/reminderStillWanted 가 week_start 정확일치로 묻던 것을 weekCoveringDays(기간 겹침)로 통일. 회귀 테스트 2개 추가. 저장소 루트에 잘못 들어온 middleware.go 사본 파일 삭제. WEEKLY_TEST_POSTGRES_DSN 으로 실제 DB 테스트 통과 확인.
- 보류 아이디어: (기록 전 예산 소진)
