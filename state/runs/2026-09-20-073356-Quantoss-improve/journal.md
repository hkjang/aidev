# 회차 노트 2026-09-20-073356-Quantoss-improve — Quantoss
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:34] base pinned — main@fd8628b
- [러너 07:34] autonomy release — 

## 정찰 노트
- [정찰 07:4x] 선택: toss.Client 백오프 Sleep 주입(가치 3/위험 2/S). 실측 `go test -count=1 -run TestSellUnknownResult… ./internal/broker` = 31.8s 로 전체 검증 35s 의 90% — 매 회차 구현·비평·수리가 이 시간을 반복 지불하므로, 같은 가치 3 인 gap_reclaim Validate(효과가 설정 실수 때만 드러남)보다 앞세웠다. CI 워크플로는 토큰 스코프 미확인으로 계속 보류.
- 추측: broker 테스트 시간 중 toss 백오프 외 잔여(`live.go:619` `time.Sleep(attempt+1)s`)가 얼마인지는 미실측 — 주입 후 10초 넘게 남으면 그쪽이 원인일 수 있음(ideas.json 에 후속으로 기록).
- 조심: `internal/toss` 는 토큰 위험 구역 — `ensureToken`/캐시 함수와 `ws.go` 재접속 backoff, `limiter.acquire` 의 Sleep 은 건드리지 말 것. 테스트 시퀀스(`statuses` 10회 ERR)를 줄여 시간을 아끼는 방식은 운영자 원칙("진짜 런타임 경로로 증명") 위반이라 금지. 재시도식·로그 문구 변경 없음 = 운영 동작 불변이어야 함.
- [러너 07:38] scout done — `toss.Client` 재시도 백오프의 `time.Sleep` 을 클라이언트 필드로 주입 가능하게 해서 `internal/broker` 테스트의 �

## 구현 노트
- [구현 07:4x] `toss.Client` 에 `Sleep func(time.Duration)`(nil→`time.Sleep`) 필드 + `(c *Client) backoff` 메서드화, `request()` 호출부 3곳 갱신, `sell_test.go` 에 주입 1줄, `client_test.go` 신규 2테스트. 커밋 31b8bce9. 전체 `go test -count=1 ./...` 35s → 8.3s, 문제 테스트 31.8s → 2.68s. 주입 줄 제거 시 32.1s 복귀 실측.
- 확신 없는 곳: `TestBackoffNilSleepFallsBackToTimeSleep` 은 진짜 `time.Sleep` 을 [0,0.5s) 만큼 잔다(단언은 "<0.5s") — 극도로 느린 CI 에서 스케줄링 지연이 100ms+ 나면 경계에서 흔들릴 수 있음. `TestBackoffUsesInjectedSleep` 의 "elapsed < 2s" 단언은 httptest 3회 왕복이라 여유 충분.
- 검증 못 한 것: 운영(실토스) 경로는 돌리지 않음 — `NewClient` 가 Sleep 을 채우지 않으므로 코드상 `time.Sleep` 과 동일하고, 대기식·로그·재시도 횟수는 diff 로 불변 확인.
- 일부러 하지 않은 것: `limiter.acquire`(client.go:70)·`ws.go` 재접속 backoff·토큰 함수는 과제서대로 미변경. `live_test.go` 주입(선택 항목)은 그 테스트가 5xx 경로를 안 타서 효과 0 → "효과 없는 변경" 원칙으로 생략. broker 잔여 7.7s(`live.go:125/619` sleep, limiter TPS 대기)는 ideas.json 에 후속으로 기록.
- 다음 역할이 조심할 것: `internal/toss` 테스트는 `package toss`(내부) 라 `c.request`/`c.backoff`/`c.sleep` 을 직접 부른다 — 시그니처를 바꾸면 같이 고쳐야 함. `-race` 는 toss·broker 두 패키지만 돌렸음(전체는 `-race` 없이).
- [러너 07:42] brief accepted — 채택 — 과제서의 근거(31.8s 실측, backoff 패키지 함수·호출부 3곳)가 코드와 정확히 일치했고 수용 기준 1~5 를 모두 만족(�
- [러너 07:42] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- [비평 07:5x] 확인: diff 는 fd8628b8..HEAD 단일 커밋 3파일(client.go +Sleep 필드/메서드화, sell_test.go 주입 1줄, client_test.go 신규). 대기식·로그 문구·재시도 횟수·`limiter.acquire`·`ws.go`·토큰 함수 불변을 diff 로 확인. `NewClient` 는 Sleep 을 안 채우므로 운영 경로는 `time.Sleep` 그대로.
- 실측: `go build`·`go vet`·`gofmt -l` 깨끗, `-race` toss+broker 통과, 전체 `-count=1` 8.4s. sell_test.go 주입 줄을 잠시 지우고 돌리면 32.7s 복귀(복원함) — 주입이 실제로 효과를 내고, `TestBackoffUsesInjectedSleep` 은 Sleep 호출 횟수·구간·elapsed<2s 를 단언해 backoff 가 Sleep 을 무시하면 반드시 실패한다.
- 못 본 것: 실토스 경로 미실행(구현자와 동일). 보안·개인정보 영향 없음(외부 입력·비밀값·저장 데이터 변경 없음).
- 남는 우려(승인): `TestBackoffNilSleepFallsBackToTimeSleep` 은 실제 [0,0.5s) 잠 + "<500ms" 단언이라 rand≈0.49 에 스케줄링 지연이 겹치면 드물게 흔들릴 수 있음 — 다음 회차에 base 를 음수로 두거나 상한을 1s 로 넓히면 결정적. 또 `client_test.go:53` 의 `*calls` 를 atomic 없이 읽는 Fatalf 메시지(실패 시에만, 서버는 이미 응답 완료라 실질 경합 없음).
- 릴리즈 노트용: 운영 동작 변화 없음 — 테스트 벽시계 35s→8.4s 가 전부. 잔여 broker 7~8s 는 `live.go` sleep/limiter 대기(ideas.json 후속).
- [러너 07:44] review approved — 리뷰 승인 (risk=low)
- [러너 07:45] pr created — https://github.com/hkjang/Quantoss/pull/74
- [러너 07:45] base rebased — c7f95f1, 재검증 통과
- [러너 07:46] ci passed — 검사 없음 — 정책으로 허용
- [러너 07:46] merge done — 3da621f
- [러너 07:47] release skipped — 릴리즈 안 함: 릴리즈 이력이 전혀 없음: git tag 0개(커밋 5274개), 버전 파일(VERSION/version.go/package.json 등) 없음, CHANGELOG/릴리즈 노트 없음, .github/workflows 없�
