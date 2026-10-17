# 과제서 (2026-10-17 최종 정찰)

- 과제: 지급 목록 다운로드의 DB 장애도 공통 점검 안내로 전달하기 (가치 3 / 위험 1 / 작업량 M)
- 왜: `apps/web/src/api/client.ts:download()`는 503 `db_unavailable` 응답에서 `ApiError`만 던지고 `reportDegraded()`를 호출하지 않아, 지급 목록을 내려받다 DB 장애를 만난 운영자에게 기존 공통 점검 배너와 복구 확인 흐름이 시작되지 않는다. 일반 `request()`와 같은 장애 알림을 이 경로에도 연결하면 오류 토스트와 함께 “잠시 점검 중이에요” 안내를 보고 복구 확인을 받을 수 있다.
- 수용 기준:
  1) 공개 `api.download()`가 503 JSON `{error:{code:'db_unavailable',message:'...'}}`를 받으면 `onServiceDegraded` 구독자를 정확히 한 번 호출하고, 기존처럼 status 503/code db_unavailable/message를 가진 `ApiError`로 reject한다. 구독 해제 뒤에는 그 구독자가 호출되지 않는다.
  2) 다른 503(`upstream_unavailable`, `write_frozen`), 일반 400/403 및 비 JSON 오류 응답은 DB 점검 알림을 보내지 않는다. 기존 오류 메시지 fallback, 성공 CSV의 파일명·blob·클릭·URL 해제 흐름과 admin/CSRF 헤더는 유지한다. 자동 다운로드 재시도는 추가하지 않는다.
  3) 실제 React 앱의 `/admin/rewards`에서 활성화된 “전체 지급 목록 내려받기” 버튼을 누르고 확인 창을 수락한 다음, export 요청 하나만 위 503으로 응답시키면 `.in-degraded[role="status"]`에 “잠시 점검 중이에요”가 보인다. 다른 초기 API는 정상이어야 하며, 다른 API가 대신 배너를 켜는 테스트는 실패 증거로 쓰지 않는다. 가능하면 다음 정상 health 확인 후 배너가 사라지는 기존 복구 동작도 확인한다.
  4) 테스트는 실제 `api.download`를 호출한다. 구독 이벤트 단위 회귀와 실제 버튼→download→AppProvider→App 배너를 통과하는 브라우저 회귀를 모두 남기고, 알림 추가 한 줄만 되돌리면 두 양성 검증이 실패하는지 확인한다. 소스 문자열 검사나 `reportDegraded` 직접 호출로 대체하지 않는다.
- 건드릴 파일 (프로덕션 1개 + 테스트 2개):
  - `apps/web/src/api/client.ts:download()` — 실패 envelope로 지금과 동일한 `ApiError` 객체를 만든 뒤 `error.code === 'db_unavailable'`일 때 기존 `reportDegraded()`를 호출하고 그 오류를 던진다. 일반 `request()`와 다운로드 전체를 통합하지 않는다.
  - `apps/web/src/api/client.test.ts` (신규) — 공개 api/download/onServiceDegraded를 import하여 위 양성·음성·구독 해제 회귀를 작성한다. vitest 환경은 node이므로 실제 `Response`를 사용하고 window.location/document.cookie 같은 브라우저 경계와 fetch만 필요한 만큼 대역으로 제공한다. 실제 함수·알림 집합을 대역으로 바꾸지 말고, afterEach에서 구독·global·timer를 복원한다.
  - `scripts/screenshots.mjs` — 기존 `seeded.admin`/`cookiesOf()`와 실제 앱을 사용하는 desktop 전용 회귀 한 건. 정상 화면 순회와 분리된 새 context를 사용해 다른 화면에 장애 상태를 남기지 않는다. 정상 보상 화면 진입 후 export POST만 route.fulfill로 503을 주고, dialog 수락·실제 버튼 클릭·배너 표시를 assert한다. 예상한 export 503 한 건만 이 시나리오에서 허용하고 기존 session()의 전체 5xx/console 오류 게이트를 느슨하게 만들지 않는다. Chrome 자체 예상 503 console 출력도 이 전용 시나리오에서만 정확히 식별한다. 버튼이 비활성화라면 기존 seed의 ready 보상 유무를 먼저 확인하고, 배너를 직접 주입하거나 테스트를 조용히 건너뛰지 않는다.
- 검증 명령:
  - 준비: `npm ci --no-audit --fund=false` (구현 단계에서만; 정찰에서는 설치하지 않았다).
  - 단위: `npm run test -- apps/web/src/api/client.test.ts` (새 테스트 추가 후). 전체: `npm run check`; `npm run build`; `git diff --check`.
  - 현재 결함 재현: 저장소 루트에서 `node --experimental-strip-types /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-17-114850-insight.yeopjari.bid-improve/probe-download-degraded.mjs`. 기준 코드는 download 알림 0으로 assertion 실패해야 하고 수정 후 1로 통과해야 한다. 이 스크립트는 실제 client/cache 소스와 실제 로컬 HTTP 서버를 사용하며 파일을 쓰지 않는다.
  - 브라우저: 격리된 개발 DB·정식 서버·Chromium 준비 후 `BASE=http://localhost:<격리서버포트> DATABASE_URL=<격리DB소유자URL> APP_DATABASE_URL=<격리DB앱URL> node scripts/screenshots.mjs <별도스크린샷디렉터리>`. 저장소가 제공하는 실제 명령이며 포트·URL은 구현자가 준비한 인스턴스 값으로 대체한다. 이 스크립트는 seed와 fake OAuth(:18999)에 의존한다. 기존 DB·서비스를 초기화하거나 종료하지 않는다.
