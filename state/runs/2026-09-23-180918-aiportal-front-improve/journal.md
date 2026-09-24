# 회차 노트 2026-09-23-180918-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:09] base pinned — main@206764e
- [러너 18:09] autonomy release — 

## 구현 노트
- 무엇/왜: Escape 로 Confirm 을 닫으면 `cancelAction` 이 실행되지 않아 Chat 의 base64 붙여넣기 취소 처리(`Main.vue:910`/`Index.vue:307`)가 유실됐다 — 이 경로는 `preventDefault()` 를 먼저 하므로 붙여넣은 텍스트가 그대로 사라진다. `Confirm.vue` 에 `onBaseClose`(Base 닫힘 = 취소)를 두고, 동작이 새 팝업을 열어도 뒤따르는 닫힘 emit 이 지우지 않도록 `onPrimary/onSecondary` 의 emit 순서를 닫힘 먼저로 바꿨다. `App.vue`·`useConfirmStore.js` 는 무변경.
- 확신 없는 곳: ①`Confirm` 은 컴포넌트 갤러리 `UiComponents.vue:488` 에서도 v-model 로 쓰인다 — Escape 가 이제 `@secondary`(거기선 console.log)를 부른다. 의도상 맞다고 판단했으나 갤러리 화면은 실행해 보지 않았다. ②실제 브라우저에서 Chat 붙여넣기 시나리오를 돌려보지 않았다(jsdom 마운트 테스트로만 증명). ③`SupportSttList.vue:10` 의 Confirm import 는 템플릿 미사용(죽은 import)이라 영향 없다고 판단했다.
- 일부러 안 한 것: `Base.vue:46` 의 `@:click` 오타(배경 클릭 닫기)는 Base 를 쓰는 모든 팝업에 한꺼번에 영향이 가고 의도적 비활성 가능성이 있어 손대지 않았다. `showHeader`/title 건도 호출부가 Error 객체를 넘기고 있어 범위 밖.
- 다음 역할 주의: 새 스펙은 실제 `App.vue` 를 `attachTo: document.body` 로 마운트하고 window keydown 을 쓴다 — `afterEach` 에서 unmount + `document.body.innerHTML=''` 로 정리한다. 전역 Alert/Confirm 은 같은 `.popup-layer.alert` 선택자를 공유하므로 두 팝업을 동시에 여는 케이스를 추가할 땐 선택자를 좁혀야 한다.
- 릴리즈: 12회째 진입 조건 미충족으로 무변경. 태그·버전·CHANGELOG·릴리즈 노트·원격 전송 일체 없음.
- [러너 18:15] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인함: Confirm.vue 만 main 으로 되돌려 재실행 → 새 스펙 8개 중 5개 실패(Escape 3, 재오픈 순서 2). HEAD 전체 26파일 471테스트 통과. 테스트는 실제 결함을 잡는다.
- 확인함: confirmStore.open 전 호출부 전수 조사 — cancel 이 non-null 인 곳은 Chat/Main.vue:915, Chat/Index.vue:307(의도된 대상), PopEditApp.vue:81(빈 함수)뿐. Escape→secondary 전환의 부작용 호출부 없음. 구현 노트 ③(SupportSttList 죽은 import)도 맞다.
- 승인이어도 남는 우려: ①커밋 메시지에 없는 emit 순서 변경이 함께 들어갔으니 릴리즈 노트에는 동작 변경 2건으로 적을 것. ②Alert.vue:18-21 은 아직 옛 순서라 형제 규약이 어긋난다 — 다음 회차 후보. ③Confirm.vue:30 주석은 배경 클릭/헤더 X 를 말하지만 Base.vue:46 @:click 오타와 showHeader 미전달로 실제 경로는 Escape 뿐.
- 기존 결함(회귀 아님, 차단 아님): Base 의 window keydown 이 인스턴스마다 등록돼 Escape 한 번에 겹친 팝업이 모두 닫힌다 — PopEditApp 위 Confirm 에서 Escape 시 편집 입력이 사라진다. 최상위만 닫는 스택 처리가 과제로 남음.
- 못 본 것: 실제 브라우저에서의 Chat 붙여넣기와 UiComponents 갤러리 화면(코드 확인만). security/legal 차단 사유 없음 — 인증·식별자·비밀값·개인정보·의존성 변화 전무.
- [러너 18:17] review approved — 리뷰 승인 (risk=low)
- [러너 18:17] pr created — https://github.com/hkjang/aiportal-front/pull/26
- [러너 18:18] ci passed — 검사 없음 — 정책으로 허용
- [러너 18:18] merge done — 92ab47b
- [러너 18:19] release failed — 릴리즈 안 함: 릴리즈 진입 조건 미충족 (13회째). 이번 회차에 직접 재확인한 근거: git tag 0개; package.json version=0.0.0 이고 package-lock.json 루트/packages[""] 도 
