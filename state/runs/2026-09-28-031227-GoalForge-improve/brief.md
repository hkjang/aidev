- 과제: GoalForge 커밋 신원을 주변 환경의 `GIT_AUTHOR_*`/`GIT_COMMITTER_*` 가 덮어쓰지 못하게 고정한다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/gitops/commit.go` 의 `MergeVerified`(50~52행)와 `CommitVerified`(121~123행)는 `git -c user.name=GoalForge -c user.email=goalforge@goalforge.invalid` 로만 신원을 지정한다. 그런데 Git 은 `GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL`/`GIT_COMMITTER_NAME`/`GIT_COMMITTER_EMAIL` 환경변수를 `user.name`·`user.email`(`-c` 로 준 것까지 포함) 보다 우선해서 읽으므로, 이 변수들이 떠 있는 환경(에이전트 하네스·CI)에서는 `-c` 가 무력화되고 AI 가 만든 커밋이 조용히 사람 이름으로 기록된다. `CommitVerified` 주석이 스스로 "so AI changes stay attributable" 이라고 적은 계약이 깨지는 자리이고, 지금 이 워크트리에서 실제로 깨져 있다(아래 증거).
  - 증거(직접 실행함): 깨끗한 트리 main@7a17631 에서 `go test ./internal/gitops/ -run TestCommitVerifiedCreatesTrailedCommitOffProtectedBranch` 가 `commit_test.go:178: author="hkjang <gagagiga@naver.com>"` 로 실패한다. 테스트는 저장소 로컬에 `user.name=GoalForge Test` 를 심고 `CommitVerified` 가 `-c user.name=GoalForge` 를 넘기는데도 저자가 전역 신원으로 잡혔다. 같은 원인으로 `internal/gitops` 의 `TestMergeVerifiedMergesCleanAndAbortsConflicts`(`%an%n%B` 로그가 `"hkjang\nMerge verified work W1..."`)와 `cmd/goalforge` 의 `TestCLIFullLifecycle`(`main_e2e_test.go:114`, `"hkjang create hello.txt..."`)도 함께 실패한다.

- 수용 기준:
  1) `MergeVerified` 와 `CommitVerified` 가 만든 커밋의 저자·커미터가 `GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL`/`GIT_COMMITTER_NAME`/`GIT_COMMITTER_EMAIL` 가 다른 값으로 설정된 환경에서도 `GoalForge <goalforge@goalforge.invalid>` 다.
  2) 새 회귀 테스트가 그것을 증명한다 — `t.Setenv` 로 위 네 변수를 딴 값(예: `Someone Else`/`someone@example.invalid`)으로 놓고, **실제 `CommitVerified`/`MergeVerified` 를 실제 git 저장소(`t.TempDir()` + `git init`)에 대고 호출**한 뒤 `git log -1 --format=%an <%ae>%n%cn <%ce>` 로 네 값 모두가 GoalForge 인지 확인한다. 손으로 만든 대역이나 git 래퍼를 끼우지 말 것.
  3) 기존 트레일러 계약은 그대로다 — 커밋 메시지에 `Goal-ID:`/`Work-Item-ID:`/`Run-ID:` 가 남고, 보호 브랜치 거부·detached HEAD 거부·`ErrMergeConflict` 시 `merge --abort` 동작이 변하지 않는다.
  4) 위 세 개의 기존 실패 테스트(`TestCommitVerifiedCreatesTrailedCommitOffProtectedBranch`, `TestMergeVerifiedMergesCleanAndAbortsConflicts`, `TestCLIFullLifecycle`)가 통과한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/gitops/commit.go` — 신원을 자식 프로세스 환경으로 못박는 작은 헬퍼를 하나 추가하고(예: `func commitIdentityEnv() []string`, `os.Environ()` 에 네 변수를 **뒤에** 덧붙여 기존 값을 이기게 한다), `MergeVerified`(50행 부근)와 `CommitVerified`(121행 부근)가 만드는 `exec.Cmd` 두 개에 `cmd.Env = commitIdentityEnv()` 를 설정. 기존 `-c user.name=`/`-c user.email=` 인자는 남겨도 무해하니 굳이 지우지 않아도 된다(설정 파일 경로도 계속 덮어 준다). `os` 를 import 에 추가해야 한다.
  - `internal/gitops/commit_test.go` — 수용 기준 2)의 회귀 테스트를 새 함수로 추가. 기존 테스트는 그대로 두고 옆에 붙일 것.
  - 참고(읽기만): `internal/verification/engine.go:223`, `internal/provider/process.go:39` 가 `policy.SessionEnvironment(os.Environ(), ...)` 로 자식 환경을 만드는 이 저장소의 기존 관례다. gitops 에 맞는 게 있으면 재사용을 고려하되, 없으면 gitops 안에 두고 policy 패키지를 건드리지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go build ./...`
  - `go test ./internal/gitops/ -run 'TestCommitVerified|TestMergeVerified' -v`  ← 1차 증명
  - `go test ./internal/gitops/ ./cmd/goalforge/`  ← `TestCLIFullLifecycle` 까지
  - `go test ./...`  ← 전체. 약 25초. **주의: 아래 "위험" 의 두 건은 이 과제와 무관하게 실패한다.**

- 위험과 피할 것:
  - **이 환경에서 `git push` 는 이 과제와 무관하게 깨져 있다.** `internal/gitops` 의 `TestPushBranchPublishesToLocalRemote`(`commit_test.go:110`)와 `internal/app` 의 `TestPublishReconcilesInsteadOfRepeating`(`effects_test.go:74`)은 로컬 bare 저장소로 push 하는데도 `fatal: 'DISABLED' does not appear to be a git repository` 로 실패한다. 로컬 경로 push 를 저장소 코드나 `~/.gitconfig` 로는 그렇게 만들 수 없으므로 **샌드박스가 git push 를 막아 놓은 환경 artifact로 판단**한다(직접 확인은 못 함 — 프로브 스크립트 실행이 차단됐다. 미확인). 이 두 개를 "고치려" 들지 말고, `PushBranch` 와 `internal/app/effects.go` 는 건드리지 말 것. 최종 보고에 "전체 테스트 통과" 라고 쓰지 말고 이 두 건은 환경 사유로 남았다고 그대로 적을 것.
  - 이 저장소에는 `.github/workflows` 가 없다(`.github` 안에 `FUNDING.yml` 뿐). CI 워크플로를 새로 만들지 말 것 — 이번 과제 범위가 아니고 보호 경로다.
  - `-c` 인자를 지우고 환경변수만 남기는 식으로 바꾸지 말 것. 두 경로(설정 파일 신원과 환경변수 신원)가 같은 값을 내야 하고, 한쪽만 손대면 다른 환경에서 갈린다.
  - `cmd.Env` 를 설정할 때 `os.Environ()` 을 빼먹고 네 변수만 넣으면 자식 git 이 `PATH`·`HOME` 을 잃는다. 반드시 `append(os.Environ(), ...)` 형태로.
  - 스코프를 `internal/gitops` 밖으로 넓히지 말 것. `rollback.go`·`reconcile.go`·`worktree.go` 에는 신원을 지정하는 커밋 경로가 grep 상 없었다(`user.name=` 은 `commit.go` 2곳뿐) — 커밋을 만드는 다른 경로를 발견하면 이번 회차에 같이 고치지 말고 아이디어로 남길 것.

- 차선 후보: **`internal/model` 패키지에 테스트 추가** — `internal/model/model.go` 가 프로젝트 전체의 핵심 타입을 담고 있는데 `[no test files]` 다(`go test ./...` 출력으로 확인). 상태 전이·검증 헬퍼 같은 순수 함수가 있으면 그 계약을 고정하는 테스트를 넣는다. 1순위가 이미 해결돼 있거나 환경 탓으로 증명이 불가능할 때 이것을 고를 것.
