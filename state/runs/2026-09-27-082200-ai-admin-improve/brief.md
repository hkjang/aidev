- 과제: 비스트리밍 chat 프록시가 upstream `Content-Type`을 검증 없이 그대로 재전송하는 문제를 스트리밍 경로와 같은 계약으로 맞추기 (가치 2 / 위험 2 / 작업량 S)

- 왜: 같은 핸들러 `chatCompletions`(`internal/server/providers.go:646-661`)에서 스트리밍 응답은 media type이 `text/event-stream`이 아니면 본문을 버리고 502 `ai_stream_invalid`로 거부하는데(`:647-654`), 비스트리밍 응답은 upstream이 준 값(`text/html`, `text/plain` 등)을 그대로 `w.Header().Set("Content-Type", contentType)`(`:661`)으로 콘솔 오리진에 내보낸다 — 빈 값일 때만 `application/json`으로 채운다(`:658-659`). 고치면 `/v1/chat/completions`가 JSON을 약속한 문서 계약과 실제 응답 헤더가 항상 일치하고, 공급자 오설정·침해로 비-JSON media type이 관리 콘솔 오리진으로 흘러드는 통로가 닫히며, 현재 통합 테스트가 전혀 지나지 않는 비스트리밍 릴레이 경로에 첫 end-to-end 커버리지가 생긴다(`grep`이 아니라 아래 "확인한 것"으로 확인).

- 수용 기준:
  1) 비스트리밍(`"stream": false`) 요청에서 공급자가 `application/json`(파라미터 유무 무관, 대소문자 무관) 또는 Content-Type 없음으로 응답하면 기존대로 200이 나가고 본문이 그대로 릴레이되며, 응답 헤더 `Content-Type`은 upstream 원문이 아니라 `application/json; charset=utf-8` 한 가지로 고정된다.
  2) 같은 요청에서 공급자가 JSON이 아닌 media type(예: `text/html; charset=utf-8`)으로 200 응답하면 본문이 caller에게 전달되지 않고 502 `ai_response_invalid`가 나가며, 응답의 `Content-Type`에 `text/html`이 들어 있지 않다.
  3) 2)의 경우 감사 이벤트 `ai.chat`이 `result=failure`, `details.reason`에 새 사유(`invalid_response_content_type` 권장)로 남는다 — 스트리밍의 `invalid_stream_content_type`과 짝이 되는 이름.
  4) 스트리밍 경로의 동작은 불변: 기존 `TestChatAnswerCutShortIsNotDeliveredAsComplete`와 `TestIntegration…`(`integration_test.go:322`의 streaming chat 200 + `[DONE]`)이 그대로 통과한다.
  5) 테스트는 손으로 만든 대역이 아니라 실제 PostgreSQL → `db.Migrate`/`db.Seed` → `New(...).Handler()` → `httptest.NewServer` → 로그인 → 공급자 생성 → 실제 HTTP `POST /v1/chat/completions` 전 경로로 1)~3)을 증명하고, 고치기 전 2)·3)이 FAIL 하는 것을 먼저 확인한다(`-v`로 PASS, SKIP 아님).

- 건드릴 파일 (3개):
  - `internal/server/providers.go:646-661` — `chatCompletions`의 응답 Content-Type 분기. 현재 `if stream { … } else if contentType == "" { contentType = "application/json" }` 구조에서, `else` 쪽을 스트리밍과 대칭으로 바꾼다: `contentType`이 비어 있지 않으면 `mime.ParseMediaType`으로 파싱하고, 파싱 실패이거나 media type이 `application/json`(그리고 `+json` suffix 허용 권장)이 아니면 `io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))`로 본문을 버린 뒤 `s.audit(... "failure", map[string]any{"model": payload["model"], "reason": "invalid_response_content_type"})` + `writeError(w, http.StatusBadGateway, "ai_response_invalid", "<한국어 메시지>")` 후 `return`. 통과하면 `contentType = "application/json; charset=utf-8"`로 고정. `mime`·`strings`·`io`는 이미 import 되어 있다(`:541`·`:648`·`:650` 사용 중).
  - `internal/server/chat_response_content_type_integration_test.go` (신규) — 아래 "재사용 셋업" 그대로 복제해 비스트리밍 사례 3개(JSON 통과 / `text/html` 거부 / Content-Type 없음 통과).
  - `docs/api.md:274` 바로 아래 — 지금 stream 응답 헤더만 적혀 있고 비스트리밍은 한 줄도 없다. "비스트리밍 응답은 `application/json; charset=utf-8`로 전달되며, 공급자가 JSON이 아닌 media type으로 응답하면 502 `ai_response_invalid`" 한 줄과, `:275`의 `details.reason` 목록에 새 사유를 추가.
  - (웹 변경 없음 → `internal/ui/dist` 재빌드 불필요. VERSION·CHANGELOG는 건드리지 말 것.)

