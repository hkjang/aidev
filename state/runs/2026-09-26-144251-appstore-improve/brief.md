- 과제: 하트를 누르면 카탈로그 목록 전체가 스켈레톤으로 사라지고 서버를 다시 조회하는 문제 (가치 4 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/public-pages.tsx` 의 `AppsPage` 가 TanStack Query 키에 `slugs`(즐겨찾기 slug 배열)를 넣지만 `queryFn` 은 `slugs` 를 전혀 쓰지 않는다. 그래서 `/apps`·`/mcp`·카테고리 화면에서 카드의 하트를 누르면 키가 바뀌어 캐시가 없는 새 쿼리가 되고, `apps.isPending` 이 되어 그리드가 사라지고 `SkeletonGrid` 가 뜬 뒤 같은 목록을 서버에서 다시 받아 온다. 즐겨찾기 표시는 클라이언트 state 라 키에서 빼도 동작이 같고, 불필요한 화면 깜빡임과 목록 재요청이 없어진다.
- 수용 기준:
  1) `/apps` 목록에서 카드의 하트를 누르면 카드 그리드가 그대로 남아 있고 스켈레톤·"앱 수 확인 중" 로딩 상태로 되돌아가지 않는다. 누른 카드의 하트 상태(`aria-pressed`)와 `localStorage` 의 `appstore.favorites` 값은 그대로 갱신된다.
  2) 하트 토글로 목록 API(`/api/v1/public/apps`) 재요청이 발생하지 않는다 — 토글 전후 호출 횟수가 같다.
  3) `/favorites` 화면은 회귀가 없다: 하트를 해제하면 그 카드가 목록에서 즉시 사라지고(클라이언트 필터), 로딩 상태로 돌아가지 않는다.
  4) 테스트는 수정 전에 실패해야 한다 — 수정 전에는 토글 후 스켈레톤/추가 fetch 가 관측되고, 수정 후에는 관측되지 않음을 같은 테스트가 증명한다.
- 건드릴 파일:
  - `web/src/pages/public-pages.tsx:222` — `AppsPage` 의 `useQuery` queryKey 배열에서 `slugs` 를 제거(`{ q, category, sort, mcp, featured, page, favoritesOnly }`). `queryFn`·`pageSize: favoritesOnly ? 100 : 24`·`filtered` 의 `slugs` 클라이언트 필터는 그대로 둘 것. `favoritesOnly` 는 `pageSize` 에 실제로 쓰이므로 키에 남겨야 한다.
  - `web/src/pages/public-pages.test.tsx` — 첫 테스트("shows a catalog favorite on /favorites and removes it when toggled off", 11행~)가 이미 `QueryClientProvider` + `MemoryRouter` + `AuthProvider` + 실제 `FavoritesProvider` + `vi.stubGlobal("fetch", …)` 로 `/apps`·`/favorites` 를 태우고 있다. 같은 패턴으로 테스트를 1~2건 추가: `fetchMock` 을 변수로 잡아(87행의 두 번째 테스트가 `fetchMock.mock.calls` 를 세는 방식과 동일) 하트 클릭 전후로 `/public/apps` 호출 수가 같은지, 클릭 직후 카드 heading 이 계속 보이고 `앱 수 확인 중`·스켈레톤으로 돌아가지 않는지 단언한다. 손으로 만든 대역 컴포넌트를 쓰지 말고 실제 `AppsPage` 를 렌더할 것.
- 검증 명령:
  - `npm --prefix web ci --no-audit --no-fund` (node_modules 없을 때만)
  - `npm --prefix web test` — 기준선 건수는 **미확인**(이번 정찰은 `node_modules` 가 없고 설치 명령이 승인 대기로 막혀 실행하지 못했다). 직전 회차 기록상 65건 전후이며 구현자가 수정 전 실제 출력으로 기준선을 먼저 남길 것.
  - `npm --prefix web run lint`, `npm --prefix web run build`, `(cd web && npx prettier --check src/pages/public-pages.tsx)`
  - `./scripts/check-offline-assets.sh web/dist`
  - 선택(오래 걸림): `CI=true npm --prefix web run test:e2e -- core.spec.ts` — Chromium 설치 필요(`cd web && npx playwright install chromium`).
- 위험과 피할 것:
  - queryKey 에서 `favoritesOnly` 를 함께 지우지 말 것 — `pageSize` 가 100/24 로 갈리므로 캐시가 섞인다.
  - 같은 파일의 개수 문구(`:357` `${apps.data.total}개 앱`)와 페이지 nav(`:406`)는 **이번 과제에서 건드리지 말 것.** 2026-09-23 회차가 정확히 그 두 줄을 고쳤고 그 변경은 지금 main(3f0c566)에 없다(review-pending 으로 남음). 중복 제출이 된다.
  - 보호 경로(internal/auth·migrations·.github/workflows·internal/webui/dist)와 서버측 `pagination(r, 24, 100)` 계약은 건드리지 말 것. Go 쪽 변경 불필요.
  - E2E 는 모바일에서 sidebar 가 화면 밖으로 닫히므로 링크 클릭 전 메뉴를 열어야 한다. desktop 통과를 전체 통과로 적지 말 것.
- 차선 후보: E2E mock API 의 public config override 무효 수정 — `web/e2e/mock-api.ts:311-322` 가 `...options.config` 를 먼저 펼치고 `oidcEnabled` 등 기본값을 뒤에 써서 테스트가 주입한 config 가 항상 덮인다. 기본값을 먼저 쓰고 `...options.config` 를 뒤로 옮기고, 그 override 를 실제로 쓰는 E2E 1건(예: 로그인 방식이 없는 설치의 안내)으로 증명한다. 현재 `core.spec.ts` 의 로그인 케이스가 별도 route 로 우회 중인지 먼저 확인할 것.
