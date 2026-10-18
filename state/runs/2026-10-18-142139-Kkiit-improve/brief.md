- 과제: 패키지를 바꾸면 적용된 쿠폰이 그대로 남아 주문 화면이 옛 패키지 기준 결제 금액을 보여준다 (가치 4 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/TalentPage.tsx:118` 의 패키지 `TextField select` 의 `onChange` 는 `setSelectedPackage(e.target.value)` 만 한다 — 같은 카드의 추가 옵션 체크박스와 쿠폰 코드 입력은 둘 다 `setCoupon(null)` 을 부르지만 패키지 select 만 빠져 있다. 그래서 쿠폰을 적용한 뒤 패키지를 바꾸면 `listPrice`(:71 `(pkg?.price ?? talent.base_price) + optionTotal`)는 새 패키지로 갱신되는데 큰 글씨 "결제 금액" 은 `coupon ? coupon.payable_amount : listPrice`(:118) 라서 **옛 패키지로 미리보기한 금액을 계속 보여 준다**. 고치면 v0.4.17(쿠폰 미리보기 옵션 누락)·v0.4.18(패키지 노출 범위)과 같은 "화면 금액이 실제 청구액과 어긋난다" 계열의 마지막 남은 경로가 닫힌다.
  - 피해는 두 가지다. ⓐ 비싼 패키지로 바꾸면 실제보다 싼 금액을 약속한다(싼 패키지로 바꾸면 더 비싸게 보여 구매를 포기시킨다). ⓑ `order`(:49)가 `coupon?.code` 를 그대로 보내므로, 새 패키지 금액이 쿠폰의 `min_order_amount` 아래로 내려가면 구매자는 "주문하기" 를 누른 마지막 순간에야 서버 거절 문구를 받는다(`createOrder` 가 `resolveCoupon` 으로 재검증한다 — `internal/httpapi/coupons.go`). 서버가 재계산하므로 **실제 청구액이 틀리는 것은 아니고, 화면과 최종 결제 경험만 어긋난다**.

- 수용 기준:
  1) 쿠폰을 적용한 상태에서 패키지를 바꾸면 적용된 쿠폰 표시(성공 Alert·"정가"/"쿠폰 할인" 두 줄)가 사라지고 "결제 금액" 이 새 패키지 기준 `listPrice` 로 돌아간다. 즉 **옛 패키지로 미리보기한 `payable_amount` 가 화면에 남는 조합이 존재하지 않는다.**
  2) 추가 옵션 체크박스를 토글했을 때의 기존 동작(쿠폰이 지워진다)이 그대로 유지된다 — 이번 변경이 그 경로를 되돌리지 않는다.
  3) 쿠폰을 적용한 뒤 아무 선택도 바꾸지 않으면 쿠폰은 그대로 적용된 채 남는다(과잉 무효화 금지).
  4) `order`(:49)가 보내는 `coupon_code` 는 화면에 실제로 적용되어 보이는 쿠폰과 항상 일치한다 — 화면에서 사라진 쿠폰 코드를 주문 요청에 몰래 실어 보내지 않는다.
  5) 테스트는 "어떤 선택 조합에서 미리보기한 쿠폰이 어떤 조합에서 유효한가" 라는 판정 규칙을 **실행으로** 고정한다: 같은 패키지·같은 옵션 집합이면 유효, 패키지가 다르면 무효, 옵션이 추가/제거되면 무효, 옵션 순서만 다르면 유효(순서 무관 비교).

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `web/src/pages/couponSelection.ts` — **신규**. 순수 모듈. 선례는 `web/src/silentSso.ts`(같은 방식으로 `silentSso.test.ts` 가 `node --test` 로 검증한다). 다음을 export:
    - `export type PreviewedCoupon = { code: string; name: string; discount_amount: number; payable_amount: number; package_id: string; option_ids: string[] }`
    - `export function couponMatchesSelection(coupon: PreviewedCoupon, packageId: string, optionIds: string[]): boolean` — `coupon.package_id === packageId` 이고 두 옵션 id 집합이 **순서 무관**으로 같을 때만 true (길이 비교 + `every(includes)`; 서버가 중복 id 를 돌려주지 않으므로 Set 길이까지 볼 필요는 없다. 중복 입력까지 막고 싶으면 Set 로 비교할 것).
    - `export function activeCoupon(coupon: PreviewedCoupon | null, packageId: string, optionIds: string[]): PreviewedCoupon | null` — 위 판정이 false 면 `null`.
    - 주석은 이 저장소 관례대로 "왜" 를 서술형 영어로 적을 것: 리셋을 호출하는 곳을 하나 빠뜨려서 생긴 결함이므로, 무효화를 **호출 책임에서 파생 계산으로 옮긴다**는 점을 적는다.
  - `web/src/pages/TalentPage.tsx` — 세 군데만:
    - `:28` 쿠폰 state 의 타입을 인라인 리터럴에서 `PreviewedCoupon | null` 로 바꾼다(`useState<PreviewedCoupon | null>(null)`). 이게 배선 보증의 핵심이다 — 타입이 `package_id`·`option_ids` 를 요구하므로 `applyCoupon` 이 선택 정보를 함께 저장하지 않으면 `tsc -b` 가 깨진다.
    - `:50` `applyCoupon` 의 `setCoupon(result)` → `setCoupon({ ...result, package_id: selectedPackage, option_ids: [...chosenOptions] })`. 미리보기 요청이 이미 `package_id: selectedPackage || undefined`, `options: chosenOptions.map((id) => ({ id }))` 를 보내므로 **요청에 실제로 쓴 값과 같은 변수를 저장**해야 한다(요청 직후 state 를 다시 읽지 말 것 — 비동기 사이에 바뀔 수 있다).
    - `:69-71` 근처에 `const shownCoupon = activeCoupon(coupon, selectedPackage, chosenOptions)` 를 두고, **`:118` 렌더의 `coupon` 세 곳(성공 Alert 조건, "정가"/"쿠폰 할인" 블록 조건, `coupon ? coupon.payable_amount : listPrice`)과 `:49` `order` 의 `coupon_code: coupon?.code || undefined` 를 모두 `shownCoupon` 으로 바꾼다.** `onClose` 핸들러의 `setCoupon(null); setCouponCode('')` 와 `:28` 의 setter 호출은 `coupon` state 를 그대로 쓴다(여기서 `shownCoupon` 으로 바꾸지 말 것). **`coupon` 식별자가 남은 자리를 하나하나 확인할 것 — 이 과제의 유일한 실수 지점이다.**
    - 옵션 체크박스의 `setCoupon(null)` 은 **그대로 두어도 되고 지워도 된다**(파생 계산이 이미 같은 결과를 낸다). 지우기로 하면 수용 기준 2 를 테스트가 아니라 눈으로 확인해야 하므로, 이번 회차에는 **남겨 두는 쪽을 권한다** — 변경 표면이 좁아진다.
    - 쿠폰 코드 TextField(`:118`)의 `setCoupon(null); setCouponError('')` 는 **건드리지 말 것**. 코드 문자열은 판정에 넣지 않았으므로 이 리셋이 계속 필요하다.
  - `web/src/pages/couponSelection.test.ts` — **신규**. `node --test` + `node:assert/strict`, 프레임워크 없음. `silentSso.test.ts` 의 머리글 주석·import 스타일(`from './couponSelection.ts'` — **확장자 `.ts` 를 붙일 것**, 기존 테스트가 그렇게 한다)을 그대로 본뜬다. 수용 기준 5 의 다섯 경우 + `coupon === null` 입력을 덮는다.

- 검증 명령 (이 저장소에서 실제로 도는 것. **`web/node_modules` 가 없으므로 `npm ci` 를 먼저 돌려야 한다**):
  ```
  npm --prefix web ci
  npm --prefix web test          # 기존 silentSso 10건 + 새 테스트. 0건으로 끝나면 환경 문제다(아래)
  npm --prefix web run lint      # eslint . --max-warnings=0
  npm --prefix web run build     # tsc -b && vite build — 타입 배선 증명의 핵심
  git diff --check
  ```
  - 먼저 **새 테스트를 red 로 만들어 두고 시작할 것**: `couponSelection.ts` 의 `couponMatchesSelection` 이 무조건 `true` 를 돌려주게 두면 패키지 불일치·옵션 변경 케이스가 실패해야 한다. 그 red 를 기록에 남기고 구현할 것.
  - Go 쪽은 건드리지 않으므로 `go test` 는 필요 없다. 그래도 한 번 돌릴 여유가 있으면 `go test ./cmd/... ./internal/...`(DSN 없으면 통합 SKIP) 로 무해함을 확인해도 된다. **PostgreSQL 통합 테스트를 이 과제의 증거로 삼을 필요는 없다.**

- 위험과 피할 것:
  - **`internal/ui/dist` 를 재생성하지 말 것.** 2026-10-16 회차 판단이 선례다: Dockerfile 이 이미지 빌드에서 `web/src` 로 `npm run build` 를 다시 돌리므로 릴리스 번들은 커밋된 dist 가 아니고, 재생성하면 추적 중인 낡은 dist 탓에 이전 회차들의 web/src 변경(AdminPage·ProfilePage·OrderWorkspacePage·NotificationSettings 청크)까지 함께 끌려 들어온다. 선례 커밋 `1783d3e` 와 같이 TypeScript 원본만 커밋한다.
  - **서버(Go)를 건드리지 말 것.** `coupons.go`·`orders.go` 는 이미 옳다(v0.4.17 에서 미리보기와 주문의 집계를 맞췄다). 이것은 순수 프런트 state 결함이다. 금액 산식(`computeDiscount`/`resolveCoupon`/`orderAmount`)에 손대면 위험 구역에 들어간다.
  - **`/coupons/preview` 를 테스트에서 반복 호출하지 말 것** — 요청 제한이 걸려 있다(`TestIntegrationCouponPreviewIsThrottled`). 이번 과제는 HTTP 를 때릴 필요가 전혀 없다.
  - **React 컴포넌트 테스트 인프라를 도입하지 말 것.** `node --test` 는 Node 의 타입 스트리핑으로 `.ts` 만 돌린다 — JSX(`.tsx`)는 스트리핑이 지원하지 않으므로 컴포넌트를 렌더하려면 vitest/jsdom 의존성 추가가 필요하고, 그것은 이번 회차 범위를 넘는 인프라 변경이다(그래서 판정 규칙을 `.ts` 순수 모듈로 빼내는 설계를 고른 것이다).
  - **증명의 한계를 정직하게 적을 것(요약·커밋 메시지에).** 새 테스트는 *판정 규칙*을 실행으로 고정하지만, `:118` 렌더가 실제로 `shownCoupon` 을 쓴다는 **배선은 `tsc -b` 통과와 diff 읽기로만 보증**된다. 런타임 테스트로 증명했다고 쓰지 말 것. (타입 교체가 `applyCoupon` 의 저장 누락은 컴파일 에러로 잡아 주지만, 렌더가 옛 `coupon` 을 계속 읽는 실수는 타입이 잡아 주지 않는다 — 그래서 위 "`coupon` 식별자가 남은 자리 확인" 이 필수다.)
  - 환경 함정: 이 워크트리의 `node --version` 은 **v22.23.1** 이고 README 는 Node 24+ 를 요구한다. 22.18+ 는 `.ts` 타입 스트리핑이 기본 켜짐이라 `npm test` 가 도는 것으로 기대하지만 **이번 정찰에서 `npm ci` 를 돌려 보지는 않았다(미확인)**. `npm test` 가 **테스트 0건**으로 끝나면 PATH 에 낡은 Node 가 끼어든 알려진 환경 문제다 — 저장소 결함으로 고치지 말고 Node 24 로 다시 돌릴 것.
  - 보호 경로(auth/migrations/workflows)는 전혀 닿지 않는다. 마이그레이션·`.github/workflows` 를 건드릴 이유가 없다.

- 차선 후보: **옵션 조회 루프가 `rows.Err()` 와 Scan 실패를 조용히 삼킨다** (가치 2 / 위험 2 / S) — `createOrder`(`internal/httpapi/orders.go:95-112`)와 `addSelectedOptions`(`internal/httpapi/coupons.go:240-252`)가 같은 패턴을 공유하므로 두 곳을 함께 고쳐야 한다. 지금 결과는 개수 불일치로 400 `invalid_option` 이라 금액 오염은 아니고 오류 문구만 부정확하다. **자연 HTTP→실제 DB 실패 재현 경로는 여전히 미확인**이므로, 고르려면 증거 설계(예: 옵션 행을 쿼리 중간에 바꾸는 경합 대신, `rows.Err()` 를 타게 하는 결정적 경로가 있는지)부터 확인할 것. 1순위가 Node 환경 문제로 전혀 검증되지 않을 때만 선택.
