- 과제: 주문 생성의 추가 옵션 가격 합산 int64 오버플로를 400 으로 거절 (가치 3 / 위험 2 / 작업량 S)

- 왜: `createOrder`(internal/httpapi/orders.go:58-102)가 `price = talentPrice`(또는 `price = *packagePrice`)에서 출발해 선택된 옵션을 `price += optionPrice`(orders.go:95)로 **상한 검사 없이** 더한다. 판매자 입력에도 상한이 없어(`validateTalent`, talents.go:263-285 는 `BasePrice < 0`·`Options[i].Price < 0` 만 거절, DB 는 `bigint CHECK (price >= 0)` = 최대 2^63-1) 합계가 int64 를 넘길 수 있다. 한 번 감싸면(예: 2^62 옵션 2개) `price` 가 음수가 되어 `orders.amount bigint CHECK (amount >= 0)`(001_initial.sql:252) 위반으로 구매자에게 원인 불명의 500 `order_failed` 가 나가고, 두 번 감싸면(예: 2^62 옵션 4개, 합 2^64) `price` 가 **0 또는 작은 양수**가 되어 주문이 201 로 성공하면서 `orders.amount`·`order_timeline`·감사 기록·`payable_amount` 에 전부 틀린 금액이 들어간다. 즉 원장에 거짓 금액이 커밋되는 경로다. 생성 시점에 400 으로 거절하면 음수/감싼 금액이 `resolveCoupon`(orders.go:131)·`reserveBudget`(orders.go:159)·`orders` 표에 도달할 길이 막힌다.

- 수용 기준:
  1) 옵션 가격 합계가 int64 를 넘기는 주문 요청이 400 으로 거절된다(새 코드, 예: `order_amount_too_large`, 한국어 메시지). `orders` 에 행이 남지 않고 `order_timeline`·`coupon_redemptions` 도 비어 있다.
  2) 두 번 감싸 `price == 0` 이 되던 조합(합계 = 2^64)이 더는 201 로 성공하지 않는다 — 이것이 이 과제의 핵심 피해다. 수정 전에는 `amount: 0` 짜리 주문이 커밋되는 것을 테스트가 먼저 보여야 한다.
  3) 정상 금액 주문(기존 `sellTalent` 수준 가격 + 평범한 옵션)은 금액·일수·쿠폰·예산 동작이 그대로이고, 합계가 정확히 `math.MaxInt64` 인 경계는 **거절하지 않는다**(오프바이원 고정).
  4) 분기를 되돌리면(가드 제거) 1)·2) 가 다시 실패한다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/httpapi/orders.go:createOrder` — 옵션 조회 루프(`for rows.Next()`, :88-100)에서 `price += optionPrice` 전에 `optionPrice > math.MaxInt64-price` 를 검사. **주의: 루프 안에서 바로 `return` 하지 말 것** — 이 루프는 아래 `rows.Close()`(:101)로 닫히고 루프 안 기존 오류 처리도 `continue` 뿐이므로, 중간 return 은 pgx 행 집합을 닫지 않고 빠져나간다. 지역 플래그(예: `amountOverflow := false`)를 세우고 `rows.Close()` 뒤, 기존 `len(selectedOptions) != len(optionIDs)` 검사(:102) 근처에서 `writeError(w, 400, …)` 하라. `math` 를 import 에 추가해야 한다(orders.go:1-13 에 아직 없음).
  - `internal/httpapi/integration_test.go` — 새 통합 테스트 1개(아래 검증 참고).
  - `docs/openapi.yaml` — `POST /orders` 응답에 핸들러가 실제로 내는 코드가 빠져 있으면 추가. 지난 회차들의 관례. **핸들러가 실제로 내보내는 것만** 적을 것.

- 검증 명령:
  - `go test ./cmd/... ./internal/...` (DSN 없으면 통합 SKIP)
  - 폐기 PostgreSQL 16 을 띄우고 `make test-integration KKIIT_TEST_DSN=postgres://…` 또는
    `KKIIT_TEST_DSN=… go test ./internal/httpapi/ -run Integration -v`
    — 과거 회차 기준 httpapi 82~92초. 깨끗한 DB(DROP/CREATE)로 돌릴 것.
  - `go vet ./cmd/... ./internal/...`, `gofmt -l cmd internal`
  - 프런트 미변경이면 `npm --prefix web test` 는 생략 가능(이 기계의 PATH Node 20 문제는 저장소 결함이 아니다 — 2026-10-03 에 기각됨).

