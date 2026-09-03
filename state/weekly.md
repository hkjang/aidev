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

## 2026-09-02 (3회차)
- 선택: 주 격자가 옮겨진 전환 주에 포함한 팀원 주간보고가 "미작성"으로 보이는 버그 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `includedMaterialsFor` 가 팀원 보고서를 `report.week_start=$2::date` 정확일치로만 붙여, 주 시작 요일 변경 후 전환 주에는 이미 작성한(그리고 weekIsFree 때문에 새 날짜로 다시 쓸 수도 없는) 팀원이 보고서 없음으로 돌아왔습니다. PPTX 는 그것을 이름 옆 "주간보고 미작성 / 해당 주차 주간보고가 없습니다" 로 바꿔 경영 보고 자리에 올립니다. 조회를 `weekCoveringDays` 기준 LATERAL 조인(겹치는 것 중 `week_start DESC` 하나)으로 바꿔 currentReport·weekIsFree 와 규칙을 통일했고, `currentIncludedMaterials` 와 `loadReport` 두 경로가 같은 함수를 쓰므로 한 번에 고쳐집니다. 회귀 테스트 1개(`currentweekgrid_test.go`, guards: includedMaterialsFor/weekCoveringDays)를 추가했고, 안 쓴 주가 다른 주 보고서로 채워지지 않는 것도 같이 확인합니다. 검증: 수정 전 코드에서 새 테스트가 실패하는 것을 확인 → 실제 DB(WEEKLY_TEST_POSTGRES_DSN)로 `go test ./...` 전체 통과, `go vet`, guard-check(새 가드가 두 대상 모두 도달), modal-close/version/openapi-check 통과, frontend lint·build·test(121개) 통과. mutation-check 잔존 변이 4건은 모두 기존 권한 분기(294·333·334행)로 예산 초과로 건너뛴 기존 가드가 담당하며 이번 변경분이 아닙니다. backup-check 는 로컬에 psql 이 없어 건너뛰었습니다(CI 에서 실행).
- 보류 아이디어:
  - 격자 이동 후 `runAutomaticCloneForUser` 의 원본 조회가 `week-7` 정확일치라 자동 복제가 한 주 조용히 건너뜁니다 (가치 3 / 위험 2 / 작업량 S)
  - `buildMailMessage` 의 From 표시이름이 순수 ASCII면 인코딩되지 않아 `,`·`<` 가 든 이름이 From 헤더를 깨뜨립니다 (가치 2 / 위험 1 / 작업량 S)
  - `meeting.go` 의 지난주 조회(`AddDate(0,0,-7)`)도 같은 전환 주 정확일치 문제를 갖는지 확인 (가치 2 / 위험 2 / 작업량 S)
  - `outlookForDueDate` 의 AT_RISK 문구가 low==high 일 때 최근 속도를 빼고 전체 평균만 말합니다 (가치 1 / 위험 1 / 작업량 S)
- 릴리즈: v0.283.0 (2026-09-02)

## 2026-09-03
- 선택: 주 격자를 옮긴 다음 주에 지난주 자동 복제가 조용히 건너뛰던 버그 수정 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `runAutomaticCloneForUser` 는 이번 주 충돌은 이미 날짜 겹침으로 묻고 있었는데 원본만 `week-7` 정확일치로 찾아, 주 시작 요일을 바꾼 다음 주에는 작성자가 이어 쓰던 전환 주 보고서(옛 격자 날짜)를 못 찾고 그 주를 processed 로 표시한 뒤 초안을 만들지 않았습니다. 원본 조회를 `weekCoveringDays` 겹침 + `week_start DESC LIMIT 1` 로 바꿔 두 반쪽의 규칙을 통일했고, 회귀 테스트 1개(`currentweekgrid_test.go`, guards: runAutomaticCloneForUser/weekCoveringDays)를 더했습니다. 전환 주 자체에는 겹치는 보고서가 있으므로 여전히 복제하지 않는다는 것도 같은 테스트가 확인합니다. 관리자 안내서의 "정확히 한 주 전 보고서" 문장과 자동화 표를 제품에 맞게 고치고 HTML·PDF 를 다시 만들었습니다. 검증: 수정 전 코드에서 새 테스트가 실패하는 것을 확인 → 실제 DB(WEEKLY_TEST_POSTGRES_DSN)로 `go test ./...` 전체 통과, `go vet`, guard-check 279개 도달, version·openapi·modal-close 검사 통과, frontend lint·build·test(121개) 통과. mutation-check 는 새 가드에서 잔존 변이 3건(287·304·330행)인데 모두 이번에 바꾸지 않은 "쓸 것이 없는" 분기(설정 꺼짐/이미 처리/커밋 생략)이고, 바꾼 조회 경로의 변이 3건은 모두 잡혔습니다. backup-check 는 로컬에 psql 이 없어 건너뛰었습니다(CI 에서 실행).
- 보류 아이디어:
  - `buildMailMessage` 의 From 표시이름이 순수 ASCII면 인코딩되지 않아 `,`·`<` 가 든 이름이 From 헤더를 깨뜨립니다 (가치 2 / 위험 1 / 작업량 S)
  - 전환 주에는 `previousWeekPlan` 이 `week_start < target` 로 물어 작성자가 지금 쓰고 있는 그 보고서를 "지난주 계획" 으로 되돌려 줍니다 (가치 3 / 위험 2 / 작업량 S)
  - `meeting.go` 의 지난주 조회(`weekBefore`)도 같은 전환 주 정확일치 문제를 갖는지 확인 (가치 2 / 위험 2 / 작업량 S)
  - `outlookForDueDate` 의 AT_RISK 문구가 low==high 일 때 최근 속도를 빼고 전체 평균만 말합니다 (가치 1 / 위험 1 / 작업량 S)
