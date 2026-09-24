# 과제서 (2026-09-23 정찰)

이번 회차는 **A(우선 과제: 릴리즈) 무변경 확정** + **B(실제 구현 대상)** 두 부분이다.
A 는 저장소 안에 합법적 수단이 없어 지시대로 아무것도 만들지 않는다. B 를 구현하라.

---

## A. 우선 과제(릴리즈 실패) — 이번 회차도 무변경. 재조사하지 말 것

- 상태: 11회째 동일 교착. **버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋을 만들지 말고 원격에 아무것도 보내지 말 것.** 판정을 skipped/released 로 낮추지도 말 것.
- 이번 정찰이 확인한 사실(재조사 아닌 1줄 확인): HEAD `38a6311`, `git status --short` 0줄, `git tag` 0개, 저장소 루트에 `.github` 디렉터리 **없음**(`ls -a` 결과에 부재). 이전 회차 조사와 동일하다.
- 해소에 필요한 **사람 입력**(출처와 함께 인계되기 전까지 진입 불가): ①시작 버전(0.0.1 대 0.1.0)과 증가 단위 ②태그 형식·주석 태그 여부 ③릴리즈 커밋 메시지 양식 ④릴리즈 노트 위치·양식·언어 ⑤GitHub Release 사용 여부(이 저장소는 GitLab CI 배포, GitHub 릴리즈 이력 0).
- 구현자가 할 일: **A 에 대해 코드를 만들지 말고**, 원장/회차 노트에 "진입 조건 미충족으로 무변경(11회째)" 만 기록하고 B 로 넘어갈 것. `docs/RELEASE.md` 도 건드리지 말 것(근거 정본이 흔들린다).

---

## B. 이번 회차 실행 과제

- 과제: 전역 Alert 의 후속 동작(callback)이 Escape 로 닫으면 유실되는 문제 수정 (가치 4 / 위험 3 / 작업량 M)
- 왜: `openAlert(text, title, callback)` 의 callback 은 `App.vue:52` 에서 **`@primary` (확인 버튼) 에만** 연결돼 있는데, `Base.vue:25-33` 이 window keydown 으로 Escape 를 받아 `close()` 를 호출하므로 사용자가 Escape 로 닫으면 callback 이 한 번도 실행되지 않는다. 실제 callback 사용처 5곳이 전부 "반드시 일어나야 하는 후속 동작"이다 — `src/api/common/interceptors.js:164,210,223` 과 `src/utils/common.js:587` 의 `redirectToLogin()`(세션 만료 재로그인), `src/components/common/popup/Global/PopSimpleBotUpdate.vue:166,172` 의 `onCancel()`(수정 불가 팝업 닫기). 지금은 세션이 만료된 사용자가 Escape 를 누르면 로그인으로 가지 않고 죽은 화면에 남고, 수정 불가 상태의 편집 팝업이 그대로 열려 있다.
- 수용 기준:
  1) 전역 Alert 가 **어느 경로로 닫히든**(확인 버튼 / Escape) 등록된 callback 이 **정확히 한 번** 실행된다.
  2) callback 없이 연 Alert(대다수 호출부)은 닫을 때 아무 것도 실행되지 않고 예외도 나지 않는다. 닫힌 뒤 `alertCallback` 이 남아 다음 Alert 에서 재실행되지 않는다(1회성).
  3) 테스트는 **가짜 컴포넌트가 아니라 실제 배선**으로 증명한다: 실제 `globalAlert.js` 싱글턴에 `openAlert(..., cb)` 를 호출해 Alert 을 띄우고, 실제 DOM 이벤트(확인 버튼 `click`, `window` 의 `keydown{key:'Escape'}`)로 닫아 `cb` 호출 횟수를 단정한다.
  4) 되돌림 실험으로 무딘 테스트가 아님을 증명한다: 수정분을 되돌리면(= callback 을 `@primary` 에만 연결) Escape 케이스가 실패하고, callback 을 닫힐 때마다 지우지 않으면 "1회성" 케이스가 실패해야 한다.
  5) 기존 `tests/unit/alertPopup.spec.js` 를 포함한 전체 스위트가 통과한다.
- 건드릴 파일:
  - `src/utils/globalAlert.js` — `openAlert` 옆에 `closeAlert()` 추가: `showAlert.value=false` 로 내리고 `alertCallback.value` 가 함수면 **꺼내서 먼저 null 로 비운 뒤** 호출(재진입·중복 실행 방지). export 에 추가.
  - `src/App.vue:52` — `<Alert v-model="showAlert" ... @primary="alertCallback"/>` 를 `:model-value="showAlert"` + `@update:model-value="closeAlert"` 로 바꾸고 `@primary` 바인딩은 제거한다. `Alert.vue` 의 `onPrimary` 는 `emit('primary')` 후 `close()` 를 부르므로(`Alert.vue:18-21`) 확인 버튼 경로도 결국 `update:modelValue` 로 수렴해 **단일 경로**가 된다 — 이래야 이중 실행이 안 난다. `useGlobalAlert()` 구조분해에 `closeAlert` 추가.
  - `tests/unit/` 에 신규 spec 1개(예: `globalAlertCallback.spec.js`).
  - **`Alert.vue` / `Base.vue` / `Confirm.vue` 는 수정하지 말 것**(아래 위험 참조).
