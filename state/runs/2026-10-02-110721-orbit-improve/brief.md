# 과제서 — 2026-10-02 (orbit, base main@3aceddd / VERSION 0.7.6)

- 과제: MCP `tools/call orbit_search_people` 이 `rows.Err()` 를 보지 않아 끊긴 사람 검색 결과를 성공으로 돌려주는 것을 고치기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/server/mcp.go:151` 의 `for rows.Next()` 루프는 `rows.Scan` 오류만 바깥 `err` 에 담고(154~157줄) 루프가 끝난 뒤 `rows.Err()` 를 전혀 보지 않은 채 `result = items`(160줄) 로 넘어간다. pgx 는 행 스트림이 실행 중에 깨져도 `Next()` 를 조용히 `false` 로 돌려주고 오류를 `rows.Err()` 에만 남기므로(v0.7.6 에서 실제 postgres 로 확인된 성질), 잘린(또는 0행) 사람 목록이 `isError` 없는 정상 `structuredContent` 로 나가고 외부 MCP 에이전트는 "검색 결과가 정말 그것뿐" 이라고 결론 낸다. v0.7.6(e16e59d)이 REST 다섯 곳에 채운 같은 가드가 저장소에서 이 한 곳만 비어 있다 — `internal/` 전체에서 `rows.Next()` 루프 28곳을 `rows.Err()` 검사와 맞대 보니 가드가 없는 곳은 `mcp.go:151` 과 `auth.go:155` 둘뿐이고, 후자는 보호 경로이며 401 fail-closed 라 이번 범위에서 의도적으로 뺀다.

- 수용 기준:
  1) 행 스트림이 실행 중에 깨지면 `tools/call orbit_search_people` 응답이 `result.isError == true` 이고 `result.content[0].text` 가 기존 일반 메시지(`"요청을 처리하지 못했습니다."`)다. 고치기 전에는 같은 요청이 `isError` 없이 `structuredContent` 를 담은 성공 응답으로 나가는 것을 **먼저 돌려 확인**할 것.
  2) 정상 검색은 예전과 똑같다 — 질의에 걸리는 사람 전원이 `structuredContent` 에 담기고 `isError` 가 없다. 이 하위 시험은 뷰를 씌우기 **전에** 같은 실행에서 통과해야 한다(가드가 멀쩡한 목록을 삼키지 않는다는 회귀 기준).
  3) 가드 한 줄만 되돌리면 1)의 시험이 다시 빨개진다(인과 고정). `gofmt -l internal/server` 무출력, `go vet ./...`, DSN 없음/있음 양쪽에서 `go test -race -count=1 ./...` 초록.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/server/mcp.go` — `mcpCall` 의 `case "orbit_search_people"` 루프(151~160줄) 뒤, `result = items` **앞**에 `rows.Err()` 검사를 넣는다. 이 case 는 다른 네 곳과 구조가 달라(스캔 오류를 바깥 `err` 에 담고 `break`) 기존 `internalError` 관용구를 그대로 쓸 수 없다. 스캔 오류를 덮어쓰지 않도록 쓸 것:
    ```go
    if err == nil {
        err = rows.Err()
    }
    result = items
    ```
    새 오류 코드·메시지·헬퍼를 만들지 말 것. 224~231줄의 기존 매핑이 그대로 받는다 — `errors.Is(err, pgx.ErrNoRows)` 가 아니므로 일반 메시지 + `isError:true` + HTTP 200 JSON-RPC 봉투(다른 MCP 오류와 같은 계약)로 나간다. SQL·질의 시그니처·`escapeLike` 는 한 글자도 바꾸지 말 것.
  - `internal/server/mcp_db_test.go` (신규) — 아래 기법으로 회귀 시험 1개(하위 2개).