- 테스트 작성 참고 (확인한 것 / 미확인):
  - 확인: `sellTalent`(integration_test.go:378)는 **옵션을 등록하지 않는다**(packages·requirements 만). 따라서 그 본문을 본보기로 삼아 테스트 안에서 `POST /api/v1/talents` 를 직접 호출해 `"options": [{"name":…, "price": 1<<62, "additional_days":0, "sort_order":0, "active": true}, …]` 를 넣고 `POST /api/v1/talents/{id}/publish` 까지 해야 한다. 옵션은 `WHERE talent_id=$1 AND active AND id=ANY($2)`(orders.go:84)로 조회되므로 `active: true` 필수.
  - 확인: 주문 본문은 `{"talent_id":…, "requirements": {"요구사항":"검증"}, "options": [{"id": "<option-uuid>"}]}` 형태(integration_test.go:567 과 orders.go:71-80). 옵션 식별자는 **요청 본문이 아니라 상품에 등록된 uuid** 여야 한다.
  - 확인: `optionIDs` 에 같은 uuid 를 두 번 넣는 수법은 통하지 않는다 — `id=ANY` 가 행을 한 번만 돌려주어 `len(selectedOptions) != len(optionIDs)` 로 400 이 된다. 서로 다른 옵션 4개를 등록해야 한다.
  - 확인: `base_price` 는 0 으로 두고 옵션만 크게 하면 된다(`price` 시작값이 상품/패키지 가격). 패키지를 고르지 않으면 `price = talentPrice`.
  - 미확인: `POST /api/v1/talents` 응답이 생성된 옵션의 `id` 를 돌려주는지. 돌려주지 않으면 `GET /api/v1/talents/{id}` 또는 `integrationServer` 가 주는 `pool` 로 `SELECT id FROM talent_options WHERE talent_id=$1 ORDER BY sort_order` 를 읽어라.
  - 미확인: 2^62 가격 옵션이 `POST /talents` 를 실제로 통과하는지. 코드상 상한이 없고 2026-10-05 회차가 `1<<57` 가격 상품의 등록·공개를 실제로 성공시킨 전례가 있어 통과할 것으로 본다. 만약 중간 어딘가(JSON·프런트 무관 서버 검증)가 거절하면 그것 자체가 더 좋은 결론이니 그대로 기록하고 차선 후보로 넘어가라.
  - 주의: `client.do` 는 `map[string]any` 로 언마샬해 숫자를 float64 로 읽는다(프로필 기록). `amount: 0` 확인은 정확하지만 큰 정수 비교는 `pool` 로 `int64` Scan 하라.

- 위험과 피할 것:
  - `orders.go` 는 원장 경로다. **이번 변경은 옵션 합산 가드 한 곳만**이다. `resolveCoupon`·`reserveBudget`·`computeDiscount`(2026-10-05 에 이미 고침)·상태표 `orderTransitions`·`pay` 경로는 건드리지 말 것.
  - `days += extraDays`(orders.go:96)와 `due := time.Now().AddDate(0,0,days)` 의 int 오버플로/시간 범위는 **이번 범위 밖**이다(옵션당 `AdditionalDays <= 365` 로 제한되어 수백만 개 옵션이 필요). 발견만 기록하고 같이 고치지 말 것.
  - `REQUIREMENT_PENDING` 은 지금 HTTP 로 도달 불가한 死상태다(2026-10-06 확인). 이 과제와 섞지 말 것.
  - 통합 테스트는 `apiUnderTest` 전역을 공유하므로 **병렬화 금지**(`t.Parallel()` 쓰지 말 것).
  - 같은 DB 에 `-run Integration` 을 **두 번** 돌리면 `TestIntegrationMailNotificationsLeaveThroughTheRelay`·`TestIntegrationAgentCanResearchBeforeBuying` 가 사전 존재 결함으로 실패한다(2026-10-06 baseline 에서 확인). 자기 변경 탓으로 오인하지 말고 깨끗한 DB 로 판정하라.
  - 보호 경로(auth.go·middleware.go·identity.go·migrations·.github/workflows)는 건드릴 필요가 없다. 마이그레이션 추가 금지 — 이 수정은 Go 입력 검증으로 끝난다.
  - 새 오류 메시지는 한국어로, 기존 `writeError(status, code, 한국어)` 관례를 따를 것. 프런트는 `error.message` 를 그대로 띄우므로 `web/`·`internal/ui/dist` 는 건드리지 말 것.

- 차선 후보: README 환경변수 계약을 "필수 4개 + 선택 `SHUTDOWN_DRAIN_SECONDS`" 로 정리 (가치 2 / 위험 1 / S) — 1순위의 2^62 옵션 등록이 서버에 거절당해 피해 재현이 불가능할 때 고를 것. README 실행 계약의 "네 개뿐" 과 `internal/config` 의 선택 환경변수가 어긋나 있다(프로필 기록, 이번 회차 미재확인 — 먼저 README 와 config.go 를 대조하라).
