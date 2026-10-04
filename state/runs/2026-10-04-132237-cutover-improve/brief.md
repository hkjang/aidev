- 과제: GET /api/activities 가 500 일 때 상황판이 "0% 완료 (0/0)" 빈 화면을 조용히 보여주는 것 수정 (가치 4 / 위험 2 / 작업량 M)
- 왜: 세 화면(`app/page.tsx:10`, `app/pc/page.tsx:9`, `app/admin/page.tsx:13`)이 각각 `const fetcher = (url) => fetch(url).then(r => r.json())` 로 **`r.ok` 를 검사하지 않는다**. GET 라우트는 읽기 실패 때 `{ activities: [], dashboardTitle, visitorCount: 0, lastUpdated, error: '저장된 상황판 데이터를 읽지 못했습니다.' }` 를 **500 과 함께** 돌려주는데(`app/api/activities/route.ts:33-44`), 이 본문은 정상 JSON 이라 파싱이 성공하므로 SWR 의 `error` 는 비고 `data` 가 그 빈 배열로 채워진다. `app/page.tsx:87` 과 `app/pc/page.tsx:80` 의 `if (error)` 분기는 **HTTP 오류에 대해 사실상 죽은 코드**이고, `data.error` 를 읽는 곳은 저장소 전체에 없다(`grep -rn "data\.error\|data?\.error" app components` 결과 0건). 결과적으로 파일 읽기 실패·권한 오류 때 전광판은 "등록된 작업 없음 / 0% 완료" 로 보이며, 모의훈련 상황실이 이것을 '아직 아무 작업도 등록되지 않았다' 와 구분할 방법이 없다.
- 수용 기준:
  1) GET 이 500 을 돌려주는 동안 `/`(표준)과 `/pc`(전광판)는 빈 트리·`0% 완료 (0/0)` 대신 **읽기 실패를 명시하는 화면 요소**를 보여준다.
  2) 한 번이라도 성공한 뒤 폴링이 500 으로 바뀐 경우에는 **마지막으로 성공한 트리를 계속 보여주면서** 상단에 "최신이 아닐 수 있음" 경고를 띄운다(훈련 중 화면을 통째로 비우지 않는다). 첫 로드부터 실패한 경우에만 전체 화면 오류 메시지를 띄운다.
  3) 관리자 화면(`/admin`)도 같은 읽기 실패를 화면에 남긴다(기존 저장 실패 배너 `admin-save-error` 와 구분되는 새 testid, 예: `admin-load-error`). 저장 실패 배너의 기존 동작·문구는 그대로 둔다.
  4) 테스트가 증명할 것: `page.route('**/api/activities', …)` 로 **GET 만** 500(위 실제 본문 모양 그대로)을 주입했을 때 ① 첫 로드 실패 시 오류 화면 ② 성공 후 실패 전환 시 직전 트리 유지 + 경고 배너 ③ 가로채기를 풀면(`page.unroute`) 경고가 사라지고 정상 트리 복귀. 손으로 만든 대역이나 소스 문자열 검사를 쓰지 말고 실제 컴포넌트·실제 fetch 경로에 주입한다(09-26 회차가 쓴 방식과 동일).
  5) 기존 e2e 25건이 그대로 통과한다.
- 건드릴 파일 (프로덕션 4개 + 테스트 1개):
  - `lib/fetchJson.ts` (신규) — `fetchActivityData(url)`: `res.ok` 가 아니면 응답 본문의 `error` 문구를 담아 throw. 세 화면이 공유한다. (`lib/` 의 다른 모듈처럼 `./types.ts` 식 `.ts` 확장자 import 관례를 따를 것. 단위 테스트를 붙일 거면 `lib/fetchJson.test.ts` 를 쓰지 말고 e2e 로만 증명해도 수용 기준은 충족된다 — `fetch` 를 대역으로 바꿔 증명하는 것은 금지.)
  - `app/page.tsx:10,33,87-99` — 지역 `fetcher` 제거하고 공유 함수 사용. `if (error)` 를 `error && !data`(전체 화면) / `error && data`(경고 배너 + 기존 트리)로 분리.
  - `app/pc/page.tsx:9,15,80-82` — 같은 변경. 전광판이라 배너는 상단 sticky 헤더 안쪽에 둘 것.
  - `app/admin/page.tsx:13,20,150` — `useSWR` 에서 `error` 도 받아 `admin-load-error` 배너 렌더. `submitActivityChange`·`saveError`·`handleUnauthorized` 는 손대지 않는다.
  - `e2e/dashboard-load-failure.spec.ts` (신규) — 수용 기준 4).
