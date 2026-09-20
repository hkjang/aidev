# 회차 노트 2026-09-20-121411-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:14] base pinned — main@94ea15d
- [러너 12:14] autonomy release — 

## 정찰 노트
- 선택: IMPORTDATA(`internal/external/fetcher.go:424`)가 `formula.DecimalNumber` 만 불러 "00123"→123, 스무 자리 번호→실수로 바꾸는데 업로드 `parseScalar` 는 두 가드로 글자로 둔다 — 같은 파일이 두 문에서 다른 표가 되는 꼴이고, 이 저장소에서 통과해 온 변경은 모두 "두 문을 일치시키는" 종류였다. 넓히는 쪽(1,200·12% 읽기)은 코드 주석이 의도로 막아 둔 것이라 제쳤고, 워크북 GET 출처 실기는 가치가 낮아 차선으로.
- 확신 없는 곳: (1) 업로드 경로가 CSV 칸을 trim 하는지 안 봤다 — 과제서에 미확인으로 적음. (2) 공용 함수를 `internal/delimited` 에 두면 delimited→formula 의존이 생기는데 순환은 아니라고 확인했지만 취향 문제일 수 있어 `formula.FileNumber` 대안을 적어 둠. (3) 고치기 전 실제 실행으로 123 이 나오는 것은 코드로만 확인(정규식 `\d+`), 테스트로 못 박는 것은 구현자 몫.
- 조심할 것: `DecimalNumber`/`decimalText`/`spreadsheetNumber.ts`/`testdata/numeric-text.json` 은 건드리지 말 것(셈 규칙이지 파일 규칙이 아니다). `unguardDelimitedValue` 를 IMPORTDATA 에 끌어오지 말 것. 새 테스트가 고치기 전에 깨지는 것을 먼저 확인할 것.
- 프로필은 없었으므로 새로 썼다(profile.md).
- [러너 12:18] scout done — IMPORTDATA 가 업로드 가져오기와 같은 규칙으로 "수처럼 생긴 글자"를 지킨다 — 앞자리 0 과 열여섯 자리 넘

## 구현 노트
- 파일에서 온 값을 칸에 담는 자를 새 `delimited.Number` 하나로 두고 `hasSignificantLeadingZero`·`tooLongToHoldExactly`·`plainNumber` 를 importexport 에서 옮겨, 업로드 `parseScalar`·XLSX 형식 없는 칸·IMPORTDATA `parseCSV` 가 모두 그것을 부른다. 고치기 전 "00123"→123 으로 깨지는 것과 fetcher 한 줄 되돌리면 다시 깨지는 것을 확인. 커밋 39588dd(수정), 12ced8b(XLSX 테스트).
- 확신 없는 곳: XLSX 형식 없는(Unset) 칸은 예전엔 앞자리 0 만 걸렀는데 이제 열여섯 자리 넘는 수도 글자로 남는다 — 엑셀 자신은 그런 수를 지수로 적어 영향이 없고 openpyxl 의 int 같은 경우만 바뀌는데, 실제 그런 xlsx 파일로는 돌려 보지 않았다(단위 테스트로만 못 박음). 과제서가 그 갈래도 같은 함수로 하라고 했다.
- 일부러 하지 않은 것: `CellTypeNumber` 갈래는 `formula.DecimalNumber` 그대로(엑셀이 이미 실수로 저장한 값 — 주석 한 줄). trim 규칙은 맞추지 않음(업로드 " 12" 는 글자, IMPORTDATA 는 수 12 — 과제서의 미확인 항목을 확인만 하고 ideas.json 에 적음). "true"/"false" 도 업로드는 불리언, IMPORTDATA 는 글자 — 새 보류 항목으로. `DecimalNumber`·`decimalText`·`spreadsheetNumber.ts`·`numeric-text.json`·`unguardDelimitedValue` 는 손대지 않음. 웹·문서·PDF 도 손대지 않음(가이드가 IMPORTDATA 의 수 규칙을 말한 적 없음).
- 다음 역할이 조심할 것: 새 테스트는 모두 DB 없이 돈다(`go test ./...` 전체 통과 확인). `internal/external` 테스트가 이제 `internal/importexport` 를 import 한다 — 순환 없음(`go list -deps` 로 확인). npm 검사는 돌리지 않았다(Go 만 바뀜).
- [러너 12:23] brief accepted — 채택 — 근거가 코드와 정확히 맞았고(정규식 `\d+`, 테스트로 123 재현), 건드릴 파일·검증 명령·피할 것 모두 그대로 따�
- [러너 12:23] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: fetcher 한 줄을 되돌려 새 IMPORTDATA 테스트가 세 칸에서 실제로 깨지는 것, gofmt·vet·delimited/external/importexport/formula 테스트 통과, delimited→formula 비순환. 보안·법무 차단 사유 없음.
- 거절 사유(구현 노트가 스스로 의심한 자리): `internal/importexport/service.go:1143` — XLSX 형식 없는 칸에 tooLongToHoldExactly 가 걸려 엑셀이 17자리로 적은 실수(`<v>2.2000000000000002</v>`, `37.019999999999996`, `1234567890123450`)가 글자로 들어온다. 손으로 만든 xlsx 로 main(수) vs HEAD(글자) 를 재현했다. "엑셀은 지수로 적는다" 는 20자리 정수에만 맞다.
- 수리가 먼저 볼 것: parseXLSXValue 의 Unset 갈래를 예전 가드(빈값·앞자리 0)+formula.DecimalNumber 로 되돌리고 `scalar_test.go:140` 의 20자리 기대값을 고칠 것. delimited.Number 와 IMPORTDATA 쪽은 그대로.
- 못 본 것: 진짜 Excel 이 저장한 파일로는 돌리지 못했다(엑셀의 17자리 기록은 알려진 동작으로 판단). npm 검사는 Go 만 바뀌어 생략.
- 다음 회차·릴리즈 노트: IMPORTDATA 가 파이썬 repr 실수(0.30000000000000004)도 글자로 두게 된다 — 업로드와 같아지는 것이지만 원격 CSV 로 SUM 하던 사용자에겐 동작 변경.
- [러너 12:27] review rejected — 리뷰 거절: internal/importexport/service.go:1143 XLSX 형식 없는(Unset) 칸에 delimited.Number 를 쓰면서 tooLongToHoldExactly 가 엑셀이 저장한 수 칸에도 걸린다. 엑셀은 이

