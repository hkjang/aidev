# 정찰 과제서 — 2026-09-23 (ReSSO, base main@73398a1 / v0.9.89)

- 과제: 로그아웃이 `post_logout_redirect_uri`를 버린 이유를 LOGOUT 감사 항목과 로그에 남기기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `oidc.go:1109`의 `redirectTo` 결정은 세 조건(`requested != ""` · `client != nil` · `PostLogoutURIAllowed`)이 모두 맞아야 목적지를 채우고, 하나라도 어긋나면 **아무 말 없이** 빈 문자열로 남아 브라우저가 RP로 돌아가지 못하고 이 서비스의 `/login?logged_out=1`(GET) 또는 204(POST)에서 멈춘다. `logoutClient`의 주석(`oidc.go:1158-1170`)이 이미 이 증상을 "the browser is stranded on this service's page at the end of a logout that otherwise worked"라고 적어 두었지만, 실제로 기록되는 것은 **저장소 오류 한 가지뿐**이다(`oidc.go:1174-1177`, `ErrNotFound`는 로그 없음). 즉 ① 모르는/꺼진 `client_id`·검증 안 되는 `id_token_hint`로 Client가 나오지 않은 경우와 ② Client는 찾았지만 요청한 URI가 등록 목록에 없는 경우(`store.PostLogoutURIAllowed` 는 `slices.Contains` 정확 일치, `clients.go:363`)가 **로그에도 감사 트레일에도 전혀 남지 않는다**. 이 둘은 RP 설정 오류에서 가장 흔한 로그아웃 지원 문의인데, 현재 LOGOUT 감사 항목은 파라미터 없는 평범한 로그아웃과 글자 하나 다르지 않아 운영자가 "왜 안 돌아왔는지"를 알 방법이 없다.

- 수용 기준:
  1) 등록된 `post_logout_redirect_uri`로의 로그아웃은 **응답이 지금과 완전히 같다**(302 + `state` 전달, POST는 204) — 감사 detail에 새 표시가 붙지 않는다.
  2) 요청에 `post_logout_redirect_uri`가 있는데 버려진 경우, **상태 코드·리다이렉트 동작은 그대로 두고**(거절하지 않는다) LOGOUT 감사 항목의 detail에 왜 버렸는지를 나타내는 값이 붙는다. 최소한 두 가지를 구분할 것: Client 자체가 없어서(모르는/꺼진 client_id, 검증 실패한 id_token_hint) → 예 `post_logout_redirect_uri: "dropped"`, `reason: "client_unknown"` / Client는 있으나 URI 미등록 → `reason: "uri_not_registered"`. 같은 사실이 `s.logger.Warn`(또는 Error) 한 줄로도 남아야 한다 — 쿠키 세션이 없어 감사 항목 자체가 안 생기는 경로가 있기 때문(`oidc.go`의 `sessionErr`가 `store.ErrNotFound`인 분기는 audit을 호출하지 않는다).
  3) **요청된 URI 원문을 감사 detail에 넣지 않는다.** 호출자가 마음대로 넣는 무제한 문자열이라 트레일에 저장·재출력된다. 식별자(`client_id`, realm)와 이유 코드만 넘긴다. 테스트가 이것을 단언할 것(detail에 요청 URI 문자열이 들어 있지 않음).
  4) 테스트는 실제 PostgreSQL·`New(...).Handler()`·`httptest` 서버를 지나는 통합 테스트로, **수정 전 코드에서 새 단언이 실제로 실패함을 확인**하고 그 사실을 회차 노트에 적을 것.

