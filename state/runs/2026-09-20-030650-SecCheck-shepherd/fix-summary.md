# 수리 요약 — PR #10 (9b4b7aa) security-ci 실패

- 문제: run 35442392185 의 실패 step 은 `Go vulnerability scan`(ci.yml step 7 `govulncheck ./...`) 하나. 로컬 재현(go1.26, govulncheck@latest, HEAD 9b4b7aa) 동일하게 exit 3 — `GO-2026-6452 / Found in excelize@v2.11.1-0.20260918021423-0434413565bf / Fixed in: N/A / seed.go:66 ExtractWorkbookDefaults calls File.GetRows`.
- 원인: vulndb 보고서(`vuln.go.dev/ID/GO-2026-6452.json`, modified 2026-09-16 그대로)에 `fixed` 이벤트가 없어(`events: [{introduced: "0"}]`) 어떤 excelize 버전도 취약으로 판정된다. 핀한 버전은 실제 수정(`rows.go:361 index < 0 ||` 하한 검사)을 담고 있고 새 태그(v2.11.0 이 마지막)도 없다. 보고서의 affected symbols 에 `File.GetRows`·`GetCols`·`GetCellValue`·`Rows.Columns` 등 모든 읽기 API 가 포함돼 호출 지점을 바꿔 피할 수도 없다.
- 조치: **커밋 없음.** 남은 수단은 워크플로 완화·`replace`·라이브러리 교체뿐인데 앞의 둘은 금지, 마지막은 이 변경의 목적 밖이다. 이 PR 이 하려던 일(panic 수정 핀 + precheck 에 같은 게이트 + operations.md 설명)은 그대로 맞고, CI 는 vulndb 가 `fixed` 를 기록해야만 초록이 된다(운영자 결정: golang/vulndb 에 fixed 추가 PR).
- 확인 명령: `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.affected[].ranges'` 에 `fixed` 가 생기면 코드 변경 없이 CI 재실행만으로 통과해야 한다.
