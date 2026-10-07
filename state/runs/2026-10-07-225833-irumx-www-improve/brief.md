# 과제서 (2026-10-07-225833, 수정 과제)

- 과제: 수정 과제 — `npm test --silent` 가 혼자서 제대로 돌게 고침(테스트 서버 포트 고정 제거 + 준비물 자동 생성) (가치 5 / 위험 2 / 작업량 M)

- 왜: 릴리즈 검증(`npm test --silent`)이 두 회차 연속 exit 1 로 끝났고, 앞 회차 로그(`state/runs/2026-10-07-214809-irumx-www-improve/verify.txt`)를 읽어 원인을 확정했다 — 실패 27건 중 `[desktop]`/`[mobile]` 분량은 Playwright 가 **우리 서버가 아닌 다른 프로젝트의 개발 서버**를 테스트한 것이다(`/contact` 응답 HTML 이 `<title>your · 회사 메일로 확인된 사람들의 소개팅</title>`, `/api/inquiry` 가 403/405 대신 404, `/no-such-page` 가 404 대신 200). `playwright.config.ts` 의 webServer 가 포트 8788·8789·8790·8791 을 고정으로 쓰면서 `reuseExistingServer: true` 라, 이 기계에서 8788(`your`)·8789(옆자리 인사이트)를 다른 프로젝트가 잡고 있으면 그 서버를 우리 서버로 알고 조용히 테스트한다. 더해 `package.json` 의 `test` 는 `playwright test` 뿐이라 테스트용 빌드 세 벌(dist·dist-draft·dist-email)을 만들지 않으므로, 러너처럼 깨끗한 작업 트리에서 단독 실행하면 `scripts/serve-static.mjs dist-draft` 가 없는 폴더를 제공하다 죽는다. 고치면 러너 검증이 환경에 좌우되지 않고 같은 명령으로 재현 가능해진다.

- 수용 기준:
  1) 다른 프로젝트가 8788·8789 를 잡고 있는 **지금 이 기계 상태 그대로**, `dist`·`dist-draft`·`dist-email` 을 모두 지운 뒤 `npm test --silent` 만 실행해 exit 0 (`npm ci` 는 미리 해도 된다). 로그의 통과 수를 과제 노트에 적을 것.
  2) 테스트를 지우거나 `test.skip`·느슨한 기대값을 넣지 않았다 — 검증 명령(`npm test --silent`)과 각 spec 의 기대값은 그대로다. `git diff main --stat` 로 바뀐 파일이 6개 안쪽임을 보일 것.
  3) `npm run build --silent` 가 그대로 `✓ 모두 통과` (exit 0). 즉 `playwright.config.ts` 변경이 Astro 빌드·`astro check`·`verify-build` 에 영향을 주지 않는다.
  4) 테스트가 남의 서버를 보면 **조용히 실패하지 않고 멈춘다**(`reuseExistingServer: false`), 그리고 두 spec 이 설정 밖에서 실행되면 즉시 알 수 있는 오류를 낸다.

