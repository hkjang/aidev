# 과제서 — 2026-09-24-050915-aiportal-front-improve

이번 회차는 **A(우선 과제=릴리즈)** 와 **B(실제 코드 작업)** 두 개다.
A 는 저장소 안에 수행 수단이 없음을 이번 워크트리에서 다시 확인했으므로 **무변경**으로 두고,
구현자는 **B 를 수행**한다. (직전 5회차 모두 같은 구조로 '채택' 판정을 받았다.)

---

## A. 우선 과제(릴리즈) — 진입 조건 미충족, 이번 회차도 무변경 (21회째)

정찰이 이번 워크트리 HEAD `8f11ecb`(`git status --porcelain` 무출력)에서 직접 재확인한 것:

- `git tag -l | wc -l` = **0**, `git rev-parse --is-shallow-repository` = **false** → 태그가 fetch 에서 빠진 게 아니라 존재한 적이 없다. 따라서 태그 형식(v 접두사)·주석 태그 여부를 정할 전례가 없다.
- `package.json:3` **version = 0.0.0** (lock name `kcb_ai`). 버전이 적힌 파일은 이것(+lock)뿐.
- `.github` 디렉터리 **없음**(GitHub Actions 워크플로 0개), `CHANGELOG.md`·`VERSION`·`scripts/`·`Makefile` **모두 없음**.
- `.gitlab-ci.yml` 의 `CI_COMMIT_TAG` 참조 **0건**(`grep -c` = 0) — 배포는 `main`/`develop` 브랜치 푸시 트리거이고 태그를 게이트로 쓰지 않는다.

→ **구현자가 할 일: 아무것도 하지 않는다.** 재조사도 불필요(위 6줄이 이번 워크트리에서 나온 실측이다).
버전 파일 수정·태그 생성·CHANGELOG/릴리즈 노트 작성·릴리즈 커밋·원격 전송(push/gh release/npm publish)을 **일체 하지 말 것**.
`docs/RELEASE.md` 도 수정하지 말 것(그 문서 스스로 `:3`, `:83`, `:84` 에서 새 관례 제정과 판정 완화를 금지한다).
판정을 `skipped` 로 낮추지도 말 것(버전 파일 `package.json` 이 존재하므로 skipped 조건 미충족).

**사람 입력 4가지가 있어야 풀린다**(21회 연속 동일 교착, 저장소 안에 합법적 수단 없음):
① 첫 릴리즈 버전(0.0.1 / 0.1.0 / 1.0.0) ② 태그 형식(v 접두사)과 주석 태그 여부 ③ 릴리즈 노트 위치·양식·언어 ④ GitHub Release 사용 여부(현 원격 배포는 GitLab).
실패 지점 자체가 워크트리 밖(외부 aidev `bin/run.sh` → `release.json` → `bin/gate.py`)이라 이 저장소에서 고칠 워크플로 파일이 없다.

---

## B. 이번 회차의 실제 과제

- **과제: 게시판 목록 3형제(NoticeList/LibraryList/RequestList)의 조회 실패가 '없습니다' 라는 사실 진술로 감춰지는 문제 수정 (가치 3 / 위험 2 / 작업량 M)**

- **왜**: 세 목록 화면의 `getBoardList` 는 `isSuccess(res)` 검사가 없어 HTTP 200 + 실패 코드 응답을 그대로 통과시키고(인터셉터 `interceptors.js:228-236` 는 이 경우를 'COM' 으로 던지지 않는다), `catch` 는 `if(e == 'COM') return` 뒤에 목록을 `[]` + totalCount 0 으로만 만든다. 그 결과 조회가 **실패**해도 화면에는 `Nodata` 의 "공지사항이 없습니다"(`NoticeList.vue:133`) / "등록된 자료가 없습니다"(`LibraryList.vue:131`) / "조회 내용이 없습니다"(`RequestList.vue:261`) 라는 **사실 진술**이 뜬다 — 사용자는 실패를 인지하지 못하고 재시도하지 않는다. 고치면 직전 회차가 상세 3형제에 세운 것과 짝이 되는 '목록 조회 실패도 알린다' 계약이 완성된다.
  기대 계약은 **새로 발명하지 않는다**: 같은 저장소의 목록 조회 표준형 `Sidemenu.fetchAppList`(`src/components/layout/Sidemenu.vue:207-211` 업무 실패 `toast(res?.data?.message || '…조회에 실패하였습니다.')` / `:231-234` `if (e !== 'COM') toast('…조회 중 오류가 발생하였습니다.')`)와 `Sidemenu.getHistoryList`(`:409-413`, `:419-423`, 직전 회차에 같은 형태로 고쳐 머지됨)를 그대로 따른다. 상세는 `openAlert`, **목록은 `toast`** 가 이 저장소의 확정된 구분이다.

