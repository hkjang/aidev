- 과제: oidcCORS가 조회 실패로 CORS 헤더를 뺀 것을 기록하기 — `realmFromPath` 오류와 `WebOriginAllowed` 오류 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/httpserver/middleware.go:87` `oidcCORS`는 두 오류를 통째로 버린다 — 98행 `if realm, err := s.realmFromPath(r); err == nil`(err 미사용)과 99행 `if allowed, allowErr := s.store.WebOriginAllowed(...); allowErr == nil && allowed`. 그래서 **등록된 오리진인데 이쪽 DB가 답하지 않은 경우**가 "등록 안 된 오리진"과 완전히 같은 무음 결과(헤더 없음)가 되고, 브라우저는 불투명한 CORS 오류만 보여 RP는 자기 설정을 의심하며, 이쪽에는 로그·감사·지표 어디에도 한 줄이 남지 않는다 — 미들웨어가 헤더만 빼고 통과시키므로 다음 핸들러는 200을 내고 접근 로그는 건강해 보인다.
  이 저장소는 같은 구분을 이미 함수로 갖고 있다: `internal/httpserver/oidc.go:60` `realmLookupFailed(r, endpoint, err)`("ErrNotFound면 진짜 없는 것, 아니면 이쪽 장애"를 판정하고 `trace_id`·`realm`·`endpoint`·`error`로 Error 한 줄을 남긴다). discovery(72행)·jwks(132행)·authorization(189행)·revocation(999행)·logout(1155행)이 모두 쓰는데 **`realmFromPath` 호출자 중 `oidcCORS`만 이 구분을 하지 않는다.**

- 수용 기준:
  1) `realmFromPath`가 `store.ErrNotFound`가 **아닌** 오류를 낼 때 `oidcCORS`가 기존 `s.realmLookupFailed(r, "cors", err)`를 호출해 Error 한 줄(`endpoint=cors`)을 남긴다. `ErrNotFound`(없는·꺼진 Realm)에는 아무것도 남기지 않는다 — 그 경우 다음 핸들러가 이미 404/401로 답하므로 중복이다.
  2) `WebOriginAllowed`가 오류를 낼 때 전용 Error 한 줄을 남긴다(`trace_id`·`realm`·`error` 포함). 오리진이 단순히 등록되지 않은 경우(`allowErr == nil && !allowed`)에는 **아무것도 남기지 않는다** — 인증 없는 호출자가 Origin 헤더만 바꿔 로그를 채울 수 있는 자리다.
  3) 응답은 한 글자도 바뀌지 않는다: 두 실패 모두 지금처럼 CORS 헤더 없이 `next.ServeHTTP`로 통과한다(fail-closed 유지). `Vary: Origin`은 항상 그대로 추가되고, 허용 경로의 헤더 넷(`Access-Control-Allow-Origin/Headers/Methods/Max-Age`)과 값도 그대로다.
  4) 테스트가 증명할 것: (a) 정상 경로 기준선 — 등록된 오리진은 `Access-Control-Allow-Origin`을 받고 로그에 새 줄이 없다, (b) 등록 안 된 오리진은 헤더가 없고 **로그에도 새 줄이 없다**(noise 회피가 계약임), (c) `WebOriginAllowed`가 쓰는 테이블을 RENAME 한 뒤 **등록된** 오리진으로 같은 요청을 보내면 헤더는 없고 전용 Error 줄이 남으며 **응답 상태 코드는 (a)와 같다**, (d) 없는 Realm 이름(`ErrNotFound`)으로 요청하면 `endpoint=cors` 줄이 남지 **않는다**.

- 건드릴 파일:
  - `internal/httpserver/middleware.go:87` `oidcCORS` — 98·99행의 `err`/`allowErr`를 풀어 쓰고 위 두 자리에만 기록 추가. 이 함수 안에서만 고칠 것(`realmFromPath`·`realmLookupFailed` 시그니처는 그대로 — 호출자가 6곳이 된다).
  - `internal/httpserver/integration_test.go` — 새 테스트 하나 추가. 로그 캡처는 기존 `lockedBuffer`(6904행, 사용처 6982·7203행)를 그대로 쓸 것. CORS 기준선은 기존 `TestIntegrationJWKSIsCacheableNoLongerThanAKeySetCanBeStale`(9344행)이 `/realms/master/protocol/openid-connect/certs`로 `https://spa.example.com`(등록됨)·`https://attacker.example.com`(미등록)을 이미 쓰고 있으니 그 시드와 경로를 재사용하면 된다(기존 테스트는 고치지 말고 새로 쓸 것).
  - 주석: 이 저장소는 "왜 이 답을 골랐는가"를 서술형으로 길게 적는다. 특히 **미등록 오리진에 일부러 기록하지 않는 이유**를 남길 것.

