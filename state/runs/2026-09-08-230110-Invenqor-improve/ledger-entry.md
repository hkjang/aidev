## 2026-09-08
- 선택: attributes.* 의 등식이 에이전트가 숫자로 보고한 값과 절대 일치하지 않음 (SQLite 모드) (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `attributes.*` 절은 문서에서 뽑은 값을 사용자가 입력한 텍스트와 비교하므로
  추출식이 두 저장 모드에서 값을 같은 텍스트로 렌더링해야 한다. PostgreSQL 의
  `#>>` 는 그렇게 하지만 SQLite 의 `json_extract` 는 저장된 SQL 타입 그대로
  돌려주고 SQLite 비교는 아무 변환도 하지 않는다 — INTEGER 는 TEXT 파라미터와
  결코 같지 않고, JSON boolean 은 `'true'` 가 아니라 1/0 으로 온다. 그래서 SQLite
  fallback 에서 `attributes.cpu_count = 8` 은 그 값을 보고한 호스트가 있어도 HTTP
  200 에 빈 목록을 돌려주고, 거울상인 `attributes.cpu_count != 8` 은 8 을 가진
  자산까지 전부 남겼다(같은 질의가 PostgreSQL 에서는 정상 동작). 에이전트 payload
  가 그대로 저장되어 cpu 수·메모리 바이트·포트 번호가 JSON number 이므로 수집
  값의 상당 부분이 해당된다. SQLite 추출식이 PostgreSQL 과 같은 텍스트로 접도록
  고쳤고(`json_type` 으로 boolean 만 `'true'`/`'false'` 로, 나머지는 `CAST(... AS
  TEXT)`), `CAST` 가 SQL NULL 을 그대로 두므로 키를 보고하지 않은 자산은 여전히
  NULL 을 뽑아 `!=` 절의 NULL 처리가 유지된다. 정렬 비교(`<,<=,>,>=`)는 숫자를
  숫자로 비교해야 하므로 `attributeExpressions` 가 텍스트형과 원본 추출식을 함께
  돌려주고 숫자 분기는 원본을 쓴다. 검증: 먼저 실패하는 테스트를 썼다 —
  `server/internal/httpapi/query_attribute_types_test.go`(숫자 등식·`!=`·정렬
  회귀, boolean `= "true"`/`= "false"`, 텍스트·버전 문자열이 그대로인지) 는 수정
  전 SQLite 에서 실제로 빈 목록으로 실패했고 수정 후 통과한다. `querydsl` 단위
  테스트에 두 모드의 등식 SQL 모양을 고정하는 테스트를 추가하고, SQLite 기대값을
  쓰던 기존 두 테스트(정렬·`!=`)를 새 렌더링에 맞게 갱신했다. `go test ./...` 를
  SQLite fallback 과 실 PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지
  통과, `go vet`·`go build`·`gofmt` 통과. 문법 응답(`Describe()`)과 HTTP 계약이
  바뀌지 않아 `openapi.yaml`·web·Rust 는 손대지 않았고 `npm`·`cargo`·`redocly` 는
  돌리지 않았다. 문서 `.md` 와 버전 범프·릴리즈 노트는 하지 않았다.
- 보류 아이디어: `attributes.*` 의 배열·객체 값이 두 저장 모드에서 다른 텍스트로 렌더링됨(SQLite 은 공백 없는 JSON, PG `#>>` 는 `{"k": "v"}`) — 이번 수정으로 스칼라는 일치시켰으므로 남은 차이는 이것뿐 (가치 3 / 위험 2 / S) · Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 우회가 필요함 (가치 3 / 위험 2 / M) · `listAgents`·설정 목록/이력·자산 상세 루프·`executeQuery` 가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — 드라이버 fault injection 없이는 테스트 불가 (가치 3 / 위험 1 / M) · MCP `asset_search` 가 0건일 때 실제 존재하는 type·status 값을 함께 돌려주어 '자산 없음'과 '필터 값 없음'을 구분하게 함 (가치 3 / 위험 2 / M) · API key 로 한 행위도 감사 기록의 `actor_type` 이 `user` 라 소유자가 콘솔에서 한 일과 구분되지 않음 (가치 2 / 위험 3 / S)
