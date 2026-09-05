## 2026-09-06
- 선택: 세션 만료를 라우트 이동 없이 즉시 복구 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 지난 세 세션이 만든 만료 감지 경로(HTTP 401과 기존 backend의 400 + 토큰 code)는 캐시된 관리자 세션을 버리기만 하고, 다시 확인하는 주체는 `router.beforeEach`뿐이었다. 그래서 한 화면에 머무르며 검색·페이지 이동·운영 현황 자동 갱신만 하는 사용자는 라우트를 옮기기 전까지 모든 조회가 "세션이 만료됐습니다. 다시 로그인하십시오."로 끝나는 화면에 갇혔다(루트 앱의 refresh 재시도는 별도 python backend endpoint라 admin-v2의 `backendBaseUrl` 계약 밖이므로 채택하지 않았다). `session.ts`에 `setSessionExpiredHandler()`를 두어 캐시를 버린 직후 복구를 알리고, guard 본문을 새 `auth/guard.ts`의 `resolveAdminAccess(force)`(`allow`/`redirected`/`denied`)로 옮겨 router guard와 만료 복구가 같은 판단과 같은 SSO 왕복 제한을 공유하게 했다. `/sso/ssologin` 자체가 만료 응답을 받는 경우는 확인이 끝나기 전에 또 확인을 시작하는 재귀가 되므로 `state.checking`으로 걸러낸다. 검증은 `npm run verify`(typecheck + vitest 80개 통과, 기존 70개 + 신규 10개로 `resolveAdminAccess` 6가지 판단과 만료 handler 호출·오래된 세대 무시·확인 중 재귀 차단·handler 예외 격리)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과, 그리고 만료 알림 호출과 재귀 차단 가드를 각각 되돌려 신규 테스트가 실제로 실패하는지 직접 확인했다. 커밋 e425b54.
- 보류 아이디어:
  - PaginationBar가 요청 페이지를 유지하다 총건수가 줄면 범위 밖 페이지에 머물러 "31–25 / 25" 같은 표시를 내는 문제 — 컴포넌트 테스트 환경 선행 필요 (가치 2 / 위험 2 / S)
  - CatalogView·ContentAccessView가 route query를 onMounted에서만 읽어 같은 경로의 query 변경(딥링크 재진입)에 반응하지 않는 문제 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils) 선행 정비 필요 (가치 3 / 위험 3 / M)
  - AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버 상태가 어긋나는 문제 (가치 2 / 위험 2 / S)
  - `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)에서 NaN을 표에 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 + `formatDateTime`이 epoch millis·`yyyyMMddHHmmss`를 날짜로 해석하지 못하는 문제 (가치 2 / 위험 1 / S)
