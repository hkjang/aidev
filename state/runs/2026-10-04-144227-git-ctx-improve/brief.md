- 과제: 알림 발송 루프의 10초 고정 틱을 주입 가능하게 해서 `internal/app` 테스트의 남은 벽시계를 줄이기 (가치 4 / 위험 2 / 작업량 M)
- 왜: `internal/notification/service.go:76` 의 `New` 가 `tick: 10 * time.Second` 를 하드코딩하고 `Run`(:79-91)이 `RunOnce` 한 번 뒤 틱까지 기다리므로, 픽스처가 `notifications` 행을 넣은 뒤 배달/포기를 보려면 최대 10초를 그냥 흘린다. 이것이 `TestPlatformChainIntegration`·`TestPlatformDegradationIntegration`(각 10.31s, 2026-10-03 회차 실측) 두 테스트 시간의 사실상 전부이고, 2026-10-03 에 워커 폴링(`WorkerPollInterval`)으로 성공한 것과 **같은 모양·같은 관례**로 풀 수 있다.
- 수용 기준:
  1) `config.Config` 에 새 필드(예: `NotificationTickInterval time.Duration`)가 생기고 `FromEnv` 는 이 필드를 채우지 않는다(`WorkerIdentity`·`WorkerPollInterval` 선례 그대로, 테스트 전용 주입 경로).
  2) 프로덕션 기본값이 그대로 10초다 — `notification.New` 가 여전히 10초를 쓰고, 비양수 값은 무시한다(`Worker.SetPollInterval` 과 같은 방어).
  3) 새 테스트 1개가 "설정한 틱이 알림 루프에 실제로 닿는다"를 증명한다: 배선이 없으면 실패하고(주입 전 red 를 반드시 눈으로 확인) 배선 뒤 통과한다. 손으로 만든 대역 없이 실제 `New(ctx, config.Config{…})` → 실제 `a.Handler().ServeHTTP` 로 설정 저장 + `notifications` 행 삽입 → 실제 `notification_deliveries` 행으로만 단정한다.
  4) `go test -tags sqlite_fts5 -count=1 ./internal/app` 의 벽시계가 같은 세션 안에서 수정 전보다 줄어든다(같은 명령 두 번 실측, 수정 전/후 숫자를 둘 다 기록). `--- PASS`/`--- SKIP` 개수는 새로 더한 테스트 1개를 빼면 변하지 않는다(`-v` 로 전후 비교).
  5) 줄어든 원인이 알림 틱임을 **되돌림 프로브**로 증명한다: 테스트 상수만 10초로 되돌리면 두 테스트가 다시 10초대로 돌아오는 것을 실측해 적는다. (같은 커밋에서도 100~117s 편차가 있으므로 "전후 숫자" 만으로는 증명이 안 된다 — 반드시 같은 세션, 반드시 프로브.)
  6) 재시도 백오프(`service.go:164` 의 `time.Minute * 1<<min(attempts-1,6)`)와 `MaxAttempts` 검증(`:420`)은 **건드리지 않는다**.
