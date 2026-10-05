- 과제: OIDC 로그인 시작·콜백이 설정 읽기 실패를 "OIDC 로그인이 설정되지 않았습니다/비활성화되었습니다"로 보고하는 문제 분리 (가치 3 / 위험 2 / 작업량 S)
- 왜: `oidcLogin`(internal/httpapi/oidc.go:121-125)과 `oidcCallback`(oidc.go:252-256)이 `s.store.OIDCConfig` 의 **모든** 오류를 `!cfg.Enabled` 와 한 조건(`if err != nil || !cfg.Enabled ...`)으로 묶어 503 `oidc_disabled` 로 내보낸다. 그래서 DB 장애나 `oidc_client_secret` 복호화 실패로 설정을 읽지 못한 순간, 멀쩡히 설정된 SSO 가 "설정되지 않았다/비활성화되었다"로 보고되어 관리자는 설정 화면을 다시 뒤지고 클라이언트는 재시도할 근거를 잃는다. 같은 파일의 `oidcPublicConfig`(oidc.go:43-47)·`oidcTest`(oidc.go:55-59)는 이미 `if err != nil && !errors.Is(err, store.ErrNotFound) { s.storeError(...) }` 로 "저장소 장애"와 "아직 설정 안 됨"을 구분하고, `oidcLogin` 자신도 같은 함수 뒤쪽 실패(144·149·154·167)는 전부 `s.storeError` 로 보고한다 — 즉 두 자리만 파일의 자기 관례에서 빠져 있다. 이 저장소의 반복 결함 가족(원인 X 를 원인 Y 로 보고)의 남은 사례다.

- 수용 기준:
  1) `oidcLogin` 이 설정 로더에서 `store.ErrNotFound` 가 **아닌** 오류를 받으면 503 `oidc_disabled` 가 아니라 `s.storeError` 결과(500 / code `internal_error` / message `요청을 처리하지 못했습니다`)를 돌려주고, 응답 본문에 주입한 드라이버 원문(예: `dial tcp 127.0.0.1:5432: connect: connection refused`)이 들어 있지 않다. `s.logger` 에는 ERROR 한 줄이 남는다(`storeError`, server.go:456-463).
  2) 설정이 아직 없을 때(로더가 `store.ErrNotFound`)와 설정이 꺼져 있을 때(`store.OIDCConfig{Enabled:false}`), 그리고 `Enabled:true` 지만 `IssuerURL`·`ClientID` 가 빈 경우는 **지금과 바이트 단위로 같은** 503 + `oidc_disabled` + `OIDC 로그인이 설정되지 않았습니다` 를 유지한다(새 설치의 동작이 변하지 않아야 한다 — `store.OIDCConfig`(internal/store/settings.go:629-633)는 `oidc` 설정 행이 없으면 `ErrNotFound` 를 그대로 올려 보낸다).
  3) `oidcCallback` 도 같은 분리를 한다: 유효한 state 쿠키를 통과한 뒤 설정 로더가 비-`ErrNotFound` 오류를 내면 500 `internal_error`, `ErrNotFound`/`Enabled:false` 면 지금과 같은 503 `oidc_disabled` + `OIDC 로그인이 비활성화되었습니다`.
  4) 테스트는 **수정 전에 먼저 실패**해야 한다: 두 분기 중 하나라도 `if err != nil || !cfg.Enabled` 로 되돌리면 같은 테스트가 같은 메시지로 다시 실패함을 확인하고 복원한다(지난 회차들의 역방향 확인 관례).
  5) 리다이렉트·쿠키(`jikim_oidc_state`)·state/nonce 검증·`validateOIDCRuntimeConfig` 분기·`oidc_discovery_failed`·`invalid_oidc_config` 의 상태코드와 문구는 한 글자도 바뀌지 않는다. `go test ./internal/httpapi/ -count=1` 과 `go vet ./...`·`gofmt -l .` 가 깨끗하다.

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `internal/httpapi/server.go` — `Server` 구조체(25-47 블록)에 seam 한 줄 추가: `oidcConfigLoader func(context.Context) (store.OIDCConfig, error)`. `New`(server.go:50-69)에서는 **배선하지 않는다**(nil-폴백 관례. `mcpOAuthLoader`(server.go:41 + mcp_oauth.go:73-74), `securityLoader`(server.go:31 + 255-256)와 같은 모양. `trackingLoader` 만 `New` 에서 배선되는 예외다).
  - `internal/httpapi/oidc.go` — ① nil-폴백 헬퍼 하나 추가(`oidcPublicConfig` 위, 파일 상단):
    ```go
    func (s *Server) loadOIDCConfig(ctx context.Context) (store.OIDCConfig, error) {
        if s.oidcConfigLoader != nil {
            return s.oidcConfigLoader(ctx)
        }
        return s.store.OIDCConfig(ctx)
    }
    ```
    ② `oidcLogin`(121-125)과 ③ `oidcCallback`(252-256)의 두 자리를 다음 모양으로 바꾼다(각 2~3줄):
    ```go
    cfg, err := s.loadOIDCConfig(r.Context())
    if err != nil && !errors.Is(err, store.ErrNotFound) {
        s.storeError(w, r, err)
        return
    }
    if !cfg.Enabled || cfg.IssuerURL == "" || cfg.ClientID == "" { // 콜백은 지금처럼 !cfg.Enabled 만
        writeError(w, r, http.StatusServiceUnavailable, "oidc_disabled", "...기존 문구 그대로...")
        return
    }
    ```
    `errors`·`store` 는 oidc.go 에 이미 import 되어 있다(44행에서 `errors.Is(err, store.ErrNotFound)` 를 쓴다). **콜백의 조건식에 `IssuerURL`·`ClientID` 검사를 새로 더하지 말 것** — 지금 `!cfg.Enabled` 뿐이고 그 뒤 `validateOIDCRuntimeConfig` 가 받는다.
  - `internal/httpapi/oidc_outage_test.go` (새 파일) — `quietServer()`(auth_outage_test.go:16-18, `&Server{logger: ...}` 뿐이라 store 는 nil 이지만 seam 이 채워지면 `s.store` 에 닿지 않는다)에 `oidcConfigLoader` 를 주입하고 실제 핸들러를 왕복시킨다. 콜백 테스트는 `sealedStateOpener(t, "sealed", oidcState{State:"abc", IssuedAt: time.Now().UTC()})`(oidc_silent_test.go:70-82)와 `oidcStateOpener` seam, 쿠키 `jikim_oidc_state=sealed`, 쿼리 `?state=abc`(**`error=` 파라미터 없이** — 있으면 225행에서 먼저 빠져나간다)로 252행까지 도달한다. 설정 로드는 provider 네트워크 호출(261행)보다 앞이라 외부 접속 없이 끝난다. 서브테스트 표는 `{오류, 기대 status, 기대 code}` 로: 비-ErrNotFound 오류 → 500/`internal_error`, `store.ErrNotFound` → 503/`oidc_disabled`, `OIDCConfig{Enabled:false}` → 503/`oidc_disabled`, `OIDCConfig{Enabled:true}`(Issuer·ClientID 빈 값, oidcLogin 만) → 503/`oidc_disabled`.

