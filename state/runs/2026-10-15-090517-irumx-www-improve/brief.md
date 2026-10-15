# 과제서 (2026-10-15 정찰) — irumx-www

- 과제: 수정 과제 — `npm test --silent` 가 깨끗한 체크아웃에서 혼자 통과하게 고침(테스트 준비물 자동 생성 + 정적 서버 기동 점검) (가치 5 / 위험 2 / 작업량 M)

- 왜: 현 HEAD(94eb01f)의 `package.json` 에는 `pretest` 가 없고 `"test": "playwright test"` 뿐인데, `playwright.config.ts` 의 webServer 4개 중 2개가 `node scripts/serve-static.mjs dist-draft 8821` · `dist-email 8823` 이고 Worker 서버는 `dist` 를 쓴다. `dist`·`dist-draft`·`dist-email` 은 모두 `.gitignore` 에 있어 깨끗한 체크아웃(= 러너 작업 트리. 이번 정찰에서 `ls -d dist dist-draft dist-email` 전부 "No such file or directory")에는 존재하지 않는다. 그래서 러너 검증 `npm test --silent` 는 코드 품질과 무관하게 구조적으로 exit 1 로 끝난다(두 회차 연속 같은 이유). 고치면 러너·사람·CI 가 같은 한 줄로 테스트를 돌릴 수 있고, 실패했을 때 원인이 로그에 한국어 한 줄로 드러난다.

- 수용 기준:
  1) `rm -rf dist dist-draft dist-email && npm test --silent` 가 **단독으로 exit 0**(현재는 webServer 기동 실패로 exit 1). 통과 건수는 직전 전체 실행 기록인 77 통과·4 건너뜀과 같은 수준이어야 하고, **건너뜀이 4건을 넘으면 안 된다**(스킵으로 덮은 것이 아님을 증명).
  2) `tests/*.spec.ts` 의 단정·기대값·timeout·`test.skip` 이 하나도 바뀌지 않았다(`git diff main -- 'tests/*.spec.ts'` 가 비어 있거나, 바뀐 줄이 접속 주소·주석뿐). `playwright.config.ts` 의 `reuseExistingServer: false` 를 되돌리지 않는다.
  3) `node scripts/serve-static.mjs dist-nope 8899` 가 **listen 하지 않고** 한국어 한 줄 + exit 1 로 끝난다(지금은 listen 에 성공한 뒤 첫 요청에서 `404.html` 무방비 `readFile` 실패로 프로세스가 죽어, Playwright 가 `Process from config.webServer was not able to start. Exit code: 1` 만 남긴다 — serve-static.mjs:51-53).
  4) `404.html` 이 없는 폴더를 제공할 때, 없는 주소 요청이 404 로 답하고 **프로세스가 살아 있어** 다음 요청도 처리한다(실제 프로세스를 띄워 소켓으로 2회 요청하는 하네스로 증명. 의존성 주입·소스 문자열 검사는 근거로 쓰지 말 것).
  5) `npm run build --silent` 가 `✓ 모두 통과` 로 exit 0 (verify-build 회귀 없음).

- 건드릴 파일 (프로덕션 2개 + 문서 1개, 그 이상 늘리지 말 것):
  - `package.json:scripts` — `"pretest": "npm run test:build && npx playwright install chromium"` 한 줄 추가. `test:build` 는 이미 `PUBLIC_TURNSTILE_SITE_KEY=1x00000000000000000000BB` 로 dist 를 만들고 dist-draft·dist-email 까지 만든다. **`test` 스크립트 본문은 바꾸지 말 것.**
  - `scripts/serve-static.mjs` — ① `listen` 전에 `base` 폴더와 `base/index.html` 존재를 확인해 없으면 한국어 한 줄 + `process.exit(1)`, `404.html` 없음은 경고만 ② 404 분기(51-53행)를 `404.html` 을 먼저 읽고 성공하면 헤더+본문, 실패하면 404 + 짧은 평문으로 바꿀 것 ③ 핸들러 전체를 try/catch 로 감싸 남은 `readFile` 실패가 그 요청만 500 으로 끝나게 할 것. (같은 변경이 과거 회차에서 채택됐으나 main 에 없다 — 이번에 다시 넣는다.)
  - `README.md:27-28` ('명령' 절. 이번 정찰에서 직접 확인) — 지금은 `npm run test:build`(27행)와 `npm test`(28행)가 나란히 적혀 사람이 앞의 것을 먼저 돌려야 하는 것처럼 읽힌다. 28행 주석에 "준비물(빌드 세 벌·Chromium)은 알아서 만든다" 를 한 구절 더할 것. 27행은 단독으로도 쓸 수 있으니 지우지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것. 순서대로):
  1. `npm ci`
  2. `rm -rf dist dist-draft dist-email`
  3. `npm test --silent` → exit 0 (**이번 회차의 러너 검증과 같은 명령**)
  4. `npm run build --silent` → `✓ 모두 통과`
  5. `node scripts/serve-static.mjs dist-nope 8899` → exit 1 + 한국어 안내
  6. (기준 4) 저장소 **밖**(예: `/tmp/ss-check/`)에 node 하네스를 두고 실제 프로세스를 spawn 해 2회 요청 — 하네스를 저장소에 커밋하지 말 것

