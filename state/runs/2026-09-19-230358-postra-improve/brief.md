# 과제서 — 2026-09-19 postra

- 과제: smtp 어댑터 단위/통합 테스트 추가 — net.Listen 스크립트 서버로 AUTH·오류 분류·STARTTLS 미광고·auto 무인증 검증 (가치 3 / 위험 1 / 작업량 M)

- 왜: `internal/adapters/smtp/client.go`(247줄)는 모든 실제 발송·알림 메일이 지나가는 유일한 SMTP 경로인데 `go test ./internal/adapters/smtp/` 가 `[no test files]` 다. 지금 유일한 덮개는 `internal/application/smtp_integration_test.go` 하나뿐이고 그것은 `auth=none`·평문·DATA 수락/끊김 두 경우만 본다. AUTH PLAIN/LOGIN 선택, `AuthError` 영구 분류, 4xx 임시/5xx 영구 분류, STARTTLS 미광고 오류, `auto` 모드 무인증 진행은 한 번도 실행된 적이 없어 어댑터를 손댈 때(예: 보류 중인 notifymail 의 OpportunisticTLS 옵션) 회귀를 잡을 수단이 없다.

- 수용 기준:
  1) `internal/adapters/smtp/client_test.go` 가 새로 생기고 `go test -race ./internal/adapters/smtp/` 가 통과한다(테스트 파일만 추가, `client.go` 는 바꾸지 않는다 — 바꿔야 통과한다면 그 사실을 회차 노트에 적고 멈춘다).
  2) 다음 시나리오가 각각 **실제 `Client{}.Send`/`TestConnection` 을 127.0.0.1 의 net.Listen 스크립트 서버에 붙여** 검증된다(대역·Fake 클라이언트 금지, 실제 net/smtp 왕복):
     a. `AuthMethod:"auto"` + Username/Password 있음 + 서버가 `250-AUTH PLAIN` 광고 → 서버가 받은 AUTH 줄이 `AUTH PLAIN <base64("\x00user\x00pass")>` 이고 발송 성공(`SendReceipt.ServerResponse == "250 accepted"`, `Uncertain == false`).
     b. 서버가 `250-AUTH LOGIN` 만 광고 → `AUTH LOGIN` 뒤 `334 VXNlcm5hbWU6`/`334 UGFzc3dvcmQ6` 챌린지에 base64 사용자명/비밀번호가 차례로 온다(`loginAuth.Next` 는 서버가 보낸 `Username:`/`Password:` 를 디코드해 비교하므로 서버는 base64 디코드된 챌린지 텍스트를 보내야 함 — net/smtp 가 디코드해서 넘김).
     c. 서버가 AUTH 에 `535 5.7.8 bad credentials` → `Send` 오류가 `errors.As(&*AuthError)` 이고 `*SendError.Temporary() == false`.
     d. `AuthMethod:"auto"` + Username 있음 + 서버가 AUTH 를 **광고하지 않음** → AUTH 명령이 한 번도 오지 않고 발송 성공(무인증 진행).
     e. `AuthMethod:"login"`(auto·none 이 아닌 값) + 서버가 AUTH 미광고 → 오류 메시지에 `does not offer AUTH` 포함, `Temporary() == true`(connect 단계 오류는 AuthError 가 아니면 임시).
     f. `RCPT TO` 에 `450 4.2.0 mailbox busy` → `Temporary() == true`; `550 5.1.1 no such user` → `Temporary() == false`; 오류 문자열에 `RCPT TO <주소>` 접두 포함. `MAIL FROM` 5xx 도 영구 한 건.
     g. `Security: domain.SecurityStartTLS` + 서버 EHLO 응답에 STARTTLS 없음 → 오류 메시지에 `does not offer STARTTLS`, MAIL 명령은 서버에 도착하지 않음, `Temporary() == true`.
     h. `TestConnection` 은 (a) 서버 기준으로 `Steps` 가 `dns`·`smtp_ehlo_auth` 둘 다 OK 이고 `diag.OK == true`; (c) 서버 기준으로 `smtp_ehlo_auth` 단계 `OK=false`·`Detail` 에 `smtp auth` 포함, 반환 err 는 nil(진단은 오류를 값으로 돌려줌).
     i. `Send` 가 끝난 뒤 `opts.Password.Reveal()` 이 비어 있다(defer Zero — `domain.SecretHandle.Zero` 동작 확인; 실제 시그니처는 `internal/domain/secret.go`(미확인 파일명 — `grep -rn "func (h \*SecretHandle) Zero"` 로 찾을 것)).
  3) 테스트는 서버 측에서 받은 명령 줄을 슬라이스로 모아 두고 단언한다(`AUTH` 줄 존재/부재, `STARTTLS` 부재, `MAIL FROM` 도착 여부). 각 시나리오는 `t.Run` 하위 테스트, 서버는 `t.Cleanup` 으로 닫고 `conn.SetDeadline(5s)` 를 둔다. `-race` 에서 통과.

