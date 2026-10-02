- 과제: 수정 과제 — 두 회차 연속 verify 를 죽이는 `npm run typecheck && npm run build` 를 **먼저 재현해 원인을 확정한 뒤** 고치고, `make test` 가 CI 와 같은 `npm run build` 를 돌게 해서 같은 실패가 다시 로컬을 통과하지 못하게 하기 (가치 5 / 위험 2 / 작업량 S)
- 왜: 2026-10-02 두 회차(145736, 175737)는 모두 `make test` exit 0 을 증거로 냈지만 러너의 verify `npm run typecheck && npm run build` 에서 exit 1 로 죽었다 — `Makefile:test` 는 `npm ci && npm run typecheck && npm test` 만 돌고 **`npm run build`(= `tsc -b && vite build`)를 한 번도 돌리지 않기** 때문에, 구현자가 아무리 성실히 검증해도 CI·release 가 실제로 돌리는 명령을 보지 못한다(`.github/workflows/ci.yml:29-31` 의 "Frontend typecheck and build", `release.yml:33-39` 의 "Test source"). 빌드 실패의 원인을 고치고 그 명령을 로컬 검증 경로에 넣으면, 릴리즈 경로를 반복해 깨뜨리는 이 구멍이 닫힌다.

- **먼저 할 일(이것을 건너뛰면 이번에도 오진한다)**: 깨끗한 워크트리에서
  `cd web && npm ci && npm run typecheck; echo "TC=$?"; npm run build; echo "BUILD=$?"`
  를 돌려 **실제 오류 문장을 그대로 캡처**하라. 정찰 세션에서는 `npm ci` 가 권한으로 막혀 이 명령을 **한 줄도 돌리지 못했다 — 아래 원인 후보는 전부 미확인 가설이다.**
  지난 회차 과제서가 "원인 진단이 틀렸다" 로 차선 판정을 받은 이유가 정확히 이 지점이니, 캡처한 문장이 아래 가설과 다르면 **가설을 버리고 실제 문장을 따라가라.**

- 수용 기준:
  1) `cd web && npm ci && npm run typecheck && npm run build` 가 이 워크트리에서 **exit 0**. 출력(마지막 20줄)과 고치기 전의 실패 문장을 원장에 글자 그대로 남긴다.
  2) 고친 뒤 원인 지점을 한 토큰만 되돌리면 **같은 실패 문장이 그대로 재현**되고, 되돌리면 다시 통과한다(지난 회차가 `$npm_node_execpath` 로 보인 그 증명 방식).
  3) `Makefile` 의 `test` 타깃이 `npm run build` 를 포함해, `make test` 가 CI 의 프런트 검증과 같은 범위를 돈다. `make test` exit 0.
  4) `npm run build` 뒤 `go test ./internal/webui/` 가 통과한다 — `vite build` 가 `emptyOutDir` 로 `internal/webui/dist` 를 비우고 `web/public/README` 를 되돌려 놓는지를 `assets_test.go:TestEmbedAnchorIsRestoredByTheWebBuild` 가 지킨다.
  5) 빌드 뒤 `git status --short` 가 깨끗하다(`.gitignore` 가 `internal/webui/dist/*` 를 무시하고 `!internal/webui/dist/README` 만 추적하므로, README 가 바뀌면 더러워진다 = 실패).
  6) `.github/workflows/` 는 **0줄** 변경. 명령을 느슨하게(`|| true`, `--force`, 단계 삭제) 만드는 변경 금지.

- 원인 후보(전부 **미확인** — 재현 출력으로 고르라). 확인한 사실만 먼저:
  - 확인됨: `web/package.json` 의 `build` 는 `tsc -b && vite build`, `typecheck` 는 `tsc -b --pretty false`. `test` 는 아직 `node --test 'test/**/*.test.ts'`(지난 회차의 `$npm_node_execpath` 수정은 머지되지 않았다 — 워크트리는 main@494d00f 이고 `scripts.test` 에 그 변수가 없다).
  - 확인됨: 조상 `node_modules` 그림자가 **아직 살아 있다** — `ls -l /home/hkjang/node_modules/.bin/node` 가 `/home/hkjang/node_modules/node/bin/node` 로 해석된다(지난 회차 실측 v20.19.2). npm 은 조상 `node_modules/.bin` 을 PATH 앞에 세우므로 `tsc`·`vite` 의 `#!/usr/bin/env node` 가 이 v20 을 집는다. 셸의 `node --version` 은 v22.23.1 이라 **셸에서 재어 보면 안 된다** — 반드시 `npm run` 안에서 재라(`"node -p \"process.version\""` 를 임시 스크립트로 추가하거나 `npm exec -- node -p process.version`).
  - 가설 ① (가장 유력): 위 v20 그림자 때문에 `vite build` 가 죽는다. `vite@7.3.6` 의 engines 는 `^20.19.0 || >=22.12.0`(package-lock.json:1705-1707)이라 v20.19.2 는 숫자상 통과하므로 **engines 경고로는 안 나타난다** — 실제 오류 문장을 봐야 한다. 맞다면 고칠 자리는 `web/package.json` 의 `build`·`typecheck` 두 줄을 지난 회차와 같은 관용으로 npm 자신의 node 에 고정하는 것:
    `"typecheck": "$npm_node_execpath node_modules/typescript/bin/tsc -b --pretty false"`,
    `"build": "$npm_node_execpath node_modules/typescript/bin/tsc -b && $npm_node_execpath node_modules/vite/bin/vite.js build"`.
    (지난 회차가 `test` 에서 이 패턴이 실제로 도는 것을 확인했다. 머지되지 않았으므로 `test` 줄도 같이 고쳐 세 줄을 한 관용으로 맞춰도 좋다 — 같은 파일 1개다.)
  - 가설 ②: `tsc -b` 쪽 실패. `web/tsconfig.json` 은 `{"files":[],"references":[{"path":"./tsconfig.app.json"}]}` 인데 `tsconfig.app.json` 에 `composite: true` 가 **없다**(실제로 열어 확인). 빌드 모드가 이를 거부하면 `TS6306`/`TS6310` 이 뜬다. 다만 지난 회차의 `make test` 가 `npm run typecheck`(= 같은 `tsc -b`)를 통과했다고 기록하므로 **typecheck 단계는 아마 무죄**다. 재현에서 `TC=0`·`BUILD=1` 이면 ② 는 버려라.
  - 가설 ③: `vite build` 가 `outDir: '../internal/webui/dist'` 를 `emptyOutDir: true` 로 비우는 과정에서 권한/경로 문제(WSL `/home/hkjang/.cache/...`). `vite.config.ts:6-11` 참조.
  - 가설 ④: 네이티브 선택적 의존성 누락. 다만 `package-lock.json` 에 `@rollup/rollup-linux-x64-gnu`(:1025)와 `@esbuild/linux-x64`(:541)가 **둘 다 들어 있음을 확인**했으므로 가능성은 낮다.

