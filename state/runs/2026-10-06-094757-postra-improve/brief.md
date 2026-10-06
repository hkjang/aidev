# 과제서 (2026-10-06 정찰) — base main@e6d2de0

- 과제: STARTTLS 주입 거부가 `tcp_connect` 단계로 잘못 보고되는 것을 `starttls` 단계로 고정 (가치 3 / 위험 1 / 작업량 S)
- 왜: POP3 `internal/adapters/pop3/client.go:113-114` 와 IMAP `internal/adapters/imap/client.go:112-114` 의 STARTTLS 주입 가드(`if n := s.text.R.Buffered(); n > 0` / `if n := s.r.Buffered(); n > 0`)가 거부할 때 `fmt.Errorf("STLS: server sent %d bytes before TLS handshake", n)` 라는 **평문 error** 를 돌려준다 — 같은 블록의 바로 다음 줄인 핸드셰이크 실패(pop3 `:121-124`, imap `:120-121`)는 `&domain.InboundError{Stage: domain.StageTLS, Command: "STLS"/"STARTTLS", Class: domain.ClassifyInbound(err), Elapsed: time.Since(tlsStart), Timeout: s.commandTO, ...}` 로 감싸는데 이 가드 한 줄만 감싸지 않는다. `internal/application/sync_diagnostics.go:584` 의 dial 실패 경로는 `syncDiagnosticStage(d, domain.StageTCPConnect, err)` 이고 `syncDiagnosticStage`(:166-188)는 `errors.As(err, &inbound)` 가 실패하면 **fallback 인 `tcp_connect`** 를 그대로 기록하므로, 보안상 가장 중요한 거부가 운영자 진단에 "TCP 연결" 실패로 보이고 경과도 0 이다. 고치면 이 거부가 제 단계로 보고된다 — `domain.StageStartTLS`(`internal/domain/inbound_error.go:46`)와 한국어 레이블 `"STARTTLS 전환"`(`sync_diagnostics.go:126`)은 **이미 있고**, `commandStage`(pop3 `client.go:186`)·imap `client.go:773` 도 STLS/STARTTLS **명령** 실패에는 그 단계를 쓴다. 가드 거부만 그 대열에서 빠져 있다.
- 수용 기준:
  1) 핸드셰이크 전에 여분 바이트를 밀어 넣는 서버에 `Dial` 하면 반환 error 가 `errors.As(err, &inbound)`(`inbound *domain.InboundError`)로 잡히고 `Stage == domain.StageStartTLS`, `Command` 가 실제 보낸 동사(POP3 `"STLS"`, IMAP `"STARTTLS"`)이며 `Elapsed > 0`, `Timeout == s.commandTO`.
  2) 거부 자체가 **그대로 유지**된다 — 여분 바이트가 있으면 `Dial` 이 실패하고 업그레이드하지 않고 연결을 닫는다(기존 `TestPOP3STLSInjectedPlaintextRefusesUpgrade`·`TestIMAPStartTLSInjectedPlaintextRefusesUpgrade` 의 단언이 전부 무수정 통과해야 한다). 정상 서버(inject="") 쪽 기존 테스트(`TestPOP3STLSUpgradeRunsCommandsOverTLS`·`TestIMAPStartTLSUpgradeRunsCommandsOverTLS`)도 무수정 통과.
  3) 진단에 서버 원문이 들어가지 않는다 — 바이트 수(숫자)만. `Class` 는 **새 레이블을 만들지 않고** `domain.ClassifyInbound` 의 값을 쓴다(평문 error → `"other"`, `syncClassLabels` 에 `"other": "알 수 없는 오류로 중단되었습니다"` 가 있음을 이번에 확인했으므로 `allowed()` 가 버리지 않는다).
  4) 수정 전 실패 재현: 위 1)의 `errors.As`/`Stage` 단언이 수정 전 FAIL → 수정 후 PASS. 변이 검증으로 인과 고정(아래).
