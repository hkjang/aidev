- 과제: 수정 과제 — PR #172 의 CI 를 두 번 떨어뜨린 게이트를 특정해 그 원인을 고치기 (외부 권고 게이트가 1순위 혐의) (가치 5 / 위험 1 / 작업량 S)
- 왜: PR #172 의 내용은 `web/src/lib/edge-vocabulary.test.ts` **한 파일 87줄 추가**가 전부입니다(`git diff --stat main...auto/2026-10-16-2058` → `1 file changed, 87 insertions(+)`, 커밋 a5959a5 하나). 시험 파일 하나는 `ci.yml` 의 Go 절·Database 절 11단계·e2e·`docker build` 를 깨뜨릴 수 없고, 아래 ①에서 프런트엔드 게이트도 정적으로 배제되므로, 같은 이유로 두 번 떨어진 게이트는 **커밋이 아니라 외부 DB 가 판정을 바꾸는 게이트**일 가능성이 가장 큽니다 — `ci.yml:68` `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` 또는 `ci.yml:117-119` `npm audit --audit-level=high`. 이 저장소는 같은 모양으로 이미 두 회차를 잃었습니다(2026-10-06 GHSA-68fv-2mgg-jv7q, 2026-10-09 GO-2026-6629). 그렇다면 **main 도 같이 빨갛고**, 고치면 PR #172 와 main 이 함께 초록이 됩니다.

- 수용 기준:
  1) 고치기 **전에** 실패 게이트를 이름으로 특정한 실제 출력을 남긴다 — 명령 전문, 권고 ID(GO-…/GHSA-…), 종료코드. 추측으로 적지 말 것. 실패를 재현하지 못했다면 "재현 실패" 로 적고 ③의 대안으로 갈 것.
  2) 고친 뒤 **같은 명령**이 EXIT=0 이고, 고침 diff 를 되돌리면 같은 권고 ID 로 다시 실패한다(2026-10-09 회차가 `impl-vuln-base.log` / `impl-vuln-reverted.log` / `impl-vuln-final.log` 로 한 그대로).
  3) 임계값·범위·워크플로를 느슨하게 하지 않았음을 `git diff --stat` 으로 보인다. `.github/workflows/` 가 diff 에 없어야 한다.

- 진단 순서 (이 순서로, 멈추는 곳이 답):
  - **1단계 (가장 쌈, 설치 불필요)** `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` — 네트워크 필요. `Vulnerability #n: GO-…` 와 **호출 경로**(`internal/…: X calls Y`)가 나오면 그것이 원인입니다. 이 저장소는 "호출하지 않는" 경고(패키지 1 / 모듈 4)를 상시 가지고 있으니 **호출 경로가 있는 것만** 실패입니다.
  - **2단계** `npm ci --prefix web` 후 `cd web && npm audit --audit-level=high`. 새 체크아웃에 `web/node_modules` 가 없습니다(이번 정찰 시점 확인). high 이상이 나오면 그것이 원인.
  - **3단계 (1·2 가 둘 다 초록이면)** 프런트엔드 게이트를 PR 브랜치에서 그대로 돌려 보기: `git checkout auto/2026-10-16-2058` 후 `make test-web` 8게이트 + `npx playwright install --with-deps chromium && npx playwright test`(`ci.yml:146-156`, `UMM_E2E_COMMAND=/tmp/umm-ci` 가 필요 — `go build -ldflags '-X main.version=ci' -o /tmp/umm-ci ./cmd/umm`). 여기서 빨개지는 것이 원인.

- 고침 (1단계가 맞을 때):
  - `go.mod` + `go.sum` 의 해당 모듈 **버전·체크섬만** 최소 올림(`go get <module>@<fixed>` → `go mod tidy` 는 쓰지 말고 diff 를 두 파일로 유지). 2026-10-09 의 81a8b5e(x/text v0.40.0→v0.41.0) 가 정확한 선례이며 그 회차는 merged 입니다. 앱 코드 0줄, 새 시험 0개.
  - 2단계가 맞을 때: `web/package-lock.json` 만(`npm audit fix` 가 `package.json` 의 범위를 바꾸지 않는지 `git diff --stat` 으로 확인 — 4f2de72 는 락파일 6줄만 바꿨습니다). `web/package.json` 의 의존성 범위·`engines`·`version` 은 그대로.
  - 3단계가 맞을 때: 범위는 `web/src/lib/edge-vocabulary.test.ts` 한 파일 안에서 끝내고, 프로덕션 모듈(`edge-vocabulary.ts`·`en.ts`·`api.ts`)은 고치지 말 것 — 그 모듈들은 이 PR 이 건드리지 않았으므로 원인이 아닙니다.

