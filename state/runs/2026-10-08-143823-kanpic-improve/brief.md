- 과제: 자정 DST 전환에서 자동화 일정 Next의 날짜 순회가 멈추지 않게 한다 (가치 4 / 위험 2 / 작업량 M)
- 왜: `Schedule.Next`가 현지 자정을 다음 날짜 커서로 만들 때 존재하지 않는 자정이 전날 23시로 정규화되어 같은 날짜를 무한 반복한다. 정상적인 `0 9 * * *` 일정도 다음 실행 시각 계산이 끝나지 않으므로, 날짜 순회를 시간대 전환에서 분리하면 일정 저장과 스케줄러가 이 입력 때문에 멎는 것을 막는다.
- 수용 기준: 1) `ParseSchedule("0 9 * * *", "America/Santiago")`에서 `Next(2026-09-05T13:00:00Z)`가 `2026-09-06T12:00:00Z`를 반환하고, 이어서 호출하면 `2026-09-07T12:00:00Z`를 반환한다. 2) 같은 시간대의 `30 0 * * *`는 사라진 9월 6일 00:30을 실행하지 않고 9월 7일 00:30(03:30Z)을 반환한다. 기존 New_York 봄 DST 건너뛰기, 일·요일 OR, 월말·윤일·별칭과 UTC 반환 계약도 유지한다. 3) 실제 `ParseSchedule`→`Next`의 반환 시각으로 증명하며, 수정 전 재현은 외부 `go test -timeout`으로 제한하여 무한 대기가 CI를 붙잡지 않게 한다. 소스 문자열 검사·손으로 주입한 날짜 대역은 쓰지 않는다.
- 건드릴 파일: `internal/automation/schedule.go:Schedule.Next` — 바깥 날짜 커서를 현지 자정이 아닌 시간대 전환 없는 달력 날짜(UTC에 연·월·일만 실은 커서 권장)로 순회한다. 후보 시각은 계속 `s.location`으로 만들고 실제 연·월·일·시·분 대조를 유지한다. `internal/automation/schedule_test.go` — 위 두 Santiago 사례와 날짜가 통째로 생략되는 Pacific/Apia의 2011-12-30 사례를 실제 반환 시각으로 검증한다(총 2파일, 프로덕션 1파일).
- 검증 명령: 저장소 루트에서 `go test ./internal/automation -run TestSchedule -v -count=1 -timeout=10s`, `go test ./internal/automation -count=1 -timeout=30s`, `go test ./...`, `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal ./pkg`, `./scripts/check-release-docs.sh`, `./scripts/check-commit-identities.sh HEAD`. 앞의 표적 테스트는 새 테스트 이름도 TestSchedule로 시작하게 한다. 정찰이 실행한 `go test ./internal/automation -count=1`은 0.012s, `go test ./...`는 exit 0. vet/build/문서 체크는 이번 정찰에서는 미실행이며 CI/기존 기록에 있는 구현 후 검증 명령이다.
- 위험과 피할 것: 범위 밖은 cron 문법·step 부호·별칭·`matchesDay`·`cronValue`·DB·서비스·HTTP·auth·migrations·workflows·가이드/PDF다. `Add(24*time.Hour)`로 현지 날짜를 넘기거나 자정을 정오로만 바꾸는 처방은 피한다(23/25시간짜리 날 및 날짜 전체가 생략되는 경우). 기존 `maxScheduleLookaheadYears=8`과 마지막 날짜 포함 범위를 유지하도록 탐색 한계도 날짜 커서와 같은 달력 기준으로 비교한다. 후보 생성 후 현지 연·월·일 비교를 지우면 없는 날이 다음 날의 실행으로 바뀐다. DST 가을 중복 시각을 두 번 실행하도록 넓히지 않는다. 운영 DB·HTTP 요청 정지 자체는 미실행이며 영향은 아래 실제 호출 배선을 읽어 추론했다.
- 차선 후보: cron step의 부호 붙은 숫자를 거절한다 (가치 2 / 위험 2 / S) — 1순위의 날짜 순회 결함이 구현 환경에서 재현되지 않을 때만 선택. `parseCronField`의 stepRaw에 기존 `isDecimalDigits`를 적용하고 기존 `step must be between` 오류를 유지한다. `*/+2`, `1-5/+2`, `1/+2`, hour/day/month/weekday의 +step을 `ErrInvalid`로 거절하고 `*/02`와 정상 step의 Next는 유지한다. 같은 두 파일만 수정하며 저장된 +step 일정이 advanceSchedule 재파싱에서 거절될 호환성 위험을 기록한다. step 상한·cronValue는 함께 바꾸지 않는다.

확인한 근거와 재현

