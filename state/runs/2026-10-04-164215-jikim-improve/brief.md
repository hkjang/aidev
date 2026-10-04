- 과제: AI 채팅 입력 검증 실패를 전부 `secret_material_rejected` 로 보고하는 문제 분리 (가치 3 / 위험 1 / 작업량 S)
- 왜: `aiChat`(internal/httpapi/ai.go:93-96)이 `ValidateAIInput` 의 **모든** 오류를 error code `secret_material_rejected` 로 내보내, 빈 prompt·허용되지 않은 role·빈 message·262144바이트 초과·`max_tokens` 범위 초과 등 Secret 과 전혀 무관한 다섯 가지 거절까지 "Secret 평문이 섞였다"로 보고된다(읽어서 확인: ai.go:58·66·69·75·83 의 5개 `errors.New`/`fmt.Errorf` 중 Secret 매처는 77-81 한 곳뿐이고, 핸들러는 `err.Error()` 만 통과시킨다). 이 저장소의 반복 결함 가족("원인 X 를 원인 Y 로 보고")에 정확히 해당하고, 고치면 클라이언트·운영자가 400 응답의 code 로 "입력 형식이 틀렸다"와 "Secret 평문을 보냈다"를 구분할 수 있다.
- 근거 보강(이 자리가 유일한 예외라는 사실): `internal/httpapi` 의 `writeError(..., http.StatusBadRequest, ...)` 호출 24곳을 전부 열어 보면 나머지는 모두 자기 원인을 가리키는 code 를 쓴다(`weak_password`·`invalid_permissions`·`reason_required`·`invalid_origin`·`invalid_oidc_state`·`webhook_not_configured`·`ai_not_configured`·`invalid_ai_config` …). **한 원인의 code 를 다섯 원인에 돌려 쓰는 자리는 ai.go:94 뿐이다.**
- 상태 메모(중요): 이 과제는 2026-10-02 회차에서 구현까지 끝났으나 **회차 결과가 verify-failed** 라 머지되지 않았다. 확인한 것: `git log --oneline -12` 에 그 커밋이 없고(머지된 것은 그 다음 회차의 `7a13e85`·`2f18d04` 뿐), HEAD 2f18d04 에서 `grep -rn "secret_material_rejected\|errAISecretMaterial" internal` 은 **ai.go:94 한 줄만** 반환한다 — 즉 변경이 하나도 남아 있지 않다. 당시 실패 원인은 이 변경이 아니라 `bash scripts/verify.sh` 의 프런트 vitest 단계가 Node 20 으로 돌아 터진 환경 문제였고, 그 원인은 그 다음 회차 `7a13e85`(v0.2.27)에서 고쳐 **이미 HEAD 에 들어와 있다**(확인: `web/package.json:6-7` `engines.node ">=22.22.2"`, `web/package.json:13` `test` 가 `"${npm_node_execpath:-node}" ./node_modules/vitest/vitest.mjs run`, `scripts/verify.sh:47` `NODE_FLOOR='22.22.2'`, `web/vitest.config.ts:21` 의 프로세스 하한 가드). 즉 같은 접근이 **사람에게 반려된 것이 아니라** 검증 환경 때문에 떨어진 것이므로 이번에 다시 하되, **반드시 `bash scripts/verify.sh` 가 exit 0 으로 끝나는 것까지 확인**해서 같은 자리에서 또 떨어지지 않게 하라.

