# 과제서 — 2026-09-20 Momento

- 과제: 방문자 검색(UserExplorerPage)의 빈 결과가 "아직 데이터가 없습니다" 대신 실제 조회 기간과 함께 "검색 결과가 없습니다"를 보여준다 (가치 2 / 위험 1 / 작업량 S)

- 왜: `web/src/pages/UserExplorerPage.tsx:69` 의 검색은 `policyRange(90, site.max_exact_days)` 로 계산한 기간(기본 90일, 정책 상한이 더 짧으면 그 값)만 조회하는데, 결과가 0건이면 DataTable(`web/src/components/DataTable.tsx:194-202`)의 기본 Empty 가 `query` 없이 "아직 데이터가 없습니다 / 데이터가 생성되면 이 표에 표시됩니다"를 내보낸다 — 이 문구는 사이트에 데이터가 없다는 뜻으로 읽혀, 100일 전에만 활동한 사람을 찾다가 "수집이 안 되나?"로 오해한다(서버 응답 `visitor_trace.go:669` 부근은 `{"query","results"}` 만 주고 기간을 돌려주지 않으므로 화면이 알고 있는 값을 써야 한다). 고치면 사용자는 몇 일 안에서 못 찾은 것인지, 정책 상한 때문에 기간이 줄었는지를 바로 알고 "바로 추적"으로 넘어갈 수 있다.

- 수용 기준:
  1) 검색어 2자 이상, 응답 `results` 가 빈 배열이면 DataTable 대신 `Empty`(`web/src/components/States.tsx:194`)가 뜨고, 제목은 "검색 결과가 없습니다", 설명에는 실제 기간이 들어간다 — 예: `최근 90일 안에 "kim" 과 일치하는 방문자가 없습니다.` 정책 상한으로 줄었을 때는 `최근 60일(조회 정책 상한) 안에 …`.
  2) 결과가 있을 때 DataTable 의 `description`(현재 "최근 활동 순입니다. …", 173-174행)에도 같은 기간 문구가 앞에 붙는다(예: "최근 90일 안의 활동을 최근 순으로 보여줍니다. 행을 클릭하면 …"). 로딩·오류 분기는 그대로.
  3) 기간 문구는 순수 함수로 분리해 node:test 로 증명한다: `web/src/pages/visitorTrace.ts` 에 `export function searchWindowLabel(days: number, maxExactDays?: number): string` 을 두고(`policyRange` 를 그대로 재사용해 같은 값을 읽게 — 두 경로가 다른 숫자를 보이면 안 됨), 테스트는 (a) maxExactDays 없음/0 → "최근 90일", (b) maxExactDays=60 → "최근 60일(조회 정책 상한)", (c) maxExactDays=180 → "최근 90일"(상한이 더 크면 표기 없음). 빈 결과 문구 함수 `searchEmptyDescription(query, days, maxExactDays)` 도 같은 파일·같은 테스트에 두어 검색어가 문구에 들어가는지 확인.

- 건드릴 파일:
  - `web/src/pages/visitorTrace.ts` — `searchWindowLabel`, `searchEmptyDescription` 추가(순수 함수, React 없음; `policyRange` 는 `../components/queryError` 에서 import — visitorTrace.ts 가 이미 순수 모듈이라 test 에서 `.ts` 로 직접 import 됨. queryError.ts 도 순수 모듈이므로 import 가능; 순환 없는지 확인).
  - `web/src/pages/UserExplorerPage.tsx` — 165-176행 분기: `search.data && search.data.results.length === 0` 이면 `<Empty title="검색 결과가 없습니다" description={searchEmptyDescription(query, 90, site.max_exact_days)} />`, 아니면 DataTable 에 description 앞에 `searchWindowLabel(...)` 붙이기. `Empty` 는 `../components/States` 에서 import(같은 줄에 ErrorState 등 이미 있음). 90 이라는 숫자는 69행의 `policyRange(90, …)` 과 같은 상수로 뽑아 두 곳이 같은 값을 읽게 할 것(예: `const SEARCH_DAYS = 90`).
  - `web/test/visitorTrace.test.mjs` — 위 3+1 케이스 추가.

- 검증 명령(모두 `web/` 에서, CI 는 `npm ci && npm audit && npm run typecheck && npm test && npm run build` 와 `npm run lint`):
  - `cd web && npm ci`
  - `npx prettier --check src test`
  - `npm run lint`
  - `npm test` (현재 98건 통과 상태; 추가 뒤 100건 이상)
  - `npm run build` (`tsc -b && vite build`)
  - Go 쪽 변경 없음. 필요하면 `go vet ./cmd/... ./internal/...` 만.

- 위험과 피할 것:
  - 서버(`internal/httpapi/visitor_trace.go`)·마이그레이션·auth 는 건드리지 않는다. 응답에 `window` 를 추가하고 싶더라도 이번엔 화면만(응답 변경은 openapi 계약 테스트까지 번진다).
  - `DataTable.tsx` 의 Empty 문구를 고치지 말 것 — 다른 화면 전부가 그 기본 문구에 기대고 있고, 그 표는 `query`(표 내부 필터)로 분기하는 것이라 이 문제와 다른 축이다. 페이지 쪽에서 분기한다.
  - `policyRange` 의 동작을 바꾸지 말 것(다른 8개 화면이 쓴다). 문구 함수는 그 결과를 읽기만 한다.
  - 운영자 지침: 같은 값을 읽는 경로(요청 기간 69행 / 문구)는 반드시 같은 상수·같은 함수에서 나오게 하고, 실제 화면 분기(빈 결과 → Empty)가 렌더되는지 tsc 와 눈으로(가능하면 `npm run dev` 로 2자 검색 후 없는 이름) 확인. 소스 문자열 grep 을 증거로 삼지 말 것.
  - 미머지 브랜치 `auto/2026-09-18-1143`(MCP OAuth) 은 UserExplorerPage·visitorTrace.ts 를 만지지 않는 것으로 보이나 미확인 — 충돌 시 이 브랜치 쪽을 유지.
  - 커밋 메시지는 한국어 conventional 스타일(`fix(console): …` / `feat(web): …`).

- 차선 후보: 사용자 편집 다이얼로그가 호출자보다 높은 역할의 계정에 편집 버튼을 숨기고 권한 select 도 호출자 역할까지만 나열한다 (가치 2 / 위험 1 / S) — `web/src/pages/AdminPage.tsx` 사용자 표, 서버는 이미 403 ROLE_ABOVE_CALLER 를 주므로 화면만; 역할 서열 비교를 순수 함수로 빼서 node:test.
