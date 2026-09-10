## 2026-09-10
- 선택: Query DSL 실행에 offset 이 없어 상한 500 을 넘는 나머지를 받아낼 방법이 아예 없음 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `query/execute` 는 최대 500행을 읽고 offset 이 없어, 조건에 맞는 자산이 그보다
  많으면 나머지에 도달할 길이 전혀 없었다 — 조건을 잘게 쪼개 여러 번 물어 이어붙이는
  것이 유일한 방법이었고, 같은 엔드포인트를 API key 로 읽는 스크립트에는 `truncated` 를
  알아차릴 콘솔조차 없다. `/api/v1/assets` 와 같은 `offset` 입력을 받고
  `offset`·`has_more`·`next_offset` 을 돌려주어 `has_more` 인 동안 `next_offset` 을
  되먹여 전체를 순회하게 했고, 같은 WHERE 에 `COUNT(*)` 를 한 번 더 걸어 `total` 로
  "몇 건이 맞았는지"를 돌려준다(감사 기록 `query.execute` 에도 `offset`·`total` 을
  남긴다). `truncated` 는 값과 의미를 그대로 두어(= `has_more`) 기존 콘솔·클라이언트는
  깨지지 않는다. 페이징을 넣으면서 정렬에 `id` 타이브레이커를 더했다 — 한 번에 수집된
  자산은 `last_seen_at` 이 초 단위로 같아, 그 컬럼만으로 정렬하면 순서가 엔진에 맡겨져
  페이지를 넘기다 어떤 행은 두 번 나오고 어떤 행은 영영 안 나온다. 루프 직후
  `rows.Err()` 검사도 함께 넣었다(보류 아이디어에 있던 `executeQuery` 부분).
  검증: `server/internal/httpapi/query_pagination_test.go` 추가 — 자산 5건을 limit 2 로
  물으면 `total` 5·`has_more` true·`next_offset` 2 이고, `next_offset` 을 따라가는 순회가
  5건을 정확히 1회씩 보며 마지막 페이지의 `next_offset` 이 5·`has_more` false 인지,
  끝을 넘긴 offset 이 빈 페이지에 `total` 5 를 주는지, 그리고 `last_seen_at` 이 같은
  자산 8건을 limit 3 으로 넘겨도 중복·누락이 없는지를 확인한다. 헛돌지 않는지 보려고
  offset 입력을 무시하도록 되돌리자 두 테스트가 각각 "offset 2 를 물었는데 0"·"8건 중
  3건만 보임"으로 실패했고, `next_offset` 을 `offset + limit` 으로 바꾸자 마지막 페이지
  단언이 6≠5 로 실패하는 것을 확인한 뒤 복구했다. 다만 `id` 타이브레이커를 빼는
  변형은 SQLite 에서 테스트가 통과한다 — 두 엔진 모두 이 크기에서는 순서가 우연히
  일정하다. 타이 테스트는 회귀 방어용으로 남겼다. `go test ./...` 를 SQLite fallback 과
  실 PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과, `go vet`·`go build`·
  `gofmt` 통과. `openapi.yaml` 에 `QueryInput.offset` 과 두 execute 엔드포인트의 새 응답
  필드를 기술하고 `@redocly/cli@2.47.0 lint` 로 경고 수가 변경 전과 같은 6개(모두 기존
  것)임을 확인했다. `docs/API_MCP_GUIDE.md` 에 페이징 설명과 curl 예시를 넣었다. web·
  Rust 는 손대지 않아 `npm`·`cargo` 는 돌리지 않았다(콘솔은 `items`·`truncated` 만 읽으므로
  추가된 키는 무해하다). 버전 범프·릴리즈 노트는 하지 않았다.
- 보류 아이디어: Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · MCP `asset_relations` 가 상대 자산의 이름·종류를 주지 않아 edge 마다 `asset_get` 을 한 번 더 부르게 만듦 (가치 3 / 위험 2 / M) · MCP `asset_search` 가 0건일 때 실제 존재하는 type·status 값을 함께 돌려주어 '자산 없음'과 '필터 값 없음'을 구분하게 함 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세 루프가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — fault injection 불가, 이번에 `executeQuery` 만 해결 (가치 3 / 위험 1 / M) · 콘솔 Query DSL 화면이 새 `total`·`offset` 을 쓰지 않아 여전히 첫 페이지만 보여줌 (가치 3 / 위험 2 / M)
