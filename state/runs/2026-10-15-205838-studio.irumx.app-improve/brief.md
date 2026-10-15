- 과제: `npm test` 가 PATH 의 엉뚱한 Node 로 `wrangler dev` 를 띄우려다 webServer 가 죽는다 — 시험 하네스를 저장소 밖 `node_modules/.bin` 으로부터 격리 (가치 5 / 위험 2 / 작업량 S)

- 왜: `playwright.config.ts:26` 이 시험 서버를 `npx wrangler dev …` 로 띄우는데, npm 은 실행할 때 **cwd 의 모든 조상 디렉터리의 `node_modules/.bin` 을 PATH 앞에 붙인다**. 이 러너 환경에는 저장소 밖(`/home/hkjang/node_modules/.bin/node`)에 Node 20 이 놓여 있어 그것이 시스템 Node 를 가리고, wrangler 4.147.0(`package.json` engines `>=22.12.0`)이 "Node.js v22 필요" 로 즉시 죽는다 → `webServer` 가 올라오지 않아 api·desktop·mobile 시험이 전부 실패하고 `npm test --silent` 가 exit 1 이 된다. 지금까지 두 회차가 이것을 **저장소 밖에서 손으로** 비켜 가며(심링크 재지정·이동) 넘겼고, 복구한 회차는 검증이 실패했다 — 저장소 안에 고정하지 않으면 다음 회차도 같은 이유로 또 실패한다.

- 확인한 근거(추측 아님):
  - `playwright.config.ts:22-31` — webServer 2개. `:23` 은 `node tests/mock-resend.mjs 8849`, `:26` 은 `node scripts/db-local.mjs … && npx wrangler dev --port 8848 …`. 둘 다 **PATH 가 주는 `node` 를 그대로 쓴다**. `timeout: 240_000`.
  - `package.json:7-9` engines `node >=22.12.0`, `:40` wrangler `4.147.0`, `:19` `"test": "playwright test"`.
  - 앞 회차 원장 2건이 같은 증상을 독립적으로 기록했다:
    - `state/runs/2026-10-15-185856-…/journal.md:18` — "이 환경에서는 `npx` 가 cwd 조상의 `/home/hkjang/node_modules/.bin/node`(v20)를 집어 `wrangler` 가 'Node.js v22 필요' 로 죽는다 — 그 심링크를 v22 로 임시 재지정해 돌리고 **원래 대상으로 복구해 두었다**". 같은 파일 `:22` — `[러너 19:33] verify failed — 실패한 검증: npm test --silent (exit 1)`.
    - `state/runs/2026-10-15-195847-…/journal.md:12` — "…`npx wrangler` 가 Node 20 으로 떠 webServer 가 죽었다. `node.off-wt-session` 으로 옮겨 두었다". 같은 파일 `:13` — `[러너 20:10] verify passed`.
    - 두 회차는 base 가 같다(`main@b00d53c`). 차이는 **저장소 밖 심링크를 되돌렸는지 여부**뿐이다.
  - **정정**: 이 base 에는 `.github/workflows/` 가 아예 없다(CI 워크플로는 머지되지 않은 브랜치 `2cdba01` 에만 있다). 따라서 "실패한 워크플로 단계" 는 GitHub Actions 파일이 아니라 **러너가 직접 돌리는 `npm test --silent`** 다. Actions YAML 을 찾아 고치려 하지 말 것.
  - 미확인: 이 세션의 샌드박스가 저장소 밖 읽기·복합 명령을 막아 `/home/hkjang/node_modules/.bin/node` 가 **지금도** 있는지는 직접 확인하지 못했다(1958 회차가 옮겨 둔 상태일 수 있다). 구현자는 `ls -la /home/hkjang/node_modules/.bin/` 로 먼저 확인하고, 없으면 아래 "재현" 의 가짜 스텁으로 재현할 것 — 결함(하네스가 PATH 를 믿는다)은 심링크가 있든 없든 그대로다.

- 수용 기준:
  1. 저장소 **밖** 디렉터리의 `node_modules/.bin` 에 Node 22.12 미만(또는 즉시 exit 1 하는 스텁) `node` 가 놓여 PATH 앞에 붙어도 `npm test` 가 전부 통과한다. 즉 webServer 2개가 저장소 밖 `node_modules/.bin` 의 `node`·`wrangler` 를 집지 않는다.
  2. 저장소 **안**(`<root>/node_modules/.bin`)의 항목과 그 밖의 평범한 PATH 항목(`/usr/bin` 등)은 그대로 남는다 — `npx`·`wrangler` 가 계속 로컬 설치본으로 돈다.
  3. 요구 Node 를 못 찾으면 240초 webServer 타임아웃 + 50여 개 실패가 아니라, **첫 몇 초 안에 원인을 말하는 한국어 오류 한 줄**로 죽는다(어느 PATH 항목이 가렸는지 포함).
  4. 시험이 증명하는 것: PATH 걸러내기 로직이 ① 저장소 밖 `node_modules/.bin` 을 **뺀다** ② 저장소 안 `node_modules/.bin` 과 일반 디렉터리는 **남긴다** ③ PATH 가 비어 있거나 상대 경로·중복이 섞여도 터지지 않는다. 단위 시험(브라우저·DB 불필요)으로.
  5. 종전 통과 수 **76개**(base `b00d53c` 기준) + 신규 단위 시험. 줄어들면 안 된다.

