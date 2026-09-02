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
