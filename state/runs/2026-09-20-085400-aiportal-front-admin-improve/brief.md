# 과제서 (2026-09-20)

- 과제: 결과 0건인 탭을 오갈 때마다 ContentAccessView·CatalogView·DirectoryView가 같은 조회를 되풀이하지 않게 한다 (가치 2 / 위험 1 / 작업량 S)

- 왜: 세 목록 화면의 `changeTab`은 "그 탭을 아직 조회했는가"를 `!items.length`(빈 배열)로 판단해, 응답이 정말 0건이거나 아직 조회 중(응답 전이라 배열이 비어 있음)일 때 탭을 누를 때마다 같은 API를 다시 부른다(빈 부서 목록·권한 규칙 0건인 사용자에게 흔하다). 탭별 "조회 완료" 플래그를 두면 0건 탭도 한 번만 조회하고, 조회 중인 탭을 다시 눌러도 중복 요청이 나가지 않으며, 실패한 탭은 지금처럼 다시 누르면 재시도한다.

- 수용 기준:
  1) DirectoryView: 부서 API가 0건(`items: []`)을 돌려준 뒤 사용자 ↔ 부서 탭을 두 번 오가도 `getDirectoryDepartments`는 정확히 1번만 호출된다(사용자 탭은 onMounted 1번 그대로).
  2) CatalogView(`?tab=shared&userId=<uuid>` 진입)·ContentAccessView(`?userId=<uuid>` 진입)에서도 같은 조건으로 API가 1번만 호출된다 — 0건 응답 후 탭 왕복에 재조회 없음.
  3) 조회 중(응답이 아직 안 온 상태)인 탭을 다른 탭 갔다가 다시 누르면 새 요청을 내지 않는다(호출 수 1 유지). 반면 조회가 실패(reject)한 탭을 다시 누르면 재조회한다(호출 수 2) — 기존 재시도 동작 유지.
  4) 기존 테스트(딥링크 재진입·탭별 loading/error 격리·범위 밖 페이지 복구)가 그대로 통과한다. 특히 `applyRouteQuery`(Catalog/Content)는 query 변경마다 `loadX()`를 직접 부르므로 플래그와 무관하게 재조회돼야 한다.
  5) 수정을 되돌리면(플래그 판단을 `!items.length`로 복구) 신규 테스트 중 "0건 후 탭 왕복" 테스트가 실제로 실패하는 것을 확인해 요약에 적는다.

- 건드릴 파일:
  - `upgrade/admin-v2/src/modules/directory/DirectoryView.vue` — `tabLoading`·`tabError`(19-20행) 옆에 `tabLoaded = reactive<Record<DirectoryTab, boolean>>({ users: false, departments: false })` 추가. `loadUsers`/`loadDepartments`의 성공 경로(`ticket.isLatest()` 안, users/departments 값을 대입하는 곳)에서 `tabLoaded.x = true`. `changeTab`(71-75행)의 `!users.value.items.length` → `!tabLoaded.users && !tabLoading.users` (departments 동일). 실패 시에는 loaded를 세우지 않아 재클릭이 재시도가 되게 둔다.
  - `upgrade/admin-v2/src/modules/catalog/CatalogView.vue` — 같은 패턴. `changeTab`(138-142행): `library`는 `!tabLoaded.library && !tabLoading.library`, `shared`는 `sharedFilters.userId &&` 조건을 유지한 채 동일 치환. `applyRouteQuery`(146-158행)는 userId가 바뀔 때 `loadSharedApps()`를 직접 부르므로 그대로 두되, `tabError` 초기화 옆에서 `tabLoaded.shared = false`로 되돌리면 안전(userId가 바뀌면 이전 0건 결과는 무효). `loadSharedApps`의 "userId 없음/UUID 아님" 조기 반환(110-115행)에서는 loaded를 세우지 않는다.
  - `upgrade/admin-v2/src/modules/content/ContentAccessView.vue` — `menus`/`rules` 배열. `changeTab`(76-80행) 동일 치환. `loadRules`의 UUID 검증 실패 조기 반환(52-56행)에서는 loaded를 세우지 않는다. `applyRouteQuery`(88-100행)는 직접 `loadRules()/loadMenus()`를 부르므로 그대로.
  - `upgrade/admin-v2/src/modules/directory/DirectoryView.test.ts` — `describe('DirectoryView 탭별 조회 상태')`에 3개 추가: (a) 부서 0건(`pageOf([])`) 응답 후 탭 왕복 → `getDirectoryDepartments` 1번, (b) `deferred()`로 부서 응답을 미룬 채 탭 왕복 → 1번, (c) 부서 reject 후 다시 누르면 2번. 기존 헬퍼 `mountView()`, `tab(wrapper, '부서')`, `deferred()`, `pageOf()` 를 그대로 쓴다.
  - `upgrade/admin-v2/src/modules/catalog/CatalogView.test.ts`, `src/modules/content/ContentAccessView.test.ts` — 각 1개(0건 후 탭 왕복 → 1번). 기존 `vi.mock('@/api/...')` 대역과 mount 헬퍼를 따른다. Content 테스트의 기본 mock이 이미 `[]`(0건)이라 바로 재현된다.
  - (선택, 같은 파일 안이라 함께 해도 됨) 세 테스트 파일에 `afterEach(() => wrapper?.unmount())` 정리가 없다 — 이번에 새로 mount하는 테스트는 끝에서 `wrapper.unmount()` 하라. 기존 테스트까지 손보는 것은 범위 밖.
  - `upgrade/admin-v2/docs/ARCHITECTURE.md` — 탭 계약 절(탭별 loading/error를 다룬 곳)에 "탭 재조회는 조회 완료 플래그로 판단, 0건도 완료, 실패는 재시도" 한 문단 추가.

