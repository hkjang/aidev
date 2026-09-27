- 과제: webhook 전송이 엔드포인트에 닿지도 못했을 때의 기록 실패가 아무 흔적도 남기지 않는 마지막 분기 막기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `deliverWebhook`의 성공·비2xx 경로(webhook.go:197-200)와 대기열 포화 경로(webhook.go:90-94)는 `completeWebhookDelivery` 실패를 `logger.Warn`으로 남기는데, 요청 생성 실패(webhook.go:175)와 전송 자체 실패(webhook.go:187, connection refused·DNS·TLS·타임아웃 — 실제로 가장 흔한 실패)는 아직 `_ =`로 반환값을 버린다. 그래서 엔드포인트가 죽은 동시에 저장소가 흔들리면 delivery 행이 `pending`으로 영원히 남고 로그에도 단서가 없어, 관리자는 목록에서 "전송도 안 됐고 실패로도 안 찍힌" 이벤트를 원인 없이 본다(v0.2.21·v0.2.22가 같은 계열의 다른 두 분기를 고쳤고 이 두 자리만 남았다).
- 수용 기준:
  1) 엔드포인트가 닿지 않고(닫힌 포트) `completeWebhookDelivery`도 오류를 낼 때 `logger.Warn` 한 줄이 정확히 남는다 — 기존과 같은 문구 `"webhook delivery 기록 실패"`, 같은 필드(`error`·`delivery_id`·`event`·`request_id`·`status_code`), `status_code`는 0.
  2) 같은 상황에서 `deliverWebhook`의 반환은 지금과 같다 — `(0, 전송 오류)`. 저장소 오류가 전송 오류 자리를 덮지 않는다(`errors.Is(deliveryErr, driverFailure)`가 false).
  3) 로그에 payload·서명 키·이벤트 data 원문이 없다(기존 `TestQueueWebhookLogsWhenSaturatedRecordingFails`의 검사와 같은 방식).
  4) 기록이 성공하는 기존 경로는 경고가 0줄이다 — `TestDeliverWebhookReportsUnreachableEndpoint`(webhook_test.go:231)는 수정 없이 통과.
  5) 테스트가 증명할 것: 수정 전에는 1)이 실패(경고 0줄)하고, 고친 뒤 통과하며, 고친 줄을 다시 `_ =`로 되돌리면 같은 테스트가 다시 실패한다.
- 건드릴 파일:
  - `internal/httpapi/webhook.go` — 175·187의 `_ = s.completeWebhookDelivery(...)`를 197-200과 같은 `if updateErr := …; updateErr != nil { s.logger.Warn(…) }` 형태로 바꾼다. 세 자리가 같은 모양이 되므로 작은 헬퍼(예: `func (s *Server) recordWebhookOutcome(delivery store.WebhookDelivery, statusCode int, deliveryErr error)`)로 모아 세 곳에서 부르는 편이 낫다. **헬퍼로 모을 때 주의**: 197-200은 지금 `delivery.RequestID`를 쓰고 포화 분기(90-94)는 `requestIDFrom(r)`을 쓴다 — 같은 값이지만(생성 시 넣은 값) 포화 분기까지 헬퍼로 바꿀 필요는 없다. 바꿔도 값은 같다 — `store.CreateWebhookDelivery`(internal/store/webhooks.go:26-42)의 `RETURNING`에 `request_id`가 있고 `item.RequestID`로 스캔하므로 프로덕션에서도 채워진다(확인함). 그래도 이번 범위는 175·187이고, 포화 분기는 통과하는 테스트가 문구·필드를 못박고 있으니 손대지 않는 것이 안전하다.
  - `internal/httpapi/webhook_test.go` — `webhookServer(t, closedURL, driverFailure)` + `captureWebhookLog(server)`로 "닿지 않는 엔드포인트 + 기록 실패" 테스트 1개 추가. `TestDeliverWebhookReportsUnreachableEndpoint`(231-254)를 그대로 본떠 URL을 닫고, `webhookWarnings(t, logs)`로 한 줄·필드·원문 없음을 확인한다. 기존 테스트는 수정하지 말고 새 함수로 추가한다.
  - 프로덕션 파일은 1개다. 그 이상으로 번지면 과제를 잘못 잡은 것이다.
- 검증 명령:
  - `go test ./internal/httpapi/ -run Webhook -count=1 -v`
  - `go test ./internal/httpapi/ -run Webhook -race -count=5`
  - `go test ./... -count=1` · `go vet ./...` · `gofmt -l .`
  - 마지막에 `./scripts/verify.sh` (npm ci가 포함되어 몇 분 걸린다; `web/node_modules`가 없다)
- 위험과 피할 것:
  - `deliverWebhook`의 **반환값·상태코드·`webhookTest`의 응답 문구를 바꾸지 말 것.** 이번 변경으로 관찰 가능하게 달라지는 것은 로그 한 줄뿐이다. `outbound_client.go`(리다이렉트 정책)·`webhookSlots` 용량·`store/webhooks.go`는 건드리지 않는다.
  - 새 seam을 추가하지 말 것 — `webhookConfigLoader`·`webhookCreator`·`webhookCompleter` 세 개로 충분하다(이미 세 개다).
  - 로그 문구·필드 이름을 새로 만들지 말 것. 191-199·92-93과 **같은** 문구·필드를 쓸 것(운영자가 grep 하는 값이다).
  - 보호 경로(`auth_handlers.go`·`oidc*.go`·`mcp_oauth.go`·`store/users.go`·`migrations/`·`.github/workflows/`)는 건드리지 않는다.
  - `CHANGELOG.md`·`scripts/version.sh`는 작업 커밋에서 건드리지 않는다(릴리즈 커밋의 몫).
  - `scripts/e2e-docker.sh`·Playwright는 `docs/screenshots/*.png`를 덮어쓴다 — 돌리지 말 것.
  - Bash 도구가 `&&` 복합 명령을 승인 요구로 막는 경우가 있다. 명령을 하나씩 나눠 실행할 것.
- 차선 후보: `aiRequestLimiter`의 사용자 표에 결정적 상한 세우기 (2/1/S) — `internal/httpapi/ai_rate_limit.go:45-51`이 `len(l.users)>4096`일 때만, 그것도 `active==0 && windowStart` 2분 초과 항목만 지워 최근 활동 사용자가 많으면 한 건도 지워지지 않는다. 2026-09-20 `login_rate_limit.go`가 쓴 방식(상수 + capacity 필드 + 분리된 결정적 evict + 흘림 테스트)을 옮기면 되고 `ai_rate_limit_test.go`가 하네스를 준다. 키가 인증 사용자 ID라 외부에서 임의로 늘릴 수 없어 가치는 낮다.
