# 과제서 (2026-09-24 정찰)

- 과제: `leaveMoim`이 "없는 Moim"·"애초에 회원이 아님"까지 409 `owner_cannot_leave`("Moim 소유자는 나갈 수 없습니다")로 답하는 것을 실제 원인대로 나누기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `backend/internal/httpapi/social.go:583 leaveMoim`은 `DELETE FROM moim_members … AND mm.role<>'owner'`의 `RowsAffected()==0`을 한 덩어리로 409 `owner_cannot_leave`로 보고하는데, 이 0행은 (a) 정말 소유자인 경우 말고도 (b) slug가 없는 경우, (c) 이미 탈퇴했거나 애초에 회원이 아닌 경우에 똑같이 발생합니다. `frontend/src/pages/MoimsPage.tsx:207`의 `join()`은 실패를 `readableError(error)`로 그대로 토스트에 띄우므로, 나가기 버튼을 두 번 누르거나(두 번째 DELETE는 0행) 목록이 오래된 상태에서 나가기를 누른 사용자에게 "Moim 소유자는 나갈 수 없습니다"라는 사실과 다른 문구가 보입니다. 원인별로 404/204/409를 나누면 사용자는 맞는 안내를 받고 운영자는 로그에서 소유자 시도와 유령 slug를 구분할 수 있습니다.

- 수용 기준:
  1) `DELETE /api/v1/moims/{slug}/join` 과 alias `DELETE /api/v1/moims/{slug}/members` 가 다음 계약을 지킨다.
     - 회원(소유자 아님)이 나가면 지금처럼 **204 No Content**, 멤버 행이 실제로 삭제된다.
     - 소유자가 나가려 하면 지금 그대로 **409 `owner_cannot_leave`**, 문구도 "Moim 소유자는 나갈 수 없습니다" 그대로 유지(멤버 행은 남는다).
     - slug가 존재하지 않으면 **404 `not_found`** — 메시지는 `getMoim`(social.go:559)과 같은 "Moim을 찾을 수 없습니다"를 재사용.
     - **공개** Moim의 비회원이 나가기를 요청하면 **204**(가입이 재요청에 200으로 멱등한 것과 대칭). 새로 삭제되는 행은 없다.
     - **비공개** Moim의 비회원이 나가기를 요청하면 **404 `not_found`** — 비공개 Moim의 존재를 드러내지 않기 위해 `getMoim`의 기존 판단(비공개+비회원=404)과 같게 둔다.
     - 위 판별 조회 자체가 실패하면 **500 `storage_error`** — `leaveMoim`의 기존 Exec 실패 경로와 같은 코드·문구("Moim에서 나갈 수 없습니다")를 재사용.
  2) `api/openapi.yaml`의 `/api/v1/moims/{slug}/join` delete와 `/api/v1/moims/{slug}/members` delete(466~489줄 부근)에 join 쪽과 같은 스타일로 `description` 한 줄만 추가해 위 계약을 적는다. **응답 목록(`responses`)에 새 코드를 추가하지 말 것** — 이 문서는 다른 경로에서도 성공 코드만 나열하는 관례이고, 지난 회차(c8d3628)에서 같은 결정을 이미 내렸다. `scripts/check-openapi-routes.mjs`의 120개 method/path 수가 변하면 안 된다.
  3) 테스트가 증명할 것: 수정 **전** 코드에서 "없는 slug"와 "공개 Moim 비회원"과 "비공개 Moim 비회원" 케이스만 409로 실패하고, "소유자 409"·"회원 204"는 수정 전후 모두 통과할 것. 즉 red 단계에서 어떤 케이스가 실패하는지 먼저 눈으로 확인한 뒤 구현한다(TDD). 저장 오류 500 케이스는 대역이 아니라 실제 PostgreSQL에서 만들어야 한다.

- 건드릴 파일:
  - `backend/internal/httpapi/social.go:583 leaveMoim` — `tag.RowsAffected()==0` 분기 안에서 판별 조회 **1회**를 추가해 404/204/409/500으로 나눈다. 권장 형태(한 번의 `QueryRow`로 끝내고 `errors.Is(err, pgx.ErrNoRows)`로 slug 없음을 잡는다):
    ```go
    var visibility string
    var isOwner bool
    err = s.repo.Pool().QueryRow(r.Context(),
        `SELECT m.visibility, EXISTS(SELECT 1 FROM moim_members WHERE moim_id=m.id AND user_id=$2 AND role='owner') FROM moims m WHERE m.slug=$1`,
        slug, getPrincipal(r).User.ID).Scan(&visibility, &isOwner)
    ```
    - `errors.Is(err, pgx.ErrNoRows)` → 404 `not_found` "Moim을 찾을 수 없습니다"
    - `err != nil` (그 외) → 500 `storage_error` "Moim에서 나갈 수 없습니다"
    - `isOwner` → 409 `owner_cannot_leave` (기존 문구 유지)
    - `visibility != "public"` → 404 `not_found`
    - 그 외 → `w.WriteHeader(http.StatusNoContent)` (기존 성공 경로와 동일)
    - 지금 코드는 slug를 인라인(`strings.ToLower(chi.URLParam(r,"slug"))`)으로 쓰므로 `slug` 지역 변수를 먼저 뽑아 두 쿼리가 같은 값을 쓰게 할 것(`joinMoim`이 이미 그렇게 한다). `errors`·`pgx` import는 같은 패키지의 `posts.go:1099`·`workflow.go:128`에 이미 있는 관례대로.
  - `backend/internal/httpapi/moim_join_postgres_integration_test.go` — 새 테스트 함수 `TestPostgreSQLLeaveMoimSeparatesNotFoundFromOwner`(이름은 재량)를 **같은 파일 또는 새 `moim_leave_postgres_integration_test.go`**에 추가. 기존 파일의 fixture 구조(`MOINA_TEST_POSTGRES_DSN` 없으면 `t.Skip`, `store.Open` → `secure.New` → `repository.CreateSession`으로 세션 쿠키·CSRF 발급 → 실제 `New(repository…)` 서버 배선으로 요청)를 그대로 복제할 것. 저장 오류 케이스가 필요하면 같은 파일 40~60줄의 **테스트 전용 트리거**(`moina_test_refuse_join_%d`) 기법을 `BEFORE DELETE ON moim_members`로 바꿔 sentinel 사용자만 거부하게 만든다. `t.Cleanup`으로 trigger/function/moims/sessions/audit_events/users를 반드시 지울 것 — 기존 파일의 cleanup 목록을 그대로 따라갈 것.
  - `api/openapi.yaml:473`·`:485` — delete에 `description` 한 줄 추가.
  - **프런트는 건드리지 말 것**: `owner_cannot_leave` 문자열은 저장소 전체에서 `social.go:590` 한 곳뿐이다(grep 확인). 프런트는 코드가 아니라 서버 `message`를 그대로 보여 주므로 수정이 필요 없다.

