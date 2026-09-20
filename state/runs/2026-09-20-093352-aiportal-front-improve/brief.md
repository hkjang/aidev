# 과제서 2026-09-20 — aiportal-front

- 과제: serviceCode 비교를 대소문자 무시 공용 매처로 통일 (PopWorkFlow / MyAgentList 가 같은 API 응답을 서로 다른 표기로 정확 일치 비교) (가치 3 / 위험 2 / 작업량 S)
- 왜: 같은 API(`inf.athena.appListGet`, `serviceMenu=workTools`)의 `serviceCode` 를 `src/components/common/popup/Global/PopWorkFlow.vue:118` 은 `=== 'myNoteBook'`, `src/views/MyAgent/MyAgentList.vue:301,307` 은 `=== 'MyNotebook'` 으로 비교한다 — 서버가 어느 표기를 주든 두 화면 중 하나는 반드시 빈 결과를 얻는다(PopWorkFlow 는 노트북 project_id 를 '' 로 돌려 워크플로 생성이 노트북 프로젝트에 붙지 않고, MyAgentList 는 '나의 노트북' 섹션이 비고 `writeMyList([])` 로 빈 캐시를 쓴다). 2026-09-08 회차에서 `myNoteBookStorage.extractAppList` 만 대소문자 무시로 고쳤고 나머지 두 읽기 경로는 그대로라, "같은 값을 읽는 경로를 모두 같게" 맞추는 운영자 규칙에 걸린 상태다.
- 수용 기준:
  1) `src/storage/myNoteBookStorage.js` 에 순수 매처 `isServiceCode(row, code)` (또는 `matchesServiceCode`) 를 export 하고, 기존 `extractAppList` 도 이 매처를 쓰도록 바꿔 기존 `tests/unit/myNoteBookStorage.spec.js` 가 그대로 통과한다(동작 변화 없음).
  2) `PopWorkFlow.vue:118` 의 `body.find(...)`, `MyAgentList.vue:299-302` 의 `noteRows` 필터, `MyAgentList.vue:307` 의 필터가 모두 이 매처를 쓴다 — 서버가 `myNoteBook` / `MyNotebook` / `MYNOTEBOOK` 어느 표기로 보내도 세 경로가 같은 행을 고른다. 리터럴은 `SERVICE_CODE_MY_NOTE_BOOK`(myNoteBookStorage.js:11, 현재 미export) 을 export 해 쓰고 `src/utils/constants.js:11` 의 `MY_NOTEBOOK: 'myNoteBook'` 과 값이 같음을 유지한다(두 상수를 합치려 들지 말 것 — constants.js 는 라우팅 MENU_KEY 용도).
  3) 테스트: `tests/unit/myNoteBookStorage.spec.js` 에 매처 스펙 추가 — (a) `'myNoteBook'`/`'MyNotebook'`/`'MYNOTEBOOK'` 모두 true, (b) `'myNoteBookX'`, `'multiModal'`, `''`, `undefined`, 비객체 row → false, (c) 실제 export 된 `extractAppList` 가 매처를 통해 같은 결과를 내는 것(mock 없이 순수 함수). 수정 전 코드 기준으로 새 스펙이 실패(매처 미존재)함을 확인하고 적는다.
- 건드릴 파일:
  - `src/storage/myNoteBookStorage.js` — `SERVICE_CODE_MY_NOTE_BOOK`/`SERVICE_CODE_WORK_TOOLS` export, `isServiceCode` 추가, `extractAppList:53-60` 내부 필터를 매처로 교체
  - `src/components/common/popup/Global/PopWorkFlow.vue:118` — `getMyNoteBookProjectId` 의 `find` 를 매처로
  - `src/views/MyAgent/MyAgentList.vue:299-308` — `noteRows` 필터와 `myNoteBookApps` 필터를 매처로(둘이 같은 조건이니 `myNoteBookApps` 는 `noteRows.flatMap` 으로 줄여도 됨)
  - `tests/unit/myNoteBookStorage.spec.js` — 매처 테스트 추가
- 검증 명령: `cd <worktree> && npm ci && npm test` (vitest run, 현재 415건 통과가 기준) → `npm run build:dev` (1~2분, 500kB 청크 경고는 정상, 끝나면 `rm -rf dist`) → `git diff --stat` 으로 변경이 위 4개 파일에 한정됨을 확인.
- 위험과 피할 것:
  - `MyAgentList.vue:271` 의 `'MyAvatar'`/`'etc'` 비교와 `:629` 의 `section.serviceCode === 'etc'` 는 건드리지 말 것(이 과제 범위는 노트북 코드 하나; 그쪽은 표기 불일치 증거가 없다).
  - `myNoteBookStorage.readMyList/readWorkList/checkAppExists` 의 캐시·폴백 로직을 바꾸지 말 것(2026-09-08 에 테스트로 고정됨).
  - 서버가 실제로 어느 표기를 주는지는 미확인 — 어느 쪽이 "지금 깨져 있는지" 단정해서 커밋 메시지에 쓰지 말고 "두 경로가 다른 표기를 정확 일치 비교" 로만 서술할 것.
  - `.vue` 는 테스트 환경이 없으므로 매처를 순수 함수로 두고 .vue 는 import 만 바꾸는 최소 diff 로. `PopWorkFlow.vue` 의 `project_id` 배열/문자열 처리(:119)는 그대로 둔다.
  - 효과 없는 변경 금지 규칙: `String(x) ?? ''` 류 정리를 이 과제에 끼워 넣지 말 것(아래 ideas.json 에 rejected 사유 있음).
- 차선 후보: Header.vue OCR 폴링을 "대기(status '0') 항목이 있을 때만 유지" 로 바꾸기 — `ocrStatusCheckStore.checkOcrStatus:223-249` 에서 `todayOcrList` 에 대기 항목이 없고 status 도 'complete' 가 아니면 `stopPolling()`. 단 변환 직후 첫 폴링에 서버가 아직 항목을 등록하지 않았을 수 있어 `resetStatus` 의 `lastChecked` 기준 60초 유예가 필요하고, 서버 응답 semantics(미사용자에게 status 가 무엇인지)가 미확인이라 1순위보다 위험이 높다(M).