- 건드릴 파일 (3개 — 더 늘리지 말 것):
  - `tests/lib/harness-path.mjs` (신규) — `repoBinOnlyPath(rawPath, repoRoot)`: PATH 를 `path.delimiter` 로 쪼개 `node_modules/.bin` 으로 끝나는 항목 중 `repoRoot` 밖인 것만 제거하고 다시 합쳐 돌려준다(`path.relative(repoRoot, path.resolve(p)).startsWith('..')` 로 밖/안 판정, 빈 항목은 버린다). 부수효과 없는 순수 함수 하나만 export 할 것 — 시험이 직접 부른다.
  - `playwright.config.ts` — 위 함수를 import 해 `const PATH = repoBinOnlyPath(process.env.PATH ?? '', import.meta.dirname)` 를 만들고, **webServer 2개 모두**에 `env: { ...process.env, PATH }` 를 준다. `command` 문자열·`url`·`timeout`·`reuseExistingServer`·`projects`·`workers: 1` 은 그대로 둘 것.
  - `scripts/db-local.mjs` — 맨 앞에 Node 버전 가드(수용 기준 3). `process.versions.node` 를 `22.12.0` 과 비교해 미달이면 실행 중인 `process.execPath` 와 원인(저장소 밖 `node_modules/.bin` 이 PATH 를 가렸을 수 있음)을 적어 `process.exit(1)`. 이 스크립트가 `:26` webServer 명령의 **첫 단계**라서 wrangler 보다 먼저 걸린다. 가드만 더하고 DB 초기화 로직은 건드리지 말 것.
  - 신규 시험 `tests/unit-harness.spec.ts` — `testMatch: /unit-.*\.spec/` 에 자동으로 잡힌다(`playwright.config.ts:33`). `tests/lib/` 는 `.spec` 이 아니므로 어느 project 에도 안 잡힌다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  1. `npm ci` — 이 워크트리에는 `node_modules` 가 아예 없다(확인함).
  2. `npx playwright install chromium` (1회)
  3. `npm run build` — 느리다. 끝에 "빌드 점검 통과" 가 나와야 한다. **`npm test` 전에 반드시** (webServer 가 `dist` 를 띄운다)
  4. `npm test` — 기대: 76 + 신규. `npm run test:unit` 로 새 단위 시험만 빨리 돌릴 수 있다.
  5. **재현(고치기 전에 먼저)**: 저장소 밖에 스텁을 만들고 PATH 앞에 붙여 `npm test` 가 지금 실패하는 것을 확인한다 —
     `mkdir -p /tmp/fakenode/node_modules/.bin`, 그 안에 `node` 라는 실행 파일로 `#!/bin/sh` + `echo "Node.js v22 필요" >&2; exit 1`, 그리고 `PATH=/tmp/fakenode/node_modules/.bin:$PATH npm test`. 고친 뒤 같은 명령이 통과해야 한다(수용 기준 1). **저장소 밖 심링크를 손으로 옮겨 통과시키는 것은 이번 회차의 금지 사항이다** — 그게 두 회차를 실패시킨 원인이다.

- 위험과 피할 것:
  - **워크플로·검증을 느슨하게 만들지 말 것**: `timeout` 늘리기, `retries` 추가, project 제외, `--silent` 바꾸기, 실패 시험 skip — 전부 금지.
  - `cron.ts`·`files.ts`·`mp4.ts` 는 **건드리지 말 것**. 머지 대기 중인 브랜치 4개가 그 파일들을 들고 있어 충돌한다.
  - `.github/workflows/` 는 이 base 에 없다. 새로 만들지도 말 것(별건, 보호 경로).
  - `reuseExistingServer: true` 를 이번에 바꾸지 말 것 — 차선 후보의 영역이고, 같이 바꾸면 원인 분리가 안 된다.
  - 환경 함정(앞 회차들이 기록함, 저장소 변경 아님): ① Playwright 브라우저가 `$HOME` 쪽에서 깨져 있으면 `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 로 돌릴 것 ② `.wrangler/test-state` 가 앞 회차 찌꺼기를 들고 있을 수 있다 — `node scripts/db-local.mjs --persist-to .wrangler/test-state` 로 비우고 `wrangler dev` 를 **새로** 띄울 것(돌아가는 wrangler 밑에서 비우면 모든 요청이 `internal` 로 죽는다) ③ `pkill -f "wrangler dev --port 8848"` 은 자기 셸을 죽인다(패턴을 쪼갤 것).
  - `process.execPath` 를 믿어 `node` 를 고정하는 방식은 **쓰지 말 것**: `playwright` 자체도 `#!/usr/bin/env node` 셰방으로 뜨므로 Playwright 가 이미 Node 20 으로 돌고 있을 수 있다 — 그때 `process.execPath` 는 똑같이 틀린 Node 를 가리킨다. 고칠 자리는 **PATH** 다.

- 차선 후보: `reuseExistingServer: true` 가 **자기가 씨를 뿌리지 않은 서버를 조용히 재사용한다** — `playwright.config.ts:28`. 앞 회차 포트 8848 의 `wrangler dev` 가 살아 있으면 `scripts/db-local.mjs` 가 돌아가는 DB 에 닿지 못한 채 Playwright 가 그 서버를 재사용해, 찌꺼기(만료 프로젝트 21건 등)를 안은 D1 위에서 보관·안내 시험이 비결정적으로 실패한다(`state/runs/2026-10-15-185856-…/journal.md:16,20` 에 이 위험이 기록돼 있다). 고칠 방향: 재사용 전에 **씨 뿌리기 표식을 확인**(예: `ops@irumfilm.test` 존재 여부나 `/api/me` 응답의 시험 표식)하고 어긋나면 재사용하지 않고 실패시킨다. `playwright.config.ts` + `scripts/db-local.mjs` 2파일. 1순위가 이미 머지돼 있거나 재현이 안 될 때 이것을 고를 것.
