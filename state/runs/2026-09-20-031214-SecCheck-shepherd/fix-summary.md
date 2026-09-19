# 수리 요약 — PR #12 (dfe3412)

- 실패 자리: security-ci `test-build-scan` step 7 `Go vulnerability scan`(`govulncheck ./...`). step 6(go test -race + vet)은 **success** — PR 의 새 테스트·문서 수정은 CI 를 통과했다 (GitHub API `actions/runs/35449769602/jobs` 로 확인).
- 로컬 재현: `govulncheck ./...` → exit 3, `GO-2026-6452 … Found in excelize/v2@v2.11.0, Fixed in: N/A`, trace `internal/store/seed.go:66 ExtractWorkbookDefaults → excelize.File.GetRows`. **`origin/main`(eb2a3b0) 에서도 똑같이 exit 3** — PR 은 `go.mod`·`go.sum`·`.github/workflows/` 를 0바이트도 건드리지 않았다.
- 외부 상태(2026-09-20): vulndb `GO-2026-6452.json` modified 2026-09-16, ranges `[{introduced: 0}]` 로 여전히 `fixed` 없음; proxy.golang.org 의 excelize 최신 태그는 v2.11.0. 즉 어떤 버전으로도 초록이 안 된다.
- **고치지 않고 커밋 없이 끝냄**: 남은 길은 워크플로 완화(금지) 또는 excelize 사용 코드 교체(이 변경의 목적과 무관·보호 구역 근처) 뿐이라 절대 규칙에 걸린다. PR 자체 결함은 없음 — 로컬에서 gofmt/vet 깨끗, 두 docs 테스트 PASS, `operations.md` 를 main 버전으로 되돌리면 새 테스트가 `notification.digest_hour` 인용을 정확히 잡아낸다(FAIL 확인 후 복원).
- 다음 판단: vulndb 에 `fixed` 이벤트가 붙거나(또는 excelize 새 태그) 사람이 게이트를 결정하기 전까지 PR #12 는 #8~#11 과 같은 외부 차단 상태로 두는 것이 맞다.
