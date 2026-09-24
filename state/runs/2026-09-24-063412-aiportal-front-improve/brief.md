# 과제서 — 2026-09-24 (base main@26a8f9d)

이번 회차는 **A(우선 과제: 릴리즈)** 와 **B(실제 구현 과제)** 두 갈래다.
A 는 진입 조건이 저장소 안에서 충족되지 않는다는 것이 이번 워크트리에서 다시 확인됐으므로 **재조사하지 말고 무변경으로 두고**, B 를 구현하라.

---

## A. [우선 과제] 릴리즈 — 무변경 (23회째, 저장소 안에 합법적 수단 없음)

- 이번 워크트리(HEAD `26a8f9d`, 작업 트리 clean)에서 실측 재확인: `git tag | wc -l` = **0**, `.github` 디렉터리 **없음**, `CHANGELOG.md`/`VERSION`/`scripts/`/`Makefile` **없음**, `package.json` version = `0.0.0`.
- `.gitlab-ci.yml` 은 `CI_COMMIT_BRANCH == main|develop` 조건의 빌드·복사 배포 전용이며 `CI_COMMIT_TAG` 참조 0건 → **고칠 워크플로 파일이 이 저장소에 없다**. 실패 지점은 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`).
- `docs/RELEASE.md` 는 릴리즈 노트·증가 정책이 아니라 **근거 문서**이고, 그 문서와 `AGENTS.md` 가 "확인되지 않은 관례를 새로 만들거나 릴리즈 판정 조건을 완화하지 않는다" 를 명시한다. `package.json` 이 존재하므로 외부 절차의 `skipped` 조건(태그·버전 파일·릴리즈 노트가 모두 없음)도 충족하지 않는다.
- **구현자가 할 일: 아무것도 하지 않는다.** 버전 파일 수정·태그·CHANGELOG 신설·릴리즈 노트·릴리즈 커밋·원격 전송 일체 금지. `docs/RELEASE.md` 도 수정 금지. 판정을 skipped/released 로 낮추지 말 것. `package.json` 의 `version` 필드 삭제는 게이트 완화라 금지.
- 막힌 것을 풀 사람 입력 4가지(직전 회차와 동일): ①첫 릴리즈 버전(0.0.1 / 0.1.0 / 1.0.0) ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부(현 배포는 GitLab CI 브랜치 배포).

---

## B. 구현 과제

- 과제: **요청 게시판(RequestList)의 검색이 무동작 — 검색어가 요청에 실리지 않는 문제 수정** (가치 4 / 위험 2 / 작업량 S)

- 왜: `SearchBox.vue` 는 `modelValue` prop 도 `input` 의 `:value`/`@input` 바인딩도 없어서(`src/components/common/Input/SearchBox.vue:3-20,30`) 호출부의 `v-model` 은 **아무것도 연결하지 않는다**. 형제 4화면(`NoticeList:95-96`, `LibraryList:93-94`, `ShareList:74-76`, `PopGlobalSearch:32-34`)은 모두 `searchId` 를 넘기고 `onSearchClick` 에서 `document.getElementById(...)` 로 DOM 값을 직접 읽어 우회하는데, **`RequestList` 만** `searchId` 를 넘기지 않고(`src/views/Request/RequestList.vue:226`) `onSearchClick`(`:85-96`)이 `searchKey.value` 를 그대로 쓴다. 그 결과 `searchKey` 는 영영 빈 문자열이고 `getBoardList` 의 `if (searchKey.value && ...)`(`:123-125`)가 never-true 라 **무엇을 입력하고 검색을 눌러도 `searchKey` 파라미터가 요청에 실리지 않고 전체 목록이 다시 온다**. 고치면 요청 게시판 검색이 실제로 동작하고, 같은 값을 읽는 5개 경로가 모두 같은 방식(DOM id)으로 같은 값을 읽게 된다.

- 수용 기준:
  1) `/request` 에서 검색 입력창에 값을 넣고 검색 버튼(`button.btn-ico.ico-search`)을 클릭하면, 실제로 나가는 요청 URL 의 쿼리에 `searchKey=<입력값>` 이 포함된다. (지금은 포함되지 않는다 — Red)
  2) 입력창에서 **Enter** 를 눌러도 같은 결과가 된다(`SearchBox.vue:30` 의 `@keydown.enter`).
  3) 앞뒤 공백만 넣거나(`'  '`) 빈 값으로 검색하면 `searchKey` 파라미터가 아예 실리지 않는다(현 `trim()` 계약 유지). 값이 있으면 `trim()` 된 값이 실린다.
  4) 검색 시 `currentPage` 가 1 로 리셋되어 요청의 `pageNum=1` 이 유지된다(`:94` 기존 동작 — 회귀 없음).
  5) 형제 화면의 동작은 바뀌지 않는다: `npm test` 기존 스펙 전부 통과, 특히 `tests/unit/boardListFailure.spec.js`(NoticeList/LibraryList/RequestList 18건)가 그대로 통과한다.
  6) 테스트는 **실제 컴포넌트 마운트 + 실제 `interceptors.js` + 실제 `inf.library.list.call`** 을 통과해 증명한다(소스 문자열 검사·수동 주입 대역 금지). 어댑터가 받은 URL 로 단정할 것.

- 건드릴 파일:
  - `src/views/Request/RequestList.vue:226` — `<SearchBox ...>` 에 `searchId="searchKey"` 추가(형제 4화면과 동일). `v-model="searchKey"` 는 지금도 무의미하므로 **제거해도 되고 둬도 된다 — 제거를 권장**(제거해도 동작 변화 0 이지만, 남겨 두면 다음 사람이 또 속는다. 둘 중 하나만 택하고 커밋 메시지에 적을 것).
  - `src/views/Request/RequestList.vue:85-96 onSearchClick` — 함수 첫머리에 형제와 **문자 단위로 같은** 두 줄을 추가:
    ```js
    const el = document.getElementById('searchKey')
    searchKey.value = (el?.value || '').trim()
    ```
    그 뒤 기존 `currentPage.value = 1` → `getBoardList()` 는 그대로 둔다. 주석 처리된 글자수 검증 블록(`:86-93`)은 **건드리지 말 것**(살리는 것은 승인되지 않은 기능 변경).
  - `tests/unit/requestListSearch.spec.js` (신규) — 아래 검증 참고.
  - **`src/components/common/Input/SearchBox.vue` 는 건드리지 말 것** (아래 위험 참조).

- 검증 명령:
  ```
  npm ci                       # node_modules 가 비어 있다. 선행 필수, 수 분 걸림
  npx vitest run tests/unit/requestListSearch.spec.js
  npx vitest run tests/unit/boardListFailure.spec.js
  npm test                     # 기준선 34파일 552테스트 → 신규 스펙만큼 증가
  npm run build:dev            # 통과 확인 후 생성된 dist/ 삭제
  ```
  (`npm run build` · `npm run lint` 는 이 저장소에 **없다**.)

  테스트 하네스는 **`tests/unit/boardListFailure.spec.js` 를 그대로 베낄 것**(이미 RequestList 를 메모리 라우터 `/request` 로 push 하고, `axios.defaults.adapter` 한 겹만 대역으로 바꿔 `requestedUrls` 에 URL 을 모으고, `writeUser({ user_id:'tester', is_admin:'N' })` 로 `creatorId` 를 채우고, `vi.stubEnv('VITE_BACKEND_API_TARGET'/'VITE_PYTHON_API_TARGET', …)` 를 모듈 로드 **전에** 한다 — 없으면 `src/api/index.js:16` 이 던진다).
  절차: 마운트 → `onMounted` 의 첫 조회가 끝나길 기다림 → 입력창에 값 주입 → 검색 클릭/Enter → **두 번째** 요청 URL 을 단정.

- 위험과 피할 것:
  - **`SearchBox.vue` 에 `modelValue` 를 새로 다는 "근본 수정" 을 하지 말 것.** 이 컴포넌트는 10곳에서 쓰이고 그중 4곳은 이미 DOM 을 직접 읽어 우회하며, `PopGlobalSearch.vue:85` 처럼 `searchQuery.value = ""` 로 ref 를 비우는 곳이 있어 양방향 바인딩을 켜면 **입력창이 갑자기 지워지는 등 관측 가능한 동작이 4~10개 화면에서 동시에 바뀐다**. 한 세션 범위를 넘고, 공유 컴포넌트를 넓히는 방향이라 운영자 지시(호출부를 좁히는 방향으로 고칠 것)에도 어긋난다. 이번 회차는 **RequestList 만 형제와 같은 모양으로 맞춘다.**
  - `document.getElementById` 를 쓰므로 테스트에서 **`mount(..., { attachTo: document.body })` 가 필요**하다(붙이지 않으면 입력이 document 에 없어 `el` 이 null 이라 Green 이 안 난다). 케이스마다 `wrapper.unmount()` 로 정리할 것 — 같은 id `searchKey` 를 쓰는 형제 스펙과 섞이면 오염된다.
  - **무효 변경 금지(2회차 연속 지적받은 자리)**: 넣은 줄을 빼고 돌렸을 때 실제로 실패하는 케이스가 있는지 변이로 확인할 것. 최소 2건 — ①`getElementById` 두 줄 제거 → 검색 케이스만 실패 ②`searchId="searchKey"` 제거 → 같은 케이스만 실패. `v-model` 제거는 동작 변화가 0 임을 알고 하는 정리이니 변이 근거를 요구하지 말 것.
  - `RequestList` 는 `inf.library.*` 를 호출한다(요청 게시판 전용 엔드포인트가 `interface.js` 에 없다 — `:93` board, `:106` library). **이것은 이번 과제가 아니다. 엔드포인트를 바꾸지 말 것** — 백엔드가 별개 저장소라 이 저장소만으로 확정 불가하고 바꾸면 화면이 통째로 죽는다.
  - `isFirst`/`searchCount`/`excludedCount`(`:82-83`, `:144-145`) 의 카운트 로직은 건드리지 말 것. 검색 결과로 이 값들이 어떻게 변해야 하는지는 **미확인**이며 화면에 관측되는 것은 `totalCount` 뿐이다.
  - 보호 경로(`src/api/common/interceptors.js`, `src/api/auth.js`, `src/storage/{authStorage,userStorage}`, `src/router`, `src/utils/page.js`)는 손대지 말 것.

- 차선 후보: **`SupportOcrList`/`SupportSttList`/`SupportImgList`/`ChatStorageList`/`PopPromptSelect` 의 SearchBox 사용 실태 확인 후 같은 결함이 있는 화면 수정** — 이 5곳은 `v-model` 자체를 쓰지 않고 `searchId` 만 넘긴다(위 grep 으로 확인). `onSearchClick` 이 DOM 을 읽는지 **이번 회차에 열어 보지 않았다(미확인)**. B 가 성립하지 않으면 이 5곳을 먼저 읽고, `searchId` 를 넘겨 놓고 DOM 을 읽지 않는 화면이 있으면 같은 방식으로 고쳐라. 그마저 없으면 `ShareList`/`HeaderAlarm`/`PopWidgetSetting` 의 `isSuccess` 검사 부재(보류 아이디어)로 갈 것.