- 건드릴 파일 (프로덕션 3개 + 테스트 2개):
  - `internal/config/config.go` — `WorkerPollInterval`(26-36행)과 같은 주석 양식으로 `NotificationTickInterval` 추가. `FromEnv` 는 무수정.
  - `internal/app/app.go` — `startBackground`(359행) 안, `backgroundWorker.SetPollInterval(a.cfg.WorkerPollInterval)`(:368) 바로 아래 양식으로 `a.notifier.SetTickInterval(a.cfg.NotificationTickInterval)` 한 줄. `a.notifier` 는 `app.go:304` 에서 이미 만들어져 있고 고루틴 기동은 `:396-399` 이므로, 설정은 `a.wg.Add(4)`(:383) **앞에** 둘 것(기동 후에 쓰면 경쟁이다).
  - `internal/notification/service.go` — `DefaultTickInterval = 10 * time.Second` 상수화 + `New`(76행)가 그것을 쓰게, `func (s *Service) SetTickInterval(d time.Duration)`(비양수 무시) 추가. `Run`·`RunOnce` 본문은 무수정.
  - `internal/app/live_integration_test.go` — **알림 배달을 기다리는 두 픽스처만** 새 상수(예: `testNotificationTick = 25 * time.Millisecond`)를 넘긴다: `:65`(`TestPlatformChainIntegration`, 대기는 `:178`) 와 `:563`(`TestPlatformDegradationIntegration`, 대기는 `:647`). 같은 파일의 나머지 5개 픽스처(`:702`·`:871`·`:1000`·`:1152`·`:1395`)와 `app_test.go` 는 알림을 기다리지 않으므로 **넘기지 말 것** — `RunOnce` 는 매 틱마다 `a.notificationDeliveryConfig` 로 설정 행을 읽으므로, 25ms 틱을 전부에 주면 공유 in-memory SQLite 에 초당 40회 읽기를 더해 `TestTheProbesAnswerWhileTheDatabaseIsBusy` 같은 부하 테스트에 잡음을 심는다. `waitFor` 의 90초 상한·200ms 폴링은 **건드리지 말 것**(2026-10-03 회차와 같은 금지).
  - `internal/app/notification_tick_test.go`(새 파일) — 수용 기준 3의 테스트. `t.Parallel()` 쓰지 말 것.
- 검증 명령:
  - `go test -tags sqlite_fts5 -count=1 ./internal/app ./internal/notification`
  - 시간 비교: 수정 전/후 각각 `go test -tags sqlite_fts5 -count=1 ./internal/app`, 그리고 `-v` 로 한 번 더 받아 `--- PASS` 개수 비교.
  - `go test -tags sqlite_fts5 -race -count=1 ./internal/app ./internal/notification` (틱을 25ms 로 줄이면 루프가 훨씬 자주 도므로 경쟁 보고 0 을 반드시 확인)
  - `go test -tags sqlite_fts5 -count=1 ./...` / `go vet ./...` / `go build -tags sqlite_fts5 ./...` / `gofmt -l ./cmd ./internal`(빈 출력) / `bash scripts/verify-version-sync.sh`
- 위험과 피할 것:
  - `internal/version` 올리기·릴리즈 노트·CHANGELOG 금지(러너가 별도 릴리즈 세션에서 한다).
  - `.github/workflows` 는 한 글자도 바꾸지 말 것 — `git diff --stat .github/` 가 빈 출력이어야 한다.
  - 재시도 백오프를 줄이는 유혹을 거부할 것. 두 테스트는 백오프에 의존하지 않는다: 포기 쪽은 `live_integration_test.go:593` 이 `maxAttempts:1` 로 저장하므로 **첫 시도 실패가 곧 `dead`** 이고, 배달 쪽은 `:100` 이 `maxAttempts:3` 이지만 첫 시도에 성공한다(소스에서 확인). 즉 틱만 줄이면 되고 "몇 번 만에 포기" 계약은 그대로 남는다 — 백오프를 건드리면 그 계약이 바뀐다.
  - `service.go:101` 의 리스 만료(`-5*time.Minute`)도 그대로 둘 것. 틱이 25ms 가 되어도 리스 창은 5분이므로 `sending` 행이 조기 회수되지 않는다(이 추론은 소스 기반, 실측 미확인).
  - `internal/app` 테스트 시간은 같은 커밋에서도 100~117s 로 흔들린다. 반드시 같은 세션에서 두 번 재고, 원인은 되돌림 프로브로 증명할 것.
  - 보호 경로(auth/session/migrations) 는 이 과제와 무관하다 — 건드리지 말 것.
  - 프로덕션 파일은 3개로 끝난다(config·app·notification). 2026-10-03 회차와 같은 이유로 2개로는 줄일 수 없다(`App` 의 유일한 입력이 `config.Config`).
- 차선 후보: `fencesContent`(internal/mcp/budget.go) 의 도구 이름 허용목록을 `format.go` 의 `contentFence` 사용처와 계약으로 묶기 — PR #44 비평의 "승인해도 남는 우려 (1)". 세 번째 포매터가 `contentFence` 를 쓰기 시작하면 아무 테스트도 깨지지 않고 그 도구만 조용히 구동작으로 남는다. `internal/mcp` 단독 테스트가 1초대라 검증이 싸다.
