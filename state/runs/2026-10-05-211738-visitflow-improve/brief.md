# 과제서 (2026-10-05 정찰)

- 과제: 비상 대피 명단이 「조회 실패」를 「체류자 0명」으로 단정하고, 그 경고를 인쇄물에서 빼는 것 닫기 (가치 4 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/RosterPage.tsx` 는 한 번도 명단을 받지 못한 상태(`roster === null`)에서도 기준 시각 `-`, 칩 `총 0명`, 본문 `현재 사내에 체류 중인 방문자가 없습니다.` 를 그대로 렌더링하고, 동시에 경고에는 `… · 마지막으로 받은 명단을 표시합니다.` 라고 **없는 명단을 표시한다고 적는다**(57행·65행·68행·86행). 대피 상황에서 「건물에 0명」과 「몇 명인지 모른다」는 전혀 다른 정보다. 게다가 그 경고 두 줄은 `@media print { display: none }` 박스 안(54~59행)에 있어 **인쇄해서 들고 나가는 종이에는 오래된/없는 명단이라는 표시가 전혀 남지 않는다** — 이 화면의 존재 이유가 인쇄인데(16~18행 주석, `window.print()` 버튼) 종이가 자기 신뢰도를 말하지 못한다.

- 수용 기준:
  1) 캐시(localStorage `visitflow_last_roster`·서비스워커 ROSTER_CACHE)가 **없는** 상태에서 `/api/v1/lobby/roster` 가 실패하면 화면이 숫자를 단정하지 않는다: 칩이 `총 0명` 이 아니고(예: `총 -명` 또는 `명단 확인 불가`), 본문이 `현재 사내에 체류 중인 방문자가 없습니다.` 가 **아니라** 불러오지 못했다는 문장이며, 경고문에 `마지막으로 받은 명단을 표시합니다` 가 없다(없는 것을 표시한다고 말하지 않는다).
  2) 캐시가 **있는** 상태에서 조회가 실패하면 지금 동작을 유지한다: 마지막 명단 행과 그 `기준 시각` 을 그대로 보여 주고 「마지막으로 받은 명단」경고를 띄운다. 서비스워커가 주는 `offline: true` 응답(web/public/sw.js:37)도 지금처럼 「오프라인 상태입니다」 경고를 띄운다 — 세 상태(실시간 / 캐시 / 전무)가 서로 구별된다.
  3) 실패·오프라인 경고가 **인쇄 미디어에서도 보인다**: `page.emulateMedia({ media: "print" })` 에서 경고 문구가 `toBeVisible()` 이고, 정상 조회일 때는 경고가 아예 없다. 「새로고침」·「인쇄」 버튼과 `PageHeader` 는 지금처럼 인쇄에서 숨는다.
  4) vitest 가 순수 함수로 네 경우(실시간 / 실패+캐시 있음 / 실패+캐시 없음 / `offline:true`)를 구별함을 고정하고, **수정 전에 빨강**(새 모듈이 없어 import 실패)에서 초록으로 간다. 기준 88개 → 추가.
  5) `bash scripts/local-e2e.sh` 가 전부 통과하고(기준 14개 + 새 스펙), 기존 스펙 `prints the emergency roster with the current headcount`(web/e2e/visit-flow.spec.ts:350)도 계속 통과한다. 새 스펙은 수정 전에 1 failed 로 증상을 먼저 보여 준다.

- 건드릴 파일 (프로덕션 2개 + 테스트 2개):
  - `web/src/roster.ts` (신규, 프로덕션): 순수 함수 하나 — 예 `rosterStatus({ roster, failed, errorMessage })` → `{ source: "live" | "cache" | "none", severity, warning, countLabel, emptyText }`. JSX·MUI·window 를 쓰지 않는 데이터 전용 모듈(`schedule.ts`·`visitors.ts` 관례 그대로).
  - `web/src/pages/RosterPage.tsx:RosterPage` — 지금 흩어진 `error`/`stale` 두 불리언의 **표시 판단**을 위 함수 한 값으로 모으고, 칩(68행)·기준 시각(65행)·빈 상태 문장(86행)·경고(57~58행)가 모두 그 한 값을 읽게 한다. 경고 블록을 `@media print { display:none }` 박스(54행) **밖으로** 옮겨 `source !== "live"` 일 때 인쇄에도 남게 한다(`Alert` 의 배경색이 인쇄에서 빠질 수 있으니 문구 자체로 읽히게 쓸 것). `load`(32행)의 성공 경로·localStorage 키·60초 폴링은 그대로 둔다.
  - `web/src/roster.test.ts` (신규, 테스트): 수용 기준 4.
  - `web/e2e/visit-flow.spec.ts` (테스트): 영구 스펙 1~2개. 서비스워커가 끼어들지 않게 **새 `test.describe` 블록 안에 `test.use({ serviceWorkers: "block" })`** 를 두고(기존 describe 에 넣지 말 것 — 같은 블록의 다른 스펙까지 바뀐다), `await page.route("**/api/v1/lobby/roster", (route) => route.abort())` 로 실제 네트워크 실패를 주입한 뒤 `/lobby/roster` 를 연다. `login(page)`(6행)·`page.emulateMedia` 를 쓴다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build` (lint=`tsc -b`, test=vitest run, 기준 88개)
  - `bash scripts/local-e2e.sh` (실제 dist 임베드 + 새 PostgreSQL + 실제 Chromium, 수 분)
  - `go build ./...` · `go vet ./...` · `gofmt -l .`(빈 출력) · `git diff --check` — Go 는 건드리지 않지만 webdist 임베드 때문에 빌드 확인
  - 끝난 뒤 `git status --porcelain` 이 의도한 4개 파일만 보이고 `cmd/visitflow/webdist/` 에 추적된 스텁 `index.html` 하나만 남는 것 확인(복원은 임시 복사본, `git checkout --` 금지)

- 위험과 피할 것:
  - **`web/public/sw.js` 를 건드리지 말 것.** `offline: true` 는 서버가 아니라 서비스워커가 붙인다(sw.js:37; `internal/app/kiosk.go:210 emergencyRoster` 는 `generatedAt`/`count`/`items` 만 보낸다 — 확인). 즉 `stale` 분기는 죽은 코드가 아니다. 서버에 `offline` 을 추가하지 말고, 화면이 두 출처를 **같은 값으로 읽게** 만드는 것만 한다.
  - `internal/app/kiosk.go`·`server.go`·`web/src/api.ts` 는 건드리지 말 것(api.ts 는 전 화면 공용 fetch·CSRF 경로다). 서버는 지금 옳다.
  - 인쇄 영역을 넓히되 `PageHeader` 와 버튼 두 개는 계속 숨겨야 한다(종이에 「새로고침」 버튼이 찍히면 수용 기준 3 위반).
  - localStorage 캐시 키와 저장 시점을 바꾸지 말 것 — 서비스워커 캐시와 함께 이 화면의 오프라인 계약이다.
  - `web/vite.config.ts:10` 의 `include: ["src/**/*.test.ts"]` 때문에 `.tsx` 테스트는 **조용히 0개로 수집**된다. 컴포넌트 테스트를 쓰려 하지 말고 순수 `.ts` + e2e 로 나눌 것(testing-library 의존성 없음).
  - 손으로 만든 대역으로 증명하지 말 것: 실제 로그인·실서버·배포 번들·실제 Chromium 에서 **수정 전 1 failed → 수정 후 통과** 를 짝지어 남길 것.
  - **미확인**: ① Playwright 1.56 에서 서비스워커가 등록된 상태로 `page.route` 가 SW 발 요청을 가로채는지 확인하지 않았다 — 그래서 `serviceWorkers: "block"` 을 권한다. 막아도 증상이 재현되지 않으면 `route.fulfill({ status: 500 })`(서버 500 은 sw.js 가 그대로 통과시키므로 실제 도달 가능한 상태다)로 바꿔 보고 실제 관찰값을 기록할 것. ② 인쇄 미디어에서 MUI `Alert` 가 어떻게 렌더되는지 실측하지 않았다. ③ 문구는 제안이며 최종 한국어는 구현자가 화면에서 읽히는 쪽으로 정할 것.

- 차선 후보: `web/e2e` 를 정규 타입 검사에 넣기 — `web/tsconfig.app.json` 의 `include: ["src"]`, `tsconfig.node.json` 은 `vite.config.ts` 뿐이라 e2e 스펙의 타입 오류는 Playwright 의 transpile-only 실행에서만 드러난다(가치 2 / 위험 2 / S). 별도 `tsconfig.e2e.json` + `npm run lint` 참조 추가로 끝내고, 릴리즈·Docker 경로는 건드리지 말 것.
