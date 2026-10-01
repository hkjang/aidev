- 과제: 프런트엔드 테스트가 워커 시작부터 깨지는 것을 고치기 — `jsdom`→`undici` 의존 사슬을 이 저장소가 고정한 Node 에서 도는 조합으로 맞추기 (가치 5 / 위험 3 / 작업량 M)

- 왜: 지난 회차의 검증이 `cd web && npm test --silent` (exit 1) 로 끝났고 같은 이유로 릴리즈 워크플로가 두 번 실패했다 — Go 쪽 변경(5bed9dc)은 전부 녹색이었는데 프런트엔드 단계 하나가 회차를 떨어뜨렸다. 이것을 고치지 않으면 어떤 과제를 골라도 릴리즈까지 가지 못한다.

- 사실로 확인한 것 (이번 정찰이 파일을 열어 확인):
  - `web/package.json`: `jsdom 30.0.1`, `vitest 4.1.10`. 테스트 환경은 `web/vite.config.ts` 의 `test: { environment: 'jsdom', setupFiles: './src/test/setup.ts' }` 하나뿐이고 pool 설정은 없다(vitest 기본값).
  - `web/package-lock.json:4963-4972`: `undici` **8.10.0** 정확 고정, `"engines": { "node": ">=22.19.0" }`, dev 의존.
  - `jsdom/lib/api.js:12` 이 `require("undici")` 를 **모듈 로드 시점에** 한다 → undici 진입점이 `lib/web/cache/cachestorage.js:20` 의 `webidl.util.markAsUncloneable(this)` 를 클래스 정의 중에 실행한다.
  - `undici/lib/web/webidl/index.js:5` = `const { markAsUncloneable } = require('node:worker_threads')`, `:161` = `webidl.util.markAsUncloneable = markAsUncloneable`. **즉 Node 의 `worker_threads` 에 그 export 가 없으면 `undefined` 가 대입되고, 지난 회차가 본 `TypeError: webidl.util.markAsUncloneable is not a function` 이 그대로 난다.** 이것이 지난 회차 진단의 기계적 근거다.
  - 워크플로 셋 다 Node **22.23.0** 을 고정하고 `npm ci && npm run test && npm run build` 를 돈다(`.github/workflows/ci.yaml:32,49,67`, `release.yaml:61,66`). 로컬 nvm 은 22.23.1, `/usr/bin/node` 는 25.9.0(지난 회차 기록).
  - `web/package.json`·`web/package-lock.json` 은 v0.9.92~v0.9.96 동안 **릴리즈 커밋의 version 줄 말고는 바뀌지 않았다**(`git log -- web/package.json web/package-lock.json`). 즉 의존 버전 자체는 안 움직였고, 바뀐 것은 이 조합을 돌리는 Node 쪽이다.

- 미확인 (구현자가 **가장 먼저** 확인할 것): 이번 정찰은 이 worktree 에 `web/node_modules` 가 없는 상태로 시작했고(확인함), `npm ci` 와 `node -e` 가 이 세션의 권한에서 거부되어 **실행으로 재현하지 못했다**. 그래서 아래 1단계가 분기점이다.

- 수용 기준:
  1) **진단 먼저.** `cd web && npm ci` 뒤 `node -p "typeof require('node:worker_threads').markAsUncloneable"` 를 `node --version` 과 함께 기록한다.
     - `"undefined"` 면 위 사슬이 원인으로 **증명된다** → 기준 2 로 간다.
     - `"function"` 이면 위 진단은 **틀렸다**. 의존성을 한 글자도 바꾸지 말고 `cd web && npx vitest run 2>&1 | head -60` 의 실제 스택을 근거로 다시 진단한 뒤, 과제서의 이 분기를 벗어난다는 사실을 회차 노트에 적고 진행한다.
  2) `cd web && npm test --silent` 가 **exit 0**, 29개 테스트 파일 전부 수집·실행되고 161개 테스트가 통과한다(v0.9.96 기준선과 같은 수). "0 passed" 나 파일 건너뛰기로 통과시키는 것은 실패로 본다.
  3) `cd web && npm run build` 와 `cd web && npm run lint`(`eslint . --max-warnings 0`) 가 그대로 통과하고, `tsc --noEmit` 도 깨지지 않는다. `webui/dist/index.html` 은 빌드가 바꾸므로 **되돌린다**.
  4) `go test -race ./internal/httpserver -run '^TestIntegration' -count=1` 이 서비스를 띄운 상태에서 기존대로 통과한다(프런트엔드 변경이 Go 쪽을 건드리지 않았음을 보이는 선).
  5) 테스트 파일(`web/src/**/*.test.tsx`)의 단언은 **고치지 않는다.** 테스트를 느슨하게 만들어 통과시키는 것은 금지다 — 깨진 것은 테스트가 아니라 런타임 조합이다.

