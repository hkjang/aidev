# 과제서 (2026-10-06, base main@003aa7b = v0.2.29)

- 과제: OIDC 일회용 코드 교환(`oidcExchange`)이 저장소 장애를 "코드 만료"·"계정 비활성화"로 보고하는 두 자리 분리 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/httpapi/oidc.go:433` `oidcExchange` 는 저장소 읽기 두 번의 **모든** 오류를 인증 거절로 접는다. (1) `s.store.ConsumeOIDCLoginCode`(440-443)는 코드가 없거나 만료됐을 때만 `store.ErrUnauthorized` 를 돌려주고(store/oidc.go:61-62) `Begin`·`Scan`·`Commit` 실패는 드라이버 원문 오류를 돌려주는데(store/oidc.go:52-54·64·67-68), 핸들러는 둘 다 401 `invalid_login_code` "로그인 코드가 만료되었거나 이미 사용되었습니다" 로 보고한다. (2) `s.store.GetUser`(445-449)의 오류와 `!user.Active` 가 한 조건으로 묶여 401 `inactive_user` "사용자 계정이 비활성화되었습니다" 가 된다. 그래서 PostgreSQL 이 흔들리는 동안 OIDC 로그인에 성공한 사용자는 "코드가 만료됐다"/"계정이 비활성화됐다"는 거짓 안내를 받고 재시도해도 같은 말을 듣는데, 운영자 쪽에는 **ERROR 로그가 한 줄도 남지 않는다**(이 함수의 그 아래 `SecurityConfig`·`CreateSession` 은 이미 `s.storeError` 로 500 + ERROR 를 남긴다 — 같은 함수 안에서 앞 두 읽기만 예외다). 고치면 장애가 장애로 보고되고(500 `internal_error` + `request_id` 붙은 ERROR 한 줄) 클라이언트가 재시도 가능한 실패와 영구 거절을 구분할 수 있다. 다른 `GetUser` 호출자는 이미 이렇게 한다 — `policySimulate`(policy_simulator.go:24-28)는 `GetUser` 오류를 그대로 `s.storeError` 로 넘긴다(직접 열어 확인). 즉 oidcExchange 가 예외다. 이것은 이 저장소의 반복 결함 가족("원인 X 를 원인 Y 로 보고": v0.2.20 저장소 장애→401/403, v0.2.28 입력 오류→Secret 혼입, v0.2.29 설정 읽기 실패→SSO 비활성)의 바로 다음 사례이고, v0.2.29 가 넣은 `oidc_outage_test.go` 하네스를 그대로 재사용한다.

- 수용 기준:
  1) `POST /api/v1/oidc/exchange` 에서 `ConsumeOIDCLoginCode` 가 드라이버 오류(예: `SQLSTATE 28P01`)를 돌려주면 응답이 **500 `internal_error`**, message 는 `요청을 처리하지 못했습니다`, 본문에 `postgres.internal`·`SQLSTATE`·`jikim_app` 이 없고, `error`·`request_id` 를 가진 ERROR 로그가 **정확히 한 줄** 남는다.
  2) 같은 호출에서 `GetUser` 가 드라이버 오류를 돌려줄 때도 1)과 같다.
  3) `ConsumeOIDCLoginCode` 가 `store.ErrUnauthorized`(코드 없음·만료)를 돌려줄 때의 응답은 **바이트 단위로 그대로** — 401 + code `invalid_login_code` + message `로그인 코드가 만료되었거나 이미 사용되었습니다`, ERROR 로그 0줄.
  4) `GetUser` 가 `store.ErrNotFound` 를 돌려줄 때와 `user.Active == false` 일 때의 응답도 그대로 — 401 + code `inactive_user` + message `사용자 계정이 비활성화되었습니다`, ERROR 로그 0줄. (삭제된 사용자에게 새 code 를 주는 것은 **이번 범위 밖**이다. 아래 "위험과 피할 것" 참고.)
  5) 테스트가 증명할 것: 위 다섯 경우를 **실제 핸들러·미들웨어 왕복**(`server.requestID(http.HandlerFunc(server.oidcExchange))`)으로 돌려 status·code·message·ERROR 줄 수를 본다. 수정 전에 먼저 써서 1)·2)가 실패하는 것을 보고(`code=invalid_login_code, want "internal_error"` 류), 고친 뒤 통과, 분기를 되돌리면 같은 둘이 다시 실패하는 것까지 확인한다.

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `internal/httpapi/server.go`:`Server` 구조체(28-48 의 seam 블록) — 함수 필드 2개 추가. 기존 이름 관례에 맞춰 `oidcCodeConsumer func(context.Context, string) (string, error)` 와 `userLoader func(context.Context, string) (model.User, error)`. (`model` 은 이미 import 되어 있다.)
  - `internal/httpapi/oidc.go`:`openOIDCState`(385-390) 바로 옆에 같은 모양의 nil-폴백 헬퍼 2개 — `func (s *Server) consumeOIDCLoginCode(ctx, code) (string, error)` → `s.store.ConsumeOIDCLoginCode`, `func (s *Server) loadUser(ctx, id) (model.User, error)` → `s.store.GetUser`.
  - `internal/httpapi/oidc.go`:`oidcExchange`(433-463) — 두 분기만 쪼갠다. 다른 줄은 손대지 않는다:
    ```go
    userID, err := s.consumeOIDCLoginCode(r.Context(), input.Code)
    if err != nil {
        if !errors.Is(err, store.ErrUnauthorized) {
            s.storeError(w, r, err)
            return
        }
        writeError(w, r, http.StatusUnauthorized, "invalid_login_code", "로그인 코드가 만료되었거나 이미 사용되었습니다")
        return
    }
    user, err := s.loadUser(r.Context(), userID)
    if err != nil && !errors.Is(err, store.ErrNotFound) {
        s.storeError(w, r, err)
        return
    }
    if err != nil || !user.Active {
        writeError(w, r, http.StatusUnauthorized, "inactive_user", "사용자 계정이 비활성화되었습니다")
        return
    }
    ```
    `errors`·`store` 는 oidc.go 에 이미 import 되어 있다(oidc.go:8·16). `storeError`(server.go:457-465)가 500 일 때만 message 를 `요청을 처리하지 못했습니다` 로 바꾸고 ERROR 를 남긴다 — 그래서 수용 기준 1)이 추가 로그 코드 없이 성립한다.
  - `internal/httpapi/oidc_exchange_outage_test.go`(신규) — 아래 하네스를 **그대로** 쓴다. 전부 확인한 것들이다:
    `quietServer()`(auth_outage_test.go:16, store 는 nil — 이 테스트는 seam 두 개만 쓰므로 nil store 에 닿지 않는다), `captureWebhookLog(server)`(webhook_test.go:88-130), `decodeErrorCode`·`assertNoDriverDetail`·`assertSingleStoreErrorLog`(oidc_outage_test.go:26-47·133-150), `driverFailure`(mcp_tool_errors_test.go:16 = `SQLSTATE 28P01` 포함), `webhookWarnings`. 요청 본문은 `{"code":"login-code"}` — `decodeJSON`(response.go:28-40)이 `DisallowUnknownFields()` 이므로 필드를 더 넣지 말 것. `assertSingleStoreErrorLog` 는 ERROR 줄에 `SQLSTATE 28P01` 이 있기를 요구하므로 장애 케이스는 `driverFailure` 를 쓸 것. 성공 거절 케이스(기준 3·4)는 ERROR 0줄을 직접 센다 — `assertSingleStoreErrorLog` 를 쓰지 말 것.
    `GetUser` 케이스의 `model.User` 는 `model.User{ID: "u-1", Active: false}` / `{Active: true}` 로 직접 만들면 된다(Active=true 경로는 그 다음 `s.store.SecurityConfig` 에서 nil store 로 panic 하므로 **happy path 는 이 테스트에 넣지 말 것** — 거절·장애 다섯 경우만 본다).

- 검증 명령 (이 저장소에서 실제로 도는 것, 하나씩 실행):
  - `cd /home/hkjang/.cache/auto-improve-wt/jikim`
  - `go test ./internal/httpapi/ -run 'OIDC' -count=1 -v` (수정 전 실패 재현 → 수정 후 통과)
  - `go test ./internal/httpapi/ -count=1` (패키지 전체 약 0.7초)
  - `go test ./internal/httpapi/ -race -count=5`
  - `go test ./... -count=1` / `go vet ./...` / `gofmt -l .`(출력 없어야 함)
  - `bash scripts/verify.sh` (프런트 59개 포함, v0.2.28·v0.2.29 회차에서 exit 0 확인됨 — `./scripts/verify.sh` 직접 실행은 Bash 도구가 막으므로 `bash` 로 부를 것)

- 위험과 피할 것:
  - **문구·code 를 새로 만들지 말 것.** `invalid_login_code`·`inactive_user` 와 두 한국어 메시지는 바이트 단위로 유지한다. `inactive_user` 는 저장소 전체에서 oidc.go:447 한 곳뿐이고 `web/src`·`docs/` 어디에도 계약으로 적혀 있지 않다(확인: grep `inactive_user` → 소스 1곳). 즉 프런트(`web/src/pages/OidcCallbackPage.tsx:21`)는 code 를 분기하지 않으므로 `web/` 를 건드릴 필요가 없다.
  - **`ErrNotFound`(삭제된 사용자)를 새 code 로 쪼개지 말 것** — 범위가 늘고 사용자 존재 여부를 노출하는 새 계약이 된다. 이번엔 기준 4)대로 기존 401 `inactive_user` 로 남긴다.
  - `store/oidc.go`·`store/users.go`·`migrations/` 는 손대지 말 것. `ConsumeOIDCLoginCode` 의 `ErrUnauthorized` 반환(store/oidc.go:62)과 `mapError`(store/store.go:173-188)는 그대로 두고 핸들러에서만 가린다.
  - `oidcLogin`(132)·`oidcCallback`(267)의 `loadOIDCConfig` 분기와 `oidc_outage_test.go` 는 v0.2.29 가 못박은 자리다 — 읽고 재사용만 하고 수정하지 말 것.
  - 보호 경로 인접: 이 핸들러는 세션을 발급한다. `setSessionCookie`·`CreateSession`·`SecurityConfig` 블록(450-462)은 한 글자도 바꾸지 말 것. `storeError` 자체도 바꾸지 말 것(다른 핸들러 수십 곳이 공유한다).
  - 교훈: 손으로 만든 대역이 아니라 실제 `Server` + 실제 미들웨어로 왕복시킬 것(이 저장소 표준). grep 결과를 증거로 제출하지 말고 테스트 출력으로 증명할 것. `CHANGELOG.md`·`scripts/version.sh` 는 작업 커밋에서 건드리지 말 것(릴리즈는 별도 커밋).
  - 미확인: 실제 PostgreSQL 장애 중 `pgx` 가 돌려주는 오류 종류는 이 환경(DSN 없음)에서 확인할 수 없다 — `driverFailure` 로 대리 검증한다. 실제 Keycloak 왕복도 미확인.

- 차선 후보: `oidcLogout`(oidc.go:406-410)이 `OIDCConfig` 읽기 실패와 "발급자 미설정"을 모두 조용히 `/login` 리다이렉트로 접는 자리 — 성공·실패가 같은 302 라 관찰 가능한 차이를 만들려면 WARN 로그라는 새 계약이 필요하다(가치 2 / 위험 2 / 작업량 S). 1순위의 분기 쪼개기가 기존 응답을 깨뜨리는 것으로 드러나면 이쪽으로 갈 것. 세 번째 후보는 `docs/guides/api-guide.md` 에 AI 엔드포인트 error code 표 추가(가치 2 / 위험 1 / 작업량 S)지만 관찰 가능한 동작 변화가 없어 운영자 반려 사유에 가깝다.
