# 과제서 — 2026-10-15-113832 (정찰)

- 과제: `.env.dev` 생성기(`scripts/ci-env.mjs`)가 DB 주소 재지정을 무시해 **검증 도구와 서버가 서로 다른 DB 를 보는 것** 고치기 (가치 4 / 위험 1 / 작업량 S)

- 왜: `scripts/e2e.mjs`(27~28줄), `a11y.mjs`(20), `ui-flows.mjs`(25), `screenshots.mjs`(21), `seed-local.mjs`(26) 는 모두 `process.env.DATABASE_URL ?? 'postgres://postgres:devpass@127.0.0.1:55447/workkit'` 로 **재지정을 허용**하는데, 같은 게이트에서 서버의 설정을 쓰는 `scripts/ci-env.mjs`(12~43줄)는 세 개의 DB URL 을 **상수로 박아** 쓴다. 그래서 `DATABASE_URL` 로 전용 DB 를 지정하면 migrate·seed·검증 스크립트는 전용 DB 로 가고 **서버만 :55447 로 가는** 반쪽 상태가 되어, 검증이 코드와 무관하게 깨진다. 이것이 자율 개선 회차가 `e2e`·`tools`·`ui`·`a11y` 를 로컬에서 끝까지 돌려 green 을 증명하지 못하고 다섯 회차 연속 `verify-failed` 로 끝난 직접 원인이다(상주 dev 컨테이너 `workkit-irumx-dev` 가 :55447 을 선점한다 — 메모리 `worktree-shares-dev-postgres`, 2026-10-14 회차에서 `file integrity: kits/… exists=NO` 로 7개 실패 확인). 고치면 워크플로를 한 줄도 느슨하게 하지 않고 CI 와 같은 검증을 워크트리에서 완주시킬 수 있다.

- 수용 기준:
  1) `node scripts/ci-env.mjs` 를 **아무 환경변수 없이** 돌리면 `.env.dev` 의 내용이 지금과 **바이트 단위로 같다**(CI 는 환경변수를 주지 않으므로 `.github/workflows/check.yml` 의 동작은 변하지 않는다). 고치기 전 `.env.dev` 를 받아 두고 `cmp` 로 증명할 것.
  2) 호스트·포트를 한 군데서 재지정하면 `WORKKIT_DATABASE_URL`·`WORKKIT_ADMIN_DATABASE_URL` **셋 다**(irumx_gateway / postgres) 그 주소로 바뀐다. 재지정 수단은 하나만 고를 것 — 권장은 `WORKKIT_DEV_DB_HOSTPORT`(예: `127.0.0.1:55448`) 같은 **호스트:포트 한 개 변수**. `DATABASE_URL` 전체를 파싱해 역할만 바꿔 끼우는 방식은 비밀번호·DB 이름까지 끌려와 역할 전환(`WORKKIT_DB_ROLE`)과 어긋날 수 있으니 피할 것.
  3) 테스트가 증명할 것: 환경변수를 준 경우와 안 준 경우를 **실제로 `ci-env.mjs` 를 실행해** `.env.dev` 를 읽고 단언한다(소스 문자열 검사 금지 — 운영자 규칙). 기존 단위 시험 디렉터리에 `scripts` 용 시험이 없으면 `packages/core/src/__tests__/` 가 아니라 **`scripts` 를 직접 실행하는 시험**으로 둘 것(`node:child_process` + 임시 작업 디렉터리). `.env.dev` 를 저장소 루트에 쓰는 부작용이 있으니 임시 디렉터리에서 돌리고 지울 것.
  4) 전용 DB 로 격리해 CI 순서를 완주한 결과를 원장에 적는다 — 특히 `node scripts/e2e.mjs` 가 **148 passed / 7 failed 가 아니라 전부 통과**로 바뀌는 것. 이것이 이 과제의 실제 가치다.

