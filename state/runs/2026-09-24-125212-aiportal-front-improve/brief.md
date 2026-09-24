# 과제서 — 2026-09-24-125212-aiportal-front-improve

이 회차에는 **A(우선 과제: 릴리즈)** 와 **B(코드 개선)** 두 가지가 있다.
A 는 결론이 이미 나 있으니 **재조사하지 말고 무변경으로 두고**, B 를 구현하라.

---

## A. 우선 과제(릴리즈) — 결론: 무변경 (27회째). 구현자는 재조사하지 말 것.

러너가 적재한 오류 대응은 "릴리즈 실패 — 다음 버전을 결정할 근거가 없다(27회째 동일 교착)" 이다.
**이번 회차 정찰이 워크트리에서 다시 확인한 것**(직접 실행해 본 것만 적는다):

- `git tag | wc -l` → **0**
- `package.json:3` → `"version": "0.0.0"` (name `kcb_ai`)
- `git status --porcelain` → 무출력(clean), HEAD = `396b9ae` (main 머지 #38)
- `ls -a` 전량에 **`.github` 없음**, `CHANGELOG*` 없음, `scripts/`·`Makefile` 없음
- 루트 파일 전량: `.env.sample .gitignore .gitlab-ci.yml .ipynb_checkpoints .vscode AGENTS.md README.md README2.md docs index.html jsconfig.json package-lock.json package.json public src test-scenarios.csv test.txt tests vite.config.js vitest.config.js`

러너 지시("워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치세요")는 **이 저장소에 수행 대상이 없다**:
고칠 워크플로 파일이 저장소에 존재하지 않고(`.github` 부재, `.gitlab-ci.yml` 은 `CI_COMMIT_TAG` 참조 0건인 브랜치 배포 전용),
실패 지점이 워크트리 **밖**(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이다.

### 구현자가 할 일 (A)
1. **아무것도 하지 마라.** 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋을 일체 만들지 말고, 원격에 아무것도 보내지 말고, `docs/RELEASE.md` 도 수정하지 마라.
2. 판정을 `skipped` / `released` 로 낮추지 마라. `package.json` 에 `version` 필드가 **실재**하므로 외부 절차의 `skipped` 조건("태그도, 버전 파일도, 릴리즈 노트도 없음")을 충족하지 않는다.
3. **게이트를 느슨하게 만드는 우회는 금지**다 — `version` 필드 삭제, `release.json` 수정, 게이트 조건 완화 모두 하지 마라.
4. 회차 노트에 위 근거를 인용하고 "무변경(27회째)" 로 기록하라. 재실측하지 마라(정찰이 이미 했다).

### 해소에 필요한 사람 입력 4가지 (직전 26회차와 동일 — 사람만 결정할 수 있다)
1. 첫 릴리즈 버전을 `0.0.1` / `0.1.0` / `1.0.0` 중 무엇으로 할지 (README·`docs/01-시작하기.md` 에 '0.0.1 / 2025-01-01 / 초기 릴리스' 기재가 있으나 실제 릴리스 증거는 미확인)
2. 태그 형식(`v1.2.3` / `1.2.3`)과 주석 태그 여부
3. 릴리즈 노트의 위치·양식·언어 (`CHANGELOG.md` 신설 여부 포함)
4. GitHub Release 사용 여부 — 현재 배포는 GitLab CI 브랜치 배포뿐이다

---

## B. 이번 회차 코드 과제

- **과제: 알림 패널(HeaderAlarm)의 OCR 상태 조회 실패가 '알림이 없습니다.' 로 감춰지는 문제 수정 (가치 3 / 위험 1 / 작업량 S)**

- **왜**: `src/components/layout/HeaderAlarm.vue` 의 `getList`(:41-71)는 `isSuccess(res)` 검사가 없어 HTTP 200 으로 도착한 업무 실패를 그대로 통과시키고(`interceptors.js:228-236` 은 status 200 응답을 거르지 않는다), `catch`(:66-69)는 `console.error` 만 찍고 조용히 삼킨다 — 두 경로 모두 `alarmList` 가 `[]` 로 남아 템플릿(:100-102)의 **'알림이 없습니다.'** 라는 *사실 진술*과 구분되지 않는다. 특히 `ocrStatusCheckStore` 폴링이 세운 `new` 배지(`Header.vue:161`)를 보고 사용자가 벨을 눌렀는데 조회가 실패하면, **배지는 빨간데 패널은 "없습니다"** 라는 모순을 보게 되고 재시도할 이유조차 알 수 없다. 이 저장소가 이미 4번(게시판 목록 3형제 `ce28031`, `ShareList` `1653ecf`) 같은 결함을 **확정된 표준형**으로 고쳤고, 남은 마지막 한 곳이 여기다.

- **기대 계약이 확인됐다(과거 회차가 '미확인' 이라 미뤘던 부분 — 정찰이 이번에 해소함)**: 과거 보류 사유는 "알림 뱃지는 주기적으로 갱신되는 보조 UI 라 실패마다 토스트를 띄우면 시끄러울 수 있다" 였다. **이는 사실이 아니다.** `Header.vue:166` 은 `<HeaderAlarm v-if="showAlarm" :open="showAlarm" />` 이고 `showAlarm` 은 `toggleAlarm()`(`Header.vue:129-131`) 즉 **사용자의 벨 클릭으로만** true 가 된다. `HeaderAlarm` 의 `watch(() => props.open, …, {immediate:true})`(:73-76)는 마운트 때 한 번만 돌고, 마운트 중에는 `open` 이 계속 true 라 재조회가 없다. 폴링은 `ocrStatusCheckStore` 쪽이지 이 컴포넌트가 아니다. → **`getList` 는 사용자가 벨을 누를 때마다 정확히 1회** 호출되는 사용자 개시 조회이며, 목록 화면과 같은 취급이 옳다.

- **수용 기준**
  1. 업무 실패(HTTP 200 + `code`가 `'200'` 아님)에 `toast(res?.data?.message || '알림 조회에 실패하였습니다.')` 가 정확히 1회 뜨고, 함수가 거기서 `return` 한다(뒤의 `markAsRead()`·목록 매핑이 실행되지 않는다).
  2. 서버 메시지가 없으면 기본 문구 `'알림 조회에 실패하였습니다.'` 가 뜬다.
  3. 예외 경로에서 `'COM'` **이 아닐 때만** `toast('알림 조회 중 오류가 발생하였습니다.')` 가 뜬다. `'COM'` 일 때는 토스트가 **0건**이어야 한다(인터셉터의 전역 Alert 과 겹치면 안 된다).
  4. 성공 경로는 그대로다 — `status === 'complete'` 면 `markAsRead()` 가 호출되고, `todayOcrList` 가 `stateNm` 이 붙은 채 `alarmList` 에 들어가며, 패널에 항목이 렌더된다. 토스트는 0건.
  5. `isLoading` 은 세 경로(성공/업무 실패/예외) 모두에서 `finally` 로 false 가 된다(기존 `finally` 유지).
  6. **변이로 인과를 분리 증명**하라(무효 변경 금지): ①업무 실패 `toast` 줄만 제거 → 기준 1·2 케이스만 실패 ②업무 실패 분기의 `return` 만 제거 → **실패하는 케이스가 있어야 한다**(없으면 `return` 을 넣지 마라) ③`catch` 토스트만 제거 → 기준 3 의 비-'COM' 케이스만 실패 ④`'COM'` 가드만 제거 → 기준 3 의 'COM' 케이스가 실패. 각 변이는 기존 38개 스펙을 깨지 않아야 한다.

- **건드릴 파일**
  - `src/components/layout/HeaderAlarm.vue:41-71` `getList` — **이 함수 하나만.**
    - import 에 `import { isSuccess, toast } from '@/utils/common'` 추가(현재 `common.js` 를 import 하지 않는다 — 확인함).
    - `const res = await inf.tools.statusOcr.call({ user_id: userId })` 바로 뒤에 표준형 블록 삽입:
      `if (!isSuccess(res)) { toast(res?.data?.message || '알림 조회에 실패하였습니다.'); return }`
    - `catch (e)` 의 `console.error(...)` 는 두고, 그 뒤에 `if (e !== 'COM') { toast('알림 조회 중 오류가 발생하였습니다.') }` 를 넣어라. **토스트는 반드시 가드 뒤**(과거 회차에서 겹침이 실제로 잡혔다).
  - `tests/unit/headerAlarmFailure.spec.js` — **신규.** 아래 "검증" 의 형태로.

- **문구·수단을 새로 발명하지 마라** — 이 저장소의 **확정된 구분**을 그대로 따른다:
  - 목록 조회 실패 = `toast` (상세 조회 실패 = `openAlert`. 알림 패널은 목록이므로 `toast`).
  - 표준형 원본: `Sidemenu.fetchAppList`(`src/components/layout/Sidemenu.vue:208-212` 업무 실패 `toast(res?.data?.message || '<대상> 조회에 실패하였습니다.')` 후 `return` / `:232-236` `if (e !== 'COM') { toast('<대상> 조회 중 오류가 발생하였습니다.') }`). **직접 열어 확인함.**

- **검증 명령**
  ```bash
  npm ci                     # node_modules 가 비어 있다 — 선행 필수, 수 분
  npm test                   # 수정 전에 먼저 돌려 기준선 확정 (기록상 38파일/579테스트 — 재확인할 것)
  npx vitest run tests/unit/headerAlarmFailure.spec.js   # Red → Green
  npm test                   # 전체 회귀
  npm run build:dev && rm -rf dist   # 빌드 확인 후 dist 삭제
  ```
  `npm run build` · `npm run lint` 는 **없다**(스크립트 미정의).

- **검증 형태(이 저장소의 정착된 증명 형태 — 벗어나지 마라)**
  - HTTP 전송(axios adapter) **한 겹만** 대역으로 바꾸고, 실제 `interceptors.js` → 실제 `inf.tools.statusOcr.call`(`GET /tools/ocr/status`, `{params:{user_id}}`) → **실제 `HeaderAlarm.vue` 마운트**를 통과시킬 것. 손으로 만든 의존물 주입이나 소스 문자열 검사는 증거로 인정되지 않는다.
  - 모듈 로드 **전에** `vi.stubEnv('VITE_BACKEND_API_TARGET', …)` / `vi.stubEnv('VITE_PYTHON_API_TARGET', …)` 필수 — 없으면 `src/api/index.js:16` 이 던진다.
  - 대역 응답: 업무 실패 = `{status:200, data:{code:'BZ99', message:'…'}}` / `'COM'` = `{status:204, data:{code:'BZ01', message:'…'}}` / 성공 = `{status:200, data:{code:'200', body:{status:'complete', todayOcrList:[{id:1, group_id:'g1', file_name:'a.pdf', status:'0'}]}}}`. (`isSuccess` 는 `common.js:55-62` — `res.data` 없으면 false / `success` boolean 우선 / 아니면 `code==='200'`. 직접 읽고 확인함.)
  - `readUser()?.user_id` 가 비면 `getList` 가 **요청 없이 즉시 return** 한다(:42-43). `writeUser` 는 `normalizeUserRecord`(`userStorage.js:50-63`)를 거쳐 `userId → user_id` 로 바꾸므로 **입력 키는 `userId`**: `writeUser({ userId: 'tester' })`. 첫 케이스에서 요청이 실제로 나갔음(URL·params)을 단정해 harness 가 유효함을 고정하라.
  - **`HeaderAlarm` 은 `{immediate:true}` 라 `open: true` 로 마운트하면 최초 조회가 그대로 나간다** — Base.vue 계열 팝업(값이 *바뀔 때만* 도는 watch)과 **다르다**. 닫은 채 마운트해 `setProps` 할 필요 없다.
  - `useRouter()` 를 쓰므로 메모리 라우터 플러그인을 붙여 마운트할 것(`readAlarm` 이 `/support/ocr/detail` 로 push).
  - 토스트/알림/`bus` 는 모듈 싱글턴이다 — 케이스마다 카운터 리셋 + 정리. `markAsRead` 는 `ocrStatusCheckStore` 를 통해 localStorage 를 만지므로 케이스마다 초기화할 것.
  - 자식 `IconBtn` / `SmLoading` 은 스텁하지 말고 그대로 마운트하라(실제 배선 통과가 요건).

- **위험과 피할 것**
  - **`catch` 의 `alarmList.value = []` 를 건드리지 마라.** 컴포넌트가 `v-if` 로 열 때마다 새로 마운트되고 마운트 중 재조회가 없어, 이 줄을 빼도 **관측 가능한 동작 변화가 0** 이다(ShareList 와 달리 '이미 렌더된 행이 사라지는' 손실이 여기엔 없다). 넣거나 빼면 무효 변경으로 비평에 걸린다.
  - `ocrStatusCheckStore.js`(폴링·localStorage), `Header.vue`, `src/api/interceptors.js`, `src/utils/common.js`, `SearchBox.vue` — **전부 손대지 마라.** 이 과제는 `HeaderAlarm.vue` 한 파일 + 신규 스펙 한 개다.
  - `markAsRead()` 호출 조건(`status === 'complete'`)과 `stateNm` 매핑 로직을 바꾸지 마라. 업무 실패 시 `return` 하면 자연히 도달하지 않는다.
  - `console.error` 를 지우지 마라(동작 변화 0 인 축소는 필요 없고, 스펙이 stderr 를 볼 수도 있다).
  - 기존 스펙 `tests/unit/ocrStatusCheckStore.spec.js` 가 스토어 동작을 고정하고 있다 — 깨뜨리지 말 것.
  - 보호 경로(auth/migrations/workflows) 는 이 과제와 무관하다. 접근하지 마라.

- **차선 후보: 심플봇 두 팝업의 `tryUpdateAppIcon`/`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼킴 (가치 3 / 위험 2 / 작업량 S)** — `src/components/common/popup/Global/PopSimpleBot.vue:542,563` / `PopSimpleBotUpdate.vue:649`. 계약을 "앱 저장은 성공 판정 유지, 아이콘 갱신 실패만 별도 토스트" 로 두고 기존 `tests/unit/simpleBotUpdateClose.spec.js`(현 동작 '아이콘 실패는 성공 판정을 바꾸지 않는다' 를 고정)를 깨지 않는 선에서 토스트만 추가할 것. **1순위가 성립하지 않을 때만** 고르고, 1순위와 같이 하지 마라.
