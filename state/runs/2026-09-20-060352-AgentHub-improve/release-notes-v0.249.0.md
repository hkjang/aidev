## 주요 변경

- `mcp.tool_call` 감사 항목의 `details` 에 자격증명 종류 `auth`(`key` | `oauth`)를 항상 남기고, Keycloak 액세스 토큰으로 들어온 호출은 OAuth 클라이언트(`azp`)를 `client` 로 함께 남깁니다. 토큰·API 키·`sub` 원문은 싣지 않으며, 거절·실패 호출의 감사 동작은 그대로입니다. (커밋 be4b357, PR #30)
- 관리자 가이드 §4.5 에 위 감사 필드 설명을 추가했습니다.

AgentHub 제어 이미지: `agenthub:v0.249.0` — 이번 릴리즈에 게시됩니다. 런타임 base 이미지 소스는 바뀌지 않아 `BASE_VERSION`(0.26.0) 은 그대로입니다.

런타임 이미지 표·오프라인 번들·SBOM·attestation 은 태그 푸시 시 `release.yaml` 워크플로가 `scripts/release-catalog-images.sh notes` 로 생성해 GitHub Release 본문에 덧붙입니다.
