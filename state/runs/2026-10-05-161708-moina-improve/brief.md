- 과제: `posts.go`의 `storage_error` 500 출구 41곳을 `writeStorageError`로 이관한다 (가치 3 / 위험 1 / 작업량 M)
- 왜: v0.1.42가 만든 `writeStorageError`(`storage_error.go:22`)는 `httpapi`의 `storage_error` 500 출구 약 160곳 중 7곳(admin 5·auth 1·social 1)에만 적용돼 있고, 가장 많은 41곳을 가진 `posts.go`는 전부 `err`를 버린다. 게다가 `posts.go`의 message는 중복이 심해("Moin을 변경할 수 없습니다" 5곳, "첨부 미디어를 확인할 수 없습니다" 6곳, "Signal을 저장할 수 없습니다" 4곳) 운영자가 500을 보고도 **어느 쿼리가 거절됐는지조차** 알 수 없다 — `handler`·`cause_type`·`pg_code` 한 줄이 붙으면 그 구분이 생긴다.
- 수용 기준:
  1) `grep -n 'storage_error' backend/internal/httpapi/posts.go`의 41줄이 전부 `writeStorageError(w, r, "<감싼 함수 이름>", err, "<기존 message>")` 형태이고, status·`code`·`message` 문자열이 **한 글자도** 바뀌지 않았다 — `git diff`에서 message 한글 리터럴의 변경이 0줄임을 확인할 것.
  2) `go build ./...` · `go vet ./...` exit 0, `staticcheck@2025.1.1` 무지적, `gofmt -l .` 빈 출력, `make check` exit 0(OpenAPI route 120개 유지).
  3) **실제 핸들러 경로**의 red→green을 한 번 관측한다: `posts_update_postgres_integration_test.go`의 `"failed UPDATE is 500 storage_error"`(:153) 케이스에 로그 단언을 더해, `posts.go`를 고치기 **전에** `"handler":"updatePost"`·`"error_code":"storage_error"`·`"pg_code":"P0001"`가 없어 FAIL하는 것을 보고, 고친 뒤 PASS하는 것을 본다. 기법은 `admin_report_resolve_postgres_integration_test.go`(`Handler()` 직전 `slog.SetDefault` + lockedBuffer)와 `storage_error_test.go`에 이미 있으니 그대로 베낄 것. 응답 단언은 red·green 양쪽에서 통과해야 한다(= 응답 불변의 증거).
  4) `go test -race -count=1 -v ./...`가 `--- FAIL` 0줄 · `--- SKIP` 0줄(DSN이 실제로 걸렸다는 증거), 최상위 `TestPostgreSQL*` PASS **44건** 유지.
- 건드릴 파일 (프로덕션 **1개**):
  - `backend/internal/httpapi/posts.go` — 41곳. 함수 경계는 직접 확인했다(`grep -n '^func '`), 출구→`handler` 인자 매핑은 이렇다:
    - `writePostError`(:117, 출구 :123) — **여기만 서명 변경이 필요하다.** 지금 `writePostError(w http.ResponseWriter, err error)`로 `r`이 없다. `writePostError(w, r, handler string, err)`로 바꾸고 호출 3곳만 고치면 된다 — `createPost`(:102), `updatePost`(:901), `remoinPost`(:1080) 셋 다 `r`이 스코프에 있다(직접 확인). 세 호출자에 각각 `"createPost"`·`"updatePost"`·`"remoinPost"`를 넘기면 공용 fallback "Moin을 저장할 수 없습니다"가 비로소 구분된다. 다른 파일에는 `writePostError` 참조가 없다(`grep -rn` 확인).
      - 주의: :901의 호출은 `*publicError`를 넘기므로 `errors.As` 분기로 빠져 storage fallback에 닿지 않는다. 그래도 서명은 통일한다.
    - `:397` → `getPost` · `:469` → `listPosts` · `:531` → `writeFollowingFeed` · `:582 :604 :609 :615 :628 :639` → `writeForMeFeed` · `:794` → `listReplies` · `:816 :827 :833 :844 :851 :860 :875 :882 :890 :907 :912 :924 :929 :933 :939 :943 :948` → `updatePost` · `:960 :966 :970` → `deletePost` · `:1004 :1010 :1017 :1022` → `putReaction` · `:1047` → `deleteReaction` · `:1061` → `putBookmark` · `:1070` → `deleteBookmark` · `:1092 :1102 :1107` → `deleteRemoin`
      (`writeFollowingFeed`·`writeForMeFeed`는 handler 메서드가 아니라 `writeFeed`가 부르는 내부 메서드지만 `r`을 받고 라우트당 하나씩 대응하므로 그 이름을 그대로 쓴다. 줄번호는 이관하면서 밀리므로 **message 문자열과 감싼 함수로 식별할 것**.)
  - `backend/internal/httpapi/posts_update_postgres_integration_test.go` — 위 수용 기준 3의 로그 단언. 테스트 파일이라 "프로덕션 파일 6개" 한도 밖.
  - 이 둘 **외에는 열지 않는다.** `social.go`(28) · `admin.go`(20) · `settings.go`(9) · `workflow.go`(9) · `preferences.go`(6) · `analytics.go`(4) · `auth.go`(6) · `outbox.go` · `smtp.go` · `oidc.go`의 나머지 출구는 다음 회차 몫이다.
