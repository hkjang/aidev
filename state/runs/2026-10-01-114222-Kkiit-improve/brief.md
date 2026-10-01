- 과제: 관리자 사용자 수정(PATCH /admin/users/{id})의 DB 저장 실패를 404 "사용자를 찾을 수 없습니다." 가 아니라 500 으로 알리기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `updateAdminUser`(internal/httpapi/admin_users.go:71-75)가 `UPDATE users SET status=$2,display_name=$3,updated_at=now() WHERE id=$1` 의 **모든 오류**를 `if err != nil || tag.RowsAffected() == 0` 로 묶어 404 `user_not_found` "사용자를 찾을 수 없습니다." 로 돌려준다. 같은 파일의 `resetAdminUserMFA`(:157-166)는 이미 `err != nil` → 500 `mfa_reset_failed` 와 `RowsAffected()==0` → 404 `mfa_not_found` 를 갈라 놓았으므로 같은 파일 안에서 비대칭이고, 조사 화면(`GET /admin/users/{id}`)에서 그 계정을 보고 있는 관리자가 상태·표시 이름을 바꾸려는 순간 "사용자를 찾을 수 없습니다" 를 받아 계정이 사라진 줄 알게 된다. 프런트는 `error.message` 를 그대로 띄우므로 서버만 고쳐도 안내가 바뀐다.

- 수용 기준:
  1) 실제 라우터·실제 PostgreSQL 로 올린 통합 테스트에서, **존재하는** 사용자 id 에 대해 저장이 DB 단계에서 실패하는 PATCH 요청이 `500` 과 저장 실패 코드(예: `user_save_failed`)를 돌려준다. 수정 전에 같은 테스트가 `status=404 want=500 body={"error":{"code":"user_not_found",...}}` 로 실패하는 것을 먼저 보여 준다.
  2) 존재하지 않는 사용자 id 에 대한 PATCH 는 종전대로 `404 user_not_found` 를 유지한다(회귀 확인).
  3) 정상 PATCH 는 종전대로 `200 {"ok":true}` 이고, `status:"suspended"` 일 때의 세션 폐기·`status:"active"` 일 때의 `AccountReactivated` 통지·`s.audit("user.update")` 가 성공 경로에서만 일어난다(실패 경로에서 감사·통지가 생기지 않는 것까지 확인).
  4) 분기를 두 방향으로 변이시키면(항상 500 / 항상 404 = 원본) 각각 테스트가 다시 실패한다.

- 증거 수법(실측된 유일한 방법): `users.display_name` 은 `text`(001_initial.sql:22)이고 핸들러는 `strings.TrimSpace` 와 빈 문자열 검사만 하므로 `"display_name": "관리자\u0000이름"` 처럼 **NUL 바이트**를 넣으면 PostgreSQL 이 SQLSTATE 22021(`invalid byte sequence for encoding "UTF8": 0x00`)로 문장 자체를 거절한다 — 행은 존재하지만 `RowsAffected()` 는 0 이다(2026-09-27·28·29 세 회차에서 webhooks.name / portfolios.title 로 실측). `status` 는 `active|suspended` 로 선검증되고 `id` 는 uuid 라 다른 입력으로는 DB 오류를 만들 수 없다.

