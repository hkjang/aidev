- 과제: `r.ContentLength > 0` 게이트가 chunked 요청의 JSON 본문을 통째로 버리는 것을 고친다 (가치 3 / 위험 2 / 작업량 M)

- 왜: `followTopic`(social.go:271)·`reviewApproval`(workflow.go:110)·`adminResolveReportAlias`(admin.go:367) 세 handler가 "본문은 선택"을 `if r.ContentLength > 0 && !decodeJSON(...)`로 구현하는데, Go net/http는 chunked transfer-encoding(그리고 content-length 없는 HTTP/2) 요청에 `ContentLength == -1`을 넣으므로 이 조건이 거짓이 되어 **클라이언트가 실제로 보낸 본문이 조용히 버려진다**. 그 결과 `POST /api/topics/{slug}/follow`는 `{"weight":80}`을 보내도 기본값 50으로 저장하며 200을 돌려주고(사실과 다른 데이터가 남음), 승인 반려와 신고 처리 alias는 사유를 보냈는데도 각각 400 `comment_required`·400 `invalid_resolution`로 거절한다. 같은 파일 `posts.go:1032`는 이미 올바른 `r.ContentLength != 0` 형태를 쓰고 있어 저장소 안에서도 계약이 갈려 있다.

- 수용 기준:
  1) chunked(= content-length 헤더 없는) `POST /api/topics/{slug}/follow`에 `{"weight":80}`을 보내면 응답이 `{"following":true,"weight":80}`이고 `user_topic_follows.weight`가 80으로 저장된다(수정 전에는 50).
  2) 본문을 아예 보내지 않는 기존 호출(content-length 0, 본문 없음)은 지금과 똑같이 동작한다 — followTopic은 weight 50으로 200, 승인 반려·신고 alias는 지금 그대로의 400.
  3) content-length가 **있는** 요청의 동작은 한 글자도 바뀌지 않는다: 잘못된 JSON은 여전히 400 `invalid_json`, 알 수 없는 필드도 400 `invalid_json`, JSON 두 개는 400 "JSON 값은 하나만 허용됩니다", 본문 한도 초과도 그대로.
  4) 테스트는 수정 전 코드에서 chunked 케이스만 실패하고(200 weight 50), content-length 케이스는 전부 통과함을 red 단계로 눈으로 확인한 뒤 수정 후 전부 통과한다.

- 건드릴 파일 (프로덕션 4개):
  - `backend/internal/httpapi/server.go` — `decodeJSON`(706줄) 바로 아래에 `decodeOptionalJSON(w, r, destination) bool` 신설. 본문이 선택인 계약 전용. 구현 지침:
    - `if r.ContentLength == 0 || r.Body == nil || r.Body == http.NoBody { return true }`
    - 그 밖에는 `decodeJSON`과 **완전히 같은** 순서(`http.MaxBytesReader(w, r.Body, maxBodyBytes)` → `DisallowUnknownFields` → `Decode` → 두 번째 `Decode`가 `io.EOF`인지)를 쓴다.
    - **단 하나의 차이**: 첫 `Decode`가 `errors.Is(err, io.EOF)`이고 `r.ContentLength < 0`일 때만 "본문 없음"으로 보아 `true`를 돌려준다. `ContentLength < 0` 조건을 반드시 함께 걸 것 — 이것이 수용 기준 3(길이가 알려진 요청의 무회귀)을 보장하는 자리다.
    - `errors`·`io`는 server.go가 이미 import한다(714줄 `errors.Is(err, io.EOF)`).
  - `backend/internal/httpapi/social.go:271` `followTopic` — `if r.ContentLength > 0 && !decodeJSON(w, r, &input)` → `if !decodeOptionalJSON(w, r, &input)`.
  - `backend/internal/httpapi/workflow.go:110` `reviewApproval` — 같은 치환.
  - `backend/internal/httpapi/admin.go:367` `adminResolveReportAlias` — 같은 치환.
  - 테스트: `backend/internal/httpapi/topics_postgres_integration_test.go`에 케이스 추가(또는 같은 패키지 새 파일 `optional_body_postgres_integration_test.go`). 기존 파일의 관례를 그대로 따를 것 — `os.Getenv("MOINA_TEST_POSTGRES_DSN")` 없으면 `t.Skip`, `store.Open` → `New(repository, …)` 실제 배선, `httptest`, `time.Now().UnixNano()` 접미사로 고유 id, `t.Cleanup`에서 삭제.
    - **chunked 요청은 손으로 필드를 세팅하지 말고 실제 전선으로 보낼 것**: `httptest.NewServer(server.Handler())`를 띄우고 `http.NewRequest(http.MethodPost, ts.URL+path, io.NopCloser(strings.NewReader(body)))` 로 **길이를 알 수 없는 Reader**(`io.NopCloser`로 감싸면 `*strings.Reader` 최적화가 풀린다)를 주면 Go 클라이언트가 실제로 `Transfer-Encoding: chunked`로 보내고 서버 쪽 `r.ContentLength`가 `-1`이 된다. 같은 테스트에서 content-length가 있는 대조군(`strings.NewReader` 그대로)도 함께 보내 두 경로가 같은 입력을 같은 값으로 읽는지 확인할 것 — 이것이 이 과제의 핵심 증거다. `req.ContentLength = -1`을 직접 대입하는 방식은 배선을 우회하므로 쓰지 말 것.
    - 세션 쿠키·CSRF 절차는 `moim_join_postgres_integration_test.go:74~120`을 **그대로 복사**할 것: `sessions` 행을 `secrets.HashToken(token)`·`secrets.HashToken(csrf)`로 직접 넣고, 요청에 `request.Header.Set("X-CSRF-Token", csrf)` + `request.AddCookie(&http.Cookie{Name: SessionCookie, Value: token})`. 서버는 `New(repository, secrets, "v0.1.37-test")` → `server.Handler()`. 새 헬퍼를 만들지 말 것.
    - `httptest.NewServer`를 쓰면 쿠키 `Secure` 속성·origin 차이로 인증이 막힐 수 있다. 막히면 같은 테스트 파일 안에서 먼저 content-length 있는 정상 요청이 200을 받는지로 배선을 확인한 뒤 chunked 케이스를 붙일 것(실패 원인이 갈리게).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - throwaway DB: `docker run --rm -d -e POSTGRES_PASSWORD=pw -p 55432:5432 postgres:17-alpine` → `export MOINA_TEST_POSTGRES_DSN='postgres://postgres:pw@127.0.0.1:55432/postgres?sslmode=disable'`
  - red 단계: `cd backend && go test -race -count=1 ./internal/httpapi/ -run 'TestPostgreSQL.*(Topic|Optional|Chunked)' -v` — 수정 전 chunked 케이스만 FAIL 하는지 확인
  - 전체: `cd backend && go test -race -count=1 ./... 2>&1 | tee /tmp/t.log` 후 `--- SKIP` 0줄인지 확인(DSN 없으면 integration이 조용히 skip 되어 증거가 되지 않는다)
  - `cd backend && go vet ./...`
  - 루트에서 `make fmt`(변경 없음) · `make check`(OpenAPI route 120개 유지)
  - staticcheck 2025.1.1이 있으면 `make lint`의 Go 구간. 프런트는 무변경이므로 vitest·e2e 생략.

