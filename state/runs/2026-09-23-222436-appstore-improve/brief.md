# 정찰 과제서 (2026-09-23, base main@a80264c)

- 과제: 즐겨찾기 화면의 앱 개수와 페이지 탐색을 즐겨찾기 기준으로 맞추기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `/favorites` 는 `FavoritesPage` → `AppsPage favoritesOnly`(`web/src/pages/public-pages.tsx:670`, `:198`)인데, 개수 문구가 서버 응답의 전체 공개 카탈로그 총계를 그대로 쓴다(`public-pages.tsx:357` `${apps.data.total}개 앱`). 즐겨찾기 2개를 저장한 사용자가 공개 앱이 30개인 설치에서 `/favorites` 를 열면 카드는 2장인데 "30개 앱" 이라고 읽는다. 또 공개 앱이 100개를 넘으면 `total > pageSize` 조건(`public-pages.tsx:406`)이 참이 되어 "다음" 버튼이 나오고, 누르면 offset=100 결과를 즐겨찾기 slug 로 거르므로 대개 빈 목록 + "즐겨찾기한 앱이 없습니다" 가 뜨는 막다른 길이 된다(서버 `listApps` 의 limit 상한이 100 — `internal/httpapi/public_handlers.go:76`, `pagination(r, 24, 100)`). 고치면 즐겨찾기 화면이 실제로 보여 주는 것과 같은 숫자를 말하고, 눌러도 아무것도 없는 페이지 버튼이 사라진다.

- 수용 기준:
  1) `/favorites` 에서 개수 문구가 화면에 그려진 즐겨찾기 카드 수와 같다(예: 서버가 `total: 137` 을 줘도 즐겨찾기가 2개면 "2개 앱"). 로딩 중 문구("앱 수 확인 중")와 `aria-live="polite"` 는 그대로 둔다.
  2) `favoritesOnly` 일 때는 서버 `total` 이 `pageSize` 보다 커도 페이지 탐색 `nav[aria-label="페이지"]`(이전/다음/N 페이지)가 렌더링되지 않는다.
  3) 일반 `/apps` 뷰의 총계 문구와 페이지 탐색 동작은 전과 동일하다(회귀 테스트로 고정).
  4) 테스트가 증명할 것: 수정 전에는 즐겨찾기 화면에서 전체 총계 문구와 "다음" 버튼이 나타나 새 테스트가 **실패**하고, 수정 후 통과한다. 반대로 일반 카탈로그 뷰 테스트는 수정 전후 모두 통과한다(즉 일반 뷰를 건드리지 않았음을 보인다).

- 건드릴 파일:
  - `web/src/pages/public-pages.tsx:AppsPage` — (a) 개수 문구(약 355~358행 `<span aria-live="polite">`)를 `favoritesOnly` 일 때 `filtered.length` 기준으로, 아니면 지금처럼 `apps.data.total` 로. (b) 페이지 탐색 블록(약 406~424행)의 조건에 `!favoritesOnly` 추가. 그 외 검색/정렬/보기 전환·`update()`·쿼리 키·`pageSize: favoritesOnly ? 100 : 24`·EmptyState 문구는 손대지 않는다.
  - `web/src/pages/public-pages.test.tsx` — 새 `describe("즐겨찾기 화면", …)`. 기존 파일의 패턴(`vi.stubGlobal("fetch", …)`, `QueryClientProvider` + `MemoryRouter` + `AuthProvider` + `FavoritesProvider`)을 그대로 쓰고, `localStorage.setItem("appstore.favorites", JSON.stringify(["agent-hub"]))` 로 즐겨찾기를 심은 뒤 `<AppsPage favoritesOnly />` 를 그린다. fetch mock 은 `items` 에 즐겨찾기 1~2개 + 비즐겨찾기 여러 개, `total: 137`, `limit: 100`, `offset: 0` 을 돌려준다. 테스트 끝에 `localStorage.clear()`(기존 `afterEach` 와 함께).
  - (있으면 좋음, 필수 아님) `web/e2e/core.spec.ts` — production 번들로 `/favorites` 검증 1건. mock 은 `total: sorted.length`, `limit: 24` 를 주므로(`web/e2e/mock-api.ts:350`) `page.addInitScript` 로 `appstore.favorites` 를 심고 `/favorites` 에서 개수 문구가 카드 수와 같고 `nav[aria-label="페이지"]` 가 없음을 단언하면 된다. 현재 HEAD 에는 즐겨찾기 E2E 가 없다(과거 PR #26 은 머지되지 않음) — **PR #26 의 다른 변경(AppCard 하트 숨김 등)은 재구현하지 말 것.**

- 검증 명령 (저장소 루트, 실제로 도는 것):
  - `npm --prefix web ci --no-audit --no-fund` (web/node_modules 없음 — 먼저 필요)
  - `npm --prefix web test` (현재 기준선: 이전 회차 기록상 65건. 새 테스트 수만큼 증가해야 함)
  - `npm --prefix web run lint`
  - `(cd web && npx prettier --check src/pages/public-pages.tsx src/pages/public-pages.test.tsx)`
  - `npm --prefix web run build` 및 `./scripts/check-offline-assets.sh web/dist`
  - E2E 를 추가했다면: `(cd web && npx playwright install chromium)` 후 `CI=true npm --prefix web run test:e2e`(desktop/mobile, 오래 걸림)
  - Go 는 이번 변경 범위 밖이지만 무료에 가까우면 `go build ./cmd/server` 정도만.

- 위험과 피할 것:
  - `internal/`(특히 `internal/auth`), `migrations/`, `.github/workflows/`, `internal/webui/dist` 는 손대지 않는다. 서버 `pagination`/`listApps` 나 API 계약(`limit` 상한 100)을 바꾸지 말 것 — 이번 과제는 프런트 표시 일관성만이다.
  - "즐겨찾기한 앱이 상위 100개 밖에 있으면 아예 안 보인다" 는 **별개의 결함**이고 서버 slug 필터 또는 다중 페이지 조회가 필요하다. 이번 범위 밖(아이디어로 남김). 이번 수정 후에도 개수 문구는 "보여 준 것" 을 말하므로 거짓말은 아니다.
  - 개수를 `slugs.length` 로 바꾸지 말 것 — 삭제·비공개 전환된 앱의 slug 가 localStorage 에 남아 있으면 카드 수와 또 어긋난다. 반드시 `filtered.length`.
  - `filtered` 는 `apps.data` 가 없을 때 `[]` 이므로, 로딩/에러 상태에서 "0개 앱" 이 튀어나오지 않게 기존 `apps.data ? … : "앱 수 확인 중"` 3항 구조를 유지할 것.
  - 과거 교훈: 입력 길이·제어문자 같은 검증 하드닝은 사람이 반려했다. 효과 없는 변경도 반려 사유다 — 이번 수정은 화면 문구와 버튼 유무가 실제로 달라져야 한다.
  - E2E 를 쓴다면 모바일 프로젝트에서는 sidebar 가 닫혀 있어 링크 클릭 전에 메뉴를 열어야 한다(반복된 실패 지점). 가능하면 `/favorites` 로 직접 이동해 메뉴 탐색을 피할 것.

- 차선 후보: E2E mock API 의 public config override 가 기본값에 덮어써지는 순서 정리 (`web/e2e/mock-api.ts` `installMockApi` 가 `options.config` 를 펼친 뒤 기본값을 써서 `oidcEnabled` 등을 덮지 못함 — 호출부 영향 확인 후 순서만 교정, 가치 2 / 위험 1 / S).
