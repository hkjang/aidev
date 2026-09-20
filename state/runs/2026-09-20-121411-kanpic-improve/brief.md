# 과제서 (2026-09-20)

- 과제: IMPORTDATA 가 업로드 가져오기와 같은 규칙으로 "수처럼 생긴 글자"를 지킨다 — 앞자리 0 과 열여섯 자리 넘는 번호는 글자로 (가치 4 / 위험 1 / 작업량 S)

- 왜: 같은 CSV 가 어느 문으로 들어오느냐에 따라 다른 표가 된다. 업로드(`internal/importexport/service.go:parseScalar`)는 `hasSignificantLeadingZero`(우편번호·사번 "00123")와 `tooLongToHoldExactly`(스무 자리 계좌번호)를 걸러 글자로 남기는데, IMPORTDATA(`internal/external/fetcher.go:424`)는 `formula.DecimalNumber` 만 바로 불러 "00123" → 123, "12345678901234567890" → 1.2345678901234567e19 로 조용히 값을 바꾼다(코드로 확인 — `decimalText` 정규식은 `\d+` 라 앞자리 0 을 막지 않고, `service_test.go:22` 는 업로드 쪽이 "00123" 을 글자로 두는 것을 이미 못 박고 있다). 운영자가 되풀이한 규칙 그대로다: 같은 값을 읽는 경로가 여럿이면 모든 경로가 같은 값을 같게 읽어야 한다.

- 수용 기준:
  1) `=IMPORTDATA(url)` 로 받은 CSV 의 "00123" 칸이 글자 "00123" 으로 남는다(수 123 이 아니다). "0", "0.5", "-0", "+0" 은 예전대로 수다(`hasSignificantLeadingZero` 의 규칙 그대로 — 두 자리 이상이면서 둘째 글자가 숫자이고 소수점이 없을 때만 글자).
  2) 열여섯 자리 넘는 정수·소수("12345678901234567890", "1234567890123456.5")는 글자로 남고, 지수로 적은 "1e30" 은 수다(`tooLongToHoldExactly` 의 규칙 그대로).
  3) 업로드와 IMPORTDATA 가 **한 함수**를 부른다 — 규칙이 두 벌로 복사되지 않는다. 같은 입력 표를 `Parse`(업로드)와 `parseCSV`(IMPORTDATA) 양쪽에 넣어 셀 값이 칸마다 같은 타입·같은 값인지 한 테스트가 증명한다(한쪽만 넓히지 말라는 교훈의 검증).
  4) 기존 테스트 전체 통과 — 특히 `TestImportDataParsesATableAndCachesIt`(1200·3.5e3 가 여전히 수), `TestParseScalarKeepsGoOnlyNumbersAsText`, `TestXLSXImportKeepsNumbersNumeric`.

- 건드릴 파일:
  - `internal/delimited/` 에 새 파일(예: `number.go`) — `func Number(text string) (float64, bool)` 하나: `formula.DecimalNumber` 를 부르되 앞자리 0·너무 긴 수는 false. `hasSignificantLeadingZero`·`tooLongToHoldExactly`·`plainNumber` 를 `internal/importexport/service.go:1112-1181` 에서 이 패키지로 **옮긴다**(복사가 아니라 이동; importexport 쪽 호출 세 곳 `service.go:1104`, `:1160`, `:1172` 는 새 함수/이동한 함수를 부르게). `internal/delimited` 는 지금 `strings`·`encoding/binary` 만 import 하고 `internal/formula` 는 delimited 를 import 하지 않으므로 delimited → formula 의존은 순환이 아니다(확인: `internal/formula/*.go` 에 `internal/delimited` 없음). 만약 delimited 가 formula 를 끌어오는 것이 마음에 걸리면 차선으로 `formula` 패키지 안에 `FileNumber` 를 두는 것도 된다 — 어느 쪽이든 한 곳.
  - `internal/importexport/service.go:parseScalar` — 세 조건을 새 한 함수로 바꾼다. `parseXLSXValue` 의 `CellTypeUnset` 갈래(`:1160`)도 같은 함수로. `CellTypeNumber` 갈래(`:1172`)는 엑셀이 이미 수라고 밝힌 칸이므로 그대로 두어도 되나, 그대로 둔 이유를 한 줄 적을 것.
  - `internal/external/fetcher.go:423-428` — `formula.DecimalNumber(text)` → 새 함수. `strings.TrimSpace` 는 지금처럼 유지(업로드 쪽 `parseScalar` 는 trim 하지 않는 것으로 보인다 — **미확인**, 구현자가 `Parse` 경로에서 CSV 칸이 어디서 trim 되는지 한 번 볼 것; 이 과제에서 trim 규칙까지 맞추지는 않는다).
  - 테스트: `internal/delimited/number_test.go`(표 기반: "00123"·"0"·"0.5"·"-0"·"007.5"·"1e30"·"12345678901234567890"·"NaN"·"1_000"), `internal/external/fetcher_test.go` 에 `TestImportDataKeepsNumbersThatUploadKeepsAsText` 같은 이름으로 실제 httptest 서버에서 받은 CSV 로 검증(기존 `TestImportDataParsesATableAndCachesIt` 의 `serve`·`withInsecureTLS`·`testFetcher` 헬퍼 재사용). 수용 기준 3 의 양쪽 비교 테스트는 `internal/importexport` 는 `external` 을 import 하지 않으므로 `internal/external` 쪽에 두거나(importexport.Parse 를 부른다), 순환이 나면 `internal/integration` 이 아닌 일반 테스트 파일 어디든 두 패키지를 모두 import 할 수 있는 곳(예: `internal/httpapi` 테스트)에 둔다.

