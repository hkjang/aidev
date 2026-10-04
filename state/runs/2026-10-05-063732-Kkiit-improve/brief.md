- 과제: 퍼센트 쿠폰 할인 계산의 int64 중간 곱셈 오버플로 방지 (가치 3 / 위험 2 / 작업량 S)
- 왜: `computeDiscount`가 `amount * terms.DiscountValue / 100`으로 계산해 최종 할인액이 int64 범위 안이어도 중간 곱셈이 넘치며, `validateTalent`는 양수 가격의 상한을 두지 않는다. 큰 금액에서도 퍼센트 쿠폰이 정확한 정수 할인액을 계산하도록 고쳐 미리보기와 주문 생성의 같은 계산 경로를 보호한다.
- 수용 기준:
  1) 양수 int64 금액과 1~100 할인율에서 정확한 내림 정수 할인액을 계산한다. `amount=1<<57`, 할인율 100이면 할인액은 원금과 같고 지급할 금액은 0이다. MaxInt64에 대해 1·50·99·100%, 100으로 나눠떨어지지 않는 금액, 최대 할인 상한을 검사한다. 최종 계산이 0이면 기존처럼 적용 불가다.
  2) 최소 주문 금액, fixed 할인, 최대 할인 상한, 잘못된 할인율 및 금액 거절은 그대로 유지한다. 금액 상한 신설이나 float64 변환으로 입력 계약을 바꾸지 않는다.
  3) 실제 `integrationServer`와 전용 PostgreSQL로 관리자 쿠폰 생성 → 판매자 상품 등록·공개 → 구매자의 `POST /api/v1/coupons/preview` → `POST /api/v1/orders`를 실행한다. 100% 쿠폰과 `int64(1)<<57` 가격으로 preview 200 / order 201, 두 응답의 discount_amount=원금·payable_amount=0을 증명한다. orders 및 coupon_redemptions 저장값도 int64로 읽어 같은지 확인한다. 수정 전 실패와 수정 후 PASS를 남기고 SKIP을 성공으로 계산하지 않는다.
- 건드릴 파일:
  - `internal/httpapi/coupons.go:computeDiscount` — percent 분기만 `(amount/100)*rate + (amount%100)*rate/100` 형태로 계산한다. 이미 검사된 amount>0, 1<=rate<=100 덕에 각 항과 합이 amount를 넘지 않는다. 기존 cap·오류 처리를 유지한다. 프로덕션 파일 총 1개.
  - `internal/httpapi/coupons_test.go` — 실제 computeDiscount의 경계 단위 테스트. 기대값은 상수 또는 테스트 전용 math/big로 계산하여 구현식을 그대로 복제하지 않는다.
  - `internal/httpapi/integration_test.go` — `TestIntegrationCouponDiscountsBuyerWithoutTouchingSellerPayout`(852행)의 등록·미리보기·주문 생성 부분을 본보기로 `TestIntegrationCouponPercentLargeAmount` 추가. `operatorClient`(545행), `sellTalent`(378행), `newClient/register/do`를 재사용한다. 결제·납품·정산 단계는 추가하지 않는다.
- 검증 명령:
  - `go test ./internal/httpapi -run 'TestComputeDiscount|TestCouponInputValidation' -count=1`
  - 전용 폐기 DB를 가리키는 KKIIT_TEST_DSN을 설정한 뒤 `go test ./internal/httpapi -run '^TestIntegrationCouponPercentLargeAmount$' -count=1 -v -timeout 300s`
  - 같은 전용 DB로 `make test-integration KKIIT_TEST_DSN="$KKIIT_TEST_DSN"`
  - `go test ./cmd/... ./internal/...` 및 `go vet ./cmd/... ./internal/...`, `gofmt -l cmd internal`(무출력)
- 위험과 피할 것: auth·session·migrations·workflows·web·internal/ui/dist·의존성은 범위 밖이다. 주문/결제/환불의 다른 산식까지 고치지 않는다. 이 회차는 금액 계산의 정확성 보강이며 실제 일반 거래에서의 발생 빈도나 피해는 확인하지 않았다. 통합 테스트는 전역 apiUnderTest를 공유하므로 t.Parallel 금지. HTTP 도우미가 숫자를 float64로 읽으므로 HTTP 경계값은 정확히 표현되는 2의 거듭제곱을 쓰고, MaxInt64의 정확한 검사는 단위 테스트 및 DB int64 Scan으로 한다. 소스 문자열 검사나 산식 복사본·Fake DB를 결함 재현으로 제출하지 않는다.
- 차선 후보: README 실행 계약을 필수 4개 + 선택 SHUTDOWN_DRAIN_SECONDS로 정리 (가치 2 / 위험 1 / S). 1순위의 실제 HTTP 재현이 성립하지 않는 경우에만 선택한다. README 45행의 “네 개뿐”과 internal/config/config.go의 Config 주석을 실제 Load/intFromEnv에 맞추고, 선택값 기본 5·허용 0~120·잘못된 값은 5를 적는다. 기존 종료 설명과 중복된 정본을 만들지 않는다. 검증: `go test ./internal/config -count=1` 및 기존 README 종료 설명과 수동 대조. 동작 변경·테스트 신설은 불필요하다.

