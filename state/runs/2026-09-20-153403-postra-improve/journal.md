# 회차 노트 2026-09-20-153403-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:34] base pinned — main@0356555
- [러너 15:34] autonomy release — 

## 정찰 노트
- 선택 이유: SMTP 어댑터가 ctx·ConnectTimeout 을 dial 에만 쓰고 그 뒤 왕복은 무기한이라, 직렬·무타임아웃인 아웃박스 워커(`scheduler.go:90~117`)가 릴레이 한 대의 침묵으로 통째로 멈추는 실제 결함이다. IMAP/POP3 는 이미 SetDeadline 패턴이 있고 smtp `client_test.go`(bf4e5a9) 픽스처가 main 에 있어 침묵 릴레이를 실제 net.Listen 으로 증명할 수 있다. Makefile lint 타깃(차선)은 가치는 있으나 코드 동작이 안 바뀌어 2순위로 뒀고, notifymail/handoff 계열은 여전히 main 에 없어 제외.
- 확신 없는 곳: (1) `context.AfterFunc` 로 raw conn 을 닫았을 때 `net/smtp` 가 돌려주는 오류 문자열/타입 — `classify` 는 non-textproto 를 temp 로 분류하니 결과는 임시지만 정확한 메시지는 미확인; (2) StartTLS 뒤 raw conn 에 건 SetDeadline 이 tls.Conn 읽기에 적용된다는 것은 표준 라이브러리 위임 동작에 근거한 추측(TestSendSTARTTLS 통과로 확인할 것); (3) `postra-contracts -check` 가 SMTPSendOptions 변화에 반응하지 않는다는 것도 미확인; (4) ADMIN_GUIDE 에 sync.command_timeout_sec 설명 절이 있는지 미확인.
- 구현자가 조심할 것: 데드라인은 총량이 아니라 명령당으로; send.go·accounts.go 두 호출처 모두 배선; `Uncertain` 규칙·AuthError 영구 분류·Password.Zero 를 바꾸지 말 것; 테스트 hang 은 `t.Cleanup` 으로 반드시 풀어 goroutine 누수·-race 타임아웃을 막을 것.
- [러너 15:38] scout done — SMTP `Send`/`TestConnection` 에 명령 시간제한(`CommandTimeoutSec`)과 ctx 취소를 dial 이후 단계(EHLO·STARTTLS·AUTH·MAIL·RC
