# 수리 요약 — PR #9 (6424eb1) security-ci 실패

- 문제: job 105879965651 은 step 6(go test -race + vet) success, **step 7 `Go vulnerability scan` 만 failure**(GitHub API 로 확인). 로컬 재현 `govulncheck ./...` exit 3 — `GO-2026-6452 … excelize/v2@v2.11.1-0.20260918021423-0434413565bf … Fixed in: N/A … seed.go:66 GetRows`. CI 와 같은 자리.
- 원인: vuln.go.dev 의 GO-2026-6452(modified 2026-09-16T18:00Z, 오늘도 동일)에 `introduced: 0` 만 있고 `fixed` 이벤트가 없어 **어떤 excelize 버전이든** 취약으로 보고됨. 근거 GHSA-fx5j-qcqg-grpf 는 `first_patched_version: 2.11.0` 이라 DB 쪽 오기록이고, golang/vulndb#6510·#6532 는 2026-09-20 현재 open. 핀 소스는 rows.go:361·cell.go:663 두 경로 모두 index<0 가드가 있어 코드는 이미 맞다.
- 고치지 않은 이유: 플래그된 심볼이 GetRows·GetCellValue·GetCols·Rows.Columns 등 excelize 의 셀 읽기 API 전부라, 코드로 초록을 만들려면 xlsx 읽기를 excelize 밖으로 다시 쓰는 가짜 수정뿐이고, 나머지 길은 ci.yml 완화(금지). 절대 규칙대로 **커밋 없이** 끝냄. 브랜치는 6424eb1 그대로, 작업 트리 clean.
- 다음 행동: `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | grep -c fixed` 가 1 이 되면 워크플로 재실행(workflow_dispatch)만으로 초록. 그 전에는 이 PR 을 포함해 main 도 같은 게이트에서 빨강이므로 병합 판단은 사람 몫.
