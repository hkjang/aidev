- 과제: 목록 조회 다섯 곳이 `rows.Err()` 를 검사하지 않아 중간에 끊긴 결과를 200 으로 돌려주는 것을 고치기 (가치 3 / 위험 2 / 작업량 S)

- 왜: pgx 는 행 스트림이 도중에 깨져도 `rows.Next()` 가 조용히 `false` 를 돌려주므로, `rows.Err()` 를 보지 않는 핸들러는 **절반만 담긴 목록을 성공(200)으로** 내보낸다 — 호출자는 사람·API 키·사용자가 실제로 없어진 것과 구분할 수 없다. 같은 파일 안에 이미 올바른 관용구가 있는데(`data.go:566`, `data.go:679`, `data.go:818`, `settings.go:371`) 다섯 곳만 빠져 있어, 한 줄짜리 검사를 채워 넣고 그중 한 경로를 실제 postgres 회귀 시험으로 고정한다.

- 수용 기준:
  1) `GET /api/v1/users` 의 행 스트림이 두 번째 행에서 깨지면 응답이 **200 + 잘린 목록**이 아니라 **500 `internal_error`** 다(고치기 전 먼저 돌려 200 으로 빨개지는 것을 확인할 것).
  2) 같은 실행에서 정상 경로(깨뜨리지 않은 테이블)의 `listUsers` 는 고치기 전과 똑같이 200 이고 모든 행을 담는다 — 가드가 정상 목록을 삼키지 않는다는 회귀 기준.
  3) `data.go:listPeople`, `personal.go:listKeys`, `personal.go:listAPIKeys`, `personal.go:listKeyPermissions` 에도 같은 검사가 **루프 직후·`writeJSON` 앞**에 들어가고, `go vet ./...` · `gofmt -l internal/server` · `go test -race -count=1 ./...` 가 전부 초록이다.
  4) 새 오류 코드·메시지·헬퍼를 만들지 않는다 — 기존 `internalError(w, r, err)` 만 쓴다.

- 건드릴 파일 (프로덕션 3개):
  - `internal/server/settings.go:listUsers` — `for rows.Next()` 루프(207~215행 부근) 직후, `writeJSON(w, 200, …)` 앞에 `if err := rows.Err(); err != nil { internalError(w, r, err); return }`. 같은 파일 371행에 이미 같은 관용구가 있으니 그대로 맞출 것.
  - `internal/server/data.go:listPeople` — 루프가 `people = append(people, p)` 로 끝나는 163행 부근. 검사는 **루프 직후, `s.userCategories(...)` 호출 앞**에 넣는다(카테고리 조회의 오류 처리와 섞지 말 것).
  - `internal/server/personal.go` — 세 핸들러 각각의 루프 직후 `writeJSON` 앞: `listKeys`(루프 ~70~81행), `listAPIKeys`(~282~291행), `listKeyPermissions`(~355~368행).
  - 테스트: `internal/server/settings_db_test.go` **신규 1파일** 권장(기존 `timetravel_db_test.go` 의 `openTestStore`/`seedUser` 를 그대로 재사용 — 같은 패키지라 import 불필요).

- 검증 명령:
  - `gofmt -l internal/server` (출력 없음), `go vet ./...`, `go build ./...`
  - `go test -race -count=1 ./...` (DSN 없으면 새 DB 시험은 SKIP)
  - 실제 DB: `ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:<빈포트>/orbit?sslmode=disable' go test -race -count=1 -v ./internal/server -run 'TestListUsers|TestOrbit'`
  - 격리 컨테이너: `docker run -d --rm -e POSTGRES_PASSWORD=orbit -e POSTGRES_DB=orbit -p <빈포트>:5432 postgres:16-alpine`. **이전 회차가 쓴 포트(55433·55439·15434·55471·55481·55491·55521·55537·49761)는 피할 것.** 공유/운영 DB 금지.

