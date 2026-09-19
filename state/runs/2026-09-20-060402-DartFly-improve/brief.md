# 과제서 — 2026-09-20 DartFly

- 과제: 이미 로그인한 사람이 `/login` 을 열면 폼 대신 return_to(없으면 `/`)로 보내기 — 서버 쪽 302 (가치 3 / 위험 1 / 작업량 S)

- 왜: 지금 `GET /login` 은 `mux.Handle("GET /", webui.Handler())` 가 정적 HTML 로 그대로 내주고, `internal/webui/js/login.js` 는 관리자가 auto_login 을 켠 설치에서만 `/api/v1/auth/me` 로 세션 유무를 본다(`shouldAttemptSilentSso` 안쪽). 그래서 세션이 살아 있는 사람이 북마크·뒤로가기·이메일 링크로 `/login` 을 열면 빈 로그인 폼이 나오고, 다시 로그인하면 세션이 새로 발급된다(그리고 사용자는 "로그아웃됐나?" 하고 헷갈린다). 서버가 세션 쿠키를 확인해 302 로 보내면 JS 없이도, 깜빡임 없이, auto_login 설정과 무관하게 바로 제자리로 간다.

- 수용 기준:
  1) 유효한 `dartfly_session` 쿠키를 들고 `GET /login` 을 부르면 302 이고 `Location` 은 `/` 이다. `GET /login?return_to=%2Fadmin%2Fusers%3Fq%3D1` 이면 `Location` 은 `/admin/users?q=1` 이다.
  2) return_to 가 `safeReturnTo` 를 통과하지 못하는 값(`https://evil.example`, `//evil.example`, `\r\n` 포함)이면 `/` 로 보낸다(밖으로 내보내는 발판이 되지 않는다).
  3) 쿠키가 없거나 위조·만료된 토큰이면 예전과 똑같이 200 에 `pages/login.html` 본문(`id="login-form"` 포함)이 나온다. `options.Sessions == nil`(메타 DB 없는 기동)에서도 200 HTML 이다.
  4) 테스트가 증명할 것: 위 1)~3) 을 `New(logger, options)` 로 만든 **실제 라우터**에 httptest 로 요청해 확인한다(핸들러 함수를 직접 부르지 말 것 — `GET /` 와 `GET /login` 패턴 우선순위가 실제 mux 에서 맞는지가 핵심). 세션 쿠키는 `options.Sessions.Create(user)` 로 실제 발급한 토큰을 쓴다(FakeSession 금지). `TestEveryRouteHasAuthorizationGate`·`TestPublicRoutesAreAllRegistered` 가 계속 통과한다.
  5) `test/smoke/pages.py` 는 로그인 뒤 `assets.go` 라우팅 표의 모든 페이지(`/login` 포함)를 방문한다 — `/login` 방문이 302→`/` 로 이어져도 `page.goto(..., wait_until="networkidle")` 는 성공하고 `>=400` 응답이 없어야 한다(스모크 통과).

- 건드릴 파일:
  - `internal/server/http.go:128` 근처 — `mux.Handle("GET /", webui.Handler())` 아래에 `mux.HandleFunc("GET /login", loginPageHandler(options))` 를 등록. Go 1.22+ ServeMux 는 더 구체적인 패턴이 이기므로 `GET /` 보다 우선한다(확인: go.mod 의 go 버전은 1.25, CI 도 1.25.x).
  - `internal/server/http.go` — 새 함수 `loginPageHandler(options Options) http.HandlerFunc`: `authenticatedSession(r, options)`(http.go:2048, 쿠키→`Sessions.Get`→`applyAuthority` 로 비활성/강등 계정까지 걸러 줌)가 ok 이면 `return_to` 쿼리를 `safeReturnTo`(http.go:1820, bool 반환)로 검사해 통과하면 그 값, 아니면 `/` 로 `http.Redirect(..., http.StatusFound)`; 아니면 `webui.Handler().ServeHTTP(w, r)` 로 넘긴다(webui.Handler() 를 매 요청 만들지 말고 New 안에서 한 번 만들어 클로저로 잡을 것).
  - `internal/server/routeguard_test.go:23` `publicRoutes` — `"GET /login": "로그인 화면 자체(세션이 있으면 return_to 로 보내고, 없으면 폼)"` 추가. 이 게이트 테스트는 `HandleFunc` 호출만 훑으므로 새 라우트는 반드시 `HandleFunc` 로 등록하고 여기 사유를 적어야 둘 다(`TestEveryRouteHasAuthorizationGate`, `TestPublicRoutesAreAllRegistered`) 통과한다.
  - `internal/server/silentsso_test.go` 또는 새 `internal/server/loginpage_test.go` — 위 수용 기준 1)~3) 테스트. 기존 헬퍼 `ssoOptions(t, false)`, `ssoGet(handler, path, cookies...)`, `cookieNamed` 를 그대로 쓸 수 있다(silentsso_test.go:50~85). SSO 가 없는 설치도 같아야 하므로 `Options{Sessions: auth.NewSessions(...)}` 정도의 최소 옵션으로도 한 케이스 두면 좋다(sessions_test.go 의 옵션 만드는 법 참고).
  - `internal/webui/js/login.js` — **바꾸지 않아도 됨**. auto_login 경로의 `hasSession()` 은 서버 302 뒤에는 도달하지 않으므로 남겨 둬도 해가 없다. 굳이 손대면 "서버가 먼저 걸러 준다"는 한 줄 주석만.
  - `docs/ADMIN_GUIDE.md` — 자동 로그인/return_to 항목 근처에 "세션이 있는 상태로 /login 을 열면 return_to(없으면 /)로 바로 간다" 한 줄.

