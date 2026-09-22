- 과제: [수정 과제] 릴리즈 실패 원인 고정 — 진입 조건이 충족되지 않으면 이번 회차 기본 실행 대상은 globalLoading 병렬 요청 조기 종료 수정 (가치 4 / 위험 3 / 작업량 M)

- 왜: 릴리즈 단계는 6회 연속 같은 이유(다음 버전 결정 근거 부재)로 failed 이고, 원인 지점은 이 저장소가 아니라 외부 `/mnt/c/Users/USER/projects/aidev/release-prompt.md` 의 절차 구조다. 이번 정찰에서 원문을 직접 읽어 교착을 재확인했으므로(아래 근거), 같은 무변경 회차를 7번째로 반복하는 대신 저장소 안에서 실제로 고칠 수 있는 확인된 버그를 이번 회차의 기본 실행 대상으로 지정한다.

## 1순위 — 릴리즈 버전 결정 (진입 조건 있음. 이번 회차 기준 **미충족**)

이번 정찰에서 직접 확인한 근거(추측 아님):
- `release-prompt.md:21` (2항) — "최근 릴리즈들의 증가 패턴을 따르세요". 이 저장소의 로컬 태그는 0개(`git tag | wc -l` → 0), `git log --oneline -10` 에 릴리즈 커밋 양식 없음 → 따를 패턴이 없다.
- `release-prompt.md:22` (3항) — "이전과 같은 방식으로". 이전 방식이 존재하지 않는다.
- `release-prompt.md:24` (5항) — `skipped` 는 "태그도, **버전 파일도**, 릴리즈 노트도 없음" 일 때만. `package.json:3` 에 `"version": "0.0.0"` 이 있으므로 이 조건은 성립하지 않는다.
- `docs/RELEASE.md:83,84` — 정본이 "임의 패치 증가나 새 관례를 만들지 않는다", "이 문서로 실패를 skipped/released 로 바꾸지 않는다" 라고 명시.
- 즉 `released` 도 `skipped` 도 될 수 없는 구조적 교착이며, 해소는 사람의 결정(승인된 증가 정책) 또는 외부 절차 수정이다. 외부 파일은 이 세션의 허용 작업 디렉터리 밖이라 읽기만 가능하고 수정할 수 없다.

**진입 조건(하나라도 빠지면 1순위를 시작하지 말 것):** 운영자 또는 상위 회차가 출처와 함께 ① 승인된 증가 단위(예: patch) ② 태그 형식(주석/경량 포함) ③ 릴리즈 커밋 메시지 양식 ④ 릴리즈 노트 위치·언어 를 모두 제시했는가. brief·journal·운영자 지시 어디에도 없으면 미충족이다.

진입 조건이 미충족이면: **버전 파일·태그·CHANGELOG 를 만들거나 고치지 말고**, `docs/RELEASE.md` 도 건드리지 말고, 아래 2순위를 이번 회차의 실제 구현 대상으로 수행하라. 원장에는 1순위를 `pending` 으로 유지하고 "외부 절차 교착, 사람 결정 필요" 로 남긴다.

## 이번 회차 기본 실행 대상 — globalLoading 병렬 요청 시 스피너 조기 종료

- 왜: `src/utils/globalLoading.js` 의 전역 상태가 boolean 한 개(`isLoading`)뿐이라, 동시에 두 요청이 진행 중일 때 먼저 끝난 쪽의 `stopLoading()` 이 아직 로딩 중인 다른 요청의 스피너까지 꺼 버린다. 실제 동시 실행 경로를 확인했다: `src/views/ChatStorage/ChatStorageList.vue:55 getChatList()` 와 같은 파일 `:90 getAppList()` 가 각각 `try { startLoading() … await } finally { stopLoading() }` 구조로 독립 호출된다(`:57/:86`, `:92/:122`). 고치면 느린 요청이 끝날 때까지 스피너가 유지되어 사용자가 "다 끝난 줄 알고" 조작하는 상황이 사라진다.

