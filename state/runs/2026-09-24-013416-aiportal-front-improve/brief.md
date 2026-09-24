# 과제서 — 2026-09-24-013416-aiportal-front-improve

이번 회차는 **A(우선 과제: 릴리즈 실패)** 와 **B(실제 실행 대상)** 두 부분이다.
A 는 저장소 안에 합법적 수단이 없음을 재확인했으므로 **무변경**으로 두고, B 를 구현한다.

---

## A. 우선 과제(릴리즈 실패) — 진입 조건 미충족, 이번 회차도 무변경 (18회째)

과제서의 지시는 "워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치라" 이지만,
**이 저장소 안에는 고칠 워크플로 파일도 스크립트도 존재하지 않는다.** 이번 회차 워크트리
(HEAD `1a8cb85`, `git status --porcelain` 무출력)에서 직접 재조회한 근거:

| 확인 항목 | 명령 | 결과 |
|---|---|---|
| git 태그 | `git tag --sort=-creatordate \| wc -l` | **0개** |
| 버전 파일 | `package.json:3`, `package-lock.json:3,9` | 모두 `0.0.0` |
| GitHub Actions | `ls .github` | **디렉터리 자체 없음** |
| CHANGELOG/VERSION/scripts/Makefile | `ls CHANGELOG.md VERSION scripts Makefile` | **전부 없음** |
| GitLab CI 의 태그 게이트 | `grep -c CI_COMMIT_TAG .gitlab-ci.yml` | **0건** (12개 job 전부 `CI_COMMIT_BRANCH == main\|develop` 브랜치 푸시 트리거) |

`docs/RELEASE.md` 는 릴리즈 노트가 아니라 '릴리즈 판단 근거' 문서이며, 스스로 "버전 파일이
있으므로 skipped 조건은 충족하지 않는다. 이 문서로 실패를 skipped/released 로 바꾸지 않는다" 고
명시한다. 실패 지점은 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이고,
저장소 안에서 통과시키는 유일한 수단은 버전 파일·태그·릴리즈 노트 관례를 **새로 발명**하는
것인데 이는 `AGENTS.md`("확인되지 않은 관례를 새로 만들거나 릴리즈 판정 조건을 완화하지 않는다")와
이 세션의 "워크플로를 느슨하게 만들어 통과시키는 것은 금지" 양쪽이 막는다.

**구현자가 할 일: 아무것도 하지 말 것.** 버전 파일 수정·태그 생성·CHANGELOG·릴리즈 노트·릴리즈
커밋·`docs/RELEASE.md` 수정·원격 전송(`git push`/`gh release`/`npm publish`)을 **일체 금지**한다.
`package.json` 의 `version` 필드 삭제로 게이트의 skipped 조건을 만드는 것도 **게이트 완화라 금지**.
회차 노트에 "진입 조건 미충족으로 무변경(18회째)" 로 기록하고 B 로 넘어갈 것.

필요한 사람 입력(6건, 변동 없음): ①버전 릴리즈 관례 도입 여부 자체(현 배포는 GitLab CI 브랜치
푸시라 태그가 배포에 불필요) ②시작 버전(0.0.1 대 0.1.0)과 증가 단위 ③태그 형식·주석 태그 여부
④릴리즈 커밋 메시지 양식 ⑤릴리즈 노트 위치·양식·언어 ⑥GitHub Release 사용 여부(현 배포는 GitLab).

---

## B. 이번 회차 실행 대상

- **과제: `Sidemenu.onDeleted` 의 'COM' 가드 누락으로 대화 삭제 실패 시 전역 Alert 과 토스트가 겹치고, 실패해도 컨텍스트 메뉴가 열린 채 남는 문제 수정 (가치 3 / 위험 2 / 작업량 M)**

- **왜:** 같은 드롭다운(`PopAppChatSetting`)이 쏘는 형제 핸들러 3개 중 `onSaved`(`Sidemenu.vue:539-544`)만
  `catch (e) { if(e == 'COM') return; toast(...) } finally { openedId.value = null }` 표준형을 지키고,
  `onDeleted`(`Sidemenu.vue:499-501`)는 **가드도 `finally` 도 없다**. 그래서 ①인터셉터가 이미 전역
  Alert 을 띄우고 `Promise.reject('COM')` 한 오류(세션 만료 `interceptors.js:223`, status≠200+BZ01
  `interceptors.js:230-236`, 에러 핸들러 `interceptors.js:319-322`)에 '삭제 중 오류가 발생했습니다.'
  토스트가 겹쳐 뜨고 ②업무 실패(`isSuccess(res)` false)·예외 어느 쪽이든 `openedId` 가 초기화되지
  않아 삭제에 실패한 항목의 드롭다운이 열린 채 남는다(성공 경로 `:491` 에서만 닫는다). 고치면 같은
  드롭다운의 세 동작(공유/삭제/보관)이 실패를 **같게** 처리하게 되고, 직전 4회차가 심플봇 두 팝업에서
  정리한 것과 같은 비대칭이 사라진다.

