- 과제: ui-flows 가 기다림 실패를 삼키고 죽는 것 — 깨진 기다림을 이름으로 알리고 남은 흐름은 계속 돌게 (가치 4 / 위험 1 / 작업량 S)
- 왜: `scripts/ui-flows.mjs` 는 모든 기다림을 `await …waitFor({timeout}).catch(() => undefined)` 로 삼킨 뒤 그 요소를 무조건 클릭한다(85·87·95·98·103·117·131·136·141·199·205·208행). 그래서 한 단계가 느려지면 `check()` 가 FAIL 한 줄만 남기고 다음 줄에서 30초 기본 타임아웃 예외로 프로세스가 죽어, CI 로그에는 "어느 기다림이 깨졌는지" 가 아니라 "클릭이 안 됐다" 만 남고 `problems` 목록 출력·뒤 블록(수금·콘솔)·`browser.close()`/`db.end()` 가 전부 생략된다 — main run 38490598257(7ae6b03) 의 실패가 정확히 이 모양이다(`FAIL 쓰기는 제안 카드로 옴` → `locator.click: Timeout 30000ms exceeded … getByRole(group, {name: /제안: 업무 만들기/}) … ui-flows.mjs:140:50`). 고치면 결제 차단이 풀려 CI 가 다시 돌 때 그 red 의 원인(ideas.json 1순위, 로컬 재현 실패로 미확정)이 한 번의 실행으로 드러나고, 한 단계가 깨져도 나머지 흐름의 결함까지 같은 실행에서 보고된다.
- 수용 기준:
  1) 정상 상태에서 출력과 종료 코드가 지금과 같다 — `node scripts/ui-flows.mjs` 가 `16 passed, 0 problem(s)` 로 끝나고 exit 0. (타임아웃 값을 늘리거나 검사를 빼서 통과시키는 것은 금지 — 게이트 완화다.)
  2) 기다림이 깨지면 로그가 그 기다림을 이름으로 말한다 — 어떤 대기가 어느 블록에서 몇 ms 뒤 깨졌는지 `problems` 에 한 줄로 남고, 마지막 `N passed, M problem(s)` + `problems` 목록이 **항상** 출력되며 exit 1 이다(스택트레이스로 죽어 목록이 안 나오는 일이 없다).
  3) 한 블록이 깨져도 남은 블록은 계속 돈다 — 에이전트 블록을 일부러 깨뜨린 실행에서 수금·콘솔 블록의 PASS 가 그대로 찍히고, 끝에 에이전트 블록의 중단 이유가 보고된다.
  4) (3) 을 주입 실패로 증명한다. 권장 주입: 135행의 `/제안: 업무 만들기/` 를 `/제안: 없는카드/` 로 바꿔 한 번 돌리고 출력을 회차 기록에 붙인 뒤 `git checkout -- scripts/ui-flows.mjs` 로 되돌린다. (주입 실패는 커밋하지 말 것.)
- 건드릴 파일: **`scripts/ui-flows.mjs` 한 개만.** (프로덕션 코드 0개)
  - `check(name, ok, detail)` — 그대로 두거나 `ok` 를 돌려주게만 확장. 문구(`PASS`/`FAIL`)와 `problems` 줄 모양은 바꾸지 말 것(회차 기록·로그 검색이 이 문구를 쓴다).
  - 새 도움 함수 두 개를 40~51행 근처에 추가:
    - `async function settle(locator, label, timeout)` — `.catch(() => undefined)` 자리를 대체. 타임아웃이면 `check(label, false, '기다림 시간 초과 ' + timeout + 'ms')` 로 기록하고 전용 센티널(예: `class Skip extends Error {}` 의 인스턴스)을 throw 해 그 블록을 중단한다. `page.waitForURL` 용으로 같은 모양의 `settleUrl(page, re, label, timeout)` 도 필요하다(로케이터가 아니라 페이지 메서드다).
    - `async function flow(name, body)` — 지금의 `console.log('\n결재: …')` + 바로 뒤 `{ … }` 블록을 감싼다. 제목을 찍고 `body()` 를 try/catch 로 돌려, 센티널이면 `중단: <label> 이후 남은 단계를 건너뜀` 을 `problems` 에 넣고, 다른 예외면 `problems` 에 `[<name>] <err.message 앞 300자>` 를 넣은 뒤 **삼키고 다음 flow 로 넘어간다**.
  - 다섯 블록(77·109·125·150·180행에서 시작하는 결재·업무·에이전트·수금·콘솔)을 `await flow('…', async () => { … })` 로 감싸고, 각 블록 안의 `.catch(() => undefined)` 기다림을 `settle`/`settleUrl` 로 바꾼다. 블록 본문의 단계 순서·선택자·타임아웃 값·`check` 문구는 그대로.
  - 214~216행의 `browser.close()` / `db.end()` / 요약 출력은 그대로 두면 이제 항상 도달한다. 종료는 지금처럼 `problems.length > 0` 이면 exit 1.
