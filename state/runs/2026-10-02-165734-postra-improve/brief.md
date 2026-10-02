- 과제: CI·릴리즈 양쪽에 공통으로 들어 있고 지난 두 회차가 **한 번도 로컬에서 돌리지 않은** 게이트를 재현해, 같은 이유로 반복 실패하는 원인을 고친다 (수정 과제 / 가치 5 / 위험 2 / 작업량 M)

- 왜: 마지막 회차가 `verify-failed` 로 끝났고 배정된 사유는 "CI failed, PR open #31 / 릴리즈 워크플로가 같은 이유로 두 번 실패" 다. 지난 두 회차의 원장은 로컬 검증 목록을 매우 상세히 남겼고, 그 목록에 **빠져 있는 게이트가 정확히 무엇인지** 역으로 특정할 수 있다(아래 1~8). 그 중 CI 와 릴리즈 **두 워크플로에 동시에 들어 있는** 단계가 "같은 이유로 두 번 실패" 를 설명하는 유일한 집합이므로 거기서부터 재현한다.

- 정찰이 확인하지 못한 것(먼저 읽을 것): 이번 정찰 세션은 **Bash 실행 권한이 거부**되어(`gh run list`, `go version`, `npm ci` 모두 승인 거부) 실패 로그도, 로컬 재현도 하지 못했다. 따라서 "어느 단계가 실패했는가" 는 **미확인**이며, 아래는 워크플로 YAML·`web/package.json`·`go.mod`·`Dockerfile` 을 읽어 좁힌 순위다. 구현자는 **추측을 고치지 말고 먼저 재현**하라. `gh run list --limit 20` / `gh run view <id> --log-failed` 가 승인되면 그것이 가장 빠른 길이다(이 저장소에서 `gh` 는 매번 승인 필요).

- 사실로 확인한 것(이번 정찰에서 파일로 읽음):
  - 로컬 main = `ef36024`, 태그 `v0.25.2` 가 **같은 커밋**을 가리킨다. PR #31(`03c4f18`)은 이미 머지됐다 — 배정 사유의 "PR open" 은 저장소 기준으로 낡은 문구다.
  - `go.mod`: `go 1.25.0` / `toolchain go1.26.6`, otel 전부 `v1.45.0`(직접·간접 일관), `Dockerfile` 의 `golang:1.26.6` 과 `ci.yml`/`release.yml` 의 `GO_VERSION: "1.26.6"` 이 서로 맞다 → **툴체인 불일치는 원인이 아니다**.
  - `7ae6372` 의 `go.mod` diff 는 간접 의존성까지 함께 올라간 정상적인 `go get` 결과 모양이다 → `go mod tidy` 드리프트 가능성은 낮지만 **미검증**(1번으로 확인).
  - CI 의 `frontend` 잡은 `build-test`·`browser-e2e` 의 `needs` 다 → frontend 가 죽으면 CI 전체가 실패로 보인다.
  - **두 워크플로에 공통인 단계**: `npm ci` → `npm run build` → `internal/transport/spa/assets` 드리프트 검사 → `npm sbom --sbom-format=cyclonedx --omit=dev --package-lock-only` → (CI `sbom` 잡 / 릴리즈 `Generate SBOM`) `cyclonedx-gomod@v1.10.0`.
  - 앞 세 개는 지난 회차가 로컬에서 exit 0 으로 확인했다(원장에 출력까지 남음). **남는 공통 단계는 두 개의 SBOM 생성뿐이다** — 이것이 1순위 가설의 근거다.

