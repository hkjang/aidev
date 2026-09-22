# 과제서 (2026-09-22, base main@c336a30, VERSION v0.1.35)

- 과제: `joinMoim`이 DB 저장 오류를 404 `not_found`로 감추는 것을 500 `storage_error`로 분리 (가치 3 / 위험 1 / 작업량 S)

- 왜: `backend/internal/httpapi/social.go:565 joinMoim`은 `INSERT INTO moim_members … ON CONFLICT DO NOTHING`의 `err != nil`을 "가입할 수 있는 공개 Moim을 찾을 수 없습니다"(404 `not_found`)로 내려, 연결 끊김·제약 위반·트리거 거부 같은 저장 실패가 사용자에게는 "없는 Moim"으로 보이고 운영자는 storage 문제를 구분할 수 없습니다. 바로 아래 `leaveMoim`(social.go:583)은 이미 같은 상황을 500 `storage_error`로 보고하고 있고, 9/19 회차가 `updatePost`에 대해 같은 분리를 이미 채택·릴리즈했으므로(커밋 9d8a6d5) 이 저장소에 확립된 관례를 아직 적용되지 않은 마지막 자리에 맞추는 일입니다.

- 수용 기준:
  1) `POST /api/v1/moims/{slug}/join`(과 같은 핸들러를 쓰는 alias `POST …/members`)에서 INSERT 자체가 PostgreSQL 오류로 실패하면 HTTP 500 · `code = "storage_error"`를 반환한다. 메시지는 `leaveMoim`의 어투에 맞춰 새로 짓되(예: "Moim에 가입할 수 없습니다") 새 오류 코드는 만들지 않는다.
  2) 사용자의 실수는 지금과 **완전히 동일하게** 유지된다 — 없는 slug·비공개 Moim은 404 `not_found` + 기존 문구 "가입할 수 있는 공개 Moim을 찾을 수 없습니다", 이미 가입한 사용자는 200 `{"joined":true}`, 소유자·기존 멤버의 재가입도 200. 성공 경로 200 `{"joined":true}`도 그대로.
  3) 테스트가 증명할 것: 수정 **전** 코드에서는 저장 오류 케이스만 404로 실패하고(다른 케이스는 통과), 수정 후 네 케이스(성공 200 / 재가입 200 / 없는 slug·비공개 404 / 저장 오류 500 + 멤버 행이 생기지 않음)가 모두 통과한다. 저장 오류는 대역 DB가 아니라 이 테스트 전용 `BEFORE INSERT ON moim_members` 트리거가 이 테스트의 sentinel 값(예: 해당 테스트 사용자 ID)만 거부하게 해 실제 `New(repository…)` 서버 배선 → 세션 cookie → CSRF → pgx 경로에서 INSERT가 실패하게 만든다(운영자 지침: 손으로 주입한 대역을 증거로 삼지 말 것).

- 건드릴 파일:
  - `backend/internal/httpapi/social.go:565 joinMoim` — `err != nil` 분기를 `writeError(w, http.StatusInternalServerError, "storage_error", …)`로 바꾸고, `tag.RowsAffected() == 0` 분기의 기존 404 `not_found`는 그대로 둔다. 같은 분기 안의 `EXISTS` 조회는 현재 `_ = …Scan(&exists)`로 오류를 버리는데, 이것까지 손대면 "가입 가능한데 조회만 실패" 케이스의 계약이 바뀌므로 **이번 범위 밖**(아이디어 파일에 남겨 둠).
  - `backend/internal/httpapi/moim_join_postgres_integration_test.go` (신규) — 템플릿은 `backend/internal/httpapi/posts_update_postgres_integration_test.go`(트리거 생성·`t.Cleanup`의 DROP TRIGGER/FUNCTION·`MOINA_TEST_POSTGRES_DSN` 미설정 시 `t.Skip`·세션 토큰/CSRF 생성 패턴)를 그대로 따른다. 공개 Moim 1개와 비공개 Moim 1개를 seed한다.
  - `api/openapi.yaml:466` `/api/v1/moims/{slug}/join` (및 471행대 `/api/v1/moims/{slug}/members`) — 현재 `responses: {'200': …}`만 있다. 이 문서는 다른 경로도 500을 나열하지 않는 관례이므로 **응답 목록을 늘리지 말고** `summary` 옆에 `description` 한 줄로 "없는 slug·비공개 Moim은 404 `not_found`, 저장 실패는 500 `storage_error`"만 적는다(9/19 회차와 같은 처리). route 개수 120은 변하지 않아야 한다.

