# Postra v0.25.8 — 기본값인 implicit TLS 가 인증서를 검증한다는 보증

**테스트 전용 릴리즈입니다.** 계정을 만들 때 수신 보안의 기본값은 implicit TLS(POP3 995 · IMAP 993)인데, **그 경로에서 서버 인증서를 검증한다는 사실을 확인하는 테스트가 양쪽 수신 어댑터에 하나도 없었습니다.** 이번 릴리즈는 그 보증을 어댑터마다 회귀 테스트 하나로 못 박습니다. **프로덕션 Go/TypeScript 코드 변경은 0줄입니다** — 기능 변경, 새 설정 키, 새 환경변수, DB 마이그레이션, 화면 변경이 **없습니다**.

**운영자가 할 일은 없습니다.** 실행 파일과 컨테이너 이미지의 동작은 v0.25.7 과 같습니다. 이번 변경이 닿는 것은 저장소의 테스트 그물뿐입니다.

## 무엇이 덮여 있지 않았나

- 계정 생성은 수신 보안을 `normSecurity(in.POP3Security, domain.SecurityTLS)`(`internal/application/accounts.go`)로 정규화합니다. 즉 **설정을 비워 둔 계정은 implicit TLS 로 붙습니다** — 대부분의 설치가 실제로 거는 다이얼입니다.
- 그런데 `internal/adapters/pop3/client_test.go` 와 `internal/adapters/imap/client_test.go` 의 TLS 테스트는 **전부** `SecurityStartTLS` + `InsecureSkipVerify: true` 였습니다. `domain.SecurityTLS` 를 쓰는 테스트가 양쪽에 0건이었습니다.
- 결과적으로 `tlsCfg` 의 `InsecureSkipVerify`(`pop3/client.go:69`, `imap/client.go:54`)를 **상수 `true` 로 고정해도 두 패키지의 어떤 테스트도 빨개지지 않았습니다.** 계정별 opt-in 이 전역 기본값으로 번지는 변경이 테스트를 통과할 수 있었다는 뜻입니다.
- 발신 쪽은 이 보증을 처음부터 갖고 있었습니다(`internal/adapters/smtp/client_test.go`, "self-signed cert is rejected without InsecureSkipVerify"). 수신 쪽만 비어 있던 **비대칭**이었습니다.
- 함께 비어 있던 것이 하나 더 있습니다: 양쪽 `connectFailure` 의 **TLS 분기**. implicit TLS 는 TCP 연결과 핸드셰이크가 dialer 호출 한 번에 일어나므로, 두 단계를 가르는 것은 **오류 타입뿐**입니다. 그 분기가 잘못되면 인증서 거부가 `서버 연결(TCP)` 실패로 보고됩니다 — v0.25.7 이 STARTTLS 쪽에서 고친 것과 같은 종류의 오진입니다.

## 무엇이 추가됐나

어댑터마다 테스트 1개(하위 2개)입니다. **같은** 자가서명 implicit-TLS 리스너에 `InsecureSkipVerify` **한 필드만** 달리해 실제 `Dialer{}.Dial` 로 붙습니다:

- **기본값(`InsecureSkipVerify` 영값)은 실패해야 합니다.** 그리고 그 실패가 `errors.As` 로 `*domain.InboundError` 이고, `Stage == tls_handshake`, `Class == "tls_certificate"`, `Elapsed > 0` 임을 단언합니다. 이 `Stage`·`Class` 쌍이 바로 `internal/application/sync_diagnostics.go` 가 **"서버 인증서를 신뢰할 수 없었습니다"** 로 읽는 조합입니다 — `tcp_connect` 이 아닙니다.
- **opt-in(`InsecureSkipVerify: true`)은 세션이 성립해야 합니다.** 이 두 번째 하위 테스트가 없으면 첫 번째는 "서버가 안 떠 있다" 와 구별되지 않아 **기본값에 대한 진술이 되지 못합니다.**
- 오류 문자열에는 어댑터 접두사(`pop3 connect:` / `imap connect:`)와 Go 의 `x509:` 문구만 담기고 **서버 인사말(`POP3 ready` / `IMAP4rev1 ready`)은 담기지 않음**을 단언합니다.
- 픽스처는 손으로 만든 대역이 없습니다: `net.Listen("tcp", "127.0.0.1:0")` + `tls.NewListener(ln, selfSigned(t))` + 실제 `Dialer{}.Dial` + 실제 `DialOptions` 입니다. 인증서는 기존 STLS 픽스처의 것을 그대로 재사용하므로 두 하위 테스트의 비교가 의미를 가집니다.

**프로덕션 코드는 손대지 않았습니다.** `#nosec G402` 주석과 계정별 `InsecureSkipVerify` opt-in 은 의도된 설계입니다 — 폐쇄망·자체 서명 메일 서버를 위해 존재하며, 이 테스트는 그 **존재가 아니라 기본값**을 못 박습니다.

