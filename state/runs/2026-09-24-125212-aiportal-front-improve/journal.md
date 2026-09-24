# 회차 노트 2026-09-24-125212-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:52] base pinned — main@396b9ae
- [러너 12:52] autonomy release — 

## 정찰 노트
- HeaderAlarm 을 골랐다: 과거 3회차가 "보조 UI라 기대 계약 미확인" 으로 미뤘는데, 이번에 `Header.vue:129-131,166` 을 열어 `showAlarm` 이 **사용자 벨 클릭으로만** true 가 되고 `v-if` 재마운트당 `getList` 가 1회만 도는 것을 확인해 그 보류 사유를 없앴다 — 폴링은 `ocrStatusCheckStore` 쪽이지 이 컴포넌트가 아니다. 이미 4번 검증된 목록 표준형을 그대로 복사하면 되므로 위험 1.
- 제친 후보: 심플봇 `catch(e){}`(차선 — 기존 `simpleBotUpdateClose.spec.js` 가 현 계약을 고정해 케이스 갱신이 필요하고 위험이 더 높다), `MainLayout:107` 미정의 핸들러·`v-model` 잔재(둘 다 관측 변화 0 = 무효 변경), `loading.vue` 타이머·`globalLoading`(계약 미확정 + 회귀 위험 M).
- 추측으로 적은 것(미확인): `npm test` 기준선 **38파일/579테스트는 이번 회차에 재실행하지 않았다** — 기록값이므로 구현자가 수정 전에 직접 돌려 확정할 것. `HeaderAlarm` 을 마운트하는 스펙이 아직 0건이라 `IconBtn`/`SmLoading`/메모리 라우터 조합이 jsdom 에서 곧바로 마운트되는지도 미확인이다.
- 구현자가 조심할 것: ①`catch` 의 `alarmList.value = []` 는 `v-if` 재마운트라 빼도 동작 변화 0 — **건드리면 무효 변경으로 걸린다** ②토스트는 반드시 `if (e !== 'COM')` 가드 **뒤**에 ③`HeaderAlarm` 은 `{immediate:true}` 라 `open:true` 마운트로 조회가 나간다(Base.vue 팝업과 반대) ④`writeUser` 입력 키는 `userId`, 읽기는 `readUser()?.user_id`.
- A(릴리즈)는 27회째 무변경이 정답이다. 이번 회차에 tag 0개 / version 0.0.0 / `.github` 부재 / 워크트리 clean(HEAD 396b9ae)을 직접 재실측해 확정했으니 **구현자는 재조사하지 말 것.**
- [러너 12:57] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을 왜: `HeaderAlarm.getList` 에 목록 조회 표준형(업무 실패 `toast` + `return`, `catch` 는 `'COM'` 가드 뒤 `toast`)을 적용했다. 조회 실패가 '알림이 없습니다.' 와 구분되지 않았고, 실패 응답이 `body` 를 달고 오면 `markAsRead()` 가 헤더 `new` 배지까지 지웠다. 기대 계약은 발명하지 않았다 — 같은 엔드포인트를 쓰는 `ocrStatusCheckStore.checkOcrStatus:218` 이 이미 같은 `isSuccess` 가드를 갖고 있는 형제 비대칭이다.
- 확신 없는 곳: ①변이 ②(`return` 제거)가 잡는 케이스는 **업무 실패 응답에 `body.status`/`todayOcrList` 가 실려 온다는 가정**에 의존한다. 백엔드가 별개 저장소라 그 응답 형태를 실물로 확인하지 못했다 — 근거는 형제 `checkOcrStatus` 가 같은 이유로 `body` 를 읽지 않는다는 코드뿐이다. ②토스트 문구('알림 조회에 실패하였습니다.' / '…중 오류가 발생하였습니다.')는 형제 어법을 따랐을 뿐 기획 확인은 없다.
- 일부러 하지 않은 것: `catch` 의 `alarmList.value = []` 와 `console.error`(정찰 지시대로 — `v-if` 재마운트라 동작 변화 0), `getList:56-60` 의 사실상 죽은 `list.length` 분기(무효 변경이라 ideas 로 분리), `HeaderShare`(열어 보지 않음, ideas 로 분리).
- 다음 역할이 조심할 것: `npm test` 기준선은 프로필의 38/579 가 아니라 **38/583** 이었고 이번 변경 후 **39/589** 다. 새 스펙은 `ocrStatusCheckStore` 싱글턴을 케이스마다 `createDefaultStatus()` 로 되돌린다 — 빼면 배지 케이스가 앞 케이스에 오염된다. `node_modules` 는 비어 있으므로 `npm ci` 선행 필수.
- 릴리즈(우선 과제): 27회째 무변경. 정찰 지시대로 재조사하지 않았고 버전·태그·노트·원격 전송 일체 없음.
- [러너 13:02] verify passed — 검증 2개 통과 (auto)
- [러너 13:02] pr created — https://github.com/hkjang/aiportal-front/pull/39
- [러너 13:02] merge stopped — 긴급 중지
