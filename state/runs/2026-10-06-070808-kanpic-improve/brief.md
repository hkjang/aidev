- 과제: 자동화 일정 cron 값이 `+5`·`-0` 같은 부호 붙은 토큰을 조용히 받아들인다 — 숫자가 아닌 글자를 거절하게 한다 (가치 2 / 위험 2 / 작업량 S)
- 왜: `internal/automation/schedule.go` 의 `cronValue` 가 이름 조회가 빗나가면 `strconv.Atoi(raw)` 로 값을 읽는다 — `Atoi` 는 부호를 허용하므로 cron 문법에 없는 `+5`·`+0`·`-0` 이 통과해 `+5 0 * * *` 가 거절 대신 "매일 00:05" 로 저장된다(파싱 사슬을 이번 회차에 끝까지 읽어 확인; 2026-10-03 정찰도 같은 결론을 기록했다). 사용자는 오타를 검증이 잡아 줄 것으로 기대하는데 통과하고, `Schedule.Expression` 에 `+5 0 * * *` 가 그대로 저장돼 다른 cron 구현·사람의 눈에는 읽히지 않는 식이 영구 기록으로 남는다. 고치면 저장 시점에 `ErrInvalid` 로 돌려준다.
- 수용 기준:
  1) `ParseSchedule("+5 0 * * *", "UTC")` 가 `errors.Is(err, ErrInvalid)` 로 거절되고, 오류 문구가 기존 `value %q must be between %d and %d` 계열 그대로여서 `internal/httpapi/automations.go` 의 400 응답 `message` 에 사람이 읽을 수 있게 실린다.
  2) 부호가 값 자리·범위 양 끝·요일 이름 자리 어디에 와도 거절된다: `+5 0 * * *`, `0 0 +1 +1 *`, `0 0 1 1 +7`, `0 0 * * +0`, `-0 0 * * *`, `+1-+3 0 * * *`. (이 목록이 **지금은 전부 통과**한다는 것이 이 과제의 빨강이다. 다만 `+1-+3` 은 범위 분해(`strings.Cut(base,"-")`)가 먼저 걸릴 수 있으니 구현자가 실제 결과를 보고 표에서 빼거나 남길 것 — 이번 회차는 실행으로 확인하지 못했다, 아래 "확인 못한 것" 참고.)
  3) 바뀌면 안 되는 것이 그대로다: `5 0 * * *`, `*/15 * * * *`, `0 0 * * MON-FRI`, `0 0 * * 7`(일요일), `@daily`·`@midnight`, `0 0 */2 * MON`(일·요일 AND), 윤일·DST 사례 — 즉 기존 `TestScheduleNextSupportsStepsRangesNamesAndTimezone`·`TestScheduleCombinesDayAndWeekdayByRestriction`·`TestScheduleAliasesMatchEquivalentCronExpressions`·`TestScheduleSkipsNonexistentDSTWallTime` 가 손대지 않은 채로 전부 초록.
  4) 테스트는 대역 없이 프로덕션 `ParseSchedule`(필요하면 `Next`)만 호출해, 고치기 전에는 "받아들여 00:05 가 됐다" 로 실패하고 고친 뒤 거절로 통과함을 보인다.
- 건드릴 파일:
  - `internal/automation/schedule.go:cronValue` — 이름 조회 실패 뒤 `strconv.Atoi` 호출 **전에** `raw` 가 십진 숫자만으로 이루어졌는지 확인하고 아니면 기존 `value %q must be between %d and %d` 오류를 돌려준다. 가장 좁은 구현은 바이트 순회(`raw` 가 비었거나 `c < '0' || c > '9'` 인 글자가 있으면 거절) 한 블록 + 왜 `Atoi` 로는 부족한지(부호 허용) 적는 한국어 주석 한 줄. `ErrInvalid` 래핑은 호출자(`ParseSchedule`)가 이미 하므로 이 함수에서 래핑하지 말 것.
  - `internal/automation/schedule_test.go` — 거절 표를 새 테스트 함수 하나로 추가(예: `TestScheduleRejectsSignedCronValues`). 기존 `TestScheduleRejectsInvalidExpressions`(209행, `{"* * * *"}`·`{"60 * * * *"}`·`{"*/0 * * * *"}`·`{"0 9 * * FUNDAY"}` …)의 표에 섞어 넣기만 하면 "무엇이 새로 거절되는가" 가 기록에 남지 않는다. 대조 사례는 이미 있는 테스트가 덮으므로 중복 작성 금지.
  - 프로덕션 1파일. 그 이상으로 번지면 과제를 잘못 읽은 것이다.
