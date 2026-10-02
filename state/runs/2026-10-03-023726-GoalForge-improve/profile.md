# GoalForge 프로필 (2026-10-03, base main@a6aa75f)

- 목적: 목표(goal)를 1급 시민으로 두는 개발 오케스트레이터. Go 프로세스가 프로젝트 상태·버전된 목표·작업 순서·검증 증거·완료 판정을 소유하고, AI 세션(Codex/Claude Code/Qwen/OpenCode)은 진실의 원천이 아니라 실행 도구로만 쓴다.
- 스택: Go (모듈 `github.com/goalforge/goalforge`). SQLite 주력 + Postgres 병존. HTTP API + 정적 대시보드, MCP 서버, TUI. 프런트엔드 빌드 도구 없음.

- 구조 (**0c9e799 → a6aa75f 사이에 9001 insertions / 76 files — 아래는 그 뒤 상태**):
  - `cmd/goalforge/` — CLI 표면. `main.go`(약 4200줄) + 도메인별 분리 파일(`standards.go`, `fleet.go`, `duration_test.go`, `main_e2e_test.go`).
  - `internal/observer/` — 여전히 무게중심. `supply.go`(작업 공급), `schedule.go`(주기·예산), `verify.go`(기준 판정), `autonomy.go`(자동 실행 승인), `automerge.go`(자동 병합 승인), `tick.go`(무인 스윕). 최근 추가: `author.go`, `goaldraft.go`, `detectors.go`. 전부 테스트 있음.
  - `internal/rrsi/` — **신규.** `propose.go`·`screen.go`·`judge.go`·`policy.go`·`change.go`·`calibrate.go`. 테스트는 `propose_test.go`·`rrsi_test.go` 둘뿐이라 파일별 커버리지 미확인.
  - `internal/prompt/` — `ideas.go` + **신규** `proposal.go`(제안 산문 파싱·렌더). 둘 다 테스트 있음.
  - `internal/standards`, `internal/patterns` — 판정 가능한 데이터로 둔 공통 개발 기준 팩. 신규 `gocli.go`, `registry.go`, `conditional_exception_test.go`.
  - `internal/app`, `internal/orchestrator`, `internal/scheduler`, `internal/planner` — 서비스 계층, 런 실행·재개·쿼터, 스케줄링, 아이디어 생성/중복제거.
  - `internal/policy` — 권한·역할·드리프트·재시도·게이트 실패 정책. `internal/gitops` — 워크트리, 커밋/머지/푸시, 롤백, reconcile.
  - `internal/store/sqlite` — 영속화, 도메인별 파일 60여 개. 신규: `appliedchange.go`, `leasehold.go`, `selection.go`, `unblock.go`, `workscope.go`(각각 테스트 동반).
  - `internal/model/` — **`model.go` 하나뿐이고 여전히 `[no test files]`.** 2026-09-29 회차의 `window.go`/`ParseWindow` 는 네 회차째 main 에 없다. "지난 회차가 넣었다" 를 전제하지 말고 반드시 코드에서 확인할 것.
  - `scripts/build-release.sh` — 크로스컴파일 + `SHA256SUMS`. CI 가 `v0.0.0-ci` 로 매 PR 실행.

- 빌드·테스트: `go build ./...` / `go vet ./...` / `gofmt -l ./cmd ./internal`(무출력) / `go test ./... -count=1`(약 60s+).
  - **2026-10-03 실측: `go test ./internal/observer/ ./internal/store/sqlite/ -count=1` 둘 다 `ok`** (observer 14.4s, store/sqlite 34.7s). 전체 실행은 이번 회차에 하지 않음.
  - 느린 패키지: `internal/store/sqlite` ~35s, `internal/observer` ~14s, `internal/api` ~9s, `internal/app` ~9s.
  - `-run` 스코핑이 잘 듣는다: `go test ./internal/observer/ -run TestTick -count=1 -v`.
  - CI: `.github/workflows/ci.yml`(push:main / PR — build·vet·test, gofmt, `go mod tidy` 드리프트, 릴리즈 아티팩트 크로스컴파일+체크섬), `release.yml`(태그 `v*` — ubuntu/windows/macos vet+test, build-provenance attestation, `gh release upload`).

