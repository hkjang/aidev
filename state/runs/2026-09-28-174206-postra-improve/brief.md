- 과제: STLS/STARTTLS 핸드셰이크 직전에 서버가 미리 보낸 평문 바이트를 감지해 업그레이드를 거부 (가치 3 / 위험 2 / 작업량 M)

- 왜: POP3 `Dial`(client.go:92-104)과 IMAP `Dial`(client.go:78-90)은 `+OK`/`OK STARTTLS` 응답을 읽은 뒤 곧바로 `tls.Client(conn,...)` 로 핸드셰이크하고, 업그레이드 후에 **리더를 새로 만들어**(pop3 `textproto.NewConn(tconn)` :103, imap `bufio.NewReader(tconn)` :89) 이전 버퍼를 통째로 버린다. 그래서 중간자가 STLS 응답 뒤에 평문 명령·응답을 주입해도 클라이언트는 **아무 오류 없이 그대로 접속을 계속한다** — 주입은 조용히 삼켜지고 서버가 규약을 어겼다는 사실이 어디에도 남지 않는다(STARTTLS command injection 계열, CVE-2011-0411 / 2021 "NO STARTTLS" 보고). TLS 는 클라이언트가 먼저 말하는 프로토콜이라 정상 서버는 핸드셰이크 전에 한 바이트도 보내지 않으므로, 남은 버퍼는 곧 규약 위반이고 거부가 안전한 판정이다. 고치면 두 인바운드 어댑터가 주입 시도를 삼키는 대신 명시적으로 실패한다.

- 수용 기준:
  1) STLS `+OK` 뒤에 여분의 평문 줄을 덧붙여 보내고 나서 TLS 핸드셰이크를 하는 스크립트 서버에 대해 `pop3.Dialer{}.Dial` 이 **오류를 반환**하고(수정 전에는 성공했다는 것을 먼저 재현해 로그로 남길 것), 반환된 연결이 닫혀 있을 것.
  2) IMAP `STARTTLS` 에서도 같은 서버 모양에 대해 `imap.Dialer{}.Dial` 이 오류를 반환할 것. 한쪽만 고치지 말 것 — 두 인바운드 어댑터가 같은 입력에 같은 판정을 내는 것이 이 과제의 핵심이다.
  3) 정상 STLS/STARTTLS 경로는 **깨지지 않을 것**: 여분 바이트 없이 업그레이드하는 서버에 대해 두 어댑터 모두 Dial 이 성공하고, 업그레이드 후 명령(POP3 `UIDL`/`LIST`, IMAP 열거)이 TLS 위에서 정상 응답을 돌려주는 것을 같은 테스트에서 단언할 것. 이 긍정 테스트가 없으면 오탐을 증명할 수 없다.
  4) 변이 검증: 새 검사 줄을 지우면 주입 테스트 2개만 실패하고 정상 경로 테스트는 통과함을 확인한 뒤 원복할 것.

