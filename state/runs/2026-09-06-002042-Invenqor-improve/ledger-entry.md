## 2026-09-06
- 선택: CSV 내보내기가 행 상한에서 조용히 잘려 부분 추출이 완전한 추출과 구분되지 않던 문제 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 두 CSV 내보내기는 행 상한에서 멈춘다 — 자산 10,000행, 감사 5,000행 —
  그런데 상한에 닿은 실행은 완전한 추출과 구별할 수 없었다. 같은
  `invenqor-assets.csv`, 같은 헤더 행, HTTP 200. 목록 API 는 `total`·`has_more`
  로 잘림을 말하지만 CSV 에는 그런 자리가 없고, 콘솔은 URL 로 이동해 내려받으므로
  응답 헤더를 볼 수 있는 것조차 없다. 검토에 첨부된 인벤토리 추출과 감사인에게
  건넨 증적 추출이 상한 너머의 행만큼 짧은 채로 전체인 양 읽혔다. 이제 두 내보내기
  모두 조건 일치 총수를 세어(자산은 `listAssets` 가 이미 쓰던 count 를
  `assetTotal` 로 뽑아 공유, 감사는 기존 `auditTotal` 재사용), 부분 추출이면
  파일 이름 자체가 `invenqor-assets-10000-of-12345.csv` 가 되고 —
  저장·전달 뒤에도 파일을 따라가는 유일한 통로다 — `X-Invenqor-Truncated`,
  `X-Invenqor-Row-Count`, `X-Invenqor-Total-Count` 헤더가 API 로 읽는 프로그램에
  같은 숫자를 준다. 감사 추출을 남기는 `audit.export` 기록에도 `matched` 와
  `truncated` 를 넣었다: 누가 무슨 증적을 가져갔는지의 기록이야말로 완전한 추출로
  읽히면 안 되는 자리다. 검증: 두 내보내기의 부분·완전 추출을 각각 확인하는 테스트
  2개 추가(파일 이름·헤더·행 수와 감사 기록의 `truncated`). PostgreSQL 은 JSONB 를
  콜론 뒤 공백과 함께 렌더링해 첫 시도가 실 PostgreSQL 에서만 실패했고, 감사 기록
  검사를 텍스트 매칭에서 JSON 디코딩으로 바꿔 두 모드에서 같게 만들었다.
  `go test ./...` 를 SQLite fallback 과 실 PostgreSQL(`scripts/test-postgres.sh`)
  양쪽에서 전 패키지 통과, `go vet`·`go build`·`gofmt`, `npm test`(130개)·
  `npm run build`(`webui/dist` 무변경 확인), `redocly lint openapi.yaml`(경고 6개,
  모두 이전과 동일한 무관 경로) 통과. Rust 쪽 파일은 건드리지 않아 `cargo` 는
  돌리지 않았다. 버전 범프·릴리즈 노트는 하지 않았다: 병렬 브랜치가 이미 v0.2.26 을
  만들어 두어 같은 번호를 쓰면 병합 시 버전 파일이 통째로 충돌한다.
- 보류 아이디어:
  - Query DSL 에 `attributes.<키>` 존재/부재 연산자가 없어 `>= ""` 같은 우회가 필요함 (가치 3 / 위험 2 / M)
  - `/api/v1/external/query/*` API key 경로의 rate limit·감사 기록에 테스트가 하나도 없음 (가치 3 / 위험 2 / M)
  - `attributes.*` 의 배열·객체 값이 두 저장 모드에서 다른 텍스트로 렌더링됨 (가치 2 / 위험 2 / S)
  - 콘솔이 마지막 scope 체크박스를 비활성화하지 않아 400 을 받고서야 알게 됨(web 변경 시 `webui/dist` 재빌드 필요) (가치 2 / 위험 2 / S)
  - `/api/v1/query/execute` 가 `has_more` 없이 limit(기본 100·최대 500)에서 자름 — 콘솔은 추측으로 알리지만 API key 로 붙는 프로그램에는 표시가 없음 (가치 2 / 위험 1 / S)
