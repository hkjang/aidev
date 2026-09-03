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

## 2026-09-03
- 선택: 목록 화면의 stale response가 최신 결과를 덮어쓰는 race 방지 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: CatalogView·DirectoryView·ContentAccessView의 조회 함수들이 요청 순서를 추적하지 않아, PaginationBar 버튼·탭 전환·행 선택처럼 `:disabled="loading"`이 걸리지 않은 동선에서 요청이 겹치면 늦게 도착한 이전 응답이 최신 결과를 덮어썼다(다른 페이지 목록 표시, 취소한 상세 재출현, 지나간 오류 메시지 노출). 새 `shared/async.ts`의 `createRequestGuard()`로 ticket을 발급해 최신 요청의 결과만 상태에 반영하고, 목록을 다시 읽을 때는 진행 중인 상세 요청을 `invalidate()`로 버리며, UUID 검증 실패 같은 조기 반환 경로에서도 loading을 정리하도록 했다. 검증은 `npm run verify`(typecheck + vitest 35개 통과, 기존 30개 + guard 단위 테스트 5개로 응답 역순 도착 시나리오 포함)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과. 커밋 bbab374.
- 보류 아이디어:
  - `unwrapPageEnvelope`가 서버 pageInfo 누락 시 요청한 pageNum/pageSize를 잃고 1/10으로 되돌려 페이지 이동이 막히는 문제 (가치 3 / 위험 1 / S)
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 (가치 3 / 위험 2 / S)
  - `monitoringParser.findRows` 재귀에 깊이 제한 추가 및 테스트 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-03
- 선택: 세션 확인 실패 시 SSO 무한 왕복 차단 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: router guard가 `ensureAdminSession()` 실패를 (관리자 권한 없음 메시지를 뺀) 전부 `redirectToSso()`로 처리해, backend `/sso/ssologin`이 장애이거나 네트워크가 끊기면 SSO가 앱으로 돌려보낼 때마다 다시 SSO로 나가는 무한 왕복이 생기고 사용자는 오류 화면조차 볼 수 없었다. 새 `auth/ssoRedirect.ts`에 sessionStorage 기반 사용량 기록을 두어 60초 창에서 자동 이동을 2회로 제한하고(초과 시 access-denied 화면 + 수동 SSO 버튼), 세션 확보 시 사용량을 비우며, 저장소를 못 쓰거나 예외를 던지면 왕복 감지가 불가하므로 자동 이동을 막도록 했다. 검증은 `npm run verify`(typecheck + vitest 46개 통과, 기존 35개 + 신규 11개로 한도·창 만료·시계 역행·저장소 장애 시나리오 포함)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과. 커밋 74f26a1.
- 보류 아이디어:
  - `unwrapPageEnvelope`가 서버 pageInfo 누락 시 요청한 pageNum/pageSize를 잃고 1/10·totalCount 0으로 되돌려 페이지 이동 버튼이 잠기는 문제 (가치 3 / 위험 1 / S)
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 (가치 3 / 위험 2 / S)
  - `monitoringParser.toPod`이 `restarts`에 비숫자 문자열이 오면 NaN을 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-03
- 선택: 서버가 pageInfo의 pageNum/pageSize를 생략할 때 요청 페이지 유지 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `docs/BACKEND_API_REFERENCE.md`·`docs/API_GUIDE.md`에 기록된 기존 backend 공통 목록 계약은 `pageInfo: { totalCount }`만 반환하는데, `unwrapPageEnvelope`가 pageNum/pageSize를 서버 값에서만 읽어 매 요청마다 1/10으로 되돌렸다. 그래서 실제 backend에서는 Catalog·Directory 목록이 2페이지 데이터를 받고도 1페이지로 표시되고 다음 버튼이 같은 페이지를 반복 요청해 페이지 이동이 사실상 막혔다(mock의 `paginate`는 요청 페이지를 그대로 돌려주므로 드러나지 않았다). 서버 값이 없거나 유효하지 않으면 요청에 사용한 값을 유지하도록 하고 `apiGetPage`가 요청 params에서 그 값을 읽어 넘기게 했으며, QUICK_WINS 문서에 계약을 명시했다. 검증은 `npm run verify`(typecheck + vitest 52개 통과, 기존 46개 + 신규 6개로 totalCount만 오는 응답·유효하지 않은 서버 값·서버 값 우선·params 파싱, 그리고 axios adapter로 `apiGetPage` 전체 경로 포함)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과. 커밋 e05b613.
- 보류 아이디어:
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 (가치 3 / 위험 2 / S)
  - `monitoringParser.toPod`이 `restarts`에 비숫자 문자열이 오면 NaN을 그대로 표에 노출하는 문제 (가치 2 / 위험 1 / S)
  - `monitoringParser.findRows` 재귀에 깊이 제한·순환 참조 방어 추가 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)
