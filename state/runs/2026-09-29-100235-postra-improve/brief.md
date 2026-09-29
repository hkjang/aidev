- 과제: POP3 인증 실패·인사말 거부·본문 dot-unstuffing을 실제 TCP 회귀 테스트로 고정 (가치 3 / 위험 1 / 작업량 S)
- 왜: 현재 POP3 테스트 5개는 누적 상한과 STLS를 검증하지만, USER/PASS 거부의 AuthError 계약과 Retrieve/Top의 본문 프레이밍에는 회귀 테스트가 없다. 실제 Dialer를 거친 검증을 추가하면 자격 증명 실패 분류와 원문 보존 계약을 보호하면서 미병합 본문 상한 구현과 충돌을 최소화할 수 있다.
- 수용 기준:
  1) 실제 127.0.0.1 TCP 서버가 USER 또는 PASS에 각각 -ERR를 반환할 때 Dial은 nil 세션과 오류를 반환하고, errors.As(err, &authErr)로 *domain.AuthError를 인식한다. USER 거부 시 PASS가 전송되지 않고, 두 경우 모두 서버가 클라이언트 연결 종료를 관찰한다. USER/PASS 성공 대조군은 같은 실제 Dialer와 domain.NewSecretHandle로 로그인 후 LIST까지 성공한다.
  2) -ERR 인사말은 nil 세션과 오류를 반환하고 AuthError로 분류되지 않는다. 서버는 인증 명령을 받지 않고 EOF를 관찰한다. 서버의 자체 timeout이나 핸들러 종료만을 EOF의 증거로 쓰지 않는다.
  3) Retrieve와 Top을 각각 실제 명령으로 실행하여, 헤더·빈 줄·일반 본문·'..foo'·'..'·'...bar'를 포함한 정상 응답이 정확한 CRLF와 함께 '.foo'·'.'·'..bar'로 반환되는지 전체 바이트를 비교한다. 단독 '.' 종결 행은 결과에서 빠지고, 같은 세션의 후속 LIST가 정확한 값으로 성공해 프레임 경계 보존을 증명한다. TOP 요청의 number/lines 인수도 서버에서 확인한다.
- 건드릴 파일: internal/adapters/pop3/client_test.go — 기존 dial/maildropServer/stlsServer의 실제 TCP 패턴을 참고해 인증/인사말/본문용 작은 스크립트 서버와 테스트를 추가. 기존 dial은 Username을 비워 인증을 건너뛰므로 인증 테스트에서는 Dialer{}.Dial에 Username과 실제 domain.SecretHandle을 전달한다. 프로덕션 변경 0개, 테스트 파일 1개.
- 검증 명령:
  - go test -race -count=1 ./internal/adapters/pop3/ ./internal/adapters/imap/ — 정찰에서 실제 성공(pop3 2.897s, imap 4.259s).
  - go test -race -count=3 -timeout=90s ./internal/adapters/pop3/ — 추가 테스트와 기존 5개가 모두 통과해야 한다(구현 후 실행).
  - make lint-format 및 git diff --check — 형식 검사(구현 후 실행).
- 위험과 피할 것: client.go, retrBody, readList, 데드라인·상한·인증 정책·오류 분류 자체는 변경하지 않는다. 04b15be는 HEAD 조상이 아님(exit 1 확인); PR #22의 사람 반려 여부는 미확인이므로 재구현/병합하지 않는다. auth/session, migrations, workflows, 설정, 프런트 자산은 범위 밖. fake session·직접 구성한 session·문자열 검색을 동작 증거로 쓰지 않는다. 이번 테스트는 어댑터 계약만 증명하며 application의 AccountCredentialError 저장까지 검증했다고 주장하지 않는다. USER/PASS 중 네트워크 단절을 AuthError로 감싸는 별도 문제는 이번 정상/명시적 거부 행렬에 섞지 않는다.
- 차선 후보: sync.max_message_bytes의 0·음수=무제한 규약을 설정 카탈로그·관리자 문서에 명시 — 동일 회귀 테스트가 다른 변경으로 이미 들어온 경우에만 검토. 설정 검증·동작을 바꾸지 않고, POP3 어댑터가 아직 본문 상한을 적용하지 않는다는 현재 한계를 숨기지 않는다.