- 건드릴 파일 (목표 2개, 최대 3개):
  - `web/package.json:scripts` — 재현으로 확정된 원인에 해당하는 스크립트 줄만. (가설 ① 이면 `build`·`typecheck`, 겸해서 `test`.)
  - `Makefile:test` — 기존 순서(`go test` → `go vet` → `npm ci` → `typecheck` → `npm test`)를 유지한 채 끝에 `npm run build` 를 잇는다. ed0352e 가 `npm test` 를 같은 방식으로 이어 붙였고 채택된 전례다. `build` 타깃(`web:` → `npm ci && npm run build`)은 건드리지 말 것.
  - (원인이 다른 곳이면) 재현이 지목한 그 파일 하나. 그 경우 과제서의 가설을 따르지 말고 원장에 "과제서의 원인 진단이 틀렸다" 고 적어라.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run typecheck && npm run build`  ← **이번 회차의 본 명령. 반드시 exit 0 을 눈으로 볼 것**
  - `cd web && npm test --silent`  (27 pass/0 fail 가 기준선)
  - `make test`  (go test → go vet → npm ci → typecheck → npm test → **새로 추가한 npm run build**)
  - `go test ./internal/webui/ -run TestEmbedAnchorIsRestoredByTheWebBuild -v`
  - `go build ./... && go vet ./... && go test ./...`
  - `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh`
  - `git status --short` 와 `git diff --check` (빌드 산출물·임시 디렉터리가 남지 않았는지)

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들어 통과시키는 것은 금지.** `.github/workflows/` 는 0줄 — 2026-09-06 에 릴리즈 경로를 반복해 깨뜨려 되돌림 PR + 자율화 강등을 받은 전례가 있다.
  - `npm run build` 는 `internal/webui/dist` 를 **비운다**. 돌린 뒤 `internal/webui/dist/README` 가 살아 있는지(= `web/public/README` 와 바이트가 같은지) 반드시 확인하라. 없어지면 `go build ./...` 가 `//go:embed dist/*` 에서 깨진다.
  - `web/test/login.test.ts` 는 `web/test/.login-test-*/` 임시 디렉터리를 만들고 **정상 종료 때만** 지운다. 출력을 `head` 로 파이프하지 말 것(지난 회차에 SIGPIPE 로 쓰레기가 남았다).
  - 셸에서 잰 `node --version`(v22.23.1)을 npm 스크립트 안의 node 와 같다고 **가정하지 말 것** — 지난 회차 오진의 원인이 정확히 이것이다.
  - grep 으로 "문자열이 바뀌었다" 를 증거로 내지 말 것. 증거는 명령의 실제 출력과 exit code 다.
  - `go.mod`·`migrations/`·`internal/auth`·`internal/oidc` 는 이번 과제와 무관 — 건드리지 말 것.

- 차선 후보: 원인이 재현되지 않고 `cd web && npm ci && npm run typecheck && npm run build` 가 **그냥 통과**하면(= 러너 환경만의 일시적 실패), 코드를 억지로 고치지 말고 수용 기준 3·4·5 만 수행하라 — `Makefile:test` 에 `npm run build` 를 이어 붙여 CI 의 프런트 검증 범위를 로컬 `make test` 와 일치시키고, 통과 출력을 원장에 남긴다(파일 1개). 그 다음 차선은 `web/package.json` 에 `esbuild` 를 devDependency 로 선언하기 — `web/test/login.test.ts` 가 선언되지 않은 vite 전이 의존성 `esbuild@0.28.2`(package-lock.json:1323)를 import 하고 있어 vite 가 그것을 올리지 않게 되는 순간 프런트 테스트가 통째로 죽는다(파일 2개, `npm install esbuild@0.28.2 --save-dev` 로 lock 재생성 필요).
