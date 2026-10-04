# 회차 노트 2026-10-04-164220-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:42] base pinned — main@92ff88c
- [러너 16:42] autonomy release — 
- [러너 16:47] scout done — 메일 본문을 8BITMIME 협상 없이 생 8비트로 보내는 것을 quoted-printable 7비트 인코딩으로 바꾼다 (가치 3 / 위�

## 구현 노트
- `internal/mail/mail.go:compose` 가 8BITMIME 광고를 보지 않고 `Content-Transfer-Encoding: 8bit` + 생 UTF-8 본문을 DATA 에 썼다(RFC 6152 위반). 본문을 `mime/quotedprintable` 로 인코딩해 **한 경로로 언제나 7비트**를 보낸다 — 광고 여부로 분기하지 않았고 `MAIL FROM` 에 `BODY=8BITMIME` 도 붙이지 않았다. 프로덕션 1파일 + 테스트 1파일.
- **확신 없는 곳·검증 못 한 것**: ① 실제 7비트 전용 사내 릴레이에서의 거부·깨짐은 **재현하지 않았다**(그런 릴레이가 없다). 증거는 "광고하지 않는 릴레이에 8비트 바이트를 싣는다"는 프로토콜 위반 자체이고 커밋 메시지에 그렇게 적었다 — 결함 재현이 아니다. ② 실제 Keycloak/SMTP E2E 는 이 저장소에서 여전히 미검증이고 이번에도 못 했다. ③ 수신 클라이언트(Outlook 등)가 QP 본문을 어떻게 보여 주는지는 눈으로 확인하지 못했다 — QP 는 1996년부터의 MIME 기본 인코딩이라 위험은 낮다고 판단했다.
- 일부러 하지 않은 것: `startSession` 에 `Extension("8BITMIME")` 배선(경로가 둘이 되고 8BITMIME 릴레이도 QP 를 받으므로 얻는 것이 없다), `normalizeBody` 변경(CRLF 정규화·끝 CRLF 보장 그대로), 254행 단언 약화(대신 QP 디코딩 기준으로 강화), `openapi`·`web`·`docs`·`VERSION`(메일 바이트는 API 계약이 아니다), 보호 경로(`internal/auth`·`migrations/`·`.github/workflows/`·SMTP 자격증명 정책).
- 다음 역할이 조심할 것: `go test ./internal/mail` 은 발송 예산·동시성 테스트가 실제로 기다려 **8~9초가 정상**이다(느리다고 끊지 말 것). `fakeRelay` 의 `noEightBit` 은 `serve` goroutine 보다 먼저 확정돼야 해서 `startRelayAs(t, host, &fakeRelay{...})` 로 릴레이를 통째로 넘긴다 — 띄운 뒤에 필드를 쓰면 `-race` 가 잡는다. `receivedBody` 헬퍼는 수신 측처럼 줄 머리 점을 떼고 QP 를 되돌리므로, 점 이스케이프를 단언할 때는 **되돌리기 전 바이트**(`\r\n..` 있고 `\r\n...` 없음)와 되돌린 뒤를 둘 다 봐야 한다.
- [러너 16:54] brief accepted — 채택 — `compose`(202행)의 `8bit` 헤더·생 본문과 `startSession`이 STARTTLS·AUTH만 보는 것, `fakeRelay.handle`이 `250 8BITMIME`를 무조건 �
- [러너 16:54] verify failed — 실패한 검증: npm run lint   # CI에서 가져옴 (exit 1)
