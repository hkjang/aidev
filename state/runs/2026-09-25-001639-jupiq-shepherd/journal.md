# PR 처리기 노트 2026-09-25-001639-jupiq-shepherd — jupiq PR #17
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: 캐시 로직 단위 테스트는 돌연변이 4종(verifyTLS 키, 실패 캐싱, TTL 경계, forget no-op)을 모두 잡고 -race 도 깨끗. go-oidc v3.16 소스로 JWKS 가 Provider 에 캐시되고 WithoutCancel 이라 resolve 의 cancel 에 안 죽음을 확인. SSRF·권한·비밀값·마이그레이션은 문제 없음.
- 결함: 배선 무검증. NewService 에서 providers 초기화를 빼도, OIDCLogin 을 예전 oidc.NewProvider 로 되돌려도 `go test ./internal/auth/ ./internal/api/` 가 둘 다 ok. nil 캐시 폴백이 전자를 런타임에서도 조용히 퇴화시킨다.
- 결함: TestOIDCLoginTokensAreDistinct 는 f3834e0 이전 코드(에러 무시)에서도 통과 — 난수 실패 거부 동작에 테스트가 없다.
- 못 본 것: DSN 이 없어 통합 테스트 미실시, 실제 Keycloak E2E 미검증, ADMIN_GUIDE.pdf 바이너리 내용은 대조하지 않음.
- 권고 reject/fix, risk medium: 방향은 맞고 되돌리기도 쉬우나, 문서가 운영자에게 사실로 적은 동작이 실행으로 증명되지 않았다. blocking 없음.
