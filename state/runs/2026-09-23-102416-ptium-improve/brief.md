# 과제서 — 2026-09-23 (ptium)

> 먼저 사실 정리(우선 과제 "릴리즈 실패" 의 실제 모습):
> - 러너가 준 실패 사유는 `interrupted before PR` 다. 이 저장소의 `.github/workflows/` 에는 **ci.yml 하나뿐이고 릴리즈 워크플로는 없다**(확인함). 릴리즈는 GitHub Actions 가 아니라 `scripts/release.sh` → `scripts/build-offline.sh` 를 사람이/러너가 돌리는 구조다.
> - 따라서 "워크플로 파일이 깨졌다" 는 형태의 결함은 **없다**. `interrupted before PR` 은 러너가 PR 단계 전에 끊겼다는 러너 쪽 상태이고, 저장소 코드로 재현·수정할 수 없다(미확인이 아니라 해당 없음).
> - 대신 **릴리즈가 두 회차째 나가지 못한 상태는 저장소에서 확인된다**: HEAD `e529843` 기준 `VERSION` 은 아직 `1.69.43`(= 마지막 태그 `4b552d1 Release 1.69.43`)인데 그 뒤로 PR #28(8739ec8), #30(80ebe4c) 두 개가 main 에 들어와 있다. 즉 릴리즈 단계가 두 번 연속 아무것도 내보내지 못했다.
> - 릴리즈 단계가 늦게, 비싸게 실패하는 실제 구멍을 하나 확인했다(아래 과제). 검사를 **느슨하게 하지 않고 더 이르게·더 넓게** 만드는 방향이다.

- 과제: 버전을 반만 올린 릴리즈를 `make test` 단계에서 잡기 — 릴리즈 스크립트가 도커 빌드를 다 돌린 뒤에야, 그것도 한 줄만 보고 통과시키는 것을 고친다 (가치 4 / 위험 1 / 작업량 S)

- 왜: 릴리즈 버전은 `VERSION` 말고도 `api/openapi.yaml`·`deploy/kubernetes.yaml`·`docs/offline-deployment.md` 세 곳에 박혀 있는데, 이 어긋남을 보는 것은 `scripts/release.sh:40-42` 뿐이고 그마저 `offline-deployment.md` 는 `ptium-$version.tar.gz` **한 패턴만** grep 한다 — 설치 안내문이 `docker image inspect ptium:1.69.43`(52-53행)처럼 옛 버전을 그대로 가리켜도 릴리즈는 통과한다. 이 파일이 번들에 그대로 복사되는 성격의 문서라, 스크립트 주석이 적어 둔 사고(1.47.0 번들이 0.11.0 을 이름)와 같은 종류다. 게다가 이 검사들은 `git push`/태그 직전, 즉 `docker buildx build` + `docker save | gzip -9` + firstrun/upgrade 컨테이너를 모두 돌린 **뒤**가 아니라 앞에 있지만, 그 실패를 CI·`make test` 는 전혀 못 본다. 스탬프 어긋남을 평소 테스트에서 잡으면 릴리즈 단계는 남은 시간을 진짜 빌드에 쓸 수 있다.

- 수용 기준:
  1) `cd server && go test ./internal/config` 가, `VERSION` 만 `1.69.44` 로 바꾸고 세 파일을 그대로 두면 **실패**하고, 실패 메시지가 어느 파일의 몇 행이 어떤 버전을 말하는지 적는다(손으로 `VERSION` 을 잠깐 고쳐 red 를 확인한 뒤 되돌릴 것).
  2) 지금 HEAD 그대로면 통과한다 — 세 파일 모두 `1.69.43` 임을 정찰에서 확인했다(`api/openapi.yaml:6323`, `deploy/kubernetes.yaml:78`, `docs/offline-deployment.md` 12곳).
  3) `scripts/release.sh` 의 `offline-deployment.md` 검사가 `ptium-$version.tar.gz` 한 줄이 아니라 **그 문서가 이름 붙인 모든 `ptium-<버전>`·`ptium:<버전>` 이 $version 인지**를 보고, 하나라도 다르면 `fail` 한다. 지금 코드에 같은 상황을 넣으면 통과하고, 고친 뒤에는 멈추는 것을 손으로 확인해 회차 요약에 적는다.
  4) `cd server && go test -race ./... && go vet ./...` 전부 통과.

