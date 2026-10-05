- 과제: `display_name varchar(190)`·`email varchar(320)` 에 담을 수 없는 NUL(U+0000) 을 프로필·사용자 수정 두 경로 모두에서 저장 전 400 으로 막기 (가치 2 / 위험 1 / 작업량 S)

- 왜: `ai_admin.app_user.display_name` 은 `varchar(190)`, `email` 은 `varchar(320)` 인데(`internal/database/migrations/001_ai_admin.sql:9-12`) PostgreSQL 의 text/varchar 는 NUL 바이트를 담을 수 없다. 그런데 이 컬럼에 쓰는 두 핸들러가 모두 `strings.TrimSpace` + `utf8.RuneCountInString` 만 보고 NUL 을 통과시킨다 — `updateProfile`(`internal/server/users.go:413-446`)의 `displayName`·`email`, `updateUser`(`internal/server/users.go:88-113`)의 `displayName`. `encoding/json` 은 `"\u0000"` 을 실제 0x00 바이트가 든 Go 문자열로 디코딩하고 `TrimSpace` 는 그것을 지우지 않으며 rune 수는 1 이므로, 호출자 입력 오류가 400 이 아니라 `UPDATE` 에서 터져 500 (`profile_update_failed`(`users.go:447`) / `user_update_failed`(`users.go:195`))으로 나간다. `updateUser` 는 `tx.Exec` 라 실패가 롤백되므로 증상이 "아무것도 저장되지 않고 500", `updateProfile` 은 `Pool.Exec` 라 "그 UPDATE 만 실패하고 500" 이다 — 둘 다 400 이어야 할 자리다. 같은 핸들러들이 길이·빈값·이메일 형식은 이미 "쓰기 전에 400" 으로 못박아 둔 자리이므로, 같은 컬럼을 쓰는 두 경로를 같은 계약으로 맞추는 일이다.

