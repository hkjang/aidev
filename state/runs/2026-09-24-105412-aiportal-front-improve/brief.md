# 정찰 과제서 — 2026-09-24-105412-aiportal-front-improve

이번 회차는 **A(우선 과제: 릴리즈)** 와 **B(실제 코드 과제)** 두 덩어리다.
A 는 결론이 이미 나 있으니 구현자는 **재조사하지 말고 손대지 말 것**. 실제 작업은 B 하나다.

---

## A. 우선 과제 — 릴리즈 실패(26회째): 이번 회차도 **무변경**이 정답

### 이번 회차에 직접 재실측한 것 (worktree `/home/hkjang/.cache/auto-improve-wt/aiportal-front`, HEAD `50b4551`)

| 항목 | 명령 | 결과 |
|---|---|---|
| 태그 | `git tag \| wc -l` | **0** |
| 워크트리 | `git status --porcelain` | 무출력(clean) |
| 버전 | `package.json:3` | `"version": "0.0.0"` (name `kcb_ai`) |
| GitHub Actions | `ls .github` | **No such file or directory** |
| GitLab 태그 규칙 | `grep -c CI_COMMIT_TAG .gitlab-ci.yml` | **0** |
| CHANGELOG / RELEASE / scripts / Makefile | `ls CHANGELOG* RELEASE* scripts Makefile` | **4건 전부 없음** |
| 루트 전량 | `ls -a` | `.gitlab-ci.yml`, `docs/`, `src/`, `tests/`, `package*.json`, `vite*.config.js` 등 — 릴리즈 자산 0 |

`docs/RELEASE.md` 는 존재하지만 **릴리즈 노트가 아니라 근거 문서**이며, 과거 유일한 버전 표기였던
README/docs 의 `| 0.0.1 | 2025-01-01 | 초기 릴리스 |` 행을 **'실제 릴리스 미확인'** 으로 확정해 둔 문서다.
증가 기준으로 쓸 수 없다.

### 왜 고칠 수 없나 — 러너 지시("워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 고치세요")의 대상이 없다

- **워크플로 파일이 이 저장소에 없다.** `.github` 부재로 Actions 0개. `.gitlab-ci.yml` 은 12개 job 전부
  `CI_COMMIT_BRANCH == main|develop` 조건의 **브랜치 배포 전용**(`git fetch/reset --hard` → `npm run build:*`
  → `cp -rf dist/* <DEPLOY_DIR>`)이고 태그에 반응하는 줄이 **0줄**이다.
