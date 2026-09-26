- 과제: Token Endpoint가 Realm 조회 실패를 `invalid_grant`로 답하는 것을 고치기 — 이쪽 장애는 500 `server_error`로 (가치 4 / 위험 1 / 작업량 S)

- 왜: `token`(internal/httpserver/oidc.go:428-432)은 `realmFromPath`의 오류를 `ErrNotFound`인지 보지 않고 전부 `writeOAuthError(400, "invalid_grant", "realm is unavailable")`로 답한다 — `realms` 테이블을 읽지 못하는 이쪽 장애에도 RP는 "네 code/refresh token은 죽었다"는 답을 받고, 그 답에 대한 표준 동작은 토큰을 버리고 사람을 다시 로그인시키는 것이라 장애가 끝나도 살릴 수 있었던 세션이 전부 버려진다. 이 논증은 이 저장소가 이미 같은 함수 안에 적어 둔 것이다(oidc.go:489-497, `ErrNoActiveSigningKey`는 그래서 500 `server_error`로 답한다) — Realm 조회만 그 앞에 남아 있다. 덧붙여 `realmFromPath` 호출자 중 `token`만 `realmLookupFailed`를 쓰지 않아(discovery:72 · jwks:132 · authorization:189 · revocation:999 · logout:1155 · cors middleware.go:117) 로그에 `endpoint=token` 한 줄도 남지 않는다 — 어제까지 `oidcCORS`가 그 유일한 예외였고(v0.9.92) 그것이 고쳐진 지금 남은 하나다.

- 수용 기준:
  1) `realms` 조회가 `store.ErrNotFound`가 아닌 오류를 내면 `POST /realms/{realm}/protocol/openid-connect/token`이 **500 `server_error`**(본문은 기존 `writeOAuthError` 모양)로 답하고, 응답 어디에도 `invalid_grant`가 없다. 메시지는 이 장애가 지나가는 것임을 말한다(예: `the realm could not be read; the token is still valid, retry after a short delay`).
  2) 같은 요청이 서버 로그에 `realmLookupFailed`의 기존 한 줄(`the Realm named in the route could not be looked up`)을 `endpoint=token`으로 남긴다. 새 로그 문구를 만들지 말고 `s.realmLookupFailed(r, "token", err)`를 그대로 쓴다.
  3) **없는 Realm과 꺼진(suspended) Realm의 답은 한 글자도 바뀌지 않는다** — 둘 다 `realmFromPath`에서 `store.ErrNotFound`이므로 400 `invalid_grant` "realm is unavailable"을 그대로 유지하고, `endpoint=token` 로그도 남기지 않는다.
  4) 테스트가 증명할 것: (a) `realms` 테이블이 없는 동안 실제 refresh token을 들고 온 token 요청이 **수정 전에는 400 `invalid_grant`** 였음(먼저 확인), 수정 후 500 `server_error` + `endpoint=token` 로그 한 줄, (b) 테이블을 되돌린 뒤 **같은 refresh token이 그대로 200으로 교환된다** — 장애 응답이 RP에게 거짓말을 했을 뿐 토큰은 살아 있었다는 것, (c) 없는 Realm은 400 `invalid_grant`이고 `endpoint=token` 줄이 없다.
  5) 지표는 더하지 않는다 — `resso_token_errors_total`의 유일한 라벨은 `grant_type`이고(metrics.go:56, 값은 `authorization_code`/`refresh_token`/`client_credentials`) Realm 조회는 `grant_type`을 읽기 전에 실패하므로 여기에 `realm` 같은 값을 넣으면 라벨의 뜻이 깨진다. 500은 `resso_http_requests_total{route,status}`에 이미 보인다. 이 판단을 주석 한 줄로 남길 것. 감사 항목도 더하지 않는다(같은 함수의 signing-key 500 경로도 남기지 않는다).

