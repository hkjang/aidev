## Relio v1.11.17 — 계약 기간 오타 하나가 수만 건의 매출 일정을 쓰고 일정 탭을 영구히 막던 문제

계약의 기간과 인식 주기는 **함께** 활성화 한 번이 쓰는 매출 일정 행 수를 정합니다. 그런데 그 조합에는 상한이 없었습니다. 종료일을 2025 대신 2205 로 잘못 적어도 그대로 통과했고, 잘못은 이미 수만 건이 기록된 뒤에야 드러났습니다. 이번 릴리즈는 `buildScheduleDates` 가 만들 수 있는 일정 항목 수에 상한을 두고, 넘길 때 두 입력 중 무엇을 바꿔야 하는지 밝히며 거절하도록 했습니다.

## 1. 원인

### 1-1. 기간과 주기의 곱에 상한이 없었다

`buildScheduleDates` 는 시작일부터 종료일까지 주기(`MONTHLY` 1개월, `QUARTERLY` 3개월, `ANNUAL` 12개월)만큼 건너뛰며 날짜를 만들고, 종료일을 넘어설 때 멈춥니다. 멈추는 조건은 종료일 하나뿐이었습니다.

```go
dates := []time.Time{}
for n := 0; ; n++ {
	date := addMonthsClamped(*start, n*step)
	if date.After(*end) {
		break
	}
	dates = append(dates, date)
}
```

`end_date` 는 DATE 컬럼이므로 9999-12-31 까지 유효한 값입니다. 2024-01-01 에 시작해 9999-12-31 까지 가는 월별 계약이면 이 반복문은 **95,712 개**의 날짜를 만들고, 그 전부가 정상적인 반환값이었습니다.

### 1-2. 그 날짜 하나하나가 활성화 트랜잭션 안의 개별 INSERT 였다

`createRevenueSchedule` 은 만들어진 날짜를 그대로 순회하며 한 건씩 씁니다.

```go
for i, date := range dates {
	_, err = tx.Exec(ctx, `INSERT INTO revenue_schedules(...) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, ...)
```

즉 오타 하나가 하나의 트랜잭션 안에서 9만 번이 넘는 INSERT 가 되었습니다. 계약을 `ACTIVE` 로 곧바로 생성하는 경로와 `DRAFT` 를 활성화하는 경로가 모두 이 함수를 지납니다.

### 1-3. 일정 목록에는 자체 LIMIT 이 없어 되돌릴 화면조차 열리지 않았다

`ListRevenueSchedules` 는 해당 계약의 행을 조건 없이 전부 돌려줍니다.

```go
rows, err := s.DB.Query(ctx, `SELECT ... FROM revenue_schedules WHERE contract_id=$1 ORDER BY sequence_no`, contractID)
```

목록에 상한이 없으므로, 잘못 활성화된 계약의 일정 탭은 그 뒤로 영영 열리지 않았습니다. 실수를 발견하는 자리와 그 실수를 고치는 자리가 같은 화면이었는데, 실수가 그 화면을 먼저 망가뜨린 것입니다.

## 2. 수정

일정 항목 수에 상한을 두고, 상한에 닿는 순간 날짜를 하나도 돌려주지 않고 거절합니다.

```go
// maxScheduleEntries bounds how much work one activation may ask for. ...
// Fifty years of monthly recognition is already far past any real contract, so
// that is where the line sits.
const maxScheduleEntries = 600
```

```go
if len(dates) == maxScheduleEntries {
	return nil, fmt.Errorf("contract period is too long for a %s revenue schedule: it would create over %d entries, so shorten the period or choose a less frequent revenueScheduleType", scheduleType, maxScheduleEntries)
}
```

600 은 월별 인식 50년입니다. 실제 계약보다 훨씬 긴 값이므로 정상 업무를 막지 않고 오타만 걸러냅니다.

검사를 `buildScheduleDates` 안에 둔 것이 핵심입니다. 일정을 쓰는 두 경로 — `ACTIVE` 로 생성, `DRAFT` 활성화 — 가 모두 이 함수를 지나고, `ActivateContract` 는 트랜잭션을 열기 **전에** 같은 함수로 사전 검증을 합니다. 따라서 사전 검증과 실제 쓰기가 언제나 같은 기준을 씁니다. 호출부마다 따로 검사를 붙였다면 다음에 추가되는 경로가 또 빠뜨릴 수 있습니다.

오류 문구도 진단이 아니라 지시입니다. 무엇이 문제인지(기간이 이 주기에 비해 너무 김)와 무엇을 바꾸면 되는지(기간을 줄이거나 더 성긴 `revenueScheduleType` 을 고르거나) 둘 다 말합니다.

## 3. 회귀 방지

- 정상 계약이 그대로 활성화되는지: 3년 월별 계약이 36건을 만듭니다. 상한은 오타를 위한 것이지 실제 계약을 막기 위한 것이 아닙니다.
- 상한을 넘기는 기간이 거절되는지: 9999년까지 가는 월별·분기·연간, 그리고 상한보다 정확히 한 달 긴 기간. 거절될 때 날짜를 하나도 돌려주지 않는 것까지 확인합니다.
- 거절 문구가 오분류되지 않는지: `serviceError` 는 메시지의 부분 문자열로 HTTP 상태를 정하므로, 이 문구가 `not found`·`permission`·`already` 같은 조각을 담아 404/403/409 로 바뀌지 않는지 확인합니다. 400 이어야 클라이언트가 입력을 고칠 수 있습니다.
- 경계값이 정확한지: 정확히 600건인 기간은 통과해야 합니다.

상한을 빼면 이 테스트들이 실제로 실패하는 것을 확인했습니다 — 95,712건이 그대로 통과한다고 출력합니다.

## 4. 적용

마이그레이션은 없습니다. 이미 기록된 일정 행은 그대로 두며, 새 활성화와 새 계약 생성에만 상한이 적용됩니다. 600건 이하를 만드는 기존 계약은 동작이 전혀 바뀌지 않습니다.
