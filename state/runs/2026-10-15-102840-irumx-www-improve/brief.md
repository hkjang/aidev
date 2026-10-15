- 과제: 수정 과제 — `npm test --silent` 가 깨끗한 체크아웃에서 혼자 통과하게 고침(테스트 준비물 자동 생성 + 정적 서버 기동 점검) (가치 5 / 위험 2 / 작업량 M)

- 왜: 베이스가 `main@94eb01f` 로 다시 고정됐고 이 커밋에는 앞 회차들이 넣었던 수정이 **하나도 없다**(직접 확인). `package.json` 의 scripts 는 `dev/fonts/images/build/preview/check/test/shots/deploy/test:build` 9개뿐이라 `pretest` 가 없고, `.gitignore` 가 `dist/`·`dist-draft/`·`dist-email/` 을 모두 무시하는데 `playwright.config.ts:21-31` 의 webServer 4개 중 셋이 그 폴더를 요구한다. 지금 작업 트리에는 `node_modules`·`dist`·`dist-draft`·`dist-email` 이 전부 없다(`ls` 로 확인). 그래서 러너가 `npm test --silent` 만 돌리면 코드 품질과 무관하게 webServer 기동 실패로 exit 1 이고, 이게 두 회차 연속 verify-failed 의 저장소 쪽 원인이다.

- 수용 기준:
  1. 깨끗한 체크아웃(= `npm ci` 후 `rm -rf dist dist-draft dist-email`)에서 **`npm test --silent` 단 한 줄만** 실행해 exit 0 이고 Playwright 가 전부 통과한다(기준값: 앞 회차 실측 77 통과·4 건너뜀). 사람이 `npm run test:build` 를 먼저 돌릴 필요가 없어야 한다.
  2. 테스트 파일·단정·기대값·timeout 을 하나도 바꾸지 않는다. `git diff main -- 'tests/*.spec.ts' playwright.config.ts` 가 **비어 있어야** 하고, 새로 추가한 `test.skip` 은 0건이어야 한다. (기존 건너뜀 4건은 `tests/site.spec.ts` 의 폭 조건부 스킵이며 이번 변경 대상이 아니다.)
  3. `node scripts/serve-static.mjs dist-nope 8899` 가 포트를 열지 않고 한국어 안내 한 줄 + exit 1 로 끝난다(지금은 조용히 listen 한 뒤 첫 요청에서 죽는다).
  4. `404.html` 이 없는 폴더를 제공할 때 없는 주소 요청이 404 응답을 돌려주고 **프로세스가 살아남아** 다음 요청도 처리한다. (지금 `scripts/serve-static.mjs:52-53` 은 `writeHead(404)` 를 먼저 쓰고 나서 `readFile(404.html)` 을 await 해, 파일이 없으면 처리되지 않은 rejection 으로 프로세스가 죽는다 — 이 줄을 직접 열어 확인했다.)
  5. `npm run build --silent` 가 여전히 exit 0 이고 `✓ 모두 통과` 를 찍는다.

- 건드릴 파일 (프로덕션 2 + 문서 1):
  - `package.json:10-21 scripts` — `"pretest": "npm run test:build && npx playwright install chromium"` 한 줄 추가. **dist 유무로 조건을 걸지 말 것**: 평범한 `npm run build` 가 만든 `dist` 에는 테스트용 Turnstile 키(`1x00000000000000000000BB`)가 없어서, 조건을 붙이면 낡은 `dist` 를 그대로 써 `tests/contact-send.spec.ts` 가 엉뚱한 이유로 빨개진다.
  - `scripts/serve-static.mjs` — ① `listen` 호출(:57) 전에 `base` 폴더와 `base/index.html` 존재를 확인해 없으면 한국어 한 줄(`제공할 빌드가 없습니다: … — 먼저 "npm run test:build" …`) + `process.exit(1)`; `404.html` 없음은 경고만 ② 404 분기(:51-54)를 `404.html` 을 **먼저 읽어 성공한 뒤에** `writeHead` 하도록 뒤집고, 읽기 실패 시 404 + 짧은 평문 ③ `createServer` 핸들러 본문 전체를 `try/catch` 로 감싸 남은 `readFile` 실패가 그 요청만 500 으로 끝나게.
  - `README.md` — '명령' 블록(28행 근처 `npm test` 줄)에 준비물을 알아서 만든다는 구절 한 마디. `npm run test:build` 줄은 남겨 둔다(따로 돌릴 수 있어야 한다).
  - 같이 해도 좋은 한 줄(선택): 같은 파일 `:42` 의 탈출 가드 `!full.startsWith(base)` → `full !== base && !full.startsWith(base + sep)`(`node:path` 의 `sep` import). 통하는 벡터는 `/..%2fdist-draft/x` 다 — `%2f` 는 WHATWG URL 정규화를 살아남고 `decodeURIComponent` 가 진짜 슬래시로 되돌려 형제 폴더로 샌다(`/../x` 는 파서가 먼저 지워서 통하지 않음). 범위가 커지면 빼도 된다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 이 순서로):
  0. **먼저 환경 진단**: 작업 트리에서 `npm exec -- node -v` 또는 `npm run env --silent | grep ^PATH`. 아래 "위험" 참조 — v20 이 나오면 저장소를 아무리 고쳐도 러너 검증은 통과하지 않는다.
  1. `npm ci`
  2. red 재현: `rm -rf dist dist-draft dist-email && npm test --silent` → exit 1 과 webServer 기동 실패 메시지를 **눈으로 보고 로그를 남긴 뒤** 고칠 것.
  3. 고친 뒤: `rm -rf dist dist-draft dist-email && npm test --silent` → exit 0.
  4. `npm run build --silent` → `✓ 모두 통과`.
  5. 기준 3: `node scripts/serve-static.mjs dist-nope 8899` → exit 1.
  6. 기준 4: 저장소 **밖**(`/tmp/…`)에 하네스를 두고 실제 프로세스를 spawn 해 404.html 없는 폴더에 2회 요청. 하네스는 커밋하지 말 것(소스 문자열 검사나 의존성 주입으로 때우지 말 것).
  7. `git diff main -- 'tests/*.spec.ts' playwright.config.ts` 가 비어 있는지 확인.

