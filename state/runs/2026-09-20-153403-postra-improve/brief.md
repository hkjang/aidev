# 과제서 — 2026-09-20-153403-postra-improve (base main@0356555, v0.23.1)

- 과제: SMTP `Send`/`TestConnection` 에 명령 시간제한(`CommandTimeoutSec`)과 ctx 취소를 dial 이후 단계(EHLO·STARTTLS·AUTH·MAIL·RCPT·DATA·QUIT)까지 전파 (가치 3 / 위험 2 / 작업량 S~M)
- 왜: `internal/adapters/smtp/client.go` 는 `ctx` 와 `ConnectTimeoutSec` 를 **dial 에만** 쓰고(`connect`, 41~48행), 그 뒤 `net/smtp` 왕복에는 데드라인이 하나도 없다 — 릴레이가 연결은 받고 MAIL/RCPT/DATA 응답을 안 주면 `Send` 가 영원히 막힌다. 이 `Send` 를 부르는 아웃박스 재시도 워커 `ProcessRetries`(`internal/application/scheduler.go:90~117`) 는 due 항목 최대 50개를 **직렬로** 처리하고 `wctx := WithActor(ctx,"worker")` 에 타임아웃이 없으므로, 릴레이 한 대가 멈추면 리더 노드의 아웃박스 전체가 멈춘다(사용자 발송 경로 `send.go:364` 도 HTTP ctx 취소를 무시). IMAP·POP3 어댑터는 이미 `secondsOr(opts.CommandTimeoutSec, 60)` + `conn.SetDeadline` 로 같은 문제를 막고 있어(`pop3/client.go:55,108`) SMTP 만 빠져 있다.

## 수용 기준
1. 릴레이가 `MAIL FROM` 을 읽고 응답을 보내지 않으면(`hang`), `Client{}.Send(ctx, opts{CommandTimeoutSec:1,...})` 가 **약 1초 안에**(테스트에선 5초 상한) `*SendError` 로 돌아오고 `Temporary()==true` 다. 기록된 명령은 `EHLO`, `MAIL` 까지이고 `RCPT` 는 없다.
2. 릴레이가 같은 자리에서 멈춰 있을 때 `CommandTimeoutSec` 를 크게(예: 60) 두고 호출 뒤 `cancel()` 하면 `Send` 가 취소 직후(5초 상한) 돌아오고 오류가 임시(`Temporary()==true`)다. — ctx 취소가 dial 이후에도 통한다는 증거.
3. `DATA` 본문을 다 받은 뒤 최종 응답 없이 멈추는 경우(`hang after DATA`, 끊지 않고 침묵)는 데드라인 초과가 `isConnectionLost`(timeout → true) 로 잡혀 기존 `dropAfterData` 와 같이 `SendReceipt{Uncertain:true}, nil` 이다 — 페이로드 인계 뒤 자동 재시도 금지 규칙(SMTP-008/009) 유지.
4. `TestConnection` 도 같은 옵션을 받아 EHLO 뒤 침묵하는 서버에서 `CommandTimeoutSec:1` 로 수 초 안에 `connect_tls_auth` 단계 실패로 돌아온다(기존 `TestTestConnection` 의 단계 값 단언 방식 그대로).
5. 기존 `internal/adapters/smtp/client_test.go` 의 모든 테스트와 `internal/application/smtp_integration_test.go` 가 변경 없이 통과한다. 옵션을 안 주면(0) 기본 60초 — IMAP/POP3 와 동일.
6. `send.go:418` 과 `accounts.go:351` 의 `SMTPSendOptions{...}` 두 곳에 `CommandTimeoutSec: a.EffectiveConfig().Sync.CommandTimeoutSec` 가 들어간다(기존 `sync.command_timeout_sec` 설정 재사용, **새 설정 키 없음**). 두 호출처가 같은 값을 읽는지 grep 으로 확인(`grep -rn "SMTPSendOptions{" internal --include=*.go`).

## 건드릴 파일
- `internal/domain/account.go:160~169` `SMTPSendOptions` — `CommandTimeoutSec int` 필드 추가(`InboundDialOptions` 의 같은 이름·주석 관례).
- `internal/adapters/smtp/client.go`
  - `connect` — dial 직후 raw `net.Conn` 을 보관하고 `context.AfterFunc(ctx, func(){ conn.Close() })` 로 취소 시 연결을 닫음(Go 1.26 이므로 사용 가능; 반환된 stop 함수는 `Send`/`TestConnection` 종료 시 호출). 각 명령 전에 `conn.SetDeadline(time.Now().Add(secondsOr(opts.CommandTimeoutSec, 60)))` — `Hello`, `StartTLS`, `authenticate`, 그리고 `Send` 의 `Mail`/`Rcpt`/`Data`/`w.Close`/`Quit` 앞. 가장 단순한 형태는 pop3 처럼 `type session struct{ c *smtp.Client; conn net.Conn; commandTO time.Duration }` + `deadline()` 메서드. **주의**: StartTLS 뒤에도 데드라인은 raw conn 에 걸면 된다(`tls.Conn.SetDeadline` 은 하부 conn 에 위임하므로 raw conn 을 계속 써도 동작) — 테스트 `TestSendSTARTTLS` 가 그대로 통과해야 함.
  - `Send` — `context.AfterFunc` 로 닫힌 뒤 나는 `net.ErrClosed`/`use of closed network connection` 오류는 `classify` 가 non-textproto → temp=true 로 이미 분류함. ctx 취소는 임시 오류로 두되, 필요하면 `ctx.Err()` 를 wrap 해서 원인이 보이게(`fmt.Errorf("MAIL FROM: %w (canceled: %v)")` 정도, 과하게 만들지 말 것).
  - `isConnectionLost` 는 건드리지 않음(timeout 은 이미 lost 로 취급).
