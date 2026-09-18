# 과제서 (2026-09-19) — 수정 과제

## 오류 대응 판정 (먼저 읽을 것)

지난 회차(`2026-09-18-162350-aiportal-front-improve`)의 `error` 는 **저장소 코드·테스트·워크플로 실패가 아니다**. 확인한 사실:

- `state/runs/2026-09-18-162350-aiportal-front-improve/evidence.json` — `stages.improve = { state: "hold", reason: "회차 예산($22)이 오늘 남은 상한을 넘음" }`, `base_sha`/`head_sha`/`pr` 전부 빈 값, `usage: []`. 즉 구현 단계가 **시작조차 하지 않고** 러너의 일일 예산 정책이 회차를 보류한 것이다.
- 이 저장소에는 `.github/workflows` 디렉터리 자체가 없다(Glob `.github/**/*` → 0건). "릴리즈 워크플로" 는 존재하지 않으며, 직전 정상 회차(2026-09-17)의 `release.json` 도 `status: "skipped"`(태그 0개·릴리즈 관례 없음)로 기록돼 있다. 따라서 "같은 이유로 두 번 실패한 릴리즈 워크플로" 에 해당하는 파일·스크립트·테스트가 저장소 안에 없다.
- 결론: **저장소 쪽에서 고칠 원인이 없다.** 원인은 러너 예산 상한(`$22` 회차 예산 > 일일 잔여)이며 이는 aidev 러너 정책 설정의 문제다. 워크플로를 느슨하게 만들 대상도 없다.

구현자는 원장(ledger)에 이번 회차를 **'수정 과제'** 로 기록하되 위 판정을 그대로 적을 것("예산 홀드는 저장소 외부 원인, 코드 수정 대상 없음"). 회차를 비워 두지 않기 위해 아래의 **예산이 가장 적게 드는 S 과제**를 함께 수행한다. 큰 작업을 잡으면 같은 예산 홀드가 반복되므로 범위를 넓히지 말 것.

---

- 과제: `mitt` 를 package.json 직접 의존성으로 선언 (가치 3 / 위험 1 / 작업량 S)
- 왜: `src/utils/eventBus.js:1` 이 `import mitt from 'mitt'` 로 쓰고 이 버스를 Sidemenu.vue·Home.vue·Chat/Main.vue·Chat/Index.vue·MyAgentList.vue·SupportList.vue·PopWidgetSetting.vue·appList.js·appUpdateLog.js·useAppList.js 10곳이 쓰는데, `package.json` `dependencies`/`devDependencies` 어디에도 `mitt` 가 없다. `package-lock.json` 을 보면 `mitt@3.0.1` 은 오직 `node_modules/@vue/devtools-kit` 의 의존성(`"mitt": "^3.0.1"`, lock 4595행)으로만 들어오고, `@vue/devtools-kit` 은 **devDependency** `vite-plugin-vue-devtools` 의 하위 패키지다. 즉 프로덕션 코드가 개발 전용 플러그인의 전이 의존성에 기대고 있어, devtools 플러그인을 제거·업그레이드하거나 npm 이 hoisting 을 바꾸면 빌드가 `Rollup failed to resolve import "mitt"` 로 깨진다.
- 수용 기준:
  1) `package.json` `dependencies` 에 `"mitt": "^3.0.1"` 이 추가되고, `package-lock.json` 최상위 `packages[""].dependencies` 에도 `mitt` 가 나타난다(`npm install mitt@^3.0.1` 로 lock 을 갱신, 버전은 이미 설치된 3.0.1 과 동일하게 유지해 다른 패키지 변동이 없어야 함 — `git diff --stat package-lock.json` 이 mitt 관련 줄만 바뀌는지 확인).
  2) `npm run build:dev` 가 성공한다(dist 는 커밋 전 삭제).
  3) `npm test` 전체 통과(직전 회차 기준 408건). 테스트가 증명할 것: `tests/unit/eventBus.spec.js` 를 신규로 추가해 실제 `src/utils/eventBus.js` 의 `bus` 를 import 하여 `bus.on/emit/off` 가 실제 mitt 인스턴스로 동작함을 확인(대역·mock 금지 — 실제 모듈 해석 경로를 통과해야 의존성 선언 누락 시 `Cannot find package 'mitt'` 로 실패하는 회귀 테스트가 된다). eventBus.js 가 `bus` 외에 무엇을 export 하는지는 파일을 열어 확인할 것(정찰은 1~3행만 봄).
- 건드릴 파일:
  - `package.json` — `dependencies` 에 `mitt` 추가 (알파벳 순서는 기존 파일도 지키지 않으므로 `marked` 다음에 두면 됨)
  - `package-lock.json` — `npm install mitt@^3.0.1` 결과만 반영
  - `tests/unit/eventBus.spec.js` — 신규 (vitest, `tests/unit/**/*.spec.js` 가 include 패턴, 환경 jsdom, `@` alias 사용 가능 — `vitest.config.js` 참조)
  - `src/utils/eventBus.js` — **수정 불필요**
- 검증 명령:
  - `npm ci` (이 워크트리에는 node_modules 가 없다 — 정찰에서 `npm test` 를 돌리자 `vitest: not found`)
  - `npm test`
  - `npm run build:dev` 후 `rm -rf dist`
  - `git diff --stat` 으로 변경이 package.json / package-lock.json / tests/unit/eventBus.spec.js 3개에 한정됨을 확인
- 위험과 피할 것:
  - `npm install` 이 다른 패키지를 끌어올리거나 lock 포맷을 바꾸면 안 된다. lock 의 다른 항목이 바뀌면 `git checkout package-lock.json` 후 `npm install mitt@3.0.1 --save-exact=false --no-audit --no-fund` 로 재시도하고, 그래도 안 되면 lock 은 손대지 말고 package.json 만 고쳐 그 사실을 원장에 적을 것.
  - 의존성 메이저 업그레이드·`vite-plugin-vue-devtools` 제거 금지.
  - 이벤트 버스 사용처(.vue 10곳) 를 리팩터하지 말 것 — 이번 과제는 선언 누락 수정뿐이다.
  - 예산: 이번 회차의 실패 원인이 예산 상한이므로 탐색을 최소화하고 위 파일만 열 것.
- 차선 후보: `useAppList.js:18` / `Sidemenu.vue:163` 의 `String(app?.knowledge_info?.status) ?? ''` 를 `String(status ?? '')` 로 고치고 `getFormattedAppList` 를 export 해 테스트 추가 (가치 2 / 위험 1 / S). 1순위가 lock 갱신 문제로 성립하지 않을 때 고를 것.
