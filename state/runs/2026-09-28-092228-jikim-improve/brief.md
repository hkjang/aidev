# 과제서 (2026-09-28, base main@185ad6a / v0.2.24)

- 과제: 로그아웃·토큰 폐기 실패(`RevokeToken`)를 조용히 버리는 세 자리를 헬퍼로 모아 경고로 남기기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `logout`(internal/httpapi/auth_handlers.go:103-111), `oidcLogout`(internal/httpapi/oidc.go:376-382), `baoRevokeSelf`(internal/httpapi/openbao.go:265-269) 세 곳 모두 `_ = s.store.RevokeToken(r.Context(), token)` 로 반환값을 버리므로, 저장소가 흔들려 폐기가 실패해도 쿠키만 지워지고 204/302 가 나간다 — 클라이언트는 로그아웃됐다고 믿지만 토큰은 TTL 이 끝날 때까지 계속 유효하고(같은 토큰을 헤더로 보내면 `withAuth` 를 그대로 통과한다) 운영자에게는 단서가 한 줄도 남지 않는다. Secret 관리 서버에서 "살아 있는 세션"은 사고 조사의 출발점이므로, 최소 변경으로 WARN 한 줄을 남겨 이 상태를 관측 가능하게 한다.
- 수용 기준:
  1) `sessionRevoker` 를 실패시킨 상태로 **실제 `s.withAuth(http.HandlerFunc(s.logout))`** 을 왕복시키면 응답은 **여전히 204** 이고 세션 쿠키 삭제(`jikim_session`, `MaxAge<0`) 도 그대로인데, `level=WARN` 줄이 **정확히 한 줄** 생긴다.
  2) 그 WARN 줄에 `error`·`request_id`·어느 경로인지 구분하는 식별 필드(예: `source`)가 있고, **토큰 문자열(`hvs.session-token`)은 로그 출력 어디에도 없다**(전체 버퍼 문자열에 `strings.Contains` 로 검사).
  3) 폐기가 성공하는 경로에서는 WARN 이 **0줄**이고 응답도 그대로다(회귀 없음).
  4) `baoRevokeSelf` 도 같은 실패 주입에서 204 + WARN 한 줄이 나온다. `oidcLogout` 은 `AuthSource != "oidc"` 인 세션이면 store 를 더 부르지 않고 `/login` 으로 302 하므로 같은 하네스로 왕복 가능하다 — 302 + WARN 한 줄을 확인한다.
  5) 헬퍼를 지우고 `_ =` 로 되돌리면 위 테스트들이 다시 실패하는 것을 확인한다(기존 `go test ./... -count=1` 은 계속 통과).
- 건드릴 파일 (프로덕션 4개, 각 1~8줄):
  - `internal/httpapi/server.go` — `Server` 구조체에 seam 필드 `sessionRevoker func(context.Context, string) error` 추가(28행의 `sessionResolver` 옆, 기존 nil-폴백 관례). 그리고 헬퍼 두 개:
    `func (s *Server) revokeSessionToken(ctx context.Context, token string) error { if s.sessionRevoker != nil { return s.sessionRevoker(ctx, token) }; return s.store.RevokeToken(ctx, token) }`
    `func (s *Server) revokeSessionTokenOrLog(r *http.Request, token, source string)` — `token == ""` 이면 아무것도 하지 않고, 실패하면 `s.logger.Warn("세션 토큰 폐기 실패", "error", err, "source", source, "request_id", requestIDFrom(r))`. (`resolveSession`/`storagePinger` 가 이미 같은 모양이다: server.go:218, 410.)
  - `internal/httpapi/auth_handlers.go:106` — `_ = s.store.RevokeToken(...)` → `s.revokeSessionTokenOrLog(r, token, "logout")` (바깥 `if token != ""` 는 그대로 두거나 헬퍼에 맡긴다. 둘 중 하나만).
  - `internal/httpapi/oidc.go:380` — 같은 치환, `source: "oidc_logout"`. 주석(376-378행의 "provider 장애가 로컬 세션을 살려두면 안 된다")과 이후 흐름은 손대지 않는다.
  - `internal/httpapi/openbao.go:267` — 같은 치환, `source: "openbao_revoke_self"`. 여기는 `token != ""` 가드가 없으므로 헬퍼의 빈 문자열 처리에 의존한다(동작 변화 없음: 빈 토큰으로 store 를 부르지 않게 되는 것은 라우트가 `withAuth` 라 실제로 도달 불가).
  - 테스트: `internal/httpapi/auth_revoke_test.go` (새 파일) — 기존 하네스를 그대로 재사용한다. `quietServer()`·`tokenRequest(method, path)`(auth_outage_test.go:16,20), `captureWebhookLog(server)`·`syncBuffer`·`webhookWarnings(t, logs)`(webhook_test.go:88-130, 같은 패키지이므로 그대로 호출 가능, JSON 핸들러로 WARN 줄을 파싱한다). `sessionResolver` 에 유효한 `model.Session`(User.ID 채우고 `AuthSource: "local"`)을 돌려주게 하면 `withAuth` 는 store 없이 통과한다(server.go:178-201 확인).
