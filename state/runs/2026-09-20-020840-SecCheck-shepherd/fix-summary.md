# fix-summary — PR #8 (693b733) — 커밋 없음

- **문제**: CI step 7 `govulncheck ./...` 가 exit 3. 로컬 재현 동일: GO-2026-6452(excelize, `internal/store/seed.go:66` `GetRows`) — vulndb 보고서(2026-09-16 18:00Z 공개)의 range 가 `introduced: 0` 뿐이고 `fixed` 이벤트가 없어 **모든 excelize 버전**(최신 태그 v2.11.0, master 의사버전 v2.11.1-0.20260919 포함)이 걸림. step 6 go test 는 CI 에서 성공, step 8 이후는 skipped.
- **이 변경과 무관**: PR diff 는 mail_test.go·Settings.tsx·capture_all.js·docs 뿐, excelize/go.mod 를 손대지 않음. main 의 마지막 초록 run(eb2a3b0, 09-16 17:01Z)은 보고서 공개 1시간 전 — 지금 main 을 다시 돌려도 같은 자리에서 실패함.
- **코드로 못 고침**: 영향 심볼 목록에 `GetRows·GetCellValue·Rows.Columns·Cols.Rows·GetCols·xlsxC.getValueFrom` 등 셀을 읽는 API 전부가 들어 있어 호출부 우회가 불가능(3곳: seed.go:66, templates.go:810·955). 유일한 방법은 워크플로 완화(금지) 또는 xlsx 라이브러리 교체(범위 밖) → 규칙대로 커밋 없이 종료. vulndb 에 `fixed` 가 붙거나 excelize 가 패치 태그를 내면 그때 go.mod 만 올리면 됨(현재 열린 #9·#10 이 go.mod 를 건드리므로 충돌 주의).
- **이 변경 자체 검증(로컬)**: gofmt 무출력, `go vet ./...` OK, `go test ./internal/web -run 'Mail|Docs'` ok, `npx tsc --noEmit` OK, vitest 11/11 pass, `vite build` OK.
- **확신 없음**: 로컬 `npm audit --audit-level=high` 는 registry quick 엔드포인트가 400 "Invalid package tree"(엔드포인트 폐기 공지) — lockfile 은 origin/main 과 동일하고 main 의 09-16 run 에서 같은 lockfile 로 step 8 통과했으므로 환경 문제로 판단하지만 CI 에서 재확인 필요.
