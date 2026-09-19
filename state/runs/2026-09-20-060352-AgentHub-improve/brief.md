# 과제서 — 2026-09-20 AgentHub

- 과제: mcp.tool_call 감사 항목에 자격증명 종류(auth=key|oauth)와 OAuth 클라이언트(client=azp)를 남긴다 (가치 3 / 위험 1 / 작업량 S)

- 왜: v0.248.0 부터 /mcp 는 API 키와 Keycloak 액세스 토큰 두 가지로 열리는데, 도구 호출 감사 항목(`internal/api/mcp.go:347`)은 details 에 `{"tool": name}` 만 남겨 운영자가 트레일에서 "이 호출이 키로 왔는지 SSO 로 왔는지, 어느 MCP 클라이언트(azp)가 불렀는지" 를 구분할 수 없다. `oauthPrincipal` 은 이미 `claims.AZP` 를 읽어 대상 검사에 쓰고 나서 버리므로, 그 값을 하나 더 돌려주고 details 에 싣기만 하면 된다 — 실제 저장·내보내기 경로(AuditTrail, CSV export, 콘솔 AdminOperations 의 `details: unknown` 렌더)는 모두 details 를 JSON 그대로 다루므로 다른 경로를 넓힐 필요가 없다(확인: `grep -rn mcp.tool_call` 이 mcp.go 한 곳뿐).

- 수용 기준:
  1) API 키로 `tools/call`(예: `agenthub_list_agents`) 을 성공시키면 audit_events 의 `mcp.tool_call` 항목 details 가 `{"tool":"agenthub_list_agents","auth":"key"}` 를 담는다(`client` 키 없음).
  2) SSO 토큰(azp=`claude-mcp`, 관리자가 허용 대상에 올린 클라이언트)으로 같은 호출을 성공시키면 details 가 `{"tool":…,"auth":"oauth","client":"claude-mcp"}` 를 담는다. azp 가 없는 토큰(aud 에 리소스 식별자만)이면 `auth:"oauth"` 만 남고 `client` 키는 비어 있는 값으로 실리지 않는다.
  3) 테스트가 증명할 것: `server.Handler()` 전체 라우터 + 실제 Postgres(mcpoauth_live_test.go 의 `mcpSSODeployment`)로 두 자격증명 각각 호출한 뒤 `db.AuditTrail(ctx, store.AuditFilter{Action: "mcp.tool_call", …})` 로 **실제 audit_events 를 읽어** 위 details 를 확인한다. 수정 전 코드에 대고 돌리면 `auth` 키 부재로 실패해야 한다(먼저 확인하고 과제 노트에 적을 것). 손으로 만든 대역 스토어나 소스 문자열 검사는 증거가 아니다.
  4) 기존 `TestMCPSSO*` 8개, `internal/api/mcp_test.go`, `TestEveryStateChangingRouteAudits`(auditcoverage_test.go) 가 그대로 통과한다. 거절된 호출·실패한 호출의 감사 동작은 바꾸지 않는다(지금도 성공만 파일함).

- 건드릴 파일:
  - `internal/api/mcpoauth.go:oauthPrincipal` — 반환값에 클라이언트 ID(`claims.AZP`, TrimSpace)를 더한다. 방법은 둘 중 하나: (a) 반환 튜플에 `string` 하나 추가, (b) 작은 구조체 `mcpPrincipal{user, scopes, auth, client}` 를 두고 키 경로도 같은 구조체로 만든다. (b)가 mcp.go 의 두 분기를 한 값으로 모으므로 권장. 호출자는 `mcp.go:56` 과 `mcpoauth_test.go:87`(`TestAnSSOPrincipalGetsTheAdministratorsScopesAndNoMore` 가 oauthPrincipal 을 직접 부름 — 시그니처 바뀌면 같이 고침) 둘뿐(`grep -n oauthPrincipal internal/api`).
  - `internal/api/mcp.go:mcp` — 분기에서 `auth`("key"/"oauth") 와 `client` 를 잡고, `:347` 의 `s.store.Audit(... map[string]any{"tool": params.Name})` 에 `auth` 를 항상, `client` 는 비어 있지 않을 때만 더한다. details 를 만드는 작은 순수 함수(예: `mcpCallDetails(tool, auth, client string) map[string]any`)로 빼면 DB 없는 단위 테스트가 하나 붙는다.
  - `internal/api/mcpoauth_live_test.go` — 새 테스트 `TestTheTrailSaysWhichDoorAToolCallCameThrough`(이름은 자유). `callMCP` 는 `mcpListTools` 본문이 고정돼 있으므로 본문을 받는 변형(`callMCPWith(handler, bearer, body)`)을 하나 더 두거나 `callMCP` 를 그 위에 얹는다. tools/call 본문(legacy 경로, MCP-Protocol-Version 헤더 없이): `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"agenthub_list_agents","arguments":{}}}`. 키 경로는 `db.CreateAPIKey(ctx, member.ID, "trail", []string{"mcp:read"}, nil)` (internal/store/secrets.go:166, 두 번째 반환값이 평문 토큰) 로 만든다. SSO 경로는 `provider.accessToken(t, "agenthub-mcp-sso-test:active", "account", map[string]any{"azp": "claude-mcp"})` — `TestMCPSSOOpensForAnAccountThePlatformKnows` 의 viaClient 와 같은 모양(설정 `Audience: "claude-mcp"`). `agenthub_list_agents` 가 `mcp:read` 범위에 있는지는 `mcp.go:200` 부근 도구표(`readable`)에서 확인.
  - `internal/api/mcp_test.go` 또는 새 파일 — details 함수 단위 테스트(빈 client 는 키를 만들지 않음).
  - `docs/ADMIN_GUIDE.md` §4.5(MCP SSO, 243행 부근 설정표 뒤) 에 한 문장: 트레일의 `mcp.tool_call` details 에 `auth` 와 `client` 가 남아 어느 클라이언트가 불렀는지 볼 수 있다. PDF 재생성은 이 회차의 관례(`md2pdf.mjs`, 이전 커밋 메시지 참고)를 따르되, 문장 하나라면 md 만 고치고 PDF 는 건드리지 않아도 됨 — 어느 쪽이든 커밋 메시지에 적을 것.

