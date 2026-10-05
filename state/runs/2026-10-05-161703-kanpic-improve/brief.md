- 과제: IMPORTDATA 가 UTF-8 이 아닌 본문을 깨진 바이트째로 칸에 넣는다 — 업로드 문은 같은 바이트를 거절한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `internal/delimited` 패키지 주석이 "두 문(업로더와 IMPORTDATA)이 들어가고, 같은 파일은 어느 문으로 들어오든 같게 읽혀야 한다"고 못 박았는데, 업로드 쪽 `importexport.parseDelimited` 는 `delimited.ToUTF8` 뒤 `utf8.Valid` 로 거절하고 IMPORTDATA 쪽 `external.parseCSV` 에는 그 검사가 없다. 그래서 CP949/latin-1 본문이 IMPORTDATA 로는 통과해 **UTF-8 이 아닌 바이트가 그대로 칸 값**이 되고, 이 값은 JSON 응답에서 U+FFFD 로 바뀌거나 Postgres text 컬럼이 거절할 수 있다(DB 거절은 이번 미확인) — 사용자가 보는 것은 "깨진 글자가 든 표" 이지 "읽을 수 없는 파일" 이 아니다.
- 재현(이번 정찰이 실제로 돌려 본 것): `internal/external` 에 임시 테스트를 넣어 같은 바이트열 `{0xC0,0xCC,0xB8,0xA7, ',', 0xB8,0xDE,0xB8,0xF0, '\n','1',',','2','\n'}`(CP949 `이름,메모\n1,2\n`)을 두 문에 넣은 결과:
  - `parseCSV(string(raw))` → `err=<nil> rows=2 cols=2`, `cell[0]="\xc0̸\xa7" utf8.Valid=false`, `cell[1]="\xb8޸\xf0" utf8.Valid=false`, `cell[2]=1`, `cell[3]=2`
  - `importexport.Parse("a.csv", raw, 1<<20)` → `err=CSV must be UTF-8 encoded`
  (임시 테스트 파일은 지웠고 작업 트리는 깨끗하다)
- 수용 기준:
  1) `delimited.ToUTF8` 를 거친 뒤에도 `utf8.Valid` 가 아닌 본문은 IMPORTDATA 가 표를 내지 않고 `#VALUE!` + 한국어 문구로 거절한다(이 파일의 다른 오류 문구와 같은 꼴 — 예: `"CSV 가 UTF-8 이 아닙니다"`). 코드 선택은 같은 함수의 기존 읽기 실패와 맞춰 `#VALUE!` 로 둔다.
  2) 지금 통과하는 본문은 **값까지 그대로**다: 맨 UTF-8, UTF-8 BOM, UTF-16LE BOM(`TestImportDataReadsACSVThatSaysItIsUTF16`), 숫자/불리언 판정(`TestImportDataKeepsNumbersThatUploadKeepsAsText`, `TestImportDataReadsBooleansLikeUpload`), 구분자 판정(`TestImportDataReadsWhicheverSeparatorCutsTheFile`) 모두 기존 그대로 초록.
  3) 새 테스트가 **같은 바이트열 하나**를 `parseCSV` 와 프로덕션 `importexport.Parse` 양쪽에 넣어 둘 다 거절하는 것을 보인다(업로드 쪽은 에러 메시지, IMPORTDATA 쪽은 `Err.Code=="#VALUE!"` 와 `Rows==0`). 손으로 만든 대역 없이 두 패키지의 실제 입구를 쓴다 — `internal/external` 테스트가 `kanpic/internal/importexport` 를 import 해도 순환이 없음을 이번에 컴파일로 확인했다.
- 건드릴 파일 (프로덕션 1개):
  - `internal/external/fetcher.go:parseCSV` — `body = string(delimited.ToUTF8([]byte(body)))` 바로 뒤에 `if !utf8.Valid([]byte(body))`(또는 `utf8.ValidString(body)`) 검사를 넣고 `formula.ExternalResult{Err: &formula.Error{Code: "#VALUE!", Message: ...}}` 를 돌린다. `unicode/utf8` 은 이 파일이 이미 import 하고 있다(line 23, `short` 가 쓴다) — 새 의존성 없음. 왜 추측하지 않고 거절하는지를 저장소 관례대로 한국어 주석으로 적는다.
  - `internal/external/fetcher_test.go` — 위 3) 의 두 문 비교 테스트 하나. 기존 `TestImportData...Like Upload` 이름 관례를 따른다.
- 검증 명령:
  - `go test ./internal/external -run TestImportData -v` (새 테스트가 수정 전 빨강 → 수정 후 초록인 것을 먼저 확인)
  - `go test ./internal/external ./internal/importexport ./internal/delimited -count=1`
  - `go test ./...`
  - `go vet ./...`, `go build ./...`, `gofmt -l ./cmd ./internal ./pkg`
  - `./scripts/check-release-docs.sh`, `./scripts/check-commit-identities.sh HEAD`
  - 되돌리기 확인: 새로 넣은 검사 한 줄만 지워 새 테스트만 다시 빨강이 되는지 본다.
- 위험과 피할 것:
  - **인코딩을 추측하지 말 것.** `ToUTF8` 는 "표시가 없으면 추측하지 않는다"를 문서로 못 박았다. CP949/latin-1 변환이나 `golang.org/x/text` 의존성 추가는 이번 과제 밖이고, 추가하면 두 문이 또 갈린다.
  - `internal/delimited/*` 와 `internal/importexport/service.go` 는 손대지 말 것 — 업로드 쪽은 이미 올바르다. 고치는 곳은 뒤처진 한 문뿐이다.
  - `parseCSV` 의 `reader.LazyQuotes = true` 는 **건드리지 말 것.** 업로드 문은 `LazyQuotes` 가 false 라 또 하나의 비대칭이지만, 어느 쪽으로 맞출지는 사용자에게 보이는 계약 재결정이다(지금 읽히는 파일이 거절되거나, 망가진 파일이 조용히 받아들여진다). 별도 과제로 남긴다.
  - `fetch`/allow-list/SSRF·리다이렉트·크기 상한 경로(`fetcher.go` 앞부분), 캐시(`TestPolicyIsNotCachedInEitherDirection`, `TestCacheSweepsExpiredAndStaysBounded`)는 손대지 않는다. 거절도 캐시되는지를 바꾸지 말 것 — 지금 실패 캐시 동작(`TestRemoteFailuresAreKeptOnlyBriefly`)을 그대로 두면 된다.
  - 오류 문구를 `#N/A` 로 바꾸고 싶어지면 멈출 것 — `ADMIN_GUIDE.md:785` 가 외부 호출 오류를 `#VALUE!` 로 적어 두어 문서와 코드가 이미 어긋나 있고(열세 회차 연속 보류 항목), 그 정리는 이번 과제가 아니다. 같은 함수의 기존 읽기 실패와 **같은 코드**를 쓰는 것만 지키면 문서 변경이 필요 없다.
  - docs/PDF 는 건드릴 필요 없다(사용자 가이드에 적힌 규칙이 바뀌지 않는다). 보호 경로 auth/apikey/migrations/.github/workflows 는 무관.
- 차선 후보: cron 필드가 `+5`·`-0` 같은 부호 붙은 수를 받아들인다 — `internal/automation/schedule.go:cronValue` 의 `strconv.Atoi` 가 부호를 허용해 `"+5 0 * * *"` 가 매일 00:05 로 돈다(2026-10-03 정찰이 확인, 이번 미확인). 프로덕션 1파일, `ErrInvalid` 래핑 유지, 저장된 일정을 깨뜨릴 수 있으니 단독으로만.
