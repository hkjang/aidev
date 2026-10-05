- 과제: Introspection Endpoint 가 판정하지 못한 Client 인증에 401 대신 200 `{"active":false}` + `resso_introspection_errors_total{stage="client_auth"}` 로 답하게 하기 (가치 3 / 위험 2 / 작업량 M)

- 왜: `introspect`(internal/httpserver/oidc.go:1178-1182)는 `authenticateOIDCClient` 의 **모든** 실패를 `writeClientAuthError` 로 넘겨 401 `invalid_client` "client authentication failed" 로 답한다. `clients` 테이블이 답하지 못했거나 저장된 digest 가 디코드되지 않은 조회는 그 Secret 에 대해 아무것도 판정하지 않았는데, 401 은 Resource Server 에게 "네 자격증명이 틀렸다" 를 말해 사람을 부르거나 자격증명을 폐기하게 만든다 — v0.9.97(954da4b)이 Limiter·`client_auth_failures`·로그를 갈라 놓고 응답만 Endpoint 별 과제로 남긴 **마지막 조각**이며, oidc.go:937-959 주석이 이 Endpoint 하나만 남았다고 스스로 지목한다(token 500 = v0.9.98 ad4e847, revoke 503 = v0.9.99 231f3f7). 고치면 이 Endpoint 의 fault 계약이 나머지 다섯 단계(`realm`·`revocation_state`·`session`·`user`·`refresh_token`)와 같아지고, 장애 중 Introspection 이 무너진 것이 `resso_introspection_errors_total` 한 계열에서 전부 보인다.

- 수용 기준:
  1) `ALTER TABLE clients RENAME TO clients_hidden` 중 **맞는** Basic 자격증명으로 보낸 `/realms/master/protocol/openid-connect/token/introspect` 20회가 전부 **200 + 본문이 정확히 `{"active":false}`** (`error` 키 없음, `WWW-Authenticate` 헤더 없음).
  2) 복구 후 **같은 Client·같은 Secret** 이 429 가 아니라 200 `active=true` + `client_id`·`exp` 를 돌려준다(v0.9.97 의 "Limiter 예산을 깎지 않는다" 회귀 방지). 진짜 access token 은 RENAME **전에** `ressooidc.Service{Store: data}.IssueUserTokens` 로 발급해 둘 것.
  3) 지표·로그: 장애 중 `resso_introspection_errors_total{stage="client_auth"} 20` **그리고** `resso_client_auth_errors_total{stage="client"} 20`, `resso_client_auth_failures_total` 는 아예 없음, ERROR `a client authentication could not be decided` 정확히 20줄, **`introspection could not judge a token` 줄은 0** (로그 중복 금지 — 아래 "건드릴 파일" 참조), Secret 원문·`client_id` 는 로그·라벨·본문 어디에도 없음.
  4) 읽고 **거절한** 것은 그대로다: 틀린 Secret 20회 → 401 `invalid_client` + `resso_client_auth_failures_total{realm="master"} 20`, 21번째 → 429 + `Retry-After`; 등록되지 않은 `client_id` → 401; **public(= `client.Type != "confidential"`) Client → 401** (이 세 경우 뒤에도 `introspection_errors{stage="client_auth"}` 는 20 에서 움직이지 않는다).
  5) 끊긴 호출자(`LOCK TABLE clients IN ACCESS EXCLUSIVE MODE` + `pg_locks` 폴링 + context 취소)는 `introspection_errors`·`client_auth_errors`·ERROR 줄을 하나도 올리지 않는다(8f2c73c 가 닫은 구멍). **순서 주의**: Limiter 검사가 조회 **앞**(oidc.go:812-816)이므로 이 단계를 틀린 Secret 소진 **앞**에 둘 것 — 뒤에 두면 429 로 막혀 조회에 도달하지 못한다. 선례는 `TestIntegrationClientAuthSeparatesABrokenDigestFromACallerThatHungUp`(integration_test.go:1835 부근)과 v0.9.99 의 revoke 테스트.

