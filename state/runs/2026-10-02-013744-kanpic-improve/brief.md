# 과제서 (2026-10-02, kanpic)

- 과제: 일정 트리거의 "일·요일 OR" 판정이 `*/N` 같은 별표 필드를 제한으로 오해해, `0 0 */2 * MON` 이 월요일이 아닌 날에도 자동화를 실행한다 (가치 3 / 위험 2 / 작업량 S)

- 왜: 이 저장소는 Vixie cron 의 "일·요일 두 필드가 **모두 제한되어 있으면** 어느 한쪽만 맞아도 실행" 규칙을 스스로 채택했다(기존 테스트 이름이 `TestScheduleAliasesLeapDayAndCronDayOrSemantics`, `docs/USER_GUIDE.md:1594` 는 `분 시 일 월 요일` 5필드 cron 과 step 지원을 약속한다). 그런데 `internal/automation/schedule.go:131` 은 "제한되지 않음"을 `raw == "*"` 로만 판정하므로 `*/2`·`*/3` 같은 별표+step 필드가 제한으로 분류되어 OR 로 넘어가고, 그 결과 `0 0 */2 * MON`(격일 중 월요일만) 이 홀숫날 전부 + 모든 월요일에 실행된다. 자동화는 보는 사람 없이 워크북 셀을 실제로 바꾸므로(`internal/automation/service.go:766` `ApplyCells`), 요청하지 않은 날의 실행은 데이터 변경으로 남는다.

- 실제로 확인한 것 (임시 테스트를 `internal/automation` 에 넣고 `go test -run` 으로 돌린 뒤 지웠음. 작업 트리는 clean):
  - `0 0 */2 * MON`, after=2026-10-30T12:00Z → `2026-10-31(Sat) 2026-11-01(Sun) 2026-11-02(Mon) 2026-11-03(Tue) 2026-11-05(Thu)`. **토·일·화·목에 실행된다.** 기대(cron 표준): 홀숫날이면서 월요일인 날만.
  - `0 0 1-31 * MON` → 매일. 이쪽은 두 필드 모두 별표로 시작하지 않으므로 OR 가 맞고, 고친 뒤에도 **바뀌지 않아야 한다**(회귀 방지용 대조 사례).
  - `0 0 * * */3` → `10-31(Sat) 11-01(Sun) 11-04(Wed) 11-07(Sat) 11-08(Sun)`. 요일 `*/3`(일·수·토)만 제한이고 일 필드는 `*` 이므로 지금도 결과는 맞다 — 즉 결함은 **한쪽이 `*/N`, 다른 쪽이 진짜 제한**일 때만 드러난다.

