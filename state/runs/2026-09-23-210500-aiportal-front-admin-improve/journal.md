# 회차 노트 2026-09-23-210500-aiportal-front-admin-improve — aiportal-front-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:05] base pinned — main@e67da65
- [러너 21:05] autonomy release — 

## 정찰 노트
- 목록 화면 3개의 재조회 blanking 을 골랐다: 같은 "조회 중" 값을 4개 화면은 이미 initialLoading/refreshing 으로 읽는데 3개만 안 고쳐진, 운영자가 되풀이한 "한쪽만 넓히지 말라" 유형이다. 외부 러너 겨냥 과제는 6회 연속 no-change 라 후보에서 영구 제외했고, AdvancedPolicyView 편집 유실은 결함은 진짜지만 runtime.json 의 enablePolicyWrites:false 때문에 현재 도달 불가라 차선으로 내렸다.
- 확신 없는 곳(과제서에 미확인이라 적음): CatalogView script 30~180행을 읽지 않아 library/sharedApps 의 "조회했다" 플래그가 이미 있는지 모른다. refresh-indicator 는 styles/ 에서 CSS 정의를 못 찾았다. 202테스트 기준선은 전 회차 기록이고 이번에 직접 돌리지 않았다.
- 구현자가 조심할 것: "조회했다" 판정을 items.length 로 하면 커밋 15e032f(다른 사용자 자료 노출)를 되살리고, snapshot!=null 로 하면 f29e2df(0건 탭 반복 조회)를 되살린다. 기존 tabLoaded 플래그만이 둘을 동시에 만족한다.
- 루트에서 npm ci 금지(crypto-js file: tarball 부재). 검증은 cd upgrade/admin-v2 && npm ci && npm run verify.
- [러너 21:09] scout done — 세 목록 화면이 재조회 중에도 표와 페이지 이동을 유지하도록 `initialLoading`/`refreshing` 패턴을 맞춘다 (가�

## 구현 노트
- 무엇을 왜: `DirectoryView`·`ContentAccessView`·`CatalogView` 가 재조회마다 `LoadingBlock v-if="loading"` 으로 표와 `PaginationBar` 를 통째로 지우던 것을, 나머지 4개 화면이 이미 쓰는 `initialLoading`/`refreshing` 으로 맞췄다. 판정값은 기존 `tabLoaded` 플래그를 그대로 재사용했다(`items.length` 는 15e032f 가 고친 결함을 되살린다).
- 확신 없는 곳: (1) `refresh-indicator` 의 CSS 정의를 `styles/` 에서 찾지 못했다 — 마크업은 `OverviewView.vue:50` 과 글자 그대로 같지만 실제 브라우저에서 어떻게 보이는지는 확인하지 못했다(테스트는 DOM 존재만 증명한다). (2) `npm run build` 와 실제 브라우저 실행은 하지 않았다 — 이 변경에 필요 없다고 판단했다. (3) `CatalogView` 공유 앱 탭의 `<p v-if="!sharedFilters.userId && !loading">` 를 `!initialLoading` 으로 바꿨다. userId 가 없으면 `loadSharedApps` 가 즉시 반환해 `refreshing` 이 설 수 없으므로 동치라고 판단했으나, 이 동치성만은 테스트가 아니라 코드 읽기로 확인했다.
- 일부러 하지 않은 것: `CatalogView` 상세 패널(`detailLoading`/`detailError`/`selectedLibrary`, 264~266행)은 f3016f9 가 의도적으로 잡아 둔 동작이라 건드리지 않았다. `ContentAccessView.applyRouteQuery` 의 `tabLoaded.access = false`/`rules.value = []` 두 줄도 순서 그대로 두었다 — 수용 기준 2가 여기에 달려 있다. `AdvancedPolicyView` 미저장 편집 문제(차선 후보)는 `enablePolicyWrites:false` 로 도달 불가라 손대지 않았다.
- 기존 테스트 1개 수정: `DirectoryView.test.ts` 의 '느린 탭의 응답이 다른 탭의 skeleton을 풀지 않는다' 는 이미 결과가 있는 사용자 탭에 skeleton 을 기대하고 있었다. 검사하는 불변식(느린 탭 응답이 다른 탭의 조회 중 표시를 풀면 안 된다)은 그대로 두고 기대 대상만 `refresh-indicator` 로 바꾸고 이름도 맞췄다. 약화가 아니라 같은 불변식의 새 표식이다.
- 다음 역할이 조심할 것: 검증은 반드시 `cd upgrade/admin-v2` 에서 하고 저장소 루트에서 `npm ci` 를 돌리지 말 것(루트 `package.json:34` 의 tarball 부재로 깨진다). 전 회차가 기록한 갱신 표시 테스트의 간헐 실패는 이번에 `npm run verify` 4회(기준선 1 + 변경 후 3) 모두 통과해 재현되지 않았다 — 원인은 여전히 미규명이니 붉게 나오면 재실행해 보고 기록할 것.
- [러너 21:15] brief accepted — 채택 — 지정한 3개 화면·행 번호·`tabLoaded` 재사용·수용 기준 3가지가 모두 현재 코드와 일치해 그대로 구현했고, 미확�
- [러너 21:15] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인함: `npm run verify` 통과(26파일 214테스트, +12). 세 .vue 를 main 판으로 되돌려 재실행하니 3파일 8테스트 실패 — 새 테스트가 대상을 실제로 검증한다. 통과한 4개는 의도된 회귀 가드(첫 조회 skeleton, userId query 변경 시 이전 사용자 행 숨김).
- 못 본 것: 실제 브라우저·UAT 화면, 스크린리더에서의 `role="status"` 실제 낭독, Java/IAM 연동. 간헐 실패는 이번 1회 실행에서 재현되지 않음.
- 승인이어도 남는 우려: route query 경로만 tabLoaded 를 내린다. 화면 안 폼으로 사용자 ID 를 바꿔 제출하면(ContentAccessView 접근권한 탭) 응답 전까지 이전 사용자 행과 `rules.length` 가 남고 '갱신 중…' 배지는 PageHeader 에 멀리 있다 — 인가 문제는 아니나 오귀속 창이다.
- `.refresh-indicator` 는 base.css 에 규칙이 없는 사실상 테스트 훅이다. 미사용 CSS 정리 회차에 지우면 12개 테스트가 함께 깨진다.
- 릴리즈 노트: CHANGELOG.md:9 에 PR 번호 `(#NN)` 가 빠져 있다(기존 항목은 모두 있음). 병합 시 채울 것.
- [러너 21:18] review approved — 리뷰 승인 (risk=low)
- [러너 21:18] pr created — https://github.com/hkjang/aiportal-front-admin/pull/23
- [러너 21:19] ci passed — 검사 없음 — 정책으로 허용
- [러너 21:19] merge done — 9aac9de
- [러너 21:21] release published — v0.1.2
- [러너 21:21] gh-release created — GitHub Release v0.1.2
- [러너 21:21] assets n/a — 이전 릴리즈에도 자산 없음