- 수용 기준:
  1) Secret 평문 분기(ai.go:77-81)만 기존과 **바이트 단위로 동일한** code `secret_material_rejected` + 한국어 메시지 `Secret 평문으로 보이는 내용은 AI에 전달할 수 없습니다` 로 400 을 반환한다.
  2) 나머지 네 거절(빈 입력, 허용되지 않은 role, 빈 message, 총 262144바이트 초과, `max_tokens` 범위 초과)은 **다른 code**(권장: `invalid_ai_request`)로 400 을 반환한다. 상태코드는 다섯 경우 모두 400 으로 그대로다.
  3) `ValidateAIInput` 의 시그니처(`func ValidateAIInput(input aiChatInput) error`)와 각 메시지 문자열은 바꾸지 않는다 — 분류만 바뀌고 사용자에게 보이는 문구는 그대로다. `secretMaterialPatterns`·`maximumAITokens`·262144 한도는 한 글자도 건드리지 않는다(보안 매처, 넓히거나 좁히지 말 것).
  4) 테스트가 증명할 것: 실제 라우트 배선(`s.requestID(s.withAuth(http.HandlerFunc(s.aiChat)))`)을 통한 왕복으로, 다섯 입력에 대해 응답 JSON 의 `error.code` 가 각각 기대값인지 본다. 수정 전에 먼저 써서 네 개가 실패하는 것을 확인하고, 핸들러 분기를 되돌리면 같은 네 개가 다시 실패하는 것도 확인한다.
  5) `bash scripts/verify.sh` 가 exit 0 (`검증 완료: jikim v0.2.27`)으로 끝난다.

