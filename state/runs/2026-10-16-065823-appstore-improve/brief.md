- 과제: 앱 상세의 가이드 문서 조회 실패를 "첨부 없음"처럼 숨기지 말고 안내와 다시 시도를 보여 준다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/features/apps/guide-documents.tsx:277-284` 의 `AppGuideDocuments` 는 `const items = documents.data ?? []; if (!items.length) return null;` 뿐이라 조회가 500/네트워크 오류로 끝나도 "가이드 문서" 카드가 통째로 사라져, 문서가 하나도 없는 앱과 화면상 구분되지 않는다(실패 분기 자체가 없음 — 코드로 확인). 고치면 방문자가 "문서가 없다" 로 오해하고 떠나는 대신 실패를 알고 그 자리에서 다시 받아 볼 수 있다.

- 수용 기준:
  1) `/apps/:slug` 에서 문서 조회가 실패하면 "가이드 문서" 카드가 남고 한국어 안내(예: "가이드 문서를 불러오지 못했습니다.")와 `다시 시도` 버튼이 보인다.
  2) `다시 시도` 를 누르면 같은 queryKey 를 재조회하고(`documents.refetch()`), 성공하면 안내가 사라지고 기존 목록·`내려받기` 링크(href/download 속성 그대로)가 나타난다.
  3) 조회가 성공했고 문서가 0건이면 지금처럼 아무것도 렌더하지 않는다(기존 테스트 `stays away when an app has nothing attached` 가 그대로 통과해야 한다).
  4) 로딩 중(첫 조회 pending)에는 오류 안내를 띄우지 않는다.
  5) Vitest 가 실제 `AppGuideDocuments` + 실제 `api` 클라이언트로 500 → 안내 노출 → 다시 시도 → 200 목록 노출의 전이를 DOM 으로 증명하고, 제품 파일만 원복하면 그 테스트가 다시 실패한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 2개):
  - `web/src/features/apps/guide-documents.tsx:277-311` `AppGuideDocuments` — `documents.isError`(또는 `documents.status === "error"`) 분기를 추가해 `<Card className="prose-card">` 안에 `<h2>가이드 문서</h2>` + 한국어 안내 + `<Button onClick={() => documents.refetch()}>다시 시도</Button>` 를 렌더. `Button`·`Card` 는 **이미 1~6행에서 import 되어 있다**(추가 import 불필요). 성공·0건의 `return null` 경로와 목록 마크업은 건드리지 말 것. 공용 `ErrorState`(`web/src/components/ui.tsx:277`)는 쓰지 말 것 — 제목이 "화면을 불러오지 못했습니다" 라 페이지 전체가 실패한 것처럼 읽히고 401/403 분기까지 끌고 들어온다. 이 화면은 비로그인 방문자도 보는 섹션이므로 화면 로컬 문구로 좁힌다(`owner-page.tsx` 의 로컬 `FINAL_RESULT` 표와 같은 선례).
  - `web/src/features/apps/guide-documents.test.tsx:282-298` `describe("guide documents on the app detail page")` — 여기에 2건 추가. 기존 `renderWithClient`(46행, `retry:false` 인 QueryClient)와 `jsonResponse`(24행)를 재사용하고, 기존 `stubFetch`(30행)는 항상 200 을 주므로 **이 테스트용 fetch mock 을 따로 하나 만들어** 1회차 500 → 2회차 200(`{ items: [storedDocument] }`)을 주면 된다(`vi.stubGlobal("fetch", ...)` 사용 방식은 기존 `stubFetch` 본문을 그대로 참고).
  - `web/e2e/core.spec.ts:500` `"가이드 문서는 비로그인 방문자도 앱 상세에서 내려받는다"` 바로 뒤에 E2E 1건 추가 — `await installMockApi(page);` 다음에 `page.route("**/api/v1/apps/agent-hub/documents", ...)` 를 등록(나중에 등록한 route 가 우선)해 첫 호출만 `route.fulfill({ status: 500, ... })`, 이후 호출은 `route.fallback()` 으로 mock 에 넘긴다. `/apps/agent-hub` 로 이동 → 안내 + `다시 시도` 확인 → 클릭 → `내려받기` 링크가 `"/api/v1/apps/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/documents/f1f1f1f1-f1f1-4f1f-8f1f-f1f1f1f1f1f1"` 로 보이는지 확인. 기존 500 테스트는 수정하지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것; 현재 worktree 에 node_modules 없음 — ci 선행 필수):
  1) `npm --prefix web ci --no-audit --no-fund`
  2) 기준선 → red → green: `npm --prefix web test -- src/features/apps/guide-documents.test.tsx` (현재 기준선 11 passed — 직전 회차 기록이며 이번 정찰에서 미실행)
  3) `npm --prefix web test` (직전 회차 17 files / 131 passed)
  4) `npm --prefix web run lint` (`--max-warnings 0`)
  5) `(cd web && npx prettier --check src/features/apps/guide-documents.tsx src/features/apps/guide-documents.test.tsx e2e/core.spec.ts)`
  6) `npm --prefix web run build` 후 `./scripts/check-offline-assets.sh web/dist`, `./scripts/check-env-contract.sh`, `./scripts/check-docs.sh`, `git diff --check`
  7) `(cd web && npx playwright install chromium)` 후 `CI=true npm --prefix web run test:e2e -- --grep '가이드 문서' --workers=2 --retries=0 --reporter=list` (desktop/mobile 둘 다 통과 확인 — desktop 만 보고 성공이라 쓰지 말 것)
  - 프런트 표시 변경이므로 Go race/build·DB 통합·Docker smoke 는 범위 밖.

- 위험과 피할 것:
  - 서버·API 계약(`internal/httpapi`, `web/src/lib/api.ts:334 appDocuments`)과 `web/src/types.ts` 는 건드리지 말 것. 변경은 렌더 분기 하나.
  - 같은 파일의 `useGuideDocumentDraft`(83~) · `GuideDocumentsField` · `guideDocumentError` 는 손대지 말 것 — 2026-10-08 회차(469e89a)가 방금 고친 자리이고 계약이 다르다(한 파서/경로를 넓히지 말고 좁히라는 운영자 지침).
  - 보호 경로(`internal/auth`, migrations, `.github/workflows`, `internal/webui/dist`) 미접근. `web/dist`·`server`·`web/test-results` 산출물은 커밋 금지(.gitignore 대상).
  - 운영 QueryClient 는 `web/src/app/providers.tsx:149` 에서 `retry: 1` 이라 실제 화면의 오류 표시는 2회 실패 후다. 테스트 client 는 `retry:false` 이므로 즉시 — 이 차이를 "프로덕션에서도 즉시 뜬다" 고 쓰지 말 것.
  - 문구는 화면 로컬로 두고 공용 `ErrorState`/`StatePanel` 의 기존 문구를 바꾸지 말 것(공용 변경은 다른 화면 회귀).
  - 출력이 변하지 않는 순수 테스트 추가만으로 끝내지 말 것 — 과거 반려 사유.
  - 추정: 바텀업 — 제품 10줄(5분) + Vitest 2건(10분) + E2E 1건(10분) + npm ci·build·E2E 포함 검증(20분) = 45분, 범위 35~60분(10회 중 8회). 예비 15분은 `npm ci`/`playwright install` 지연용이며 범위 확대에 쓰지 말 것.

- 차선 후보: `/favorites` 에서 검색어를 넣으면 즐겨찾기가 있어도 "즐겨찾기한 앱이 없습니다" 만 뜬다 (`web/src/pages/public-pages.tsx:385` 부근 favoritesOnly 고정 안내 — 검색 결과 없음과 즐겨찾기 없음을 분리). 단, `0898f7a` 가 main 에 없는 상태로 남아 있어 원격 PR 과 중복될 수 있으니 착수 전 `git log main --oneline | grep -i favorite` 와 `gh pr list --state open` 으로 중복을 먼저 확인할 것.