- 건드릴 파일 (6개 — 이 범위를 넘기지 말 것):
  - `playwright.config.ts` — defineConfig 위에서 빈 포트 5개(mock-resend·wrangler·dist-draft·dist-email·wrangler inspector)를 한 번만 고르고 `IRUMX_TEST_PORTS` 환경변수에 담아 자식 프로세스에 물려준다(설정 파일이 프로세스마다 다시 읽히므로 매번 새로 고르면 서버와 테스트의 포트가 어긋난다). webServer 명령·url·각 project 의 `baseURL` 을 그 값으로 바꾸고 `reuseExistingServer: false`.
  - `scripts/free-ports.mjs` (신규, ~28줄) — `listen(0,'127.0.0.1')` 으로 n 개를 **동시에** 열어 포트를 읽고 닫아 공백 구분 한 줄로 출력. 동시에 열지 않으면 같은 포트를 두 번 받는다.
  - `package.json` — `"pretest": "npm run test:build && npx playwright install chromium"` 추가. `scripts.test` 는 `playwright test` 그대로.
  - `tests/worker-test.env` — 포트가 들어간 `RESEND_API_BASE`·`ALLOWED_ORIGINS` 는 테스트 때 덮어쓴다는 주석만. 기본값은 남긴다(이 파일만으로 `wrangler dev` 를 띄우는 용도).
  - `tests/contact-send.spec.ts` — 하드코딩된 `http://127.0.0.1:8790/__mails` 와 `Origin: 'http://127.0.0.1:8788'` 를 `process.env.MOCK_RESEND_BASE`·`WORKER_ORIGIN` 으로. 없으면 throw.
  - `tests/inbound.spec.ts` — 같은 방식으로 `MOCK` 만 환경변수화.
  - 포트가 들어가는 Worker 변수는 `.wrangler/test-ports.env`(gitignore 됨 — `.gitignore` 에 `.wrangler/` 확인)에 쓰고 `wrangler dev ... --env-file tests/worker-test.env --env-file .wrangler/test-ports.env` 로 나중 파일이 이기게 한다.
  - **지름길**: 이 변경은 앞 회차 브랜치에 그대로 있다 — `git cherry-pick 127343b`(= `auto/2026-10-07-2030`, 커밋 1개, 8 files: 위 6개 + `README.md` 명령 설명 + `docs/qa-report.md` 기록). 체리픽해도 **수용 기준 1·3 을 직접 돌려 확인할 것** — 그 브랜치는 머지되지 못했고 이 회차의 코드 기준으로 다시 통과를 봐야 한다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  [ -d node_modules ] || npm ci --no-audit --no-fund
  rm -rf dist dist-draft dist-email
  npm test --silent          # pretest 가 빌드 세 벌 + chromium 설치 → playwright test. 전체 6~10분
  npm run build --silent     # ✓ 모두 통과
  node scripts/free-ports.mjs 5   # 서로 다른 포트 5개가 한 줄로 나오는지
  ```
  (러너 검증은 `[ -d node_modules ] || npm ci --no-audit --no-fund` → `npm test --silent` 두 줄이다 — `verify.json` 확인)

- 위험과 피할 것:
  - **워크플로·테스트를 느슨하게 만들어 통과시키는 것 금지.** 기대값을 바꾸거나 스킵 조건을 넣지 말 것. 바뀌는 것은 "어디에 접속하는가" 와 "준비물을 누가 만드는가" 뿐이다.
  - `worker/index.ts`·`worker/svix.ts`·`src/lib/inquiry.ts`·`wrangler.jsonc`·`public/_headers` 는 건드리지 말 것(운영 문의 접수·메일 전달·CSP). 이번 과제는 테스트 배선만이다.
  - `pretest` 때문에 검증 1회 시간이 늘어난다(앞 회차 `npm test` 만 230초 → 빌드 세 벌이 더해져 6~10분). 러너의 단계 제한 시간은 **미확인**이다. `test:build` 를 더 빠르게 만들려고 빌드를 빼지 말 것 — 오래된 빌드를 보고 통과하는 문제가 다시 생긴다.
  - `wrangler dev` 에 `--env-file` 을 두 번 주면 나중 값이 이긴다는 것은 앞 회차 실행(wrangler 4.147, 75 통과)에서 경험적으로만 확인됐다 — 체리픽 후에도 `/api/inquiry` 관련 7건이 통과하는지로 다시 확인할 것(403/405/429 가 404 로 나오면 ALLOWED_ORIGINS 덮어쓰기가 안 먹은 것이다).
  - `--inspector-port` 도 빈 포트를 줘야 한다. 주지 않으면 wrangler 가 기본 9229 를 쓰고 다른 프로젝트의 wrangler 와 충돌한다.
  - 빈 포트를 고른 뒤 실제로 띄우기까지 미세한 틈이 있다(TOCTOU). 고정 포트보다 훨씬 낫지만 완전 무결은 아니다 — 재시도 로직을 새로 만들지 말고 그대로 둘 것.
  - 보호 경로 `.github/workflows/` 는 이번에 건드리지 않는다(main 에 아직 없다). `ci.yml` 재투입은 다음 회차 몫이다.

- 차선 후보: `harfbuzzjs`·`fontverter` 를 `package.json` devDependencies 에 선언(lock 의 `^1.6.2`·`^2.0.0`) + `scripts/verify-deps.mjs` 를 `npm run build` 첫 단계로 — `scripts/verify-build.mjs:18-19` 가 선언 없는 두 패키지를 import 해 `subset-font` 전이 의존성 호이스팅에 기대고 있다. 앞 회차 브랜치 `auto/2026-10-07-1758` 에 있다. 단, 1순위를 고치지 않으면 어차피 `npm test --silent` 에서 또 막히니 1순위를 먼저 끝낼 것.