- **수용 기준:**
  1. `pInf.chat.deleteHistorySession` 이 **세션 만료 또는 status≠200 + `code:'BZ01'`** 응답을 돌려줘
     인터셉터가 전역 Alert 을 띄우고 `'COM'` 으로 reject 할 때, `onDeleted` 는 **토스트를 띄우지 않는다**
     (전역 Alert 만 뜬다).
  2. `'COM'` 이 **아닌** 예외(예: 네트워크 오류)에서는 기존처럼 `toast('삭제 중 오류가 발생했습니다.')`
     가 **그대로** 뜬다 — 안내가 사라지면 안 된다.
  3. HTTP 200 + `isSuccess(res)` false 인 업무 실패에서는 기존처럼 `toast('삭제에 실패했습니다.')` 가
     뜨고, **`historyMap` 에서 해당 세션이 지워지지 않으며** `getHistoryList` 재조회도 일어나지 않는다
     (현 동작 유지 — 이 케이스는 수정 전후 모두 통과해야 하며 범위가 좁음을 고정한다).
  4. 업무 실패·`'COM'`·비-`'COM'` 예외 **세 실패 경로 모두에서 `openedId` 가 `null` 로 초기화**되어
     드롭다운(`AppChatSetting`)이 DOM 에서 사라진다. 형제 `gotoShare`(`:444`)·`onSaved`(`:543`)와 같다.
  5. 성공 경로는 무변경 — `toast('대화가 삭제되었습니다.')`, `historyMap` 에서 해당 `session_id` 제거,
     `getHistoryList(appId, true)` 재조회가 그대로 일어난다.
  6. 테스트가 증명할 것: 위 5개가 **실제 `interceptors.js` → 실제 `pInf.chat.deleteHistorySession.call`
     → 실제 `Sidemenu.vue` 마운트**를 통과해 성립한다는 것. 수정을 되돌리면 1·4 케이스가 다시 실패해야
     한다(인과 분리 증명).

- **건드릴 파일:**
  - `src/components/layout/Sidemenu.vue:448-501` `onDeleted` — ①`catch (e)` 의 토스트 **앞**에
    `if (e == 'COM') return` 추가(저장소 표준형: `Sidemenu.vue:540`, `PopSimpleBotUpdate.vue:679`)
    ②`openedId.value = null` 을 성공 경로(`:491`)에서 빼서 `finally { openedId.value = null }` 로
    옮김(`gotoShare:443-445`·`onSaved:542-544`와 동일 형태). **`isSuccess` false 의 early `return`
    자리, `historyMap` 갱신, `getHistoryList` 재조회는 손대지 말 것.**
  - `tests/unit/sidemenuDeleteCommonError.spec.js` (신규) — 아래 검증 경로.

- **검증 명령:**
  ```
  npm ci                 # node_modules 가 비어 있다. 선행 필수, 수 분 소요
  npx vitest run tests/unit/sidemenuDeleteCommonError.spec.js   # 신규 스펙만 빠르게
  npm test               # 기준선 30파일 503테스트 — 이보다 줄면 안 됨
  npm run build:dev      # 통과 후 dist/ 삭제할 것. npm run build / npm run lint 는 없다
  ```

