# 과제서 — 2026-09-19 (수정 과제: 릴리즈 워크플로 실패 원인 제거)

- 과제: 릴리즈 워크플로(`.github/workflows/release.yml` build job)가 두 번 같은 이유로 실패한 원인을 찾아 코드·스크립트 쪽에서 고치기 (가치 5 / 위험 2 / 작업량 M)

- 왜: 러너 기록에 "마지막 회차 error, hold: budget — 릴리즈 워크플로가 같은 이유로 두 번 실패" 로 적혀 있다. 이 정찰 세션은 GitHub(`gh run list`)·네트워크 접근이 없어 **실패한 단계와 로그를 직접 보지 못했다(미확인)**. 따라서 이 과제서는 (1) 실패 단계를 로컬에서 재현해 특정하고, (2) 워크플로를 느슨하게 하지 않고 근본 원인을 고치는 절차서다. main@be3e9ce(v0.77.13) 기준으로 워크플로 파일·스크립트·테스트 구성은 읽었고, `go test -tags sqlite_fts5 ./internal/manifest/...` 는 통과했다(전체 `./...` 는 백그라운드로 돌렸으나 세션 종료 전 결과를 못 받음 — 미확인).

- 수용 기준:
  1) `release.yml` build job 이 로컬에서 재현 가능한 모든 단계(아래 검증 명령)가 exit 0 으로 끝난다. 실패한 단계가 무엇이었는지 journal.md 에 **명령·출력 첫 오류 줄**을 그대로 적는다.
  2) 원인이 코드·의존성·스크립트·문서 동기화에 있으면 그것을 고친다. `release.yml`·`ci.yml`·`scripts/verify-version-sync.sh`·`test/release/version-sync.test.sh` 의 검사 조건을 완화하거나 단계를 지우거나 `continue-on-error`·`|| true` 를 넣는 것은 금지.
  3) 고친 원인에 대해 실패를 재현하는 테스트(또는 기존 테스트가 잡는 것)가 있어야 한다. 수정 전에 실패하고 수정 후에 통과함을 직접 확인해 기록한다.
  4) 원장(회차 노트)에 '수정 과제' 로 기록: 어떤 단계가 왜 실패했고 무엇을 고쳤는지.

- **정찰에서 직접 관찰한 유일한 실패(가장 먼저 볼 것)**: main@be3e9ce 에서 `go test -tags sqlite_fts5 ./...` 전체 실행 시 `FAIL git-ctx/internal/app 102.539s` (다른 패키지는 전부 ok). 같은 패키지를 단독으로 `go test -tags sqlite_fts5 ./internal/app/` 다시 돌리니 ok. 즉 **전체 병렬 실행(부하) 아래에서만 깨지는 flaky 테스트**가 `internal/app` 에 있다. 실패한 테스트 이름은 출력을 필터링해 놓쳤음(미확인). 재현: `go test -tags sqlite_fts5 -count=1 ./... 2>&1 | grep -v http_request | grep -E '^(--- FAIL|panic|.*_test.go:)'` 를 2~3회, 그리고 `go test -tags sqlite_fts5 -race -count=1 ./internal/app/...`. 100초 넘게 걸린 점으로 보아 타임아웃·대기(폴링/컨텍스트 데드라인)에 의존하는 테스트가 유력 — 최근 추가된 `internal/app/mcpoauth_test.go`(가짜 IdP·JWKS 서빙)와 index/작업 큐 대기 테스트를 먼저 의심할 것. 고칠 때는 테스트의 sleep/데드라인을 늘리는 대신 실제 완료 신호를 기다리게 하거나 프로덕션 쪽 경합을 고친다; `t.Skip`·`-p 1` 강제·타임아웃 상향 금지.