- **수용 기준**
  1. 세 화면 각각에서, HTTP 200 + `code: '500'`(또는 `success:false`) 응답이 오면 `toast(res?.data?.message || <화면별 기본 문구>)` 가 **1회** 뜬다. 응답에 `message` 가 없으면 기본 문구가 뜬다.
  2. 세 화면 각각에서, 통신 오류(네트워크 예외 등 `'COM'` 이 아닌 예외)가 나면 `toast(<화면별 오류 문구>)` 가 **1회** 뜬다.
  3. 인터셉터가 이미 전역 Alert 을 띄우고 `'COM'` 으로 던진 경우(검증된 대역 응답 `{ status: 204, data: { code: 'BZ01', message: '…' } }`)에는 **토스트가 뜨지 않는다**(Alert 과 중복 안내 금지).
  4. 성공 응답에서는 토스트가 **0회**이고 목록 행이 정상 렌더된다(수정 범위가 성공 경로를 건드리지 않음을 고정).
  5. **업무 실패 시 목록은 비고 totalCount 는 0 이 된다** — 이는 현행 동작의 보존이다(현재도 `body` 가 배열이 아니라 `[]` 가 되고 `setTotalCount(undefined)` → `page.js:21` 이 `Number(undefined||0)`=0 으로 만든다). 2페이지를 성공 조회한 뒤 3페이지가 업무 실패했을 때 **이전 페이지 행이 남지 않는다**는 케이스를 최소 1건 넣어, 새로 추가하는 초기화 줄이 무효 변경이 아님을 증명할 것.
  6. 테스트는 **HTTP 전송(axios adapter) 한 겹만 대역**으로 바꾸고 실제 `interceptors.js` → 실제 `inf.board.list.call`/`inf.library.list.call` → 실제 컴포넌트 마운트를 통과시킨다. 첫 케이스에서 `/board`·`/library` 요청이 **실제로 나갔음을 단정**해, 폼/가드에 막혀 조용히 통과하는 형태가 아님을 고정할 것.
  7. 변이 검증: ①업무 실패 토스트 줄 제거 → 해당 케이스만 실패 ②catch 토스트 줄 제거 → 해당 케이스만 실패 ③`if (e == 'COM') return` 가드 제거(또는 토스트를 가드 앞으로 이동) → COM 중복 안내 케이스만 실패. 각각 확인 후 되돌릴 것.

- **건드릴 파일**
  - `src/views/Notice/NoticeList.vue:46` `getBoardList` — `const res = await inf.board.list.call(...)` 바로 뒤에 `if (!isSuccess(res)) { toast(res?.data?.message || '공지사항 목록 조회에 실패하였습니다.'); noticeBody.value = []; setTotalCount(0); return }`. `catch` 는 기존 `if(e == 'COM') return` **그대로 두고** 그 다음 줄에 `toast('공지사항 목록 조회 중 오류가 발생하였습니다.')` 추가(가드보다 **뒤**여야 한다). import 에 `isSuccess`, `toast` 추가 — `@/utils/common.js` 에서 온다(`Sidemenu.vue:17` 이 그 형태).
  - `src/views/Library/LibraryList.vue:45` `getBoardList` — 같은 형태. 문구는 '자료실 목록 조회에 실패하였습니다.' / '자료실 목록 조회 중 오류가 발생하였습니다.'
  - `src/views/Request/RequestList.vue:132` `getBoardList` — 같은 형태이나 상태 이름이 다르다: `AllBody.value = []`, `totalCount.value = 0`. 문구는 '요청 목록 조회에 실패하였습니다.' / '요청 목록 조회 중 오류가 발생하였습니다.' 기존 `console.error('[admin board list error]', e)` 는 그대로 둘 것.
  - `tests/unit/boardListFailure.spec.js` (신규) — `tests/unit/boardDetailFailure.spec.js` 를 그대로 본떠서 쓸 것. 그 파일의 `VIEWS` 배열 + 메모리 라우터 + `vi.stubEnv('VITE_BACKEND_API_TARGET', …)` 패턴이 이미 검증돼 있다.