- 수용 기준:
  1) `PATCH /api/v1/profile` 에 `{"displayName":"관리자\u0000"}`(JSON escape — 실제 NUL 바이트) 를 보내면 **400** 이고 응답 코드는 그 핸들러가 이미 길이 위반에 쓰는 `profile_invalid` 다(새 오류 코드를 만들지 말 것).
  2) `PATCH /api/v1/profile` 에 `{"displayName":"관리자","email":"a\u0000@example.com"}` 는 **400 `email_invalid`** 다.
  3) `PATCH /api/v1/users/{id}` 에 `{"displayName":"관리자\u0000"}` 는 **400 `name_invalid`** 다(그 핸들러의 기존 코드).
  4) 위 세 거부 각각에 대해 그 요청 **직후 DB 의 `display_name`·`email`·`updated_at` 이 요청 전과 같다** — 거부가 저장을 막았다는 직접 증거로 매번 SELECT 로 확인할 것. (수정 전에는 1·3 이 500 으로, 2 가 500 으로 red 가 나야 한다. red 를 먼저 확인하고 시작할 것.)
  5) 회귀: ① 리터럴 여섯 글자 `\u0000`(백슬래시·u·0·0·0·0 — JSON 으로는 `"\\u0000"`)은 NUL 이 아니므로 **200 으로 저장되고 그대로 조회**된다, ② 한글 190자 정확히 200 + 저장값 일치, 191자 400 `profile_invalid`, ③ `@` 없는 이메일 400 `email_invalid`, 정상 이메일 200, ④ `updateUser` 의 빈 문자열·191자 400 `name_invalid` 가 그대로다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/server/users.go:updateProfile` — 기존 `request.DisplayName == "" || utf8.RuneCountInString(...) > 190` 조건에 NUL 판정을 더하고, `email` 의 `!strings.Contains(value,"@") || utf8.RuneCountInString(value) > 320` 조건에도 같은 판정을 더한다.
  - `internal/server/users.go:updateUser` — `value == "" || utf8.RuneCountInString(value) > 190` 조건에 같은 판정을 더한다.
  - 판정은 세 곳이 같은 규칙을 쓰도록 같은 파일에 작은 헬퍼 하나(예: `strings.ContainsRune(value, 0)` 를 감싸는 `storableInVarchar`)로 두고, 상한 리터럴 옆 주석과 같은 스타일로 **왜**(대응 컬럼이 `varchar` 라 NUL 을 담을 수 없음)를 한 줄 남긴다. 이 저장소에 NUL 판정 헬퍼는 아직 없다(`grep` 아님 — `internal/server/*.go` 를 직접 확인했고 main 에 없다).
  - 새 통합 테스트 1개: `internal/server/profile_nul_integration_test.go`(이름 자유). 셋업은 `internal/server/profile_unknown_field_integration_test.go:25-54` 를 그대로 베낄 것 — `database.Open` → `DROP SCHEMA IF EXISTS ai_admin/aiportal CASCADE` → `db.Migrate` → `db.Seed("admin@example.com","integration-password")` → `secrets.New(bytes.Repeat([]byte{7},32))` → `New(db,cipher,logger).Handler()` → `signIn(t, handler)` → `session.do(...)`, 그리고 `SELECT id FROM ai_admin.app_user WHERE username='admin@example.com'` 로 `adminID` 를 받아 `PATCH /api/v1/users/{adminID}` 에 쓴다. 실제 라우트는 `server.go:139`(`PATCH /api/v1/profile`, csrf)와 `server.go:130`(`PATCH /api/v1/users/{id}`, csrf + `users.write`)이고 시드 admin 이 그 권한을 가진다.
  - 손대지 말 것: `web/src`, `internal/ui/dist`, VERSION/CHANGELOG, 마이그레이션.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 전용 폐기 PostgreSQL 을 **새 포트**로 띄운 뒤(55531 권장 — 55432·55433·55439·55444·55451·55461·55471·55481·55491·55501·55511·55521·15434 는 과거 회차 이력이 있다) `export TEST_POSTGRES_DSN=...`
  - red/green: `go test -race -count=1 ./internal/server/ -run TestProfile -v` — **SKIP 이 아니라 PASS** 인지 반드시 확인할 것(DSN 이 없으면 통합 테스트는 조용히 SKIP 된다).
  - 전체: `go test -race -count=1 ./...` (internal/server 약 120~140초)
  - `make lint`(gofmt·go vet·verify-version), `go build ./...`
  - 웹 변경이 없으므로 `npm test`·`make build`(=`internal/ui/dist` 재빌드)는 하지 않는다.

- 위험과 피할 것:
  - **미머지 브랜치 `auto/2026-10-04-1142`(f66d25c·ff9ee06·0daad74·169de71)가 같은 파일의 `updatePreferences` 를 고치는 중이고 main 에 없다.** `updatePreferences`/`loadPreferences`/`objectValue` 는 건드리지 말고, 그 브랜치가 도입한 이름(`jsonObjectHasNUL`·`storablePreferenceObject`)은 재사용하지도 같은 이름으로 새로 만들지도 말 것 — 나중에 머지될 때 충돌한다. 이번 과제는 `updateProfile`·`updateUser` 두 함수 안에서만 끝난다.
  - `username` 컬럼(로그인 경로)·`auth`·`oidc.go`·`workflow.go`·`internal/database/migrations`·`.github/workflows` 는 건드리지 말 것.
  - 오류 코드를 통일하려 하지 말 것. 핸들러마다 이미 다른 코드(`profile_invalid`/`name_invalid`/`email_invalid`)를 쓰고 `docs/api.md` 가 그것을 적고 있다 — NUL 도 각 핸들러의 기존 코드로 거부한다.
  - 길이 검사를 rune 에서 byte 로 바꾸지 말 것(이 저장소는 한국어 입력 때문에 rune 기준이 계약이다).
  - **미확인**: PostgreSQL 이 varchar 파라미터의 NUL 을 거부한다는 것(SQLSTATE 22021 `character_not_in_repertoire`, `invalid byte sequence for encoding "UTF8": 0x00`)은 이번 회차에서 **코드로 재현하지 못했다** — 이 환경에서 `docker run` 이 권한으로 막혔다. 2026-10-04 회차가 jsonb 쪽 같은 성질(22P05)을 실제 DB 로 확인했고 varchar 는 더 기본적인 제약이라 신뢰도는 높지만, 구현자는 **수정 전 red(500)를 먼저 찍어 전제를 확인하고** 시작할 것. 만약 500 이 아니라 그냥 저장된다면(= 전제가 틀렸다면) 이 과제를 버리고 아래 차선으로 갈 것.
  - `docs/api.md` 의 프로필·사용자 절에 NUL 거부를 한 문장으로 적되, 다른 절은 건드리지 말 것(문서 수정은 같은 커밋에 넣는 것이 이 저장소 관례다).

- 차선 후보: 비스트리밍 chat 본문 절단 시 감사 `reason` 계약을 테스트로 고정 — `relayChatBody`(`internal/server/providers.go:726-`)의 `client_write_failed`·`upstream_read_failed`·`upstream_timeout`·`request_cancelled` 네 사유를 `chat_truncation_integration_test.go` 셋업 재사용으로 `result=failure`·`details.reason`·`complete=false` 까지 못박는다. 프로덕션 파일 0개로 가장 싸고 위험이 없다(열한 회차 연속 차선).
