## 2026-09-05
- 선택: SQLite 모드에서 `attributes.*` 절이 문자열로 저장된 값만 찾던 문제 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `attributes.*` 경로는 Agent가 보고한 JSON 문서에서 값을 꺼내는데, 호스트가
  자기 자신에 대해 보고하는 값 대부분은 문자열이 아니라 숫자나 참거짓이다.
  PostgreSQL의 `#>>`는 저장된 값을 텍스트로 렌더링해 `= 10`이 `"10"`을,
  `= true`가 `"true"`를 만나지만, SQLite의 `json_extract`는 SQL 값(INTEGER·REAL,
  참거짓은 1과 0)을 그대로 돌려주고 SQLite는 숫자를 그 자릿수 텍스트와 같다고
  보지 않고 모든 텍스트보다 작다고 본다. 절의 값은 항상 텍스트로 바인딩되므로
  SQLite 모드에서는 `attributes.cpu_count = 10`이 무엇을 보고했든 모든 자산에
  대해 거짓이었고, `!=` 쪽은 정확히 10코어인 호스트까지 포함해 전부 참이었다.
  문자열로 저장된 값만 매칭됐고 둘 다 HTTP 200에 그럴듯한 목록이라 표시가 없었으며
  `/query/validate`는 이미 valid라고 답한 뒤였다 — PostgreSQL은 정상 동작하므로
  두 모드를 비교해야만 보였다(실제로 같은 데이터로 두 모드를 돌려 확인). 이제
  fallback은 속성 경로를 PostgreSQL과 같게 렌더링하고(참거짓은 `'true'`/`'false'`,
  나머지는 `CAST(... AS TEXT)`), 정렬 비교의 숫자 쪽만 추출값 그대로 두어 기존
  숫자 정렬 수정을 되돌리지 않게 했다. 콘솔이 렌더링하는 문법 참조에 숫자·참거짓
  표기법을 적었다(서버 `Describe()`가 내려주므로 `webui/dist` 재빌드 불필요).
  검증: querydsl 단위 테스트 2개(두 모드 컴파일 결과 고정)와
  `/api/v1/query/execute` 통합 테스트 1개(숫자·참거짓·문자열·키 없음 자산으로
  `=`/`!=`/정렬 7가지)를 추가, 기존 컴파일 기대문자열 2건 갱신. 수정 전 동일
  질의가 SQLite에서 전부 0건이던 것을 확인한 뒤 수정 후 통과. `go test ./...`를
  SQLite fallback과 실제 PostgreSQL(`scripts/test-postgres.sh`) 양쪽에서 전 패키지
  통과, `go vet`·`go build`·`gofmt`, `npm test`(130개)·`npm run build`
  (`webui/dist` 무변경 확인), `redocly lint openapi.yaml`, `cargo build --locked`
  통과. 버전 범프와 릴리즈 노트는 하지 않았다: 병렬 브랜치
  `auto/2026-09-05-1110`이 이미 v0.2.26을 만들어 두어 같은 번호를 쓰면 병합 시
  버전 파일 전체가 충돌한다.
- 보류 아이디어:
  - `exportAssets` CSV가 limit(기본 10,000)에서 조용히 잘림. 목록 API는 `total`·`has_more`를 주는데 CSV에는 아무 표시가 없음 (가치 3 / 위험 2 / M)
  - Query DSL이 `attributes.<키>`의 존재/부재 자체를 묻는 연산자를 제공하지 않음(지금은 `>= ""` 같은 우회가 필요) (가치 3 / 위험 2 / M)
  - `/api/v1/external/query/*` API key 경로의 한도·감사 로그 커버리지 (가치 3 / 위험 2 / M)
  - `attributes.*` 경로가 배열·객체를 두 모드에서 다른 텍스트로 렌더링함(SQLite는 공백 없는 JSON, PostgreSQL은 `{"k": "v"}`) (가치 2 / 위험 2 / S)
  - 콘솔이 마지막 scope 체크박스를 비활성화하지 않아 이제 400을 받고서야 알게 됨(web 변경 시 `webui/dist` 재빌드 필요) (가치 2 / 위험 2 / S)