- 검증 명령 (모두 워크트리 루트에서):
  - `go test ./internal/automation -run TestSchedule -v` — 새 테스트 빨강 → 초록, 기존 7개 테스트 전부 초록
  - `go test ./internal/automation -count=1` — 이번 회차에 실제로 돌려 `ok kanpic/internal/automation 0.014s` 확인함
  - `go test ./...`, `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal ./pkg`(출력 없어야 함)
  - `./scripts/check-release-docs.sh`, `./scripts/check-commit-identities.sh HEAD`
  - 되돌리기 확인: 넣은 숫자 검사 블록만 지웠을 때 **정확히 새 테스트만** 같은 출력으로 다시 빨강이 되는지 볼 것.
- 위험과 피할 것:
  - **이미 저장된 일정이 깨질 수 있다.** 거절 범위를 "값 토큰이 십진 숫자가 아니다" 로만 좁게 두고, step 검사(`parseCronField` 의 `strconv.Atoi(stepRaw)` + `1..maximum-minimum+1`)·범위 분해·`sundaySeven`(7→0) 매핑·`cronAliases`·`matchesDay` 의 일·요일 OR 판정·`Next` 의 DST 검사·`maxScheduleLookaheadYears` 는 손대지 말 것. step 자리의 부호(`*/+2`)까지 함께 고치려 하면 건드릴 자리가 둘로 늘고 step 상한 검사와 뒤섞인다 — 이번 회차 범위 밖.
  - `migrations`, `internal/automation/service.go`, `internal/httpapi/*`, `.github/workflows`, docs/PDF 는 건드리지 않는다(사용자에게 보이는 규칙이 아니라 잘못된 입력의 거절이므로 가이드 변경 불필요 — 따라서 PDF 재생성도 없다).
  - automation 패키지는 2026-10-02·10-03 두 회차에 연속 선택됐다. 같은 파일이지만 자리가 겹치지 않는지(그 두 회차는 `parseCronField` 의 `unrestricted` 판정과 `cronAliases`) 확인하고, `cronValue` 밖으로 나가지 말 것.
- 확인 못한 것 (추측으로 적은 자리):
  - 런타임 재현을 이 세션에서 **실행하지 못했다** — 샌드박스가 `go run`(과 임시 모듈 프로브)을 승인하지 않았다. 근거는 파싱 사슬 전체 읽기(`ParseSchedule` → `parseCronField` → `cronValue`: 이름 맵 조회 → `strconv.Atoi` → `minimum..maximum` 범위 검사뿐, 부호 필터 없음)와 2026-10-03 정찰의 기록이다. 구현자는 **먼저 빨강 테스트로 현재 동작을 찍어 보고**, 수용 기준 2)의 목록 중 실제로 통과하는 것만 남길 것.
  - `internal/httpapi/automations.go:280` 이 `errors.Is(err, automation.ErrInvalid)` → 400 + `err.Error()` 를 `message` 에 싣는다는 것은 2026-10-03 회차가 코드로 확인한 내용이고 이번 회차 미재확인이다.
- 차선 후보: 자동화 일정 다음 실행 시각 계약을 월말·시간대 표로 못 박는다 (`0 0 31 * *`@Asia/Seoul 의 2월 건너뜀, `0 0 29 2 *` 4년 주기, `30 1 * * *`@America/New_York 가을 DST 1회 실행). 테스트 전용이라 프로덕션 변경 0 — 1순위가 "지금 코드가 이미 거절한다" 로 무너지면 이것을 고를 것. 그래도 `TestScheduleSkipsNonexistentDSTWallTime` 이 못 박은 현 동작(봄철 사라진 시각 건너뜀)은 바꾸지 말 것.