- 재현 순서(위에서부터, 각각 그대로 복사해 실행. 모두 `/home/hkjang/.cache/auto-improve-wt/postra` 에서):
  1. Go SBOM — CI `sbom` 잡 + 릴리즈 `Generate SBOM` 공통, 지난 두 회차 미실행:
     `go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0`
     `cyclonedx-gomod app -json -output /tmp/sbom.cdx.json -main cmd/postra .`
     (`go install` 자체가 Go 1.26 에서 깨지는 경우와, 설치는 되지만 `app` 이 새 `go` 출력 파싱에 실패하는 경우를 구분해서 적어라.)
  2. 브라우저 SBOM — 같은 두 워크플로 공통:
     `cd web && npm ci --no-audit --no-fund && npm sbom --sbom-format=cyclonedx --omit=dev --package-lock-only > /tmp/fe-sbom.cdx.json`
  3. `npm audit --omit=dev --audit-level=high` (web 에서) — 지난 두 회차는 `--no-audit` 로만 설치했으므로 **한 번도 돈 적이 없다**. 외부에 새 advisory 가 뜨면 재실행마다 똑같이 실패한다("같은 이유로 두 번").
  4. `go install golang.org/x/vuln/cmd/govulncheck@v1.6.0 && govulncheck ./...` — `7ae6372` 가 노린 게이트이므로 실제로 비었는지 확인.
  5. `go mod tidy` 후 `git diff --exit-code go.mod go.sum` (확인 뒤 `git checkout -- go.mod go.sum`).
  6. `docker build -t postra:ci .` — 어느 회차도 돌린 적이 없다(도커 없으면 "미실행" 으로 적고 넘어가라).
  7. `cd web && npm run build` 후 `git status --porcelain --untracked-files=all -- internal/transport/spa/assets` 가 빈지 재확인(지난 회차는 통과했다 — 회귀 확인용).
  8. 릴리즈 전용: `go build -o /tmp/postra ./cmd/postra && go run scripts/mkimage.go /tmp/postra /tmp/postra.tar postra:0.25.2 v0.25.2 $(git rev-parse HEAD)`.
  - 2~8 중 실패가 나오면 **거기서 멈추고** 그 단계만 고쳐라. 1~8 전부 통과하면 "배정된 실패는 저장소 기준으로 낡았다" 를 출력으로 증명하고 **차선 후보**로 넘어가라.

- 수용 기준:
  1) 실패한 단계의 **실제 명령 출력**(에러 문구 포함)이 보고서에 그대로 남는다 — 어느 게이트인지 추측이 아니라 출력으로 특정된다.
  2) 수정은 **원인 쪽**이다. 금지: 게이트 삭제·`continue-on-error`·`|| true`·`--audit-level` 완화·`exclude`·`-skip`·`--passWithNoTests`·드리프트 검사 약화. 도구 버전을 **동작하는 버전으로 올려 핀하는 것은 허용**(느슨해지는 것이 아니라 핀의 갱신이다).
  3) 수정 후 같은 명령이 exit 0 이고, 그 단계가 CI·릴리즈 **양쪽 YAML 에서 같은 값으로** 맞춰져 있다(한쪽만 고치면 릴리즈가 또 깨진다 — 운영자 지시: 릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과를 확인할 것).
  4) 기존 Go·프런트 검증이 회귀하지 않는다: `go build ./... && go vet ./...`, `go test -race -count=1 ./...` 실패 0, `make lint`, `go run ./cmd/postra-contracts -check`, `cd web && npm run typecheck && npm test`(50 files / 426 tests) 가 모두 통과하고 `git status --porcelain --untracked-files=all -- internal/transport/spa/assets` 가 비어 있다.
  5) 가설 1(SBOM 도구)이 맞으면 테스트가 아니라 **명령 출력**이 증거다 — 이 단계에는 단위 테스트를 새로 만들지 마라(소스 문자열 grep 을 증거로 제출하는 것도 금지).

