- 과제: `storage_error` 500 출구가 pg 오류를 버려 운영자 로그에 원인이 남지 않는 것을 `writeStorageError` 한 곳으로 모은다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `httpapi`에는 `writeError(w, 500, "storage_error", …)` 출구가 약 160개 있고, 그 전부가 `err`를 버린다 — 운영자는 "신고를 처리할 수 없습니다" 500을 보고도 그것이 connection 고갈인지 제약 위반인지 디스크 full인지 알 수 없고, 사용자 신고 외에는 조사할 단서가 없다. 같은 패키지의 `oidc_discovery_error.go:97 writeOIDCDiscoveryError`가 이미 `observability.Logger(r.Context()).WarnContext(…, "error_code", …, "failure_reason", …, "cause_type", …)`로 request_id와 함께 원인을 남기는 관례를 갖고 있으므로, 그 모양을 `storage_error`에 그대로 옮기면 최근 네 회차가 새로 만든 500 출구(`resolveReport`·`changePassword`·`adminResetPassword`·`followTopic`)가 처음으로 관측 가능해진다.

- 수용 기준:
  1) 새 헬퍼 `writeStorageError(w, r, handler, err, message)`가 `writeError(w, 500, "storage_error", message)`와 **똑같은 응답**(status 500, `{"code":"storage_error","message":<그대로>}`)을 내보내고, 추가로 구조화 로그 한 줄을 남긴다. **HTTP 응답 본문·status·헤더는 바이트 단위로 불변** — 기존 integration 테스트가 전부 통과해야 한다(특히 아래 네 경로를 덮는 `admin_report_resolve_postgres_integration_test.go`, `password_session_revoke_postgres_integration_test.go`, `topic_follow_postgres_integration_test.go`).
  2) 로그에 `error_code=storage_error`, `handler=<핸들러 이름>`, `cause_type=<%T>`, `pg_code=<SQLSTATE 또는 빈 값>`이 있고 **request_id가 함께 찍힌다**(`observability.HTTPMiddleware`가 넣는 것). pg 메시지 전문·사용자 입력 원문·`err.Error()` 전체는 **절대 찍지 않는다** — `oidc_discovery_error_test.go:85`의 "upstream body was exposed" 단언이 지키는 바로 그 계약이다.
  3) 테스트가 증명해야 하는 것: (a) `slog.New(slog.NewJSONHandler(&logs, nil))` + `observability.HTTPMiddleware(logger)`로 감싼 핸들러에서 `writeStorageError`를 호출하면 `logs`에 위 네 필드와 `request_id`가 모두 나타난다, (b) 같은 호출의 응답이 `writeError(w,500,"storage_error",msg)`와 동일하다, (c) **민감 문자열을 담은 err를 넣어도 `logs`에 그 문자열이 나타나지 않는다**(sentinel 문자열로 단언 — `oidc_discovery_error_test.go`의 `secretBody` 패턴 복사), (d) `*pgconn.PgError{Code:"23505"}`를 넣으면 `pg_code=23505`가 찍히고, 평범한 `errors.New`은 `pg_code`가 빈 값이다.

