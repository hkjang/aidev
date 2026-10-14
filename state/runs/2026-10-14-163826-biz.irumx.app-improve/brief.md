- 과제: PR #6 CI 가 저장소 단계 이전에 죽는 것 — `services:` 의 postgres 를 명시적 단계로 바꿔 실패를 로그에 남게 하기 (가치 4 / 위험 2 / 작업량 S)
- 왜: PR #6 의 내용물은 `packages/core/src/__tests__/domain.test.ts` **+58행 한 파일뿐**이고(`git diff --stat main a65df42`), 직전 회차가 바로 그 내용으로 `npm ci → npm run check(164 passed) → npm run build → ci-env → docker postgres:16-alpine → migrate(10 migrations) → seed-local → fakes+server → e2e 300 passed` 까지 로컬에서 전부 통과시켰다. 그런데 CI 는 3초 만에 annotation 2건으로 죽고 저장소 쪽 단계는 **하나도 실행되지 않았다** — 즉 실패는 PR 내용이 아니라 잡이 시작되기 전 구간(현재 워크플로에서 그 구간은 `services:` 컨테이너 초기화가 유일하다)에서 난다. 이 구간은 지금 로그가 남지 않아 세 회차째 원인을 못 읽고 있으므로, 같은 게이트를 하나도 약화시키지 않은 채 그 구간을 **출력이 남는 단계로 끌어내리는 것**이 이번 회차의 목적이다.

- 수용 기준:
  1) `.github/workflows/check.yml` 에서 `services:` 블록을 제거하고, `actions/checkout` 직후에 PostgreSQL 을 띄우는 **명시적 run 단계**를 넣는다. 이미지·포트·자격증명은 **지금과 글자 그대로 동일**해야 한다: `postgres:16-alpine`, 호스트 포트 `55451`→컨테이너 `5432`, `POSTGRES_PASSWORD=devpass`, `POSTGRES_DB=biz`. 준비 대기는 `pg_isready -U postgres` 로 하고(현재 health-cmd 와 동일), 최대 대기는 현재 설정(5s 간격 × 20회 = 100초)보다 **짧지 않게** 잡는다. 실패 시 `docker logs` 를 찍고 비0으로 종료한다.
  2) 기존 단계 `Lint, types, unit tests` / `Build` / `Development settings` / `Migrate and seed` / `Start the fakes and the server` / `End-to-end` / `Browser` / `Demo company` / `Work flows through the screens` / `Every screen` / `Accessibility` / 두 개의 업로드·로그 단계는 **순서·명령·`if:` 조건까지 그대로 유지**한다. `continue-on-error`·`|| true`·단계 삭제·`timeout-minutes` 확대·테스트 축소는 **금지**. 이번 변경으로 통과 조건이 1mm도 느슨해지면 안 된다.
  3) 로컬에서 워크플로 47~60행과 동일한 순서를 직접 재현해 통과시킨다 — 새 컨테이너 기동 명령 → `node scripts/migrate.mjs` → `node scripts/seed-local.mjs` → fakes + `apps/server/dist/index.js` → `node scripts/e2e.mjs` 가 **300 passed, 0 failed**. 직전 회차가 같은 순서를 통과시켰으므로 이 수치는 기준선이다. 달라지면 새 단계가 뭔가를 바꾼 것이니 멈추고 원인을 적을 것.
  4) `npm run check`(ESLint + 세 workspace tsc + 전체 **164개** 시험)와 `npm run build` 가 통과한다. 이 두 명령의 출력 수치를 회차 노트에 그대로 인용한다.
  5) 워크플로 YAML 이 실제로 파싱되는 것을 보인다: `node -e "require('fs')"` 수준이 아니라, 저장소에 이미 있는 도구로 확인되지 않으면 **최소한 탭 문자·CRLF·BOM 이 없음**을 확인하고 그 사실을 노트에 적는다(현재 파일은 공백 들여쓰기·LF 다).

- 건드릴 파일:
  - `.github/workflows/check.yml`:21-33 — `services:` 블록 제거.
  - `.github/workflows/check.yml`:34-35 사이 — `actions/checkout@v4` 뒤에 `- name: PostgreSQL` run 단계 신설(위 기준 1). **프로덕션 코드 변경 0파일.**
  - (이 과제는 워크플로 1파일로 끝난다. 다른 파일에 손대지 말 것.)

