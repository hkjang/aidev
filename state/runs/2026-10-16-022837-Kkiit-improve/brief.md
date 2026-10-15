- 과제: 쿠폰 미리보기가 선택한 추가 옵션 금액을 빼먹어 주문 화면이 실제 결제 금액보다 싼 값을 보여준다 (가치 4 / 위험 2 / 작업량 M)
- 왜: `previewCoupon`(internal/httpapi/coupons.go:144)은 주석에 "주문이 적용할 규칙과 똑같이" 라고 적어 두고도 `orderAmount`(coupons.go:180)로 **기본가 또는 패키지 가격만** 집계하는데, `createOrder`(internal/httpapi/orders.go:73-124)는 `talent_options` 에서 읽은 옵션 가격을 `price += optionPrice` 로 더한 뒤 그 합계로 `resolveCoupon` 을 호출한다. 그래서 옵션을 고른 뒤 쿠폰을 적용하면 화면의 "결제 금액"(web/src/pages/TalentPage.tsx:118 의 `coupon.payable_amount`)에는 옵션 금액이 아예 빠지고 할인액도 옵션을 제외한 금액 기준으로 계산돼, 버튼을 누르는 순간 생성되는 주문 금액과 다른 값을 구매자에게 약속한다. 같은 간극 때문에 `min_order_amount` 조건을 옵션 포함으로는 충족하는 쿠폰이 미리보기에서는 409 로 거절돼 구매자가 쓸 수 있는 쿠폰을 못 쓴다.
- 수용 기준:
  1) `POST /api/v1/coupons/preview` 가 선택한 추가 옵션을 받아(예: `"options":[{"id":"<uuid>"}]`) 응답의 `amount`·`discount_amount`·`payable_amount` 가 같은 옵션으로 생성한 주문의 `amount`·`discount_amount`·`payable_amount` 와 **정확히 일치**한다(퍼센트 쿠폰으로 검증 — 기본가만 보던 종전 값과 달라지는 입력을 쓸 것).
  2) `options` 를 보내지 않은 기존 요청의 응답은 지금과 동일하다(기존 `TestIntegrationCouponPercentLargeAmount`·`TestIntegrationCouponDiscountsBuyerWithoutTouchingSellerPayout`·`TestIntegrationCouponPreviewIsThrottled` 가 수정 없이 통과).
  3) `min_order_amount` 가 기본가보다는 크고 기본가+옵션 합계보다는 작거나 같은 쿠폰이, 옵션을 포함한 미리보기에서 200 으로 적용되고(종전에는 409 `coupon_not_applicable`) 같은 구성의 주문도 201 로 통과한다.
  4) 미리보기가 상품에 없거나 `active=false` 인 옵션 id 를 받으면 주문과 같은 기준으로 400 `invalid_option` 을 돌려주고, 옵션 합계가 int64 를 넘기면 400 `order_amount_too_large` 를 돌려준다(주문 쪽 문구·코드와 동일하게).
  5) 테스트는 "옵션을 고른 미리보기와 실제 주문의 세 금액이 같다" 를 실제 라우터·실제 PostgreSQL 왕복으로 증명한다. 수정 전에 1)·3) 테스트가 실패하는 것을 먼저 확인하고 기록할 것.
- 건드릴 파일:
  - `internal/httpapi/coupons.go:144 previewCoupon` — 요청 구조체에 `Options []map[string]any \`json:"options,omitempty"\`` 추가(주문 요청과 같은 모양), uuid 파싱 실패는 주문과 같은 400 `invalid_option`. `orderAmount` 로 기본 금액을 구한 뒤 같은 `tx` 로 옵션 가격을 읽어 더한다. **`orderAmount` 의 시그니처는 바꾸지 말 것**(호출자는 이 한 곳이지만 변경 범위를 줄이는 쪽이 낫다) — 대신 같은 파일에 작은 헬퍼(예: `selectedOptionTotal(ctx, tx, talentID, ids) (int64, string, bool)`)를 두고 `SELECT id,price FROM talent_options WHERE talent_id=$1 AND active AND id=ANY($2)` 로 읽어, ⓐ 읽은 행 수가 요청 id 수와 다르면 `invalid_option` ⓑ `optionPrice > math.MaxInt64-amount` 면 `order_amount_too_large` 로 거절한다. `math` import 추가 필요(현재 coupons.go 에 없음).
  - `internal/httpapi/orders.go` — **읽기만 하고 고치지 말 것.** createOrder 의 옵션 루프를 공용 헬퍼로 추출하고 싶겠지만, 그 함수는 원장(amount·budget·coupon_redemptions)에 쓰는 경로이고 직전 세 회차가 거기서 작업했다. 15 줄 중복을 감수하고 미리보기 쪽에만 헬퍼를 두는 것이 이번 회차의 선택이다(커밋 메시지에 이유를 적을 것).
  - `web/src/pages/TalentPage.tsx:50 applyCoupon` — 본문에 `options: chosenOptions.map((id) => ({ id }))` 추가(주문 호출 `order`(:49)와 같은 모양). 옵션 체크박스는 이미 `setCoupon(null)` 로 쿠폰을 초기화하므로(:118) 다른 상태 변경은 필요 없다.
  - `docs/openapi.yaml` — `/coupons/preview`(:824) 요청 스키마에 선택 `options` 와 새 400 응답 코드를 적는다. `openapi_test.go` 는 경로 집합만 비교하므로 테스트가 강제하지는 않지만 문서 정본이다.
  - `internal/httpapi/integration_test.go` — 위 1)·3)·4) 테스트 추가. 기존 쿠폰 통합 테스트(:853, :890)와 `sellTalent` 도우미를 본뜰 것.
  - `internal/ui/dist/**` — TalentPage 수정을 실제로 배포하려면 번들 재생성이 필요하다(embed). `npm --prefix web run build` 가 도는 환경이면 재생성해 함께 커밋하고(선례: c8be99b, 71fb0b9 가 web/src 와 dist 를 같은 커밋에 담았다), Node 버전 문제로 돌지 않으면 **번들을 손대지 말고** TypeScript 원본 수정만 커밋한 뒤 "번들 미재생성" 을 커밋 메시지에 명시하라(선례: 1783d3e 는 web/src 만 바꿨다). 서버·테스트·문서만으로도 수용 기준 1)·3)·4) 는 증명된다.