## 수리 노트
- 지적 중 맞았던 것: 전부. 엑셀식 17자리 값(`2.2000000000000002`·`37.019999999999996`·`1234567890123450`)을 테스트 표에 넣어 HEAD 의 `parseXLSXValue(…, Unset)` 이 네 값을 글자로 내는 것을 재현했다. 구현 노트의 "엑셀은 지수로 적는다" 는 20자리 정수에만 맞았다.
- 고친 방법(b52a394): Unset 갈래를 `value == "" || delimited.HasSignificantLeadingZero(value)` + `formula.DecimalNumber` 로 되돌림. 가드는 `delimited` 에서 export 해 importexport 에 사본을 두지 않았다. `delimited.Number`·IMPORTDATA·CSV 업로드 쪽은 그대로.
- 검증: gofmt·vet·build·`go test ./...` 18개 패키지 ok. 고치기 전 새 테스트가 네 줄에서 깨졌다.
- 여전히 확신 없는 곳: openpyxl 이 int 로 적은 20자리 정수는 Unset 칸에서 다시 실수(1.2345678901234567e19)로 뭉개진다 — main 과 같은 동작이지만 CSV 업로드와는 다르다. XLSX 는 엑셀 자신이 정확히 담지 못하는 값이라 CSV 의 "적힌 대로" 약속을 할 수 없다고 보고 그대로 두었다. 진짜 Excel 이 저장한 파일로는 여전히 돌려 보지 못했다.
- [러너 12:29] repair done — # 수리 요약 (수리 시도 1)  - 문제: `parseXLSXValue` 의 CellTypeUnset 갈래가 `delimited.Number` 를 불러 `tooLongToHoldExactly` 가 엑셀이 17자리로 적은 실수(`2.20000000000000

## 비평 노트
- 확인한 것: 수리 후 세 커밋 전체를 다시 읽고 fetcher 한 줄(`delimited.Number`→`formula.DecimalNumber`)과 XLSX Unset 가드를 각각 되돌려 새 테스트가 실제로 깨지는 것(IMPORTDATA 3칸, XLSX 4값)을 재현했다. gofmt·vet·build·`go test ./...` 전체 통과, delimited→formula 비순환. XLSX Unset 갈래는 main 과 동작이 같다(가드 본문 동일, 호출처만 이동).
- 못 본 것: 진짜 Excel 이 저장한 xlsx 로는 돌리지 않았다. npm 검사는 Go 만 바뀌어 생략.
- 승인이어도 남는 우려(릴리즈 노트): IMPORTDATA 원격 CSV 의 파이썬 repr 실수(0.30000000000000004)와 열여섯 자리 넘는 정수(1000000000000000000 처럼 실수로 정확히 담기는 것 포함)가 이제 글자로 들어와 SUM 에서 빠진다 — 업로드와 같아지는 것이지만 사용자에겐 동작 변경.
- 다음 회차: 두 문의 남은 불일치는 trim(IMPORTDATA 는 TrimSpace 후 판정, 업로드는 " 12" 를 글자로)과 "true"/"false"(업로드만 불리언). delimited 가 formula 전체를 끌어오게 된 것은 취향 문제로 두었다.
- 보안·법무 차단 사유 없음: 신뢰할 수 없는 입력은 RE2 정규식·ParseFloat 만 지나고 서버 측 요청·권한·개인정보 처리 변화 없음.
- [러너 12:31] review approved — 리뷰 승인 (risk=low)
- [러너 12:31] pr created — https://github.com/hkjang/kanpic/pull/27
- [러너 12:38] ci passed — 검사 2개 모두 success
- [러너 12:38] merge done — b52a394
- [러너 12:50] release published — v0.251.0
- [러너 12:52] assets verified — v0.251.0 자산 2개 (이전 v0.250.0: 2)
