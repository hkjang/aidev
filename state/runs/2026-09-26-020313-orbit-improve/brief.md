- 과제: `GET /orbit?at=` 이 `earliest_at` 을 빼먹어 현재/과거 두 경로의 응답 계약이 어긋나는 것을 고치고, Time Travel 질의 매개변수·응답 필드를 openapi.go·docs/API.md 에 적는다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 같은 `GET /api/v1/orbit` 이 현재(`internal/server/data.go:getOrbit`, 772줄)에는 `earliest_at` 을 담아 주는데 과거(`internal/server/timetravel.go:writeOrbitAt`, 165~173줄)에는 그 키가 아예 없다 — 두 경로가 같은 자원을 다른 모양으로 돌려준다. 화면은 `web/src/pages/OrbitPage.tsx:67` 의 `if (orbit.earliest_at) setEarliest(...)` 로 가리고 있지만, API 키·MCP 로 `?at=` 만 부르는 호출자는 시간 여행 가능 구간을 알 방법이 없고 `at`·`historical`·`earliest_at` 은 `internal/server/openapi.go:13` 에도 `docs/API.md` 에도 한 줄이 없다.
- 수용 기준:
  1) `GET /api/v1/orbit?at=<과거시각>` 응답에 `earliest_at` 이 들어가고, 같은 데이터에서 `?at=` 없이 부른 응답의 `earliest_at` 과 같은 값이다(둘 다 `s.orbitRange` 한 곳에서 나온다 — 두 번째 질의를 새로 쓰지 말 것).
  2) 기록이 하나도 없는 사용자는 두 경로 모두 `earliest_at` 이 `null` 이고, 기존 키(`center`·`nodes`·`contexts`·`links`·`categories`·`generated_at`, 과거 경로의 `at`·`historical`)는 하나도 사라지거나 이름이 바뀌지 않는다.
  3) 테스트가 증명할 것: 실제 postgres 위에서 **실제 핸들러 `getOrbit` 을 두 번**(`?at=` 있음/없음) 불러 JSON 을 디코딩해, (a) 과거 응답에 `earliest_at` 키가 존재하고 (b) 두 응답의 `earliest_at` 이 같은 시각이며 (c) 과거 응답의 `historical` 이 true 임을 각각 확인한다. 고치기 전에 먼저 돌려 (a) 가 빨개지는 것을 확인하고 시작할 것.
- 건드릴 파일:
  - `internal/server/timetravel.go:writeOrbitAt` — `s.orbitLinks`/`s.userCategories` 를 부르는 자리 옆에서 `s.orbitRange(r.Context(), u.ID)` 를 부르고 실패는 `internalError(w, r, err)` 로(같은 함수의 기존 세 호출과 똑같은 모양), 응답 맵에 `"earliest_at": first` 추가. 다른 키·순서·주석은 건드리지 말 것.
  - `internal/server/timetravel_db_test.go` — 새 `TestOrbitAtResponseCarriesEarliestAt`. 기존 헬퍼 `openTestStore`/`seedUser`/`seedPerson`/`seedInteraction` 를 그대로 쓰고, 핸들러 호출은 `internal/server/data_test.go:20` 의 방식 그대로 `req.WithContext(context.WithValue(req.Context(), userContextKey, User{ID: userID}))` + `httptest.NewRecorder()` + `s := &Server{store: st}` + `s.getOrbit(rec, req)` 로 한다(라우터·미들웨어를 새로 세우지 말 것).
  - `internal/server/openapi.go:13` — `"/orbit"` 의 `operation("Orbit 그래프 조회", "orbit:read")` 에 `at` 질의 매개변수(선택, RFC3339 또는 `YYYY-MM-DD`)를 붙인다. **먼저 `operation()` 의 시그니처를 읽고**, 매개변수를 받을 자리가 없으면 함수를 바꾸지 말고 `"/orbit"` 항목만 리터럴로 풀어 쓸 것(다른 경로는 그대로).
  - `docs/API.md` — Time Travel 절 신설: `?at=` 두 형식, 날짜만 준 값은 **UTC 자정**으로 읽힌다는 것(`timetravel.go:parseOrbitAt` 의 `"2006-01-02"` 파싱이 그렇다), 응답의 `historical`·`at`·`earliest_at`, 그리고 "중요도·소속·고정 여부는 이력이 없어 오늘 값을 쓴다"(`timetravel.go` 머리 주석에 이미 적힌 한계). `docs/cru-manual.md` 는 코드와 어긋난 곳이 있으니 베끼지 말고 실제 핸들러를 정본으로 삼을 것.
