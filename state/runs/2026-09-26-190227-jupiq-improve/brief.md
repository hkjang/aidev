- 과제: 프로필·OIDC가 쓰는 users.email을 저장 시점에 다듬고, 보낼 수 없는 주소는 프로필 API가 400으로 거부한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/store/users.go:225 UpdateProfile`이 JSON 입력을 그대로 `UPDATE users SET display_name=$2,email=$3` 하고, 같은 파일의 `UpsertOIDCUser`(185행 INSERT·189행 UPDATE)도 OIDC `email` claim을 그대로 넣는다. 그래서 `"   "`·`"\t"`·`"nonsense"`(@ 없음) 같은 값이 users.email에 남고, 메일 수신자 해석(`internal/mail/config.go:186 validAddress`)이 그것을 버려 그 사용자는 알림을 못 받는데 어디에도 흔적이 남지 않는다.
- 왜(이어서): v1.8.2는 소비 측(`ExpiringAPIKeys`의 btrim)에서만 막았고, `internal/store/mail.go:170-175` 주석이 직접 "SQL로 주소 모양까지 흉내 내면 두 파서가 갈리므로, 그 계열은 UpdateProfile·UpsertOIDCUser의 입력 검증으로 막아야 한다"고 적어 둔다. 이번 과제가 그 남은 절반이다.
- 수용 기준:
  1) `PATCH /api/v1/auth/me`에 `{"email":"  a@corp.internal  ","display_name":" 홍길동 "}`를 보내면 200이고, 저장·응답 값이 `a@corp.internal`·`홍길동`(앞뒤 공백 없음)이다.
  2) `{"email":"   "}`(탭·줄바꿈·NBSP 포함 공백류)는 200이면서 빈 문자열로 저장된다 — "주소를 지운다"는 뜻이고, `users.email`은 `NOT NULL DEFAULT ''`(migrations/001_initial.sql:10)라 빈 값은 계속 허용해야 한다.
  3) `{"email":"nonsense"}`처럼 `mail.ValidAddress`가 거부하는 비어 있지 않은 값은 400 `invalid_email`(한국어 메시지)이고 `users.email`이 바뀌지 않는다. 빈 값은 절대 400이 아니다.
  4) OIDC 로그인의 `email` claim이 `"\t bob@corp.internal \r\n"`이어도 `UpsertOIDCUser`의 **생성·갱신 두 갈래 모두** 다듬은 값을 저장한다. 다만 OIDC 쪽은 모양이 이상해도 **거부하지 않는다**(로그인을 끊지 않는다) — 다듬기만 한다.
  5) 테스트가 수정 전에는 실패하고 수정 후 통과함을 한 실행에서 보인다: (a) 실제 PostgreSQL + 실제 `*store.Store`로 `UpdateProfile`·`UpsertOIDCUser`(두 갈래) 저장값 단언, (b) `internal/api`에서 프로덕션 핸들러를 거친 400 `invalid_email`과 빈 값 200 단언. 손으로 만든 대역 금지.
- 건드릴 파일:
  - `internal/store/users.go:UpdateProfile` — `strings.TrimSpace`한 displayName·email로 UPDATE. (strings는 이 파일에 이미 import되어 있는지 확인하고 없으면 추가)
  - `internal/store/users.go:UpsertOIDCUser` — 185행 INSERT와 189행 UPDATE가 **같은** 다듬은 값을 쓰도록 함수 진입부에서 한 번만 정규화(두 갈래가 갈리면 안 된다). username은 건드리지 말 것(로그인 식별자·`lower(username)` 비교에 얽혀 있다).
  - `internal/api/auth_handlers.go:meUpdate(96-113행)` — `decodeJSON` 뒤, `UpdateProfile` 호출 전에 `strings.TrimSpace` + `email != "" && !mail.ValidAddress(email)`이면 `apiError(..., http.StatusBadRequest, "invalid_email", "…")`. 같은 파일 바로 아래 `mailTest`의 `invalid_recipient` 처리가 선례다. 감사 로그(111행)는 저장된 `user.Email`을 쓰므로 그대로 둔다.
  - `openapi/openapi.yaml:74-85`(`/auth/me` patch) — `'400': { $ref: '#/components/responses/Error' }` 한 줄 추가. 바로 위 비밀번호 변경(67행)이 같은 모양이다.
  - `internal/store/mail_integration_test.go:180` 주석 — "UpdateProfile은 입력을 다듬지 않으므로 공백류만 든 주소가 실제로 저장된다"가 이 변경으로 거짓이 된다. 이 테스트는 `createTestUser`(직접 INSERT)를 쓰므로 **테스트 자체는 계속 통과**하지만 주석을 사실에 맞게 고칠 것.
  - 새 테스트 파일(예: `internal/store/profile_normalize_integration_test.go`, `internal/api` 내 기존 핸들러 테스트에 추가).
- 검증 명령(이 저장소에서 실제로 도는 것):
  - `gofmt -l .` (무출력) / `go vet ./...` / `go build ./...`
  - `go test -count=1 ./internal/api ./internal/store` — 오늘 기준선 통과 확인함(api 0.046s, store 0.006s ok).
  - `go test -count=1 -run 'OpenAPI|UndocumentedRoute' ./internal/api` — openapi.yaml을 고쳤으면 반드시.
  - DB: `docker run -d --rm -e POSTGRES_PASSWORD=pw -p 55432:5432 postgres:16-alpine` → `export JUPIQ_INTEGRATION_TEST_DSN='postgres://postgres:pw@127.0.0.1:55432/postgres?sslmode=disable'` → `make test-integration`. 끝나면 컨테이너 제거. `-v`로 SKIP 0건 확인(DSN 없이는 통합이 전부 skip되어 증거가 되지 않는다).
  - `go test -count=1 ./...` / `./scripts/check-version.sh`(1.8.2)
- 위험과 피할 것:
  - `internal/auth/oidc.go`는 건드리지 말 것(보호 경로). 311행이 claim을 그대로 넘기지만 정규화는 `store/users.go` 한 곳에서 해야 두 갈래(생성·갱신)가 같은 값을 쓴다.
  - OIDC 경로에서 주소를 **거부**하면 로그인이 끊긴다. 거부는 프로필 API에서만.
  - 빈 email을 400으로 막지 말 것 — 주소를 지울 수 없게 되고 `Seed` 계정(email='')의 프로필 저장이 깨진다.
  - `decodeJSON`에 `DisallowUnknownFields`를 넣지 말 것 — 프런트 `web/src/pages/PersonalPage.tsx:37`이 disabled `department` 필드까지 실어 보낸다.
  - `mail.ValidAddress`를 흉내 낸 새 검사를 만들지 말 것(파서가 갈린다). `internal/store`는 이미 `internal/mail`을 import한다(store/mail.go).
  - 프런트는 `<Input type="email" />`이라 브라우저 검증이 있을 수 있으나 API 계약이 정본이다. 프런트 변경은 이번 범위 밖.
- 차선 후보: `ListMailDeliveries`의 limit 상한 처리 — `internal/store/mail.go:118`이 `limit > 200`일 때 상한 200이 아니라 기본값 50으로 떨어뜨려 `store.go:195 pageBounds`의 "상한으로 자른다" 관례와 어긋난다(가치 2 / 위험 1 / 작업량 S).
