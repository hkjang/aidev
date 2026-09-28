- 과제: `go test ./...` 를 이 저장소에서 실제로 통과시킨다 — 커밋 신원이 주변 환경에 밀리는 진짜 버그를 고치고, 샌드박스가 push 를 막았을 때 테스트가 그것을 오진하지 않게 한다 (가치 5 / 위험 2 / 작업량 M)

- 왜: 마지막 회차가 `go test ./...` (exit 1) 로 verify-failed 했고 릴리즈가 같은 이유로 두 번 막혔다. 원인은 두 축이고 둘 다 직접 재현했다 — ① GoalForge 가 만드는 커밋의 저자가 `GoalForge` 가 아니라 실행 환경의 사람 이름(`hkjang <gagagiga@naver.com>`)으로 찍힌다(프로덕션 결함) ② 하네스가 push 를 의도적으로 막아 놓은 상태에서 push 의존 테스트가 실패로 죽는다(환경 제약을 결함으로 오진). ①을 고치면 AI 변경의 귀속이 정확해지고, ②를 고치면 검증 게이트가 "GoalForge 코드가 깨졌다" 와 "이 기계는 push 를 못 한다" 를 구분한다.

- 수용 기준:
  1) 이 워크트리에서 `go test ./...` 가 exit 0. (현재 6건 실패 — 아래 "현재 실패 목록" 참고)
  2) `internal/gitops` 의 회귀 테스트가 `GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL`/`GIT_COMMITTER_*` 를 일부러 다른 값으로 세팅한 상태(`t.Setenv`)에서 `CommitVerified`·`MergeVerified` 를 실제 git 저장소에 호출해 저자·커미터가 모두 `GoalForge <goalforge@goalforge.invalid>` 임을 단정한다. 고치기 전에 이 테스트가 **실패하는 것을 먼저 눈으로 확인**하고 나서 프로덕션 코드를 고칠 것.
  3) push 의존 테스트는 push 가 가능한 환경(GitHub Actions 등)에서는 **여전히 실제로 실행되어 실제 push 회귀를 잡는다.** 스킵 판정은 문자열 `DISABLED` 매칭이 아니라 "이 환경에서 평범한 `git push` 가 되는가" 를 직접 재 보는 방식이어야 한다(아래 설계 참고).
  4) `go vet ./...`, `gofmt -l ./cmd ./internal`(무출력), `go build ./...` 통과 — `.github/workflows/ci.yml` 의 lint 잡이 이 세 가지를 본다.