- 검증 명령 (이 저장소에서 실제로 도는 순서 — 2026-10-15 회차에 13단계 전부 로컬 통과 확인됨):
  ```
  npm ci --no-audit --fund=false
  npm run check            # eslint + tsc 3개 + vitest (ui-flows 는 vitest 대상이 아니지만 eslint 대상이다 — 반드시 통과시킬 것)
  npm run build
  node scripts/ci-env.mjs
  docker run -d --name biz-pg -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=biz -p 55451:5432 postgres:16-alpine
  DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55451/biz BIZ_APP_DB_PASSWORD=devapppass BIZ_GATEWAY_DB_PASSWORD=devgatewaypass node scripts/migrate.mjs
  DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55451/biz node scripts/seed-local.mjs
  nohup node scripts/fake-yeopjari.mjs 18911 & nohup node scripts/fake-ai.mjs 18912 &
  set -a; . ./.env.dev; set +a; nohup node apps/server/dist/index.js > server.log 2>&1 &
  curl -sf http://localhost:8811/api/v1/health      # {"ok":true,"db":true,...}
  npx playwright install chromium                   # --with-deps 는 sudo 비밀번호를 요구해 실패한다 — deps 없이 받을 것
  node scripts/seed-demo.mjs
  node scripts/ui-flows.mjs                         # 기준 (1): 16 passed, 0 problem(s)
  ```
  주입 실패(기준 3·4)는 마지막 줄만 다시 돌리면 된다. 뒷정리: `docker rm -f biz-pg`, fake 프로세스 종료, `server.log`·`fake-*.log` 삭제(`dist/`·`.env.dev` 는 이미 `.gitignore`).
- 위험과 피할 것:
  - **게이트를 느슨하게 만들지 말 것.** 타임아웃(10000·15000) 상향, `check` 삭제, 실패를 경고로 격하, exit 0 으로 끝내기는 모두 금지. 이 변경의 목적은 같은 실패를 **더 잘 보고하는 것** 뿐이다.
  - `.github/workflows/*` 는 건드리지 말 것. CI 는 지금 **계정 결제·지출한도 차단**으로 단계 0개에서 죽는다(annotation 원문: `The job was not started because recent account payments have failed or your spending limit needs to be increased`). 즉 이 변경으로 CI 가 녹색이 되는지는 이 회차에 확인할 수 없다 — 로컬 실행이 유일한 증거다. 그래도 과제는 성립한다(결제 복구 시 원인이 로그에 남는 것이 목적).
  - **ideas.json 1순위(제안 카드 15초 미출현)를 추측으로 고치지 말 것.** 로컬 4/4 통과라 원인 미확정이다. 이번 회차는 **진단 가능성만** 올린다.
  - 다른 스크립트(`scripts/e2e.mjs`·`screenshots.mjs`·`a11y.mjs`)에 같은 도움 함수를 복제·추출하지 말 것 — 운영자 지시: 여러 경로의 비슷한 유틸은 통합하지 말고 해당 경로만 좁혀 고친다. 공용 모듈 신설도 금지.
  - 73~75행의 `person()`(로그인 3명)은 flow 밖이다. 거기서 실패하면 지금처럼 죽는다 — 이번 범위 밖으로 둘 것(범위 확장 금지). 필요하면 ideas.json 에 남기면 된다.
  - 미확인: 주입 실패 시 `card.isVisible()` 이 false 를 돌려주는지 예외를 던지는지는 실행으로 확인하지 않았다(Playwright 의 `isVisible()` 은 통상 false). `settle` 이 먼저 센티널을 던지면 이 질문 자체가 사라지는 설계이므로, 그렇게 짜는 쪽을 권한다.
- 차선 후보: main 푸시의 concurrency `cancel-in-progress` 가 머지 커밋 run 을 매번 취소하는 것 (3/1/S) — `.github/workflows/check.yml:13-15` 의 `cancel-in-progress: true` 를 `${{ github.ref != 'refs/heads/main' }}` 로. 게이트 강화이지 완화가 아니지만 보호 경로 단독 변경이므로, 1순위가 성립하지 않을 때만 그 한 줄만 바꿀 것. 그마저 어려우면 `packages/core/src/routes/finance.ts:734-735` 의 프로젝트 손익 조회 from/to(기본값 `'1900-01-01'`·`'9999-12-31'` 가 `v.day` 를 지나지 않고 `to < from` 도 거르지 않음)를 `period()` 경유로 좁히기 (3/2/S, DB e2e 필요).