**일부러 묶지 않은 것 하나를 밝혀 둡니다.** `tlsCfg` 의 `MinVersion` 은 이 테스트가 보증하지 않습니다. Go 클라이언트의 기본 최소 버전이 이미 TLS 1.2 라서 그 줄을 지워도 이 테스트는 깨지지 않으며, 깨진다고 적는 것은 거짓 주장이 됩니다. 테스트 주석에도 그렇게 적혀 있습니다.

## 검증

- **변이 4종으로 인과를 고정했습니다.** 프로덕션 변경이 없는 보증 추가 회차라 "수정 전 실패" 가 존재하지 않으므로, 대신 보증을 하나씩 되돌려 테스트가 실제로 그 경로를 묶는지 확인했습니다. 네 번 모두 **거부 하위 테스트만** 실패하고 opt-in 하위 테스트는 통과했습니다:
  - `pop3/client.go:69` 를 `InsecureSkipVerify: true` 로 → `Dial accepted a self-signed certificate with the default options; want verification`
  - pop3 `connectFailure` 의 `switch` 에서 `"tls_certificate"` 제거 → `Stage = "tcp_connect", want "tls_handshake" — tcp_connect would tell the operator the connection failed`
  - `imap/client.go:54` 에 같은 변이 → 같은 거부 메시지
  - imap `connectFailure` 에 같은 변이 → 같은 단계 메시지
  - 각각 원복하고 잔재 0건을 재확인했습니다(`InsecureSkipVerify: true` 양쪽 0건, `case "tls_certificate", "tls_handshake":` 양쪽 1건).
- **플레이크 확인**: `go test -race -count=3` 로 `./internal/adapters/pop3/`·`./internal/adapters/imap/` 양쪽 통과, 플레이크 0건.
- `go test -race -count=1 -timeout=900s ./...` 25개 패키지 실패 0건
- `go build ./...`, `go vet ./...`, `make lint-format` 무지적, `go mod tidy` 무변경, canonical API 계약 검사(`postra-contracts -check`) 종료 코드 0
- `make lint-security` — 141개 파일 지적 0건
- `make frontend-check` 통과 후 `internal/transport/spa/assets` 변화 없음 — 태그에 담긴 `/app` 번들이 최신입니다(릴리즈 워크플로가 같은 검사를 다시 합니다)
- 외부 PostgreSQL(`POSTRA_TEST_PG`)과 브라우저 e2e 는 돌리지 않았습니다 — 테스트 전용 Go 변경이라 범위 밖입니다.

## 업그레이드 및 호환성 주의사항

DB 마이그레이션, 새 설정 키, 새 필수 환경변수가 없습니다. 실행 파일 또는 컨테이너 이미지를 교체한 뒤 재시작하면 끝이며, **교체하지 않아도 잃는 것이 없습니다** — 운영 동작은 v0.25.7 과 동일합니다. 되돌릴 때도 v0.25.7 로 그대로 내려갈 수 있습니다.

- 인증서 검증 **동작**은 바뀌지 않았습니다. 자가서명 인증서를 쓰는 메일 서버에 붙고 있다면 지금도 계정의 `InsecureSkipVerify` opt-in 이 필요하고, 그 의미도 그대로입니다. 이번 릴리즈는 그 opt-in 을 켜지 않은 계정이 **앞으로도** 인증서를 검증한다는 것을 저장소가 스스로 지키게 만든 것뿐입니다.
- 동기화가 `TLS 핸드셰이크` 단계에서 "서버 인증서를 신뢰할 수 없었습니다" 로 멈추면, 그것은 네트워크 문제가 아니라 **서버 인증서 체인을 검증할 수 없었다**는 뜻입니다. 서버 인증서와 중간 CA 를 먼저 확인하고, 자체 서명이 의도된 환경이라면 해당 계정에서만 opt-in 하세요.
- v0.25.7 의 `STARTTLS 전환` 단계 진단, v0.25.5 의 거부된 `LIST` 크기 선별 경고, v0.25.4 의 크기 제한 없는 계정 본문 잘림 수정, v0.25.3 의 POP3 `fetch` 진단 교정, v0.25.2 의 OpenTelemetry v1.45.0, v0.25.1 의 "계정당 인바운드 연결 하나"와 `mail.idle_enabled`, v0.25.0 의 동기화 실패 진단은 바뀌지 않았습니다.

## 배포 파일

- `postra-0.25.8-linux-amd64`: 정적 단일 실행 파일
- `postra-0.25.8.tar.gz`: 폐쇄망 `docker load` 이미지, `postra:0.25.8` (linux/amd64)
- `postra-0.25.8-sbom.cdx.json`: Go CycloneDX SBOM
- `postra-0.25.8-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 SHA-256