- 수용 기준:
  1) `0 0 */2 * MON`(그리고 대칭 사례 `0 0 1 * */2`) 이 "양쪽 모두 맞는 날"에만 실행된다 — 즉 별표로 시작하는 필드가 하나라도 있으면 일·요일은 AND 로 합쳐진다.
  2) 기존 4개 테스트가 그대로 통과한다. 특히 `TestScheduleAliasesLeapDayAndCronDayOrSemantics` 의 `0 9 1 * MON` → 2026-08-03 OR 사례와 `*/15 9-17 * * MON-FRI`(분·시의 `*/15` 는 OR 판정에 쓰이지 않으므로 영향 없음), `TestScheduleSkipsNonexistentDSTWallTime` 은 손대지 말 것.
  3) 테스트가 증명해야 하는 것: 고치기 **전에 빨강**인 사례(`0 0 */2 * MON` 의 다음 5회에 토·일·화·목이 섞여 있다)와, 고친 뒤에도 **초록으로 남아야 하는 대조 사례**(`0 0 1-31 * MON` 은 매일, `0 0 * * */3` 은 일·수·토, `* * * * MON` 은 월요일만, `0 9 1 * *` 은 매월 1일) 를 같은 표 하나로 함께 못 박는다. 구현을 원복하면 1)의 사례만 다시 빨강이 되어야 한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/automation/schedule.go:29-32 cronField` — `wildcard bool` 의 의미를 "필드가 별표로 시작한다(=제한되지 않았다)" 로 바로잡는다. 이름을 `unrestricted` 로 바꿔도 좋다(패키지 내부 전용 — 아래 참조 범위 확인 결과 참고).
  - `internal/automation/schedule.go:131 parseCronField` — `wildcard: raw == "*"` 를 `strings.HasPrefix(raw, "*")` 로 바꾼다. Vixie cron 은 필드 **전체의 첫 글자**만 보므로 `1,*/2` 는 제한으로 둔다(즉 `strings.Split` 안쪽 세그먼트가 아니라 `raw` 를 봐야 한다). `raw` 는 `strings.Fields` 가 쪼갠 조각이라 공백이 없다 — 확인함.
  - `internal/automation/schedule.go:115-128 matchesDay` — 4갈래 switch 를 "한쪽이라도 제한되지 않았으면 AND, 둘 다 제한이면 OR" 로 줄인다. 아래 두 줄이 기존 네 갈래와 평범한 `*` 입력에서 **값이 같다**는 것을 손으로 확인했다(`*` 는 그 필드의 `allowed` 가 전부 true 이므로 AND 가 다른 쪽 판정과 같아진다):
    ```go
    if s.day.unrestricted || s.weekday.unrestricted {
        return dayMatch && weekdayMatch
    }
    return dayMatch || weekdayMatch
    ```
    왜 그런지(crontab(5) 의 "both fields restricted" 규칙)를 저장소 관례대로 한국어 주석으로 적을 것.
  - `internal/automation/schedule_test.go` — 위 수용 기준 3)의 표 테스트를 더한다. 기존 테스트는 고치지 말고 **추가**만 할 것. 각 사례는 `ParseSchedule` → `Next` 를 연속 5회 돌려 날짜·요일 목록을 비교하는 꼴이 읽기 쉽다(정찰이 그 방식으로 재현했다).

- 참조 범위 확인: `rg wildcard --include=*.go` 결과 `cronField.wildcard` 는 `schedule.go:119/121/123/131` 에서만 쓰인다(다른 `wildcard` 히트는 `internal/formula`·`internal/auth`·`internal/httpapi` 의 무관한 이름). 분·시·월 필드의 `wildcard` 는 읽는 곳이 없으므로 이 변경의 영향은 일·요일 판정 한 곳뿐이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 빨강 확인 → 초록: `go test ./internal/automation -run TestSchedule -v`
  - `go test ./internal/automation -count=1`
  - `go test ./...` (정찰 이번 실행에서 20패키지 전부 ok, `internal/automation` 커버리지 14.5%)
  - `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal ./pkg` (출력 없어야 함)
  - `./scripts/check-release-docs.sh` (현재 README VERSION = v0.258.0), `./scripts/check-commit-identities.sh HEAD`
  - 구현을 원복해 1)의 사례가 다시 빨강이 되는지 확인할 것.

- 위험과 피할 것:
  - **DST·윤일·시간대 쪽은 손대지 말 것.** `Next` 의 `start`/`limit` 계산, 존재하지 않는 벽시계 시각을 건너뛰는 `local.Hour() != hour` 검사(`schedule.go:103`)는 의도된 동작이고 `TestScheduleSkipsNonexistentDSTWallTime` 이 그것을 못 박고 있다. 이번 과제는 `matchesDay` 와 `wildcard` 판정 세 자리로 끝난다.
  - **`parseCronField` 의 `allowed` 채우기 로직(step/range/이름/`sundaySeven` 7→0 매핑)을 건드리지 말 것.** `wildcard` 플래그만 바꾸면 되고, `allowed` 를 건드리면 `*/15 9-17 * * MON-FRI` 같은 기존 사례가 함께 흔들린다.
  - **`SAT-SUN` 이 `range start exceeds end` 로 거절되는 것(정찰이 확인)은 이번에 고치지 말 것** — 감싸는(wrap-around) 범위를 지원하느냐는 별개의 계약 결정이고, `6-7` 로 토·일을 적을 길이 이미 있다.
  - `internal/automation/service.go`(1262줄, DB 필요)·`migrations`·`.github/workflows`·`internal/auth`·`internal/apikey` 는 건드리지 말 것. `advanceSchedule`(service.go:748)·`nextScheduleTime`(service.go:1176)·`RunDueSchedules`(service.go:609) 는 `Next` 를 부르기만 하므로 수정 불필요이며, 그쪽을 건드리면 Postgres 통합 테스트가 필요해져 한 세션을 넘긴다.
  - **문서·PDF 는 건드리지 않아도 된다** — `docs/USER_GUIDE.md:1604` 는 "`*`, 목록, 범위, step, 영문 이름, `@` 별칭 지원"만 말하고 일·요일 OR 규칙을 적은 적이 없다(`docs/*.md` 에서 확인). 문서를 손대면 PDF 재생성이 따라붙어 범위가 커진다.
  - 과거 교훈: 손으로 만든 대역을 쓰지 말 것 — 여기서는 `ParseSchedule`/`Next` 가 바로 프로덕션 함수이므로 그대로 부르면 된다. grep 결과를 증거로 내지 말고 `Next` 가 돌려준 실제 시각으로 단언할 것.

- 차선 후보: **`@midnight`·`@annually` cron 별칭을 더한다** (가치 2 / 위험 1 / 작업량 S) — `schedule.go:12-18 cronAliases` 에 `@hourly/@daily/@weekly/@monthly/@yearly` 는 있으나 Vixie cron 의 `@midnight`(=`@daily`)·`@annually`(=`@yearly`) 가 없어 `@midnight` 이 "schedule cron must contain five fields" 로 거절되는 것을 정찰이 확인했다. 맵 두 줄 + 테스트 한 줄이고 `docs/USER_GUIDE.md:1604` 의 별칭 목록 갱신이 따라붙는다(PDF 동반 여부는 구현자가 판단). 1순위가 성립하지 않을 때만 고를 것.
