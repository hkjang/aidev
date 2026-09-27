# GoalForge 프로필 (2026-09-28)

- 목적: 목표(goal)를 1급 시민으로 두는 개발 오케스트레이터. Go 프로세스가 프로젝트 상태·버전된 목표·작업 순서·검증 증거·완료 판정을 소유하고, AI 세션(Codex/Claude Code/Qwen/OpenCode)은 진실의 원천이 아니라 실행 도구로만 쓴다. (README.md 첫 단락)

- 스택: Go (모듈 `github.com/goalforge/goalforge`). 저장소는 SQLite 주력 + Postgres 구현 병존(`internal/store/sqlite`, `internal/store/postgres`). HTTP API + 정적 대시보드(`internal/api`, `docs/`), MCP 서버(`internal/mcp`). 프런트엔드 빌드 도구 없음 — `docs/` 는 소스 그대로의 HTML/CSS/JS. Go 파일 175개 / 약 27,700줄.

- 구조 (5~10줄):
  - `cmd/goalforge/main.go` — CLI 표면 전체가 한 파일(2949줄). e2e 테스트는 `main_e2e_test.go`.
  - `internal/app` — 서비스 계층(`service.go`, `plan.go`, `effects.go`, `evalexec.go`). 런의 외부 효과 조정이 여기.
  - `internal/orchestrator`, `internal/scheduler`, `internal/planner` — 런 실행·재개·프리플라이트·쿼터, 작업 스케줄링, 아이디어 생성/중복제거.
  - `internal/policy` — 권한·역할·드리프트·재시도·게이트 실패·검증 표면 정책. 테스트 밀도가 가장 높다(파일마다 `_test.go`).
  - `internal/gitops` — 워크트리, 커밋/머지/푸시, 롤백, 조정(reconcile), diff 검사.
  - `internal/verification` + `internal/evaluation` — 게이트 실행 엔진과 평가 케이스 재실행.
  - `internal/store/sqlite` — 영속화. 파일 40여 개로 도메인별 분할(`workitems.go`, `approval.go`, `evidence.go`, `recovery.go`, …).
  - `internal/provider/{claude,codex,qwen,opencode}` — CLI 어댑터와 stream-json 이벤트 파서. 그 외 `audit`(레닥션/암호화), `notify`, `usage`, `diagnostics`, `report`, `prompt`, `model`.

- 빌드·테스트:
  - `go build ./...`
  - `go test ./...` — 전체 약 25초. 느린 패키지: `internal/store/sqlite` ~6.8s, `internal/app` ~4.3s, `internal/orchestrator` ~3.7s, `internal/api` ~3.3s.
  - 패키지 단위: `go test ./internal/gitops/ -run TestCommitVerified -v` 처럼 `-run` 스코핑이 잘 듣는다.
  - CI 없음. `.github` 안에는 `FUNDING.yml` 뿐이고 워크플로 파일이 하나도 없다 — 로컬 `go test` 가 유일한 게이트다.

- 관례: 커밋 메시지는 영어 명령형 한 줄 제목에 무엇을 바꿨는지가 아니라 **무엇이 달라지는지**를 쓴다("Stop a run from certifying its own work", "Reconcile external effects instead of repeating them"). 작업은 `improve/<주제>` 브랜치 → PR → 머지 커밋. 코드 주석은 영어 기본이고 정책적 판단이 걸린 자리에 한국어 괄호주가 섞인다(예: `gitops/commit.go` 의 `자동 병합 금지`). 자식 프로세스 환경은 `policy.SessionEnvironment(os.Environ(), policy.RoleImplementation)` 로 만든다(`internal/verification/engine.go:223`, `internal/provider/process.go:39`). 문서는 `README.md`(431줄, 명령 레퍼런스가 여기 다 있다)와 `docs/acceptance-audit.md`. 소스에 TODO/FIXME/XXX/HACK 이 단 하나도 없다 — 미해결 사항을 코드에 남기지 않는 저장소다.

- 위험 구역:
  - `internal/store/sqlite/store.go`(1136줄) 의 스키마·마이그레이션 — 건드리면 기존 프로젝트 DB 가 깨진다.
  - `internal/policy/*` — 권한 모델과 역할 분리. 최근 커밋 두 건(`ed5bae4`, `4bf0190`)이 "구현 세션에 운영자보다 낮은 권한", "런이 자기 작업을 스스로 인증하지 못하게" 를 못박았다. 완화 방향 변경은 곧 회귀다.
  - `internal/gitops/commit.go`·`rollback.go` — 사용자 저장소에 커밋/머지/푸시를 실제로 수행한다. `PushBranch` 는 명시적 사용자 승인(SEC-011) 뒤에만 호출돼야 하고 절대 force 하지 않는다.
  - `.github/` — 워크플로가 없는 상태 자체가 현재 설계다. 릴리즈·빌드 경로는 보호 경로로 취급한다.

- 자주 깨지는 곳: (회차 기록이 아직 없어 이번 회차 관찰만) git 을 실제로 실행하는 테스트가 주변 환경에 진다. 커밋 신원(`-c user.name` 이 환경변수에 밀린다)과 push(로컬 bare 대상도 차단된다) 두 축 모두.

- 검증 함정:
  - **git 실행 테스트가 주변 환경 의존**이다. 이 워크트리(main@7a17631, 트리 깨끗)에서 `go test ./...` 가 4건 실패한다: `cmd/goalforge` `TestCLIFullLifecycle`(main_e2e_test.go:114), `internal/app` `TestPublishReconcilesInsteadOfRepeating`(effects_test.go:74), `internal/gitops` 의 `TestMergeVerified…`·`TestPushBranch…`·`TestCommitVerified…`. 즉 **"전체 통과" 를 기준선으로 삼을 수 없다** — 변경 전 기준선을 먼저 찍고 비교할 것.
  - 실패 축 1: 전역 `~/.gitconfig` 의 `user.name=hkjang` 및 (추정) `GIT_AUTHOR_*` 환경변수가 `git -c user.name=GoalForge` 를 이겨 커밋 저자가 사람 이름으로 잡힌다. 전역 `core.hooksPath=/home/hkjang/.gitglobal/hooks` 도 설정돼 있다(내용 확인 못 함 — 미확인).
  - 실패 축 2: 로컬 bare 저장소로의 `git push` 가 `fatal: 'DISABLED' does not appear to be a git repository` 로 죽는다. 저장소 코드로는 재현 불가능한 형태여서 샌드박스가 push 를 막은 것으로 판단하지만 직접 확인은 못 했다(미확인).
  - 6개 테스트 파일이 각자 `git init` + 신원 설정 + `t.Skipf("git unavailable")` 를 중복 구현한다(`internal/api/setup_test.go:21`, `internal/evaluation/runner_test.go:19`, `internal/app/plan_test.go:41`, `internal/app/service_test.go:628,705`, `internal/app/effects_test.go:25`). 환경 격리가 한 곳에 없다.
  - `internal/model`, `internal/procctl`, `internal/testscript` 는 `[no test files]`.
  - 샌드박스 제약: 정찰 세션에서 `env`, `git config --global --list`, 임의 셸 스크립트 실행이 차단된다. 진단은 `go test` 와 파일 읽기로만 가능하다.
