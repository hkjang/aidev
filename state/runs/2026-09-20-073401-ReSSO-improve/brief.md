# 과제서 2026-09-20 — ReSSO

- 과제: UserInfo POST가 form-encoded `access_token`(RFC 6750 §2.2)을 받게 하고, 헤더와 본문에 둘 다 오면 400 `invalid_request`로 거절 (가치 2 / 위험 1 / 작업량 S)

- 왜: `POST /realms/{realm}/protocol/openid-connect/userinfo`는 라우트로 등록되어 있지만(`internal/httpserver/server.go:114-115`) 토큰은 `bearerToken(r)`(`oidc.go:1162`)이 `Authorization` 헤더에서만 읽으므로, RFC 6750 §2.2대로 본문에 `access_token=…`을 실어 보내는 RP·SDK는 유효한 토큰으로도 401 `invalid_token`을 받는다 — "만료됐다"는 답이라 RP는 토큰을 의심하고 refresh를 반복한다. 고치면 POST 라우트가 실제로 GET과 다른 것을 받게 되고, 두 방식이 동시에 오는 것은 §2 규정("하나만 써야 한다")대로 400으로 구분해 답한다.

- 수용 기준:
  1) `POST …/userinfo`에 `Content-Type: application/x-www-form-urlencoded` 본문 `access_token=<유효 토큰>`만 보내면 200이고 응답 claims가 같은 토큰을 헤더로 보낸 GET과 같다(`sub` 비교).
  2) `Authorization: Bearer …`와 본문 `access_token`이 **둘 다** 있으면(값이 같아도) 400, 본문 `{"error":"invalid_request"}`, `WWW-Authenticate: Bearer error="invalid_request"`. 본문의 토큰이 잘못됐으면(예: `access_token=garbage`) 기존과 같은 401 `invalid_token`. GET에서는 본문/쿼리를 절대 읽지 않는다 — `GET …/userinfo?access_token=…`은 여전히 401 (§2.3 쿼리 방식은 일부러 지원하지 않음: 접근 로그에 토큰이 남는다).
  3) 새 연동 테스트 하나가 위 세 경우(본문만 → 200 + claims 일치, 둘 다 → 400 invalid_request, GET 쿼리 → 401)를 한 번에 고정하고, **수정 전 핸들러에서 첫 단언(본문만 → 200)이 실제로 401로 실패함**을 확인해 회차 노트에 적는다.

- 건드릴 파일:
  - `internal/httpserver/oidc.go:userInfo` (약 705-812행) — `raw := bearerToken(r)` 자리에서 POST일 때 `r.ParseForm()` 후 `r.PostForm.Get("access_token")`(**`r.FormValue`는 쿼리도 읽으므로 쓰지 말 것**)을 함께 보고, 헤더·본문 둘 다 있으면 `w.Header().Set("WWW-Authenticate", `+"`Bearer error=\"invalid_request\"`"+`)` + `writeOAuthError(w, http.StatusBadRequest, "invalid_request", …)`로 끝낸다. `writeBearerError`(1209행)는 401 고정이므로 새 helper를 두거나 두 줄 인라인. 이 파일의 다른 핸들러가 `r.ParseForm()`을 쓰는 방식(422·835·958행)을 따를 것. `bearerToken` 자체는 introspection/revocation 등 다른 자리에서도 쓰일 수 있으니(미확인) 시그니처를 바꾸지 말고 userInfo 안에서만 합친다.
  - `internal/httpserver/integration_test.go` — `TestIntegrationUserInfoRefusesWhenRolesCannotBeRead`(9106행) 근처에 새 테스트. 토큰 얻는 방식·`userInfoAccepts` 헬퍼(1248-1262행)를 참고해 같은 흐름(로그인→코드→토큰)으로 access token을 얻을 것.
  - `docs/compatibility.md:20` — `| UserInfo / JWKS | 구현 |` 행에 "POST는 본문 `access_token`(RFC 6750 §2.2)도 받으며 헤더와 본문이 함께 오면 `invalid_request`, 쿼리 파라미터(§2.3)는 받지 않는다" 한 문장. PDF·캡처 재생성 없음(문장만).
  - 선택: `internal/httpserver/oidc.go:29` 부근 주석에 왜 쿼리 방식을 안 받는지 한 줄 — 있으면 좋고 없어도 됨.

- 검증 명령:
  - `eval "$(scripts/test-services.sh)"` 로 컨테이너 변수 네 개를 세운 뒤 `go test -race ./internal/httpserver/ -run 'TestIntegration.*UserInfo' -v` (SKIP이 찍히면 변수가 안 선 것 — ok로 읽지 말 것)
  - `make lint` (golangci-lint는 `$(go env GOPATH)/bin`; gofmt 정렬을 첫 실행에서 잡을 수 있음)
  - `make test` 전체(httpserver ~2분, store ~1.5분) — 끝나면 `git checkout webui/dist/index.html`
  - 수정 전 실패 확인: `git stash`는 금지(공유 스택) — 새 테스트만 먼저 추가하고 핸들러 수정 전에 위 `go test -run` 한 번 돌려 401 실패를 본 뒤 핸들러를 고친다.

- 위험과 피할 것:
  - `auth.go`·`middleware.go`(CSRF 예외 목록)·`migrations`·`.github/workflows`는 건드리지 않는다 — OIDC 프로토콜 경로는 이미 세션 미들웨어 밖이라 CSRF 예외를 더할 필요가 없다(라우트 워커 테스트 `integration_test.go:795-812`가 POST userinfo를 이미 목록에 넣고 있음; 그 테스트가 빈 본문 POST에 무엇을 기대하는지 확인하고 깨지지 않게 할 것 — 빈 본문·헤더 없음은 기존대로 401이어야 한다).
  - `r.FormValue`·`r.Form`은 쿼리 문자열을 합쳐 읽는다 — 반드시 `r.PostForm`. GET에서 ParseForm을 부르지 말 것.
  - `writeUserInfoUnavailable`(500) 분기 순서는 그대로 — realm 조회·revocation 상태의 "판단 불가" 구분은 지난 회차가 의도적으로 세운 것.
  - 토큰 없음(헤더도 본문도 없음)의 답을 §3.1대로 error 코드 없는 401로 바꾸는 것은 이번 범위 밖(아이디어 파일에 따로 둠). 운영자 규칙: 실제 동작이 바뀌지 않는 수정은 넣지 말 것 — 이 과제는 본문-only 요청의 답이 401→200으로 바뀌므로 해당 없음.
  - 감사 로그·접근 로그에 토큰 원문을 남기지 말 것(운영자 지시: 식별자만).

- 차선 후보: 인가 접근 로그에 `client_id` 남기기 — `middleware.go`의 요청 로그에 인가 Endpoint(`…/protocol/openid-connect/auth`)일 때만 `r.URL.Query().Get("client_id")`를 더한다(가치 2 / 위험 1 / S). 값이 사용자 입력이므로 길이 제한(예 128자)을 두고, 로그 필드 이름은 silent 거절 로그와 같은 `client_id`로 맞춰 grep이 한 줄로 되게 할 것. 연동 테스트는 로그 핸들러를 잡아 필드 유무를 단언.
