# 수리 요약 — kanpic PR #35

- 지적이 맞았다. `internal/httpapi` 에 새 테스트 3개를 먼저 써서 재현했다(pgx 처럼 취소된 ctx 에서 실패하는 저장소 대역 `cancelAwareRepository`): 칸 저장 알림 0통, `workbookAudience` 빈 워크북·수신자 nil, 댓글 알림은 언급 메일만 나가고 워크북 관계자 메일은 사라졌다.
- 원인은 비평 그대로다 — 수신자를 **구하는** 읽기(SheetWatchRules·GetWorkbook·GetWorkbookSharing·LookupUsers)가 `r.Context()` 에 매달려 있어 `mail.Notify` 에 닿지도 못했다. service.go 안의 WithoutCancel 은 수신자가 정해진 다음이라 구제 불가.
- 수리 (1)번 갈래: `internal/httpapi/mail.go` 에 `notifyContext`(= `WithTimeout(WithoutCancel(ctx), 10s)`, automations.go:249 와 같은 자)를 두고 `notifyWatchers`·`workbookAudience`·`notifyCommentMail`·`putWorkbookShare`·`createAccessRequest`·`decideAccessRequest` 의 알림 채비에 적용했다. 프로덕션 파일 3개, 새 테스트 1개.
- 검증: `go test ./... -race` 18패키지 통과(exit 0), `go vet ./...`·`gofmt -l` 깨끗, `check-release-docs.sh`·`check-commit-identities.sh HEAD` 통과. 고치기 전 같은 테스트 3개가 위 자리에서 실패하는 것을 먼저 봤다.
- 남는 것: 실제 Postgres 로 취소 갈래를 돌려 보지는 않았다(전제는 puddle pool.go:338-344 독해). 비평가가 지적한 `book.Title` 빈 제목은 이제 사라진다.