- 기준 SHA `6e71a7a`, 최신 릴리즈 v0.264.0. 초안은 step 검증으로 먼저 저장했으나 정상 일정의 무한 순회를 실행으로 확인하여 최종 선택을 변경했다.
- `schedule.go:Next`의 바깥 for는 `time.Date(date.Year(), date.Month(), date.Day()+1, 0,0,0,0,s.location)`으로 증가한다. 실제 Go 실행에서 Santiago 커서는 `2026-09-05 00:00 -0400` → `2026-09-05 23:00 -0400` → 같은 시각 → 같은 시각이었다. `0 9 * * *`의 Next는 200ms 안에 반환하지 않았다. 이 시간 측정만으로 무한 반복을 단정한 것이 아니라 커서 비전진과 루프 본문을 함께 확인했다.
- 저장소 파일은 바꾸지 않았다. 지정된 run 디렉터리의 `probe_schedule_test.go`와 `probe-overlay.json`으로 원본 테스트를 메모리 빌드에만 대체했다. 재현 명령: `go test -overlay=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-143823-kanpic-improve/probe-overlay.json ./internal/automation -run TestScoutMidnightProbe -v -timeout=3s`. 이 탐색용 probe는 사실을 출력하고 PASS하므로 회귀 테스트로 복사하지 말 것.
- `service.go:nextScheduleTime`과 `advanceSchedule`이 프로덕션 `ParseSchedule`→`Next`를 직접 호출한다. `service.go`의 `case TriggerSchedule` 검증 분기는 cron을 정규화한다. `httpapi/automations.go:writeAutomationError`는 `ErrInvalid`를 400으로 내보낸다. 이번 과제는 이 배선을 바꾸지 않는다.
- Santiago의 수정 후 기대 시각은 현지 09:00/00:30과 UTC 오프셋으로 산정했다. 수정 후 실행은 구현자 몫이다. Apia 사례의 현재 실패 여부는 정찰에서 미실행: `0 9 * * *`, after `2011-12-29T19:00:00Z` → 다음 존재하는 12월31일 09:00(`2011-12-30T19:00:00Z`)를 대조 사례로 검증할 것.

접근 비교와 구현 순서

- 권장: UTC 달력 커서로 날짜만 순회, 실제 예약 시각은 현지 시간으로 생성. 1파일의 날짜 순회만 바꾸며 문법·저장 형식은 보존한다. 가장 중요한 가정은 UTC 커서의 연·월·일·요일이 원래 선택하려던 현지 달력 날짜와 동일하다는 것; 탐색 경계와 생략일 테스트로 확인한다.
- 대안: 현지 정오 커서. 수정량은 작지만 날짜가 통째로 생략되는 시간대의 별도 처리가 필요하여 선택하지 않았다.
- 대안: 분 단위 절대시간 순회 또는 외부 cron 라이브러리. 분 단위 순회는 8년 탐색 비용이 커지고 라이브러리는 현재 OR·DST·별칭 계약을 바꾸므로 이번 범위에서 제외했다. 보류는 정상 입력으로 루프가 멎는 결함을 남겨 선택하지 않았다.
- [미착수] 1. 위 재현을 `-timeout=10s`로 빨강 확인하고 Next의 날짜 순회 및 경계만 수정하여 새 Santiago 테스트와 기존 TestSchedule을 초록으로 만든다. 증명은 첫 표적 명령. 사람 승인 대기 없음; 기대 시각/기존 계약이 충돌하면 과제서 수정 사유를 journal에 남기고 범위를 넓히지 않는다.
- [미착수] 2. Apia 및 다음날 연속 호출 대조를 추가하고 패키지 검증 명령을 통과시킨다. 사람 승인 대기 없음; 기존 New_York DST 테스트의 기대값 변경 금지.
- [미착수] 3. 전체 Go 테스트·vet·build·서식·문서/커밋 검사로 마무리하고 실제 결과와 미검증 범위를 기록한다. 이후 비평 단계가 검토한다.

산정 근거

- 하향식 숫자 맞추기가 아닌 작업 분해 추정: 재현/테스트 7–10분, 날짜 커서와 경계 수정 8–12분, 대조 및 전체 검증 8–12분 = 기본 23–34분. 알려진 불확실성(생략일·경계 검증)에 별도 contingency 5–8분, 총 28–42분. 경험적 신뢰도는 중간이며 확률로 보정된 추정이 아니다. 관리 예비비는 0분(새 범위는 별도 회차).
- 유사 사례로 10-02의 일정 OR 수정(프로덕션 1+테스트 1, 실제 Next 대조)을 참고했다. 실제 소요 시간 기록은 없어 숫자 교차검증은 할 수 없고 범위 유사성만 확인했다. 환경 설치·DB 마이그레이션·웹·PDF는 제외; 기존 Go 환경과 캐시를 사용한다.
- 범위·가정·작업 분해·불확실성을 기록하는 원칙은 [GAO 추정 가이드](https://www.gao.gov/products/gao-20-195g)를 참고했다. 위 분 단위 숫자는 해당 문헌의 수치가 아니라 이번 코드와 실행에 근거한 정찰자의 추정이다.
- 적용 스킬: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `.../technology/skills/implementation-planning/SKILL.md`, `.../technology/skills/solution-exploration/SKILL.md`. 전용 Skill 도구가 없어 로컬 SKILL.md를 직접 읽었다.
