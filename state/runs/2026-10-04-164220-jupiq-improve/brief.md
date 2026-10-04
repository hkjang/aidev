# 과제서 — 2026-10-04-164220-jupiq-improve (jupiq, base main@92ff88c, VERSION 1.8.7)

- 과제: 메일 본문을 8BITMIME 협상 없이 생 8비트로 보내는 것을 quoted-printable 7비트 인코딩으로 바꾼다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/mail/mail.go:compose`(202행)가 `Content-Transfer-Encoding: 8bit` 헤더를 붙이고 `normalizeBody`가 돌려준 **생 UTF-8 본문**을 그대로 DATA 로 쓰는데, `Deliver`·`startSession` 어디에도 `client.Extension("8BITMIME")` 확인이 없다(`startSession`은 STARTTLS 와 AUTH 두 확장만 본다 — 실제로 읽어서 확인했다). RFC 6152 는 서버가 8BITMIME 를 광고하고 `MAIL FROM ... BODY=8BITMIME` 로 선언했을 때만 8비트 본문을 허용하므로, 이 패키지가 스스로 전제한 대상 환경(패키지 주석: "사내 릴레이는 포트 25·인증 없음·TLS 없음이 흔하므로")의 7비트 전용 릴레이에서는 한글 제목·본문 메일이 거부되거나 본문이 깨져 도착한다. 제목은 이미 `mime.QEncoding.Encode` 로 7비트화되어 있어 **본문만 규약을 어기고 있다** — 한 함수에서 같은 규칙을 적용하면 끝난다.

- 수용 기준:
  1) `compose`(또는 그것이 부르는 새 헬퍼)의 출력 바이트가 **전부 7비트**다(모든 바이트 < 0x80). 한글 제목·본문을 넣어도 그렇다.
  2) 헤더가 `Content-Transfer-Encoding: quoted-printable` 로 바뀌고 `Content-Type: text/plain; charset=UTF-8` 은 그대로다.
  3) 본문을 `mime/quotedprintable.NewReader` 로 되돌리면 원래 UTF-8 본문과 **바이트 단위로 같다**(줄 끝은 CRLF 정규화 뒤 기준). 긴 한 줄(76자 넘는 한글)이 소프트 줄바꿈(`=` + CRLF)으로 접히고도 되돌려지는 것이 테스트로 증명돼야 한다.
  4) **8BITMIME 를 광고하지 않는 릴레이**로 실제 발송이 성공하고, 릴레이가 받은 DATA 에 8비트 바이트가 없다. 증명은 기존 `fakeRelay`(`internal/mail/mail_test.go:17-140`, 실제 `net.Listen` + 실제 SMTP 대화 + 실제 `smtp.Client`)를 써서 하고, 손으로 만든 대역 객체나 소스 문자열 검사로 하지 않는다. `fakeRelay.handle`(118행)이 지금은 EHLO 응답에 `250 8BITMIME` 을 **무조건** 쓰므로, 광고를 끌 수 있는 필드(예: `noEightBit bool`)를 테스트 파일에 더하고 그 릴레이로 `Deliver` 를 호출해 수정 전에는 8비트 바이트가 DATA 에 실려 나가는 것을 먼저 빨갛게 보여라.
  5) 줄 머리 점 이스케이프는 여전히 `smtp.Client.Data` 의 DotWriter 한 곳에서만 일어난다(점이 둘 보이지 않는다). `normalizeBody` 의 CRLF 정규화·끝 CRLF 보장도 유지한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/mail/mail.go:compose` — `Content-Transfer-Encoding` 을 `quoted-printable` 로 바꾸고, `normalizeBody(message.Body)` 결과를 `mime/quotedprintable.NewWriter` 로 인코딩해 쓴다. 표준 라이브러리이므로 새 의존성 없다. 왜 그렇게 했는지(8BITMIME 협상이 없으므로 항상 7비트로 보낸다)를 이 저장소 관례대로 **영어 주석**으로 옆에 남길 것 — `mail.go` 의 기존 주석은 한국어지만 `internal/store/resource_usage.go` 쪽은 영어다. 이 파일은 한국어 주석이 관례이므로 **한국어로 맞추는 쪽이 안전하다**(mail.go 전체가 한국어 주석이다 — 확인했다).
  - `internal/mail/mail_test.go` — `fakeRelay` 에 8BITMIME 광고를 끄는 필드 추가 + 신규 테스트(7비트 단언, QP 왕복, 광고 없는 릴레이 발송). **그리고 기존 `TestDeliverSpeaksPlainSMTPWithoutAuthByDefault`(254행)의 `"\r\n..점으로 시작하는 줄\r\n"` 단언이 깨진다** — 본문이 QP 로 인코딩되면 그 줄은 `..=EC=A0=90...` 형태가 되기 때문이다(줄 머리 `.` 자체는 QP 가 건드리지 않으므로 DotWriter 의 이중 점은 그대로 남는다). 이 단언을 **약화시키지 말고** "QP 로 디코딩한 뒤 `.점으로 시작하는 줄` 이 있고, 수신 바이트에는 이중 점이 남아 있다" 로 더 강하게 고쳐라.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -count=1 ./internal/mail` — 집중. **이번 정찰이 base 에서 실제로 돌렸다: `ok github.com/hkjang/jupiq/internal/mail 8.640s`** (발송 예산·동시성 테스트가 실제로 기다리므로 8~9초가 정상이다 — 느리다고 끊지 말 것).
  - `gofmt -l .`(무출력), `go vet ./...`, `go build ./...`, `go test -count=1 ./...`, `go test -count=1 -race ./internal/mail`
  - 통합은 이 변경과 무관하지만 게이트가 요구하면: `JUPIQ_INTEGRATION_TEST_DSN=... make test-integration`
  - `git diff --check`(무출력)

- 위험과 피할 것:
  - **메시지 바이트를 고정한 기존 단언이 깨진다.** 위에 적은 254행 단언 외에 `Subject: =?utf-8?q?` / `From: =?utf-8?q?` 단언은 헤더 인코딩이므로 그대로 통과해야 한다 — 통과하지 않으면 헤더를 잘못 건드린 것이다.
  - **협상 방식(8BITMIME 를 광고하면 8bit, 아니면 QP)으로 분기하지 말 것.** 두 경로가 되면 `Extension` 결과를 `Deliver` → `compose` 로 배선해야 하고, 8BITMIME 릴레이도 quoted-printable 을 문제없이 받으므로 얻는 것이 없다. 한 경로로 유지해라.
  - `MAIL FROM` 에 `BODY=8BITMIME` 를 붙이지 말 것(`smtp.Client.Mail` 이 이미 서버 광고를 보고 SMTPUTF8/8BITMIME 를 처리하며, QP 로 보내면 애초에 필요 없다).
  - 보호 경로를 건드리지 말 것: `internal/auth`, `migrations/`, `.github/workflows/`, SMTP 자격증명 정책(`credentialsAllowed`·`loginAuth`·`startSession`). 이번 변경은 `compose` 와 테스트뿐이다.
  - `openapi/openapi.yaml`·`web/`·`docs/`·`VERSION` 은 건드릴 것이 없다(메일 메시지 바이트는 API 계약이 아니다).
  - 다른 회차 산출물과 겹치지 않게: base(92ff88c) 에는 marshal-first `writeJSON`(helpers.go:29 는 아직 `WriteHeader` 먼저 + `Encode` 오류 버림)과 `resource_usage.go` 의 `- interval '1 hour'` 롤업 수정이 **둘 다 없다** — 머지 전인 다른 회차 PR 이므로 이번에 손대지 말 것.

- 차선 후보: **메일 발송 기록 `limit` 의 잘못된 정수(abc·-1·0)를 실제 HTTP 로 고정하는 회귀 테스트** (가치 2 / 위험 1 / 작업량 S). `internal/api/mail_handlers.go:21` 의 `queryIntOrReject(w, r, "limit", 50)` 가 `queryInt`(helpers.go:87, 비정수·음수를 400 `invalid_query` 로 거부, `limit=0` 은 통과시켜 store 의 `boundedLimit(0, 50, 200)` 이 50 으로 되돌린다 — 소스로 확인했다) 를 타는 것은 코드상 맞지만 HTTP 경계에서 고정돼 있지 않다. `internal/api/mail_deliveries_integration_test.go` 가 base 에 이미 있으므로 거기에 서브테스트만 더하면 된다. 실결함이 아닌 테스트 공백이라 1순위보다 가치가 낮다.

---
## 실측 (이번 정찰이 실제로 한 것)
- 읽은 파일: `internal/mail/mail.go`(전체), `internal/mail/message.go`(전체), `internal/mail/config.go`(93-216행), `internal/mail/service.go`(함수 목록 + `resolve` 238-275행), `internal/mail/mail_test.go`(17-60, 100-140, 254-285행), `internal/api/helpers.go`(1-60, 75-175행), `internal/api/mail_handlers.go`(1-60행), `internal/store/store.go`(198-210행), `internal/store/resource_usage.go`(113-116행).
- 함께 확인했고 **결함이 아니었던 것**(다음 정찰이 다시 파지 않도록):
  - `compose` 의 `Subject` 헤더 주입: `mime.QEncoding.Encode` 의 `needsEncoding` 이 `\r`·`\n`(< 0x20)을 인코딩 대상으로 보므로 CRLF 가 `=0D=0A` 로 인코딩된다 — 주입 불가.
  - `To` 헤더 주입: `compose` 는 `message.To` 를 생으로 쓰지만 그 앞에서 `smtp.Client.Rcpt` 가 CR/LF 를 거부하므로 DATA 까지 가지 못한다. 또 `Service.resolve`(service.go:265)가 DB 에서 온 주소에 `validAddress` 를 적용하고 `validAddress`(config.go:192)가 `" <>,\r\n\t\""` 를 거부한다.
  - `Config.Link` 의 이중 슬래시: `ReadConfig`(config.go:104)가 `strings.TrimRight(..., "/")` 로 이미 자른다.
  - `Sort` 방향 주입: `internal/store/sort_test.go:27` 에 이미 회귀 테스트가 있다.
- **미확인**: 실제 7비트 전용 사내 릴레이에서의 거부·깨짐은 재현하지 않았다(그런 릴레이가 없다). 1순위의 증거는 "8BITMIME 를 광고하지 않는 릴레이에 8비트 DATA 를 보낸다"는 **프로토콜 위반 자체**이며, 그것은 로컬 `fakeRelay` 로 실제 SMTP 대화로 재현할 수 있다. 구현자는 이 구분을 커밋 메시지에 그대로 적을 것(결함 재현이 아니라 규약 위반 제거 + 상호운용성 확보).
