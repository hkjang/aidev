- 과제: 릴리즈 검증 게이트 복구 — `cd web && npm test --silent` 과 루트 `npm run lint` 를 "구버전 node 가 PATH 를 가로챈 상태" 와 "`web/node_modules` 가 없는 상태" 양쪽에서 통과하게 만들기 (가치 5 / 위험 2 / 작업량 S)

- 왜: 러너의 verify 가 **여섯 번 연속** 같은 두 명령에서 끊겨 그 회차의 수정이 전부 버려졌다 — `cd web && npm test --silent (exit 1)` 가 2026-10-01 14:58 · 10-02 00:30 · 10-03 07:18 · 10-04 14:58, `npm run lint   # CI에서 가져옴 (exit 1)` 가 10-01 19:24 · 10-01 21:21 (각 회차 `journal.md` 의 `[러너] verify failed` 줄에서 직접 확인). 그래서 main 은 아직 98c5081(v0.9.66)이고 1d63b7c·eed58c8·5ed6876·f927cea·4700560·5fcc7a3·e64199a 가 전부 미머지로 남아 있다(`git log --graph --all` 로 확인 — 모두 98c5081 직속 단일 커밋). 이 게이트를 통과시키지 않으면 이 저장소의 어떤 과제도 릴리즈에 닿지 못한다.

- 수용 기준:
  1) 저장소 **루트에서** `cd web && npm test --silent` 가 exit 0 이고, 테스트 파일/테스트 수가 줄지 않는다(기존 9개 + 이번에 들어오는 테스트).
  2) 저장소 **루트에서** `npm run lint` 가 exit 0 이고 출력에 `Missing script` 가 없다(= web 의 eslint 가 실제로 돌았다).
  3) `web/node_modules` 를 **지운 상태에서** 위 두 명령이 그대로 exit 0 이다(러너의 새 워크트리 상태). 이 조건이 이번 회차의 **새 작업**이다 — 아래 "미확인/새 작업" 참조.
  4) `/home/hkjang/node_modules/.bin/node`(= Node 20.19.2, `readlink -f` + 그 패키지의 `package.json` 으로 확인)가 PATH 앞에 있어도 vitest 가 Node ≥22.19.0 으로 돈다.
  5) 느슨하게 만든 것이 없다: `--passWithNoTests`·`|| true`·`--max-warnings`·`exit 0`·`continue-on-error`·`skip`/`todo`·테스트 삭제·`vitest.config.ts` 의 `include` 축소 전부 금지. `.github/workflows/ci.yml`·`web/package-lock.json`·`web/vite.config.ts`·`web/vitest.config.ts`·`web/src/test/keep-dist-placeholder.test.ts` 는 무변경(`git status --short` 로 증명).
  6) Go 쪽 무회귀: `go build ./...` 0, `go vet ./...` 0, `go test ./... -count=1` 전체 통과, `go run ./cmd/api-surface-audit` 누락 0.

