# 과제서 — 2026-10-07 (irumx-www)

- 과제: 테스트용 정적 서버(`scripts/serve-static.mjs`)가 요청 한 번에 죽지 않게 하고, 제공할 폴더가 준비되지 않았으면 기동 시점에 알려 주게 (가치 4 / 위험 1 / 작업량 S)

- 왜: `scripts/serve-static.mjs:53` 의 `readFile(join(base, '404.html'))` 이 try/catch 없이 async 요청 핸들러 안에 있어, `404.html` 이 없는 폴더를 제공 중일 때 없는 주소로 요청이 한 번만 들어와도 unhandled rejection 으로 프로세스가 내려간다(Node 기본 `--unhandled-rejections=throw`). 게다가 폴더가 아예 없거나 `index.html` 이 없어도 `listen` 은 성공하므로, Playwright `webServer` 는 60초를 기다린 뒤 `Timed out waiting 60000ms from config.webServer` 만 남기고 죽는다 — 네 개 서버 중 어느 것이 왜 문제인지 알 수 없다. 이 정확한 증상이 지난 회차에서 실제로 사람 시간을 잡아먹었다(개선 기록 2026-10-07 「npm test 가 혼자 돌게」 항목의 실패 재현: `Error: ENOENT: no such file or directory, open '.../dist-email/404.html'`).

- 수용 기준:
  1) **먼저 재현해서 증명할 것.** `404.html` 이 없고 `index.html` 만 있는 임시 폴더를 만들어 서버를 띄우고, 없는 주소로 요청을 한 번 보낸 뒤 **두 번째 요청이 실패하는 것**(프로세스가 죽었음)을 확인한다. 고친 뒤 같은 절차에서 첫 요청이 404 응답을 돌려주고 **두 번째 요청도 정상 응답**하는 것을 확인한다. 재현 로그를 결과 보고에 그대로 붙인다.
  2) `404.html` 이 없을 때 요청은 404 상태 + 짧은 평문 본문으로 응답하고 서버는 살아 있는다. `404.html` 이 있을 때의 동작(404 상태 + 그 파일 내용 + `text/html; charset=utf-8`)은 **지금과 똑같아야 한다** — `tests/site.spec.ts` 의 「없는 주소는 404 상태와 안내 페이지」 기대값을 바꾸지 말 것.
  3) `listen` 하기 전에 `base` 폴더와 `base/index.html` 존재를 확인하고, 없으면 한국어 한 줄로 무엇이 없고 무엇을 하면 되는지 알려 준 뒤 **exit 1** 로 끝낸다. 예: `serve-static: /…/dist-draft 에 index.html 이 없다 — npm run test:build 를 먼저 돌릴 것`. (`404.html` 없음은 치명이 아니므로 기동은 시키되 경고 한 줄만 찍는다 — 수용 기준 2 가 이미 덮는다.)
  4) 정상 경로 회귀 없음: `npm test` 의 기존 기대값·스펙 파일을 하나도 고치지 않고 통과한다.
  5) (같은 함수 안의 1줄, 같이 고칠 것) `scripts/serve-static.mjs:42` 의 탈출 방지 가드 `!full.startsWith(base)` 는 형제 폴더를 막지 못한다 — `base=/a/dist` 일 때 `/../dist-draft/x` 가 `/a/dist-draft/x` 로 정규화되고 `startsWith('/a/dist')` 가 **참**이라 통과한다. `base + sep` 로 비교하도록 고치고(`full !== base && !full.startsWith(base + sep)`), 정상 경로(`/`, `/about`, `/_astro/…`)가 그대로 200 인 것을 확인한다.