- 건드릴 파일:
  - `internal/adapters/smtp/client_test.go` (신규, `package smtp` 내부 테스트 — `loginAuth`·`classify`·`AuthError` 에 직접 접근 가능) — 스크립트 서버 헬퍼 `fakeRelay(t, relayOpts) (port int, got *recorded)` 하나를 두고 시나리오별 옵션(광고할 EHLO 확장 목록, AUTH 응답 코드, RCPT/MAIL 응답 코드, DATA 뒤 끊기 여부)만 바꾼다. 모델은 `internal/application/smtp_integration_test.go:30-87` 의 `textproto.NewConn` 루프와 `internal/adapters/imap/client_test.go:20` 의 `fakeServer` 스타일.
  - 참고만(수정 금지): `internal/adapters/smtp/client.go` — `connect`(39)·`authenticate`(79)·`classify`(130)·`loginAuth`(140)·`TestConnection`(159)·`Send`(182). `internal/domain/account.go:161` `SMTPSendOptions`(`Security` 는 `domain.SecurityTLS`/`SecurityStartTLS`/평문 상수 — 정확한 이름은 account.go 에서 확인), `domain.NewSecretHandle([]byte)`(`internal/domain` 에 있음, 파일명 미확인).

- 검증 명령:
  - `go test -race -count=1 ./internal/adapters/smtp/` (신규 테스트)
  - `go test -race ./internal/application/ -run TestSMTPIntegration` (기존 통합 테스트 불변 확인)
  - `gofmt -l ./internal/adapters/smtp/` 가 빈 출력, `go vet ./internal/adapters/smtp/`
  - 마지막에 `go build ./... && go test -race ./...` (전체는 수 분 걸림)

- 위험과 피할 것:
  - **`client.go` 를 고치지 말 것.** 이 과제는 테스트 공백 보강이다. 테스트를 쓰다 어댑터 결함을 발견하면 테스트를 `t.Skip` 하지 말고 그 시나리오를 빼고 회차 노트에 결함을 적는다(수정은 다음 회차의 과제로).
  - 운영자 규칙: 대역·FakeTask 로 증명하지 말 것 — 여기서는 실제 `net/smtp.Client` 가 실제 TCP 소켓으로 스크립트 서버와 대화해야 한다. `smtp.Auth` 인터페이스만 단독 호출하는 단위 테스트(`loginAuth.Next` 직접 호출)는 보조로는 좋지만 위 시나리오의 대체가 될 수 없다.
  - 실제 STARTTLS 핸드셰이크(광고 시 TLS 전환)와 `SecurityTLS`(465) 는 자체서명 인증서 생성(crypto/x509 + ecdsa, `InsecureSkipVerify: true`)이 필요하다 — 저장소 테스트에 그런 헬퍼가 아직 없다(grep 으로 확인). 시간이 남으면 `tls.Listen` + 자체서명으로 (g) 의 짝인 "광고 시 STARTTLS 후 MAIL 이 TLS 위에서 도착" 한 건을 더하되, 없어도 수용 기준은 충족된다.
  - `TestConnection` 은 `net.DefaultResolver.LookupHost(opts.Host)` 를 먼저 부른다 — `Host` 를 `"127.0.0.1"` 로 두면 DNS 조회 없이 통과한다(`localhost` 는 환경에 따라 ::1 을 돌려주므로 피할 것). `dialTLSConfig.ServerName` 도 Host 라 평문 시나리오에선 무관.
  - AUTH LOGIN 챌린지: net/smtp 는 `334 <base64>` 를 디코드해 `Next(fromServer)` 로 넘기므로 서버는 `334 VXNlcm5hbWU6`(= "Username:") 를 보내야 `loginAuth` 가 응답한다. 평문 `334 Username:` 을 보내면 디코드 실패로 다른 오류가 난다 — 그건 어댑터 결함이 아니다.
  - net/smtp 의 `PlainAuth` 는 TLS 가 아니고 서버가 localhost 가 아니면 "unencrypted connection" 오류를 낸다. 서버 이름은 `opts.Host`(=127.0.0.1) 로 넘어가는데 net/smtp 는 `ServerInfo.Name` 이 "localhost"/"127.0.0.1"/"::1" 이면 허용한다 — 그래서 Host 는 반드시 `127.0.0.1` 로 둔다(이 조건이 안 맞으면 (a) 가 실패하는데 어댑터 결함이 아니라 테스트 설정 문제다).
  - 보호 경로(auth·migrations·workflows) 는 건드리지 않는다. CI 는 `go test -race -covermode=atomic ./...` 를 돌리므로 새 테스트가 5초 안에 끝나게 데드라인을 짧게 둔다.

- 차선 후보: CI `build-test` 잡의 `Vet` 앞에 `gofmt -l` 검사 한 단계 추가(`.github/workflows/ci.yml:123`, `test -z "$(gofmt -l ./cmd ./internal)"`; 이 회차에서 gofmt 상태는 권한 제한으로 미확인 — 구현 전 로컬에서 `gofmt -l ./cmd ./internal` 이 빈 출력임을 먼저 확인하고, 더럽다면 그 파일들을 gofmt 로 정리하는 것까지 포함) (가치 2 / 위험 1 / S).
