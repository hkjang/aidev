# 과제서 (2026-09-23 12:04 정찰) — 기준 HEAD 595c0c5, 브랜치 auto/2026-09-23-1204

- 과제: [수정 과제] 릴리즈 실패의 저장소 측 수단 부재 확정 — 진입 조건 미충족이면 기본 실행 대상은 ChatStorageDetail.vue:887 고아 startLoading() 제거 (가치 5 / 위험 2 / 작업량 S)

- 왜: 배정된 릴리즈 실패는 "다음 버전·태그·릴리즈 노트 관례를 확정할 근거가 없음" 이며, 이번 회차에 워크트리에서 직접 재확인한 결과 원인은 저장소 코드가 아니라 사람이 정해야 하는 관례의 부재다(아래 근거). 따라서 저장소 안에서 이 실패를 코드 수정으로 없앨 방법이 없고, 억지로 통과시키는 유일한 수단(package.json 의 version 필드 삭제 → skipped 조건 충족)은 "워크플로를 느슨하게 만들어 통과" 에 해당하므로 금지다. 대신 8회 연속 무변경을 반복하지 않도록, 진입 조건이 미충족일 때의 기본 실행 대상을 실물로 확인된 사용자 영향 버그 하나로 지정한다.

## A. 릴리즈 실패에 대해 이번 회차에 직접 확인한 것 (재조사 불필요 — 그대로 인용해도 된다)

- `.github` 디렉터리 없음 → **이 저장소에는 GitHub Actions 워크플로 파일이 존재하지 않는다.** 배정문이 말하는 "워크플로" 는 외부 aidev 실행기(`bin/run.sh release_project` → `release.json` → `bin/gate.py`)이며 워크트리 밖이다. 따라서 "워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 고친다" 를 이 저장소 안에서 수행할 대상이 없다.
- `git tag` 0개. `git status --porcelain` 0줄(clean).
- `package.json:3` = `"version": "0.0.0"`, `"private": true`. 증가 이력 없음(이전 회차들이 `git log -p --follow` 로 확정, 이번 회차 재확인 생략).
- `.gitlab-ci.yml`: `ofc_build_main` 등이 `$CI_COMMIT_BRANCH == "main"` 에서 전용 Runner 의 `BUILD_DIR` 로 `git reset --hard` 후 `npm run build:ofc` → `ofc_deploy_main` 이 `cp -rf "$BUILD_DIR"/dist/* "$DEPLOY_DIR"`. **브랜치 push 기반 배포이며 태그·버전·릴리즈 자산을 전혀 사용하지 않는다.** 태그 신설은 파이프라인이 소비하지 않는 새 관례가 된다.
- `docs/RELEASE.md` 말미(83-84행 근처): "현재 근거로 임의 패치 증가나 새 관례를 만들지 않는다", "버전 파일이 있으므로 skipped 조건은 충족하지 않는다. 이 문서로 실패를 skipped/released 로 바꾸지 않는다". AGENTS.md 도 "릴리즈 판정 조건을 완화하지 않는다" 를 명시.
- `npm test`/`npm run build:dev` 는 이번 회차 **미실행**(`node_modules` 없음, 릴리즈 실패는 앱 CI 실패가 아니므로 재현 대상이 아니다).

### 진입 조건 (이 중 하나라도 없으면 아래 B 로 간다)
1. 사람이 승인한 **증가 단위**(0.0.0 에서 시작할 첫 버전)와 **태그 형식**(`v0.0.1` 대 `0.0.1`, 주석 태그 여부)이 출처와 함께 인계되었는가.
2. **릴리즈 커밋 양식·릴리즈 노트 위치·언어**, GitHub Release 사용 여부가 인계되었는가.
3. 없으면: 저장소 파일·커밋·태그·CHANGELOG 를 만들지 말고, 원격 전송(push/tag/gh release)도 하지 말고, 이 과제서의 A 절을 근거로 `failed` 유지 사유만 원장에 5줄 이내로 남긴 뒤 **B 를 실행한다**. (이 판정을 `skipped`/`released` 로 낮추지 말 것.)

## B. 기본 실행 대상 — ChatStorageDetail.vue:887 고아 `startLoading()` 제거