- 수용 기준:
  1) `startLoading()` 이 2번 중첩 호출된 뒤 `stopLoading()` 이 1번 불리면 `isLoading.value === true` 로 유지되고, 두 번째 `stopLoading()` 에서 비로소 `false` 가 된다.
  2) 짝이 맞지 않는 여분의 `stopLoading()`(시작 없이 호출, 또는 시작 1회에 종료 2회)이 내부 카운터를 음수로 만들지 않고 `isLoading` 은 `false` 로 안정된다 — 다음 `startLoading()` 한 번으로 정상적으로 다시 켜져야 한다(스피너가 영구히 안 켜지거나 영구히 안 꺼지는 회귀 금지).
  3) 기존 단일 start/stop 동작과 `loadingTxt`/`hasTxt` 계약이 그대로다. 특히 텍스트 있는 start 가 진행 중일 때 텍스트 없는 start 가 겹쳐도 마지막 `stopLoading()` 전까지 표시 텍스트가 사라지지 않는지 결정하고 테스트로 고정할 것(현재는 무조건 덮어쓴다 — 이 동작 변경은 선택 사항이며, 바꾸지 않기로 했다면 "덮어쓴다"를 테스트로 명시 고정).
  4) 새 스펙이 **수정 전에 실패(Red)** 하고 수정 후 통과하며, 카운터를 되돌리면 다시 실패함을 확인해 인과를 증명할 것.

- 건드릴 파일:
  - `src/utils/globalLoading.js:10-30` — 모듈 스코프에 `const pending = ref(0)` 를 추가하고 `startLoading` 에서 증가, `stopLoading` 에서 `pending.value = Math.max(0, pending.value - 1)` 후 0일 때만 `isLoading.value = false`. `isLoading`/`loadingTxt`/`hasTxt`/`timeoutMs` 의 **export 형태와 이름은 바꾸지 말 것**(29개 파일 110개 호출부가 구조분해로 쓴다).
  - `tests/unit/globalLoading.spec.js` (신규) — 위 수용 기준 1~3을 고정. 순수 JS 모듈이라 기존 vitest/jsdom 설정으로 바로 돈다(Vue SFC 플러그인 불필요). 각 테스트는 모듈 전역 싱글톤을 공유하므로 `beforeEach` 에서 잔여 카운트를 0으로 되돌리는 방법을 명시할 것(`vi.resetModules()` + 동적 `import()` 권장 — 그래야 테스트 간 누수가 없다).
  - 호출부 `.vue` 파일은 **수정하지 말 것.** 확인한 범위(`src/views/ChatStorage/ChatStorageList.vue`, `src/views/Chat/Index.vue`)는 이미 `try/finally` 로 1:1 짝이 맞아 카운터만 넣으면 그대로 동작한다.

- 검증 명령 (이 저장소에서 실제로 도는 것. `node_modules` 가 없으므로 먼저 설치):
  ```
  npm ci
  npx vitest run tests/unit/globalLoading.spec.js   # 수정 전 Red 확인용
  npm test                                          # 전체 회귀 (현재 19개 spec)
  npm run build:dev                                 # 통과 후 dist/ 는 지울 것
  ```
  `npm run build` / `npm run lint` 는 **존재하지 않는다**(scripts 는 dev / build:* / preview:* / test / test:watch).

- 위험과 피할 것:
  - **가장 큰 위험은 스피너가 영구히 안 꺼지는 것이다.** 이 실패 모드는 지금의 조기 종료보다 나쁘다. 110개 호출부 전수 감사는 한 세션에 불가능하므로, 반드시 floor-at-0 를 넣고 수용 기준 2)를 테스트로 고정하라. `finally` 없이 start 하는 호출부를 하나라도 발견하면 그 파일을 고치려 하지 말고 브리프 범위를 줄여 보고하라.
  - 카운터를 `ref` 가 아닌 일반 변수로 두어도 되지만, `isLoading` 은 반드시 기존처럼 `ref` 로 남겨야 한다(템플릿 반응성).
  - 보호 경로 회피: `src/api/`(인증), `src/storage/auth*`, `src/router/`, `.gitlab-ci.yml`, `package.json` 의 `version`, `docs/RELEASE.md` 는 이번 회차에 건드리지 말 것.
  - 운영자 지시: 효과 없는 변경 금지 — 카운터를 넣고도 관측 동작이 그대로면 넣지 말 것. Red→Green 으로 실제 동작 변화를 증명하라.
  - 운영자 지시: 손으로 만든 대역이 아니라 실제 `globalLoading.js` 모듈을 import 해서 검증할 것.

- 차선 후보: README 「문제 해결」절의 존재하지 않는 `npm run build` 를 실제 스크립트(`build:dev` 등 모드별)로 정정하고 `docs/09` 의 끊긴 문서 링크를 정리 (가치 2 / 위험 1 / 작업량 S). globalLoading 감사 중 `finally` 없는 호출부가 여럿 발견되어 위 과제를 안전하게 끝낼 수 없을 때만 이것으로 바꿔라.
