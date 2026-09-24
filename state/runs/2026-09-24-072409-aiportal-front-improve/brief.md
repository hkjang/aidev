# 과제서 — 2026-09-24-072409-aiportal-front-improve

이번 회차는 **A(우선 과제: 릴리즈 실패)** 와 **B(실제로 코드를 고칠 과제)** 두 덩어리다.
A 는 저장소 안에 합법적 수단이 없음을 이번 회차에 **읽기 전용으로 다시 실측해** 확인했으므로 **무변경**이고,
구현자는 A 를 **재조사하지 말고 손대지 말 것**. 실제 작업은 B 하나다.

---

## A. 우선 과제(릴리즈 실패 24회째) — 무변경으로 두고 사람 입력을 기다린다

이번 회차에 이 워크트리에서 읽기 전용으로 재확인한 실측(모두 직접 실행):
- `git tag | wc -l` = **0** (태그 전무)
- `package.json` version = **`0.0.0`** (증가 이력 없음)
- 저장소 루트 `ls -a` 전량: `.env.sample .git .gitignore .gitlab-ci.yml .ipynb_checkpoints .vscode AGENTS.md README.md README2.md docs index.html jsconfig.json package-lock.json package.json public src test-scenarios.csv test.txt tests vite.config.js vitest.config.js`
  → **`.github` 없음 / `CHANGELOG.md` 없음 / `VERSION` 없음 / `scripts/` 없음 / `Makefile` 없음**
- `git log --oneline -5` = `1c607a3` (main 머지) 가 HEAD, 릴리즈 커밋 없음, 워킹 트리 clean

