- 과제: 관리 화면이 오래된 트리 스냅샷을 통째로 덮어써 다른 관리자의 변경을 조용히 되돌리는 것을 409 로 막기 (가치 4 / 위험 2 / 작업량 M)
- 왜: `app/admin/page.tsx` 의 `onStatusChange`(285행)·`onUpdateActivity`(328행)는 SWR 스냅샷 `data?.activities` **전체 배열**을 PUT 하고, `route.ts:63` 의 `Array.isArray(body.activities)` 분기가 그것을 그대로 저장한다 — 그 사이 다른 관리자가 바꾼 모든 행이 조용히 옛 값으로 되돌아간다. 평소 창은 1초 폴링만큼 좁지만, 10-04 가 추가한 `admin-load-error`(GET 실패) 상태에서는 스냅샷이 임의로 오래되어도 저장 버튼이 그대로 동작하므로 되돌림 범위가 무제한이다.
- 수용 기준:
  1) 관리 화면의 전체 배열 PUT 2곳이 `expectedLastUpdated: data.lastUpdated` 를 함께 보낸다.
  2) 서버는 `body.activities` 배열 분기에서 `expectedLastUpdated` 가 **있고** 저장된 `currentData.lastUpdated` 와 다르면 아무것도 쓰지 않고 409 `{ error, code: 'STALE_SNAPSHOT' }` 를 돌려준다. 필드가 없으면 지금과 똑같이 동작한다(기존 e2e 보호).
  3) 409 를 받으면 `admin-save-error` 배너에 "다른 관리자가 먼저 바꿨습니다…" 가 뜨고, 서버 데이터는 **바뀌지 않는다**.
  4) 신규 e2e 가 손으로 만든 대역 없이 증명한다: GET 만 `page.route` 로 끊어 스냅샷을 고정 → 같은 컨텍스트 `page.request.put` 으로 "다른 관리자" 변경을 넣음 → UI 에서 상태 버튼 클릭 → 409 배너가 뜨고, 라우팅 해제 후 GET 으로 읽은 값이 "다른 관리자" 변경 그대로인지 확인.
  5) 기존 e2e 29건이 모두 통과한다(특히 `e2e/admin-save-failure.spec.ts:34,40` 과 `e2e/admin-put-contract.spec.ts` 의 `expectedLastUpdated` 없는 `{activities}` PUT 은 계속 200).
- 건드릴 파일:
  - `app/api/activities/route.ts:63` — `Array.isArray(body.activities)` 분기 **안에서만** `typeof body.expectedLastUpdated === 'string' && body.expectedLastUpdated !== currentData.lastUpdated` 이면 409 조기 반환. `matched` 플래그·검증 순서·`validateActivityImport`·`isAdminRequest`(49행)·GET 응답 모양은 손대지 말 것.
  - `app/admin/page.tsx:326` (`submitActivityChange({ activities, targetId: id, newStatus })`) 와 `:332` (`{ activities: newActivities, targetId: id }`) — `expectedLastUpdated: data?.lastUpdated` 추가. `submitActivityChange`(76행) 자체는 `res.ok` 가 false 면 서버 `error` 문구를 그대로 배너에 넣으므로 409 전용 분기는 필요 없다(401 분기만 별도). 추가 수정이 필요 없는지 76-103행을 먼저 읽고 판단할 것.
  - `e2e/admin-stale-overwrite.spec.ts` (신규) — 위 4). GET 만 가로채고 PUT 은 통과시키는 패턴은 `e2e/dashboard-load-failure.spec.ts:57-59` 에 이미 있다(이번 회차에 열어 확인): `await page.route('**/api/activities', async (route) => { if (route.request().method() !== 'GET') return route.fallback(); await route.fulfill({...}); })`. 로그인·백업/복원 헬퍼(`loginAsAdmin`, `beforeEach`/`afterEach` 의 `readData`→원복)도 같은 파일과 `e2e/admin-save-failure.spec.ts:30-42` 의 모양을 그대로 따를 것.
  - (선택) `docs/ADMIN_GUIDE.md` — 동시 편집 충돌 안내 1절.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm ci --legacy-peer-deps` (이 워크트리에 node_modules 없음, 수 분)
  - `npx tsc --noEmit`
  - `node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts` — **`npm run test:unit` 은 이 환경에서 `"lib/**/*.test.ts"` 글롭이 확장되지 않아 조용히 0건이거나 실패한다. 파일을 명시해 `# pass 95 / # fail 0` 을 확인할 것.**
  - `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e` (기존 29 + 신규 2~3)
  - `rm -rf playwright-report test-results` 후 `npm run lint` — **e2e 산출물이 ESLint 에 잡혀 257 errors 가 난다. `eslint.config.mjs` 의 `globalIgnores` 에 경로를 추가하는 접근은 09-29 에 머지되지 않았으므로(커밋 0de2f1e 는 main 의 조상이 아님) 다시 제출하지 말 것.**
  - `npm run build`
- 위험과 피할 것:
  - **409 검사는 반드시 opt-in** 으로. `e2e/admin-save-failure.spec.ts:33` 의 `beforeEach` 와 `:38-40` 의 `afterEach` 가 `expectedLastUpdated` 없이 `{activities: ...}` 를 PUT 해 200 을 기대한다(이번 회차에 열어 확인) — 필드 없는 요청의 동작을 바꾸면 여러 spec 의 셋업/복원이 통째로 깨진다. 업로드 라우트(`app/api/activities/import/route.ts`)는 건드리지 말 것.
  - `lib/types.ts:13-17` 의 `ActivitiesData` 에 `lastUpdated: string` 가 있음을 확인했다 — `data?.lastUpdated` 는 타입상 안전하다(`data` 가 undefined 인 최초 로드에는 필드를 보내지 않게 할 것).
  - `lastUpdated` 는 매 쓰기마다 `new Date().toISOString()` 로 갱신된다(route.ts:101) — 밀리초 해상도라 같은 밀리초 두 쓰기가 이론상 충돌을 못 잡는다. 이것까지 고치려 하지 말 것(범위 밖, 과제서에 기록).
  - 보호 경로 `app/api/auth/**`·`lib/adminSession.ts`·`proxy.ts`(CSP nonce)·`Dockerfile`·`amplify.yml` 은 건드리지 말 것.
  - 1초 SWR 폴링을 "저장 성공" 의 증거로 쓰지 말 것 — 판정은 PUT 응답과 라우팅 해제 후의 GET 으로만.
  - `onStatusChange` 의 클라이언트 전파 규칙(하위 전체 + 1단계 상위)은 ADMIN_GUIDE 4.3 과 일치하므로 **이번에 옮기거나 바꾸지 말 것**(별도 M 과제).
- 차선 후보: 존재하지 않는 `targetId` 의 `action:'move'`·`'delete'`·`targetId+newStatus` PUT 을 200 대신 400 으로 거부 (2/2/S). 이번 회차에 코드로 재확인함 — `moveActivity`(`lib/treeUtils.ts:27`)는 `targetIndex===-1` 이면 원본 반환, `deleteActivity`(`:91`)는 무변화, `route.ts:76` 의 `map` 은 아무 항목에도 안 맞는다 — 셋 다 새 `lastUpdated` 와 200 "저장 성공". 고치는 자리는 `route.ts:63-92` 한 곳(`updated.some(a => a.id === body.targetId)` 확인 후 404/400). `admin-put-contract.spec.ts` 의 `{dashboardTitle}` 단독 PUT 200 기대를 깨지 말 것.
