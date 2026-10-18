# 과제서 (2026-10-18, base main@e34ade5)

- 과제: `/favorites` 에서 검색해 0건이면 "즐겨찾기한 앱이 없습니다" 대신 검색 조건 때문임을 알리고 검색을 지울 수 있게 한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/src/pages/public-pages.tsx:385-403` 의 `EmptyState` 가 `favoritesOnly` 만 보고 문구를 고르므로, 즐겨찾기가 있는 사용자가 `/favorites` 의 검색창(같은 파일 `:287-303`, `favoritesOnly` 에서도 그대로 렌더됨)에 일치하지 않는 말을 넣으면 "즐겨찾기한 앱이 없습니다 / 앱 상세 또는 카드에서 하트를 눌러 보관하세요." 가 떠서 보관한 앱이 사라진 것처럼 보인다. 검색 때문이라는 안내도, 지울 버튼도 없다(`actions` 는 `:398` 에서 `!favoritesOnly` 일 때만 렌더). 검색어만 지우면 돌아온다는 사실을 화면이 말해 주게 한다.

- 수용 기준:
  1) `/favorites?q=…` 처럼 좁히는 파라미터가 있고 결과가 0건이면 제목·설명이 "검색 조건 때문" 임을 말하고, 누르면 검색이 지워져 보관한 앱이 다시 보이는 버튼이 렌더된다.
  2) 좁히는 파라미터가 없는 `/favorites` 의 0건은 기존 "즐겨찾기한 앱이 없습니다" 제목·설명 그대로이고 버튼이 없다(기존 E2E `web/e2e/core.spec.ts:785` 가 이 문구를 단정한다 — 깨뜨리지 말 것).
  3) 카탈로그(`/apps`)의 "조건에 맞는 앱이 없습니다" / "검색어나 필터를 바꾸어 보세요." / `필터 초기화` 는 한 글자도 바뀌지 않는다.
  4) 테스트가 증명할 것: 즐겨찾기 slug 를 `localStorage("appstore.favorites")` 에 넣고 `/favorites?q=…` 로 렌더해 (a) 새 안내가 보이고 (b) 버튼을 누르면 보관한 앱 카드가 다시 보이는 DOM 전이, 그리고 (c) 검색 없는 0건에서는 기존 문구가 그대로인 것.

- 건드릴 파일 (프로덕션 1개):
  - `web/src/pages/public-pages.tsx` — `AppsPage` 의 `:385-403` `EmptyState` 분기 하나만. 권장 구현: `q`/`mcp`/`featured` 중 하나라도 활성인지를 뜻하는 지역 상수(예: `const narrowed = !!q || mcp || featured;` — 세 값은 `:205-208` 에 이미 있다)를 만들고, `favoritesOnly && narrowed` 일 때만 새 제목(예: "검색 조건에 맞는 즐겨찾기 앱이 없습니다")·설명·버튼을 쓴다. 버튼은 `:247-255` 의 기존 `update` 헬퍼로 `q`(그리고 활성인 `mcp`/`featured`)만 지우는 쪽을 권한다 — `setParams({})` 는 `view`·`sort` 까지 날린다.
  - `web/src/pages/public-pages.test.tsx` — 기존 `renderCatalog`(`:133-149`)는 경로가 `/apps`·`/favorites` 로 고정이라 `initialEntries` 를 받도록 넓히거나 `:242` 의 인라인 `MemoryRouter` 패턴을 복사해 `/favorites?q=…` 로 렌더한다. `heldCatalog`(`:96-131`)는 `q` 를 무시하고 넘긴 앱을 그대로 돌려주므로, 검색 0건은 "favorite slug 와 겹치지 않는 앱만 반환" 으로 만들면 된다(실제 서버는 `q` 로 걸러 준다). 실제 `api` 클라이언트·`FavoritesProvider`·`AuthProvider` 를 쓰고 `fetch` 만 대체하는 기존 방식을 유지할 것.
  - `web/e2e/core.spec.ts` — 선택(권장). `mock-api.ts:352-361` 이 `q` 를 실제로 걸러 주므로 실제 번들에서 재현 가능하다: `localStorage` 에 즐겨찾기 1개 주입 → `/favorites` → 검색창에 일치하지 않는 말 입력 후 `검색` → 새 안내 확인 → 버튼 클릭 후 카드 복귀. 기존 즐겨찾기 E2E(`:741`, `:790`)는 수정하지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것, worktree 에 `node_modules` 없음):
  - `npm --prefix web ci --no-audit --no-fund`
  - `npm --prefix web test -- src/pages/public-pages.test.tsx` (기준선 → red → green → 제품 파일만 원복해 재실패 확인 → 복원)
  - `npm --prefix web test` · `npm --prefix web run lint` · `(cd web && npx prettier --check src/pages/public-pages.tsx src/pages/public-pages.test.tsx e2e/core.spec.ts)` · `npm --prefix web run build`
  - `./scripts/check-offline-assets.sh web/dist` · `./scripts/check-env-contract.sh` · `./scripts/check-docs.sh` · `git diff --check`
  - E2E 를 넣었으면 `(cd web && npx playwright install chromium)` 후 `CI=true npm --prefix web run test:e2e -- --grep '즐겨찾기' --workers=2 --retries=0 --reporter=list` (desktop·mobile 둘 다 통과해야 함. 모바일은 사이드바 메뉴를 먼저 열어야 링크가 잡힌다)

- 위험과 피할 것:
  - **개수 표시(`:360-363` 의 `${apps.data.total}개 앱`)와 페이지 이동(`:411-429`) 은 건드리지 말 것.** 바로 그 둘을 고친 `0898f7a`("count and page the favorites view by what it shows")가 `origin/auto/2026-09-23-2224` 에 남은 채 `main@e34ade5` 에 끝까지 병합되지 않았다(`git branch -a --contains 0898f7a` 로 확인). 사람이 받지 않은 접근이므로 같은 범위를 다시 올리지 말 것. 이번 변경은 빈 상태 문구만이라 그 커밋과 줄이 겹치지 않는다.
  - 공용 `web/src/components/ui.tsx` 의 `EmptyState`(`:258-274`)·`StatePanel` 계약·기본 문구는 바꾸지 말 것 — 다른 화면이 같이 바뀐다. 이 저장소 선례대로 화면 로컬 분기로 좁힌다.
  - `pageSize: 100` 후 slug 필터라는 구조(`:238-245`)와 쿼리 키(`:225-228`)는 그대로 둘 것. 100개 밖 즐겨찾기 누락은 별개의 보류 아이디어다.
  - 보호 경로(`internal/auth`, `migrations`, `.github/workflows`, `internal/webui/dist`)와 서버·API 계약은 건드리지 않는다. Go 쪽 변경 없음.
  - 미확인: 이번 세션에서 `npm ci`·테스트를 실행하지 않았다(원격 PR 목록도 `gh` 권한이 없어 확인 못 함). 위 명령은 직전 회차들이 실제로 통과시킨 것들이다.

- 차선 후보: 앱 상세·홈의 다른 비필수 섹션이 조회 실패를 "없음" 으로 숨기는지 확인해, 발견한 화면 **하나만** `isError` 분기로 좁혀 고친다 (가치 2 / 위험 1 / 작업량 S — 2026-10-16 회차와 같은 패턴이므로 공용 컴포넌트로 묶지 말 것)
