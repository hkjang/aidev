## 2026-09-02
- 선택: 주 시작 요일 변경 후 이미 제출한 팀원에게 작성 권고 메일이 가는 버그 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공 — PR https://github.com/hkjang/weekly/pull/1 (예산 $3 소진으로 에이전트가 원장을 못 남겨 러너가 대신 기록)
- 요약: queueTeamReminders/reminderStillWanted 가 week_start 정확일치로 묻던 것을 weekCoveringDays(기간 겹침)로 통일. 회귀 테스트 2개 추가. 저장소 루트에 잘못 들어온 middleware.go 사본 파일 삭제. WEEKLY_TEST_POSTGRES_DSN 으로 실제 DB 테스트 통과 확인.
- 보류 아이디어: (기록 전 예산 소진)
- 릴리즈: v0.281.0 (2026-09-02)

## 2026-09-02 (2회차)
- 선택: 주 격자가 옮겨진 전환 주에 작성 화면이 기존 보고서를 열지 못하는 버그 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `currentReport`(GET /api/v1/reports/current)가 `week_start` 정확일치로만 찾아, 주 시작 요일 변경 후 전환 주에는 작성자의 보고서가 화면에서 사라지고 그 자리의 빈 편집기가 저장 때마다 409 REPORT_PERIOD_OVERLAPS 만 돌려줬습니다(weekIsFree 는 이미 기간 겹침으로 막고 있었으므로 두 화면이 서로 다른 규칙을 쓰던 셈). 조회를 `weekCoveringDays` 로 통일하고 회귀 테스트 1개(`currentweekgrid_test.go`, guards: currentReport/weekCoveringDays)를 추가했습니다. 검증: 실제 DB(WEEKLY_TEST_POSTGRES_DSN)로 `go test ./...` 통과, `go vet`, guard-check(1개 도달), mutation-check(전부 캐치), modal-close/version/openapi-check 통과, frontend lint·build·test(121개) 통과. 수정 전 코드에서 새 테스트가 실패하는 것도 확인했습니다.
- 보류 아이디어:
  - 격자 이동 후 `runAutomaticCloneForUser` 의 원본 조회가 `week-7` 정확일치라 자동 복제가 한 주 조용히 건너뜁니다 (가치 3 / 위험 2 / 작업량 S)
  - `buildMailMessage` 의 From 표시이름이 순수 ASCII면 인코딩되지 않아 `,`·`<` 가 든 이름이 From 헤더를 깨뜨립니다 (가치 2 / 위험 1 / 작업량 S)
  - `currentIncludedMaterials` 도 같은 전환 주 문제를 갖는지 확인 (가치 2 / 위험 2 / 작업량 S)
  - `outlookForDueDate` 의 AT_RISK 문구가 low==high 일 때 최근 속도를 빼고 전체 평균만 말합니다 (가치 1 / 위험 1 / 작업량 S)
- 릴리즈: v0.282.0 (2026-09-02)
