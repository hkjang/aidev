# 회차 노트 2026-09-24-063412-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:34] base pinned — main@26a8f9d
- [러너 06:34] autonomy release — 

## 정찰 노트
- A(릴리즈)는 23회째 진입 조건 미충족 — 실측만 재확인하고 무변경 지시로 남겼다. B 로 RequestList 검색 무동작을 골랐다: 직전 회차가 남긴 후보 중 유일하게 '기대 계약 미확인' 이 아니고(형제 4화면이 이미 DOM 직독 방식을 확정해 둠) 사용자에게 보이는 기능이 실제로 죽어 있다. 심플봇 catch(e){}·globalLoading 은 계약 미확인이라 또 제쳤다.
- SearchBox.vue 에 modelValue·input 바인딩이 없다는 것과 호출부 5곳의 비대칭은 소스로 확정했다. **다만 마운트로 재현하지는 않았다** — 구현자는 Red 가 실제로 나는지부터 확인할 것.
- 추측으로 적은 것: 차선 후보의 5화면(SupportOcr/Stt/Img, ChatStorage, PopPromptSelect)이 DOM 을 읽는지는 열어 보지 않아 미확인. `isFirst`/`searchCount` 가 검색 시 어떻게 변해야 하는지도 미확인이라 건드리지 말라고 적었다.
- 구현자 주의: ①SearchBox 자체를 고치는 '근본 수정' 금지(10화면 동시 변경) ②`getElementById` 때문에 테스트는 `attachTo: document.body` 필요 ③`v-model` 제거는 동작 변화 0 인 정리이니 변이 근거를 요구하지 말 것, 나머지 두 줄은 변이로 증명할 것.
- [러너 06:38] scout done — **요청 게시판(RequestList)의 검색이 무동작 — 검색어가 요청에 실리지 않는 문제 수정** (가치 4 / 위험 2 / �

## 구현 노트
- 무엇/왜: `RequestList` 만 `SearchBox` 에 `searchId` 를 넘기지 않고 `onSearchClick` 이 (아무것도 연결되지 않는) `searchKey.value` 를 그대로 써, 요청 게시판 검색이 완전 무동작이었다. 형제 4화면과 문자 단위로 같은 `getElementById('searchKey')` + `trim()` 두 줄을 넣고 `searchId="searchKey"` 를 넘겼다. 무의미하던 `v-model="searchKey"` 는 제거(동작 변화 0 인 정리 — 변이 근거 없음을 알고 한 것).
- 확신 없는 곳: (1) `v-model` 제거로 `SearchBox` 루트 div 에 붙던 fallthrough attr `modelvalue=""` 가 사라진다 — `src/assets` 전체 grep 결과 이를 쓰는 CSS 선택자는 없다(유일한 매칭은 `_layout.scss:627` 의 주석). (2) 검색 결과가 `isFirst`/`searchCount`/`excludedCount` 를 어떻게 바꿔야 하는지는 여전히 **미확인**이라 고정하지 않았다 — 검색이 이제 실제로 동작하므로 `excludedCount`(1페이지 고정글 수) 기반 번호 매기기가 검색 결과에서 어떻게 보이는지는 검증 못 했다. (3) 실제 백엔드에 `searchKey` 를 보냈을 때의 응답은 확인 불가(백엔드 별개 저장소) — 단정은 나가는 URL 까지만.
- 일부러 안 한 것: `SearchBox.vue` 자체 수정(10화면 공유, `PopGlobalSearch` 처럼 ref 를 비우는 곳이 있어 양방향 바인딩을 켜면 동시다발 회귀), 주석 처리된 글자수 검증 블록 부활(승인 안 된 기능 변경), `inf.library.*` → request 엔드포인트 교체(백엔드 확인 불가), 릴리즈 일체(23회째 진입 조건 미충족).
- 다음 역할 주의: 신규 스펙은 `document.getElementById` 를 통과시켜야 해 `attachTo: document.body` 로 마운트한다. 같은 id `searchKey` 를 쓰는 `boardListFailure.spec.js` 와 섞이면 오염되므로 케이스마다 `unmount` 가 필수다. `npm ci` 선행 필수(수 분). `npm run build`·`npm run lint` 는 이 저장소에 없다.
- [러너 06:42] brief accepted — 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경, B 를 수용 기준 1~6 그대로 구현·검증했다. 과제서가 경고한 함정을 모
- [러너 06:43] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인함: 변이 검증(RequestList.vue 만 main 판으로 되돌려 신규 스펙 5/8 실패 — 테스트가 대상을 실제로 실행한다), 전체 suite 35파일 560테스트 통과(기준선 34/552), `npm run build:dev` 성공 후 `dist/` 제거·작업 트리 무변경, 형제 4화면(NoticeList:95,114 / LibraryList:93,112 / ShareList:75,93)과 문자 단위 동일한 관례 준수. **판정: approve, risk low, 보안·법무 차단 없음**(인증/인가/비밀값/의존성 변경 0건, 신규 개인정보 범주 0건, 검색어는 URLSearchParams 인코딩).
- 구현자의 "확신 없는 곳" 3건 모두 시험함 — (1) fallthrough attr 제거는 CSS 미사용이라 무해, (2) `excludedCount`/`searchCount` 는 검색 시 currentPage=1 에서 매번 재계산되고 `searchCount` 는 주석 처리된 검증 블록 외에 소비처가 없어 회귀 아님, (3) 백엔드 응답 미확인은 이 저장소에서 확인 불가능한 항목이라 정당한 한계.
- 못 본 것: 실브라우저에서의 동작(테스트는 jsdom), `inf.library.list` 가 `groupCode=100` 에 대해 `searchKey` 를 어떻게 해석하는지(백엔드 별개 저장소 — 단정은 나가는 URL 까지만).
- 승인이어도 남는 우려(다음 회차 후보, 이번 diff 밖): **2페이지에서 검색하면 목록 요청이 2번 나간다** — `onSearchClick` 이 `currentPage.value = 1`(watch 비동기 발화) 과 `getBoardList()` 를 둘 다 호출. 임시 스펙으로 실제 2건 전송 확인. main 에도 있던 선존재 결함이며, 형제 3화면은 `usePaging.resetPageAndGet()` 를 써서 없다 — RequestList 만 `useTableUtils` 를 쓰는 **형제 비대칭**이 원인. 또 id `searchKey` 를 `PopPromptSelect.vue:148` 이 공유하므로(현재 동시 DOM 경로는 못 찾음) 이 우회 관례의 표면이 한 화면 넓어졌다.
- 릴리즈: 버전 파일·태그·CHANGELOG 무변경, 저장소에 관례를 정할 입력이 여전히 없어 23회째 no-change 유지. 릴리즈 노트 한 줄 = "요청 게시판 검색창에 입력한 검색어가 실제 조회에 반영된다".
- [러너 06:46] review approved — 리뷰 승인 (risk=low)
- [러너 06:46] pr created — https://github.com/hkjang/aiportal-front/pull/35
- [러너 06:47] ci passed — 검사 없음 — 정책으로 허용
- [러너 06:47] merge done — 5b7a15b
- [러너 06:48] release failed — 릴리즈 안 함: 릴리즈 관례를 정할 근거가 저장소에 없어 다음 버전을 결정할 수 없다(24회째, 사람 입력 필요). 이번 회차에 읽기 전용으로 재조회한 실