- 작업 순서:
  **0단계 (재현 — 먼저 할 것).** 깨끗한 트리에서 `cd web && npm test --silent` 를 돌려 실패를 기록한다. 내가 이 워크트리에서 실제로 돌린 결과는 **exit 127 / `sh: 1: vitest: not found`** 이다(`web/node_modules` 없음). 러너는 같은 명령에 **exit 1** 을 보고했다 — 즉 러너 환경은 이 상태와 다르다(러너가 먼저 의존성을 설치했고 vitest 가 실제로 **실패**했을 가능성, 또는 러너가 비정상 종료코드를 1 로 정규화해 보고할 가능성). **두 경우를 모두 막는 것이 이번 과제다.** `npm ci` 후 다시 돌려 jsdom 쪽 실패(아래 1단계의 증상)가 재현되는지도 확인해 기록할 것.

  **1단계 (알려진 두 층 복원 — 재작성 말고 cherry-pick).** `git cherry-pick f927cea` 한 번으로 아래가 한꺼번에 들어온다(그 커밋의 부모가 98c5081 이라 충돌 없이 적용될 것 — 미확인, 적용해 확인할 것):
  - `web/package.json`: `engines.node: ">=22.19.0"`(jsdom 30 이 끌어오는 undici 8 의 요구치) + `test`/`test:watch` 를 런처 경유로.
  - `web/scripts/run-with-supported-node.mjs`: npm 이 run-script PATH 앞에 상위 `node_modules/.bin` 을 붙여 **Node 20.19.2** 가 vitest 의 `#!/usr/bin/env node` 를 가로채는 문제를 우회(그 상태에서 `webidl.util.markAsUncloneable is not a function` 으로 jsdom 환경 8파일이 기동조차 못 한다). PATH 의 node 를 믿지 않고 `npm_node_execpath` → `process.execPath` 순으로 고른다.
  - `package.json`(루트) + `scripts/web-run.mjs`: 루트에서 `npm run lint|test|build` 를 web 의 같은 스크립트로 **위임**만 한다. 루트에 `package.json` 이 없으면 npm 이 상위로 올라가 `/home/hkjang/package.json` 을 프로젝트 루트로 집어(`npm prefix` = `/home/hkjang`, 지난 회차 실행 확인) `npm error Missing script: "lint"` 로 exit 1 이 된다.
  - 딸려 오는 것: `web/src/test/test-script-interpreter.test.ts`, `web/src/test/root-npm-scripts.test.ts`, `.gitignore`(루트 `node_modules/`), `docs/OPERATIONS.md`, `docs/RELEASE_GUIDE.md`.
  - `5ed6876`(eslint 가 `.js`/`.mjs` 를 규칙 0개로 린트하는 문제)는 **이번 과제에 필요 없다** — 린트는 그 커밋 없이도 exit 0 이었다. 파일 수를 아끼기 위해 가져오지 말 것(ideas.json 에 후보로 남겨 둠).

  **2단계 (이번 회차의 새 작업 — `web/scripts/run-with-supported-node.mjs` 보강).** f927cea 를 그대로 되살리는 것만으로는 **부족하다**. 그 커밋이 들어간 10-02 회차도 같은 명령에서 verify 가 실패했고, 코드를 읽어 보면 두 구멍이 남아 있다:
  - (a) `resolveLocalBin(name)` 이 `web/node_modules/<name>/package.json` 을 `readFileSync` 로 바로 읽는다 → **`web/node_modules` 가 없으면 ENOENT 가 try 없이 던져져** 스택 트레이스와 함께 **exit 1** 로 죽는다(127 이 아니라 1 — 러너가 본 숫자와 일치한다). 로컬 bin 이 없으면 **먼저 설치**하도록 고칠 것: `web/node_modules/vitest` 가 없을 때만 선택된 interpreter 로 `npm ci` 를 한 번 돌리고(`npm_execpath` 를 쓰거나 `npm` 을 spawn), 설치가 실패하면 그 종료코드로 즉시 끝낸다. 설치가 끝났는데도 bin 이 없으면 원인을 적고 실패할 것. **이미 설치돼 있으면 아무 것도 설치하지 않는다**(검증 시간 보호).
  - (b) 후보가 `npm_node_execpath` 와 `process.execPath` **둘뿐**이다. 러너의 npm 자체가 가로채인 Node 20 으로 떠 있으면 두 후보가 모두 하한 미달이라 런처가 설계대로 exit 1 로 멈춘다. 후보 목록을 넓힐 것: 위 둘 → `$NVM_DIR/versions/node/*/bin/node`(버전 내림차순) → `/usr/local/bin/node` → `/usr/bin/node`. **상위 `node_modules/.bin` 경로는 후보에서 제외**한다(그게 문제의 근원이다). 하한을 넘기는 첫 후보를 쓰고, 하나도 없으면 지금처럼 후보 목록과 PATH 가로채기 설명을 찍고 exit 1(조용히 구버전에서 돌리지 말 것).
  - 두 보강은 기존 테스트 `web/src/test/test-script-interpreter.test.ts` 의 단정을 깨지 않아야 한다. 깨지면 테스트를 지우지 말고 새 동작에 맞게 **추가**할 것.

  **3단계 (검증 — 수용 기준 3 을 실제로 돌릴 것).** `mv web/node_modules /tmp/nm-backup` 로 비운 뒤 루트에서 `cd web && npm test --silent` → exit 0 을 확인하고, 되돌린 뒤 한 번 더 확인한다. 끝으로 `npm run build`(루트 또는 web) 뒤 `git status --short` 가 **완전히 빈 출력**(= `web/dist` 항목 없음, `web/dist/.gitkeep` 무변경)인지 확인한다.

