- 과제: 시트가 여러 장으로 이어질 때 뒷장의 `!source` 인용을 그 장이 실제로 보여 주는 행 범위로 쓰기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `writeSheet`(server/internal/docs/tables.go:109)의 `source(last)` 클로저와 `rangeOf`(tables.go:224)가 범위를 언제나 `A1:` 에서 시작하므로, 20행 시트가 세 장으로 이어질 때 세 장이 각각 `분기 실적!A1:B9` · `분기 실적!A1:B17` · `분기 실적!A1:B21` 을 인용한다(이번 정찰에서 `writeSheet` 를 직접 불러 출력 확인). 뒷장의 인용은 그 장에 없는 앞장의 행까지 통째로 삼켜, 출처를 따라간 사람이 "이 장의 숫자가 시트 어디에 있나" 를 알 수 없고 장이 늘어날수록 모든 장이 사실상 같은 곳을 가리킨다. 고치면 이어지는 장들이 시트를 겹치지 않고 이어서 가리킨다(`A1:B9` → `A10:B17` → `A18:B21`).
- 수용 기준:
  1) 20행 2열 시트(헤더 1줄 + 본문 20줄)가 슬라이드 3장이 될 때 `!source` 가 차례로 `분기 실적!A1:B9`, `분기 실적!A10:B17`, `분기 실적!A18:B21` 이다 — 끝 행은 지금과 같고 시작 행만 이어진다.
  2) 한 장에 다 들어가는 시트(기존 테스트 전부: `docs_test.go:17,30,82`, `sheetcells_test.go:61`, `sheetconcealed_test.go:87,123,235`, `sheetrichtext_test.go:114`, `sheetwidth_test.go:139,187,201`)의 인용은 한 글자도 바뀌지 않는다 — 첫 장은 언제나 `A1` 에서 시작한다.
  3) 숨긴 행이 있는 시트에서도 시작 행이 `placement.row` 를 거친 **시트의 실제 행 번호**다 — 앞 8줄 뒤에 숨긴 행이 섞여 있으면 둘째 장의 시작 행은 배열 index 가 아니라 그 시트 좌표다. 테스트는 실제 xlsx ZIP → `Read()` → `deck.ParseSource` 까지 통과시켜(`sheetconcealed_test.go` 의 기존 ZIP 조립 헬퍼 재사용) 증명할 것.
- 건드릴 파일 (프로덕션 2, 테스트 1):
  - `server/internal/docs/tables.go:224 rangeOf` — 시그니처에 첫 행을 더해 `A%d:%s%d` 로 쓰기(`rangeOf(sheet, firstRow, column, lastRow int)`). 음수 방어는 지금처럼 `max(row,0)` 유지. 주석("A1:C9 처럼")도 이어지는 장을 설명하게 한 줄 고칠 것 — 이 저장소는 주석에 왜 그렇게 읽는지를 산문으로 적는다.
  - `server/internal/docs/tables.go:123 source` 클로저 — `func(first, last int) string` 으로 바꿔 `from.row(kept[first])` 를 함께 넘긴다. 첫 장은 `source(0, len(body))`(grid 0 = 헤더 행 = 시트 1행). 이어지는 장은 루프에서 `last` 를 더하기 **전에** 그 조각의 첫 grid index 가 `last+1` 임을 쓰고(`start := last + 1`), `last += len(piece)` 뒤에 `source(start, last)`. (검산: 첫 장 last=8 → kept[8] → 9행, 둘째 조각 all[8]=rows[9] → 10행 → `A10:B17`.)
  - `server/internal/docs/longtable_test.go` — 위 수용 기준 1·3의 테스트를 먼저 넣어 red 를 확인한 뒤 고칠 것. 이 파일에는 지금 `!source` 단언이 하나도 없다.
- 검증 명령:
  - `cd server && go test ./internal/docs` (약 3초)
  - `cd server && go test -race ./... && go vet ./... && gofmt -l internal/docs`
  - `git diff --check`
  - 웹·API·문법 문서 변경이 없으면 `make test` 의 웹 단계는 건너뛰어도 된다(지난 두 회차의 관례).
- 위험과 피할 것:
  - `rangeOf` 호출자는 `tables.go:125` 하나뿐이다(정찰에서 grep 확인). `internal/golden/testdata` 에 `A1:` 문자열은 없고 `docs/USER_GUIDE.md:278` 의 예시는 한 장짜리라 그대로 맞다 — 문서는 건드리지 말 것. 릴리즈 노트(과거 기록)는 절대 고치지 말 것.
  - 숫자 파서(`allNumeric`·`amountOf`·`bareFigure`)와 `deck` 쪽 파서(`parseNumber`·`chartFields`)는 한 글자도 건드리지 말 것 — 2026-09-09 에 두 번 반려된 자리다.
  - 모양 판정(`chart := columns==2 && allNumeric(body,1)` + `carried` AND)과 `writeBody` 는 2026-09-26 에 고친 자리다. 인용만 고치고 판정·경고 문구·`(계속)` 접미는 그대로 둘 것.
  - 표는 헤더 행을 뒷장에도 다시 쓰지만 인용 범위에는 1행을 넣지 않는다(스프레드시트 범위는 떨어진 두 구간을 못 쓰고, 그 장의 숫자가 어디서 왔는지가 인용의 목적이다). 이 판단을 커밋 메시지·주석에 한 줄로 남길 것.
  - `kept` 는 `trimmed(rows)` 가 돌려주는 map 이므로 `first` 도 반드시 `kept[...]` → `from.row(...)` 두 단계를 거칠 것. 한쪽만 거치면 숨긴 행·생략 행 시트에서 좌표가 어긋난다(수용 기준 3이 무는 자리).
  - 버전·릴리즈 노트·배포 매니페스트는 손대지 않는다(현재 VERSION 1.69.47).
- 차선 후보: `tables.go:180` 이 `writer.go:132 continued()` 를 쓰지 않고 `" (계속)"` 을 직접 붙여, 시트 이름이 이미 "실적 (계속)" 이면 "실적 (계속) (계속)" 이 되는 것 — `continued(heading)` 로 바꾸고 `longtable_test.go` 에 한 케이스를 더한다(가치 2 / 위험 1 / 작업량 S, 파일 2개).
