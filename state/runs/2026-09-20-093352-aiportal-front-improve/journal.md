# 회차 노트 2026-09-20-093352-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:33] base pinned — main@210207e
- [러너 09:33] autonomy release — 

## 정찰 노트
- 선택 이유: 보류 1순위였던 `String(status) ?? ''` 정리는 status 소비처가 Sidemenu.vue:721 의 'failed'/'processing' 비교뿐이라 'undefined' 와 '' 의 결과가 같아 효과 없는 변경(운영자 반려 사유)이라 rejected 로 내렸다. 대신 같은 API 응답을 PopWorkFlow.vue:118('myNoteBook') 과 MyAgentList.vue:301,307('MyNotebook') 이 다른 표기로 정확 일치 비교하는 문제를 골랐다 — 서버 표기와 무관하게 한쪽은 반드시 깨져 있고, 순수 매처 + 테스트로 증명 가능한 S 과제다.
- 확신 없는 곳: 서버가 실제로 어느 표기를 보내는지 미확인(그래서 "어느 화면이 지금 깨졌다" 는 단정을 과제서에서 금지). PopWorkFlow 의 project_id 가 빈 값일 때 워크플로 생성이 어떻게 실패하는지는 코드로만 추정.
- 구현자 주의: MyAgentList 의 'MyAvatar'/'etc' 비교와 myNoteBookStorage 의 캐시·폴백 로직은 범위 밖. .vue 는 import 교체 최소 diff, 매처는 mock 없는 순수 함수 테스트로. 기존 myNoteBookStorage.spec.js 415건 기준 통과 유지.
- 프로필은 2026-09-19 것이 현 코드와 일치해 다시 쓰지 않았다(node_modules 없음 → `npm ci` 선행은 프로필 검증 함정에 이미 있음).
- [러너 09:36] scout done — serviceCode 비교를 대소문자 무시 공용 매처로 통일 (PopWorkFlow / MyAgentList 가 같은 API 응답을 서로 다른 표기�

## 구현 노트
- 무엇/왜: 같은 appListGet 응답의 serviceCode 를 PopWorkFlow('myNoteBook')·MyAgentList('MyNotebook')·extractAppList(대소문자 무시)가 서로 다르게 읽던 것을 `myNoteBookStorage.isServiceCode` 하나로 통일했다(커밋 839f98d, 4개 파일 +77/-11). 매처 스펙 6건은 수정 전 코드에서 `isServiceCode is not a function` 으로 실패함을 확인한 뒤 구현했다.
- 검증: `npm ci` → `npm test` 421건 통과(기존 415 + 신규 6) → `npm run build:dev` 성공(dist 삭제) → `git status` 로 4개 파일만 변경 확인.
- 확신 없는 곳: (1) .vue 두 곳(PopWorkFlow:119, MyAgentList:299-305)은 컴포넌트 테스트 환경이 없어 코드 리뷰와 빌드 통과로만 확인. (2) `extractAppList(body, '')` 처럼 빈 serviceCode 를 넘기면 예전엔 serviceCode 가 빈 행을 골랐고 지금은 `isServiceCode` 가 false 를 돌려 빈 배열 — 호출부는 모두 상수를 넘기므로(getMyNoteBookList/getWorkToolsList) 실제 동작 변화는 없지만 계약이 미세하게 바뀐 지점이다. (3) 서버가 실제로 어느 표기를 주는지는 여전히 미확인 — 어느 화면이 "지금 깨져 있었는지" 는 단정하지 않았다.
- 일부러 하지 않은 것: MyAgentList 의 'MyAvatar'/'etc' 비교(:272,:281-285,:629)·`pickSectionByName` 의 menuName 매칭·PopWorkFlow 의 project_id 배열/문자열 처리·myNoteBookStorage 캐시/폴백 로직은 과제서 범위 밖이라 그대로 뒀다. `String(x) ?? ''` 류 정리도 끼워 넣지 않았다.
- 다음 역할 주의: `MENU_KEY.MY_NOTEBOOK`('myNoteBook', constants.js:11)과 `SERVICE_CODE_MY_NOTE_BOOK` 은 값이 같아야 한다는 스펙이 있다 — 한쪽만 바꾸면 테스트가 잡는다. PopWorkFlow 가 myNoteBookStorage 를 새로 import 하는데 순환 import 는 없다(myNoteBookStorage → commonStorage/userStorage/api/constants/common 만 의존, 빌드 통과).
- [러너 09:40] brief accepted — 채택 — 과제서의 근거(PopWorkFlow.vue:118 'myNoteBook' / MyAgentList.vue:301,307 'MyNotebook' 정확 일치 비교)가 현 코드와 정확히 일치�
- [러너 09:40] verify passed — 검증 2개 통과 (auto)
- [러너 09:40] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 09:40] pr created — https://github.com/hkjang/aiportal-front/pull/17