- 건드릴 파일 (프로덕션 4개):
  - `package.json`(루트, 신규 — f927cea) — web 의 `lint|test|build` 위임만. `workspaces` 선언 금지.
  - `scripts/web-run.mjs`(신규 — f927cea) — 허용 목록 `lint|test|build` 외 거부. 자식 npm 에는 `process.execPath` 가 아니라 **`npm_node_execpath`** 를 넘긴다(지난 회차가 여기서 구버전을 web 까지 전파시켰다).
  - `web/package.json`(f927cea) — `engines.node` + `test`/`test:watch` 의 런처 경유. 의존성·`package-lock.json` 은 손대지 말 것.
  - `web/scripts/run-with-supported-node.mjs`(f927cea + **2단계 보강**) — 이번 회차의 실제 작업 지점.
  - 테스트·문서: `web/src/test/test-script-interpreter.test.ts`, `web/src/test/root-npm-scripts.test.ts`(cherry-pick), 2단계 동작을 덮는 테스트 추가, `docs/OPERATIONS.md`(정본 PDF 없음 — `ls docs/*.pdf` 로 재확인, 재생성 불필요).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm test --silent`  (러너가 실패를 보고한 바로 그 명령 — 루트에서 이 형태로 돌릴 것)
  - `npm run lint`  (저장소 루트에서. 러너가 "CI에서 가져옴" 으로 돌리는 형태)
  - `cd web && npm run build`  → 이어서 `git status --short` 가 빈 출력
  - `go build ./...` · `go vet ./...` · `go test ./... -count=1` · `go run ./cmd/api-surface-audit`

- 위험과 피할 것:
  - **고치는 대상은 검증 배선뿐이다.** `internal/` Go 코드·`web/src` 제품 코드는 건드리지 말 것(이번 회차에 Go 변경 0 이어도 정상이다).
  - `.github/workflows/ci.yml` 무변경 — CI 의 web 잡은 `defaults.run.working-directory: web` 이라 루트 문제의 영향을 받지 않는다. 거기를 손대면 CI 와 러너가 더 갈라진다.
  - `web/package-lock.json` 을 `npm install` 로 재생성하지 말 것(lockfileVersion 3, `@rolldown/binding-*` 16종 포함 — 정상이다).
  - `web/vite.config.ts` 의 `keepDistPlaceholder`·`emptyOutDir`, `web/embed.go` 의 `//go:embed all:dist` 는 릴리즈 경로다 — 느슨해지는 방향으로 바꾸지 말 것.
  - `gofmt -l internal/proxy/dataworks_runtime.go` 는 HEAD 에서도 기존 CRLF 때문에 파일명을 출력한다 — 줄 끝을 건드리지 말 것.
  - 과거 교훈: 이 계열에서 세 회차가 "원인을 가설로 적고 그 가설이 반증됐다". 그래서 0단계 재현을 먼저 하고, 재현 결과가 이 과제서와 다르면 **재현 쪽을 믿고** 저널에 적은 뒤 그 원인을 고칠 것.
  - 미확인(추측으로 적은 것): (1) 러너 verify 환경에 `web/node_modules` 가 있는지, 러너의 npm 이 어느 node 로 도는지 — 둘 다 직접 볼 수 없어 **양쪽을 모두 견디는 구현**을 요구했다. (2) 러너가 보고한 exit 1 이 실제 종료코드인지 정규화된 값인지. (3) `git cherry-pick f927cea` 가 충돌 없이 붙는지(부모가 98c5081 이라 붙을 것으로 봤다). (4) 러너 환경에 네트워크/npm 캐시가 있는지 — 없으면 2단계 (a) 의 `npm ci` 가 실패한다. 그 경우 `npm ci` 실패를 그 종료코드와 원인 메시지로 드러내고 멈출 것(조용히 통과시키지 말 것).

- 차선 후보: keep-dist-placeholder 테스트의 `logLevel:'silent'` 를 걷어 실제 vite 실패 원인이 로그에 남게 하기 (3/1/S, `web/src/test/keep-dist-placeholder.test.ts` 1파일) — 1순위가 0단계에서 전혀 재현되지 않을 때 고를 것.
