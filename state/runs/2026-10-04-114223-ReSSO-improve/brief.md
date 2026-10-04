- 과제: Token Endpoint 가 판정하지 못한 Client 인증에 401 대신 500 `server_error` 로 답하게 하기 (가치 4 / 위험 2 / 작업량 M)

- 왜: `clientAuthUndecided`(oidc.go:891)는 v0.9.97 에서 Limiter·지표·로그까지 갈라 놓았지만 **응답은 일부러 그대로 뒀고**, 그 자리의 주석이 스스로 다음 할 일을 지목한다 — "these three endpoints answer a failed client authentication identically today and their contracts for a fault on this side are not the same one — the token endpoint's 500, revoke's 503 temporarily_unavailable, introspection's 200 active=false — so that is a change per endpoint, not one made in the helper they share."(oidc.go:875 부근) 그래서 `clients` 테이블이 답하지 못하면 RP 는 401 `invalid_client` "client authentication failed" 를 받는데, 그것은 "네 Secret 이 틀렸다" 는 뜻이고 표준 RP 동작은 사람을 부르거나 자격증명을 폐기하는 것이다 — Secret 은 처음부터 맞았고 장애가 끝나면 그대로 다시 돌 수 있었는데. 같은 논지가 이 Endpoint 안에 이미 세 번 있다(`realmLookupFailed` 475-481, `userLookupFailed`, `ErrNoActiveSigningKey`) — Client 인증만 그 앞에 남아 있다.
  이번 회차는 **세 Endpoint 중 token 하나만** 담는다. 나머지 둘은 계약이 다르므로(revoke 503, introspection 200 active=false) 별도 회차다.

