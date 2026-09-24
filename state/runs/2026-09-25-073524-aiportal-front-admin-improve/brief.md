- 과제: 페이지 이동·새로고침이 제출하지 않은 검색 조건을 몰래 적용하지 않게 한다 (가치 3 / 위험 2 / 작업량 M)
- 왜: `DirectoryView`·`CatalogView` 의 네 목록은 요청을 만들 때마다 검색 폼에 `v-model` 로 묶인 `filters`/`libraryFilters`/`sharedFilters` 를 그대로 읽는다. 사용자가 조건을 고치고 **검색 버튼을 누르지 않은 채** `PaginationBar` 로 다음 페이지를 누르거나 새로고침을 누르면, 화면에 보이던 조건이 아니라 입력만 하고 확정하지 않은 조건으로 조회된다. 특히 `CatalogView` 공유 앱 탭은 사용자 ID 칸만 고쳐 놓고 2페이지를 누르면 **다른 사용자의 공유 앱 2페이지**가 그 자리에 들어오는데, 총건수·페이지 번호는 이전 사용자 기준이라 15e032f 가 막으려던 "이전 사용자의 행" 문제가 반대 방향으로 다시 생긴다. 조회 조건을 제출 시점에 확정하면 표에 보이는 것과 요청이 항상 같은 조건이 된다.

- 수용 기준:
  1) `DirectoryView` 사용자 탭에서 검색 입력을 `"kim"` 으로 바꾸고 제출하지 않은 채 `PaginationBar` 의 `change` 로 2페이지를 요청하면, `getDirectoryUsers` 가 받는 `keyword` 는 직전에 제출된 값(제출한 적이 없으면 `undefined`)이고 `"kim"` 이 아니다. 부서 탭도 같다.
  2) `CatalogView` 공유 앱 탭에서 사용자 ID 입력을 다른 UUID 로 바꾸고 제출하지 않은 채 2페이지를 요청하면, `getSharedApps` 가 받는 `userId` 는 화면에 표시 중인(= route query 또는 직전 제출) 사용자다. 자료실 탭에서 `groupCode`/`viewYn`/`searchOption`/`searchKey` 를 고치고 제출하지 않은 채 페이지 이동·새로고침을 해도 `getLibraryPage` 가 받는 값은 직전 제출값이다.
  3) 검색 버튼 제출은 지금처럼 입력값을 확정해 1페이지부터 조회한다. 테스트는 `vi.mock` 한 API 함수의 **호출 인자**로 (1)(2)(3)을 각각 증명해야 한다 — 표의 행 개수가 아니라 요청 인자가 증거다. 기존 테스트(폼 `submit` 트리거로 검색을 확인하는 것들)는 그대로 통과해야 한다.

- 건드릴 파일:
  - `upgrade/admin-v2/src/modules/directory/DirectoryView.vue`
    - `:30` `const filters = reactive({ users: '', departments: '' })` 는 입력용으로 그대로 두고, **제출 확정값**을 담는 `applied` reactive 를 하나 더 둔다.
    - `loadUsers`(`:37`)·`loadDepartments`(`:57`) 가 `filters.*` 대신 확정값만 읽게 한다. `lastPageWhenOutOfRange` 재귀(`:48`, `:71`)도 같은 값을 다시 읽으므로 자동으로 맞는다.
    - 확정 후 1페이지를 읽는 `submitUsers()`/`submitDepartments()` 를 추가하고, 템플릿 `:114` `@submit.prevent="loadUsers(1)"` / `:146` `@submit.prevent="loadDepartments(1)"` 를 그것으로 바꾼다.
    - 새로고침 버튼 `:95` 와 `PaginationBar` `:139`·`:168` 은 그대로 둔다(확정값을 쓰게 되는 것이 이번 변경의 목적).
  - `upgrade/admin-v2/src/modules/catalog/CatalogView.vue`
    - `:38`~`:39` 의 `libraryFilters`(4키)·`sharedFilters`(2키)에 대응하는 확정값 reactive 를 둔다.
    - `loadLibrary`(`:78`)의 `getLibraryPage` 인자와 `loadSharedApps`(`:124`)의 **UUID 검증 early-return 을 포함해** `userId`·`searchKey` 를 확정값에서 읽게 한다.
    - `changeTab`(`:154`)의 `tab === 'shared' && sharedFilters.userId` 조건도 확정값으로 바꾼다 — 안 그러면 ID 칸에 글자만 넣고 탭을 오가는 것으로 조회가 트리거된다.
    - `applyRouteQuery`(`:164`)는 route 가 조건의 출처이므로 `sharedFilters.userId` 와 확정값 **양쪽**에 query 의 `userId` 를 넣고, `searchKey` 는 양쪽에서 비운다. 여기를 빠뜨리면 `CatalogView.query-race.test.ts` 가 깨진다.
    - 템플릿 `:207` `@submit.prevent="loadLibrary(1)"` / `:239` `@submit.prevent="loadSharedApps(1)"` 를 확정 함수로 바꾼다.
  - `upgrade/admin-v2/src/modules/directory/DirectoryView.test.ts`, `upgrade/admin-v2/src/modules/catalog/CatalogView.test.ts` — 위 수용 기준 케이스 추가. `DirectoryView.test.ts:282` 에 이미 `wrapper.findComponent(PaginationBar).vm.$emit('change', 4)` 로 페이지를 넘기는 방식이 있으니 그대로 쓰면 된다. `CatalogView.test.ts:224`·`:246` 의 `.catalog-filter` / `.shared-app-filter` submit 트리거도 그대로 쓴다.

