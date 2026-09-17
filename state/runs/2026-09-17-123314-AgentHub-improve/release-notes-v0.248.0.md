## 주요 변경

- MCP OAuth 2.1 리소스 서버: 기존 API 키 체계를 그대로 둔 채 Keycloak 액세스 토큰으로도 `/mcp` 에 접속할 수 있습니다 — 보호 리소스 메타데이터, `resource_metadata` 를 담은 401 도전, 토큰 검증(RS/ES/PS 서명·issuer·aud/azp·exp/nbf·typ·cnf), `sub` 기반 계정 매핑, 콘솔 **MCP SSO** 탭, `/auth/methods` 의 `mcpSso` 안내, 관리자·사용자 가이드 갱신 (커밋 ff2dd99).

AgentHub 제어 이미지: `agenthub:v0.248.0` — 이번 릴리즈에 게시됩니다.

런타임 이미지·SBOM·attestation·오프라인 번들 목록은 릴리즈 워크플로가 태그 푸시 시 `runtime-images.json` 카탈로그에서 자동으로 생성합니다.
