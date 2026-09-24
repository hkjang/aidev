# 과제서 — 2026-09-24-044412-aiportal-front-improve

이번 회차는 **A(우선 과제, 릴리즈)** 와 **B(실행 과제)** 두 항목이다.
A 는 저장소 안에 합법적 수단이 없어 **무변경**이 정답이고, 실제로 손을 움직일 대상은 B 다.

---

## A. 우선 과제(자동 배정) — 릴리즈 실패: 이번 회차도 무변경이 정답이다

- 과제: 릴리즈 버전 결정 입력 부재 (가치 5 / 위험 2 / 작업량 S — 단, **저장소 안에서 수행 불가**)
- 판정: **재조사하지 말고 아무것도 만들지 말 것.** 20회 연속 같은 원인이며 이번 회차 자동 적재된
  실패 설명 자체가 이미 워크트리 HEAD `fb38bdd` 에서 재조회한 근거를 담고 있다. 정찰이 이번 회차에
  직접 재확인한 것: `git tag` **0개**, `package.json:3` `version` = `0.0.0`(`name` 은 `kcb_ai`,
  `private: true`), `git status --porcelain` 무출력.
- 왜 저장소 안에서 못 고치나:
  1. 과제서 지시("워크플로 파일과 실패한 단계의 스크립트를 고치라")의 대상이 **이 저장소에 없다**.
     `.github` 디렉터리 자체가 없고, `.gitlab-ci.yml` 은 `CI_COMMIT_BRANCH == "main"|"develop"` 조건의
     **브랜치 푸시 배포 전용**(`CI_COMMIT_TAG` 참조 0건)이다. 실패 지점은 워크트리 밖의 외부 절차
     (aidev `bin/run.sh` → `release.json` → `bin/gate.py`)다.
  2. 저장소 안에서 게이트를 통과시키는 유일한 수단은 **버전 파일·태그·릴리즈 노트 관례를 새로 발명**하는
     것인데, 이는 이번 세션 프롬프트("워크플로 자체를 느슨하게 만들어 통과시키는 것은 금지")와
     `AGENTS.md`("확인되지 않은 관례를 새로 만들거나 릴리즈 판정 조건을 완화하지 않는다"),
     `docs/RELEASE.md`(스스로 "릴리즈 노트나 새 릴리즈 정책이 아니며 … 임의 패치 증가나 새 관례를
     만들지 않는다") 셋 모두에 정면으로 걸린다.
  3. `package.json` 이라는 **버전 파일이 존재**하므로 외부 절차의 `skipped` 조건도 충족하지 않는다.
     `package.json` 의 `version` 필드를 지워 `skipped` 를 만드는 것은 게이트 완화라 **금지**.
- **구현자가 할 일: 아무것도 하지 않는다.** 버전 파일·태그·`CHANGELOG`·릴리즈 노트·릴리즈 커밋·
  원격 전송을 일체 만들지 말고, `docs/RELEASE.md` 도 수정하지 말 것. 판정을 `skipped`/`released` 로
  낮추지도 말 것. 회차 노트에는 "진입 조건 미충족으로 무변경(20회째)" 와 **사람 입력 필요 4가지**
  (①첫 릴리즈 버전 ②태그 형식·주석 태그 여부 ③릴리즈 노트 위치·양식·언어 ④GitHub Release 사용 여부)를
  그대로 남길 것. 형제 저장소 `aiportal-front-admin` 의 `v0.1.x` 관례를 이식하는 것도 **금지**(별개
  저장소의 승인되지 않은 관례 이식).

---

## B. 실행 과제 — 게시판 상세 3형제의 조회 실패가 '빈 자리표시자 화면' 으로 감춰지는 문제 수정

- 과제: `NoticeDetail`/`LibraryDetail`/`RequestDetail` 의 `fetchBoardDetail` 이 `isSuccess` 검사 없이
  `res?.data?.body` 를 읽고 예외를 통째로 삼키는 문제 수정 (가치 3 / 위험 2 / 작업량 M)
- 왜: 세 화면 모두 조회가 실패하면 `notice/library = null` 이 되는데, 화면 바인딩 computed 가
  자리표시자 문자열로 대체되어(`NoticeDetail.vue:24-26` `'공지사항 제목'` / `'YYYY-MM-DD HH:MM:SS'` /
  `'공지'`, `LibraryDetail.vue:22-24` · `RequestDetail.vue:26-28` `'자료실 제목'` / `'YYYY-MM-DD HH:MM:SS'` /
  `'매뉴얼'`) **조회 실패가 "제목 없는 빈 게시글" 처럼 정상 렌더된다** — 사용자는 실패했다는 사실 자체를
  알 수 없고 새로고침할 이유도 얻지 못한다. 고치면 같은 저장소의 상세 조회 표준형
  (`SupportOcrDetail.vue:31-57` `getOcrs`)과 같은 안내를 받게 된다.
- 기대 계약은 **새로 발명하지 않는다.** `SupportOcrDetail.getOcrs` 가 이미 확정해 둔 것을 그대로 쓴다:
  - 업무 실패(`!isSuccess(res)`) → `openAlert(res?.data?.message || '<X> 상세 조회에 실패하였습니다.')`
    후 `return` (`SupportOcrDetail.vue:39-42`)
  - `catch` → `if (e == 'COM') return` 후 `openAlert('<X> 상세 조회에 실패하였습니다')`
    (`SupportOcrDetail.vue:55-58`) — `'COM'` 은 인터셉터(`interceptors.js:228-236`)가 이미 전역 Alert 을
    띄우고 reject 한 오류라 겹쳐 안내하면 안 된다.

### 수용 기준
1. 세 화면 각각에서 HTTP 200 + `code` 불일치(업무 실패) 응답이 오면 전역 Alert 이 **1회** 뜨고 그 문구는
   응답의 `message` 이며, `message` 가 없으면 화면별 기본 문구가 뜬다. 자리표시자 제목만 뜬 채 조용히
   끝나지 않는다.
2. 공통 처리되지 않은 예외(네트워크 오류 등)에서도 전역 Alert 이 **1회** 뜬다.
3. 인터셉터가 `'COM'` 으로 reject 하는 오류(예: `{ status: 204, data: { code: 'BZ01', message: '…' } }`)
   에서는 **인터셉터의 Alert 만** 뜨고 이 화면이 Alert 을 **추가로 띄우지 않는다**(중복 안내 금지).
4. 성공 경로는 그대로다 — `body` 로 제목·작성일·본문·첨부가 렌더되고 Alert 은 0회.
5. 실패 시 기존 동작(`notice/library = null`, `isLoading = false`)은 유지된다.
6. 테스트는 위 1~5 를 **실제 `interceptors.js` → 실제 `inf.board.detail.call` / `inf.library.detail.call`
   → 실제 컴포넌트 마운트**로 증명한다. 수정 전 Red 가 실제로 나는 것을 먼저 확인하고(업무 실패·예외
   케이스가 Alert 0회로 실패해야 한다), 수정 후 전부 통과해야 한다. 소스 문자열 검사·손으로 만든 대역
   컴포넌트는 증거로 쓰지 않는다.

### 건드릴 파일
- `src/views/Notice/NoticeDetail.vue:57-66` — `fetchBoardDetail`. `isSuccess` 를
  `import { toast } from '@/utils/common'`(`:10`) 에 추가하고 `useGlobalAlert`(`@/utils/globalAlert.js`)
  를 새로 import. 업무 실패 분기 + `catch` 의 `'COM'` 가드·Alert 추가. 기본 문구:
  `'공지사항 상세 조회에 실패하였습니다.'`
- `src/views/Library/LibraryDetail.vue:60-72` — 동일. 기본 문구: `'자료실 상세 조회에 실패하였습니다.'`
  업무 실패 시 `convertViewer()` 를 건너뛰어도 안전하다(정찰이 확인: `convertViewer:108-109` 는
  `!libraryContent.value` 면 즉시 return 하고 실패 시 `library.value = null` → content 는 `''`).
- `src/views/Request/RequestDetail.vue:64-76` — 동일(이 파일도 `inf.library.detail` 을 쓴다 — 그대로 둘 것).
  기본 문구: `'요청 상세 조회에 실패하였습니다.'`
- `tests/unit/boardDetailFailure.spec.js`(신규) — `tests/unit/sidemenuHistoryFailure.spec.js` 의 하네스를
  그대로 본뜬다: `vi.stubEnv('VITE_BACKEND_API_TARGET', 'http://localhost:9999')` 를 **동적 import 앞**에 두고
  (`sidemenuHistoryFailure.spec.js:21-27`), axios adapter 한 겹만 대역으로 바꾸고, `createMemoryHistory`
  라우터에 `/notice/detail/:boardId` 등 실제 경로를 얹어 `route.params.boardId` 가 채워지게 한다
  (`router/index.js:54, 81, 85`). 컴포넌트는 `onMounted` 에서 조회하므로 마운트 후 `settle()` 만 하면 된다.

### 검증 명령
```
npm ci                      # node_modules 가 비어 있으면 선행 필수(수 분)
npx vitest run tests/unit/boardDetailFailure.spec.js     # 수정 전 Red 확인 → 수정 후 통과
npm test                    # 전체. 기준선은 직전 회차 기록상 32파일 516테스트(정찰이 직접 실행하진 않음)
npm run build:dev           # 통과 확인 후 dist/ 삭제
```

### 위험과 피할 것
- `src/api/common/interceptors.js`, `src/api/interface.js`, `src/storage/*`, `src/router` 는 **건드리지 말 것**.
  이번 수정은 세 `.vue` 의 `fetchBoardDetail` 안에서 끝난다.
- **`toast` 로 바꾸지 말 것.** 세 파일이 이미 `toast` 를 import 하고 있어 손이 가기 쉽지만, 상세 조회
  실패의 확정된 형제 계약은 `SupportOcrDetail`/`SupportSttDetail` 의 `openAlert` 다. 목록 화면
  (`NoticeList.vue:76`, `LibraryList.vue:74`, `RequestList.vue:168`)은 `if (e == 'COM') return` 만 있고
  안내가 없으므로 **목록의 현 동작을 근거로 삼지 말 것**(그쪽은 이번 범위 밖이고 별도 판단이 필요하다).
- 전역 Alert 은 모듈 싱글턴이라 테스트 간 상태가 샌다 — 각 케이스 `beforeEach/afterEach` 에서 닫아 초기화할 것
  (`sidemenuHistoryFailure.spec.js` 가 하는 방식 그대로).
- 자리표시자 computed(`'공지사항 제목'` 등)를 지우거나 빈 상태 UI 를 새로 만드는 것은 **범위 밖**이다.
  승인되지 않은 UI 변경이 된다. 이번엔 "실패를 알린다" 까지만.
- 세 파일은 문자 단위로 거의 같다 — **한쪽만 고치지 말 것**. 과거 이 저장소 결함 대부분이 형제 경로
  비대칭이었다. 세 화면 모두에 같은 케이스를 걸어 검증한다.
- Red 가 실제로 나는지 먼저 확인할 것. 라우터 params 가 비면 `boardId` 가 `undefined` 라 요청이 다른
  모양으로 나가 조용히 통과할 수 있다 — 첫 케이스에서 `/board/{id}` 요청이 **실제로 나갔음**을 단정해 고정할 것.

### 차선 후보
`PopSimpleBot.vue:542,563` / `PopSimpleBotUpdate.vue:649` 의 `tryUpdateAppIcon`·`tryUpdateLog` 가
`catch (e) {}` 로 예외를 완전히 삼키는 문제 — 단, '아이콘 실패가 앱 저장 전체의 실패인가' 라는 기대 계약이
**미확인**이고 `simpleBotUpdateClose.spec.js` 가 현 동작('아이콘 실패는 성공 판정을 바꾸지 않는다')을
이미 고정하고 있다. 고른다면 계약을 "앱 저장은 성공, 아이콘 실패만 별도 토스트" 로 두고 그 기존 케이스부터
갱신할 것.
