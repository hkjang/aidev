# 과제서 (정찰, 2026-10-15)

- 과제: 수정 과제 — `npm test --silent` 가 **러너 작업 트리에서** 통과하게 고침(npm 이 PATH 에 끼워 넣는 상위 `node_modules/.bin/node` v20 무력화 + 테스트 준비물 자동 생성) (가치 5 / 위험 2 / 작업량 M)

- 왜: 러너 검증 `npm test --silent` 가 이 저장소에서 두 회차 연속 exit 1 로 끝났다. 원인이 **둘**인데 지난 두 회차는 ①만 고쳤고(그마저 main 에 없다) ②를 "환경 문제" 로 남겨 같은 실패를 반복했다 — ① 베이스 `package.json`(94eb01f)에 `pretest` 가 없어 gitignore 된 `dist`·`dist-draft`·`dist-email` 을 아무도 만들지 않고 `playwright.config.ts` 의 webServer 가 죽는다 ② npm 이 cwd 에서 루트까지 **모든** 상위 `node_modules/.bin` 을 원래 PATH 보다 앞에 붙이므로, 작업 트리가 `/home/hkjang/.cache/auto-improve-wt/irumx-www` 인 한 `/home/hkjang/node_modules/.bin/node`(= node 패키지 **20.19.2**)가 `astro`·`wrangler`·`playwright` 의 `#!/usr/bin/env node` 를 가로채 `engines: >=22.12` 를 위반한다. ②를 저장소 쪽에서 막으면 검증을 느슨하게 하지 않고도 `npm test` 가 이 기계에서 혼자 통과한다.

## 이번에 직접 확인한 근거 (추측 아님)
- `/usr/lib/node_modules/npm/node_modules/@npmcli/run-script/lib/set-path.js:27-37` — `do { pathArr.push(resolve(p,'node_modules','.bin')); p = dirname(p) } while (p !== pp)` 로 **루트까지 올라가며** .bin 을 넣고, 원래 `PATH` 는 `pathArr.push(nodeGypPath, PATH)` 로 **맨 뒤**에 붙는다. node 실행 파일의 디렉터리는 **넣지 않는다**. → 상위 .bin 의 `node` 가 항상 이긴다.
- `/home/hkjang/node_modules/node/package.json:3` → `"version": "20.19.2"` (npm 패키지 `node`, 홈에 설치됨). 이것이 가로채는 주범.
- `/usr/lib/node_modules/npm/node_modules/@npmcli/config/lib/set-envs.js:110` → `env.NODE = env.npm_node_execpath = config.execPath` — **npm 은 자기를 실행한 node 의 절대 경로를 스크립트에 알려 준다.** 처방의 근거.
- 셸 `node -v` → `v22.23.1`. 작업 트리에 `node_modules/`·`.github/` 없음, `package.json` 에 `pretest` 없음(`ls -a`, Read 로 확인). 즉 지난 두 회차의 성공은 main 에 없다 — 그대로 다시 올리는 것은 금지된 반복이므로 ①과 ②를 **한 번에** 고친다.

## 수용 기준
1. `/home/hkjang/.cache/auto-improve-wt/irumx-www`(= `/home/hkjang` **아래**)에서 `npm ci` → `rm -rf dist dist-draft dist-email` → **`npm test --silent` 한 줄만** → exit 0, **77 passed · 4 skipped · 0 failed**. `/tmp` 클론 결과는 증거로 쓰지 말 것 — 거기선 고치기 전에도 통과한다.
2. 같은 작업 트리에서 `npm run build --silent` → `✓ 모두 통과`, exit 0. (고치기 전엔 `Node.js v20.19.2 is not supported by Astro!` 로 exit 1 이어야 한다 — 이 red 를 먼저 눈으로 보고 로그를 남길 것.)
3. 처방이 최종 출력까지 닿았음을 end-to-end 로 증명: 같은 작업 트리에서 npm 스크립트가 실제로 쓰는 node 가 v22 임을 보일 것(예: `npm run check` 가 통과 — v20 이면 Astro 가 거부한다). 소스 문자열 검사나 `/tmp` 결과로 대체하지 말 것.
4. 검증을 느슨하게 하지 않았음: `git diff main -- 'tests/*.spec.ts' playwright.config.ts` 가 **비어 있고**, 새로 추가한 `test.skip` 은 **0건**, `engines.node` 는 `>=22.12.0` 그대로, spec 의 단정·기대값·timeout 변경 0건. (건너뜀 4건은 원래 있던 `tests/site.spec.ts` 의 폭 조건부 스킵.)
5. `pretest` 를 dist 유무로 조건화하지 **않았다**. (평범한 `build` 의 dist 에는 테스트 Turnstile 키가 없어 조건을 붙이면 낡은 dist 로 `contact-send.spec.ts` 가 엉뚱하게 실패한다.)

