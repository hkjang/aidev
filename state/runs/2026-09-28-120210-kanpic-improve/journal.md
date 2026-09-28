# 회차 노트 2026-09-28-120210-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:02] base pinned — main@3e38232
- [러너 12:02] autonomy release — 

## 정찰 노트
- 보류 목록의 "To 헤더 인코딩" 을 코드로 좁히다 진짜 결함을 찾았다 — 인코딩 갈래는 `client.Rcpt` 가 표시 이름을 먼저 거절하므로 죽은 코드라 rejected 로 내리고(ideas.json), 살아 있던 것은 **봉투와 헤더가 같은 주소를 다르게 읽는 것**이었다. 운영자가 되풀이한 "같은 값을 읽는 경로가 둘이면 둘을 함께 본다" 에 그대로 걸린다.
- 추측이 아니다: 임시 테스트를 `internal/mail` 에 넣고 기존 `startRelay` 로 프로덕션 `Deliver` 를 통과시켜 헤더 블록이 `To:` 에서 끊기고 Subject 이하 일곱 줄이 본문이 되는 것을 와이어에서 봤다(브리프에 출력 그대로 붙였다). 임시 파일은 지웠고 `git status` 는 깨끗하다.
- 차선(ADMIN_GUIDE `#N/A`)을 1순위로 올리지 않은 이유: 다섯 회차 연속 보류인데 매번 PDF 재생성에 걸렸고 문서만 바뀌어 가치가 낮다. 줄 번호 785·fetcher.go 줄 번호는 이번에 재확인하지 않았다(미확인) — 구현자가 고치기 전에 다시 볼 것.
- 구현자가 조심할 것: `Message` 는 값으로 받으므로 `Deliver` 안에서 `message.To` 를 갱신해도 호출자에 새지 않는다(의도한 바). `service.go` 는 지난 회차가 막 고쳤으니 건드리지 말고, `normalizeBody` 의 점 이스케이프 주석(c0ab986)도 되돌리지 말 것.
- [러너 12:06] scout done — 메일의 "누구에게" 를 봉투와 헤더가 같은 값으로 읽게 한다 — 지금 `Rcpt` 는 trim 하고 `To:` 헤더는 날것을 

## 구현 노트
- `Deliver` 가 수신자를 한 곳에서만 정규화하게 해 봉투(`client.Rcpt`)와 `To:` 헤더가 같은 값을 쓰게 했다. 끝에 CRLF 가 붙은 주소에서 헤더 블록이 To 에서 끝나 Subject 이하 일곱 줄이 본문이 되던 결함이다. 주소 가운데 CR/LF 는 다듬을 수 없으므로 연결 전에 `ErrInvalid`(한국어)로 거절한다.
- 확신 없는 곳: 새 `ErrInvalid` 거절로 **전에는 릴레이가 받아 주던 주소가 이제 막힐 수 있다** — 다만 막히는 것은 주소 *가운데* CR/LF 뿐이고 그건 전에도 `client.Rcpt` 가 `smtp: A line must not contain CR or LF` 로 막던 값이라 새로 깨지는 정상 입력은 없다고 본다(문구와 오류 종류만 바뀐다). 호출부(`service.go:99,118`)가 `ErrInvalid` 를 다른 오류와 다르게 다루지 않는 것은 코드로 확인했다.
- 검증 못 한 것: 실제 SMTP 릴레이(테스트는 in-process 가짜 relay), 웹·DB 통합·브라우저 E2E — Go 서버 안쪽 변경이라 돌리지 않았다.
- 일부러 하지 않은 것: 표시 이름(`홍길동 <a@b>`) 지원과 `To` 헤더 `mime.QEncoding` 인코딩 — 프로덕션 `Message.To` 는 언제나 맨 주소라 죽은 코드가 된다(ideas.json 에 rejected 로 적음). `service.go`·TLS/AUTH·`normalizeBody` 는 손대지 않았다.
- 다음 역할이 조심할 것: 이 테스트들은 외부 의존이 없고 `-race -count=2` 까지 돈다. `relay.transcript()` 가 비어 있음을 '연결조차 없었다' 의 증거로 쓰므로, 가짜 relay 가 나중에 연결 자체를 기록하게 바뀌면 거절 테스트를 함께 고쳐야 한다. 감사에 남는 `Delivery.Recipient` 는 `Deliver` 의 정규화와 별개 값이라 `@` 든 식별자 갈래에서 갈릴 수 있다(ideas.json 에 새 후보로 적음).
- [러너 12:10] brief accepted — 채택 — 정찰의 와이어 재현이 지금 코드와 정확히 맞았고, 건드릴 파일 2개·`startRelay` 재사용·`SetSender` 금지·표시 이름 
- [러너 12:10] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: main 의 mail.go·message.go 만 되돌려 새 테스트를 돌려 부속 6개가 빨강임을 직접 봤다(원장의 실패 재현과 일치) — 되돌린 파일은 HEAD 로 복구, git status 깨끗. go test ./... · -race -count=2 · vet · build · gofmt 전부 통과.
- 못 본 것: 웹 테스트·DB 통합·브라우저 E2E·실제 SMTP 릴레이. 변경이 internal/mail 3파일에 갇혀 있어(웹·SQL·워크플로 무변경) 감수했다.
- 승인이어도 남는 우려: (1) 거절 테스트의 `transcript()` 비어 있음은 'SMTP 명령 0건'까지만 증명한다 — 검사가 dial 뒤로 옮겨져도 초록이다. (2) FromName 이 비면 encodeAddress 가 FromAddress 를 날것으로 From 헤더에 붙인다(message.go:31, validate 는 @ 만 본다) — 관리자 설정값이라 차단은 아니지만 같은 꼴의 다음 후보. (3) 감사 표 Delivery.Recipient 는 여전히 별개 정본(service.go:95,113).
- 릴리즈 노트: 관리자가 보는 오류가 영어 `smtp: A line must not contain CR or LF` 에서 한국어 문구·ErrInvalid 로 바뀐다.
- [러너 12:14] review approved — 리뷰 승인 (risk=low)
- [러너 12:14] pr created — https://github.com/hkjang/kanpic/pull/36
- [러너 12:21] ci passed — 검사 2개 모두 success
- [러너 12:21] merge done — 73465c6
- [러너 12:33] release published — v0.257.0
