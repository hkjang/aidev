# 과제서 (정찰, 2026-09-29) — base main@746a704, VERSION 1.2.30

- 과제: `PATCH /api/v1/profile`이 저장하지 않는 `department` 필드를 200으로 받아 주는 문제를 없애기 (가치 2 / 위험 1 / 작업량 S)

- 왜: `updateProfile`(`internal/server/users.go:412-417`)의 요청 구조체에는 `Department *string \`json:"department"\``가 있지만, 같은 함수의 UPDATE 문(`users.go:439-441`)은 `display_name`과 `email`만 쓰고 `request.Department`는 저장소 전체에서 **어디서도 읽지 않는다**(`grep -rn "Department" internal/`의 유일한 hit이 그 선언이다). `ai_admin.app_user`에 `department` 컬럼도 없다(`internal/database/migrations/001_ai_admin.sql:8-23`). `decodeJSON`은 `DisallowUnknownFields()`(`internal/server/response.go:44`)라 다른 모르는 필드는 모두 400 `invalid_json`인데, 이 한 필드만 선언이 남아 있어서 `{"displayName":"관리자","department":"영업팀"}`이 200 `{"updated":true}`로 돌아오고 부서는 사라진다 — 호출자는 저장됐다고 믿는다. 필드 선언을 지우면 이 요청도 저장소의 일반 계약과 같은 400이 되어, API가 지키지 못하는 약속을 하지 않는다.

- 수용 기준:
  1. `PATCH /api/v1/profile`에 `department`가 들어 있으면(값이 문자열·빈 문자열·`null` 어느 쪽이든) 200이 아니라 400으로 거부되고, `app_user` 행은 전혀 바뀌지 않는다(`display_name`·`updated_at` 불변). 어떤 코드로 거부하든 상관없지만 **저장소의 기존 unknown-field 계약과 같은 응답**이어야 한다(구조체 필드를 지우면 `invalid_json`이 자동으로 나온다 — 새 코드를 만들 필요 없음).
  2. 기존 동작은 그대로다: `{"displayName":"관리자"}` 200 + 저장, 한글 190자 200 / 191자 400 `profile_invalid`, 중복 email 409 `email_conflict`, email 320자 초과·`@` 없음 400 `email_invalid`, 대상 계정 없음 404 `user_not_found`.
  3. 통합 테스트가 위 1을 실제 PostgreSQL + 실제 라우터(`New(db, cipher, logger).Handler()`)로 증명한다 — 수정 전 FAIL(200이 나온다), 수정 후 PASS이며 `SKIP`이 아님을 `-v`로 확인한다. 거부 후 DB의 `display_name`이 요청 값으로 바뀌지 않았음을 SELECT로 확인한다(200만 보고 끝내지 말 것).
  4. `docs/api.md`의 프로필 표(줄 90 `| PATCH | /api/v1/profile | 인증 + CSRF | 로컬 계정 표시명/email 변경 |`)와 그 아래 설명 문단(줄 97)에 "본문은 `displayName`과 `email`만 받고 그 밖의 필드는 400 `invalid_json`"임을 한 문장으로 적는다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/server/users.go:412-417` — `updateProfile`의 요청 구조체에서 `Department *string \`json:"department"\`` 한 줄을 지운다. `DisplayName`·`Email` 처리와 SQL, `p.AuthSource == "local"` 분기는 손대지 말 것(OIDC 계정의 email 무시는 `docs/api.md:90`의 "로컬 계정" 문구대로 의도된 동작이다).
  - `internal/server/profile_preference_integration_test.go` — 기존 `TestProfileAndPreferenceSaveReportTheActualOutcome`(같은 파일 22행부터)의 셋업을 그대로 쓴다: `database.Open` → `DROP SCHEMA IF EXISTS ai_admin CASCADE; DROP SCHEMA IF EXISTS aiportal CASCADE` → `db.Migrate` → `db.Seed(ctx, "admin@example.com", "integration-password")` → `secrets.New(bytes.Repeat([]byte{7}, 32))` → `New(...).Handler()` → `signIn(t, handler)`(`db_error_integration_test.go:100`) → `session.do(t, handler, http.MethodPatch, "/api/v1/profile", map[string]any{...})` → `assertFailure(t, response, http.StatusBadRequest, "invalid_json")`. 새 `Test…` 함수를 추가하는 쪽이 기존 테스트의 임시 CHECK 제약·preference 단계와 얽히지 않아 안전하다.
  - `docs/api.md` — 위 기준 4.
  - 웹은 건드리지 말 것. `web/src/types/api.ts:27`의 `CurrentUser.department?`도 죽은 선언이지만(서버 `/auth/me`는 Principal을 직렬화하고 `internal/auth`에 Department 필드가 없다) 런타임 코드가 아니고, 웹을 고치면 커밋된 `internal/ui/dist` 재빌드가 따라와 변경 범위가 커진다 — 이번 회차 범위 밖이다.

