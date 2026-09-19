# 수리 요약 — PR #13 (07c19c9) CI step 7 `govulncheck ./...` 실패

- 재현: 로컬 `GOTOOLCHAIN=local govulncheck ./...` (govulncheck v1.8.0, DB 2026-09-16 18:00 UTC) → CI 와 동일하게 GO-2026-6452 / `internal/store/seed.go:66:25 ExtractWorkbookDefaults → excelize.File.GetRows`, exit 3.
- 원인: vulndb 보고서 `https://vuln.go.dev/ID/GO-2026-6452.json` 의 범위가 `introduced: "0"` 뿐이고 `fixed` 이벤트가 없음(2026-09-20 확인). 따라서 proxy 의 최신 태그 v2.11.0 은 물론 fix 커밋 이후의 master 의사판 `v2.11.1-0.20260919114233-e44e6306e9f7` 도 전부 "affected" 로 판정되어, 어떤 excelize 판으로도 초록이 안 됨. 이 PR 의 diff 는 `web/` 테스트 3파일뿐이고 Go 코드·go.mod 는 origin/main 과 동일 — main 도 같은 자리에서 똑같이 빨강.
- 코드로 푸는 유일한 길은 affected 심볼(GetRows·GetCellValue·Rows.Columns·GetCols 등 읽기 API 전부)을 쓰지 않도록 `seed.go` 의 xlsx 파싱을 통째로 다시 쓰는 것 — 이 PR 의 목적(화면 경로↔라우트 대조 vitest)과 무관한 파일의 기능 재작성이며, 워크플로 완화는 금지이므로 **고치지 않고 커밋 없이 종료**.
- 결론: 지적된 실패는 이 변경의 결함이 아니라 외부 vulndb 게이트. vulndb 에 `fixed` 가 실리거나(엑셀라이즈 v2.11.1 태그 + DB 반영) 사람이 게이트를 판단할 때까지 이 PR 은 CI 초록이 불가능. 재검증 방법: `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | grep -c fixed` 가 0 이 아니게 되면 `go get github.com/xuri/excelize/v2@<fixed판>` 후 `govulncheck ./...`.