- **테스트 하네스(이 저장소에서 검증된 형태를 그대로 쓸 것):**
  - HTTP 전송(**axios adapter**) 한 겹만 대역으로 바꾸고 나머지는 전부 실물을 통과시킨다
    — `shareAppCommonError.spec.js`, `simpleBotCreateCommonError.spec.js`,
    `simpleBotUpdateClose.spec.js` 가 확립한 형태. **손으로 주입한 대역 컴포넌트·소스 문자열 검사로
    증거를 삼지 말 것**(운영자 반복 지시).
  - **'COM' 을 만드는 검증된 대역 응답: `{ status: 204, data: { code: 'BZ01', message: '…' } }`**
    (`simpleBotUpdateClose.spec.js:202-204`). `pInf` 는 `inf` 와 같은 `createInstance` 를 쓰므로
    (`pythonInterface.js:8` `createInstance('/papi', …)`) 인터셉터가 동일하게 'COM' 을 던진다 —
    이전 회차에 확인됨.
  - **드롭다운까지 렌더시키는 경로(가장 비싼 부분, 순서대로):**
    1. `Sidemenu` 를 `openSide: false` 로 마운트 → pinia + `useRouter`/`useRoute` 필요
       (`vue-router` 의 `createRouter`+`createMemoryHistory` 로 실물 라우터를 쓰거나
       `useRouter`/`useRoute` 만 모킹). `SmLoading`·`Nodata`·`Ico`·`IconBtn` 은 스텁 가능하나
       **`AppChatSetting`(`views/Chat/Popup/PopAppChatSetting.vue`)은 실물로 둘 것** — 삭제 버튼이
       여기 있다.
    2. `openSide` 를 true 로 바꾸면 `watch`(`:547-557`)가 `fetchAppList(true)` 를 돈다 →
       **`inf.app.getSideList`**(`Sidemenu.vue:206`)에 앱 1개를 돌려줄 대역 응답 필요.
    3. 앱을 클릭해 펼치면 `getHistoryList`(`:387`)가 **`pInf.chat.getHistoryList`**(`:408`)를 돈다 →
       세션 1개(`id`, `session_id`, `app_id`, `prj_id`, `text`)를 돌려줄 것.
    4. `IconBtn label="채팅 설정"` 클릭(`toggleMore`, `:144`) → `openedId` 설정 → `AppChatSetting` 렌더.
    5. `AppChatSetting` 의 삭제 버튼 클릭 → `emit('deleted', { session_id, item })`
       (`PopAppChatSetting.vue:22`). **확인 팝업(Confirm)은 없다** — 클릭 즉시 emit 된다.
  - **주의(실제 코드 함정):** `onDeleted` 는 `item?.prj_id || item.item.prj_id`(`Sidemenu.vue:468`)를
    읽는다. `AppChatSetting` 이 보내는 payload 에는 `item` 키가 항상 있으므로 정상이나, 테스트에서
    핸들러를 직접 부르는 형태로 우회하면 `item.item` 이 없어 TypeError 가 난다 — **실제 emit 경로로만
    돌릴 것**. `appId` 는 `item?.app_id || item?._raw?.app_id || openedAppId.value` 순으로 읽으므로
    세션 객체에 `app_id` 를 넣어 두면 안전하다.
  - `toast`/`openAlert` 는 모듈 싱글턴이라 케이스 간 상태가 샌다 — 각 케이스에서 정리할 것
    (`restoreMocks: true` 는 켜져 있다).

- **범위 축소 규칙(중요):** 위 5단계 렌더 경로를 세션 안에 구동하지 못하겠으면, **대역 컴포넌트나
  핸들러 직접 호출로 대체하지 말고 범위를 줄여 보고할 것.** 그 경우 차선 후보로 넘어간다.

- **위험과 피할 것:**
  - `src/api/common/interceptors.js`·`src/api/auth.js`·`src/storage/{authStorage,userStorage}`·
    `src/router`·`.gitlab-ci.yml` 은 **건드리지 말 것**(보호 경로).
  - `Sidemenu.vue` 의 다른 `catch` 5곳(`:233, 365, 416, 441`)은 **이번 범위가 아니다.** 특히
    `gotoShare:441` 은 API 호출 없이 클립보드 복사만 하므로 'COM' 이 날 수 없다(이전 회차에 rejected).
    한 곳만 고치라는 뜻이 아니라, **근거가 확인된 `onDeleted` 하나만** 고치라는 뜻이다.
  - `openedId` 를 `finally` 로 옮길 때 **성공 경로에서 두 번 대입되지 않게** `:491` 의 대입은
    반드시 제거할 것(남겨도 동작은 같으나 중복이다).
  - 과거 교훈: `finally` 에 **성공 전용 후속 동작**(`emit`/`close`/이동/목록 갱신)을 넣지 말 것.
    `openedId = null`(UI 닫기)은 성공·실패 모두에서 옳으므로 `finally` 가 맞지만, `historyMap` 갱신과
    `getHistoryList` 재조회는 **성공 경로에 그대로 둘 것**.
  - 운영자 지시: 관측 가능한 동작이 바뀌지 않는 정리(미사용 import 제거 등)를 이 커밋에 끼워 넣지 말 것.
  - `Sidemenu` 마운트 하네스는 이 저장소에 전례가 없다 — 이것이 이 과제의 유일한 실질 비용이다.
    **먼저 Red 가 실제로 나는지 확인한 뒤** 수정할 것(수정 전 1·4 케이스가 실패해야 한다).

- **차선 후보: `PopSimpleBot.tryUpdateAppIcon`/`tryUpdateLog` 와 `PopSimpleBotUpdate.tryUpdateAppIcon`
  의 `catch(e){}` 완전 삼킴 정리** (`PopSimpleBot.vue:542,563` / `PopSimpleBotUpdate.vue:649`,
  가치 3 / 위험 2 / 작업량 S). 아이콘 변경만 실패해도 사용자는 성공으로 안내받는다. 단
  **'아이콘 실패가 전체 실패인가' 라는 기대 계약이 미확인**이고, `simpleBotUpdateClose.spec.js` 의
  '아이콘 수정(appEdit) 실패는 전체 수정 성공 판정을 바꾸지 않는다' 케이스가 현 동작을 고정하고
  있다. 계약을 '앱 수정 자체는 성공으로 두되 아이콘 실패는 별도 토스트로 알린다' 로 좁게 잡고
  그 기존 케이스를 함께 갱신하는 형태로만 진행할 것.
