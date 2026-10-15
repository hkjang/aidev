# 과제서 — 2026-10-15-110829 review.yeopjari.bid

- 과제: CI 게이트를 로컬에서 한 줄로 재현하는 `scripts/ci-local.sh` 를 추가하고, `check.yml` 과 어긋난 README 「검증」 절을 그에 맞추기 (가치 4 / 위험 1 / 작업량 S)

- 왜: `.github/workflows/check.yml:1-3` 머리말은 이 워크플로가 "the same gate as a laptop (README 「검증」)" 라고 선언하는데, 실제로 두 목록은 다르다 — README 「검증」(README.md:94-103)에는 CI 가 돌리는 `node scripts/ci-env.mjs`·`scripts/migrate.mjs`(고정 `DATABASE_URL`/비밀번호 3종)·서버 기동·a11y 직전의 `node scripts/seed-demo.mjs` 가 없고, 반대로 CI 에는 `ui-flows.mjs`·`screenshots.mjs` 가 없다. 그래서 "CI 와 같은 검증을 로컬에서" 돌리려면 매번 사람이 워크플로 YAML 9단계를 손으로 옮겨 적어야 하고, 실제로 2026-10-14 회차가 그 옮겨 적기에 한 세션을 통째로 썼다. 한 줄짜리 재현 스크립트를 두면 이후 회차가 CI 응답을 기다리지 않고도 같은 게이트를 통과했음을 증명할 수 있다.

- 수용 기준:
  1) `bash scripts/ci-local.sh` 가 `.github/workflows/check.yml` 의 `steps` 를 **같은 순서·같은 환경변수**로 실행한다: `npm ci --no-audit --fund=false` → `npm run check` → `npm run build` → `node scripts/perf.mjs` → `node scripts/ci-env.mjs` → `DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55443/review REVIEW_APP_DB_PASSWORD=devapppass REVIEW_GATEWAY_DB_PASSWORD=devgatewaypass node scripts/migrate.mjs` → 서버 기동(`.env.dev` 적용 후 `apps/server/dist/index.js`, `/api/v1/health` 를 최대 60초 폴링) → `node scripts/e2e.mjs` → `node scripts/seed-demo.mjs` + `node scripts/a11y.mjs`.
  2) 어느 단계든 실패하면 `set -e` 로 즉시 0 이 아닌 코드로 끝나고, 각 단계 시작 시 단계 이름을 한 줄 찍어 어디서 깨졌는지 바로 보인다. 실패 시 `tail -n 300 server.log`(check.yml:63-65 의 `if: failure()` 와 같은 것)를 찍고, 스크립트가 띄운 서버는 종료한다(`trap`).
  3) PostgreSQL 이 `127.0.0.1:55443` 에 없으면 즉시 멈추고 README.md:85 의 `docker run …postgres:16-alpine` 한 줄을 안내한다 — 묵묵히 실패하지 않는다.
  4) README 「검증」 절(README.md:94-103) 맨 앞에 `bash scripts/ci-local.sh  # CI 와 똑같은 게이트를 그대로` 를 넣고, 기존 개별 명령 목록은 "그 밖에 사람이 보는 검사" 로 남긴다(`ui-flows.mjs`·`screenshots.mjs` 는 CI 에 없다는 점을 한 줄로 밝힐 것).
  5) a11y 단계 앞의 브라우저 설치는 **`npx playwright install chromium`**(`--with-deps` 없이)으로 쓴다 — 로컬에서 `--with-deps` 는 `sudo: a password is required` 로 죽는다(2026-10-14 관측). `check.yml` 의 `--with-deps` 는 그대로 둔다.

- 건드릴 파일 (프로덕션 0개 / 전부 도구·문서):
  - `scripts/ci-local.sh` — 신규. 위 9단계 순차 실행 + 단계 로그 + `trap` 서버 정리 + DB 선행 확인. 기존 `scripts/dev-reset.sh` 의 셸 관례(헤더 주석·`set -euo pipefail`)를 그대로 따를 것.
  - `README.md:94-103` — 「검증」 절 맨 앞 줄 추가와 주변 한두 줄 손질.
  - (선택) `.github/workflows/check.yml:1-3` 머리말을 "the same gate as `scripts/ci-local.sh`" 로 한 단어만 고치는 것은 허용. **그 외 워크플로 수정 금지** — 단계 삭제·`continue-on-error`·타임아웃 완화 등 느슨하게 만드는 변경은 절대 금지다.

