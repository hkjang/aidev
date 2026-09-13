## 2026-09-13
- 선택: Silent SSO (OIDC prompt=none) — auto_login 설정 뒤에 두고 무한 루프 3중 방지 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 캠페인 "silent-sso-2026-09" 목표를 구현했다. `auth.oidc.auto_login` 설정(기본 꺼짐, 관리 화면 체크박스 + `POSTRA_OIDC_AUTO_LOGIN`)을 추가하고, 로그인 페이지가 최상위 이동으로 `/ui/auth/oidc/start?prompt=none&return_to=…` 을 한 탭 세션에 한 번만 시도하도록 했다(sessionStorage 표시, 읽기 실패는 '이미 시도'로 간주, 로그아웃 시 억제 표시, 콜백은 login_required/interaction_required/consent_required 를 받으면 `/ui/login?sso=none` 으로 보내 주소에 표시). 서버는 auto_login 이 꺼져 있으면 `?prompt=none` 을 조용히 일반 로그인으로 바꾸고, `return_to` 는 `/` 로 시작하고 `//` 로 시작하지 않는 앱 내부 경로만 받아 gate → 로그인 폼 → 서명된 flow 쿠키 → 콜백까지 전달한다. 검증: 새 테스트 7개(SafeReturnTo·login_required 판별·BeginOIDC 다운그레이드·gate 딥링크·로그인 페이지 트리거 유무·콜백 거절 처리·로그아웃 마커) 추가, `go build ./... && go vet ./... && go test -race ./...` 전부 통과. 관리자 가이드 3.3 절 추가.
- 보류 아이디어: (1) OIDC 콜백에서 CompleteOIDC 실패 시 sso=error 마커로 로그인 페이지 리다이렉트해 주소를 정리(가치 2/위험 2/S); (2) `OIDCConfigured`·`OIDCAutoLoginEnabled` 가 로그인 렌더마다 SystemSettings 를 두 번 읽음 — 한 번에 읽는 헬퍼로 합치기(가치 2/위험 1/S); (3) `oidcStart` 에서 discovery 실패 시 502 를 로그인 페이지에 인라인으로 보여주는데 incident 기록도 남기기(가치 2/위험 1/S); (4) `internal/application/incidents.go` 가 gofmt 미적용 상태 — CI 에 gofmt 검사 추가(가치 2/위험 1/S).

