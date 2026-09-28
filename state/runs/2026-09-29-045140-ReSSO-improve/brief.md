- 과제: UserInfo가 판정하지 못해 거절한 요청을 `resso_userinfo_errors_total{stage}` 로 세기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `writeUserInfoUnavailable`(internal/httpserver/oidc.go:968)은 이쪽 장애 여섯 자리(`realm`·`revocation_state`·`user`·`session`·`realm_roles`·`client_roles`)를 ERROR 로그 한 줄로만 남기고 지표에는 아무것도 남기지 않아, 운영자는 `resso_http_requests_total{route,status}`의 500 하나만 보고 어느 조회가 무너졌는지 알려면 로그를 뒤져야 한다. 같은 논지로 introspection(`stage`)·authorization(`stage`)·silent(`result`)에는 이미 라벨 달린 계열이 있고(metrics.go:29·35·41) `errors_total` 계열 넷 중 userinfo만 빠져 있다 — 한 줄짜리 배선으로 이 Endpoint가 다른 셋과 같은 방식으로 관측된다.
- 수용 기준:
  1) `GET /metrics` 에 `resso_userinfo_errors_total` 이 `stage` 라벨 하나로 등록되고, userinfo가 500 `server_error` 로 거절할 때마다 해당 stage가 1 오른다. 라벨 값은 코드에 이미 있는 여섯 리터럴뿐이며 요청에서 온 값은 라벨에 넣지 않는다.
  2) **응답은 한 글자도 바뀌지 않는다** — 500 `server_error` / `"the request could not be completed"` 본문, 기존 로그 문구 `userinfo could not judge the request it was given`(`trace_id`·`realm`·`stage`·`error`), 401 `invalid_token`(없는 Realm·없는/꺼진 계정·없는 세션·검증 실패 토큰)과 400(폼·토큰 이중 제시) 경로 전부 그대로.
  3) 평범한 거절은 세지 않는다: 만료·위조 토큰의 401, 꺼진 계정의 401, 없는 Realm의 401에서 이 계열이 0으로 남는 것을 테스트가 단언한다(호출자가 지표를 올릴 수 없다는 뜻 — 인증 없는 호출자가 경보를 울리면 안 된다).
  4) 새 연동 테스트가 실제 PostgreSQL에서 stage 최소 셋(`realm_roles`·`client_roles`·`user`)을 테이블 RENAME으로 재현해 라벨별 값 1을 읽고, 표를 되돌린 뒤 같은 토큰이 다시 200을 받는 것까지 확인한다.
  5) `docs/operations.md` 의 경보 목록에 이 계열 불릿 하나(여섯 stage의 뜻, 왜 500이 요청 카운터만으로는 안 보이는지, 찾을 로그 문구). ADMIN_GUIDE·PDF·캡처는 건드리지 않는다.
- 건드릴 파일 (프로덕션 2개):
  - `internal/httpserver/metrics.go` — 상수 블록(22~45행 사이, `metricIntrospectionErrors`·`metricAuthorizationErrors` 옆)에 `metricUserInfoErrors = "resso_userinfo_errors_total"` 을 이 저장소의 서술형 주석(왜 이 계열이 따로 필요한가: 500은 요청 카운터에 보이지만 어느 조회가 죽었는지는 아무 데도 없다)과 함께 추가하고, `registerMetrics`(49행)에 `registry.Counter(metricUserInfoErrors, "userinfo requests the service could not judge, by the lookup that failed.", "stage")` 한 줄.
  - `internal/httpserver/oidc.go:968 writeUserInfoUnavailable` — 로그 줄 **앞**에 `s.metrics.Add(metricUserInfoErrors, 1, stage)` 한 줄 + 왜 `store.ErrNotFound` 필터가 여기 없는지의 주석(아래 '확인한 사실' 참고). 함수 시그니처·호출자 여섯 곳은 그대로.
  - `internal/httpserver/integration_test.go` — 새 테스트 `TestIntegrationUserInfoCountsTheLookupsItCouldNotMake`(이름은 재량). 본은 바로 위의 `TestIntegrationUserInfoRefusesWhenRolesCannotBeRead`(10338행)를 그대로 따를 것: `openHTTPIntegrationStore` → `Bootstrap` → `EnsureActiveSigningKey` → `CreateClient`(scope `openid roles`) → `CreateUser` → `CreateSession` → `ressooidc.Service{Store: data}.IssueUserTokens` 로 진짜 access token, `httptest.NewServer(New(...).Handler())`, 장애는 `ALTER TABLE <t> RENAME TO <t>_hidden` + **즉시 복구(또는 `t.Cleanup`)**.
  - `docs/operations.md` — 90·91행(`resso_introspection_errors_total`·`resso_authorization_errors_total`) 옆에 불릿 하나.
