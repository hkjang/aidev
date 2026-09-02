# aiportal-front-admin 자율 개선 기록

## 2026-09-02
- 선택: admin-v2 runtime config의 scheme-relative URL 우회 차단 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `validateRuntimeConfig`와 `scripts/validate-runtime-config.mjs`가 `//evil.test`, `/\evil.test`를 "상대 경로"로 보고 통과시켜 `allowedHosts` allowlist를 우회할 수 있었다. `backendBaseUrl`은 `access_token` 헤더·`withCredentials`와 함께 쓰이고 `authSsoUrl`은 `window.location.assign` 대상이라 토큰 유출·오픈 리다이렉트로 이어진다. 상대 경로 값을 sentinel origin(`https://relative.invalid/`)에 resolve해 origin이 유지되는지 확인하도록 두 검증 지점을 동일하게 고치고 회귀 테스트 3개를 추가했다. 검증은 `npm run verify`(typecheck + vitest 24개 통과, 기존 21개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 18개 검증) 전체 통과, 그리고 악성 값을 넣은 임시 runtime.json으로 build-time script가 exit 1을 내는지 직접 확인했다. 커밋 fdb7ab2.
- 보류 아이디어:
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 수정 (가치 3 / 위험 2 / S)
  - 절대 URL allowlist를 hostname뿐 아니라 port·scheme까지 비교하도록 강화 (가치 3 / 위험 2 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)
  - `monitoringParser.findRows` 재귀에 깊이 제한·순환 참조 방어 추가 및 테스트 (가치 2 / 위험 1 / S)
  - 루트 앱 Playwright 설정만 있고 `@playwright/test` dependency와 test script가 없는 상태 정비 (가치 3 / 위험 2 / M)
