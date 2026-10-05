- 과제: 녹화 중복 제거가 서로 다른 `selectOption` 단계를 삼켜 틀린 옵션이 녹화되는 것 수정 (가치 4 / 위험 2 / 작업량 S)

- 왜: `server.js:7359-7366` 의 중복 제거 서명이 `action` / `locator` / `previous.value === step.value` 만 비교한다. 그런데 `selectOption` 단계는 `step.values = [masked.value]` 로만 값을 담고 `step.value` 는 **절대 설정되지 않는다**(`server.js:7326-7332` 의 else 분기는 `fill` 전용 — 직접 읽어 확인). 그래서 같은 `<select>` 에서 600ms 안에 일어난 **서로 다른** 옵션 선택 두 건이 `undefined === undefined` 로 "동일" 판정돼 뒤의 것이 버려지고, 조기 `return` 이라 **먼저 온 중간 값이 살아남고 사용자가 실제로 고른 값이 사라진다**. 녹화된 시나리오가 조용히 틀린 옵션을 고르고, 그대로 export 되어 replay 된다(`server.js:2982` 가 `step.values` 를 그대로 코드로 렌더한다). 서명을 넓히면 녹화가 실제 선택을 보존한다.

- 수용 기준:
  1) 같은 `<select>` 에 600ms 안에 **서로 다른** 옵션을 두 번 선택하면 녹화에 `selectOption` 단계가 **2개** 남고, 두 번째의 `values` 가 마지막으로 고른 옵션이다. (지금은 1개이고 첫 번째 값만 남는다 — 이것이 고치기 전 실패 재현이다.)
  2) 중복 제거 자체는 살아 있다: 같은 `<select>` 에 600ms 안에 **같은** 옵션을 두 번 선택하면 단계가 **1개**다. 이 단정이 없으면 "값 비교를 참조 비교로 바꿔 중복 제거를 통째로 꺼 버린" 회귀가 통과한다.
  3) 새 검사는 프로덕션 배선 그대로여야 한다: 실제 `server.js`, 실제 chromium 세션, 실제 REST API. 손으로 만든 step 객체를 `onEvent` 에 직접 넣어 증명하지 말 것.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server.js:7359-7366` — `SessionManager.startRecording` 안 `onEvent` 의 중복 제거 비교. 서명을 `action` / `locator` / `value` / `values` / `key` 로 넓힐 것. `values` 는 **배열**이라 참조 비교가 항상 false 가 되므로 `locator` 가 이미 쓰는 방식과 같게 `JSON.stringify` 로 비교해야 한다:
    ```
    && previous.value === step.value
    && JSON.stringify(previous.values ?? null) === JSON.stringify(step.values ?? null)
    && previous.key === step.key
    ```
    `key` 는 방어로만 넣을 것 — 레코더는 `event.key !== "Enter"` 로 걸러 Enter 만 보내므로(`server.js:3820-3825`) **현재 배선에서 press 로는 이 결함이 재현되지 않는다**. 보류 아이디어에 적혀 있던 "press 의 `key` 도 삼켜진다" 는 과장이다. press 를 수용 기준이나 실패 재현으로 쓰지 말 것.
    위 7359-7366 의 주석("A click on a checkbox arrives twice, once from click and once from change")은 그대로 유효하다 — 체크박스는 `click` 리스너와 `change` 리스너가 둘 다 `kind:"click"` 을 보내고(`server.js:3795-3812`) 그 단계는 `value`/`values`/`key` 가 모두 없으므로 넓힌 서명으로도 합쳐진다. 주석을 지우지 말 것.
  - `tools/smoke-test.mjs:3909` — 기존 브라우저 검사 `"recording captures what was done and the result replays"` 가 이미 `/demo/test-page` 를 열고 녹화를 켠 뒤 `3921-3925` 의 `for (const [action, body] of [...])` 루프로 API 를 때린다. 그 루프 **바로 뒤에**(또는 인접한 새 `check()` 로) 아래를 추가하면 된다 — 새 페이지·새 세션을 만들 필요가 없다:
    ```
    ["select-option", { locator: { testId: "role-select" }, values: ["operator"] }],
    ["select-option", { locator: { testId: "role-select" }, values: ["admin"] }],
    ```
    그리고 `recording.steps.filter((s) => s.action === "selectOption")` 가 2건이고 마지막 `values` 가 `["admin"]` 임을 단정. 수용 기준 2 는 `values: ["admin"]` 을 연달아 두 번 보내 1건만 남는 것으로 확인.
    확인한 사실: 라우트는 kebab-case **`select-option`** 이다(`server.js:10545`), 요청 바디 키는 `values`(배열) 또는 `value`(`server.js:6582-6587`). `/demo/test-page` 의 `<select>` 는 `data-testid="role-select"` 이고 옵션은 `observer` / `operator` / `admin` 세 개다(`public/demo.html:45`, `public/locales/en.json:152-165`). **데모 페이지에 체크박스는 없다** — 수용 기준 2 를 체크박스로 쓰려 하지 말 것.
    고정 `sleepMs` 대기를 새로 넣지 말고 관측 가능한 사후조건(`steps` 개수)을 폴링할 것 — 2026-10-04 회차가 고정 대기로 생긴 플레이크를 고쳤다. 루프 뒤의 기존 `wait-for { sleepMs: 800 }`(`tools/smoke-test.mjs:3929`)은 이번 범위가 아니니 건드리지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) `npm run check` — 곧 `node --check server.js`. 의존성 불필요, 즉시 끝나고 조용히 exit 0.
  2) **선행**: 이 워크트리에 `node_modules` 가 **없다**(이번 회차 실측). `npm ci`(수 분) → `npx playwright install chromium`(~360MB) 가 필요하다. 시간 예산을 여기에 먼저 떼어 둘 것. 샌드박스에서 `npm ci` 가 권한으로 막힐 수 있다(2026-10-04 에 실제로 막혔다) — 막히면 즉시 `npm install` 로 전환할 것.
  3) **빠른 재현/확인 프로브(6분 스위트를 기다리지 말 것)**: 스크래치 포트·디렉터리 환경변수로 `node server.js` 를 띄우고, 세션 생성 → `/demo/test-page` goto → `POST /api/sessions/{id}/recording/start` → `POST /api/sessions/{id}/pages/{pageId}/select-option` 을 `["operator"]`, `["admin"]` 로 연달아 → `GET /api/sessions/{id}/recording` 의 `steps` 를 찍는다. 고치기 전 `selectOption` 1건(`values: ["operator"]`), 고친 뒤 2건이어야 한다. 2026-10-04 회차가 똑같은 방식(실제 서버·실제 브라우저·실제 REST)으로 성공했다.
  4) **전체**: `SMOKE_REQUIRE_BROWSER=1 npm test` (**약 6분**). 이 base 에서 pristine 베이스라인은 이미 **빨갛다** — 2026-10-04 측정으로 `149 passed, 1 failed` / `148 passed, 2 failed` 였고(이번 회차에 재측정하지 않음 — **미확인**) 실패 2건은 이 과제와 무관한 기존 결함(플레이그라운드 녹화 패널의 낡은 읽기, Playwright step duration 음수)이다. **그 2건을 고치려 하지 말 것 — 범위 밖이다.** 통과 기준은 "새 단정이 통과하고 실패 건수가 베이스라인보다 늘지 않는다".

- 위험과 피할 것:
  - **보호 경로를 건드리지 말 것**: `principals.json` 역할·범위 권한, 승인 게이트(`decidedByVerified`), MCP `tools/list` 필터, 도커 entrypoint/마운트, `.github/workflows`(main 에 아직 없다). 이 과제는 전부 불필요하다.
  - **스크럽 경계와 인접하다**: 비교는 `maskRecordedValue` 를 **거친 뒤의** `step.values` 를 읽는다(`server.js:7327-7332`). 비교나 경고 메시지에 `payload.value` 원문을 끌어오지 말 것 — 마스킹된 값만 쓸 것. `warnings` 에 값을 새로 넣지 말 것.
  - `previous.values === step.values` 같은 **참조 비교 금지** — 배열이라 늘 false 가 되어 중복 제거가 통째로 꺼진다. 수용 기준 2 가 이것을 잡는다.
  - 600ms 창, `payload.kind` 분류, `cleanObject` 호출 순서를 바꾸지 말 것. 이번 변경은 "같은 값을 더 정확히 비교" 하나뿐이다.
  - 같은 값을 읽는 경로를 한쪽만 고치지 말 것: 녹화 `steps` 는 `GET .../recording`, `.../recording/stop` 응답, `.../recording/export`(`server.js:2982` 의 코드 렌더), 그리고 `/playground` 패널이 **같은 배열**을 읽는다. 이번 수정은 배열 자체를 고치므로 네 경로가 함께 맞는다 — export 가 두 번째 `selectOption` 을 코드로 내는지 프로브에서 한 번 눈으로 확인할 것.
  - **파일을 늘리지 말 것.** 프로덕션 `server.js` 1개, 테스트 `tools/smoke-test.mjs` 1개로 끝난다. 새 파일을 만들 필요가 없다.

- 차선 후보: **검증된 녹화 패널 동기화 수정(커밋 `d2f428d`)을 단독으로 다시 올리기** — `main@a2a44d4` 에 이것이 없어 `npm test` 가 지금도 그 검사에서 실패한다. `tools/smoke-test.mjs` 1파일, 프로덕션 코드 0개. **반드시 단독으로** — 직전 회차가 이것을 음수 duration 과 한 과제서에 묶었다가 no-change 로 끝났다. 음수 step duration 클램프는 **이번 회차의 1·2순위로 고르지 말 것**: 두 회차 연속 과제서로 나갔고 둘 다 no-change 였다.
