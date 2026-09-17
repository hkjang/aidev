# PR #16 fix summary (commit 195827a)

- **재현**: 172.19.24.103에 STARTTLS 없이 `AUTH LOGIN`만 광고하는 가짜 릴레이로 Deliver를 돌리니 비밀번호가 base64 평문으로 도착했고, `AUTH PLAIN LOGIN`이면 PlainAuth가 `unencrypted connection`으로 실패해 EHLO→QUIT만 남았다 — 심사 지적 그대로. `loginAuth.Start`의 `server.Name != a.host`는 둘 다 `config.Host`라 항상 거짓이었다.
- **정책**: 자격증명은 TLS 위이거나 호스트가 `localhost`/`127.0.0.1`/`::1`(표준 PlainAuth와 동일)일 때만 보낸다. `startSession`이 `TLSConnectionState`로 먼저 걸러 PLAIN·LOGIN이 같은 `ErrInvalid` 오류를 내고, `Validate`는 `security=none`+username(비-루프백)을 저장 시점에 거부한다. ADMIN_GUIDE.md 표·오류표에 기록(PDF는 재생성하지 않음).
- **예산/기록**: `deliveryBudget = 시도수×2×Timeout + 대기`(dial Timeout + 세션 deadline), `SendNow`도 `2×Timeout+5s`; `complete()`는 발송 컨텍스트 대신 독립 컨텍스트(10s)로 `FinishMailDelivery`를 써 행이 queued로 남지 않는다.
- **동시성**: `Service.slots`(버퍼 4 채널 세마포어)로 동시 발송을 묶고, 넘치는 수신자는 Notify에서 기록만 먼저 남기고 자리가 나면 보낸다.
- **테스트 추가**(mail_test.go): 비-루프백 평문 AUTH LOGIN/PLAIN LOGIN 거부(전송 로그에 AUTH·base64 비밀번호 없음), 루프백 LOGIN/PLAIN 인증 성공, Validate 정책, greeting 없는 릴레이 → Attempts=2·failed, 예산 소진 뒤 기록 성공(fakeStore가 ctx.Err를 존중), 동시 발송 상한. `go vet ./...`, `go test -race ./...`, integration 단계 모두 통과.
