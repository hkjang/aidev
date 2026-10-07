## 2026-10-07
- 선택: 빌드 스크립트가 쓰는 미선언 전이 의존성(harfbuzzjs·fontverter) 명시 + 미선언 import 검사 추가 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `scripts/verify-build.mjs:18-19` 가 `harfbuzzjs`·`fontverter` 를 직접 import 하는데 `package.json` 에 선언이 없어 `subset-font` 전이 의존성의 npm 호이스팅에만 기대고 있었다. lock 에 있는 그대로(`^1.6.2`·`^2.0.0`) devDependencies 에 선언하고, 같은 실수가 다시 들어오지 못하게 `scripts/verify-deps.mjs`(scripts/*.mjs·worker/*.ts·src/**/*.ts 의 bare import 대조)를 만들어 `npm run build` 첫 단계로 끼웠다. 검증: 새 검사를 고치기 전에 먼저 돌려 실패를 확인 → 선언 추가 후 통과 → 선언을 다시 지워 `npm run build` 가 1단계에서 멈추는 것까지 확인 → `npm ci` 재설치 후 `npm run build` 전체 통과(`verify-build` 도 `✓ 모두 통과`).
- 실패 재현: `node scripts/verify-deps.mjs` → `✗ scripts/verify-build.mjs: 선언 없는 import 'harfbuzzjs' — package.json 에 넣을 것` / `✗ scripts/verify-build.mjs: 선언 없는 import 'fontverter' — package.json 에 넣을 것` (exit=1). 선언 추가 후 같은 명령이 `✓ 선언 확인 23개 파일` (exit=0).
- 보류 아이디어: ① 서비스 페이지 목록 드리프트 감지(verify-build 사이트맵 목록 vs tests/site.spec.ts PAGES — 합치지 말고 감지만) ② src/lib/inquiry.ts 순수 함수 단위 테스트 ③ worker/index.ts handleInbound 첨부 누적 용량 계산 오류(초과분을 total 에서 되돌리지 않아 이후 작은 첨부까지 skip) ④ handleInbound 의 FORWARD_TO 루프 감지가 500 을 돌려 Resend 가 무한 재시도 ⑤ verify-deps 범위를 tests/·astro.config.mjs·playwright.config.ts·*.astro 로 넓히기(지금은 과제서 범위대로 3곳만)
- 과제서: 채택 — 근거(미선언 import, lock 의 1.6.2·2.0.0, 호이스팅 의존)가 코드·lock 과 정확히 일치했고 수용 기준 3개를 모두 충족했다.

