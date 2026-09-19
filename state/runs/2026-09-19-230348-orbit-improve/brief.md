# 과제서 (2026-09-19)

- 과제: Time Travel 슬라이더 — 드래그 중에는 화면만 바꾸고 요청은 놓을 때 한 번, 늦게 온 응답은 버린다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `web/src/pages/OrbitPage.tsx` 의 `TimeTravel` 은 MUI `Slider` 의 `onChange` 마다 부모의 `setTravelTo` 를 부르고, `load`(useCallback, deps `[travelTo]`)가 `useEffect` 로 매번 다시 돌아 드래그 한 칸마다 `/orbit?at=…` 과 `/rediscover` 두 요청이 나간다(1년치 슬라이더면 수백 요청). 게다가 응답이 순서대로 오지 않으면 먼저 보낸 요청의 응답이 나중에 도착해 마지막 날짜의 우주를 덮어써서, 슬라이더가 가리키는 날과 화면이 다른 상태로 남을 수 있다. 고치면 요청은 손을 뗄 때 한 번만 나가고, 어느 순서로 와도 마지막 요청의 결과만 화면에 남는다.

- 수용 기준:
  1) 마우스/터치로 슬라이더를 끄는 동안에는 `TimeTravel` 카드의 날짜 표시(`formatDate`)만 바뀌고 `/orbit`·`/rediscover` 요청은 나가지 않는다. 손을 떼는 순간(`onChangeCommitted`) 한 번만 부모 `onChange` 가 불려 요청이 한 번 나간다. 키보드(방향키)는 MUI 가 한 칸마다 `onChangeCommitted` 도 함께 부르므로 지금처럼 칸마다 요청되어도 된다(회귀 아님).
  2) `OrbitPage.load` 는 요청마다 순번을 매기고(useRef 카운터), 응답이 왔을 때 그 순번이 최신이 아니면 `setNodes` 등 상태를 건드리지 않는다. 오류도 마찬가지로 최신 요청의 오류만 보여 준다.
  3) 테스트가 증명할 것: (a) 순번 지킴이 순수 모듈 — 먼저 시작한 요청의 응답이 나중에 와도 무시되고 마지막 요청의 응답만 적용됨; (b) 날짜↔슬라이더 칸 변환 순수 함수 — 마지막 칸(오늘)은 `undefined`(현재), 그 전 칸은 ISO 문자열, 첫 칸은 `earliest` 와 같은 날; (c) `TimeTravel` 컴포넌트 — 드래그(`mouseDown`→`mouseMove`) 중에는 `onChange` 가 불리지 않고 `mouseUp` 뒤 한 번 불린다. (c) 가 jsdom 에서 MUI Slider 좌표 계산 때문에 성립하지 않으면(`getBoundingClientRect` 를 `{width:100,left:0,…}` 로 흉내 내는 MUI 공식 테스트 방식을 먼저 시도) 그 사실을 회차 노트에 적고 (a)(b) 로 대신한다.
  4) 기존 `vitest --run`(95개)·`tsc -b && vite build` 가 그대로 통과한다. 슬라이더의 시각·문구·"현재로" 단추 동작은 바뀌지 않는다.

- 건드릴 파일:
  - `web/src/timeTravel.ts` (신규, 순수 모듈 — 이 저장소 관례: `eclipse.ts`/`forecast.ts` 처럼 로직은 `.ts` 로 빼고 옆에 `.test.ts`):
    - `dayToTravelValue(startMs: number, days: number, day: number): string | undefined` — 지금 `TimeTravel` 의 `onChange` 안에 있는 계산(`start + day*86_400_000`, `dayOf(at) >= days ? undefined : at.toISOString()`)을 그대로 옮긴다.
    - `createLatestGuard()` → `{ next(): number; isCurrent(token: number): boolean }` — 요청 순번 지킴이.
  - `web/src/timeTravel.test.ts` (신규): 수용 기준 3(a)(b).
  - `web/src/pages/OrbitPage.tsx`:
    - `TimeTravel`(334행~): `const [draft, setDraft] = useState<number>()` 를 두고 `Slider` 의 `value={draft ?? dayOf(current)}`, `onChange={(_, day) => setDraft(day as number)}`, `onChangeCommitted={(_, day) => { setDraft(undefined); onChange(dayToTravelValue(start, days, day as number)); }}`. 제목의 날짜 표시(368행)는 `draft` 가 있으면 `start + draft*86_400_000` 의 날짜를 보여 준다. `value` prop 이 바뀌면(부모가 "현재로" 로 지움) `draft` 도 비운다.
    - `load`(43행~): `const requests = useRef(createLatestGuard())`, `const token = requests.current.next()` 를 맨 앞에, `await Promise.all` 뒤와 `catch` 안에서 `if (!requests.current.isCurrent(token)) return;`. `useRef` import 추가.
    - 선택: `TimeTravel` 을 `web/src/components/TimeTravel.tsx` 로 옮겨 export 하면 3(c) 컴포넌트 테스트를 쓸 수 있다(옮길 때 `formatDate` import 는 `../api`). 옮기지 않아도 (a)(b) 만으로 과제는 성립한다.

