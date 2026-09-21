- 과제: XLSX에서 생략된 행의 실제 r 좌표를 슬라이드 출처에 반영 (가치 3 / 위험 2 / 작업량 S)
- 왜: workbook.go의 worksheet.Rows.Reference는 XML에서 읽지만 onScreen은 placement.rows에 배열 index만 넣어, 행 1·5·9만 저장된 시트를 A1:B3에서 가져왔다고 표시한다(코드 경로로 확인; 신규 재현 테스트는 아직 실행하지 않음). 실제 행 번호를 기존 placement에 전달하면 사용자가 슬라이드 수치의 원본 행을 정확히 찾을 수 있다.
- 수용 기준:
  1) 실제 XLSX ZIP에 row r="1", r="5", r="9"와 해당 A/B 셀을 넣어 Read→deck.ParseSource를 통과시키면 데이터 슬라이드의 Sources[].Locator는 `분기 실적!A1:B9`이고, 라벨·수치·행 순서는 그대로다. 숫자 두 열의 차트와 세 열의 표 모두 확인한다.
  2) 생략 행+명시적 빈 행+hidden 행/열을 함께 넣어도 마지막 보이는 행/열의 원래 좌표를 인용한다. 예: r=1 머리글, r=5 hidden 데이터, r=7 빈 행, r=9 표시 데이터, B열 숨김·A/C열 유지 → A1:C9, 숨긴 행/열 경고는 기존 규칙 그대로. 9개 이상 본문 행으로 이어지는 슬라이드 각각의 끝 행도 실제 좌표여야 한다(기존 maximumRows=8).
  3) 행 좌표만 큰 작은 ZIP(예: 마지막 r=1048576)도 빈 행을 채우지 않고 읽힌다. r 누락·0·음수·문자열·정수 오버플로는 기존 배열 index로 대체하여 panic/새 오류 없이 기존 관용을 유지한다. 연속 r과 r 없는 기존 XLSX, CSV, 숨김, rich text 회귀 테스트가 통과한다. 테스트는 함수 결과만 대역으로 넣지 말고 ZIP→Read→ParseSource의 Sources·Items/Blocks를 검사한다.
- 건드릴 파일:
  - server/internal/docs/workbook.go: worksheet.Rows.Reference, onScreen — 표시 행의 placement.rows를 만들 때 유효한 양의 행 번호를 0-based로 변환. 작은 보조 함수가 필요하면 이 파일에 한정한다. r은 TrimSpace 뒤 strconv.Atoi로 파싱하고 1..1048576 범위만 수용하는 것으로 이번 계약을 고정한다. 무효 r은 현재 index로 대체하며, cell.Reference에서 행 번호 추론·정렬·중복 해소는 범위 밖이다.
  - server/internal/docs/sheetrows_test.go (신규 권장): 위 ZIP 기반 회귀. 실제 열어 본 sheetconcealed_test.go의 concealedBook/textCell/numberCell 재사용 가능. sheetrichtext_test.go의 Read→deck.ParseSource→Item.Number/Block.Rows 검증 방식 참고.
  - server/internal/docs/sheetconcealed_test.go: 필요하면 생략 좌표와 숨김 조합 회귀 추가. 기존 사례의 출처 기대값을 바꿔 통과시키지 않는다.
  - 읽기 참조: tables.go의 placement.row, trimmed, writeSheet의 source 클로저, rangeOf; deck/source.go의 SourceSlide.Sources 및 SourceCitation.Locator. 이들은 이미 kept 인덱스를 placement로 변환하므로 원칙적으로 구현 수정 불필요.
- 검증 명령: 저장소 루트에서 `cd server && go test ./internal/docs` → `cd server && go test -race ./...` → `cd server && go vet ./...`; 각각 루트에서 시작하는 독립 명령이다. 마지막 `git diff --check`. 정찰에서 전체 race와 vet는 실제 통과(캐시 포함); 새 회귀는 구현자가 먼저 실패 확인해야 한다. 웹 변경이 없어 npm 설치·빌드는 불필요.
- 위험과 피할 것: gridOf는 XML row 한 개당 grid 한 줄이라는 계약을 지킨다. r 크기만큼 빈 행을 할당하거나 onScreen의 hidden 매칭 index를 바꾸지 말 것. rangeOf의 시작 A1·누적 범위 정책은 유지하며 정확한 시작 셀 도입으로 범위를 키우지 않는다. 숫자 파서(amountOf/parseNumber/parseBareNumber/chartFields), 렌더러 부호·축, 숨긴 열 범위 전개, auth/migrations/workflows, VERSION·릴리즈 노트는 건드리지 않는다. 유효하지 않은 r의 대체 정책은 이번 과제의 제안이며 실제 Excel/LibreOffice 파일의 비정상 r 사례는 미확인이다.
- 차선 후보: 숨긴 앞 열 뒤의 AAA 이상 열을 출처에서 올바르게 표기 — tables.go:columnLetter는 두 글자까지만 계산하여 index=702를 [A로 쓴다. 이 함수만 일반 26진 열 이름 변환으로 바꾸고, A..ZZ를 숨겨 AAA의 값이 보이는 작은 XLSX ZIP→Read→ParseSource에서 Locator A1:AAA2 및 기존 A/Z/AA/ZZ를 검증한다(가치 2 / 위험 1 / 작업량 S). 1순위가 이미 해결되어 있거나 재현 불가능할 때만 선택.

실행 계획·추정: 실패 회귀 10분, onScreen의 좌표 매핑 10분, 숨김/이어쓰기/무효 좌표 회귀 10분, 전체 검증 5분, 예비 10분으로 총 45분. 기존 placement만 채우는 방안을 선택했으며 gridOf를 밀집 행으로 확장하는 방안은 메모리 위험 때문에 제외한다. 예비 시간을 넘기면 범위를 늘리지 말고 검증 결과와 미완료를 남긴다.
스킬 제약: 요청한 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 현재 도구 목록에 Skill/skills.list/skills.read가 없고 로컬 .codex/.claude 및 실행 home 검색에서도 찾지 못했다. 해당 절차·반환 형식은 미확인이라 적용했다고 주장하지 않으며, 사용자 지정 형식과 자체 추정·대안 비교를 따른다.
