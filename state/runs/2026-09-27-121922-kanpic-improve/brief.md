- 과제: 알림 메일 본문의 점 이스케이프가 SMTP 와이어에서 한 번만 일어나는지 못 박는다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/mail/message.go:normalizeBody` 는 줄 앞 `.` 을 손으로 한 번 더 붙이는데(`strings.ReplaceAll(body, "\r\n.", "\r\n..")`), 본문을 실제로 내보내는 `internal/mail/mail.go:123` 의 `client.Data()` 가 돌려주는 것은 `net/textproto` 의 dot writer 이고 그것도 줄 앞 점을 이스케이프한다(**미확인 — 이번 회차에 Go 를 못 돌렸다. 구현자가 1단계에서 확인할 것**). 그렇다면 받는 사람 화면에 `.점으로 시작` 이 `..점으로 시작` 으로 보인다. 기존 테스트 `TestDeliverWithoutAuthentication`(mail_test.go:151)은 `strings.Contains(body, "..점으로 시작")` 이라서 와이어에 `...점으로 시작` 이 실려도 그대로 통과한다 — 이 갈래를 가려낼 수 없는 단언이다.

- 수용 기준:
  1) `internal/mail/mail_test.go` 에 실제 `fakeRelay`(이미 있는 in-process SMTP 서버, mail_test.go:26·68)로 `Deliver` 를 통과시켜 **와이어에 실제로 실린 DATA 본문**을 RFC 5321 규칙대로 되돌린(줄 앞 `..`→`.`) 결과가 `Message.Body` 의 각 줄과 **정확히 같음**을 단언하는 테스트가 있다. 사례에 `.점으로 시작`, `..이미 두 점`, 첫 줄이 `.` 인 본문을 포함한다.
  2) 그 테스트가 지금 코드에서 실제로 **빨강** 이면(= dot writer 가 이스케이프한다) `normalizeBody` 의 손 이스케이프 두 줄(`if strings.HasPrefix(body, ".")` 와 `ReplaceAll(body, "\r\n.", "\r\n..")`)을 지워 초록으로 만들고, 지운 뒤 한 줄을 되살려 다시 빨강이 되는 것을 확인한다.
     - 만약 **초록**이면(= dot writer 가 이스케이프하지 않는다) 프로덕션 코드는 옳다. 그때는 코드를 고치지 말고, mail_test.go:151 의 느슨한 `Contains` 단언을 1)의 정확 비교로 **바꾸는 것까지만** 하고 끝낸다(이 계약이 다시 흔들리지 않게 못 박는 것이 이번 성과다).
  3) `Subject`·`Content-Type`·CRLF 줄끝 계약은 그대로다 — 기존 mail 테스트 6개가 모두 통과한다.

- 건드릴 파일 (2개):
  - `internal/mail/mail_test.go` — 새 테스트 1개 추가(`fakeRelay`·`relayConfig`·`Deliver` 재사용, 새 대역을 만들지 말 것), 151행의 느슨한 `Contains` 단언을 정확 비교로 교체.
  - `internal/mail/message.go:normalizeBody` — 수용 기준 2)가 빨강일 때만, 손 이스케이프 두 줄 제거. CRLF 정규화와 끝 `\r\n` 보장은 남긴다(주석에 "와이어의 점 이스케이프는 net/textproto dot writer 가 한다" 를 적을 것).

- 검증 명령:
  - `go test ./internal/mail -run 'TestDeliver|TestNotification' -v` (빨강→초록을 여기서 본다)
  - `go test ./internal/mail`
  - `go test ./...`
  - `gofmt -l ./cmd ./internal ./pkg` · `go vet ./...` · `go build ./...`
  - `./scripts/check-release-docs.sh` · `./scripts/check-commit-identities.sh HEAD`

- 위험과 피할 것:
  - **손으로 만든 대역으로 증명하지 말 것.** `compose()` 의 반환 문자열만 보는 단위 테스트로는 이 결함이 원리적으로 안 보인다(문제는 `compose` 와 `client.Data()` 의 **합**이다). 반드시 `fakeRelay` 를 거친 와이어 바이트를 본다.
  - TLS·STARTTLS·AUTH(`startSession`, `loginAuth`), `config.tlsConfig()` 의 `SkipVerify`, `helloName` 은 건드리지 말 것 — 이번 과제와 무관하고 운영 보안 경로다.
  - `To:` 헤더가 인코딩·검증 없이 붙는 것(message.go:13)은 이번 범위 밖이다. `client.Rcpt` 가 CR/LF 를 먼저 거부하므로 주입은 막혀 있다(Go 의 `validateLine`). 보류 아이디어로만 남긴다.
  - `internal/mail/service.go` 의 수신자 결정·이벤트 설정은 건드리지 말 것. 본문 문구(`Notification.Render`·`quote`)도 바꾸지 말 것 — 문구를 바꾸면 무엇이 결함 수정인지 흐려진다.
  - 문서·PDF 재생성 불필요(가이드는 메일 본문의 점 이스케이프를 말한 적이 없음 — 미확인이지만 관리자 가이드 SMTP 절에 와이어 수준 설명은 없을 것으로 본다. 있으면 그때 함께 고칠 것).
  - 웹·DB 통합·브라우저 E2E 는 돌리지 않아도 된다(Go 서버 안쪽, 스키마 무관).

- 차선 후보: 관리자 가이드의 외부 호출 문제 해결 줄을 실제 오류 코드와 맞춘다 (가치 2 / 위험 1 / 작업량 S) — `docs/ADMIN_GUIDE.md:785` 은 증상을 "`#VALUE!` 만 돌려준다" 로 적었지만, 설정 읽기 실패·`external.enabled` 꺼짐·호출 수 초과·호스트 거절은 `internal/external/fetcher.go:124,130,136,158,171` 에서 모두 `#N/A` 다(이번 회차에 확인). `#VALUE!` 는 fetcher.go:399 의 CSV 읽기 실패 한 갈래뿐이다. 표의 증상 칸을 `#N/A` 로 바로잡고 `#VALUE!` 는 CSV 읽기 실패로 따로 적는다. `docs/ADMIN_GUIDE.pdf` 재생성이 따라온다(`scripts/generate_pdf.js`, `web/node_modules/playwright-core` 필요 — 없으면 `cd web && npm ci` 선행). 검증: `./scripts/check-release-docs.sh`.
