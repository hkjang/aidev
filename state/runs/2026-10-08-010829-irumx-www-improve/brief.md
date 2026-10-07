- 과제: 수정 과제 — `npm test --silent` 가 혼자서 통과하게 고침(테스트 준비물 자동 생성 + 테스트 포트 고정 제거) (가치 5 / 위험 2 / 작업량 M)

- 왜: 릴리즈 검증 명령이 `npm test --silent` 인데 main 에는 이 명령이 혼자서 돌 수 있는 장치가 하나도 없다 — `npm test` 는 `playwright test` 뿐이라 Playwright webServer 가 제공할 `dist-draft`·`dist-email` 을 아무도 만들지 않고(기동 실패), 설령 빌드가 있어도 `playwright.config.ts` 가 포트 8788·8789 를 하드코딩한 채 `reuseExistingServer: true` 라서 **이 기계에서 그 포트를 잡고 있는 다른 프로젝트의 개발 서버**를 우리 서버로 알고 테스트한다. 고치면 러너 검증이 통과하고 연속 2회 verify-failed 가 끝난다.
- 이번 회차에 확인한 사실(추측 아님):
  - `package.json:10-21` — 스크립트는 `test: "playwright test"` 뿐, `pretest` **없음**. `test:build` 는 따로 사람이 돌려야 한다.
  - `scripts/` 에 `free-ports.mjs` **없음**(Glob 으로 확인, 9개 파일뿐).
  - `playwright.config.ts:20-28` — webServer 4개가 각각 8790·8788·8789·8791 하드코딩, 네 개 모두 `reuseExistingServer: true`.
  - `ss -ltnp` 실측: `:8788` = pid 2806847, `:8789` = pid 2814318 — **지금도 다른 프로세스가 점유 중**.
  - 이미 만들어진 해법이 `origin/auto/2026-10-07-2258` 의 커밋 `00b14d7` 에 있다(머지 안 됨, review-pending). 변경 범위: `package.json`(+1), `playwright.config.ts`(+60/-?), `scripts/free-ports.mjs`(신규 28줄), `tests/contact-send.spec.ts`, `tests/inbound.spec.ts`, `tests/worker-test.env`, `README.md`, `docs/qa-report.md` — 8 파일.

- 수용 기준:
  1) `dist`·`dist-draft`·`dist-email` 을 **모두 지운 상태**에서 `npm test --silent` 만 실행해 exit 0. (사람이 `npm run test:build` 를 먼저 돌리지 않아도 된다)
  2) `:8788`·`:8789` 를 다른 프로세스가 점유한 채로도 위 1)이 통과한다. 즉 Playwright 가 남의 서버에 붙지 않는다 — `reuseExistingServer` 가 false 이고 포트를 런타임에 확보한다.
  3) `npm run build --silent` 가 여전히 `✓ 모두 통과` 로 끝난다(exit 0).
  4) **각 spec 의 단정과 기대값을 바꾸지 않았다.** `git diff main --stat` 에서 `tests/*.spec.ts` 의 변경은 "접속 주소를 환경변수로 받는다" 에 한정되고, 지운 테스트·추가한 `test.skip` 이 없다. (검증을 느슨하게 해 통과시키는 것은 금지)

