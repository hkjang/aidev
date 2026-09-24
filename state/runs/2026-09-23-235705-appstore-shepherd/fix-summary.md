# fix-summary — appstore PR #30 (commit 6487a31)

- 문제: 비평 지적이 맞았다. `page` 는 favoritesOnly 와 무관하게 질의에 들어가므로 카탈로그가 100개를 넘으면 /favorites?page=2 에 100번째 밖 즐겨찾기가 실제로 렌더됐고, PR 이 nav 를 지우면서 그 즐겨찾기를 도달 불가로 만들었다. 재현: mock 을 offset=100 에서 즐겨찾기 `late-app` 을 주도록 고친 Vitest 가 수정 전 "즐겨찾기한 앱이 없습니다" 로 실패.
- 고침(비평의 선택지 b): favoritesOnly 일 때 queryFn 이 서버가 알려준 pageSize 만큼 offset 을 올려 카탈로그를 끝까지 이어 받아 합친 뒤 slug 로 거른다. 그래서 `filtered.length` 개수가 전체 즐겨찾기를 뜻하게 되고 nav 제거가 참이 된다. 주석 :408-409 를 사실대로("이미 끝까지 읽었으므로 갈 페이지가 없다") 다시 썼다. queryKey 의 slugs 는 제거(클라이언트 필터라 재조회 불필요), 일반 /apps 뷰는 page/pageSize 24 그대로.
- 테스트: 틀린 전제를 굳히던 `later pages cannot hold more favorites` 를 `reads every catalog page, so a favorite past the first one still shows` 로 바꾸고 mock 을 offset 인지형(1페이지 4건 + 2페이지 `Late App`, total 137)으로 만들었다. 단언은 느슨해지지 않았고 오히려 늘었다(Late App 표시 + "1개 앱" + nav 없음).
- 검증: `npm --prefix web test` 69 passed(15 files), `run lint` 통과, `run build` + check-offline-assets 통과, `CI=true npm --prefix web run test:e2e` 73 passed / 1 skipped(기존 모바일 전용 skip), `go build ./cmd/server` 통과. 인과 확인: 구현 파일만 HEAD 로 되돌리면 새 테스트만 실패하고 복구하면 통과.
- 남은 위험: 즐겨찾기 화면은 카탈로그 크기에 비례해 순차 요청을 한다(100개 이하 1회, 137개면 2회; 1만 개면 100회). 사내 카탈로그 규모에서는 문제없다고 판단했고, 서버 limit 상한(100) 계약은 건드리지 않았다.
