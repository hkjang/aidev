- 과제: `ParseRetryAfter` 가 상한 없는 재시도 힌트를 무인 워커의 수면 시간으로 그대로 넘기지 않게 한다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/policy/retry.go:ParseRetryAfter` 는 공급자 **에러 메시지 문자열**에서 긁어낸 수를 아무 상한 없이 `time.Duration` 으로 바꿔 돌려주고, `DecideRetry` 의 `FailureRateLimit` 분기(`retry.go:118-120`)는 `*retryAfter > 0` 이기만 하면 그것을 그대로 `Delay` 로 쓰며, `cmd/goalforge/main.go:885`(파싱·결정)→`908-911`(`policy.WaitForRetry`)의 무인 `run --loop` 워커가 그 시간만큼 실제로 잠든다. 함수 주석은 "short-wait hint", 결정 사유 문자열은 "short rate limit; honoring Retry-After" 라고 말하지만 코드에는 짧다는 보장이 하나도 없어서, `429 Too Many Requests. Retry-After: 86400`(공급자가 일일 상한에 실제로 돌려주는 값) 한 줄이면 워커가 printf 한 줄만 남기고 24 시간 멈춘다 — 사람이 보지 않는 자율 실행에서 이것은 장애와 구별되지 않는다.

- 수용 기준:
  1) 문서화된 상한을 **넘는** 힌트에 대해 `ParseRetryAfter` 가 `nil` 을 반환하고, 그 결과 `DecideRetry(FailureRateLimit, …)` 가 기존 `Backoff` 사다리(최대 10분 + 지터)로 떨어진다. 새 `RetryAction` 이나 새 사유 문자열을 만들지 않는다.
  2) 기존 `TestParseRetryAfter` 의 정상 입력 6개가 기대값 변경 없이 그대로 통과한다. 특히 `"rate limit reached, retry after 1 hour"` → 정확히 `time.Hour` 는 지금도 honored 여야 하므로 **상한은 포함(inclusive)** 이어야 한다(`hint > max` 만 거절, `hint == max` 는 통과).
  3) 단위 테스트가 손으로 만든 `*time.Duration` 을 주입하지 말고 `main.go:885` 와 **같은 합성**으로 증명한다: `msg := "429 Too Many Requests. Retry-After: 86400"` 에 대해 `policy.DecideRetry(policy.ClassifyFailure(errors.New(msg), ""), 1, 3, policy.ParseRetryAfter(msg), nil)` 의 `Delay` 가 사다리 값(≤ 10분)이고 24시간이 아님을 단언한다. (`ClassifyFailure` 가 이 문자열을 `FailureRateLimit` 으로 분류하는 것은 `retry.go:83` 의 `"429"`·`"too many requests"` 키워드로 읽어서 확인했다. `jitter=nil` 이면 지터가 꺼져 결정적이다 — `Backoff` 주석.)
  4) 표현 불가능한 곱도 같은 가드에 걸린다: `"retry in 99999999999 hours"` 같은 입력이 `nil` 을 반환한다(아래 "미확인" 참조).

- 건드릴 파일 (프로덕션 1개):
  - `internal/policy/retry.go:ParseRetryAfter` — 상한 상수(예: `maxRetryAfter = time.Hour`)를 함수 바로 위에 주석과 함께 두고, 두 경로 모두에서 **변환 전에** 거른다.
    - 헤더 경로(`retryAfterHeader`, `strconv.Atoi`): 지금 `seconds > 0` 만 본다. `time.Duration(seconds) * time.Second` 로 바꾸기 전에 `seconds` 가 상한 초과인지 먼저 판정할 것(`int` 곱을 먼저 하면 큰 값에서 int64 랩어라운드가 나서 음수가 되어 버리고, 그러면 "상한 초과" 와 "음수" 를 구별할 수 없다).
    - 구문 경로(`retryAfterDuration`, `strconv.ParseFloat`): `amount` 가 아직 `float64` 인 자리에서 `amount * float64(unit) > float64(maxRetryAfter)` 와 비유한 수를 거절한 **뒤에** `time.Duration(...)` 으로 변환할 것. 이것은 `cmd/goalforge/main.go:dayDuration` 에서 이미 채택된 것과 같은 모양의 가드다(커밋 `d69110f`/`b465e1a`).
    - 주석의 "short-wait hint" 서술에 실제 상한을 적어 코드와 문서가 같은 말을 하게 할 것.
  - `internal/policy/retry_test.go` — `TestParseRetryAfter` 에 거절 케이스를 더하고, 위 3)의 합성 테스트를 새 테스트 함수로 추가. 기존 케이스의 기대값은 바꾸지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/policy -count=1` (이번 회차에 기준선 미실행 — 구현자가 **고치기 전에** 한 번 돌려 green 을 확인하고 시작할 것)
  - `go test ./... -count=1` (이전 회차 실측 약 60초대, sqlite 46s·observer 21s가 지배적. `pushurl=DISABLED` 로 인한 기존 skip 4건은 정상)
  - `go build ./...`, `go vet ./...`, `gofmt -l ./cmd ./internal`(무출력), `go mod tidy` 후 `go.mod`/`go.sum` 드리프트 없음 — CI(`.github/workflows/ci.yml`)가 보는 항목.

