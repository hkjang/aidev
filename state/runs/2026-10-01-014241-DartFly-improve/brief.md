# 과제서 (정찰, 2026-10-01)

- 과제: 관리자 변경 이력 화면 **'저장된 결과' 탭**의 100건 벽과 무반응 검색창 — 실행 감사 탭·`/saved` 와 같은 페이지 이동·페이지내 검색 표시 붙이기 (가치 3 / 위험 2 / 작업량 M)

- 왜: `admin-history.js:71 loadSaved()` 는 `api('/api/v1/query/saved?limit=100')` 로 **한 번만** 부르고 카운트에는 서버가 준 `page.total` 을 그대로 씁니다. 저장본이 350건이면 화면은 `350건` 이라고 말하면서 100행만 보여 주고 101번째부터는 관리자가 닿을 방법이 아예 없습니다(이동 바 없음). 게다가 탭 전환 시 `searchEl.placeholder` 가 `'SQL 검색...'` 으로 남고(admin-history.js:226 의 삼항은 schema/data 만 분기) `searchEl` 의 input 리스너가 `load()` → `loadSaved()` 를 부르지만 `loadSaved` 는 `searchEl.value` 를 **읽지 않습니다** — 관리자가 검색어를 쳐도 목록이 그대로여서 "검색이 고장났다" 로 보입니다. 같은 저장소의 `/saved`(bad8c9a)와 실행 감사 탭(68a51eb)이 이미 이 문제를 풀어 둔 관례가 있으니 그것을 그대로 옮기면 탭 사이 동작이 일치합니다.

- 수용 기준:
  1) 저장본이 페이지 크기(100)를 넘을 때 '저장된 결과' 탭 목록 아래에 `← 이전` / `다음 →` / `페이지 n / m` 이동 바가 나오고, `다음 →` 를 누르면 `/api/v1/query/saved?limit=100&offset=100` 을 요청해 101번째 이후가 보인다. 100건 이하·첫 페이지면 이동 바는 나오지 않는다(또는 두 단추가 모두 disabled).
  2) 카운트가 `{from}-{to} / 총 {total}건` 형식이 되어 "보이는 것"과 "전체"가 구별되고, 검색어가 있을 때는 `· 이 페이지에서 N건` 이 덧붙어 검색이 현재 페이지 안에서만 돈다는 것이 드러난다. 검색어를 입력하면 실제로 목록 행이 줄어든다(`sql`·`db_name`·`user_name` 중 하나에 대소문자 무시 포함). 빈 결과 문구는 검색 중이면 `이 페이지에는 없습니다.`, 그 밖이면 기존 안내.
  3) 마지막 페이지의 저장본을 모두 지워 그 페이지가 비면(삭제 또는 보존 정리) 빈 화면에 갇히지 않는다 — 서버가 준 `total` 로 마지막 페이지를 계산해 **한 번만** 다시 부르고, 카운트 범위가 `101-100` 처럼 뒤집히지 않는다(빈 페이지면 `0-0 / 총 N건`).
  4) 테스트가 증명할 것: ① `다음 →` 가 `offset=100` 을 요청하고 두 번째 페이지 행이 그려짐 ② 늦게 도착한 이전 페이지 응답이 최신 페이지를 덮어쓰지 못함(이동 단추가 `load()` 를 거쳐 `listRequest` 순번을 올림) ③ 여러 페이지가 한꺼번에 사라진 뒤 `다음 →` 를 눌러도 total 기준 마지막 페이지로 되돌아와 행이 보임 ④ 검색어 입력 시 행 수가 줄고 카운트에 `이 페이지에서` 가 붙음 ⑤ 기존 11개 테스트(탭 전환·실행 감사 pager·상세 다이얼로그)가 그대로 green.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/webui/js/admin-history.js`
    - 최상단 상태: `let execOffset = 0; const EXEC_PAGE = 100;` 옆에 `let savedOffset = 0;`(페이지 크기는 `EXEC_PAGE` 를 공용 `PAGE` 로 쓰거나 `SAVED_PAGE = 100` 을 따로 두되 **둘을 같은 값으로**; 서버 기본값도 100 — `internal/resultsave/service.go:97`).
    - `loadSaved(seq)`(:71): `fetchSaved(at)` 헬퍼(`new URLSearchParams({limit, offset})`)로 분리하고, 실행 감사와 같은 순서로 ① 빈 페이지 + `savedOffset > 0` 이면 `Math.max(0, Math.ceil(total/PAGE)-1)*PAGE` 로 한 번 재요청 ② `seq !== listRequest` 가드는 **매 await 뒤마다** ③ `page.offset` 이 숫자면 `savedOffset` 에 반영 ④ 카운트/빈 문구/검색 필터를 `saved.js:44-56 renderList()` 문구 그대로 맞춤 ⑤ 표 아래에 이동 바 append.
    - 이동 바: `execPager(count, total)`(:146)을 `pager(offset, count, total, onGo)` 형태로 일반화하거나 `savedPager()` 를 같은 스타일 문자열(`display:flex;gap:8px;...`)로 하나 더 두기 — **어느 쪽이든 클릭 핸들러는 `load()` 를 불러야** 합니다(직접 `loadSaved()` 를 부르면 순번이 안 올라가 68a51eb 가 고친 경합이 되살아납니다).
    - 탭 전환 핸들러(:220-228): `execOffset = 0` 옆에 `savedOffset = 0`, placeholder 삼항에 `active === 'saved' ? 'SQL·DB·저장자 검색(이 페이지)...'` 분기 추가.
    - 검색 input 리스너(:230): `execOffset = 0` 과 함께 `savedOffset = 0`(검색은 페이지 안 필터지만, 페이지를 첫 장으로 되돌리는 쪽이 실행 감사와 일치).
    - 삭제 핸들러(:83 의 `del` 리스너) 는 이미 `await load()` 를 부르므로 그대로 두면 기준 3 의 step-back 경로를 탄다.
  - `test/js/admin-history.test.mjs`: 기존 하네스 그대로 사용. `page(first)` 의 `requests` 큐·`pager()`/`pagerButton(label)`/`pagerInfo()`/`offsetOf()` 헬퍼(:84-90)가 `#list` 안의 이동 바를 이미 찾아 주므로 저장 탭에도 바로 쓸 수 있습니다. 응답 만들기는 `savedRow(id)`(:43) 를 재사용하고 `savedPage(from,count,total,offset)` 를 `execPage`(:39) 꼴로 하나 추가. 검색 테스트는 `settle()`(디바운스 250ms 보다 긴 300ms)을 쓸 것.