결론(직전 23회와 동일, 완화 금지):
- **고칠 워크플로 파일이 이 저장소에 없다.** `.github` 가 없어 GitHub Actions 는 0개이고, `.gitlab-ci.yml` 은
  `CI_COMMIT_BRANCH == "main"` 조건의 빌드·복사 배포 전용으로 `CI_COMMIT_TAG` 참조가 0건이다(태그에 반응하는 자동화 없음).
  실패 지점은 워크트리 **밖**(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이다.
- `package.json` 에 version 필드가 존재하므로 외부 절차의 `skipped` 조건('태그도 버전 파일도 릴리즈 노트도 없음')을 충족하지 않는다.
  **판정을 skipped/released 로 낮추지 말 것.** `AGENTS.md` 와 `docs/RELEASE.md:84` 가 판정 완화·관례 신설을 금지한다.
- README/docs 의 `0.0.1 | 2025-01-01 | 초기 릴리스` 는 `docs/RELEASE.md` 가 '실제 릴리스 미확인' 으로 못박은 **문서 기재**다.
  믿으면 다음이 0.0.2, 안 믿으면 0.0.1 이 되어 답이 갈리므로 번호를 확정할 수 없다.
- 형제 저장소(`aiportal-front-admin`) 의 버전 관례 이식도 **승인되지 않은 관례 이식**이라 금지.

구현자가 할 일: **아무것도 하지 않는다.** 버전 파일·태그·CHANGELOG·릴리즈 노트·릴리즈 커밋·원격 전송 전부 금지,
`docs/RELEASE.md` 도 수정 금지. 회차 노트에 위 실측과 '무변경(24회째)' 을 기록만 한다.

막힌 것을 풀 **사람 입력 4가지**(직전 회차와 동일):
① 첫 릴리즈 버전 = 0.0.1 / 0.1.0 / 1.0.0 중 무엇인가
② 태그 형식(`v1.2.3` 여부)과 주석 태그(`-a`) 사용 여부
③ 릴리즈 노트의 위치·양식·언어
④ GitHub Release 사용 여부(현 배포 경로는 GitLab CI 의 브랜치 배포, 원격은 GitHub)

참고: 이번 머지분(RequestList 검색 무동작 수정)은 product-launch 기준 tier 3(릴리즈 노트만)이라 별도 런치 모션은 불필요하다.

---

## B. 이번 회차에 실제로 고칠 것

- 과제: **프롬프트 선택 팝업(PopPromptSelect)의 검색이 배경 게시판 화면의 검색창을 읽는다 — 중복 DOM id `searchKey`** (가치 4 / 위험 1 / 작업량 S)

- 왜: `SearchBox.vue` 는 `modelValue` 가 없어 호출부가 전부 `document.getElementById('<searchId>')` 로 DOM 값을 직접 읽는 관례인데,
  전역 팝업 `PopPromptSelect.vue` 만 **route 화면 4곳(`NoticeList`/`LibraryList`/`ShareList`/`RequestList`)이 이미 쓰는 일반 id `searchKey` 를 그대로 재사용**한다
  (`PopPromptSelect.vue:107` `getElementById('searchKey')`, `:148` `searchId="searchKey"`).
  이 팝업은 `App.vue:49-62` 의 `Teleport to body` 안 `PopSimpleBotUpdate`(`:780`)/`PopSimpleBot`(`:702`) 내부에서 열리므로
  **게시판 화면이 뒤에 마운트된 채로 함께 존재**하고, `getElementById` 는 문서 순서상 먼저 오는 `#app` 쪽(게시판) input 을 돌려준다
  → 팝업에 무엇을 입력해도 `keyword` 가 빈 문자열이 되어 프롬프트 검색이 무동작이고,
  뒤 게시판 검색창에 글자가 남아 있으면 **그 글자로 프롬프트 목록이 검색된다**(엉뚱한 결과). 형제 팝업·목록은 모두 고유 id
  (`totalSearchKey`·`ocrSearchKey`·`sttSearchKey`·`imgSearchKey`·`chatSearchKey`)를 쓰고 있어 **이 한 곳만 비대칭**이다.

- 수용 기준:
  1) 게시판 목록 화면(`/notice`)이 먼저 마운트돼 `input#searchKey` 가 문서에 있는 상태에서 `PopPromptSelect` 를 열고
     팝업 입력란에 `템플릿` 을 입력 후 검색(아이콘 클릭)하면, 실제로 나가는 `GET /app/prompt` 요청의 `key_word` 가 `템플릿` 이다.
  2) 같은 상황에서 **게시판 검색창에 `공지` 가 입력돼 있고 팝업 입력란은 비어 있으면**, 나가는 요청의 `key_word` 는 `''` 이다
     (배경 화면의 글자가 프롬프트 검색에 새지 않는다).
  3) 팝업 단독(게시판 없음) 상황의 기존 동작이 유지된다: 입력 후 검색 → `key_word` 에 그 값, 앞뒤 공백은 `trim()` 되고,
     공백만 입력하면 `key_word` 가 `''`, 검색 시 페이지는 1로 리셋된다(`resetPageAndGet`).
  4) 테스트는 **수정 전 1)·2) 가 실패(Red)** 하고 수정 후 전부 통과함을 보인다. 3) 은 수정 전후 모두 통과해 범위가 좁음을 고정한다.
  5) 변이 1건으로 인과를 분리 증명: 템플릿의 `searchId` 만 원래 값으로 되돌리면 1)·2) 만 실패하고 나머지는 통과한다.

- 건드릴 파일 (두 줄만. 그 외 금지):
  - `src/components/common/popup/Global/PopPromptSelect.vue:148` — `<SearchBox searchId="searchKey" …>` → `searchId="promptSearchKey"`
  - `src/components/common/popup/Global/PopPromptSelect.vue:107` (`onSearch`) — `getElementById('searchKey')` → `getElementById('promptSearchKey')`
  - 신규 스펙 `tests/unit/promptSelectSearch.spec.js` — 아래 검증 형태로 작성
  - **`src/components/common/Input/SearchBox.vue` 는 건드리지 말 것**(10화면 공유. 공유 쪽을 넓히지 말고 호출부를 형제와 같게 맞추는 방향이 이 저장소의 확정된 수정 방향).

- 검증 명령:
  - `npm ci` (node_modules 가 비어 있으면 선행 필수, 수 분)
  - `npx vitest run tests/unit/promptSelectSearch.spec.js` (신규 스펙 단독)
  - `npm test` — 기록상 기준선 **35파일 560테스트**(이번 정찰에서는 실행하지 않았다. 실제 기준선은 구현자가 수정 전에 한 번 돌려 확정할 것)
  - `npm run build:dev` 후 `dist/` 삭제. (`npm run build` · `npm run lint` 는 이 저장소에 없다)