- 건드릴 파일 (프로덕션 2개):
  - `internal/httpserver/oidc.go:token`(428-432행) — `if err != nil` 안에서 `s.realmLookupFailed(r, "token", err)`로 갈라, 참이면 500 `server_error`, 거짓이면 기존 400 `invalid_grant`. 왜 `invalid_grant`가 이 경우에 해로운지를 이 저장소의 주석 밀도로 적을 것(489-497행의 문장과 같은 논지, 중복 서술이 아니라 "그 앞에 하나 더 있었다"로).
  - `docs/operations.md:87`(`resso_token_errors_total` 불릿) — 그 자리에 「Realm 조회 실패는 이 계열에 세지 않고(라벨이 `grant_type`이라) 500 `server_error`로 나가며, 서버 로그 `the Realm named in the route could not be looked up`의 `endpoint=token`으로 찾는다」는 한 문장을 더할 것. 400 `invalid_grant`(호출자 문제) vs 500 `server_error`(이쪽 문제) 구분은 이미 `docs/ADMIN_GUIDE.md:570`에 있고 맞으므로 고치지 말 것(문장만 더하므로 PDF·캡처 재생성 없음 — 최근 회차 관례). ADMIN_GUIDE의 문장은 이미 맞으므로 고치지 말 것.
  - `internal/httpserver/integration_test.go` — 새 테스트 `TestIntegrationTokenSaysWhenItCouldNotReadTheRealm`. 배선은 프로덕션 그대로: `openHTTPIntegrationStore(t)` → `New(data, logger, nil, nil).Handler()` + `httptest.NewServer`, 토큰은 `service.IssueUserTokens(...)`(선례 7435·8721행), 로그는 `lockedBuffer`(정의 6901-6917행, 사용처 6982·7203·9479)로 `slog.NewTextHandler(logs, nil)`, 저장소 장애는 `data.Pool.Exec(ctx, "ALTER TABLE realms RENAME TO realms_hidden")` + `t.Cleanup`으로 되돌리기(선례 2898·3700·7144·8092행). client 인증 방식은 기존 token 테스트가 쓰는 것을 그대로 따를 것(`client_secret_basic`/`post` 중 그 테스트가 쓰는 쪽).

- 검증 명령:
  - `eval "$(scripts/test-services.sh)"` 를 먼저 같은 셸에서. 출력이 비면 준비 실패이므로 eval 하지 말 것.
  - `go test -race ./internal/httpserver -run '^TestIntegrationTokenSaysWhenItCouldNotReadTheRealm$' -count=1 -v` — **수정 전 oidc.go로 먼저 돌려 (a)가 실제로 400 `invalid_grant`로 실패하는 것을 확인**하고 그 출력을 요약에 남길 것.
  - `go test -race ./internal/httpserver -run '^TestIntegration(SuspendedRealmRefusesEveryProtocolEndpoint|RefreshToken|Token)' -count=1` — 기존 token·suspended 계약 회귀. (suspended 테스트는 `status < 400`만 보므로 여기서 답이 바뀌지 않는 것을 3)의 근거로 따로 단언할 것.)
  - `make lint` (golangci-lint + govulncheck + ESLint), `make test` (전체 수 분; httpserver 약 110~120s, SKIP 0 = "integration test(s) did not run" 경고 없음을 눈으로 확인).
  - 빌드가 바꾼 `webui/dist/index.html`은 되돌리고 `git diff --check`.

- 위험과 피할 것:
  - **응답 코드를 바꾸는 유일한 조건은 `realmLookupFailed(...) == true`다.** `errors.Is(err, store.ErrNotFound)` 쪽(없는 Realm·꺼진 Realm)을 함께 바꾸면 suspended Realm 계약과 "없는 것을 물어볼 수 있게 하지 않는다"는 이 저장소의 규칙이 깨진다.
  - 400 → 500 상태 변경은 이 회차의 전부다. `handleAuthorizationCodeGrant`/`handleRefreshGrant`/`handleClientCredentialsGrant` 안의 어떤 `invalid_grant`도 건드리지 말 것(485·516·536·560·594행) — 그것들은 호출자 문제이고 별 과제다.
  - 오류 코드로 `temporarily_unavailable`을 새로 도입하지 말 것. 이 Endpoint는 이미 이쪽 장애를 `server_error`로 답하고(494·504행) 문서도 그 쌍으로 적혀 있다(ADMIN_GUIDE.md:570). revoke가 503 `temporarily_unavailable`인 것은 RFC 7009 §2.2.1이 그것을 명시하기 때문으로, token Endpoint에는 그 근거가 없다(미확인이므로 근거 없이 새 문자열을 만들지 말 것).
  - `realmFromPath`·`realmLookupFailed`의 **시그니처와 로그 문구는 그대로** 둘 것 — 호출자가 일곱이다.
  - `internal/httpserver/auth.go`(세션·쿠키), `internal/store/migrations`, `.github/workflows/`는 건드리지 않는다.
  - `ALTER TABLE realms RENAME`은 그 서버의 거의 모든 경로를 죽인다. 로그인·세션을 요구하는 준비 작업은 RENAME **전에** 끝내고, cleanup에서 반드시 되돌릴 것(되돌리지 않으면 같은 컨테이너를 쓰는 뒤 테스트가 줄줄이 깨진다).
  - gofmt 정렬이 lint 첫 실행에서 자주 잡힌다.

- 차선 후보: UserInfo 거절 카운터 `resso_userinfo_errors_total` 추가 — `writeUserInfoUnavailable`(oidc.go:855)이 stage 여섯을 로그로만 남기고, token·introspection·authorization·silent 네 계열과 달리 userinfo만 카운터가 없다. 라벨은 그 여섯 stage의 고정 집합으로 하고 미검증 입력을 라벨에 넣지 말 것(가치 2 / 위험 1 / 작업량 S).