- 검증 명령:
  - `npm ci --no-audit --fund=false` (ESLint 지원 종료 경고 1건은 정상)
  - `npm run check`  ← 164 passed 기대
  - `npm run build`
  - `node scripts/ci-env.mjs`  ← `.env.dev written`, exit 0
  - `docker run -d --name bizpg -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=biz -p 55451:5432 postgres:16-alpine` → `pg_isready` 대기 →
    `DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55451/biz BIZ_APP_DB_PASSWORD=devapppass BIZ_GATEWAY_DB_PASSWORD=devgatewaypass node scripts/migrate.mjs` (`applied 10 migrations`) →
    `DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55451/biz node scripts/seed-local.mjs` →
    `node scripts/fake-yeopjari.mjs 18911` · `node scripts/fake-ai.mjs 18912` · `. ./.env.dev` · `node apps/server/dist/index.js` → `/api/v1/health` 가 `{"ok":true,"db":true,…}` →
    `node scripts/e2e.mjs`  ← **300 passed, 0 failed**
  - 끝나면 컨테이너 삭제·fake 프로세스 종료·로그 파일 삭제(`dist/`·`.env.dev` 는 이미 `.gitignore`).

- 위험과 피할 것:
  - **원인은 아직 미확정이다.** 직전 회차가 `gh auth status` = "not logged into any GitHub hosts", 비공개 저장소라 토큰 없는 공개 API 는 `404` 라서 annotation 2건을 읽지 못했다. 이번 정찰 세션에서도 `gh` 는 Bash 권한에 막혀 아예 실행되지 않았다. 따라서 위 "`services:` 초기화 구간" 은 **가장 잘 맞는 가설이지 확정된 원인이 아니다** — 과제서·커밋 메시지·회차 노트에 사실로 쓰지 말고 가설로 쓸 것.
  - **구현자의 세션에서 `gh` 가 인증되어 있다면 그것이 최우선이다.** `gh run list --branch auto/2026-10-14-1208`, `gh api repos/hkjang/biz.irumx.app/check-runs/<id>/annotations` 로 annotation 2건을 먼저 읽고, 그 내용이 위 가설과 다르면 **가설을 버리고 증거를 따르라**. 읽은 원문을 노트에 인용할 것.
  - annotation 이 "Invalid workflow file" 계열이면 `services:` 는 범인이 아니다 — 그 경우 워크플로를 고치지 말고(무엇이 틀렸는지 정확히 알기 전에는) 차선 후보로 넘어가고, 읽은 메시지를 노트에 남겨 다음 회차가 바로 쓰게 하라.
  - 같은 3초·annotation 2건 실패가 **biz / law / yeopjari 세 저장소에서 같은 주기에** 났다는 기록이 보류 아이디어에 있다. 세 저장소가 동시에 같은 모양으로 죽었다면 계정·과금·러너 쪽(저장소 밖) 원인일 수 있고, 그러면 이 변경으로도 CI 는 통과하지 않는다. **그 가능성을 노트에 명시하라.** CI 가 여전히 죽어도 로컬 재현(기준 3·4)이 통과했다면 그것을 솔직히 적고 끝낼 것 — 통과시키려고 게이트를 깎지 말 것.
  - 건드리지 말 것: `packages/core/src/util/validate.ts`(이번 PR 의 시험 대상 — 변이 시험으로 건드렸다면 반드시 `git checkout --` 로 복원), `routes/sso.ts`·`auth/*`, `db/migrations`, finance 쓰기 경로. PR #6 의 `domain.test.ts` +58행은 **그대로 두라** — 이미 비평 승인을 받은 내용이다.
  - 커밋은 한국어 설명형(`영역: 무엇을 왜`). 워크플로 1파일 외에는 아무것도 커밋하지 말 것.

- 차선 후보: **KST 일 경계 `kstDateKey`·`minuteStart` 의 쿼터 창 계약 시험** (3/1/S) — `packages/core/src/util/time.ts` 는 시험 0건이고 `quota.ts` 가 유일한 소비처다. UTC 15:00 경계(`2026-03-14T14:59:59.999Z`→`'2026-03-14'`, `2026-03-14T15:00:00Z`→`'2026-03-15'`)와 `minuteStart` 의 초·밀리초 절삭만 `packages/core/src/__tests__/` 에 고정한다. 프로덕션 변경 0파일, 순수 함수, 가드 변이로 유효성 증명 가능. 미사용 표면(`kstWindowOn`·`overlaps`·`isoDay`·`formatKst`·`backoffMs`)은 늘리지 말 것.
