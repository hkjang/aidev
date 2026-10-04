# 검증 기록 — 26cefd5

## 재현·원인·수정

- 원인: 양수 amount와 1~100%에서 최종 할인액은 int64 범위 내이지만 기존 amount*rate 중간값은 넘친다. 100%와 1<<57의 실제 HTTP 미리보기는 수정 전 409였다.
- 수정: coupons.go percent 분기의 계산만 `(amount/100)*rate + (amount%100)*rate/100`으로 변경. 나눗셈 내림과 입력 계약, cap/fixed/최소 주문/오류 처리는 유지했다.
- 같은 테스트가 수정 전 실패→수정 후 통과→원래 산식 복원 시 재실패했다. 최종 수정본으로 전체 검증을 수행했다.
- 입력·배선 문제는 실제 관리자 쿠폰 생성 및 판매자 상품 등록·공개가 성공하고 산식만 바꿨을 때 preview/order가 통과하는 것으로 배제했다. SQL 상품 생성이나 대역을 사용하지 않았다.

## 추가한 검증

- TestComputeDiscountPercentInt64Boundaries: 14개 금액×100개 할인율, math/big 기대값. MaxInt64의 1·50·99·100% 포함, 100의 배수·나머지·내림 0 검사.
- TestComputeDiscountLargeAmountCapsAndMinimum: 큰 금액의 cap, cap 미도달, cap 0, 최소 주문 경계, fixed 및 원금 초과 할인 상한 검사. 기존 잘못된 입력 테스트에 비율 0/음수와 percent 금액 0/음수 추가.
- TestIntegrationCouponPercentLargeAmount: 실제 integrationServer + PostgreSQL에서 관리자 쿠폰 생성→sellTalent 등록·공개→구매자 preview/order, 응답 및 DB int64 Scan 검증. 100%/1<<57에서 원금=144115188075855872, 할인액=원금, 지급액=0. orders.amount/discount_amount와 coupon_redemptions.discount_amount 일치; 지급액은 저장된 주문 금액에서 할인액을 빼서 확인한다.

## 실행 명령 및 결과

- `go test ./internal/httpapi -run 'TestComputeDiscount|TestCouponInputValidation' -count=1`: 수정 전 FAIL(unit-red.log), 수정 후 `ok ... 0.005s`(unit-green.log), 원래 산식 복원 시 FAIL(unit-revert-red.log).
- 전용 KKIIT_TEST_DSN으로 `go test ./internal/httpapi -run '^TestIntegrationCouponPercentLargeAmount$' -count=1 -v -timeout 300s`: 수정 전 409 FAIL(integration-red.log), 수정 후 PASS 0.414s(integration-green.log), 산식 복원 시 같은 409 FAIL(integration-revert-red.log).
- 같은 폐기 DB를 초기화한 뒤 `GOFLAGS=-json make test-integration KKIIT_TEST_DSN="$KKIIT_TEST_DSN"`: PASS, 88건, 실패 0, SKIP 0, 95.538s(integration-all.jsonl).
- 같은 폐기 DB를 초기화한 뒤 `GOFLAGS=-json go test ./cmd/... ./internal/...`: PASS, 테스트·서브테스트 221건, 실패 0, 테스트 SKIP 0, httpapi 97.072s(go-test-all.jsonl). 테스트 파일이 없는 cmd/kkiit·internal/mail/mailtest·internal/ui의 패키지 skip 이벤트는 테스트 SKIP과 구별했다.
- `go vet ./cmd/... ./internal/...`: exit 0, 무출력.
- `go build ./cmd/... ./internal/...`: exit 0, 무출력.
- `gofmt -l cmd internal`: exit 0, 무출력.
- `git diff --check`: exit 0, 무출력.

## 범위·미검증

- 프로덕션 1파일(coupons.go), 테스트 2파일. 인증·마이그레이션·의존성·웹·빌드/릴리즈 경로는 변경하지 않았다.
- 큰 금액의 결제·납품·정산 및 다른 주문/환불 산식은 이번 범위 밖으로 추가 검증하지 않았다. 일반 거래의 발생 빈도나 실제 피해도 확인하지 않았다.
- 웹 검사 및 release 빌드는 실행하지 않았다. 이번 변경은 서버 금액 계산이며 해당 경로를 변경하지 않았다; Go 패키지 빌드는 실행했다.
- Skill 도구가 노출되지 않아 로컬 technology/{completion-verification,systematic-debugging,test-driven-development}/SKILL.md를 직접 읽어 적용했다.
- 전용 PostgreSQL 16 컨테이너만 사용했으며 검증 후 삭제했다. 후속 역할은 자체 폐기 DB에 KKIIT_TEST_DSN을 지정해야 한다.
