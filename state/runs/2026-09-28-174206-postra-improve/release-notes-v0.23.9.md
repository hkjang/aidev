# Postra v0.23.9 — TLS 핸드셰이크 전에 서버가 끼워 넣은 평문을 조용히 버리던 문제 수정

POP3 `STLS`·IMAP `STARTTLS` 업그레이드를 승인하는 응답 **뒤**, TLS 핸드셰이크가 시작되기 **전**에 서버가 밀어 넣은 평문을 감지해 접속을 거부하는 패치 릴리즈입니다. 예전에는 그 바이트가 교체되는 리더와 함께 조용히 버려져, 서버가 규약을 어겼다는 사실이 로그에도 오류에도 남지 않았습니다. 새 설정 키, DB 마이그레이션, 새 필수 환경변수는 없습니다.

## 주입이 삼켜지고 아무 흔적도 남지 않던 문제

- TLS 는 클라이언트가 먼저 말하는 규약입니다. 업그레이드를 승인하는 응답(POP3 `+OK begin TLS negotiation`, IMAP 태그 완료 `OK`)을 보낸 뒤 규약을 지키는 서버는 ClientHello 를 보기 전까지 아무것도 보내지 않습니다. 그러므로 그 응답 뒤에 이미 도착해 있는 바이트는 곧 규약 위반이며, 평문 구간에 명령을 끼워 넣어 암호화된 세션의 응답으로 오해하게 만드는 STARTTLS command injection(CVE-2011-0411, 2021년 "NO STARTTLS" 조사)의 모양 그대로입니다.
- 두 인바운드 어댑터는 승인 응답을 읽은 직후 리더를 새로 만들었습니다 — POP3 는 `textproto.NewConn(tconn)`, IMAP 은 `bufio.NewReader(tconn)`(`internal/adapters/pop3/client.go`, `internal/adapters/imap/client.go`). 주입된 평문은 버려지는 옛 버퍼 안에 들어 있었으므로 **그대로 사라졌고**, 접속은 성공한 채 동기화가 계속됐습니다. RFC 가 요구하는 "핸드셰이크 전 버퍼를 버려라"는 만족했지만, 위반을 **알리지는** 않았기 때문에 그런 서버에 붙었다는 사실을 운영자가 알 방법이 없었습니다.
- 계정의 접속 호스트는 계정 소유자가 고르는 값이므로 이 입력은 신뢰할 수 없습니다.

- 이제 두 `Dial` 이 승인 응답과 `tls.Client` **사이**에서 리더에 남은 바이트 수를 확인합니다(POP3 `s.text.R.Buffered()`, IMAP `s.r.Buffered()`). 0 이 아니면 `conn.Close()` 로 연결을 닫고 `STLS: server sent %d bytes before TLS handshake` / `STARTTLS: server sent %d bytes before TLS handshake` 로 접속 자체를 실패시킵니다. 정상 서버는 이 시점에 보낸 것이 없으므로 아무 대가도 치르지 않습니다.
- **탐지는 구조상 best-effort 입니다.** 버퍼에 이미 도달한 바이트만 보이므로, 커널 수신 큐나 전선에 아직 떠 있는 주입은 보이지 않습니다. 이 검사는 창을 좁히는 것이지 닫는 것이 아니며, 릴리즈 노트와 코드 주석 모두 그 이상을 보증하지 않습니다.
- **SMTP 발송 경로는 이번 검사에 포함되지 않았습니다.** 표준 라이브러리 `net/smtp` 의 `c.StartTLS` 를 쓰고 있어 같은 지점의 리더 핸들에 접근할 수 없습니다. 범위 밖으로 남겨 두었고, 이번 릴리즈가 SMTP 를 고쳤다는 뜻이 아닙니다.
- 프로토콜 프레이밍 상수, 데드라인 정책, `retrBody`·`readList`·`maxListBytes`·IMAP 응답 상한, 설정 키는 하나도 손대지 않았습니다.

## 검증

