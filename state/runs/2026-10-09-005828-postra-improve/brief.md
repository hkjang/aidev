- 과제: POP3·IMAP greeting 실패 진단의 Timeout을 실제 connectTO로 맞춘다 (가치 3 / 위험 1 / 작업량 M)
- 왜: 두 Dial은 greeting을 connectTO로 기다리지만 InboundError.Timeout에는 commandTO를 기록하므로, 기본 설정에서는 15초 제한을 소진해도 60초 중 조기에 중단된 것으로 표시된다. 실제 단계 예산을 기록하면 작업·장애 진단을 읽는 운영자가 외부 조기 중단과 서버 무응답을 올바르게 구분할 수 있다.
- 수용 기준: 1) 실제 TCP 침묵 서버에 ConnectTimeoutSec=1, CommandTimeoutSec=4로 Dial하면 POP3·IMAP 모두 errors.As로 얻은 InboundError가 StageGreeting/Class=timeout/Command=""/Timeout=1초이고 Elapsed>0이다. 2) POP3 -ERR 및 IMAP * BYE의 즉시 거절도 greeting/rejected와 connectTO를 기록하고 연결을 닫으며, 성공 greeting 뒤 명령 실패는 기존 commandTO를 기록한다. 3) 실제 어댑터와 CreateAccount→StartSync→GetJob을 거친 양쪽 프로토콜의 저장 Job은 failed/greeting/timeout, TimeoutMS=1000이며 Summary에 제한 1.0초가 나오고 “이 단계의 제한 시간에는 이르지 않았습니다”가 나오지 않는다; 서버 표식은 저장 진단에 포함되지 않는다.
- 건드릴 파일: internal/adapters/pop3/client.go:Dial — greeting 실패 WrapInbound의 마지막 인자를 connectTO로; internal/adapters/imap/client.go:Dial — readBoundedLine 실패와 * BYE 두 반환에서 domain.WrapInbound(..., time.Since(start), connectTO)를 직접 사용; internal/adapters/pop3/client_test.go — 실제 Dialer greeting/후속 명령 회귀; internal/adapters/imap/client_test.go — 같은 회귀; internal/application/sync_diagnostics_test.go — 실제 어댑터로 최종 Job 진단 회귀. 총 5파일, 프로덕션 2파일. 추가 공통 헬퍼·계약·UI·문서 변경은 필요 없다.
- 검증 명령: 아래 순서와 실측 구분 참고.
- 위험과 피할 것: timeout 메타데이터만 바꾸며 read deadline·secondsOr 기본값·s.commandTO·공통 session.failed·retry 정책은 유지한다. domain.WrapInbound는 기존 InboundError가 있으면 그대로 반환하므로 s.failed 결과를 바깥에서 다시 감싸는 수정은 효과가 없다. STARTTLS Buffered 가드·TLS 설정·auth/migrations/.github/workflows/spa/assets·POP3 retrBody(PR #22)는 건드리지 않는다. 직전 회차 59faa19의 ensureIndex 수정은 이 main@c40b91d에 없지만 이미 별도 구현됐으므로 다시 구현하거나 병합하지 않는다.
- 차선 후보: README 보안 스캐너 절에 cyclonedx-gomod v1.10.0의 워크트리 제한을 기록 (가치 1 / 위험 1 / S) — 1순위가 이미 다른 변경으로 해결된 경우만. 10-02 회차의 worktree 실패/full clone·shallow 성공 기록에 근거한 환경 한계로 명시하고 보편적 도구 결함으로 단정하지 않는다. CI YAML을 고치지 않고 `git diff --check`로 검증한다. 이번 정찰에서 SBOM 재실행은 미확인.

범위·근거
- pop3/client.go:90–95: readGreeting(connectTO) 뒤 WrapInbound(..., s.commandTO). readGreeting은 직접 deadline을 설정한다.
- imap/client.go:79–96: SetDeadline(start.Add(connectTO)) 뒤 읽기 오류·BYE 모두 s.failed 사용. :797의 failed는 항상 commandTO를 넘긴다.
- application/sync_diagnostics.go:166 syncDiagnosticStage는 InboundError.Timeout을 TimeoutMS에 그대로 옮기고 :287 syncTiming은 ElapsedMS가 TimeoutMS의 90% 미만이면 조기 중단 문구를 낸다. docs/SYNC_DIAGNOSTICS.md도 greeting에는 연결 예산을 쓴다고 명시한다.
- 기존 TestSyncFailureNamesTheStepAndKeepsNoServerText는 scriptedInbound가 만든 60초/60초 오류를 사용하므로 실제 Dial의 잘못된 예산을 검출하지 못한다.
- 이번 정찰은 읽기·기존 테스트 실행만 했다. 새 침묵 서버 실패 재현과 수정 후 결과는 미확인이며 구현자가 아래 실행으로 증명한다.

구현 순서 (현재 모두 미착수, 사람 승인 체크포인트 없음)
1. 어댑터 회귀를 만들고 수정 전 실패를 확인한 뒤, 같은 단계에서 위 세 반환만 수정해 통과시킨다. 새 테스트 이름은 TestPOP3GreetingTimeoutBudget / TestIMAPGreetingTimeoutBudget을 권장한다. POP3는 pop3ScriptServer(t, "", nil, true)가 연결 유지·EOF 확인에 재사용 가능하다(서버 deadline 5초). IMAP에는 accept 뒤 아무 greeting도 보내지 않고 클라이언트 EOF까지 기다리는 짧은 TCP 픽스처를 둔다. Context는 Background 또는 충분히 긴 제한을 사용하고, timeout 클래스까지 단언해 서버 cleanup EOF나 context 만료의 거짓 통과를 막는다. 즉시 거절은 민감한 표식이 포함된 -ERR/* BYE로 확인하되 raw err.Error()의 무표식은 요구하지 않는다(POP3의 기존 원문 에러는 이 과제 범위 밖).
   증명: `go test -race -count=1 -timeout=180s ./internal/adapters/pop3/ ./internal/adapters/imap/`. 같은 값 두 개로 테스트하면 결함이 숨으므로 반드시 connect와 command 예산을 다르게 둔다. 성공 greeting 뒤 거부하는 명령을 보내 commandTO=4초인 대조군을 둔다. 시간 정확 일치·빡빡한 벽시계 상한은 피한다.
2. application/sync_diagnostics_test.go에 TestSyncGreetingTimeoutUsesConnectBudget을 추가해 POP3/IMAP 하위 사례를 실제 TCP 서버로 실행한다. newTestApp→app.POP3=pop3.Dialer{} 또는 app.IMAP=imap.Dialer{}를 사용한다. settingsAdmin→AdminSettingsCatalog의 Revision→AdminPatchSettings(SettingsPatch{Revision: ..., Values: map[string]string{"sync.connect_timeout_sec":"1", "sync.command_timeout_sec":"4"}})로 실제 설정 배선을 거친다(settings_service_test.go:TestSettingsLiveAndRestartSemantics 참고). CreateAccountInput의 InboundProtocol과 POP3Port에 프로토콜·루프백 포트를 넣고 평문/사용자명 없음으로 계정을 만든다(다른 필드는 imap_sync_test.go:TestSyncViaIMAP 참고). fastRetries(t)를 써 재시도 대기만 줄이고 실제 timeout은 유지한다; 서버는 재시도 연결을 모두 accept해야 한다. syncAndWait는 10초 한계이므로 기존 10초 재시도 대기를 그대로 두지 않는다. fastRetries가 패키지 전역을 바꾸므로 t.Parallel 금지. GetJob 진단·Summary 및 가능하면 기존 incident 확인 루프도 검증한다. 저장 Job 전체 JSON에는 서버 표식이 없음을 확인한다(표식 검증은 실제 거부 응답 사례를 사용).
   증명: `go test -race -count=1 -timeout=180s ./internal/application -run '^(TestSyncGreetingTimeoutUsesConnectBudget|TestSyncFailureNamesTheStepAndKeepsNoServerText|TestSyncViaIMAP)$'`. 수정 전에는 새 테스트의 TimeoutMS/제한 문구 단언이 실패해야 한다. 합성 InboundError나 소스 문자열 검사로 대체하지 않는다.
3. 위 두 단계가 통과하면 영향 패키지 전체와 정적 검사를 실행한다. 범위가 커지면 새 기능을 넣지 말고 이 과제서에 차이를 적는다.
   증명: `go test -race -count=1 -timeout=900s ./internal/application ./internal/adapters/imap ./internal/adapters/pop3`; `go build ./...`; `go vet ./...`; `make lint-format`; `go run ./cmd/postra-contracts -check`; `git diff --check`. 전체 저장소/PG/브라우저 게이트 성공으로 확대 보고하지 않는다.

정찰 실측 (2026-10-09, main@c40b91d)
- `go test -race -count=1 -timeout=180s ./internal/adapters/pop3/ ./internal/adapters/imap/`: PASS, POP3 5.399s / IMAP 4.591s.
- `go test -race -count=1 -timeout=180s ./internal/application -run '^(TestSyncFailureNamesTheStepAndKeepsNoServerText|TestSyncViaIMAP)$'`: PASS, 1.690s.
- 나머지 최종 검증 명령은 저장소 Makefile/CI·이전 실측에서 확인한 명령이며 이번 정찰 미실행. git status --porcelain 빈 출력.

접근 비교·작업량 근거 (세 스킬 적용)
- 선택: 어댑터의 예산 메타데이터를 바로잡는다. 원인과 출력 사이가 확인됐고 프로덕션 2파일에 한정된다.
- 대안: application에서 greeting의 TimeoutMS를 설정값으로 덮기 — 다른 어댑터 호출자는 계속 잘못된 오류를 받고 계층 간 중복이 생겨 제외. 모든 failed에 timeout 인자를 추가 — 향후 유연성은 있으나 이번 범위보다 큰 변경이라 제외. 문서로만 우회 — 원인 오진이 남아 차선보다 가치가 낮다.
- 가장 중요한 가정: 읽기 deadline의 connectTO 사용이 의도된 정책이라는 점. 양쪽 주석과 SYNC_DIAGNOSTICS 문서가 일치한다. 실제 운영 장애 사례·새 테스트 실패 출력은 미확인이다.
- Bottom-up: 어댑터 테스트·수정 10–13분, 실제 앱 배선·출력 테스트 10–13분, 검증·정리 6–8분 = 기본 26–34분. 알려진 불확실성(재시도 픽스처 cleanup·race) 예비 4–6분을 별도 두어 총 30–40분, 신뢰 중간(주관적 범위, 통계적 보증 아님). 기존 진단 수정 회차의 2개 어댑터+회귀 구조와 유사하지만 앱 TCP 테스트가 추가돼 S에서 M으로 조정했다. 과거 회차 시간 데이터가 없어 독립적인 유사 추정치는 산출하지 않았다. 관리 예비는 0분이며 새로운 정책 요구는 다음 회차로 보류한다.
