# 과제서 2026-09-20 — visitflow

- 과제: CSV·XLSX 가져오기에서 엑셀이 숫자로 저장한 휴대전화(앞자리 0 탈락·지수 표기)를 복원 (가치 3 / 위험 1 / 작업량 S)
- 왜: 엑셀에서 휴대전화 칸을 숫자로 입력하면 `01012345678`이 `1012345678`(앞 0 탈락, 10자리)로 저장되고, 숫자 서식에 따라 `1.01E+09` 같은 지수 표기로도 나온다. 이번 정찰에서 excelize v2.9.1 `GetRows`로 실제 재현했다: 숫자 셀 `1012345678` → `"1012345678"`, 지수 서식 셀 → `"1.01E+09"`, 사용자 지정 서식 `000-0000-0000` → `"010-1234-5678"`(이건 이미 정상), 문자열 셀 → `"01012345678"`. 현재 `visitorInputsFromRows`(internal/app/import.go:207)는 `len(normalizePhone(phone)) < 7`만 검사하므로 10자리 `1012345678`은 **경고 없이** 통과해 잘못된 번호(`101-****-5678`)로 저장되고 알림 SMS가 엉뚱한 곳으로 간다. 지수 표기는 5자리로 줄어 경고는 뜨지만 사용자는 원인을 모른다. 같은 현상은 CSV에서도 난다(엑셀이 숫자 열을 CSV로 저장해도 앞 0이 빠진다).
- 수용 기준:
  1) 전화 셀이 숫자만 8~10자리이고 `0`으로 시작하지 않으면 앞에 `0`을 붙여 `VisitorInput.Phone`에 넣는다(`1012345678` → `01012345678`, `3112345678` → `03112345678`, `21234567` → `021234567`). 이미 `0`으로 시작하거나 하이픈·공백이 있는 값(`010-1234-5678`, `01012345678`)은 바이트 그대로 유지된다.
  2) 지수 표기(`1.012345678E+09`, `1.012345678e9` 등 `^\d+(\.\d+)?[eE][+-]?\d+$`)는 `strconv.ParseFloat` 뒤 정수이고 1e15 미만이면 정수 문자열로 되돌린 다음 1)의 규칙을 적용한다. 정밀도가 잘린 `1.01E+09`는 `1010000000` → `01010000000`이 되므로 복원해서는 안 된다 — 지수 표기의 가수 유효 자릿수(소수점 이하 포함)가 9자리 이상일 때만 복원하고, 아니면 값을 건드리지 않아 종전처럼 "이름 또는 휴대전화를 확인하세요" 경고가 남게 한다.
  3) 복원은 CSV·XLSX 공통 경로인 `visitorInputsFromRows`의 `phone` 셀에서만 일어난다(`readVisitorImportRows`의 XLSX 분기에 `RawCellValue` 옵션을 넣지 말 것 — 다른 열의 표시값이 바뀐다). 이름·이메일·회사 등 다른 열은 변하지 않는다.
  4) 테스트가 증명할 것: (a) DB 없이 도는 단위 테스트 — 새 헬퍼(예: `importPhone(value string) string`)의 표 테스트: 위 사례 + 11자리 이상(`21012345678`)·7자리 이하·`+82 10 1234 5678`·빈 값은 무변경, `1.01E+09`는 무변경. (b) 기존 `TestVisitorImportAcceptsExcelExports`(internal/app/import_test.go:60, `VISITFLOW_TEST_DSN` 필요)의 XLSX 블록에 `book.SetCellValue(sheet, cell, 1012345678)`(int)로 넣은 행과 `CustomNumFmt: "0.000000000E+00"` 스타일을 입힌 행을 더해, 미리보기 응답의 `phone`이 `01012345678`이고 그 행에 경고가 없는 것을 확인한다 — 진짜 excelize 출력이 핸들러를 통과하는 경로로 증명(문자열을 직접 넣는 대역 금지). (c) 헬퍼를 항등 함수로 바꾼 변이에서 (a)·(b)가 실제로 실패하는 것을 확인하고 sed로 되돌린다(`git checkout --` 금지 — 과거 두 번 미커밋 편집을 잃었다).
  5) `docs/USER_GUIDE.md`의 "협력사 열 명을 한 번에 신청하기"(151행 부근)에 한 문장 추가: 휴대전화 칸이 숫자 서식이어도 앞 0을 보완한다는 것. PDF 재생성은 하지 않아도 된다(한 문장 변경, md2pdf는 이 저장소 밖 도구).