- 관례: 커밋 제목은 **한국어 한 줄로 무엇이 달라지는지**(무엇을 바꿨는지가 아니다). 본문은 길고 각 설계 판단에 "이렇게 하지 않으면 무엇이 문제인가" 를 붙인다 — 가장 강한 관례. `feature/…`·`fix/…`·`improve/…`·`docs/…` 브랜치 → PR → 머지. 릴리즈마다 `가이드 기준 버전을 vX.Y.0 으로` 커밋이 따라온다(현재 v0.41.0). 코드 주석은 영어 기본, 정책 판단 자리에 한국어. 사용자 향 문자열은 한국어. 자식 프로세스 환경은 `policy.SessionEnvironment(os.Environ(), policy.RoleImplementation)`. 문서는 `README.md`, `docs/STANDARDS.md`, `docs/acceptance-audit.md`. 소스에 TODO/FIXME 없음.

- 위험 구역:
  - `internal/observer/autonomy.go`·`automerge.go` — 자동 승인 봉투. "봉투가 무엇을 허용하는지 판단하는 규칙은 `AutoApprove` 한 곳에만" 이 명시적 결정(복제했다 지운 이력 있음). 규칙 복제는 회귀. 멱등성·기록 같은 비(非)규칙 가드는 이 결정과 별개이지만, 커밋 본문에 그 구분을 써야 통과한다.
  - `internal/policy/*` — 권한 모델과 역할 분리("구현 세션에 운영자보다 낮은 권한", "런이 자기 작업을 스스로 인증하지 못하게"). 완화는 곧 회귀.
  - `internal/store/sqlite/store.go` 스키마·마이그레이션 — 기존 프로젝트 DB 가 깨진다.
  - `internal/gitops/commit.go`·`rollback.go` — 실제 커밋/머지/푸시. `PushBranch` 는 명시 승인(SEC-011) 뒤에만, force 금지.
  - `.github/workflows/*` — 보호 경로. 검증을 통과시키려고 느슨하게 고치는 것 금지.

- 자주 깨지는 곳:
  - **"계산해 두고 아무도 읽지 않는 값"** 이 반복된다(5b83c30 이 스스로 고백한 부류). 2026-10-01 이 `ProjectTick.Note` 를 로그에 닿게 했으나 `AutoDecisions.Detail`/`Refused` 는 호출부에 따라 아직 버려진다.
  - **`var _ = time.Now`** 가 `automerge.go:168`·`tick.go` 끝에 남아 있다 — 시간 기반 검사가 빠진 자리의 흔적.
  - **멱등성.** 무인 스윕이 15분마다 같은 함수를 돌리는데, 발급·기록 경로가 "이미 했는지" 를 보지 않는 자리가 있다(`AutoApproveMerges` → `RequestScopedApproval`, 2026-10-03 확인). 순회에서 불리는 함수를 고칠 때는 두 번 불러 보는 테스트를 기본으로 쓸 것.
  - git 을 실제로 실행하는 테스트가 주변 환경에 진다. 커밋 신원 축은 `commitIdentityEnv()`(gitops/commit.go), push 축은 `requirePushable` 프로브로 닫혔고 셋 다 main 에 있다. `requirePushable` 은 `internal/gitops`·`internal/app`·`cmd/goalforge` 에 동일 구현 복제(패키지 테스트가 서로를 임포트할 수 없어 불가피).

- 검증 함정:
  - 샌드박스가 **복합 셸 명령(`&&`, `;`, 파이프)과 `gh`, `printenv`, `git config --list` 를 승인 없이 막는다.** 명령을 하나씩 쪼개 실행할 것. PR 상태를 `gh` 로 확인할 수 없다.
  - 하네스가 `GIT_CONFIG_COUNT=2`/`remote.origin.pushurl=DISABLED` 를 주입해 **`origin` 이라는 이름의 remote 로만** push 를 막는다. 경로를 직접 준 push 는 성공한다 — 프로브를 경로로 짜면 통과해 버린다.
  - `AutoDecisions.Refused` 는 `map[string]string` — 순회 순서에 의존한 단정은 무작위로 깨진다.
  - `internal/observer` 테스트는 실제 `*store.Store`(SQLite)와 실제 git 저장소를 쓴다. 대역 주입으로 봉투 결함을 증명하려 하면 배선을 보지 못한다.
  - 머지되지 않은 회차가 누적되어 있다(`model.ParseWindow` 가 그 예). 선행 PR 에 의존하는 과제는 고르지 말 것.
