# 과제서 (2026-09-23, base main@a7b64fe / VERSION 1.2.24)

- **과제**: 역할 수정(`PATCH /api/v1/roles/{id}`)의 `name`을 DB 상한(varchar(120))에 맞춰 요청 단계에서 검증 (가치 2 / 위험 1 / 작업량 S)

- **왜**: `internal/server/users.go:258 updateRole`은 요청의 `name`을 아무 검사 없이 `UPDATE ai_admin.role SET name=coalesce($2,name)`(users.go:293)에 넘기는데, `internal/database/migrations/001_ai_admin.sql:33`의 컬럼은 `name varchar(120) NOT NULL`이라 121자 이상이면 PostgreSQL이 값 초과 오류(22001)를 내고 핸들러가 이유 없는 500 `role_update_failed`("역할을 저장하지 못했습니다.")로 끝난다 — 관리자는 무엇이 잘못됐는지 알 수 없다. 같은 이유로 `"   "`(공백만) 이름은 그대로 저장되어 역할 카드 제목이 빈 칸이 된다(`web/src/pages/RolesPage.tsx:73`의 antd `required` 규칙은 공백을 통과시키고, 이름 길이 제한도 걸려 있지 않다). 이 저장소는 이미 같은 계약을 다른 입력 경로에 적용해 왔다(`users.go:22` `query_too_long`, `legacy.go:317`, `audit_events.go:47`, `mcp.go:225`) — 역할 이름만 예외로 남아 있다.

- **수용 기준**:
  1) 121자(rune 기준) 이상의 `name`으로 `PATCH /api/v1/roles/{id}`를 보내면 DB 쓰기 없이 **400** `name_too_long`을 받고, 역할 이름·수정 시각이 바뀌지 않는다. 메시지는 한국어로, 기존 `writeError` 문체(예: `역할 이름은 120자 이하여야 합니다.`)를 따른다.
  2) `name`이 공백만(`"   "`)이거나 빈 문자열이면 **400** `name_required`(예: `역할 이름을 입력해 주세요.`)로 거부하고, 저장되는 값은 앞뒤 공백을 제거한 값이다. `name`을 **아예 보내지 않은**(JSON에 키 없음 → `*string` nil) 요청은 지금처럼 이름을 바꾸지 않고 통과해야 한다(설명·권한만 수정하는 기존 호출을 깨면 안 된다).
  3) 테스트가 증명할 것: (a) 수정 전에는 121자 이름이 실제로 500을 받는다는 재현, (b) 수정 후 121자 → 400 `name_too_long`·DB의 `role.name` 불변, (c) 정확히 120자(한글 120자 = 360바이트 — 바이트 기준 상한이면 오거부될 값) → 200이고 DB에 그대로 저장, (d) 공백만 → 400 `name_required`·DB 불변, (e) 앞뒤 공백이 있는 정상 이름 → 200이고 저장값은 trim된 값, (f) `name` 미전송 + `permissionCodes`만 보낸 요청 → 200이고 이름 불변(회귀 방지).

- **건드릴 파일**:
  - `internal/server/users.go:updateRole` — `decodeJSON` 직후, `SELECT code FROM ai_admin.role` 조회보다 **앞**(또는 최소한 `tx.Begin` 앞)에 `request.Name != nil`일 때만 `strings.TrimSpace` → 빈 값이면 400 `name_required`, `len([]rune(...)) > 120`이면 400 `name_too_long`. trim한 값을 `request.Name`에 다시 담아 UPDATE가 trim된 값을 쓰게 한다. `strings`는 이미 import되어 있다(users.go:22 사용).
  - `internal/server/roles_name_integration_test.go` (신규) — 아래 "검증 명령" 참고. 기존 하니스 그대로 재사용: `database.Open` → `DROP SCHEMA IF EXISTS ai_admin CASCADE; DROP SCHEMA IF EXISTS aiportal CASCADE` → `db.Migrate` → `db.Seed(ctx,"admin@example.com","integration-password")` → `secrets.New(bytes.Repeat([]byte{7},32))` → `New(db,cipher,slog...).Handler()` → `signIn(t, handler)`(`internal/server/db_error_integration_test.go:100`)와 `credentials.do(t, handler, "PATCH", target, body)`(같은 파일 118행). 대상 역할은 시드된 `user` 역할의 고정 UUID `00000000-0000-4000-8000-000000000004`(`internal/database/database.go:173`) — `super_admin`은 users.go:283에서 409 `system_role_locked`로 막히므로 쓰면 안 된다. `TEST_POSTGRES_DSN` 미설정이면 `t.Skip`.
  - `docs/api.md` — 290~294행 역할 표 아래에 한 줄: 역할 이름은 앞뒤 공백을 제거해 저장하며 1~120자여야 하고, 어기면 400 `name_required` / `name_too_long`. (`PATCH /api/v1/roles/{id}` 행의 설명도 "역할 이름·설명·permission 변경"으로 정정할지 판단 — 현재 "역할 설명·permission 변경"이라 이름 변경이 빠져 있다.)
  - 선택: `internal/server/openapi.go`에 역할 수정 요청 스키마가 있으면 `maxLength: 120`을 반영(있는지 먼저 확인할 것, 미확인).

