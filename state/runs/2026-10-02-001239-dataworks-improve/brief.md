- 과제: 릴리즈 검증 `npm run lint` exit 1 복구 — 저장소 루트에 web 검증을 위임하는 `package.json` 을 두어 CI 의 `defaults.run.working-directory: web` 없이도 **같은 eslint 가 실제로 돌게** 한다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `.github/workflows/ci.yml` 의 web 잡(이 파일을 직접 열어 확인)은 `defaults.run.working-directory: web` 에 의존해 `npm ci`·`npm run lint`·`npm test`·`npm run build` 를 돌린다. 그래서 그 명령들을 CI 에서 뽑아 **저장소 루트에서** 그대로 돌리는 러너·릴리즈 자동화·새 기여자에게는 `npm run lint` 가 영구히 exit 1 이다(npm 이 상위 디렉터리로 올라가 저장소 밖의 `package.json` 을 프로젝트 루트로 집어 `npm error Missing script: "lint"`). 지난 두 회차가 `cd web && npm ci && npm run lint` 는 clean main(98c5081)과 `1d63b7c` 적용 트리 양쪽에서 **exit 0** 임을 실행으로 확인했고, 재현되는 exit 1 은 루트 실행 경로 하나뿐이었다(2026-10-01 원장: `npm error Missing script: "lint"`, 설치 누락의 127 과 구분됨). 그 회차 과제서가 "루트 `package.json` 을 만들지 말 것" 으로 이 경로를 금지했기 때문에 원인이 그대로 남아 같은 검증이 두 번 연속 실패했다. 고치면 (a) 릴리즈 검증이 루트에서든 web 에서든 같은 린트를 실제로 실행하고, (b) 저장소 루트에서 `npm run <script>` 가 **저장소 밖의 package.json 으로 탈출하는** 현재 상태(홈 디렉터리의 남의 스크립트가 실행될 수 있음)가 닫힌다.

- 0단계(먼저 재현, 결과에 따라 분기): 이 정찰 세션은 `npm` 실행이 권한으로 막혀 **직접 재현하지 못했다(미확인)**. 구현자는 코드를 고치기 전에 다음 두 개를 돌려 출력을 원장에 그대로 붙일 것.
  1. `cd web && npm ci` → `npm run lint; echo EXIT=$?`
  2. 저장소 루트에서 `npm run lint; echo EXIT=$?`
  - (1)이 exit 0 이고 (2)가 exit 1 이면 아래 본 과제를 그대로 진행한다.
  - (1)이 exit 1 이면 **그쪽이 진짜 원인이다.** 본 과제를 버리고 (1)의 실제 오류를 고친다(그 경우 eslint 규칙을 끄거나 `ignores` 로 파일을 빼는 방향은 금지 — 소스를 고쳐 규칙을 만족시킨다). 지난 두 회차 모두 (1)은 exit 0 이었으므로 가능성은 낮다.
  - (2)가 exit 127 / `eslint: not found` 로 나오면 설치 누락이지 스크립트 누락이 아니다 — 아래 2단계의 의존성 확인 분기가 그 경우를 덮는다.

- 검토한 대안과 선택 이유:
  - **(채택) 루트 `package.json` + web 로 위임하는 `lint`/`test`/`build`.** 실제 eslint·vitest·vite 를 돌리므로 검증이 느슨해지지 않고, 러너가 쓰는 명령 문자열을 바꾸지 않아도 된다. 비용: 루트에 npm 파일 1개가 생긴다.
  - (기각) 러너 쪽에서 `cd web` 을 붙이게 한다 — 러너 설정은 이 저장소 밖이라 자율 회차에서 바꿀 수 없다. 저장소는 계속 "루트에서 CI 명령이 안 도는" 상태로 남는다.
  - (기각) `.github/workflows/ci.yml` 의 web 잡 각 step 에 `working-directory` 를 풀어 쓴다 — 지금도 CI 는 통과하므로 실패를 고치지 못하고, 보호 경로(workflows)를 건드리는 위험만 생긴다.
  - (기각) 루트 `package.json` 에 `workspaces: ["web"]` 선언 — node_modules 호이스팅과 `web/package-lock.json`·CI 의 `cache-dependency-path: web/package-lock.json` 전제를 깨뜨린다. **절대 하지 말 것.**
  - (기각) `scripts/web-verify.sh` 같은 별도 스크립트 — 러너가 실행하는 문자열은 `npm run lint` 이므로 실패가 그대로 남는다.
  - 가장 크게 기대고 있는 가정: **러너가 이 명령을 저장소 루트에서 돌린다**(지난 회차의 재현 결과로부터의 추론이며 러너 설정을 직접 보지는 못했다 — 미확인). 0단계가 이 가정을 먼저 검사한다.