- 건드릴 파일:
  - `internal/httpserver/oidc.go:introspect` (1178-1182) — `s.writeClientAuthError(w, r, err)` **앞**에 분기 하나:
    `if errors.Is(err, errClientAuthUndecided) { s.metrics.Add(metricIntrospectionErrors, 1, "client_auth"); writeJSON(w, http.StatusOK, map[string]any{"active": false}); return }`.
    **`recordUnjudgedIntrospection` 을 쓰지 말 것** — 그 헬퍼는 ERROR `introspection could not judge a token` 을 한 줄 더 쓰고, `clientAuthUndecided`(oidc.go:972-983)가 이미 `stage`·`route`·사유를 담은 ERROR 를 남겼으므로 같은 사건이 두 줄이 된다(v0.9.99 가 revoke 에서 로그를 더하지 않은 것과 같은 판단). 지표만 직접 더하고, **왜 헬퍼를 우회했는지**를 주석으로 남길 것. 분기는 `err != nil || !ok || client.Type != "confidential"` 통짜 조건 **밖/앞**에 두어야 public Client 와 `ok == false` 는 401 그대로 남는다.
  - `internal/httpserver/oidc.go:937-959` 주석 — "Introspection does not, so the marked error reaches writeClientAuthError … That is left for that endpoint to do." 가 이 변경으로 **틀린 말이 된다**. 세 Endpoint 모두 끝났다는 사실과, 첫 문단 "two of them no longer do" → "all three" 로 함께 고칠 것(v0.9.99 가 "one of them" → "two of them" 을 놓쳐 뒤늦게 고친 자리다).
  - `internal/httpserver/oidc.go:recordUnjudgedIntrospection` 주석(1259-1274) — "A Realm, a session, an account or a refresh token that is not there…" 가 단계를 열거하므로, `client_auth` 는 이 헬퍼를 지나지 않는다는 한 문장을 더할지 판단할 것(권장: 더한다).
  - `docs/operations.md:90` — v0.9.99 가 써 둔 `client_auth_errors_total` 불릿의 "introspection 만 401" 문장이 **틀린 말이 된다**. 세 Endpoint 의 답(token 500 / revoke 503 + 감사 / introspection 200 `active=false`)으로 고칠 것. 한 문장 더하기가 아니라 **고치기**다.
  - `docs/operations.md:91` — `introspection_errors_total` 불릿에 `client_auth` 단계의 뜻(Secret 이 틀린 것이 아니라 판정하지 못한 것, 틀린 Secret 은 여전히 401 이고 `client_auth_failures` 로 간다)을 한 문장. **`README.md:193` 은 계열이 늘지 않으므로 손대지 말 것**, ADMIN_GUIDE·PDF·캡처 재생성 없음.
  - `internal/httpserver/integration_test.go` — 새 테스트 `TestIntegrationIntrospectionSaysWhenItCouldNotDecideTheClient`. 실제 PostgreSQL + `New(data, logger, nil, metrics)` + httptest, 로그는 `lockedBuffer`(7287 부근), 장애는 `ALTER TABLE clients RENAME TO clients_hidden` + 단언 **전** 즉시 복구 + `t.Cleanup`. 프로덕션 파일은 `oidc.go` **하나**.

- 검증 명령:
  - `eval "$(scripts/test-services.sh)"` (출력이 비면 준비 실패 — eval 하지 말 것) 뒤 **같은 셸에서**:
  - `go test -race ./internal/httpserver -run '^TestIntegrationIntrospectionSaysWhenItCouldNotDecideTheClient$' -count=1 -v` — 프로덕션 변경 **전에** 먼저 돌려 401 로 실패하는 것을 기록할 것.
  - `go test -race ./internal/httpserver -count=1` (약 115~140s)
  - `make lint` (golangci-lint v2.13.1 + govulncheck + eslint --max-warnings 0; gofmt 정렬이 첫 실행에서 자주 걸린다)
  - `make test` (exit 0 + `"integration test(s) did not run"` 경고가 **Makefile 레시피 에코 말고는** 없음 = SKIP 0; vitest 29파일/161테스트) → 뒤에 `webui/dist/index.html` 복원, `git diff --check`, `git status` 에 빌드 산출물 없음.