- 검증 기법 (v0.7.6 이 실제 postgres 로 성립시킨 뷰 주입을 그대로 옮긴다):
  `internal/server/settings_db_test.go:30~58 breakUsersRowStream` 이 정본 예시다. 같은 모양으로 `people` 을 가린다 —
  `ALTER TABLE people RENAME TO people_probe` → 동명 뷰 생성 → `t.Cleanup` 으로 `DROP VIEW IF EXISTS people` + `ALTER TABLE people_probe RENAME TO people`. `people` 의 컬럼은 `001_init.sql:73~87` 의 13개가 전부이고 이후 마이그레이션에 `ALTER TABLE people` 이 없다(확인함) — 뷰 SELECT 목록은 `id, user_id, <CASE display_name>, company, role_title, avatar_url, email_cipher, phone_cipher, note_cipher, key_version, first_met, created_at, updated_at` 순서를 그대로 지킬 것:
  ```sql
  CREATE VIEW people AS SELECT id, user_id,
    CASE WHEN display_name LIKE 'boom%' THEN display_name::int::text ELSE display_name END AS display_name,
    company, role_title, avatar_url, email_cipher, phone_cipher, note_cipher,
    key_version, first_met, created_at, updated_at FROM people_probe;
  ```
  - 씨앗은 뷰를 씌우기 **전에** 넣는다(CASE 컬럼 때문에 뷰는 자동 갱신이 안 된다). `timetravel_db_test.go:53 seedPerson` 은 `display_name` 을 "시험 인물" 로 고정하므로 `settings_db_test.go:18 seedNamedUser` 처럼 이름을 받는 작은 변형(`seedNamedPerson`)을 테스트 파일에 두고, 같은 파일의 `seedUser`(41줄)·`seedRelationship`(72줄)·`openTestStore`(23줄)는 그대로 재사용한다. MCP 질의는 `people p JOIN relationships r` 이므로 **사람마다 relationships 행이 반드시 있어야** 한다(프로필의 알려진 함정).
  - 프로덕션 배선으로 호출할 것(손으로 만든 대역 금지): `s := &Server{store: st}` 에 `s.mcp(rec, req)` 로 실제 JSON-RPC 요청 본문을 준다 —
    `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"orbit_search_people","arguments":{"query":"boom"}}}`.
    요청 컨텍스트에 `userContextKey = User{ID: userID}` 를 넣는다. `authContextKey` 는 넣지 않아도 된다 — `requestHasScope`(236~239줄)는 `authInfo` 가 없으면 `info.APIKey == false` 라 true 를 준다(코드 확인).
  - 쿼리가 `ORDER BY r.importance DESC` 로 블로킹 Sort 를 쓰므로, v0.7.6 의 교훈대로 잘린 결과는 1행이 아니라 **0행**일 가능성이 높다(postgres 가 DataRow 를 하나도 보내기 전에 ErrorResponse 를 보냄). 결함의 성질은 같다 — 시험은 "행이 몇 개 빠졌나" 가 아니라 **`isError` 가 있는가**로 판정할 것. 몇 행이 나오는지에 의존하는 단정은 쓰지 말 것.
  - 뷰를 쓰는 시험은 `t.Parallel()` 을 쓰지 말 것(`people` 을 공유하는 다른 DB 시험과 섞인다). 시험 뒤 `pg_class` 로 `people` 이 다시 `relkind='r'` 이고 `people_probe` 가 없음을 확인할 것.
  - `people` 을 rename 하면 `relationships`/`interactions`/`memories`/`person_links` 의 FK 가 `people_probe` 를 따라간다(테이블 rename 이라 제약은 함께 이동) — 이 점은 **미확인**이다. rename 이 거부되거나 cleanup 이 꼬이면 `relationships` 를 같은 방식으로 가리고 `relationship_label` 로 터뜨리는 쪽으로 물러설 것(MCP 질의의 WHERE·SELECT 양쪽에 그 컬럼이 있다). 그 경우 뷰 컬럼 목록에 `003_anchored_relationships.sql` 의 `anchored` 를 빠뜨리지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/server` (무출력)
  - `go vet ./...`
  - `go build ./...`
  - `go test -race -count=1 ./...` — DSN 없이 초록(새 DB 시험은 SKIP). 이번 정찰에서 `go test ./...` 기준선 초록 확인.
  - `ORBIT_TEST_DATABASE_URL='postgres://…' go test -race -count=1 -v ./internal/server -run 'TestMCP'` — 격리 postgres 필수.
  - 격리 컨테이너: `docker run --rm -d -p <미사용포트>:5432 -e POSTGRES_PASSWORD=… postgres:16-alpine`. **이미 쓴 포트 금지**: 55433·55439·15434·55471·55481·55491·55521·55537·55603·49761. 공유/운영 DB 금지.

- 위험과 피할 것:
  - `auth.go:155` 의 같은 누락은 **이번 범위 밖**(보호 경로, 401 fail-closed). 한 회차에 같이 담지 말 것.
  - `queryMemories`·`personConnections`·`escapeLike` 는 REST·AI 와 공유한다 — 시그니처·SQL 금지.
  - `internal/store/migrations` 의 기존 파일 수정 금지, 새 마이그레이션도 이번엔 불필요. `.github/workflows` 금지.
  - CI 에 postgres 가 없어 새 DB 시험은 CI 에서 SKIP 된다 — DSN 을 직접 주고 PASS 를 확인하지 않으면 증거가 없다.
  - grep 결과("`rows.Err()` 문자열이 없다")를 증거로 제출하지 말 것. 실제 핸들러 호출의 응답 JSON 으로 before/after 를 보일 것.
  - 뷰 DDL 을 되돌리지 않은 채 끝내면 같은 컨테이너의 다른 DB 시험이 전부 깨진다. `t.Cleanup` 필수.

- 차선 후보: AI 제공자 SSE 파서 회귀 시험 — `internal/server/ai.go:206~283 proxyAIStream` 의 여러 `data:` 줄 합치기(248·253·278줄), 빈 줄 프레임 경계(270~273), EOF flush(281), `[DONE]` 무시(255)를 `httptest` 제공자 서버로 고정한다. DB 불필요하고 프로덕션 함수를 직접 부를 수 있다(기존 `TestExtractDelta` 는 맵 추출만 본다). 가치 2 / 위험 1 / 작업량 S.