- 행 스트림을 결정적으로 깨뜨리는 방법 (이번 과제의 핵심 — 정찰은 **실행해 보지 못했으므로 "미확인"**, 구현자가 가장 먼저 확인할 것):
  `store.Open` 으로 마이그레이션을 올린 뒤, 테스트 안에서 `users` 테이블을 가리고 **행별로 실패하는 뷰**를 씌운다. 프로덕션 질의(`SELECT id,username,email,display_name,role,status,last_login_at,created_at FROM users ORDER BY created_at`)는 한 글자도 바꾸지 않는다.
  ```sql
  ALTER TABLE users RENAME TO users_probe;
  CREATE VIEW users AS SELECT id, username, email,
    CASE WHEN username LIKE 'boom%' THEN username::int::text ELSE display_name END AS display_name,
    role, status, last_login_at, created_at FROM users_probe;
  ```
  `username::int` 는 상수가 아니라 **행마다 평가**되므로 계획 시점이 아니라 그 행에 닿았을 때 `22P02` 로 실패한다. `created_at` 이 더 늦은 `boom…` 사용자를 두 번째로 심으면 postgres 가 DataRow 한 개를 보낸 뒤 ErrorResponse 를 보내고, pgx 는 `rows.Next()` 를 한 번 true → 그 다음 false 로 돌려주며 `rows.Err()` 에만 오류가 남는다.
  - 확인 순서: ① 뷰를 씌우고 **고치기 전** 코드로 `listUsers` 를 불러 `200 + users 1개` 가 나오는지(= 결함 재현). 나오지 않으면(예: 500 이 바로 나옴) 이 주입법이 성립하지 않는 것이니 **아래 대안**으로 갈 것. ② 고친 뒤 500 `internal_error`. ③ `DROP VIEW users; ALTER TABLE users_probe RENAME TO users;` 로 `t.Cleanup` 에서 원복(다른 시험과 섞이지 않게 이 시험은 `t.Parallel()` 금지).
  - 대안(①이 실패할 때): 같은 뷰 기법을 `api_keys` + `personal.go:listAPIKeys` 로 옮겨 보거나, 그래도 안 되면 **수용 기준 1·2 를 버리고 프로덕션 5줄 수정만** 남긴 뒤 `go vet`·`go test -race ./...`·기존 시험 전부 초록으로 마무리할 것. 억지로 대역(fake Rows)을 만들어 증명하지 말 것 — `s.store.DB` 는 구체 타입 `*pgxpool.Pool` 이라 끼워 넣을 수 없고, 운영자가 손수 만든 대역으로 결함을 증명하는 것을 반복해서 반려했다.

- 위험과 피할 것:
  - `internal/server/auth.go:userByAPIKey`(154행 부근)도 `rows.Err()` 가 없지만 **이번 범위 밖**이다 — 보호 경로(auth)이고, 스트림이 깨지면 "invalid key"(401)로 fail-closed 라 데이터 손실이 아니다. 건드리지 말 것.
  - `internal/server/mcp.go:149` 의 검색 루프도 빠져 있지만 이번엔 제외한다 — 그 루프는 스캔 오류를 바깥 `err` 에 담고 `break` 하는 다른 구조라 `if err == nil { err = rows.Err() }` 같은 별도 판단이 필요하고, JSON-RPC 오류 매핑까지 봐야 해서 파일 수와 판단이 함께 늘어난다. 다음 회차 후보로 남길 것.
  - SQL 문자열·질의 시그니처를 바꾸지 말 것. `queryMemoriesLimit`·`orbitLinks` 등 REST/MCP/AI 가 공유하는 헬퍼는 이번에 손대지 않는다.
  - `internal/store/migrations` 에 파일을 더하지 말 것 — 뷰 기법은 **테스트 안의 일회용 DDL** 이고 커밋되는 것은 테스트 코드뿐이다.
  - `listPeople` 의 검사 위치를 `userCategories` 오류 처리 뒤로 밀지 말 것(그러면 카테고리 조회가 먼저 실패해 원인 오류가 가려진다).
  - CI 에는 postgres 가 없어 새 DB 시험은 SKIP 된다. 커밋 전에 DSN 을 주고 실제로 PASS 시킨 로그를 남길 것.

- 차선 후보: AI 제공자 SSE 파서(`internal/server/ai.go:proxyAIStream`)의 여러 `data:` 줄 합치기·프레임 경계·EOF flush 를 `httptest` 제공자 서버로 고정하는 회귀 테스트 (가치 2 / 위험 1 / 작업량 S). DB 가 필요 없고 프로덕션 함수를 그대로 부르므로, 위 뷰 기법이 성립하지 않아 1순위가 무너지면 이쪽으로 갈아탈 것.
