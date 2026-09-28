- 과제: `toss.Client.request` 의 429 분기 — 마지막 시도에서 실제 429 오류를 `retry-exhausted`(Status 0) 로 덮어쓰고, 서버가 준 `Retry-After` 를 지수로 곱해 한 요청 안에서 최대 15분을 자는 문제 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/toss/client.go:250` 의 429 분기는 같은 루프의 5xx 분기(`client.go:262`)와 달리 `attempt < maxAttempts-1` 가드가 없어서, 다섯 번째 429 를 받아도 한 번 더 자고 루프를 빠져나가 `client.go:293` 의 `&APIError{Code:"retry-exhausted"}` (Status=0, 서버 code/message/requestId 없음)를 돌려준다 — 호출부가 상태 코드로 분기하는데(`internal/broker/live.go:616` 의 `ae.Status != 429`, `live.go:676` 의 `ae.Status == 404`/`422`, `cmd/quantoss/doctor.go:83` 의 `ae.Status == 403`) 429 였다는 사실이 사라진다. 또 `c.backoff(attempt, max(ra,1), …)` 는 `wait = base*2^attempt + [0,0.5)s` 이므로 서버가 `Retry-After: 60` 을 주면 60→120→240→480초, 한 `request()` 호출이 약 900초 동안 호출 고루틴을 붙잡는다(HTTP.Timeout 은 10초인데). `Retry-After` 는 "이만큼 뒤에 다시 오라" 는 절대 지시라 지수로 증폭할 값이 아니다. 고치면 (a) 레이트리밋 소진이 로그·doctor·broker 에서 429 로 식별되고 (b) 실시간 매매 경로가 서버 지시보다 오래 멈추지 않는다.

- 수용 기준:
  1) 서버가 429 를 `maxAttempts`(5) 번 연속 내면 `request` 가 `*APIError{Status:429}` 를 반환한다(서버 envelope 의 `error.code`/`message` 가 실려 있고, `Code` 가 `"retry-exhausted"` 가 아니다). 요청 횟수는 여전히 5회.
  2) 마지막 시도(attempt == maxAttempts-1)의 429 에서는 더 이상 대기하지 않는다 — 주입한 `Client.Sleep` 호출 횟수가 4회(재시도 사이에만)여야 하고 5회가 아니다.
  3) `Retry-After: 30` 을 주는 서버에 대해 각 대기가 `[30s, 30.5s)` 안이다(현행은 30/60/120/240 으로 증가). `Retry-After` 헤더가 없거나 파싱 불가(HTTP-date 등)이면 현행대로 base=1 의 지수 백오프(`[1s,1.5s)`, `[2s,2.5s)`, …)를 유지한다.
  4) 기존 5xx 경로 회귀 없음: `internal/toss/client_test.go` 의 `TestBackoffUsesInjectedSleep`(503 2회 → 200, Sleep 2회, `[1s,1.5s)`/`[2s,2.5s)`)와 `TestBackoffNilSleepFallsBackToTimeSleep` 이 수정 없이 그대로 통과.
  5) 신규 테스트를 먼저 써서 수정 전 실패(1·2·3 각각), 수정 후 통과를 확인한다. 테스트는 `httptest` + 실제 `toss.NewClient` 로 만든 `*Client` 의 공개 배선(`c.BaseURL`, `c.HTTP = srv.Client()`, `c.Sleep` 주입)을 쓰고, 손으로 만든 대역 트랜스포트를 쓰지 않는다 — 기존 `retryServer` 헬퍼와 같은 방식.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/toss/client.go` — `request()` 안 `case resp.StatusCode == 429:` (250~253행) 한 곳만.
    - 가드 추가: `case resp.StatusCode == 429 && attempt < maxAttempts-1:` — 이렇게 하면 마지막 429 는 아래 `case resp.StatusCode >= 400:` (265행) 으로 떨어져 서버 envelope 을 그대로 파싱한 `*APIError{Status:429, …}` 를 반환한다(5xx 분기가 이미 쓰는 것과 똑같은 패턴이라 새 코드 경로가 아니다).
    - `Retry-After` 존중: `ra > 0` 이면 `c.backoff(0, ra, …)` 로 부른다(attempt=0 → `ra*2^0 + jitter` = `ra + [0,0.5)s`). `ra <= 0`(헤더 없음/파싱 실패)이면 현행대로 `c.backoff(attempt, 1, …)`. `backoff` 의 시그니처·본문·`재시도 대기` 로그 키는 건드리지 않는다.
    - 선택: 파일 머리 주석(`client.go:5` "429 / 5xx 지수 백오프 재시도")에 Retry-After 존중을 한 줄로 반영.
  - `internal/toss/client_test.go` — 테스트 추가. 기존 `retryServer` 를 재사용하지 말고(경로가 5xx 전용) 429 용 핸들러를 새로 하나 더 두는 편이 안전하다. 최소 3개: ① 429 × 5 → `*APIError` Status=429·Code 가 서버 값, 요청 5회, Sleep 4회 ② `Retry-After: 30` 에서 각 대기가 `[30s,30.5s)` ③ `Retry-After` 없음/`Retry-After: Wed, 21 Oct 2026 07:28:00 GMT` 에서 지수 백오프 유지.