- 건드릴 파일:
  - `internal/app/import.go:importPhone`(신규, `importConsent` 옆) — 지수 표기 복원 + 앞 0 보완. `visitorInputsFromRows`(import.go:199)에서 `phone := importPhone(cell(row, "phone"))`로 한 곳만 바꾼다. 208행의 `len(normalizePhone(phone)) < 7` 경고와 220행의 `Phone: phone` 저장이 같은 값을 읽도록 변수 하나를 쓴다(운영자 규칙: 같은 값을 읽는 경로가 여럿이면 모두 같은 값을 읽게).
  - `internal/app/import_test.go` — 단위 표 테스트 1개 추가, `TestVisitorImportAcceptsExcelExports` XLSX 블록에 숫자 셀 2행 추가(`excelize.Style{CustomNumFmt: &s}`로 지수 서식).
  - `docs/USER_GUIDE.md` — 한 문장.
- 검증 명령:
  - `gofmt -l internal/ && go vet ./...`
  - `go test ./internal/app -run 'TestImport|TestVisitorImport' -count=1 -v` (DSN이 없으면 `newTestEnv`가 통합 테스트를 skip 한다 — integration_test.go:61 확인 — 그러므로 XLSX 통합 블록은 반드시 아래 DSN 실행으로 증명할 것)
  - `docker run -d --name vf-pg -e POSTGRES_USER=visitflow -e POSTGRES_PASSWORD=visitflow -e POSTGRES_DB=visitflow -p 5432:5432 postgres:16-alpine` 뒤 `VISITFLOW_TEST_DSN='postgres://visitflow:visitflow@127.0.0.1:5432/visitflow?sslmode=disable' go test ./... -count=1` (CI와 동일, internal/app 약 50초)
  - 프런트는 건드리지 않으므로 `cd web && npm ci && npm run lint && npm test && npm run build`는 마지막 확인용으로 한 번.
- 위험과 피할 것:
  - 미머지 브랜치 `origin/auto/2026-09-16-1212`(메일)와 `origin/auto/2026-09-18-0533`(MCP OAuth)가 `admin.go`·`settings.go`·`server.go`·`auth.go`·`keys.go`·`integration_test.go`·`SettingsPage.tsx`·`AdminPage.tsx`·`KeysPage.tsx`·`ADMIN_GUIDE.*`·`API_AND_MCP.md`를 바꾼다. 이 파일들은 손대지 말 것 — 충돌만 낳는다. `import.go`·`import_test.go`·`VisitFormPage.tsx`·`USER_GUIDE.md`는 두 브랜치 모두 건드리지 않는다(확인함).
  - 앞 0 보완 범위를 8~10자리로 좁게 유지할 것. 11자리 이상(국제번호 `821012345678`)이나 7자리 이하는 건드리지 않는다. `normalizePhone`(visits.go:25)은 공용이라 바꾸지 말 것 — 해시(`phone:`+digits) 조회·알림 수신자에 쓰여 기존 데이터와 어긋난다.
  - 지수 표기 복원은 정밀도가 보존된 경우에만. `1.01E+09` 같은 잘린 값을 `01010000000`으로 만들어 저장하면 조용한 오염이 되므로 그대로 두고 경고에 맡긴다. 이것이 실제 출력이 바뀌는 변경인지(효과 없는 수정 금지 규칙) — 바뀐다: 현재 `1012345678`은 경고 없이 그대로 저장된다.
  - 행마다 "앞 0을 보완했습니다" 경고를 넣지 말 것 — 숫자 열이면 100행 전부에 뜬다. 미리보기 화면의 편집 가능한 전화 필드에 복원값이 보이므로 그것으로 충분하다.
  - 변이 되돌리기는 sed 또는 임시 복사본으로. `git checkout -- 파일` 금지.
- 차선 후보: 가져오기 미리보기 응답에 인식된 헤더 행 번호(`headerRow`)와 무시된 열 이름(`ignoredColumns`)을 더하고 `VisitFormPage.tsx:87` 부근의 `importVisitors`에서 "읽지 않은 열: …"을 경고 목록에 붙이기 — `importHeaders`(import.go:125)가 별칭에 없는 열(`핸드폰`·`소속`)을 조용히 버리는 문제. 같은 파일 범위(import.go·import_test.go·VisitFormPage.tsx)이고 미머지 브랜치와 겹치지 않는다.