- 검증 명령:
  - `docker run -d --rm -p 55432:5432 -e POSTGRES_PASSWORD=postgres --name moina-pg postgres:16-alpine` (CI source와 같은 메이저) 뒤
    `cd backend && MOINA_TEST_POSTGRES_DSN='postgres://postgres:postgres@127.0.0.1:55432/postgres?sslmode=disable' go test -race -count=1 -v ./... 2>&1 | tee /tmp/t.log; grep -c -- '--- SKIP' /tmp/t.log`
  - red 확인은 범위를 좁혀서: `... go test -race -count=1 -run TestPostgreSQLUpdatePostSeparatesStorageErrorFromNotEditable -v ./internal/httpapi/`
  - `cd backend && go build ./... && go vet ./... && gofmt -l .`
  - 루트에서 `make check`, `make fmt`(형식 검사만 — 수정하지 않는다)
  - `cd backend && go run honnef.co/go/tools/cmd/staticcheck@2025.1.1 ./...`
  - 프런트·e2e는 응답이 불변이므로 돌리지 않는다(지난 두 회차와 같은 판단).
- 위험과 피할 것:
  - **응답을 바꾸지 말 것.** 프런트가 `readableError`로 서버 message를 그대로 띄우므로 한 글자만 바뀌어도 사용자 눈에 보이는 변경이고 e2e/시각 회귀를 건드린다. 새 status가 없으므로 `api/openapi.yaml`은 손대지 않는다(관례: 오류는 `description`에만, 성공 `responses` 확대 금지).
  - **`storage_error`가 아닌 500은 건드리지 말 것.** `posts.go`에는 `cursor_error` 500이 둘 있다(`:547`·`:656`, "Flow 커서를 만들 수 없습니다" — 직접 확인). 이 둘은 DB 오류가 아니므로 그대로 둔다. `writeError`에서 status가 500이 아닌 호출도 절대 바꾸지 않는다. 그 사이의 `RowsAffected()==0` 404 / `store.IsNotFound` / `store.IsConflict` 409 분기도 그대로 둔다.
  - **`storage_error.go`(헬퍼)를 수정하지 말 것.** 이미 `cause_type`·`pg_code`만 찍고 pg 메시지 전문·`Detail`·`Hint`는 일부러 뺀 것이다(위반한 행의 이메일·토큰 해시가 새는 자리이고 `oidc_discovery_error_test.go`가 그것을 단언한다). 호출부에서 `slog.Default()`를 직접 쓰지 말 것 — `request_id`는 헬퍼 안의 `observability.Logger(r.Context())`가 넣는다.
  - 유일한 실질 위험은 `handler` 인자를 감싼 함수와 **다르게** 적는 것이고 컴파일로 잡히지 않는다. 줄번호를 믿지 말고 함수 블록 단위로 위에서 아래로 순서대로 작업할 것.
  - 보호 경로 회피: `store/migrations`·`.github/workflows/*`·`auth.go`·`oidc.go`·`mcp_oauth.go`·`server.go`를 열 필요가 없다.
  - 과거 교훈: 회차 TIMEOUT 2건이 "인과 증명 후 회귀 확인 과잉 반복"으로 났다. 전체 테스트는 **2~3회**로 끝낼 것. 로그인 5분/동일 IP 5회 제한 때문에 integration을 여러 번 반복하면 인증 단계에서 실패한다.
  - `posts.go`는 1140줄이고 `updatePost` 하나에 17곳이 몰려 있다. 범위가 커 보이면 `updatePost`부터 끝내고(= 수용 기준 3의 red→green 경로) 나머지를 마무리할 것.
- 차선 후보: `settings.go`(9) + `preferences.go`(6) + `analytics.go`(4) = 프로덕션 파일 3개·19곳 이관 — `posts.go` 41곳이 한 세션에 과하다고 판단되면 이쪽. 범위가 작고 위험이 더 낮지만 설정/관리 경로라 운영 가치가 낮고, 이 경로들에는 저장 실패 integration 테스트가 없어 red→green 증명에 새 테스트가 필요하다. 3순위: `e2e/README`의 테스트 실행 순서 설명을 `package.json`의 정본(visual→accessibility→smoke)에 맞춰 정정 (1/1/S).
