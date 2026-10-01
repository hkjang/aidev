# 회차 노트 2026-10-02-013744-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:37] base pinned — main@5d8e9d5
- [러너 01:37] autonomy release — 
- [러너 01:43] scout done — 일정 트리거의 "일·요일 OR" 판정이 `*/N` 같은 별표 필드를 제한으로 오해해, `0 0 */2 * MON` 이 월요일이 아닌

## 구현 노트
- `internal/automation/schedule.go` 한 곳: 일·요일 OR 판정의 "제한되지 않음" 을 `raw == "*"` 에서 `strings.HasPrefix(raw, "*")` 로 바꾸고(플래그 이름 `wildcard`→`unrestricted`), `matchesDay` 의 4갈래 switch 를 "한쪽이라도 비제한이면 AND, 둘 다 제한이면 OR" 로 줄였다. `0 0 */2 * MON` 이 홀숫날 전부 + 모든 월요일에 돌던 것을 crontab(5) 대로 "격일 중 월요일" 로 고친 것이다.
- 확신 없는 곳·검증 못 한 것: 멀티바이트/비UTC 조합에서는 새 표를 UTC 로만 돌렸다 — 기존 Asia/Seoul·America/New_York 테스트가 통과하는 것으로만 확인했다. 웹·DB 통합 테스트·브라우저 E2E 는 돌리지 않았다(Go 패키지 안쪽 순수 함수 변경이고 스키마·API 응답이 바뀌지 않는다). `service.go` 의 `advanceSchedule`/`nextScheduleTime` 이 `Next` 를 부르기만 하는 것은 읽어 확인했으나 Postgres 로 실행하지는 못했다.
- 일부러 하지 않은 것: `parseCronField` 의 `allowed` 채우기(step/range/이름/7→0 매핑), `Next` 의 DST·윤일·`local.Hour()!=hour` 검사, `SAT-SUN` 감싸는 범위 거절, `@midnight`/`@annually` 별칭(차선 후보로 남김). 문서·PDF 도 손대지 않았다 — `docs/*.md` 는 일·요일 OR 규칙을 적은 적이 없어 고칠 문장이 없다.
- 다음 역할이 조심할 것: 새 테스트 `TestScheduleCombinesDayAndWeekdayByRestriction` 의 대조 4사례(`0 0 1-31 * MON`, `0 0 * * */3`, `* * * * MON`, `0 9 1 * *`)는 고치기 **전부터** 초록이었다 — 이 변경이 OR 사례를 흔들지 않았다는 증거이므로 지우지 말 것. 빨강이었던 것은 `0 0 */2 * MON`·`0 0 1 * */2` 두 사례뿐이고, `strings.HasPrefix` 한 줄만 원복하면 정확히 그 둘만 다시 빨강이 된다(확인함). 날짜 기대값은 2026-10-30 기준 절대 날짜라 시계와 무관하다.
- [러너 01:47] brief accepted — 채택 — 과제서의 재현 결과가 지금 코드와 정확히 일치했고(`0 0 */2 * MON` 이 토·일·화·목에 실행), 건드릴 파일 2개·세 �
- [러너 01:47] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: `strings.HasPrefix` 한 줄을 `raw == "*"` 로 되돌려 직접 돌렸고 원장의 `실패 재현` 출력과 글자까지 동일하게 2개 부속만 빨강이 됐다 — 테스트가 바뀐 경로를 실제로 지난다. 기대 날짜 6건은 손으로 요일을 다시 계산해 맞았고, `go build ./...`·`go vet`·`gofmt -l internal/automation`·`go test ./...` 모두 깨끗하다. `matchesDay` 는 ISC/Vixie cron 의 `(DOM_STAR||DOW_STAR) ? AND : OR` 과 플래그 설정 위치까지 일치하는 충실한 포팅이다.
- 확인함(범위): `wildcard` 잔재 없음(automation 안), 프런트엔드에 일·요일 판정 중복 없음(AutomationPanel 은 cron 문자열만 전달), `docs/*.md` 에 OR 규칙 문장 없음 — 문서 미수정이 맞다. 불가능한 식은 생성 시 `nextScheduleTime` 이 거절하므로 AND 전환이 새 "8년 내 없음" 저장 상태를 만들지 않는다.
- 못 본 것: 웹 lint/test/build, DB 통합, 브라우저 E2E 미실행(순수 함수 변경이라 영향 없다고 판단). 비UTC 조합은 기존 Asia/Seoul·New_York 테스트 통과로만 확인.
- 승인이어도 남는 우려(릴리즈 노트에 꼭): `service.go:609` 는 `next_run_at<=now` 로만 뽑고 `matchesDay` 를 다시 보지 않으므로, **이미 저장된** 구 OR 의미의 `next_run_at` 은 배포 후 한 번 더 그 시각에 실행된 뒤 `advanceSchedule`(:761) 에서 새 의미로 자리를 잡는다. 또 `*/2 * MON-FRI` 류는 실행 횟수가 줄어드는 사용자 가시 변경이다.
- 또 하나: 커밋 메시지의 "crontab(5) 대로" 는 느슨하다 — man page 의 괄호("i.e., aren't *")를 글자대로 읽으면 `*/2` 는 제한이고, 이번 수정은 man page 가 아니라 참조 구현(필드 첫 글자) 을 따른다. 코드 주석은 그 사실을 정확히 적고 있으니 코드는 문제없고, 릴리즈 문구만 "Vixie/ISC cron 구현과 일치" 로 쓰는 게 정확하다.
- [러너 01:51] review approved — 리뷰 승인 (risk=low)
- [러너 01:51] pr created — https://github.com/hkjang/kanpic/pull/38
- [러너 02:02] ci passed — 검사 2개 모두 success
- [러너 02:02] merge done — b1f3a81
- [러너 02:16] release published — v0.259.0
- [러너 02:18] assets verified — v0.259.0 자산 2개 (이전 v0.258.0: 2)
