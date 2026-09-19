# PR 처리기 노트 2026-09-19-113709-releasedock-shepherd — releasedock PR #18
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: 마이그레이션 025 는 ADD COLUMN IF NOT EXISTS 만, 기본 꺼짐. 새 테스트 11개를 임시 Postgres 16 에서 실행해 전부 통과했고, 실제 핸들러 체인·가짜 IdP 실키·버퍼 로거를 통과하는 진짜 테스트임. 계정 생성 없음, ViaAPIKey 게이트 3곳 모두 적용, 빈 scope 교집합은 거부, REST 는 토큰 거부.
- 결함 1 (재현함): mcp.oauth.resource·publicUrl 이 비면 mcpResource 가 X-Forwarded-Host 로 aud 허용값을 만들어, 다른 리소스용 토큰이 /mcp 를 200 으로 염 (mcpoauth.go:106-110). validMCPOrigin 은 withAuth 뒤에서 r.Host 만 봐서 못 막음.
- 결함 2: 거부 로그(server.go:349)에 request_id 없음 — X-Request-ID 는 w.Header() 에만 있음. 테스트도 request_id 를 단언하지 않음.
- 결함 3: mcpSigningKey 가 cache.mu 를 잡은 채 discovery+JWKS 를 호출하고, 빈 캐시에서 실패하면 재시도 제한이 없음 (mcpoauth.go:663-684).
- 못 본 것: 실제 Keycloak 과의 연동, 프론트 SettingsPage 렌더링. 권고: reject/fix — 방향은 맞고 세 곳만 고치면 됨; 캠페인 규칙 1·2·3·4 위반이라 사람에게 넘길 문제가 아니라 코드 수정 사안.
