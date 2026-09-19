# PR 처리기 노트 2026-09-19-114208-git-ctx-shepherd — git-ctx PR #30
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: 기본값 꺼짐·계정 미생성·비활성 미부활·role 미반영·빈 scope 교집합 거부·서명/iss/exp/nbf/typ/cnf 거부가 모두 실제 Handler() 배선을 통과하는 테스트로 실행됨(go test app/auth/mcp 통과, gofmt·vet 통과, origin/main cf3b598 과 충돌 없음). KeyID 게이트 전수 치환 확인.
- 결함(reject/fix): 거부 로그에 request_id 없음(mcpoauth.go:326)과 테스트 미단언(mcpoauth_test.go:391); mcpResource 의 r.Host 유래 aud 폴백(mcpoauth.go:182-186)과 localhost 기본 식별자 미거부; resourceVerifier 가 v.mu 잡은 채 discovery 하고 ctx 취소 무시·실패 미캐시(resourcetoken.go:132-149).
- 못 본 것: 실제 Keycloak 26 상대 동작, PostgreSQL 빌드 모드에서의 테스트(로컬 sqlite 만 실행), -race.
- 근거: 캠페인 규칙 1·2·3·4 가 각각 명시적 거절 사유였고 네 곳 모두 코드에서 직접 확인함. 방향 자체는 옳고 고치면 되는 범위라 human 이 아니라 fix.
