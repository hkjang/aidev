# 과제서 (2026-10-18)

- 과제: Lobby Dashboard 가 로비 현황 조회 실패를 「방문 예정 0 · 현재 방문중 0 · 표시할 방문자가 없습니다」로 단정하는 것 닫기 (가치 4 / 위험 2 / 작업량 S)

- 왜: `web/src/pages/LobbyPage.tsx:20` 의 `data` 가 `{ counts: {}, items: [] }` 로 시작하고 `load` 의 `catch` 는 `setError(...)` 만 하므로 `/api/v1/lobby/today` 가 한 번도 성공하지 못한 상태에서 실패하면 화면은 네 MetricCard 를 `방문 예정 0 · 현재 방문중 0 · 퇴실 완료 0 · 미방문 0` 으로, 탭을 `현재 방문자 0` 으로, 목록을 `표시할 방문자가 없습니다.` 로 렌더링한다 — 조회가 끊긴 장애가 「아무도 안 왔고 건물에 아무도 없다」는 사실 주장으로 보이고, 그중 「현재 방문중」은 화면이 스스로 `재난 시 체류 기준 인원` 이라고 적어 둔 값이다(47행). 둘째로 한 번 성공한 뒤의 재조회 실패는 `setData` 를 부르지 않으므로 **옛 숫자가 그대로 남는데**, 그 사실을 알리는 것은 닫을 수 있는(`onClose`) error Alert 하나뿐이어서 닫으면 화면에는 최신인 척하는 숫자만 남는다. SSE(`/api/v1/lobby/stream`)가 이벤트마다 재조회를 걸므로 이 상태는 오래 유지된다. 고치면 로비 담당자가 하루 종일 보는 화면에서 「실시간 / 오래된 값 / 조회 실패」가 서로 구별된다.
- 같은 결함 유형을 v2.8.16(RosterPage)·v2.8.19(AdminPage 「운영 지표」)에서 이미 닫았고 둘 다 머지됐다. 이번은 **그 사이에 남은 같은 화면군의 마지막 큰 구멍**이다. AdminPage 상단 네 MetricCard(차선 후보)보다 가치가 높다고 판단한 근거: LobbyPage 는 로비 상주 인력의 주 화면이고, 숫자 옆 helper 문구가 그 값을 재난 대응 기준이라고 스스로 선언한다.

- 수용 기준:
  1) `/api/v1/lobby/today` 가 **한 번도 성공하지 못한 채** 실패하면 네 MetricCard 자리에 숫자가 단 하나도 렌더링되지 않고(`0` 도 `-` 도 없음), 대신 「로비 현황을 불러오지 못했습니다 … 체류 인원이 0명이라고 판단하지 마세요」류의 한국어 경고 한 개가 그 자리를 **치환**한다. 「현재 방문자」 탭 라벨에도 숫자가 붙지 않는다.
  2) 같은 실패 상태에서 목록 영역이 `표시할 방문자가 없습니다.` 라고 말하지 않는다(= 빈 목록과 조회 실패가 구별된다).
  3) 한 번 성공한 뒤의 재조회 실패에서는 받아 둔 숫자를 계속 보여 주되, **닫을 수 없는** 표시로 그 값이 마지막 성공 조회의 것임을 밝힌다(예: 기준 시각 + 「최신이 아닐 수 있습니다」). 즉 실시간 / 오래된 값 / 전무가 서로 다른 화면을 만든다.
  4) 정상 200 응답이 **진짜로** 0 을 담은 신규/한가한 사업장에서는 네 카드가 `0` 을 보여 주고 어떤 실패 문구도 뜨지 않는다(거짓 양성 가드). 값이 0 이라는 사실은 실패 근거로 쓰지 않는다.
  5) 테스트가 증명할 것 — 단위 테스트(vitest)가 `loading / failed(캐시 없음) / failed(캐시 있음) / ready(정상 0)` 네 상태가 서로 다른 표시값을 낳는 것을, Playwright 2개가 ① 실패 시 숫자 부재 + 경고 존재 + `표시할 방문자가 없습니다.` 부재 + `page.on("pageerror")` 0건, ② 정상 경로에서 네 카드가 숫자를 보여 주고 실패 문구가 없는 것을 실제 번들·실서버·실제 Chrome 에서 고정한다.

