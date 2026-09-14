Keycloak에 로그인한 사용자를 로그인 화면 없이 들여보내는 자동 로그인
(silent SSO)을 릴리스합니다. OIDC 설정에 새 항목이 추가되고 기본값은
꺼짐이어서 기존 동작은 그대로이므로 minor를 올립니다.

- OIDC 설정 `auto_login`: 세션이 없는 방문자에게 prompt=none으로 제공자에
  한 번 묻고, login_required면 `/login?sso=none`으로 조용히 돌아옴
- 탭 세션당 한 번 시도, 스스로 로그아웃하면 억제, 콜백 오류는
  `/login?sso=error`로 표시해 제공자와 앱 사이를 오가지 않음
- `GET /api/v1/auth/oidc/login`을 IP별 분당 120회로 제한하고 초과하면
  제공자를 부르지 않고 `/login?sso=limited`로 돌려보냄
- 로그인 리미터가 성공 뒤에도 IP 창을 유지하는 의도를 주석과 테스트로 남김
- 관리자 가이드 4.4·6.4·7.3절과 PDF, OpenAPI 설명 갱신

