# 과제서 (2026-09-27, 정찰)

- 과제: 관리자 화면 활동 목록에 항상 보이는 "최상위 작업 추가" 진입점과 빈 상태 안내 추가 (가치 3 / 위험 1 / 작업량 S)

- 왜: `app/admin/page.tsx:257-325` 은 `tree.map(node => <ActivityTree .../>)` 만 렌더하고, 최상위(parentId: null) 추가 버튼은 `components/ActivityTree.tsx:111-115` 의 `isRoot && ...` 안에만 있다. 따라서 `data.activities` 가 빈 배열이면 관리 화면에 작업 추가 수단이 하나도 없고(빈 흰 카드만 보임), JSON 업로드 말고는 복구할 방법이 없다 — 초기 설치 직후와 전체 삭제 후가 정확히 이 상태다. API 는 이미 `action:'add', parentId:null` 을 지원하므로(`app/api/activities/route.ts:62` → `lib/treeUtils.ts:47 addActivity`) 화면 쪽 진입점만 붙이면 끝난다.

- 수용 기준:
  1) 관리자로 로그인한 `/admin` 에서 activities 가 비어 있어도 최상위 작업 추가 버튼이 보이고, 누르면 `PUT /api/activities {action:'add', parentId:null}` 이 200 으로 돌아오며 `GET /api/activities` 의 activities 에 `parentId: null, level: 1, title: "새 액티비티", status: "대기"` 항목이 1건 생긴다.
  2) 트리가 비어 있을 때 "등록된 작업이 없습니다" 류의 빈 상태 안내 문구가 보이고, 트리가 비어 있지 않으면 그 문구는 보이지 않는다(추가 버튼은 두 경우 모두 보인다).
  3) 추가 PUT 이 실패하면(500 을 `page.route` 로 주입) 기존 `data-testid="admin-save-error"` 배너에 실패가 표시되고, 화면에 성공한 것처럼 항목이 늘어나지 않는다 — 즉 기존 `submitActivityChange` 를 그대로 재사용해야 한다.
  4) 테스트가 증명할 것: 빈 트리에서 시작해 화면 조작만으로 최상위 작업을 만들 수 있다는 것(현재는 클릭할 버튼 자체가 없어 반드시 실패한다). 구현 전에 먼저 e2e 를 써서 실패를 확인할 것(TDD).

- 건드릴 파일 (프로덕션 2개 이하):
  - `app/admin/page.tsx` — `tree.map` 을 감싼 `<div className="space-y-1 bg-white rounded-3xl shadow-sm p-2">`(257행) 안/아래에 최상위 추가 버튼 1개 추가. 핸들러는 새로 만들지 말고 기존 `submitActivityChange({ action: 'add', parentId: null })`(77행)을 그대로 호출한다. `tree.length === 0` 일 때 빈 상태 문구를 함께 렌더. 버튼에 `data-testid="add-root-activity"` 를 붙여 e2e 로케이터로 쓸 것(`ActivityTree` 안의 `title="같은 레벨 추가"` 버튼과 이름이 겹치지 않게).
  - (선택) `components/ActivityTree.tsx` — 최상위 추가가 목록 바깥으로 올라오면 `isRoot && title="같은 레벨 추가"` 버튼(111-115행)은 중복이다. 지워도 되고 남겨도 되지만, **지운다면 기존 e2e 가 그 버튼을 쓰지 않는지 먼저 확인**할 것. 확실하지 않으면 이번 회차에는 건드리지 말 것(수용 기준에 없음).
  - `e2e/tree-ops.spec.ts` 또는 신규 `e2e/admin-empty-tree.spec.ts` — 아래 "테스트 작성 요령" 참고.
  - (선택) `docs/ADMIN_GUIDE.md` — 작업 추가 절차를 설명한 절이 있으면 한 줄 갱신. 본문 미확인이므로 있을 때만.

