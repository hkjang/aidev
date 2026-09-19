# PR 처리기 노트 2026-09-19-110948-madi-shepherd — madi PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 지적 전부 맞음: knowledge_package.go:506 게이트가 OAuthSubject 를 빠뜨려 SSO inspect 가 원문·prompt 를 그대로 돌려줬고, 새 테스트도 그 경로를 안 탔음. 틀린 지적 없음.
- 고친 방법: 게이트를 `p.TokenID != "" || p.OAuthSubject != ""` 로 packagePrincipal 과 동일 조건으로 맞춤(1줄). TestPostgresMCPOAuthTokens 에 SSO create→inspect→export 단계를 추가해 원문 비노출·export_required·export_count·consent 를 단언(수정 전 실패, 후 통과 확인).
- 커밋 de9c452, push 안 함.
- 확신 없는 곳: 저장소의 다른 `TokenID != ""` 게이트(impact-exception/approval/automation 등)는 모두 `ScopeRestricted` 와 함께 검사하고 SSO 원칙은 ScopeRestricted=true 로 만들어지므로(integrations_mcp_oauth.go:361) 이미 막힘 — 이 506행만 TokenID 단독이었음. MCP 가 디스패치하는 경로만 훑었고 그 외 REST 는 SSO 토큰이 401 이므로 손대지 않음. 전체 `go test -race ./internal/server` 는 기본 10분 타임아웃을 넘겨 CI 플래그(-timeout 45m)로 재실행했고 그 결과는 fix-summary 마지막 줄 참고.
