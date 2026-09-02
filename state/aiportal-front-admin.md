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

## 2026-09-02
- 선택: runtime config allowlist를 host(port 포함) 비교로 강화하고 평문 http 차단 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `validateNetworkValue`가 `url.hostname`만 allowlist와 비교해 `https://api.intranet.local:8443`처럼 allowlist된 host의 다른 port로 우회할 수 있었고, `http://` 절대 URL도 그대로 통과해 `access_token` 헤더와 `withCredentials` cookie가 평문으로 나갈 수 있었다. `url.host` 비교로 바꾸고(allowedHosts 항목을 `host`/`host:port`로 정규화, 기본 port 명시는 동일 취급), loopback(`localhost`/`127.0.0.1`/`[::1]`) 외 http를 거부하며, 절대 URL이 없어도 allowedHosts 항목 형식(scheme·경로·자격증명 포함 금지)을 검사하도록 했다. 동일 규칙을 `scripts/validate-runtime-config.mjs`에도 반영했다. 검증은 `npm run verify`(typecheck + vitest 30개 통과, 기존 24개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 18개) 전체 통과, 그리고 임시 runtime.json 7종(평문 http, 다른 port, host:port 항목, loopback http, 잘못된 allowedHosts 항목 2종, 상대 경로)으로 build-time script의 exit code를 직접 대조했다. 커밋 d2796b9.
- 보류 아이디어:
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 수정 (가치 3 / 위험 2 / S)
  - 목록 화면(Catalog/Directory/ContentAccess)의 stale response가 최신 결과를 덮어쓰는 race 방지 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils)이 없어 선행 정비 필요 (가치 3 / 위험 3 / M)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)
  - `monitoringParser.findRows` 재귀에 깊이 제한 추가 및 테스트 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)