- 검증 명령: `cd upgrade/admin-v2 && npm ci && npm run verify` (typecheck + vitest). 특정 파일만 빠르게 보려면 `cd upgrade/admin-v2 && npx vitest run src/modules/directory src/modules/catalog`.
  - **미확인**: 이번 정찰 세션은 샌드박스가 `npm` 실행을 막아 기준선을 직접 돌리지 못했다. 직전 회차(9aac9de)에서 측정된 기준선은 26파일 214테스트이며, 이후 커밋은 릴리즈 커밋과 PR #17 병합뿐이라 크게 달라지지 않았을 것으로 본다 — 구현자는 **변경 전에 기준선을 한 번 찍고** 시작할 것.

- 위험과 피할 것:
  - `tabLoaded`/`initialLoading`/`refreshing` 판정을 건드리지 말 것. 직전 회차(9aac9de)가 세 화면에서 맞춰 놓은 것이고, `items.length` 나 `snapshot != null` 로 되돌리면 15e032f·f29e2df 가 되살아난다. 이번 변경은 "요청 인자를 어디서 읽는가" 만 바꾸는 것이다.
  - `CatalogView.query-race.test.ts`·`ContentAccessView.query-race.test.ts` 는 실제 axios adapter 를 통과하는 배선 테스트다. route query → 확정값 반영을 빠뜨리면 여기서 먼저 깨진다. 깨지면 구현이 틀린 것이지 테스트가 낡은 것이 아니다.
  - `ContentAccessView` 는 **범위 밖**이다. 접근 권한 탭에는 `PaginationBar` 가 없고 `loadRules` 는 폼 제출과 `applyRouteQuery` 에서만 불리므로 이 결함이 없다(이번에 파일을 열어 확인함). 같은 회차에 건드리지 말 것.
  - 보호 경로 회피: `src/`(legacy), `upgrade/admin-v2/src/auth/`, `api/http.ts`, `.gitlab-ci.yml`, `deploy/`, 버전 파일·태그는 건드리지 않는다.
  - 루트에서 `npm ci` 를 돌리지 말 것 — 루트 `package.json:34` 의 `"crypto-js": "file:crypto-js-4.2.0.tgz"` 가 tarball 부재로 설치를 깨뜨린다.

- 차선 후보: **AdvancedPolicyView 저장·새로고침이 미저장 편집을 덮어씀** (3/2/M) — `runSave()` → `load()` → `hydrate()` 가 `form.deleteTerm`/`adminLimit`/`userLimit` 을 덮고, 확장자 체크박스는 `v-model` 이 snapshot 객체를 직접 고쳐 snapshot 교체와 함께 사라진다. 단 `public/config/runtime.json` 의 `enablePolicyWrites:false` 때문에 현재 배포본에서 저장 경로에 도달할 수 없어, 증거는 테스트로만 세울 수 있다.