- 검증 명령 (이 저장소에서 실제로 도는 명령. 단, 이번 정찰 세션에서는 실행 승인이 나지 않아 재실행하지 않았으므로 구현자가 직접 돌릴 것):
  - 집중: `cd backend && MOINA_TEST_POSTGRES_DSN=<throwaway postgres:17-alpine DSN> go test -race -run 'TestPostgreSQLJoinMoim' ./internal/httpapi`
  - 전체: `cd backend && MOINA_TEST_POSTGRES_DSN=… go test -race ./...` (DSN 없으면 integration은 skip되고 저장 오류 케이스가 증명되지 않으므로 반드시 DSN을 띄울 것)
  - `cd backend && go vet ./...`
  - 루트 `make fmt`, 루트 `make check` (`scripts/check-openapi-routes.mjs`의 120 method/path 계약 포함)
  - 프런트는 무변경이므로 vitest·e2e는 생략 가능(`frontend/src/pages/MoimsPage.tsx:207`이 `/moims/{slug}/members`를 호출하지만 오류 `code`로 분기하지 않음 — grep으로 확인).

- 위험과 피할 것:
  - 보호 경로 회피: `store/migrations`에 새 파일을 만들지 말 것. 트리거는 테스트 안에서 만들고 `t.Cleanup`에서 반드시 DROP 한다(테스트 실패로 트리거가 남으면 이후 모든 테스트의 가입이 깨진다). `auth.go`·`oidc.go`·`mcp_oauth.go`·미들웨어는 건드리지 않는다.
  - 3b95053의 교훈: OpenAPI를 손볼 때 핸들러만 보고 미들웨어가 실제로 내는 403(`forbidden`·`invalid_csrf`)을 지우지 말 것. 이번에는 description 한 줄만 더하고 기존 응답 항목은 삭제·수정하지 않는다.
  - 404→500 확대 금지: `RowsAffected()==0` 경로와 비공개 Moim 경로가 500으로 새지 않는지 테스트로 못 박을 것. 여기가 깨지면 "없는 Moim"이 사용자에게 500으로 보인다.
  - `leaveMoim`·`updatePost`·`deletePost`는 이미 올바르므로 다시 손대지 말 것(효과 없는 변경은 반려 사유).
  - 리팩터 금지: social.go의 다른 핸들러들로 같은 정리를 확대하지 말고 `joinMoim` 하나만.

- 차선 후보: POST /media 거절 경로(415 `unsupported_media`, CSRF·인증 거부)를 실제 multipart로 `New(repository…)` 배선까지 지나는 PostgreSQL integration 테스트로 덮기 — `media_upload_postgres_integration_test.go`에 케이스 추가. 1순위가 성립하지 않을 때(예: 저장 오류를 트리거로 재현할 수 없을 때)만 고를 것이며, 이미 릴리즈된 413 `media_too_large` 수정은 반복하지 말 것.

## 근거로 실제 열어 본 것
- `backend/internal/httpapi/social.go:565-594` (joinMoim·leaveMoim 전문)
- `backend/internal/httpapi/server.go:215-218` (join/members 4개 route 등록, 모두 `requirePermission("social:write")`)
- `api/openapi.yaml:466-487` (join·members 응답이 200/204만)
- `backend/internal/httpapi/posts_update_postgres_integration_test.go:1-60` (트리거 기반 저장 오류 재현 선례)
- `Makefile` (test·fmt·check 타깃), `frontend/src/pages/MoimsPage.tsx:207` grep
- 미확인: `go test`/`make` 계열 명령을 이번 세션에서 실행하지 못했고(도구 승인 거부), PostgreSQL DSN도 이 환경에 없음. 트리거로 INSERT 실패를 만드는 것이 `ON CONFLICT DO NOTHING`과 결합했을 때도 `Exec`에서 오류로 올라오는지는 posts 선례로부터의 추론이며 직접 재현하지 않았다.
