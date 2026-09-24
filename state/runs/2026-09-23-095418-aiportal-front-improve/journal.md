# 회차 노트 2026-09-23-095418-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:54] base pinned — main@66e6086
- [러너 09:54] autonomy release — 

## 정찰 노트
- 우선 과제(릴리즈)는 이번에도 저장소 안에 고칠 대상이 없다: `.github` 디렉터리 부재·태그 0개를 직접 확인했고 실패 지점은 외부 `release-prompt.md:21`(2항, 없는 증가 패턴 요구)과 `:24`(5항, 버전 파일 부재 요구) 의 상호배타다. 저장소에서 통과시키는 유일한 수단인 `package.json` version 필드 삭제는 게이트 완화라 금지했다. 7회 연속 무변경을 되풀이하지 않으려고 실제 실행 과제를 지정했다.
- 고른 이유: `ChatStorageDetail.vue:887` 고아 `startLoading()` 은 형제 구현 `Chat/Index.vue:2309`(startLoading 없음)와 `writeDoc`=동기 localStorage(`useDocStroage.js:6`) 두 가지로 "제거가 정답"이 확정되는 드문 건이다. 참조 카운트·SupportOcr·docs/RELEASE.md 는 각각 선행 조건·판단 필요·정본 훼손 이유로 제쳤다.
- 확신 없는 곳: `npm test` 기준선 20파일 433테스트는 이전 회차 기록을 인용한 것이고 이번에 실행하지 않았다(node_modules 없음). 스피너가 실제 화면에서 남는 것도 코드 경로 추적이지 브라우저 재현이 아니다.
- 구현자 주의: `.vue` 는 현재 vitest 설정으로 마운트 불가(`@vitejs/plugin-vue` 가 devDeps 에는 있으나 `vitest.config.js` 에 미연결, `@vue/test-utils` 없음). SFC 런타임 증명이 불가함을 숨기지 말고 그대로 적을 것 — 소스 문자열 검사를 증거로 포장하면 반려된다.
- 차선 1(vitest SFC 지원 활성화)은 이 저장소가 반복해서 과제를 접는 근본 원인을 없애므로, 1순위가 막히면 주저 말고 그쪽으로 갈 것.
- [러너 09:59] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `Alert.vue`/`Confirm.vue` 의 `v-html="text"` 에 서버 메시지(`res?.data?.message`)가 그대로 들어가 태그가 실제 DOM 요소로 생성됐고, 반대로 본문의 `\n` 은 `.alert-text` 에 `white-space` 가 없어 줄바꿈으로 안 보였다. `src/utils/alertText.js`(이스케이프 + 개행·`<br>` 표기만 `<br>`)를 추가해 두 컴포넌트에 연결했다. 증명 수단이 없던 문제를 먼저 풀려고 `vitest.config.js` 에 `@vitejs/plugin-vue` 를 붙이고 `@vue/test-utils` 를 devDep 에 추가했다.
- 확신 없는 곳: jsdom 에서 `innerHTML` 로 삽입된 `<script>` 는 실행되지 않으므로 이번 테스트가 증명한 것은 "요소가 만들어진다/안 만들어진다" 까지다. 실제 브라우저에서 `onerror` 가 실행되는지는 재현하지 않았다. 또 `openAlert`/`confirmStore.open` 호출부를 grep 해 `<br>` 외의 태그가 없음을 확인했지만 서버가 내려주는 문구는 확인할 수 없어, 어딘가 HTML 서식을 의도한 메시지가 있으면 이제 문자 그대로 보인다.
- 일부러 하지 않은 것: 정찰 1순위였던 `ChatStorageDetail.vue:887` 고아 `startLoading()` 과 `SupportOcr.vue:204` 누수는 손대지 않았다 — 두 뷰는 의존이 무거워 이번에 켠 SFC 마운트로 실제로 테스트 가능한지 확인하지 않았고, 증명 없이 고치면 반려 대상이다. `Base.vue:44` `@:click` 오타와 `showHeader` 미전달로 `title` 이 안 보이는 문제도 UI 기대 계약 확인이 필요해 아이디어로만 남겼다. `dompurify` 는 `markdown.js` 가 전역 훅을 등록해 두어 얽히지 않도록 쓰지 않았다.
- 다음 역할이 조심할 것: `npm ci` 후 `npm install -D @vue/test-utils` 를 했고 `package.json` 의 dependencies 재정렬은 되돌려 devDep 한 줄만 추가했다(`package-lock.json` 은 +252줄). 빌드 산출물은 커밋 전에 지웠고 워크트리는 clean. 커밋 2개(`43ed569` 테스트 설정, `228811d` 수정)로 나눠 두었으니 설정만 되돌려도 수정은 남는다.
- [러너 10:08] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인한 것: `alertText.js` 우회 가능성(리터럴 `<br>` 만 되살리고 역참조 없음 → 속성 주입 불가), 실제 sink 유입 경로(`SupportOcr.vue:85` 등 `${file.name}`, `interceptors.js:231` 서버 message), 호출부 HTML 전수 grep(`PopSimpleBotUpdate.vue:166/172`, `UiComponents.vue:439/488` 의 `<br>`·`<br/>` 뿐 — 모두 보존), `Alert.js` 명령형 경로도 같은 `Alert.vue` 를 쓰므로 함께 고쳐짐. `npx vitest run` 21파일 444테스트 통과, 워크트리 clean, lock 신규 17개 전부 dev·허용 라이선스.
- 테스트 유효성은 별도로 증명했다: node+jsdom 으로 `innerHTML` 이 `<script>`/`<img>` 요소를 **생성함**을 직접 확인했으므로 `alertPopup.spec.js:35-37` 은 수정 전 코드에서 실패한다. 단 같은 테스트의 `window.__hacked` 단언은 수정 전에도 통과하는 무효 단언(구현 노트가 이미 밝힌 한계).
- 못 본 것: 실제 브라우저에서의 `onerror` 실행 재현, `npm run build:dev`(소스 변경이 alias 경유 import 1줄이라 생략), 서버가 내려주는 문구의 실제 형태.
- 승인이어도 남는 우려: 서버가 이미 이스케이프한 문구를 보내면 엔티티가 문자 그대로 보인다(릴리즈 노트에 "팝업 본문은 평문, `<br>` 만 줄바꿈" 명시 권장). 남은 sink 중 `ChatStorageDetail.vue:1017` 과 `Chat/Index.vue:2460` 은 `msg.content` 를 이스케이프 없이 `<br>` 치환만 하므로 같은 처방이 바로 맞는 다음 회차 1순위.
- 릴리즈 판정은 이 PR 로 바뀌지 않는다: version 두 필드 0.0.0, 태그 0개 유지.
- [러너 10:12] review approved — 리뷰 승인 (risk=low)
- [러너 10:12] pr created — https://github.com/hkjang/aiportal-front/pull/22
- [러너 10:13] ci passed — 검사 없음 — 정책으로 허용
- [러너 10:13] merge done — 228811d
- [러너 10:14] release failed — 릴리즈 안 함: 다음 버전·태그·릴리즈 노트 관례를 확정할 근거가 없어 릴리즈를 수행하지 못했다. HEAD 595c0c5 에서 직접 재조회한 근거: git tag 0개(.git/r
