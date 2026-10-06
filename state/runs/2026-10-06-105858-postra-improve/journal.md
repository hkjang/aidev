# 회차 노트 2026-10-06-105858-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:59] base pinned — main@bf290e8
- [러너 10:59] autonomy release — 

## 정찰 노트
- 고른 이유: 수신 어댑터의 implicit TLS(계정 기본 보안)에 인증서 검증 테스트가 양쪽 0건임을 grep 으로 실측했고(pop3/imap client_test.go 에 domain.SecurityTLS 0건, 모든 TLS 테스트가 SecurityStartTLS+InsecureSkipVerify:true), 발신 SMTP 에는 같은 보증이 이미 있어(smtp/client_test.go:443) 비대칭이 분명했다. 프로덕션 변경 0 + 기존 selfSigned 픽스처 재사용이라 위험이 가장 낮다.
- 제친 후보: POP3 retrBody 상한(가치 4)은 PR #22 사람 반려 여부가 또 미확인이라 규칙대로 제외. App.tsx 동적 import 는 모든 청크 해시가 바뀌어 자산 재생성이 따라붙는다. jsdom isolate:false 와 "알 수 없는 Security 값" 두 건은 각각 이득 미측정·관찰 사례 없음으로 rejected 로 닫았다.
- 추측으로 적은 것(미확인): IMAP 성공 쪽 하위 테스트의 픽스처 모양 — selectInbox() 는 Username 이 비어도 무조건 호출되므로 SELECT 응답을 답해 주거나 PREAUTH greeting 을 써야 하는데 실제로 돌려 보지 않았다. 과제서에 "한 번 돌려 보고 맞출 것" 으로 명시했고, 길어지면 IMAP 을 떼고 POP3 만 담으라고 적었다.
- 확인한 것: Go 의 implicit TLS 자가서명 실패는 ClassifyInbound(inbound_error.go:77)에서 tls_certificate 로 분류되고 connectFailure 가 그것을 StageTLS 로 올린다 — 수용 기준 1 은 코드상 성립한다. tls_certificate 레이블은 syncClassLabels:144 에 이미 있어 새 레이블이 필요 없다.
- 구현자가 조심할 것: 프로덕션 코드 0줄을 지킬 것(이번 회차는 보증 추가다). #nosec G402 주석과 InsecureSkipVerify 의 opt-in 의미는 의도된 설계이고, 바로 옆 STARTTLS 주입 가드는 지난 회차가 건드린 보안 판정이다. 변이 검증으로 InsecureSkipVerify 를 true 로 고정해 봤다면 반드시 원복하고 git diff --stat 으로 프로덕션 0개를 재확인할 것.
- [러너 11:03] scout done — POP3·IMAP **implicit TLS** 의 인증서 검증 기본값과 `connectFailure` 단계 매핑을 회귀 테스트로 못 박는다 (가치 3 

## 구현 노트
- 무엇·왜: 수신 어댑터 양쪽(`pop3`/`imap`)에 implicit TLS 인증서 검증 테스트 1개씩 추가(커밋 97a5b3e). 계정 기본 보안이 implicit TLS 인데 그 경로에 TLS 검증 테스트가 0건이었고, `connectFailure` 의 `tls_certificate`→`StageTLS` 분기도 미검증이었다. **프로덕션 코드 0줄** — 테스트 2파일만.
- 확신 없는 곳·검증 못 한 것: (1) `MinVersion: tls.VersionTLS12` 는 **묶지 못했다** — Go 의 클라이언트 기본 최소가 이미 TLS 1.2 라 그 줄을 지워도 테스트가 통과한다. 주석에 그렇게 명시하고 ideas.json 에 별 항목으로 남겼다(과제서 문구를 그대로 믿지 말 것). (2) 외부 PostgreSQL(`POSTRA_TEST_PG`)·브라우저 e2e·프런트 빌드는 돌리지 않았다 — Go 테스트 전용 변경이라 범위 밖. (3) 에러 문자열 단언은 Go 의 x509 문구에 의존한다(`"x509:"` 부분 문자열) — Go 툴체인이 그 접두사를 바꾸면 깨질 수 있다. 전체 문장이 아니라 접두사만 봐서 취약성은 낮게 잡았다.
- 일부러 하지 않은 것: `#nosec G402`·계정별 `InsecureSkipVerify` opt-in·STARTTLS 주입 가드·`connectFailure` 의 `switch`·`ClassifyInbound` 순서 — 전부 의도된 설계이자 보안 판정이라 미접촉. 새 Stage/Class 레이블 없음, `domain.SyncDiagnostic` 필드 추가 없음(계약·web·spa/assets 로 번지므로). `web/` 아래 새 파일 0개.
- 다음 역할이 조심할 것: 두 테스트는 **같은 리스너**에 두 번 Dial 하므로 픽스처가 accept 루프다 — 하위 테스트를 `t.Parallel()` 로 바꾸면 의미가 흔들린다(한 필드만 다른 비교가 핵심). DB·네트워크 외부 의존 없이 루프백만 쓰므로 어디서나 돈다. `-count=3` 으로 양쪽 플레이크 0 확인(13.4s / 11.4s). 과제서가 쓴 `domain.IMAPDialOptions` 는 존재하지 않는다 — 실제 타입은 `domain.InboundDialOptions`(`POP3DialOptions` 별칭).
- [러너 11:11] brief accepted — 채택 — 근거가 전부 코드와 맞았다(`domain.SecurityTLS` 가 양쪽 테스트 파일에 0건, 두 어댑터의 `InsecureSkipVerify` 배선, `connect
- [러너 11:12] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 확인한 것: 변이 2회로 테스트가 실제로 문다는 증명 — `InsecureSkipVerify` 를 상수 true 로 고정(양쪽 FAIL), `connectFailure` 에서 `tls_certificate` 분기 제거(양쪽 Stage=tcp_connect FAIL). 변이 원복 후 트리 깨끗, 프로덕션 0줄 재확인.
- 주석의 구체 주장 전부 코드 대조 일치(accounts.go:62 기본값, domain/account.go:42 의 수신 보안 필드 공유, smtp/client_test.go:443, imap client.go:137/159, 줄번호 69/54). 픽스처 고루틴이 `t.*` 를 안 써 종료 후 패닉 경로 없음. `-race -count=3` 플레이크 0, 두 패키지 전체 `-race` 통과, `gofmt -l` 깨끗, gosec 은 테스트 파일 미검사.
- 못 본 것: `POSTRA_TEST_PG` PG 검사·브라우저 e2e·프런트 빌드·저장소 전체 `go test ./...`. Go 테스트 전용 변경이라 범위 밖으로 뒀다. `gh` 미인증이라 PR 상태 미확인.
- 릴리즈 노트가 조심할 것: **`MinVersion: tls.VersionTLS12` 는 묶이지 않았다** — 양쪽 어댑터에서 그 줄을 지워도 전 패키지가 통과한다. 노트에 "TLS 하한을 고정했다" 류로 쓰면 거짓이 된다. 이번 보증은 "implicit TLS 의 인증서 검증 기본값 + tls_certificate→StageTLS 단계 매핑" 딱 둘이다.
- 다음 회차가 알아야 할 것: 에러 단언이 Go 의 `"x509:"` 접두사에 의존(툴체인 문구 변경 시 깨질 수 있음), 주석에 박힌 `client.go:69`/`client.go:54` 줄번호는 해당 파일 수정 시 썩는다. MinVersion 고정 테스트는 독립 후보로 남아 있다.
- [러너 11:15] review approved — 리뷰 승인 (risk=low)
- [러너 11:16] pr created — https://github.com/hkjang/postra/pull/37
- [러너 11:21] ci passed — 검사 10개 모두 success
- [러너 11:21] merge done — 97a5b3e
