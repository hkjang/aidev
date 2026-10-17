# 회차 노트 2026-10-17-152316-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:23] base pinned — main@601659d
- [러너 15:23] autonomy release — 

## 정찰 노트
- 자동 배정된 「오류 대응」의 실제 원인은 CI 가 아니라 러너 중단(`interrupted before PR`)이다. check.yml 을 읽어 느슨하게 만들 자리가 없음을 확인했고, 직전 회차가 못 끝낸 같은 과제를 프로덕션 1개+테스트 1개로 좁혀 다시 실었다 — 과제 규모가 아니라 미완이 문제였다고 봤다.
- 204 캐시 무효화·isSafeLink 정리를 제친 이유: 전자는 api.cached 소비 화면이 여전히 검색되지 않아 사용자 피해가 미확인, 후자는 가치가 문서 정리 수준이다. download 의 db_unavailable 누락은 같은 장애가 화면마다 다르게 보이는 실제 불일치다.
- 추측으로 적은 것: 503 응답 본문 모양은 errors.ts:40/108 과 db-breaker.test.ts:181 에서 유추했고 export 라우트의 실제 503 응답을 돌려 보지는 않았다. 브라우저에서 배너가 뜨는지도 미확인(state/app.tsx:107 가 리스너를 소비하는 것까지만 확인).
- 구현자 주의: node_modules 가 없으니 npm ci 로 시작할 것. apps/web 첫 테스트 파일이므로 window/document/URL 전역을 세우고 반드시 복원하고, fetch·Response 를 모킹하지 말고 포트 0 http 서버로 실제 요청을 보낼 것.
- 금지: errors.ts·db.ts·pipeline.ts·routes/admin.ts 수정, request()/download() 통합, 테스트에 수령 정보 원문 넣기.
- [러너 15:26] scout done — 지급 목록 CSV 내려받기의 DB 장애·네트워크 실패를 공통 점검 안내로 전달하기 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- 무엇을 왜: `apps/web/src/api/client.ts` 의 `download()` 가 `request()` 와 달리 503 `db_unavailable` 에서 `reportDegraded()` 를 안 부르고 fetch 실패를 날 `TypeError` 로 흘려, 지급 목록 내려받기만 공통 점검 배너를 못 띄웠다. fetch 를 try/catch 로 감싸 `network` `ApiError`(status 0) 로 바꾸고, `!response.ok` 분기에서 만든 error 가 `db_unavailable` 이면 리스너 먼저 → 같은 error throw. 프로덕션 1개 + 신규 테스트 1개. 커밋 b55ab2d.
- 확신 없는 곳·검증 못 한 것: (1) 브라우저에서 `.in-degraded` 배너가 실제로 뜨는 것은 **확인 못 했다** — `onServiceDegraded` 가 정확히 1회 불리는 데까지만 증명했다. (2) e2e(`scripts/e2e.mjs`)·스크린샷은 실 Postgres·Chromium 이 필요해 **안 돌렸다**. (3) 새 테스트가 node 환경에 `window`/`document`/`URL.createObjectURL` 을 심는다 — `afterEach` 로 복원하지만 다른 web 테스트가 생기면 순서 간섭을 다시 봐야 한다. (4) fetch 실패 재현에 포트 1(연결 거절)을 쓴다. 기존 `db-breaker.test.ts` 와 같은 방식이지만 환경에 따라 달라질 수 있다.
- 일부러 안 한 것: `request()`/`download()` 통합(성공 경로 계약이 blob 저장 vs JSON 파싱으로 다름), `download()` 의 `signal`·`x-insight-stale`·idempotency-key(별도 조각으로 ideas.json 에 남김), 204 캐시 무효화, 서버 코드·워크플로 파일·배너 UI.
- 다음 역할이 조심할 것: 이 테스트는 DB 도 브라우저도 필요 없다 — `npm run test -- apps/web/src/__tests__/client-download.test.ts` 로 단독 실행된다. 전체는 `npm run check`(7파일 113개 통과)·`npm run build` 통과. 테스트 응답 본문에 수령 정보 원문은 없다(더미 CSV 한 줄).
- [러너 15:31] brief accepted — 채택 — 과제서가 지목한 두 결함이 현재 코드에서 그대로 재현되었고, 지정한 프로덕션 1개 + 테스트 1개 범위와 수용 기
- [러너 15:31] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: client.ts 를 b55ab2d^ 로만 되돌려 새 테스트 2건이 원장과 **같은 출력**으로 실패하는 것을 직접 재현했고(복구 후 check 7파일/113개 통과, 트리 깨끗), 알림→배너 사슬을 client.ts:253 → state/app.tsx:107 → App.tsx:76 로 코드에서 이었다. vendor 패치 두 사본 동일·errored() 가 initial 을 비워 무한 재연결 없음까지 읽었다.
- 못 본 것: e2e·스크린샷(실 Postgres·Chromium 필요) 미실행 — 브라우저에서 .in-degraded 가 실제로 그려지는 것은 여전히 미검증. functions/[[path]].ts 의 R2 사본 경로도 실제 Workers 에서 돌려 보지 않았다.
- 승인이어도 남는 우려: (1) **릴리즈 전 R2 버킷 insight-page-copies 를 만들어야 한다** — 만드는 스크립트가 저장소에 없고 없으면 Pages 배포가 실패한다. (2) errors.ts:198 의 메시지 정규식이 DB 아닌 외부 연결 실패(메일·OAuth)도 '데이터베이스 점검 중' 503 으로 분류할 수 있다. (3) copies/g* R2 키에 수명 규칙이 없다(내용은 공개 k-익명 집계).
- 거절 아님 — 범위는 지정된 2개 파일 그대로, 인증·결제·마이그레이션·비밀값 미접촉, revert 로 되돌아온다. 보안·법무 차단 사유 없음.
- [러너 15:36] review approved — 리뷰 승인 (risk=low)
- [러너 15:36] pr created — https://github.com/hkjang/insight.yeopjari.bid/pull/4
- [러너 15:37] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 
