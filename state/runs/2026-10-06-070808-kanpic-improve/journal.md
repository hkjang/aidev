# 회차 노트 2026-10-06-070808-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:08] base pinned — main@c215e18
- [러너 07:08] autonomy release — 

## 정찰 노트
- `cronValue` 의 부호 허용을 골랐다: 프로덕션 1파일·`cronValue` 한 함수로 끝나고 기존 테스트 7개가 대조군을 이미 덮는다. DST 재결정(테스트가 현 동작을 의도로 못 박음)과 LazyQuotes·compareLists·지수 표기(사용자에게 보이는 계약 재결정)는 자율 회차가 결정할 일이 아니라 제쳤고, ADMIN_GUIDE 오류 코드는 "어느 오류 칸인지" 가 아직 좁혀지지 않아 또 미뤘다.
- 추측으로 적은 것: 런타임 재현을 못 했다 — 샌드박스가 `go run`(임시 프로브 모듈)을 승인하지 않아 근거가 파싱 사슬 읽기 + 2026-10-03 정찰 기록이다. 수용 기준 2)의 `+1-+3` 은 범위 분해가 먼저 걸릴 수 있어 구현자가 빨강으로 확인하고 표에서 빼도 된다. `httpapi` 400 경로도 지난 회차의 확인을 인용했을 뿐 이번에 다시 열지 않았다.
- 구현자가 조심할 것: 먼저 빨강을 찍어 "지금 무엇이 통과하는지" 를 기록에 남기고, 거절을 값 토큰의 십진 숫자 검사로만 좁게 둘 것. step 자리(`*/+2`)·`unrestricted` 판정·`cronAliases`·`matchesDay`·`Next` DST 는 지난 두 회차가 손댄 자리라 이번에 함께 건드리면 회귀를 구분할 수 없다.
- 실제로 돌린 것: `go test ./internal/automation -count=1` → `ok kanpic/internal/automation 0.014s`. `go run`·`go mod tidy` 는 권한 거부로 실행 못 함(정찰이 만든 `probe/` 디렉터리는 런 디렉터리 안에 있고 저장소는 건드리지 않았다).
- 프로필을 다시 썼다: 이전 프로필의 "fetcher.go parseCSV 에 UTF-8 검사가 없다" 정정이 0fd9038 으로 더는 맞지 않고 7일이 지나 어긋난 줄이 생겼다.
- [러너 07:14] scout done — 자동화 일정 cron 값이 `+5`·`-0` 같은 부호 붙은 토큰을 조용히 받아들인다 — 숫자가 아닌 글자를 거절하게

## 구현 노트
- `internal/automation/schedule.go:cronValue` 의 `strconv.Atoi` **앞**에 `isDecimalDigits` 검사를 넣어 `+5`·`+0`·`+7` 같은 부호 붙은 값 토큰을 기존 `value %q must be between %d and %d` 문구로 거절한다. 그 전에는 `+5 0 * * *` 가 "매일 00:05" 로 저장됐다.
- 확신 없는 곳: step 자리(`parseCronField` 의 `strconv.Atoi(stepRaw)`)는 과제서 범위 밖이라 손대지 않았고, `*/+2` 가 지금도 통과하는지는 **실행으로 확인하지 않았다** — ideas.json 에 다음 회차 후보로 남겼다. `internal/httpapi/automations.go` 의 400 경로도 이번에 다시 열지 않았다(지난 회차 확인 인용).
- 과제서 수용 기준 2) 중 `-0 0 * * *` 는 **고치기 전에도 이미 거절**됐다(범위 분해가 먼저 걸려 왼쪽이 빈 문자열). 표에서 빼지 않고 회귀 방지로 남겼으므로 그 부속만 처음부터 PASS 다 — 새로 거절되는 것은 나머지 5사례다.
- 일부러 하지 않은 것: `cronAliases`·`matchesDay`·`sundaySeven`·`Next` 의 DST 검사·`service.go`·migrations·httpapi·docs/PDF. 사용자에게 보이는 규칙이 아니라 잘못된 입력의 거절이라 가이드·PDF 변경 없음.
- 다음 역할이 조심할 것: 이 테스트는 DB·네트워크 없이 돈다(`go test ./internal/automation -run TestSchedule -v`). **이미 저장된 일정 중 부호 붙은 식이 있으면 재시작 때 `ErrInvalid` 로 거절된다** — 의도된 동작이지만 운영 데이터에 그런 행이 있는지는 확인할 수 없었다.
- [러너 07:17] brief accepted — 채택 — 과제서의 파싱 사슬 분석이 지금 코드와 정확히 맞았고(부호 5사례가 모두 통과), 건드릴 파일 2개(프로덕션 1)·`c
- [러너 07:17] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- main 의 schedule.go 로 되돌려 새 테스트를 실제로 빨강으로 찍었다: 6부속 중 5개 FAIL(`+5 0 * * *` → 00:05 수락, `+1-+3` 도 수락), `-0` 만 애초에 PASS — 구현 노트·원장의 실패 재현 줄과 정확히 일치한다. 구현자가 "실행으로 확인 못 함" 이라 적은 `*/+2`(step 자리)는 과제서 범위 밖이 맞아 결함으로 보지 않았다.
- 새로 거절되는 집합을 좁혀 확인: cronValue 에 닿는 토큰 중 `-` 접두는 범위 분해가 먼저 걸러내므로 실제 신규 거절은 `+` 접두뿐이고 유효 cron 문법 손실은 없다. isDecimalDigits 의 ASCII 바이트 비교는 unicode.IsDigit 보다 옳다.
- 승인이어도 남는 우려(릴리즈 노트에 적을 것): 이미 저장된 부호 붙은 식이 있으면 `advanceSchedule` 이 실패해 next_run_at 이 전진하지 않아 그 자동화만 멈추고 폴링마다 오류가 로그에 쌓인다(중복 실행 없음, 화면에서 cron 수정 시 복구). 운영 DB 에 그런 행이 있는지는 확인 불가.
- 안 본 것: 웹 npm 검증과 DB 통합·E2E — diff 가 Go 2파일(프로덕션 1)뿐이고 web/src·migrations 가 없어 생략했다. 보안·법무 차단 사유 없음(인가·비밀값·외부 요청·개인정보 미접촉, 입력 검증을 좁히는 방향).
- 다음 회차: automation 패키지가 10-02·10-03·10-06 세 번 연속이라 다음은 다른 패키지를 고르는 편이 회귀 원인 구분에 낫다.
- [러너 07:21] review approved — 리뷰 승인 (risk=low)
- [러너 07:21] pr created — https://github.com/hkjang/kanpic/pull/42
- [러너 07:28] ci passed — 검사 2개 모두 success
- [러너 07:29] merge done — a77cce4
