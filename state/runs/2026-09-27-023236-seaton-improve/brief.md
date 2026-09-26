# 과제서 — 2026-09-27 (base main@c0735bf)

- 과제: 좌석맵의 검색어·도면 선택을 주소에 반영해 새로고침·링크 공유에도 남게 하기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `SeatMapPage` 는 URL 쿼리를 **읽기만 한다**. `web/src/pages/SeatMapPage.tsx:324` 이 `const [searchParams] = useSearchParams()` 로 setter 를 아예 받지 않고, 읽는 자리는 세 곳뿐이다 — `:350`(`edit=1` → 초기 editMode), `:384`(`map` → loadBase 가 고를 도면), `:684`(`q` → 첫 검색). 그래서 화면에서 검색창에 이름을 넣거나(`:1530` `setQuery`, `:1529` form `onSubmit={search}`) 도면 버전을 바꿔도(`:1445` `chooseMap`) 주소는 `/` 그대로여서, 새로고침하면 검색 결과와 고른 도면이 사라지고 주소를 복사해 줘도 상대는 다른 화면을 본다. 반대 방향은 이미 만들어져 있다 — `MapsPage.tsx:432` 가 `/?map=${m.id}&edit=1`, `AppShell.tsx:506`·`EmployeesPage.tsx:404`·`DashboardPage.tsx:168` 가 `/?q=...` 로 들어오고, 지난 회차(0925145)가 로그인 리다이렉트에 `returnTo` 쿼리까지 보존하게 했다. 즉 깊은 링크를 **소비**하는 길은 다 났는데 좌석맵 화면에서 그 링크를 **만들** 방법이 없다.

- 수용 기준:
  1) 좌석맵에서 검색창에 이름·사번을 넣고 제출하면 주소가 `/?q=<입력값>` 으로 바뀐다(히스토리를 더럽히지 않도록 `replace`). 그 주소를 새로고침(또는 새 탭에서 열기)하면 검색창에 같은 말이 채워지고 같은 좌석이 선택·포커스된다.
  2) 검색을 지우면 `q` 가 주소에서 사라지고, 도면 버전을 바꾸면 `map` 이 그 도면 id 로 갱신된다. 이때 주소에 이미 있던 다른 쿼리(`edit=1` 등)는 지워지지 않는다.
  3) 검색이 **두 번 나가지 않는다** — 제출이 직접 부르는 `runSearch` 와 `:683` 의 `q` 감시 useEffect 가 같은 입력을 두 번 처리하면 안 된다(아래 위험 참고). 테스트가 이것을 증명해야 한다: 쿼리를 짓는 순수 규칙의 vitest 왕복(짓기→읽기)과, 실서버 E2E 의 `q` 왕복(제출 → 주소 확인 → reload → 검색 결과·선택 유지).

- 건드릴 파일 (프로덕션 2개 + 테스트 2개):
  - `web/src/lib/seatMapLink.ts` (신규) — 쿼리 규칙을 **한 곳에** 둔다. `readSeatMapParams(params: URLSearchParams): { mapId: string; query: string; edit: boolean }` 와 `writeSeatMapParams(prev: URLSearchParams, next: { mapId?: string; query?: string }): URLSearchParams`(빈 값이면 키 삭제, 모르는 키는 그대로 보존). 저장소의 반복된 교훈("같은 값을 두 경로가 따로 읽어 어긋난다", `web/src/lib/` 한 곳에 두고 양쪽이 import)을 그대로 따른다 — `silentSso.ts` 의 `loginPathFor`/`returnToFrom` 이 같은 꼴의 선례다.
  - `web/src/lib/seatMapLink.test.ts` (신규) — 위 두 함수의 왕복과 경계: 빈 검색어는 키 삭제, `edit=1`·미지의 키 보존, 공백만 있는 `q` 는 없는 것과 같음, `map` 갱신이 `q` 를 건드리지 않음.
  - `web/src/pages/SeatMapPage.tsx` — `:324` 를 `const [searchParams, setSearchParams] = useSearchParams()` 로 바꾸고, `:350`·`:384`·`:684` 의 `searchParams.get(...)` 을 `readSeatMapParams` 로 바꾼다. `search`(`:679`)가 `runSearch` 를 부른 뒤(또는 앞에) `setSearchParams(writeSeatMapParams(searchParams, { query }), { replace: true })`, `chooseMap`(`:636`)이 `setSearchParams(writeSeatMapParams(searchParams, { mapId: id }), { replace: true })`. `loadBase`(`:370`)는 그대로 둔다(마운트 1회, deps `[]`).
  - `web/e2e/seatmap.spec.ts` 에 spec 1~2건 추가(또는 `web/e2e/seat-deeplink.spec.ts` 신규) — 씨드 직원 이름으로 검색 → `expect(page).toHaveURL(/\?q=/)` → `page.reload()` → 검색 결과 목록과 좌석 선택이 그대로임을 확인. 이름은 `web/e2e/seed.mjs` 가 만든 직원을 쓰고, 로그인은 `web/e2e/helpers.ts` 의 기존 로그인 헬퍼를 쓴다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci && npm run lint && npm test && npm run build`  (`lint` 는 `tsc -b`, `test` 는 vitest — 현재 117건)
  - `gofmt -l . && go vet ./... && go test ./...`  (서버는 건드리지 않지만 회차 관례)
  - E2E: `docker build -t seaton:e2e .` → PostgreSQL 16 과 함께 기동 → `cd web && E2E_BASE_URL=http://127.0.0.1:18781 E2E_USERNAME=admin E2E_PASSWORD=... npx playwright test e2e/seatmap.spec.ts e2e/seat-detail.spec.ts e2e/login.spec.ts`
  - **역검증을 먼저**: 최근 성공 회차(09-23·09-24·09-26)의 공통 방식대로 변경 전 HEAD 이미지(`git archive` 로 구운 `seaton:e2e-before`)에서 새 spec 이 "주소가 `/?q=` 가 되지 않음" 으로 붉은 것을 확인한 뒤 수정본에서 초록을 확인한다. vitest 도 새 테스트가 헬퍼 부재로 먼저 붉은 것을 보고 구현한다.