- 건드릴 파일 (프로덕션 2개):
  - `web/src/lobbyCounts.ts` — **신규 순수 모듈**. `web/src/adminMetrics.ts`(이미 머지된 선례)와 같은 모양으로 쓸 것:
    - `export type LobbyToday = { counts: Record<string, number>; items: LobbyVisitor[]; truncated?: boolean; limit?: number }`
    - `export type LobbyLoad = { state: "loading" } | { state: "failed"; message: string; last: LobbyToday | null; lastAt: string } | { state: "ready"; value: LobbyToday }`
    - `export function lobbyPanel(load: LobbyLoad)` → `{ summary: { kind: "notice"; severity: "info" | "warning"; message: string } | { kind: "counts"; counts: Record<string, number>; staleNote?: string }, items: LobbyVisitor[], emptyText: string, currentTabLabel: string }`
    - `adminMetrics.ts` 의 `withCause(sentence, errorMessage)` 와 **같은 순서**를 지킬 것: 한국어 문장이 먼저, 원인은 괄호 뒤(네트워크 실패의 `e.message` 는 브라우저의 영어 "Failed to fetch" 다). 함수를 복사해 오거나 그대로 재구현해도 되고, `adminMetrics.ts` 를 import 하지는 말 것(두 화면이 한 모듈에 묶이면 다음 회차의 수정 범위가 넓어진다).
  - `web/src/pages/LobbyPage.tsx` — `LobbyPage()`:
    - 20행 `const [data, setData] = useState<{ counts: ...; items: ... }>({ counts: {}, items: [] })` → `useState<LobbyLoad>({ state: "loading" })`. 이 교체가 핵심이다: `?? 0` 을 **타입으로 표현 불가능**하게 만들어 `tsc` 가 모든 독자(47행 네 카드, 48행 탭 라벨·`data.truncated`·`data.limit`·`data.items`·빈 상태 문장)에 처리를 강제한다. v2.8.18(KeysPage)·v2.8.19(AdminPage)에서 효과가 확인된 방법이다.
    - 28행 `load` 의 `try` 성공 경로 → `setData({ state: "ready", value })`; `catch` → `setData((prev) => ({ state: "failed", message: ..., last: prev.state === "ready" ? prev.value : prev.state === "failed" ? prev.last : null, lastAt: ... }))`. 마지막 성공 시각은 `ready` 로 바뀔 때 함께 기록할 것.
    - 47·48행 렌더 → `lobbyPanel(data)` 한 값만 읽게 한다. 실패(캐시 없음)면 `<Grid>` 네 카드 전체를 `Alert severity="warning"` 하나로 치환한다(v2.8.19 가 「운영 지표」에서 같은 선택을 했고, 라벨만 남겨 `-` 를 뿌리는 쪽은 "숫자를 말하지 않는다" 를 라벨별로 지켜야 해서 더 약하다).
    - 기존 `error` state 는 **행동 실패**(직접 체크인·퇴실)용으로 그대로 두고, 조회 실패 메시지는 새 판별 값에서만 나오게 할 것. 조회 실패를 양쪽에 중복 표시하면 수용 기준 1 의 "경고 한 개" 가 깨진다.
  - 테스트: `web/src/lobbyCounts.test.ts`(신규, vitest), `web/e2e/visit-flow.spec.ts`(새 `test.describe` 하나에 스펙 2개 추가 — 기존 describe 를 고치지 말 것).
  - 선택: `docs/USER_GUIDE.md` 또는 `docs/ADMIN_GUIDE.md` 에 한 문장. **PDF 재생성은 하지 말 것**(이 저장소의 관례).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build` — `lint` 는 `tsc -b`, `test` 는 `vitest run`. 기준값: **103 passed**(v2.8.19 기록). 새 단위 테스트만큼 늘어야 한다.
  - `bash scripts/local-e2e.sh` — Docker·Go·Node·npm·curl 필요. 실제 `dist` 임베드 + 새 PostgreSQL + 실서버 + 실제 Chrome. 기준값: **28 passed**(`grep -c "test("` 는 27 로 나오는데 `visit-flow.spec.ts:452` 의 `for (const path of [...])` 가 한 `test(` 를 두 번 등록하기 때문이다 — 기록이 맞고 grep 이 과소 집계다). 수분 소요.
  - 수정 **전** 번들(제품 파일만 `git show HEAD` 로 되돌린 상태 + 새 스펙)에 먼저 돌려 실패를 눈으로 받을 것. `cmd/visitflow/webdist/` 복원은 `local-e2e.sh` 의 임시 백업이나 `/tmp` 복사본으로 하고 **`git checkout --` 는 쓰지 말 것**(과거 미커밋 작업 유실 기록).
  - `go test ./... -count=1`(서버 미변경 확인용, `VISITFLOW_TEST_DSN` 미설정이면 PostgreSQL 통합은 SKIP), `go vet ./...`, `gofmt -l .`(빈 출력), `git diff --check`(빈 출력).
  - e2e 실패 재현 방법: `await page.route("**/api/v1/lobby/today*", (route) => route.abort())` 후 `page.goto("/lobby")`. `web/public/sw.js:37` 은 `/api/v1/lobby/roster` 외의 `/api/` 를 캐시하지 않으므로 `serviceWorkers: "block"` 은 필요 없다(선례 2건에서 확인). 쿼리스트링이 붙으므로 글롭에 `*` 를 빠뜨리지 말 것. 수용 기준 3 은 성공 1회 뒤 `page.route` 를 설치하고 SSE 또는 탭/검색 변경으로 재조회를 유발해 확인한다.

- 위험과 피할 것:
  - **같이 건드리지 말 것**: `web/src/api.ts`(공용 fetch/CSRF), `web/public/sw.js`, `internal/app/*`(서버·`/api/v1/lobby/today` 계약), `web/src/adminMetrics.ts`·`web/src/roster.ts`(이미 닫힌 선례), 마이그레이션·auth/세션.
  - `LobbyPage` 의 **EventSource 수명주기와 `loadRef` 패턴**(36·37행)은 손대지 말 것 — 주석이 "필터가 바뀔 때마다 재구독하면 SSE 가 끊긴다" 는 과거 결함을 기록해 두었다. `connected` 표시 문구(`재연결 중 · 화면 조회는 정상`)도 이번 범위 밖으로 둔다(SSE 상태에 대한 말이고, 함께 고치면 파일 하나에 두 과제가 섞인다).
  - 26행 `api<ReferenceData>("/api/v1/reference-data").then(setReference).catch(() => undefined)` 의 조용한 실패는 **별도 보류 아이디어**다(로비 필터 드롭다운에만 영향). 이번에 합치지 말 것.
  - `data.truncated` / `data.limit ?? 300` 의 「표시 한도 초과」 info Alert 은 `ready` 일 때만 뜨게 할 것 — 실패 상태에서 `truncated` 를 읽으면 또 하나의 근거 없는 주장이 된다.
  - 값이 `0` 이라는 사실을 실패 근거로 쓰지 말 것(수용 기준 4). 신규 사업장·한가한 시간대에는 네 값 모두 정상적으로 0 이다.
  - 테스트 함정: `web/vite.config.ts:10` 이 `src/**/*.test.ts` 만 수집하므로 `.tsx` 테스트는 조용히 0개로 집계된다 → 판정 로직을 `.ts` 모듈로 빼야 vitest 가 덮는다. `web/e2e` 는 `npm run lint`(tsc -b) 범위 밖이므로 필요하면 `./node_modules/.bin/tsc --noEmit` 로 따로 확인할 것(보류 아이디어로 남아 있는 배선 변경은 이번에 하지 말 것).
  - 기존 28개 스펙 중 `/lobby`(Lobby Dashboard 본체)로 가는 것은 **없다**(`/lobby/scan`·`/lobby/roster`·`/lobby/walk-in` 만 있다, `visit-flow.spec.ts` 확인) → locator 충돌 걱정은 없지만, `getByText("현재 방문중")` 같은 선택자는 helper 문구·탭 라벨과 겹칠 수 있으니 카드 컨테이너로 한 번 좁힐 것.
  - 미확인: 실패 시 `0` 이 그대로 그려지는 것을 **실제 브라우저에서 아직 눈으로 보지는 않았다**(이 과제서는 20·28·47·48행 코드 근거로만 썼다). 구현자가 수정 전 번들에 새 스펙을 먼저 돌려 확인할 것.

- 차선 후보: **관리 Dashboard 상단 네 MetricCard 와 「운영 준비도」가 `/api/v1/admin/dashboard` 실패를 `0`·전진한 진척 막대로 단정하는 것 닫기** (가치 2 / 위험 1 / 작업량 S). `web/src/pages/AdminPage.tsx:28` 의 `counts` 가 `{}` 로 시작하고 33행이 `counts.today ?? 0`·`counts.pendingApproval ?? 0`·`counts.watchlist ?? 0` 을 읽는다. 특히 「운영 준비도」의 `LinearProgress value={Math.max(8, 100 - Number(value) * 8)}` 는 조회 실패에서 **100% 에 가까운 "모두 해소됨" 막대**를 그린다. 같은 파일의 `metrics` 는 v2.8.19 에서 이미 `MetricsState` 판별 값으로 고쳤으므로, 같은 `Dashboard` 함수 안에서 `counts` 에 같은 패턴을 적용하고 `web/src/adminMetrics.ts` 에 변환 함수를 덧붙이면 프로덕션 2개 파일로 끝난다. 1순위보다 약한 이유: 상단 공용 error Alert(23행)이 실패를 함께 보고한다.
