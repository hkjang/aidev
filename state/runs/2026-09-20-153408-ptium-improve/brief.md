# 과제서 — 2026-09-20 (ptium)

- 과제: xlsx 가져오기에서 숨긴 행·열(`hidden="1"`)을 표에 넣지 않고, 뺐다고 경고 한 줄로 알리기 (가치 3 / 위험 2 / 작업량 S~M)

- 왜: `server/internal/docs/workbook.go` 의 `worksheet` 구조체(45행)는 `<row>` 의 `hidden` 속성도 `<cols><col hidden="1">` 도 읽지 않아, `gridOf`(231행)가 사용자가 화면에서 숨겨 둔 보조 계산 행·열(필터로 걸러 낸 행, 접어 둔 그룹, 수식이 참조하는 코드 열)을 그대로 표에 넣는다. 그러면 슬라이드 표에 사용자가 보지 못하던 열이 끼어들고, 숫자 보조 열 하나 때문에 `allNumeric` 판정이 바뀌어 표가 차트로 넘어가기도 한다. 숨긴 **시트**는 이미 빼고 경고까지 하므로(`sheetHidden`, 113행·163행) 같은 규칙을 행·열에도 적용하면 "화면에서 보이는 것이 슬라이드가 된다" 가 일관된다.

- 수용 기준:
  1) `<row r="3" hidden="1">` 인 행은 그리드에서 사라지고, 나머지 행은 순서 그대로 남는다(`hidden="true"` 도 같음 — `counts1904` 가 쓰는 `"1"|"true"` 규칙 재사용).
  2) `<cols><col min="2" max="3" hidden="1"/></cols>` 인 열은 그리드의 모든 행에서 그 열이 **제거**되어(빈칸으로 남기지 않음) B·C 를 빼고 A·D 가 붙는다. `min`/`max` 는 1 기준이고 범위이며, 빠진/이상한 값은 무시한다.
  3) 무언가를 뺀 시트에는 `warnings` 에 한 줄 — 예: `<시트 라벨>의 숨긴 행 2개와 열 1개는 가져오지 않았습니다`(행만/열만이면 그것만; `sheetLabel(filename, sheet)` 를 쓰고 `writeSheet` 의 문구 톤을 따를 것). 뺀 것이 없으면 경고도 없다.
  4) 모든 행이 숨겨진 시트는 빈 시트와 같이 슬라이드가 되지 않는다(기존 `len(trimGrid(rows)) >= 2` 경로가 그대로 처리). 통합 문서의 모든 시트가 그렇다면 기존 오류 "이 통합 문서에는 읽을 표가 없습니다" 가 나온다 — 이 문구를 바꾸지 말 것.
  5) 테스트가 증명할 것: (a) 숨긴 행이 표에서 빠지고 경고가 남 (b) 숨긴 열 범위가 빠지고 남은 열이 붙음 (c) 숫자 보조 열을 숨긴 시트가 숨김 처리 뒤 `::table` 로 남는지(또는 그 반대) 한 건 — 즉 분류기 입력이 바뀌는 것을 `Read()` 끝까지 통과한 `Source` 로 확인 (d) 숨긴 것이 없는 시트는 경고 0건, 기존 `sheethidden_test.go`·`sheetcells_test.go` 는 손대지 않고 그대로 통과.

