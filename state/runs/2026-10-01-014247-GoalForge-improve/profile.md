# GoalForge 프로필 (2026-10-01, base main@0c9e799)

- 목적: 목표(goal)를 1급 시민으로 두는 개발 오케스트레이터. Go 프로세스가 프로젝트 상태·버전된 목표·작업 순서·검증 증거·완료 판정을 소유하고, AI 세션(Codex/Claude Code/Qwen/OpenCode)은 진실의 원천이 아니라 실행 도구로만 쓴다.
- 스택: Go (모듈 `github.com/goalforge/goalforge`). SQLite 주력 + Postgres 병존. HTTP API + 정적 대시보드, MCP 서버, TUI. 프런트엔드 빌드 도구 없음.

- 구조:
  - `cmd/goalforge/` — CLI 표면. `main.go`(약 4200줄) + 도메인별 분리 파일(`standards.go`, `fleet.go`, `duration_test.go`, `main_e2e_test.go`).
  - `internal/observer/` — **최근 5회차의 무게중심.** 저장소를 읽어 부족한 부분을 작업으로 공급(`supply.go`), 평가 주기·예산(`schedule.go`), 기준 판정(`verify.go`), 자동 실행 승인(`autonomy.go`), 자동 병합 승인(`automerge.go`), 무인 스윕(`tick.go`). 전부 테스트 있음.
  - `internal/standards`, `internal/patterns` — 판정 가능한 데이터로 둔 공통 개발 기준 팩, 여러 프로젝트에 걸친 패턴 승격·철회.
  - `internal/app`, `internal/orchestrator`, `internal/scheduler`, `internal/planner` — 서비스 계층, 런 실행·재개·쿼터, 스케줄링, 아이디어 생성/중복제거.
  - `internal/policy` — 권한·역할·드리프트·재시도·게이트 실패 정책.
  - `internal/gitops` — 워크트리, 커밋/머지/푸시, 롤백, reconcile.
  - `internal/store/sqlite` — 영속화, 도메인별 파일 50여 개. `internal/browser`, `internal/capture` 는 신규.
  - `scripts/build-release.sh` — 크로스컴파일 + `SHA256SUMS`. CI 가 `v0.0.0-ci` 로 매 PR 실행.

- 빌드·테스트: `go build ./...` / `go vet ./...` / `gofmt -l ./cmd ./internal`(무출력) / `go test ./... -count=1`.
  - **2026-10-01 실측: `go test ./...` exit 0 (전부 통과).** 약 60s. 느린 패키지: `internal/store/sqlite` 25.9s, `internal/observer` 13.4s, `internal/api` 8.7s, `internal/app` 8.5s, `internal/orchestrator` 6.5s, `internal/verification` 4.0s.
  - `[no test files]`: `internal/model`, `internal/procctl`, `internal/testscript` — 세 개뿐.
  - `-run` 스코핑이 잘 듣는다: `go test ./internal/observer/ -run TestTick -count=1 -v`.
  - CI: `.github/workflows/ci.yml`(push:main / PR — build·vet·test, gofmt, `go mod tidy` 드리프트, 릴리즈 아티팩트 크로스컴파일+체크섬), `.github/workflows/release.yml`(태그 `v*` — ubuntu/windows/macos 3 플랫폼 vet+test, build-provenance attestation, `gh release upload`).

- 관례: 커밋 제목은 **한국어 한 줄로 무엇이 달라지는지**를 쓴다(무엇을 바꿨는지가 아니다). 본문은 길고, 각 설계 판단에 "이렇게 하지 않으면 무엇이 문제인가" 를 붙인다 — 이 저장소의 가장 강한 관례다. `feature/…`·`fix/…`·`improve/…`·`docs/…` 브랜치 → PR → 머지 커밋. 코드 주석은 영어 기본, 정책 판단 자리에 한국어. 사용자 향 문자열은 한국어. 자식 프로세스 환경은 `policy.SessionEnvironment(os.Environ(), policy.RoleImplementation)`. 문서는 `README.md`(명령 레퍼런스), `docs/STANDARDS.md`, `docs/acceptance-audit.md`. 소스에 TODO/FIXME 없음.

- 위험 구역:
  - `internal/observer/autonomy.go`·`automerge.go` — 자동 승인 봉투. "봉투가 무엇을 허용하는지 판단하는 규칙은 `AutoApprove` 한 곳에만" 이 명시적 결정이다(같은 검사를 스윕에 복제했다가 지운 이력 있음). 여기에 규칙을 복제하는 변경은 회귀다.
  - `internal/policy/*` — 권한 모델과 역할 분리. "구현 세션에 운영자보다 낮은 권한", "런이 자기 작업을 스스로 인증하지 못하게". 완화는 곧 회귀.
  - `internal/store/sqlite/store.go` 스키마·마이그레이션 — 기존 프로젝트 DB 가 깨진다.
  - `internal/gitops/commit.go`·`rollback.go` — 실제 커밋/머지/푸시. `PushBranch` 는 명시 승인(SEC-011) 뒤에만, force 금지.
  - `.github/workflows/*` — 보호 경로. 검증을 통과시키려고 느슨하게 고치는 것 금지.

- 자주 깨지는 곳:
  - git 을 실제로 실행하는 테스트가 주변 환경에 진다. 커밋 신원 축은 `commitIdentityEnv()`(gitops/commit.go)로, push 축은 `requirePushable` 프로브로 닫혔고 **셋 다 main 에 머지되어 있다**(확인함).
  - `requirePushable` 이 `internal/gitops`, `internal/app`, `cmd/goalforge` 세 패키지에 동일 구현으로 복제되어 있다(패키지 테스트가 서로를 임포트할 수 없어 불가피).
  - 새 기능이 "계산해 두고 아무도 읽지 않는 값" 을 남기는 패턴이 반복된다 — 5b83c30 이 스스로 고백한 부류이고, `tick.go` 의 `ProjectTick.Note`(어디서도 출력되지 않음)와 버려지는 `AutoDecisions.Detail`/`Refused` 가 지금 같은 상태다.

- 검증 함정:
  - 샌드박스가 **복합 셸 명령(`&&`, `;`, 파이프)과 `gh`, `printenv`, `git config --list` 를 승인 없이 막는다.** 명령을 한 번에 하나씩 쪼개 실행할 것. PR 상태를 `gh` 로 확인할 수 없다.
  - `AutoDecisions.Refused` 는 `map[string]string` — 순회 순서에 의존한 단정은 무작위로 깨진다.
  - `internal/observer` 테스트는 실제 `*store.Store`(SQLite)와 실제 git 저장소를 쓴다. 대역 주입으로 봉투 결함을 증명하려 하면 배선을 보지 못한다.
  - 2026-09-29 회차가 "채택" 판정을 받고 구현한 `internal/model/window.go`(`ParseWindow`)가 **main 에 없다.** 머지되지 않은 회차가 있으므로 "지난 회차가 고쳤다" 를 전제하지 말고 코드에서 직접 확인할 것.
