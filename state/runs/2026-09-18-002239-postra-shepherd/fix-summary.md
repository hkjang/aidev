# PR #11 fix summary

- 문제: `CollectHandoffClaim` 은 로그인 없는 공개 경로에서 실행되어 컨텍스트에 principal 이 없는데, `a.audit(ctx, …)` 가 `userIDFrom(ctx)` 로 사용자를 정해 수령 이벤트가 항상 `DefaultUserID` 에 기록됐다. 다중 사용자 환경에서 다른 사용자의 메일 수령이 본인 감사 로그(`SearchAudit`)에 남지 않고 기본 사용자 계정에 남의 message ID 로 찍히는 결함이 리뷰 지적대로 맞음을 확인했다.
- 재현: `TestHandoffCollectionIsAuditedToClaimOwner` 를 추가해 `usr_other` 가 발급한 claim 을 principal 없는 컨텍스트로 수령한 뒤, 이벤트가 소유자 감사 로그에는 없고 기본 사용자에게 붙는 것을 실패로 확인했다.
- 수정: `internal/application/handoff.go` 에서 수령 감사 이벤트를 `a.audit` 대신 `Store.AppendAudit` 로 직접 기록하며 `UserID: claim.UserID` 를 넘기도록 바꿨다 (actor 는 기존처럼 컨텍스트에서).
- 검증: `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과, 핸드오프 테스트는 `-race` 로도 통과.
- 커밋: `a44dc07 fix(handoff): audit claim collection to the claim owner, not the default user, since the collect route carries no principal` (push 하지 않음).
