# 구현 검증 결과

- model 계약: `go test ./internal/model -count=1` exit 0; model 0.002s.
- 입력 표면: `go test ./internal/model ./internal/mcp ./cmd/goalforge -count=1` exit 0; model 0.003s, MCP 3.279s, CLI 7.165s.
- 영향 패키지: `go test ./internal/observer ./internal/store/sqlite -count=1` exit 0; observer 45.827s, sqlite 88.718s.
- 최종 게이트: `go test ./... -count=1 -json` exit 0; 전체 원본 출력은 full-test.jsonl. 32개 패키지 pass, 825개 test/subtest pass, 테스트 4개 skip, fail 0. procctl/testscript는 테스트 파일 없음.
- `go vet ./...`, `go build ./...`: 각각 exit 0, 출력 없음.
- `gofmt -l ./cmd ./internal`, `git diff --check`: exit 0, 출력 없음.
- `go mod tidy` 및 `git diff --exit-code -- go.mod go.sum`: exit 0, 드리프트 없음.

## 실행 경로와 한계

- TestParseCriterion: 정상 6개(요구 4개 + 값 대소문자 보존 + 미등록 kind 구문 허용), 잘못된 구문 7개. 오류 문구 및 오류시 반환 구조체에는 의존하지 않음.
- TestCLICriterionContract: 기존 runCLI/runCLIWithError/gitIn 사용. 임시 Git 저장소 등록 후 동일 입력 2개를 실제 run으로 저장. 빈/미등록 kind 혼합 변경시 reason 제공, 실제 DB CurrentGoal 전체 값 비교. provider 실행·commit·push 전제 없음, parallel 없음.
- TestMCPCriterionContract: 기존 fixture와 rpcEnvelope 사용. 실제 Serve의 tools/call goal_set 호출, result.isError와 JSON-RPC error 구분, 동일 입력 및 실패 보존 확인. 대역 Store/직접 handler 호출 없음.
- 신규 테스트는 첫 실행부터 통과했으며 프로덕션 결함을 재현한 작업이 아님. 프로덕션 파일 및 기존 하네스 수정 없음.
- 기존 테스트 TestPushBranchPublishesToLocalRemote, TestCLIFullLifecycle, TestRestoreVerifiesRecordsAndSettlesOutsideWork, TestPublishReconcilesInsteadOfRepeating은 origin push probe가 `fatal: 'DISABLED' does not appear to be a git repository`로 실패하여 기존 정책대로 skip. 환경이나 skip 조건을 바꾸지 않음.
- Linux에서 실행. Windows/macOS, 릴리즈 교차 빌드 및 게시 미실행. 빌드/릴리즈 경로는 변경하지 않았고 릴리즈는 이번 세션 범위 밖.
