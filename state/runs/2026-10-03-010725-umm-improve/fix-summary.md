# fix-summary (수리 2)

- 문제: `web/package.json` 의 `engines.node: ">=22.22.2"` 가 Node 25 를 통과시켰다. 25 부터 내장 `localStorage` 가 globalThis 접근자로 무플래그 제공돼 jsdom 것을 가리고 `localStorage.clear is not a function` 으로 186중 58건이 깨진다(22·23·24 에는 그 전역이 없음을 직접 확인). 래퍼의 판정기는 `>=x.y.z` 만 읽어 참 범위를 적을 수 없었다.
- 재현: `/usr/bin/node node_modules/vitest/vitest.mjs run` → 11파일·58건 실패 EXIT=1. `PATH=/usr/bin:/bin:...nvm... npm test` 도 같은 58건 EXIT=1 — 비평가 지적 셋 다 그대로 확인.
- 고침: `engines.node` 를 `^22.22.2 || ^24.15.0` 으로 바꾸고, 판정기를 새 `web/scripts/node-range.mjs`(`||`·`^`·`>=`·`>`·`<=`·`<`·정확일치, 못 읽는 범위는 undefined 로 거절)로 교체했다. `web/scripts/node-range.test.mjs` 가 선언을 직접 읽어 20/22/23/24.0/25/26 거절·22.22.2/22.23.1/24.15+ 수락을 단언한다(vitest include 에 `scripts/**/*.test.mjs` 추가, 29건 신규). 되돌림 확인: 하한을 `>=22.22.2` 로 돌리면 그 시험 5건이 실패한다.
- 검증 EXIT=0: 루트·web `npm test`(19파일 215건), `test:offline-queue`, `typecheck`, `lint`, `build`, `verify:pwa`, `check-version.sh`, `check-i18n.mjs`, web `npm ci --dry-run`. `PATH=/usr/bin:...` 순서도 EXIT=0 이고, Node 25 만 있을 땐 "this package needs Node ^22.22.2 || ^24.15.0; found ... 25.9.0" 으로 EXIT=1 거절 — 단언 실패 58건 대신 Node 를 이름으로 지목한다. Go 파일 0건 변경이라 go test 는 돌리지 않았다.
