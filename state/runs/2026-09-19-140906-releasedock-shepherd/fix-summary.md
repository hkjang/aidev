# 수리 요약 — PR #18 (commit ff47e4e)

1. **Host 스푸핑 (mcpResource)**: `requestOrigin()` 폴백을 삭제하고 `mcpOAuthConfig(From)` 에서 `*http.Request` 인자 자체를 제거했다. mcp.oauth.resource 와 publicUrl 이 모두 비면 `Active=false`, `InactiveWhy="no resource identifier…"` 로 /mcp 는 챌린지 없는 401, 메타데이터는 404. 추가로 `putGeneralSettings` 는 mcp_oauth_enabled=TRUE 이고 mcp_oauth_resource='' 인 동안 publicUrl 비우기를 400 으로 거절한다.
2. **request_id 로그 (withAuth)**: `MCP OAuth token rejected` Warn 에 `w.Header().Get("X-Request-ID")` 를 `request_id` 로 붙인다(비어 있으면 생략). 테스트는 각 거절 케이스와 inactive 케이스에서 응답의 X-Request-ID 로 로그 줄을 찾아 그 줄이 실패한 검사 텍스트를 담고 있는지 단언한다(이전 케이스의 로그로 통과하는 일이 없도록).
3. **키 캐시 (mcpSigningKey)**: 뮤텍스는 필드만 지키고 네트워크는 잠금 밖에서(요청 ctx 로) 수행; 같은 fetch 가 필요한 요청은 `inflight` 채널로 single-flight, 캐시된 kid 는 fetch 중에도 즉시 검증. `attempted`/`lastErr` 를 기록해 실패한 fetch 도 mcpOAuthKeyRetry(10s) 동안 재시도하지 않는다(콜드 캐시 실패는 캐시된 503, 리프레시 실패는 이전 키 유지).
4. 새 테스트 `TestTheResourceIdentifierNeverComesFromTheRequest`, `TestKeyFetchesNeitherBlockOtherTokensNorHammerTheIssuer` 를 원본 코드에 대해 돌려 네 가지 지적 모두 그대로 실패함을 확인(200 통과, request_id 없음, 캐시된 kid 가 fetch 대기, 실패 재시도 무제한). 수정 후 `go test ./...`(TEST_POSTGRES_DSN 설정, 임시 postgres:16 컨테이너) 전부 통과, 새 테스트 `-race -count=5` 통과, gofmt/vet 깨끗.
5. web/·runner/ 는 손대지 않아 그 CI 단계는 영향 없음. docs/MCP.md 는 요청 폴백을 언급하지 않아 수정 불필요.