- 위험과 피할 것: auth/session, 서버 export/AAL/감사, db.ts의 차단기 정책, errors.ts, migrations, workflows, vendor, quota.windowStart는 범위 밖이다. 기존 다운로드 401 처리·204 캐시·Retry-After 누락 등 인접 개선도 묶지 않는다. 민감한 지급 정보나 드라이버 원문을 로그·audit details에 넣지 않는다. 이번 작업은 오류 배선 한 곳만 수정하며 DB 장애 자체를 해결하는 과제가 아니다.
- 차선 후보: README·`engineering/architecture.md`의 낡은 고정 테스트 개수를 검증 범위 설명으로 교체하기 (가치 2 / 위험 1 / S). 1순위가 이미 해결되어 있거나 브라우저 증명까지 45분 내 수행할 준비가 불가능할 때만 선택하고 이유를 기록한다. 문서 작업에서도 실행하지 않은 최신 통과 개수를 새 사실처럼 넣지 않는다.

## 확인한 근거와 미확인 범위
- 기준 main@601659d/v0.1.6. 실제 호출은 `apps/web/src/routes/Admin.tsx:AdminRewards/exportList`의 `/api/v1/admin/rewards/export`이며, `state/app.tsx:AppProvider`는 `onServiceDegraded`로 degraded=true를 만들고 30초마다 health를 확인한다. `App.tsx:App/DegradedNotice`가 배너를 렌더한다. 이 세 파일은 읽기 근거이며 수정 대상이 아니다.
- 정찰에서 Node 22.23.1로 실제 client.ts/cache.ts를 import하고 로컬 HTTP 서버가 동일 503 JSON을 반환하도록 실행: download의 ApiError(code/status)는 올바르지만 알림 0회, 일반 api.post는 알림 1회. 함수 본문은 수정하지 않았고 확장자 없는 cache import만 Node resolve hook으로 연결했다. 브라우저의 window.location/document.cookie는 최소 환경 객체이므로 UI 자체 재현은 미확인이다.
- `npm run test -- packages/core/src/__tests__/rules.test.ts`는 `vitest: not found`(127). node_modules가 없으며 단위/check/build/브라우저 전체 통과를 주장하지 않는다. seed가 바로 활성화 가능한 ready 보상을 제공하는지와 Chromium 설치 상태도 미확인이다.

## 구현 순서·확인 지점·작업량 근거
범위 밖은 위 위험 목록과 같다. 사람 승인 단계는 없으며 각 확인 지점은 구현자가 자동으로 검증하고 회차 노트에 기록한다.
1. [미착수] 현재 probe와 신규 api/client 단위 테스트로 누락을 재현한다. 증거: 위 node 명령 및 단일 vitest의 정확한 양성 실패. 체크포인트: 실패 원인이 import/설정이 아닌 알림 0회인지 확인한다.
2. [미착수] download 오류 분기만 수정한다. 증거: probe와 단일 vitest 통과. 체크포인트: 음성 오류와 성공 분기가 유지되는지 확인한다.
3. [미착수] 기존 화면 검증에 독립 브라우저 회귀를 추가하고 실제 버튼으로 배너를 확인한다. 증거: screenshots 명령과 배너 assertion, 수정 되돌림 실패 후 복구. 체크포인트: export 이외의 오류로 배너가 켜진 것이 아닌지 확인한다.
4. [미착수] `npm run check`, `npm run build`, `git diff --check`로 마무리한다. 현실이 계획과 다르면 범위를 넓히지 말고 과제서/노트의 상태와 차선 선택을 명시한다.
- 상향식 추정: API 재현·단위 6~8분, 최소 수정 2~3분, 브라우저 배선 검증 12~16분, 전체 검사·정리 5~8분 = 기본 25~35분. 알려진 변동(기존 seed/예상 503 게이트 조정) 대응 예비 5~10분을 별도로 둔 총 30~45분, 신뢰도 중간이며 통계적 백분위는 아니다. 의존성/격리 DB/Chromium 준비 가능을 가정하며 신규 환경 설치가 길어지면 재추정한다. 관리 예비는 배정하지 않았고 범위 확장으로 쓰지 않는다. 이전 1프로덕션+2테스트 회차와 파일 규모는 비슷하지만 수행 시간 기록이 없어 유사 사례로 분 단위 추정을 정당화하지 않았다.

## 대안 비교 및 스킬 적용
- 최소 수정(선택): 기존 알림 호출만 download 오류에 연결. 운영 사용자 호출과 재현 증거가 있고 프로덕션 1파일로 끝난다.
- 공통 응답 파서로 통합: 오류 계약 전체와 파일 다운로드를 함께 만져 범위·회귀 위험이 커 이번에는 제외한다.
- UI 토스트 문구만 변경: 사용자에게 오류는 이미 보이지만 공통 복구 확인이 시작되지 않아 요구를 만족하지 않는다. 보류/현상 유지도 가능하나 재현된 알림 누락이 남는다.
- 가장 큰 가정: 기존 seeded admin으로 활성 지급 버튼을 사용할 수 있어 45분 안에 최종 화면까지 검증 가능하다는 것(미확인).
- Skill 호출 도구가 없어 설치 파일을 직접 읽어 적용했다: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`(근거·범위·예비 분리), 같은 plugins 아래 `technology/skills/implementation-planning/SKILL.md`(수정·증명·확인 지점), `technology/skills/solution-exploration/SKILL.md`(대안·선택·핵심 가정). estimating의 references/sources.md도 확인했으며 외부 통계나 권위에 근거한 비용 추정은 하지 않았다.