- 건드릴 파일: 진단 결과에 따라 **두 파일 이내** — `go.mod`·`go.sum` / 또는 `web/package-lock.json` / 또는 `web/src/lib/edge-vocabulary.test.ts`. 세 갈래를 동시에 고치지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` (CI 와 **같은** 명령, `ci.yml:68`)
  - `go mod verify` / `go vet ./...` / `POSTGRES_DSN=<격리 DSN> go test -p 1 ./... -count=1` (DSN 없으면 통합이 SKIP — 통과로 포장 금지) / `go test -race -count=1 ./internal/intelligence` / `go build ./cmd/umm`
  - `npm ci --prefix web` 선행 후 `make test-web` (8게이트: offline-queue → audit → typecheck → lint → check-i18n → test → build → verify:pwa)
  - `./scripts/check-version.sh`
  - 릴리즈 경로를 건드렸다면 `docker build --build-arg VERSION=$(tr -d '[:space:]' < VERSION) .` (운영자 규칙 4번)

- 위험과 피할 것:
  - **금지: 워크플로를 느슨하게 해 통과시키기.** `--audit-level=high` 낮추기, govulncheck 범위를 `./...` 에서 줄이기, `-race` 대상 축소, 게이트 `continue-on-error`, 실패 단계 삭제 전부 금지. 보호 경로 `.github/workflows/`·`Dockerfile`·`VERSION`·`migrations/`·`internal/auth` 는 읽기만.
  - **의존성 메이저 업그레이드 금지.** 권고를 닫는 **최소** 패치 버전만. `go mod tidy`·`npm audit fix --force` 는 diff 를 넓혀 금지.
  - `Makefile` 을 손대지 말 것 — "`make test-go` 가 CI 의 Go 게이트 다섯 개 중 두 개만 돌린다" 는 별개 과제이고 2026-10-06 에 같은 방식으로 verify-failed 했습니다(운영자 규칙 1번).
  - 마크다운 파서(docs 분류기·deck 렌더러·compile) 통합 금지(운영자 규칙 3번). 감사 `details` 에 검출 문자열 넘기지 말 것(규칙 2번).
  - 새 체크아웃은 `web/node_modules` 가 없습니다. 설치 전의 `exit 127`·`not found` 는 저장소 결함이 아닙니다(이 함정으로 세 회차를 잃었습니다). `ls -d web/node_modules` 를 먼저 보세요.
  - `VERSION`(0.76.8)은 올리지 말고, 커밋 트레일러(`Co-Authored-By` 등)를 붙이지 마세요.

- **이번 정찰이 정적으로 배제한 것 (다시 의심하지 말 것)**:
  - **prettier 줄 길이 아님** — `web/.prettierrc.json:2` 의 `printWidth` 가 **120** 이고 새 파일의 가장 긴 줄은 약 108자입니다. `npm run lint`(`oxlint src && prettier --check src`)가 이 파일 때문에 떨어질 이유가 없습니다.
  - **check-i18n 아님** — 새 파일의 한국어 리터럴(`연결됨`·`뒷받침함`·`상충함`·`구체화함`·`확장됨`·`이어짐`·`직접 연결`·`에이전트`·`Dream 확장`·`가져오기`·`자동 추천`)은 전부 `edge-vocabulary.ts` 가 이미 쓰는 기존 키이고 새 키가 아닙니다.
  - **Go/DB/e2e/docker 를 이 diff 가 유발한 것 아님** — diff 에 Go·SQL·설정 파일이 0개입니다.

- **미확인 (정찰이 확인하지 못한 것 — 사실처럼 쓰지 말 것)**:
  - CI 로그를 읽지 못했습니다. 이 세션에서 `gh pr view 172` 가 승인되지 않아 실패 단계 이름을 **직접 보지 못했습니다**. 위의 "외부 권고 게이트" 는 diff 범위에서 나온 **추론**입니다 — 1단계로 실제 출력을 먼저 얻으세요.
  - `govulncheck`·`npm audit` 도 이 정찰 세션에서는 네트워크 명령이 차단돼 **돌리지 못했습니다**. 지금 권고가 있는지 없는지 모릅니다.
  - 러너가 전한 "릴리즈 워크플로가 두 번 실패" 는 그대로 믿지 마세요 — `.github/workflows/release.yml:4` 는 `on: push: tags: ['v*']` 뿐이라 **PR 로는 돌지 않습니다**. 실패한 것은 `ci.yml` 입니다(2026-10-09 회차도 같은 정정을 했습니다).

- 차선 후보: `format` 에도 설치 가드 한 줄 — `web/package.json:20` 의 `format` 에는 `preformat` 이 없어 미설치 체크아웃에서 이름 없는 `exit 127` 로 죽습니다(`typecheck`/`lint`/`build`/`dev`/`preview`/`e2e` 는 이미 `node scripts/require-installed.mjs` 로 닫혀 있음). `web/package.json` 에 `"preformat": "node scripts/require-installed.mjs prettier"` 한 줄 + 기존 `web/scripts/require-installed.test.mjs` 에 fixture 시험 하나. 소스를 **쓰는** 명령이므로 실제 `src` 에 돌리지 말고 fixture 로만 재현할 것 (가치 2 / 위험 1 / S).