범위와 접근 선택
- 가장 작은 대안은 USER/PASS 거부만 검증하는 것이다. 같은 픽스처로 인사말과 성공 대조군까지 확인할 수 있으므로 이 부분은 함께 포함한다.
- 선택안은 단일 테스트 파일에 작은 실제 TCP 픽스처를 추가하여 인증/인사말/RETR/TOP 계약을 고정하는 것이다. 기존 테스트 인프라를 이용하고 프로덕션 정책을 바꾸지 않는다.
- 더 큰 대안인 StartSync까지의 통합 행렬은 계정/스토어/설정 배선까지 확장된다. 실제 어댑터 경로의 공백을 메우는 이번 목표에는 필요하지 않아 보류한다.
- 현상 유지는 비용이 없지만 반복적으로 수정 중인 어댑터의 핵심 정상/거부 계약이 계속 무방비로 남는다. 문서 정렬은 가치가 있으나 이 실행 경로의 회귀를 잡지 못하므로 차선이다.
- 핵심 가정: 현재 코드의 명시적 -ERR 분류와 올바른 dot-stuffed 본문 처리가 의도된 계약이며, 새 테스트는 현재 코드에서 통과할 것으로 예상한다. 정찰은 이 신규 행렬을 실행하지 않았으므로 실제 결과는 미확인이다. 결함 수정이나 수정 전 실패를 꾸며낼 과제가 아니다.

실행 순서와 체크포인트 (전부 구현자 자체 검토, 사람 승인 대기 없음)
1. [pending] client_test.go에 인증/인사말 픽스처와 성공·거부 테스트를 추가한다. 서버 goroutine의 오류/명령 기록은 채널로 본 테스트에 전달하고, listener와 accepted conn은 t.Cleanup으로 닫으며 서버 deadline과 제한된 채널 대기를 둔다. EOF 단언 전에 클라이언트 cleanup으로 연결을 닫아 실패를 가리지 않는다. 증명: go test -race -count=1 -timeout=90s ./internal/adapters/pop3/. 체크포인트: 실제 Dialer·SecretHandle 사용과 명령 순서를 확인한다.
2. [pending] 같은 파일에서 RETR/TOP 본문과 후속 LIST 검증을 추가한다. 범용 서버 프레임워크를 만들지 말고 작은 응답 스크립트를 쓴다. 증명: go test -race -count=3 -timeout=90s ./internal/adapters/pop3/. 체크포인트: 두 공개 경로가 각각 실행됐고 바이트 전체 비교를 하는지 확인한다.
3. [pending] go test -race -count=1 ./internal/adapters/pop3/ ./internal/adapters/imap/; make lint-format; git diff --check를 각각 실행한다. 체크포인트: 프로덕션 파일 변경 0, 기존 상한/STLS 테스트 유지, 실제 출력 기록 후 완료 표시. 예상과 다르면 과제서를 먼저 수정하고 프로덕션 수정을 몰래 추가하지 않는다.

작업량 산정
- 근거는 읽은 client_test.go의 기존 net.Listen/Dialer 픽스처이며 bottom-up 추정이다: 인증/인사말 10–14분, RETR/TOP 8–12분, 검증/정리 5–7분 = 기본 23–33분.
- 알려진 불확실성인 EOF 동기화·goroutine 정리 실패에 별도 예비 5–7분을 둬 합계 28–40분, 45분 세션 안을 목표로 한다. 중간 신뢰도의 주관적 범위이며 실제 속도 데이터에 근거한 확률 보장은 아니다. 기존 회차는 시간이 기록되지 않아 유추법으로 수치를 교차 검증할 근거가 없다.
- 범위 밖 결함용 관리 예비는 0분이다. 새 결함은 별도 아이디어로 남기며 테스트 대상·정책을 확장하지 않는다. 첫 인증 테스트 통과 시 남은 시간을 재산정한다.

읽은 근거
- internal/adapters/pop3/client.go: Dial, readResponse, cmd, retrBody, Retrieve, Top, Close.
- internal/adapters/pop3/client_test.go: dial, maildropServer, stlsServer, 기존 TestPOP3* 5개.
- internal/domain/account.go: POP3DialOptions, POP3Session, AuthError; internal/domain/secret.go: NewSecretHandle, Zero.
- internal/application/app.go: dialInbound가 어댑터 오류를 그대로 반환. internal/application/sync.go: runSync의 errors.As 분기에서 AccountCredentialError 처리.
- Skill 호출 도구는 제공되지 않아 호출하지 못했다. 대신 아래 설치 원문과 estimating-and-contingency/references/sources.md를 직접 읽어 절차를 반영했다(외부 산정 표준의 정량 검증을 수행한 것은 아님).
  - [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md)
  - [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md)
  - [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md)
