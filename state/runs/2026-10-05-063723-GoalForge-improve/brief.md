- 과제: ParseCriterion과 CLI·MCP의 완료 조건 저장 계약을 실제 경로로 고정한다 (가치 3 / 위험 1 / 작업량 M)
- 왜: main@96c90ac의 internal/model은 여전히 테스트가 없고, 완료 조건의 type@kind=value를 읽는 ParseCriterion의 공백·대소문자·구분자 계약을 직접 보호하지 못한다. 실제 CLI와 MCP로 같은 조건을 저장하고 다시 읽는 테스트까지 추가하면 요구한 증거 종류가 입력 과정에서 빠지거나 잘못된 입력이 활성 목표를 바꾸는 회귀를 막을 수 있다.
- 수용 기준:
  1) 실제 model.ParseCriterion을 표 기반으로 호출한다: build_passed=true → (build_passed,true,빈 kind), " SaveNote @ JoUrNeY = true " → (SaveNote,true,journey), "latency@test=<=200ms" → (latency,<=200ms,test), "payload@review=a=b@c" → (payload,a=b@c,review). type/값의 대소문자와 값 내부 =/@는 보존되고 바깥 공백은 정리된다.
  2) 빈 문자열, = 없음, "=true", "x=", "x=   ", "@test=true", "x@=true"는 오류다. 오류 문구 전체나 오류 시 부분 반환 구조체에 의존하지 않는다. 알 수 없는 kind는 구문 파서가 받아도 저장 계층의 ValidGateKind가 거절한다는 책임 경계를 유지한다.
  3) 같은 유효 입력(" SaveNote @ JoUrNeY = true ", "latency@test=<=200ms")을 CLI의 실제 run dispatch와 MCP tools/call goal_set에 각각 넣고, 실제 SQLite CurrentGoal로 재조회한 Type/ExpectedValue/RequiredKind가 같음을 증명한다. 각 표면에서 기존 목표를 만든 뒤 유효 criterion과 빈 kind 또는 미등록 kind를 섞은 변경을 시도하면 실패하고, 기존 활성 목표의 ID·Version·Criteria가 그대로여야 한다. 변경 사유는 반드시 제공해서 'reason 누락'이라는 엉뚱한 실패를 피한다.
- 건드릴 파일:
  - internal/model/model_test.go (신규): TestParseCriterion — 위 구문 계약. 프로덕션 함수는 internal/model/model.go:ParseCriterion(확인함).
  - cmd/goalforge/criterion_test.go (신규): TestCLICriterionContract — 기존 main_e2e_test.go:runCLI/runCLIWithError/gitIn을 재사용하여 run → goalSet → Store.SetGoal을 지난 뒤 CurrentGoal을 확인한다. 임시 DB(GOALFORGE_DB), 임시 저장소 및 t.Chdir 사용. 프로젝트 등록까지만 필요하며 provider 실행·commit·push는 필요 없다.
  - internal/mcp/criterion_test.go (신규): TestMCPCriterionContract — server_test.go:fixture와 TestStdioProtocolLifecycle의 Serve + tools/call 패턴을 참고한다. fixture에는 이미 목표가 있으므로 goal_set에 reason을 준다. 도구 오류는 JSON-RPC error와 다른 result.isError이며 기존 테스트가 이 계약을 보여준다. 실제 DB를 읽어 저장·롤백을 확인한다.
  - 프로덕션 변경 0개, 테스트 파일 3개. 위 기존 파일은 읽기 참고용이며 변경 대상 아님.
- 검증 명령:
  - go test ./internal/model ./internal/mcp ./cmd/goalforge -count=1
  - go test ./internal/observer ./internal/store/sqlite -count=1
  - gofmt -l ./cmd ./internal
  - 구현 최종 게이트: go test ./... -count=1 및 go vet ./... (CI에 있는 명령; 전체 실행은 정찰에서 미실행).
- 위험과 피할 것: 이번 과제는 확인된 테스트 공백 보강이며 새 프로덕션 결함을 재현했다고 주장하지 않는다. ParseCriterion 구현·kind 정책·목표 버전 규칙·auth·마이그레이션·workflows를 수정하지 않는다. FakeTask/가짜 Store/별도 파서를 만들지 않고 실제 함수·CLI dispatch·MCP Serve·SQLite로 검증한다. 서로 다른 표면에 동일한 원문과 기대 필드를 적용한다. t.Chdir/t.Setenv/전역 stdout을 쓰는 CLI 테스트에 t.Parallel을 넣지 않는다. 기존 긴 lifecycle의 fake provider나 push 프로브를 새 테스트의 전제로 삼지 않는다. 미머지 ParseWindow 작업과 git 테스트 헬퍼 공유화는 범위 밖이다. 문자열 검색을 동작 증거로 삼지 않는다.
- 차선 후보: ParseCriterion 단독 계약 테스트 (가치 3 / 위험 1 / 작업량 S) — 표면 통합 테스트가 기존 하네스 변경 또는 프로덕션 수정을 요구해 45분을 넘길 경우, 같은 과제의 model_test.go 부분만 완결한다. 관찰하지 않은 저장 계약을 통과했다고 쓰지 않는다.

