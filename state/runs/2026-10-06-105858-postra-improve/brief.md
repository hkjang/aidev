# 과제서 (2026-10-06 정찰, base `bf290e8` / v0.25.7)

- 과제: POP3·IMAP **implicit TLS** 의 인증서 검증 기본값과 `connectFailure` 단계 매핑을 회귀 테스트로 못 박는다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 계정 생성 시 수신 보안의 **기본값이 implicit TLS** 인데(`internal/application/accounts.go:62` `normSecurity(in.POP3Security, domain.SecurityTLS)`), 그 경로의 인증서 검증을 묶은 테스트가 수신 어댑터 양쪽에 **하나도 없다** — `grep -n "domain.SecurityTLS"` 가 `internal/adapters/pop3/client_test.go`·`internal/adapters/imap/client_test.go` 에서 0건이고 두 파일의 TLS 테스트는 전부 `SecurityStartTLS` + `InsecureSkipVerify: true` 다(2026-10-06 실측). 발신 쪽은 같은 보증을 이미 갖고 있다(`internal/adapters/smtp/client_test.go:443` "self-signed cert is rejected without InsecureSkipVerify"). 그래서 `InsecureSkipVerify: opts.InsecureSkipVerify`(pop3/client.go:69, imap/client.go:54)가 언젠가 상수 `true` 나 `MinVersion` 누락으로 바뀌어도 어떤 테스트도 빨개지지 않고, 운영자 진단이 이 실패를 "서버 인증서를 신뢰할 수 없었습니다"(sync_diagnostics.go:144)로 읽게 해 주는 `connectFailure` 의 TLS 분기(pop3/client.go:173-179, imap/client.go:168-175)도 미검증이다.
- 수용 기준:
  1) 기본 `domain.POP3DialOptions{Security: domain.SecurityTLS}`(= `InsecureSkipVerify` 를 **명시하지 않음**)로 자가서명 implicit-TLS 서버에 `Dialer{}.Dial` 하면 실패하고, 그 에러가 `errors.As` 로 `*domain.InboundError` 이며 `Stage == domain.StageTLS`·`Class == "tls_certificate"` 다. IMAP 쪽도 같은 단언(`domain.IMAPDialOptions` — 필드명은 열어서 확인할 것).
  2) **같은 픽스처**에 `InsecureSkipVerify: true` 만 더해 Dial 하면 세션이 성립한다(그 뒤 `Close()`). 이것이 없으면 테스트는 "항상 실패" 를 증명하는 것이라 기본값을 못 묶는다.
  3) 진단 에러 문자열에 서버 원문·인증서 내용이 들어가지 않는다(`pop3 connect:`/`imap connect:` + Go 의 x509 문구만) — 기존 관례대로 `strings.Contains` 로 단언해도 좋다.
  4) 프로덕션 코드 변경 **0줄**. `git status --porcelain` 에 테스트 파일만 남는다.
- 건드릴 파일 (프로덕션 0개):
  - `internal/adapters/pop3/client_test.go` — 새 테스트 1개 + 작은 implicit-TLS 픽스처. `selfSigned(t) *tls.Config`(:94, 이미 서버용 `Certificates`+`MinVersion` 을 담아 반환)을 `tls.NewListener(ln, selfSigned(t))` 에 그대로 넣고, `+OK POP3 ready\r\n` 을 쓴 뒤 `USER`/`PASS`/`QUIT` 에 `+OK` 로 답하는 `stlsServer`(:122) 축약판. `net.Listen("tcp","127.0.0.1:0")` + `t.Cleanup(ln.Close)` + `conn.SetDeadline` 은 그 함수 모양을 따를 것. 주소→옵션 변환은 `dialStartTLS`(:180)의 `net.SplitHostPort`+`strconv.Atoi` 관용구 재사용.
  - `internal/adapters/imap/client_test.go` — 같은 모양. `selfSigned`(:1006)·`starttlsServer`(:1035)·`dialStartTLS`(:1102) 가 자리하고, 평문 greeting 은 `* OK IMAP ready\r\n`, 로그인 없이 끝내려면 `Username: ""` 로 두면 `LOGIN` 과 `selectInbox` 를 건드리지 않는지 **열어서 확인**할 것 — imap/client.go:136-160 은 `!preAuth && opts.Username != ""` 일 때만 LOGIN 하지만 `selectInbox()` 는 무조건 호출하므로, 성공 쪽 하위 테스트는 `SELECT` 에 `* 0 EXISTS` + `tag OK` 를 답해 주거나 greeting 을 `* PREAUTH` 로 두는 등 한 번은 실제로 돌려 보고 맞출 것(미확인).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -race -count=1 -timeout=180s ./internal/adapters/pop3/` — 실측 약 5초
  - `go test -race -count=1 -timeout=180s ./internal/adapters/imap/` — 실측 약 4.3초
  - `go vet ./...` / `make lint-format`(gofmt, 빈 출력이어야 함)
  - 변이 검증: `pop3/client.go:69` 를 `InsecureSkipVerify: true` 로 임시 고정하면 **거부 하위 테스트만** 실패하고 opt-in 하위 테스트는 통과해야 한다. 확인 후 반드시 원복하고 `git diff --stat` 로 프로덕션 파일 0개를 재확인.
  - 프런트 미변경이므로 `internal/transport/spa/assets` 재생성·`npm` 게이트는 불필요(`git status --porcelain -- internal/transport/spa/assets` 가 비어 있음을 확인만).
- 위험과 피할 것:
  - **프로덕션 코드를 고치지 말 것.** 이 회차는 보증을 추가하는 것이고, `#nosec G402` 주석(pop3/client.go:66-69)과 `InsecureSkipVerify` 계정 opt-in 의미는 의도된 설계다(오프라인·사내 메일 서버 지원).
  - STARTTLS 주입 가드(pop3:121, imap:117)·`connectFailure` 의 `switch` 분기·`domain.ClassifyInbound` 의 순서는 건드리지 말 것 — 지난 회차(1c9f526)가 바로 이 블록을 지났고, 보안 판정이다.
  - 새 Stage/Class 레이블을 만들지 말 것. `tls_certificate` 는 `syncClassLabels`(sync_diagnostics.go:144)에 이미 있다.
  - 손으로 만든 대역 금지: 반드시 `net.Listen` 루프백 + `tls.NewListener` + **실제** `Dialer{}.Dial` 과 실제 DialOptions 로 돌릴 것(이 저장소의 확립된 관례이자 운영자의 반복 지시).
  - `web/` 아래에 파일을 만들지 말 것(Tailwind v4 자동 탐지 → 자산 드리프트 33건 교훈).
  - 두 어댑터를 같이 담아도 파일은 2개(둘 다 테스트)지만, IMAP 의 `selectInbox` 때문에 성공 쪽 픽스처가 길어지면 **IMAP 을 떼고 POP3 만** 이번 회차에 담을 것 — 쪼개는 쪽이 맞다.
- 차선 후보: `cyclonedx-gomod` 가 git **워크트리**에서 `failed to determine version of main module` 로 죽는 것(full clone·shallow no-tags 에서는 통과 — CI 결함 아님)을 `README.md:296-315` 의 보안 스캐너 절에 한 줄로 메모해 다음 정찰·수리의 오진을 막는다 (가치 1 / 위험 1 / S, 문서 단독). 워크플로 YAML 은 손대지 말 것.
