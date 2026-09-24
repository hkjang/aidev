# 회차 노트 2026-09-23-163413-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:34] base pinned — main@38a6311
- [러너 16:34] autonomy release — 

## 정찰 노트
- A(릴리즈)는 11회째 동일 교착이라 지시대로 재조사 없이 1줄만 확인(HEAD 38a6311, tag 0개, .github 없음, 워크트리 무변경)하고 무변경으로 고정했다. 실행 과제는 B로 배정한다.
- B를 고른 이유: 미-await 감사는 이번에 돌려 결함 0건으로 종결했고(useFileAccept는 await Promise.all 안, groupLogin은 form submit 동기 호출), showHeader/@:click/timeoutMs 세 후보는 "고치면 무엇이 옳은가"가 미확정이라 또 차선행이 된다. Alert callback 유실은 호출부 5곳이 전부 필수 후속 동작(redirectToLogin/onCancel)이라 기대 계약이 코드로 증명돼 있고 수정면이 globalAlert.js + App.vue:52 두 곳뿐이다(showAlert 쓰는 곳 grep 확인).
- 추측으로 적은 것: App.vue 마운트 가능 여부(pinia/RouterView/ToastManager 의존) — 이번 정찰은 npm ci 미실행이라 테스트를 돌려 보지 못했다. 기준선 24파일 458테스트도 직전 회차 기록 기준이다.
- 구현자가 조심할 것: @primary 와 @update:model-value 를 둘 다 연결하면 확인 버튼에서 callback 이 두 번 실행된다 — 단일 경로(update:model-value)로 수렴시키고 callback 은 꺼내서 null 로 비운 뒤 호출할 것.
- 범위 밖으로 못 박은 것: Base.vue:46 @:click, Alert 의 showHeader, Confirm 의 같은 계열 문제, interceptors/auth/authStorage 수정.
- [러너 16:38] scout done — 전역 Alert 의 후속 동작(callback)이 Escape 로 닫으면 유실되는 문제 수정 (가치 4 / 위험 3 / 작업량 M)

## 구현 노트
- 무엇을/왜: `openAlert` 의 callback 이 `App.vue` 에서 `@primary`(확인 버튼)에만 걸려 Escape 로 닫으면 유실됐다. `globalAlert.js` 에 `closeAlert()` 를 추가(callback 을 꺼내 null 로 비운 뒤 실행)하고 App.vue 를 `:model-value` + `@update:model-value="closeAlert"` 단일 경로로 바꿨다. 커밋 3b645d1.
- 확신 없는 곳: 없음에 가깝지만 **비평가가 먼저 볼 곳은 수용 기준 2 의 "1회성"** 이다. 과제서가 제안한 형태(닫은 뒤 다음 openAlert 후 닫기)는 `openAlert` 가 callback 을 항상 덮어쓰므로 변이 실험에서 실패하지 않는 무딘 테스트였다. 실제로 판별되는 두 계약(닫힌 뒤 `alertCallback.value === null` 직접 단정 / callback 이 새 Alert 을 열면 새 callback 이 유실되지 않음)으로 바꿔 각각 변이로 증명했다.
- 검증 못 한 것: 실제 브라우저에서의 leave 트랜지션 중 동작. jsdom 에서는 `fade` 트랜지션이 즉시라 "닫히는 중 Escape 재입력" 을 재현하지 못했다. 다만 `Base.vue:26` 이 `props.modelValue` 를 검사하므로 코드상 재진입 경로는 없다.
- 일부러 안 한 것: 과제서 지시대로 `Alert.vue`/`Base.vue`/`Confirm.vue` 미수정(Base.vue:46 `@:click` 오타, Alert 의 showHeader, Confirm 의 같은 계열 문제는 모두 범위 밖). `src/utils/alerts.js` 죽은 중복 모듈 제거는 동작 변화가 없어 차선으로만 남겼다. 릴리즈(A)는 11회째 무변경 — 재조사하지 않았고 `docs/RELEASE.md` 포함 아무것도 건드리지 않았다.
- 다음 역할이 조심할 것: `tests/unit/globalAlertCallback.spec.js` 는 실제 `App.vue` 를 마운트한다(pinia 플러그인 + RouterView 스텁으로 마운트 가능함을 확인했다). 전역 Alert 은 모듈 싱글턴이라 `beforeEach`/`afterEach` 에서 `showAlert`/`alertText`/`alertCallback` 을 정리하고 `unmount()` 로 window 리스너를 거둔다 — 이 정리를 빼면 다른 spec 으로 상태가 샌다. DB·네트워크는 필요 없다.
- [러너 16:44] brief accepted — 채택 — A(릴리즈)는 지시대로 무변경으로 두고, B 를 수용 기준 1~5 그대로 구현·검증했다. 다만 수용 기준 2 의 "1회성" 을
- [러너 16:44] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인한 것: 전체 스위트 직접 실행(25파일 463테스트 통과) + 변이 검증 — src/App.vue·src/utils/globalAlert.js 만 main 으로 되돌리니 신규 5개 중 3개가 실패한다. 구현자가 의심한 "수용 기준 2 의 1회성" 은 실제로 판별력 있는 계약으로 바뀌어 있었고 무딘 테스트가 아니다. 워크트리는 되돌린 뒤 무변경으로 복구했다(stash 잔여 0).
- callback 호출부는 6곳(노트의 5곳은 오기): interceptors.js:164/210/223, common.js:587, PopSimpleBotUpdate.vue:166/172. 전부 필수 후속 동작이라 Escape 로도 실행되는 새 계약이 안전하다. 보안/법무 차단 사유 없음 — 오히려 세션 만료 알림을 Escape 로 닫아도 로그인 이동이 보장된다.
- 못 본 것: 실제 브라우저의 leave 트랜지션 중 Escape 재입력(jsdom 한계, 구현자 자기신고와 동일). Base.vue:26 의 modelValue 가드로 코드상 재진입 경로는 없다고 판단했다.
- 승인이어도 남는 우려: globalAlert.js:36 alertCallback 이 소비처 0곳인데도 export 되어 있어 누가 showAlert 를 직접 false 로 만들면 같은 유실이 재발한다 — export 면 축소가 다음 후보. src/utils/alerts.js 는 importer 0곳 죽은 모듈인데 동명 closeAlert 를 export 해 오용 여지가 있다.
- 다음 회차 후보(같은 계열, 이번 범위 밖 확인): App.vue:51 Confirm 은 아직 v-model+@primary 라 Escape 시 cancelAction 유실. Base 가 인스턴스마다 window keydown 을 달아 Escape 한 번에 모든 팝업이 동시에 닫힌다. Base.vue:46 `@:click` 오타를 고치면 배경 클릭도 closeAlert 로 수렴하므로 그때 재검증 필요.
- [러너 16:46] review approved — 리뷰 승인 (risk=low)
- [러너 16:46] pr created — https://github.com/hkjang/aiportal-front/pull/25
- [러너 16:47] ci passed — 검사 없음 — 정책으로 허용
- [러너 16:47] merge done — 3b645d1
- [러너 16:49] release failed — 릴리즈 안 함: 릴리즈 관례 미확인으로 진입 조건 미충족(12회째). 이번 회차에 직접 재조회한 근거: (1) git tag 0개이며 remote.origin.tagOpt 미설정 + is-shallow-r