- 위험과 피할 것:
  - **`pretest` 를 "dist 가 없을 때만 빌드" 로 조건화하지 말 것.** 평범한 `npm run build` 로 만든 `dist` 는 테스트용 Turnstile 키가 없어, 조건을 붙이면 낡은 dist 를 그대로 써 `contact-send.spec.ts` 가 엉뚱하게 실패한다. 매번 `test:build` 를 돌리는 쪽이 맞다.
  - **옛 커밋(127343b·00b14d7·55ea83b)을 통째로 체리픽하지 말 것.** 그 커밋들의 큰 부분(`scripts/free-ports.mjs` + `playwright.config.ts` 동적 포트 + `tests/worker-test.env` + 두 spec 의 환경변수화)은 **고정 포트 8788·8789 를 남의 서버가 잡고 `reuseExistingServer: true` 였던 시절** 처방이다. 현 main 은 포트가 8820~8823 이고 `reuseExistingServer: false` 라 그 위험이 이미 없다(이번 정찰 `ss -ltn`: 8820~8823·8788·8789 전부 비어 있음). 필요한 것은 그 커밋들의 package.json 한 줄뿐이다. 동적 포트는 별도 과제로 둔다.
  - `npm run preview` 가 8820 을 쓰므로, 검증 중 preview/dev 서버를 동시에 띄우지 말 것(포트 점유 시 즉시 실패).
  - `pretest` 는 빌드를 세 벌 만들어 `npm test` 를 느리게 한다(과거 회차 기록: 이미지 다량 처리로 수 분. 캐시가 차 있으면 수십 초). 이것을 이유로 빌드 단계를 빼거나 spec 을 건너뛰게 만들지 말 것.
  - 보호 경로 회피: `worker/`, `src/lib/inquiry.ts`, `src/`, `public/_headers`, `scripts/deploy.mjs` 는 건드리지 않는다. `.github/` 도 이번 과제에서 만들지 않는다(현 HEAD 에 없음 — 별도 과제).
  - 커밋 메시지·주석은 한국어.

- 미확인(추측으로 적은 것): 이번 정찰 세션에서는 `npm ci` 실행 권한이 없어(샌드박스 거부) **red 재현을 직접 돌리지 못했다**. 근거는 파일 사실뿐이다 — `package.json`(pretest 없음), `playwright.config.ts`(dist-draft·dist-email 서버), `.gitignore`(dist* 무시), `ls`(세 폴더 전부 없음), `serve-static.mjs:51-53`. 구현자는 **먼저 2~3번 명령으로 red 를 눈으로 확인한 뒤** 고치기 시작할 것. 통과 건수 77/4 는 2026-10-15 회차 기록이고 이번 실측이 아니다. 외부 Turnstile 연결이 끊긴 환경에서는 3건이 `net::ERR_INTERNET_DISCONNECTED` 로 실패할 수 있다(과거 기록) — 그때는 변경 전 원본으로도 같은 3건이 실패하는지 확인해 선재 실패임을 증명할 것.

- 차선 후보: `scripts/verify-build.mjs:18-19` 가 `harfbuzzjs`(`Blob as HbBlob, Face as HbFace`)·`fontverter`(`convert`)를 직접 import 하는데 `package.json` devDependencies 에 둘 다 없다(이번 정찰에서 두 파일 모두 직접 확인 — 전이 의존성 호이스팅에 의존). 이것을 명시 선언으로 올리는 과제. 파일 1~2개, 검증은 `npm ci` → `npm run build`.
