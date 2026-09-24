# 과제서 (2026-09-23 정찰)

- 과제: 세 목록 화면이 재조회 중에도 표와 페이지 이동을 유지하도록 `initialLoading`/`refreshing` 패턴을 맞춘다 (가치 4 / 위험 2 / 작업량 M)

- 왜: 같은 "조회 중" 값을 읽는 경로가 7개 화면에 있는데 4개(`OverviewView`, `PolicyCenterView`, `OperationsView`, `AdvancedPolicyView`)만 `initialLoading`/`refreshing` 으로 고쳤고, 실제로 사용자가 가장 오래 머무는 목록 화면 3개(`DirectoryView`, `CatalogView`, `ContentAccessView`)는 아직 `<LoadingBlock v-if="loading" /><template v-else>` 라서 재조회할 때마다 표가 통째로 사라진다. `DirectoryView`/`CatalogView` 는 `PaginationBar` 까지 `v-else` 안에 있어서 다음 페이지를 누르는 순간 페이지 이동 막대 자체가 사라졌다 돌아오고(연속 클릭 불가), 사용자는 방금 무엇이 바뀌었는지 비교할 자리를 잃는다. 정책 쓰기가 잠겨 있는 현재(`public/config/runtime.json` 의 `enablePolicyWrites: false`) 이 세 화면이 실제로 쓰이는 화면이라 효과가 바로 보인다.

- 수용 기준:
  1) `DirectoryView`, `ContentAccessView`, `CatalogView` 에서 **이미 결과가 있는 상태**로 재조회(새로고침 버튼·검색·페이지 이동·같은 탭 재조회)하면 표의 기존 행과 `PaginationBar` 가 화면에 그대로 남고, 대신 `PageHeader` 의 `#actions` 에 `<span class="badge badge--muted refresh-indicator" role="status">갱신 중…</span>` 이 나타난다.
  2) **아직 결과가 없는 상태**(최초 진입, 탭 첫 클릭, 그리고 `ContentAccessView.applyRouteQuery`/`CatalogView.applyRouteQuery` 가 query 변경으로 이전 결과를 버린 직후)에는 지금과 똑같이 `LoadingBlock` skeleton 만 보이고 이전 조건의 행은 절대 보이지 않아야 한다. 이 구분은 `items.length` 가 아니라 기존 `tabLoaded` 류 플래그로 판단한다 — 결과가 진짜 0건인 탭이 skeleton 에 갇히지 않게 하기 위해서다.
  3) 테스트는 세 화면 각각에 대해 (a) 첫 조회 중에는 skeleton 이 보이고 행이 없다, (b) 첫 응답이 온 뒤 두 번째 조회가 진행 중일 때 이전 행 수가 그대로 유지되고 `refresh-indicator` 가 보인다, (c) `ContentAccessView` 에서 `userId` query 가 다른 값으로 바뀌면 재조회가 끝나기 전까지 이전 사용자의 행이 **보이지 않는다**(2번 기준의 반대 방향) 를 증명해야 한다. (c)를 빠뜨리면 이 변경이 커밋 15e032f·f3016f9 가 막아 둔 "다른 사용자 자료를 보고 있다고 오해" 를 되살린다.

- 건드릴 파일 (모두 `upgrade/admin-v2/` 기준):
  - `src/modules/directory/DirectoryView.vue` — script 에 `const initialLoading = computed(() => tabLoading[activeTab.value] && !tabLoaded[activeTab.value])`, `const refreshing = computed(() => tabLoading[activeTab.value] && tabLoaded[activeTab.value])` 추가. 기존 `loading` computed(24행)는 버튼 `:disabled` 용으로 남긴다. 템플릿 114~115행 `<LoadingBlock v-if="loading" :rows="7" /><template v-else>` → `v-if="initialLoading"` / `v-else`, 부서 탭 146~147행도 동일. `PageHeader` `#actions`(88~93행)에 `refreshing` 배지 추가.
  - `src/modules/content/ContentAccessView.vue` — 같은 방식(`tabLoading`/`tabLoaded` 는 23·27행에 이미 있음). 141행(메뉴 트리)·163행(접근 권한 표) 의 `v-if="loading"`→`initialLoading`. `applyRouteQuery`(93~112행)가 이미 `tabLoaded.access = false` 와 `rules.value = []` 를 세우므로 수용 기준 2가 자동으로 지켜진다 — **이 두 줄을 지우거나 순서를 바꾸지 말 것**. `#actions`(120행)에 `refreshing` 배지 추가.
  - `src/modules/catalog/CatalogView.vue` — 208~209행(자료실), 239~240행(공유 앱) 의 `v-if/v-else-if="loading"` 을 같은 패턴으로. 264~266행 상세 패널(`detailLoading`/`detailError`/`selectedLibrary`)은 f3016f9 가 의도적으로 잡아 둔 동작이므로 **건드리지 말 것**. 이 파일의 script(30~180행) 는 이번 정찰에서 읽지 않았다(미확인) — `library`/`sharedApps` 에 대응하는 "이미 조회했다" 플래그가 `tabLoaded` 류로 이미 있는지 먼저 확인하고, 없으면 `DirectoryView` 와 같은 이름·모양으로 새로 만들되 `applyRouteQuery`(175행 `useRouteQuerySync(['tab','userId'], applyRouteQuery)`) 가 query 변경 시 그 플래그를 반드시 false 로 되돌리게 할 것.
  - 테스트: 기존 `src/modules/directory/DirectoryView.test.ts`, `src/modules/content/ContentAccessView.test.ts`, `src/modules/catalog/CatalogView.test.ts` 에 케이스 추가(새 파일보다 기존 파일 확장을 우선). 보류 중인 응답을 직접 만든 Promise 로 잡고 `flushPromises` 사이에서 DOM 을 확인하는, 같은 폴더의 `*.query-race.test.ts` 가 쓰는 방식을 그대로 따를 것.

