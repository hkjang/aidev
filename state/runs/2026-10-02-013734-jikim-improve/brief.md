- 과제: AI 채팅 입력 검증 실패를 전부 `secret_material_rejected` 로 보고하는 문제 분리 (가치 3 / 위험 1 / 작업량 S)
- 왜: `aiChat`(internal/httpapi/ai.go:93-96)이 `ValidateAIInput` 의 **모든** 오류를 error code `secret_material_rejected` 로 내보낸다. 빈 prompt·허용되지 않은 role(`tool` 등)·빈 message·262144바이트 초과·`max_tokens` 범위 초과까지 Secret 과 전혀 무관한 다섯 가지 거절이 "Secret 평문이 섞였다"로 보고되어, 기계 판독용 code 에 의존하는 클라이언트와 로그를 읽는 운영자가 자기 요청에 Secret 이 들어갔다고 오해한다. 이 저장소가 반복해 고쳐 온 결함(저장소 장애를 401/403 으로, 기록 실패를 전송 실패로 보고)과 같은 가족이다 — **원인 X 를 원인 Y 로 보고하는 것**.

- 수용 기준:
  1) Secret 과 무관한 거절(빈 입력·잘못된 role·빈 message·크기 초과·`max_tokens` 범위)은 `secret_material_rejected` 가 **아닌** code 로 나간다. `invalid_ai_request` 를 쓸 것(이웃 코드 `invalid_json`·`ai_disabled`·`invalid_ai_config` 와 같은 house style).
  2) 실제 Secret 패턴 요청(`secretMaterialPatterns` 적중)은 code `secret_material_rejected` 와 기존 한국어 메시지("Secret 평문으로 보이는 내용은 AI에 전달할 수 없습니다")를 **바이트 단위로 그대로** 유지한다. 이것이 유일하게 바뀌지 않아야 하는 분기다.
  3) 두 분기 모두 HTTP 상태코드는 **400 그대로**다(상태코드를 바꾸지 말 것). 응답 본문의 `error.message`·`error.request_id` 모양도 그대로다.
  4) 테스트가 실제 `aiChat` 핸들러 왕복으로 code 분기를 증명한다 — `ValidateAIInput` 단위 호출만으로는 수용 기준 1·2 를 보일 수 없다(code 는 핸들러가 붙인다).
  5) 기존 `TestValidateAIInputMessagesAndTokenLimit`(security_test.go:16-40, `ValidateAIInput` 을 6번 직접 호출)이 **수정 없이** 계속 통과한다.

- 건드릴 파일 (프로덕션 2개):
  - `internal/httpapi/ai.go` — (a) 패키지 수준 sentinel `errAISecretMaterial = errors.New("Secret 평문으로 보이는 내용은 AI에 전달할 수 없습니다")` 를 추가. (b) `ValidateAIInput`(ai.go:55-86)의 `secretMaterialPatterns` 루프(ai.go:78-82)만 그 sentinel 을 반환하게 바꾸고 **나머지 네 개 `errors.New`/`fmt.Errorf` 는 그대로 둔다** — 함수 시그니처(`func ValidateAIInput(input aiChatInput) error`)를 바꾸지 말 것(수용 기준 5). (c) `aiChat`(ai.go:92-96)에서 `errors.Is(err, errAISecretMaterial)` 로 code 를 `secret_material_rejected` / `invalid_ai_request` 로 분기.
  - `internal/httpapi/ai_input_errors_test.go` (신규) — 아래 하네스로 왕복 테스트.
  - 그 외 파일은 건드릴 필요가 없다. `ai.go` 는 `errors` 를 이미 import 한다(ai.go:60 `errors.New` 사용 확인).