- 건드릴 파일 (프로덕션 4개 + 테스트 1개 — 여섯 개 한도 안):
  - `backend/internal/httpapi/storage_error.go` (**신규**) — `writeStorageError(w http.ResponseWriter, r *http.Request, handler string, err error, message string)`. 본문은 `oidc_discovery_error.go:97-107`을 그대로 베낀다: `observability.Logger(r.Context()).ErrorContext(r.Context(), "저장 실패", "error_code", "storage_error", "handler", handler, "cause_type", deepestErrorType(err), "pg_code", pgSQLState(err))` 후 `writeError(w, http.StatusInternalServerError, "storage_error", message)`. `cause_type` 구현은 `deepestOIDCDiscoveryErrorType`(`oidc_discovery_error.go:116`)과 똑같은 `errors.Unwrap` 루프 — **그 함수를 직접 호출해도 되고**(같은 패키지라 가능) 범용 이름으로 복사해도 된다. 재사용이 더 좋지만 OIDC 전용 이름을 공용으로 쓰는 게 거슬리면 `storage_error.go`에 `deepestErrorType`을 두고 OIDC 쪽은 건드리지 말 것(OIDC 테스트 계약을 흔들지 않는 쪽이 안전).
  - `pgSQLState(err) string` — 같은 새 파일에 두고 `var pgErr *pgconn.PgError; if errors.As(err, &pgErr) { return pgErr.Code }; return ""`. **계층 걱정은 확인 결과 없다**: `httpapi`의 프로덕션 파일이 이미 `github.com/jackc/pgx/v5`를 직접 import한다(`admin.go:18`, `server.go:35`, `posts.go:23`, `preferences.go:14`, `workflow.go:14`, `outbox.go:20`, `notification_digest.go:12`), `resolveReport`는 `s.repo.Pool().Begin`으로 pgx 트랜잭션을 직접 다루고 `tag.RowsAffected()`의 `tag`는 이미 `pgconn.CommandTag`다. `pgx/v5/pgconn`은 `store`·`event`·`media`·`feed`가 이미 쓰는 같은 모듈의 하위 패키지이므로 새 의존성이 아니다.
  - 참고: `store.go:423-427`에 `store.IsNotFound`/`store.IsConflict`가 있지만 **SQLSTATE 문자열을 돌려주는 헬퍼는 없다**(`IsConflict`는 23505·23503을 bool로만 접는다). `store`에 새 exported 헬퍼를 추가하면 프로덕션 파일이 5개가 되니, `pgSQLState`는 `httpapi/storage_error.go`에 비공개로 두고 `store`는 건드리지 말 것.
  - `backend/internal/httpapi/admin.go` — **직접 읽어 확인한** 다섯 출구를 `writeStorageError(w, r, "<handler>", err, <기존 message 그대로>)`로 바꾼다. `resolveReport`의 네 곳은 전부 저장 실패다: `:389` `Pool().Begin` 실패, `:395` `UPDATE reports` Exec 실패, `:404` `INSERT INTO moderation_actions` 실패, `:409` `tx.Commit` 실패 — 네 곳 모두 message가 `"신고를 처리할 수 없습니다"`이고 그 사이의 `:398` `RowsAffected()==0`은 404 `not_found`이므로 **건드리지 말 것**. 그리고 `adminResetPassword`의 `:224`(`"비밀번호를 초기화할 수 없습니다"`) — 바로 위 `:221`의 `store.IsNotFound(err)` → 409 `not_local_user` 분기는 그대로 둔다. **message 문자열은 한 글자도 바꾸지 말 것** — 프런트가 `readableError`로 그대로 띄우고 integration 테스트가 본문을 단언한다.
  - `backend/internal/httpapi/auth.go:325` — `changePassword`의 `UpdatePasswordAndRevokeSessions` 실패 출구(`"비밀번호를 변경할 수 없습니다"`). 같은 함수 `:318`의 `password_error`(해시 실패)는 `storage_error`가 아니므로 범위 밖.
  - `backend/internal/httpapi/social.go:286` — `followTopic`의 INSERT 실패 출구(`"Topic을 Link할 수 없습니다"`, 3fe4190이 만든 것). 뒤따르는 `:290` `RowsAffected()==0` → 404는 그대로.
  - 호출 지점은 총 7곳이다(resolveReport 4 + adminResetPassword 1 + changePassword 1 + followTopic 1). `handler` 인자에는 핸들러 함수 이름을 그대로 넣어 로그에서 네 출구를 구분할 수 있게 하되, 네 출구를 더 세분하고 싶으면 `handler`를 `"resolveReport"`로 두고 message가 이미 같으니 `op`류 필드를 **새로 만들지는 말 것**(필드 수를 수용 기준 2의 네 개로 유지).
  - `backend/internal/httpapi/storage_error_test.go` (**신규**) — 위 (a)~(d). DB 불필요한 순수 `httptest` 테스트라 DSN 없이도 돈다.
  - **나머지 ~150개 `storage_error` 출구는 이번 범위가 아니다.** 한 회차에 다 바꾸면 파일 15개를 건드리게 되고, 과거 추적에서 파일 10개 이상 변경은 47%가 재작업을 탔다. 헬퍼가 머지된 뒤 파일 단위로 나눠 옮기는 것이 후속 과제다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) `cd backend && go build ./... && go vet ./...`
  2) 새 테스트만: `cd backend && go test -race -count=1 -run 'StorageError' -v ./internal/httpapi/`
  3) 전체(DB 포함 — **DSN을 걸어야 integration이 실제로 돈다**): throwaway `docker run --rm -d -e POSTGRES_PASSWORD=… -p …:5432 postgres:16-alpine`(CI source와 같은 메이저) 후 `MOINA_TEST_POSTGRES_DSN=… go test -race -count=1 ./...`. `-v`로 돌려 **`--- SKIP` 0줄**(DSN이 실제로 걸렸다는 증거)과 최상위 `TestPostgreSQL*` **PASS 44건**(v0.1.41 기준, 직전 회차 기록)을 확인할 것. 44건이 44로 유지되고 새 `TestStorageError*`가 늘어나는 것이 정상이다.
  4) `make fmt`(형식 검사만, 수정하지 않음) · `staticcheck@2025.1.1 ./...` · `make check`(OpenAPI route **120개** 유지 — 이번 과제는 라우트·응답 코드를 늘리지 않으므로 120이 바뀌면 뭔가 잘못된 것)
  5) 프런트·e2e는 **돌리지 않아도 된다**: 응답 본문이 불변이므로 `readableError` 경로가 바뀌지 않는다. OpenAPI도 **손대지 말 것** — 새 status code가 없다(관례상 `responses` 확대 금지).