- 재사용 셋업 (실제로 열어서 확인함): `internal/server/chat_truncation_integration_test.go:22-110`이 이 과제에 필요한 전 배선을 이미 갖고 있다 — `TEST_POSTGRES_DSN` 없으면 `t.Skip`, `DROP SCHEMA … CASCADE` → `db.Migrate` → `db.Seed("admin@example.com","integration-password")` → `secrets.New(bytes.Repeat([]byte{9},32))` → `New(db,cipher,slog…).Handler()` → `httptest.NewServer(handler)` → `signIn(t, handler)`(`db_error_integration_test.go:100`) → `session.do(t, handler, POST, "/api/v1/ai/providers", {name,type:"openai-compatible",baseUrl:upstream.URL,defaultModel:"test-model",contextWindow,maxOutputTokens})` → 세션 쿠키 + `X-CSRF-Token`을 붙인 실제 `POST {service.URL}/v1/chat/completions`. 차이점은 둘뿐이다: 합성 upstream이 hijack·close 대신 정상 JSON 본문을 쓰고 원하는 `Content-Type`을 세팅할 것, 그리고 요청 body에 **`"stream": false`를 명시**할 것(`providers.go:582-589` — 생략하면 공급자 `streamDefault`와 `ai.stream_default` 설정으로 결정되어 스트리밍이 된다). 감사 확인은 같은 파일 `:119-` 의 `SELECT COALESCE(details->>'reason','') FROM ai_admin.audit_event …` 패턴을 그대로 쓸 것.

- 검증 명령:
  - 새 포트로 전용 폐기 DB: `docker run -d --rm -p 55471:5432 -e POSTGRES_PASSWORD=postgres --name ai-admin-scout-pg postgres:16-alpine` 후 `export TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55471/postgres?sslmode=disable'` (55432·55433·55439·55444·55451·55461은 과거 회차가 쓴 이력이 있으니 피할 것). **주의: 통합 테스트가 스키마를 DROP 하므로 공유·운영 DB 금지.**
  - `go test -run 'TestChat' -count=1 -v ./internal/server/` — 새 테스트가 수정 전 FAIL 하는 것을 먼저 확인.
  - `go test -race -count=1 ./...` (internal/server 약 80~105초).
  - `make lint` (gofmt·go vet·`scripts/verify-version.sh`), `go build ./...`.
  - 웹 변경이 없으므로 `npm test`·`npm run build`는 생략.

- 위험과 피할 것:
  - **엄격화의 실제 위험**: `text/plain`으로 JSON을 돌려주는 공급자는 이 변경 뒤 502가 된다. 이것은 의도된 계약 변경이며 스트리밍 경로의 기존 선례와 대칭이다 — 다만 수용 기준 1)의 허용 집합(`application/json` + 파라미터 + `+json` suffix + 빈 값)을 임의로 줄이지 말 것. 반대로 "거부하지 말고 조용히 JSON으로 덮어쓰기"로 바꾸면 비-JSON 본문이 JSON이라고 라벨링되어 나가므로 수용 기준 2)를 만족하지 못한다.
  - `relayChatBody`·`chatRelay`·스트리밍 분기(`:647-657`)·`panic(http.ErrAbortHandler)` 절단 처리(`:676-681`)는 건드리지 말 것 — 별개 계약이고 기존 테스트가 걸려 있다.
  - `writeError`는 `w.WriteHeader`(`:662`) **앞에서만** 호출할 수 있다. 새 거부 분기는 반드시 `:661` 이전에 있어야 한다.
  - 감사 `details`에는 식별자·사유만 넣고 upstream 본문이나 원문 헤더 전체를 넣지 말 것(운영자 지시). media type 문자열을 details에 넣고 싶다면 파싱된 media type만, 파라미터 없이.
  - 오류 메시지는 저장소 계약대로 `writeError(status, code, 한국어 메시지)` 형태를 지킬 것.
  - 보호 경로(`internal/auth`, `auth_handlers.go`, `oidc.go`, `workflow.go`, `internal/database/migrations`, `.github/workflows`, `internal/ui/dist`)는 이번 과제와 무관하다 — 열지 말 것.
  - 과거 교훈: 정찰이 지정한 "구현 방식"이 수용 기준과 어긋나면 수용 기준을 따르고 방식을 바꿀 것(2026-09-25 선례).

- 확인한 것 / 추측인 것:
  - 확인: `providers.go:582-589`(stream 결정), `:646-681`(Content-Type 분기와 릴레이), `chat_truncation_integration_test.go:1-120`(셋업 전체), `integration_test.go:283-325`(합성 upstream이 `text/event-stream`을 쓰고 chat이 스트리밍이라 비스트리밍 경로를 지나는 테스트는 저장소에 **없다**), `docs/api.md:269-277`(비스트리밍 응답 계약 미기재), `ai_response_invalid`·`invalid_response_content_type`은 저장소 전체에 아직 없는 새 이름.
  - 추측(미확인): 공급자 생성 시 `streamDefault` 기본값이 true인지 정확히는 확인하지 않았다 — 그래서 요청에 `"stream": false`를 **명시**하라고 적었다. `docs/security.md`·`web/src`에 chat 응답 Content-Type을 읽는 코드가 있는지는 확인하지 않았다(구현 전 `grep -rn "chat/completions" web/src` 한 번 권장).

- 차선 후보: 감사 CSV 문서의 "다운로드가 성공했다면 필터에 맞는 이벤트가 모두 담긴 것입니다" 문구를 같은 문서 표의 "최대 50,000건" 상한과 맞추기 (가치 1 / 위험 1 / 작업량 S) — `docs/api.md` 한 문장만 고치는 문서 변경. 1순위가 성립하지 않을 때만 고를 것.
