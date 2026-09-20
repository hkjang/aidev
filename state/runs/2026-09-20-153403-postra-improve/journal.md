# 회차 노트 2026-09-20-153403-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:34] base pinned — main@0356555
- [러너 15:34] autonomy release — 

## 정찰 노트
- 선택 이유: SMTP 어댑터가 ctx·ConnectTimeout 을 dial 에만 쓰고 그 뒤 왕복은 무기한이라, 직렬·무타임아웃인 아웃박스 워커(`scheduler.go:90~117`)가 릴레이 한 대의 침묵으로 통째로 멈추는 실제 결함이다. IMAP/POP3 는 이미 SetDeadline 패턴이 있고 smtp `client_test.go`(bf4e5a9) 픽스처가 main 에 있어 침묵 릴레이를 실제 net.Listen 으로 증명할 수 있다. Makefile lint 타깃(차선)은 가치는 있으나 코드 동작이 안 바뀌어 2순위로 뒀고, notifymail/handoff 계열은 여전히 main 에 없어 제외.
- 확신 없는 곳: (1) `context.AfterFunc` 로 raw conn 을 닫았을 때 `net/smtp` 가 돌려주는 오류 문자열/타입 — `classify` 는 non-textproto 를 temp 로 분류하니 결과는 임시지만 정확한 메시지는 미확인; (2) StartTLS 뒤 raw conn 에 건 SetDeadline 이 tls.Conn 읽기에 적용된다는 것은 표준 라이브러리 위임 동작에 근거한 추측(TestSendSTARTTLS 통과로 확인할 것); (3) `postra-contracts -check` 가 SMTPSendOptions 변화에 반응하지 않는다는 것도 미확인; (4) ADMIN_GUIDE 에 sync.command_timeout_sec 설명 절이 있는지 미확인.
- 구현자가 조심할 것: 데드라인은 총량이 아니라 명령당으로; send.go·accounts.go 두 호출처 모두 배선; `Uncertain` 규칙·AuthError 영구 분류·Password.Zero 를 바꾸지 말 것; 테스트 hang 은 `t.Cleanup` 으로 반드시 풀어 goroutine 누수·-race 타임아웃을 막을 것.
- [러너 15:38] scout done — SMTP `Send`/`TestConnection` 에 명령 시간제한(`CommandTimeoutSec`)과 ctx 취소를 dial 이후 단계(EHLO·STARTTLS·AUTH·MAIL·RC

## 구현 노트
- 무엇/왜: `smtp/client.go` 에 `session`(raw conn + 명령당 `SetDeadline` + `context.AfterFunc(ctx, conn.Close)`) 을 넣어 dial 이후 EHLO~QUIT 전부에 `CommandTimeoutSec`(기본 60초)·ctx 취소가 걸리게 했고, `send.go`/`accounts.go` 두 호출처에 `Sync.CommandTimeoutSec` 를 배선했다(b8c46d1). 수정 전 `hangAfter:"MAIL"` 릴레이로 `Send` 가 5초 ctx 에도 영원히 걸리는 것을 재현한 뒤 고쳤다.
- 확신 없는 곳: (1) ctx 취소·데드라인 테스트는 평문 레그만 — STARTTLS 뒤에는 raw conn 에 건 데드라인/Close 가 `tls.Conn` 읽기를 푼다는 표준 라이브러리 동작에 기대며 기존 `TestSendSTARTTLS` 가 통과하는 것까지만 확인. (2) `deadlineWriter` 는 32KiB 청크마다 갱신하지만 `DotWriter` 내부 bufio 가 flush 하는 시점은 청크 경계와 어긋날 수 있어 "느린 릴레이에서 큰 첨부" 는 실제 느린 링크로 검증하지 못했다(3MB 루프백 온전 도착만 확인).
- 일부러 하지 않은 것: `isConnectionLost` 무수정(timeout 은 이미 lost), `Uncertain`·`AuthError` 영구·`Password.Zero` 의미 불변, 새 설정 키 없음, ADMIN_GUIDE 는 `command_timeout_sec` 절 자체가 없어 손대지 않음, 워커 `ProcessRetries` 항목당 타임아웃은 별도 항목으로 보류.
- 과제서와 다른 점: `TestConnection` 단계 이름은 코드의 `smtp_ehlo_auth`(과제서의 `connect_tls_auth` 아님); 제안된 변이("Mail 앞 deadline() 만 제거")는 직전 명령의 절대 데드라인이 남아 통과해 버리므로 `deadline()` 전체 무효화(침묵 MAIL 테스트가 5초에서 실패)와 `AfterFunc` 제거(취소 테스트 무한 대기)로 증명.
- 다음 역할이 조심할 것: 새 테스트 그룹 `TestSendCommandTimeout` 은 릴레이를 `t.Cleanup` 까지 침묵시키므로 합계 ≈3초 걸리는 게 정상; gosec 은 PATH 에 없어 `GOBIN=/tmp/gobin go install ...@v2.28.0` 으로 설치해 Issues 0 확인; 프런트·`spa/assets` 미변경(npm 단계 불필요).
- [러너 15:49] brief accepted — 채택 — 근거(dial 에만 ctx·타임아웃, 직렬 무타임아웃 워커, POP3 `SetDeadline` 패턴, 기존 fakeRelay 픽스처)가 모두 코드와 맞�
- [러너 15:50] verify passed — 검증 9개 통과 (auto)
- [러너 15:50] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 15:50] pr created — https://github.com/hkjang/postra/pull/18
