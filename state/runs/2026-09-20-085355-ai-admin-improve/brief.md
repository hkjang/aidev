# 과제서 2026-09-20 — ai-admin

- 과제: `rotateKey`가 소유자 계정이 비활성인 키를 그대로 회전하던 문제 수정 — v1.2.21의 `user_not_active`·`key_target_unavailable` 계약을 회전 경로에도 적용 (가치 2 / 위험 1 / 작업량 S)

- 왜: `POST /api/v1/keys/{id}/rotate`(`internal/server/keys.go:181 rotateKey`)는 키 status·만료·scope·권한은 검사하지만 **키 소유자(`api_key.user_id`) 계정이 `locked`/`disabled`인지는 보지 않는다** — 관리자가 `keys.manage`로 남의 키를 회전하면 아무도 못 쓰는 새 키가 발급되고 기존 키는 `rotated`로 닫히며, 고권한 키면 승인 요청까지 접수된 뒤 `executeApprovedOperation`의 회전 분기(`workflow.go:419~`)도 소유자 상태를 확인하지 않아 비활성 계정에 새 키를 만든다(발급 분기 `workflow.go:386-392`는 v1.2.21에서 `errKeyTargetUnavailable`로 막았지만 회전 분기에는 그 검사가 없음 — 읽어서 확인). 고치면 발급과 회전이 같은 계약(404/409, 승인 요청 미접수, 대기 중 비활성화 시 409 `key_target_unavailable`+pending 유지)을 가져 `docs/api.md:280`의 설명이 회전에도 참이 된다.

- 수용 기준:
  1) 소유자 계정이 `disabled`(또는 `locked`)인 활성 키를 관리자가 회전하면 409 `user_not_active`(details `status`에 현재 상태), 새 `api_key` 행 없음, 기존 키 status는 `active` 그대로(트랜잭션 롤백).
  2) 고권한 키 워크플로(`api_key_privileged`)가 켜진 상태에서 같은 요청을 보내도 409이고 `approval_request`가 0건 늘어남(승인 요청 접수 전에 거부).
  3) 승인 대기 중 소유자를 비활성화한 뒤 검토자가 승인하면 409 `key_target_unavailable`, 요청은 `pending` 유지, 새 키 없음, 기존 키 `active` 유지 — 발급 경로와 동일한 계약.
  4) 소유자가 활성이면 기존 동작 불변: 201 + `secret`·`showOnce`, 기존 키 `rotated`(기존 테스트 `keys_security_test.go`·`workflow` 관련 테스트가 그대로 통과).
  5) 통합 테스트가 1)~4)를 실제 `New(...).Handler()`와 PostgreSQL을 지나며 증명하고, 검사 블록을 제거하면 실제로 실패함(되돌림 검증 후 복구).

- 건드릴 파일:
  - `internal/server/keys.go:rotateKey` — `SELECT user_id,name,scopes,expires_at,status FROM ai_admin.api_key WHERE id=$1 FOR UPDATE`(198행)를 `app_user`와 JOIN해 소유자 status를 함께 읽거나, 그 직후 `SELECT status FROM ai_admin.app_user WHERE id=$1`을 한 번 더 조회. `status != "active"` 검사(207행)와 권한 검사(211행) 다음, 승인 요청 생성(241행) **앞**에서 소유자 status가 `active`가 아니면 `writeError(w, http.StatusConflict, "user_not_active", "비활성 계정의 API 키는 회전할 수 없습니다.", map[string]string{"status": ownerStatus})`. 조회 실패는 기존대로 500 `key_rotate_failed`. 소유자가 호출자 본인이면(`owner == p.ID`) 이미 인증된 활성 계정이므로 검사를 건너뛰어도 되지만, 항상 검사해도 무해하다(단순한 쪽 택할 것).
  - `internal/server/workflow.go` 회전 분기(419행 이후, `SELECT user_id,... FROM ai_admin.api_key ... FOR UPDATE` 근처) — 발급 분기 386-392행과 같은 `SELECT EXISTS(SELECT 1 FROM ai_admin.app_user WHERE id=$1 AND status='active')` 검사를 넣고 실패 시 `errKeyTargetUnavailable` 반환(339행의 매핑이 이미 409 `key_target_unavailable`로 바꿔 준다 — 메시지가 "받을 계정" 표현이라 회전에도 읽히는지 확인, 필요하면 문구만 다듬기).
  - `internal/server/key_target_integration_test.go` — 기존 `TestKeyIssuanceForAnotherUserChecksTheTargetAccount`의 셋업(스키마 리셋·Seed·`signIn`·`disabledOwner`/`activeOwner` 삽입·`assertKeyCount`·`signInReviewer`)을 재사용해 `TestKeyRotationChecksTheOwnerAccount`를 같은 파일 또는 새 파일 `key_rotate_target_integration_test.go`에 추가. 키는 `db.Pool.Exec`로 직접 INSERT하거나 활성 소유자에게 발급 후 `UPDATE app_user SET status='disabled'`로 전환.
  - `docs/api.md:282` 회전 문단에 계약 한 문장 추가(소유자 비활성 409 `user_not_active`, 승인 대기 중 비활성화 409 `key_target_unavailable`).
  - `internal/server/server_test.go` 등 단위 테스트가 있으면 분류 표를 더해도 좋으나 필수는 아님.

- 검증 명령:
  - `docker run -d --name ai-admin-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=ai_admin -p 55432:5432 postgres:16-alpine` 후 `TEST_POSTGRES_DSN='postgres://postgres:postgres@localhost:55432/ai_admin?sslmode=disable' go test -race -count=1 -run 'TestKey|TestApproval|TestWorkflow' ./internal/server/ -v` (skip 아닌 실행인지 `-v`로 확인)
  - `TEST_POSTGRES_DSN=... go test -race -count=1 ./...` (internal/server 약 60~70s)
  - `make lint` (`gofmt -l .`·`go vet ./...`·`scripts/verify-version.sh`) · `go build ./...`
  - 웹 변경 없음 → `npm test` 생략 가능.

- 위험과 피할 것:
  - `internal/auth`·`auth_handlers.go`·`oidc.go`·마이그레이션·`.github/workflows`·VERSION·CHANGELOG는 건드리지 않는다.
  - `rotateKey`는 `FOR UPDATE` 행 잠금을 잡고 있고 승인 요청 생성 전 `tx.Rollback`(249행)을 명시적으로 한다 — 새 검사는 그 Rollback **앞**(잠금 안, 승인 접수 전)에 두어 실패 시 defer Rollback으로 정리되게 한다. 승인 분기 뒤로 넣으면 수용 기준 2)가 깨진다.
  - 승인 실행 경로(`workflow.go`)는 이미 `approvalExecutionProblem` 매핑(339행)이 있으니 새 오류 코드를 만들지 말고 `errKeyTargetUnavailable`을 재사용한다.
  - `internal/ui/dist`가 diff에 섞이지 않도록 커밋 전 `git status` 확인.
  - 오류 응답 규약: `writeError(w, status, code, 한국어 메시지, details)`.

- 차선 후보: `safeCSVCell`이 선행 tab(0x09)·CR(0x0D)을 중화하지 않음 — `internal/server/audit_events.go:220`에서 `strings.ContainsRune("=+-@", rune(value[0]))`를 `"=+-@\t\r"`로 넓히고 `server_test.go:155` 옆에 표 테스트(`=`, `+`, `-`, `@`, `\t`, `\r`, 일반 문자열, 빈 문자열, 선행 공백) 추가. DB 불필요, 15분 과제. (가치 2 / 위험 1 / S)