- 위험과 피할 것:
  - **응답을 바꾸는 것이 유일한 진짜 위험이다.** message 문자열·status·`code`를 건드리면 네 개의 기존 integration 테스트와 프런트 토스트가 동시에 깨진다. 헬퍼는 `writeError`를 **감싸기만** 하고 대체 포맷을 만들지 말 것.
  - **로그에 비밀을 흘리는 것**: `err.Error()` 전문, `pgErr.Message`/`Detail`/`Hint`(여기에 위반한 행의 값이 들어간다 — 이메일·토큰 해시가 샐 수 있다), 요청 본문, 비밀번호. `cause_type`과 SQLSTATE 코드만. `changePassword`·`adminResetPassword`는 비밀번호 경로이므로 특히 조심.
  - 보호 경로: `auth.go`·`admin.go`는 인증/세션 파일이지만 이번 변경은 **각 파일의 500 출구 한두 줄**뿐이다. 인증 로직·세션 발급·`store/migrations`·`.github/workflows`는 손대지 말 것.
  - `oidc_discovery_error.go`를 리팩터하지 말 것 — `deepestOIDCDiscoveryErrorType`을 공용으로 "정리"하려다 `oidc_discovery_error_test.go`의 `cause_type` 계약을 흔들 수 있다. 재사용은 호출만, 이동·개명은 금지.
  - `slog.Default()`가 아니라 `observability.Logger(r.Context())`를 쓸 것 — 전자는 request_id가 안 붙어 수용 기준 2가 깨진다(`server.go:354`·`:389`가 후자를 쓰는 관례).
  - 로그인 5분/동일 IP 5회 제한이 있으니 integration을 반복 실행할 때 인증 단계 실패를 변경 탓으로 오해하지 말 것.
  - 과거 교훈: 인과 증명이 끝난 뒤 회귀 확인을 과잉 반복해 회차가 TIMEOUT으로 유실된 사례가 있다. 전체 테스트는 **한 번** 깔끔하게 돌리고 끝낼 것.

- 차선 후보: **`joinMoim`(social.go:576)·`createReport`(social.go:1108)의 `_ = Scan(&exists)`가 조회 오류를 404로 감추는 것** — 1순위가 `pgconn` 계층 문제로 축소되고도 가치가 없다고 판단되면 이것으로. 단 이 후보는 SELECT 실패 재현 수단(전용 스키마 헬퍼)이 선행이고 sentinel BEFORE 트리거로는 재현 불가라는 것이 여러 회차에 확인됐다 — 재현 수단 없이 대역 테스트로 때우지 말 것. 그보다 쉬운 세 번째 후보: **e2e README의 테스트 실행 순서 설명 충돌 정정**(`package.json`이 정본: visual→accessibility→smoke, 1/1/S).
