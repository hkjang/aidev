- 과제: webhook 대기열 포화 시 기록 실패가 아무 흔적도 남기지 않는 문제를 고치고, 포화·3xx 분기를 실제 왕복 테스트로 고정 (가치 3 / 위험 1 / 작업량 M)
- 왜: `queueWebhook`(internal/httpapi/webhook.go:87-89)의 `default` 분기는 `_ = s.completeWebhookDelivery(...)`로 반환값을 버린다. 그래서 전송 대기열이 가득 찬 동시에 DB가 흔들리면 delivery 행은 `pending`에 영원히 남고 로그에도 아무것도 안 남아, 관리자는 "전송도 안 됐고 실패로도 안 찍힌" 이벤트를 원인 없이 마주한다. 같은 함수의 다른 store 실패는 전부 `s.logger.Warn`으로 보이게 되어 있고(webhook.go:57, 67, 74, 84), 직전 회차(129bdf7)가 webhook.go:191-194에 정확히 이 패턴을 심어 놨는데 이 한 자리만 빠졌다.
- 수용 기준:
  1) 대기열이 포화된 상태에서 `queueWebhook`을 부르고 기록이 실패하면 `s.logger`에 경고가 한 줄 남는다. 필드는 webhook.go:192-193과 같은 식별자만 — `delivery_id`, `event`, `request_id`, `error` — 이고 `payload`·`SigningSecret`·시크릿 경로 값은 절대 포함하지 않는다(감사·로그에 원문 금지 규칙).
  2) 포화 시에는 엔드포인트로 요청이 한 건도 가지 않고, `CompleteWebhookDelivery`가 statusCode 0 + "대기열이 가득 찼습니다" 오류로 정확히 한 번 호출된다.
  3) 대기열에 자리가 있을 때는 기존 동작이 그대로다 — 엔드포인트가 요청을 받고, 포화 경고는 남지 않는다.
  4) 엔드포인트가 302를 주면 `deliverWebhook`은 리다이렉트를 따라가지 않고(두 번째 URL로 요청이 오지 않음) `(302, err)`를 반환하며 `webhookTest`가 `ok:false` + `status_code:302`를 낸다.
  5) 테스트가 실제로 그 분기를 지난다는 증명: 새 경고를 지우면 1)이 실패하고, `outbound_client.go:31-33`의 `http.ErrUseLastResponse`를 `nil`로 되돌리면 4)가 실패하는 것을 확인해 노트에 적는다. (소스 문자열 검사(grep)는 증거로 쓰지 말 것.)
- 건드릴 파일:
  - `internal/httpapi/webhook.go:87-89` — `default` 분기의 `_ =`를 `if updateErr := …; updateErr != nil { s.logger.Warn(…) }`로. webhook.go:191-194와 같은 문구·같은 필드 구성을 쓸 것. **`queueWebhook`의 반환(void)·`completeWebhookDelivery`에 넘기는 statusCode 0·오류 문구는 바꾸지 말 것** — 관리 화면이 읽는 delivery 레코드의 계약이다.
  - `internal/httpapi/webhook_test.go` — 테스트 추가. 기존 하네스 `webhookServer(t, endpointURL, completeErr)`(webhook_test.go:31-55)를 그대로 쓴다. 이미 실제 `newOutboundHTTPClient` + 실제 `deliverWebhook`으로 돌고 `webhookCompleter` seam이 `deliveryErr`를 `*[]error`에 모아 준다.
  - 새 seam은 만들지 말 것. 지난 회차가 이미 세 개(`webhookConfigLoader`/`webhookCreator`/`webhookCompleter`)를 추가했고, 이 과제에는 더 필요 없다.