- 왜: `src/views/ChatStorage/ChatStorageDetail.vue:886-902 openLawViewer` 는 `startLoading()` 을 부른 뒤 `writeDoc(doc)`(동기 localStorage 쓰기), `emit('set-viewer', doc)`, `await nextTick()`, `scrollToBottom(true)` 만 하고 **어떤 경로에서도 `stopLoading()` 을 부르지 않는다**. 네트워크 I/O 가 전혀 없으므로 스피너를 켤 이유 자체가 없다. 결과: 채팅 저장 화면에서 법률 문서 링크를 누르면 전체 화면 로딩 오버레이가 **60초** 덮고 있다가 "네트워크 오류입니다" 문구를 띄운 뒤 꺼진다.
- 60초의 근거(추측 아님): 전역 스피너는 `src/App.vue:52` 의 `<Loading v-model="isLoading" :hasTxt :loadingTxt @timeout="stopLoading" />` 이고 여기서 `:timeoutMs` 를 **넘기지 않는다** → `src/components/common/loader/loading.vue:24-26` 의 기본값 `Number(apiTimeOutMs) || 30000`, `src/utils/config.js:1` `apiTimeOutMs = 60000`. (`globalLoading.js:13` 의 `timeoutMs = 600000` 은 export 되지만 아무도 소비하지 않는다 — 이번 과제에서 **건드리지 말 것**.)
- 같은 파일의 나머지 네 쌍은 정상이다: 342/366, 418/477, 485/535, 935/951. `MainLayout.vue:22 setViewer` 도 로딩을 끄지 않는다. 형제 구현(Chat/Index.vue)에는 `startLoading` 자체가 없다.
- 수용 기준:
  1) `openLawViewer` 실행 후 전역 `isLoading` 이 `false` 로 남는다(호출 전후 모두 false). 뷰어 열기 동작(`set-viewer` emit, `writeDoc` 호출, `toggleMore`)은 그대로다.
  2) `startLoading` 을 추가로 호출하거나 참조 카운트를 도입하지 않는다 — **887 한 줄 제거**가 전부다. `stopLoading()` 을 뒤에 덧붙이는 방식은 택하지 말 것(불필요한 켜기/끄기 왕복이 남는다).
  3) 테스트가 증명할 것: (a) `openLawViewer` 경로를 지난 뒤 `useGlobalLoading().isLoading.value === false` 이고 `set-viewer` 가 기대 payload 로 emit 되었다 — 즉 "스피너를 안 켠다" 와 "기능은 그대로" 를 같은 테스트가 함께 고정한다. (b) 수정 전에는 이 단정이 실패해야 한다(Red 확인 필수). (c) `src/utils/globalLoading.js` 의 singleton start/stop 계약 spec 을 함께 추가한다(같은 모듈을 두 번 import 해도 상태 공유, `startLoading('txt')` 시 `hasTxt`/`loadingTxt`, `stopLoading()` 후 false). 참조 카운트 동작은 **계약으로 고정하지 말 것**.
- 건드릴 파일:
  - `src/views/ChatStorage/ChatStorageDetail.vue:887` — `startLoading()` 한 줄 제거. 다른 로딩 쌍·스트리밍 로직은 손대지 않는다.
  - `tests/unit/globalLoading.spec.js` (신규) — 위 (c). 순수 JS 모듈이라 현재 vitest 설정으로 바로 돈다.
  - 위 (a) 를 증명하는 spec: **먼저 `ChatStorageDetail.vue` 를 `@vue/test-utils` 로 마운트할 수 있는지 확인하라.** `vitest.config.js` 에 `@vitejs/plugin-vue` 가 붙어 있어 SFC 마운트는 가능하지만(`tests/unit/alertPopup.spec.js` 가 선례), 이 뷰는 의존이 무거워 **마운트 성공 여부는 미확인**이다. 마운트가 스텁 지옥이 되면 컴포넌트를 억지로 마운트하지 말고, `openLawViewer` 가 하는 일을 대신 검증할 수 있는 최소 범위로 줄이고 그 사실을 원장에 적어라(소스 문자열 검사로 대체하는 것은 증거로 인정되지 않는다 — 운영자 지시).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm ci` — `node_modules` 가 없다. 반드시 먼저.
  - `npm test` — vitest run. 기준선은 **인계값** 21파일 / 444테스트 통과(이번 정찰에서 재측정하지 않았다). 수정 후 증가한 테스트가 모두 통과해야 한다.
  - 인과 증명: 887행을 되돌리면 새 단정이 실패함을 확인.
  - `npm run build:dev` — 통과 확인 후 `dist/` 삭제.
- 위험과 피할 것:
  - `src/utils/globalLoading.js` 의 구현을 바꾸지 말 것(참조 카운트 금지). 호출부 누수 2건이 남은 상태에서 카운터를 넣으면 0 으로 복귀하지 못하고, `loading.vue` 의 60초 자동 해제가 카운터를 되돌리지 않아 카운터와 화면이 영구히 어긋난다.
  - `SupportOcr.vue:203-223` 을 같이 고치지 말 것. 이번 회차에 추가로 확인한 문제가 있다: 205행 `inf.tools.ocrParse.call(formData)` 를 **await 하지 않아** try/catch 가 비동기 실패를 애초에 잡지 못한다. 따라서 catch 에 `stopLoading()` 을 넣는 "간단한 수정" 은 실제 업로드 실패에서 실행되지 않는 무효 변경이 된다(운영자 지시: 출력·동작이 바뀌지 않는 수정 금지). 별도 과제로 남긴다.
  - `App.vue` 의 `Loading` 호출부에 `:timeoutMs` 를 넘기지 말 것 — 자동 해제가 60초에서 10분으로 늘어 증상이 나빠진다. 어느 쪽이 의도인지 미확인.
  - 보호 경로 회피: `src/api/auth*`, `src/storage/authStorage*`, `src/router/`, `.gitlab-ci.yml`, `docs/RELEASE.md` 는 이번 과제에서 손대지 않는다.
- 차선 후보: `tests/unit/globalLoading.spec.js` 만 단독 추가(계약 고정 전용, 가치 3 / 위험 1 / 작업량 S). `ChatStorageDetail.vue` 마운트가 불가능하고 (a) 를 실물로 증명할 방법이 없다면, 코드 수정 없이 계약 spec 만 추가하고 887행 문제는 pending 으로 남겨라 — 증명 없는 동작 변경을 밀어 넣지 말 것.