- 검증 명령:
  - 전용 폐기 PostgreSQL을 새 포트로 띄운다(과거 회차가 쓴 55432·55433·55439·55444·55451·55461·55471·55481·15434는 피하고 예: 55491):
    `docker run -d --rm --name ai-admin-scout-pg -p 55491:5432 -e POSTGRES_PASSWORD=postgres postgres:16-alpine`
    `export TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55491/postgres?sslmode=disable'`
  - red 확인: `go test -count=1 -run TestProfileRejectsUnsupportedFields ./internal/server/ -v`(새 테스트 이름은 자유. `-v`로 SKIP이 아님을 확인)
  - green + 회귀: `go test -race -count=1 ./...`(internal/server 약 100~130초)
  - `make lint`(`gofmt -l`·`go vet`·`scripts/verify-version.sh` — VERSION 1.2.30과 일관해야 한다), `go build ./...`
  - 웹 변경이 없으므로 `npm test`·`npm run build`·`internal/ui/dist` 재빌드는 하지 않는다.

- 위험과 피할 것:
  - `VERSION`·`CHANGELOG.md`는 건드리지 말 것(릴리즈 단계 전용).
  - `internal/auth`, `oidc.go`, `workflow.go`, `internal/database/migrations`, `.github/workflows`, `internal/ui/dist`는 이 과제와 무관하다 — 열지 말 것. **새 마이그레이션으로 `department` 컬럼을 추가하는 방향은 금지**(위험 구역이고 한 세션 범위를 넘는다).
  - `TEST_POSTGRES_DSN`이 없으면 통합 테스트가 조용히 SKIP된다. "ok"만 보고 통과로 읽지 말고 `-v`로 PASS를 확인할 것(프로필의 검증 함정).
  - 통합 테스트가 `DROP SCHEMA … CASCADE`를 하므로 공유·운영 DB 금지, 같은 DB로 병렬 실행 금지.
  - 이 과제는 200 → 400으로 바뀌는 **동작 변경**이다. 호출자를 먼저 확인했다: `web/src`에 `/api/v1/profile`로 `department`를 보내는 코드가 없고(`grep -rn "department" web/src`는 레거시 부서 디렉터리 화면만 hit), `docs/api.md`에도 `department` 요청 필드 문서가 없고, `internal/server/*_test.go`의 기존 프로필 호출 4곳(`integration_test.go:190,195,200`, `profile_preference_integration_test.go:60,69`)도 `displayName`/`email`만 보낸다. 그래도 구현자는 자신이 직접 다시 확인할 것.
  - 이 과제서가 지정하는 것은 **결과**(기준 1~4)다. 필드 삭제가 가장 싼 방법이라고 보지만, 돌려 보니 기준과 어긋나면(예: 어떤 호출자가 실제로 `department`를 보내고 있다면) 방법을 바꾸고 무엇을 왜 바꿨는지 적을 것 — 2026-09-25 회차처럼 지정된 방법이 기준을 만족하지 못하면 기준을 따르는 쪽이 맞다.

- 차선 후보: 비스트리밍 chat 본문이 중간에 끊겼을 때의 감사 `reason` 계약을 테스트로 고정 (가치 2 / 위험 1 / 작업량 S, 프로덕션 파일 0개). `relayChatBody`(`providers.go:721-751`)는 `stream` 플래그와 무관하게 `client_write_failed`/`client_flush_failed`/`upstream_timeout`/`request_cancelled`/`upstream_read_failed`를 구분하고 `chatCompletions`(`providers.go:700-706`)가 `panic(http.ErrAbortHandler)`로 끊는다. 이번 회차에 코드를 직접 읽어 프로덕션 결함이 아니라 테스트 공백임을 확인했다 — `chat_truncation_integration_test.go`는 스트리밍 경로만, `chat_response_content_type_integration_test.go`는 정상 본문만 덮는다. `"stream": false`를 명시하고 upstream이 `Content-Length`보다 짧은 본문을 보내 끊는 fake 공급자로 200 이후 절단 → 감사 `result=failure` + `details.reason=upstream_read_failed`·`complete=false`를 고정하면 된다.
