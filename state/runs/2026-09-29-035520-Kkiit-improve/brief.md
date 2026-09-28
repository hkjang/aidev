- 과제: 포트폴리오 수정·삭제의 DB 저장 실패를 404 "포트폴리오를 찾을 수 없습니다." 가 아니라 500 으로 알리기 (가치 2 / 위험 1 / 작업량 S)
- 왜: `updateMyPortfolio`(internal/httpapi/discovery.go:196)와 `deleteMyPortfolio`(discovery.go:211)가 `if err != nil || tag.RowsAffected() == 0` 로 DB 오류와 "행 없음" 을 한 줄에 묶어 둘 다 404 `portfolio_not_found` 로 돌려준다. 바로 위 `createMyPortfolio`(discovery.go:171-175)는 같은 표의 INSERT 실패를 이미 500 `create_failed` "포트폴리오를 저장하지 못했습니다." 로 보고하므로, 목록에서 포트폴리오를 보고 있는 소유자가 수정만 하면 "없다" 는 안내를 받는 비대칭이 생긴다.

- 수용 기준:
  1) 소유자 본인의 **존재하는** 포트폴리오에 저장 실패를 일으키는 `PUT /api/v1/me/portfolios/{id}` 가 500 `portfolio_save_failed` 를 돌려준다(현재는 404 `portfolio_not_found`).
  2) 소유자의 것이 아닌(또는 없는) id 로 보낸 PUT·DELETE 는 종전대로 404 `portfolio_not_found` 를 유지한다 — 기존 동작 회귀 없음.
  3) 통합 테스트가 **수정 전 코드에서 실제로 실패**하고(`status=404 want=500`), 수정 후 통과하며, 분기를 두 방향으로 변이시키면(항상 500 / 항상 404) 각각 다시 실패한다.

- 건드릴 파일 (프로덕션 2개):
  - `internal/httpapi/discovery.go:updateMyPortfolio` — `if err != nil || tag.RowsAffected() == 0` 을 `if err != nil { writeError(w, 500, "portfolio_save_failed", "포트폴리오를 저장하지 못했습니다.") ; return }` 와 `if tag.RowsAffected() == 0 { 404 portfolio_not_found }` 로 가른다. 문구는 `createMyPortfolio` 의 "포트폴리오를 저장하지 못했습니다." 와 동일하게 맞춘다.
  - `internal/httpapi/deleteMyPortfolio`(discovery.go:211) — 같은 형태로 갈라 500 `portfolio_delete_failed` "포트폴리오를 삭제하지 못했습니다." / 404 를 남긴다. (아래 "위험" 참고: 이 DELETE 는 파라미터가 uuid 뿐이라 HTTP 로 DB 오류를 만들 수 없다. 2026-09-28 회차의 `deleteMyWebhook` 과 정확히 같은 처지였고 그때도 회귀(404 유지)만 확인하고 함께 갈랐다 — 같은 판단을 따른다.)
  - `internal/httpapi/integration_test.go` — 새 테스트 하나 추가(테스트 파일이라 파일 수 예산 밖으로 세지 않음).
  - `docs/openapi.yaml:169-185` — `/me/portfolios/{id}` 의 `put` 응답에 `'400'`(핸들러가 실제로 내는 `invalid_portfolio`)·`'500'`, `delete` 응답에 `'404'`·`'500'` 추가. **확인함**: 현재 put 은 `'200'`·`'404'` 만, delete 는 `'204'` 만 적혀 있다.

- 증거 수법 (확인한 것):
  - `portfolios.title` 은 `text NOT NULL`(002_marketplace_extensions.sql:16), `description` 은 `text NOT NULL DEFAULT ''`(:17) — **확인함**.
  - `portfolioInput.validate()`(discovery.go:97-119)는 `strings.TrimSpace` + 룬 길이(제목 2~160, 설명 ≤4000) + 배열 개수만 본다. NUL 바이트를 거르지 않는다 — **확인함**. 따라서 `"title": "저장 실패\u0000"` 은 400 을 통과해 UPDATE 파라미터로 들어가고, PostgreSQL 이 SQLSTATE 22021 (`invalid byte sequence for encoding "UTF8": 0x00`)로 문장을 거절한다. 이 수법은 2026-09-27(reports)·2026-09-28(webhooks) 두 회차에서 실측된 것이다.
  - 라우트: `PUT /api/v1/me/portfolios/{id}` 와 `DELETE …` 는 `s.require("talents.write", …)`(router.go:56-57) — **확인함**. 통합 테스트에서는 `registerSeller(t, server, "…")` 로 만든 클라이언트를 쓸 것(integration_test.go:749 이 이 헬퍼로 포트폴리오 경로를 이미 쓴다). 일반 `register()` 클라이언트는 `talents.write` 가 없어 403 이 날 수 있다 — **미확인**, seller 헬퍼를 쓰면 이 문제를 피한다.
  - 본보기 테스트: `TestIntegrationWebhookUpdateSeparatesSaveFailureFromNotFound`(integration_test.go:5378-5417). 구조(POST 로 만들고 → NUL 로 PUT 500 → 랜덤 uuid 로 PUT 404)를 그대로 옮기면 된다.

- 검증 명령:
  - `make test-integration KKIIT_TEST_DSN=postgres://…` — **전용 폐기 DB 필수**, `internal/httpapi` 만 80~100초.
  - `go test ./cmd/... ./internal/...` (같은 DSN 으로 통합 포함 전체)
  - `go vet ./cmd/... ./internal/...` · `gofmt -l cmd internal`(무출력)
  - DSN 없이 통과하는 것은 통합 검증이 아니다 — 통합은 SKIP 된다. 반드시 DSN 을 걸고 새 테스트가 실제로 돌았는지 로그로 확인할 것.

- 위험과 피할 것:
  - `internal/ui/dist` 를 건드리지 말 것. 프런트는 `error.message` 를 그대로 띄우므로 서버만 고쳐도 안내가 바뀐다. `web/` 도 불필요.
  - `discovery.go:24`(다른 핸들러)와 `discovery.go` 의 공개 조회 경로(`listSellerPortfolios`)는 이번 과제 밖 — 건드리지 말 것.
  - `deleteMyPortfolio` 는 HTTP 로 DB 오류를 만들 수 없으므로 **500 경로를 증명했다고 쓰지 말 것**. 수용 기준 1 은 PUT 으로만 증명되고, DELETE 는 404 회귀 확인까지가 정직한 범위다.
  - 테스트 병렬화 금지(`integrationServer` 가 전역 `apiUnderTest` 를 쓴다).
  - grep 결과를 증거로 제출하지 말 것 — 실제 HTTP→DB 왕복 출력으로 증명할 것.
  - docker 가용성은 이번 정찰에서 **확인하지 못했다**(명령 승인 거부). 지난 4회차에서는 29.7.2 였다. 없으면 아래 차선으로 전환.

- 차선 후보:
  1) **관리자 사용자 수정(`updateAdminUser`, admin_users.go:72)의 같은 결함** — `display_name` 이 `text`(001:22)라 같은 NUL 수법이 통한다. 다만 계정 상태·세션 폐기에 붙어 있어 위험 한 단계 위(가치 3 / 위험 2 / S).
  2) **docker 가 없을 때**: `docs/openapi.yaml` 의 `/me/portfolios/{id}` 응답 코드 누락(위에 적은 put `'400'`·delete `'404'`)을 라우터·핸들러 코드와 대조해 채우는 문서 단독 과제 — DB 불필요, 단위 테스트(`openapi_test.go`)로 검증(가치 1 / 위험 1 / S).
