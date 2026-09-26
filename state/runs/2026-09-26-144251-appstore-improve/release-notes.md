AppStore v2.11.6

즐겨찾기 하트를 누르면 카탈로그 목록 전체가 스켈레톤으로 사라지고 같은 목록을 서버에서 다시 받아 오던 문제를 고친 patch 릴리스입니다.

- `web/src/pages/public-pages.tsx` 의 카탈로그 TanStack 쿼리 키에 브라우저 쪽 즐겨찾기 `slugs` 가 들어 있으나 `queryFn` 은 그것을 쓰지 않아, `/apps`·`/mcp`·카테고리 화면에서 하트를 누를 때마다 캐시 없는 새 쿼리가 되어 그리드가 스켈레톤으로 바뀌고 같은 목록을 다시 받아 왔습니다. 키에서 `slugs` 만 제거했습니다.
- `pageSize` 100/24 를 가르는 `favoritesOnly` 는 키에 그대로 두어 두 캐시가 섞이지 않게 했습니다. 즐겨찾기 추가·해제 동작과 `/favorites` 화면은 v2.11.5 와 똑같습니다.
- 카탈로그 응답을 붙잡아 진행 중 상태를 관측하는 Vitest 2건과, 실제 번들에서 `/api/v1/apps` 요청 수를 세는 Playwright 1건을 더했습니다.
- 서버와 `internal/*` 는 바뀌지 않았고 schema 변경이 없어 기존 설치는 image만 교체하면 됩니다.