- 테스트 작성 요령 (이 저장소의 실제 관례, `e2e/admin-save-failure.spec.ts:29-40` 과 `e2e/tree-ops.spec.ts:33-43` 확인함):
  - `test.beforeEach` 에서 `original = await readData(request)` 로 현재 데이터를 저장하고 `loginAsAdmin(request)` 후 `PUT {activities: []}` 로 빈 트리를 만든다. `test.afterEach` 에서 `original` 을 되돌린다. `validateActivityImport` 는 빈 배열을 통과시킨다(`lib/activityData.ts:139-143`, 배열이기만 하면 됨).
  - 화면 로그인은 `page.goto('/admin')` → `page.getByPlaceholder('관리자 비밀번호').fill(...)` → 로그인 버튼(`tree-ops.spec.ts:57-59` 와 같은 방식).
  - 실패 주입은 손으로 만든 대역이 아니라 `page.route('**/api/activities', ...)` 로 실제 fetch 경로에 500 을 넣을 것(`admin-save-failure.spec.ts` 가 쓰는 방식).
  - **성공 판정에 SWR 1초 폴링을 쓰지 말 것.** PUT 응답(`page.waitForResponse`) 또는 `request.get('/api/activities')` 결과로 판정한다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  npm ci --legacy-peer-deps        # node_modules 없을 때만. 수 분 걸림
  npm run lint
  npx tsc --noEmit
  npm run test:unit                # node --test "lib/**/*.test.ts" — 현재 79건/25 suites
  rm -rf playwright-report test-results   # lint 전에 반드시. 안 지우면 lint 가 trace 뷰어 번들을 검사해 수천 건을 냄
  PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e   # 현재 16건
  npm run build
  ```
  (이번 정찰에서는 `node_modules` 가 없어 위 명령을 **하나도 실행하지 못했다 — 미확인**. 회차 기록상 2026-09-26 에는 모두 통과했다.)

- 위험과 피할 것:
  - 보호 경로를 건드리지 말 것: `app/api/auth/**`, `lib/adminSession.ts`, `proxy.ts`, `Dockerfile`, `amplify.yml`, `lib/mail/*`. 이번 과제는 `app/admin/page.tsx` 한 파일이면 끝난다. **`app/api/activities/route.ts` 와 `lib/treeUtils.ts` 는 수정하지 말 것** — 이미 `parentId: null` 추가를 지원한다(확인함).
  - 상태 전파 로직(`app/admin/page.tsx:264-306` 의 `onStatusChange`)은 이번 과제와 무관하다. 같이 손대면 09-20 에 고친 자리를 다시 흔든다.
  - 기존 `submitActivityChange` 를 우회해 새 `fetch` 를 쓰지 말 것 — 09-26 회차가 PUT 6곳을 그 함수 하나로 모아 실패 표시를 붙였다. 새 진입점이 우회하면 그 회차 성과가 무너진다.
  - 알려진 경계: `addActivity` 의 새 id 는 `${Date.now()}`(`lib/treeUtils.ts:49`)라 같은 밀리초에 두 번 추가되면 id 중복으로 400 이 난다. 왕복 요청이 끼어 실무상 재현되기 어렵다. 고치려 들지 말고, 필요하면 요청 중 버튼 disable 정도로 그칠 것(수용 기준 아님).
  - `data` 가 아직 undefined 일 때도 버튼이 렌더될 수 있다. 핸들러가 `data` 를 읽지 않으므로 안전하지만, `data.activities` 를 참조하는 코드를 새로 넣지 말 것.
  - lint 실행 전 `playwright-report/`·`test-results/` 삭제(위 검증 명령 참고).

- 차선 후보: `validateActivityImport` 단위 테스트 추가 — 신규 `lib/activityData.test.ts`. `lib/activityData.ts` 의 실제 함수를 import 해서 (a) 최상위 비객체·`activities` 비배열 → `ActivityImportValidationError` throw, (b) `'진행중'` → `'진행'` 정규화와 `normalizedLegacyStatuses` 카운트(209-212행), (c) id 중복(236-246행)·부모 부재·`level !== parent.level + 1`(262-270행)·순환(273-299행) 각각의 `issues[].path`, (d) 오류 51건 이상일 때 마지막 issue 가 `path: '$'` 의 생략 안내(301-306행)를 고정한다. 프로덕션 코드 변경 0, 검증은 `npm run test:unit`. PUT 과 JSON 업로드 두 경로가 공유하는 유일한 순수 검증 함수인데 전용 테스트가 없다(`lib/` 테스트는 adminSession·treeUtils·mail 4·tracking 2 = 8개, `activityData.test.ts` 부재를 이번에 재확인함).
