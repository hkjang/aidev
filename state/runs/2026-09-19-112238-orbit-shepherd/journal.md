# PR 처리기 노트 2026-09-19-112238-orbit-shepherd — orbit PR #6
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: 전체 diff 를 읽고 go build/vet/test 와 진짜 Postgres 로 TestDB* 6건을 실행해 모두 통과. 기본값 꺼짐, 마이그레이션 추가만, aud 에 Host 미사용, 빈 scope 교집합 거부, 계정 미생성·미부활, 공개 경로는 RFC 9728 둘뿐, 비밀값 노출 없음, external() 게이트 전부 갱신을 파일에서 확인.
- 결함(캠페인 규칙 위반): auth.go:172 거부 로그에 request_id 없음(chi RequestID 가 이미 있음); 거부 로그를 고정하는 버퍼 핸들러 테스트 없음; mcpoauth.go:389 oauthProvider 가 뮤텍스를 잡은 채 discovery 를 하고 실패를 캐시하지 않으며 WithoutCancel 로 요청 취소를 무시(go-oidc 3.20 은 키셋에 Background 를 따로 씀).
- 결함(불일치): metadataURL() 이 리소스 경로를 그대로 붙이지만 라우터는 /mcp 만 서빙 — 사용자 지정 리소스 경로면 SPA 로 떨어지는 주소를 알린다.
- 못 본 것: 실제 Keycloak 26 과의 왕복은 가짜 IdP 로만 확인; 웹 UI 는 diff 로만 봄(빌드/실행 안 함).
- 권고: reject/fix, risk medium. 인증 경로지만 되돌리기 쉽고 위 사유만 고치면 재심사 가능.
