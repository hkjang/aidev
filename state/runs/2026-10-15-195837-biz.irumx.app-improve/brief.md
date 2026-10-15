# 과제서 — 2026-10-15-195837-biz.irumx.app-improve

- 과제: ui-flows 가 깨진 기다림에 끌려 죽는 것 — 다섯 흐름을 `flow()` 로 감싸 중단 이유를 흐름 이름과 함께 남기고 남은 흐름·요약·정리 코드는 계속 돌게 (가치 4 / 위험 1 / 작업량 S)
- 왜: `scripts/ui-flows.mjs` 는 기다림 12군데(85·87·95·98·103·117·131·136·141·199·205·208행)를 `.catch(() => undefined)` 로 삼킨 뒤 그 요소를 무조건 클릭하므로, 한 단계가 느려지면 `check()` 가 `FAIL` 한 줄만 찍고 **바로 다음 줄**에서 Playwright 기본 타임아웃(30초) 예외가 난다. 이 파일은 모듈 최상위 `await` 로만 되어 있고 `try` 가 한 곳도 없어(파일 전체 확인) 그 예외가 그대로 프로세스를 죽인다 → 216행의 요약 줄(`N passed, M problem(s)`)·218행의 `problems` 목록·뒤 블록(수금·콘솔)·214~215행 `browser.close()`/`db.end()` 가 **전부 생략**된다. 결제 차단 직전 마지막 실제 main run 38490598257 이 바로 이 모양으로 죽었고(`locator.click: Timeout 30000ms exceeded … getByRole(group,{name:/제안: 업무 만들기/}) … ui-flows.mjs:140:50`), 그래서 "15초 안에 제안 카드가 왜 안 떴는지" 가 로그에 남지 않았다. 이 과제가 들어가면 같은 실패가 다음부터는 한 번의 실행으로 흐름 이름 + 예외 첫 줄 + 그 흐름이 거기까지 통과시킨 `check` 들과 함께 남는다.

- 수용 기준:
  1) **지금 죽는 것을 먼저 보인다(사전 실패).** 흐름 하나를 일부러 실패시킨 상태로 `node scripts/ui-flows.mjs` 를 돌려, 마지막 요약 줄과 `problems` 목록과 뒤 블록(`수금:`·`콘솔:` 헤더)이 출력에 **없다**는 것을 붙여 둘 것. 권장 변이(짧게 끝남): 114행을 `await page.getByLabel('없는 라벨 업무 제목').fill(title, { timeout: 1000 })` 로 바꿔 '업무' 흐름만 1초에 터뜨린다.
  2) 같은 변이를 유지한 채 수정 후 다시 돌리면 ① `업무` 뒤의 `에이전트:`·`수금:`·`콘솔:` 블록이 계속 돌고 ② 마지막 요약 줄이 찍히고 ③ `problems` 에 흐름 이름과 예외 첫 줄이 담긴 한 줄(예: `- 업무 흐름이 끝까지 돌았다 :: TimeoutError: locator.fill: Timeout 1000ms exceeded…`)이 있고 ④ 종료코드는 1 이다(`echo $?`). 변이를 `git checkout -- scripts/ui-flows.mjs` 가 아니라 손으로 되돌려 수정만 남길 것.
  3) 변이를 되돌린 정상 상태에서 `node scripts/ui-flows.mjs` 가 **`16 passed, 0 problem(s)`** 로 전과 똑같이 끝난다(현재 값: `check()` 호출 16개 — 90·96·105·106 / 118·122 / 137·139·143·147 / 155·165·174 / 201·206·210행). 성공한 흐름은 `check()` 를 **추가로 부르지 않는다** — 통과 수가 16 에서 바뀌면 게이트 의미가 달라진 것이므로 잘못된 구현이다.
  4) `npm run check` 통과(`eslint .` 가 `scripts/**/*.mjs` 를 보고 있다 — eslint.config.js:111).

- 건드릴 파일: **`scripts/ui-flows.mjs` 한 개뿐**(프로덕션 코드 0개).
  - `flow(name, body)` 를 `check()`(43행) 바로 아래에 추가. 권장 형태:
    ```js
    /** Runs one flow; a broken wait stops that flow by name, not the run. */
    async function flow(name, body) {
      try {
        await body()
      } catch (err) {
        check(`${name} 흐름이 끝까지 돌았다`, false, `${err?.name ?? 'Error'}: ${String(err?.message ?? err).split('\n')[0].slice(0, 200)}`)
      }
    }
    ```
  - 다섯 블록을 그대로 감싼다. `console.log` 헤더(77·109·125·150·180행)는 지금 위치에 두고 바로 뒤의 `{ … }` 만 바꾼다:
    `console.log('\n결재: …')` → `await flow('결재', async () => { …기존 본문 그대로… })`
    이름은 `결재` / `업무` / `에이전트` / `수금` / `콘솔`.
  - 본문은 **한 줄도 고치지 않는다**(들여쓰기만 그대로 유지해도 되고 재들여쓰기 해도 된다 — diff 를 작게 하려면 유지). 각 블록의 `const title`·`const page`·`const dialog`·`const after` 는 블록 스코프라 화살표 함수로 감싸도 이름 충돌이 없다.