- **실패 지점이 워크트리 밖**이다: 외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`.
  이 저장소 안에 고칠 파일이 존재하지 않는다.
- **게이트 완화는 금지**다. `package.json` 의 `version` 필드를 지우면 외부 절차의 `skipped` 조건
  ('태그도, 버전 파일도, 릴리즈 노트도 없음')을 인위적으로 만들어 통과시키는 것이 되므로 **하지 말 것**.
  버전 파일이 존재하므로 현재 판정은 `skipped` 도 `released` 도 아니다 — **낮추지 말 것**.
- **첫 릴리즈 버전을 임의로 정하는 것은 관례를 새로 만드는 일**이라 사람의 몫이다. 형제 저장소
  (`aiportal-front-admin`)의 `v0.1.x` 를 이식하는 것도 승인되지 않은 관례 이식이므로 금지.

### 구현자가 할 일 (A)

**아무것도 하지 않는다.** 재조사도 하지 말 것. 다음을 **일체 만들지 말 것**:
버전 파일 갱신 · 태그 · CHANGELOG · 릴리즈 노트 · 릴리즈 커밋 · 원격 전송 · `docs/RELEASE.md` 수정.
회차 노트에는 "진입 조건 미충족으로 무변경(26회째)" 과 아래 사람 입력 4가지를 그대로 옮겨 적을 것.

### 사람 입력 필요 4가지 (직전 25회차와 동일 — 에스컬레이션 대상)

1. 첫 릴리즈 버전을 **0.0.1 / 0.1.0 / 1.0.0** 중 무엇으로 할지
2. 태그 형식(`v` 접두사 여부)과 주석 태그(`git tag -a`) 사용 여부
3. 릴리즈 노트의 위치·양식·언어 (`CHANGELOG.md` 신설 여부 포함)
4. GitHub Release 사용 여부 (현재 GitLab CI 는 브랜치 배포 전용이라 태그에 반응하는 자동화가 없음)

> 참고: GitLab CI 가 main 머지 시점에 이미 배포하므로 **버전 릴리즈 없이도 서비스는 나간다.**
> 즉 이것은 장애가 아니라 **버전 체계 도입 여부라는 제품 판단**이 비어 있는 상태다.

---

## B. 이번 회차 실제 과제

- **과제**: 위젯 설정 팝업(PopWidgetSetting)이 **저장 실패에도 팝업을 닫고 홈 위젯을 새로고침시켜 사용자의 설정이 조용히 되돌아가는 문제 수정** (가치 4 / 위험 2 / 작업량 M)

- **왜**:
  `src/components/common/popup/Global/PopWidgetSetting.vue:20-24` 의 `onClick` 은
  `saveWidgetSetting()` 을 **`await` 없이** 호출하고 곧바로 `close()` 한다. 그리고
  `saveWidgetSetting`(`:142-150`)은 **`isSuccess(res)` 검사가 없어** HTTP 200 으로 도착한 업무 실패를
  그대로 통과시킨 뒤 `bus.emit(BUS_EVENT.WIDGET_CHANGED)`(`:147`)를 쏜다 —
  `Home.vue:448` 이 이 이벤트로 `getWidgetList`(`:385-412`)를 다시 돌리므로 **저장되지 않은 옛 서버 값이
  화면에 다시 그려지고**, 사용자는 팝업이 닫히고 위젯 순서/켜짐이 원래대로 돌아가는 것만 본다(안내 0건).
  예외 경로는 더 나쁘다: `catch (e) {}`(`:148-149`)가 **본문이 비어 있어** 예외를 통째로 삼키고
  `'COM'` 가드도 토스트도 없다 — 팝업은 이미 닫혔고 아무 일도 일어나지 않는다.
  같은 저장소의 **저장 팝업 표준형은 이미 확정돼 있다**: `PopSimpleBotUpdate.vue:670-681` 은
  `if (!isSuccess(res)) { toast('수정에 실패했습니다.'); return }` 로 **닫지도 않고 부모에 알리지도 않으며**,
  `catch` 는 `if (e == 'COM') return` 뒤에 토스트를 띄운다. 이 한 곳만 비대칭이다(= 이 저장소 결함의 주류).

- **수용 기준**:
  1) `inf.widget.personal.call` 이 **HTTP 200 + 업무 실패**(`isSuccess(res)` false)로 응답하면
     ① 토스트가 뜨고 ② **팝업이 닫히지 않으며**(`update:modelValue(false)` 미발생)
     ③ **`BUS_EVENT.WIDGET_CHANGED` 가 발생하지 않는다**(홈이 옛 값으로 덮어쓰지 않는다).
  2) 응답에 메시지가 있으면 `res?.data?.message` 를 쓰고, 없으면 고정 문구를 쓴다.
     **문구를 새로 발명하지 말 것** — 저장 표준형 `PopSimpleBotUpdate:671` 과 같은 계열로
     업무 실패 `res?.data?.message || '위젯 설정 저장에 실패하였습니다.'`,
     예외 `'위젯 설정 저장 중 오류가 발생하였습니다.'` 를 쓴다.
  3) 예외 경로에서 `e == 'COM'` 이면 **아무 안내도 하지 않고 return**(인터셉터가 이미 전역 Alert 을 띄웠으므로
     겹치면 안 된다), `'COM'` 이 아니면 토스트를 띄우고 **팝업을 닫지 않는다**.
  4) **성공 경로는 지금과 똑같이 동작한다**: `WIDGET_CHANGED` 1회 발생 + 팝업 닫힘.
     (성공 케이스는 수정 전후 모두 통과해야 한다 — 범위가 좁음을 고정한다.)
  5) 테스트는 **HTTP 전송(axios adapter) 한 겹만 대역**으로 바꾸고 실제 `interceptors.js` →
     실제 `inf.widget.personal.call`(**PUT `/widget/personal?<qs>`**) → **실제 `PopWidgetSetting.vue` 마운트**를
     통과해 증명해야 한다. 손으로 만든 가짜 `inf` 나 소스 문자열 검사는 증거로 인정되지 않는다.
     최소 케이스: ①성공(닫힘 + 이벤트 1회) ②업무 실패(메시지) ③업무 실패(기본 문구) ④예외 비-COM
     ⑤예외 COM(토스트 0건, 전역 Alert 과 겹치지 않음) ⑥업무 실패 후 **팝업이 열린 채 남아 재저장 가능**.
  6) **변이로 인과를 분리 증명**한다(운영자 지시: 무효 변경 금지). 최소 3건 —
     ①`isSuccess` 분기 제거 ②`await` 추가 제거(= 닫힘이 먼저 일어남) ③`'COM'` 가드 제거.
     각 변이가 **서로 다른 부분집합**을 실패시켜야 한다. 어떤 줄을 빼도 전부 통과한다면 **그 줄은 넣지 말 것.**

- **건드릴 파일** (이 3곳 외에는 열지도 말 것):
  - `src/components/common/popup/Global/PopWidgetSetting.vue:20-24` `onClick` —
    `saveWidgetSetting()` 을 `await` 하고, **저장이 성공했을 때만** `close()` 하도록 바꾼다
    (예: `saveWidgetSetting` 이 성공 여부를 boolean 으로 돌려주게 하고 `if (!ok) return` 후 `close()`).
    `emit('click')` 의 위치도 함께 판단할 것 — 현재는 저장 전에 무조건 쏜다.
    **주의**: `MainLayout.vue:107` 의 `@click="onWidgetSave"` 는 **핸들러가 정의돼 있지 않다**
    (`onWidgetSave` 가 `MainLayout.vue` 어디에도 없음 — grep 확인). 즉 이 emit 은 현재 아무 데도 닿지 않는다.
    따라서 **`emit('click')` 을 옮기거나 지우는 것은 관측 가능한 동작 변화가 0 이다 — 건드리지 말 것.**
    `onWidgetSave` 미정의 자체도 이번 과제 범위 밖이다(별도 아이디어로 남겼다).
  - `src/components/common/popup/Global/PopWidgetSetting.vue:142-150` `saveWidgetSetting` —
    `const res = await inf.widget.personal.call(qs)` 로 응답을 받고, `isSuccess(res)` 실패 시
    토스트 후 `return false`; 성공 시에만 `bus.emit(BUS_EVENT.WIDGET_CHANGED)` 후 `return true`.
    `catch` 는 `if (e == 'COM') return false` 뒤 토스트 후 `return false`.
  - `src/components/common/popup/Global/PopWidgetSetting.vue:1-10` import —
    `import { isSuccess, toast } from '@/utils/common.js'` 추가(현재 이 파일은 둘 다 import 하지 않는다).
  - `tests/unit/widgetSettingSaveFailure.spec.js` (신규) — 위 6케이스.

- **검증 명령** (이 저장소에서 실제로 도는 것):
  ```
  npm ci                                          # node_modules 가 비어 있다 — 선행 필수, 수 분
  npx vitest run tests/unit/widgetSettingSaveFailure.spec.js   # 신규 스펙 단독(Red → Green)
  npm test                                        # 전체
  npm run build:dev && rm -rf dist                # 빌드 확인 후 dist 삭제
  ```
  - **기준선**: 직전 회차 기록상 **37파일 / 573테스트**. `tests/unit` 의 spec 파일 수가 37개인 것은
    이번 회차에 확인했으나 **테스트 건수는 재실행하지 않았다(미확인)** — 구현자가 **수정 전에 `npm test` 를
    한 번 돌려 기준선을 직접 확정**한 뒤 작업할 것.
  - **`npm run build` 와 `npm run lint` 는 없다**(scripts 에 vite dev/build:*/preview 와 vitest 뿐).

- **위험과 피할 것**:
  - **공유·보호 경로 금지**: `src/api/interface.js`, `src/api/common/interceptors.js`,
    `src/utils/common.js`(`isSuccess`/`toast`), `src/utils/eventBus.js`, `src/views/Home.vue`,
    `src/layouts/MainLayout.vue`, `Custom.vue`/`Base.vue` — **전부 손대지 말 것.**
    이 결함은 **호출부 한 파일에서 표준형에 맞추는 것**으로 끝난다(이 저장소의 확정된 방향).
  - **무효 변경 금지**(운영자 반복 지시). 특히 ①`emit('click')` 이동/삭제 ②`getWidgetList` 의 `catch` 정리
    ③`console.log('[widgetList raw]')` 제거 같은 **관측 가능한 동작 변화가 0 인 청소는 넣지 말 것**.
    넣기 전에 반드시 **그 줄을 빼고 돌려 실패하는 케이스가 있는지** 변이로 확인한다.
  - **토스트는 반드시 `'COM'` 가드 *뒤*에** 둘 것. 앞에 두면 인터셉터의 전역 Alert 과 겹친다
    (과거 회차에서 실제로 걸린 자리).
  - **전역 싱글턴 정리**: `vitest.config.js` 는 `restoreMocks: true` 지만 전역 토스트/알림/로딩은
    모듈 싱글턴이라 **케이스마다 초기화**할 것. `bus` 도 싱글턴이므로 `bus.on` 한 리스너를
    케이스마다 `bus.off` 하고 호출 횟수를 리셋할 것(안 하면 이벤트 카운트가 누적돼 오탐).
  - **환경 변수**: 모듈 로드 **전에** `vi.stubEnv('VITE_BACKEND_API_TARGET', …)`,
    `vi.stubEnv('VITE_PYTHON_API_TARGET', …)` 필수. 없으면 `src/api/index.js:16` 이 던진다.
  - **'COM' 을 만드는 검증된 대역 응답**: `{ status: 204, data: { code: 'BZ01', message: '…' } }`.
    업무 실패(HTTP 200 + 실패)는 인터셉터를 통과해 컴포넌트까지 도달하므로 status 200 으로 만들 것.
  - **`readUser()` 가 없으면 `getWidgetList` 의 `userId` 가 빈 문자열**이 된다. 마운트 전에
    `writeUser(...)` 로 사용자를 심을 것. **입력 키는 `user_id` 가 아니라 `userId`** —
    `writeUser` 가 `normalizeUserRecord` 를 거치기 때문이다(직전 회차에서 실제로 걸린 함정).
    단 `saveWidgetSetting` 자체는 사용자에 의존하지 않으므로, 목록 조회가 비어도 저장 경로는 테스트 가능하다.
  - **열림 watch 는 값이 바뀔 때만 돈다**(`:99-110`, `immediate: true` 지만 `hasLoadedOnce` 가드가 있다).
    목록 조회를 재현하려면 **닫은 채 마운트하고 `setProps({ modelValue: true })` 로 열 것**.
    `MainLayout` 은 `v-if` 로 이 팝업을 붙였다 뗐다 하므로 `hasLoadedOnce` 는 실제로는 매번 초기화된다.
  - 저장 버튼은 `Custom.vue` 의 `@primary` 로 들어온다(`:161`). 테스트에서 버튼을 찾기 어려우면
    **`onClick` 을 DOM 버튼 클릭으로 구동하는 쪽을 먼저 시도**하고, 정 안 되면
    `wrapper.findComponent(Custom).vm.$emit('primary')` 로 실제 배선을 통과시킬 것.
    **`onClick` 을 직접 호출해 통과시키지 말 것** — 배선 결함을 못 본다(운영자 지시).

- **차선 후보**:
  **심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch (e) {}` 로 예외를 완전히 삼킴**
  (`PopSimpleBot.vue:542,563` / `PopSimpleBotUpdate.vue:649`).
  단, 이쪽은 **기대 계약이 미확인**이고(`아이콘 저장 실패가 앱 저장 전체의 실패인가`)
  `tests/unit/simpleBotUpdateClose.spec.js` 가 현 동작('아이콘 실패는 성공 판정을 바꾸지 않는다')을
  이미 고정하고 있다 — 고치려면 계약을 '앱 저장은 성공, 아이콘 실패만 별도 토스트' 로 두고
  **기존 케이스부터 갱신**해야 한다. 1순위가 성립하지 않을 때만 고를 것.

