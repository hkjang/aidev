# 과제서 (2026-09-23-215420 aiportal-front)

이번 회차는 **A(우선 과제: 릴리즈)** 와 **B(실제 실행 대상)** 두 부분이다.
A 는 진입 조건이 충족되지 않으므로 **무변경**으로 두고, 실제로 구현할 것은 B 다.

---

## A. 우선 과제(릴리즈 실패) — 진입 조건 미충족, 이번 회차 무변경 (가치 5 / 위험 2 / 작업량 S)

- 왜: 릴리즈가 15회 연속 같은 사유로 실패한다. 이번 회차에 정찰이 워크트리에서 직접 재확인했다 —
  `git tag` **0개**, `package.json` `"version": "0.0.0"`, 저장소 루트에 **`.github` 디렉터리 없음**(워크플로 0개),
  `CHANGELOG.md`/`VERSION`/`scripts/`/`Makefile` 없음, 작업 트리 청정(HEAD `38cc2c3`).
  `.gitlab-ci.yml` 은 `main`/`develop` **브랜치 푸시**로 도는 배포 전용이며 태그·버전·릴리즈 노트를 다루지 않는다.
  `docs/RELEASE.md` 의 "미확인 항목과 한계" 절은 **버전 파일이 존재하므로 `skipped` 조건을 충족하지 않는다**고,
  또 **이 문서로 실패를 `skipped`/`released` 로 바꾸지 않는다**고 명시한다.
- 과제서의 "워크플로 파일과 실패한 단계의 스크립트를 고치라" 는 지시는 **이 저장소 안에서 수행할 수 없다**.
  실패 지점이 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이고, 저장소 안에서
  통과시키는 유일한 수단은 버전 파일·태그·릴리즈 노트 관례를 **새로 만드는 것**인데 이는
  `AGENTS.md`("확인되지 않은 관례를 새로 만들거나 릴리즈 판정 조건을 완화하지 않는다")와
  이 세션의 "워크플로를 느슨하게 만들어 통과시키는 것은 금지" 양쪽에 걸린다.
- **구현자가 할 일(정확히 이것만)**: 아무것도 하지 않는다. 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·
  원격 전송을 **일체 만들지 말고**, `docs/RELEASE.md` 도 수정하지 말고, 판정을 `skipped`/`released` 로 낮추지 말 것.
  재조사도 하지 말 것(이번 회차 정찰이 이미 했고 근거는 위에 있다). 원장에는 "진입 조건 미충족으로 무변경(15회째)" 로 기록.
- 필요한 사람 입력: ①버전 릴리즈 관례 도입 여부 자체(현 배포는 GitLab CI 브랜치 푸시라 태그가 배포에 불필요)
  ②시작 버전(0.0.1 대 0.1.0)과 증가 단위 ③태그 형식·주석 태그 여부 ④릴리즈 커밋 메시지 양식
  ⑤릴리즈 노트 위치·양식·언어 ⑥GitHub Release 사용 여부(현 원격 배포는 GitLab).

---

## B. 이번 회차 실행 대상

- 과제: 심플봇 **수정** 팝업이 저장 실패에도 팝업을 닫고 부모에 'update' 를 알려 사용자의 수정 내용이 유실되는 문제 수정 (가치 4 / 위험 2 / 작업량 M)

- 왜: `PopSimpleBotUpdate.vue` 의 `onCreate()`(652-681행)는 실패 경로에서 `toast('수정에 실패했습니다.')` 후
  `return` 하지만, **`finally` 블록(677-680행)이 `emit('update', true)` 와 `close()` 를 무조건 실행한다**.
  그래서 저장이 실패해도 팝업이 닫히고 사용자가 입력한 이름·설명·프롬프트·첨부가 통째로 사라지며,
  `App.vue:37-43` 의 `handleSimpleBotUpdate` 가 호출자의 `updateCallback()`(목록 새로고침)을 실행해
  **성공한 것처럼 목록이 갱신된다**. 형제 컴포넌트 `PopSimpleBot.vue` 의 `onCreate()`(566-604행)는
  성공 경로 안에서만 `emit('created', …)` + `close()` 를 하고 실패하면 팝업을 열어 둔다 —
  즉 "실패하면 닫지 않는다" 가 원 저자의 계약이 코드에 남은 형태다. 고치면 사용자가 실패 후 바로 재시도할 수 있고
  목록이 거짓으로 갱신되지 않는다.

