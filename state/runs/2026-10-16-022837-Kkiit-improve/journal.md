# 회차 노트 2026-10-16-022837-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@26c912f
- [러너 02:28] autonomy release — 

## 정찰 노트
- 설정 테스트가 두 회차(10/08 문서, 10/09 config 테스트) 연속이어서 이번에는 사용자가 돈으로 체감하는 결함을 찾았다. previewCoupon 이 옵션 금액을 빼고 집계하는 것을 createOrder·TalentPage 와 대조해 확인했고, 보류 아이디어의 rows.Err 건은 실제 결과가 400 invalid_option 에 걸려 금액 오염이 아님을 이번에 읽어 확인했으므로 제쳤다.
- 확신 없는 곳: 브라우저로 화면 값을 재현하지 않았다(코드 대조만). 번들(internal/ui/dist) 재생성 가능 여부와 재생성 시 변경 파일 수는 미확인이라 과제서에 두 갈래로 적었다. 미리보기 응답을 쓰는 다른 화면이 있는지도 조사하지 않았다.
- 구현자가 조심할 것: /coupons/preview 는 throttle 이 걸려 있어 새 테스트에서 호출을 남발하면 429 로 흔들린다. orders.go 의 옵션 루프를 공용 헬퍼로 추출하지 말고(원장 경로·직전 3회차가 손댄 자리) 미리보기 쪽에 작은 헬퍼를 두는 쪽으로 적었다.
- [러너 02:33] scout done — 쿠폰 미리보기가 선택한 추가 옵션 금액을 빼먹어 주문 화면이 실제 결제 금액보다 싼 값을 보여준다 (가�

## 구현 노트
- `previewCoupon` 이 선택한 추가 옵션 금액을 집계하지 않아 주문보다 싼 결제 금액을 약속하던 것을 고쳤다. `coupons.go` 에 `addSelectedOptions` 를 두어 `talent_options` 등록 가격으로 합산하고, 없거나 중단된 옵션은 400 `invalid_option`, int64 초과는 400 `order_amount_too_large` 로 거절한다(코드·문구 모두 `createOrder` 와 동일). `orders.go` 는 과제서 지시대로 읽기만 했고 15줄 중복은 의도적이다.
- **확신 없는 곳·검증 못 한 것**: ⓐ 브라우저로 화면을 띄워 재현하지 않았다 — `TalentPage.applyCoupon` 의 `options` 추가는 코드 대조와 lint/단위 테스트까지만. ⓑ 미리보기 응답을 쓰는 다른 화면이 있는지 조사하지 않았다(정찰 노트의 미확인 항목 그대로). ⓒ `addSelectedOptions` 는 `createOrder` 처럼 `rows.Err()` 를 보지 않고 Scan 실패를 `continue` 한다 — 기존 패턴을 일부러 본뜬 것이고, 고치려면 두 곳을 함께 봐야 한다(ideas.json 에 기록).
- **일부러 하지 않은 것**: `internal/ui/dist` 재생성. Node 22 에서 `npm run build` 는 성공했지만 Dockerfile 이 이미지 빌드에서 `web/src` 로 번들을 다시 만들어 릴리스에 실리는 것은 커밋된 dist 가 아니고, 재생성하면 추적 중인 낡은 dist 탓에 이전 회차의 web/src 변경(AdminPage·ProfilePage·OrderWorkspacePage·NotificationSettings 청크)까지 끌고 들어온다. 선례 1783d3e 와 같이 TypeScript 원본만 커밋했다. 그 결과 **브라우저에 보이는 숫자는 번들이 다시 만들어질 때까지 그대로**다 — 서버 변경은 하위 호환이라 회귀는 없다.
- **다음 역할이 조심할 것**: 새 통합 테스트 3건은 `KKIIT_TEST_DSN` 이 있어야 돈다(없으면 SKIP). 나는 폐기 PostgreSQL 16 컨테이너를 DROP/CREATE 한 깨끗한 DB 로 돌렸다. `/coupons/preview` 는 계정당 10회 throttle 이고 가장 많이 부르는 새 테스트가 4회이므로 여유는 있지만 호출을 더 늘리지 말 것. `t.Parallel()` 금지(`apiUnderTest` 전역 공유).
- [러너 02:42] brief accepted — 채택 — 지정한 파일·수법(요청 구조체에 `Options []map[string]any`, `orderAmount` 시그니처 유지, 같은 tx 로 읽는 별도 헬퍼, `math
- [러너 02:42] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인한 것: `addSelectedOptions`(coupons.go:220-260)를 `createOrder` 옵션 루프(orders.go:68-120)와 한 줄씩 대조 — 패키지/기본가 집계 순서, 중복 id, 오버플로 검사 위치, Scan 실패 처리, talent_id 스코프가 모두 일치. 새 통합 테스트 3건은 128,000/32,000/96,000 처럼 수정 전 값(100,000/25,000/75,000)과 구분되는 상수를 단언하고 미리보기 응답을 실제 주문 응답과 비교하므로 수정 전 코드에서 통과할 수 없다. 원장의 `- 실패 재현:` 두 줄이 정확히 이 증상이다. gofmt/vet/openapi_test 재실행 통과.
- 못 본 것: DSN 통합 테스트를 직접 돌리지 않았다(원장의 깨끗한 DB 실행 기록을 신뢰). 브라우저 재현도 안 했다.
- 번들: 구현자의 판단이 맞음을 Dockerfile:1-16 + release.yml 로 확인 — 릴리스 이미지는 web/src 에서 다시 빌드한 번들을 덮어쓴다. 릴리스 노트에는 "웹 번들 재빌드 후 화면에 반영" 을 적을 것.
- 남는 우려(차단 아님): ⓐ 헬퍼의 `query_failed` 가 호출부에서 400 으로 나간다(주문은 500) — openapi 의 새 400 목록에도 없다. ⓑ mcp.go:121 `preview_coupon` 스키마에 `options` 가 없어 MCP 에이전트는 이번에 고친 결함을 그대로 본다. 다음 회차 후보.
- 보안·법무: 차단 소견 없음. 새 쿼리는 파라미터 바인딩이고 읽는 값은 이미 공개된 옵션 가격, 라우트는 orders.buy + throttle 뒤. 신규 개인정보 없음.
- [러너 02:46] review approved — 리뷰 승인 (risk=low)
- [러너 02:46] pr created — https://github.com/hkjang/Kkiit/pull/24
- [러너 02:46] ci passed — 검사 없음 — 정책으로 허용
- [러너 02:47] merge done — f16ee73
- [러너 02:59] release published — v0.4.17
- [러너 03:00] assets verified — v0.4.17 자산 1개 (이전 v0.4.16: 1)
