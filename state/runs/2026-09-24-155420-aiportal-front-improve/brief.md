# 과제서 — 2026-09-24-155420-aiportal-front-improve

기준 HEAD: `396b9ae` (main 머지 #38). 작업 트리 clean.

이번 회차는 **A(우선 과제=릴리즈)** 와 **B(코드 과제)** 두 건이다. A 는 결론이 이미 나 있으니 재조사하지 말고, B 를 구현하라.

---

## A. 우선 과제(릴리즈) — **무변경이 정답. 재조사하지 말 것 (28회째)**

- 과제: 릴리즈 버전 결정 입력 부재 (가치 5 / 위험 2 / 작업량 S — 단, **저장소 안에서 수행 불가**)
- 왜: 릴리즈 게이트가 다음 버전을 요구하는데 이 저장소에는 따라 할 관례가 0건이다. 관례를 새로 발명하는 것은 사람의 일이므로 하지 않는다.
- **이번 회차 재실측(2026-09-24 15:5x, 이 워크트리에서 직접 확인함)**:
  1. `git tag | wc -l` = **0**
  2. `package.json` `"version": "0.0.0"`
  3. `ls -a` 전량에 **`.github` 없음**, `CHANGELOG*` 없음, `scripts/`·`Makefile` 없음
  4. `.gitlab-ci.yml` 은 브랜치 배포 전용(`CI_COMMIT_TAG` 참조 0건)
  5. `git log --oneline -5` 에 릴리즈 커밋 양식 0건 (전부 `fix:` + merge PR)
- 구현자가 할 일: **아무것도 하지 않는다.** 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋을 만들지 말고, 원격에 아무것도 보내지 말고, `docs/RELEASE.md` 도 수정하지 말 것. 판정을 `skipped`/`released` 로 낮추지 말 것. `package.json` 의 `version` 필드 삭제 같은 **게이트 완화 우회는 금지**.
- 러너 지시("워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 고치세요")는 **수행 대상이 없다**: 이 저장소에 워크플로 파일이 없고(`.github` 부재), 실패 지점이 워크트리 밖의 외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py` 다.
- 해소에 필요한 사람 입력 4가지(27회차와 동일): ①첫 릴리즈 버전 `0.0.1`/`0.1.0`/`1.0.0` ②태그 형식(`v1.2.3`/`1.2.3`)·주석 태그 여부 ③릴리즈 노트 위치·양식·언어(CHANGELOG.md 신설 여부) ④GitHub Release 사용 여부.

---

## B. 코드 과제 (이번 회차에 실제로 구현할 것)

- 과제: **헤더 공유 패널(HeaderShare)의 `isSuccess()` 가 인자 없이 호출돼 조회 실패가 '프로젝트가 없습니다' 로 감춰지는 문제 수정** (가치 4 / 위험 1 / 작업량 S)

- 왜: `src/components/layout/HeaderShare.vue:32` 이 `if(!isSuccess()){` 로 **응답 인자를 빠뜨린 채** 호출한다 — `isSuccess`(`src/utils/common.js:55-56`)는 `if (!res || !res.data) return false` 라서 **성공·실패 가리지 않고 항상 false** 를 돌려주고, 그런데 분기 안에 `return` 이 없어 바로 다음 줄 `:35` `projectList.value = res.data.body ?? []` 가 덮어쓴다. 즉 가드가 **한 줄도 작동하지 않는다**(성공은 우연히 정상 동작, 실패는 통과). 그 결과 HTTP 200 으로 도착한 업무 실패(`interceptors.js:228-236` 은 status 200 을 거르지 않는다)가 `body` 없이 `[]` 가 되어 `:167` 의 '공유 받을 프로젝트가 없습니다.' 라는 **사실 진술**과 구분되지 않고, 예외는 `:36-38` 에서 **아무 안내 없이 패널만 닫는다**(`close()`). 고치면 같은 저장소의 목록 조회 표준형과 일치해 사용자가 실패를 실패로 본다.

- 수용 기준:
  1. **업무 실패**(`{status:200, data:{code:'BZ99', message:'권한이 없습니다.'}}`)에서 `toast('권한이 없습니다.')` 가 1회 뜨고, 패널의 목록이 갱신되지 않으며(빈 문구로 대체되는 경로를 타지 않음), 전역 Alert 은 뜨지 않는다.
  2. `message` 가 없는 업무 실패(`{status:200, data:{code:'BZ99'}}`)에서는 기본 문구 `'공유 가능한 프로젝트 조회에 실패하였습니다.'` 가 토스트로 뜬다.
  3. **성공 후 재조회가 업무 실패**하면 이미 렌더된 프로젝트 행이 그대로 남는다(= 업무 실패 분기의 `return` 이 실제로 일을 한다). ShareList 에서 같은 변이(②)가 이 한 건만 잡아냈으므로 **이 케이스를 반드시 넣을 것**.
  4. **비'COM' 예외**(예: adapter 가 `Promise.reject(new Error('boom'))`)에서 `toast('공유 가능한 프로젝트 조회 중 오류가 발생하였습니다.')` 가 뜬다.
  5. **'COM' 경로**(`{status:204, data:{code:'BZ01', message:'…'}}`)에서는 토스트가 0건이고 인터셉터의 전역 Alert 만 뜬다(겹치지 않는다).
  6. 변이로 인과를 분리 증명할 것(무효 변경 금지 — 운영자 반복 지시): ①업무 실패 토스트 줄 제거 → 1·2·(3 일부) 만 실패 ②**업무 실패 분기의 `return` 만 제거 → 3) 이 실패** ③catch 토스트 제거 → 4) 만 실패 ④`'COM'` 가드 제거 → 5) 가 실패. 네 변이 모두 기존 38개 스펙은 그대로 통과해야 한다.

- 건드릴 파일:
  - `src/components/layout/HeaderShare.vue:28-42` `getList` — **여기만** 고친다.
    - `:32` `if(!isSuccess()){ projectList.value = [] }` 를 이 저장소의 목록 조회 표준형으로 교체:
      `if (!isSuccess(res)) { toast(res?.data?.message || '공유 가능한 프로젝트 조회에 실패하였습니다.'); return }`
      (인자 `res` 를 넘기는 것 + `return` 두 가지가 모두 필요하다.)
    - `:36-38` `catch` — **`close()` 앞에 `'COM'` 가드와 토스트를 넣는다**: `if (e == 'COM') return` → `toast('공유 가능한 프로젝트 조회 중 오류가 발생하였습니다.')`. `projectList.value = []` 는 유지해도 되지만, **`close()` 를 그대로 둘지는 구현자 판단**이다 — 토스트를 띄우면서 패널을 닫으면 사용자가 문맥을 잃는다. 형제(`ShareList`/`Sidemenu`)는 닫지 않으므로 **`close()` 제거를 권장**하되, 제거가 수용 기준 4 를 넘어 다른 케이스를 깨지 않는지 변이로 확인하고, 애매하면 `close()` 는 남기고 토스트만 추가해도 기준 1~5 는 충족한다.
  - `tests/unit/headerShareFailure.spec.js` (신규) — 아래 하네스 참고.
  - **손대지 말 것**: `src/utils/common.js`(`isSuccess` 자체), `src/api/interface.js`, `HeaderAlarm.vue`, `Header.vue`, `onShareAppSubmit`(`:72-115` — 이미 `!result.ok` + `'COM'` 가드가 있다), 템플릿의 빈 상태 문구.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  npm ci                       # node_modules 가 비어 있으면 선행 필수, 수 분
  npx vitest run tests/unit/headerShareFailure.spec.js
  npm test                     # 전체
  npm run build:dev            # 통과 후 dist/ 삭제
  ```
  - **기준선은 직접 측정할 것.** 이 워크트리에는 `tests/unit/` 에 **38개 spec** 이 있고 `headerAlarm*` 스펙은 **없다**(직전 회차 결과가 아직 이 base 에 머지되지 않았다). 프로필에 적힌 "579" 와 직전 기록의 "583" 이 엇갈리므로 **수정 전에 `npm test` 를 한 번 돌려 숫자를 확정**한 뒤 진행하라.
  - `npm run build` · `npm run lint` 는 **없다**.

- 테스트 하네스 (정착된 증명 형태 — `tests/unit/shareListFailure.spec.js` 를 그대로 본떠라):
  - HTTP 전송(axios adapter) **한 겹만** 대역으로 바꾸고, 실제 `interceptors.js` → 실제 `inf.athena.shareList.call` → 실제 `HeaderShare.vue` 마운트를 통과시킬 것. 손으로 만든 의존성 주입 대역은 금지(운영자 반복 지시).
  - 요청 URL 단정값: **`/athena/share/list`** (`src/api/interface.js:243-246`, `call: () => getApi().get('/athena/share/list', {})` — **파라미터 없음**).
  - 모듈 로드 **전** `vi.stubEnv('VITE_BACKEND_API_TARGET', …)` / `vi.stubEnv('VITE_PYTHON_API_TARGET', …)` 필수(없으면 `api/index.js:16` 이 던진다).
  - **`HeaderShare` 의 열림 watch 는 `{immediate:true}`(`:117-120`)** 이므로 `HeaderAlarm` 과 같이 **`open:true` 로 마운트하면 그 자리에서 조회가 나간다**(다른 팝업들과 달리 `setProps` 로 열 필요 없음).
  - `getList` 는 `readUser()` 를 쓰지 않는다(파라미터 없음) — 그래도 `useRouter()` 를 쓰므로 **메모리 라우터가 필요**하고, `IconBtn`/`TextIconBtn`/`SmLoading` 은 jsdom 에서 그대로 마운트된다(HeaderAlarm 회차에서 확인됨).
  - 토스트는 `setToastProxy`(`@/utils/common.js`)로 수집하고, 전역 Alert 은 `useGlobalAlert()` 의 실제 싱글턴 ref 를 `watch` 로 관찰할 것. 둘 다 모듈 싱글턴이라 **케이스마다 초기화 + `wrapper.unmount()`**.
  - `isSuccess` 대역 규약: 업무 실패 `{status:200, data:{code:'BZ99', message:'…'}}` / 'COM' `{status:204, data:{code:'BZ01', message:'…'}}` / 성공 `{status:200, data:{code:'200', body:[{id:'p1', name:'기획팀 프로젝트'}]}}` — **`body` 가 객체가 아니라 배열**이다(`:35` `res.data.body ?? []`, 템플릿은 `project.id`/`project.name` 을 읽는다, `:133-141`).

- 위험과 피할 것:
  - **`isSuccess` 자체를 고치지 말 것.** 22개 화면이 공유한다. 호출부만 형제와 같게 맞추는 것이 이 저장소의 확정된 방향이다(운영자 지시: "공유 컴포넌트를 넓히지 말고 호출부를 형제와 같게").
  - **토스트를 `'COM'` 가드 *뒤*에 둘 것.** 앞에 두면 인터셉터 전역 Alert 과 겹쳐 수용 기준 5 가 깨진다(ShareList·HeaderAlarm 회차에서 변이 ④로 실제로 잡혔다).
  - **문구를 새로 발명하지 말 것.** 목록 조회 표준형(`Sidemenu.fetchAppList:208-212`, `ShareList.getShareList` 1653ecf)의 `'<대상> 조회에 실패하였습니다.'` / `'<대상> 조회 중 오류가 발생하였습니다.'` 어법을 그대로 쓴다. 상세 조회는 `openAlert`, **목록 조회는 `toast`** — 이 구분을 넘지 말 것.
  - **무효 변경 금지.** 줄을 넣기 전에 빼고 돌려 실패하는 케이스가 있는지 변이로 확인할 것(수용 기준 6). 특히 `return` 한 줄은 수용 기준 3 이 없으면 무효 변경으로 보인다.
  - `HeaderAlarm.vue` 는 이 base 에서 아직 미수정 상태다. **같이 고치려 들지 말 것** — 직전 회차 PR 과 충돌한다.
  - 보호 경로(`src/api/common/interceptors.js`, `src/api/auth.js`, `src/storage/*`, `src/router`) 는 건드리지 않는다.

- 차선 후보: **`onShareAppSubmit`(`HeaderShare.vue:72-115`)의 `String(shareInfo.id)` 가 `selectedProject` 없을 때 문자열 `'undefined'` 가 되어 `if (!projectId)` 가드를 통과하는 문제** — `isShareDisabled`(`:55-57`)가 버튼을 막고 있어 현재 도달 경로가 있는지 미확인이다. 1순위가 성립하지 않으면 먼저 도달 가능성을 확인하고, 도달 불가면 보류 아이디어의 `심플봇 두 팝업의 tryUpdateAppIcon/tryUpdateLog 예외 삼킴` 으로 갈 것(단 `simpleBotUpdateClose.spec.js` 가 현 계약을 고정하므로 케이스 갱신이 선행).

---

### 미확인으로 남긴 것 (구현자가 확인해야 함)
- `npm test` 기준선 숫자(38파일 / 579 또는 583 테스트) — **이번 정찰에서 실행하지 않았다.**
- `catch` 의 `close()` 를 제거하는 것이 다른 화면에 영향이 있는지 — `HeaderShare` 는 `Header.vue:169` 에서 `v-if="showShare"` 로만 렌더되므로 영향 범위는 이 패널 하나로 보이나, 실제 마운트로 확인할 것.
- `inf.athena.shareList.call()` 이 파라미터를 전혀 받지 않는다는 점은 `interface.js:243-246` 원문으로 확인했으나, 백엔드가 세션에서 사용자를 읽는지는 이 저장소만으로 확정 불가(백엔드 별도 저장소).
