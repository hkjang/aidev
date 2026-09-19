# PR 처리기 노트 2026-09-20-004625-vibe-coders-shepherd — vibe-coders PR #20
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: 기본 꺼짐, 새 공개 경로는 RFC 9728 메타데이터뿐(비활성 시 404), 계정 생성·소생·토큰 role 반영 없음, 빈 스코프 교집합 거부, 재진입 주체는 ctx 로만 전달. go test -run 'OAuth|MCP' 통과, go vet 클린. 테스트는 가짜 IdP·실제 HTTP·실제 파이프라인을 통과하는 진짜 검증.
- 거절 1: mcp_oauth.go:181 resource·RedirectURI 둘 다 비면 requestOrigin(Host/X-Forwarded-Host) 로 aud 허용값을 만든다 — env 경로는 RedirectURI 를 검증하지 않아 도달 가능. 캠페인 규칙 위반, 문서도 이를 명시.
- 거절 2: mcp_oauth.go:311 거부 로그에 request_id 없음(traceIDFromRequest 존재), 테스트도 단언 안 함.
- 거절 3: keycloak.go 의 discovery/JWKS fetch 가 뮤텍스 안에서 실행(기존 코드지만 이 PR 이 /mcp 매 요청 경로로 올림).
- 못 본 것: web 테스트 미실행. 권고 fix(medium) — 세 항목 모두 기계적으로 고칠 수 있고 사람 판단이 필요한 정책 문제는 아님.