- 건드릴 파일 (프로덕션 2개):
  - `web/package.json` — 아래 옵션 중 하나. **1안(권장): `"overrides": { "undici": "<markAsUncloneable 를 쓰지 않는 최신 7.x>" }` 를 더한다.** jsdom 30 을 그대로 두고 문제의 한 패키지만 내린다. 2안: `devDependencies.jsdom` 을 최신 **29.x** 로 내린다(29 계열은 undici ^7 을 요구한다 — 설치해 확인할 것). 3안은 아래 "위험" 참조.
  - `web/package-lock.json` — `npm install`(또는 `npm install --package-lock-only`)로 **재생성**. 손으로 고치지 말 것. `undici` 항목의 `version`·`resolved`·`integrity` 와 `engines`, 그리고 `node_modules/jsdom` 의 `dependencies.undici` 범위가 서로 맞는지 눈으로 확인한다.
  - 선택: 같은 변경으로 `jsdom`/`undici` 를 왜 고정했는지 한 줄 주석을 남길 자리가 package.json 에 없으므로, `README.md` 의 개발 섹션이나 `docs/` 가 아니라 **커밋 메시지 본문**에 남긴다(이 저장소는 "왜" 를 코드 주석에 길게 쓰지만 JSON 에는 쓸 수 없다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci` → `cd web && npm test --silent` (= 러너의 실패한 검증 그 자체. 반드시 이것으로 확인)
  - `cd web && npm run test`(상세 출력) · `cd web && npm run build` · `cd web && npm run lint`
  - `eval "$(scripts/test-services.sh)"` 로 서비스를 잡은 **같은 셸에서** `make test` — 끝의 `"integration test(s) did not run"` 경고가 **없어야** SKIP 0 이다(프로필의 검증 함정).
  - `make lint` (golangci-lint + govulncheck + eslint)
  - `git diff --check`, `git status` 로 `webui/dist/index.html` 복원 확인

- 위험과 피할 것:
  - **3안(워크플로의 Node 핀을 25.x 로 올리기)은 이번 회차에 고르지 말 것.** `.github/workflows/release.yaml` 은 보호 경로이고, 운영자 규칙 "릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것" 이 걸린다. 게다가 지난 회차가 `/usr/bin/node` 25.9.0 으로 돌렸을 때 `CommandPalette.test.tsx`·`realms.test.tsx` 의 **19개가 실패**했다 — Node 를 올리면 과제가 두 개가 된다. Node 핀은 22.23.0 그대로 둔다.
  - `vite.config.ts` 의 `test.pool` 을 바꿔 우회하지 말 것 — 지난 회차가 `pool=threads` 로 바꿔도 같은 오류가 났다고 기록했다. 증상만 옮긴다.
  - `environment: 'jsdom'` → `happy-dom` 교체는 29개 테스트 파일 전부의 동작이 바뀔 수 있어 한 세션에 끝나지 않는다.
  - npm 레지스트리 접근이 필요하다. 네트워크가 막히면 **의존성을 바꾸지 말고** 기준 1 의 진단 결과와 막힌 사실만 회차 노트에 남기고 멈출 것(반쯤 바뀐 lockfile 을 남기는 것이 최악이다).
  - `web/node_modules` 는 이 worktree 에 없다 — `npm ci` 가 첫 명령이다. `Makefile:70` 의 `test: web/node_modules` 전제도 그래서다.
  - 전역 설정(nvm default, `~/.npmrc`)을 백업 없이 바꾸지 말 것 — 운영자 규칙.

- 차선 후보: 기존 Token 연동 테스트 둘(`TestIntegrationTokenSaysWhenItCouldNotReadTheAccount`:integration_test.go:1130, `…ReadTheRealm`)에 `srv.Metrics()` 등록기를 붙여 `resso_token_errors_total{grant_type}` 배선을 고정하기 (가치 2 / 위험 1 / 작업량 S, 프로덕션 파일 0개). **단 이것은 프런트엔드가 고쳐지지 않으면 역시 `make test` 에서 떨어진다** — 1순위가 네트워크 등으로 성립하지 않을 때만, 그리고 검증을 `go test -race ./internal/httpserver` 로 한정해 돌릴 것.