- 수용 기준:
  1. 저장소 루트에서 `npm run lint` 가 **exit 0** 이고, 출력이 web 의 eslint 실행 결과다(`Missing script` 가 아니고, 린트 대상 파일 수가 `cd web && npm run lint` 와 동일 — 지난 회차 기록으로는 57~58파일, errors 0 / warnings 0).
  2. 저장소 루트에서 `npm test` 와 `npm run build` 도 web 의 같은 스크립트를 실행한다(각각 vitest run, `tsc -b && vite build`). `npm run build` 뒤 `git status --short` 가 **완전히 빈 출력**이어야 한다(`web/dist` 항목 없음, `web/dist/.gitkeep` 유지). 루트 `npm test` 가 `webidl.util.markAsUncloneable is not a function` 류로 실패하면 그것은 환경의 구버전 Node 심 문제이며 아래 "선택 작업" 분기를 따른다.
  3. web 안에서의 기존 동작이 변하지 않는다: `cd web && npm ci` → `npm run lint`·`npm test`·`npm run build` 가 루트 `package.json` 추가 **전후 동일한 결과**(루트 package.json 이 vite/vitest/eslint 의 설정·루트 탐색을 바꾸지 않았다는 증거). 세 명령의 exit code 와 린트 파일 수·테스트 파일 수를 전후로 비교해 원장에 적을 것.
  4. 느슨해진 것이 없다: `.github/workflows/ci.yml`, `web/package-lock.json`, `web/eslint.config.js`, `web/vite.config.ts`, `web/vitest.config.ts`, `web/src/test/keep-dist-placeholder.test.ts` 무변경. 루트 스크립트에 `|| true`, `--no-error-on-unmatched-pattern`, `--max-warnings`, `exit 0`, `--passWithNoTests` 없음. diff 전체를 이 패턴들로 확인한 결과를 적을 것.
  5. 테스트가 증명하는 것: **저장소 루트에서 npm 스크립트를 실행하면 이 저장소의 package.json 이 선택되고 web 의 린트가 실제로 돌아 exit 0** 이라는 것. 문자열 검사가 아니라 `child_process.spawnSync` 로 실제 실행해 exit code 와 출력으로 단언한다.
  6. `go build ./...`·`go vet ./...`·`go test ./... -count=1`·`go run ./cmd/api-surface-audit`(누락 0) 가 그대로 통과한다(Go 는 무변경이지만 루트에 파일이 추가되므로 한 번 확인).