- 건드릴 파일(실패 단계가 확정된 뒤, 최소로 — 셋 이하를 목표로):
  - `.github/workflows/ci.yml` — 실패한 잡의 해당 스텝만(예: `sbom` 잡의 `cyclonedx-gomod` 설치 버전, `govulncheck` 핀).
  - `.github/workflows/release.yml` — 같은 스텝의 쌍(`Generate SBOM` / `Generate browser SBOM`). 두 파일을 같은 값으로 맞출 때만 함께 건드린다.
  - `web/package.json` / `web/package-lock.json` — 3번(audit)이 원인일 때만. `npm audit fix` 류의 광범위 변경 대신 지목된 패키지 하나의 최소 버전 올림(메이저 업그레이드 금지).
  - `go.mod` / `go.sum` — 4·5번이 원인일 때만, `go get <module>@<patch>` 로 최소 변경.
  - `Dockerfile` — 6번이 원인일 때만.
  - 손대지 말 것: `internal/adapters/pop3/client.go`, `internal/application/*`, `web/src/**`, `web/scripts/engine-node.mjs`, `web/src/styles.css` 의 `@source not "../scripts"`(이 한 줄을 지우면 자산 드리프트 33건이 즉시 재발한다 — 지난 회차가 실측).

- 검증 명령(수정 후 전부 돌릴 것):
  - 실패했던 그 명령 자체(위 1~8 중 해당 항목) → exit 0
  - `go build ./... && go vet ./...`
  - `go test -race -count=1 ./...`
  - `make lint` (gofmt + gosec v2.28.0)
  - `go run ./cmd/postra-contracts -check`
  - `cd web && npm ci --no-audit --no-fund && npm run typecheck && npm test && npm run build`
  - `git status --porcelain --untracked-files=all -- internal/transport/spa/assets` (빈 출력)
  - `git diff --check`

- 위험과 피할 것:
  - `.github/workflows/**` 는 보호 경로다. 같은 커밋에서 **다른 개선을 끼워 넣지 마라** — 이 회차는 깨진 게이트 하나만 고친다.
  - 재현 없이 YAML 을 손보지 마라. 지난 회차의 교훈이 정확히 이것이다: 과제서가 지목한 원인(`Makefile` 설치 누락)이 아니라 실제 원인(npm PATH 가 engines.node 미달 인터프리터를 가린다)이 문제였고, 재현이 그것을 가렸던 가설을 걷어냈다.
  - 외부 상태에 의존하는 게이트(3·4번)는 "오늘의 advisory" 때문에 실패할 수 있다. 그 경우에도 억제·예외 추가가 아니라 의존성 올림으로 고치고, 올린 이유(advisory ID)를 커밋 본문에 적어라.
  - PostgreSQL(`POSTRA_TEST_PG`)·Chromium e2e 는 로컬에 환경이 없으면 "미실행" 으로 명시하라 — Go 전체 통과를 그들이 돌았다는 근거로 쓰지 말 것.
  - `git stash` 금지(워크트리 공유). 설정해 둘 것이 있으면 임시 WIP 커밋을 써라.

- 차선 후보: **POP3 가져오기(fetch) 단계 진단 수정 재제출** — 2026-10-02 첫 회차가 `internal/adapters/pop3/client.go` 의 `Retrieve`(:347)·`Top`(:358) 에서 `domain.WrapInbound(..., 0, s.commandTO)` 로 경과 0 을 넘기는 것과 `Top` 의 동사 오기(`"RETR"`)를 고쳤고(프로덕션 5줄 + 실제 TCP 스크립트 서버 테스트 1개), Go 전체 검증을 통과했으나 **당시 프런트 verify 가 깨져 머지되지 못했다**. 그 블로커는 `16feb8b` 로 main 에서 풀렸고, 해당 커밋은 아직 main 에 없다(확인: `git log --oneline -12` 에 없음). 위 1~8 이 전부 통과해 배정된 실패가 낡은 것으로 판명되면 이것을 그대로 다시 올려라 — 비용 대비 가장 확실하다. 2순위: `sync.max_message_bytes` 의 0·음수=무제한 규약을 설정 카탈로그·관리자 문서에 명시(10회 연속 차선).