- 건드릴 파일:
  - `internal/httpserver/oidc.go:oidcLogout` — `redirectTo` 결정부(현재 `if requested := values.Get("post_logout_redirect_uri"); requested != "" && client != nil && store.PostLogoutURIAllowed(*client, requested)`)를 세 조건이 각각 왜 실패했는지 알 수 있게 풀어 쓰고, 버린 이유를 지역 변수(예 `droppedReason string`)에 담는다. 그 아래 두 LOGOUT `s.audit(...)` 호출(성공 경로 `partialIfNot(ended)` 와 세션 조회 실패 경로) 모두에 detail로 실어 준다. **`endSession`은 성공 시 detail로 `nil`을 돌려주므로**(`internal/httpserver/personal.go:155-170`) nil 맵에 쓰지 말고 필요할 때 새로 만들 것.
  - `internal/httpserver/oidc.go:logoutClient` — `ErrNotFound`·`!found.Enabled`로 nil을 돌려주는 두 경로가 지금 아무 말도 하지 않는다. 호출자가 이유를 구분할 수 있게 하거나(반환값 추가), 최소한 로그 한 줄을 남긴다. 시그니처를 바꾼다면 호출자는 `oidcLogout` 두 곳뿐이다(`id_token_hint` 분기와 `client_id` 분기).
  - `internal/httpserver/integration_test.go` — 새 통합 테스트 하나. 기존 `TestIntegrationLogoutTakesTheHintARelyingPartyActuallyHolds`(6810행)가 Realm 부트스트랩 → `EnsureActiveSigningKey` → `CreateClient`(`PostLogoutRedirectURIs`) → `CreateUser` → `CreateSession` → `ressooidc.Service{Store: data}.IssueUserTokens` → `httptest.NewServer(New(data, logger, nil, nil).Handler())` + `CheckRedirect = ErrUseLastResponse` 의 골격을 그대로 제공하므로 복사해 쓸 것. 감사 항목은 store의 감사 조회로 읽는다(다른 통합 테스트가 쓰는 방식을 따를 것 — 조회 함수 이름은 **미확인**이니 `integration_test.go`에서 `AUDIT`/`audit` 검색으로 확인).
  - 문서: `docs/operations.md` 의 로그아웃 관련 절에 "RP로 돌아가지 못하고 로그인 화면에 멈춘다"를 이유 코드로 진단하는 한두 문장. (문장만 더하므로 PDF·캡처 재생성 없음 — 이전 회차 관례.)

- 검증 명령:
  ```
  eval "$(scripts/test-services.sh)"     # 같은 셸에서 계속할 것. 출력이 비면 준비 실패이므로 eval 하지 말 것
  go test -race ./internal/httpserver -run '^TestIntegrationLogout' -count=1 -v   # SKIP 0 이어야 함
  make lint                              # golangci-lint v2.13.1 / govulncheck / ESLint. gofmt 정렬이 자주 걸린다
  make test                              # 전 패키지 race + vet + vitest + build. 수 분 걸림
  git diff --check && git status --short # 빌드가 바꾸는 webui/dist/index.html 변경은 되돌릴 것
  ```

- 위험과 피할 것:
  - **거절하지 말 것.** 이 파일은 "로그아웃을 일찍 끊으면 세션이 살아남고 트레일에 아무것도 안 남는다"를 이미 두 번 명시적으로 거부했다(`oidc.go`의 realm 조회 404 주석, `endSession` 주석). 400/404를 새로 만들지 말고 기록만 더할 것. 상태 코드·리다이렉트·쿠키 삭제(`clearBrowserCookies`) 순서는 그대로.
  - `PostLogoutURIAllowed`의 정확 일치 규칙을 느슨하게(prefix·정규화) 바꾸지 말 것 — 오픈 리다이렉트로 이어진다. 이번 과제는 **왜 막혔는지 기록**이지 **더 허용**이 아니다.
  - `id_token_hint`를 `Verify`로 되돌리지 말 것(만료된 ID 토큰이 정상 입력이라는 이유가 주석에 남아 있다). `oidcLogout`의 `sub`/`aud` 대조는 정책 변경이므로 이번 범위 밖.
  - 감사 detail에 원문 문자열 금지(운영자 반복 지시). 이유 코드와 식별자만.
  - `internal/store/migrations`, `.github/workflows/*`, `internal/httpserver/auth.go` 는 건드리지 말 것.
  - 로그 레벨: 이 파일은 실패에 `logger.Error`를 쓴다. RP 설정 오류는 서비스 장애가 아니므로 `Warn`이 더 맞지만, 기존 `logoutClient`의 저장소 오류 로그가 Error인 점과 섞이지 않게 한 가지로 정할 것(**어느 쪽이 관례인지는 미확인**).
  - `_ = r.ParseForm()`(`oidc.go:1070`)의 오류 무시는 **이번 과제가 아니다**. 같이 고치려 들지 말 것.

- 차선 후보: `oidcLogout`의 POST 폼을 읽지 못했을 때(1MiB 초과·깨진 인코딩) 파라미터가 통째로 사라진 것을 같은 방식으로 기록 (가치 2 / 위험 1 / 작업량 S) — `oidc.go:1069-1071`이 `_ = r.ParseForm()`로 오류를 버려, 본문이 안 읽히면 `id_token_hint`·`post_logout_redirect_uri`가 조용히 없는 것이 되고 세션만 끝난다. 1순위와 같은 자리·같은 기법이며, 1순위가 성립하지 않을 때(예: 이미 기록되고 있었다면) 이것을 할 것. 여기서도 **거절이 아니라 기록**이다.
