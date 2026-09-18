## What's Changed
* auto-improve: fix: 운영 대시보드 누적 결제가 결제 상태를 잘못 읽어 늘 0원이던 것을 고치다 by @hkjang in https://github.com/hkjang/Kkiit/pull/7

운영 대시보드의 '누적 결제' 가 늘 0원이던 버그를 고친다. 결제는 `payments` 에 `state='captured'` 로만 쌓이고 주문 케이스의 `money.paid`·회원 상세도 그 값을 읽는데, 대시보드 `gmv` 만 `'succeeded'` 를 합산하고 있었다. 대시보드 쿼리를 `captured` 로 맞추고, 결제 뒤 `gmv` 가 주문 케이스의 `money.paid` 만큼 늘어나는지 통합 테스트로 확인한다.

**Full Changelog**: https://github.com/hkjang/Kkiit/compare/v0.4.1...v0.4.2
