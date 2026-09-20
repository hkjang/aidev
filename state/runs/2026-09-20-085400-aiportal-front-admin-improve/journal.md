# 회차 노트 2026-09-20-085400-aiportal-front-admin-improve — aiportal-front-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:54] base pinned — main@1e83c2a
- [러너 08:54] autonomy release — 

## 정찰 노트
- 선택 이유: 보류 1순위 "0건 탭 재조회"를 코드로 재확인(세 화면 changeTab 모두 `!items.length` 그대로, 조회 중 재클릭도 중복 요청) — S 크기, 위험 1, 세 화면 모두 기존 컴포넌트 테스트와 헬퍼(mountView·tab·deferred)가 있어 API 호출 수로 바로 증명 가능. AdvancedPolicyView hydrate(3/2/M)는 45분에 빠듯하고 skeleton 분리(2/2/M)는 범위 결정이 먼저라 제쳤다.
- 확신 없는 곳: 정찰 세션은 권한 제한으로 vitest를 못 돌렸다(base 통과 여부 미확인, 09-19 기록으로 154개 추정). Catalog `applyRouteQuery`에서 `tabLoaded.shared=false` 초기화가 필요한지는 추측 — userId 변경 시 `loadSharedApps()`를 직접 부르므로 없어도 동작은 같을 수 있다.
- 구현자 주의: 플래그는 `changeTab`에서만 읽고 latest 응답에서만 세울 것(늦게 온 이전 응답이 세우면 안 됨). 실패 탭은 재클릭 재시도 유지. `src/auth`·`router.ts`는 304e89e 미병합이라 건드리지 말 것. 되돌려서 신규 테스트가 실제 실패하는지 확인해 요약에 적을 것.
- [러너 08:56] scout done — 결과 0건인 탭을 오갈 때마다 ContentAccessView·CatalogView·DirectoryView가 같은 조회를 되풀이하지 않게 한다 (가�

## 구현 노트
- 무엇/왜: Directory·Catalog·ContentAccess 세 화면의 `changeTab`이 `!items.length`로 "미조회"를 판단해 0건·조회 중 탭을 누를 때마다 같은 API를 다시 불렀다. 탭별 `tabLoaded` 플래그(최신 ticket 응답에서만 세움)로 바꾸고 `!tabLoaded && !tabLoading`일 때만 조회. 커밋 f29e2df, 신규 테스트 7개, `npm run build` 전체 통과(vitest 161개).
- 정찰이 "추측"이라 한 `applyRouteQuery`의 `tabLoaded.shared=false`는 실제로 필요했다 — `/catalog?tab=shared&userId=A`→`/catalog?userId=B`처럼 library 갈래로 가면 `loadSharedApps()`가 안 불리고 A의 0건 결과가 완료로 남는다. Content에도 같은 이유로 `tabLoaded.access=false`를 넣었고(과제서에는 "그대로"라 적혔으나 같은 값을 읽는 경로를 같게 두는 규칙에 따라 맞춤) 둘 다 되돌려 실패하는 테스트로 고정했다.
- 확신 없는 곳: Content `applyRouteQuery`가 menus 갈래일 때 access 플래그를 내리므로 `/content?tab=access&userId=A`(규칙 있음)→`/content`→접근 권한 탭 클릭이 이전엔 재조회 없이 A의 규칙을 보여줬고 이제는 빈 조건으로 재조회한다 — 의도한 동작 변화이나 사용자가 A의 목록이 남길 기대했다면 다르게 볼 수 있다.
- 검증 못 한 것: 브라우저에서 실제 backend로 눌러보지는 않았다(mock api 모듈 위 컴포넌트 테스트로 호출 수만 셌다). 파싱 배선(`unwrapPageEnvelope`)은 이 변경과 무관.
- 일부러 하지 않은 것: 기존 테스트 18개의 `afterEach` unmount 정리(범위 밖, 신규 7개만 unmount) · `src/auth`·`router.ts`(304e89e 미병합) · `applyRouteQuery`가 조건이 안 바뀐 활성 탭도 재조회하는 비효율(ideas.json에 신규 등록).
- 다음 역할 주의: 되돌리기 확인은 `changeTab` 판단 복구(5개 실패)와 `applyRouteQuery` 무효화 줄 제거(2개 실패) 두 갈래로 따로 했다. Directory '조회 중인 탭 재클릭' 테스트는 skeleton 존재를 기대하므로 향후 initialLoading/refreshing 분리 시 함께 손봐야 한다.
- [러너 09:02] brief accepted — 채택 — 근거(세 화면 `changeTab`의 `!items.length` 판단, 헬퍼·mock 위치, 행 번호)가 코드와 정확히 일치했고 base에서 기존 18개
- [러너 09:02] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인: diff 7파일 전부 읽음, `npm run verify` 통과(161개), 세 .vue 를 main 으로 되돌려 신규 테스트 6/7 실제 실패 확인(나머지 1개 'Directory 실패 후 재클릭' 은 설계 회귀 방지용). 플래그는 `isLatest()` 뒤에서만 세워지고 `changeTab` 에서만 읽힘을 코드로 확인. 못 본 것: 실제 backend 브라우저 동작.
- 구현자의 의심(Content menus 갈래의 access 플래그 무효화)은 옳은 동작 — 빈 조건에서 A 의 규칙을 남기는 게 이전 결함이었다.
- 판정 approve, risk low, 보안·법무 차단 없음(새 입력·데이터·권한 변화 없음).
- 남는 우려(기존 결함): `applyRouteQuery` 의 조회 안 하는 갈래가 in-flight 요청을 invalidate 하지 않아 늦게 온 이전 조건 응답이 플래그를 세운다(Content:105·Catalog:164 else 갈래에 `*Request.invalidate()` 한 줄). 다음 회차 후보로 남김.
- 릴리즈 노트: 사용자 체감 변화는 `/content?userId=A`→`/content` 뒤 접근 권한 탭이 전체 조건으로 재조회된다는 점 하나.
- [러너 09:06] review approved — 리뷰 승인 (risk=low)
- [러너 09:06] pr created — https://github.com/hkjang/aiportal-front-admin/pull/19
- [러너 09:06] ci passed — 검사 없음 — 정책으로 허용
- [러너 09:06] merge done — f29e2df
- [러너 09:08] release skipped — 릴리즈 안 함: 릴리즈 이력이 전혀 없는 저장소: git 태그 0개, 릴리즈 커밋 없음, GitHub Release·워크플로 없음, CHANGELOG/릴리즈 노트 없음. 버전 파일(package.
