# 회차 노트 2026-09-24-050915-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:09] base pinned — main@8f11ecb
- [러너 05:09] autonomy release — 

## 정찰 노트
- 목록 3형제를 골랐다: 상세 3형제(a6ed70e)가 방금 '상세=openAlert' 전례를 세웠고, 그때 '미확인' 이라 미뤘던 목록 계약은 같은 저장소의 `Sidemenu.fetchAppList`(:207-211, :231-234)·`getHistoryList` 가 이미 '목록=toast' 로 확정해 둔 것이라 문구를 발명할 필요가 없다. 차선(심플봇 아이콘 `catch(e){}`)은 기대 계약이 여전히 미확인이라 제쳤다.
- 확신 없는 곳: RequestList 의 업무 실패 early return 이 `isFirst`/`searchCount`(:141-144)·`excludedCount`(:149-152) 갱신을 건너뛰는데, 그 차이가 화면에 관측되는지는 **미확인**이다 — 과제서에 "확인하고 있으면 케이스로 고정" 으로 남겼다. RequestList/LibraryList 의 마운트 하네스는 전례가 없다(상세 3형제 스펙이 가장 가까운 본보기).
- 구현자가 조심할 것: ①토스트를 `if(e=='COM') return` **뒤**에 둘 것(앞이면 전역 Alert 과 중복) ②업무 실패 분기의 목록 초기화는 **빼고 돌려 실패하는 케이스가 있을 때만** 넣을 것(직전 2회차가 무효 변경으로 걸린 자리) ③LibraryList 는 `route.params.type`, RequestList 는 `readUser()` 가 없으면 조회 파라미터가 비어 조용히 통과한다.
- A(릴리즈)는 21회째 무변경. 이번 워크트리에서 실측 재확인(tag 0 / shallow=false / version 0.0.0 / .github 없음 / CI_COMMIT_TAG grep -c = 0). 구현자는 재조사하지 말고 손대지 말 것.
- [러너 05:13] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을/왜: 목록 3형제(`NoticeList:60` / `LibraryList:58` / `RequestList:132`)의 `getBoardList` 에 `isSuccess` 업무 실패 토스트(+return)와 catch 의 비-`'COM'` 예외 토스트를 넣었다. 조회 실패가 Nodata '…없습니다' 라는 사실 진술로 감춰지던 자리다. 문구·수단은 `Sidemenu.fetchAppList`(:208-212, :233-236)를 그대로 따랐다(목록=toast, 상세=openAlert 구분 유지). 커밋 ce28031.
- **확신 없는 곳**: ①업무 실패 시 `return` 이라 이전 목록·`totalCount`·Pagination 이 **그대로 남는다**(성공 후 재조회 실패 케이스로 고정함). 이는 `fetchAppList` 전례와 같은 선택이지만, '실패하면 목록을 비우는 게 맞다' 가 기대 계약이면 뒤집어야 한다 — 사람 확인 대상. ②RequestList 의 업무 실패 early return 이 `isFirst`/`searchCount`(:144-147)·`excludedCount`(:152-155) 갱신을 건너뛴다. 정찰이 남긴 숙제인데 **화면에 관측되는 차이를 못 찾아** 케이스로 고정하지 않았다. ③토스트 기본 문구('공지사항/자료실/요청 목록 조회에 …')는 기존 양식을 따라 지었을 뿐 승인된 문구는 아니다.
- 일부러 하지 않은 것: 업무 실패 분기의 목록 초기화(표준형대로 `return` 만 뒀다. '비우는 쪽' 변이는 돌려 보지 않았고, 대신 현재 선택인 '이전 목록 유지' 를 케이스로 고정했다) / `RequestList` 의 `console.error` 제거 / `SearchBox` 의 `v-model` 미배선(별건, 아래) / 릴리즈 관련 일체(22회째 무변경, 정찰 지시대로 재조사도 안 함).
- 다음 역할이 조심할 것: 새 스펙 `tests/unit/boardListFailure.spec.js` 는 전역 싱글턴(globalLoading·globalAlert·toastProxy)을 케이스마다 직접 초기화한다 — beforeEach/afterEach 를 지우면 `RequestList` 의 `Nodata v-if="!isLoading …"` 때문에 조용히 깨진다. 변이 실험 시 `git checkout` 으로 되돌리지 말 것(미커밋 수정본까지 날아간다 — 이번에 한 번 당했다).
- 다음 회차 1순위 후보: `SearchBox.vue` 에 `modelValue` prop/바인딩이 없어 `v-model="searchKey"` 가 무동작이고, Notice/Library 는 `getElementById('searchKey')` 로 우회하지만 **RequestList 는 `searchId` 를 안 넘겨 검색어가 영영 빈 문자열**이다(읽어서 확인, 마운트 재현은 안 함).
- [러너 05:20] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인: 변이 검증(3개 뷰만 main 으로 되돌려 새 스펙 실행 → 18중 12 실패, 원복 후 전체 34파일/552테스트 통과)·COM 가드 위치·interceptors 3개 'COM' 경로·isSuccess 정의·범위 이탈·revert 안전성. 판정 approve, blocking 없음.
- 구현 노트 ② 해소: searchCount 는 RequestList.vue:89 주석 블록에서만 읽히고 excludedCount(:159)는 early return 이 남긴 이전 목록과 짝이 맞는다 — 관측 차이 없음, 케이스 미고정은 결함 아님.
- 못 본 것: 백엔드가 이 저장소에 없어 /board·/library 목록 응답이 실제로 code/success 봉투를 주는지 미확인. 안 주면 성공 조회가 토스트+빈 화면으로 퇴행한다(useSettingStore.js:90 의 'isSuccess(res) || body' 가 봉투 없는 엔드포인트의 존재를 시사). 다음 회차가 실응답으로 확인할 자리.
- 남는 우려(릴리즈 노트용): 새 문구 3종과 '업무 실패 시 이전 목록 유지' 선택 모두 전례를 따랐을 뿐 사람 승인은 없다. 뒤집으려면 boardListFailure.spec.js 의 마지막 케이스를 함께 고쳐야 한다.
- 다음 회차 1순위 동의: SearchBox 에 modelValue 가 없어 세 화면의 v-model 이 무동작이고 RequestList(:226)는 searchId 미전달로 검색어가 항상 빈 문자열이다.
- [러너 05:23] review approved — 리뷰 승인 (risk=low)
- [러너 05:23] pr created — https://github.com/hkjang/aiportal-front/pull/34
- [러너 05:24] ci passed — 검사 없음 — 정책으로 허용
- [러너 05:24] merge done — ce28031
- [러너 05:25] release failed — 릴리즈 안 함: 릴리즈 관례가 저장소 안에 존재하지 않아 다음 버전을 결정할 수 없다(23회째 동일 원인, 무변경). 이번 워크트리에서 실측 재확인: git ta