- 건드릴 파일 (프로덕션 코드 0개, 테스트 지원 스크립트 1개):
  - `scripts/serve-static.mjs` — ① 파일 상단 `listen` 전 기동 점검 추가(기준 3) ② 요청 핸들러의 404 분기(51~54행)를 `404.html` 읽기 실패에 견디게(기준 2) ③ 42행 경로 가드 `sep` 보강(기준 5). `node:path` 에서 `sep`, `node:fs` 에서 `existsSync` 를 추가로 import 하게 된다.
  - 그 외 파일은 건드리지 말 것. 특히 `package.json`·`playwright.config.ts` 는 **손대지 말 것** — 아래 "위험" 참고.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) 재현·수정 확인(의존성 불필요, 순수 node):
     ```
     mkdir -p /tmp/ss-ok /tmp/ss-no404
     printf '<h1>hi</h1>' > /tmp/ss-ok/index.html
     printf '<h1>404</h1>' > /tmp/ss-ok/404.html
     printf '<h1>hi</h1>' > /tmp/ss-no404/index.html      # 404.html 없음
     # (a) 404.html 없는 폴더: 없는 주소 요청 뒤에도 서버가 살아 있어야 한다
     node scripts/serve-static.mjs /tmp/ss-no404 8799 &
     curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8799/nope   # 404 기대
     curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8799/       # 200 기대 (고치기 전에는 연결 실패)
     # (b) 404.html 있는 폴더: 기존 동작 그대로
     node scripts/serve-static.mjs /tmp/ss-ok 8798 &
     curl -s -i http://127.0.0.1:8798/nope | head -3                        # 404 + text/html, 본문에 '404'
     # (c) 준비물 없음: 기동 시점에 한 줄로 알려 주고 exit 1
     node scripts/serve-static.mjs /tmp/does-not-exist 8797; echo "exit=$?"  # exit=1 기대
     ```
  2) 전체 회귀: `npm run build --silent` (exit 0, 끝에 `✓ 모두 통과`) → `npm run test:build` → `npm test --silent`.
     **주의**: 이 브랜치의 main 에는 아직 `pretest`·`scripts/free-ports.mjs` 가 **없다**(이번 정찰에서 `package.json`·`ls scripts/` 로 직접 확인). 그래서 `npm test` 는 ① `npm run test:build` 를 먼저 돌려 `dist`·`dist-draft`·`dist-email` 를 만들어야 하고 ② 고정 포트 8788·8789 를 이 기계의 다른 프로젝트 개발 서버가 잡고 있으면 엉뚱하게 실패한다(개선 기록에 증거 있음: `/contact` 응답이 `<h1>your</h1>`). 포트가 막혀 있으면 그 사실과 `lsof -i :8788 -i :8789` 출력을 결과에 적고, 위 1)번 순수 node 검증을 근거로 삼을 것 — **테스트나 기대값을 느슨하게 고쳐 통과시키지 말 것.**

- 위험과 피할 것:
  - **`package.json`·`playwright.config.ts` 를 건드리지 말 것.** 다른 브랜치 `auto/2026-10-07-2258`(origin 에 푸시됨, review-pending)이 바로 그 두 파일에 `pretest` + `scripts/free-ports.mjs` 를 넣고 있다. 같은 파일을 이번 회차에서 또 고치면 머지 충돌이 난다. 이번 과제는 `scripts/serve-static.mjs` 한 파일로 끝나므로 어느 쪽이 먼저 들어가도 충돌하지 않는다.
  - `harfbuzzjs`·`fontverter` 선언 + `verify-deps.mjs` 는 이번 과제가 **아니다**. 이미 한 회차가 그대로 했고 결과가 verify-failed 였다(막힌 원인이 `npm test` 환경이고, 그 수정이 아직 main 에 없다). 같은 것을 또 올리지 말 것.
  - `.github/`(CI 워크플로)는 보호 경로다 — 이번 과제에서 만들지 말 것.
  - `worker/index.ts`·`worker/svix.ts`·`src/lib/inquiry.ts` 는 운영 중인 문의 접수·메일 전달 경로다. 이번 과제와 무관하니 열지 말 것.
  - 404 폴백 본문을 바꿀 때 `tests/site.spec.ts` 의 404 단정을 깨지 않도록 주의 — **`404.html` 이 있을 때의 응답은 바이트 단위로 지금과 같아야 한다.** 폴백은 `404.html` 이 없을 때만 쓰인다.
  - 기동 점검을 과하게 만들지 말 것(예: 모든 자산 검사, dist 내용 검증). 폴더·`index.html` 두 가지면 충분하다. 범위가 늘면 쪼갤 것.