- 검증 명령 (이 저장소에서 실제로 도는 순서. `.github/workflows/check.yml` 42~63행과 같다. 전체 10분 이상):
  ```sh
  npm ci --no-audit --fund=false            # node_modules 가 없으면 (ESLint 지원 종료 경고 1건은 정상)
  npm run check                             # eslint + 세 workspace tsc + vitest
  npm run build
  node scripts/ci-env.mjs                   # .env.dev written
  docker run -d --rm --name biz-pg -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=biz -p 55451:5432 postgres:16-alpine
  until docker exec biz-pg pg_isready -U postgres; do sleep 1; done
  DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55451/biz BIZ_APP_DB_PASSWORD=devapppass BIZ_GATEWAY_DB_PASSWORD=devgatewaypass node scripts/migrate.mjs   # applied 10 migrations
  DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55451/biz node scripts/seed-local.mjs
  nohup node scripts/fake-yeopjari.mjs 18911 > fake-yeopjari.log 2>&1 &
  nohup node scripts/fake-ai.mjs 18912 > fake-ai.log 2>&1 &
  set -a; . ./.env.dev; set +a; nohup node apps/server/dist/index.js > server.log 2>&1 &
  curl -sf http://localhost:8811/api/v1/health     # {"ok":true,"db":true,…}
  npx playwright install chromium           # --with-deps 는 sudo 비밀번호를 요구해 실패한다
  node scripts/seed-demo.mjs
  node scripts/ui-flows.mjs; echo "exit=$?"        # 이 과제의 증거는 이 명령의 출력이다
  ```
  뒷정리: `docker rm -f biz-pg`, fake/server 프로세스 종료, `*.log` 삭제. `dist/`·`.env.dev` 는 이미 `.gitignore` 에 있다.

- 위험과 피할 것:
  - **타임아웃을 늘리지 말 것.** 15초·30초 숫자를 키우는 것은 게이트 완화이고 과거 회차에서 금지됐다. 이 과제는 "느린 것을 참아 주기" 가 아니라 "죽는 이유를 남기기" 다.
  - **`.catch(() => undefined)` 12군데를 지우지 말 것.** 지우면 기다림 실패가 곧바로 흐름 중단이 되어 통과 수가 달라지고, 변경이 "진단 추가" 가 아니라 "게이트 강화" 로 번진다. 이번 회차는 블록 감싸기만.
  - **73~75행의 `person()` 로그인 3건은 건드리지 말 것** — `flow` 밖이라 로그인 실패 시 여전히 보고 없이 죽지만, 같은 파일이라도 별개 조각으로 분리해 뒀다(ideas.json `[2/1/S]`). 범위 확장 금지.
  - **`scripts/e2e.mjs`·`screenshots.mjs`·`a11y.mjs`·`.github/workflows/**` 는 열지도 말 것.** 특히 `check.yml` 의 `cancel-in-progress` 는 별개 과제다.
  - 1·2·3 블록은 같은 `sales` 페이지를 쓴다. 앞 블록이 중간에 죽은 뒤 뒤 블록이 돌면 각 블록이 첫 줄에서 `page.goto` 를 하므로 보통은 독립이지만, `FAIL` 이 연쇄로 더 날 수 있다 — 기준 3)(정상 상태 16 passed)이 이것을 잡아 준다.
  - CI 자체는 지금 저장소 밖 이유(GitHub Actions 결제·지출한도 차단)로 `steps=0` 3초에 죽는다. 이 과제로 CI 가 녹색이 되지는 **않는다** — 목표는 결제 복구 후 `ui-flows` 가 또 실패했을 때 로그에 원인이 남게 하는 것이다. CI 녹색을 증거로 삼지 말고 위 로컬 실행 출력을 증거로 쓸 것.
  - 미확인: 변이 없이 실패를 재현하는 길(= main 의 진짜 red 원인)은 이번에도 모른다. 로컬에서는 4회 연속 16/16 통과했다. 원인을 추측해 셀렉터나 대기 조건을 고치지 말 것.

- 차선 후보: **프로젝트 손익 조회의 from/to 역순·범위 밖 기본값** — `packages/core/src/routes/finance.ts:734-735` 가 `from`/`to` 가 비면 `'1900-01-01'`·`'9999-12-31'` 을 `v.day` 를 지나지 않고 쿼리에 넣고, `to < from` 도 거르지 않아 0원 손익을 정상 응답으로 낸다. 기본값 의미를 바꾸면 DB e2e 가 필요하고 finance 쓰기 경로에 가깝다(위험 2). 1순위가 성립하지 않을 때만.
