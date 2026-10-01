- 과제: 키 권한(scope) 수정의 `name`을 `varchar(160)` DB 상한·trim 계약에 맞춰 400으로 막기 (가치 2 / 위험 1 / 작업량 S)
- 왜: `updateKeyScope`(`internal/server/keys.go:392-422`)는 `name`을 trim·길이 검증 없이 `ai_admin.key_scope.name varchar(160)`(`internal/database/migrations/001_ai_admin.sql:138`)에 그대로 넣는다 — 161자 이름은 PostgreSQL `22001`이 되어 400이 아니라 500 `scope_update_failed`로 나가고, `""`·공백만 있는 이름은 그대로 저장돼 키 권한 목록이 이름 없이 렌더링된다. 같은 저장소의 `updateRole`(`users.go:271-281`)·`updateUser`(`users.go:106-113`)·`createKey`(`keys.go:116-118`)는 모두 trim 후 `utf8.RuneCountInString` 상한을 DB 조회 **전에** 검사하므로, 이 핸들러만 계약이 빠져 있다.
- 수용 기준:
  1) `PATCH /api/v1/key-scopes/{code}`에 161 rune `name`을 보내면 400 `name_invalid`(`writeError`)이고 `key_scope` 행의 `name`·`updated_at`이 **불변**이다(500이 아니다).
  2) 정확히 160 rune(한글 160자 포함)은 200이고 저장값이 요청 원문과 같다 — 상한은 바이트가 아니라 rune 기준이어야 한다(`len()` 바이트 검사로 바꾸면 이 사례가 실패해야 한다).
  3) `name`이 `""` 또는 공백만이면 400 `name_invalid`이고 저장값 불변. 앞뒤 공백이 붙은 정상 이름은 200이고 trim된 값이 저장된다.
  4) `name`을 아예 보내지 않는 기존 호출(예: `{"enabled":false}`, `{"description":"..."}`, `{"assignableByUser":true}`)은 동작이 바뀌지 않는다 — `coalesce($2,name)`로 기존 이름이 유지되고, 보호 scope의 400 `scope_protected`와 없는 code의 404 `scope_not_found`도 그대로다.
  5) 테스트가 증명할 것: 수정 전에는 161자에서 500이 나는 것을 먼저 red로 확인하고(실제 PostgreSQL), 수정 후 1~4가 모두 PASS(SKIP 아님)여야 한다.
- 건드릴 파일:
  - `internal/server/keys.go:updateKeyScope` — `keyscope.Protected(code)` 검사 **다음**, `s.db.Pool.Exec` **앞에** `request.Name != nil`이면 `value := strings.TrimSpace(*request.Name)`; `value == "" || utf8.RuneCountInString(value) > 160`이면 400 `name_invalid`(메시지는 한국어, 예: "키 권한 이름이 올바르지 않습니다."), 통과하면 `request.Name = &value`. `updateRole`(`users.go:271-281`)의 형태를 그대로 따르고 상한 리터럴 옆에 왜 160인지(`key_scope.name varchar(160)`) 주석을 남길 것. `strings`·`utf8`은 이미 import되어 있다(`keys.go:116-118`이 둘 다 쓴다). 보호 scope 검사 뒤에 두는 이유: `scope_protected`가 계속 우선 응답이어야 하고 기존 테스트 `TestProtectedKeyScopesRejectUserAssignability`(`keys_security_test.go:16`)가 그 순서를 본다.
  - `internal/server/keys_security_test.go` (기존 파일에 추가) — **DB 없이 도는 단위 테스트**로 검증 행렬을 고정할 수 있다. `TestProtectedKeyScopesRejectUserAssignability`(:16-35)가 이미 `(&Server{}).updateKeyScope(recorder, request)`를 nil DB로 직접 호출한다(검사가 DB 접근 전에 끝나므로 panic하지 않는다). 같은 패턴(`chi.NewRouteContext()` + `URLParams.Add("code", …)` + `chi.RouteCtxKey`)으로 비보호 code(`ai:chat`)에 161 rune·`""`·`"   "` 이름을 보내 400 `name_invalid`를, 160 rune 한글 이름은 **400이 아님**을 확인한다(nil DB라 그 경로는 Exec에서 panic하므로, 통과 사례는 단위 테스트로 상태 단정을 하지 말고 통합 테스트에 맡길 것 — 단위 테스트는 거부 사례만 본다).
  - `internal/server/key_scope_name_integration_test.go` (신규) — 저장·불변·200 경로를 증명한다. 전용 폐기 PostgreSQL → `db.Migrate`/`db.Seed` → `New(db,cipher,logger).Handler()` → `signIn` → `PATCH /api/v1/key-scopes/{code}`. 재사용할 헬퍼: `signIn`(`db_error_integration_test.go:100`), `sessionCredentials.do`, `doJSON`/`doJSONWithCookies`(`integration_test.go`), 실패 단정 헬퍼(`assertFailure` 류). CSRF가 걸린 라우트(`protected.With(s.csrf).Patch`, `server.go:127`)이므로 세션 헬퍼를 통해 호출할 것 — 직접 요청을 만들면 403이 된다. 대상 code는 `keyscope.Protected`가 **아닌** 시드 scope를 쓸 것: `Seed`가 넣는 비보호 scope는 `ai:chat`·`ai:models`·`mcp:tools`·`profile:read`이고 보호 scope는 `admin:read`·`admin:write`·`legacy:config`다(`internal/database/database.go:274-280`, `internal/keyscope/policy.go:4`). 저장값은 `SELECT name,updated_at FROM ai_admin.key_scope WHERE code=$1`로 직접 확인하고, 200 경로는 `GET /api/v1/key-scopes` 응답의 `name`도 함께 본다.
  - `docs/api.md` — 키 권한 수정 항목에 `name`의 trim·160자 계약과 400 `name_invalid`를 한 줄 추가(이 저장소는 계약 변경을 같은 커밋에서 문서에 반영한다).
  - `description`은 **건드리지 말 것** — `key_scope.description`은 `text`여서 대응하는 DB 상한이 없고, 숫자를 새로 만들면 근거 없는 상한이 된다. 이번 회차 범위는 `name` 하나다.
