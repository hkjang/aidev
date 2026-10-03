- 과제: **[수정 과제]** `npm test` 를 두 회차 연속 떨어뜨린 두 결함(플레이그라운드 녹화 패널의 낡은 읽기 + 음수 step duration)을 고쳐 릴리즈 검증을 초록으로 되돌림 (가치 5 / 위험 1 / 작업량 S)

- 왜: 이 base(pristine `main@a2a44d4`)에서 전체 스모크 스위트는 직전 회차에 두 번 돌려 **두 번 다 실패**했다(`149 passed, 1 failed` / `148 passed, 2 failed`). 실패는 ① `a person can record and edit a scenario from the playground page` — 검사가 새로고침이 끝나기 전의 패널을 읽어 `["0. gotohttp://127.0.0.1:3911/demo/test-page?lang=ko↑↓✕"]` 한 행만 보는 동기화 결함(두 번 모두 실패, 러너가 말한 "같은 이유"), ② `a run records per-step durations and the step that failed` — 서버가 Playwright 의 음수 `step.duration` 을 그대로 API 로 내보내 `durationMs: -1868` 같은 값이 나오는 제품 결함(2회 중 1회)이다. ①은 이미 검증된 수정 커밋이 로컬에 있으나 머지되지 않았고, ②는 README 가 명시한 계약("음수가 되지 않습니다")을 서버가 지키지 않는 것이다. 둘을 고치면 릴리즈 게이트가 다시 초록이 되고, 이미 올라간 CI 의 smoke 잡을 게이트로 승격할 조건이 갖춰진다.

- 수용 기준:
  1) `npm test` 가 exit 0 으로 끝나고 요약 줄에 `0 failed` 가 찍힌다(브라우저가 깔린 체크아웃에서). 같은 트리에서 2회 연속 통과를 확인한다 — ①은 확률적 플레이크였으므로 1회 통과는 증거가 약하다.
  2) `a person can record and edit a scenario from the playground page` 검사가 **낡은 1행 읽기로는 더 이상 통과할 수 없다**: 패널이 `testId=` 를 담을 때까지 다시 묻고, 읽은 행 수가 1 이하이면 단정이 깨진다.
  3) `GET /api/runs/:id` 가 돌려주는 `tests[].durationMs` 와 `tests[].steps[].durationMs`(중첩 포함)에 음수가 없다 — 음수를 만드는 두 경로(생성된 step 리포터, json 리포트 폴백)와 테스트 레벨 duration 세 곳이 **같은 입력을 같은 값으로** 읽는다. `tools/smoke-test.mjs:2815` 의 기존 단정이 그대로(느슨해지지 않은 채) 통과해야 한다.

- 건드릴 파일 (프로덕션 2개):
  - `tools/smoke-test.mjs:4067-4120` — `check("a person can record and edit a scenario from the playground page")`. **로컬 커밋 `d2f428d` ("Read the recording panel only once the refresh has landed") 를 그대로 체리픽하면 된다.** 이 회차에 HEAD 의 4092-4117 과 그 커밋의 diff 컨텍스트가 **한 줄도 다르지 않음을 대조해 확인**했다(즉 충돌 없이 적용된다). 내용: (a) `#refreshRecordingBtn` 클릭 → `assert/text` 로 `[data-testid='recording-steps']` 컨테이너가 `testId=` 를 담을 때까지 최대 10회 다시 누름(버튼은 핸들러 동안 스스로 disabled 되므로 두 번째 클릭이 앞 새로고침을 기다린다), (b) `assert(before > 1, …)` 추가, (c) 삭제 뒤의 `wait-for { sleepMs: 1200 }` 고정 대기를 행 수가 줄 때까지 250ms×40 폴링으로 교체. `[data-testid='recording-steps']` 컨테이너가 HEAD 에 실재하는 것은 확인했다(`public/playground.html:104`, `public/assets/playground.js:393`).
  - `server.js` — 음수 duration 을 API 경계에서 정규화. **세 곳을 한 변경에서 함께** 고칠 것(한쪽만 고치면 다른 경로가 같은 값을 다르게 읽는다):
    - `server.js:1193-1197` — 생성되는 step 리포터 소스(템플릿 문자열) 안의 `trim()`: `durationMs: typeof step.duration === "number" ? step.duration : null`.
    - `server.js:1246-1255` — `flattenReportSteps()`: `durationMs: typeof step.duration === "number" ? step.duration : undefined`. (json 리포트 폴백 경로)
    - `server.js:2421` — 테스트 레벨 `durationMs: last?.duration ?? null`.
    권장: 음수를 0 으로 끌어올린다(`Math.max(0, …)`). null 로 떨어뜨리면 `tools/smoke-test.mjs:2817` 의 "every step duration was null" 단정과 충돌할 여지가 있고, `public/assets/runs.js:134` 의 막대 폭 계산도 0 을 이미 다룬다. 왜 음수가 오는지 한 줄 주석으로 남길 것(콜드 chromium 의 `Before Hooks`/fixture 구간에서 Playwright 가 음수를 보고한다 — 실측 `-1868 / -1646 / -457`).
  - 테스트는 **새로 쓰지 않아도 된다**: `tools/smoke-test.mjs:2815` 가 이미 음수를 FAIL 로 잡는 단정이고, 수용 기준 2) 가 녹화 검사를 낡은 읽기로 통과할 수 없게 만든다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) `npm ci` — **이 워크트리에는 `node_modules` 가 없다**(이 회차에 확인). 없으면 `npm test` 는 `Cannot find module 'swagger-ui-dist/package.json'` 로 즉사한다. 수 분 걸리니 먼저 떼어 둘 것.
  2) `npx playwright install chromium` — 브라우저가 없으면 문제의 두 검사가 **SKIP 되어 결함이 보이지 않는다**(~360MB).
  3) `npm run check` (= `node --check server.js`) — `server.js` 를 고쳤으니 반드시.
  4) `npm test` — 러너가 실패한 바로 그 명령. 고치기 **전에 한 번** 돌려 FAIL 줄 두 개를 눈으로 확인하고(실패 재현), 고친 **뒤 2회** 돌려 둘 다 `0 failed / exit 0` 인지 볼 것. 1회 약 6분.
  5) 릴리즈 체크리스트 1절과 같은 `SMOKE_REQUIRE_BROWSER=1 npm test` 로 마무리(브라우저 미설치를 FAIL 로 승격 — 기대값 `150 passed, 0 failed, 0 skipped`).