- 건드릴 파일 (프로덕션 2개):
  - `scripts/ci-env.mjs` — `env` 객체(12~43줄)의 `WORKKIT_DATABASE_URL`(17)·`WORKKIT_ADMIN_DATABASE_URL`(19) 두 줄이 쓰는 호스트:포트를 상수 하나(`const hostport = process.env.WORKKIT_DEV_DB_HOSTPORT ?? '127.0.0.1:55447'`)에서 끌어오게. 파일 상단 영어 주석에 한 줄 추가(이 저장소 관례: 상단 설명 주석은 영어).
  - `scripts/dev-reset.sh` — 같은 변수를 존중하게(9~11줄의 세 `DATABASE_URL` 상수). **선택**: 1번 파일만으로 수용 기준 1~4 가 성립하면 건드리지 말 것.
  - 금지: `.github/workflows/check.yml`(보호 경로, 지금 동작이 그대로 유지되는 것이 수용 기준 1), `packages/core/src/env.ts`, `packages/tools/**`.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm ci` (워크트리에 node_modules 가 **없다** — 확인함. 안 하면 `vitest: not found`)
  - `npx vitest run <새 시험 파일>` → `npm run check` (eslint + 4개 패키지 typecheck + vitest, 직전 회차 기준 11파일 136 passed)
  - `npm run build`
  - 격리 DB 로 CI 순서 완주:
    `docker run -d --name wk-pg-wt -p 55448:5432 -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=workkit postgres:16-alpine`
    → `WORKKIT_DEV_DB_HOSTPORT=127.0.0.1:55448 DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55448/workkit WORKKIT_APP_DB_PASSWORD=devapppass WORKKIT_GATEWAY_DB_PASSWORD=devgatewaypass node scripts/migrate.mjs`
    → `DATABASE_URL=…55448/workkit node scripts/seed-local.mjs` (미디어 바이트를 이 워크트리의 `data/media` 에 새로 깐다 — 7개 실패의 원인이 사라지는 지점)
    → `WORKKIT_DEV_DB_HOSTPORT=127.0.0.1:55448 node scripts/ci-env.mjs` → fakes(18997/18998) + 서버(8797) 띄우기 → `DATABASE_URL=…55448/workkit node scripts/e2e.mjs`
    → `npx playwright install --with-deps chromium` → `node scripts/tools-flows.mjs` → `node scripts/ui-flows.mjs` → `node scripts/a11y.mjs`
  - 끝나면 `docker rm -f wk-pg-wt`. **상주 컨테이너 `workkit-irumx-dev`(:55447)는 절대 건드리지 말 것** — 본 체크아웃의 개발 DB 다.

- 위험과 피할 것:
  - **CI 실패의 1차 원인은 저장소 밖일 가능성이 높다.** 이 세션은 `gh` 가 인증되지 않고(사설 저장소라 익명 API 는 404) 다른 회차 디렉터리도 읽을 수 없어 PR #5 의 잡 주석을 **확인하지 못했다(미확인)**. 확립된 증거는 2026-10-14 수리 회차가 인증 API 로 읽어 둔 주석 *"The job was not started because recent account payments have failed or your spending limit needs to be increased."* + `steps: []` 뿐이고, 서로 다른 diff(PR #2~#5)가 같은 모양으로 실패한 것이 이 설명과 맞는다. **그러니 워크플로나 테스트 단언을 "통과하게" 고치지 말 것** — 2026-10-14 2333 회차가 그 가설(db-breaker 벽시계 상수)을 2코어 핀·5배 과구독·10회 반복으로 측정해 **기각**했고(`dropped` 는 시간이 아니라 시도 횟수로 묶여 항상 12), 메모리 `db-breaker-test-is-not-flaky`·`ci-check-fails-on-actions-billing-block` 에 남아 있다. 저장소 안에서 할 수 있는 일은 **로컬에서 CI 와 같은 검증을 완주시켜 green 을 증명하는 것**이고, 이 과제가 바로 그 걸림돌을 치운다.
  - 같은 종류의 설정 생성기를 **통합하려 하지 말 것**(운영자 규칙). `ci-env.mjs` 와 `dev-reset.sh` 는 각자의 계약을 지킨다 — 좁게만 고친다.
  - `.env.dev` 는 저장소 루트에 쓰이고 `.gitignore` 여부를 **확인하지 않았다(미확인)** — 커밋에 들어가지 않는지 `git status` 로 확인할 것.
  - 역할 전환(`WORKKIT_DB_ROLE: workkit_app`)·비밀번호 상수는 손대지 말 것. 공유 Supabase 와 `db/` 마이그레이션은 보호 구역이다(커밋 8abcd8a·63dee8d).

- 차선 후보: **`toTable` 의 고유화 접미사가 시트의 실제 열 이름과 충돌한다** (`packages/tools/src/table/types.ts:97~105`, 가치 3 / 위험 2 / 작업량 S) — `자산번호,자산번호,자산번호 (2)` 가 `['자산번호','자산번호 (2)','자산번호 (2)']` 로 접히는 것을 직전 회차가 probe 로 확인했다. 비어 있는 이름까지 번호를 올리는 방향(`X (2)` 가 있으면 `X (3)`). 단위 시험만으로 끝나고(`npx vitest run compare.test.ts`) 프로덕션 1파일. 1순위가 성립하지 않거나(예: `ci-env.mjs` 재지정이 이미 다른 경로로 되어 있었다면) Docker 를 쓸 수 없을 때 이것을 고를 것.