- 손으로 만든 대역 없이 `net.Listen("tcp", "127.0.0.1:0")` 스크립트 서버를 실제 `Dialer` 에 물려 양쪽 어댑터에 같은 서버 모양으로 4건을 추가했습니다(`TestPOP3STLSInjectedPlaintextRefusesUpgrade`, `TestPOP3STLSUpgradeRunsCommandsOverTLS`, `TestIMAPStartTLSInjectedPlaintextRefusesUpgrade`, `TestIMAPStartTLSUpgradeRunsCommandsOverTLS`). 픽스처 서버는 승인 응답과 주입 평문을 **한 번의 쓰기로** 내보내, 클라이언트가 읽고 있는 응답과 함께 도착하게 합니다.
- 주입 케이스는 접속 실패에 더해 **서버 쪽 연결 핸들러가 EOF 로 끝나는 것**을 채널로 단언합니다 — 즉 클라이언트가 평문 연결을 실제로 닫았는지까지 확인하며, 오류 문자열만 보고 넘어가지 않습니다.
- 오탐은 곧 메일 수집 중단이므로 정상 경로를 같은 비중으로 고정했습니다: 여분 바이트가 없는 업그레이드는 접속에 성공하고, 세션의 연결이 `*tls.Conn` 이며, POP3 `UIDL`·`LIST` 와 IMAP 열거가 TLS 위에서 값까지 일치합니다.
- 변이 검증: 두 검사의 조건을 뒤집으면 주입 테스트 2건만 실패하고 정상 경로 테스트와 나머지 전부는 통과함을 확인한 뒤 원복했습니다. 수정 전 코드에서의 실패 모습은 `Dial accepted a server that injected plaintext before the STLS handshake` / `... before the STARTTLS handshake` 이며, 같은 실행에서 정상 업그레이드 테스트 2건은 통과했으므로 픽스처가 아니라 프로덕션 경로의 결함임이 갈립니다.
- `go test -race -count=3 ./internal/adapters/pop3/ ./internal/adapters/imap/`(ok 7.699s / 11.604s), 전체 `go test -race -count=1 ./...`(실패 0), `go build ./...`, `go vet ./...`
- `make lint`: `gofmt` 빈 출력 + CI와 같은 플래그의 `gosec v2.28.0` Issues 0(Files 138)
- 계약 검사(`postra-contracts -check`, exit 0), `git diff --check`
- 프런트엔드 소스 변경이 없어 포함된 `/app` 번들은 v0.23.8과 동일합니다.

테스트는 로컬 루프백 서버와 임시 자체 서명 인증서만 사용하며 운영 메일 서버나 운영 데이터에 접근하지 않습니다. 실제 POP3·IMAP 서버에서의 동작과 외부 PostgreSQL, 32비트 빌드는 이번에 검증하지 않았습니다.

## 업그레이드 및 호환성 주의사항

DB 마이그레이션, 새 설정 키, 새 필수 환경변수는 없습니다. DB·SecretStore·ObjectStore·KEK를 백업하고 유지한 상태에서 실행 파일 또는 컨테이너 이미지를 교체한 뒤 재시작하세요.

- 규약을 지키는 서버에서는 동작이 전혀 달라지지 않습니다. 승인 응답 뒤 ClientHello 를 기다리는 서버는 이 검사에 걸릴 바이트를 애초에 보내지 않습니다.
- `STLS`/`STARTTLS` 승인 뒤에 무언가를 더 보내는 서버에 대해서만 동작이 바뀌며, 그 경우 접속이 실패하고 해당 계정의 그 회차 동기화가 실패로 집계된 뒤 다음 회차에서 다시 시도합니다. 오류 메시지에 남은 바이트 수가 함께 남으므로 서버 쪽 문제인지 판단할 근거가 됩니다. 이런 서버를 쓰고 있었다면 업그레이드 후 해당 계정이 실패로 바뀔 수 있으니, 접속 호스트 설정과 서버 구현을 먼저 확인하세요.
- 암호화가 처음부터 적용되는 접속(`SecurityTLS`, 즉 implicit TLS)과 평문 접속은 이 경로를 지나지 않으므로 영향이 없습니다.
- 이 검사는 버퍼에 도달한 바이트만 보는 best-effort 이며, 신뢰할 수 있는 서버에만 붙는다는 운영 원칙을 대신하지 않습니다. 가능하면 `STLS`/`STARTTLS` 대신 implicit TLS(POP3 995, IMAP 993)를 쓰는 것이 여전히 더 안전합니다.
- SMTP 발송 경로는 `net/smtp` 의 `StartTLS` 를 쓰므로 같은 검사가 들어가 있지 않습니다.
- POP3 다중행 응답 상한(`maxListBytes` 32 MiB)·IMAP 경로의 상한(`maxLineBytes` 1 MiB, `maxResponseBytes` 8 MiB, `maxResponseLiterals` 64)·드레인 예산·설정 키는 v0.23.8 그대로이며 MCP·OAuth·API Key 동작도 같습니다. [v0.23.8 릴리즈 내용](v0.23.8.md)을 참고하세요.

## 배포 파일

- `postra-0.23.9-linux-amd64`: 정적 단일 실행 파일
- `postra-0.23.9.tar.gz`: 폐쇄망 `docker load` 이미지, `postra:0.23.9` (linux/amd64)
- `postra-0.23.9-sbom.cdx.json`: Go CycloneDX SBOM
- `postra-0.23.9-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 SHA-256