- 건드릴 파일 (프로덕션 2 + 테스트 2 = 4개):
  - `internal/adapters/pop3/client.go` — `Dial` 의 주입 가드 `return` 을 `&domain.InboundError{Stage: domain.StageStartTLS, Command: "STLS", Class: domain.ClassifyInbound(err), Elapsed: …, Timeout: s.commandTO, Err: fmt.Errorf("STLS: server sent %d bytes before TLS handshake", n)}` 로. 경과 측정 시작점은 `s.cmd("STLS")` 호출 직전에 `stlsStart := time.Now()` 를 잡는다(같은 함수의 `tlsStart`/`greetStart`/`dialStart` 관용구와 동일). `Class` 를 채울 `err` 가 없으므로 `Class: "other"` 를 직접 쓰거나 생성한 `Err` 를 `ClassifyInbound` 에 넘길 것 — 둘 다 `"other"` 가 되지만 후자가 저장소 관용구에 가깝다.
  - `internal/adapters/imap/client.go` — 같은 모양의 가드(`:112-114`), `Command: "STARTTLS"`, 시작점은 `s.exec("STARTTLS")`(`:95`) 직전.
  - `internal/adapters/pop3/client_test.go:200` `TestPOP3STLSInjectedPlaintextRefusesUpgrade` — 기존 테스트에 1)의 단언을 **추가**한다. 픽스처는 이미 있다: `stlsServer(t, inject, uidl, list string)`(`:116`), `dialStartTLS(t, addr) (domain.POP3Session, error)`(`:180`, 실제 `Dialer{}.Dial` + 실제 `domain.POP3DialOptions{Security: domain.SecurityStartTLS}`), `selfSigned(t)`(`:92`). **새 픽스처·새 파일을 만들 필요가 없다.**
  - `internal/adapters/imap/client_test.go:1119` `TestIMAPStartTLSInjectedPlaintextRefusesUpgrade` — 같은 추가. 픽스처 `starttlsServer(t, inject string) (string, <-chan struct{})`(`:1035`), `dialStartTLS(t, addr)`(`:1102`), `selfSigned(t)`(`:1006`) 모두 존재.
  - 두 패키지 모두 손으로 만든 대역이 아니라 `net.Listen` 루프백 + 실제 `Dialer{}` 로 돈다(픽스처 주석에서 확인).
- 검증 명령:
  - `go test -race -count=1 -timeout=180s ./internal/adapters/pop3/` (약 3~5초)
  - `go test -race -count=1 -timeout=300s ./internal/adapters/imap/` (실측 **미확인** — 이번 정찰은 돌리지 않았다)
  - `go build ./... && go vet ./...`
  - `go test -race -count=1 ./internal/application/` (약 83초 — 진단 조립 회귀 확인)
  - `make lint-format`(빈 출력) · `make lint-security`(gosec Issues 0) · `go run ./cmd/postra-contracts -check`(exit 0) · `git diff --check`
  - 변이 검증: 새 `InboundError` 를 다시 `fmt.Errorf(...)` 로 되돌리면 **새 단언만** 실패하고 기존 거부·정상 업그레이드 테스트는 통과함을 POP3·IMAP 각각 확인한 뒤 원복.
  - 프런트 미변경이므로 `git status --porcelain --untracked-files=all -- internal/transport/spa/assets` 가 **빈 출력**임을 확인(자산 재빌드 불필요).
- 위험과 피할 것:
  - `domain.SyncDiagnostic` 에 **필드를 추가하지 말 것** — `api/`·web·`internal/transport/spa/assets` 로 번진다. 이 과제는 기존 필드만 바르게 채운다.
  - `syncClassLabels`/`syncStageLabels` 에 **새 레이블을 추가하지 말 것**. 필요한 둘(`starttls`, `other`)이 이미 등록돼 있다.
  - 가드의 **판정 로직과 거부 자체를 바꾸지 말 것**(보안 경로): `Buffered() > 0` 조건, `conn.Close()`, 업그레이드 중단은 그대로. 이 과제는 오로지 "같은 거부를 바른 단계로 보고" 다.
  - `retrBody`(PR #22 / 04b15be 영역 — 사람 반려 여부 미확인)·`rawReadLimit`·`maxBytes > 0` 규약·설정 파싱·`.github/workflows`·`spa/assets`·OIDC/세션/마이그레이션은 손대지 말 것.
  - `web/` 아래에 **새 파일을 만들지 말 것**(Tailwind v4 자동 콘텐츠 탐지 → 자산 드리프트 33건 교훈).
  - IMAP 은 계정당 동시 연결이 하나다 — 테스트에서 두 번째 연결을 열지 말 것.
  - 두 패키지를 한 번에 하는 것이 맞다(같은 한 줄 결함의 쌍둥이, 파일 4개). 다만 IMAP 쪽이 예상 밖으로 커지면 **POP3 만** 담고 IMAP 은 다음 회차로 쪼갤 것.
- 차선 후보: POP3 implicit TLS 인증서 검증 기본값 회귀 테스트 (가치 2 / 위험 1 / S) — `pop3/client.go:62-72` 의 `tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: opts.InsecureSkipVerify}` 와 `SecurityTLS` 분기(`:73`)를 기존 `selfSigned(t)` 헬퍼 + 루프백 서버로 묶어, 기본값(`InsecureSkipVerify` 미지정)에서 자가서명 서버 연결이 `Stage == domain.StageTLS`·`Class == "tls_certificate"` 로 거부되고(`connectFailure`, `client.go:165-174` 가 TLS 오류 타입으로 단계를 가린다) 명시 opt-in 에서만 통과함을 고정한다. 프로덕션 변경 0개.