- 건드릴 파일:
  - `server/internal/config/` 에 새 테스트 파일 한 개(예: `stamped_test.go`, `package config`). **이웃한 `internal/config/shipped_test.go` 의 방식을 그대로 따를 것** — 그 파일이 이미 `root` 로 저장소 루트를 잡아 `deploy/kubernetes.yaml`·`.env.offline.example` 를 `os.ReadFile` 로 읽는다(`told_test.go:34` 의 `"../../../README.md"` 식 상대 경로도 같은 관례). 새 헬퍼를 만들지 말고 있는 것을 쓸 것.
    - 읽을 것: 루트 `VERSION`(`strings.TrimSpace`), `api/openapi.yaml` 의 `service.version: <v>`, `deploy/kubernetes.yaml` 의 `image: ptium:<v>`, `docs/offline-deployment.md` 의 `ptium-<v>`·`ptium:<v>` **전부**.
  - `scripts/release.sh:42` — `docs/offline-deployment.md` 의 단일 grep 을 위 규칙으로 바꾼다(`grep -nE` 로 다른 버전을 말하는 줄을 찾아 있으면 `fail`, 그 줄을 출력). 40·41행은 그대로 두어도 된다.

- 검증 명령:
  - `cd server && go test ./internal/config` (red→green 을 위 1)·2) 순서로)
  - `cd server && go test -race ./... && go vet ./...`
  - `bash -n scripts/release.sh` (문법), 그리고 3) 의 손 확인: 임시 복사본에서 `offline-deployment.md` 의 한 줄을 옛 버전으로 바꿔 스크립트의 해당 검사 구간만 떼어 돌려 볼 것. **`scripts/release.sh` 를 끝까지 실행하지 말 것**(89-92행이 실제로 `git push`·`gh release create` 를 한다).
  - `git diff --check`

- 위험과 피할 것:
  - **`.env.offline.example` 과 `docker-compose.offline.yml` 은 테스트 대상에 넣지 말 것.** 이 둘은 `build-offline.sh:45-48` 이 빌드 때 `sed` 로 다시 스탬프하는 템플릿이라 저장소 안의 값이 옛 버전인 것이 설계다. 여기까지 검사를 넓히면 있지도 않은 결함으로 CI 를 빨갛게 만든다.
  - **`docs/offline-deployment.md` 의 맨 버전 문자열(261·277·286행의 `1.52.1`, `1.9.0`/`1.10.0`/`0.1.0`, `1.11.0`)은 옛 릴리즈를 일부러 가리키는 서술이다.** `ptium-` / `ptium:` 접두가 붙은 것만 매칭할 것 — 정규식을 `\d+\.\d+\.\d+` 로 넓히면 즉시 오탐이다.
  - `VERSION`·릴리즈 노트·`api/openapi.yaml` 의 버전 값을 **올리지 말 것**. 이번 과제는 검사만 더한다(버전 올리기는 릴리즈 역할의 몫).
  - `.github/workflows/ci.yml` 은 건드리지 말 것(보호 경로). 새 테스트는 `go test ./...` 에 저절로 들어가므로 CI 수정이 필요 없다.
  - `scripts/release.sh` 의 검사를 **지우거나 약하게 하지 말 것** — 이번 변경은 전부 조이는 방향이어야 한다.
  - `docker` / `kubectl` / `PyYAML` 이 이 호스트에 있는지는 **미확인**(권한으로 확인하지 못함). `build-offline.sh` 를 돌리는 과제가 아니므로 문제되지 않지만, 아래 차선을 고른다면 먼저 확인할 것.

- 차선 후보: **`build-offline.sh:73-103` 의 매니페스트 검사가 "확인할 수 없었다" 와 "매니페스트가 잘못됐다" 를 섞는 것 고치기** — `python3` 이 아예 없으면 note 한 줄 남기고 넘어가는데(101-103행), `python3` 은 있고 `PyYAML` 이 없으면 `import yaml` 이 터져 `"The Kubernetes manifest in this bundle is not valid."` 로 릴리즈를 멈춘다. 매니페스트는 멀쩡한데 릴리즈가 죽는 자리이고, 바로 위 주석(64-71행)이 "잠든 minikube 는 깨진 매니페스트가 아니다" 라고 같은 교훈을 적어 두었다. 고칠 때는 디코드에 성공했는데 필드가 없는 경우(진짜 실패)만 `exit 1` 로 남기고, 해석기를 못 돌린 경우만 note 로 내릴 것 — 이건 느슨하게 만드는 것이 아니라 101-103행과 같은 계약으로 맞추는 것이다. 단, **이 호스트에 PyYAML 이 실제로 없는지 먼저 확인하고**, 있으면 이 수정은 아무 동작도 바꾸지 않으므로 고르지 말 것(운영자 규칙: 효과 없는 변경 금지).
