# 회차 노트 2026-09-25-073524-aiportal-front-admin-improve — aiportal-front-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:35] base pinned — main@3cc045a
- [러너 07:35] autonomy release — 

## 정찰 노트
- 골랐다: 네 목록이 요청 시점에 v-model 필터를 직접 읽어 **제출하지 않은 조건이 페이지 이동·새로고침에 새어 드는** 결함. AdvancedPolicy 미저장 편집(차선)은 enablePolicyWrites:false 로 실사용 도달이 막혀 있고, applyRouteQuery 중복 재조회·deferred 중복은 이번 과제와 같은 파일을 건드려 제쳤다. 외부 러너 과제는 6회 no-change 라 rejected 로 내렸다.
- 확실한 것: DirectoryView.vue·CatalogView.vue·ContentAccessView.vue 를 직접 열어 loadUsers/loadDepartments/loadLibrary/loadSharedApps 가 filters 를 요청 인자로 그대로 읽는 것, PaginationBar @change 가 loader 를 바로 부르는 것, ContentAccessView 에는 PaginationBar 가 없어 범위 밖인 것을 확인했다.
- 미확인(추측으로 적은 것): 테스트 기준선을 못 찍었다 — 이 세션은 `npm ci` 승인이 거부됐다. brief 의 214테스트는 직전 회차 수치이고 파일 26개만 이번에 확인했다. 구현자는 변경 전 기준선을 반드시 먼저 찍을 것.
- 조심할 것: tabLoaded/initialLoading/refreshing 판정은 건드리지 말 것(15e032f·f29e2df·9aac9de). CatalogView.applyRouteQuery 에서 route 의 userId 를 확정값에도 넣지 않으면 CatalogView.query-race.test.ts 가 먼저 깨진다 — 그건 구현이 틀린 신호다.
- 증거 형태: 표의 행 개수가 아니라 vi.mock 한 API 함수의 **호출 인자**로 증명하게 수용 기준을 못박았다.
- [러너 07:40] scout done — 페이지 이동·새로고침이 제출하지 않은 검색 조건을 몰래 적용하지 않게 한다 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- 무엇·왜: DirectoryView 에 `applied`, CatalogView 에 `appliedLibrary`/`appliedShared` 를 두고 네 loader 의 요청 인자·`loadSharedApps` 의 UUID early-return·`changeTab` 의 shared 판정이 확정값만 읽게 했다. 제출(`submitUsers`/`submitDepartments`/`submitLibrary`/`submitSharedApps`)과 `applyRouteQuery` 만이 확정값을 세운다. 커밋 de5e466.
- 확신 없는 곳: `changeTab` 의 `appliedShared.userId` 한 줄은 **테스트로 독립 고정되지 않았다** — 그 줄만 입력값으로 되돌려도 새 테스트가 통과한다(`loadSharedApps` 의 early-return 이 이미 요청을 막는다). 변경 전 전체로는 실패했으므로 무의미한 테스트는 아니지만, 그 줄의 방어는 요청 차단이 아니라 불필요한 `tabError`/`sharedApps` 초기화 회피뿐이다. 비평가는 여기를 먼저 볼 것.
- 일부러 안 한 것: CatalogView 템플릿의 빈 상태 안내(`!sharedFilters.userId`, `:244`)와 빈 행 판정(`:260`)은 여전히 입력값을 읽는다 — 과제서가 요청 인자만 옮기라 했고 실사용 영향이 작아 ideas.json 에 새 항목(1/1/S)으로 남겼다. `ContentAccessView` 와 `tabLoaded`/`initialLoading`/`refreshing` 은 손대지 않았다.
- 검증: 기준선 `npm ci && npm run verify` 27파일 **220**테스트 → 새 테스트 10개 중 **9개가 변경 전 실패**(나머지 1개는 수용 기준 3 제출 동작 보존 가드라 전후 모두 통과, 실패 목록을 구현 stash 후 재실행해 이름까지 확인) → 구현 후 `npm run verify` 27파일 **230**테스트 통과, 재실행도 230 통과. 전 회차의 간헐 실패는 이번 5회 실행에서 재현되지 않았다.
- 다음 역할 주의: 이 저장소는 `npm ci` 를 반드시 `upgrade/admin-v2` 에서만 돌린다(루트 `package.json:34` 의 crypto-js tarball 부재로 루트 설치가 깨진다). 새 테스트는 `vi.mock` 대역 기반이라 `CatalogView.query-race.test.ts` 의 실제 axios 배선과 함께 봐야 하며, 그 배선 테스트도 이번 변경 후 통과한다.
- [러너 07:44] brief accepted — 채택 — 지정한 두 파일·행 번호·`applied` 분리 설계·`applyRouteQuery` 주의점·수용 기준 3가지가 모두 현재 코드와 일치해 �
- [러너 07:44] verify passed — 검증 2개 통과 (policy)
- [러너 07:45] pr created — https://github.com/hkjang/aiportal-front-admin/pull/24
- [러너 07:45] merge stopped — 긴급 중지