- 위험과 피할 것:
  - `decodeJSON`(server.go:706) **자체를 고치지 말 것.** 본문이 필수인 handler 수십 곳이 이것을 쓰고 있고, EOF 허용을 거기에 넣으면 빈 본문이 400 대신 zero-value로 통과한다. 반드시 새 함수를 추가하고 세 call site만 바꾼다.
  - `ContentLength < 0` 조건을 빼먹으면 content-length가 있는 빈/공백 본문까지 통과하게 되어 수용 기준 3이 깨진다. 여기가 이 과제가 깨질 단 하나의 자리다.
  - `admin.go:284` `adminDeletePost`의 `r.ContentLength = 0`은 의도적으로 본문을 버리는 다른 코드다. 건드리지 말 것.
  - OpenAPI `responses` 목록을 늘리지 말 것(이 저장소 관례: 오류는 `description` 문장). 이번 변경은 문서화된 계약을 **지키게** 만드는 수정이라 사실 openapi 변경 없이도 맞다. `make check`의 route 120개가 유지되어야 한다.
  - 보호 경로 회피: auth.go·oidc.go·mcp_oauth.go·store/migrations·.github/workflows 는 건드리지 않는다. `workflow.go`·`admin.go` 변경은 각각 한 줄 치환에 그칠 것(승인 정책·권한 판정 로직에 손대지 말 것).
  - `followTopic`은 최근 회차(2026-09-25, verify-failed)가 `social.go:282`의 404/500 분리를 시도했던 자리다. **282줄은 이번 과제에서 건드리지 말 것** — 271줄의 본문 디코드만 바꾼다. 두 변경을 섞으면 실패 원인이 갈리지 않는다.
  - 샌드박스에 따라 docker·go 실행이 권한 거부될 수 있다. 그때는 실행하지 못한 명령을 "미실행"으로 정직하게 적고, red 단계를 돌리지 못했다면 결함 재현을 주장하지 말 것.

- 차선 후보: `resolveReport`(admin.go:382)의 `if err != nil || tag.RowsAffected() == 0` → 저장 실패는 500 `storage_error`("신고를 처리할 수 없습니다"), 0행만 404 `not_found` 기존 문구 유지. `joinMoim`·`leaveMoim`·`updatePost`에서 3회 릴리즈로 굳은 관례와 같고, `reports` 테이블에 테스트 전용 `BEFORE UPDATE` 트리거로 sentinel 신고 id만 거부하면 저장 오류 분기를 실제 pgx로 재현할 수 있다(`joinMoim` 회차에서 검증된 기법). 프로덕션 파일 1개(admin.go) + 테스트 1개.