- 건드릴 파일:
  - `server/internal/docs/workbook.go:45 worksheet` — `Rows[].Hidden string \`xml:"hidden,attr"\``, `Cols []struct{ Min, Max int \`xml:"min,attr"\`/\`xml:"max,attr"\`; Hidden string \`xml:"hidden,attr"\` } \`xml:"cols>col"\`` 추가.
  - `server/internal/docs/workbook.go:231 gridOf` — 시그니처는 **그대로 두고**(테스트가 부름) 숨긴 행은 `continue` 로 건너뛰거나, 아니면 새 함수 `shown(sheet worksheet, grid [][]string) (rows [][]string, hiddenRows, hiddenColumns int)` 를 두어 `readWorkbook` 129행의 `rows := gridOf(...)` 바로 뒤에서 적용하는 쪽을 권장 — 열 제거 시 `gridOf` 가 만든 각 `line` 에서 숨긴 열 인덱스(0 기준 = min-1..max-1)를 빼면 된다. 주의: 숨긴 행을 세는 것은 `<row>` 요소 기준이고, 빈 행(셀 없음)이 숨겨진 경우도 개수에 들어가는데 어차피 `trimGrid` 가 지우는 행이므로 **셀이 하나라도 있는 행만** 세어 경고 숫자가 화면과 맞게 할 것.
  - `server/internal/docs/workbook.go:143` 근처 — `writeSheet` 가 실제로 슬라이드를 썼을 때(`count > 0`)만 숨김 경고를 `warnings` 에 붙인다(안 쓴 시트의 경고는 소음).
  - 새 테스트 `server/internal/docs/sheetconcealed_test.go` — 기존 `hiddenBook` 은 `<sheetData>` 만 감싸므로 `<cols>` 를 넣을 수 있게 `hiddenSheet` 에 `cols string` 필드를 더하거나(빈 문자열이면 지금과 동일) 새 빌더를 쓴다. 테스트 이름은 문장형(`TestAHiddenRowIsNotInTheTable`, `TestAHiddenColumnIsCutOutOfEveryRow`, `TestHidingAHelperColumnLeavesTheSheetATable`).
  - 문서: `docs/USER_GUIDE.md` 에 xlsx 가져오기 절이 있으면 "숨긴 시트·행·열은 가져오지 않습니다" 한 줄(있는 문장을 넓히는 정도; 없으면 생략 가능. 미확인).

- 검증 명령:
  - `cd server && go test -race ./internal/docs/ -run 'Hidden|Concealed|Sheet' -v`
  - `cd server && go vet ./... && go test -race ./...` (25개 패키지, DSN 없이 통과해야 함; `golden` 포함)
  - 웹 변경 없음이면 `make test` 의 웹 단계는 건너뛰어도 됨.

- 위험과 피할 것:
  - **숫자 파서(`deck.parseNumber`·`parseBareNumber`·`chartFields`·`docs.amountOf`/`allNumeric`)는 건드리지 말 것** — 2026-09-09 두 번 반려된 자리. 이 과제는 분류기의 *입력*(그리드)만 바꾸지 판정 로직은 그대로다.
  - `gridOf` 의 "참조 없는 셀은 다음 칸" 규칙(228행 주석)을 깨지 말 것 — 열 제거는 `gridOf` 가 위치를 확정한 **뒤**에 하는 것이 안전하다.
  - 숨긴 열을 빈칸으로 두면 `trimGrid` 은 끝 열만 지우므로 가운데 빈 열이 표에 남는다 — 반드시 제거할 것(수용 기준 2).
  - `width="0"`·`collapsed`·`outlineLevel` 은 이번엔 다루지 않음(`hidden` 속성만). 필터 상태(`<autoFilter>`)도 별도 해석하지 않음 — Excel 은 걸러진 행에 `hidden="1"` 을 쓰므로 그것으로 충분.
  - 저장소에 xlsx 픽스처 파일이 없어 골든 회귀는 영향 없어야 하지만 `go test ./...` 로 확인할 것.
  - 커밋 메시지는 영어 한 문장 서술체, 도구 서명 없음. `VERSION`·릴리즈 노트는 손대지 않음.

- 차선 후보: deck 의 세 숫자 파서(`parseNumber`·`parseBareNumber`/`chartFields`·`docs.amountOf`) 계약을 한 표 테스트로 묶기 (가치 2 / 위험 1 / 작업량 S) — 같은 입력 표(`"1,200"`, `"1 200"`, `"120, 118"`, `"-3%"` …)를 셋에 넣어 각자가 무엇을 돌려주는지 고정하는 **테스트만** 추가하고 구현은 바꾸지 않는다(파서 통합·확장 금지).