- 검증 명령:
  - `gofmt -l internal/server` (출력 없어야 함), `go vet ./...`
  - `go test -race ./...` — DSN 없이. 새 DB 테스트는 SKIP 되고 나머지가 초록이어야 한다.
  - DB 붙여서: `docker run -d --rm -e POSTGRES_PASSWORD=orbit -p 55481:5432 postgres:16-alpine` 뒤
    `ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55481/postgres?sslmode=disable' go test -race -count=1 -v ./internal/server -run TestOrbit`
    (포트 55433·55439·15434·55471 은 다른 세션이 쓸 수 있으니 피했다. 공유·운영 DB 금지, 끝나면 컨테이너 정리.)
  - `cd web && npm run test -- --run` 은 이번 변경이 웹을 건드리지 않으므로 선택. 웹을 한 줄도 고치지 않았다면 생략해도 된다.
- 위험과 피할 것:
  - **`internal/server/auth.go`·`auth_test.go`·`web/src/silentSso.ts` 는 열지도 말 것** — v0.7.0(머지 df2242f)에서 silent SSO 로 크게 바뀐 인증 경로다. `internal/store/migrations`·`.github/workflows` 도 보호 경로다.
  - `getOrbit` 쪽 응답은 한 글자도 바꾸지 말 것. 이번 과제는 과거 경로를 현재 경로에 **맞추는** 것이지 둘을 새 모양으로 통일하는 것이 아니다.
  - `orbitRange` 질의를 복사해 새로 쓰지 말 것. 2026-09-20 회차가 이 질의를 조인에서 두 서브쿼리로 고쳐 놓았고 `TestOrbitRangeMatchesLegacyJoin` 이 그것을 지키고 있다 — 사본이 생기면 다음에 한쪽만 바뀐다.
  - `web/src/pages/OrbitPage.tsx:67` 의 `if (orbit.earliest_at)` 가드는 그대로 둘 것. 서버가 항상 보내게 됐다고 가드를 지우면 옛 서버와 새 화면을 섞어 쓸 때 슬라이더가 사라진다.
  - `time.Time` 비교는 `==` 가 아니라 `.Equal()` — pgx 가 돌려주는 값의 위치 정보가 다르다. 응답을 JSON 으로 디코딩해 비교한다면 문자열끼리 비교해도 되지만 `*time.Time` 로 받으면 반드시 `.Equal()`.
  - CI 에는 postgres 가 없어 DB 테스트가 늘 SKIP 된다 — "CI 가 초록" 을 이 변경의 증거로 쓰지 말고 위의 DSN 명령을 실제로 돌린 출력을 남길 것.
- 작업량 근거(추정의 기초): 아래로 쪼갠 뒤 합친 값이다 — writeOrbitAt 수정 4줄(5분) / DB 테스트 1개 신설 40~60줄(20~30분, 기존 헬퍼 재사용·docker 기동 포함) / openapi.go 의 `/orbit` 항목(10분, `operation()` 시그니처를 먼저 읽어야 해서 여기가 가장 흔들린다) / docs/API.md Time Travel 절(10분) / gofmt·vet·`go test -race ./...`(5분). 합계 **50~65분, 10번 중 8번은 이 구간**. 빠져나가는 쪽은 하나뿐이다: `operation()` 이 매개변수를 받지 못해 `/orbit` 항목을 리터럴로 풀어 써야 하면 +15분. 그 경우 문서(openapi.go·API.md)를 떼어 내고 수용 기준 1~3 만 마치는 것이 옳다 — 계약 수정이 이 과제의 본체이고 문서는 여유분이다. **웹은 이번 범위 밖**이다(OrbitPage 가드를 손대지 않으므로 vitest·vite build 가 필요 없다).
- 차선 후보: `orbitAt` 의 교류 조회에 상한이 없는 것 (`internal/server/timetravel.go:93` 의 `SELECT person_id,occurred_at,weight FROM interactions WHERE user_id=$1 AND occurred_at<=$2` — 사람 조회에는 `LIMIT 1000` 이 있는데 여기에는 없어 교류가 많은 사용자의 전체 이력이 한 요청에 메모리로 올라온다). 다만 단순히 LIMIT 을 걸면 `closeness`·`momentum`·`last_interaction_at` 이 조용히 틀어지므로, 집계를 DB 로 내리는 방향이어야 하고 그러면 S 를 넘는다. 1순위가 성립하지 않으면 이쪽보다 `orbitAt` 의 `contexts`(분류 집계) 를 실제 postgres 로 고정하는 순수 테스트 보강을 먼저 고를 것.