- 위험과 피할 것:
  - **검색 두 번 나가는 것이 이 과제의 유일한 진짜 함정.** `:683` 의 useEffect 는 deps 에 `searchParams` 가 있고 `lastSearchRef.current = \`${mapId}:${term}\`` 로만 재실행을 막는다. 제출로 URL 에 `q` 를 쓰면 이 effect 가 다시 깨어나 같은 검색을 한 번 더 보내고 포커스를 다시 옮긴다. 제출 경로에서도 같은 키로 `lastSearchRef.current` 를 먼저 세워 effect 가 조용히 지나가게 하라(effect 안에서만 세우지 말 것).
  - `chooseMap` 은 `drop`·저장 뒤 새로고침 용도로도 불린다(`:708`, `:755`, `:766`, `:1270` — 모두 `chooseMap(mapId)`). 같은 id 로 다시 부를 때 `setSearchParams` 가 같은 값을 쓰는 것은 무해해야 한다(값이 바뀌지 않으면 쓰지 않는 편이 안전).
  - `edit` 은 **읽기만** 하고 쓰지 않는다(editMode 토글 `:1402` 은 URL 과 무관하게 둔다). 필터·색상 모드·확대 상태도 이번 범위 밖 — 범위를 넓히지 말 것.
  - 서버(`internal/app/`)와 보호 경로(`auth.go`, `migrations.sql`, `.github/workflows`)는 건드리지 않는다. 이 과제는 화면 안에서 끝난다.
  - 씨드는 건물·층·도면이 각 1개뿐이라(`web/e2e/seed.mjs:101-124`) 도면 버전 선택란은 `floorMaps.length > 1` 조건 때문에 화면에 나오지 않는다 → **도면(`map`) 쪽은 E2E 로 증명하지 말고 vitest 로 증명**하고, E2E 는 `q` 왕복만 본다. (씨드를 늘리려 하지 말 것 — 다른 spec 이 도면 수에 기대고 있을 수 있다.)
  - `helpers.ts:14` 는 로그인 뒤 경로가 `/login` 으로 시작하지 않을 때까지 기다린다. `/` 의 쿼리를 바꾸는 것은 여기에 영향이 없지만, 로그인 리다이렉트 규칙(`silentSso.ts`)은 손대지 말 것.

- 차선 후보: **E2E 환경 의존 spec 2건(mcp-oauth·tracking)의 준비 절차를 `web/e2e/README.md` 한 절로 고정** (2/1/S). 매 회차 "이 2건은 실패인가?" 를 다시 판별하는 비용이 반복되고 있다(09-26 회차가 변경 전 이미지에서도 같이 실패함을 다시 확인했다). 가짜 Keycloak IdP·`E2E_COLLECTOR_HOST`가 필요하다는 것을 spec 밖 문서로 남기면 다음 회차가 시간을 아낀다. 코드 위험 0.

## 추정 근거 (basis of estimate)
- 분해: 순수 헬퍼 2개 + 테스트(15분) / SeatMapPage 의 다섯 자리 연결(15분) / E2E 1~2건과 역검증 이미지 2회 빌드(20~40분, 도커 빌드가 지배).
- 범위: 45분 목표는 **vitest·tsc·build 까지**. Docker 역검증까지 포함하면 60~80분(8/10 신뢰). 도커 빌드 시간이 이 추정의 유일한 큰 변수다.
- 제외: 필터·색상모드·확대 상태의 URL 반영, `edit` 쓰기, 씨드 확장, 서버 변경, PDF 재생성(09-17 이후 관례대로 굽지 않음).
- 가장 크게 기대는 가정: `:683` useEffect 의 `lastSearchRef` 키 방식이 제출 경로에서도 재사용 가능하다는 것. 이 가정이 틀리면(예: effect 를 없애야 한다면) 작업량은 M 으로 오른다.