- 위험과 피할 것:
  - **규범 해석이 이 과제의 유일한 미확인 지점이다.** oidc.go:941 이 introspection 의 fault 계약을 "200 active=false" 로 못박고 있고(저장소 내부 권위), 955-958 이 남긴 질문은 "RFC 7662 가 인증을 확인하지 못한 호출자에게 200 을 주라는 뜻이냐" 다. 이번 회차에 **RFC 7662 원문은 읽지 않았다(미확인)** — 구현자는 §2.2/§2.3 문안을 직접 확인하고 주석에 근거를 적을 것. 판단 근거로 쓸 수 있는 것: 본문 `{"active":false}` 는 모르는 Token·남의 Token 에 주는 답과 **한 글자도 같아서 새로 흘리는 정보가 없다**. 원문이 "properly authorized 가 아닌 호출자는 반드시 401" 로 읽히면 이 과제를 **중단하고 차선 후보로 갈 것** — 추측으로 밀지 말 것.
  - 응답이 200 으로 바뀌면 장애 중 `resso_http_requests_total{route,status="200"}` 가 올라간다(revoke 의 503 과 방향이 반대). 그래서 fault 신호는 `introspection_errors` 뿐이고, 끊긴 호출자 구멍은 `clientAuthUndecided` 의 `ours` 게이트가 이미 막는다 — 기준 5 는 그 게이트가 **살아 있음**을 단언하는 것이지 새로 만드는 것이 아니다.
  - `writeClientAuthError`(985-1000) 안에서 세 Endpoint 를 합치지 말 것 — 그 헬퍼는 Rate limit 만 보고 401 을 쓴다. 분기는 반드시 호출부 **앞**.
  - `integration_test.go:3298` 의 `for path, endpoint := range {"/token/introspect","/revoke"}` 루프는 **꺼진 Client** 에 401 을 요구한다 — 그대로 통과한다(확인함): `authenticateOIDCClient`(oidc.go:843-845)는 꺼진 Client 에 `(client, false, nil)` 을 돌려주므로 `err == nil` 이고 센티널 경로가 아니다. 같은 이유로 public Client(`client.Type != "confidential"`)도 `err == nil` 이다. 그래서 새 분기는 **`errors.Is(err, errClientAuthUndecided)` 하나만** 보면 되고 `ok`·`Type` 조건은 건드릴 필요가 없다.
  - 다른 introspect 호출 지점(integration_test.go:846·2386·4510·5122·5219·5336·7623·10332·10488)은 정상 자격증명을 쓰므로 영향이 없어야 한다 — `go test -race ./internal/httpserver -count=1` 전체로 확인할 것.
  - 보호 경로는 건드리지 않는다: `auth.go`·`middleware.go`·`internal/store/migrations`·`.github/workflows`. 새 지표 계열을 **만들지 말 것**(기존 `metricIntrospectionErrors` 에 라벨값 하나만 추가 — `registerMetrics` 수정 불필요, `metrics.go` 는 손대지 않는다).
  - 사람이 받지 않은 `5bed9dc`(코드 교환 400 합치기)로 번지지 말 것.

- 차선 후보: 기존 Token 연동 테스트 둘(`TestIntegrationTokenSaysWhenItCouldNotReadTheRealm` 962행 부근 · `…ReadTheAccount` 1130행 부근)에 등록기를 붙여 `resso_token_errors_total{grant_type}` 배선을 고정하기 (가치 2 / 위험 1 / 작업량 S, 프로덕션 파일 0개). 둘이 `New(data, logger, nil, nil)` 로 서버를 세워 지표를 읽을 수 없으므로, `New(data, logger, nil, metrics)` + `srv.Metrics().WritePrometheus` 로 바꾸고 판단 셋을 단언한다: Realm 장애는 **일부러 세지 않음**(oidc.go:475-481), 계정 장애는 **셈**, Client 인증 미판정도 **세지 않음**(oidc.go:504-510 주석). 선례는 같은 파일 1389-1416·1560-1592.
