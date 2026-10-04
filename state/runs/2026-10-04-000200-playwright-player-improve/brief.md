- 과제: run 결과의 음수 step duration 을 Playwright 값이 들어오는 세 경로 모두에서 클램프 (가치 4 / 위험 2 / 작업량 S)
- 왜: `GET /api/runs/:id` 가 Playwright 의 `step.duration` 을 그대로 통과시켜 `durationMs: -1868` 같은 음수를 API 로 내보낸다(직전 회차가 콜드 chromium 의 Before Hooks 구간에서 `-1868/-455/-457/-1644/-1646` 실측). `README.md:948` 은 "단조 시계로 측정하므로 음수가 되지 않습니다" 로 계약을 명시하고 `tools/smoke-test.mjs:2815` 가 이미 그 계약을 FAIL 로 잡아 릴리즈 게이트 `npm test` 를 떨어뜨린다. 고치면 문서화된 계약이 실제로 지켜지고 스위트 실패 1건이 사라져 CI smoke 잡을 게이트로 승격하는 데 한 칸 가까워진다.

- 수용 기준:
  1) 같은 실패 run(스모크의 `timeline` 스크립트, 콜드 chromium)에서 `durationMs` 가 음수인 step 이 하나도 없다.
  2) 양수 duration 은 값이 바뀌지 않고, 지금 `null`/`undefined` 가 나오는 자리는 그대로 `null`/`undefined` 다 — 전부 0 으로 뭉개는 변경은 불합격(`tools/smoke-test.mjs:2817` 이 "every step duration was null" 을 따로 잡는다).
  3) `tools/smoke-test.mjs` 의 `a run records per-step durations and the step that failed` 가 통과한다. **단정을 느슨하게 고쳐서 통과시키는 것은 불합격** — 프로덕션 쪽을 고칠 것.
  4) 세 경로가 같은 입력을 같은 값으로 읽는지 확인한다. 한 경로만 고치면 리포터 출력과 폴백이 같은 run 에 서로 다른 값을 낸다.

- 건드릴 파일 (프로덕션 1개 = `server.js`, 세 지점):
  - `server.js:1196` — `renderStepReporter(outputPath)` 가 **반환하는 템플릿 문자열 안**(확인함: 함수는 `server.js:1178` 에서 시작해 백틱 문자열을 리턴하고, 그 안의 `trim()` 이 step 트리를 매핑한다). 여기 코드는 별도 리포터 파일로 디스크에 써져 **다른 프로세스에서 실행되므로 서버 본체의 헬퍼를 호출할 수 없다** — 반드시 그 자리에 인라인으로 넣을 것(`typeof step.duration === "number" ? Math.max(step.duration, 0) : null` 꼴).
  - `server.js:1252` — `flattenReportSteps()`. 리포터 출력이 없는 옛 레코드용 폴백. 서버 본체라 헬퍼 사용 가능하지만, 한 줄이면 인라인도 무방. `undefined` 반환을 유지할 것(`cleanObject` 가 키를 떨어뜨리는 데 의존한다).
  - `server.js:2421` — 테스트 레벨 `durationMs: last?.duration ?? null`. 같은 클램프. `?? null` 유지.
  - 접근 선택(권장안): **음수 → 0 클램프**. 대안은 음수 → `null`("미측정" 으로 표기)인데, Playwright 는 끝나지 않은 step 에 `-1` 센티넬을 쓰므로 의미상은 `null` 쪽이 더 정직하다. 그러나 실측값이 `-1868` 처럼 센티넬이 아닌 큰 음수라 둘을 구분할 수 없고, README 계약("음수가 되지 않는다")과 스모크 단정 둘 다 0 클램프로 충족되며 "모든 duration 이 null" 회귀 위험도 없다. 그래서 0 클램프를 권한다. 선택 이유를 한 줄 주석으로 남길 것.

- 검증 명령:
  - `npm run check` (= `node --check server.js`). 의존성 없이 돌고 빠르다.
  - `npm test` (= `node tools/smoke-test.mjs`). **1회 약 6분.** 음수를 실제로 보려면 브라우저가 필요하다: 이 워크트리에는 `node_modules` 가 **없다(확인함)** — 없으면 `Cannot find module 'swagger-ui-dist/package.json'` 로 죽는다. 선행: `npm ci`(수 분) → `npx playwright install chromium`(~360MB) → `SMOKE_REQUIRE_BROWSER=1 npm test`. 과거 회차에서 `npm ci` 가 샌드박스 권한으로 막힌 적이 있으니 시간 예산을 먼저 떼어 둘 것.
  - 순서: **고치기 전에 한 번 돌려 실패 문자열(`negative step duration: [...]`)을 먼저 본 뒤** 고치고 다시 돌릴 것. 음수는 콜드 chromium 에서 잘 나오므로 첫 실행에서 보일 가능성이 높지만 확률적이다 — 첫 실행에 안 나오면 그 사실을 적고, 그때는 리포터 출력(run 디렉터리의 `steps.json`)에서 음수 원본을 직접 확인해 증거로 삼을 것.
  - 작업량 근거(basis): 코드 변경 3줄 + 주석 — 10분 안. 벽시계는 거의 전부 `npm ci` + chromium 설치 + 스위트 2회(6분×2)다. 총 35~55분으로 보고, `npm ci` 가 막히면 범위를 넘어간다(그 경우 `npm run check` 와 `steps.json` 증거까지만 남기고 보고할 것).

- 위험과 피할 것:
  - **스모크 단정을 지우거나 완화하지 말 것.** 과거 반려 사유다.
  - 녹화 패널 동기화 수정(`d2f428d`)·하네스 수정(`254a099`)·CI 워크플로(`5476435`)를 이번 과제에 **묶지 말 것**. 직전 회차가 앞의 둘을 한 과제서에 묶었다가 회차 결과 no-change 로 끝났다. 이번은 한 조각만.
  - `elapsedMs()` 로 계산되는 `durationMs`(server.js:2245/2256/4877/4888/5081/6325/6365/6378/6424/6447, 세션 타임라인 계열)는 단조 시계라 이미 계약을 지킨다 — **건드리지 말 것.** `server.js:4982` 의 `run.summary.stats.duration`(run 레벨, Playwright 출처)은 이번 범위 밖이며 음수가 나오는지 **미확인**이다.
  - 같은 값을 읽는 두 번째 소비자가 있다: `server.js:4998`(LLM 실패분석 evidence 가 저장된 run 레코드의 `step.durationMs` 를 되읽는다). 위 세 지점을 고치면 새 run 에서는 두 소비자가 같은 값을 본다. **이미 디스크에 저장된 옛 run 레코드의 음수는 그대로 남는다** — 마이그레이션은 하지 말고(이 저장소에 마이그레이션 관례 자체가 없다) 그 한계를 PR 설명에 적을 것.
  - 보호 경로(`principals.json` 권한, `secrets/` 스크러빙, docker entrypoint, `URL_ALLOWLIST`)는 건드리지 않는다.
  - grep 결과를 증거로 제출하지 말 것 — 증거는 스위트 출력(또는 `steps.json` 의 음수 원본)이다.

- 차선 후보: 검증된 녹화 패널 동기화 수정(커밋 `d2f428d`, 변경 파일 `tools/smoke-test.mjs` 1개)을 **단독으로** 다시 올리기. 이 워크트리에서 `git cat-file -t d2f428d` → `commit` 으로 존재를 확인했다(브랜치 `auto/2026-10-04-0307` 계열). 1순위가 재현되지 않거나 `npm ci` 가 막혀 음수를 관측할 수 없을 때 이것으로 전환할 것.
