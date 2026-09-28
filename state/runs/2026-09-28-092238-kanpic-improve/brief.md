- 과제: 알림 메일의 "누가 받는가" 계약을 실제 relay 로 못 박고, 요청이 끊겨도 알림이 조용히 사라지지 않게 한다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/mail/service.go` 는 **테스트가 한 줄도 없다** — `go test ./internal/mail -coverprofile` 로 패키지 전체가 35.3% 이고, `mail_test.go` 안에 `NewService`·`Notify`·`SendNow`·`Service{` 문자열이 0건이다(grep 확인). mail.go(전송)·message.go(문구)는 `fakeRelay` 로 촘촘히 덮여 있는데, 정작 **누구에게 보낼지 정하는 자(`resolve`)와 관리자가 끈 이벤트를 거르는 자(`Notify` 의 게이트)** 는 아무도 보고 있지 않다 — 알림이 본인에게 되돌아가거나, 관리자가 끈 이벤트가 나가거나, 같은 사람에게 두 번 가도 회귀를 잡을 장치가 없다. 그 위에 한 가지 의심 갈래가 있다: `Notify` 는 요청 컨텍스트로 `Config`·`resolve` 를 부르는데(service.go:74,78), 같은 파일의 `SendNow` 는 `context.WithoutCancel(ctx)` 를 쓴다(service.go:106) — 브라우저가 칸 저장·댓글 등록 직후 떠나 요청이 취소되면 변경은 커밋됐는데 알림만 조용히 사라질 수 있다.

- 수용 기준:
  1) `internal/mail/service_test.go` 가 `NewService(nil, settings, directory, logger)` 로 만든 **진짜 `Service`** 를 `startRelay`/`relayConfig`(mail_test.go:27,125) 의 in-process SMTP 서버에 붙여, 프로덕션 전송자 `Deliver`(기본값, `SetSender` 로 갈아끼우지 말 것)를 통과한 **와이어의 `RCPT TO:` 줄**로 수신자를 단언한다.
  2) 수신자 계약이 표로 고정된다 — 행동한 본인은 빠진다(대소문자 무시, service.go:170) / 같은 주소로 풀리는 둘은 한 번만 간다(소유자가 공유 목록에도 있는 흔한 경우, service.go:193) / 디렉터리가 모르는 식별자는 조용히 빠진다 / `@` 가 든 식별자는 디렉터리 없이 그대로 주소로 쓰인다(service.go:186) / 빈 문자열·공백만 있는 수신자는 빠진다.
  3) 게이트가 고정된다 — `config.Enabled=false` 면 relay 에 아무 연결도 오지 않는다 / `mail.notify_comment=false` 면 `EventComment` 는 안 나가고 다른 이벤트는 나간다 / 키가 아예 없으면 나간다(`Allows` 의 기본 허용, mail.go:65-70) / `SendNow` 는 이벤트 게이트를 **일부러 무시**한다(관리자 테스트 버튼은 알림을 다 꺼도 눌려야 한다) — 이 의도된 차이를 테스트가 말로 적어 못 박는다.
  4) 끊긴 요청 갈래를 판정한다. 이미 취소된 컨텍스트로 `Notify` 를 부르는 테스트를 **먼저 빨강으로 확인**해 본다:
     - **빨강이면** (= relay 에 아무것도 안 옴) service.go:73 `Notify` 첫머리에서 `ctx = context.WithoutCancel(ctx)` 로 떼어내고(같은 파일 `SendNow` 의 선례와 같은 자), 왜 떼는지 — "요청은 끝나도 이미 일어난 변경의 알림은 나가야 한다" — 를 저장소 관례대로 주석에 적는다. 고친 뒤 테스트 초록, 그 한 줄을 되돌리면 다시 빨강인 것까지 확인한다.
     - **초록이면** (= 취소가 이 경로에 영향이 없으면) service.go 는 손대지 말고 1~3 만 남긴다. 회차 노트에 "취소 갈래는 문제 없었다" 를 근거와 함께 적는다.

- 건드릴 파일:
  - `internal/mail/service_test.go` (신규) — 위 표. 패키지 내부 테스트(`package mail`)여야 한다. `settingsProvider`·`directory` 가 비공개 인터페이스라 외부 테스트 패키지에서는 못 만든다.
  - `internal/mail/service.go:73 Notify` — 수용 기준 4가 빨강일 때만 컨텍스트 한 줄 + 주석.
  - (그 외 프로덕션 파일 없음. mail.go·message.go·config.go 는 건드리지 않는다.)

- 검증 명령:
  - `go test ./internal/mail -run 'TestNotify|TestSendNow|TestDeliver' -v` (빨강 → 초록 확인용)
  - `go test ./internal/mail -race` (`Notify` 가 고루틴을 띄우므로 반드시 `-race`)
  - `go test ./...` · `go vet ./...` · `go build ./...` · `gofmt -l ./cmd ./internal ./pkg`
  - `./scripts/check-release-docs.sh` · `./scripts/check-commit-identities.sh HEAD`
  - 웹·DB 통합·브라우저 E2E 는 불필요(Go 서버 안쪽, 스키마 무변경).

- 위험과 피할 것:
  - **`pool` 은 반드시 nil 로 둘 것.** `record`·`complete` 는 `s.pool == nil` 이면 조용히 빠져나가므로(service.go:140,152) DB 없이 돈다. 단 `Deliveries` 는 nil 풀에서 패닉하니 **부르지 말 것** — 그건 `internal/integration` 의 몫이다.
  - **`SetSender` 로 전송을 대체하지 말 것.** 운영자 규칙(손으로 만든 대역으로 증명 금지)이고, 실제 `Deliver`+`fakeRelay` 가 이미 있다. 다만 `directory` 만은 대역이 불가피하다 — 유일한 프로덕션 구현 `internal/httpapi/mail.go:17-32 mailDirectory.LookupEmails` 의 **키 모양을 그대로 흉내 낼 것**: 소문자 userID→email 과 소문자 email→email 을 **둘 다** 넣는다. 한쪽만 넣으면 실제와 다른 계약을 못 박게 된다. 수용 기준 4를 볼 때는 그 대역이 `ctx.Err()` 를 먼저 보게 한다(pgx 가 그렇게 동작한다 — 이 점은 코드로 재확인할 것, **정찰은 미확인**).
  - **비동기 동기화.** `Notify` 는 수신자마다 고루틴을 띄운다(service.go:89). `relay.transcript()` 는 뮤텍스로 보호되니 `RCPT TO` 줄이 n 개 될 때까지 짧게(≤2초) 폴링하는 헬퍼를 쓰고, "안 보낸다" 를 볼 때는 짧게 기다린 뒤 transcript 가 빈 것을 본다. **`relay.body` 는 필드 하나라 마지막 DATA 만 남는다**(mail_test.go:112) — 본문을 볼 때만 수신자 1명짜리 사례로.
  - `deliver` 는 실패하면 2초 자고 한 번 더 보낸다(service.go:127). **실패 경로를 `Notify` 로 테스트하지 말 것** — 테스트가 느려지고 흔들린다. 전송 실패는 이미 mail_test.go 가 `Deliver` 수준에서 덮는다.
  - 보호 경로를 건드리지 말 것: `tlsConfig`·`loginAuth`(mail.go)·`internal/auth`·`migrations`·`.github/workflows`. 테스트의 루프백 허용을 운영 정책으로 옮기지 않는다.
  - 문구·본문은 이번 범위 밖이다(2026-09-27 회차가 본문 이스케이프를 이미 정리했다). 문서 변경 없음 → PDF 재생성 불필요.

- 차선 후보: **관리자 가이드의 외부 호출 오류 코드를 실제와 맞춘다** — `docs/ADMIN_GUIDE.md:785` 는 증상을 `#VALUE!` 로만 적었지만 설정 읽기 실패·`external.enabled` 꺼짐·호출 수 초과·호스트 거절은 모두 `#N/A` 이고(`internal/external/fetcher.go:124,130,136,158,171`) `#VALUE!` 는 CSV 읽기 실패 한 갈래뿐이다(fetcher.go:399). 관리자가 그 줄을 보고 원인을 못 찾는다. 줄 번호는 이전 회차가 확인한 값이므로 **고치기 전에 실제 파일에서 재확인**하고, 고치면 `scripts/generate_pdf.js` 로 ADMIN_GUIDE.pdf 를 다시 구울 것(web/node_modules/playwright-core 필요).
