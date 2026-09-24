# 과제서 (2026-09-23 정찰)

## 먼저: 우선 과제(릴리즈 실패)에 대한 이번 회차 판정

이번 정찰이 HEAD `66e6086` 에서 직접 재확인한 사실:

- `git tag | wc -l` → `0`. `.github/` 디렉터리 자체가 없음(`ls -a` 출력에 없음). `scripts/`·`Makefile`·`CHANGELOG*` 없음.
- `package.json:3` → `"version": "0.0.0"`, `"private": true`.
- 외부 절차 `/mnt/c/Users/USER/projects/aidev/release-prompt.md` 원문 직접 열람:
  - `:21`(2항) "최근 릴리즈들의 증가 패턴을 따르세요" — 따를 패턴이 존재하지 않음.
  - `:24`(5항) `skipped` 조건 = "태그도, **버전 파일도**, 릴리즈 노트도 없음" — `package.json:3` 이 있으므로 미충족.
  - `:22`(3항) 은 "못 고치면 ... failed 로 남기세요" 를 정상 종료로 인정함.
- 저장소 정본 `docs/RELEASE.md:83,84` 가 임의 증가·새 관례 제정과 failed→skipped/released 전환을 명시적으로 금지.

**결론: "릴리즈 워크플로 파일과 실패한 단계의 스크립트·테스트"에 해당하는 것이 이 저장소에 존재하지 않는다.** 실패 지점은 외부 `release-prompt.md` 의 2·5항 상호배타 조건이며, 그 파일은 이 워크트리 밖이라 구현자가 고칠 수 없다. 저장소 쪽에서 통과시키는 유일한 수단은 `package.json` 의 `version` 필드를 지워 5항 `skipped` 를 성립시키는 것인데, 이는 **게이트를 느슨하게 만들어 통과시키는 행위이므로 금지**다. 사람이 결정할 항목(증가 단위·태그 형식·릴리즈 커밋 양식·노트 위치·GitHub Release 사용 여부)은 7회째 미인계 상태다.

따라서 구현자는 릴리즈 관련 파일(`package.json`/`package-lock.json` 의 version, 태그, `docs/RELEASE.md`, CHANGELOG 신설)을 **일절 건드리지 말고**, 아래 실제 과제를 수행하라. 이번 회차를 또 무변경으로 끝내지 말 것.

---

## 이번 과제

- **과제: `ChatStorageDetail.vue` 의 고아 `startLoading()` 으로 법률 문서 열람 후 전역 스피너가 영구히 안 꺼지는 버그 수정 (가치 4 / 위험 2 / 작업량 S)**

- **왜:** `src/views/ChatStorage/ChatStorageDetail.vue:886 openLawViewer` 는 `startLoading()` 을 부르고 함수 어느 경로에서도 `stopLoading()` 을 부르지 않는다(같은 파일의 `startLoading|stopLoading` 10개 출현을 전수 확인: 342/366, 418/477, 485/535, 935/951 은 짝이 맞고 **887 만 짝이 없다**). `stopLoading` 은 `src/utils/globalLoading.js:28` 에서 전역 singleton `isLoading` 을 false 로 되돌리는 유일한 경로이고, 부모 `src/layouts/MainLayout.vue:22 setViewer` 는 `showViewer` 만 바꿀 뿐 로딩을 끄지 않으므로, 보관함 상세에서 법률 링크를 한 번 누르면 화면 전체가 스피너에 덮인 채 남는다.

- **핵심 근거(이대로 고칠 것):** 같은 이름의 형제 구현 `src/views/Chat/Index.vue:2309 openLawViewer` 는 **`startLoading()` 자체가 없다.** 두 구현은 `toggleMore` → `writeDoc(doc)` → `emit('set-viewer', doc)` → `await nextTick()` 로 동일하고, `writeDoc` 은 `src/storage/useDocStroage.js:6` 의 동기 localStorage 쓰기라 네트워크 I/O가 전혀 없다. 즉 스피너가 필요한 함수가 아니다. **올바른 수정은 `ChatStorageDetail.vue:887` 의 `startLoading()` 한 줄을 제거해 형제 경로와 동작을 일치시키는 것**이다. `stopLoading()` 을 뒤에 덧붙이는 방향은 두 경로를 계속 갈라 두므로 선택하지 말 것.