- 검증 명령:
  - `go test ./internal/httpapi -run '^TestIntegrationCoupon' -count=1 -v` — `KKIIT_TEST_DSN` 미설정이면 전부 SKIP 이므로 폐기 PostgreSQL 16 DSN 을 반드시 붙일 것.
  - `make test-integration KKIIT_TEST_DSN=postgres://…` (폐기 전용 DB, timeout 300s, 과거 실측 82~93초)
  - `go test ./cmd/... ./internal/...`, `go vet ./cmd/... ./internal/...`, `gofmt -l cmd internal`, `git diff --check`
  - 프런트를 건드렸으면 `npm --prefix web test` 와 `npm --prefix web run lint`(Node 24+). 번들을 재생성했으면 `git status --short internal/ui/dist` 로 변경 범위를 확인하고 커밋 여부를 판단할 것.
- 위험과 피할 것:
  - `/coupons/preview` 에는 **요청 제한(throttle)** 이 걸려 있다(`TestIntegrationCouponPreviewIsThrottled`, integration_test.go:1446). 새 테스트에서 미리보기를 여러 번 부르면 429 로 흔들릴 수 있으니 호출 횟수를 최소로 두거나 기존 테스트가 쓰는 방식을 그대로 따를 것.
  - `decodeJSON`(types.go:102)은 `DisallowUnknownFields` 이므로 `options` 필드를 **추가**하는 것은 안전하지만(지금은 보내면 400 invalid_json), 필드 이름을 주문과 다르게 지으면 프런트 한 곳에서 두 모양을 쓰게 된다. 주문과 같은 `[{ "id": … }]` 모양을 유지할 것.
  - `createOrder`·`resolveCoupon`·`computeDiscount` 의 산식은 건드리지 말 것. 퍼센트 오버플로(10/05)와 옵션 합산 오버플로(10/07)는 이미 수정됐고, 이번 과제는 **미리보기를 주문에 맞추는 것**이지 주문 쪽 계산을 바꾸는 것이 아니다.
  - 보호 경로(auth/middleware/identity, migrations, .github/workflows) 는 건드리지 않는다. 마이그레이션 불필요.
  - `client.do` 의 `map[string]any` 숫자는 float64 다. 금액 비교는 float64 로 정확히 표현되는 값을 쓰거나 `pool` 로 int64 를 읽어 비교할 것.
  - 통합 테스트는 `apiUnderTest` 전역을 공유하므로 `t.Parallel()` 금지.
  - 미확인: 옵션을 고른 상태에서 쿠폰을 적용했을 때의 화면 값은 코드 대조로만 확인했고 브라우저로 재현하지 않았다. 또 미리보기에 옵션이 들어가면서 달라지는 다른 응답 필드가 있는지(예: 적립·정산 화면)는 조사하지 않았다.
- 차선 후보: Config.Load 필수 환경변수 누락·공백 및 비밀번호 원문 유지 계약 테스트 (가치 2 / 위험 1 / 작업량 S) — 1순위가 DB 없이 증명 불가하거나 docker 를 못 쓰는 환경이면 이쪽으로. `internal/config/config_test.go` 한 파일만 늘려 실제 `Load` 로 필수 네 값 각각의 누락 거절·공백만 입력 거절·정상값 TrimSpace·비밀번호 원문 보존(12바이트 하한)을 고정한다. `go test ./internal/config -count=1 -v` 로 검증, `t.Parallel` 금지, `t.Setenv`+`os.Unsetenv` 로 실제 미지정과 빈 문자열을 구분할 것.
