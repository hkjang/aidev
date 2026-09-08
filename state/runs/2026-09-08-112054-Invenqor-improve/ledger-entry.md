## 2026-09-08
- 선택: `/api/v1/external/*` API key 경로에 테스트가 하나도 없음 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 이 브랜치의 `*_test.go` 어디에서도 `/api/v1/external/` 문자열이 나오지
  않았다 — rate limit 뿐 아니라 API key 자격증명 경로 전체(401 과
  `WWW-Authenticate` 챌린지, 429 와 `Retry-After`, scope 거절, 감사 기록에 남는
  이름)가 읽히기만 하고 한 번도 실행되지 않았다. 프로그램이 실제로 쓰는 유일한
  입구인데 그렇다. `server/internal/httpapi/external_api_test.go` 에 테스트 3개를
  추가했다: (1) 한도를 넘긴 키는 `Retry-After: 60` 과 `API_RATE_LIMITED` 로 429 를
  받고, 같은 순간 다른 키는 200 을 받으며(limiter 가 KeyID 로 나뉘어 있다는 것 —
  공유 키였다면 재시도 루프에 빠진 통합 하나가 배포 전체를 429 로 만든다), 창이
  지나면 다시 통과한다. `server.apiRateLimit` 을 `newAgentRateLimiter(2,
  time.Minute)` 로 바꾸고 `now` 를 고정 시계로 교체해 600회를 두드리지 않는다.
  (2) 키 없음·우리 키가 아닌 값·우리 접두사에 모르는 비밀·폐기된 키 모두 빈
  목록이 아니라 401 로 거절되고 챌린지 헤더가 붙는다. (3) `assets.read` 만 가진
  키는 쓰기에서 403 `FORBIDDEN` 을 받고, 쓰기가 허용된 키가 만든 자산의 감사
  기록 `actor_name` 은 소유자가 아니라 `api-key:importer` 다. 검증: 새 테스트가
  헛돌지 않는지 확인하려고 `Retry-After` 설정 줄을 지우고 `Allow(KeyID)` 를
  `Allow("shared")` 로 바꿔 실패하는 것을 본 뒤 되돌렸다. `go test ./...` 를
  SQLite fallback 과 실 PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지
  통과, `go vet`·`go build`·`gofmt` 통과. Go 테스트 파일 하나만 추가했으므로
  web·Rust·`openapi.yaml` 은 손대지 않았고 `npm`·`cargo`·`redocly` 는 돌리지
  않았다. 문서 `.md` 와 버전 범프·릴리즈 노트도 하지 않았다.
- 보류 아이디어: MCP `asset_search` 의 `type`·`status` 오타가 "해당 자산 없음"으로 답해짐 — 다만 `type` 은 열린 집합이라 고정 enum 은 정답이 아님이 이번에 확인됐다 (가치 3 / 위험 2 / M) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세의 sources/history/relations 루프가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 (가치 3 / 위험 1 / M) · 잘못된 API key 는 rate limit 을 전혀 소비하지 않아(429 검사가 `Authenticate` 성공 뒤에 있음) 키 추측 시도만 무제한 (가치 2 / 위험 3 / M) · API key 로 한 행위도 감사 기록의 `actor_type` 이 `user` 라 소유자가 콘솔에서 한 일과 구분되지 않음 (가치 2 / 위험 3 / S)
