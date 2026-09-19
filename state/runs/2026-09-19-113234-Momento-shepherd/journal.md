# PR 처리기 노트 2026-09-19-113234-Momento-shepherd — Momento PR #14
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 심사 노트
- 확인: go build/vet, 로컬 Postgres 16 컨테이너로 MCP/SSO 통합 테스트 전부 통과. 마이그레이션 추가만·기본 꺼짐·공개 경로는 RFC 9728 두 개뿐·계정 생성/role 승격 없음·세 게이트 모두 Programmatic() 갱신·merge-tree 로 main 과 충돌 없음.
- 결함 1(재현): resource·public_url 이 모두 빈 기본 설치에서 mcpResource 가 r.Host 로 aud 허용값을 만들어, 다른 앱용 토큰이 Host 헤더만으로 통과(임시 테스트로 200 확인 후 삭제).
- 결함 2: oauthProvider 가 뮤텍스를 잡은 채 discovery, ctx 취소 무시, 실패 미캐시. 결함 3: 거부 로그 테스트 없음(테스트 로거가 Stderr/LevelError), request_id 없음(미들웨어도 없음).
- 못 본 것: 실제 Keycloak 에 대고는 돌리지 않음(가짜 IdP 만), web 빌드/npm test 미실행.
- 권고: reject/fix, risk high — 인증 경계 우회라 사람 판단 사항은 아니고 코드 수정으로 해결 가능.
