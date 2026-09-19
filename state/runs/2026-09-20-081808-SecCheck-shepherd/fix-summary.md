# 수리 요약 — PR #13 (07c19c9) CI 실패

- 문제: ci.yml step 7 `govulncheck ./...` 가 GO-2026-6452(excelize v2.11.0, `seed.go:66 GetRows`)로 exit 3. 로컬 재현 동일(govulncheck v1.8.0, DB 2026-09-16 18:00). PR 변경(web/ 3파일, Go 미변경)과 무관하며 origin/main 도 같은 자리에서 빨강.
- 원인: vulndb 항목이 `introduced: 0` 만 있고 `fixed` 이벤트가 없음(2026-09-20 08:19 curl 재확인). 항목이 참조하는 수정 커밋 93f0b3c·태그 v2.11.0 은 이미 우리가 쓰는 버전 — 즉 실제로는 패치된 코드인데 DB 결함으로 모든 버전이 취약으로 잡힘.
- 코드로 못 푸는 이유: affected symbols 에 GetRows·GetCellValue·GetCols·Rows.Columns 등 excelize 의 모든 읽기 API 가 들어 있어 호출 교체가 불가능하고, excelize 제거는 이 PR 의 목적(화면 경로 ↔ 라우트 대조 테스트)과 무관한 대규모 변경. 워크플로 완화는 금지.
- 조치: 커밋 없음. PR 자체 검증은 초록(`cd web && npx vitest run` 13/13, `go build ./...`·`go vet` 통과).
- 판단: vulndb 에 `fixed: 2.11.0` 이 실리거나 excelize 패치 릴리즈가 나오기 전까지 이 게이트는 어떤 PR 도 못 넘음. 사람 심사로 CI 게이트 예외 처리(또는 보류) 필요.
