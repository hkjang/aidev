# v0.305.0 — 내 기록 시험이 달력이 아니라 마감 규칙이 닫은 주를 셉니다

제품의 코드는 v0.304.0 과 같습니다. 이번 판이 고치는 것은 **월요일마다 실패하던 시험** 하나입니다.

`TestMyRecordCountsTheWeeksTheArrearsListCounts` 는 세 주를 연달아 낸 사람의 기록이 `연속 3주` 로 읽히는지 봅니다. 시험은 "지난주" 를 **달력**으로 세었습니다 — 이번 주 시작에서 7일을 뺀 주. 그런데 기록에 드는 주는 **마감이 지난 주**뿐이고, 마감은 달력이 아니라 규칙이 정합니다. 기본 규칙(`workflow.deadline_days=7`, `deadline_hour=24`)에서 지난주의 마감은 `시작일 + 7일 + 24시간` = 이번 주 **화요일 0시**라, 월요일에는 지난주가 온종일 아직 열려 있습니다. `myParticipation` 은 열린 주를 기록에 넣지 않으므로(그것이 바로 지키려던 규칙) 낸 세 주 가운데 하나가 빠져 `연속 2주, want 3` 이 났습니다. 코드가 옳고 시험이 틀린 경우라 `participation.go` 는 손대지 않았습니다.

## 규칙이 닫은 주부터 셉니다

- 시험은 `closedWeekStart(now, rule)` 로 서버의 `deadlineRule` 을 읽어 **마감이 지난 가장 최근 주**를 찾고, 거기서부터 `week(1..8)` 을 셉니다. 셈은 SQL 의 `deadlinePassedFor` 와 같은 `시작일 + days + hours <= now` 이고 서비스 시간대를 따릅니다.
- 같은 시나리오를 `workflow.deadline_days=13` 으로 **한 번 더** 돌립니다. 13일이면 일요일 밤에도 지난주의 마감이 내일이라, 오늘이 무슨 요일이든 "지난주가 아직 열린" 경로가 시험에 듭니다 — 월요일이 오기를 기다리지 않아도 됩니다. 이 서브테스트는 지난주가 정말 열려 있는지 먼저 확인하고 아니면 스스로 실패합니다.

## 검증

새 서브테스트에서 앵커를 옛 것(`current.AddDate(0,0,-7*back)`)으로 되돌리면 금요일인 오늘도 정확히 원장의 문장(`연속 2주, want 3`, 두 곳)으로 실패하고, 고친 앵커로는 두 규칙 모두 통과합니다. 실제 PostgreSQL 위에서 전체 Go 테스트와 `go vet`·`gofmt`, 가드 검사(`--changed`, `myParticipation` 도달), OpenAPI·모달·버전 검사를 통과했고, `mutation-check --test TestMyRecordCountsTheWeeksTheArrearsListCounts` 가 적용한 변이 셋을 모두 잡았습니다(caught 3 / survived 0).

## 업그레이드

마이그레이션도, 새 환경변수도, 응답 모양의 변경도 없습니다. 바뀐 파일은 시험 하나(`internal/app/participation_test.go`)뿐이라 제품의 동작은 v0.304.0 과 같습니다. 기존과 같이 PostgreSQL 을 백업한 뒤 v0.305.0 이미지를 올리면 됩니다.