- 차선 후보: **`scripts/verify-build.mjs` 의 서비스 경로 목록 드리프트 감지** — `scripts/verify-build.mjs:210` 에 11개 서비스 경로가 손으로 박혀 있고(`/services/yeopjari` … `/services/irum-eval`), 같은 목록이 `tests/site.spec.ts:3` `PAGES` 에도 따로 박혀 있다. 이번 정찰에서 `src/content/services.ts` 의 slug 11개와 두 목록이 **지금은 정확히 일치**함을 확인했다(드리프트 없음 — 그래서 1순위가 아니다). 두 목록을 **합치지 말고**(한쪽은 dist 파일 계약, 한쪽은 HTTP 응답 계약 — 운영자 지침: 여러 경로의 같은 파서는 통합하지 말고 좁히는 방향으로만), `dist/services/*.html` 를 읽어 사이트맵에 없는 것이 있으면 실패시키는 **감지만** 더한다. 건드릴 파일 1개(`scripts/verify-build.mjs`), 검증은 `npm run build`.

## 추정 근거 (basis of estimate)
- 분해: 기동 점검 ~15분 / 404 폴백 예외 처리 ~10분 / 경로 가드 1줄 ~5분 / 재현·검증 ~15분 = **45분 (S)**.
- 범위에 포함: 위 네 가지와 순수 node 재현. 범위 밖(명시적 제외): `npm test` 가 혼자 돌게 만드는 일, CI, 의존성 선언, Worker 코드.
- 가정: 구현자 기계에 `node_modules` 가 설치되어 있다. **이 정찰 세션의 워크트리에는 `node_modules` 가 없었고 `npm ci` 실행 권한이 없어 `npm run build`·`npm test` 를 직접 돌려 보지 못했다(미확인).** 그래서 1차 검증을 의존성이 필요 없는 순수 node 절차로 잡았다.
- 신뢰 구간: 수용 기준 1~3·5 는 코드만 보고 결정 가능하므로 변동이 작다. 변동은 거의 전부 2)번 전체 회귀에서 나온다(포트 점유 여부) — 막히면 +0분(보고로 대체), 안 막히면 +5~10분.

## 이번 정찰에서 실제로 확인한 것 / 못 한 것
- 확인(직접 열람): `scripts/serve-static.mjs` 전체 57줄, `scripts/verify-build.mjs:200-215`, `tests/site.spec.ts:1-12`, `package.json` 전체, `src/content/services.ts` 의 slug 목록, `ls scripts/`, `git log --oneline -12`, `git branch -a`.
- 확인: main(80e35fc)에 `.github/` **없음**, `scripts/free-ports.mjs`·`scripts/verify-deps.mjs` **없음**, `package.json` 에 `pretest` **없음**, `harfbuzzjs`·`fontverter` 선언 **없음**. 즉 앞선 네 회차의 성공 결과 중 어느 것도 아직 main 에 들어오지 않았다.
- **미확인**: 404 크래시의 런타임 재현(이 세션에서 `node`/`curl` 실행 권한이 거부됐다). 근거는 코드 독해뿐이다 — 53행 `await readFile(...)` 이 async 핸들러 안에서 try/catch 없이 rejection 을 낸다. 그래서 수용 기준 1을 "먼저 재현하라" 로 못 박았다. **재현되지 않으면 과제를 중단하고 보고할 것.**
- **미확인**: `npm run build`·`npm test` 가 이 기계에서 지금 통과하는지(`node_modules` 없음, `npm ci` 권한 없음).