- 검증 명령 (이 저장소에서 실제로 도는 것. Bash 도구가 `&&` 복합 명령과 `./scripts/*.sh` 직접 실행을 막으므로 하나씩 실행하고 스크립트는 `bash scripts/...` 로 부를 것):
  - `go test ./internal/httpapi/ -run 'OIDC' -count=1 -v`
  - `go test ./internal/httpapi/ -count=1` (base a949932 에서 **0.739초, green 확인**)
  - `go test ./... -count=1` / `go vet ./...` / `gofmt -l .`(출력 없어야 함)
  - `go test ./internal/httpapi/ -race -count=5`
  - 마지막에 `bash scripts/verify.sh` (느리다. v0.2.28 base 에서 exit 0 로 끝나는 것이 지난 회차 기록이다 — 프런트 59/59. 이번 변경은 `web/` 를 건드리지 않으니 여기서 깨지면 환경 문제다)

- 위험과 피할 것:
  - `internal/httpapi/oidc*.go` 는 보호 경로다. **인증 판정을 바꾸지 말 것**: 모든 새 분기는 여전히 로그인을 거절한다(503 → 500 으로 "원인 이름"만 바뀐다). 토큰·세션 발급 경로(421-443)는 이번 범위가 아니다.
  - `oidcExchange`(oidc.go:426-430, `if err != nil || !user.Active` → `inactive_user`)와 `oidcLogout`(oidc.go:387-391)에도 같은 가족의 사례가 있지만 **이번에는 건드리지 말 것** — `GetUser` seam 이 더 필요해 파일·seam 수가 늘고, exchange 는 세션 발급 경로다. 다음 회차 후보로 ideas.json 에 남겨 두었다.
  - `errors.Is(err, store.ErrNotFound)` 를 빼먹고 모든 오류를 `storeError` 로 보내면 **설정 안 한 새 설치가 503 `oidc_disabled` 대신 404 `not_found`** 를 받는다(`errorStatus`, server.go:439-453). 수용 기준 2)가 이것을 막는 자리다.
  - `storeError` 는 500 일 때 본문 메시지를 `요청을 처리하지 못했습니다` 로 덮고 원문은 로그로만 보낸다(server.go:456-463). 원문이 본문으로 새지 않는지 테스트로 함께 못박을 것(auth_outage_test.go 의 선례).
  - `web/src`·`docs/` 어디에도 `oidc_disabled`·`invalid_oidc_config` 문자열이 없다(grep 확인) — 프런트/문서 계약을 깨지 않는다. 기존 테스트도 `oidc_disabled` 를 단언하지 않는다(httpapi `*_test.go` grep 확인).
  - `New` 에서 `s.oidcConfigLoader = st.OIDCConfig` 로 배선하지 말 것 — `st` 가 nil 인 테스트 경로에서 메서드 값이 nil 수신자를 잡아 두게 되고, 이 저장소의 seam 관례는 nil-폴백이다.
  - `scripts/e2e-docker.sh`·Playwright 는 `docs/screenshots/*.png` 를 덮어쓴다. 이번 과제에는 불필요하니 실행하지 말 것.

- 차선 후보: `docs/guides/api-guide.md` 에 AI 엔드포인트(`POST /api/v1/ai/chat`) error code 표 추가 — 적을 code 다섯이 소스에서 확정된다(`secret_material_rejected`·`invalid_ai_request`(ai.go:98-104), `ai_disabled`(ai.go:112), `invalid_ai_config`(ai.go:116·244), `ai_rate_limited`(ai_rate_limit.go 에서 확인 필요 — **미확인**)). 다만 관찰 가능한 코드 동작이 바뀌지 않는 문서 전용 변경이라 운영자 반려 사유에 가깝다. 1순위가 성립하지 않으면 `oidcExchange`(oidc.go:426-430)의 `GetUser` 실패/`!user.Active` 분리를 먼저 보라.
