# 과제서 — 2026-10-18-073859-umm-improve (베이스 main@745dcf3, VERSION 0.76.8)

- 과제: Go 버전 핀 드리프트를 CI 게이트와 릴리즈 이미지 **양쪽**에서 닫기 — `go.mod` 의 `x/net`·`toolchain` + `Dockerfile:9` 의 빌더 이미지 (가치 5 / 위험 2 / 작업량 S)
- 왜: 이 회차의 베이스 `main@745dcf3` 는 `golang.org/x/net v0.57.0`(`go.mod:39`)과 `toolchain go1.26.7`(`go.mod:8`)을 그대로 갖고 있어 CI 의 `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`(`ci.yml:66`) 게이트가 빨갛다 — 2026-10-16 회차가 이것을 `13b7e00` 로 고쳤지만 그 커밋은 브랜치 `auto/2026-10-16-2209` 에 review-pending 으로 남아 **main 에 들어오지 않았다**(`git branch -a --contains 13b7e00` → 그 브랜치 둘뿐). 더 중요한 것은 **go.mod 만 고쳐도 릴리즈 산출물은 안 고쳐진다**는 점이다: CI 는 `ci.yml:37` 의 `go-version-file: go.mod` 로 Go 를 고르지만, 릴리즈는 `release.yml:26` 의 `docker build` 가 `Dockerfile:9` 의 `FROM golang:1.26.7-alpine` 로 빌드하고 공식 golang 이미지는 `GOTOOLCHAIN=local` 이라 `toolchain` 줄을 무시한다(2026-10-16 회차가 이미지에서 바이너리를 꺼내 `go version /app/umm` → `go1.26.7` 로 확인). 둘을 같이 닫으면 CI 게이트와 릴리즈 이미지가 처음으로 같은 Go 패치를 쓴다.

- 수용 기준:
  1) `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` 가 `No vulnerabilities found.` / EXIT=0. **이 명령을 가장 먼저, 고치기 전에** 돌려 권고 ID·호출 경로·종료코드를 로그로 남길 것(추측으로 적지 말 것).
  2) 되돌림 드릴: `git checkout -- go.mod go.sum Dockerfile` 하면 1단계에서 본 **같은 권고 ID 들이 글자 그대로 다시** 나고 EXIT≠0, 복구하면 다시 EXIT=0.
  3) **최종 산출물로** 증명(운영자 규칙 4번 — 중간 단계가 효과를 삼킨 전례가 바로 이 자리다): `docker build --build-arg VERSION=0.76.8 .` 로 만든 이미지에서 `/app/umm` 을 꺼내 `go version <바이너리>` 가 **`go1.26.9` 이상**이고 `go1.26.7` 이 아님. `docker build` EXIT=0 만으로는 수용 기준을 만족하지 않는다. 첫 빌드가 CACHED 면 `--no-cache-filter go-build --progress=plain` 로 다시 띄울 것.
  4) 느슨하게 만든 것 0건: `git diff --name-only` 가 **정확히** `Dockerfile`·`go.mod`·`go.sum` 이고 `.github/workflows/`·`VERSION`·`migrations/`·`internal/auth`·`Makefile`·`web/` 이 없다.

