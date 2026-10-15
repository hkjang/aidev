# 과제서 (2026-10-16, base main@2b2ef67 / VERSION 1.2.37)

- 과제: 승인 요청 제목(`varchar(300)`)·결정 코멘트(`text`)의 NUL(U+0000) 을 저장 전 400 으로 막기 (가치 3 / 위험 2 / 작업량 S)

- 왜: PostgreSQL 은 `varchar`/`text` 어느 쪽에도 NUL 을 담을 수 없는데(22P05), `createApproval`(`internal/server/workflow.go:160-164`)은 `strings.TrimSpace`+`utf8.RuneCountInString(…)>300` 만 보고, `decideApproval`(`workflow.go:235-243`)도 빈값·2000자만 보므로 `"\u0000"` 이 든 제목/코멘트가 그대로 INSERT/UPDATE 로 내려가 호출자 입력 오류가 400 이 아니라 500(`approval_create_failed`/`approval_failed`)으로 나간다. 같은 저장소가 이미 `app_user.display_name`·`email`(addbbf7), `role.name`·`api_key.name`·공급자 `name`/`default_model`(3dce1a1)에서 "컬럼이 담을 수 없는 값은 쓰기 전에 400" 으로 못박아 둔 자리와 같은 결함이며, 승인 경로만 남아 있다. 고치면 승인 요청·결정이 입력 오류를 400 으로 되돌려주고, 결정 트랜잭션이 DB 오류로 롤백되는 경로가 하나 줄어든다.