- **수용 기준:**
  1. `src/views/ChatStorage/ChatStorageDetail.vue` 의 `openLawViewer` 실행 후 전역 `isLoading` 이 true 로 남지 않는다.
  2. `ChatStorageDetail.vue` 안의 `startLoading` 출현 수와 `stopLoading` 출현 수가 같아지고, 나머지 4쌍(342/366, 418/477, 485/535, 935/951)의 `try/finally` 구조는 그대로다.
  3. `openLawViewer` 의 나머지 동작(`toggleMore.value = true`, `writeDoc(doc)`, `emit('set-viewer', doc)`, `await nextTick()`, `scrollToBottom(true)`)은 변경되지 않는다 — diff 는 실질 1줄이어야 한다.
  4. 테스트가 증명할 것: `src/utils/globalLoading.js` 의 `useGlobalLoading()` 이 **모듈 수준 singleton** 이라 서로 다른 호출자가 같은 `isLoading` ref 를 공유하고, `startLoading()` 뒤 `stopLoading()` 이 없으면 `isLoading.value` 가 true 로 남는다는 계약. (이 spec 이 버그의 "왜 영구히 안 꺼지는가"를 고정한다.)

- **건드릴 파일:**
  - `src/views/ChatStorage/ChatStorageDetail.vue:887` — 고아 `startLoading()` 제거. (`:886` 함수 본문 외에는 손대지 않음)
  - `tests/unit/globalLoading.spec.js` (신규) — `@/utils/globalLoading` 를 직접 import 해 실제 Vue `ref` 로 검증: (a) 두 번 `useGlobalLoading()` 을 호출해도 `isLoading` 이 같은 ref 인 것, (b) `startLoading('문구')` 가 `loadingTxt`/`hasTxt` 를 세우고 인자 없는 `startLoading()` 이 둘을 비우는 것, (c) `stopLoading()` 없이는 true 가 유지되고 `stopLoading()` 이 false 로 되돌리는 것. **모듈 singleton 이므로 각 테스트 끝에 `stopLoading()` 으로 정리해 테스트 간 누수를 없앨 것.**

- **검증 명령:**
  ```
  npm ci            # node_modules 없음. 처음이면 오래 걸림
  npm test          # 기준선: 20 파일 433 테스트 통과 (2026-09-22 회차 기록). 신규 spec 추가 후 21 파일
  npm run build:dev # SFC 수정이 빌드를 깨지 않는지. 끝나면 dist/ 삭제할 것
  ```
  수정 지점 직접 확인(수정 전/후 비교용, 이것만으로 증거 삼지 말 것):
  ```
  grep -n "startLoading\|stopLoading" src/views/ChatStorage/ChatStorageDetail.vue
  ```
  수정 전 887 에 `startLoading()` 이 있고 짝이 없음 → 수정 후 9개 출현, 4쌍 전부 짝.

- **위험과 피할 것:**
  - **`globalLoading` 에 참조 카운트를 넣지 말 것.** 다른 호출부에 아직 누수가 있다(`src/views/Support/Ocr/SupportOcr.vue:203-223` 은 `stopLoading()` 이 `setTimeout` 안에만 있어 catch 경로에서 누락). 카운터를 먼저 넣으면 0 으로 복귀하지 못해 스피너가 영구 유지되는, 현재보다 나쁜 회귀가 난다. 이번 범위는 `ChatStorageDetail` 한 곳이다.
  - **`SupportOcr.vue` 는 이번에 건드리지 말 것.** 성격이 다르고(비동기 + 폴링 + `setTimeout` 2초 후 라우팅) 판단이 필요하다 — 별도 과제로 남긴다.
  - `.vue` 는 현재 vitest 설정(`vitest.config.js` 에 `@vitejs/plugin-vue` 플러그인 없음)으로 **마운트 테스트가 불가능**하다. `@vue/test-utils` 도 devDependencies 에 없다. 따라서 4)의 증명은 순수 JS 모듈인 `globalLoading.js` 계약까지이고, SFC 호출부의 런타임 증명은 이번 세션에 **불가**하다. 이 한계를 구현 노트에 그대로 적을 것 — 소스 문자열 검사를 런타임 증거로 포장하지 말 것.
  - 릴리즈 관련 파일(version, 태그, `docs/RELEASE.md`, CHANGELOG 신설) 일절 금지. `.gitlab-ci.yml` 금지.
  - `src/views/Chat/Index.vue` 는 이미 올바른 형태이므로 수정하지 말 것(효과 없는 변경 금지).

- **차선 후보:**
  1. **vitest 에 SFC 지원 활성화** — `vitest.config.js` 에 `plugins: [vue()]` 추가(`@vitejs/plugin-vue@^6.0.1` 은 이미 devDependencies 에 있음, 신규 의존성 불필요). 증명은 `@vue/test-utils` 없이 `vue` 의 `createApp` + jsdom 으로 작은 기존 컴포넌트를 실제 마운트하는 spec 1개. 성공하면 위 과제의 SFC 런타임 증명 공백과 이후 모든 컴포넌트 테스트가 열린다. (가치 4 / 위험 2 / 작업량 M — 큰 컴포넌트는 의존성 때문에 마운트 못 할 수 있으니 작은 것부터)
  2. `docs/RELEASE.md:45` 의 `assert ... == '0.0.0'` 하드코딩 — **지금은 손대지 말 것**(근거 정본이 흔들림). 위 두 개가 모두 막혔을 때만 재검토.