- 현재 실패 목록 (main@dc28e64, 트리 깨끗, 이번 회차에 직접 실행해 확인):
  - 신원 축 3건: `cmd/goalforge` `TestCLIFullLifecycle`(main_e2e_test.go:115 `missing "GoalForge" in auto-commit`), `internal/gitops` `TestMergeVerifiedMergesCleanAndAbortsConflicts`(commit_test.go:51), `internal/gitops` `TestCommitVerifiedCreatesTrailedCommitOffProtectedBranch`(commit_test.go:178 `author="hkjang <gagagiga@naver.com>"`).
  - push 축 3건: `internal/gitops` `TestPushBranchPublishesToLocalRemote`(commit_test.go:110), `internal/app` `TestPublishReconcilesInsteadOfRepeating`(effects_test.go:74), `cmd/goalforge` `TestRestoreVerifiesRecordsAndSettlesOutsideWork`(main_e2e_test.go:576). 모두 `exit status 128: fatal: 'DISABLED' does not appear to be a git repository`.
  - `TestCLIFullLifecycle` 은 **두 축 모두**에 걸린다. 신원만 고치면 115행을 지나 push 단계(≈143행)에서 다시 죽는다. 두 축을 같이 닫아야 수용 기준 1 이 성립한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 3개):
  1. `internal/gitops/commit.go` — **프로덕션 수정은 여기 하나뿐.** 현재 `MergeVerified`(:48-52)와 `CommitVerified`(:119-123)가 `git -c user.name=GoalForge -c user.email=…` 만 쓴다. `-c` 는 **설정 파일**만 이기고 `GIT_AUTHOR_NAME`/`GIT_COMMITTER_NAME` **환경변수한테는 진다** — 그래서 사람 이름이 찍힌다. 파일 상단 상수 `commitAuthorName`/`commitAuthorEmail`(:17-20) 옆에 헬퍼를 하나 추가하라:
     ```go
     // -c user.name 은 설정 파일만 이긴다. 주변 환경이 GIT_AUTHOR_*/GIT_COMMITTER_* 를
     // 내보내고 있으면 그쪽이 이겨 GoalForge 의 커밋이 사람 이름으로 찍힌다.
     func commitIdentityEnv() []string {
         return append(os.Environ(),
             "GIT_AUTHOR_NAME="+commitAuthorName, "GIT_AUTHOR_EMAIL="+commitAuthorEmail,
             "GIT_COMMITTER_NAME="+commitAuthorName, "GIT_COMMITTER_EMAIL="+commitAuthorEmail)
     }
     ```
     그리고 `merge.Env = commitIdentityEnv()`, `commit.Env = commitIdentityEnv()` 두 줄. `os` 임포트 추가 필요(현재 `os/exec` 만 있다). **기존 `-c` 인자는 지우지 말 것** — 두 경로가 같은 값을 내므로 남겨 두는 편이 안전하다. `os.Environ()` 을 보존해야 `GIT_CONFIG_*` 안전장치와 `PATH` 가 살아 있다.
  2. `internal/gitops/commit_test.go` — (a) 수용 기준 2 의 새 테스트 추가(`t.Setenv("GIT_AUTHOR_NAME", "Someone Else")` 등 4개). (b) `TestPushBranchPublishesToLocalRemote`(:85-120, `run(bare,"init","--bare",…)` → `run(source,"remote","add","origin",bare)`) 에 push 가능성 프로브 추가.
  3. `internal/app/effects_test.go` — `effectFixture`(:20-56). remote 를 만들고 `origin` 을 붙인 직후에 프로브. 이 픽스처가 `TestPublishReconcilesInsteadOfRepeating` 을 먹인다.
  4. `cmd/goalforge/main_e2e_test.go` — 픽스처 두 곳 모두: `gitIn(t, remote, "init","--bare",…)`/`gitIn(t, repo, "remote","add","origin", remote)` 가 있는 :88-89(`TestCLIFullLifecycle`)과 :535-544(`TestRestoreVerifies…`).

- push 가능성 프로브 설계 (문자열 매칭 금지 — 이게 이 과제의 핵심 판단):
  테스트가 방금 만든 **자기 소유의 bare 저장소**로 평범한 `git push` 를 한 번 해 보고, 실패하면 `t.Skipf("이 환경은 로컬 bare 저장소로도 push 할 수 없다: %v %s", err, output)`. 성공하면 테스트를 **끝까지 정상 실행**한다. 예: `HEAD` 를 스크래치 ref 로 밀고(`git push <bare> HEAD:refs/heads/goalforge-push-probe`) 성공하면 바로 지운다(`git push <bare> :refs/heads/goalforge-push-probe`, 실패는 무시).
  - 왜 이게 게이트를 느슨하게 만드는 것이 아닌가: 프로브는 **GoalForge 코드를 거치지 않는 맨 git** 이다. 맨 git 이 되는데 `PushBranch`/`GuardEffect`/CLI 가 못 밀면 테스트는 그대로 실패한다. CI(`.github/workflows/ci.yml`, `release.yml`)의 러너는 push 가 되므로 거기서는 스킵되지 않고 전부 실행된다.
  - `remote.origin.pushurl` 값을 읽어 `"DISABLED"` 인지 비교하는 방식은 **채택하지 말 것**: 하네스의 매직 문자열에 테스트를 묶는 것이고, 다른 방식의 push 차단은 여전히 오진한다.
  - `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_*`/`remote.origin.pushurl` 을 `t.Setenv`/`git config` 로 **덮어써서 안전장치를 우회하지 말 것.** 그건 샌드박스를 뚫는 것이고 이번 과제의 목적이 아니다.

