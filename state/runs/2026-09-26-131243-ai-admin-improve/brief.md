- 과제: 승인 결정 코멘트(`decision_comment`)의 trim·NULL·길이 계약을 `approval_action.comment`와 일치시키기 (가치 2 / 위험 2 / 작업량 S)
- 왜: `internal/server/workflow.go`의 `decideApproval`은 같은 트랜잭션 안에서 한 결정을 두 곳에 기록하는데, `:282`의 `approval_action` INSERT는 `nullString(strings.TrimSpace(request.Comment))`를 쓰고 `:290`·`:292`의 `approval_request` UPDATE는 `request.Comment` 원문을 그대로 쓴다 — 그래서 앞뒤 공백이 붙은 코멘트는 두 기록의 값이 달라지고, 코멘트 없이 승인하면 action에는 `NULL`이지만 `approval_request.decision_comment`에는 빈 문자열 `''`이 남아 `GET /api/v1/approvals`의 `decisionComment`가 `null` 대신 `""`로 나온다(`workflow.go:139`가 `*string`을 그대로 직렬화). 두 컬럼 모두 상한 없는 `text`(`internal/database/migrations/001_ai_admin.sql:169,186`)여서 길이 검증도 없다. 한 값을 한 번 계산해 양쪽에 쓰고 요청 단계에서 상한을 걸면 감사 기록이 서로 어긋나지 않고, 거대한 코멘트가 500이 아니라 400으로 거부된다.
- 수용 기준:
  1) 코멘트를 `strings.TrimSpace`로 한 번만 정규화해 `approval_action.comment`와 `approval_request.decision_comment` 양쪽에 같은 값을 쓴다. `"  승인합니다  "`로 승인하면 두 컬럼 모두 `승인합니다`.
  2) 코멘트를 생략하거나 공백만 보내고 승인하면 `approval_request.decision_comment`도 `NULL`이 되어 API `decisionComment`가 `null`이다(현재는 `""`). `nullString(...)`을 두 UPDATE에도 적용.
  3) `len([]rune(comment)) > 2000`이면 `tx.Begin` 전에 400 `comment_too_long`으로 거부하고 승인 요청·action은 전혀 변하지 않는다(`status` 여전히 `pending`, `approval_action` 0건). 한글 2000자는 200으로 통과해야 한다(바이트가 아니라 rune 기준 — 저장소 관례).
  4) 반려의 기존 400 `comment_required`(공백만 → 거부)는 그대로 동작한다. 승인은 코멘트 없이도 계속 200.
  5) 통합 테스트가 위 1~4를 실제 PostgreSQL에서 증명하고, 수정 전에는 1·2·3이 실패하는 것을 구현자가 직접 확인한다.
- 건드릴 파일:
  - `internal/server/workflow.go:decideApproval` — `request.Comment` 디코딩 직후 `comment := strings.TrimSpace(request.Comment)` 한 줄을 만들고, `:231`의 반려 검사와 `:282` INSERT, `:290`·`:292` UPDATE가 모두 이 변수를 쓰게 한다. `:290`·`:292`의 인자를 `nullString(comment)`로 교체. 길이 검사는 `decision == "rejected"` 검사 바로 옆(즉 `principalFrom`/`tx.Begin` 앞)에 둘 것.
  - `internal/server/approval_comment_integration_test.go`(신규) — 기존 `internal/server/key_rotate_target_integration_test.go:25-60`의 셋업을 그대로 복제: `TEST_POSTGRES_DSN` 없으면 `t.Skip`, `database.Open` → `DROP SCHEMA IF EXISTS ai_admin CASCADE; DROP SCHEMA IF EXISTS aiportal CASCADE` → `db.Migrate` → `db.Seed("admin@example.com","integration-password")` → `secrets.New(bytes.Repeat([]byte{7},32))` → `New(db, cipher, slog…).Handler()`. 승인 요청을 만드는 가장 짧은 경로도 그 파일에 있다: `UPDATE ai_admin.system_setting SET value='true'::jsonb WHERE key='workflow.enabled'` + `UPDATE ai_admin.workflow_policy SET enabled=true WHERE code='api_key_privileged'` 후 `admin.do(t, handler, POST, "/api/v1/keys", {"name":…,"scopes":["ai.providers.write" 같은 고권한 scope],"expiresInDays":7})` → 202 + `data.approvalId`. 검토자 세션은 기존 헬퍼 `signInReviewer(t, db, handler)`(`internal/server/key_target_integration_test.go:134`), 관리자 세션은 `signIn(t, handler)`(`internal/server/db_error_integration_test.go:100`). 요청자와 검토자가 달라야 `self_approval_denied`를 피한다.
  - `docs/api.md` — 332~333행의 승인/반려 표 아래에 코멘트 계약을 한 문단으로: 앞뒤 공백 제거 후 저장, 빈 값은 `null`, 2000자 초과는 400 `comment_too_long`, 반려는 여전히 사유 필수.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 전용 폐기 DB: `docker run -d --name ai-admin-scout-pg -e POSTGRES_PASSWORD=postgres -p 55461:5432 postgres:16-alpine` → `export TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55461/postgres?sslmode=disable'` (55432·55433·55439·55444·55451·15434는 다른 세션이 점유한 이력이 있음)
  - `go test -count=1 ./internal/server/ -run ApprovalComment -v` — SKIP이 아니라 PASS인지 반드시 확인
  - `go test -race -count=1 ./...` (internal/server 약 80~100초)
  - `make lint`, `go build ./...`
  - 웹 변경 없음 — `web/src` 전체에 `decisionComment` 참조가 없음을 확인했다. `npm test`·`npm run build`·`internal/ui/dist` 재빌드 불필요.
- 위험과 피할 것:
  - `workflow.go`는 승인 트랜잭션(위험 구역). 새 검증은 반드시 `tx.Begin` **앞**에 둘 것 — 뒤에 두면 열린 트랜잭션을 남긴다(`defer tx.Rollback`이 있긴 하나 계약이 흐려진다).
  - `executeApprovedOperation`·`classifyApprovalExecutionError`·`errKeyStale`/`errKeyTargetUnavailable` 분기는 이번 과제와 무관하다. 건드리지 말 것.
  - `:290`(다단계 승인 중간 단계)과 `:292`(최종 결정) **둘 다** 고칠 것. 한쪽만 고치면 과거 교훈의 "같은 값을 읽는 두 경로의 계약 불일치"를 그대로 재현한다. 다단계(`approval_levels=2`) 경로까지 테스트하려면 서로 다른 검토자 2명이 필요해(`reviewer_already_acted`) S 범위를 넘을 수 있다 — 기본 `approval_levels=1` 경로로 수용 기준을 증명하고, 다단계는 코드 동일성으로 확보하면 충분하다.
  - 상한 2000은 이 저장소에 선례가 없는 새 숫자다(기존 선례: 검색어 200, 이름 160, 제목 300). 2000이 과하다고 판단되면 낮춰도 되지만, 그 숫자를 `docs/api.md`와 테스트에 일관되게 쓸 것.
  - 마이그레이션·`internal/auth`·`oidc.go`·`.github/workflows`·`internal/ui/dist`는 건드리지 않는다. 컬럼 타입(`text`)은 바꿀 필요 없다 — 요청 단계 검증으로 충분하다.
- 차선 후보: 감사 CSV 문서의 "성공 다운로드 = 필터에 맞는 전체 이벤트" 문구를 50,000건 상한으로 정정 (가치 1 / 위험 1 / S, `docs/api.md`만 수정 — 단 해당 문장의 현재 표현은 이번 정찰에서 직접 확인하지 않았다(미확인)).