- 검증 명령:
  1) `eval "$(scripts/test-services.sh)"` — 출력이 비면 준비 실패이므로 eval 하지 말 것. **같은 셸에서** 아래를 실행.
  2) 수정 **전** 새 테스트를 돌려 (c)의 단언이 실제로 실패하는 것을 먼저 확인할 것: `go test -race ./internal/httpserver -run '<새 테스트 이름>' -count=1 -v`
  3) `go test -race ./internal/httpserver -run '^TestIntegrationJWKSIsCacheableNoLongerThanAKeySetCanBeStale$|<새 테스트 이름>' -count=1 -v` — SKIP 0을 눈으로 확인.
  4) `make lint` (golangci-lint + govulncheck + ESLint), `make test` (전체 수 분, httpserver 약 115s).
  5) 빌드가 바꾼 `webui/dist/index.html`은 되돌릴 것. `git diff --check`.

- 위험과 피할 것:
  - **응답을 바꾸지 말 것.** 조회가 실패했다고 CORS를 허용하거나 요청을 거절하면 이것은 기록 과제가 아니라 정책 변경이 된다. 지금의 fail-closed를 그대로 둔다.
  - **오리진 원문을 로그에 넣지 말 것(또는 길이를 엄격히 제한할 것).** `Origin`은 검증되지 않은 호출자 입력이고, 운영자 규칙은 감사·로그로 넘기는 값을 식별자로 제한한다. `realm`(`chi.URLParam`)은 `realmLookupFailed`가 이미 남기고 있으니 선례가 있다.
  - `auth.go`(세션·쿠키), `oidc.go`의 토큰 발급·로그아웃 경로, `internal/store/migrations`, `.github/workflows/*`는 건드리지 말 것.
  - `oidcCORS`는 `server.go:108`에서 프로토콜 라우트 전체에 걸린다 — 요청마다 도는 자리라 기록은 장애 두 경우로만 좁혀야 한다(수용 기준 1·2).
  - `WebOriginAllowed`는 `internal/store/clients.go:348`에 있다. **어느 테이블을 읽는지 먼저 열어서 확인할 것** — (c)의 RENAME 대상이 `clients`인지 별도 origin 테이블인지 이 정찰에서는 미확인이다. 선례: `TestIntegrationLogoutSaysWhyItDroppedTheRedirect`가 `clients` RENAME으로 `client_unavailable`을 재현한다.
  - `gofmt` 정렬이 lint 첫 실행에서 자주 잡힌다.

- 차선 후보: **UserInfo 거절 카운터 `resso_userinfo_errors_total`** — `internal/httpserver/metrics.go`에 token·introspection·authorization·silent 네 계열이 있는데 userinfo에만 없다. `writeUserInfoUnavailable`(oidc.go:855)은 이미 `stage`(realm/revocation_state/user/session/realm_roles/client_roles)를 로그로만 남긴다. 다만 UserInfo의 500·401은 `resso_http_requests_total{route,status}`에 이미 보이므로 카운터가 더하는 것은 "몇 건인가"가 아니라 "여섯 검사 중 어디서 걸렸나"다 — 1순위보다 가치가 낮다. 라벨 카디널리티는 고정 집합으로 묶을 것.
