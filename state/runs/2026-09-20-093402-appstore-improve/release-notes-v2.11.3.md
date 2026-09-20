AppStore v2.11.3

자동 로그인(Silent SSO)이 켜진 설치에서 Keycloak 세션이 없어 로그인 화면으로 돌아왔을 때, 왜 자동으로 로그인되지 않았는지 아무 설명이 없던 문제를 고친 patch 릴리스입니다.

- Keycloak 세션이 없으면 콜백이 `/login?sso=none(&returnTo=…)` 으로 보내는데(`internal/httpapi/auth_handlers.go`), `LoginPage` 는 `returnTo` 만 읽어 사용자는 자동 시도가 있었다는 것도, 왜 로그인 화면에 왔는지도 알 수 없었습니다. 관리자 가이드 장애 대응 표만 "주소창의 `sso=none` 을 보라" 고 안내하고 있었습니다.
- `web/src/pages/auth-pages.tsx` 의 `LoginPage` 가 주소의 `sso=none` 을 읽어 SSO 가 켜진 설치에서만 SSO 버튼 바로 위에 `role="status"` 안내 한 줄("회사 계정 세션이 없어 자동으로 로그인하지 않았습니다. 아래 버튼으로 로그인하세요.")을 그립니다. SSO 링크 주소와 bootstrap 로그인 뒤 `returnTo` 이동은 그대로이며, 서버와 `internal/auth` 는 바뀌지 않았습니다.
- 사용자 가이드 2.2 에 한 문단, 관리자 가이드 장애 대응 표 "확인" 칸에 한 구절을 더하고 두 가이드 PDF 를 다시 구웠습니다.
- Schema 변경이 없어 기존 설치는 image만 교체하면 됩니다.
