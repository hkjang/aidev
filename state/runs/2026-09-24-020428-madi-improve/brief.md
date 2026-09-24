# 과제서 — 2026-09-24-020428-madi-improve

- **과제**: main@fd2c3b5 의 CI 를 붉게 만드는 외부 드리프트 2건만 최소 통합 — govulncheck GO-2026-6348(grpc v1.82.1→v1.83.1)과 `deploy/runtime-apk.lock` 의 ca-certificates·tzdata 드리프트 (가치 5 / 위험 2 / 작업량 M)

- **왜**: 이번 회차의 기준선 main@fd2c3b5 에는 `go.mod:87 google.golang.org/grpc v1.82.1` 과 `deploy/runtime-apk.lock:11-12,69`(ca-certificates-bundle=20260611-r0, ca-certificates=20260611-r0, tzdata=2026c-r0)이 그대로 있어, `.github/workflows/ci.yml` 의 `test` job "Reject reachable Go vulnerabilities" 단계와 `offline-image` job "Build service image" 단계가 이 브랜치에서도 실패한다(같은 두 단계가 PR #7·#9·#10 에서 반복 실패). 이 두 줄을 고치면 이 브랜치와 이후 모든 PR 의 CI 가 다시 초록이 되고, 오프라인 이미지가 실제로 빌드된다.

- **수용 기준**:
  1. `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` 가 "No vulnerabilities found" 로 exit 0 (수정 전에는 GO-2026-6348 이 `internal/server/jobs.go:341 (*Server).StartJobs` 도달 경로로 보고되며 exit 3 — **먼저 수정 전 출력을 남겨 원인을 증명한 뒤** 고칠 것).
  2. `docker build --target runtime-base .` 가 exit 0 (수정 전에는 `apk` 가 `ca-certificates-…-r0: breaks: world[ca-certificates=20260611-r0]` 류로 exit 123). 그리고 그 스테이지의 `apk info -v` 목록이 lock 의 69개 항목과 diff 없이 일치.
  3. `node scripts/licenses.mjs --check` 가 `{"ok":true,...}` 로 exit 0 — go.mod/go.sum 을 바꿨으므로 `web/public/licenses-manifest.json` 재생성이 **필수**다(스크립트 10행이 go.mod·go.sum 해시를 manifest 에 담는다). CI 는 이 단계를 govulncheck 보다 **먼저** 돌리므로 빠뜨리면 실패 지점만 앞당겨진다.
  4. `go build ./... && go vet ./...` exit 0, `go test -count=1 ./tests/deployment-contract` ok, `node --test tests/natural-date.mjs tests/silent-sso.mjs` pass 2.
  5. `git diff --stat` 이 아래 "건드릴 파일" 5개(+ licenses 산출물)만 보여줄 것. `.github/`·`Dockerfile`·`internal/`·`web/src/` 는 diff 0.

- **건드릴 파일**:
  - `go.mod:87` — `google.golang.org/grpc` 를 v1.82.1 → **v1.83.1**. `go get google.golang.org/grpc@v1.83.1 && go mod tidy` 로 수행. tidy 가 `golang.org/x/sys`·`golang.org/x/text` 를 indirect→direct 로 옮기면 **되돌리지 말 것**(실제 직접 import 임이 09-19 회차에 확인됨).
  - `go.sum` — 위 명령이 생성하는 줄만.
  - `deploy/runtime-apk.lock:11,12,69` — `ca-certificates-bundle`, `ca-certificates`, `tzdata` 세 줄. **세 줄을 함께** 올릴 것: 09-19 회차에 ca-certificates·tzdata 두 줄만 올렸더니 `tests/image-smoke` 가 `unexpected runtime source inventory size`(origins 53≠52)로 실패했다. ca-certificates 와 ca-certificates-bundle 은 같은 aport 라 버전이 갈리면 origin 이 하나 늘어난다. 나머지 66줄·주석 2줄·테스트의 69/52 상수는 **손대지 말 것**.
  - `web/public/licenses-manifest.json`, `web/public/licenses.txt` — `node scripts/licenses.mjs` 재생성 결과(손으로 편집 금지).

- **버전 확정 방법(중요, 추측 금지)**: 8451b5b 가 쓴 값은 09-19 시점의 `ca-certificates=20260909-r0`, `ca-certificates-bundle=20260909-r0`, `tzdata=2026d-r0` 이지만, Alpine 3.24 저장소는 그 뒤로 또 움직였을 수 있다. 반드시 고정 이미지에서 실제 버전을 읽어 쓸 것:
  `docker run --rm alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b sh -c 'apk update >/dev/null && apk policy ca-certificates ca-certificates-bundle tzdata'`
  그 출력의 최신 버전으로 세 줄을 맞춘 뒤 수용 기준 2 로 증명한다.

- **검증 명령**(이 저장소에서 실제로 도는 것, 위→아래 순서):
  ```
  npm ci --prefix web && npm run build --prefix web     # web/dist 필요: go build 의 all:dist embed
  go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...  # 수정 전/후 각각
  node scripts/licenses.mjs && node scripts/licenses.mjs --check
  go build ./... && go vet ./...
  go test -count=1 ./tests/deployment-contract
  node --test tests/natural-date.mjs tests/silent-sso.mjs
  docker build --target runtime-base .                   # lock 증명(빠름, 전체 이미지 아님)
  git diff --check
  ```
  예산이 남으면(그리고 그때만) `docker build -t madi:ci . && bash scripts/verify-image.sh madi:ci`. **주의**: `/tmp/verify-image.txt` 같은 고정 경로 산출물은 지난 회차 잔재일 수 있다. 실행 전 지우고 이번 실행의 mtime 을 확인한 출력만 믿을 것.

- **위험과 피할 것**:
  - **범위**. 지난 세 회차가 같은 드리프트를 고치면서 `web/dist/.gitkeep`+vite 플러그인+README 까지 함께 끌어와 회차가 verify-failed 로 끝났다. 이번에는 **그 묶음을 빼고** 위 5개 파일만 만진다. 깨끗한 체크아웃의 `go build` 문제(`web/embed.go:7 pattern all:dist`)는 CI 가 go 단계 전에 `npm run build` 를 돌리므로 **CI 실패 원인이 아니다** — 이번 범위 밖.
  - 보호 경로 무수정: `.github/workflows/*`, `Dockerfile`, `internal/server/*`(인증·세션·SQL·backup_tables.go), `scripts/*`, `tests/*`. 특히 `tests/image-smoke` 의 69 APK / 52 origins / 2 모델 상수를 낮추는 식으로 검사를 완화하지 말 것 — 그건 고친 게 아니다.
  - `go mod tidy` 가 grpc 외 다른 모듈까지 크게 움직이면 멈추고 `go get` 대상만 되돌아볼 것. 기대 diff 는 go.mod/go.sum 합쳐 10~15행 수준.
  - 로컬 통과는 원격 CI/릴리즈 성공을 뜻하지 않는다. 회차 노트에 그렇게 쓰지 말 것.

- **차선 후보**: 위 두 건 중 lock 쪽 Docker 검증이 이 환경에서 불가능하면(도커 데몬 없음), **grpc/govulncheck 한 건만** 위 수용 기준 1·3·4·5 로 완결하고 lock 은 손대지 않은 채 노트에 남긴다(반쯤 고친 lock 은 이미지 빌드를 더 나쁘게 만든다). 그것도 막히면 `tests/natural-date.mjs` 에 연도 경계·상한 회귀 검사 보강(2/1/S).

---

## 미확인으로 남긴 것 (정찰이 직접 실행하지 못함)
- `govulncheck` 를 이번 회차에 직접 돌리지 못했다(도구 실행 승인 거부). GO-2026-6348 이 **지금도** 보고된다는 것은 `go.mod:87` 의 v1.82.1 과 `internal/server/jobs.go:341 StartJobs` 존재로 추론한 것이며, 09-19·09-22 회차의 실제 재현 기록과 일치한다. 구현자는 수용 기준 1 의 "수정 전 출력" 으로 먼저 확정할 것.
- `docker info` 도 확인하지 못했다. 도커 가용성은 09-22 회차 기록(실제 이미지 빌드 성공)에 근거한 추정이다.
- 원격 CI 의 최신 run 상태는 이번에 조회하지 않았다. 실패 단계는 파일 내용과 과거 회차 기록에서 추론했다.