- 수용 기준:
  1) `inf.app.createApp` 이 업무 실패(HTTP 200 + `isSuccess(res)` false)를 돌려주면 `PopSimpleBotUpdate` 가
     **열린 채로 남고**(`update:modelValue` false 가 emit 되지 않음), **`update` 이벤트도 emit 되지 않는다**.
     `toast('수정에 실패했습니다.')` 는 그대로 뜬다.
  2) 첨부가 있을 때 `inf.app.updateAppFile` 단계가 실패해도 1)과 같다(팝업 유지, `update` 미emit, 토스트 1회).
  3) 성공 경로(두 호출 모두 `isSuccess` true)에서는 기존대로 `update` 가 **정확히 1회** emit 되고 팝업이 닫힌다.
     `tryUpdateAppIcon()`(647-650행)의 실패는 지금처럼 삼켜도 되며 성공 판정을 바꾸지 않는다.
  4) 공통 오류('COM' — 세션 만료/BZ01)로 예외가 나면 인터셉터가 이미 전역 Alert 을 띄우므로
     `catch` 에서 **중복 토스트를 띄우지 않는다**(`if (e == 'COM') return` 을 `toast` 앞에 둘 것.
     저장소 표준형: `PopSimpleBotUpdate.vue:347-349`, `ChatStorageList.vue:82-84`). 이 경우에도 `update` 는 emit 되지 않는다.
  5) 성공·실패 어느 경로에서도 `isSendLoading.value` 가 false 로 돌아온다(재클릭 가능).
  6) 테스트는 위 1)~5)를 **실제 `interceptors.js` → 실제 `inf.app.*.call` → 실제 컴포넌트 마운트**로 증명해야 한다.
     대역은 axios adapter(HTTP 전송) 한 겹만. 수정 전에 최소 2건이 실패(Red)하는 것을 먼저 확인하고,
     수정을 되돌리면 같은 건이 다시 실패함을 확인해 인과를 증명할 것.

- 건드릴 파일:
  - `src/components/common/popup/Global/PopSimpleBotUpdate.vue:652-681` — `onCreate()`:
    `emit('update', true)` 와 `close()` 를 `finally` 에서 **성공 경로 마지막**(`await tryUpdateAppIcon()` 뒤)으로 옮긴다.
    `finally` 에는 `isSendLoading.value = false` 만 남긴다. `catch` 의 `toast('수정에 실패하였습니다.')` 앞에
    `if (e == 'COM') return` 을 넣는다.
  - `tests/unit/simpleBotUpdateClose.spec.js` — 신규. 기존 `tests/unit/shareAppCommonError.spec.js`(실제 인터셉터 +
    adapter 대역 패턴)와 `tests/unit/simpleBotPromptReload.spec.js`(이 컴포넌트 마운트 패턴)를 그대로 참고할 것.
    `validateCreateApp()` 이 `name`·`description` 등을 요구하므로 `input#name`/`textarea#description` 을
    **실제 DOM 으로 채운 뒤** 저장 버튼을 눌러야 한다(지난 회차에서 확인된 함정).
  - 그 외 파일은 건드리지 않는다. 특히 `App.vue` 와 `useSimpleBotUpdateStore.js` 는 **무변경**.

- 검증 명령:
  - `npm ci` (node_modules 비어 있음 — 선행 필수)
  - `npx vitest run tests/unit/simpleBotUpdateClose.spec.js` (Red 확인 → 수정 → Green)
  - `npm test` — **기준선 28파일 487테스트**. 신규 스펙만큼 늘어야 하고 기존은 하나도 깨지면 안 된다.
  - `npm run build:dev` 통과 후 생성된 `dist/` 삭제. (`npm run build`·`npm run lint` 는 **없다**.)

- 위험과 피할 것:
  - `PopSimpleBot.vue`(생성 팝업)는 이미 올바르다 — **건드리지 말 것**. 이번 수정은 Update 쪽 단방향이다.
  - `emit('update')` 를 성공 경로로 옮길 때 **중복 emit** 을 만들지 말 것(App.vue 가 `updateCallback()` 을 실행하므로
    2회 emit 이면 목록이 두 번 새로고침된다). 수용 기준 3)의 "정확히 1회" 를 단정으로 박아 증명할 것.
  - 팝업을 닫지 않게 바뀌므로 `simpleBotUpdate.state.editItem` 이 유지된다 — 사용자가 직접 닫는 경로
    (`Custom` 팝업의 닫기 → `onUpdate`/`@cancel`)가 여전히 동작하는지 케이스 하나로 확인할 것.
  - 보호 경로(`src/api/common/interceptors.js`, `auth.js`, `storage/authStorage`, `router`)는 **읽기만** 하고 수정 금지.
  - 전역 토스트·Alert·로딩은 모듈 싱글턴이라 테스트 간 상태가 샌다 — 케이스마다 정리할 것.
  - 이 컴포넌트의 `startLoopTyping` 은 150ms `setInterval` 이다. 즉시 resolve 하는 대역으로는 결함이 안 드러날 수 있으니
    필요하면 deferred 응답으로 실제 시간을 흘려보낼 것(지난 회차 교훈).

- 차선 후보: **`PopSimpleBotUpdate.vue:647-650` 의 `tryUpdateAppIcon()` 이 `inf.app.appEdit` 실패를 `catch(e){}` 로
  완전히 삼키는 문제** — 아이콘 변경만 실패해도 사용자는 성공으로 안내받는다. 다만 "아이콘 실패는 전체 실패인가"가
  기대 계약 미확인이라 1순위로 두지 않았다. 1순위가 성립하지 않을 때만 고르고, 고른다면 계약을
  "아이콘 실패는 별도 토스트로 알리되 앱 수정 자체는 성공" 으로 좁혀 잡을 것.