- 권장 구현(한 가지 모양 — 2026-10-02 회차가 이 형태로 구현해 구현자 판정 "채택" 을 받았다):
  - `internal/httpapi/ai.go` 에 패키지 수준 sentinel `var errAISecretMaterial = errors.New("Secret 평문으로 보이는 내용은 AI에 전달할 수 없습니다")` 를 두고, 77-81 루프만 `return errAISecretMaterial` 로 바꾼다(나머지 네 `errors.New`/`fmt.Errorf` 는 그대로 — 메시지가 입력값에 따라 달라지는 자리를 sentinel 로 묶지 말 것).
  - 핸들러 93-96 을 `errors.Is(err, errAISecretMaterial)` 로 갈라 code 만 고른다. `errors` 는 이미 import 되어 있다(ai.go:8).

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/httpapi/ai.go:55-96` — `ValidateAIInput` 의 Secret 분기에 sentinel, `aiChat` 의 `writeError` 호출을 `errors.Is` 분기로. 이 파일의 다른 부분(`validateAIRuntimeConfig`, SSE 경로, `aiLimiter`)은 건드리지 말 것.
  - `internal/httpapi/ai_input_errors_test.go` (신규) — 아래 "검증 배선" 참고.

- 검증 배선 (실제로 열어 본 것):
  - 라우트: `internal/httpapi/server.go:141` = `mux.Handle("POST /api/v1/ai/chat", s.withAuth(http.HandlerFunc(s.aiChat)))`. **`requestID` 는 이 줄에 없다** — 서버 전체를 감싸는 쪽에 있으니, 테스트에서 직접 체인을 만들 때는 `s.requestID(s.withAuth(http.HandlerFunc(s.aiChat)))` 로 감싸면 `request_id` 까지 채워진 실제 왕복이 된다(`revokeServer` 를 쓰는 기존 테스트들이 이 형태다).
  - 거절은 `s.store.AIConfig(...)`(ai.go:97) **앞에서** 반환되므로 **store 가 nil 이어도** 다섯 경우 모두 핸들러를 왕복할 수 있다. 이것이 DB 없이 증명할 수 있는 이유다.
  - 하네스는 이 저장소 표준인 `quietServer()`(`auth_outage_test.go:16` — `&Server{logger:...}` 뿐, store 는 nil) + `sessionResolver` 하나만 주입하는 형태를 쓰고, 최신 표본은 `revokeServer`(`auth_revoke_test.go:16-29`)다. 그 안의 `server.sessionResolver = func(context.Context, string) (model.Session, error)` 가 `model.Session{User: model.User{ID:"user-1", Role:"admin", AuthSource: …}}` 를 반환하는 두 줄만 그대로 가져오면 된다. 손으로 만든 대역(FakeStore 류)을 새로 만들지 말 것 — 실제 핸들러·미들웨어를 돌려야 한다.
  - 인증 토큰: `requestToken`(`server.go:205-216`)이 `Authorization: Bearer <t>` → 쿠키 `jikim_session` → 헤더 `X-Vault-Token` 순으로 읽는다. 아무 non-empty 값이면 `sessionResolver` 가 통과시킨다(`withAuth` server.go:179-203).
  - 요청 본문은 `decodeJSON`(`response.go:28-40`)이 `DisallowUnknownFields()` 로 읽으므로 테스트 JSON 에 `aiChatInput`(ai.go:23-28)에 없는 키(`prompt`/`messages`/`max_tokens`/`context` 외)를 넣으면 `invalid_json` 으로 먼저 떨어져 검증 분기에 도달하지 못한다.
  - 응답 모양은 `writeError`(`response.go:22-26`) = `{"error":{"code","message","request_id"}}`.
  - 기존 `TestValidateAIInputMessagesAndTokenLimit`(`internal/httpapi/security_test.go:16`)은 `ValidateAIInput` 을 직접 부르고 메시지를 보므로 문구를 바꾸면 깨진다. **수정 없이 통과해야 한다.**
  - 기준선: base 2f18d04 에서 `go test ./internal/httpapi/ -count=1` → `ok … 0.682s` (정찰이 실제로 돌려 확인).
  - 262144바이트 초과 케이스를 쓸 때 주의: 긴 문자열은 `secretMaterialPatterns` 의 마지막 패턴 `\b[A-Za-z0-9+/_=-]{48,}\b` 에 걸려 **Secret 분기로 먼저 빠진다**(검사 순서가 크기→Secret 이라 크기 초과가 먼저 반환되지만, 분기 순서를 바꾸지 말 것). 공백이 섞인 문자열(예: `strings.Repeat("가 ", …)` 또는 `strings.Repeat("ab ", …)`)로 길이를 채워 의도한 분기를 때리는지 code 로 확인할 것.

- 검증 명령 (이 저장소에서 실제로 도는 것, 하나씩 실행):
  - `go test ./internal/httpapi/ -count=1`  (패키지 전체가 1초 안쪽)
  - `go test ./internal/httpapi/ -run 'AI|Ai' -count=1 -v`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `gofmt -l .`  (출력이 비어 있어야 한다)
  - `go test ./internal/httpapi/ -race -count=5`
  - `bash scripts/verify.sh`  (느리다 — `npm ci` 포함. v0.2.27 의 Node 가드가 들어온 뒤이므로 통과해야 한다. 여기서 떨어지면 지난 회차와 같은 자리이므로 로그를 그대로 남겨 보고할 것)

- 위험과 피할 것:
  - `secretMaterialPatterns`·`maximumAITokens`·262144 비교는 보안 매처다. 패턴을 넓히거나 좁히지 말고, 262144 리터럴(74행)과 `maximumAITokens`(21행)의 중복도 이번에 통합하지 말 것(관찰 가능한 동작이 안 바뀌는 리팩터는 반려 사유다).
  - 보호 경로(`auth_handlers.go`·`oidc*.go`·`mcp_oauth.go`·`store/users.go`·`migrations/`·`.github/workflows/`)는 건드리지 않는다. 이 과제는 그 경로에 닿지 않는다.
  - 새 code 문자열(`invalid_ai_request`)이 저장소에 이미 쓰이고 있지 않은지 먼저 확인할 것(`grep -rn invalid_ai_request .` — 정찰 시점 0건). `docs/guides/api-guide.md`·`docs/guides/compatibility.md` 가 AI 엔드포인트의 error code 를 계약으로 문서화해 두었는지 확인하고, 되어 있다면 같은 커밋에서 한 줄 갱신할 것(문서를 쓸 때 소스에서 확인하고 옛 가이드를 남기지 말 것).
  - CHANGELOG 는 작업 커밋에서 건드리지 않는다(릴리즈는 별도 커밋에서 `scripts/version.sh` + `CHANGELOG.md`).
  - `web/` 는 손대지 않는다. 프런트 검증은 `npm ci` 부터 시작해 시간이 든다.

- 차선 후보: **`AIConfig` 의 temperature 범위 클램프 누락 보강** (`internal/store/settings.go:566-610` 이 `MaxTokens`·`TimeoutSeconds` 만 클램프한다). 단, 쓰기 경로 `validateSetting`(settings.go:228-232)이 0~2 를 강제해 직접 DB 쓰기 없이는 도달 불가이므로 **관찰 가능한 변화를 먼저 증명**해야 한다 — 증명이 안 되면 고르지 말고, 대신 `docs/guides/api-guide.md` 의 AI 엔드포인트 error code 표를 소스에서 확인해 정확히 맞추는 문서 과제로 바꿀 것.