- 수용 기준:
  1) `POST /api/v1/approvals` 에 `title` 이 NUL 을 포함(선행/후행/단독 3사례)하면 **400 `approval_invalid`** 이고 `ai_admin.approval_request` 행 수가 변하지 않는다(현재는 500 `approval_create_failed` — 미확인, 아래 참고).
  2) `POST /api/v1/approvals/{id}/reject` (및 `/approve`)의 `comment` 가 NUL 을 포함하면 **400**(코드는 `comment_invalid` 권장)이고, 대상 요청의 `status` 가 `pending` 그대로이며 `approval_action` 행이 생기지 않고 `decision_comment` 가 NULL 로 남는다(현재는 500 `approval_failed`).
  3) NUL 제목은 정책 우회(`workflowRequired`=false)보다 **앞에서** 거부된다 — `workflow.enabled=false` 인 상태에서도 NUL 제목은 400 이고 `{"status":"bypassed"}` 200 이 아니다(검사 위치가 기존 길이 검사와 같은 자리이므로 자동 충족되지만 테스트로 고정할 것).
  4) 회귀: 여섯 글자 리터럴 `\u0000`(백슬래시+u+0000 문자열)은 계속 202 로 저장되고 조회 시 같은 값으로 돌아온다. 기존 상한(제목 300자 통과 / 301자·공백만 400 `approval_invalid`, 코멘트 2000자 통과 / 2001자 400 `comment_too_long`, 반려 빈 코멘트 400 `comment_required`)이 그대로 유지된다.
  5) `TestApprovalCommentContract` 가 그대로 PASS 한다(코멘트 정규화·NULL 계약 불변).
  6) red→green→역검증: 추가한 NUL 검사만 중성화(예: 조건을 `false` 로)하면 1)·2) 사례가 다시 500 으로 FAIL 하고 회귀 4) 는 계속 PASS 함을 로그로 남긴다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/server/workflow.go:createApproval` — 기존 400 조건 `request.Title == "" || utf8.RuneCountInString(request.Title) > 300` 에 `|| !storableInVarchar(request.Title)` 를 더한다. 오류 코드·메시지는 기존 `approval_invalid` 를 그대로 쓴다. `varchar(300)` 리터럴 옆 근거 주석 한 줄(기존 `users.go:421` 주변 스타일).
  - `internal/server/workflow.go:decideApproval` — `comment_too_long` 검사 바로 뒤(트랜잭션 `Begin` 보다 앞)에 `!storableInVarchar(comment)` → `writeError(w, 400, "comment_invalid", …)`. `text` 컬럼도 NUL 을 담을 수 없다는 점을 주석에 명시(기존 234행 주석이 "unbounded text columns" 를 이미 설명하므로 그 문장에 붙여도 됨). 코드명을 새로 만들기 싫으면 `comment_too_long` 재사용 대신 `comment_invalid` 를 쓰고 `docs/api.md` 에 적을 것.
  - `internal/server/users.go:421 storableInVarchar` — **읽기만**. 이름·구현을 바꾸지 말고 같은 패키지에서 호출한다(`strings.ContainsRune(value, 0)`).
  - `internal/server/approval_nul_integration_test.go` (신규) — `approval_comment_integration_test.go:26-62` 셋업을 그대로 복제: `TEST_POSTGRES_DSN` 없으면 `t.Skip` → `database.Open` → `DROP SCHEMA IF EXISTS ai_admin CASCADE; DROP SCHEMA IF EXISTS aiportal CASCADE` → `db.Migrate` → `db.Seed("admin@example.com","integration-password")` → `secrets.New(bytes.Repeat([]byte{7},32))` → `New(db,cipher,logger).Handler()` → `signIn(t,handler)` → `UPDATE ai_admin.system_setting SET value='true'::jsonb WHERE key='workflow.enabled'`. 제목 사례는 `UPDATE ai_admin.workflow_policy SET enabled=true WHERE code='general_review'` 후 `POST /api/v1/approvals` `{"policyCode":"general_review","title":…,"payload":{…}}`(`general_review` 는 `executableWorkflowPolicy`(`workflow.go:838-845`)에 없으므로 `approval_policy_reserved` 로 막히지 않는다). 코멘트 사례는 같은 파일처럼 `UPDATE … WHERE code='api_key_privileged'` + `signInReviewer(t,db,handler)` + `POST /api/v1/keys {"name":…,"scopes":["admin:read"],"expiresInDays":7}` → 202 `data.approvalId` → `reviewer.do(... "/api/v1/approvals/"+id+"/reject", {"comment": …})`. 헬퍼 정의 위치(확인): `signIn` = `db_error_integration_test.go:100`, `signInReviewer` = `key_target_integration_test.go:134`. `workflow_policy.enabled` 기본값은 `false`(`001_ai_admin.sql:149`)이므로 두 정책 모두 명시적으로 `enabled=true` 로 올려야 INSERT 까지 내려간다.
  - `docs/api.md` — 승인 절에 제목·코멘트가 NUL 을 담을 수 없다는 한 문장(기존 역할/키/공급자 절 문장과 같은 어투). 감사 CSV 절·다른 절은 건드리지 않는다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 전용 폐기 DB 띄우기: `docker run -d --rm --name ai-admin-approval-nul-pg -e POSTGRES_PASSWORD=test-password -e POSTGRES_USER=ai_admin -e POSTGRES_DB=ai_admin_test -p 55581:5432 postgres:16-alpine` (포트는 다른 회차 컨테이너와 겹치지 않게; 기존 컨테이너 재사용 금지)
  - `TEST_POSTGRES_DSN='postgres://ai_admin:test-password@127.0.0.1:55581/ai_admin_test?sslmode=disable' go test -count=1 -run '^TestApprovalNULWrites$' -v ./internal/server` (신규 테스트명은 자유, `-v` 로 SKIP 아님을 확인)
  - `TEST_POSTGRES_DSN=… go test -race -count=1 -run '^Test(ApprovalNULWrites|ApprovalCommentContract|KeyRotateTarget.*)$' -v ./internal/server`
  - `TEST_POSTGRES_DSN=… go test -race -count=1 ./...` (internal/server 약 140~160초)
  - `make lint` (gofmt·go vet·verify-version 1.2.37), `go build ./...`, `git diff --check`
  - 끝나면 `docker rm -f ai-admin-approval-nul-pg`

- 위험과 피할 것:
  - `workflow.go` 는 보호 구역(승인 실행·API 키 회전)이다. **두 검증 지점만** 고치고 `executeApproved*`·`errKeyStale`·회전 트랜잭션·`createApprovalRecordIn` 의 SQL 은 건드리지 말 것. 검사는 모두 `tx, err := s.db.Pool.Begin(...)` 보다 앞이어야 한다.
  - `request.Payload`(`$5::jsonb`, `createApprovalRecordIn:197-209`)의 NUL 도 같은 꼴로 500 이 나지만 **이번 범위 밖**이다 — jsonb 값을 걷는 새 헬퍼가 필요하고, 미머지 브랜치 `auto/2026-10-04-1142`(f66d25c)가 `jsonObjectHasNUL`·`storablePreferenceObject` 라는 이름을 이미 쓰고 있어 충돌한다. 그 두 이름은 만들지도, 재사용하지도 말 것.
  - `updatePreferences`(`users.go`)·`updateKeyScope`(`keys.go`)는 이번에 손대지 않는다(각각 미머지 브랜치 f66d25c, verify-failed 커밋 b5146f2 와 얽혀 있음).
  - 판정은 디코딩된 Go 문자열의 rune 으로 한다(원문 substring 검사 금지). 그래야 여섯 글자 리터럴 `\u0000` 은 계속 저장된다(수용 기준 4).
  - 감사 `details` 에 요청 원문(제목·코멘트)을 넣지 말 것. 기존 `s.audit(..., map[string]any{"policyCode": …})` 형태를 유지한다.
  - 기존 오류 코드·메시지·HTTP 상태를 바꾸지 말 것(`approval_invalid` 문구 유지, trim 정책 변경 금지). 웹(`web/`)·`internal/ui/dist`·VERSION·CHANGELOG·migration·`.github/workflows` 는 건드리지 않는다.
  - DB 통합 테스트는 `ai_admin`·`aiportal` 스키마를 DROP CASCADE 한다. 공유·운영 DB 금지, 다른 회차 컨테이너와 포트·DB 공유 금지.
  - **미확인**: 현재 코드가 NUL 제목/코멘트에서 실제로 500 을 내는지는 이번 정찰에서 DB 를 띄워 재현하지 않았다(근거는 세 번 확인된 같은 패턴 + `varchar(300)`/`text` 컬럼 타입). 구현자는 반드시 red 를 먼저 찍고, 만약 400/202 가 나오면(예: 다른 곳에서 이미 거부) 과제를 차선 후보로 교체할 것.

- 차선 후보: chat 요청의 `stream` 타입 오류와 두 토큰 한도 동시 지정의 HTTP 거부 계약을 통합 테스트로 고정 (`providers.go:611-634` 의 `stream_invalid`·`max_tokens_conflict`·`max_tokens_invalid` 세 코드는 `internal/server` 테스트에서 문자열 일치가 없다 — 프로덕션 0파일, 기존 `chat_request_json_integration_test.go` fixture 재사용, 잘못된 타입·동시 지정 400 + upstream 호출 0회와 정상 값 200 + 1회를 고정).