- 검증 명령(이 저장소에서 실제로 도는 것):
  - `npm ci` — node_modules 가 **비어 있다**(이번 정찰 확인: `ls node_modules | wc -l` = 0). 선행 필수, 수 분 소요.
  - `npx vitest run tests/unit/globalAlertCallback.spec.js` — 수정 전 Red 확인 → 수정 후 통과.
  - `npm test` — 전체. 기준선은 직전 회차 기록 기준 **24파일 458테스트**(이번 정찰은 미실행이므로 실제 기준선은 구현자가 수정 전에 한 번 돌려 확정할 것).
  - `npm run build:dev` — 통과 확인 후 생성된 `dist/` 는 삭제할 것. (`npm run build`, `npm run lint` 는 **없다**.)
- 위험과 피할 것:
  - **테스트 마운트 대상**: 프로덕션 배선을 통과시키려면 `App.vue` 를 마운트하는 것이 정석이나, `App.vue` 는 pinia 스토어 3개·RouterView·ToastManager·ocrStatusCheckStore 폴링을 끌고 온다. `createPinia()` 플러그인 + `RouterView` 스텁으로 마운트가 되는지 **먼저 15분 안에 확인**하고, 되면 `App.vue` 마운트로 증명하라. **안 되면 범위를 줄여 보고**하라 — 대신 실제 `Alert.vue` + 실제 `globalAlert.js` 싱글턴을 `App.vue:52` 와 **같은 바인딩**으로 마운트해 증명하고, 과제서 대비 무엇을 못 했는지 원장에 명시하라. (소스 문자열 검사는 증거로 금지.)
  - `Base.vue:46` 의 `@:click="close"` 오타(정상 코드는 :45 에 주석 처리)(배경 클릭 닫기 비동작)는 **이번 범위 밖**이다. 바로 위 줄에 정상 코드가 주석 처리돼 있어 의도적 비활성일 수 있고, 고치면 모든 팝업의 배경 클릭 닫기가 한꺼번에 살아난다. 건드리지 말 것.
  - `Alert.vue` 가 `showHeader` 를 넘기지 않아 `title` 과 X 버튼이 렌더링되지 않는 것(`Base.vue:11,50`)도 **범위 밖**이다. 지금 보이게 하면 `SupportOcrDetail.vue:88` 등 3곳이 title 자리에 넘기는 **Error 객체**가 화면에 찍힌다.
  - `Confirm.vue`(App.vue:51, `useConfirmStore`)도 Escape 로 닫으면 `cancelAction` 이 실행되지 않는 같은 계열 문제지만 이번 범위 밖이다. 같이 고치지 말고 아이디어로만 남겨라.
  - 보호 경로: `src/api/common/interceptors.js`, `src/api/auth.js`, `src/storage/authStorage.js` 는 **읽기만** 하라. 이번 수정은 이 파일들을 바꾸지 않고도 그들의 의도(세션 만료 시 재로그인)를 복구한다.
  - 전역 Alert 상태는 모듈 싱글턴이라 테스트 간 상태가 샌다. 각 케이스에서 `showAlert`/`alertCallback` 을 정리하고 window 리스너가 쌓이지 않게 `unmount()` 하라.
  - 운영자 지시: 관측 가능한 동작이 바뀌지 않는 수정은 넣지 말 것 — 이 과제는 Escape 경로의 동작이 실제로 바뀐다(되돌림 실험으로 증명하라).
- 차선 후보: **inf.* 미-await 전수 감사 결과를 근거로 한 후속 정리 — 이번 정찰이 이미 감사를 돌렸고 실제 결함은 0건이었다**(`grep -rn "inf\.[a-z]*\.[a-zA-Z]*\.call" src | grep -v await` → 3건: `src/utils/useFileAccept.js:87,88` 은 `await Promise.all([...])` 안이라 정상, `src/utils/common.js:477` 의 `inf.auth.groupLogin.call` 은 `interface.js:23-40` 이 form 을 만들어 submit 하는 동기 side-effect 라 await 가 무의미). 따라서 차선은 **`src/utils/alerts.js` 가 `globalAlert.js` 와 같은 역할의 중복 모듈이면서 import 하는 곳이 0곳인 죽은 코드**임을 확인하고(`grep -rn "utils/alerts" src` → 0건) 제거하는 것 — 운영자 지시 "정본을 하나로 유지할 것" 에 해당한다. 단 동작 변화가 없는 정리이므로 1순위가 성립할 때는 하지 말 것.
