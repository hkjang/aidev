# 정찰 과제서 (2026-09-23, main@d1d3682 기준)

- 과제: 저장 결과 목록의 200건 벽 제거 — 서버가 이미 주는 total/offset 으로 페이지 이동을 붙이고, 검색 범위를 정직하게 표시 (가치 3 / 위험 2 / 작업량 M)

- 왜: `/saved` 화면은 `internal/webui/js/saved.js:23` 에서 `/api/v1/query/saved?limit=200` 만 한 번 부르고 끝나서, 저장 결과가 200건을 넘는 설치에서는 201번째부터가 화면에 존재하지 않습니다 — 게다가 `#saved-count` 는 "불러온 200건 중 필터된 수" 만 보여 주므로 사용자는 그게 전부라고 믿게 되고, 검색창은 그 200건만 훑기 때문에 예전 SQL 을 찾으면 "없습니다" 가 나옵니다(서버에는 있는데도). 서버 쪽 `Service.List`(internal/resultsave/service.go:85)·`listSavedResultsHandler`(internal/server/resultsave.go:56)는 이미 `limit`(최대 500)·`offset` 을 받고 `total` 을 돌려주므로, 화면만 고치면 안 보이던 데이터가 보이고 "몇 건 중 몇 건" 이 사실대로 나옵니다.

- 수용 기준:
  1) 저장 결과가 페이지 크기를 넘는 설치에서 `/saved` 에 `← 이전 / 다음 →` 이동이 나오고, 다음 페이지를 누르면 서버에 `offset` 이 올라간 요청이 가고 그 페이지의 항목이 목록에 표시된다. 첫 페이지에서 `이전` 은 비활성, 마지막 페이지에서 `다음` 은 비활성이다.
  2) `#saved-count` 가 `admin-history.js` 와 같은 형식(`{from}-{to} / 총 {total}건`)으로 **서버가 준 total** 을 보여 준다. 저장 결과가 0건이면 `0-0 / 총 0건` 류로 깨지지 않는다.
  3) 검색창의 필터가 "지금 페이지 안에서만" 이라는 것이 화면에 드러난다(예: 검색어가 있을 때 카운트에 `이 페이지에서 N건` 을 덧붙이거나, 결과 없음 문구를 `이 페이지에는 없습니다` 로). 서버 검색을 새로 만들지는 않는다.
  4) 삭제 후에도 보던 페이지가 유지된다: `#saved-delete` 처리에서 현재 `offset` 으로 다시 불러오고, 그 페이지가 비었는데 `offset > 0` 이면 한 페이지 앞으로 물러난 뒤 다시 부른다(마지막 항목을 지우고 빈 화면에 갇히지 않는다).
  5) 테스트가 증명할 것: (a) 다음/이전 클릭이 실제로 `offset` 이 바뀐 URL 로 요청을 보내고 목록이 갈린다, (b) 카운트 문구가 필터된 개수가 아니라 서버 total 을 쓴다, (c) 마지막 페이지의 유일한 항목을 삭제하면 이전 페이지로 물러나 다시 부른다, (d) 기존 상세 경합·삭제 경합 회귀(saved.test.mjs 의 현재 11개)가 그대로 통과한다.

- 건드릴 파일:
  - `internal/webui/js/saved.js` — `loadList()`(현재 23행, `limit=200` 고정)를 `PAGE` 상수 + 모듈 스코프 `offset` 으로 바꾸고 `URLSearchParams` 로 `limit`/`offset` 전달; `renderList()`(29행) 의 카운트 문구; 삭제 핸들러(109~129행) 의 `await loadList()` 자리에 페이지 보정; 목록 아래에 pager 바 추가.
  - 참고(복사할 본): `internal/webui/js/admin-history.js:101 loadExecutions` / `:120 execPager` — 이 저장소의 pager 관례(인라인 style 바, `← 이전`/`다음 →`, `페이지 n / m`, `{from}-{to} / 총 {total}건`). `admin-dataview.js:67` 도 같은 모양. **새 CSS 클래스를 만들지 말고 이 관례를 그대로 쓸 것.**
  - `internal/webui/pages/saved.html` — 필요하면 pager 를 담을 요소 하나만(`#saved-pager`). admin-history 는 요소 없이 JS 로 append 하므로 HTML 무변경도 가능; 둘 중 하나로 통일.
  - `test/js/saved.test.mjs` — 기존 하네스(`page()` 가 production 모듈을 vm 으로 실행)에 케이스 추가. 초기 응답 stub 이 `{items:[...]}` 만 주므로 `total`/`offset`/`limit` 를 포함하도록 손봐야 한다(기존 11개가 깨지지 않게).
  - 서버는 손대지 않는다: `limit` 최대 500·기본 100 은 그대로 두고 화면에서 200 이하를 보낸다.