- 위험과 피할 것:
  - **상한값은 판단이지 버그 수정이 아니다.** 1시간을 권하는 근거: ① 기존 테스트가 1시간을 honored 로 고정하고 있어 그 아래로 내리면 기대값을 바꿔야 한다 ② 계정 쿼터 소진은 `FailureAccountQuota`→`WaitQuotaReset`(폴링 금지)로 **이미 따로** 분류되므로, `FailureRateLimit` 에 남는 긴 힌트는 사다리 폴링이 더 싸고 관측 가능하다(매 재시도마다 printf). 이 가정이 틀리면(= 긴 rate-limit 힌트를 정말 존중해야 하면) 상한을 24시간으로 올리는 선택지도 성립하지만, 그때도 ①·④ 가드는 그대로 유효하다. 고른 값을 코드 주석에 근거와 함께 적을 것.
  - `Backoff`·`backoffSteps`·`DecideRetry` 매트릭스·`ClassifyFailure` 키워드 목록·`WaitForRetry` 는 건드리지 말 것. `FailureRateLimit` 분기에 `maxAttempts` 검사가 없는 것(= rate limit 은 영원히 재시도)은 **별개의 설계 선택**이며 이번 범위 밖이다.
  - 다른 기간 파서와 통합 금지(운영자 교훈). `internal/notify/suppress.go:repeatWindow` 는 `time.ParseDuration` 을 쓰고 이 결함이 없음을 이번에 읽어서 확인했다 — 손대지 말 것.
  - `cmd/goalforge/main.go` 는 건드릴 필요가 없다. 호출부는 이미 `policy.ParseRetryAfter(runErr.Error())` 한 줄이고, 고칠 자리는 파서 안이다.
  - **미확인**: `time.Duration(float64)` 의 범위 초과 변환 결과를 이 세션에서 직접 측정하지 못했다(`go run` 실행이 이 환경에서 승인되지 않음). 이전 회차 기록은 amd64 에서 `MinInt64`(음수), arm64 에서 `MaxInt64`(양수 ≈292년)라고 적고 있고 CI 매트릭스에 macos(arm64)가 있다. 구현자는 수용 기준 4)의 테스트를 **쓰기 전에** 해당 입력의 실제 변환 결과를 한 번 측정해, 가드를 끄면 정말 빨강이 되는지(= 인과) 확인할 것. 만약 이 환경에서 이미 음수가 되어 `*retryAfter > 0` 에 걸러진다면 4)는 "계약 고정" 테스트이지 결함 재현이 아니며, 그렇게 보고할 것(arm64 에서는 걸러지지 않는다는 점은 그대로 유효하다).
  - 보호 경로(auth/session/migrations/`.github/workflows`/observer 승인 봉투)는 전혀 건드리지 않는 과제다.

- 차선 후보: **영문 정적 페이지 `docs/index_en.html` 의 손상된 `<title>`·메타데이터 복구** — `og:locale`/`description` 이 한국어로 남아 있고 title 에 반복 손상 문자열이 있다(이전 회차 관찰, 이번에 재확인하지 않음 — 구현자가 먼저 파일을 열어 지금도 그런지 확인하고 시작할 것). head 한 조각으로 범위를 한정하고 페이지 전체 번역으로 번지지 말 것.