- 검증 명령 (이 워크트리에는 `web/node_modules` 가 없다 — 먼저 설치):
  - `cd web && npm ci --no-audit --no-fund`
  - `cd web && npx vitest --run` (기존 95개 + 신규)
  - `cd web && npm run build` (`tsc -b && vite build`)
  - `go test ./...` (프런트만 바꾸므로 그대로 통과해야 한다 — 이번 정찰에서 확인: 전부 ok)

- 위험과 피할 것:
  - `/rediscover` 는 `travelTo` 를 쓰지 않는데도 `load` 안에서 같이 다시 불린다. 이번엔 **분리하지 말 것** — 로딩·오류 처리(`ErrorView`/`LoadingView`)가 한 묶음이라 나누면 범위가 커진다. 차선 아이디어로 남긴다.
  - 서버(`internal/server/timetravel.go`, `data.go`)는 건드리지 않는다. auth/migrations/workflows 도 무관.
  - `earliest` 는 현재 화면(`/orbit` 무인자)에서만 오고 과거 화면 응답엔 없다(60~61행 주석). 순번 지킴이 때문에 이 값을 잃지 않게, `setEarliest` 도 최신 응답에서만 하되 `orbit.earliest_at` 이 있을 때만 갱신하는 지금 조건은 유지한다.
  - MUI `Slider` 의 `onChange` 시그니처는 `(event, value: number | number[], activeThumb)` — `as number` 캐스트를 그대로 쓴다. 제어 컴포넌트에서 `value` 를 `draft` 로 바꿔 끼우면 드래그 중 리렌더가 늘 수 있으나 `OrbitCanvas` 는 `nodes` 가 안 바뀌면 다시 계산하지 않으므로 괜찮다(`useMemo` 로 `eclipses` 만 걸려 있음 — 확인).
  - 뒤로 가기·"현재로" 단추(390행)는 부모 `onChange(undefined)` 를 직접 부르므로 `draft` 초기화 useEffect 없으면 슬라이더가 옛 칸에 남는다 — 위의 "value prop 바뀌면 draft 비움" 을 빠뜨리지 말 것.
  - 커밋 메시지는 이 저장소 관례대로 한국어 한 줄, `fix(orbit): …` 형식(예: `fix(orbit): Time Travel 슬라이더가 끄는 동안 우주를 거듭 부르던 문제`).

- 차선 후보: `orbitRange` 의 `people LEFT JOIN interactions` 데카르트 곱 제거 — `internal/server/timetravel.go:140` 의 SQL 을 `SELECT least((SELECT min(created_at) FROM people WHERE user_id=$1),(SELECT min(occurred_at) FROM interactions WHERE user_id=$1))` 로. `/orbit` 현재 화면을 부를 때마다(`data.go:740`) 도는 쿼리라 사람×교류 곱이 매 페이지 로드에 든다. 의미는 같다(least 는 NULL 을 무시; 교류는 사람에 딸려 있어 사람이 없으면 교류도 없다 — FK 는 미확인). main 에는 DB 종단 테스트 기반이 없어(`ORBIT_TEST_DATABASE_URL`/`newDBServer` 는 미머지 auto/2026-09-18-1033 에만 있음) `go vet`·`go test ./...` 통과와 SQL 문 정적 검토로 끝내야 한다 — 그래서 2순위.