## 건드릴 파일 (프로덕션 1개 + 문서 1개)
- `package.json:10-21` `scripts` — 두 가지만:
  1. `pretest` 추가: `npm run test:build && npx playwright install chromium`. 조건부 금지(기준 5).
  2. **각 스크립트 문자열 앞에 PATH 교정 한 조각**을 붙인다:
     `export PATH="${npm_node_execpath%/*}:$PATH"; ` …(이어서 원래 명령)
     - `npm_node_execpath` 는 npm 이 주는 "지금 npm 을 돌리는 node" 의 절대 경로다(위 set-envs.js:110). 머신 경로를 박아 넣는 것이 아니고 engines 를 낮추는 것도 아니다 — **engines 가 요구하는 그 node 로 도구를 돌리는 것**이다.
     - `export …;` 로 시작해야 뒤따르는 `&&` 체인 전체에 적용된다(`VAR=x cmd1 && cmd2` 는 cmd1 에만 걸린다).
     - **중첩 `npm run` 은 PATH 를 다시 덮으므로 바깥에만 붙이면 소용없다.** 최소한 `build`, `check`, `test:build`, `pretest`, `test` **전부**에 붙일 것. 일관성을 위해 `dev`·`preview`·`fonts`·`images`·`shots`·`deploy` 까지 붙이는 것을 권한다(같은 파일, 같은 한 조각).
     - `npm_node_execpath` 가 비어 있어도 안전하다(`${…%/*}` 가 빈 문자열 → `PATH=":$PATH"`). 더 엄격히 하려면 `[ -n "$npm_node_execpath" ] && export PATH=…;` 형태도 가능하나 `set -e` 없는 sh 에서 뒤 명령이 계속 도는지 직접 확인할 것.
- `README.md:28 근처` — `npm test` 가 준비물(빌드 세 벌·Chromium)을 알아서 만든다는 한 줄. 기존 `npm run test:build` 줄은 지우지 말 것.
- 이번 회차에 **건드리지 말 것**: `tests/*.spec.ts`, `playwright.config.ts`, `scripts/serve-static.mjs`, `worker/`, `src/`, `.github/`(베이스에 없음), `engines`.

## 검증 명령 (이 저장소에서 실제로 도는 것, 전부 작업 트리에서)
```
# red 먼저 (고치기 전)
npm ci
npm run build --silent          # → Astro v20 거부로 exit 1 기대, 로그 저장
rm -rf dist dist-draft dist-email && npm test --silent   # → exit 1

# green (고친 뒤)
npm run build --silent          # → "✓ 모두 통과", exit 0
npm run check                   # → astro check 통과 (= 도구가 v22 로 돈다는 end-to-end 증거)
rm -rf dist dist-draft dist-email && npm test --silent   # → 77 passed · 4 skipped · 0 failed, exit 0
git diff main -- 'tests/*.spec.ts' playwright.config.ts  # → 비어 있어야 함
```
`pretest` 가 빌드 세 벌(이미지 411장 × 3)을 돌려 전체 2분 안팎 걸린다(이전 회차 실측: 테스트 자체는 28초). 느리다고 조건부로 만들지 말 것.

## 위험과 피할 것
- **이 처방이 기대는 가정 하나**: 러너가 `npm test` 를 **v22 로 돌아가는 npm** 으로 부른다는 것(그래야 `npm_node_execpath` 가 v22 를 가리킨다). 셸 `node -v` = v22.23.1 이라 성립할 것으로 보지만 **미확인**이다 — `npm exec` 가 이 세션에서 차단돼 직접 못 읽었다. 구현자는 **가장 먼저** 작업 트리에서 `npm run check` 류로 교정 전/후 node 버전을 실제로 확인하라. 만약 npm 자체가 v20 으로 돌고 있으면 이 처방은 무력하고, 그때는 저장소로 해결할 수 없는 환경 문제임을 원장에 명확히 적고 ①(pretest)만 남기지 말고 **차선 후보로 전환**하라.
- Windows 기본 셸(cmd.exe)에서는 `export`·`${v%/*}` 가 깨진다. 이 저장소는 WSL/Linux 에서 쓰이고 `.npmrc` 는 없다. 전 사용자에게 영향 가는 `script-shell` 설정을 새로 넣지 말고, README 에 sh 계열 셸 전제를 한 줄 적는 선에서 멈출 것.
- 포트: `playwright.config.ts` 는 8820~8823 고정 + `reuseExistingServer: false`. 점유되어 있으면 즉시 실패한다 — 실패하면 node 문제로 단정하기 전에 `ss -ltn` 으로 확인하라. 포트 동적 할당(`scripts/free-ports.mjs`)은 과거 회차의 미머지 변경이고 이번 범위 **밖**이다.
- `scripts/verify-build.mjs` 가 `harfbuzzjs`·`fontverter` 를 직접 import 하는데 devDependencies 에 선언이 없다(이번 Read 로 devDependencies 8개 확인; import 줄은 미확인). `npm ci` 후 `npm run build` 가 모듈 없음으로 깨지면 이 과제가 아니라 그 선언 누락이다 — 그때만 최소로 더하고 원장에 적을 것.
- 옛 커밋(`00b14d7`·`415c9c0`·`d72fe85`) 무조건 체리픽 금지. 그 변경들은 ②를 고치지 않아 같은 실패로 끝났다.
- 빌드 산출물(`dist*`) 커밋 금지.

## 차선 후보
`scripts/serve-static.mjs` 하드닝 — ① listen 전에 제공할 폴더와 `index.html` 확인 후 없으면 한국어 한 줄 + exit 1 ② 404 분기에서 `404.html` 을 **먼저 읽어 성공한 뒤** 헤더를 쓰고, 실패하면 404 + 짧은 평문(현재는 헤더를 먼저 쓰고 무방비 `readFile` → 프로세스가 죽는다) ③ 경로 가드를 `full !== base && !full.startsWith(base + sep)` 로 바꿔 `..%2f` 형제 폴더 접근 차단. **이 과제는 npm 을 거치지 않고 `node scripts/serve-static.mjs …` 를 직접 spawn 하는 저장소 밖 하네스로 검증할 수 있어, node@20 문제가 해결되지 않아도 증명이 가능하다.** 1순위의 가정(위)이 깨지면 이것으로 전환하라.