- 검증 명령 (이 저장소에서 실제로 도는 것, 이 순서로):
  - `gofmt -l internal/toss/` → 출력 없음
  - `go vet ./internal/toss/`
  - `go build ./...`
  - `go test -count=1 -race ./internal/toss/` (이번 정찰 실측: 캐시 없이 0.139s)
  - `go test -count=1 ./internal/toss/ ./internal/broker/` (정찰 실측 toss 0.139s / broker 7.753s — broker 7.7s 는 `live.go` 자체 sleep 과 limiter TPS 대기라 정상)
  - 마지막으로 `go test -count=1 ./...` 전체

- 위험과 피할 것:
  - **`ensureToken`·401 분기·토큰 캐시(`loadTokenCache`/`saveTokenCache`/`invalidateToken`)는 건드리지 말 것.** 토스는 client 당 유효 토큰이 하나라 재발급이 실거래 프로세스의 토큰을 죽인다(`client.go:91` 주석, 커밋 78ee1393·c9a2645f 가 이 문제를 고친 이력).
  - `backoff` 메서드의 시그니처를 바꾸지 말 것 — 5xx·network 호출부 2곳이 같이 쓰고 기존 테스트 2개가 대기식을 단언한다. 이번 변경은 "429 분기가 backoff 를 어떤 인자로 부르는가" 만 바꾼다.
  - `maxAttempts`(5) 와 요청 횟수를 줄이지 말 것 — 수용 기준 1이 5회 요청을 요구한다. 가드는 "마지막 회차에 자지 않는다" 일 뿐 "덜 시도한다" 가 아니다.
  - 실제 API·`internal/zz_dbg` 는 실행 금지(실거래 토큰 캐시 `data/.token.json` 공유). 테스트는 `httptest` 만.
  - `Retry-After` 를 HTTP-date 로 파싱하는 기능을 새로 넣지 말 것 — 범위 밖이고, 현행 `strconv.ParseFloat` 실패 시 지수 백오프로 떨어지는 동작을 유지하면 된다. (토스가 429 에 `Retry-After` 를 실제로 보내는지는 **미확인** — 보내지 않으면 `ra<=0` 경로로 현행과 동일하게 동작하므로 어느 쪽이든 안전하다.)
  - `.github/workflows` 는 이 저장소에 없다(확인함). 릴리즈 경로는 러너의 `gofmt`/`vet`/`build`/`test` 이므로 위 명령을 전부 통과시킬 것.

- 차선 후보: `internal/toss/client.go` 의 401 재발급 경로 단위 테스트 — `attempt < 2` 조건 때문에 3회차 이후 401 은 `*APIError{Status:401}` 로 즉시 반환되는데 이 경계가 테스트로 고정돼 있지 않다. `Sleep` 주입 덕에 벽시계 없이 `httptest` 로 검증 가능하나, **인증·토큰 캐시 구역이라 읽기(테스트 추가)만 하고 프로덕션 코드는 고치지 말 것**. 1순위가 성립하지 않을 때만.
