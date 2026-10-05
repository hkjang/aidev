- 과제: MCP `initialize` 응답에 `Mcp-Session-Id` 헤더를 실어 세션 추적을 살린다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `internal/mcp/server.go:openSession` 은 `initialize` 때마다 `mcp_sessions` 에 UUID 행을 넣지만, 그 id 를 **응답 헤더로 내보내지 않는다** — `Mcp-Session-Id` 는 `touchSession`/`closeSession` 에서 **읽기만** 한다(저장소 전체에서 `SessionHeader` 참조는 server.go:22,347,358 세 곳뿐이고 `w.Header().Set(SessionHeader, …)` 은 0건). 그래서 MCP 클라이언트는 세션 id 를 알 길이 없어 후속 요청에 되돌려 보내지 못하고, `last_seen_at` 은 INSERT 기본값(`DEFAULT NOW()`, migrations/0001_init.sql:187)에 영구히 머물며 `closed_at` 은 영원히 NULL 이다. 결과로 관리 콘솔의 활성 세션 지표(`internal/api/admin.go:92`, `:287-288` 의 `closed_at IS NULL AND last_seen_at > NOW() - INTERVAL '1 hour'`)는 **쓰는 중인 세션을 1시간 뒤 0으로 떨어뜨리면서도 끝난 세션을 영구히 열린 것으로 남긴다** — 두 방향으로 다 틀린다. 헤더 한 줄을 실으면 이미 작성돼 있지만 한 번도 돌아 본 적 없는 touch/close 경로가 실제로 동작하고, 지표가 맞아진다.