- `internal/adapters/smtp/client_test.go` — `relayOpts` 에 `hangAfter string` (예: `"MAIL"`, `"EHLO"`, `"DATA"` — 해당 명령을 읽은 뒤 응답 없이 `<-done` 대기, 테스트 종료 시 `t.Cleanup` 으로 해제) 옵션 추가; 위 수용 기준 1~4 를 각각 테스트. `fakeRelay` 의 기존 동작(옵션 미지정 시)은 불변.
- `internal/application/send.go:418`, `internal/application/accounts.go:351` — `CommandTimeoutSec` 배선.
- (선택) `docs/ADMIN_GUIDE.md` 의 `sync.command_timeout_sec` 설명에 "SMTP 발송 명령에도 적용" 한 줄 — 해당 절이 있는지 미확인, 없으면 생략.

## 검증 명령
```
gofmt -l ./cmd ./internal                      # 빈 출력이어야 함(CI 에 없음)
go build ./... && go vet ./...
go test -race -count=3 ./internal/adapters/smtp/           # 새 테스트 포함, 걸리는 시간 ≈ hang 테스트 합계 ~3s 이내
go test -race ./internal/application/ ./internal/domain/    # 배선·통합 테스트
go test -race ./...
go run ./cmd/postra-contracts -check           # SMTPSendOptions 는 계약 밖이라 변화 없어야 함(미확인 — 바뀌면 생성물 갱신)
```
변이 검증(권장): 구현 후 `deadline()` 호출을 `Mail` 앞에서만 지워 보고 수용 기준 1 테스트가 실패하는지, `context.AfterFunc` 를 지우면 기준 2 가 실패하는지 확인한 뒤 원복.

## 위험과 피할 것
- **가짜 대역으로 증명하지 말 것** — 운영자 규칙. 실제 `net.Listen` 릴레이가 침묵하는 것으로 증명하고, 실제 `Client{}.Send` 를 실제 `net/smtp` 로 통과시킨다(기존 client_test.go 픽스처 재사용).
- **같은 값을 읽는 경로 두 곳**(send.go / accounts.go) 모두 배선할 것 — 한쪽만 하면 시험 발송은 끊기고 실제 발송은 영원히 막히는 불일치가 생긴다.
- `Sync.CommandTimeoutSec` 기본 60초는 `config.go:281` — 기본값을 바꾸지 말 것(IMAP/POP3 와 공유).
- 데드라인 값이 총 연결 시간이 아니라 **명령당** 이어야 한다(큰 첨부 DATA 업로드가 60초를 넘을 수 있음). `io.Copy(w, message)` 는 청크마다 데드라인이 갱신되지 않으므로 DATA 쓰기 구간은 데드라인을 여유 있게(예: 쓰기 전 한 번 설정하고, 느린 업로드는 60초 안에 진행이 있으면 정상이라 가정 — 이 부분은 **추측**; 정 불안하면 `io.Copy` 를 32KiB 단위 루프로 돌리며 매 청크 전에 `deadline()` 갱신).
- `Password.Zero` defer, `AuthError` 영구 분류, `Uncertain` 규칙 등 기존 의미를 바꾸지 말 것 — 기존 테스트가 지킴.
- 보호 경로(auth/migrations/workflows) 는 건드리지 않음. 프런트 변경 없음 → `spa/assets` 재빌드 불필요.
- CI 의 gosec(medium+) 는 `conn.Close()` 반환값 무시(G104, low) 만 낼 수 있어 실패하지 않음; 푸시 전에 `go install github.com/securego/gosec/v2/cmd/gosec@v2.28.0 && gosec -severity medium -exclude-dir=scripts ./...` 로 확인.

## 차선 후보
- Makefile 에 CI 와 같은 플래그의 `lint`(gosec@v2.28.0, `-severity medium -exclude-dir=scripts ./...`) 타깃 + `gofmt -l` 검사 추가 (가치 3 / 위험 1 / S) — v0.23.0 이 gosec 을 깨뜨린 채 main 에 들어간 재발 방지. 코드 변경 없이 Makefile 만이라 1순위가 성립하지 않을 때(예: Go 1.26 의 `context.AfterFunc` 로 `net/smtp` 내부 conn 이 예상과 다르게 동작해 45분 안에 못 끝날 때) 고를 것.