- 검증 명령 (`upgrade/admin-v2`에서):
  - `npx vitest run src/modules/directory src/modules/catalog src/modules/content` (빠른 확인)
  - `npm run verify` (vue-tsc + vitest 전체 — 09-19 기준 154개 + 신규 5개 = 159개 기대)
  - `npm run build` (verify → vite build → runtime-config → offline → integrity manifest 19개 → verify-integrity; 수 분)
  - 정찰 세션에서는 권한 제한으로 vitest를 직접 돌리지 못했다(미확인) — 구현자는 먼저 기존 테스트가 base에서 통과하는지 확인하고 시작하라.

- 위험과 피할 것:
  - `src/auth/*`·`src/app/router.ts`는 건드리지 말 것 — 09-17 커밋 304e89e(`auto/2026-09-17-0233`)가 아직 main 미병합이라 충돌 여지가 있다(`git branch --no-merged main`으로 확인함).
  - `.gitlab-ci.yml`·`scripts/*.mjs`는 배포 게이트 — 손대지 말 것. `.gitlab-ci.yml`에 평문 PAT가 있으니 값을 옮겨 적지 말 것.
  - 세 화면의 `applyRouteQuery`·검색·페이지 이동·새로고침 버튼은 모두 `loadX()`를 직접 부르므로 플래그가 그 경로를 막으면 안 된다 — 플래그는 `changeTab`에서만 읽는다. 딥링크 테스트(09-07)와 범위 밖 페이지 복구 테스트(09-09)가 회귀 감지기다.
  - 운영자 규칙: "같은 값을 읽는 경로가 여럿이면 모두 같게" — 세 화면에 같은 패턴을 같은 이름(`tabLoaded`)으로 넣고 셋 다 테스트로 증명하라. 한 화면만 고치지 말 것.
  - `createRequestGuard`의 ticket 판정(`ticket.isLatest()`)은 그대로 두라 — loaded 표시는 latest 응답에서만 세운다. 늦게 온 이전 응답이 플래그를 세우면 안 된다.
  - 실제 동작이 안 바뀌는 수정은 넣지 말 것 — 되돌려서 테스트가 실패하는지 반드시 확인.

- 차선 후보: CatalogView·DirectoryView·ContentAccessView 테스트 파일에 `afterEach` wrapper 정리 추가(공용 mount 헬퍼 없이 파일별로만) (2/1/S) — 1순위가 어떤 이유로 성립하지 않으면(예: 기존 테스트가 base에서 이미 깨져 있음) 이것을 고른다. 그다음은 AdvancedPolicyView의 hydrate가 미저장 편집을 덮어쓰는 문제 (3/2/M)이나 45분 안에는 빠듯하다.