- 검증 명령 (하나씩 실행할 것 — 이 환경의 Bash 는 `&&` 복합 명령을 막는다):
  - `go test ./internal/httpapi/ -run 'Revoke|Logout' -count=1 -v`
  - `go test ./internal/httpapi/ -run 'Revoke|Logout' -race -count=5`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `gofmt -l .`
  - `./scripts/verify.sh` (npm ci 부터 도므로 오래 걸린다. `web/node_modules` 가 없다)
- 위험과 피할 것:
  - **보호 경로다**(auth_handlers.go·oidc.go·openbao.go). 추가는 **로깅뿐**이어야 한다 — 상태코드·쿠키·리다이렉트·`baoResponse` 본문·`withAuth` 판정은 한 글자도 바꾸지 말 것. 폐기 실패를 500 으로 바꾸는 것은 새 계약이므로 이번 범위 밖(로그아웃이 실패로 보이면 클라이언트가 쿠키를 지우지 않을 수 있다).
  - **토큰·쿠키 값·payload 를 로그에 넣지 말 것**(운영자 반복 지시: 감사·로그에는 식별자만). `source` 는 코드가 정하는 상수여야 하고 요청에서 오는 값을 넣지 말 것.
  - `r.Context()` 를 그대로 쓸 것. webhook 계열처럼 `context.Background()` 로 바꾸지 말 것 — 여기는 동기 요청 경로이고 "기록에 상한을 둔다/취소를 무시한다"는 새 정책이다.
  - seam 은 `sessionRevoker` **하나만** 추가한다. `store.OIDCConfig` 등을 위해 seam 을 더 늘려야 할 것 같으면 그 테스트를 포기하고(수용 기준 4 의 oidcLogout 부분) 나머지로 마감한 뒤 노트에 남길 것 — 프로덕션 파일은 4개를 넘기지 말 것.
  - `internal/store/` 와 `migrations/`, `web/` 은 건드리지 않는다. CHANGELOG 도 작업 커밋에서 건드리지 않는다(릴리즈 커밋에서만).
  - 미확인: `RevokeToken` 이 이미 폐기된 토큰에 대해 어떤 오류를 주는지(`ErrNotFound` 여부)는 확인하지 않았다. 오류 종류로 분기하지 말고 "오류면 WARN" 으로 단순하게 둘 것.
- 견적 근거(이 숫자가 무엇을 가정하는가):
  - 분해: ① seam+헬퍼(server.go) ② 치환 3줄 ③ 실패 주입 왕복 테스트 3개 ④ 되돌려 실패 재현 ⑤ `verify.sh`. 방법은 유추(analogous) — 같은 모양의 2026-09-25·09-27 회차 두 건이 각각 한 세션 안에 끝났다.
  - 범위: 코드·테스트 30~50분. 여기에 `./scripts/verify.sh`(npm ci 포함)가 더해진다 — 이것이 가장 큰 변동 요인이고 `web/node_modules` 가 없으므로 처음 한 번은 길다. 10회 중 8회는 한 세션 안에 끝난다고 본다.
  - 제외(이번 범위 아님): 폐기 실패의 HTTP 상태코드 변경, `RevokeToken` 재시도, `internal/store` 수정, 프런트 변경, CHANGELOG.
  - 여유(contingency): oidcLogout 테스트 1개. seam 이 더 필요해지면 이것만 버리고 나머지 수용 기준으로 마감한다 — 과제를 키우는 방향으로 쓰지 말 것.
- 차선 후보: `aiRequestLimiter` 의 사용자 표에 결정적 상한 세우기 (2/1/S) — `internal/httpapi/ai_rate_limit.go:45-51` 의 축출이 `len(l.users) > 4096` 일 때만, 그것도 `active == 0 && windowStart` 2분 초과 항목만 지우므로 최근 활동 사용자가 많으면 한 건도 지우지 못한다. `newAIRequestLimiter()` 의 `now` 필드를 주입하는 기존 `ai_rate_limit_test.go` 하네스로 "5000명 주입 후 표 크기 상한 유지"를 결정적으로 보일 수 있다(키가 인증 사용자 ID 라 외부 주도 증식은 불가 → 가치는 낮다).
