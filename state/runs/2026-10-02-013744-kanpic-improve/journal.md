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
