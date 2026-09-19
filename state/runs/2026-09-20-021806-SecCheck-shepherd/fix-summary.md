# 수리 요약 — PR #11 (5063150)

- 문제: security-ci run 35446551303 은 **step 7 `Go vulnerability scan` 하나만** 실패(GitHub API `jobs` 로 확인, step 6 테스트·vet 은 성공). 로컬 `govulncheck ./...` 도 같은 자리에서 exit 3 — GO-2026-6452(excelize, `internal/store/seed.go:66` GetRows 호출), `Fixed in: N/A`.
- 원인: 오늘(09-20) vuln.go.dev 의 GO-2026-6452.json 은 `modified 2026-09-16`, affected range 가 `introduced: 0` 뿐이고 `fixed` 이벤트 없음 → 어떤 excelize 버전(v2.11.0 최신, master 의사버전 포함)으로도 빨강. 이 브랜치의 diff(`docs/admin-guide.md` 삭제 338줄)는 Go 코드·go.mod 를 0바이트도 안 건드려 원인과 무관하며, origin/main 도 같은 자리에서 빨강.
- 고치지 않은 이유: 코드로 풀리는 길이 없다(excelize 제거는 새 기능/리팩터로 범위 밖, `replace`·vendor 우회·ci.yml 완화는 절대 규칙 위반). 따라서 **커밋 없이 종료**.
- 재확인한 것: `go build ./...`·`go vet ./...`·`go test ./internal/web -run 'Guide|Doc|Readme|Screenshot'` PASS, `admin-guide` 참조 0건, 작업 트리 clean.
- 남은 길: golang/vulndb 에 `fixed` 이벤트가 추가될 때 CI 재실행(workflow_dispatch)만 하면 됨 — 운영자 결정 항목.