- 검증 명령:
  ```
  docker run -d --name yeopjari-review-dev -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=review -p 55443:5432 postgres:16-alpine
  bash scripts/ci-local.sh
  ```
  기대: 마지막 줄까지 가서 `e2e` 가 `218 passed / 0 failed`(main@6ea982c 기준선), a11y 가 `심각한 접근성 위반 없음`, `perf.mjs` 가 `223.7 KB initial JS (gzip), budget 230 KB` 근처. **이 수치들은 2026-10-14 회차 관측값을 옮긴 것이고 이번 정찰에서 재실행하지 못했다(미확인 — 이 샌드박스는 `npm`/`npx` 실행이 승인 거부된다). 구현자가 실제로 돌려 확정할 것.**
  더해서 깨짐을 증명할 것: 아무 단계 하나를 일부러 실패시켜(예: `packages/core/src/util/text.ts` 에 구문 오류를 잠깐 넣어 `npm run check` 를 깨뜨림) 스크립트가 **그 단계 이름을 찍고 0 이 아닌 코드로 멈추는지** 확인한 뒤 되돌릴 것. 통과만 보고 끝내지 말 것.

- 위험과 피할 것:
  - **이번 회차의 지시는 "워크플로 실패 원인을 고치라" 지만, 그 원인은 저장소 안에 없을 가능성이 높다.** 2026-10-14 회차가 확인한 것: `check` 잡이 단계를 하나도 실행하지 않고 2~3초 만에 실패하며(= `startup_failure`), 같은 패턴이 2026-10-14T01:09Z 를 기점으로 계정의 무관한 저장소 5곳 13/13 에서 동시에 시작됐다. PR #5(`origin/auto/2026-10-15-0938`)도 `git diff --stat origin/main` 으로 확인했듯 `pipeline.ts`·`text.ts`·새 테스트 3파일뿐이고 `.github/`·`package.json`·`package-lock.json` 을 건드리지 않았다(이번 정찰에서 직접 확인). `.github/workflows/check.yml` 에는 재사용 워크플로도 secret 참조도 composite action 도 없다(이번 정찰에서 전문 66줄을 읽음). **그러니 "워크플로를 고쳐서 통과시키려" 하지 말 것 — 금지이고, 효과도 없다.** 이 과제는 그 대신 "같은 검증을 로컬에서 재현" 이라는 지시의 재현 가능한 부분을 저장소에 영구히 남기는 것이다.
  - 같은 결론(「저장소는 결백하다」)을 **또 한 번 코드 변경 0건으로 보고하지 말 것** — 2026-10-14 회차가 그렇게 끝나 `변경없음` 판정을 받았다. 이번엔 반드시 산출물을 남긴다.
  - `scripts/dev-reset.sh` 와 섞지 말 것: `dev-reset.sh` 는 DB 를 비우고 개발 서버를 띄우는 **개발용**이고, `ci-local.sh` 는 CI 단계를 그대로 밟는 **검증용**이다. 하나로 합치려 들지 말고 나란히 둔다(운영자 지시: 같은 종류가 여러 경로에 있으면 통합 말고 좁히기).
  - `scripts/seed-demo.mjs` 는 호스트가 `127.0.0.1`/`localhost` 가 아니면 거부한다 — 스크립트가 다른 호스트를 쓰게 만들지 말 것.
  - 보호 경로 금지: `db/migrations/`, `packages/core/src/auth/*`, `routes/sso.ts`, `routes/admin/*` 는 이 과제에서 열 이유가 없다.
  - PR #4(쿼터 `retryAfterSeconds`)·PR #5(`clientAddress`) 의 내용을 이 브랜치에 다시 올리지 말 것 — 중복 PR 이 된다.

- 차선 후보: **`util/text.ts` 나머지 순수 함수 단위 테스트 보강** (가치 3 / 위험 1 / 작업량 S, 프로덕션 파일 0개, `npm run check` 하나로 검증 — DB·브라우저 불필요). `packages/core/src/__tests__/text.test.ts`(108줄)는 `isSafeLink` 중심이고 미덮인 것: `mailboxKey`(`+tag` 제거·local-part 가 `+tag` 뿐일 때 폴백·`a@b@c.com`), `maskEmail`(점 없는 도메인·한 글자 local·`@` 로 시작), `koreanParticle`(종성 유무·끝이 숫자·끝이 `)`·한글 없는 단어), `excerpt`(코드포인트 경계·공백 접기·max 경계 120), `tidyBody`(`\r\n`→`\n`·빈 줄 4개 이상 접기·`trimEnd`). `mailboxKey` 는 "한 메일함 한 번 인증" 제품 규칙을 지탱하므로 가장 먼저 덮을 것. 5회차 연속 보류 중인 항목이다.
