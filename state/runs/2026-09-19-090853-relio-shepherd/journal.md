## 수리 노트

- 지적 4건 모두 맞았음(로그에 request_id 없음 / 로거 주입 테스트 없음 / 뮤텍스 잡고 네트워크 / 메타데이터 경로 무차별 200). 틀린 지적 없음.
- 고친 방법: `AuthenticateMCPToken` 이 `request_id`·`message`·`detail`(Refusal.Cause 원문) 로깅, ctx 에 ID 없으면 `request_id=missing`; DB 조회 두 곳을 `mcpSettings`/`mcpAccountLookup` 훅으로 분리해 실제 진입점을 버퍼 로거로 테스트. `mcpSigningKey` 를 `mcpDiscovery`+`mcpKey` 로 나누고 진행 중 round trip 을 `fetch`(done 채널, WithoutCancel 고루틴)로 공유, 실패 discovery 30초 음성 캐시. 메타데이터는 루트·리소스 path 만 200. 커밋 07c60fa.
- 로그 키는 캠페인 규칙대로 `request_id` 를 썼으나 이 저장소의 다른 로그 줄은 `requestId` 를 씀 — 일관성 vs 규칙 중 규칙을 택함(ADMIN_GUIDE 에 `request_id` 로 적음).
- 확신 없는 곳: 동시성 테스트가 시간 기반(delay 300ms, 허용 3×/4×)이라 아주 느린 CI 러너에서 흔들릴 수 있음(-race 6회 연속 통과는 확인). JWKS 미지 kid 스로틀은 원래대로 초당 1회 유지하되 진행 중 fetch 가 있으면 합류하도록 했음.

## 심사 노트

- 확인: 거절 사유 4건 모두 07c60fa 에서 해소됨. 변이 검증으로 새 테스트가 대상을 실제로 잡는지 확인 — request_id 제거→로그 테스트 실패, detail 제거→실패, discovery 를 잠금 안에서 호출자마다 수행하도록 되돌림→동시성 테스트 실패(8회 discovery, 2.5s), metadataPathServes 항상 true→경로 테스트 실패. go build/vet, -race 로 oidc·auth·server·admin 통과, 동시성 테스트 -race -count=5 안정.
- 확인: 기본 꺼짐(mcp.oauth.enabled=false, 꺼진 상태의 /mcp 401 은 예전과 동일한 `Bearer realm="Relio MCP"`, 메타데이터 404), 공개 경로는 RFC 9728 문서뿐, aud 허용값은 설정값(mcp.oauth.resource / system.service_url / mcp.oauth.audience)만이고 Host 미사용, 계정 자동 생성·재활성화 없음, 토큰 원문 미로깅(테스트 단언), 마이그레이션은 INSERT … ON CONFLICT DO NOTHING 만, KeyID 기반 게이트(REST 채널·CSRF·meta 채널)는 모두 OAuth 주체를 포함.
- 못 본 것: 실제 Keycloak 과의 종단 흐름(가짜 IdP 로만 검증), 프런트 SSO 카드의 저장 동작(빌드만 통과), 실제 DB 를 거치는 MCPOAuthSettings/mcpAccount 쿼리(훅으로 우회됨).
- 참고: origin/main 의 문서화되지 않은 REST OIDC 액세스 토큰 경로(Authenticate→OIDCValidator, 전체 권한)가 제거됨 — 커밋·문서에 명시된 의도적 축소이고 보안상 더 좁아지므로 승인 사유에 포함. 로그 키 `request_id` 는 저장소 관례 `requestId` 와 다르나 캠페인 규칙을 따른 것이라 스타일 문제로만 둠.
- 권고: merge, risk medium(인증 경로).
