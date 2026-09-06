## 2026-09-06
- 선택: `/api/v1/query/execute` 가 행 상한에서 잘린 결과를 전체인 양 돌려주던 문제 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `/api/v1/query/execute` 는 행 상한(기본 100·최대 500)에서 멈추면서 그
  사실을 말하지 않았다. 400건이 맞는 질의가 100건과 HTTP 200, `"count": 100`
  으로 돌아왔고, 정확히 100건이 맞는 질의와 구분할 방법이 응답에 없었다. 이
  엔드포인트는 프로그램용이다 — API key 가 닿을 수 있는 유일한 Query DSL 입구라,
  "패치가 빠진 호스트"를 묻고 4분의 1만 받아 전부인 줄 알고 조치할 수 있었다.
  감사 기록의 `result_count` 도 잘린 수를 그대로 남겨 무엇을 묻고 무엇을
  받아갔는지의 기록까지 완전한 것처럼 읽혔다. 콘솔은 `result.length === limit`
  으로 추측했는데 그 추측은 반대 방향으로도 틀린다: limit 100 에 정확히 100건인
  완전한 답을 "잘렸을 수 있음"이라고 표시해 없는 행을 찾게 만들었다. 이제 상한보다
  한 행 더 읽고 버려 `has_more` 를 확정하고, 적용된 `limit` 을 함께 응답·감사
  기록에 넣는다(limit 을 보내지 않은 호출자가 기본값을 알아야 더 큰 값을 요청할
  수 있던 것도 없앤다). 콘솔은 추측 대신 서버의 `has_more` 를 쓴다. 같은 루프에서
  `rows.Err()` 를 확인하도록 했다 — 서버의 다른 목록 질의 30여 곳이 모두 그렇게
  하고, 반복 중 오류로 짧아진 목록을 `has_more: false` 로 보고하는 것은 행 상한이
  하던 거짓말과 같기 때문이다. 검증: 5개 자산에 대해 limit 3(잘림)·limit 5(정확히
  채운 완전한 답)·limit 미지정(기본 100 보고)을 응답과 감사 metadata 양쪽에서
  확인하는 통합 테스트 1개와 콘솔 라벨 단위 테스트 2개 추가. 감사 metadata 는
  텍스트가 아니라 JSON 으로 디코딩해 비교했다(PostgreSQL 의 JSONB 렌더링 공백이
  다르다). `go test ./...` 를 SQLite fallback 과 실 PostgreSQL
  (`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과, `go vet`·`go build`·
  `gofmt`, `npm test`(132개)·`npm run build`(`webui/dist` 재빌드 후 커밋),
  `redocly lint openapi.yaml`(경고 6개, 모두 이전과 동일한 무관 경로) 통과.
  Rust 쪽 파일은 건드리지 않아 `cargo` 는 돌리지 않았다. 문서 `.md` 는 손대지
  않았다: 이 저장소에서 `docs/*.md` 는 릴리즈 커밋에서 PDF 와 함께만 갱신된다.
  버전 범프·릴리즈 노트도 하지 않았다: 병렬 브랜치가 이미 v0.2.26 을 만들어 두어
  같은 번호를 쓰면 병합 시 버전 파일이 통째로 충돌한다.
- 보류 아이디어:
  - Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 같은 우회가 필요함 (가치 3 / 위험 2 / M)
  - `/api/v1/external/query/*` API key 경로의 rate limit·감사 기록에 테스트가 하나도 없음 (가치 3 / 위험 2 / M)
  - `listAgents`·설정 목록/이력·자산 상세의 sources/history/relations 루프가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 (가치 3 / 위험 1 / M)
  - MCP `asset_search`·`agents` 의 `has_more` 가 `len(items)==limit` 추측이라 마지막 페이지를 "더 있음"으로 알림 (가치 2 / 위험 1 / S)
  - API key 로 한 행위도 감사 기록의 `actor_type` 이 `user` 라 소유자가 콘솔에서 한 일과 구분되지 않음 (가치 2 / 위험 3 / S)
