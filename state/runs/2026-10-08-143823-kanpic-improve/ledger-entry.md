## 2026-10-08
- 선택: 자정 DST 전환에서 자동화 일정 Next의 날짜 순회가 멈추지 않게 한다 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 현지 자정이 전날 23시로 정규화되어 같은 날짜를 반복하던 `Schedule.Next`의 날짜 커서와 8년 탐색 경계를 UTC 달력으로 옮겼으며, 경계의 시·분과 현지 후보 시각의 연·월·일·시·분 검사를 유지했다. 실제 `ParseSchedule`→`Next`에서 Santiago 09:00 연속 실행(9/6·9/7 12:00Z), 사라진 00:30 건너뛰기(9/7 03:30Z), Apia 생략일(2011-12-30 19:00Z)·명시한 12/30이 다음 날로 바뀌지 않음(2012-12-29 19:00Z), 월말·가을 DST 1회·8년 윤일 경계·없는 날짜의 ErrInvalid·UTC 반환을 검증하고 원본 프로덕션 코드 overlay에서 Santiago의 10초 timeout이 다시 발생함을 확인했다. `go test ./internal/automation -run TestSchedule -v -count=1 -timeout=10s`(ok 0.020s), `go test ./internal/automation -count=1 -timeout=30s`(ok 0.019s), `go test ./...`(exit 0, automation 0.015s·나머지 테스트 패키지 캐시), `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal ./pkg`(출력 없음), `./scripts/check-release-docs.sh`(release docs ok: v0.264.0), `./scripts/check-commit-identities.sh HEAD`가 모두 통과했으며 프로덕션 1파일·테스트 1파일을 hkjang 명의의 f2c1ad1로 커밋했다(DB·HTTP·웹·integration 태그 테스트는 범위 밖으로 미실행).
- 실패 재현: `panic: test timed out after 10s` / `TestScheduleNextTraversesCalendarDates/Santiago_midnight_DST (10s)` — 수정 전 표적 명령의 원문이며 test-red.log에 전체 출력 보존; 별도 실행의 Santiago_missing_00:30과 Apia_missing_day도 동일하게 10초 timeout, 원본 overlay 재실패는 test-revert.log에 보존.
- 보류 아이디어: cron step의 +부호 거절 (가치 2 / 위험 2 / S) — 1순위 재현으로 차선 미실행; 저장된 +step 호환성 주의.
- 보류 아이디어: IMPORTDATA와 업로드의 LazyQuotes 차이 (가치 2 / 위험 3 / M) — 계약 결정 전 파서 통합 금지.
- 보류 아이디어: 관리자 가이드 외부 호출 오류 코드를 실제 #N/A와 맞춤 (가치 2 / 위험 1 / S) — PDF 재생성 동반, 이번 범위 밖.
- 보류 아이디어: compareKey의 0x10·0b10 식별자 합침 (가치 3 / 위험 3 / S) — 실제 입구 실행과 대사 키 계약 확인 필요.
- 과제서: 채택 — 현재 코드에서 Santiago 자정 커서 비전진과 실제 Next timeout이 재현됐고, 지정한 2파일만 수정하여 모든 반환 시각 수용 기준을 통과했다.
