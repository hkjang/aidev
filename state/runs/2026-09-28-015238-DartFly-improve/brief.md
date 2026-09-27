- 과제: 관리자 감사 화면(실행 감사 탭)의 늦은 응답 덮어쓰기와 빈 페이지 갇힘 수정 (가치 3 / 위험 2 / 작업량 M)
- 왜: `internal/webui/js/admin-history.js` 의 실행 감사 탭은 2026-09-20·09-23 회차가 `saved.js` 에서 이미 고친 두 결함을 그대로 갖고 있습니다 — (1) `load()`/`loadExecutions()` 에 요청 순번이 없어 탭을 바꾸거나 검색어를 고친 뒤 늦게 도착한 응답이 새 선택을 덮어쓰고, (2) `loadExecutions` 가 `items` 가 비면 `empty()` 로 early return 해 pager 를 붙이지 않으므로 보존 정책·삭제로 `total` 이 줄어든 뒤 빈 페이지에 닿으면 '← 이전' 이 없어 탭을 바꾸는 수밖에 없습니다. 고치면 관리자가 감사 기록을 볼 때 엉뚱한 탭 내용이 섞이거나 빈 화면에 갇히지 않습니다.
- 수용 기준:
  1) 실행 감사 응답이 in-flight 인 상태에서 다른 탭(스키마/데이터/저장결과)으로 바꾼 뒤 그 응답이 도착해도 화면이 실행 감사 표로 바뀌지 않는다. 반대로 늦게 온 이전 페이지 응답이 새 페이지 결과를 덮어쓰지 않는다.
  2) `offset>0` 인데 서버가 빈 `items` 를 돌려주면 `saved.js:34-37` 과 같이 한 페이지 물러나 다시 부른다(`offset` 감소 → 재요청). 물러날 곳이 없을 때(`offset===0`)만 기존 안내 문구를 보인다.
  3) 카운트 문구가 빈 페이지에서 `101-100 / 총 50건` 같은 역전 범위를 내지 않는다(현재 `admin-history.js:106-107` 은 그렇게 나옵니다).
  4) 새 `test/js/admin-history.test.mjs` 가 구현 전 red → 구현 후 green 이고, 탭 전환·검색·기존 pager 동작 회귀가 없다.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/webui/js/admin-history.js`
    - 모듈 스코프에 `let listRequest = 0;` 추가. `load()`(59행) 에서 `const seq = ++listRequest;` 를 잡아 각 `loadX(seq)` 에 넘기고, 각 로더가 `await api(...)` 직후 `if (seq !== listRequest) return;` 로 빠져나오게 합니다. `load()` 의 `catch` 도 같은 가드를 걸어야 늦은 실패가 새 화면에 `empty(cause.message)` 를 덮어쓰지 않습니다.
    - `loadExecutions`(101행): 요청을 `fetchExecutions(at)` 헬퍼로 빼고, `!items.length && execOffset > 0` 이면 `execOffset = Math.max(0, execOffset - EXEC_PAGE)` 로 한 번 물러나 재요청(무한 루프 방지를 위해 한 번만). 응답의 `page.offset` 이 숫자면 `execOffset = page.offset` 으로 맞춥니다(saved.js:40 과 같은 관례).
    - `empty()` 로 early return 하기 전에도 pager 가 남도록 하거나(2)의 후퇴로 그 상태 자체를 없애거나 — 후자가 `saved.js` 에서 검증된 쪽입니다. 다만 `empty()` 가 `listEl.textContent` 를 쓰고 `table()` 이 `listEl.replaceChildren` 을 쓰므로, pager 를 남기는 쪽을 고른다면 `listEl.append(execPager(page))` 를 early return **뒤가 아니라 앞**으로 옮기고 `empty()` 가 pager 를 지우지 않게 구조를 바꿔야 합니다.
    - 카운트 문구(106-107행): `items.length === 0` 이면 `0건` 또는 `총 {total}건` 으로 떨어뜨려 역전 범위를 막습니다.
    - pager 클릭 핸들러(126·128행)가 `loadExecutions()` 를 직접 부릅니다 — 순번을 도입하면 여기서도 `load()` 를 거치거나 `++listRequest` 를 갱신해야 클릭끼리도 보호됩니다. 이 두 줄을 빠뜨리면 수용 기준 1이 절반만 충족됩니다.
  - `test/js/admin-history.test.mjs` (신규) — `test/js/saved.test.mjs` 의 vm+가짜 DOM 패턴을 그대로 복사해 씁니다(파일 등록 불필요: `internal/webui/jstest_test.go:TestBrowserModuleTests` 가 `test/js/*.test.mjs` 를 glob 으로 잡습니다).
- 테스트 하네스 주의 (saved.test.mjs 를 복사할 때 admin-history.js 가 더 요구하는 것):
  - `document.querySelector` 스텁은 `#list` `#search` `#count` `#tabs` `#detail-dialog` `#detail-title` `#detail-body` `#close-detail` 를 모두 돌려줘야 합니다(모듈 최상위 3·4·5·6·184행에서 바로 잡고 `#close-detail` 에는 즉시 `addEventListener`).
  - 가짜 `Element` 에 `showModal()`/`close()` 를 더하세요 — `saved.test.mjs` 의 `Element` 에는 없고 `detailDialog.showModal()`(89·115·155·181행)이 부릅니다.
  - `initLayout` 은 `Promise.resolve({ user: { role: 'SYSTEM_ADMIN' } })` 를 돌려줘야 `load()` 까지 갑니다(199-201행이 role 을 검사).
  - 컨텍스트에 `sessionStorage`(39-42행)·`window`(43행 `location.href`)·`setTimeout`/`clearTimeout`(197행 debounce, **250ms** — saved.js 의 200ms 가 아님)·`URLSearchParams`·`fmtDate`·`api` 가 필요합니다. import 는 `saved.test.mjs` 와 같이 `.replace(/^import .*;\n/gm, '')` 로 지우면 됩니다(1행이 한 줄짜리 import 라 안전).
  - 탭 전환은 `tabsEl` 의 `click` 리스너가 `event.target.closest('button[data-tab]')` 를 쓰므로, 가짜 이벤트로 `{ target: { closest: () => fakeButton } }` 을 넘기고 `tabsEl.querySelectorAll` 이 배열을 돌려주게 하세요.
  - `api` 는 `saved.test.mjs` 처럼 resolve/reject 를 큐에 쌓아 두고 **순서를 뒤집어** resolve 하는 방식으로 경합을 재현하세요 — 이 뒤집기가 결함 1의 red 증거입니다.
- 검증 명령:
  - `node test/js/admin-history.test.mjs` (구현 전 red 확인 → 구현 후 green)
  - `go test -v -run TestBrowserModuleTests ./internal/webui/...` — 출력에 `admin-history.test.mjs` 가 **Skip 아닌 PASS** 로 찍히는지 눈으로 확인(Node 없으면 조용히 Skip 합니다)
  - `go test -race ./...` · `go vet ./...` · `gofmt -l .`(무출력) · `go build ./cmd/dartfly`
  - 화면 동작이 바뀌므로 가능하면 `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` 까지. 다만 `test/smoke/run.sh:34,100` 의 `pkill -f "$BIN"` 이 호출한 셸 자신에 걸려 exit 144 로 죽은 전례가 있으니(2026-09-26 회차) 런처를 별도 파일로 빼서 부르세요.
  - **미확인**: 이 정찰 세션은 샌드박스 권한 때문에 `node`/`go test` 를 한 번도 돌리지 못했습니다. baseline green 은 확인되지 않았으니 구현자가 먼저 기존 JS 회귀를 한 번 돌려 출발점을 잡으세요.
- 위험과 피할 것:
  - 서버는 건드리지 마세요. `internal/history/service.go:100-111 ListExecutionAudit` 가 이미 `limit`·`offset` 을 클램프하고 `ExecutionPage{Items,Total,Limit,Offset}`(service.go:70-72) 을 돌려줍니다 — 클라이언트만 고치면 됩니다.
  - 같은 파일의 `loadSaved`(68행)·`loadSchema`(135행)·`loadData`(162행) 에는 pager 가 없고 서버가 배열만 돌려줍니다(`/history/schema`·`/history/data` 는 `writeSuccess(w, r, items)`). 이번 회차에 pager 를 새로 붙이지 마세요 — 범위가 커집니다. 순번 가드만 공통으로 적용합니다.
  - CSP 함정: `style-src 'self'` 는 **HTML 인라인 style 속성**을 막습니다. JS 의 `style.cssText`(124행이 이미 씀)는 괜찮습니다. 새 요소에 스타일이 필요하면 JS 로 주거나 `u-fs12` 같은 기존 유틸 클래스를 쓰세요(saved.js:78 참고).
  - 숨김 함정: 인라인 `display:flex` 가 `hidden` 속성을 이깁니다 — 이동 바를 숨길 일이 생기면 `saved.js:74-77` 처럼 `hidden` 과 `style` 을 함께 끄세요.
  - 보호 경로(auth·session·SSO·migrations·.github/workflows)는 이번 과제와 무관하니 건드리지 마세요.
- 차선 후보: 401 리다이렉트 분기를 `layout.js api()` 로 일원화 — `admin-databases.js:20`·`admin-migration.js:14`·`app.js:343`·`mcp.js:108`·`resources.js:37,45` 에 같은 분기가 복사돼 있음 (가치 2 / 위험 1 / 작업량 S). 동작 변화 없는 정리라 가치는 낮습니다.
