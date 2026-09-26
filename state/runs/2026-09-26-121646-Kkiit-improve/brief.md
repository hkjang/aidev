- 과제: 쿠폰 코드 중복을 수정 저장에서 404 "쿠폰을 찾을 수 없습니다" 가 아니라 409 중복으로 알리기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/httpapi/coupons.go:306` 의 `updateCoupon` 은 UPDATE 의 모든 오류를 `err != nil || tag.RowsAffected() == 0` 한 덩어리로 묶어 404 `coupon_not_found` 로 돌려주므로, 관리자가 쿠폰 코드를 이미 존재하는 다른 쿠폰의 코드로 바꾸면 "쿠폰을 찾을 수 없습니다" 라는 잘못된 안내를 받고 원인을 알 수 없다(같은 파일 `createCoupon:285` 는 반대로 모든 INSERT 오류를 409 `coupon_exists` 로 묶어 DB 장애도 중복으로 보고한다). 두 저장 경로에서 고유 제약 위반(SQLSTATE 23505)만 409 로, 나머지는 각자 맞는 코드로 구분하면 관리자가 무엇을 고쳐야 하는지 알 수 있고 장애가 중복으로 위장되지 않는다.
- 수용 기준:
  1) 쿠폰 두 개를 만든 뒤 두 번째 쿠폰을 첫 번째와 같은 `code` 로 `PUT /api/v1/admin/coupons/{id}` 하면 409 와 중복을 뜻하는 `error.code`(예: `coupon_exists`)가 오고, 존재하지 않는 id 에 대한 PUT 은 지금처럼 404 `coupon_not_found` 를 유지한다.
  2) 거절된 요청이 DB 를 바꾸지 않는다 — 두 번째 쿠폰의 `code` 는 `GET /api/v1/admin/coupons` 에서 원래 값 그대로다.
  3) 테스트는 실제 라우터·실제 PostgreSQL(통합) 로 위 두 경우를 증명하고, 구현을 되돌리면 1) 이 404 로 실패하는 것을 먼저 확인한다(손으로 만든 대역·소스 문자열 검사 금지).
- 건드릴 파일:
  - `internal/httpapi/coupons.go:updateCoupon` — UPDATE 오류에서 `*pgconn.PgError` 의 `Code == "23505"` 를 구분해 409 중복, 그 밖의 오류는 500(또는 기존 문구 유지), `RowsAffected()==0` 만 404.
  - `internal/httpapi/coupons.go:createCoupon` — INSERT 오류를 같은 방식으로 나눠 23505 만 409 `coupon_exists`, 나머지는 500. (미확인: 저장소에 `pgconn` 오류 코드를 보는 선례가 없다 — `grep pgconn|SQLState|23505 internal/**/*.go` 가 비었다. `github.com/jackc/pgx/v5/pgconn` 은 pgx/v5 의 일부라 go.mod 변경 없이 import 가능한지 구현자가 먼저 확인할 것.)
  - `internal/httpapi/integration_test.go` — 기존 `TestIntegrationCouponDiscountsBuyerWithoutTouchingSellerPayout`(851행~) 옆에 새 통합 테스트를 추가. 확인된 헬퍼: `server, pool := integrationServer(t)`, `operator := operatorClient(t, server, pool, "couponop")`, `strings.ToUpper(uniqueName("SAVE"))` 로 코드 생성, `operator.do(method, path, body, expectedStatus)` 가 기대 상태 코드를 직접 단언하므로 409/404 확인에 그대로 쓸 수 있다. 이름 접미사는 `uniqueName` 을 그대로 쓸 것(v0.4.6 에서 폭 고정됨).

- 확인한 근거:
  - `internal/database/migrations/002_marketplace_extensions.sql:100` 의 `code text NOT NULL UNIQUE` — 중복 코드 UPDATE 는 반드시 고유 제약 위반이 되고, `updateCoupon` 의 `err != nil || tag.RowsAffected() == 0` 가 그것을 404 로 덮는다(코드 읽기로 확인, 실제 요청으로는 미확인 — 구현자가 먼저 재현할 것).
  - `coupon_exists`·`coupon_not_found` 문자열은 저장소 전체에서 `internal/httpapi/coupons.go` 에만 있다(dist 제외). 프런트·문서·스크립트가 이 코드값에 의존하지 않으므로 상태 코드 구분을 넣어도 다른 곳이 깨지지 않는다.
  - 기존 쿠폰 단위 테스트는 `internal/httpapi/coupons_test.go` 의 `computeDiscount`·`validate` 뿐이라 이 경로를 덮지 않는다.
  - docker 가용성은 이번 회차에서 미확인(권한 때문에 실행하지 않음). 2026-09-23·24 회차는 임시 `postgres:16-alpine` 으로 통합을 실제 실행했다. 없으면 통합이 SKIP 되므로 수용 기준 1·2 를 증명할 수 없다 — 그때는 차선 후보로 넘어갈 것.
- 검증 명령:
  - `go test ./internal/httpapi/ -run TestCoupon` (단위)
  - 버릴 PostgreSQL 16 을 띄우고 `make test-integration KKIIT_TEST_DSN=postgres://...` 또는 `KKIIT_TEST_DSN=... go test ./internal/httpapi/ -run TestIntegration -timeout 300s` (httpapi 통합 전체 약 85~95초, 전용 폐기 DB 필수)
  - 마지막에 `go vet ./cmd/... ./internal/...`, `gofmt -l .`
- 위험과 피할 것:
  - `internal/httpapi/auth.go`·`middleware.go`·`identity.go`·`orders.go`·`finance.go`·`database/migrations`·`.github/workflows` 는 건드리지 말 것. 이 과제는 `coupons.go` 와 통합 테스트만으로 끝난다.
  - 프런트(`web/src/pages/CouponsAdmin.tsx`)와 `internal/ui/dist` 는 건드리지 말 것 — 빌드 산출물 커밋 금지이고 이 수정은 서버 응답만 바꾼다. 관리자 화면 문구를 고치고 싶어지면 이번 회차 범위에서 제외하고 아이디어로 남길 것.
  - `couponInput.validate()`(coupons.go:196~) 는 이미 코드 3~40자·공백 금지·대문자화를 하고 있으니 형식 검증을 새로 추가하지 말 것(기존 운영 데이터를 깨뜨린다).
  - `resolveCoupon` 은 `upper(code)=$1` 로 조회한다. 저장은 항상 대문자이므로 이번 변경에서 조회 쪽을 손대지 말 것 — 같은 값을 읽는 두 경로를 한쪽만 바꾸면 안 된다.
  - DSN 없이 `go test` 가 통과하는 것은 통합 검증이 아니다(통합은 SKIP 된다). 통합 테스트는 전역 `apiUnderTest` 때문에 병렬 실행 금지.
- 차선 후보: README 환경변수 계약을 필수 4개(`POSTGRES_DSN`/`BOOTSTRAP_ADMIN`/`BOOTSTRAP_ADMIN_PASSWORD`/`ENCRYPTION_KEY`) + 선택 `SHUTDOWN_DRAIN_SECONDS`(기본 5, 0~120) 로 정리하기 — `internal/config/config.go:33~40` 의 `Load` 와 README 48행 표가 어긋난다(확인함: README 표는 필수만 나열하고 84행 본문이 `SHUTDOWN_DRAIN_SECONDS` 를 설명한다). DB·docker 없이 끝나지만 가치 2.
