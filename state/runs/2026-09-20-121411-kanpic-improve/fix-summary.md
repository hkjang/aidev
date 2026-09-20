# 수리 요약 (수리 시도 1)

- 문제: `parseXLSXValue` 의 CellTypeUnset 갈래가 `delimited.Number` 를 불러 `tooLongToHoldExactly` 가 엑셀이 17자리로 적은 실수(`2.2000000000000002`, `37.019999999999996`, `1234567890123450`)에도 걸려 수 칸이 글자로 들어왔다 — 지적 그대로이며, 새 테스트 표로 HEAD 에서 네 값 모두 글자로 나오는 것을 재현했다.
- 고침(b52a394): Unset 갈래를 예전 가드(`value == "" || HasSignificantLeadingZero`) + `formula.DecimalNumber` 로 되돌리고, 그 가드는 `delimited.HasSignificantLeadingZero` 로 export 해 사본을 두지 않았다. 왜 CSV 의 자를 쓰지 않는지(엑셀 raw 값은 이미 실수라 17자리 기록이 흔함)를 주석과 테스트 설명에 적었다.
- `scalar_test.go` 의 `TestParseXLSXValueUntypedCellsFollowTheFileRule` 은 20자리를 글자로 못 박던 줄을 수로 바꾸고 엑셀식 17자리 값 세 개를 더했다. `delimited.Number`·IMPORTDATA·CSV 업로드 쪽은 손대지 않았다.
- 검증: `gofmt -l internal/` 빈 출력, `go vet ./...`, `go build ./...`, `go test ./...` 18개 패키지 전부 ok. 고치기 전 같은 테스트가 네 줄에서 실패했다.