- 건드릴 파일 (프로덕션 2개):
  - `internal/adapters/pop3/client.go` — `Dial` 의 STLS 블록. `s.cmd("STLS")` 성공(:93-96) 과 `tls.Client`(:97) **사이**에, 이미 버퍼에 남은 평문이 있으면 `conn.Close()` 후 오류를 반환. 버퍼는 `s.text.R.Buffered()` 로 본다(`textproto.Conn` 이 `textproto.Reader` 를 임베드하고 그 `R *bufio.Reader` 가 exported — **컴파일로 확인할 것**, 이 세션에서는 `go doc` 이 샌드박스에 막혀 소스로 확인하지 못했다). 접근이 막히면 대안: Dial 에서 `bufio.NewReader(conn)` 를 직접 들고 `textproto.NewReader` 에 넘겨 핸들을 보관한다.
  - `internal/adapters/imap/client.go` — `Dial` 의 STARTTLS 블록. `s.exec("STARTTLS")` 성공(:79-82) 과 `tls.Client`(:83) 사이에 `s.r.Buffered() > 0` 검사. `s.r` 는 :70 에서 만든 `*bufio.Reader` 라 그대로 쓸 수 있다.
  - 두 곳 모두 **근거 주석**을 달 것(이 저장소 관례 — maxListBytes/maxLineBytes 선례). 주석에 "TLS 는 클라이언트 우선이므로 핸드셰이크 전 서버 바이트는 규약 위반" 과 "버퍼에 도달한 바이트만 보이므로 best-effort 탐지(커널에 남아 있는 in-flight 바이트는 못 본다)" 두 가지를 적을 것. 과장된 보증을 주석·릴리즈 노트에 쓰지 말 것.
  - `internal/adapters/pop3/client_test.go` — 헬퍼 추가. 기존 `dial`(:20)은 `Security: domain.SecurityNone` 로 고정돼 있으니 `dialStartTLS`(`Security: domain.SecurityStartTLS, InsecureSkipVerify: true`)를 따로 만들 것. 서버 픽스처는 `maildropServer`(:41) 의 accept 루프 모양을 재사용하되 STLS 명령에 `+OK`(+선택적으로 주입 줄) 를 쓰고 `tls.Server(conn, tlsCfg)` 로 감싼 뒤 같은 루프를 계속 돈다.
  - `internal/adapters/imap/client_test.go` — 같은 모양의 STARTTLS 픽스처. (이 파일의 기존 헬퍼 이름 `dial`/`dialMax`/`fakeServer` 는 프로필에서 옮겨 적은 것으로 이번 세션에 직접 열지 않았다 — **미확인**, 열어 보고 맞출 것.)
  - TLS 설정은 새로 쓰지 말고 `internal/adapters/smtp/client_test.go:85 selfSigned(t)` 를 그대로 이식할 것(rsa 키 생성 + 자가서명, :104 에서 `MinVersion: tls.VersionTLS12`). 서버 쪽 업그레이드는 같은 파일 :167 `tls.Server(conn, tlsCfg)` 가 선례다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -race -count=3 ./internal/adapters/pop3/ ./internal/adapters/imap/`  ← 주 검증. 이번 세션에 `go test -race -count=1 ./internal/adapters/pop3/ ./internal/adapters/imap/ ./internal/adapters/smtp/` 를 실제로 돌려 pop3 ok 3.144s / imap ok 4.471s / smtp ok 1.098s 로 전부 통과함을 확인했다(기준선). 그 외 전체 `./...`·프런트·외부 PostgreSQL·브라우저·gosec 은 이번 세션에 실행하지 않았다.
  - `go build ./... && go vet ./...`
  - `go test -race -count=1 ./...` (전체 수 분, application 패키지가 가장 김)
  - `make lint` (gofmt -l + gosec v2.28.0 -severity medium, Issues 0 이어야 함)
  - `go run ./cmd/postra-contracts -check`
  - `git diff --check`, `git status --porcelain internal/transport/spa/assets` 가 비어 있을 것(프런트 미변경)

- 위험과 피할 것:
  - **`retrBody`(pop3/client.go:230)에 손대지 말 것.** POP3 본문 상한은 미병합 브랜치 `auto/2026-09-21-1520`(04b15be) / PR #22 의 소유이고, 사람 반려 여부를 이번에도 확인하지 못했다(샌드박스가 `gh` 를 거부 — 3회 연속 미확인). 같은 접근을 재제출하지 말 것. 이번 과제는 `Dial` 의 STLS 블록만 건드리므로 그 브랜치와 겹치는 줄이 거의 없다(단 04b15be 가 `s := &session{...}` 부근을 고쳤다면 인접 충돌은 가능 — 리베이스 시 확인).
  - `maxListBytes`/`listLineOverhead`/`readList`, IMAP 의 `maxLineBytes`/`maxResponseBytes`/`maxResponseLiterals`/`literalMargin`/`drainChunk`, `readLine` 과 `readLineNoReset` 의 데드라인 정책 차이(IDLE 28분 창) — 전부 손대지 말 것.
  - **SMTP 는 범위 밖.** `internal/adapters/smtp/client.go:67` 은 `net/smtp` 의 `c.StartTLS` 를 쓰므로 리더 버퍼에 접근할 수 없다. 같은 검사를 넣으려면 stdlib 대체가 필요해 이번 회차 크기를 넘는다 — 릴리즈 노트에 "SMTP 도 고쳤다" 고 쓰지 말 것.
  - 오탐이 곧 메일 수집 중단이다. 정상 경로 테스트(수용 기준 3)를 반드시 같이 낼 것. `InsecureSkipVerify` 는 테스트 픽스처에서만 켜고 프로덕션 기본값을 건드리지 말 것.
  - 보호 경로 — `internal/application/oidc*.go`, `httpapi/browser_auth.go`, auth/session, mcpserver OAuth/DCR, DB migrations, SecretStore/KEK, `.github/workflows`, `internal/transport/spa/assets` 는 이번 과제와 무관하니 열지 말 것.
  - 손으로 만든 대역(fake 세션)으로 증명하지 말 것 — `internal/application` 의 POP3/IMAP 테스트 일부는 fake 라 어댑터 배선 결함을 못 본다. 반드시 `net.Listen("tcp","127.0.0.1:0")` + 실제 `Dialer` 로 증명할 것.
  - grep 결과를 증거로 제출하지 말 것. 수정 전 "Dial 이 성공했다" 는 실제 테스트 실행 출력으로 남길 것.

- 차선 후보: POP3 어댑터 프로토콜 테스트 확대 (가치 2 / 위험 1 / 작업량 S) — 프로덕션 변경 0. 공백 4개: (1) USER/PASS 거부 시 `Dial` 이 `*domain.AuthError` 를 반환하고 `errors.As` 로 잡히는가(POP-011 의 "잘못된 자격증명에 무한 재시도 금지" 가 이 타입에 달려 있다. `app.go` 의 `dialInbound` 는 오류를 감싸지 않고 그대로 반환하는 것을 이번에 확인했다), (2) 인사말이 `-ERR` 인 경로, (3) STLS 업그레이드 성공 경로, (4) `retrBody` 의 dot-unstuffing — `..foo` → `.foo`, 단독 `.` 은 종료(테스트만 추가, `retrBody` 코드는 건드리지 말 것). 1순위의 TLS 픽스처 이식이 막히면 이쪽으로 전환할 것.