범위·근거·미확인

- 기준 main@6f7c6cd / v0.4.13. coupons.go:50의 산식, talents.go:263 validateTalent, orders.go:createOrder의 resolveCoupon 호출, coupons.go:previewCoupon/resolveCoupon을 직접 읽었다. 두 HTTP 경로는 이미 같은 computeDiscount를 사용하므로 호출 경로를 통합하거나 변경할 이유가 없다.
- 기존 쿠폰 테스트는 보통 크기만 다루고 중간 곱셈 경계를 다루지 않는다. 산식과 타입으로 결함을 판단했으며 새 입력의 실제 HTTP 결과는 이번 정찰에서 미확인이다. 구현자가 첫 단계에서 이를 확인해야 한다.
- 정찰 실행: `go test ./cmd/... ./internal/...` PASS; 기존 TestComputeDiscount 3건 PASS; 기존 쿠폰 통합 테스트는 KKIIT_TEST_DSN 부재로 SKIP. Docker 서버 29.7.2, Go 1.26.7 확인. 새 DB/서버는 띄우지 않았으며 저장소 변경 없음.

구현 순서와 체크포인트 (모두 미착수; 사람 승인 대기 없음)

1. 위 두 테스트 파일에 회귀 검증을 추가하고 단위/단일 통합 명령을 실행한다. 체크포인트: 현재 코드에서 경계값 실패를 관찰한다. HTTP 재현이 안 되면 원인을 과제서에 기록하고, 10분 안에 입력·테스트 배선 문제인지 확인되지 않으면 차선으로 전환한다. 금액 검증을 우회하는 SQL 상품 생성은 하지 않는다.
2. coupons.go의 percent 계산만 바꾸고 같은 단위/단일 통합 명령을 실행한다. 체크포인트: 새 경계값과 기존 cap/fixed 동작 모두 PASS. 예상 밖 계약이 드러나면 과제서를 먼저 고친다.
3. 전체 통합·Go test/vet·포맷 검증을 실행한다. 체크포인트: SKIP 없는 통합 결과와 테스트/문서 외 프로덕션 1파일 범위를 확인하고 기록한다.

해법 비교와 추정 근거

- 채택: 몫·나머지 산식. 기존 유효 입력과 내림 규칙을 유지하며 새 런타임 의존성이 없다.
- 대안: 가격 상한을 강제하면 더 넓은 금액 안전 정책에는 도움이 되나 상품·패키지·옵션·API 계약의 별도 결정이 필요하여 이번 범위를 넘는다. math/big를 운영 계산에 도입하는 방법도 정확하지만 이 한 산식에는 불필요한 타입 변환이 붙는다. 현상 유지는 경계값 오동작을 남긴다.
- 가장 큰 가정: 기존 HTTP 상품 등록·공개 배선으로 해당 양수 int64 가격을 저장하고 주문 생성까지 도달할 수 있다. 코드는 이를 지지하지만 실측은 구현 단계에 남았다.
- 상향식 추정: 회귀 테스트·실제 DB 준비 10~15분, 수정 3~5분, 전체 검증·검토 10~15분 = 기본 23~35분. 알려진 불확실성(DB 준비/테스트 설정)에 예비 5~8분을 별도로 두어 총 28~43분. 통계적 신뢰구간이 아닌 중간 확신의 작업 추정이다. 과거 1개 핸들러+통합 테스트 회차와 변경 규모는 유사하나 실제 소요 시간 자료가 없어 독립 수치 교차 검증은 불가하다. 관리 예비는 배정하지 않으며 새 범위는 다음 회차로 넘긴다.
- 요청한 세 스킬은 Skill 도구가 노출되지 않아 로컬 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/`의 해당 SKILL.md를 직접 읽고 적용했다. 비교·범위·단계별 증명·체크포인트·추정 근거를 위에 반영했다.