- **검증 명령** (이 저장소에서 실제로 도는 것)
  - `npm ci` — **node_modules 가 비어 있으면 선행 필수, 수 분 소요**
  - `npx vitest run tests/unit/boardListFailure.spec.js` — 작성 중 빠른 확인용
  - `npm test` — 전체. **기준선은 33 파일 / 534 테스트**(직전 회차 결과). 신규 스펙만큼 늘고 기존은 하나도 줄지 않아야 한다.
  - `npm run build:dev` — 통과 후 생성된 `dist/` 는 **삭제**할 것. (`npm run build`·`npm run lint` 는 이 저장소에 **없다**.)

- **위험과 피할 것**
  - **라우터 params 함정**: `LibraryList` 는 `route.params.type`(`:47` `libType`)과 `route.query.title` 을 읽는다 — 메모리 라우터를 `/library/list/:type` 로 만들고 `/library/list/001?title=자료실` 로 push 해야 `groupCode` 가 채워진다. `RequestList` 는 `readUser()`(`:30`)로 `creatorId` 를 만든다 — `writeUser(...)` 로 사용자 상태를 먼저 심고, `afterEach` 에서 `clearUser()` 할 것(`boardDetailFailure.spec.js` 가 이미 그렇게 한다). `NoticeList` 는 params 의존이 없다.
  - **`RequestList` 만의 부작용**: 성공 경로에 `isFirst`/`searchCount`(`:141-144`)와 `excludedCount`(`:149-152`) 갱신이 있다. 업무 실패 시 early return 하면 이것들이 갱신되지 않는데, 이는 '실패했으니 카운트를 건드리지 않는다' 로 옳다 — 다만 **관측 가능한 차이가 있는지 한 번 확인하고**, 있다면 그 동작을 케이스로 고정할 것. 없다면 그냥 두고 과제서에 적힌 대로 진행.
  - **무효 변경 금지**(직전 2회차에서 실제로 걸린 자리): 관측 가능한 동작 변화가 0 인 줄은 넣지 말 것. 수용 기준 5 가 그 판정 방법이다 — 초기화 줄을 넣기 전에 **빼고 돌려서 실패하는 케이스가 있는지** 확인하고, 없으면 넣지 말고 그 사실을 보고할 것.
  - **`'COM'` 가드의 위치**: 토스트를 `if(e == 'COM') return` **앞**에 두면 전역 Alert 과 겹친다. 이 저장소가 반복해서 고쳐 온 결함이 정확히 이 형태다.
  - **건드리지 말 것**: `src/api/common/interceptors.js`, `src/api/interface.js`, `src/utils/page.js`(`usePaging` 은 세 화면 밖에서도 쓰인다), `src/router/index.js`, 상세 3형제와 `boardDetailFailure.spec.js`(직전 회차 계약이다 — 통과 상태를 유지할 것).
  - **테스트 간 상태 누수**: 전역 토스트/알림은 모듈 싱글턴이다. 각 케이스에서 토스트 목록·유저 상태를 정리할 것(`restoreMocks: true` 는 켜져 있지만 싱글턴 상태는 안 지워진다).
  - `RequestList` 가 `inf.library.list` 를 쓰는 것은 **현행 그대로 둘 것**(요청 게시판 전용 엔드포인트가 `interface.js` 에 없다 — 백엔드가 별개 저장소라 이 저장소만으로 확정 불가). 테스트도 `/library` 로 나가는 현 동작을 고정한다.

- **차선 후보**: `PopSimpleBot.vue:542,563` / `PopSimpleBotUpdate.vue:649` 의 `tryUpdateAppIcon`·`tryUpdateLog` 가 `catch(e){}` 로 예외를 완전히 삼키는 문제 — '앱 저장은 성공, 아이콘 실패만 별도 토스트' 로 계약을 정하고 현 동작을 고정하고 있는 `simpleBotUpdateClose.spec.js` 의 해당 케이스부터 갱신하는 형태. (기대 계약이 미확인이라 1순위보다 근거가 약하다.)