- 검증 형태(이 저장소에 정착된 것을 그대로 쓸 것):
  - HTTP 전송(axios adapter) 한 겹만 대역 → 실제 `interceptors.js` → 실제 `inf.app.getPrompt.call`(`src/api/interface.js:322-325`, `getApi().get('/app/prompt', { params })`) → 실제 컴포넌트 마운트.
  - 모듈 로드 **전에** `vi.stubEnv('VITE_BACKEND_API_TARGET', 'http://localhost:9999')`(없으면 `api/index.js:16` 이 던진다).
  - 기존 `tests/unit/simpleBotPromptReload.spec.js:1-60` 의 하네스(respond/makeHttpError/flushMicrotasks, `writeUser`/`clearUser`)를 그대로 재사용하면 빠르다.
  - `PopPromptSelect` 는 `getList` 가 `readUser()` 의 user_id 를 요구한다(`:56-64`) → `writeUser` 로 채우지 않으면 `openAlert` 후 요청이 아예 안 나간다.
  - `Base.vue:43` 이 `v-if="modelValue"` 이므로 팝업 input 은 **열려 있을 때만** DOM 에 있다. `modelValue: true` 로 마운트하거나 `setProps` 로 열 것
    (열림 watch `:112-118` 가 `keyword=''` 후 `resetPageAndGet()` 하므로 **최초 조회 요청이 한 번 나간다** — 검색 요청은 그 다음 요청으로 골라야 한다).
  - **문서 순서가 결함의 본질이다**: 두 컴포넌트 모두 `mount(..., { attachTo: document.body })` 로 붙이고, **게시판(`NoticeList`)을 먼저 마운트**한 뒤 팝업을 마운트해야 `getElementById` 가 게시판 것을 집는다(현실의 `#app` → teleport 순서와 같다).
  - `NoticeList` 는 메모리 라우터로 `/notice` 를 실제 push 해야 목록 요청이 정상적으로 나간다. 게시판 목록 요청(`/board`)도 같은 adapter 로 들어오므로 **URL 로 요청을 갈라서** 세야 한다.
  - 케이스마다 두 wrapper 를 `unmount` 할 것(같은 id 를 쓰는 형제 스펙 `requestListSearch.spec.js`·`boardListFailure.spec.js` 와 DOM 이 섞인다).
  - 전역 알림/토스트/로딩은 모듈 싱글턴이라 케이스마다 초기화 필요(`restoreMocks: true` 로는 안 지워진다).

- 위험과 피할 것:
  - **무효 변경 금지**(운영자 반복 지시). 이 수정은 관측 가능한 변화가 있다 — 수용 기준 1)·2) 가 그것을 증명한다. 변이(기준 5)로 인과를 반드시 분리해 둘 것.
  - `PopPromptSelect.vue` 의 나머지(`getList` 의 `isSuccess`/`openAlert` 처리, 페이징, `onSelect` 의 payload, `raw?.recent_version.id` 접근)는 **이번 범위가 아니다**. 손대지 말 것.
  - 게시판 4화면(`NoticeList`/`LibraryList`/`ShareList`/`RequestList`)의 `searchKey` id 는 **그대로 둔다**. 라우트는 서로 배타적이라 그쪽끼리는 충돌하지 않으며, 4곳을 바꾸는 편이 변경면이 넓고 형제 스펙 4벌을 함께 흔든다. 고유 id 를 새로 받는 쪽은 팝업 하나면 충분하다.
  - `NoticeList:114` / `LibraryList:112` / `ShareList:93` / `PopGlobalSearch:110` 에 남아 있는 아무것도 연결하지 않는 `v-model="searchKey"` 정리는 **별건**이다(동작 변화 0). 이번에 끼워 넣지 말 것.
  - 보호 경로(`src/api/common/interceptors.js`, `src/api/auth.js`, `src/storage/{auth,user}Storage`, `src/router`, `src/utils/page.js`)는 건드리지 않는다.
  - 과거 교훈: 손으로 주입한 대역·소스 문자열 검사로 증명하지 말 것. **실제 마운트 + 실제 요청 파라미터**로 증명한다.

- 차선 후보: **`isSuccess` 검사가 없는 잔여 3곳 중 `ShareList` 하나만** 목록 표준형(`Sidemenu.fetchAppList` 의 업무 실패 `toast(res?.data?.message || '<대상> 조회에 실패하였습니다.')` + `if (e !== 'COM')` 토스트)에 맞추기. `HeaderAlarm`/`PopWidgetSetting` 은 보조 UI 라 기대 계약이 미확인이므로 **함께 묶지 말 것**.
