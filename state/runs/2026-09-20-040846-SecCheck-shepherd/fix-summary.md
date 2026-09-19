# fix-summary — PR #12 (dfe3412) CI 실패

- 실패 자리: run 35449769602 job `test-build-scan` step 7 `Go vulnerability scan`(`govulncheck ./...`) 만 failure. step 6 `Unit and database integration tests`(새 docs_test 포함)는 success, 이후 step 은 전부 skipped (GitHub API `/actions/runs/35449769602/jobs` 로 확인).
- 로컬 재현: 이 브랜치와 `origin/main`(eb2a3b0) 임시 worktree 모두 `govulncheck ./...` exit=3, `GO-2026-6452 … Fixed in: N/A` (excelize v2.11.0, `internal/store/seed.go:66` GetRows 호출). 2026-09-20 현재 `vuln.go.dev/ID/GO-2026-6452.json` 의 affected ranges 는 `{"introduced":"0"}` 뿐으로 `fixed` 이벤트가 여전히 없음.
- 판정: PR 변경(docs/operations.md, internal/web/docs_test.go 2개 파일)은 go.mod·seed.go·워크플로를 건드리지 않아 이 실패와 무관하며, 어떤 excelize 버전으로도 초록이 안 되는 외부 차단. 코드로 고칠 길이 없고 ci.yml 완화는 금지 규칙이므로 **수정·커밋 없이 종료**.
- 부수 확인: `gofmt -l`·`go vet ./internal/web/`·`go test ./internal/web/ -run 'Doc|Guide|Settings'` 통과. 브랜치 상태 clean.