- 검증 명령 (반드시 `cd upgrade/admin-v2` 에서):
  - `npm ci`  (새 worktree 의 `node_modules` 는 비어 있다. **저장소 루트에서 `npm ci` 를 실행하지 말 것** — 루트 `package.json:34` 의 `"crypto-js": "file:crypto-js-4.2.0.tgz"` tarball 이 없어 설치가 깨진다.)
  - `npm run verify`  (= `vue-tsc --noEmit` + `vitest run`. 변경 전 기준선은 26파일 202테스트 통과다. 먼저 기준선을 한 번 찍고 시작할 것.)
  - 좁혀 돌릴 때: `npx vitest run src/modules/directory src/modules/content src/modules/catalog`
  - `npm run build` 는 이 변경에 필요 없다(수분 소요 + 4개 산출물 검사).

- 위험과 피할 것:
  - **`auth/`, `api/http.ts`, `.gitlab-ci.yml`, `deploy/`, `src/`(루트 legacy 앱)는 이번 과제에서 열지 말 것.** 세션·SSO·배포 경로이고 이번 변경과 무관하다.
  - "조회했다" 판정을 `items.length > 0` 이나 `snapshot != null` 로 하지 말 것. `ContentAccessView`/`CatalogView` 는 query 로 조건이 바뀌므로 길이 기준이면 이전 조건의 행을 그대로 보여 준다(커밋 15e032f 가 고친 바로 그 결함). 반대로 결과가 진짜 0건인 탭은 f29e2df 가 `tabLoaded` 로 고쳤으니 그 플래그를 재사용하는 것이 두 요구를 동시에 만족시키는 유일한 값이다.
  - 세 화면을 한꺼번에 고치되 **한 화면씩 완결**하라(수정 → 해당 테스트 통과 → 다음). 중간에 시간이 모자라면 `CatalogView` 를 남기고 `DirectoryView`+`ContentAccessView` 만 커밋한 뒤 남긴 이유를 명시할 것. 세 화면 중 둘만 조용히 고치고 다 했다고 적지는 말 것.
  - `refresh-indicator` 클래스는 `styles/` 에서 CSS 정의를 찾지 못했다(미확인 — 시각 효과가 없을 수 있다). 새 CSS 를 만들지 말고 `OverviewView.vue:50` 의 마크업을 글자 그대로 복사해 네 화면과 같게 유지할 것.
  - 커밋은 1개, 메시지는 저장소 관례대로 `fix(admin-v2): 재조회 중에도 목록 표와 페이지 이동을 유지한다`. `CHANGELOG.md` 의 `[Unreleased]` 아래 `### Fixed` 에 한 줄 추가(`docs/RELEASE.md` 관례). 버전 파일·태그는 건드리지 말 것.

- 차선 후보: **`AdvancedPolicyView` 가 다른 패널을 저장하면 입력 중이던 값을 조용히 서버 값으로 되돌리는 문제** — `src/modules/policy/AdvancedPolicyView.vue:89 runSave()` 가 성공 뒤 `load()`→`hydrate()`(62행)를 부르며 `form.deleteTerm`/`adminLimit`/`userLimit` 을 덮어쓰고, 확장자 체크박스는 `v-model="extension.useYn"`(238행)로 `snapshot` 객체 자체를 고치므로 `snapshot.value` 교체와 함께 사라진다. 사용자가 확장자 3개를 토글한 뒤 금칙어 하나를 지우면 토글이 소리 없이 되돌아가고, 그 상태로 "파일 유형 저장" 을 누르면 **서버의 옛 값이 다시 저장된다**. 다만 `public/config/runtime.json` 의 `enablePolicyWrites: false` 때문에 현재 배포본에서는 모든 저장 버튼과 체크박스가 비활성이라 도달 불가라서 1순위에서 밀렸다.
