# 회차 노트 2026-09-24-072409-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:24] base pinned — main@1c607a3
- [러너 07:24] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 1순위였던 '나머지 5화면 검색 배선 실태' 를 이번에 전량 확인해 종결했고(4화면 모두 고유 id + getElementById 로 정상), 그 조사에서 **PopPromptSelect 만 게시판 4화면과 같은 일반 id `searchKey` 를 쓰는 중복 id 결함**이 드러났다 — 이 저장소 결함의 주류(형제 비대칭)에 정확히 들어맞고 두 줄이면 끝나며 관측 가능한 변화가 명확해, 기대 계약이 미확인인 후보들(isSuccess 잔여 3곳·심플봇 아이콘 catch·loading 타이머)을 제쳤다.
- 추측으로 적은 것: 문서 순서(#app 게시판 input 이 teleport 팝업보다 먼저 온다)는 App.vue:49-62 의 Teleport 구조에서 추론한 것이고 **브라우저/jsdom 에서 실제로 재현해 보지는 않았다** — 구현자는 첫 케이스에서 '수정 전에 팝업 검색이 배경 값을 집는다' 를 Red 로 단정해 이 추론을 먼저 검증할 것. `npm test` 기준선(기록상 35파일 560테스트)도 이번에 돌리지 않아 미확인이다.
- 조심할 것: 두 wrapper 를 `attachTo: document.body` 로 붙이고 **게시판을 먼저** 마운트해야 결함이 재현된다. 팝업 열림 watch 가 최초 조회를 한 번 쏘므로 검색 요청은 그 다음 요청으로 골라야 하고, `writeUser` 없으면 요청이 아예 안 나간다. 케이스마다 unmount(형제 스펙과 id 충돌).
- 릴리즈(A): 읽기 전용 재실측 후 무변경 24회째. 구현자는 재조사도 하지 말 것.
- [러너 07:29] scout done — **프롬프트 선택 팝업(PopPromptSelect)의 검색이 배경 게시판 화면의 검색창을 읽는다 — 중복 DOM id `searchKey`**

## 구현 노트
- 무엇을/왜: `PopPromptSelect.vue:107,148` 의 DOM id `searchKey` → `promptSearchKey` 두 줄. 이 팝업은 Teleport 안에서 배경 게시판 화면과 DOM 을 공유하는데 게시판 4화면이 같은 id 를 쓰고 있어, `getElementById` 가 문서 순서상 먼저 오는 게시판 input 을 집었다. 실제로 재현됨: 팝업에 `템플릿` 을 쳐도 `key_word=''`, 팝업이 비고 게시판에 `공지` 가 남아 있으면 `key_word='공지'`.
- 확신 없는 곳: 정찰 노트의 사실 두 개가 **틀렸다**. ①`writeUser` 는 `normalizeUserRecord` 를 거쳐 입력 키가 `userId` 다(`user_id` 로 넣으면 빈 값이 되어 요청이 아예 안 나간다 — 형제 `requestListSearch.spec.js:114` 도 `user_id` 로 넣고 있어 그쪽 `creatorId` 가 실제로 채워지는지는 **확인하지 못했다**). ②팝업 열림 watch 는 값이 바뀔 때만 돌아서 `modelValue: true` 로 마운트하면 최초 조회가 나가지 않는다 — 닫고 마운트 후 `setProps` 로 열어야 한다.
- 수용 기준 5 의 변이 예측도 실제와 달랐다: 템플릿 `searchId` 만 되돌리면 1)·2) 만이 아니라 4건이 실패한다(읽는 id 와 심는 id 가 어긋나 팝업이 자기 입력란을 못 찾음). 인과 분리 목적은 달성(두 줄 모두 load-bearing: 각각 4건/5건 실패).
- 일부러 안 한 것: `SearchBox.vue`, 게시판 4화면의 `searchKey` id, 무배선 `v-model` 잔재 4곳(동작 변화 0 = 무효 변경), `getList` 의 `isSuccess`/페이징/`onSelect` — 전부 범위 밖. 릴리즈(A)는 재조사 없이 완전 무변경(24회째).
- 다음 역할이 조심할 것: `tests/unit/promptSelectSearch.spec.js` 는 `attachTo: document.body` 로 두 컴포넌트를 붙이고 게시판을 **먼저** 마운트해야 결함이 재현된다. 케이스마다 두 wrapper 를 unmount 하지 않으면 같은 id 를 쓰는 형제 스펙(`requestListSearch`·`boardListFailure`)과 DOM 이 섞인다. DB·네트워크는 필요 없다.
- 검증: `npm test` 36파일 567테스트 통과(수정 전 기준선 35/560 을 실제로 돌려 확정), `npx vitest run tests/unit/promptSelectSearch.spec.js` 7/7, `npm run build:dev` 통과 후 `dist/` 삭제. 커밋 f4e5d5d.
- [러너 07:35] brief accepted — 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경, B 를 지정된 두 줄만 고쳐 구현했고 `SearchBox.vue` 와 게시판 4화면의 id �
- [러너 07:35] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인한 것: 소스 2줄을 searchKey 로 되돌려 실제로 Red 재현(2건 실패, '공지' 누출 포함) 후 복원(트리 clean) — 신규 스펙은 수정 전 코드에서 통과하지 않는다. `npm test` 36파일 567테스트 직접 재실측. promptSearchKey 전역 유일성·CSS 셀렉터 미참조 확인. 인증/인가/비밀값/삭제 미접촉, 신규 개인정보 수집 없음 → 보안·법무 차단 사유 없음.
- 못 본 것: `npm run build:dev` 재실행(템플릿 문자열 2개만 바뀌어 빌드 파손 경로 없음으로 판단), 실브라우저 문서 순서(jsdom 재현으로 갈음).
- 남는 우려 1: `tests/unit/requestListSearch.spec.js:114` 가 `writeUser({ user_id: 'tester' })` 로 넣는데 normalizeUserRecord 입력 키는 `userId` 라 저장값이 `user_id: ''` 다 — 그 스펙은 주석과 달리 creatorId 빈 값으로 돌고 있다(RequestList.vue:112). 구현자가 미확인으로 남긴 자리이며 실제로 어긋남. 이번 PR 범위 밖이지만 다음 회차 후보.
- 남는 우려 2: 게시판 4화면은 여전히 서로 같은 id `searchKey`(동시 라우팅 없어 오늘은 무해), NoticeList:114/LibraryList:112/ShareList:93 의 무동작 `v-model="searchKey"` 잔재 3곳.
- 판정 approve / risk low / blocking 없음. 릴리즈는 정책 결손으로 24회째 무변경.
- [러너 07:37] review approved — 리뷰 승인 (risk=low)
- [러너 07:37] pr created — https://github.com/hkjang/aiportal-front/pull/36
- [러너 07:38] ci passed — 검사 없음 — 정책으로 허용
- [러너 07:38] merge done — f4e5d5d
- [러너 07:40] release failed — 릴리즈 안 함: 릴리즈 진입 조건 미충족(25회째, 무변경). 절차 1 재실측 결과 이 저장소에는 따라 할 이전 릴리즈 방식이 없다: git 태그 0개, 릴리즈 커밋