- 건드릴 파일 (프로덕션 3 + 테스트 1 + 문서 1; 이 목록을 넘기면 과제를 쪼갤 것):
  - `package.json` (**신규, 저장소 루트**) — `{"name":"dataworks","private":true,"scripts":{...}}`. `workspaces` 금지, `dependencies`/`devDependencies` 금지. 스크립트는 web 으로 위임:
    `"lint": "node scripts/web-run.mjs lint"`, `"test": "node scripts/web-run.mjs test"`, `"build": "node scripts/web-run.mjs build"`.
    (더 단순한 `"lint": "npm --prefix web run lint"` 로 가도 좋다. 단 `--prefix` 가 실제로 cwd 를 web 으로 바꾸는지 **실행으로 확인**할 것 — 바뀌지 않으면 eslint 가 루트를 린트해 설정을 못 찾거나 파일 수가 달라진다. 확인이 애매하면 `"cd web && npm run lint"` 형태로. 어느 형태든 수용 기준 1 의 "파일 수 동일" 로 판정한다.)
    `"type"` 필드는 넣지 말 것(루트에 .js 파일이 없고, web 은 자기 `package.json` 의 `"type":"module"` 을 그대로 쓴다).
  - `scripts/web-run.mjs` (**신규**) — 인수로 받은 web 스크립트 이름을 `web/` 에서 실행하고 **exit code 를 그대로 전달**한다. 구현 지침: (a) `web/node_modules` 가 없으면 먼저 `npm ci` 를 web 에서 돌린다(CI 의 Install step 과 동일한 명령이며, 실패 시 그 exit code 로 즉시 종료 — 설치 실패를 삼키지 말 것). (b) 알 수 없는 스크립트 이름은 거부(허용 목록 `lint`/`test`/`build`). (c) 자기 자신을 다시 호출하는 형태(`npm run lint` 를 루트 cwd 로)는 만들지 말 것 — 무한 재귀가 된다. 반드시 cwd 를 `web` 으로 두고 web 의 스크립트를 부른다. (d) 기존 `web/scripts/run-with-supported-node.mjs` 가 쓰는 방식대로 PATH 의 node 를 믿지 말고 `process.execPath` / `process.env.npm_execpath` 를 쓸 것.
  - `.gitignore` — 루트 `node_modules/` 추가. 현재 `web/node_modules/` 만 무시하므로, 루트 `package.json` 이 생긴 뒤 누군가 루트에서 `npm install` 을 돌리면 추적되지 않은 `node_modules/` 가 생겨 "빌드 뒤 `git status --short` 가 깨끗" 규칙(수용 기준 2)을 깬다.
  - `web/src/test/root-npm-scripts.test.ts` (**신규 테스트**) — `// @vitest-environment node`. 저장소 루트를 `fileURLToPath(new URL('../../..', import.meta.url))` 로 구해 `spawnSync` 로 **실제** `npm run lint` 를 루트 cwd 에서 실행하고 `status === 0`, 출력에 `Missing script` 없음을 단언. 타임아웃은 `{ timeout: 180_000 }`(이 저장소에는 선례가 있다 — `web/src/test/keep-dist-placeholder.test.ts` 는 실제 `vite build` 를 돌린다). **루트 `npm test` 는 절대 spawn 하지 말 것**(vitest 안에서 vitest 를 다시 띄워 무한 재귀). npm 바이너리는 `process.env.npm_execpath` 가 있으면 `process.execPath` + 그 경로로, 없으면 `npm` + `shell: true` 로.
  - `docs/RELEASE_GUIDE.md` — 7행 `- [ ] \`web/\`에서 \`npm ci\`, \`npm run lint\`, \`npm test\`, \`npm run build\` 통과` 를, 루트에서도 같은 검증을 `npm run lint`/`npm test`/`npm run build` 로 돌릴 수 있다는 사실과 "루트 스크립트는 web 으로 위임만 하며 완화하지 않는다" 를 더해 갱신. (`docs/OPERATIONS.md` 2절에도 한두 줄 덧붙여도 좋다. `ls docs/*.pdf` 로 OPERATIONS·RELEASE_GUIDE 정본 PDF 가 없음을 먼저 확인할 것 — 지난 회차들이 OPERATIONS 는 없음을 확인했다.)

