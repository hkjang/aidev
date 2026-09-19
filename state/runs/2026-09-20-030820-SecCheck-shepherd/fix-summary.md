# fix-summary — PR #11 (5063150) security-ci 실패

- 문제: GitHub API 로 job 105906357236 의 step 목록을 직접 확인 — 실패 step 은 **7 `Go vulnerability scan`** 하나뿐(1~6 성공, 이후 skipped). 로컬 `govulncheck ./...` 도 동일하게 exit 3: GO-2026-6452(excelize `GetRows`, `internal/store/seed.go:66`) `Fixed in: N/A`.
- 원인: 오늘(09-20) vuln.go.dev 의 GO-2026-6452 는 여전히 `modified 2026-09-16`, events `[{introduced: 0}]` 로 `fixed` 이벤트가 없고, proxy 의 excelize 최신 태그도 v2.11.0(현재 핀)이라 **어떤 버전으로 올려도 통과할 수 없다**. `origin/main`(eb2a3b0) 트리를 별도 worktree 로 검사해도 같은 exit 3 — 이 PR 의 변경(`docs/admin-guide.md` 삭제 1파일)과 무관한 외부 vulndb 결함.
- 고친 것: **없음(커밋 0건)**. 통과시키는 방법은 ci.yml 완화·`replace`/vendor 우회뿐인데 둘 다 절대 규칙 위반이고, go.mod 는 열린 PR #9·#10 과 충돌한다.
- 이 PR 자체의 정합성은 재확인: `admin-guide` 참조 0건, `go test ./internal/web -run 'Guide|Doc|Readme|Screenshot'` PASS.
- 다음 단계(운영자 결정): vulndb 에 `fixed` 가 추가될 때까지 대기하거나 golang/vulndb 에 보고서 갱신을 제출 — 그 전에는 main 포함 어떤 브랜치도 이 게이트를 넘지 못하므로 재수리 과제로 되돌리지 말 것.