- 배선 사실 (확인함 — 구현자가 다시 조사하지 않아도 된다):
  - 라우트는 `server.go:141` `mux.Handle("POST /api/v1/ai/chat", s.withAuth(http.HandlerFunc(s.aiChat)))`.
  - **검증 거절은 `s.store` 를 건드리기 전에 반환한다** — `decodeJSON`(ai.go:89) → `ValidateAIInput`(ai.go:93) → 거절 시 return. `s.store.AIConfig(...)` 는 ai.go:97 로 그 **뒤**다. 따라서 거절 경로 테스트는 **store 가 nil 이어도 돈다**. `aiLimiter` 도 이 지점 전에는 관여하지 않는다.
  - 하네스는 2026-09-28 회차 것을 그대로 재사용: `quietServer()`(auth_outage_test.go:16) + `server.sessionResolver = func(context.Context, string) (model.Session, error) { return model.Session{User: model.User{ID: "user-1", Role: "admin", AuthSource: "local"}}, nil }` 뒤에 `s.requestID(s.withAuth(http.HandlerFunc(s.aiChat)))` 를 `httptest.NewRecorder` 로 왕복. `revokeServer`(auth_revoke_test.go:16-28)가 정확히 이 모양이니 그대로 베낄 것. `sessionRevoker` 는 이 과제에 필요 없다.
  - `decodeJSON` 은 `DisallowUnknownFields()` 다(response.go:32). 테스트 JSON 본문에 `aiChatInput` 에 없는 키를 넣으면 `invalid_json` 으로 빠져 엉뚱한 걸 증명하게 된다 — `prompt`·`messages`·`max_tokens` 만 쓸 것.
  - 응답 모양은 `writeError`(response.go:22-26): `{"error":{"code":…,"message":…,"request_id":…}}`. 테스트는 `error.code` 를 꺼내 보면 된다.
  - 테스트 케이스 권장 5개: ① `{"prompt":""}` → `invalid_ai_request` ② `{"messages":[{"role":"tool","content":"x"}]}` → `invalid_ai_request` ③ `{"prompt":"hello","max_tokens":<maximumAITokens+1>}` → `invalid_ai_request` ④ `{"prompt":"password=super-secret-value"}` → `secret_material_rejected` + 메시지 원문 일치 ⑤ 네 경우 모두 status 400.
  - 수정 전 실패 재현: ①②③ 이 `code=secret_material_rejected` 로 실패해야 한다(④⑤는 처음부터 통과). 고친 뒤 통과시키고, `aiChat` 의 분기를 지워 되돌리면 같은 3개가 다시 실패함을 확인할 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/httpapi/ -run 'AI|ValidateAI' -count=1 -v`
  - `go test ./internal/httpapi/ -count=1` (2026-10-02 base eeca591 에서 **0.589s, green** 확인)
  - `go test ./... -count=1` / `go vet ./...` / `gofmt -l .`
  - `bash scripts/verify.sh` (Go test·vet·gofmt + npm ci·vitest·lint·build + 문서·Compose). **`npm ci` 부터 시작하니 시간이 든다** — `web/node_modules` 가 작업 트리에 없다.
  - Bash 도구가 `&&` 복합 명령과 `./scripts/*.sh` 직접 실행을 승인 요구로 막는다. 하나씩 나눠 실행하고 verify 는 `bash scripts/verify.sh` 로 부를 것.

- 위험과 피할 것:
  - `secret_material_rejected` 는 **저장소 전체에서 ai.go:94 한 곳에만** 나온다(go·ts·tsx·md 전체 grep 확인). 프런트엔드·`docs/guides/api-guide.md`·OpenAPI 어디에도 못박힌 계약이 없으므로 code 를 쪼개도 깨지는 곳이 없다. 반대로 **②의 메시지 원문은 테스트가 보게 하라** — 유일하게 보존해야 하는 문자열이다.
  - `ValidateAIInput` 은 **exported** 이고 `security_test.go` 가 6번 직접 호출한다. 시그니처를 `(code string, err error)` 같은 튜플로 바꾸면 그 테스트가 깨진다 — sentinel + `errors.Is` 로 갈 것(수용 기준 5).
  - `secretMaterialPatterns` 목록·`maximumAITokens` 값·262144 한도·role 허용 집합을 **건드리지 말 것**. 이번 과제는 분류만 바꾸는 것이고, 매처를 넓히거나 좁히는 것은 과거 운영자 반려 사유다.
  - `ai.go` 의 상류 호출부(`aiStream`·`aiIntegrationTest`·ai.go:150·244의 요청 본문 조립)와 `aiRequestLimiter` 는 손대지 말 것. 보호 경로(`auth_handlers.go`·`oidc*.go`·`mcp_oauth.go`·`store/users.go`·`migrations/`·`.github/workflows/`)는 이 과제에서 전혀 필요 없다.
  - `scripts/e2e-docker.sh`·Playwright 는 `docs/screenshots/*.png` 를 덮어써 작업 트리를 더럽힌다 — 실행 금지.
  - 작업 커밋에서 `CHANGELOG.md`·`scripts/version.sh` 를 건드리지 말 것(릴리즈는 별도 커밋).

- 차선 후보: **`AIConfig` 의 temperature 범위 클램프 누락** (`internal/store/settings.go:566-610` — `MaxTokens`·`TimeoutSeconds` 는 읽기 시점에 클램프하는데 `Temperature` 만 클램프가 없다). 단, 쓰기 경로 `validateSetting`(settings.go:228-232)이 이미 0~2 를 강제하므로 **직접 DB 쓰기 없이는 도달 불가**이고 관찰 가능한 변화가 없을 수 있다 — 1순위가 성립하지 않을 때만, 그리고 도달 경로를 먼저 증명한 뒤에 고를 것.