- 검증 명령:
  - `cd /home/hkjang/.cache/auto-improve-wt/DartFly`
  - `node --test test/js/` (또는 개별 `node test/js/saved.test.mjs` — `internal/webui/jstest_test.go:34` 가 각 .mjs 를 `node <file>` 로 돌린다)
  - `go test ./internal/webui/` (node 없으면 Skip 되므로 node 로도 직접 돌릴 것)
  - `go test -race ./...`, `go vet ./...`, `gofmt -l .`
  - 화면 변경이므로 `DF_SMOKE_REQUIRE_BROWSER=1 bash test/smoke/run.sh` (실제 바이너리+MariaDB+Chromium, 수 분). **정찰에서는 미실행 — 구현자가 반드시 돌릴 것.**
  - 실제 증거(권장, 과거 회차가 채택된 방식): 스모크로 띄운 실제 바이너리에 로그인해 저장 API 로 페이지 크기+1건 이상을 만들고 `/saved` 에서 다음/이전 클릭과 마지막 항목 삭제를 Chromium 으로 확인. vm 대역만으로는 배선 증거가 되지 않는다.

- 위험과 피할 것:
  - **서버 API·저장소 SQL 을 넓히지 말 것.** 서버 검색(`q`)을 새로 만드는 쪽으로 번지면 `internal/store/mariadb/resultsave_repository.go` 의 `savedSelect`·권한 조건(`admin OR user_seq=?`)을 건드리게 되어 권한 누출 위험이 생긴다. 이번 과제는 화면 전용이다.
  - 직전 회차들이 고친 **상세 경합 로직을 되돌리지 말 것**: `detailRequest` 순번(56~76행), 삭제 시 `++detailRequest`·`selectedID` 처리(117~123행), `deleting` 가드. 페이지 이동을 넣으면서 `clearDetail()` 호출 순서를 바꾸면 5cd5ca0 회귀가 되살아난다 — 페이지를 넘겨도 선택 중이던 상세는 다른 페이지 항목이므로, `selectedID` 강조가 현재 페이지에 없을 때 상세를 지울지 남길지 정하고 테스트로 못 박을 것(권장: 상세는 그대로 두고 강조만 사라짐).
  - `tsvCell` 을 쓰는 TSV 복사(97~102행)는 e6bd782 에서 확정된 계약이다. 건드리지 말 것.
  - 보호 경로(auth/session/SSO, migrations, .github/workflows)는 이번 과제와 무관하다 — 열지 말 것.
  - 기존 테스트 하네스의 `Element.querySelectorAll()` 은 `' active'` 가 든 자식만 돌려주는 대역이다. pager 단추를 `list` 안에 넣으면 이 대역의 가정과 얽힐 수 있으니, pager 는 목록 컨테이너 **밖**(별도 요소)에 두는 편이 안전하다.
  - 삭제 후 `items` 를 지역에서 걸러 낸 뒤 다시 부르는 현재 흐름(124~126행)은 이중 렌더가 된다. 페이지 보정을 넣을 때 한 번만 렌더하도록 정리하되, 그 과정에서 "삭제 직후 목록에서 사라짐" 동작은 유지할 것.

- 차선 후보: **저장 결과 조회·삭제 실패의 HTTP 상태가 원인과 어긋나는 문제** — `getSavedResultHandler`(internal/server/resultsave.go:93)는 `Service.Get` 의 모든 실패를 `404 DF_SAVE_NOT_FOUND` 로, `deleteSavedResultHandler`(113행)는 모든 실패를 `400 DF_SAVE_DELETE_FAILED` 로 낸다. 그래서 메타 DB 장애나 "저장본이 손상됐습니다"(service.go:112) 도 클라이언트에는 "없는 결과" 로 보이고, 모니터링은 4xx 로 집계해 장애를 못 본다. 없음/권한(404)과 저장소 장애·손상(500)을 오류값으로 갈라 주면 된다(가치 3 / 위험 1 / 작업량 S). — 참고: "드라이버 오류 원문 노출" 걱정은 **이미 해결됨**. `writeProblem`(internal/server/http.go:1682)이 `safeProblemDetail`(internal/server/dberror.go:79)로 한 곳에서 거르고 Warn 로그를 남긴다. 같은 것을 또 고치지 말 것.
