# PR 처리기 노트 2026-09-20-014424-vibe-coders-shepherd — vibe-coders PR #20
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 세 지적 모두 맞았다: Host 폴백은 env 경로(SSO_KEYCLOAK_REDIRECT_URI 미검증)에서 실제 도달했고 aud 대조값이 호출자 손에 있었음; 거부 로그에 request_id 없음; 두 캐시 함수가 락 안에서 네트워크 호출.
- 고친 방법: `mcpOAuthState()` 에서 요청 인자 자체를 제거해 Host 가 들어올 길을 없앰(reason 으로 비활성), 거부 로그에 `request_id=traceIDFromRequest(r)`, 캐시는 락 안 조회/저장·락 밖 fetch·stale 즉시 반환+백그라운드 갱신·30초 negative cache. 각 항목에 red→green 테스트(커밋 943feab).
- 확신 없는 곳: 미지 kid negative cache 를 "30초 안에 받은 JWKS 에 없으면 refetch 안 함"으로 구현해 Keycloak 키 회전 직후 최대 30초 새 kid 가 거부될 수 있음(관행적 min-refresh-interval 이지만 기존 동작보다 보수적). 백그라운드 갱신은 `context.Background()`+8초 타임아웃이라 요청 ctx 와 무관.

## 심사 노트
- 확인: 세 거절 사유 모두 소스에서 닫힌 것을 봄 — mcpOAuthState() 무인자·aud 허용값이 설정에서만 나옴, 거부 로그 request_id=X-Request-ID(응답 헤더와 동일 단언), 캐시는 락 밖 fetch·stale 즉시 반환·30초 negative cache.
- 실행: 새 mcp_oauth 테스트 2개를 943feab^ 의 mcp_oauth.go 에 대고 돌려 둘 다 실패(red) 확인; 고친 트리에서 `go test -race` MCPOAuth|Keycloak 및 `go test ./internal/...` 전부 통과, web tsc·SsoTab vitest 26개·run-tool node:test 통과.
- 못 본 것: keycloak_cache_test 는 옛 keycloak.go 와 컴파일이 안 돼 red 를 실행으로 못 봄(락 보유 중 fetch 면 1초 내 반환이 불가능하다는 구조로 판단). 실제 Keycloak 키 회전 30초 창은 운영에서만 보임.
- 권고 merge: 결함 없음. 참고로 92bc502(run-tool.mjs) 는 범위 밖 빌드 스크립트 변경이나 작고 검증됨; 미지 kid 동시 fetch 에 singleflight 없음(기존보다 개선).