- 작업 순서 (각 단계 끝에서 트리가 동작하는 상태):
  1. 0단계 재현 → 출력 보관. 분기 판정.
  2. `web/src/test/root-npm-scripts.test.ts` 를 먼저 써서 **red** 확인(`cd web && npx vitest run src/test/root-npm-scripts.test.ts`, 또는 `npm test` 전체). 이때 실패 메시지가 `Missing script: "lint"` 를 담고 있어야 한다 — 담고 있지 않다면 가정이 틀렸으니 멈추고 원장에 적는다.
  3. 루트 `package.json` + `scripts/web-run.mjs` + `.gitignore` 추가 → 같은 테스트 green.
  4. 수용 기준 1~4 를 명령으로 확인(루트 `npm run lint`/`npm test`/`npm run build`, 그 뒤 `git status --short`, 그리고 `cd web` 에서 같은 세 명령).
  5. 프로덕션 파일(`package.json`, `scripts/web-run.mjs`)만 되돌려 2단계의 red 가 **다시 재현**되는지 확인 후 복원(이 저장소의 관례).
  6. 루트에서 `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `go run ./cmd/api-surface-audit`, `git diff --check`.
  7. 문서 갱신 → 커밋(제목 예: `fix(build): run web verification from the repository root`).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 루트: `npm run lint` / `npm test` / `npm run build` / `git status --short`
  - web: `cd web && npm ci && npm run lint && npm test && npm run build`
  - Go: `go build ./...` · `go vet ./...` · `go test ./... -count=1`(proxy 38~40초, store 14~15초) · `go run ./cmd/api-surface-audit`(누락 0, 550 routes / 612 OpenAPI paths)
  - 형식: `gofmt -l` 은 **Go 파일을 건드리지 않았으면 돌릴 필요 없다**(저장소의 기존 CRLF 때문에 무관한 파일명이 나온다). `git diff --check` 는 돌릴 것.

- 선택 작업 (수용 기준 2 의 `npm test` 가 환경 때문에 깨질 때만):
  - clean main(98c5081)에는 2026-10-01 회차의 Node 런처 커밋 `1d63b7c`(`web/scripts/run-with-supported-node.mjs` + `web/package.json` 의 `engines.node: ">=22.19.0"`)가 **들어 있지 않다.** 그 회차 원장은 이 워크트리류 환경에서 `cd web && npm test` 가 홈 디렉터리의 `node_modules/.bin/node`(Node 20.19.2)가 PATH 를 가로채 jsdom 30 → undici 8.10.0 로드 실패(`webidl.util.markAsUncloneable is not a function`)로 **exit 1** 임을 실행으로 증명했다. 루트 `npm test` 가 바로 그 오류를 내면 같은 수정을 이 브랜치에 되살릴 것(가능하면 `git cherry-pick 1d63b7c`, 객체가 없으면 두 파일을 직접 재작성). 이때 프로덕션 파일은 3 → 5개가 된다 — 상한 안이지만 그 이상으로 번지면 멈추고 루트 위임만으로 커밋한다.
  - 이 분기는 **lint 가 아니라 test 를 살리기 위한 것**이다. 이번 회차의 실패한 검증은 `npm run lint` 이므로 1순위는 어디까지나 수용 기준 1 이다.

- 위험과 피할 것:
  - `.github/workflows/ci.yml` 을 건드리지 말 것(보호 경로이며 지금도 통과한다). `web/package-lock.json` 을 `npm install` 로 재생성하지 말 것.
  - `workspaces` 선언 금지(호이스팅이 바뀌어 `web/package-lock.json` 과 CI 캐시 전제가 깨진다).
  - 루트에 `package-lock.json` 을 만들 필요는 없다. 다만 만들었다면 `npm ci` 가 루트에서도 돌아야 하고, 그 경우에도 **web 의 의존성은 `scripts/web-run.mjs` 가 설치**한다는 점을 유지할 것(루트 `npm ci` 는 web 을 설치하지 않는다).
  - 재귀: 루트 스크립트가 다시 루트 cwd 에서 `npm run <같은 이름>` 을 부르면 무한 루프. 테스트에서 루트 `npm test` 를 spawn 하는 것도 같은 함정.
  - 홈 디렉터리의 구버전 `node` 심(위 선택 작업)과 `web/node_modules` 부재(exit 127)는 서로 다른 실패다 — 섞어서 보고하지 말 것.
  - 자동 `npm ci` 는 web 의 의존성이 없을 때만 돌려야 한다. 매번 설치하면 검증이 몇 분씩 길어지고 lock 문제를 가린다.
  - `web/dist/*` 는 gitignore 되고 `!web/dist/.gitkeep` 만 추적된다 — `npm run build` 뒤 `.gitkeep` 이 사라지지 않았는지 반드시 확인(v0.9.57 의 `keepDistPlaceholder` 플러그인이 지킨다. 그 테스트의 단정을 약화하지 말 것).

- 추정(근거 포함): 유사 회차 기준(2026-10-01 의 런처 추가 회차가 프로덕션 2 + 테스트 1 + 문서 1) 25~45분, 80% 신뢰. 분해: 0단계 재현 `npm ci` 포함 8~12분, 테스트 red 5분, 루트 파일 작성 5분, 전체 검증(go test 전체 약 1분 + web 3명령 2~4분) 10분, 문서·커밋 5분. 예비(contingency) 10분은 선택 작업(Node 런처 되살리기)에 할당 — 이 하나가 알려진 변동 요인이다. 그 밖의 미지 범위는 예비로 덮지 않고, 프로덕션 파일 5개를 넘는 순간 범위를 줄여 커밋한다.

- 차선 후보: **keep-dist-placeholder 테스트의 `logLevel:'silent'` 를 실패 시에만 로그를 올려 재시도하도록 고치기** (가치 3 / 위험 1 / 작업량 S — `web/src/test/keep-dist-placeholder.test.ts`). 두 회차 연속 이 파일이 진단 비용을 올렸다. 단정(`.gitkeep` 이 빌드 후에도 존재)은 절대 약화하지 말고, 실패 경로에서만 `logLevel:'info'` 로 한 번 더 빌드해 vite 의 원인을 출력에 남기는 형태로. 0단계 재현이 본 과제의 가정을 반증했을 때 이것을 고를 것.
