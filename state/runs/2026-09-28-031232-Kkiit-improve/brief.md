- 과제: 웹훅 수정 저장의 DB 실패를 404 "웹훅을 찾을 수 없습니다." 가 아니라 500 으로 알리기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `updateMyWebhook`(internal/httpapi/webhooks.go:174)이 `UPDATE webhooks ...` 의 결과를 `if err != nil || tag.RowsAffected() == 0` 로 한 덩어리로 묶어, 저장이 DB 오류로 실패해도 "웹훅을 찾을 수 없습니다."(404 `webhook_not_found`)를 돌려준다. 같은 파일의 `createMyWebhook`(webhooks.go:135)은 같은 테이블의 INSERT 오류를 이미 500 `create_failed` "웹훅을 저장하지 못했습니다." 로 올바르게 보고하므로, 사용자는 방금 만든 웹훅을 목록에서 보고 있으면서 수정만 하면 "없는 웹훅" 이라는 안내를 받는다 — 같은 실패가 경로에 따라 다르게 보고되는 비대칭이다.

- 수용 기준:
  1) 자기 소유의 존재하는 웹훅에 대해 저장이 DB 오류로 실패하는 PUT `/api/v1/me/webhooks/{id}` 가 500 `webhook_save_failed`(또는 기존 `create_failed` 와 짝이 맞는 코드) 와 "웹훅을 저장하지 못했습니다." 계열 메시지를 돌려준다. 404 `webhook_not_found` 가 아니다.
  2) 존재하지 않거나 남의 소유인 id 에 대한 PUT 은 종전대로 404 `webhook_not_found` 를 유지한다(회귀 없음). 정상 입력의 PUT 은 종전대로 200 이고, `rotate_secret: true` 일 때 응답의 `secret`·`warning` 동작도 그대로다.
  3) 테스트는 위 두 갈래가 **실제 라우터 + 실제 PostgreSQL** 을 통과한 HTTP 왕복에서 서로 다른 상태 코드로 갈린다는 것을 증명해야 한다. 수정 전에 먼저 실패시키고(현재는 1) 이 404 로 떨어진다), 수정 후 통과시킬 것. 분기를 두 방향으로 변이시켜(항상 500 / 항상 404) 각각 다시 실패하는지 확인할 것.

- 건드릴 파일 (프로덕션 2개):
  - `internal/httpapi/webhooks.go:174` `updateMyWebhook` — `tag, err := s.DB.Exec(...)` 뒤의 `if err != nil || tag.RowsAffected() == 0` 을 두 갈래로 분리. `err != nil` → `writeError(w, 500, "webhook_save_failed", "웹훅을 저장하지 못했습니다.")`, `tag.RowsAffected() == 0` → 종전 404 `webhook_not_found`. `createMyWebhook` 의 문구("웹훅을 저장하지 못했습니다.")를 그대로 맞출 것.
  - `internal/httpapi/webhooks.go:194` `deleteMyWebhook` — 같은 파일 20줄 아래에 `DELETE FROM webhooks WHERE id=$1 AND owner_id=$2` 에 대해 완전히 동일한 `err != nil || tag.RowsAffected() == 0` → 404 가 있다. 한쪽만 고치면 같은 표의 같은 실패가 또 두 가지로 보고되므로 같이 가르되(`err != nil` → 500 `webhook_delete_failed`), **이 경로는 HTTP 로 DB 오류를 만들 입력이 없다**(파라미터가 uuid 두 개뿐). 수용 기준 3 의 증명 대상은 PUT 이고, DELETE 는 회귀만 확인한다(정상 204, 없는 id 404).
  - `docs/openapi.yaml:258-269` `/me/webhooks/{id}` `put.responses` — 지금 `'200'` 과 `'404'` 만 있다. `'400'`(핸들러가 `invalid_webhook` 으로 실제로 내보내는데 문서에 없음)과 `'500'` 을 `{ $ref: '#/components/responses/Error' }` 로 한 줄씩 추가. 다른 경로는 이번에 손대지 말 것.
  - 테스트: `internal/httpapi/integration_test.go` 에 통합 테스트 1개 추가(아래 "증거 설계" 참고). 필요하면 `internal/httpapi/webhooks_test.go` 에 순수 단위 보강.

- 기존 배선 (실제로 열어 확인함 — 새로 만들지 말고 이것을 그대로 쓸 것):
  - 라우트는 `internal/httpapi/router.go:46` `mux.HandleFunc("PUT /api/v1/me/webhooks/{id}", s.require("webhooks.manage.self", s.updateMyWebhook))`, 생성은 같은 파일 45행, 삭제는 47행.
  - `internal/httpapi/integration_test.go:225-233` 에 이미 본보기가 있다: `seller := newClient(t, server.URL)` → `seller.register(sellerName)` → `seller.do(http.MethodPost, "/api/v1/me/webhooks", map[string]any{"name": ..., "target_url": ..., "events": []string{"*"}}, http.StatusCreated)`. `do(method, path, body, expectedStatus)` 가 응답 맵을 돌려주므로 `hook["id"]` 로 PUT 경로를 만들면 된다. 등록 직후의 일반 계정이 `webhooks.manage.self` 를 갖는다는 것은 이 기존 테스트가 201 을 기대하는 것으로 증명된다.
  - 새 테스트는 기존 이벤트 백본 테스트 안에 끼워 넣지 말고 독립 `TestIntegration...` 함수로 둘 것(그 테스트는 수신 서버·주문 흐름까지 엮여 있어 실패 원인이 흐려진다).
  - `internal/httpapi/webhooks_test.go` 는 지금 순수 단위 4개뿐이고 DB 를 쓰지 않는다 — 여기에 통합을 넣지 말 것.

