- 과제: DELETE /mcp 세션 종료에 인증과 소유자 검증 추가 (가치 4 / 위험 2 / 작업량 M)
- 왜: `internal/mcp/server.go:45 Handle` 의 DELETE 분기는 `s.auth.PrincipalFor(r)` 를 전혀 호출하지 않고 바로 `closeSession(r)` 을 실행하며, `closeSession` 의 SQL 은 `UPDATE mcp_sessions SET closed_at=NOW() WHERE id=$1` 로 소유자를 보지 않는다(server.go:382-391). GET(server.go:66)·POST의 initialize(server.go:176)는 모두 인증을 요구하므로 DELETE 만 계약이 어긋나 있고, 세션 ID 를 알아낸 제3자나 다른 사용자가 남의 세션을 종료 처리(관리 콘솔의 활성 세션 집계 왜곡)할 수 있다.
- 수용 기준:
  1) 인증 헤더 없는 `DELETE /mcp` + 유효한 `Mcp-Session-Id` → 401 과 GET 과 동일한 `WWW-Authenticate` 챌린지(`s.unauthorized` 재사용), 그리고 해당 세션의 `closed_at` 은 그대로 NULL.
  2) 인증된 사용자 B 가 사용자 A 의 세션 ID 로 DELETE → 세션 존재 여부를 알려 주지 않도록 204 를 반환하되 `closed_at` 은 NULL 유지. A 가 자기 세션으로 DELETE → 204 이고 `closed_at` 이 설정됨. (`closeSession` 을 `WHERE id=$1 AND user_id=$2` 로 좁히는 것이 최소 변경. `Identity.Principal` 이 항상 `UserID: id.User.ID` 를 채우므로(internal/auth/service.go:231) apikey·oauth 양쪽에서 user_id 가 비지 않는다. 혹시 user_id NULL 행이 있을 수 있다면 `AND (user_id=$2 OR (user_id IS NULL AND username=$3))` 로 보완.)
  3) 테스트가 증명할 것: 위 세 경우(미인증·타인·본인)의 HTTP 상태와 `closed_at` 의 실제 변화. `closed_at` 은 관리자 세션으로 `GET /api/admin/sessions?limit=200` (라우트: internal/api/admin.go:79 → `listMCPSessions`, internal/api/admin_users.go:276) 의 응답에서 해당 id 를 찾아 확인한다 — 소스 문자열 검사나 직접 주입한 의존성이 아니라 실제 HTTP 경로로 증명할 것.
- 건드릴 파일:
  - `internal/mcp/server.go:Handle` — DELETE 분기에서 먼저 `PrincipalFor` 를 호출, 실패 시 `s.unauthorized(w, r, err)` 로 401. 성공 시 `s.closeSession(r.Context(), principal, r)` 후 204.
  - `internal/mcp/server.go:closeSession` — 시그니처에 `ctx`·`tools.Principal` 를 받아 UPDATE 를 소유자로 좁힌다. 반환값(갱신 행 수)은 응답에 쓰지 말 것(열거 오라클 방지). `touchSession` 은 건드리지 말 것.
  - `hack/e2e/e2e_test.go` — 헬퍼 `mcpInitialize(t, key) string` 추가: `POST /mcp` 로 `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"…","clientInfo":{"name":"e2e","version":"0"}}}` 를 보내고 응답의 `Mcp-Session-Id` 헤더를 돌려준다(server.go:190-195 에서 설정). 헬퍼 `mcpDelete(t, key, sessionID) int`, 그리고 `TestMCPSessionCloseRequiresOwner` 추가. 기존 `tool()` 처럼 호출 사이 500ms sleep 으로 분당 120 요청 제한을 피한다(e2e_test.go:93-96 주석). 기존 `aliceKey`/`bobKey`/`adm` (setup 의 전역)을 그대로 쓴다.
  - 그 외 파일은 건드리지 않는다(총 3파일, 프로덕션 1파일).
- 검증 명령 (모두 .github/workflows/ci.yml 에 있는 것 또는 e2e 파일 헤더에 적힌 것):
  - `go build ./...`
  - `go vet ./...`
  - `TEST_DATABASE_URL=postgres://confmcp:confmcp@localhost:5432/confmcp_test?sslmode=disable go test ./... -count=1 -p 1`
  - e2e: `scripts/dev.sh up` 후 `CONFMCP_URL=http://127.0.0.1:18088 MOCK_CONFLUENCE=http://127.0.0.1:18090/confluence go test -tags e2e -count=1 -v ./hack/e2e -run 'MCPSession'` → 수정 전 실패(미인증 DELETE 가 204 이고 closed_at 설정됨), 수정 후 통과를 둘 다 기록할 것. 마지막에 전체 e2e(`-run` 없이)도 한 번 돌려 회귀 없음을 확인.
  - 주의: 이 정찰 세션에서는 샌드박스 제약으로 `go vet`/`go test` 를 실제로 돌리지 못했다(명령 자체는 CI 에서 확인된 것). 구현자가 먼저 베이스라인을 돌려 녹색인지 볼 것.
- 위험과 피할 것:
  - `internal/auth`·`internal/api/server.go:PrincipalFor`·인증 미들웨어는 건드리지 말 것. 변경은 MCP 핸들러 안으로 한정.
  - GET/POST 경로의 인증·응답 형식을 바꾸지 말 것. DELETE 만 바꾼다.
  - 마이그레이션(`internal/database/migrations/0001_init.sql`)은 손대지 말 것 — `mcp_sessions` 스키마에 이미 `user_id`·`username`·`closed_at` 이 있다(0001_init.sql:267-278).
  - 세션이 없거나 타인의 것일 때 404/403 으로 구분해 응답하지 말 것 — 세션 ID 열거 오라클이 된다. 204 로 통일.
  - 감사 로그에 세션 ID 원문을 details 로 넘기지 말 것(운영자 지침). 필요하면 사용자 식별자만.
  - 웹 콘솔과 `hack/mcp.sh` 는 `DELETE /mcp` 를 호출하지 않는다(grep 확인: web 에는 vite 프록시 설정만, mcp.sh 에는 DELETE 없음) — 클라이언트 회귀 위험은 낮다.
- 차선 후보: GET /mcp 스트림이 `touchSession` 을 호출하지 않아 장기 SSE 세션의 `last_seen_at` 이 갱신되지 않는 문제(server.go:65-93; POST 경로만 server.go:273·292 에서 touch). 관리 콘솔의 "활성 세션" 집계(internal/api/admin.go:99)가 실제보다 적게 나온다. 작업량 S, 프로덕션 1파일.
