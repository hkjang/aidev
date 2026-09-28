# GoalForge 프로필 (2026-09-29)

- 목적: 목표(goal)를 1급 시민으로 두는 개발 오케스트레이터. Go 프로세스가 프로젝트 상태·버전된 목표·작업 순서·검증 증거·완료 판정을 소유하고, AI 세션(Codex/Claude Code/Qwen/OpenCode)은 진실의 원천이 아니라 실행 도구로만 쓴다. (README.md 첫 단락)

- 스택: Go (모듈 `github.com/goalforge/goalforge`). SQLite 주력 + Postgres 구현 병존(`internal/store/sqlite`, `internal/store/postgres`). HTTP API + 정적 대시보드(`internal/api`, `docs/`), MCP 서버(`internal/mcp`), TUI(`internal/tui`). 프런트엔드 빌드 도구 없음.

- 구조:
  - `cmd/goalforge/main.go` — CLI 표면 전체가 한 파일(약 2900줄). e2e 는 `main_e2e_test.go`.
  - `internal/app` — 서비스 계층(`service.go`, `plan.go`, `effects.go`, `evalexec.go`). 외부 효과 조정(reconcile)이 여기.
  - `internal/orchestrator`, `internal/scheduler`, `internal/planner` — 런 실행·재개·프리플라이트·쿼터, 스케줄링, 아이디어 생성/중복제거.
  - `internal/policy` — 권한·역할·드리프트·재시도·게이트 실패 정책. 테스트 밀도 최고.
  - `internal/gitops` — 워크트리, 커밋/머지/푸시, 롤백, reconcile, diff 검사.
  - `internal/verification` + `internal/evaluation` — 게이트 실행 엔진, 평가 케이스 재실행.
  - `internal/store/sqlite` — 영속화. 도메인별 파일 40여 개.
  - `internal/provider/{claude,codex,qwen,opencode}` — CLI 어댑터와 stream-json 파서. 그 외 `audit`, `notify`, `usage`, `diagnostics`, `report`, `prompt`, `model`, `tui`, `procctl`, `testscript`.
  - `scripts/build-release.sh` — 플랫폼별 크로스컴파일 + `SHA256SUMS` 생성. CI 가 `v0.0.0-ci` 로 매 PR 마다 돌린다.

- 빌드·테스트:
  - `go build ./...` / `go vet ./...` / `gofmt -l ./cmd ./internal`(무출력이어야 함)
  - `go test ./...` — 전체 약 40초. 느린 패키지: `internal/store/sqlite` ~17s, `internal/app` ~7s, `internal/api` ~6.4s, `internal/orchestrator` ~5.9s, `internal/verification` ~3.6s, `cmd/goalforge` ~3.3s.
  - `-run` 스코핑이 잘 듣는다: `go test ./internal/gitops/ -run TestCommitVerified -count=1 -v`.
  - **CI 있음(프로필 이전 판의 "CI 없음" 은 틀렸다).** `.github/workflows/ci.yml`(push:main / PR — build·vet·`go test ./... -count=1`, gofmt, `go mod tidy` 드리프트, 릴리즈 아티팩트 크로스컴파일+체크섬 검증) 과 `.github/workflows/release.yml`(태그 `v*` — ubuntu/windows/macos 3 플랫폼 `go vet`+`go test`, 그 뒤 build-provenance attestation 과 `gh release upload`).

- 관례: 커밋 메시지는 **한국어 한 줄 제목**에 무엇을 바꿨는지가 아니라 **무엇이 달라지는지**를 쓴다(`상태 데이터베이스가 무한히 자라던 것을 정리할 수 있게 한다`, `사라진 워크트리를 git 오류가 아니라 무슨 일인지로 말한다`). 작업은 `feature/…`·`fix/…`·`improve/…` 브랜치 → PR → 머지 커밋. 코드 주석은 영어 기본에 정책 판단 자리에 한국어 괄호주(`gitops/commit.go` 의 `자동 병합 금지`). 워크플로 주석은 "무엇이 없던 시절 무엇이 문제였나" 를 서술형으로 쓴다. 자식 프로세스 환경은 `policy.SessionEnvironment(os.Environ(), policy.RoleImplementation)`. 문서는 `README.md`(명령 레퍼런스)와 `docs/acceptance-audit.md`. 소스에 TODO/FIXME 가 하나도 없다.

- 위험 구역:
  - `internal/store/sqlite/store.go` 의 스키마·마이그레이션 — 기존 프로젝트 DB 가 깨진다.
  - `internal/policy/*` — 권한 모델과 역할 분리. "구현 세션에 운영자보다 낮은 권한", "런이 자기 작업을 스스로 인증하지 못하게" 가 못박혀 있다. 완화는 곧 회귀.
  - `internal/gitops/commit.go`·`rollback.go` — 사용자 저장소에 실제 커밋/머지/푸시. `PushBranch` 는 명시적 승인(SEC-011) 뒤에만, 절대 force 금지.
  - `.github/workflows/*` — 보호 경로. 검증을 통과시키려고 여기를 느슨하게 고치는 것은 금지.

- 자주 깨지는 곳: git 을 실제로 실행하는 테스트가 주변 환경에 진다. 커밋 신원과 push 두 축이 반복해서 같은 자리에서 깨진다. 2026-09-28/09-29 두 회차의 수정은 아직 main 에 들어오지 않았다(main@dc28e64 기준 `internal/model` 은 여전히 `[no test files]`, 신원 실패도 그대로 재현된다).

- 검증 함정:
  - **main@dc28e64 에서 `go test ./...` 가 6건 실패한다** — "전체 통과" 를 기준선으로 삼지 말고 변경 전 기준선을 먼저 실측할 것.
    - 신원 축 3건: `TestCLIFullLifecycle`(main_e2e_test.go:115), `TestMergeVerifiedMergesCleanAndAbortsConflicts`(commit_test.go:51), `TestCommitVerifiedCreatesTrailedCommitOffProtectedBranch`(commit_test.go:178). 원인: 환경의 `GIT_AUTHOR_*`/`GIT_COMMITTER_*` 가 `git -c user.name=` 를 이긴다(`-c` 는 설정 파일만 이긴다).
    - push 축 3건: `TestPushBranchPublishesToLocalRemote`(commit_test.go:110), `TestPublishReconcilesInsteadOfRepeating`(effects_test.go:74), `TestRestoreVerifiesRecordsAndSettlesOutsideWork`(main_e2e_test.go:576). 전부 `fatal: 'DISABLED' does not appear to be a git repository`. 하네스가 push 를 의도적으로 막은 것으로 판단(2026-09-28 원장이 `GIT_CONFIG_*` 주입으로 특정). 우회 금지.
  - 6개 테스트 파일이 각자 `git init`+신원 설정+`t.Skipf` 를 중복 구현한다(`internal/api/setup_test.go:21`, `internal/evaluation/runner_test.go:19`, `internal/app/plan_test.go:41`, `internal/app/service_test.go:628,705`, `internal/app/effects_test.go:25`). 환경 격리가 한 곳에 없다.
  - `internal/model`·`internal/procctl`·`internal/testscript` 는 `[no test files]`.
  - 정찰 세션 샌드박스 제약: `printenv`, `git config --list`, `git var`, 복합 셸 명령이 승인 없이는 막힌다. 진단은 `go test` 출력과 파일 읽기로만 가능하다.