- 증거 설계 (핵심 — 여기서 막히면 차선으로):
  - `validateWebhookInput`(webhooks.go:56)은 `Name` 을 `strings.TrimSpace` 하고 1~100 **바이트** 길이만 본다. NUL 바이트를 거르지 않는다 — 확인함(소스를 열어 읽음).
  - `webhooks.name` 은 `text` 컬럼이다(`internal/database/migrations/002_marketplace_extensions.sql:331`). 2026-09-27 회차에서 `reports.details`(같은 `text`)에 NUL 을 넣었을 때 PostgreSQL 이 SQLSTATE **22021** 로 INSERT 를 거절한 것이 실측으로 확인되어 있다. 따라서 `{"name":"내 웹훅\u0000","target_url":"https://hooks.example.com/kkiit","events":["OrderPAID"]}` 로 PUT 하면 검증은 통과하고 UPDATE 만 실패한다 — **이번 회차에 실제로 실행해 보지는 않았다(미확인)**. 구현자는 핸들러에 임시 프로브를 넣어 SQLSTATE 를 눈으로 확인한 뒤 프로브를 제거할 것(2026-09-27 회차와 같은 방식).
  - 주의: `100자 이하` 검사는 바이트 기준이므로 한글 이름은 짧게 쓸 것. NUL 도 1바이트를 차지한다.
  - `rotate_secret` 을 켜면 UPDATE 이전에 토큰 생성·암호화가 먼저 일어난다. 증거 테스트는 `rotate_secret` 없이(기본 false) 보낼 것 — 암호화 경로를 섞으면 실패 원인이 흐려진다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 통합(필수): `make test-integration KKIIT_TEST_DSN=postgres://...` — 반드시 **버릴 PostgreSQL 16**(docker). 지난 4회차에서 docker 29.7.2 가 가용했다(이번 회차에는 `docker version` 이 승인 거부로 막혀 **재확인하지 못했다 — 미확인**). `internal/httpapi` 통합은 80~100초 걸리고 `integrationServer` 는 전역 `apiUnderTest` 를 쓰므로 병렬 실행 금지.
  - 전체: `go test ./cmd/... ./internal/...`
  - `go vet ./cmd/... ./internal/...`
  - `gofmt -l cmd internal` (무출력이어야 함)
  - 프런트를 건드리지 않았음을 확인하는 용도로만: `npm --prefix web test` (기존 10건)

- 위험과 피할 것:
  - **`internal/ui/dist` 와 `web/` 을 건드리지 말 것.** 프런트는 서버가 준 `error.message` 를 그대로 띄우므로 서버만 고치면 안내가 바뀐다(지난 두 회차에서 같은 판단으로 통과). 빌드 산출물 커밋 금지는 절대 규칙이다.
  - 보호 경로(`auth.go`, `middleware.go`, `identity.go`, `orders.go`, `finance.go`, `internal/database/migrations`, `.github/workflows`)는 이번 과제와 무관하다. 열지 말 것.
  - `validateWebhookInput` 에 NUL 거부를 **추가하지 말 것.** 그렇게 하면 증거 입력이 400 으로 막혀 고치려던 500/404 분기를 영영 증명할 수 없게 되고, 과제가 "검증 강화" 라는 다른 일로 바뀐다. 입력 검증 강화는 별도 회차 후보다.
  - `isUniqueViolation`(coupons.go:21)은 여기서 쓸 일이 없다. `webhooks` 에는 UNIQUE 제약이 없다(스키마 확인함: PK `id` 뿐). 23505 분기를 새로 만들지 말 것 — 도달 불가능한 코드가 된다.
  - 문구·오류 코드는 새로 발명하지 말고 같은 파일의 기존 것(`create_failed` / "웹훅을 저장하지 못했습니다.")과 짝을 맞출 것.
  - 다른 26곳의 `err != nil || tag.RowsAffected() == 0` 을 같이 고치려 하지 말 것(organizations.go, discovery.go, keys.go, admin_users.go, finance.go, mcp.go …). 한 회차 한 조각이다.
  - 소스 문자열 검사(grep)를 증거로 제출하지 말 것. 실제 HTTP → 실제 DB 왕복만 증거다.

- 차선 후보: **관리자 사용자 수정(`updateAdminUser`, internal/httpapi/admin_users.go:72)의 같은 결함** — `UPDATE users SET status=$2,display_name=$3 ...` 의 모든 오류를 404 `user_not_found` "사용자를 찾을 수 없습니다." 로 묶는다. `users.display_name` 도 `text`(001_initial.sql:22)이고 핸들러는 trim·빈 값 검사만 하므로 같은 NUL 수법이 통한다 — 관리자가 그 사용자의 상세 화면을 보면서 "사용자를 찾을 수 없습니다" 를 받는다. 1순위보다 값은 비슷하지만 계정 상태·세션 폐기에 붙어 있어 위험이 한 단계 높다. 1순위의 NUL → 22021 가정이 깨지면(예: pgx 가 NUL 을 미리 걷어내면) 이쪽도 같은 가정에 기대므로 소용없다 — 그때는 3순위로 `docs/openapi.yaml` 의 POST /reports 에 누락된 `'404'` 추가(reports.go:69 의 `resource_not_found`)로 내려갈 것.
