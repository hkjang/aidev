- 과제: 유실된 마지막 커밋 `972113f` 복원 — `followTopic`의 저장 실패를 404 `not_found`에서 500 `storage_error`로 분리 (가치 3 / 위험 1 / 작업량 S)
- 왜: `backend/internal/httpapi/social.go:282`의 `if err != nil || tag.RowsAffected() == 0`이 INSERT/UPDATE 실패와 "없는 Topic"을 한 덩어리로 404 "Topic을 찾을 수 없습니다"로 보고합니다(이번 회차에 현재 코드에서 직접 읽어 확인). 같은 파일 `unfollowTopic`(social.go:290)은 이미 쓰기 실패를 500 `storage_error`로 답하므로 한 파일 안에서 계약이 어긋나 있고, `frontend/src/pages/DiscoveryPages.tsx:16`·`:69`가 `readableError(error)`로 서버 message를 그대로 띄우므로 저장 장애가 사용자에게 "없는 Topic"으로 보입니다.
- 수용 기준:
  1) `POST /api/v1/topics/{slug}/follow`의 Exec 오류는 500 `storage_error`("Topic을 Link할 수 없습니다"), 0행(없는 slug)은 **기존** 404 `not_found`("Topic을 찾을 수 없습니다") — 두 분기로 분리.
  2) 성공 200 본문(`{"following":true,"weight":n}`), 기본 weight 50, 400 `invalid_weight`(1~100 밖), 그리고 **본문 없는 요청이 여전히 200**임이 무변경. 특히 `decodeOptionalJSON`(server.go:728)을 `decodeJSON`으로 되돌리지 말 것 — 972113f는 `1d8cf40` **이전**에 작성됐으나 social.go:271의 현재 `decodeOptionalJSON` 호출은 그대로 두는 것이 정답입니다.
  3) 새 integration 테스트 5케이스(성공·weight 갱신·400·404·저장 실패 500)가 **수정 전 코드에서 저장 실패 1케이스만 실패**하고 나머지 4케이스는 통과함을 red 단계로 눈으로 확인한 뒤, 수정 후 5/5 통과. 저장 실패는 테스트 전용 `BEFORE INSERT ON user_topic_follows` 트리거가 sentinel `user_id`만 거부해 만들고 대역(mock)을 쓰지 않습니다.
  4) OpenAPI route 수 120개 유지(`make check`), `responses` 목록 확대 없이 `description` 한 줄만 추가.
- 건드릴 파일 (프로덕션 1개):
  - `backend/internal/httpapi/social.go:followTopic` — 282줄의 합쳐진 조건을 `if err != nil { 500 storage_error }` / `if tag.RowsAffected() == 0 { 404 not_found }`로 분리.
  - `backend/internal/httpapi/topic_follow_postgres_integration_test.go` (신규, 972113f에 207줄로 존재) — 5케이스.
  - `api/openapi.yaml` — `POST /topics/{slug}/follow`(현재 392줄 근처, summary `Topic Link`)에 `description` 1줄.
