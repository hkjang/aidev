# 수리 요약 (커밋 2e3a62e)

- P1(workbook.go:273) 맞음. `ulimit -v 3GB` 아래에서 `<col min="3" max="4294967295" hidden="1"/>` 6셀 시트를 `Read()` 하니 `onScreen` 에서 `fatal error: out of memory`(1.7GB 사용 중) 로 재현. 고침: 범위를 map 으로 펼치지 않고 `[][2]int` 범위 목록을 그대로 두고 그리드 실제 폭(셀이 있는 열까지) 안에서만 열마다 포함 여부를 묻는다 — 할당은 XML 의 `<col>` 개수와 그리드 폭에만 비례. 회귀 테스트 `TestAColumnRangeHiddenToTheEdgeOfTheSheetCostsNothing`(max=4294967295 그대로) 가 고치기 전엔 OOM, 후엔 통과.
- P2(workbook.go:136, sheetconcealed_test.go:116) 맞음. 숨긴 열을 잘라낸 뒤 열 개수로만 `A1:B3` 을 쓰니 실제 D열이 B열로 인용됐다. 고침: `onScreen` 이 남긴 행·열이 원본 시트의 몇 번째였는지 `placement` 로 돌려주고, `writeSheet` 가 `trimmed()`(빈 줄을 뺀 뒤 어느 줄이 남았는지 함께 돌려주는 `trimGrid`) 와 합쳐 각 슬라이드 마지막 행·열의 **원본 좌표** 로 `A1:<열><행>` 을 쓴다. 잘못된 단언 `!A1:B3` → `분기 실적!A1:D3` 으로 고쳤고, 숨긴 행 테스트(`A1:C7`)·행+열 테스트(`A1:D2`) 에도 출처 단언을 더했다. 세 단언 모두 고치기 전 코드로 되돌리면 실패함을 확인.
- 부수 효과 하나: 이어지는 슬라이드(`(계속)`) 의 출처가 조각 길이(`A1:C5`) 가 아니라 그 조각의 마지막 원본 행(`A1:C13`) 까지로 바뀐다 — 같은 규칙의 결과이며 기존 테스트는 출처를 단언하지 않아 영향 없음. 범위 시작은 이전과 같이 `A1` 로 두었다(선행 숨긴/빈 행이 있어도).
- 검증: `make test` 전체(go test -race 25개 패키지 · go vet · npm typecheck+build) 통과. `writeSheet` 시그니처에 `placement` 인자가 붙어 `longtable_test.go` 호출 3곳에 `placement{}` 를 넣었다(단언 변경 없음).