- 검증 명령:
  - `node test/js/admin-history.test.mjs` (구현 전 새 테스트가 red 인 것을 먼저 확인)
  - `go test -v -run TestBrowserModuleTests ./internal/webui/...` — 출력에 `admin-history.test.mjs` 가 **Skip 없이 PASS** 로 찍히는지 확인(Node 없으면 조용히 Skip 되는 함정)
  - `go test -count=1 -race ./...` · `go vet ./...` · `gofmt -l .`(무출력) · `go build ./cmd/dartfly`
  - 실제 바이너리 확인(권장, 프로덕션 배선 증거): 실제 MariaDB + `go build` 한 바이너리를 띄워 저장본을 100건 넘게 만들고 관리자로 `/admin/history` → '저장된 결과' 탭에서 `다음 →` 클릭(네트워크에 `offset=100`), 마지막 페이지 전부 삭제 후 갇히지 않는 것, 검색창 입력이 실제로 행을 줄이는 것, CSP 위반 콘솔 0건을 확인. 스모크 전체는 `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh`(수 분).
  - **미확인**: 이 정찰 세션은 샌드박스 승인 요구로 `node`/`go test` 를 한 번도 실행하지 못했습니다(baseline green 여부 미확인). 구현자는 먼저 손대기 전 상태에서 위 명령을 돌려 기존 11개가 green 인 것을 확인하고 시작하세요.

- 위험과 피할 것:
  - **서버·SQL 은 건드리지 마세요.** `/api/v1/query/saved` 는 이미 `limit`/`offset`/`total` 을 주고(`internal/resultsave/service.go:96-107` 에서 `limit<=0||>500 → 100`, `offset<0 → 0` 로 클램프) 권한 조건은 `ListSavedResults(userID, admin, ...)` 안에 있습니다. 서버 쪽 `q` 검색을 붙이려 들면 권한 조건(admin OR user_seq=?)을 손대게 되어 권한 누출 위험 — 이번 범위 밖입니다(보류 아이디어의 L 항목).
  - `empty()` 는 `listEl` 을 통째로 덮으므로 **빈 페이지에는 이동 바가 남지 않습니다.** `/saved` 처럼 "한 페이지만 물러나기" 로는 여러 페이지가 한꺼번에 사라진 경우 여전히 갇힙니다(2026-09-28 회차가 실행 감사에서 겪은 그 차이) — 반드시 `total` 로 마지막 페이지를 계산하세요.
  - CSP `style-src 'self'` 가 **HTML 인라인 style 속성**을 막습니다. HTML(`internal/webui/pages/admin-history.html`)을 고쳐 pager 요소를 심지 말고, 지금처럼 JS 에서 `style.cssText` 로 주세요(`execPager` 가 하는 방식). 반대로 `hidden` 만 끄면 인라인 `display:flex` 가 이겨 숨김이 안 먹습니다(`saved.js:73-75` 주석 참고).
  - 실행 감사 탭의 동작·문구·테스트는 회귀 가드입니다 — `execPager` 를 일반화할 경우 실행 감사 쪽 출력(`페이지 n / m` 위치, 단추 순서 `info, prev, next`)이 한 글자도 바뀌지 않아야 기존 테스트가 통과합니다.
  - 보호 경로 무접촉: `internal/auth`·세션·SSO, `internal/store/mariadb/migrations`, `.github/workflows`, `deploy/build-release.sh`.
  - 파일 수를 늘리지 마세요(프로덕션 1개면 충분). '스키마·데이터 탭 200건 상한 표시' 를 같이 하고 싶어도 이번 회차에는 넣지 말 것 — 서버가 total 을 주지 않아 성격이 다른 변경입니다.

- 차선 후보: **스키마·데이터 탭이 서버 상한(200건)에서 잘린 것을 화면에 드러내기 (가치 2 / 위험 1 / 작업량 S)** — `loadSchema`/`loadData`(admin-history.js:161,189)가 `countEl.textContent = ${items.length}건` 으로 서버가 잘라 보낸 배열 길이를 정확한 총건수처럼 표시합니다(서버 상한은 `internal/history/service.go:79,93` 의 `limit<=0||>1000 → 200`, 핸들러 기본값도 200 — `internal/server/history.go:29,95`). 서버가 total 을 주지 않으므로 `items.length === 200` 일 때 `200건(표시 상한) · 검색으로 좁혀 보세요` 처럼 표시하는 것까지가 이번 범위이고, 서버에 total 을 추가하는 것은 범위 밖입니다. 프로덕션 파일 1개.