- 검증 명령 (순서대로):
  1. 기준선 먼저: `go test ./... 2>&1 | tail -40` — 위 6건이 그대로 나오는지 확인(추측 말고 실측).
  2. 신원 축: `go test ./internal/gitops/ -run 'TestCommitVerified|TestMergeVerified' -count=1 -v`
  3. push 축: `go test ./internal/gitops/ -run TestPushBranch -count=1 -v`, `go test ./internal/app/ -run TestPublishReconciles -count=1 -v`
  4. e2e: `go test ./cmd/goalforge/ -run 'TestCLIFullLifecycle|TestRestoreVerifies' -count=1 -v`
  5. 전체: `go test ./... -count=1` → **exit 0 이어야 한다.** (전체 약 40초; `internal/store/sqlite` ~17s, `internal/app` ~7s 가 느리다.)
  6. `go vet ./...` · `gofmt -l ./cmd ./internal` · `go build ./...`

- 위험과 피할 것:
  - **`.github/workflows/ci.yml`·`release.yml` 를 건드리지 말 것.** 테스트를 빼거나 `continue-on-error` 를 붙여 통과시키는 것은 금지다. 이번 수정은 워크플로 파일을 한 줄도 바꾸지 않는다.
  - `internal/gitops/commit.go` 의 `PushBranch` 안전장치를 완화하지 말 것 — SEC-011(명시적 사용자 승인 뒤에만 호출), force 금지. 시그니처·에러 문구를 바꿀 이유가 없다.
  - `internal/policy/*`(역할·권한 분리), `internal/store/sqlite/store.go`(스키마·마이그레이션)는 이번 과제와 무관하다. 열지 말 것.
  - `os.Environ()` 을 버리고 빈 `Env` 를 세팅하면 `PATH`·`HOME`·`GIT_CONFIG_*` 가 사라져 다른 테스트가 무너진다. 반드시 `append(os.Environ(), …)`.
  - 6개 테스트 파일이 각자 `git init`+신원+skip 을 중복 구현하고 있다(`internal/api/setup_test.go:21`, `internal/evaluation/runner_test.go:19`, `internal/app/plan_test.go:41`, `internal/app/service_test.go:628,705`, `internal/app/effects_test.go:25`). **이번 회차에 공통 헬퍼로 리팩터하지 말 것** — 파일 수가 불어나 회차가 사람 손을 타게 된다. 위 4개 파일만 건드린다.
  - 프로필의 "CI 없음 / `.github` 에 워크플로 없음" 은 **낡았다.** `ci.yml`·`release.yml` 이 지금 존재한다(이번 회차에 확인). 커밋 메시지는 한국어 한 줄 제목 관례(`무엇이 달라지는지`)를 따를 것 — 최근 커밋 예: `상태 데이터베이스가 무한히 자라던 것을 정리할 수 있게 한다`.
  - 미확인: 하네스가 push 를 막는 구체적 기전(`GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_0=remote.origin.pushurl`/`VALUE_0=DISABLED` 주입)은 2026-09-28 회차 원장의 기록이고, 이번 정찰 세션에서는 `printenv`·`git config --list` 가 샌드박스에 막혀 **직접 재확인하지 못했다.** 확인된 것은 에러 문구 `fatal: 'DISABLED' does not appear to be a git repository` 뿐이다. 프로브 방식은 기전을 몰라도 성립하므로 이 불확실성이 설계를 흔들지는 않는다.

- 차선 후보: 신원 축만 닫기(`internal/gitops/commit.go` + `commit_test.go` 2개 파일, 작업량 S). 진짜 프로덕션 버그이고 단독으로도 가치가 있다. 다만 이것만으로는 `go test ./...` 가 여전히 exit 1 이라 verify 는 계속 실패한다 — push 축까지 못 갈 상황이면 그 사실을 회차 노트에 명시할 것.