- 위험과 피할 것:
  - **검사를 느슨하게 해서 통과시키지 말 것** (러너의 금지 사항). ①은 단정을 **더 강하게** 만드는 수정이고(`before > 1`), ②는 테스트가 아니라 제품을 고치는 수정이다. `tools/smoke-test.mjs:2815-2818` 의 단정을 완화하는 방향은 반려 사유다.
  - 같은 값을 읽는 경로가 셋이다(생성 리포터 / json 폴백 / 테스트 레벨). 한 곳만 고치고 "통과했다" 고 하지 말 것 — 생성 리포터는 **템플릿 문자열 안**이라 눈에 잘 띄지 않는다. 실행 경로는 보통 생성 리포터 쪽이고 `flattenReportSteps` 는 폴백이므로, grep 으로 고쳤다고 끝내지 말고 실제 run 응답의 중첩 steps 까지 확인할 것.
  - 로컬 커밋 `254a099`("Report a failed server boot…", harness 자가검사 4파일)도 미머지지만 **이번 과제에 넣지 말 것** — 이번 실패의 원인이 아니고 파일 수와 위험만 늘린다. 별 회차로.
  - 보호 경로를 건드리지 않는다: `principals.json` 권한, `secrets/` 스크러빙, Dockerfile/entrypoint, `.github/workflows/`(이 base 에는 아직 없다).
  - 확률적 플레이크다: 유휴 상태에서는 ①이 통과할 수 있다. 재현이 안 되면 전체 스위트를 동시에 한 벌 더 돌려 부하를 준 채 확인할 것(직전 회차에 이 방법으로 3/3 재현).
  - **미확인**: 러너의 verify 머신에 chromium 이 깔려 있는지는 확인하지 못했다. 만약 브라우저 없는 체크아웃이라면 이 두 검사는 SKIP 되고 `npm test` 는 **다른 이유로** 실패하는 것이다. 그러니 **첫 행동은 코드 수정이 아니라 `npm test` 를 돌려 FAIL 줄을 읽는 것**이다. FAIL 줄이 위 두 개가 아니면 그 줄이 말하는 검사로 과제를 바꿀 것(아래 차선 후보).

- 차선 후보: 브라우저 없는 체크아웃에서 `npm test` 가 FAIL 하는 검사를 찾아 고치기 — 2026-10-03 정찰이 "스모크 2건이 거짓 FAIL" 이라고 적었으나 그 회차는 error 로 끝나 **검증되지 않았다**. 브라우저 없이 `npm test` 를 돌려 FAIL 줄을 먼저 확보하고, browser-dependent 검사가 `skip` 대신 `fail` 로 떨어지는 지점(`tools/smoke-test.mjs:34-45` 의 `check()` 가 bare `'skip'` 만 인식한다)을 고칠 것.
