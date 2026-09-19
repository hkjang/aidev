# 과제서 2026-09-20 — Quantoss

- 과제: `toss.Client` 재시도 백오프의 `time.Sleep` 을 클라이언트 필드로 주입 가능하게 해서 `internal/broker` 테스트의 벽시계 31초 제거 (가치 3 / 위험 2 / 작업량 S)

- 왜: 2026-09-18 zz_dbg 분리 후에도 `go test -count=1 ./...` 가 ~35초 걸리는데, 그중 31.8초(이번 회차 실측: `go test -count=1 -run TestSellUnknownResultIsResolvedBeforeNewOrder ./internal/broker/` → `ok 31.765s`)가 한 테스트가 실제 `toss.Client` 의 5xx 재시도 백오프(`client.go:187 backoff()` — 1·2·4·8초 + 난수, 조회 2회 × 4회 대기)를 진짜로 자는 비용이다. 재시도 로직·횟수·배선은 그대로 두고 "자는 함수" 만 바꿔 끼울 수 있게 하면 전체 검증이 ~3초로 줄어 매 회차 구현·비평·수리 단계가 빨라진다.

- 수용 기준:
  1) `internal/toss/client.go` 의 `Client` 에 `Sleep func(time.Duration)` 필드가 추가되고, nil 이면 `time.Sleep` 을 쓴다(limiter nil 패턴과 같은 "nil 이면 기본값" 관례). `NewClient` 는 필드를 채우지 않아도 되고(nil → 기본) 운영 경로 동작은 바뀌지 않는다.
  2) 패키지 함수 `backoff(attempt, base, reason)` 이 `(c *Client) backoff(...)` 메서드(또는 sleep 함수를 인자로 받는 형태)로 바뀌고, `request()` 의 세 호출부(`client.go:233` network, `:242` 429, `:253` 5xx)가 모두 그것을 탄다. 대기 시간 계산식(`base*2^attempt + rand*0.5`)과 `slog.Warn("재시도 대기", ...)` 로그는 그대로.
  3) `internal/broker/sell_test.go` 의 `sellAPI.client()`(`sell_test.go:95~101`) 에서 `c.Sleep = func(time.Duration) {}` 한 줄로 주입한다. 테스트 서버·상태 시퀀스·단언은 바꾸지 않는다 — `TestSellUnknownResultIsResolvedBeforeNewOrder` 가 여전히 "5xx 10회 → 결과 미확인 → 두 번째 Sell 에서 FILLED 확정, 새 주문 0" 을 진짜 `toss.Client` 재시도 경로로 증명해야 한다(재시도 횟수는 `f.getCalls` 로 관찰됨).
  4) `internal/toss` 에 단위 테스트 1개 추가: `httptest` 서버가 5xx 를 2번 내고 200 을 주는 경로에서 `Sleep` 주입 함수가 호출된 횟수(2)와 전달된 시간이 `1~1.5s`, `2~2.5s` 범위인지, 그리고 `Sleep == nil` 인 클라이언트도 패닉 없이 동작하는지(429 등 사용 안 하는 짧은 경로로 — 실제 sleep 을 피하려면 `Retry-After: 0` 을 주는 429 는 `max(ra,1)` 이라 1초 잠 — 그러니 nil 검사는 별도 테스트로 "backoff 를 직접 부르되 attempt=0, base=0" 처럼 최소 대기로 확인하거나, `Sleep` nil 이면 `time.Sleep` 을 가리키는 헬퍼 `c.sleep()` 을 테스트에서 nil/non-nil 로 확인).
  5) `time go test -count=1 ./...` 가 10초 이내(기대 ~3초), `gofmt -l .` 출력 없음, `go vet ./...`·`go build ./...` 통과.

- 건드릴 파일:
  - `internal/toss/client.go:78~93 Client` — `Sleep func(time.Duration)` 필드 추가(주석: 테스트 주입용, nil 이면 time.Sleep).
  - `internal/toss/client.go:187~191 backoff` — 메서드화 또는 sleep 인자 추가. `client.go:233/242/253` 호출부 3곳 갱신.
  - `internal/toss/client_test.go`(신규) — 위 4).
  - `internal/broker/sell_test.go:95~101 sellAPI.client` — `c.Sleep = func(time.Duration) {}` 주입. `sell_test.go:191` 주석("넉넉히 준다")은 그대로 두어도 됨.
  - 선택: `internal/broker/live_test.go:69~73` 도 같은 주입(현재 느리지 않으므로 필수 아님).

- 검증 명령(이 순서로, 저장소 루트 `/home/hkjang/.cache/auto-improve-wt/Quantoss` 에서):
  1. `gofmt -l .` → 출력 없음
  2. `go vet ./...`
  3. `go build ./...`
  4. `go test -count=1 ./internal/toss/ ./internal/broker/ -v -run 'Backoff|TestSellUnknownResult'` → 새 테스트와 느린 테스트가 통과하고 broker 테스트가 1초 미만
  5. `time go test -count=1 ./...` → 전체 10초 이내(현재 ~35s)
  6. 회귀 확인: `sell_test.go` 의 `c.Sleep = ...` 줄을 잠시 지우고 4번을 다시 돌려 다시 ~31초가 걸리는지 보고 되돌린다(주입이 실제 경로를 타는 증거).

- 위험과 피할 것:
  - `internal/toss` 는 위험 구역(토큰 캐시 — 같은 키로 두 프로세스면 서로 무효화). 이번 변경은 `ensureToken`·`loadTokenCache`·`saveTokenCache`·`invalidateToken` 을 **건드리지 않는다**. `ws.go:106~119` 의 WebSocket 재접속 backoff(`time.After`)도 별개 — 건드리지 않는다.
  - `limiter.acquire` 의 `time.Sleep`(`client.go:70`) 은 토큰 버킷 대기라 짧고 테스트에 영향 없음 — 범위 밖. 같이 바꾸면 "실제 출력·동작이 바뀌지 않는 수정" 으로 반려 근거가 된다.
  - 재시도 횟수(`maxAttempts = 5`)·조건·대기식·로그 문구를 바꾸지 말 것. 테스트 쪽에서 `statuses` 시퀀스를 줄여 시간을 아끼는 접근은 금지 — 운영자 원칙: "프로덕션 배선과 진짜 런타임 객체를 통과하는 것으로 증명".
  - `NewClient` 외 구조체 리터럴로 `toss.Client` 를 만드는 운영 코드는 없음(이번 회차 grep 확인) — 그래도 nil 기본값 처리는 필수.
  - 커밋 메시지는 한국어 한 줄(저장소 관례).

- 차선 후보: `Config.Validate` 가 gap_reclaim 파라미터(`config.go:108~110` GapMin/GapMax/GapPullMin/GapVolMult, `config.go:517~523` 에서 읽음)를 검사하지 않음 — `Validate`(`config.go:650`) 의 `pos`/`nonNeg` 헬퍼로 `0 < GapMin < GapMax`, `GapPullMin >= 0`, `GapVolMult > 0` 추가 + `config_test.go` 의 `TestValidateRejectsDangerousValues`(`:59`) 패턴으로 케이스 추가. (가치 3 / 위험 1 / S)