실행 순서와 체크포인트 (구현 전, 전부 pending):
1. model_test.go에 사용자 입력 계약을 추가한다. 증명: go test ./internal/model -count=1. 사람 승인 대기 없이 성공한 뒤 다음 단계.
2. CLI/MCP 각 기존 하네스를 이용해 같은 입력 저장 및 실패 후 기존 목표 보존을 검증한다. 증명: go test ./internal/model ./internal/mcp ./cmd/goalforge -count=1. 기존 하네스 예상과 다르면 과제서를 수정하고, 테스트를 통과시키려고 정책을 바꾸지 않는다. 사람 승인 대기 없음.
3. gofmt 후 영향 패키지 및 최종 게이트를 실행한다. 증명: 위 검증 명령. 환경 실패와 코드 실패를 구분해 기록한다. 구현자가 결과로 상태를 갱신한다.

선택 비교:
- 단독 파서 테스트(S)는 가장 작지만 호출부가 RequiredKind를 버리는 회귀를 잡지 못한다. 선택안은 실제 입력 표면 두 개의 저장 결과까지 보호하며 선행 PR에 의존하지 않는다.
- 무인 스윕의 WRONG_KIND 알림은 가치가 있지만 Tick의 Note만 추가해도 sweepReport의 Acted 조건 때문에 조용한 스윕에는 출력되지 않는다. 알림 정책 설계까지 요구하므로 이번 회차에서 제외한다.
- 자동 병합 일일 한도와 옛 승인 철회는 기록·정책·감사 경계 변경이 필요해 제외한다. 파서 통합·API 변경·공유 헬퍼 리팩터는 하지 않는다.

작업량 근거 (pmo:estimating-and-contingency):
- bottom-up 추정: 모델 표 5–7분, CLI 저장/실패 8–10분, MCP 저장/실패 8–10분, 최종 확인 4–6분 = 기본 25–33분.
- 알려진 불확실성 예비시간 5–8분(MCP 도구 오류 decoding, 기존 목표 reason/버전 setup)으로 총 30–41분, 45분 상한. 약 80% 내 완료를 예상하는 정성 추정이며 통계적 보장은 아니다.
- 전제: 기존 실제 하네스 재사용, 테스트 파일 3개, 외부 provider·network 불필요. 이전 성공 회차의 패키지 테스트 재사용 방식을 참고했으나 구현 소요시간 기록이 없어 수치 유추법 비교는 불가하다. 알려지지 않은 범위 확대를 위한 관리 예비시간은 배정하지 않는다(0분); 생기면 차선으로 축소한다.

정찰 확인:
- baseline main@96c90ac, 시작 git status --short 무출력. CLAUDE.md/AGENTS.md/별도 roadmap 파일은 검색에서 발견하지 못했다. README, docs/STANDARDS.md, docs/SELECTION.md, docs/acceptance-audit.md, GUIDE 도입부, git log -30, CI/release 설정 확인. cmd/internal/scripts TODO/FIXME 검색 결과 없음.
- 실행: go test ./internal/model ./internal/mcp ./cmd/goalforge -count=1 exit 0 (model은 [no test files], mcp 0.513s, CLI 2.429s).
- 실행: go test ./internal/model ./internal/observer ./internal/store/sqlite -count=1 exit 0 (observer 18.999s, sqlite 45.340s). 전체 suite·vet·build 및 Windows/macOS 실행은 미확인.
- 읽은 핵심 배선: model.go:ParseCriterion, main.go:goalSet 및 evaluation criterion 입력, mcp/tools.go:goalSet, sqlite/store.go:SetGoal/CurrentGoal, policy/gatekind.go:ValidGateKind. Store.SetGoal의 tx.Rollback defer와 kind 검증 순서를 확인했지만 신규 실패/원자성 시나리오는 구현자가 실행으로 증명해야 한다.
- 초안을 먼저 저장한 뒤 실제 배선과 baseline 결과를 반영해 이 파일을 덮어썼다.

적용 스킬: 전용 Skill 도구가 제공되지 않아 다음 로컬 원문을 직접 읽고 적용했다.
- [pmo:estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md) (references/sources.md도 확인; 외부 비용 수치는 사용하지 않음)
- [technology:implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md)
- [technology:solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md)