- 검증 명령:
  - 전용 폐기 컨테이너(과거 회차가 쓴 55432·55433·55439·55444·55451·55461·55471·55481·55491·15434는 피하고 새 포트, 예: 55501):
    `docker run -d --rm --name ai-admin-scope-test -e POSTGRES_PASSWORD=postgres -p 55501:5432 postgres:16-alpine`
  - 단위(DSN 없이 돈다, 빠름): `go test -count=1 -run 'KeyScope' ./internal/server/ -v`
  - 통합: `TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55501/postgres?sslmode=disable' go test -count=1 -run 'KeyScope' ./internal/server/ -v` (SKIP이 아니라 PASS인지 반드시 확인)
  - 전체: `TEST_POSTGRES_DSN=... go test -race -count=1 ./...` (internal/server 약 100~135초)
  - `make lint` (gofmt·go vet·`scripts/verify-version.sh` — VERSION 1.2.31 일관), `go build ./...`
  - 웹 변경이 없으므로 `npm test`·`internal/ui/dist` 재빌드는 하지 않는다. VERSION·CHANGELOG는 건드리지 않는다(릴리즈 단계 전용).
- 위험과 피할 것:
  - `keyscope.Protected(code)` 분기와 `assignable_by_user=CASE WHEN $7 ...` SQL은 손대지 말 것 — 보호 scope 정책(`scope_protected` 400, 강제 false)이 여기 걸려 있다. 새 검사는 그 분기 **앞뒤 어디든 DB Exec 전**이면 되고, SQL 문자열은 그대로 둔다.
  - `internal/auth`·`workflow.go`·`oidc.go`·`internal/database/migrations`·`.github/workflows`·`internal/ui/dist`는 이번 과제와 무관하다. 마이그레이션을 고쳐 컬럼을 넓히는 방향은 금지(이 저장소는 새 마이그레이션 추가를 금지에 가깝게 취급한다).
  - 2026-09-25 교훈: 과제서가 지정한 "방법"이 수용 기준과 모순될 수 있다. 위 구현 지시가 기준 1~4를 만족하지 못하면 **기준을 우선**하고 무엇을 바꿨는지 적을 것.
  - 2026-09-28 교훈: 과제서가 적은 라우트가 실제와 다른 사례가 있었다. 라우트는 `server.go:127`에서 확인했고 `PATCH /api/v1/key-scopes/{code}`(관리 REST `/api/v1`, 권한 `keys.manage`)다 — 구현 시 한 번 더 확인할 것.
  - rune vs byte: 한글 160자 사례를 반드시 테스트에 넣고, `len()`으로 바꾸면 FAIL 하는 역검증까지 할 것(2026-09-28 선례).
- 차선 후보: 비스트리밍 chat 본문이 Content-Length보다 짧게 끊겼을 때의 감사 `reason` 계약을 테스트로 고정 (2/1/S, 프로덕션 파일 0개) — `relayChatBody`(`providers.go:721-751`)가 `client_write_failed`/`upstream_read_failed` 등을 구분하고 `chatCompletions`(:700-706)가 `panic(http.ErrAbortHandler)`로 끊는다. `"stream": false`를 명시하고 선언한 Content-Length보다 짧은 본문을 보내 끊는 fake 공급자로 `result=failure` + `details.reason=upstream_read_failed`·`complete=false`를 고정한다. 재사용 셋업은 `chat_truncation_integration_test.go`.