- **검증 명령** (이 저장소에서 실제로 도는 것):
  ```
  # 전용 폐기 DB. 55432/55433/55439/15434 는 다른 세션이 점유 중이므로 새 포트를 쓸 것
  docker run -d --rm --name ai-admin-brief-pg -p 55440:5432 \
    -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=aiadmin postgres:16-alpine
  export TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55440/aiadmin?sslmode=disable'

  go test -v -count=1 ./internal/server -run '^TestUpdateRoleValidatesTheName$'   # SKIP 이 아니라 PASS 여야 함
  go test -count=1 ./...
  go test -race -count=1 ./...        # internal/server 약 70~80초
  make lint                            # gofmt -l · go vet · scripts/verify-version.sh (1.2.24 일관)
  go build ./...
  ```
  웹 변경이 없으면 `cd web && npm test`는 생략 가능(과거 회차의 관례). 끝나면 `docker rm -f ai-admin-brief-pg`.

- **위험과 피할 것**:
  - **되돌림 재현 필수**: 검증 블록을 지우면 121자 요청이 실제로 500 `role_update_failed`가 되는 것을 눈으로 확인한 뒤 복구할 것. "출력이 바뀌지 않는 수정은 넣지 말라"는 운영자 규칙 때문에, 재현이 안 되면 이 과제는 성립하지 않는다(그때는 차선 후보로).
  - `super_admin` 보호 분기(users.go:283 `system_role_locked`)와 `PermissionCodes` 검증·트랜잭션 순서는 건드리지 말 것. 새 검사는 **트랜잭션 밖·DB 접근 전**에 두어 잠금 시간을 늘리지 않는다.
  - `request.Name`은 `*string`이다. nil(미전송)과 `""`(전송했으나 빈 값)을 반드시 구분할 것 — nil까지 400으로 막으면 설명·권한만 바꾸는 기존 웹 호출(`RolesPage.tsx:33`은 항상 name을 보내지만 API 계약상 선택 필드)이 깨진다.
  - `description`(text, 무제한)은 이번 범위 밖이다. 같이 손대지 말 것.
  - 보호 경로 회피: `internal/auth`, `internal/server/oidc.go`, `workflow.go`, `internal/database/migrations`, `.github/workflows`, `internal/ui/dist`는 이번 과제에서 열 필요가 없다. **마이그레이션을 추가해 컬럼을 넓히는 방향은 금지** — 요청 검증만 한다.
  - 통합 테스트는 `DROP SCHEMA`를 하므로 운영/공유 DB 금지, 같은 DB로 병렬 실행 금지. DSN 없으면 skip이므로 `-v`로 PASS를 확인할 것.
  - VERSION·CHANGELOG는 건드리지 않는다(릴리즈 단계의 일). 커밋 전 `git status`로 빌드 산출물이 섞이지 않았는지 확인.

- **차선 후보**: `decideApproval`이 `approval_action.comment`에는 trim한 값(NULL 허용)을, `approval_request.decision_comment`에는 원문을 길이 제한 없이 저장해 같은 결정의 두 기록이 달라짐 (`internal/server/workflow.go:282` vs `:290`·`:292`) — 두 경로가 같은 값을 같게 읽도록 trim한 값 하나를 두 쿼리에 모두 넘기고 상한을 둔다. 단 `workflow.go`는 승인 트랜잭션이라 위험 구역이므로 1순위가 성립하지 않을 때만.
