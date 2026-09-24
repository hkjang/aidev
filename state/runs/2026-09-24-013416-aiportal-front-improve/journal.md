# 회차 노트 2026-09-24-013416-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:34] base pinned — main@1a8cb85
- [러너 01:34] autonomy release — 

## 정찰 노트
- A(릴리즈)는 18회째 진입 조건 미충족을 워크트리에서 직접 재조회해 확정했다(태그 0 / version 0.0.0 / `.github`·CHANGELOG·VERSION·scripts·Makefile 전무 / `CI_COMMIT_TAG` 0건). 고칠 워크플로 파일이 저장소에 없으므로 무변경이 유일한 합법 결론이고, 판정을 skipped/released 로 낮추지 않았다.
- B 로 `Sidemenu.onDeleted`(:499)를 골랐다. 형제 `onSaved`(:539 가드 있음)·`gotoShare`(:444 finally 로 닫음)와 나란히 있는 비대칭이라 기대 계약이 **코드에 이미 증거로 남아 있다** — 계약 확정이 선행 조건인 다른 후보들(loading 타이머, showHeader, `tryUpdateAppIcon` 삼킴)을 이 이유로 제쳤다.
- 확신 없는 곳: `Sidemenu` 마운트 하네스가 이 저장소에 전례가 없는 것이 유일한 실질 비용이다. 과제서의 5단계 렌더 경로(`getSideList` → `getHistoryList` → 앱 펼치기 → `toggleMore` → 삭제 버튼)는 코드를 읽어 구성한 것이지 **실행해 본 것이 아니다**(`node_modules` 가 비어 `npm ci` 를 돌리지 않았다). 앱 펼치기 트리거(`isOpen`/`toggleApp`)의 정확한 DOM 은 미확인.
- 구현자가 조심할 것: `item.item.prj_id`(:468) 때문에 핸들러 직접 호출 우회는 TypeError 가 난다 — 반드시 실제 emit 경로로 돌릴 것. `openedId` 를 `finally` 로 옮기되 `historyMap` 갱신·`getHistoryList` 재조회는 성공 경로에 그대로 둘 것(과거 교훈: `finally` 에 성공 전용 동작 금지).
- 5단계 경로를 세션 안에 구동하지 못하면 대역 컴포넌트로 대체하지 말고 범위를 줄여 보고하고 차선(심플봇 `tryUpdateAppIcon` 삼킴)으로 넘어갈 것.
- [러너 01:38] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `Sidemenu.vue:499` `onDeleted` catch 에 `if (e == 'COM') return` 한 줄 추가. 인터셉터가 이미 전역 Alert 을 띄우고 'COM' 으로 거절한 오류에 토스트가 겹치던 문제로, 같은 파일 `fetchAppList`(:233)·형제 `onSaved`(:540)와 계약을 맞춘 것이다.
- **과제서 지시 중 하나는 근거를 확인해 기각했다**: `openedId` 를 `finally` 로 옮기라는 부분. `PopAppChatSetting.DelCt` 가 `'deleted'` 직후 `'close'` 를 emit 하고 `Sidemenu.vue:740` 의 `@close="openedId = null"` 이 이를 비우므로 성공·실패 모두 드롭다운은 이미 닫힌다 — 무효 변경이라 하지 않고, 그 사실을 고정하는 테스트 케이스를 대신 넣었다. 비평가는 이 판단을 먼저 보기 바란다.
- 확신 없는 곳: 신규 하네스는 `showDelete=true` 인 사이드메뉴 경로만 통과한다. 같은 `PopAppChatSetting` 을 쓰는 다른 화면(보관함 등)은 열어 보지 않았다. `settle()` 의 마이크로태스크 회수는 기존 스펙에서 복사한 경험값이라 느린 기기에서의 여유는 미확인.
- 일부러 하지 않은 것: `getHistoryList`(:416) 의 bare catch(설정 오류까지 삼켜 빈 목록으로 보인다 — 하네스 구축 중 실제로 당했다)는 '조회 실패를 알릴 것인가' 계약이 미확인이라 보류로만 올렸다. 릴리즈는 18회째 진입 조건 미충족으로 무변경(태그 0 / version 0.0.0 / `.github`·CHANGELOG·VERSION·scripts·Makefile 전무 / `CI_COMMIT_TAG` 0건을 워크트리에서 직접 재확인).
- 다음 역할이 조심할 것: **이 스펙은 `VITE_PYTHON_API_TARGET` 스텁이 없으면 조용히 빈 목록이 되어 엉뚱한 곳에서 실패한다**(`api/index.js:16` 이 던지고 `getHistoryList` 가 삼킨다). 히스토리 대역 응답은 `thumbnail_user.message`·`application_id` 표기를 써야 `mapRows` 의 `.filter(x => x.text)` 를 통과한다.
- 검증: `npx vitest run` 31파일 509테스트 통과(기준선 30/503), `npm run build:dev` 통과 후 `dist/` 삭제. 변이 1건(가드 줄 제거)으로 같은 2건만 재실패함을 확인.
- [러너 01:44] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- approve / low: main 대비 1줄 가드와 신규 테스트만 변경. 실제 결함 및 보안·법무 차단 근거 없음.
- 실제 DOM 삭제 → pInf → 인터셉터의 두 BZ01 분기와 일반 예외·업무 실패 단언 확인. 가드 제거 시 중복 토스트 단언이 실패하는 구조이며 변이는 직접 재실행하지 않음.
- deleted 직후 close emit으로 드롭다운이 닫힘을 확인해 finally 생략 판단에 동의. npm test 31파일/509테스트 및 diff --check 통과, 소스 무변경.
- 세션 만료·지연 응답·다른 화면의 삭제 동작과 빌드·릴리즈 근거는 이번 리뷰에서 재검증하지 않음. 고정 settle 대기는 향후 비동기 변경 시 보완 여지.
- [러너 01:45] review approved — 리뷰 승인 (risk=low)
- [러너 01:45] pr created — https://github.com/hkjang/aiportal-front/pull/31
- [러너 01:46] ci passed — 검사 없음 — 정책으로 허용
- [러너 01:46] merge done — 75fe530
- [러너 01:47] release failed — 릴리즈 안 함: 릴리즈 관례가 저장소에 존재하지 않아 다음 버전을 근거로 결정할 수 없다(19회째 동일 원인). 이번 회차 워크트리 HEAD 0031fde 에서 직접 �
