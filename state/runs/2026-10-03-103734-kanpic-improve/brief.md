- 과제: 자동화 일정이 `@midnight`·`@annually` 를 받아들이고, 모르는 `@별칭` 은 "다섯 필드" 가 아니라 지원 별칭을 알려 주며 거절한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/automation/schedule.go:12-18` 의 `cronAliases` 에는 다섯 개(`@hourly @daily @weekly @monthly @yearly`)만 있고, Vixie cron 의 표준 별칭인 `@midnight`(=`@daily`)·`@annually`(=`@yearly`) 가 빠져 있다. 지금 이 두 개를 자동화 일정 칸에 넣으면 `invalid automation: schedule cron must contain five fields` 로 거절되는데(아래 "정찰이 실제로 본 것" 참조), 사용자는 다섯 필드짜리 식을 쓴 적이 없으므로 이 문구는 무엇이 틀렸는지 알려 주지 못한다 — `@reboot` 처럼 **영영 지원할 수 없는** 별칭도 같은 문구로 거절되므로 사용자는 오타인지 미지원인지 구분할 길이 없다. 두 별칭을 더하고 `@` 로 시작하는 입력에는 지원 목록을 담은 전용 오류를 주면, 흔한 두 식이 바로 동작하고 나머지는 한 번에 원인을 알 수 있다.

- 정찰이 실제로 본 것 (이 base, `6ef472a` 에서 임시 테스트를 한 번 돌려 확인하고 지웠다):
  - `@midnight` / `@annually` / `@reboot` → 모두 `PARSE ERR: invalid automation: schedule cron must contain five fields`
  - 대조로 지금 도는 것들: `0 12 * * mon` → 2026-11-02 Mon 12:00 부터 매주 월요일, `0 0 1 jan *` → 2027-01-01 부터 매년, `0 0 1 * MON` → 11/01(일) + 모든 월요일(두 필드 모두 제한 → OR, b1f3a81 의 의도대로), `0 0 */2 * *` → 10/31·11/01·11/03·11/05, `0 0 31 * *`(Asia/Seoul) → 10/31·12/31·2027-01-31·2027-03-31(31일 없는 달 건너뜀), `0 0 29 2 *` → 2028·2032·2036·2040-02-29, `30 1 * * *`(America/New_York) → 11/01 01:30 **-0400** 한 번만(가을 DST 중복 시각에 두 번 돌지 않는다).
  - 즉 결함은 **별칭 맵의 누락과 오류 문구** 한 군데뿐이고, 파서·`Next`·DST·일/요일 OR 판정은 손댈 필요가 없다.

- 수용 기준:
  1) `ParseSchedule("@midnight", "UTC")` 의 `Next` 결과가 `ParseSchedule("0 0 * * *", "UTC")` 와 연속 실행 시각에서 같다.
  2) `ParseSchedule("@annually", "UTC")` 의 `Next` 결과가 `ParseSchedule("0 0 1 1 *", "UTC")` 와 같다. 대소문자 무시(`@MIDNIGHT`)도 지금 `strings.ToLower` 경로 그대로 동작한다.
  3) `ParseSchedule("@reboot", …)` 와 `ParseSchedule("@nope", …)` 는 여전히 `ErrInvalid` 로 거절되지만, 오류 문구가 "five fields" 가 아니라 **지원하는 별칭 목록**을 담는다(`errors.Is(err, ErrInvalid)` + 문구에 `@midnight` 포함을 단언). `@reboot` 은 영구 저장된 `next_run_at` 기반 스케줄러에 의미가 없으므로 **지원하지 않는다**.
  4) 테스트는 손으로 만든 대역 없이 프로덕션 `ParseSchedule`→`Next` 를 연속 3~4회 돌려 **실제 반환 시각**을 비교한다(기존 `schedule_test.go` 의 표 방식 그대로). 기존 다섯 별칭·`0 0 1 * MON`·DST·윤일 사례는 그대로 통과해야 한다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/automation/schedule.go:12` `cronAliases` — `"@midnight": "0 0 * * *"`, `"@annually": "0 0 1 1 *"` 두 줄 추가.
  - `internal/automation/schedule.go:45-50` `ParseSchedule` — 별칭 조회가 빗나갔을 때 `strings.HasPrefix(lowered, "@")` 이면 `len(parts)!=5` 검사 **전에** 지원 별칭을 나열한 `ErrInvalid` 오류를 돌려준다. 별칭 목록은 `cronAliases` 키를 정렬해 만들 것(맵 순회 순서가 무작위라 문구가 흔들리면 테스트가 깜빡인다). 기존 다섯 필드 경로의 문구는 바꾸지 말 것.
  - `internal/automation/schedule_test.go` — `TestScheduleAliasesLeapDayAndCronDayOrSemantics`(33행) 의 별칭 표에 `@midnight`·`@annually` 행 추가, `TestScheduleRejectsInvalidExpressions`(159행) 에 `@reboot`·`@nope` 행 추가.
  - `docs/USER_GUIDE.md:1604` — "`@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly` 별칭" 문장에 두 별칭을 더한다. 이 줄은 지금 구현과 정확히 일치하므로(정찰 확인) 고치지 않으면 문서가 틀리게 된다.
  - `docs/USER_GUIDE.pdf` — 저장소 관례상 사용자 가이드 변경에는 PDF 가 동반된다. `scripts/generate_pdf.js` 로 **USER_GUIDE 하나만** 재생성한다(페이지 오류 한 줄은 무해하고 크기 변동은 정상 — 과거 기록).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/automation -run TestSchedule -v` — 고치기 전에 새 사례가 **빨강**인 것을 먼저 볼 것.
  - `go test ./internal/automation -count=1`
  - `go test ./...`
  - `go vet ./...` / `go build ./...` / `gofmt -l ./cmd ./internal ./pkg`(출력 없어야 함)
  - `./scripts/check-release-docs.sh` / `./scripts/check-commit-identities.sh HEAD`
  - 되돌림 확인: 추가한 맵 두 줄을 지우면 새 별칭 사례만 같은 문구로 다시 빨강이 되는지 볼 것.

- 위험과 피할 것:
  - `parseCronField` 의 range·step 로직, `matchesDay` 의 일·요일 OR 판정(b1f3a81 이 막 고친 자리), `Next` 의 DST `local.Hour()!=hour` 검사, `maxScheduleLookaheadYears` 는 **손대지 말 것**. 보류 목록의 `SAT-SUN` 감싸는 범위와 DST 재실행은 계약 결정이 먼저라 이번 범위 밖이다(정찰 확인: `SAT-SUN` 은 `range start exceeds end` 로 거절되고 `6-7` 로 쓸 길이 이미 있다).
  - `internal/automation/service.go`·`migrations`·`.github/workflows` 는 건드리지 말 것.
  - `strconv.Atoi` 가 `+5` 를 5 로 받는 느슨함을 함께 조이지 말 것 — 별개 과제이고 기존 일정을 깨뜨릴 수 있다.
  - 오류 문구를 바꿀 때 `ErrInvalid` 래핑(`fmt.Errorf("%w: …", ErrInvalid)`)을 유지할 것. httpapi 가 이것으로 400 을 고른다(미확인 — 구현자가 `errors.Is(…, ErrInvalid)` 사용처를 한 번 확인하면 좋다).
  - 커밋·릴리즈에 Claude/Anthropic 표기를 넣지 말 것.

- 차선 후보: **자동화 일정의 다음 실행 시각 계약을 월말·시간대 표로 못 박는다** — 위 "정찰이 실제로 본 것" 의 대조 6줄(`0 0 31 * *`@Asia/Seoul 의 월 건너뜀, `0 0 29 2 *` 의 4년 주기, `30 1 * * *`@America/New_York 의 가을 DST 중복 시각 1회 실행)은 지금 전부 올바르게 동작하지만 테스트가 없다. `internal/automation` 커버리지가 낮아 회귀 방어 가치가 있다. 다만 **테스트 전용**이라 2026-09-22 회차처럼 no-change 판정이 날 위험이 있으니, 1순위가 성립하지 않을 때만 고르고 기존 DST·윤일 테스트의 의도는 바꾸지 말고 추가만 할 것.