- 검증 명령:
  - `go build ./... && go vet ./...`
  - DB 없이(CI 와 동일): `go test -race ./cmd/... ./internal/...`
  - DB 있이(live): `AGENTHUB_TEST_DSN=<postgres dsn> AGENTHUB_ENCRYPTION_KEY=<32바이트 키> go test -race -p 1 -count=1 ./internal/api/ -run 'MCPSSO|Trail|Door'` — 이전 회차들은 Postgres 16 컨테이너(`docker run -d -e POSTGRES_PASSWORD=… -p 5432:5432 postgres:16`)로 돌렸음. 새 테스트는 수정 전 코드에서 실패하는 것을 먼저 확인.
  - 웹은 건드리지 않으므로 `npm run lint`·`build` 는 선택. `scripts/release-catalog-images.sh check-versions` 는 관례상 실행.
  - cmd/runtime-proxy·internal/dlp 를 건드리지 않으므로 BASE_VERSION 상향 불필요(현재 main 의 BASE_VERSION 은 0.26.0 — 보류 목록의 "0.27.0 으로 올렸다" 는 미병합 브랜치 auto/2026-09-18-1403 의 이야기).

- 위험과 피할 것:
  - details 에 토큰·키 원문·해시·sub 를 넣지 말 것(감사 details 는 그대로 내보내진다 — 2026-09-09 교훈). azp 는 클라이언트 ID(설정에 적는 공개 식별자)라 괜찮지만, 그것도 TrimSpace 만 하고 길이가 200 을 넘으면 자를 것(IdP 가 주는 값이라 신뢰하지 말 것).
  - `authentication` 미들웨어·REST 경로·전송 방식은 건드리지 않는다. `oauthPrincipal` 의 거절 순서·메시지를 바꾸지 않는다(live 테스트 8개가 메시지를 고정함).
  - 키 경로에서 API 키 이름/ID 까지 남기고 싶어질 수 있으나 `UserAndScopesByAPIKey` 시그니처를 넓혀야 하므로 이번 범위 밖 — 차선 후보로 남김.
  - 두 경로가 같은 값을 다르게 읽는 일이 없도록, details 를 만드는 함수는 하나만 두고 두 분기가 그것을 쓰게 할 것(운영자 지침: 읽는 경로가 여럿이면 end-to-end 로 같게).
  - live 테스트 병렬 실행 시 store 패키지가 흔들리는 기존 순서 문제가 있으니 `-p 1` 로 돌릴 것.

- 차선 후보: langflow-e2e.mjs 의 sessionGateway 복원 수정 — `web/scripts/langflow-e2e.mjs:169` 가 `GET /api/v1/admin/settings/sessionGateway` 를 읽지만 그 라우트는 PUT 전용(405)이라 `.body?.value ?? {}` 가 빈 객체가 되고, finally 의 `put(... { value: gateway })` 가 그 배포의 게이트웨이 설정을 빈 값으로 덮어쓴다(2026-09-20 소스 재확인, 그대로임). `guide-shots.mjs:108-115` 처럼 `GET /api/v1/admin/settings` 전체에서 `sessionGateway` 키를 꺼내고, 없으면 복원 자체를 건너뛰도록 고친다. Ready 상태의 Langflow 런타임이 있어야만 발동하는 분기라 이 환경에서는 end-to-end 실행이 불가(미확인) — 그 점을 노트에 적고, 최소한 스크립트가 파싱·lint 를 통과하고 405 응답으로 빈 값을 복원하지 않는 것을 node 단위 테스트(`web/scripts/*.test.mjs` 관례, `node --test`)로 고정할 것.
