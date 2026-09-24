# PR 처리기 노트 2026-09-24-204712-appstore-shepherd — appstore PR #23
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: go test -race (auth/httpapi/mcp) ok, web 70/70 통과. 기본값 꺼짐·계정 자동생성 없음(subject만)·비활성 거부·마이그레이션 없음·주체 분기 전수(AuthMethod는 REST 전용)·비밀값 미노출까지 소스로 확인.
- 차단(security): mcp_oauth.go:88-103 mcpResource 가 Host/X-Forwarded-Proto 로 리소스 식별자를 만들고 그 값이 :256-263 aud 허용값이 된다. migrations/000001_init.sql:165 siteUrl="" 기본값 + 시스템 설정이 siteUrl 비우기를 막지 않아(admin_settings_handlers.go:619) 도달 가능. 캠페인 규칙 3, 6개 저장소 재현.
- 거절 사유 추가: access_token.go:56-78 이 뮤텍스를 잡은 채 oidc.NewProvider 디스커버리(15s) 호출 + WithoutCancel + 실패 미캐시 → Keycloak 장애 시 /mcp 직렬화(규칙 4). mcp_oauth_integration_test.go:31-33 은 CI 에 DSN 이 없어 SKIP 확인 — 거부 로그의 request_id/원문 cause 단언과 수락 경로 전체가 실행되지 않음(규칙 2).
- 못 본 것: 실제 Postgres·Keycloak 종단 동작(통합 테스트 skip), E2E(Playwright 미실행), PDF 본문.
- 권고 human: Host 폴백 제거는 한 줄이지만 "기본 설치에서 MCP SSO 가 켜지지 않는다" 는 제품 판단이 따라오므로 운영자가 정할 일.