- 위험과 피할 것:
  - **러너 작업 트리의 node@20 오염 — 저장소 밖 원인, 이번 수정으로 해결되지 않는다.** 앞 회차가 측정한 바로는 `/home/hkjang/node_modules/.bin/node` 가 node 20.19.2 를 가리키고, npm 은 cwd 의 **모든 상위** `node_modules/.bin` 을 PATH 맨 앞에 붙이므로 `/home/hkjang/.cache/auto-improve-wt/irumx-www` 에서 돌리는 모든 npm 스크립트가 v20 으로 돈다(Astro 는 `>=22.12`, wrangler 는 `>=22` 를 요구해 즉시 exit 1). 이번 회차에 그 심링크가 **아직 존재한다는 것까지는 확인**했다(`ls` 가 `/home/hkjang/node_modules/node/bin/node` 로 해석해 샌드박스 경계 오류를 냈다). 다만 이 세션은 npm 실행 권한이 없어 **버전 자체는 재확인하지 못했다 — 미확인**. 그래서 검증 0단계를 먼저 돌려라. v20 이 나오면 ① 저장소 수정은 그대로 끝내고 ② 검증은 `/tmp` 로 클론해(`git clone <worktree> /tmp/irumx-green`) 거기서 돌려 증거를 남기고 ③ 원장에 "저장소 수정 완료 / 러너 환경 오염으로 작업 트리 검증 불가" 를 분명히 적어라. **저장소에 node 경로를 박아 넣거나 engines 를 낮추는 우회는 금지**(검증을 느슨하게 만드는 쪽이다).
  - 앞 회차 브랜치(`d72fe85`, `00b14d7`, `127343b`)의 **무조건 체리픽 금지**. 과거에 `scripts/free-ports.mjs` + 동적 포트까지 묶여 들어온 적이 있는데, 현 `playwright.config.ts` 는 이미 `reuseExistingServer:false` 와 8820~8823 이라 남의 서버 오인 위험이 없다. 이번 범위는 포트 건드리기가 **아니다** — 기준 2가 `playwright.config.ts` diff 를 비워 두라고 요구한다.
  - `worker/`, `src/lib/inquiry.ts`, `scripts/deploy.mjs`, `wrangler.jsonc`, `public/_headers` 는 운영 메일·배포·CSP 경로다. 이번 과제와 무관하니 열지 말 것.
  - `pretest` 는 이미지 411장을 세 번 최적화하는 `test:build` 를 끌고 들어와 **느리다**(앞 회차 실측: 캐시 있는 상태로 `npm test` 전체 1분59초, 테스트 자체는 약 28초). 느리다고 조건부로 건너뛰지 말 것 — 그 최적화는 별도 회차 과제다.
  - 빌드 산출물(`dist*`)을 커밋하지 말 것.

- 차선 후보: `harfbuzzjs`·`fontverter` 직접 의존성 선언 — `scripts/verify-build.mjs:18-19` 가 직접 import 하는데 `package.json` devDependencies(23-31행, 이번에 직접 확인)에 둘 다 없어 전이 의존성 호이스팅에 기대고 있다. 파일 1~2개, 검증은 `npm ci && npm run build`. (단, 1순위가 성립하지 않는 경우란 사실상 0단계에서 환경 오염이 드러난 때뿐인데, 그 경우엔 차선 후보도 같은 이유로 검증이 막힌다 — 그때는 1순위를 끝내고 환경 문제를 원장에 올리는 쪽이 낫다.)
