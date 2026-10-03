# 회차 노트 2026-10-03-103734-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:37] base pinned — main@6ef472a
- [러너 10:37] autonomy release —

## 정찰 노트
- 고른 이유: automation 은 지난 회차가 고친 자리 바로 옆이라 맥락이 싸고, 임시 테스트 한 번으로 `@midnight`·`@annually`·`@reboot` 이 "schedule cron must contain five fields" 라는 엉뚱한 문구로 거절되는 것을 실제로 확인했다. mail 계열 보류(SendNow·mail_deliveries)는 Postgres 가 있어야 재현되고 네 회차 연속 선택돼 포화라 제쳤고, 숫자·compareLists 계열은 "어느 쪽 계약이 맞는지" 결정이 먼저라 정찰이 혼자 정할 일이 아니라 제쳤다.
- 확인한 것: 위 세 식의 거절 문구, 그리고 대조 6줄(`0 0 1 * MON`, `0 0 */2 * *`, `0 0 31 * *`@Asia/Seoul, `0 0 29 2 *`, `30 1 * * *`@America/New_York 가을 DST 1회 실행, `5/15 * * * *`)이 지금 올바르게 도는 것 — 임시 테스트 파일을 만들어 한 번 돌리고 지웠다(작업 트리 깨끗).
- 추측으로 적은 것(미확인): httpapi 가 `errors.Is(…, ErrInvalid)` 로 400 을 고르는지 확인하지 않았다(래핑만 유지하면 안전). USER_GUIDE PDF 재생성도 이번에 돌려 보지 않았다.
- 조심할 것: 별칭 목록 문구를 `cronAliases` 맵에서 만들 때 정렬하지 않으면 Go 맵 순회 순서로 테스트가 깜빡인다. `matchesDay`·`parseCronField` range·DST 검사는 b1f3a81 이 막 손본 자리이니 건드리지 말 것. 범위는 프로덕션 1파일(schedule.go)+테스트 1+문서 2.
- [러너 10:43] scout done — 자동화 일정이 `@midnight`·`@annually` 를 받아들이고, 모르는 `@별칭` 은 "다섯 필드" 가 아니라 지원 별칭을 알

## 구현 노트
- `cronAliases` 에 `@midnight`(=`@daily`)·`@annually`(=`@yearly`) 두 줄을 더하고, 별칭 조회가 빗나간 `@` 입력은 다섯 필드 검사 전에 정렬한 지원 목록(`cronAliasNames()`, `sort.Strings`)을 담은 `ErrInvalid` 로 거절한다. `@reboot` 은 `next_run_at` 기반 스케줄러에 뜻이 없어 일부러 미지원이고 같은 오류로 거절된다. 커밋 1d42074, 프로덕션 1파일.
- 정찰이 미확인으로 남긴 것을 둘 다 확인했다: `httpapi/automations.go:280` 이 `errors.Is(err, automation.ErrInvalid)` 로 400 을 고르고 `err.Error()` 를 응답 `message` 에 그대로 실으므로 새 목록이 API 사용자에게 닿는다. PDF 재생성도 실제로 돌렸다.
- **확신 없는 곳**: PDF 는 내용 스트림이 서브셋 폰트라 바이트 안 ASCII 검색으로 글자를 확인할 수 없었다(기존 `@hourly` 조차 안 잡힌다). 그래서 인쇄 직전 DOM 의 `innerText` 에 여덟 토큰이 모두 있는 것으로만 확인했고, **PDF 지면을 눈으로 본 것은 아니다**(`pdftoppm` 없음). 1604행은 번호 목록이 아닌 한 줄 문단이라 과거의 번호 유실 함정은 피했다고 본다.
- PDF 크기는 1,858,042→1,840,414 바이트. 과거 기록의 ~750KB 급 하락은 없었고 `PAGE ERROR` 줄도 이번엔 안 났다(mermaid 가 CDN 에서 로드됨).
- 일부러 하지 않은 것: `cronValue` 의 `strconv.Atoi` 가 `+5` 를 받는 느슨함(기존 저장된 일정을 깨뜨릴 수 있어 ideas.json 에 단독 과제로 남김), 감싸는 `SAT-SUN` 범위(Vixie cron 도 미지원 — `rejected` 로 내렸다), 봄철 DST 건너뜀(기존 테스트가 의도로 못 박은 계약), 월말·시간대 계약 표(차선 후보 — 1순위가 성립해 미실행, `pending` 유지).
- 다음 역할이 조심할 것: 새 테스트 둘 다 DB·네트워크 없이 돈다(`go test ./internal/automation`). 오류 문구를 또 손대면 `TestScheduleRejectsUnknownAliasesWithSupportedList` 가 "five fields" 부재와 지원 별칭 7개 전부 포함을 단언하므로 목록을 `cronAliases` 에서 만드는 성질을 깨지 말 것. 웹·DB 통합·브라우저 E2E 는 Go 파서 안쪽 변경이라 돌리지 않았다.
- [러너 10:48] brief accepted — 채택 — 과제서의 재현(`@midnight`·`@annually`·`@reboot` 이 모두 "five fields" 로 거절)이 지금 코드와 정확히 일치했고, 건드릴 �
- [러너 10:49] verify passed — 검증 7개 통과 (auto)
