## 2026-09-09
- 선택: Query DSL 실행 결과가 limit 에 걸려도 완전한 답처럼 보임 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `executeQuery` 는 limit(기본 100, 상한 500)만큼 읽은 결과를 전부인 것처럼
  `{items, count, ast}` 로 돌려줬다 — `last_seen_at < "now - 720h"` 가 1,200건에
  해당해도 100건과 `count` 100 이 오고, 나머지 1,100건이 있다는 표시가 어디에도
  없다. 이 엔드포인트에는 offset 이 없어 두 번째 요청으로 드러날 여지조차 없고,
  같은 핸들러가 API key 용 `/api/v1/external/query/execute` 에도 걸려 있어
  스크립트는 콘솔처럼 "행 수가 limit 과 같네" 하고 눈치챌 방법도 없다. 2026-09-08
  CSV 내보내기 수정과 같은 부류다. 이제 limit 보다 한 행 더 읽어(정확히 한도에서
  끝나는 결과는 여전히 '완전'으로 보고) `truncated` 와 실제 적용된 `limit` 을
  응답에 넣고, `query.execute` 감사 기록에도 `truncated` 를 남겨 부분 답이 전체
  인벤토리를 본 증거로 남지 않게 했다. 콘솔은 `len(result) === limit` 추측으로
  "잘렸을 수 있음"이라 적던 자리를 서버가 알려주는 사실("잘림" + 조치 안내)로
  바꿨다. 검증: `server/internal/httpapi/query_truncation_test.go` 추가 — limit=3
  (5건 중 잘림)에서 3행·`truncated` true·`limit` 3, limit=5(정확히 한도에서 끝나는
  완전한 결과)와 limit 미지정(기본 100)에서 `truncated` false 를 확인하고,
  `audit_logs.after_json` 에 3행 실행은 truncated=true, 5행 실행은 false 로 남는지
  확인한다(타임스탬프가 같을 수 있어 순서 대신 `result_count` 로 구분). 헛돌지
  않는지 보려고 `truncated` 계산과 응답 필드를 각각 `false` 로 바꿔 실패하는 것을
  확인한 뒤 되돌렸다. `go test ./...` 를 SQLite fallback 과 실
  PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과, `go vet`·
  `go build`·`gofmt` 통과. `npm test`(130건)·`npm run build` 통과하고
  `server/internal/webui/dist` 를 재빌드해 커밋했다(빌드 전 dist 가 체크인된 것과
  일치함을 먼저 확인해 CI 의 `git diff --exit-code` 대조가 안전함을 검증).
  `openapi.yaml` 의 두 query/execute 엔드포인트에 `limit`·`truncated` 응답 필드를
  기술하고 CI 와 같은 `@redocly/cli@2.47.0 lint` 통과(경고 6개는 모두 기존 것).
  Rust 는 손대지 않아 `cargo` 는 돌리지 않았다. 문서 `.md` 와 버전 범프·릴리즈
  노트는 하지 않았다.
- 보류 아이디어: `attributes.*` 의 배열·객체 값이 두 저장 모드에서 다른 텍스트로 렌더링됨(SQLite 은 공백 없는 JSON, PG `#>>` 는 `{"k": "v"}`) (가치 3 / 위험 2 / S) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · Query DSL 실행에 offset 이 없어 상한 500 을 넘는 나머지를 받아낼 방법이 아예 없음 — `/api/v1/assets` 처럼 offset·total 을 주는 것이 대안 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세 루프·`executeQuery` 가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — 드라이버 fault injection 없이는 테스트 불가 (가치 3 / 위험 1 / M) · MCP 의 `has_more` 가 `len(items) == limit` 추측이라 마지막 페이지에서 거짓말하고, offset 없는 agents 도구는 가져올 수 없는 페이지를 약속함 (가치 2 / 위험 1 / S)