- 실패 단계 특정 순서 (release.yml build job 의 순서 그대로; 앞에서 실패하면 그것이 원인):
  1. `Validate tag and source version` — 태그 `v0.77.13` 의 커밋이 `internal/version/version.go` `const Version = "0.77.13"` 과 일치하는지. `git rev-list -n 1 v0.77.13` 과 `git rev-parse HEAD` 비교. **release.sh 헤더 주석에 "검증 실패 중간에 태그가 이전 커밋에 밀려 워크플로가 잘못된 트리를 빌드한 일이 두 번 있었다"** 고 적혀 있다 — 같은 유형(태그가 release 커밋이 아닌 앞 커밋을 가리킴)인지 `git log -1 v0.77.13` 으로 먼저 확인할 것.
  2. `Verify version metadata` — `sh scripts/verify-version-sync.sh <repo-root>` 와 `sh test/release/version-sync.test.sh`. 검사 대상 8개 파일: `internal/version/version.go`, `docs/openapi.yaml`(`info.version`), `deploy/kubernetes/base/deployment.yaml`(`image: git-ctx:vX`), `docs/offline-deployment.md`(VERSION= 예시·아카이브명·이미지태그·--build-arg·verify 스크립트 인자), `docs/test-plan.md`, `docs/index.html`·`docs/index_en.html`(`"softwareVersion"`), `docs/completion-audit.md`(최신 "vX 릴리스 전 검증 결과:" 헤딩과 그 블록의 Kustomize/Docker 줄), `docs/release-notes-v0.77.13.md`(헤딩 `# git-ctx v0.77.13`, 아카이브명). 릴리즈 노트 파일은 존재함을 확인했다. 한 파일이라도 0.77.12 로 남아 있으면 그 파일을 0.77.13 으로 맞춘다.
  3. `Verify formatting, build and tests` — `gofmt -l ./cmd ./internal` 빈 출력, `go build -tags sqlite_fts5 ./...`, `go vet ./...`, `go test -tags sqlite_fts5 ./...`, `go test -tags sqlite_fts5 -race ./...`, `node --check web/{app,clients,guides,roles}.js`, `for f in test/web/*.test.js; do node "$f"; done`. `-race` 에서만 실패하면 그 테스트의 데이터 레이스를 고친다(테스트를 skip 하지 말 것).
  4. `Verify PostgreSQL, pgvector and Vault integrations` — 로컬에 서버가 없으면 skip 됨(환경변수 미설정). 재현 불가면 "미확인" 으로 적을 것.
  5. `Verify reachable dependencies` — `go install golang.org/x/vuln/cmd/govulncheck@v1.7.0 && govulncheck ./...`. **코드가 안 바뀌었는데 같은 이유로 두 번 실패했다면 이 단계가 가장 유력하다**(취약점 DB는 매일 갱신되므로 새 GO-2026-xxxx 가 도달 가능 경로에 뜨면 같은 태그가 계속 실패). 해결은 해당 모듈을 취약점이 고쳐진 **패치/마이너 버전**으로 `go get` 올리고 `go mod tidy`, 전체 테스트 재통과. 메이저 업그레이드는 금지. 네트워크가 없으면 미확인으로 남긴다.
  6. `Build the exact linux/amd64 release image` → `package-offline-image.sh` → `verify-offline-image.sh`. Docker 가 있으면 `ci.yml` docker job 의 명령 그대로 재현. Dockerfile 이 테스트를 다시 돌리고 VERSION 태그와 `internal/version` 불일치를 거부한다(미확인 — Dockerfile 본문은 안 읽음).

- 건드릴 파일 (원인에 따라 하나):
  - 버전 동기화 실패 → 위 8개 문서/설정 파일 중 어긋난 것. `internal/version/version.go` 는 바꾸지 않는다(태그와 묶여 있음).
  - govulncheck 실패 → `go.mod`/`go.sum` 만 (해당 모듈 패치 버전). 
  - 테스트/레이스 실패 → 실패한 `*_test.go` 가 가리키는 프로덕션 코드. 
  - 태그가 잘못된 커밋 → 코드 수정 없음. journal 에 "태그 v0.77.13 가 <sha> 를 가리키고 Version 이 <x>" 를 적고, 태그 재지정은 사람 작업임을 명시(태그 삭제·강제 푸시 금지).

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  ```
  sh scripts/verify-version-sync.sh "$PWD"
  sh test/release/version-sync.test.sh
  test -z "$(gofmt -l ./cmd ./internal)"
  go build -tags sqlite_fts5 ./...
  go vet ./...
  go test -tags sqlite_fts5 ./...
  go test -tags sqlite_fts5 -race ./...
  for f in web/app.js web/clients.js web/guides.js web/roles.js; do node --check "$f"; done
  for f in test/web/*.test.js; do node "$f"; done
  go install golang.org/x/vuln/cmd/govulncheck@v1.7.0 && govulncheck ./...   # 네트워크 필요
  ```

- 위험과 피할 것:
  - `.github/workflows/*.yml` 은 보호 경로 — 검사 완화·단계 삭제·타임아웃 늘리기 금지. 워크플로 파일 자체에 버그가 있다고 확신할 때만(예: 잘못된 경로) 최소 수정하고 그 근거를 journal 에 적는다.
  - `scripts/verify-version-sync.sh` 의 정규식을 넓히지 말 것 — 검사 대상 문서를 고치는 쪽으로.
  - 의존성 메이저 업그레이드 금지, `go.mod` 의 `go 1.25.0`/`toolchain go1.26.6` 은 손대지 말 것.
  - 반려 이력 PR(fbcff94·214bf01) 영역(매니페스트 경고 확장·Gradle 파서)은 이번 과제와 무관 — 손대지 말 것.
  - 확인 못 한 것: 실제 실패 단계·로그(GitHub 접근 없음), Dockerfile 본문, integration 테스트(서버 없음). 추측을 사실처럼 적지 말 것.

- 차선 후보: 로컬에서 위 모든 단계가 통과해 원인이 재현되지 않으면 — (a) `gh run view --log-failed` 로 실패 로그를 받아 원인을 기록하고, 코드 원인이 아니면(예: GitHub 서비스 컨테이너 pull 실패, 러너 예산 hold) 원장에 "코드 원인 아님, 재실행 대상" 으로 남기고 코드 변경 없이 마친다. (b) 그래도 시간이 남으면 보류 아이디어 중 `${VAR}` 플레이스홀더 오탐 마스킹(가치 3 / 위험 2 / S, `internal/contentsecurity/sanitize.go`)을 고른다 — 값이 온전히 `${…}`·`$VAR` 이면 건너뛰고 실제 값에 `${` 가 섞인 경우는 계속 가리는 테스트 포함.