- 가장 쉬운 길: `git cherry-pick -n 972113f` 로 세 파일을 그대로 되살리고, 검증을 통과한 뒤 원본 메시지로 커밋(`-n` 은 "검증 전 커밋 금지"를 지키기 위함 — 2026-09-29 회차가 같은 방식으로 성공). **충돌 없음 확인 근거**: `git diff --stat 972113f~1 HEAD` 이 세 파일 중 두 개만 바뀌었고 그 변경 위치가 972113f 의 hunk 와 겹치지 않습니다 — openapi.yaml 은 1~7줄(version 0.1.37→0.1.40)과 638~660줄(admin 신고 description 3줄)뿐이고 972113f 의 hunk 는 392줄, social.go 는 271줄(`decodeJSON`→`decodeOptionalJSON`)뿐이고 972113f 의 hunk 는 279~285줄입니다. (실제 cherry-pick 은 이번 정찰에서 미실행 — 코드 변경 금지 때문입니다. `git apply --check` 도 권한 거부로 못 돌렸습니다.)
- 복원한 테스트가 지금 코드에서 컴파일·통과하는지 확인한 것들 (모두 HEAD 에서 직접 확인):
  - `New(repo, secrets, version)` 3인자 그대로(server.go:108), `SessionCookie`(server.go:39), `Store.CreateSession(ctx, model.Session)`(store/store.go:197) 전부 동일.
  - 테스트가 쓰는 `pgQuoteLiteral` 은 같은 패키지 `posts_update_postgres_integration_test.go:176` 에 이미 있습니다(972113f 가 만든 것이 아니므로 복원해도 중복 선언이 아닙니다).
  - 첫 케이스가 `strings.NewReader("")`(→ `r.ContentLength == 0`)로 본문 없이 보내는데 `decodeOptionalJSON` 은 `ContentLength == 0` 에서 **즉시 true** 를 돌려줍니다(server.go:729). 즉 `1d8cf40` 의 "길이가 알려진 빈 본문은 400" 규칙(`ContentLength < 0` 일 때만 EOF 관용, server.go:740)에 걸리지 않고 200/404 를 유지합니다. 972113f 와 `1d8cf40` 은 충돌하지 않습니다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - throwaway DB: `docker run -d --rm -e POSTGRES_PASSWORD=postgres -p 55432:5432 postgres:16-alpine` (CI source 는 PostgreSQL 16)
  - `MOINA_TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55432/postgres?sslmode=disable' go test -race -count=1 ./...` (backend 디렉터리에서)
  - red 확인용: `... go test -race -count=1 -run TestPostgreSQLFollowTopicSeparatesStorageErrorFromNotFound -v ./internal/httpapi/`
  - `go vet ./...`, `make fmt`(형식 검사만, 수정 안 함), `make check`(OpenAPI route 120개), staticcheck 2025.1.1
  - `-v` 출력에서 `--- SKIP` 0줄(DSN 이 실제로 걸렸다는 증거)과 최상위 `TestPostgreSQL*` 개수 증가(직전 43 → 44)를 확인하세요.
- 위험과 피할 것:
  - **프런트·e2e 는 손대지 마세요.** `e2e/` 전체에 `follow` 문자열이 **0건**(이번 회차 grep 확인)이라 시각 회귀·smoke 는 이 변경을 지나지 않습니다. 2026-09-25 에 972113f 의 PR 이 verify-failed 한 원인은 image 잡의 `API and browser smoke` 였는데, e2e 가 follow 를 전혀 호출하지 않으므로 그 실패는 이 변경과 인과가 없습니다 — 그때 실패를 이번 변경 탓으로 단정하지도, 이번 변경을 그 실패의 수정으로 섞지도 마세요. 다시 같은 곳에서 걸리면 별건으로 보고하세요.
  - `decodeOptionalJSON` 본체(server.go:721~751)와 `decodeJSON`(:706)은 수십 handler 가 공유합니다 — 무변경.
  - `store/migrations` 금지(checksum), `auth.go`·`oidc.go`·`mcp_oauth.go`·`.github/workflows/*` 금지.
  - 트리거는 sentinel `user_id` 만 거부하게 유지하고 `t.Cleanup` 에서 `DROP TRIGGER`/`DROP FUNCTION` 하세요 — CI source 는 네 패키지가 한 DB 를 공유합니다.
  - 로그인 5분/동일 IP 5회 제한이 있으니 세션은 테스트처럼 `CreateSession` + 쿠키로 직접 만들고 실제 로그인을 반복하지 마세요.
  - `t.Skip`·`test.retry`·`continue-on-error`·`//lint:ignore` 추가 금지. 인과가 증명된 뒤 회귀 확인을 과잉 반복하지 마세요(과거 TIMEOUT 원인).
- 차선 후보: 새 500 출구가 pg 오류를 버려 운영자 로그에 원인이 없는 것 — `admin.go:resolveReport`·`auth.go:changePassword`·`admin.go:adminResetPassword` 의 500 경로가 `writeError` 만 호출하고 `err` 를 버립니다. 구조화 로그 한 줄(handler 이름·식별자·SQLSTATE 만, 사용자 원문·pg 메시지 전문 금지)을 추가하고 `slog` 핸들러를 갈아끼워 관측하는 테스트를 붙이는 과제(가치 3 / 위험 2 / S). 1순위가 성립하지 않을 때만 고르세요.