- 구현자가 알아 둘 배선 사실 (확인함):
  - `quietServer()`는 `auth_outage_test.go:16-18`에서 `&Server{logger: …}`만 채운다 → **`webhookSlots`가 nil이다.** nil 채널로의 send는 영원히 블록하므로 `select`는 항상 `default`를 탄다. 즉 아무 것도 안 하면 테스트는 자동으로 포화 분기를 탄다. 수용 기준 3)의 "자리가 있을 때"를 보려면 `server.webhookSlots = make(chan struct{}, 16)`(운영값, server.go:60과 동일)을 직접 넣고, 2)의 포화를 보려면 `make(chan struct{}, 1)`에 미리 하나 넣어 채워라.
  - 자리가 있는 경로는 `go func()`로 비동기 전송한다 — 엔드포인트 핸들러에서 채널로 신호를 받아 기다려라. `time.Sleep`으로 때우지 말 것.
  - `queueWebhook`은 `*http.Request`를 받고 `sessionFrom(r)`의 실패를 무시한다(`session, _ :=`). `httptest.NewRequest`만으로 세션 없이 호출된다. 이벤트가 걸러지지 않게 `webhookConfigLoader`의 `Events` 맵에 쓰려는 eventType을 넣어라(현재 하네스는 `integration.test`만 켜 있다).
  - 로그를 검사하려면 `webhookServer`가 돌려준 `server`의 `server.logger`를 `slog.New(slog.NewJSONHandler(&buf, nil))`로 바꿔 끼우면 된다.
  - 3xx가 실패로 세는 근거는 `internal/httpapi/outbound_client.go:31-33`의 `CheckRedirect → http.ErrUseLastResponse`(리다이렉트 미추종)와 `webhook.go:186-188`의 `< 200 || >= 300` 판정이다. 즉 **현재 동작은 이미 맞다** — 4)는 버그 수정이 아니라 계약 고정이다. 브리프에서 이것을 버그라고 주장하지 말 것.
- 검증 명령:
  - `go test ./internal/httpapi/ -run 'Webhook' -count=1 -v`
  - `go test ./internal/httpapi/ -run 'Webhook' -count=20`  (비동기 전송 경로가 있으니 반복으로 flake 확인)
  - `go test ./... -count=1` · `go vet ./...` · `gofmt -l .`
  - `./scripts/verify.sh` (전체. `web/node_modules`가 없어 `npm ci`부터 도므로 시간이 걸린다)
- 위험과 피할 것:
  - `-race`로 한 번 돌려 볼 것(`go test ./internal/httpapi/ -run Webhook -race -count=5`). 비동기 전송 goroutine과 테스트가 같은 `*[]error`를 만지므로 `webhookCompleter` 안에서 슬라이스에 쓰는 것을 뮤텍스나 채널로 보호해야 한다 — 기존 4개 테스트는 전부 동기 경로라 이 문제가 없었다.
  - `migrations/`, `internal/httpapi/auth_handlers.go`, `oidc*.go`, `mcp_oauth.go`, `.github/workflows/`는 건드리지 말 것.
  - `retryWebhookDelivery`(webhook.go:140-162)는 이번 범위 밖이다. 저것은 `s.store.WebhookDelivery`를 직접 불러 seam이 하나 더 필요하고, seam 추가는 이번에 금지다.
  - `CHANGELOG.md`·`scripts/version.sh`는 릴리즈 커밋에서만 갱신하는 관례다. 작업 커밋에서 손대지 말 것.
  - 커밋 메시지는 영어 conventional(`fix:`), 코드 주석·로그 문구는 한국어 — 이 저장소 관례.
  - 로그 한 줄 추가가 전부인 변경으로 보이지 않게, 수용 기준 2)~4)의 테스트를 반드시 같이 낼 것. 다만 "테스트만 늘렸다"로 끝내지도 말 것 — 1)의 관찰 가능한 변화가 이 과제의 본체다.
- 차선 후보: `aiRequestLimiter`의 사용자 표에 실질적 상한이 없는 문제 정리 (`internal/httpapi/ai_rate_limit.go:45-51` — `len(l.users) > 4096`에서 `active == 0 && windowStart 2분 초과`인 항목만 지우므로 최근 활동 사용자가 4096을 넘으면 한 건도 축출되지 않고 표가 계속 자란다. 2026-09-20 `login_rate_limit.go`가 확립한 방식 — 상수 + `capacity` 필드 + 분리된 결정적 `evict` — 를 옮기고, 실제 `newAIRequestLimiter()`로 상한을 낮춘 테스트로 증명한다. 키가 인증된 사용자 ID라 외부에서 늘릴 수 없어 가치는 2.)