- 건드릴 파일 (6개 + 문서 2개 — 그 이상 늘리지 말 것):
  - `package.json` — `scripts.pretest` 추가: `npm run test:build` + `npx playwright install chromium`. `test` 와 `test:build` 자체의 내용은 바꾸지 말 것(러너 검증 명령이 `npm test`).
  - `scripts/free-ports.mjs` (신규) — 서로 다른 빈 TCP 포트 N개를 찾아 출력. `net.createServer().listen(0)` 로 OS 에 받고 즉시 닫는 방식 권장. 받은 포트를 `IRUMX_TEST_PORTS` 로 자식에게 물려준다.
  - `playwright.config.ts` — 하드코딩 포트 4개(8790·8788·8789·8791)를 확보한 포트로 바꾸고 `reuseExistingServer: false`. 각 `projects[].use.baseURL` 도 같은 값을 쓰게. wrangler 에 들어가는 포트 의존 변수(`RESEND_API_BASE`·`ALLOWED_ORIGINS`)는 `tests/worker-test.env` 를 직접 고치지 말고 **gitignore 된 추가 env 파일**(`.wrangler/test-ports.env`)로 덮어쓸 것 — `tests/worker-test.env` 는 커밋되는 파일이고 포트가 매번 달라진다.
  - `tests/contact-send.spec.ts`, `tests/inbound.spec.ts` — 가짜 Resend 주소를 `MOCK_RESEND_BASE`, Worker 주소를 `WORKER_ORIGIN` 환경변수로 받게. **환경변수가 없으면 조용히 기본값으로 넘어가지 말고 throw** 할 것(없는데 통과하면 잘못된 서버를 보고 있다는 뜻).
  - `tests/worker-test.env` — 포트 치환 지점이 보이게 최소 수정(위 설명대로 실제 포트는 `.wrangler/test-ports.env` 가 덮는다).
  - 문서: `README.md` '명령' 절에 `npm test` 가 이제 준비물을 스스로 만든다고 한 줄, `docs/qa-report.md` 에 짧은 기록.
  - **가장 빠른 길**: 위 변경이 이미 `origin/auto/2026-10-07-2258` 의 `00b14d7` 에 있다. `git cherry-pick 00b14d7` 로 가져와 **읽고 검증한 뒤** 쓰거나, 충돌하면 수동으로 같은 내용을 쓸 것. 체리픽했다고 끝내지 말고 아래 검증을 직접 돌려 증거를 남길 것(이전 회차가 "체리픽했다" 까지만 하고 verify-failed 로 끝난 전례가 있다).

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  1. 고치기 전 실패 재현: `rm -rf dist dist-draft dist-email && npm test --silent; echo "exit=$?"` → `ENOENT … dist-email/404.html` / `Process from config.webServer was not able to start` (exit 1)
  2. `node scripts/free-ports.mjs 5` → 서로 다른 포트 5개
  3. `rm -rf dist dist-draft dist-email && npm test --silent; echo "exit=$?"` → **exit 0**
  4. `npm run build --silent; echo "exit=$?"` → `✓ 모두 통과`, exit 0
  5. `git diff main --stat` 로 바꾼 파일이 8개 이하인지, spec 의 단정이 안 바뀌었는지 확인
  - **이 정찰 세션은 위 명령을 실제로 돌리지 못했다(미확인).** 작업 트리에 `node_modules` 가 없고(`ls`: No such file or directory), `npm ci` 가 비대화형 세션에서 승인 거부로 막혔다. 따라서 "고치기 전 exit 1" 은 이번 회차에 **재현하지 않았고**, 근거는 아래 정적 사실이다 — 그 정적 사실들은 직접 열어서 확인했다(`package.json` 에 `pretest` 없음, `scripts/free-ports.mjs` 없음, `playwright.config.ts:20-28` 의 하드코딩 포트 + `reuseExistingServer: true`, `ss -ltnp` 로 `:8788`·`:8789` 점유 중). **구현자는 1번 재현부터 직접 돌려 실패를 눈으로 본 뒤 고칠 것.**
  - 실측 참고(이전 회차 기록, 이번 회차 미재측): `npm ci` 약 6초, `npm run build` 약 60초, `npm test` 약 37~56초. 3번은 `pretest` 가 빌드 세 벌을 만들므로 **3~5분** 걸린다. `npx playwright install chromium` 이 처음이면 114MiB 다운로드가 추가된다.
  - `dist`·`dist-draft`·`dist-email` 은 지금 작업 트리에 **애초에 없다**(확인함) — 1번의 `rm -rf` 는 사실상 무해하다.

- 위험과 피할 것:
  - **검증을 느슨하게 만들어 통과시키는 것 금지.** 테스트 삭제·`test.skip` 추가·`timeout` 늘리기·기대값 완화로 해결하지 말 것. 바꿀 것은 "어디에 접속하는가" 와 "준비물을 누가 만드는가" 뿐이다.
  - 외부 연결이 없는 환경에서는 `tests/contact-send.spec.ts:75` 와 `tests/site.spec.ts:7`(`/contact`, desktop·mobile) 3건이 `net::ERR_INTERNET_DISCONNECTED`(challenges.cloudflare.com, Turnstile 위젯)로 **선재 실패**한다. 내 변경 탓이 아님을 증명하려면 변경을 되돌린 상태에서 같은 3건이 똑같이 실패하는지 확인할 것. 이 3건을 스킵 조건으로 막지 말 것.
  - `worker/index.ts`·`worker/svix.ts`·`src/lib/inquiry.ts` 는 운영 중인 문의 접수 경로다 — 이번 과제에서 **건드리지 말 것**.
  - `.github/workflows/` 는 보호 경로다. 이번 회차에 추가하지 말 것(main 에 아직 없지만, `npm test` 가 혼자 돌게 된 다음 회차 일이다).
  - `harfbuzzjs`·`fontverter` 선언 + `verify-deps.mjs` 는 이번 회차 범위가 **아니다**(별 과제). 같이 넣으면 파일 수가 늘고 원인 귀속이 흐려진다.
  - 포트를 확보한 순간과 서버가 listen 하는 순간 사이에 TOCTOU 틈이 남는다 — 이번 회차에 해결하지 말고 알려진 한계로 둘 것.
  - 바꾼 파일 수를 8개 안쪽으로 유지할 것. 범위가 넘치면 문서 2개를 먼저 덜어낼 것.

- 차선 후보: 서비스 페이지 목록 드리프트 감지 — `scripts/verify-build.mjs` 의 손으로 적힌 서비스 경로 목록과 실제 `dist/services/*.html` 를 대조해, 빌드된 서비스 페이지가 목록에 없으면 실패시킨다. `tests/site.spec.ts` 의 `PAGES` 와 **합치지 말 것**(한쪽은 dist 파일 계약, 한쪽은 HTTP 응답 계약). 파일 1개, 검증 `npm run build`.