- 릴리즈: v0.284.0 (2026-09-03)

## 2026-09-03 (2회차)
- 선택: 전환 주에 `previousWeekPlan` 이 지금 쓰고 있는 보고서를 "지난주 계획"으로 되돌려 주던 버그 수정 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 작성 화면 옆 "지난주 계획" 패널이 `week_start < target` 로 물어, 주 시작 요일을 바꾼 전환 주에는 작성자가 이어 쓰는 바로 그 보고서(옛 격자의 이른 날짜)를 지난주 계획으로 돌려줬습니다 — 눈앞의 업무가 자기 자신과 짝지어지고, 이미 100% 로 보고한 것까지 아직 남은 일로 다시 제안됩니다. 정작 한 주 전 계획은 한 줄 아래에 있어 닿지 않았습니다. "이전"을 날짜가 이른 것이 아니라 이번 주가 시작하기 전에 일곱 날이 끝난 것으로 바꾸고(`weekCoveringDays` 의 나머지 반쪽인 `weekEndedBefore` 를 `weekstart.go` 에 추가), 회귀 테스트 1개(`currentweekgrid_test.go`, guards: previousWeekPlan/weekEndedBefore)를 더했습니다. 이 끝점에는 그동안 통합 테스트가 하나도 없었습니다. 검증: 수정 전 코드에서 새 테스트가 실패하는 것을 확인 → 실제 DB(WEEKLY_TEST_POSTGRES_DSN)로 `go test ./...` 전체 통과, `go vet`, guard-check(두 대상 모두 도달), version·openapi·modal-close 검사 통과, frontend lint·build·test(121개, 두 시간대) 통과. mutation-check 는 바꾼 조회 경로의 변이를 모두 잡았고 잔존 1건은 이번에 건드리지 않은 `weekStart` 질의 인자 파싱의 400 분기(80행)입니다. backup-check 는 로컬에 psql 이 없어 건너뛰었습니다(CI 에서 실행).
- 보류 아이디어:
  - `buildMailMessage` 의 From 표시이름이 순수 ASCII면 인코딩되지 않아 `,`·`<` 가 든 이름이 From 헤더를 깨뜨립니다 (가치 2 / 위험 1 / 작업량 S)
  - `meeting.go` 의 지난주 조회(`weekBefore`)도 같은 전환 주 정확일치 문제를 갖는지 확인 (가치 2 / 위험 2 / 작업량 S)
  - `issueoutcome.go` 의 `r.week_start < $2` 도 전환 주에 같은 주 보고서를 "이전 결과"로 셈하는지 확인 (가치 2 / 위험 2 / 작업량 S)
  - `previousWeekPlan` 의 `weekStart` 질의 인자 파싱 400 분기에 가드가 없습니다 (가치 1 / 위험 1 / 작업량 S)
  - `outlookForDueDate` 의 AT_RISK 문구가 low==high 일 때 최근 속도를 빼고 전체 평균만 말합니다 (가치 1 / 위험 1 / 작업량 S)
- 릴리즈: v0.285.0 (2026-09-03)