- 건드릴 파일 (프로덕션 3개, 새 시험 파일 0개):
  - `go.mod:39` — `golang.org/x/net v0.57.0` → `v0.60.0`. `go get golang.org/x/net@v0.60.0` 로.
  - `go.mod:8` — `toolchain go1.26.7` → `toolchain go1.26.9` (한 줄 수동). **`go 1.26.0` 디렉티브(`go.mod:3`)는 그대로** 둘 것 — 바로 위 주석이 "developer on any Go 1.26 patch release can build with GOTOOLCHAIN=local" 라고 그 선택의 이유를 적어 두었다. `toolchain` 값을 올리면 그 주석의 숫자가 어긋나므로 주석의 `1.26.7` 언급이 있으면 같이 맞출 것.
  - `go.sum` — 위 결과 체크섬만. **`go mod tidy` 금지** (diff 가 번진다).
  - `Dockerfile:9` — `FROM golang:1.26.7-alpine AS go-build` → `golang:1.26.9-alpine`. **이 한 줄만.**

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`  ← 1순위, 고치기 전/후/되돌림 3회
  - `go mod verify` · `go vet ./...` · `go build ./cmd/umm`
  - `POSTGRES_DSN=<격리 DSN> go test -p 1 ./... -count=1` — DSN 없으면 통합이 SKIP 이다. **DSN 이 살아 있었음을 대조군으로 증명**할 것: 같은 시험을 `env -u POSTGRES_DSN` 로 돌려 `--- SKIP` 이 나오는 것을 붙일 것. DSN 없는 PASS 를 통과로 포장 금지.
  - `go test -race -count=1 ./internal/intelligence`
  - `go test ./internal/intelligence -run 'Quality|Vocabulary|Threshold|Semantic' -v`
  - `docker build --build-arg VERSION=0.76.8 .` → 이미지에서 바이너리 추출 → `go version`
  - `./scripts/check-version.sh` (`0.76.8 is consistent`)
  - 웹·Go 소스 0줄이므로 `make test-web` 은 범위 밖. 돌린다면 `npm ci --prefix web` 선행 필수(**이 체크아웃에 `web/node_modules` 없음을 정찰이 확인**) — 미설치 상태의 exit 127 은 저장소 결함이 아니다.

- 위험과 피할 것:
  - `Dockerfile` 은 보호 경로다 — **9 행 한 줄만**. `node:24-alpine`(2행)·`alpine:3.24`(최종 스테이지)·`-trimpath`/`-ldflags`·`CGO_ENABLED=0` 는 손대지 말 것.
  - x/net 을 올리면 MVS 가 `x/crypto`·`x/sys`·`x/text`·`x/sync` 를 **강제로** 끌어올린다(2026-10-16 회차가 상류 go.mod 로 강제임을 확인). 기회주의적 업그레이드로 적지 말고 "MVS 강제" 로 적을 것.
  - `13b7e00`(브랜치 `auto/2026-10-16-2209`, review-pending) 가 `go.mod`/`go.sum` 의 **같은 자리**를 고친다. 그 브랜치를 머지·cherry-pick 하지 말고 이 브랜치에서 독립적으로 고칠 것. 머지 시 go.mod/go.sum 충돌이 날 수 있음을 커밋 메시지에 적지 말고(잡음) 중재에 맡길 것.
  - `.github/workflows/` 를 고쳐 게이트를 느슨하게 만드는 방향 금지. `govulncheck` 은 커밋이 아니라 취약점 DB 가 판정을 바꾸므로 "스캔을 끄기/핀하기" 도 금지.
  - `VERSION` 을 올리지 말 것(일반 개선 회차). 커밋 트레일러(`Co-Authored-By` 등) 금지, 커밋 메시지는 영어 동작 설명.
  - 마크다운 파서·다운로드 파일 이름 통합(29149e5 로 되돌려진 PR #167)과는 무관한 과제다 — 그 근처를 건드리지 말 것.

- 차선 후보: **`format` 에도 설치 가드 한 줄(`preformat`)** — `web/package.json:18` 의 `"format": "prettier --write src"` 에만 `pre*` 가 없다(정찰이 `package.json:9-26` 을 직접 읽어 확인: `predev:10`·`prebuild:12`·`pretypecheck:14`·`prelint:16`·`pree2e:23`·`prepreview:25` 는 이미 있고 `test*` 는 `run-on-supported-node.mjs` 를 지난다). 고침은 `web/package.json` 한 줄 + `web/scripts/require-installed.test.mjs` 의 **기존 fixture 방식**으로만 재현 — 소스를 쓰는 명령이므로 실제 `src` 에 돌리지 말 것. 기존 가드 모듈(`web/scripts/require-installed.mjs`)을 재사용하고 두 진단 경로를 통합하지 말 것(운영자 규칙 3번).

## 확인 상태 (정찰이 실제로 본 것 / 미확인)
정찰 세션에서 Bash 실행 승인이 읽기 전용에만 났습니다. 아래를 그대로 믿지 말고 1단계에서 실행으로 메우세요.
- **확인(파일을 직접 읽음)**: `go.mod:3` `go 1.26.0` / `go.mod:8` `toolchain go1.26.7` / `go.mod:39` `x/net v0.57.0` / `:40` `x/sync v0.22.0` / `:41` `x/sys v0.47.0` / `:42` `x/text v0.41.0`; `Dockerfile:2` `node:24-alpine`, `:9` `golang:1.26.7-alpine`, 최종 `alpine:3.24`; `ci.yml:37` `go-version-file: go.mod`, `:51-70` Go 게이트 5개(vet → `go test -p 1 ./...` → race → govulncheck:66 → 임베딩 리포트); `release.yml:26` `docker build --build-arg VERSION=...`; `VERSION` 0.76.8; `13b7e00` 이 main 에 없음; `web/node_modules` 없음; `web/package.json:18` 에 `preformat` 없음.
- **미확인 — govulncheck 의 지금 출력.** 베이스 SHA 가 2026-10-16 회차와 **동일**(`745dcf3`)하므로 그 회차의 기록(x/net 5건: GO-2026-6617·-6612·-6611·-6610·-6603 + 표준 라이브러리 12건, `exit status 3`)을 근거로 적었으나 **내가 돌린 결과가 아닙니다**. 만약 1단계에서 **이미 초록**이면 이 과제는 성립하지 않으니 차선 후보로 가고, 권고가 **다른 것**이면 그 ID 를 고쳐 적고 과제서의 버전 숫자를 재계산할 것.
- **미확인 — `golang:1.26.9-alpine` 태그의 존재와 `x/net v0.60.0` 의 가용성.** `docker pull golang:1.26.9-alpine` / `go list -m -versions golang.org/x/net` 으로 먼저 확인할 것. 1.26.9 가 없으면 **govulncheck 이 `Fixed in` 으로 말하는 버전 이상의 실재하는 최신 1.26.x** 로 올리고 `go.mod` 의 `toolchain` 을 **같은 값**으로 맞출 것(두 숫자가 어긋나면 이 과제의 요점이 사라진다).
- **미확인 — `actions/setup-go@v6` 가 `go-version-file: go.mod` 에서 `toolchain` 줄을 읽는지.** 읽지 않고 `go 1.26.0` 디렉티브를 쓴다면 CI 는 go1.26.0 을 깔고 `GOTOOLCHAIN` 기본값(auto)으로 `toolchain` 을 내려받아 결국 1.26.9 가 되지만, 이 경로를 가정하지 말고 1단계 govulncheck 결과로 판단할 것.