- 건드릴 파일:
  - `internal/httpapi/admin_users.go:71-75` — `updateAdminUser` 의 한 줄 분기를 `if err != nil { writeError(w, 500, "user_save_failed", "사용자 정보를 저장하지 못했습니다.") ; return }` / `if tag.RowsAffected() == 0 { 404 user_not_found }` 로 가른다. 같은 파일 `resetAdminUserMFA` 의 형태를 그대로 따를 것. **이 파일의 다른 핸들러(updateAdminUserRoles:162, resetAdminUserMFA)는 이미 갈라져 있으므로 건드리지 말 것.**
  - `internal/httpapi/integration_test.go` — 새 테스트 1개(예: `TestIntegrationAdminUserUpdateSeparatesSaveFailureFromNotFound`). 배선: `server, pool := integrationServer(t)` → `operator := operatorClient(t, server, pool, "useradmin")` → `grantRole(t, pool, operatorUsername(t, pool, operator), "super_admin")`(**`operator` 역할에는 `users.manage` 가 없다** — 001_initial.sql:523 에 `admin.access/approvals.manage/talents.review/orders.manage` 만 있고 `users.manage` 는 `super_admin` 전부 부여(:521)로만 들어온다; 같은 배선이 integration_test.go:2227·2250 에 이미 있고 그 뒤 호출이 200 을 받으므로 권한은 요청마다 읽힌다) → 대상 사용자는 `name := uniqueName("targetuser")` + `newClient(t, server.URL)` + `register(name)` 로 만들고 id 는 기존 테스트와 같은 방식으로 pool 에서 직접 읽는다: `pool.QueryRow(context.Background(), "SELECT id::text FROM users WHERE username=$1", name).Scan(&userID)` (integration_test.go:2229 가 본보기) → `operator.do(http.MethodPatch, "/api/v1/admin/users/"+userID, map[string]any{"status":"active","display_name":"관리자\u0000이름"}, http.StatusInternalServerError)`.
  - `docs/openapi.yaml:1005-1011` — `/admin/users/{id}` 의 `patch` 응답에 핸들러가 **실제로 내보내는** `'400'`(`invalid_status` / `display_name_required`)·`'404'`(`user_not_found`)·새 `'500'` 을 더한다. 지금은 `'200'`·`'409'` 만 있다.
  - 프로덕션 파일 1개 + 테스트 1개 + 문서 1개 = 3개. **더 늘리지 말 것.**

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/httpapi/ -run 'TestIntegrationAdminUserUpdate' -v -timeout 300s` (DSN 필수, `--- PASS` 가 아니라 `SKIP` 이면 통합이 안 돈 것 — 실패로 취급)
  - `make test-integration KKIIT_TEST_DSN=postgres://…` (전용 폐기 DB, `internal/httpapi` 만 80~100초)
  - `go test ./cmd/... ./internal/...`
  - `go vet ./cmd/... ./internal/...`
  - `gofmt -l cmd internal` (무출력)
  - `npm --prefix web test` (10건 — 이번 변경은 프런트를 건드리지 않으므로 통과 확인만)
  - DSN 없이 Go 테스트가 통과하는 것은 통합 검증이 **아니다**. `docker version` 은 지난 4회차에서 29.7.2 로 가용했으나 이번 정찰에서 재확인하지 않았다(미확인). 깨끗한 DB(DROP/CREATE)에서 돌릴 것.

- 위험과 피할 것:
  - `admin_users.go` 는 계정 상태(`status`)와 **세션 폐기**를 함께 건드리는 준위험 경로다. 고칠 것은 **오류 분류 한 곳뿐**이다. 성공 경로의 세션 폐기(`UPDATE sessions SET revoked_at=now() …`)·`notifyAccountChange`·`s.audit` 를 옮기거나 조건을 바꾸지 말 것. 실패 분기는 `return` 하므로 그 아래는 그대로 둔다.
  - `auth.go`·`middleware.go`·`identity.go`·`internal/database/migrations`·`.github/workflows` 는 열지 말 것.
  - `internal/httpapi` 통합 테스트는 전역 `apiUnderTest` 를 쓰므로 **`t.Parallel()` 금지**.
  - `internal/ui/dist` 는 추적되는 산출물이다 — 재빌드해도 **커밋 금지**. 프런트는 `error.message` 를 그대로 띄우므로 `web/` 을 건드릴 이유가 없다.
  - 기존 `s.audit(r, "user.update", …, in, "success")` 가 `display_name` 원문을 감사에 넘기는 점은 이번 범위가 **아니다**(실패 경로는 audit 에 닿지 않는다). 발견으로만 남기고 고치지 말 것 — 범위가 늘면 파일 수 규칙이 깨진다.
  - grep 으로 문자열이 있다는 것은 증거가 아니다. 반드시 HTTP→실제 DB 왕복으로 상태 코드를 보여 줄 것.
  - 확신이 서지 않으면 임시 프로브로 실패 경로의 `*pgconn.PgError.Code` 와 그 행의 존재(`row_exists`)를 눈으로 확인한 뒤 프로브를 **제거**하고 커밋할 것(지난 3회차가 쓴 방법).

- 차선 후보: `retryDomainEvent`(internal/httpapi/events.go:103-107)의 `err != nil || tag.RowsAffected() == 0` → 404 `event_not_retryable` 분기 가르기. 다만 파라미터가 uuid 하나뿐이라 **HTTP 입력으로 DB 오류를 만들 경로가 없다** — 증거는 404 회귀 확인 + 분기 단위 검증까지만 가능하다. 1순위가 성립하지 않을 때(예: NUL 이 22021 을 만들지 못할 때)만 고르고, 그때는 `organizations.go:256`(uuid·enum 뿐, 같은 제약)보다 이쪽이 단순하다.
