# 회차 노트 2026-10-05-211738-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:17] base pinned — main@6e9bd1b
- [러너 21:17] autonomy release — 

## 정찰 노트
- 보류 1·2순위(명단 조회 실패와 0명 구별 / 인쇄물 경고)가 사실은 `RosterPage.tsx` 한 파일의 같은 결함이라 하나로 묶었다. 가치 4(대피 상황의 안전 정보)·위험 1·프로덕션 2파일로 끝난다. 다른 후보는 제쳤다: e2e 타입 검사는 가치 2, local-e2e 경합은 검증·릴리즈 경로, `Scan` 삼킴은 NOT NULL 때문에 실패 테스트가 불가, `csvCell`은 근거 미확인.
- 사실 확인: `offline:true` 는 서버가 아니라 `web/public/sw.js:37` 이 붙인다(`kiosk.go:210` 은 generatedAt/count/items 만) — `stale` 분기는 죽은 코드가 아니므로 지우면 오프라인 경고를 잃는다. 보류 항목이던 KeysPage `.catch` 누락은 이미 고쳐져 있어 `done` 으로 내렸다.
- 과제서에서 추측으로 적은 것: Playwright 가 서비스워커 발 요청까지 `page.route` 로 가로채는지(그래서 `serviceWorkers:"block"` 권고, 안 되면 500 fulfill), 인쇄 미디어에서 MUI `Alert` 의 실제 렌더, 한국어 문구 최종안.
- 구현자 주의: `sw.js`·`api.ts`·`kiosk.go` 를 건드리지 말 것(서버는 옳다). 인쇄 영역을 넓히되 PageHeader·버튼 두 개는 계속 숨겨야 한다. `vite.config.ts:10` 때문에 `.tsx` 테스트는 조용히 0개로 수집되니 순수 `.ts` + 실브라우저 e2e 로 나눌 것.
- [러너 21:22] scout done — 비상 대피 명단이 「조회 실패」를 「체류자 0명」으로 단정하고, 그 경고를 인쇄물에서 빼는 것 닫기 (가

## 구현 노트
- 왜/무엇: 명단을 한 번도 받지 못한 상태의 조회 실패가 `총 0명`·「체류 중인 방문자가 없습니다」로 읽혔고, 경고는 없는 명단을 「마지막으로 받은 명단을 표시합니다」라고 적었으며 그 경고가 인쇄물에서 빠졌다. 신규 순수 모듈 `web/src/roster.ts` 의 `rosterStatus` 한 값으로 기준시각·칩·빈 상태 문장·경고를 모으고, 경고 블록을 `@media print` 숨김 박스 밖으로 옮겼다(프로덕션 2파일).
- 확신 없는 곳: ① 인쇄 **실물** 출력은 확인하지 못했다 — `emulateMedia({media:"print"})` 에서 `toBeVisible()` 과 버튼·PageHeader 의 `toBeHidden()` 만 봤고, 종이·PDF 에서 Alert 의 테두리/글자색이 실제로 어떻게 나오는지는 미확인(인쇄에서 배경색이 빠질 수 있어 테두리+검은 글자+굵게를 덧붙인 것은 예방적 조치다). ② 새 스펙 18번은 `context.setOffline(true)` 로 실제 서비스워커의 `offline:true` 응답을 쓰는데, 서비스워커가 페이지를 제어하기까지 `waitForFunction` 으로 기다린 뒤 `reload()` 한다 — 느린 CI 에서 타이밍이 흔들릴 여지가 있다(여기서는 2회 연속 통과, 변이로 비어 있지 않음도 확인). ③ `errorMessage` 가 괄호로 들어가므로 브라우저 영어 메시지("Failed to fetch")가 종이에 그대로 찍힌다 — 한국어 문장을 앞에 두는 선까지만 했다.
- 일부러 하지 않은 것: `sw.js`·`kiosk.go`·`api.ts` 는 손대지 않았다(서버는 옳고 `offline` 은 sw 가 붙인다). `.tsx` 컴포넌트 테스트도 쓰지 않았다(`vite.config.ts:10` 이 `src/**/*.test.ts` 만 수집, testing-library 없음). PDF 가이드는 재생성하지 않았다(USER_GUIDE.md 한 문장만 수정). 60초 폴링의 늦은 응답 경합은 `ideas.json` 에 가치 1로 남겼다.
- 다음 역할 주의: e2e 스펙 4개는 `bash scripts/local-e2e.sh`(도커·새 DB·실제 Chromium, 약 1분 + 빌드)에서만 돈다. 3개는 `test.describe("emergency roster trust")` 안에서 `test.use({serviceWorkers:"block"})` 이고, 4번째는 별도 describe 에서 **서비스워커를 켠 채** 돌아야 의미가 있으니 둘을 합치지 말 것. `npm run lint` 는 여전히 `web/e2e` 를 타입 검사하지 않는다(기존 공백, 보류 항목).
- [러너 21:35] brief accepted — 채택 — 지정한 파일 4개·근거(57·65·68·86행)·수용 기준 1~5가 지금 코드와 정확히 맞았고 프로덕션 2개 파일로 끝났다. �
- [러너 21:35] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: `roster.ts`·`RosterPage.tsx`·`roster.test.ts`·e2e 4개 스펙·USER_GUIDE 한 문장을 전부 열었고, `npm run lint`(tsc -b) 통과·`npx vitest run` **98 passed (4 files)** 를 직접 재실행했다. 원장의 실패 재현 ②(수정 전 RosterPage + 새 스펙 → `getByText('명단 확인 불가')` not found, error-context 에 `총 0명`·「체류 중인 방문자가 없습니다」·「마지막으로 받은 명단을 표시합니다」)가 이번 변경이 고치는 증상과 정확히 일치한다. 판정 **approve** / risk low / blocking 없음(엔드포인트·인가·개인정보 수집·의존성 무변경).
- 못 본 것: `scripts/local-e2e.sh`(도커·실브라우저)는 돌리지 않았다 — e2e 4개의 실제 통과는 원장의 18 passed 기록에만 근거한다. 인쇄 실물(종이/PDF)도 미확인.
- 승인이어도 남는 우려 ①: e2e 스펙 2의 `page.locator(".MuiChip-label").first()` 는 AppShell.tsx:146 의 역할 칩을 집어 `expect(chip).toHaveText(count)` 가 항상 참이다 — 인원 수 유지는 실제로 단언되지 않는다(회귀 테스트는 스펙 1이라 차단은 아님). 다음 회차에 `getByText(/^총 \d+명$/)` 로 좁힐 것.
- 우려 ②(릴리즈 노트에 쓸 것): 칩이 핵심 문구 `명단 확인 불가` 를 들고 있으나 filled Chip 은 색 배경 위 흰 글자이고 Chrome 은 인쇄 시 배경을 끄므로 종이에서 안 보일 수 있다(기존 `총 N명` 도 동일, 신규 결함 아님). Alert·빈 상태 문장이 검은 글자로 같은 사실을 적어 종이가 거짓을 말하지는 않는다.
- 우려 ③: 캐시가 있을 때 마운트 직후 첫 조회 전까지 캐시를 `live`(경고 없음)로 읽는다 — main 과 동일한 기존 동작이지만 `loading` 플래그로 닫을 자리. 60초 폴링 경합과 `web/e2e` 미타입검사도 그대로 남는다.
- [러너 21:39] review approved — 리뷰 승인 (risk=low)
- [러너 21:39] pr created — https://github.com/hkjang/visitflow/pull/36
- [러너 21:43] ci passed — 검사 2개 모두 success
- [러너 21:43] merge done — 6adaea9
- [러너 21:51] release published — v2.8.16
- [러너 21:54] assets verified — v2.8.16 자산 1개 (이전 v2.8.15: 1)
