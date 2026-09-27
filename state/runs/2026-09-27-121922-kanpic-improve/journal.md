# 회차 노트 2026-09-27-121922-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:19] base pinned — main@0077205
- [러너 12:19] autonomy release — 

## 정찰 노트
- 숫자 파서 계약은 다섯 회차 연속 캤고 남은 항목은 계약을 먼저 못 박아야 하는 M 들뿐이어서(compareLists 007.5, 1e3 지수) 이번엔 아직 안 건드린 `internal/mail` 로 옮겼다. 프로필에 없던 패키지 6개(ai·analytics·automation·mail·observability·presentation)를 발견해 프로필을 새로 썼다.
- 고른 것: `normalizeBody` 의 손 점-이스케이프가 `client.Data()` 의 net/textproto dot writer 와 겹쳐 이중 이스케이프일 가능성. 확실한 것은 **기존 테스트 mail_test.go:151 의 `Contains(body, "..점으로 시작")` 이 `...점으로 시작` 도 통과시켜 두 갈래를 가려낼 수 없다**는 것 — 이건 코드를 직접 읽고 확인했다.
- **확신 없는 곳**: dot writer 가 실제로 줄 앞 점을 이스케이프하는지 이번에 확인 못 했다(샌드박스가 `go run`·`find /` 를 막았고 예산이 남지 않았다). 그래서 과제서를 두 갈래로 썼다 — 빨강이면 두 줄 제거, 초록이면 프로덕션 코드는 옳으니 손대지 말고 단언만 정확 비교로 바꿔 끝낸다. 어느 쪽이든 성과가 남는다.
- 구현자가 조심할 것: `compose()` 반환 문자열만 보는 단위 테스트로는 결함이 원리적으로 안 보인다(문제는 compose 와 Data() 의 합). 이미 있는 `fakeRelay` 로 와이어 바이트를 봐야 하고 새 대역을 만들지 말 것. TLS/AUTH·`service.go` 수신자 결정·본문 문구는 건드리지 말 것.
- [러너 12:24] scout done — 알림 메일 본문의 점 이스케이프가 SMTP 와이어에서 한 번만 일어나는지 못 박는다 (가치 3 / 위험 2 / 작업�

## 구현 노트
- 무엇을 왜: `normalizeBody` 가 줄 앞 점을 손으로 이스케이프하고 `client.Data()` 의 net/textproto dot writer 가 또 해서 받는 사람이 `..점으로 시작` 을 보고 있었다. 손 이스케이프 두 줄을 지웠다(CRLF 정규화·끝 `\r\n` 은 남김). 정찰이 미확인으로 둔 dot writer 동작은 **이스케이프한다** 로 판정됐다 — 과제서의 '빨강' 갈래.
- 증거: `fakeRelay` 에 실제로 실린 DATA 를 받는 쪽처럼 되돌려(`undotWireBody`) `Message.Body` 와 정확히 비교한다. 고치기 전 5개 빨강(`...점으로 시작` 이 와이어에 있었다), 고친 뒤 9개 초록, 지운 줄 하나를 되살리니 다시 5개 빨강.
- 느슨한 단언도 교체: mail_test.go:151 의 `Contains(body, "..점으로 시작")` 은 `...점으로 시작` 도 통과시켰다. 이제 같은 정확 비교를 쓴다.
- 확신 없는 곳: 없음 — 실제 net/smtp 배선과 와이어 바이트로 확인했다. 다만 이 변경은 **받는 사람 화면에 보이는 문자열을 바꾼다**(점이 하나 줄어든다). 그것이 의도다.
- 일부러 안 한 것: TLS/STARTTLS/AUTH·`config.tlsConfig()`·`service.go` 수신자 결정·본문 문구는 손대지 않았다. `To:` 헤더 인코딩은 범위 밖(ideas.json 에 남김). 문서는 와이어 수준 점 이스케이프를 말한 적이 없어(`docs/*.md` grep 무소득) PDF 재생성 없음.
- 다음 역할이 조심할 것: 프로덕션 변경은 `internal/mail/message.go` 한 파일·두 줄 제거뿐. `compose` 는 부르는 곳이 `Deliver` 하나뿐이고 언제나 dot writer 로 나가는 것을 확인했으니, 나중에 `compose` 를 dot writer 를 안 거치는 곳에서 부르면 이스케이프가 사라진다 — 그때는 그 경로에서 이스케이프해야 한다. 이 테스트들은 DB·네트워크 없이 돈다(루프백 in-process relay).
- [러너 12:28] brief accepted — 채택 — 과제서가 두 갈래로 써 둔 것 중 '빨강' 갈래가 그대로 맞았고(dot writer 는 이스케이프한다), 건드릴 파일 2개·`fake
- [러너 12:28] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 이중 이스케이프 주장을 go1.26.7 `net/textproto/writer.go:74` dotWriter.Write(줄 앞 `.` 에 점 추가)와 `net/smtp` 의 `Data()` 로 1차 확인 — 손 이스케이프 제거가 옳다. `compose` 호출자는 `mail.go:123` 하나뿐이다.
- 실패 재현을 직접 했다: HEAD 테스트 + main 의 `message.go` 로 5빨강(와이어에 `...점으로 시작`), 되돌리기를 풀면 9초록. 구현 노트와 일치한다. `undotWireBody` 의 경계(fakeRelay 가 종료 줄 `.` 을 body 에서 뺀다, TrimPrefix 가 점 하나만 뗀다)도 봤고 `..이미 두 점` 케이스가 이중 이스케이프를 여전히 잡는다. 본문 첫/마지막 줄이 `.` 하나여도 DATA 조기 종료 없음.
- 못 본 것: 웹·DB 통합·E2E(이 변경과 무관). 실제 외부 MTA 로의 발송.
- 승인이어도 남는 우려 — 릴리즈 노트: 이 변경은 **받는 사람 화면의 문자열을 바꾼다**(점 하나 줄어든다). `docs/*.md` 에 와이어 점 이스케이프 언급이 없어 가이드·PDF 갱신은 불필요.
- 다음 회차: `normalizeBody` 가 더 이상 스스로를 보호하지 않으니 dot writer 를 안 거치는 경로에서 `compose` 를 부르면 그 경로에서 이스케이프해야 한다(주석에만 적혀 있고 코드가 강제하지 않는다). 보안·법무 차단 소견 없음 — `To:` 무인코딩은 `net/smtp` validateLine 이 RCPT 에서 막고 이번 diff 밖이다.
- [러너 12:31] review approved — 리뷰 승인 (risk=low)
- [러너 12:31] pr created — https://github.com/hkjang/kanpic/pull/34
- [러너 12:39] ci passed — 검사 2개 모두 success
- [러너 12:39] merge done — c0ab986
