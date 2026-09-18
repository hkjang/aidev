# fix-summary (commit c075f4f)

- 문제: `mcpOAuthScopes` 가 토큰 범위와 관리자 천장의 교집합이 비면 `[]` 를 돌려주고, `/mcp` 는 빈 범위를 "세션 인증(무제한)"으로 읽어 mcp·write 검사를 모두 건너뛰었다 — 천장 `read mcp` 에 `write` 만 실은 토큰이 쓰기 도구까지 여는 권한 상승. 라우터 통과 테스트로 재현 확인(토큰이 범위 검사를 지나 계정 조회까지 감).
- 고침: `mcpOAuthScopes` 가 `([]string, *mcpOAuthRefusal)` 를 돌려주고 교집합이 비면 `403 insufficient_scope` 거절을 반환, `verifyMCPAccessToken` 이 그 거절을 그대로 올려 `refuseMCPToken` 이 403 + 로그 cause 로 내보낸다. 비어 있지 않은 교집합·어휘 밖 scope 만 실은 토큰의 동작은 그대로.
- 테스트: `TestMCPOAuthScopesAreCappedByAdmin` 의 `{"read mcp","write",""}` 케이스를 거절 기대로 바꾸고 `{"mcp","read"}`·`{"read mcp","openid write"}` 추가; 새 `TestTokenOutsideAdminScopeCeilingIsRefused` 가 실제 라우터·가짜 IdP 로 403 insufficient_scope 와 로그를 단언.
- 문서: docs/API_AND_MCP.md 에 "교집합이 비면 403 insufficient_scope" 한 구절 추가.
- 검증: `gofmt -l` 깨끗, `go vet ./...` 통과, `go test ./...` 전부 통과.
