## 설치

```sh
tar -xzf goalforge_v0.19.0_linux_amd64.tar.gz
sha256sum -c SHA256SUMS --ignore-missing
./goalforge version
```

| 플랫폼 | 파일 |
| --- | --- |
| Linux x86_64 | `goalforge_v0.19.0_linux_amd64.tar.gz` |
| Linux arm64 | `goalforge_v0.19.0_linux_arm64.tar.gz` |
| macOS Intel | `goalforge_v0.19.0_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `goalforge_v0.19.0_darwin_arm64.tar.gz` |
| Windows x86_64 | `goalforge_v0.19.0_windows_amd64.zip` |
| Windows arm64 | `goalforge_v0.19.0_windows_arm64.zip` |

---

## GoalForge 가 만든 커밋이 운영자 본인 이름으로 찍히고 있었습니다

코드는 신원을 고정하고 있었습니다. 병합과 커밋을 만드는 두 지점 모두 `git -c user.name=GoalForge -c user.email=goalforge@goalforge.invalid` 을 넘겼습니다. 의도는 분명합니다 — 자동화가 만든 커밋은 자동화의 이름으로 남아야 하고, 나중에 이력을 읽는 사람이 "이건 사람이 판단한 것"과 "이건 게이트가 통과시킨 것"을 구별할 수 있어야 합니다.

그런데 `-c` 는 **설정 파일만** 이깁니다. `GIT_AUTHOR_NAME`·`GIT_COMMITTER_NAME` 같은 환경변수에는 집니다. git 의 우선순위에서 환경변수가 `-c` 보다 위이기 때문입니다.

결과는 이렇습니다. GoalForge 를 CI 안에서, 또는 커밋 신원을 환경변수로 넘기는 래퍼 아래에서 돌리고 있었다면 — 흔한 구성입니다 — GoalForge 가 자동으로 만든 병합 커밋과 검증 커밋의 저자·커미터가 **그 환경에 설정된 사람 이름**으로 기록되어 왔습니다. 고정했다고 믿은 신원이 조용히 밀려나 있었고, 감사 이력은 사람이 하지 않은 일을 사람 이름으로 남기고 있었습니다.

**이미 만들어진 커밋은 이 릴리즈가 고쳐 주지 않습니다.** 자동 병합을 써 왔다면 `git log --format='%an <%ae> %s'` 로 GoalForge 가 만든 커밋의 저자를 한 번 확인해 보시기를 권합니다.

## 무엇이 달라졌는가

`internal/gitops/commit.go` 가 이제 신원을 **환경변수 수준에서** 고정합니다. `MergeVerified` 와 `CommitVerified` 는 자식 git 프로세스에 `GIT_AUTHOR_NAME`·`GIT_AUTHOR_EMAIL`·`GIT_COMMITTER_NAME`·`GIT_COMMITTER_EMAIL` 을 직접 실어 보냅니다. 주변 환경이 무엇을 설정해 두었든 GoalForge 가 만든 커밋은 `GoalForge <goalforge@goalforge.invalid>` 입니다.

회귀 테스트가 네 가지를 각각 단정합니다 — 두 함수 × 저자·커미터. 수정 전에 이 네 단정이 모두 실패하는 것을 먼저 확인했습니다:

```
commit_test.go:163: CommitVerified author="Someone Else <someone@example.invalid>",
    want "GoalForge <goalforge@goalforge.invalid>"
```

## 테스트가 환경 때문에 꺼지면서 push 회귀까지 함께 꺼지고 있었습니다

같은 수정에서 두 번째 문제를 닫았습니다. push 가 막힌 환경에서 테스트 6건이 실패하고 있었고, 이것을 "막혔으면 건너뛴다" 로 처리하면 게이트 자체가 느슨해집니다 — push 를 정말로 깨뜨린 변경도 조용히 통과합니다.

그래서 문자열을 보고 건너뛰는 대신, 각 테스트가 **방금 만든 자기 소유의 bare 저장소로 맨 git push 를 한 번 해 보는** 능력 프로브를 씁니다. GoalForge 코드를 거치지 않으므로, 프로브는 환경의 능력만 재고 테스트 대상 코드는 면제하지 않습니다. 양쪽을 실측해 확인했습니다:

- push 가 되는 환경에서는 6건이 **건너뛰지 않고 전부 실행되어 통과**합니다. CI 에서는 스킵되지 않습니다.
- 같은 환경에서 `PushBranch` 를 일부러 깨뜨리면 스킵이 아니라 **실패**합니다 (`src refspec ... does not match any`).

## 검증

`go test ./... -count=1` **exit 0** — 기준선 6건 실패에서 0건으로. `go vet ./...`, `gofmt -l ./cmd ./internal` 무출력, `go build ./...`, `go mod tidy` 드리프트 없음. 릴리즈 산출물은 여섯 플랫폼 전부 교차 컴파일되고, 리눅스 아카이브에서 꺼낸 바이너리가 자기 버전을 보고하며 `SHA256SUMS` 가 맞습니다.

산출물의 출처는 다음으로 검증할 수 있습니다:

```sh
gh attestation verify goalforge_v0.19.0_linux_amd64.tar.gz --repo hkjang/goalforge
```
