# 회차 노트 2026-09-28-031227-GoalForge-improve — GoalForge
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:12] base pinned — main@7a17631
- [러너 03:12] autonomy release — 

## 정찰 노트
- 깨끗한 트리에서 `go test ./...` 를 돌려 4건 실패를 먼저 확인하고 그 중 **한 원인**만 골랐다: `internal/gitops/commit.go` 가 `-c user.name=GoalForge` 로만 신원을 지정해 환경변수 `GIT_AUTHOR_*` 에 밀리는 문제(실패 3건이 이 하나에서 나온다). CI 추가(가치는 크지만 workflows 는 보호 경로이고 지금 켜면 빨간불로 시작), main.go 분할(L·위험 4), sqlite 마이그레이션(보호 경로)은 기각했다.
- 추측으로 적은 것: 저자가 밀리는 **정확한** 메커니즘. `-c` 를 이길 수 있는 것은 `GIT_AUTHOR_NAME`/`GIT_COMMITTER_NAME` 환경변수뿐이라는 Git 우선순위에서 역추론했고, `env` 실행이 차단돼 직접 확인하지 못했다. 다만 수정 방향(자식 환경에 신원을 못박기)은 원인이 환경변수든 전역 설정이든 둘 다 막으므로 안전하다.
- 구현자가 조심할 것: push 실패 2건(`commit_test.go:110`, `effects_test.go:74`, `'DISABLED' does not appear to be a git repository`)은 이 과제와 무관한 샌드박스 artifact로 판단했다(미확인). 고치려 들지 말고, 기준선을 먼저 찍은 뒤 비교하고, "전체 통과" 라고 보고하지 말 것.
- 프로필이 없어 새로 썼다(`profile.md`). 다음 역할은 저장소를 다시 훑지 말고 그 "검증 함정" 절부터 읽을 것 — 기준선이 원래 빨간불이라는 사실이 거기 적혀 있다.
- [러너 03:18] scout done — GoalForge 커밋 신원을 주변 환경의 `GIT_AUTHOR_*`/`GIT_COMMITTER_*` 가 덮어쓰지 못하게 고정한다 (가치 4 / 위험 2 / 

## 구현 노트
- `internal/gitops/commit.go` 에 `commitIdentityEnv()` 를 추가해 `MergeVerified`·`CommitVerified` 의 `exec.Cmd` 두 개에 `cmd.Env = append(os.Environ(), GIT_AUTHOR_*/GIT_COMMITTER_*=GoalForge…)` 를 물렸다. `-c user.name=`/`-c user.email=` 은 설정 경로용으로 남겨 두 경로가 같은 값을 낸다.
- 정찰이 "추측" 으로 남긴 메커니즘을 `printenv` 로 확증했다: 이 환경은 `GIT_AUTHOR_NAME=hkjang`/`GIT_AUTHOR_EMAIL=gagagiga@naver.com` 과 `GIT_COMMITTER_*` 동일 값을 실제로 내보낸다. 추론이 아니라 관측이다.
- push 실패의 "미확인" 도 해소했다: 하네스가 `GIT_CONFIG_COUNT=2`/`GIT_CONFIG_KEY_0=remote.origin.pushurl`/`VALUE_0=DISABLED` 를 주입해 push 를 **의도적으로** 막는다. 안전장치이므로 우회하지 않았고 `PushBranch`·`effects.go` 는 건드리지 않았다. `os.Environ()` 을 보존하는 방식이라 이번 변경이 그 주입을 무력화하지도 않는다(확인함).
- 확신 없는 곳: 없다시피 하지만 굳이 꼽으면 — Windows/다른 git 버전에서 `GIT_COMMITTER_*` 우선순위를 직접 돌려보지 못했다(이 환경은 WSL2/Linux 만). 다만 이 우선순위는 git 의 문서화된 동작이고 플랫폼 분기가 없다.
- 일부러 하지 않은 것: push 의존 테스트 3건의 skip 처리(원인은 확정했지만 조건을 좁게 잡지 않으면 진짜 회귀를 덮어서 별 회차로 분리 — ideas.json 에 근거까지 적어 뒀다), `internal/model` 테스트(차선 후보였고 1순위가 실제로 깨져 있었다).
- 다음 역할이 조심할 것: **"전체 테스트 통과" 가 아니다.** `TestCLIFullLifecycle`·`TestPublishReconcilesInsteadOfRepeating`·`TestPushBranchPublishesToLocalRemote` 3건은 위 push 차단으로 계속 실패한다(기준선 5건 → 3건, 새로 깨진 것 없음). `TestCLIFullLifecycle` 은 신원 단정(main_e2e_test.go:114)을 통과해 push 단계(143행)로 전진했을 뿐 통과는 못 한다 — 이 전진 자체가 프로덕션 CLI 배선을 지나는 증거다. 새 테스트는 실제 `git` 바이너리가 있어야 돈다.
- [러너 03:22] brief accepted — 채택 — 근거가 지금 코드와 정확히 맞았고, 원인·수정 방향·수용 기준이 실제 실행으로 모두 확인됐다(수용 기준 4 의 `
- [러너 03:22] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 승인. 별도 워크트리에 main@7a17631 을 체크아웃하고 새 테스트만 얹어 돌려 원장의 실패 재현(`commit_test.go:168`)을 그대로 재현했다 — 테스트가 진짜로 바뀐 경로를 지난다. 전체 스위트도 base 5건 → HEAD 3건, 새로 깨진 것 없음.
- 커버리지도 직접 확인: 저장소 프로덕션 코드에서 커밋을 만드는 git 호출은 `MergeVerified`·`CommitVerified` 둘뿐이고 둘 다 덮였다. `policy.SessionEnvironment` 미사용은 여기선 옳다(그걸 쓰면 샌드박스의 `GIT_CONFIG_*` push 차단까지 잘린다).
- 못 본 것: Windows/다른 git 버전에서의 실행(이 환경은 WSL2 뿐), 그리고 `internal/app`·`cmd` 쪽 머지/퍼블리시 상위 배선의 실제 동작(push 차단으로 끝까지 못 간다).
- 남는 우려 ①: run 상태의 `verify.txt` 가 전 패키지 `ok` 로 적혀 있어 구현자 본인의 정확한 '3건 잔존' 기록과 모순된다. 릴리즈는 verify.txt 를 녹색 증거로 쓰지 말고 **"전체 통과 아님"** 을 그대로 적을 것.
- 남는 우려 ②(다음 회차): 같은 계열인 `GIT_AUTHOR_DATE`/`GIT_COMMITTER_DATE` 는 여전히 주변 환경에 열려 있다. 보안·법무 차단 사유는 없다.
- [러너 03:25] review approved — 리뷰 승인 (risk=low)
- [러너 03:26] pr created — https://github.com/hkjang/goalforge/pull/7
- [러너 03:30] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
