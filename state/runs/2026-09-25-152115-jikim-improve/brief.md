- 과제: Webhook 전송 결과 기록 실패(저장소 장애)를 "엔드포인트가 거절했다"로 보고하는 문제 분리 (가치 3 / 위험 2 / 작업량 M)

- 왜: `internal/httpapi/webhook.go:165-167`의 `deliverWebhook`은 상대 엔드포인트가 200을 돌려줘도 `s.store.CompleteWebhookDelivery`가 실패하면 그 저장소 오류를 delivery 오류 자리에 넣어 반환한다. 그러면 `webhookTest`(webhook.go:94-102)는 `ok:false` + "Webhook endpoint가 요청을 수락하지 않았습니다"를, `retryWebhookDelivery`(webhook.go:131-137)는 `ok:false`를 내보내, 관리자는 자기 DB 장애를 고객 엔드포인트 장애로 오진하고 멀쩡한 URL을 계속 고치게 된다. 이 저장소가 이미 명시한 교훈("저장소 장애를 인증/권한 실패로 접지 말고 구분할 것", `auth_outage_test.go` 주석)이 webhook 경로에만 적용되지 않은 자리다.

- 수용 기준:
  1) 엔드포인트가 200을 주고 `CompleteWebhookDelivery`만 실패하면 `deliverWebhook`은 `(200, nil)`을 돌려주고, `webhookTest` 응답은 `ok:true` + `status_code:200` + "서명된 테스트 이벤트를 전송했습니다"가 된다. 기록 실패는 `s.logger.Warn`으로만 남는다(요청 ID·delivery ID 포함, 드라이버 원문은 클라이언트 응답에 넣지 않는다).
  2) 엔드포인트가 500을 주고 `CompleteWebhookDelivery`도 실패하면 반환 오류는 여전히 **전송 오류**("webhook 응답 상태 500")이고 `ok:false`·`status_code:500`이다 — 저장소 오류가 전송 오류를 덮어쓰지 않는다.
  3) 엔드포인트 연결 자체가 실패한 기존 경로(webhook.go:155-159)의 동작은 그대로다: `(0, 전송오류)`.
  4) 테스트가 증명할 것: 위 세 경우를 **실제 `httptest.NewServer` 엔드포인트 + 실제 `newOutboundHTTPClient` + 실제 `deliverWebhook`/`webhookTest` 핸들러**로 왕복해 확인한다. 수정 전에 1)이 실패(`ok:false`)하는 것을 먼저 보고 나서 고칠 것.

- 건드릴 파일:
  - `internal/httpapi/server.go:19-43` (`Server` 구조체) — 이 저장소가 이미 쓰는 seam 패턴(`auditRecorder`, `storagePinger`, `securityLoader`)과 똑같이 `webhookCompleter func(context.Context, string, int, error) error` 필드 하나 추가. `New()`에서는 배선하지 말 것 — 기존 seam들처럼 nil이면 `s.store`로 떨어지는 방식이다.
  - `internal/httpapi/webhook.go` — `CompleteWebhookDelivery` 호출 세 군데(145, 157, 165)를 `s.completeWebhookDelivery(...)` 같은 작은 메서드로 모으고, 그 메서드가 `if s.webhookCompleter != nil { … }` 아니면 `s.store.CompleteWebhookDelivery(...)`를 부르게 한다(`server.go:406`의 `storagePinger` 분기와 같은 모양).
  - `internal/httpapi/webhook.go:165-168` (`deliverWebhook`) — 핵심 수정. `updateErr`를 반환값에 섞지 말고 로그로만 남기고, 항상 `(response.StatusCode, err)`를 돌려준다.
  - `internal/httpapi/webhook_test.go` — 현재 13줄(서명 테스트 하나)뿐이다. 위 3~4개 케이스를 여기에 추가.

- 검증 명령:
  - `go test ./internal/httpapi/ -run Webhook -count=1 -v`
  - `go test ./... && go vet ./... && gofmt -l .`
  - 마지막에 `./scripts/verify.sh` (npm ci 포함이라 시간이 걸린다. `web/node_modules`가 작업 트리에 없다)

- 위험과 피할 것:
  - `webhookSignature`(webhook.go:171-177)의 서명 계산과 `X-Jikim-*` 헤더 이름·값은 외부 계약이다. 건드리지 말 것. 기존 서명 테스트의 고정 해시가 그대로 통과해야 한다.
  - `newOutboundHTTPClient`의 `CheckRedirect`(리다이렉트 미추종, `outbound_client.go:31-33`)는 서명이 다른 호스트로 새지 않게 하는 방어다. 손대지 말 것.
  - `queueWebhook`의 `webhookSlots` 대기열·`context.Background()` 사용(webhook.go:53-65, 145, 157, 165)은 의도된 것이다 — 요청 취소로 기록이 사라지지 않게 하려는 것이니 `r.Context()`로 바꾸지 말 것.
  - `New()`에서 새 seam을 배선하지 말 것. 이 저장소의 다른 seam 15개는 전부 nil 기본값 + `s.store` 폴백이고, `New`에 넣으면 그 관례가 깨진다.
  - 보호 경로(`auth_handlers.go`, `oidc*.go`, `mcp_oauth.go`, `store/users.go`, `migrations/`, `.github/workflows/`)는 이번 과제와 무관하다. 열지 말 것.
  - CHANGELOG.md는 릴리즈 커밋에서만 갱신하는 관례다. 작업 커밋에서 건드리지 말 것.
  - 로그·감사로 넘기는 값에 서명 키(`cfg.SigningSecret`)나 payload 원문을 넣지 말 것. delivery ID와 event 타입만 쓸 것.

- 차선 후보: aiRequestLimiter의 사용자 표에 실질적인 상한이 없는 문제 정리 (가치 2 / 위험 1 / 작업량 S) — `internal/httpapi/ai_rate_limit.go:45-51`의 축출은 `len(l.users) > 4096`일 때만 돌면서 `active == 0 && windowStart가 2분 초과`인 항목만 지우므로, 최근 활동 사용자가 4096명을 넘으면 아무것도 지워지지 않는다. 2026-09-20에 `loginRateLimiter`에 적용해 채택된 방식(상한을 상수+필드로 빼고 `evict`를 분리해 ① 만료 ② 유휴 ③ `windowStart` 오름차순으로 결정적으로 지우기)을 그대로 옮기면 된다. 단, 키가 인증된 사용자 ID라 공격자가 늘릴 수 없으니 가치는 로그인 쪽보다 낮고, "관찰 가능한 변화가 없다"는 반려를 피하려면 `newAIRequestLimiter()` 실제 생성자 + 낮춘 상한으로 표 크기와 `active > 0` 항목 생존을 단언하는 테스트가 반드시 있어야 한다.