- 검증 명령 (워크트리 루트 `/home/hkjang/.cache/auto-improve-wt/moina` 기준):
  - red 확인(수정 전): `cd backend && MOINA_TEST_POSTGRES_DSN=... go test ./internal/httpapi/ -run 'LeaveMoim' -count=1 -v`
  - `cd backend && go test -race -count=1 ./...` — DSN 없으면 integration이 `SKIP` 되므로 **반드시 DSN을 주고** 돌릴 것. 지난 회차들은 throwaway `docker run --rm -e POSTGRES_PASSWORD=… -p …:5432 postgres:17-alpine` 로 DSN을 만들어 `--- SKIP` 0줄을 확인했다. 이 방식을 그대로 쓸 것.
  - `cd backend && go vet ./...`
  - `make fmt` (변경 없어야 함), `make check` (OpenAPI route 계약 120개 유지)
  - `make lint` (staticcheck 2025.1.1)
  - 프런트·e2e는 무변경이므로 vitest·시각 회귀는 생략 가능. 생략했다면 그렇게 적을 것.

- 위험과 피할 것:
  - **보호 경로 회피**: `auth.go`·`oidc.go`·`mcp_oauth.go`·`store/migrations/`·`.github/workflows/`는 이번 과제에서 전혀 건드릴 일이 없다. migration을 새로 추가할 필요도 없다(스키마 변경 없음).
  - **`joinMoim`을 같이 손보지 말 것.** join 쪽의 `_ = …Scan(&exists)`(social.go:574)가 조회 오류를 삼키는 것은 아이디어 목록에 별도로 남아 있다. 이번에 함께 고치면 c8d3628에서 막 릴리즈한 join 계약을 다시 흔들게 된다. leave 한 handler만 건드린다.
  - **문구를 새로 만들지 말 것.** 404는 `getMoim`의 "Moim을 찾을 수 없습니다", 500은 leave 기존 "Moim에서 나갈 수 없습니다", 409는 기존 "Moim 소유자는 나갈 수 없습니다"를 그대로 재사용한다. 새 message는 프런트 토스트에 그대로 노출되므로 번역·톤 검토 대상이 된다.
  - **과거 교훈**: (1) PATCH 문서에서 handler만 보고 middleware 403을 지웠다가 3b95053에서 되돌린 적이 있다 — 이 두 경로는 `auth.With(s.requirePermission("social:write"))`(server.go:216·218)를 거치므로 401/403/invalid_csrf 응답은 handler 코드에 없어도 실제로 발생한다. OpenAPI에서 기존 응답을 **지우지 말 것**. (2) 운영자 규칙 "실제 동작이 바뀌지 않는 수정은 넣지 말 것" — 이 과제는 응답 코드와 사용자에게 보이는 문구가 실제로 바뀌므로 해당하지 않지만, 겸사겸사 리팩터를 끼워 넣지 말 것. (3) 대역(fake)·소스 문자열 검사를 증거로 쓰지 말 것 — 반드시 실제 `New(repository…)` 배선과 pgx를 통과하는 integration으로 증명한다.
  - **추가 쿼리 비용**: 판별 조회는 `RowsAffected()==0`인 드문 경로에서만 돌기 때문에 정상 탈퇴는 쿼리 수가 늘지 않는다. 성공 경로에 조회를 추가하지 말 것.
  - **미확인(추측)**: `moims.visibility`가 `'public'`/`'private'` 두 값만 갖는다는 것은 `createMoim`(social.go:502)의 검증에서 읽은 것이고 migration의 CHECK 제약은 직접 열어 확인하지 않았다. 코드에서 `visibility == "public"`으로 판정하면(부정 비교가 아니라) 제3의 값이 들어와도 안전한 쪽(404)으로 떨어진다 — 그렇게 쓸 것.

- 차선 후보: **`getMoim`(social.go:553)이 DB 조회 오류를 404 `not_found`로 흡수하는 것을 `errors.Is(err, pgx.ErrNoRows)` 기준으로 404와 500 `storage_error`로 분리** (가치 2 / 위험 1 / 작업량 S). `updatePost`(9d8a6d5)·`joinMoim`(c8d3628)에서 두 번 채택된 것과 똑같은 패턴이라 확실하지만, 같은 유형이 세 회차 연속이 되는 것이 부담이라 1순위로 두지 않았다. 1순위가 성립하지 않으면(예: leave 계약 판단이 과하다고 판단되면) 이것을 택한다.
