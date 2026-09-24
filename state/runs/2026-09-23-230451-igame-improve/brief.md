# 정찰 과제서 (2026-09-23) — igame @ main 3441eee

- 과제: `POST /api/v1/sessions/{id}/finish` 의 `result` 가 JSON 오브젝트인지 검증 (가치 3 / 위험 1 / 작업량 S)

- 왜: `finishGameSession`(internal/api/catalog.go:468~505)은 클라이언트가 보낸 `result` 를 그대로 `result=result||$4` 로 이어 붙이는데, jsonb `||` 는 "배열이 아닌 피연산자를 1원소 배열로 바꾼 뒤 배열 연결"이라서 `result:[1,2]` 나 `result:5` 를 보내면 `game_sessions.result`(001_initial.sql:136, `jsonb NOT NULL DEFAULT '{}'`)가 오브젝트에서 **배열로 바뀌어 버린다** — 그 뒤 `submitScore` 의 `result=result||jsonb_build_object('score',$3)`(catalog.go:598)도 키를 넣는 대신 배열에 덧붙여서 `result->>'score'` 가 영영 읽히지 않는다. 바로 옆 형제 핸들러 `startGameSession`(catalog.go:198~213)은 같은 클라이언트 jsonb(`metadata` → `client_info`)에 대해 이미 `"metadata must be a JSON object"` 400 을 내고 있으므로, 이 과제는 새 정책이 아니라 같은 파일 안의 기존 계약을 빠진 한쪽에 맞추는 일이다. SDK 타입도 `result?: Record<string, unknown>`(sdk/gamehub-js/src/*.ts:46)로 이미 오브젝트를 계약으로 적고 있다.

- 수용 기준:
  1) `POST /api/v1/sessions/{id}/finish` 에 `"result"` 가 오브젝트가 아닌 JSON(배열 `[1,2]`, 숫자 `5`, 문자열 `"x"`, `null`, 불리언)으로 오면 **DB 를 건드리기 전에** 400 `invalid_result` 로 거부하고, 해당 세션의 `status` 와 `result` 가 요청 전과 똑같이 남는다(세션이 `active` 로 남아야 한다 — 거부된 요청이 세션을 finish 시키면 안 된다).
  2) 기존 정상 경로는 그대로다: `result` 를 생략하거나 `{}` 로 보내면 지금처럼 200 + `status:"finished"`, 오브젝트를 보내면 그 키들이 `game_sessions.result` 에 병합되고 `result` 는 여전히 JSON 오브젝트다. RealmGuard/Defense 세션의 409 `authoritative_result_required`, 토큰·소유자 불일치의 409 `invalid_session` 도 변화 없음.
  3) 테스트가 증명할 것: (a) 수정 전이면 배열 `result` 가 통과해 `game_sessions.result` 가 `jsonb_typeof = 'array'` 가 된다는 사실(= 고치기 전 Red), (b) 수정 후 같은 요청이 400 이고 행이 불변이라는 것, (c) 오브젝트 `result` → `submitScore` 순서로 갔을 때 `result->>'score'` 가 읽힌다는 것. 진짜 `Router()` 를 httptest 로 띄우고 실제 `POST /api/v1/games/{slug}/sessions` 로 만든 세션·세션 토큰을 쓸 것(손으로 INSERT 한 행이나 가짜 대역으로 갈음하지 말 것).

- 건드릴 파일:
  - `internal/api/catalog.go:finishGameSession` — `if len(in.Result)==0 { in.Result=[]byte("{}") }` 바로 뒤에, `startGameSession` 과 같은 방식(`json.Unmarshal(in.Result, &map[string]any{})` 실패 시 400)으로 오브젝트 검사를 추가. **에러 코드는 `invalid_result`**(형제 핸들러의 `invalid_metadata` 와 구분), 메시지는 `result must be a JSON object`. 검사는 DB 조회(`SELECT g.slug …`)보다 **앞**에 두어 거부 요청이 DB 를 건드리지 않게 한다. 두 핸들러가 한 줄짜리 공통 헬퍼(예: `decodeJSONObject`)를 공유해도 좋지만, 공유한다면 `startGameSession` 의 기존 에러 코드·메시지 문자열은 한 글자도 바꾸지 말 것(기존 테스트·SDK 가 읽는다).
  - `internal/api/catalog_pg_test.go`(또는 새 `internal/api/session_finish_pg_test.go`) — `IGAME_TEST_DSN` 규약. `achievements_pg_test.go` 의 `migratedPool(t)` / `insertTestUser` / `insertTestSession` / `insertTestGame` / `httptest.NewServer(New(pool,…).Router())` 패턴을 그대로 따를 것. `migratedPool` 은 기본 스키마를 공유하므로 고유 tag 를 붙이고 `t.Cleanup` 으로 지울 것.
  - 문서: `docs/api.md` 의 `POST /sessions/{id}/finish` 행에 "result 는 JSON 오브젝트여야 한다" 한 줄. (`sdk/gamehub-js` 타입은 이미 오브젝트라 수정 불필요 — 프런트/SDK 는 건드리지 말 것.)

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l cmd internal migrations`
  - `go vet ./cmd/... ./internal/... ./migrations/...`
  - `go build ./...`
  - `go test ./cmd/... ./internal/... ./migrations/... -count=1` (DSN 없으면 PG 테스트는 skip)
  - 실 DB: 일회용 `postgres:17-alpine` 컨테이너를 띄우고 README 절차대로 pgcrypto 를 확장 전용 스키마에 설치한 뒤
    `make test-db DSN='postgres://…'` (Makefile:37 — `internal/api` 와 `internal/database` 를 함께 돌린다), 이어서 `go test -race -count=3 ./internal/api/ -run <새 테스트>` (IGAME_TEST_DSN 설정)
  - `bash scripts/check-release-contract.sh`
  - 이 호스트에 docker 가 살아 있고 `postgres:17-alpine` 이미지가 이미 있다(확인함). **다른 프로젝트의 기존 컨테이너(`weekly-test-pg`, `resso-test-pg`)를 쓰지 말고 새 일회용 컨테이너를 띄울 것.**

- 위험과 피할 것:
  - jsonb `||` 의 "비배열은 1원소 배열로 승격" 규칙은 PostgreSQL 문서에 근거한 것이고 **이 세션에서 실제 psql 로 확인하지는 못했다(미확인)**. 구현자는 손대기 전에 일회용 DB 에서 `SELECT ('{}'::jsonb || '[1,2]'::jsonb)::text, jsonb_typeof('{}'::jsonb || '5'::jsonb)` 를 한 번 찍어 전제를 확인하고, 만약 PG 가 오히려 에러를 내서 현재도 거부된다면 **동작이 안 바뀌는 수정이 되므로 이 과제를 버리고 차선 후보로 갈 것**(운영자 규칙: 실제 출력이 바뀌지 않는 수정은 넣지 않는다).
  - 같은 `result` 컬럼을 쓰는 다른 경로를 한쪽만 넓히지 말 것. 확인한 바로는 클라이언트 원문 jsonb 가 `result` 로 들어가는 곳은 `finishGameSession` 한 곳뿐이고, `submitScore`(catalog.go:598)·Defense(defense.go:1633)·RealmGuard(realmguard.go:1554)는 모두 서버가 만든 `jsonb_build_object`/요약이라 안전하다. 구현자는 이 점을 수정 전에 grep 으로 재확인할 것.
  - **범위를 넓히지 말 것**: 끝난 세션에 계속 `result` 를 덧붙일 수 있는 문제(`status IN ('active','finished')`, catalog.go:500)는 재시도 정책이 확정되지 않아 이전 회차들이 일부러 보류한 별개 건이다. 이번에 같이 고치지 말 것.
  - 보호 경로 회피: `internal/api/auth.go`·`throttle.go`(계정+주소 조합 키는 `throttle_test.go:37` 이 의도적으로 고정한 설계다 — 건드리지 말 것), `migrations/*.sql`(체크섬 불변), `.github/workflows/*` 는 이번 과제에서 손대지 않는다. 마이그레이션 없이 끝나는 과제다.
  - `decodeJSON` 은 `DisallowUnknownFields` 이므로 테스트 요청 본문에 없는 필드를 넣으면 400 이 난다. 세션 생성은 `decodeOptionalJSON` 이라 본문 생략이 가능하다.

- 차선 후보: **Migrate 의 context 취소 시 롤백·재시도 계약을 실제 PostgreSQL 로 검증** (가치 3 / 위험 1 / 작업량 M) — 여러 회차에서 보류로 남은 항목. `internal/database/database.go` 의 `Migrate` 가 파일별 트랜잭션 도중 ctx 가 취소되면 DDL 과 `schema_migrations` 이력이 함께 롤백되고, 같은 DSN 으로 재실행하면 처음부터 다시 적용되어 체크섬·`applied_at` 계약이 유지되는지를 `internal/database/database_pg_test.go` 에 테스트로 추가한다(소스 변경 없이 테스트만). 검증은 위와 같은 `make test-db DSN=…`.