### 작업량 근거 (basis of estimate — 분해·가정·범위)

| 조각 | 방법 | 추정 |
|---|---|---|
| `PopWidgetSetting.vue` 3곳 수정 | bottom-up | 0.5h |
| 신규 스펙 6케이스 (기존 `shareListFailure`/`simpleBotUpdateClose` 하네스 재사용) | analogous | 1.5h |
| 변이 3건 | analogous | 0.5h |
| `npm ci` + `npm test` + `build:dev` | parametric(과거 회차 실측) | 0.5h (대부분 대기) |
| **합계** | | **3.0h** |

- **범위**: 위 3곳 + 신규 스펙 1파일. **제외**: 공유 컴포넌트, `Home.vue`, `MainLayout.vue`,
  `HeaderAlarm`, 심플봇 팝업, `onWidgetSave` 미정의, 릴리즈 관련 일체.
- **범위(80% 신뢰)**: **2.5h ~ 4.5h**.
- **컨틴전시 +0.5h(known unknown)**: `Custom.vue` 의 `@primary` 배선을 테스트에서 구동하는 방법을
  찾는 데 드는 시간. `draggable`(vuedraggable) 이 jsdom 마운트에서 문제를 일으킬 가능성이 여기 포함된다.
- **가정**: ①`inf.widget.personal.call` 이 axios 응답을 그대로 돌려준다(`interface.js:81` 이
  `return getApi().put(...)` 인 것을 확인했다) ②`isSuccess`(`utils/common.js:55-61`, **이번 회차에 원문 확인**)는
  `res.data` 가 없으면 false, `success` 가 boolean 이면 그 값, 아니면 `code` 가 `'200'`/`200` 일 때만 true —
  따라서 **업무 실패 대역은 `{ status: 200, data: { code: 'BZ99', message: '…' } }`** 로 만들면 된다
  ③`vuedraggable` 이 jsdom 에서 마운트된다 — **여기가 이 계획이 깨진다면 가장 먼저 깨질 자리다.**
  `package.json:59` 에 `"vuedraggable": "^4.1.0"` 이 실제 의존성으로 있는 것은 확인했으나,
  **기존 스펙 37개 중 `draggable` 을 마운트하는 것은 하나도 없다**(`grep -rln draggable tests/` 무출력).
  즉 이 저장소에서 jsdom 마운트 전례가 없다 — **구현자는 스펙을 쓰기 전에 빈 케이스로 마운트만 먼저
  해 볼 것.** 마운트가 불가능하면 1순위를 버리고 차선 후보로 갈 것(스텁으로 우회하지 말 것 —
  실제 컴포넌트를 통과하지 않는 증거는 인정되지 않는다).