- 수용 기준:
  1) `clients` 테이블을 읽을 수 없는 동안 `POST /realms/master/protocol/openid-connect/token` 이 **500 + `{"error":"server_error", …}`** 로 답하고, 본문에 `invalid_client` 가 없고, `WWW-Authenticate` 헤더가 붙지 않는다.
  2) 테이블 복구 뒤 **같은 Client 가 같은 맞는 Secret 으로 200 + access_token** 을 받는다(429 가 아니다). 즉 v0.9.97 의 Limiter 보존이 깨지지 않았음을 같은 테스트가 계속 증명한다.
  3) 판정된 거절은 한 글자도 바뀌지 않는다: ① 틀린 Secret 20회 → 401 `invalid_client` + `resso_client_auth_failures_total{realm="master"} 20`, 21번째 429 ② 등록되지 않은 `client_id`(`store.ErrNotFound`) → 401 + failures +1 ③ 저장할 수 없는 `client_id`(`bad-%ff`) → 401 ④ **읽혔지만 디코드되지 않는 `secret_hash`(stage=`secret`) → 500**(이쪽 장애다) ⑤ Rate limit 429 → 그대로 429 + `Retry-After`.
  4) **요청을 이미 끊은 호출자는 500 을 받지 않는다** — `r.Context().Err() != nil` 로 undecided 가 된 경우는 500 을 쓰지 않는다. 8f2c73c 가 닫은 구멍(인증 없는 호출자가 마음대로 장애 신호를 올리는 것)을 `resso_http_requests_total{route,status="500"}` 에서 다시 열면 안 된다. 테스트가 이것을 단언해야 한다(아래 "위험" 의 재현 수단 참조).
  5) `resso_token_errors_total` 은 **더하지 않는다**. 이 시점의 `grant_type` 은 아직 `client.GrantTypes` 와 대조되지 않은 미검증 입력이라 라벨 카디널리티가 열린다(metrics.go:88·105 의 `methodLabel`·`routePattern` 주석이 같은 이유를 설명한다). 이 실패는 이미 `resso_client_auth_errors_total{stage}` 가 세고 500 은 `resso_http_requests_total{route,status}` 에 보인다 — 475-481 의 Realm 분기가 똑같은 이유로 지표를 더하지 않았고, 그 판단을 주석으로 남길 것.
  6) Secret 원문·`client_id` 는 응답·로그·라벨 어디에도 나타나지 않는다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpserver/oidc.go`
    - `clientAuthUndecided`(891행) — 반환을 `(undecided, ours bool)` 로 바꾼다(또는 동등한 신호). `ours` 는 "이쪽이 보고할 장애" 즉 지표·로그를 남긴 경우만 true, `r.Context().Err() != nil` 로 빠져나가는 경우는 `(true, false)`. **`r.Context().Err()` 를 두 번 읽지 말 것** — 호출자에서 다시 읽으면 두 경로가 같은 값을 다르게 읽는다.
    - `authenticateOIDCClient`(751행) — `ours` 일 때만 오류를 새 센티널로 감싼다: `fmt.Errorf("%w: %w", errClientAuthUndecided, err)`. 감싸지 않은 경로(판정된 거절, 끊긴 호출자, rate limit)는 지금 그대로 흘러간다. 센티널은 `rateLimitedError`(744행) 바로 옆에 `var errClientAuthUndecided = errors.New(…)` 로.
    - `token`(485-488행) — `s.writeClientAuthError` 를 부르기 **앞**에 `if errors.Is(err, errClientAuthUndecided)` 분기를 두고 `writeOAuthError(w, http.StatusInternalServerError, "server_error", …)`. 문안은 아는 것만 말할 것 — 권장: `the client credential could not be verified; no attempt has been counted against this client, retry after a short delay`.
    - introspection(1097행)·revocation(1228행) 호출자는 **건드리지 않는다**. `writeClientAuthError` 는 `errors.As(&rateLimitedError)` 만 보므로 감싼 오류가 와도 지금과 똑같이 401 을 쓴다 — 두 Endpoint 의 답은 이번 회차에 안 바뀐다. 그 사실과 왜 미뤘는지를 875행 부근 주석에서 갱신할 것(세 Endpoint 가 "identically" 답한다는 문장이 이 변경으로 **틀린 말이 된다**).
  - `internal/httpserver/integration_test.go` — **기존 테스트 둘을 고쳐야 한다**(아래 위험 참조) + 새 단언.
  - `docs/operations.md` — `resso_client_auth_errors_total` 불릿에 한 문장(이 실패가 token 에서 500 이 되고 왜 `client_auth_failures` 가 아닌지). 쓰기 전에 그 불릿을 읽어 v0.9.97 의 문장과 겹치지 않는지 확인할 것. `README.md` 지표 표는 계열이 늘지 않으므로 손대지 않는다. ADMIN_GUIDE·PDF·캡처 재생성 없음.

- 검증 명령:
  ```
  eval "$(scripts/test-services.sh)"        # 출력이 비면 준비 실패 — eval 하지 말 것
  go test -race ./internal/httpserver -run '^TestIntegrationClientAuth' -count=1 -v
  go test -race ./internal/httpserver -count=1          # 약 120s
  make lint                                             # 0 issues 여야 한다
  make test                                             # "integration test(s) did not run" 경고가 없어야 한다(SKIP 0)
  git checkout webui/dist/index.html ; git diff --check
  ```
  먼저 고치기 전 코드로 새 단언을 돌려 **빨간 것을 눈으로 볼 것**(`integration_test.go:1635` 가 지금은 401 을 요구하므로, 단언을 500 으로 바꾸면 수정 전에는 실패한다).

- 위험과 피할 것:
  - **기존 테스트 둘이 지금 401 을 단언한다. 이번 회차에 회귀 범위를 다 재어 봤고, 고쳐야 하는 단언은 그 둘뿐이다.**
    ① `integration_test.go:1635-1637` (`TestIntegrationClientAuthSaysWhenItCouldNotDecide`, 1537행, 954da4b 가 추가) — `clients` 은닉 중 20회 전부 `401 invalid_client` 를 요구한다. 이것을 500 `server_error` 로 고치는 것이 이번 변경의 계약 변경 그 자체다. 같은 테스트의 다른 단언(복구 뒤 200, `client_auth_failures` 부재, 틀린 Secret 20회 401 + 21번째 429 = 1683행, 등록되지 않은 `client_id` 401 = 1725행, 저장 불가 `client_id` 401 = 1762행)은 **한 글자도 바꾸지 말 것** — 그것이 수용 기준 2·3 이다.
    ② `integration_test.go:1881-1883` (`TestIntegrationClientAuthSeparatesABrokenDigestFromACallerThatHungUp`, 1805행; `tokenPath` 는 1832행) — 깨진 digest(stage `secret`)에 `401 invalid_client` 를 요구한다. 이쪽도 500 이 된다(수용 기준 3④).
    같은 테스트의 `answered()`(1934-1946행)는 `resso_http_requests_total` 중 `route="/realms/{realm}/protocol/openid-connect/token"` 인 **모든 줄의 값을 더한다 — status 라벨로 걸러내지 않는다**(확인함). 그래서 401→500 으로 바뀌어도 이 헬퍼는 깨지지 않는다. 수용 기준 4 를 단언하려면 **status 라벨을 보는 리더를 따로 하나 더 두고**(같은 파싱, `status="500"` 만 합산) 끊긴 요청 전후로 그 값이 변하지 않음을 검사할 것.
    ③ `integration_test.go:3233-3235` 는 `/token/introspect`·`/revoke` 만 돌므로 영향 없다(확인함).
    ④ `clients` 를 RENAME 하는 다른 자리 둘은 token 을 때리지 않는다(확인함): `8227` 은 로그아웃 302/`client_unavailable` 감사 테스트, `10630` 은 `oidcCORS` 의 OPTIONS preflight 테스트로 둘 다 `authenticateOIDCClient` 를 지나지 않는다.
  - **수용 기준 4 의 재현 수단은 이미 있다 — 새로 만들지 말고 재사용할 것.** `TestIntegrationClientAuthSeparatesABrokenDigestFromACallerThatHungUp`(1805행)이 ⓐ `blocker` 트랜잭션으로 `LOCK TABLE clients IN ACCESS EXCLUSIVE MODE`(1930행) ⓑ `context.WithCancel` 로 만든 요청을 고루틴에서 보내고(1950-1968행) ⓒ `pg_locks` 를 폴링해 그 요청이 실제로 조회에 도달했는지 확인한 뒤(1970-1986행) ⓓ `hangUp()` 으로 끊고 ⓔ `answered()` 가 움직일 때까지 기다려 핸들러 종료를 동기화한다(1995-2003행). `release()` 가 `t.Cleanup` 에 걸려 있으니 **락을 이 테스트 밖으로 새게 하지 말 것** — 같은 컨테이너의 다른 테스트가 전멸한다.
  - 세 Endpoint 를 한 번에 고치지 말 것. 헬퍼에서 응답을 합치지 말 것 — 계약이 셋이고, 그것이 이 저장소가 반복해 적어 둔 설계다.
  - `store.ErrNotFound` 를 세는 경로에서 빼지 말 것(Limiter 의 식별자 대입 bound).
  - 보호 경로를 건드리지 않는다: `internal/store/migrations`, `.github/workflows/`, `auth.go`(세션·쿠키), `middleware.go`.
  - **`RedeemAuthorizationCode` 의 모든 실패를 400 으로 합치는 것(`handleAuthorizationCodeGrant`)은 고르지 말 것.** 그 변경은 `5bed9dc` 로 구현됐으나 `main` 에 들어 있지 않고(이번 회차에 `TestIntegrationTokenSaysWhenItCouldNotRedeemTheCode` 가 `integration_test.go` 에 없음을 확인), 사람이 받지 않은 접근은 재제출하지 않는다.
  - `make lint` 첫 실행의 gofmt 정렬과 빌드가 바꾸는 `webui/dist/index.html` 복원을 잊지 말 것.

- 차선 후보: **기존 Token 연동 테스트 둘에 등록기를 붙여 `resso_token_errors_total{grant_type}` 배선을 고정** (가치 2 / 위험 1 / 작업량 S, 프로덕션 파일 0개). `TestIntegrationTokenSaysWhenItCouldNotReadTheRealm`(962행)과 `…ReadTheAccount`(1130행)이 각각 1003·1171행에서 `New(data, logger, nil, nil)` 로 서버를 세워(이번 회차에 확인) v0.9.93·v0.9.94 의 지표 판단을 아무것도 고정하지 않는다. `observability.NewRegistry()` 를 넘기고 `metrics.WritePrometheus` 로 읽어, Realm 장애는 **세지 않고**(일부러 그렇게 뒀다 — 475-481 주석) 계정 장애는 `{grant_type="refresh_token"}`·`{grant_type="authorization_code"}` 각 1 로 센다는 것을 단언한다. 선례는 같은 파일 1389-1416·1560-1592 행.