- 검증 명령:
  - `gofmt -l . && go vet ./...`
  - `go test -race ./internal/server/ -run 'TestLoginPage|TestSilentSso|TestSso|TestEveryRouteHasAuthorizationGate|TestPublicRoutesAreAllRegistered'`
  - `go test -race ./...` (JS 테스트 `test/js/*.test.mjs` 도 `internal/webui/jstest_test.go` 가 node 로 함께 돌림; node v22 있음)
  - `bash test/smoke/run.sh` — 실제 바이너리+MariaDB 컨테이너+Chromium(playwright) 31페이지. 이전 회차에서 MariaDB 컨테이너가 "TLS certificate is not yet valid" 로 한 번 실패한 적 있음 → 재시도.
  - 같은 스모크 서버에서 손으로: `curl -s -c jar -X POST $BASE/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":..,"password":..}'` 로 세션 받은 뒤 `curl -s -b jar -o /dev/null -w '%{http_code} %{redirect_url}\n' "$BASE/login?return_to=%2Fsaved"` → `302 …/saved`, 쿠키 없이 → `200`, `return_to=https://evil.example` → `302 …/`.

- 위험과 피할 것:
  - `authenticatedSession` 은 `applyAuthority` 로 계정 상태를 다시 확인한다 — 비활성화된 계정의 살아 있는 쿠키는 ok=false 라 폼이 나온다. 이것이 의도이며, 우회하려고 `Sessions.Get` 만 직접 부르지 말 것(session 보호 경로).
  - 로그아웃(http.go:2004 근처)은 쿠키를 MaxAge -1 로 지우고 `/login` 으로 보내므로 302 루프는 생기지 않는다. 그러나 **`/login` 이 `/login` 으로 되돌리는 return_to**(`return_to=/login`)는 자기 자신을 가리켜 302 가 무한히 반복될 수 있다 → `safeReturnTo` 통과 뒤 경로가 `/login` (또는 `/login?…`)이면 `/` 로 취급하는 한 줄을 넣고 테스트 한 케이스 추가.
  - `safeReturnTo` 자체를 넓히거나 고치지 말 것(SSO 콜백·조용한 시도가 같은 함수를 쓴다 — 여러 경로가 같은 값을 읽는 자리이니 이번 과제는 '읽는 쪽 추가'만).
  - `webui.Handler()` 가 내부에서 임베드 파일시스템을 여는 비용이 있으면 요청마다 만들지 말 것(assets.go 확인).
  - JS 쪽 `/api/v1/auth/me` 호출을 지우지 말 것 — auto_login 이 켜진 설치에서 조용한 시도 전에 세션을 보는 기존 규칙(silentsso.test.mjs 가 지킴)이다.
  - auth/migrations/workflows 는 건드리지 않는다. 002 DDL 변경 없음.

- 차선 후보: 로그인 페이지 조용한 SSO 시도 중 폼 깜빡임 제거 — `login.js` 에서 `system/info` 응답으로 `shouldAttemptSilentSso` 가 true 로 판정될 때만 `.login-card` 의 폼을 숨기고, `beginSilentSso` 로 떠나지 않는 모든 경로(`system/info` 실패·예외 포함, `finally`)에서 반드시 다시 보이게. 검증은 `node test/js/silentsso.test.mjs` 확장이 아니라 login.js 를 DOM 대역 없이 시험하기 어려우므로 스모크의 Chromium 으로 확인해야 한다(그래서 2순위).
