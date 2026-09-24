# 회차 노트 2026-09-23-215420-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:54] base pinned — main@38cc2c3
- [러너 21:54] autonomy release — 

## 정찰 노트
- 우선 과제(릴리즈)는 이번에도 진입 조건 미충족이다. 워크트리에서 직접 재확인했다: git tag 0개, package.json version=0.0.0, .github 없음, 작업트리 청정(HEAD 38cc2c3). 실패 지점이 워크트리 밖이라 저장소 안에 합법적 수단이 없어 무변경으로 남기라고 과제서에 못박았다.
- B 로 PopSimpleBotUpdate.onCreate 의 finally 가 실패에도 emit(update)+close() 하는 문제를 골랐다. 형제 PopSimpleBot.onCreate 가 성공 경로에서만 emit+close 해 기대 계약이 코드에 남아 있고, 호출부 4곳이 모두 목록 새로고침 콜백을 넘기므로 사용자 피해(입력 유실 + 거짓 갱신)가 관측 가능하다.
- 제친 후보: loading.vue 타이머/globalLoading 참조 카운트(기대 계약 미확정 — 회귀 위험), alerts.js 죽은 모듈·미사용 import(동작 변화 없음 — 운영자 지시상 무효 변경 금지), showHeader(호출부 정리가 선행).
- 확신 없는 곳(추측): validateCreateApp 이 요구하는 정확한 필드 집합은 이번에 다시 읽지 않았다(지난 회차 기록 기준 name+description 으로 적음). tryUpdateAppIcon 의 실패를 성공 판정에 넣지 않는다는 것은 현재 동작 보존일 뿐 확인된 계약은 아니다.
- 구현자가 조심할 것: emit(update) 중복(목록 2회 새로고침) 금지, App.vue·useSimpleBotUpdateStore 무변경, 실패 후 팝업이 열린 채 남으므로 사용자 직접 닫기 경로를 케이스 하나로 확인할 것.
- [러너 21:58] scout done — 심플봇 **수정** 팝업이 저장 실패에도 팝업을 닫고 부모에 'update' 를 알려 사용자의 수정 내용이 유실되는

## 구현 노트
- 무엇/왜: `PopSimpleBotUpdate.onCreate()` 의 `finally` 에 있던 `emit('update', true)` + `close()` 를 성공 경로 마지막(`await tryUpdateAppIcon()` 뒤)으로 옮기고 `finally` 에는 `isSendLoading` 해제만 남겼다. 저장 실패에도 팝업이 닫혀 입력이 유실되고 목록이 거짓 갱신되던 문제. `catch` 의 토스트 앞에 `if (e == 'COM') return` 을 추가해 인터셉터 전역 Alert 과의 중복 안내를 없앴다. 신규 스펙 `tests/unit/simpleBotUpdateClose.spec.js` 10건(adapter 한 겹만 대역, 나머지는 실제 배선).
- 확신 없는 곳: ①`isSendLoading` 은 `script setup` 이라 직접 단정할 수 없어 "실패 후 다시 누르면 `/app/create` 가 2회 나간다"는 관측 가능한 대체 증거로만 확인했다. ②실패 후 팝업이 유지되면 신규 첨부(`oldYn != 'Y'`)가 재시도마다 새 타임스탬프 이름으로 다시 업로드된다 — 서버가 중복 문서를 쌓는지는 백엔드가 없어 확인 못 했다(ideas.json 에 후속으로 남김). ③`tryUpdateAppIcon()` 실패를 성공 판정에 넣지 않는 것은 기존 동작 보존일 뿐 확인된 계약이 아니다 — 스펙에 그 현 동작을 고정하는 케이스를 넣어 뒀다.
- 일부러 안 한 것: `PopSimpleBot.vue`(생성 팝업) 무변경 — 이미 올바르고 과제서가 금지했다. `App.vue`·`useSimpleBotUpdateStore.js` 무변경. 우선 과제 A(릴리즈)는 진입 조건 미충족으로 재조사 없이 완전 무변경(15회째).
- 다음 역할 주의: 이 스펙은 네트워크 없이 돈다(axios adapter 대역). 폼 검증이 첨부 1개 이상까지 요구하므로 팝업을 닫힌 상태로 마운트한 뒤 `setProps({modelValue:true})` 로 열어 실제 `/app/{id}` 상세 응답으로 폼을 채운다 — `appDetailBody()` 모양을 바꾸면 전체 케이스가 무너진다. 전역 토스트/Alert 싱글턴은 `afterEach` 에서 비운다.
- [러너 22:03] brief accepted — 채택 — A(릴리즈)는 지시대로 재조사 없이 무변경으로 두고, B 를 수용 기준 1~6 그대로 구현·검증했다. 과제서가 "`name`·`
- [러너 22:03] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 신규 스펙을 main 의 `PopSimpleBotUpdate.vue` 로 되돌려 실제로 돌려 10건 중 6건 실패를 확인했고, 전체 스위트는 29파일 497테스트 통과(기준선 487+10).
- 구현자의 '확신 없는 곳' 3건 모두 확인: `isSendLoading` 은 재클릭 시 `/app/create` 2회 관측으로 대체 검증됨, 아이콘 실패 무시는 현 동작으로 고정됨, 재시도 시 신규 첨부 재업로드는 실재하나 수정 전에도 재입력 시 동일했으므로 회귀 아님(백엔드 없어 서버 중복은 미확인).
- 남는 우려(다음 회차): 실패로 열린 팝업을 Escape/배경 클릭으로 닫으면 `onUpdate()` 만 타서 store 의 `editItem`/`updateCallback` 이 남는다 — 이번 diff 소관은 아니나 도달 빈도가 올라갔다. 첨부 재업로드 중복도 후속 후보.
- 못 본 것: `npm run build:dev` 미실행(.vue 7줄 변경이고 vitest 가 plugin-vue 로 이미 컴파일·마운트하므로 생략). `catch` 의 `if (e == 'COM') return` 은 같은 오류 경로라 범위 이탈로 보지 않았다.
- 우선 과제 A(릴리즈)는 진입 조건 미충족으로 무변경 — 릴리즈 노트에 16회째 no-change 로 남길 것.
- [러너 22:05] review approved — 리뷰 승인 (risk=low)
- [러너 22:05] pr created — https://github.com/hkjang/aiportal-front/pull/29
- [러너 22:06] ci passed — 검사 없음 — 정책으로 허용
- [러너 22:06] merge done — c19bf5c
- [러너 22:07] release failed — 릴리즈 안 함: 릴리즈 진입 조건 미충족(16회째). 워크트리(HEAD 5d948b7)에서 직접 재조회한 근거: git tag 0개, package.json/package-lock.json version 모두 0.0.0 이며 �
