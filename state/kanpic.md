# kanpic 자율 개선 기록

## 2026-09-02
- 선택: 엑셀 규칙과 어긋난 서식 함수(PROPER·FIXED·DOLLAR) 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: PROPER 가 빈칸·탭·붙임표·밑줄 뒤만 낱말의 처음으로 보아 `PROPER("o'neil")`→"O'neil", `PROPER("76budget")`→"76budget" 을 내던 것을 엑셀 규칙대로 "글자가 아닌 것 뒤는 모두 낱말의 처음"(unicode.IsLetter)으로 고쳤고, `formatNumber` 가 음수 자릿수를 0 으로 깎아 `FIXED(1234.567,-2)`→"1,235" 를 내던 것을 소수점 왼쪽 반올림("1,200")으로 고쳤다(ROUND 와 답이 어긋나 있었다). 덤으로 ea6fd45 가 잘못 담은 18MB `workbook.test` 실행 파일을 지우고 `.gitignore` 에 `*.test` 를 넣었다. 검증: `gofmt -l`, `go build ./...`, `go test ./...`(전체 통과), `cd web && npm test`(480개 통과) 및 `npm run build`, `scripts/check-release-docs.sh`, `scripts/check-commit-identities.sh`. 커밋 3개(03dda1d, e50238e, c4ae80d).
- 보류 아이디어:
  - CEILING/FLOOR 가 `number*factor < 0` 이면 무조건 #NUM! 을 내는데, 엑셀·구글 시트는 `CEILING(-4.5,2)`=-4, `FLOOR(-4.5,2)`=-6 을 낸다(MROUND 에만 맞는 규칙). 참조 구현으로 확인 후 고칠 것.
  - `internal/external/fetcher.go` 의 응답 캐시가 만료 항목을 지우지 않아 무한히 커진다. 만료 정리 + 상한 필요.
  - 같은 캐시가 실패도 담아, 관리자가 allowed_hosts 를 고쳐도 최대 5분간 "허용되지 않은 호스트" 가 남는다. 설정 변경 시 무효화 필요.
  - `internal/external/fetcher.go` 의 `short()` 가 160바이트에서 자르며 UTF-8 글자를 쪼갠다(한글 오류 메시지가 깨진다).
  - `NUMBERVALUE` 미구현(#NAME?), `VALUE("12:00")` 이 시각을 읽지 못한다.
- 릴리즈: v0.227.0 (2026-09-02)

## 2026-09-02
- 선택: CEILING·FLOOR 의 음수 배수 규칙을 엑셀에 맞춘다 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `evaluateMath` 가 `number*factor < 0` 이면 CEILING·FLOOR·MROUND 를 모두 #NUM! 로 냈는데, 그것은 MROUND 에만 맞는 규칙이라 엑셀·구글 시트가 답을 내는 `CEILING(-4.5,2)`(=-4), `FLOOR(-4.5,2)`(=-6), `CEILING(-2.5)`(=-2) 가 오류였다. CEILING·FLOOR 는 "양수 + 음의 배수" 만 #NUM! 로 남기고, 몫이 음수일 때는 0 에서 멀어지는 쪽과 위쪽이 반대이므로 반올림 방식(roundAwayFromZero/roundTowardZero)을 몫의 부호에서 뒤집도록 고쳤다 — 십진 셈(`CEILING(-0.1-0.2,0.1)`=-0.3)과 부호가 같은 기존 답은 그대로다. 검증: 엑셀 도움말의 부호 표를 옮긴 `TestCeilingAndFloorFollowExcelSignRules` 를 더하고 `gofmt -l`, `go build ./...`, `go test ./...`(전체 통과), `cd web && npm ci && npm test`(480개 통과), `scripts/check-release-docs.sh`, `scripts/check-commit-identities.sh`. 커밋 2개(1663756, d98ec44), 릴리즈 노트 v0.229.0 과 README VERSION·USER_GUIDE 갱신.
- 보류 아이디어:
  - `internal/external/fetcher.go` 의 응답 캐시가 만료 항목을 지우지 않아 무한히 커진다. 만료 정리 + 상한 필요.
  - 같은 캐시가 실패도 담아, 관리자가 allowed_hosts 를 고쳐도 최대 5분간 "허용되지 않은 호스트" 가 남는다. 설정 변경 시 무효화 필요.
  - `internal/external/fetcher.go` 의 `short()` 가 160바이트에서 자르며 UTF-8 글자를 쪼갠다(한글 오류 메시지가 깨진다).
  - `NUMBERVALUE` 미구현(#NAME?), `VALUE("12:00")` 이 시각을 읽지 못한다.
  - `TRUNC` 이 `math.Pow`/이진 실수로 자리를 잘라 `TRUNC(2.29,1)` 같은 값에서 어긋날 수 있다. ROUND·CEILING 이 쓰는 `decimalRound` 로 통일할 것.
- 릴리즈: v0.229.0 (2026-09-02)
- 릴리즈: v0.229.0 (2026-09-02)

## 2026-09-03
- 선택: 외부 호출 캐시가 정책 검사를 가로막지 않고 상한을 지킨다 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `internal/external/fetcher.go` 의 답 캐시에 겹쳐 있던 세 가지를 함께 고쳤다 — (1) 허용 목록·https·주소 형식을 보는 정책 검사(`CheckURL`)를 `fetch` 에서 `one` 의 맨 앞으로 올려 캐시 바깥에 두어, 관리자가 `external.allowed_hosts` 를 고치면 `cache_seconds`(기본 5분)를 기다리지 않고 곧바로 통하고 목록에서 뺀 호스트의 지난 답도 캐시에서 나오지 않는다, (2) 만료 항목을 지우는 자리가 없어 무한히 자라던 지도에 `store()` 를 두어 새 답이 들어올 때 만료분을 쓸고 512개 상한에서 가장 먼저 만료될 항목을 내보낸다, (3) `short()` 가 160바이트에서 잘라 한글을 반 글자로 남기던 것을 160글자·글자 경계로 고쳤다. 검증: 새 테스트 3개(`TestPolicyIsNotCachedInEitherDirection`, `TestCacheSweepsExpiredAndStaysBounded`, `TestShortKeepsKoreanLettersWhole`)와 `gofmt -l`, `go vet`, `go build ./...`, `go test ./...`(전체 통과), `scripts/check-release-docs.sh`, `scripts/check-commit-identities.sh`. 커밋 1개(1266384), 릴리즈 노트 v0.230.0 과 README VERSION 갱신. 웹 변경이 없어 npm 검사는 돌리지 않았다.
- 보류 아이디어:
  - `NUMBERVALUE` 미구현(#NAME?), `VALUE("12:00")` 이 시각을 읽지 못한다.
  - `TRUNC` 이 `math.Pow`/이진 실수로 자리를 잘라 `TRUNC(2.29,1)` 같은 값에서 어긋날 수 있다. ROUND·CEILING 이 쓰는 `decimalRound` 로 통일할 것.
  - `csvNumber` 가 IMPORTDATA 의 `"1,200"`·`"12%"`·앞뒤 통화 기호를 글자로 남긴다. 시트가 읽는 수의 범위와 맞출지 검토할 것.
  - 외부 호출 캐시 키가 함수·주소만 담아, `external.max_kb` 를 올려도 캐시가 남아 있는 동안은 "크기를 넘습니다" 가 그대로다(정책 값도 키에 넣거나 크기 오류는 담지 않기).
  - `internal/external` 의 동시 접근에 `-race` 시험이 없다. 한 번의 재계산이 여러 주소를 동시에 부르는 길을 시험으로 고정할 것.
- 릴리즈: v0.230.0 (2026-09-03)
- 릴리즈: v0.230.0 (2026-09-03)
