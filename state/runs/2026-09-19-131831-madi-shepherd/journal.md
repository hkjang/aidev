# PR 처리기 노트 2026-09-19-131831-madi-shepherd — madi PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: 기본 꺼짐·공개 경로는 RFC 9728 메타데이터뿐·계정 생성/승격 없음·Host 미사용·빈 교집합 명시 거부·설정 PUT 검증→저장 순서. go vet 및 두 새 테스트를 madi-review-pg 로 실제 실행해 PASS.
- 결함: knowledge_package.go:506 의 `TokenID != ""` 게이트가 OAuthSubject 를 빼먹어 SSO 클라이언트가 inspect_knowledge_package 로 원문 package 를 export 단계 없이 받는다(캠페인 madi 규칙 그대로). system_status_api.go:323 도 같은 누락.
- 캠페인 규칙 위반: mcpOAuthProvider 가 뮤텍스를 잡은 채 discovery 네트워크 호출(+WithoutCancel); 테스트에 request_id 비어있지 않음 단언과 조작 Host 케이스 없음.
- 못 본 것: 결함 1 을 실제 SSO 패키지 생성→inspect 로 재현하지는 않았고 코드 경로(128행은 넓혔으나 506행은 그대로)로 확정했다. 프런트(AdminPages/PersonalPages)는 diff 읽기만 했다.
- 권고: reject/fix, risk medium. 결함 1 수정 + 테스트 보강이면 재심사 승인 가능.