- 수용 기준:
  1) `POST /mcp` 의 `initialize` 응답(200)에 `Mcp-Session-Id` 헤더가 있고, 그 값이 `mcp_sessions` 에 방금 들어간 행의 `id` 와 같은 UUID 다(= `result.instructions` 안의 "세션:" 값과도 같다).
  2) 그 헤더 값을 `Mcp-Session-Id` 로 되돌려 보낸 `tools/list` 요청 뒤 해당 행의 `last_seen_at` 이 `created_at` 보다 **커진다**(touchSession 이 실제로 돈다). `DELETE /mcp` 를 같은 헤더로 보내면 204 이고 `closed_at IS NOT NULL` 이 된다.
  3) 인증 실패하는 `initialize`(토큰 없음 → 401)에는 `Mcp-Session-Id` 헤더가 **붙지 않고** `mcp_sessions` 행도 생기지 않는다 — 세션 발급이 인증 뒤라는 것을 테스트가 증명해야 한다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/mcp/server.go` — `dispatch` 의 시그니처에 `http.ResponseWriter` 를 더하고(`dispatch(w, r, req)`; 호출부는 단건 경로 `:132` 와 배치 루프 `:115` 두 곳), `initialize` 가 `openSession` 에서 받은 `sessionID` 가 빈 문자열이 아닐 때 `w.Header().Set(SessionHeader, sessionID)` 를 호출하게 한다. `handlePost` 는 dispatch 이후에 `writeJSON`(→ `WriteHeader`)을 부르므로 헤더 설정 시점은 안전하다 — **이 순서를 뒤집지 말 것.** `initialize` 는 인증 실패 시 `openSession` 전에 `fail(...)` 로 반환하므로(`:173-176`) 기준 3)은 추가 분기 없이 성립한다. 배치 안에 `initialize` 가 여러 개면 마지막 것이 헤더를 덮는데, 그건 현실적으로 없는 경우이고 스펙도 정하지 않았으니 그대로 둘 것(더 손대면 범위가 커진다).
  - `internal/api/mcpoauth_e2e_test.go` — 기존 프로덕션 배선 하네스(`newGateway`: 실제 `db.Migrate`, `settings.Store`, Keycloak 스텁 `keycloakStub`, Bitbucket 스텁 `newBitbucketStub`, 실제 chi 라우터)를 **확장**한다. 새 목을 만들지 말 것. `gw.post(t, path, bearer, body)` 는 추가 헤더를 못 넣으므로(`:248`) 같은 모양의 변형 하나를 더하거나(예: `postHdr(t, path, bearer, body string, hdr map[string]string)`) 새 테스트 안에서 `http.NewRequest` 로 직접 보낸다 — `g.srv.Client().Do` 와 `g.srv.URL` 을 쓰면 된다. 새 테스트 함수(예: `TestMCPSessionIDIsIssuedAndTracked`)로 두고 `TestMCPOAuthDiscoveryAndCall` 은 건드리지 말 것(그 테스트는 OAuth 디스커버리 전체를 보는 긴 시나리오다). `gateway` 구조체는 지금 `srv`/`store`/`ctx` 세 필드뿐이고 **pool 이 없다**(`:121-125`, 확인함) — `newGateway` 안에 이미 있는 `db.Pool` 을 `pool *pgxpool.Pool` 필드로 노출시켜(테스트 파일만 수정) `mcp_sessions` 를 직접 SELECT 해 `created_at`/`last_seen_at`/`closed_at` 을 비교할 것.

- 검증 명령:
  ```
  docker run -d --rm --name bbmcp-test-db -e POSTGRES_USER=bbmcp -e POSTGRES_PASSWORD=bbmcp \
    -e POSTGRES_DB=bbmcp_test -p 5432:5432 postgres:16-alpine
  go build ./... && go vet ./...
  TEST_DATABASE_URL='postgres://bbmcp:bbmcp@localhost:5432/bbmcp_test?sslmode=disable' \
    go test ./internal/api/ -run TestMCP -count=1 -v
  TEST_DATABASE_URL='postgres://bbmcp:bbmcp@localhost:5432/bbmcp_test?sslmode=disable' \
    go test ./... -count=1 -p 1
  ```
  - `-p 1` 은 필수다(패키지들이 같은 DB 를 공유하고 테스트마다 TRUNCATE 한다).
  - `TEST_DATABASE_URL` 이 없으면 `newGateway` 가 **조용히 `t.Skip`** 한다(`mcpoauth_e2e_test.go:130-132`). `-v` 로 `--- SKIP` 0건을 확인하지 않은 "통과"는 통과가 아니다.
  - 수정 전에 수용 기준 1) 이 실패하는 것을 먼저 보고(헤더 빈 문자열) 그 출력을 회차 노트에 남길 것.
  - **미확인**: 이 정찰 세션에서는 Bash 가 제한돼 `go build`/`go vet`/`go test` 를 한 번도 돌리지 못했다. 위 명령은 프로필(2026-10-03, 당시 실제로 돌려 확인됨)에서 가져온 것이고 `a3f2955` 이후 커밋이 없어 유효할 것으로 보지만, 구현자가 먼저 베이스라인을 돌려 깨끗한지 확인할 것.

- 위험과 피할 것:
  - **`internal/auth/*`, `internal/api/oauth.go`, `auth_handlers.go` 를 건드리지 말 것.** 이번 과제는 세션 id 발급이지 인증이 아니다. Keycloak 리다이렉트 URI·사일런트 SSO 로 최근에 두 번 깨진 자리다.
  - **세션 id 검증(없거나 모르는 id 에 404)을 같이 넣지 말 것.** 스펙상 권장이지만 지금 헤더를 보내지 않는 클라이언트가 전부 깨진다. 별 회차로 미룰 것.
  - `internal/database/migrations` 는 손댈 필요가 없다 — 열은 이미 다 있다. 마이그레이션 추가 금지.
  - `dispatch` 는 값이 아니라 포인터로 `*Request` 를 받는다. `handleGet`(SSE)과 `closeSession` 은 이미 `w`/`r` 를 직접 받으므로 바꿀 필요가 없다.
  - 운영자 규칙: 손으로 만든 대역으로 증명하지 말 것 — 증거는 반드시 `newGateway` 를 거친 실제 HTTP 왕복과 실제 `mcp_sessions` 행이어야 한다. grep 으로 "`w.Header().Set` 이 있다"를 증거로 내지 말 것.
  - `instructions()` 는 세션 id 를 본문 텍스트에도 넣는다(`:218`, `:225`). 헤더를 추가하면서 **본문 쪽을 지우지 말 것** — 기존 테스트가 instructions 를 본다.

- 차선 후보: `internal/api/mcpoauth_e2e_test.go` 하네스로 `tools/call` 응답의 KB 절단 경로를 검증(`internal/mcp/server.go:303-316`, `cfg.MaxResponseKB` 기본 512, `limit*1024` 바이트에서 **UTF-8 문자 중간을 자른다** → 한국어 응답에서 깨진 바이트가 JSON 으로 나간다. `string(text[:limit*1024])` 가 그 자리. 프로덕션 1파일·S. 단 `MaxResponseKB` 를 작게 설정해 경로를 타게 만드는 방법을 `settings` 쪽에서 먼저 확인해야 한다 — 미확인).