- 검증 명령:
  - 준비: `eval "$(scripts/test-services.sh)"` (출력이 비면 준비 실패 — eval 하지 말 것). 같은 셸에서 아래를 돌린다.
  - 실패 재현(수정 전 코드로 먼저): `go test -race ./internal/httpserver -run '^TestIntegrationUserInfoCountsTheLookupsItCouldNotMake$' -count=1 -v`
  - 회귀: `go test -race ./internal/httpserver -run '^TestIntegrationUserInfo' -count=1 -v` (SKIP 0 확인)
  - 전체: `make lint` (0 issues), `make test` (exit 0, 끝의 "integration test(s) did not run" 경고가 **없어야** SKIP 0; httpserver 약 110~120s), 그리고 빌드가 바꾼 `webui/dist/index.html` 은 되돌릴 것 + `git diff --check`.
- 확인한 사실 (이번 정찰이 실제로 열어 본 것):
  - `writeUserInfoUnavailable` 호출자는 여섯 곳뿐이고(oidc.go:839·878·892·906·938·943) **모두 `store.ErrNotFound`(또는 그에 해당하는 분기)를 이미 앞에서 걸러 401로 보낸다** — 839는 `!errors.Is(err, store.ErrNotFound)`, 878은 `ressooidc.ErrTokenStateUnavailable` 센티넬, 892·906도 명시적 `!errors.Is(..., ErrNotFound)`. 938·943(Role 조회)은 필터가 없지만 지금도 그 오류는 그대로 500이 된다. 따라서 `recordUnjudgedIntrospection`(1088행)처럼 헬퍼 안에서 `ErrNotFound`를 거르는 코드는 **필요 없다**(introspection은 필터 앞에서 불려서 필요했던 것) — 필터를 넣으면 죽은 코드가 되고 Role 경로의 계약만 흐려진다. 이 판단을 주석 한 줄로 남길 것.
  - `New`(server.go:64-68)는 `metrics == nil` 이면 `observability.NewRegistry()` 를 스스로 만들고 `registerMetrics` 를 부른다 — `s.metrics.Add` 는 언제나 안전하다. 또 `func (s *Server) Metrics() *observability.Registry`(server.go:80)가 있으므로 테스트는 **등록기를 직접 넘기지 않아도** `srv := New(data, logger, nil, nil); httptest.NewServer(srv.Handler())` 뒤 `srv.Metrics().WritePrometheus(&sb)` 로 읽을 수 있다(`WritePrometheus`는 observability/metrics.go:132, `io.Writer` 인자). 기존 선례대로 `observability.NewRegistry()` 를 넘겨도 된다 — 둘 중 하나만 고를 것.
  - `observability.Registry.Add`(metrics.go:86)는 등록되지 않은 이름을 조용히 버린다. 즉 `registerMetrics` 에 `Counter` 줄을 빠뜨리면 테스트가 0을 보고 실패한다(무음 실패 아님).
  - RENAME 재현 대상: `user_roles`·`user_client_roles`(10414행의 기존 선례 그대로 → stage `realm_roles`·`client_roles`), `users`(→ stage `user`, 선례 9ac4615). `realms` RENAME은 stage `realm`(선례 b762ea8)이나 다른 미들웨어·핸들러도 같이 무너지므로 굳이 넣지 말 것.
- 위험과 피할 것:
  - **라벨 카디널리티**: `stage` 에는 코드 리터럴 여섯만 들어간다. 요청에서 온 값(realm 이름·client_id·토큰)을 라벨에 넣지 말 것 — metrics.go의 `methodLabel`·`routePattern` 주석이 이 규칙의 근거다.
  - 401/400 경로에 지표를 달지 말 것. Bearer 토큰은 인증 없이 아무나 보낼 수 있어 그쪽을 세면 외부인이 경보를 울릴 수 있다(operations.md 91행의 `auth_time` 사고가 바로 그 사례다).
  - `oidcUserInfo` 의 조회 순서·401 판정·POST 폼 1MiB 상한·토큰 이중 제시 400 은 손대지 말 것. 헬퍼 시그니처 변경도 금지(호출자 여섯 곳으로 번진다).
  - 보호 경로(auth.go·migrations·.github/workflows) 는 건드리지 않는다. 이번 과제는 그럴 이유가 없다.
  - `make test` 뒤 `webui/dist/index.html` 이 바뀌어 있으면 되돌릴 것(이 저장소의 반복 실수).
  - 새 테스트가 지표를 **전역 누적**으로 읽는다는 점 주의: 한 서버 인스턴스의 등록기이므로 같은 테스트 안의 앞선 요청이 남긴 값이 더해진다. 단언은 "정확히 1" 대신 각 단계 전후의 증분으로 쓰거나, 순서를 고정해 누적값을 단언할 것.
- 차선 후보: 기존 Token 연동 테스트 둘(`TestIntegrationTokenSaysWhenItCouldNotReadTheAccount`:1130 부근·`TestIntegrationTokenSaysWhenItCouldNotReadTheRealm`)이 `New(data, logger, nil, nil)` 로 서버를 세워 프로덕션이 실제로 세는 `resso_token_errors_total{grant_type}` 를 단언하지 못하는 것을 고치기 — 위에서 확인한 `srv.Metrics()` 로 등록기를 읽으면 테스트 파일 하나만 바뀌고 프로덕션 위험 0이다(가치 2 / 위험 1 / 작업량 S).
