- 과제: 릴리즈 일관성 검사가 배포 산출물의 `agenthub-base:v<BASE_VERSION>` 태그도 지킨다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/buildinfo/release_version_test.go` 의 `TestReleaseVersionIsConsistentAcrossDeploymentSurfaces` 는 `agenthub:v<VERSION>` 을 7개 배포 표면에서 지키지만, 같은 파일들에 손으로 박힌 **base 이미지 태그**는 어느 테스트도 보지 않는다 — 저장소에서 `agenthub-base:v<숫자>` 를 박아 둔 배포 산출물은 정확히 두 곳(`deploy/kubernetes/ollama-bridge.yaml:19` 의 `image: agenthub-base:v0.27.0`, `README.md:173` 의 `minikube image load agenthub-base:v0.27.0`)이고 둘 다 보초가 없다(grep 으로 전수 확인). BASE_VERSION 상향은 자율 회차에서 실제로 일어나므로(bb0c720 이 0.26.0→0.27.0), 다음 상향 때 릴리즈 커밋이 이 두 줄을 빠뜨리면 minikube/오프라인 설치가 **존재하지 않는 태그**를 끌어와 bridge 파드가 ImagePullBackOff 로 죽는다. 기존 보초(`scripts/release-catalog-images.sh` 의 `check_version_mirrors`)로는 덮을 수 없다 — 그것은 카탈로그의 `controlBuildArg`·`buildInfoSymbol` 두 필드만 읽고(`runtime-images.json:26-27`), 배포 매니페스트를 가리키는 필드가 카탈로그에 없다.

- 수용 기준:
  1) `deploy/kubernetes/ollama-bridge.yaml` 또는 `README.md` 가 현재 `BASE_VERSION` 과 다른 base 태그를 들고 있으면 `go test ./internal/buildinfo` 가 **어느 파일이 어떤 문자열을 빠뜨렸는지 이름을 대고** 실패한다(기존 checks 루프의 `%s does not reference release %s through %q` 와 같은 밀도의 메시지).
  2) 지금 저장소(BASE_VERSION=0.27.0, 두 파일 모두 v0.27.0)에서는 통과한다 — 이 변경으로 새로 깨지는 테스트가 없다.
  3) 돌연변이로 보초가 실제로 일함을 보인다: ① `ollama-bridge.yaml` 의 태그만 `v0.26.0` 으로 되돌리면 그 파일만 지목하며 실패 ② `README.md:173` 만 되돌리면 그 파일만 지목하며 실패 ③ `BASE_VERSION` 만 `0.28.0` 으로 올리면 **두 파일 모두** 지목된다(= 실제 사고의 모양). 세 변이 모두 되돌리고 `git status` 로 원상 확인.
  4) `BASE_VERSION` 의 형식도 `VERSION` 과 같은 수준으로 확인한다 — 읽어서 `strings.TrimSpace` 한 뒤 기존 `VERSION` 과 같은 semver 정규식으로 검사하고, 아니면 `t.Fatalf`(bb0c720 때 `0.26.0-dev` 같은 접미사 붙은 값이 buildinfo 기본값에 실제로 있었으므로 형식 가정을 적어 둘 것).

- 건드릴 파일 (프로덕션 0개, 테스트 1개):
  - `internal/buildinfo/release_version_test.go:TestReleaseVersionIsConsistentAcrossDeploymentSurfaces` — 11-20행에서 `VERSION` 을 읽는 것과 **같은 방식**으로 `BASE_VERSION` 도 읽고(같은 `root`, `os.ReadFile`, `TrimSpace`, 같은 semver 정규식 — 정규식은 지역 변수로 한 번만 컴파일), 22-33행의 `checks` 슬라이스에 두 항목을 더한다:
    - `{"deploy/kubernetes/ollama-bridge.yaml", []string{"image: agenthub-base:v" + baseVersion}}`
    - `README.md` 는 이미 27행에 항목이 있으므로 **새 항목을 만들지 말고** 그 `want` 슬라이스에 `"minikube image load agenthub-base:v" + baseVersion` 을 덧붙인다(한 파일 한 항목이라는 기존 모양을 지킬 것).
  - 주석: 이 저장소 관례대로 "왜 이 자리인가" 를 산문으로 — 두 버전이 왜 따로 움직이는지(`internal/runtime/spawner.go:38` 과 `internal/api/respond.go:87` 의 기존 주석이 이미 그 이유를 적어 두었으니 그것과 어긋나지 않게), 그리고 왜 "모든 `agenthub-base:v*` 가 최신" 이 아니라 **이 두 배포 경로만** 보는지(아래 위험 참조)를 적을 것.

- 검증 명령 (이번 회차에 실제로 돌려 본 것부터):
  - `go test -count=1 -run TestReleaseVersionIsConsistentAcrossDeploymentSurfaces ./internal/buildinfo` — 이번 정찰에서 현재 코드로 `ok … 0.002s` 확인. 돌연변이 확인은 이 빠른 명령으로 할 것.
  - `go test -count=1 ./internal/buildinfo` — **주의: 로컬에서 17~46초** 걸린다(같은 패키지의 과거 커밋 재생 테스트가 9p gitdir 의 `git show` 를 쓴다. CI 에는 없는 환경 비용이고 저장소 결함이 아님 — 측정으로 확정됨).
  - `go build ./...` · `go vet ./internal/buildinfo` · `gofmt -l internal/buildinfo` · `git diff --check`
  - `go test -race -p 1 -count=1 ./cmd/... ./internal/...` (DSN 없으면 live skip)
  - `bash scripts/release-catalog-images.sh validate` 와 `check-versions` — **버전 상향은 필요 없을 것이다**: 바꾸는 파일이 `internal/buildinfo/` 하나뿐이고 `runtime-images.json` 의 sourcePaths 에 그 경로가 있는지는 이번에 **확인하지 않았다(미확인)**. 커밋한 뒤 `check-versions` 를 한 번 돌려 아무 이미지도 지목되지 않는지 직접 볼 것 — `check_versions` 는 `${GITHUB_SHA:-HEAD}` 의 **커밋**을 읽으므로 작업 트리 상태로는 재현되지 않는다(스크립트 665-698행).

- 위험과 피할 것:
  - **"모든 `agenthub-base:v*` 태그가 최신" 은 틀린 규칙이다.** 저장소에는 픽스처용 가짜 태그가 많다 — `internal/operator/controller_test.go`·`policygate_test.go`·`gpu_test.go`, `internal/runtime/spawner_baseimage_test.go`, `internal/runtimespec/image_test.go`, `web/scripts/custom-runtime-e2e.mjs` 가 `agenthub-base:v0.4.0` 류를 들고 있다. 전수 스캔이나 정규식 `agenthub-base:v[0-9.]+` 로 저장소를 훑지 말고, **경로 두 개를 명시한 리터럴 checks 항목**으로만 좁힐 것(기존 테스트가 이미 그 모양이다).
  - `deploy/kubernetes/ollama-bridge.yaml`·`README.md` 의 **값은 바꾸지 말 것** — 지금 둘 다 v0.27.0 으로 맞다(이번에 확인). 배포 매니페스트 버전 갱신은 릴리즈 세션의 일이라는 절대 규칙이 있고, 이 과제는 보초를 세우는 것이지 값을 고치는 것이 아니다.
  - 기존 7개 `checks` 항목을 재배열·수정하지 말 것. 더하기만 한다. 특히 26행 `compose.yaml` 의 `VERSION: ` 검사는 `BASE_VERSION: ` 과 부분 문자열로 겹치지 않는지 확인은 되지만(`VERSION: 0.264.0` vs `BASE_VERSION: 0.27.0`), 그 줄은 건드릴 이유가 없다.
  - 보호 경로(`.github/workflows`, `internal/store/migrations`, auth/세션)는 건드리지 않는다. 이 과제는 애초에 거기 닿지 않는다.
  - `scripts/release-catalog-images.sh` 를 넓히려는 유혹을 피할 것 — 같은 종류의 검사가 두 경로(쉘 게이트와 Go 스위트)에 이미 나뉘어 있고, 운영자 지시는 "여러 경로의 같은 종류 파서는 통합하지 말고 좁히는 방향으로만" 이다. 0fd2810 이 쉘 쪽을 맡았으니 이번 것은 Go 쪽에 둔다.

- 차선 후보: **복원 실패가 `guide-shots.mjs` 의 problems 요약 출력을 건너뛰게 하는 것을 고친다 (2/1/S)** — `web/scripts/guide-shots.mjs:129` 의 `await` 가 throw 하면 :138-143 의 요약과 `exitCode=1` 이 건너뛰어진다. 단, `guide-settings-check.test.mjs` 가 guide-shots.mjs 의 source 를 `requestTimeoutMs` 부터 `if (problems.length)` 까지 **잘라서** 실행하므로 그 두 앵커 줄을 옮기면 테스트 경계가 같이 움직인다. (이번 정찰에서 재확인하지 않음 — **미확인**, 구현자가 먼저 두 파일을 열어 볼 것.)