- 검증 명령 (이 저장소에서 실제로 쓰여 온 것):
  - `npx tsc --noEmit`
  - `npm run lint`  (※ `playwright-report/`·`test-results/` 는 09-29 이후 globalIgnores 에 있으므로 `rm -rf` 우회는 필요 없다)
  - 단위: `node --test lib/activityData.test.ts lib/adminSession.test.ts lib/treeUtils.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts` — 직전 회차 기록상 95 pass/0 fail. **`npm run test:unit` 은 쓰지 말 것**: `"lib/**/*.test.ts"` 글롭이 이 환경에서 확장되지 않아 `Could not find …` 를 찍고 exit 0 으로 끝난다(10-01·10-03 회차에서 재현).
  - e2e: `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e` — 기존 25 + 신규 3 = 28 기대.
  - `npm run build`  (프런트 3파일을 바꾸므로 생략하지 말 것)
  - ※ 이 정찰 세션에서는 샌드박스가 `node --test` 실행을 거부해 **단위 테스트를 직접 돌려 보지 못했다(미확인)**. 의존성 설치가 필요하면 `npm ci --legacy-peer-deps`(수 분).
- 위험과 피할 것:
  - **보호 경로 금지**: `app/api/auth/**`, `lib/adminSession.ts`, `proxy.ts`(CSP/nonce), `Dockerfile`, `amplify.yml` 는 이번 과제에 전혀 필요 없다. GET 라우트(`route.ts:27-46`)의 응답 모양도 **바꾸지 말 것** — 바꾸면 세 화면 전부와 e2e 가 같이 움직인다. 이번 변경은 전부 클라이언트 쪽이다.
  - **기존 PUT 실패 e2e 를 깨지 말 것**: `e2e/admin-save-failure.spec.ts:59,95` 와 `e2e/admin-empty-tree.spec.ts:74` 의 `page.route` 는 모두 `if (route.request().method() !== 'PUT') return route.fallback()` 로 **GET 을 통과시킨다** — 이번 회차에서 세 곳 모두 열어 확인했다. 따라서 fetcher 가 `res.ok` 를 보게 되어도 이 3개 테스트는 영향을 받지 않아야 한다. 받으면 그건 회귀다.
  - **빈 트리와 읽기 실패를 혼동하지 말 것**: `activities: []` 는 정상 상태이기도 하다(09-27 이 추가한 `empty-activity-tree` 안내·`add-root-activity` 버튼이 그것을 전제로 하고 e2e 가 고정하고 있다). 200 + 빈 배열은 지금처럼 빈 상태 안내로 남아야 하고, 새 오류 표시는 **HTTP 실패일 때만** 나와야 한다.
  - **폴링을 성공 판정에 쓰지 말 것**(과거 교훈): 배너 등장/소멸은 `expect(locator)` 로 기다리고, 저장값 확인은 `page.request` GET 으로 한다.
  - 세 화면이 같은 값을 읽는 세 경로이므로 **한쪽만 고치지 말 것**(운영자 지시). 한 공유 함수로 바꾸고 세 화면 모두에서 결과를 확인한다.
  - `data?.activities?.length > 0`(`app/page.tsx:92`)은 fetcher 가 `any` 를 돌려주기 때문에 통과하는 식이다. 공유 함수에 좁은 반환 타입(`ActivitiesData`)을 붙이면 여기서 tsc 오류가 날 수 있다 — 그러면 해당 식을 `(data?.activities?.length ?? 0) > 0` 로 함께 고치면 된다. 타입을 넓혀 피하지 말 것.
- 차선 후보: **존재하지 않는 `targetId` 의 move/delete/상태변경 PUT 을 200 대신 400 으로 거부** (가치 2 / 위험 2 / 작업량 S). `moveActivity` 는 `targetIndex === -1` 이면 원본을 그대로 반환하고(`lib/treeUtils.ts:27`), `deleteActivity` 는 아무것도 지우지 않으며, `targetId+newStatus` 분기(`app/api/activities/route.ts:75-78`)의 `map` 은 아무 항목에도 맞지 않는다 — 셋 다 새 `lastUpdated` 와 200 을 돌려주므로 `res.ok` 만 보는 관리 화면에 오류가 보이지 않는다(10-01 이 고친 fall-through 의 남은 구멍). 고치는 자리는 `route.ts:63-92` 한 곳: 세 분기에서 `updated.some(a => a.id === body.targetId)` 를 확인하고 미존재면 기존 400 과 같은 모양으로 거부. `{dashboardTitle}` 단독 PUT 의 200 기대(`e2e/admin-save-failure.spec.ts:169`)를 깨지 말 것.