- 검증 명령(저장소 루트에서, 모두 실제로 도는 것을 확인함):
  - `gofmt -l internal/` (출력 없어야 함)
  - `go vet ./...`
  - `go build ./...`
  - `go test ./internal/external/ ./internal/importexport/ ./internal/delimited/ ./internal/formula/`
  - `go test ./...` (전체, 약 1~2분)
  - `scripts/check-release-docs.sh`, `scripts/check-commit-identities.sh HEAD`
  - 웹은 손대지 않으므로 npm 검사는 생략 가능(단 `testdata/numeric-text.json` 은 손대지 말 것 — 그 파일은 "셈에 끼어드는 값" 의 규칙이고 이 과제는 "파일에서 온 값" 의 규칙이다. 두 규칙이 다른 것은 `DecimalNumber` 주석이 의도로 적어 두었다).

- 위험과 피할 것:
  - `formula.DecimalNumber` 자체와 `decimalText` 정규식은 **바꾸지 말 것** — 수식 엔진의 `numberFromText` 와 격자의 `spreadsheetNumber.ts` 가 같은 자를 쓰고, 거기서 "00123" 은 셈에 123 으로 들어가는 것이 맞다(엑셀도 그렇다). 이 과제는 파일에서 온 값을 **칸에 담을 때**의 규칙만 맞춘다.
  - `valueOfText`/`textValueNumber`(VALUE 함수)도 건드리지 말 것. 보류 아이디어의 "IMPORTDATA 가 1,200·12% 를 글자로 남긴다" 는 이 과제가 아니다 — 그것은 넓히는 방향이고, 업로드도 똑같이 글자로 남기므로 지금은 두 문이 이미 일치한다.
  - `unguardDelimitedValue`(아포스트로피 가드 풀기)는 IMPORTDATA 에 넣지 말 것 — 원격 CSV 는 이 제품이 내보낸 것이 아니고, 넣으면 원격 파일의 `'` 로 시작하는 글자 값이 바뀐다. 범위 밖.
  - 캐시(`external.cache_seconds`)·허용 호스트·정책 코드는 건드리지 말 것.
  - 효과 없는 변경 금지: 고치기 전에 새 테스트가 실제로 "00123 → 123" 으로 **깨지는 것을 먼저 확인**하고 커밋 메시지나 회차 노트에 적을 것.
  - 커밋은 한국어, `fix(external): …` 꼴. Claude 공동 저자 표기 없음.

- 차선 후보: 워크북 GET 응답(`GET /api/v1/workbooks/{id}`)에 넘겨받은 출처(`handoff` 필드)를 실어 편집기가 열릴 때마다 `GET /api/v1/workbooks/{id}/handoff` 404 요청 하나를 덜 보낸다 — `internal/httpapi/handoff.go` 의 출처 조회를 워크북 핸들러에서 함께 부르고 `web/src/lib/handoff.ts` 의 `fetchHandoffOrigin` 을 응답 필드 우선으로. 가치 2 / 위험 1 / S. 1순위가 성립하지 않을 때(예: 순환 import 가 풀리지 않을 때)만.
