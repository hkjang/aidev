- 과제: 승인 검증의 fail-open 두 곳 닫기 — PR 버전 확인 실패 시 통과, 단일 사용 승인의 동시 재사용 (가치 5 / 위험 2 / 작업량 S)
- 왜: `internal/approval/approval.go:Check` 는 승인에 PR 버전이 고정되어 있어도 현재 버전을 못 가져오면(`Executor.prVersion` 은 Bitbucket 오류 시 조용히 `nil` 반환) 버전 비교를 건너뛰고 통과시키고, 승인 소비가 `ByID` 조회 → 별도 `UPDATE` 로 나뉘어 있어 같은 `approvalId` 로 동시에 들어온 호출이 모두 승인을 통과할 수 있다. README 가 내세우는 "fail-closed 기본"·"1회용 승인" 두 약속이 바로 이 두 줄에서 깨진다.
- 수용 기준:
  1) 승인이 PR 버전을 고정했는데 현재 버전을 확인할 수 없으면(`currentPRVersion == nil`) `approval.ErrStale` 로 거부되고, 실행기에서 `tools.CodeApprovalStale` 로 나온다. 이때 승인 레코드는 `consumed` 로 바뀌지 않고 `approved` 로 남아(Bitbucket 복구 후 재시도 가능) 있어야 한다.
  2) 승인 소비가 `UPDATE … WHERE id=$1 AND status='approved'` 단일 조건부 UPDATE 로 바뀌고, `RowsAffected()==0` 이면 `ErrStale`("이미 사용된 승인입니다") 를 반환한다. 승인된 승인 1건에 `Check` 를 20개 고루틴으로 동시에 걸면 **정확히 1건만** 성공한다.
  3) 기존 승인 테스트 3건(`TestWriteToolRequiresApprovalBoundToArguments`, `TestApprovalGoesStaleWhenArgumentsChange`, `TestMergeApprovalGoesStaleWhenPullRequestAdvances`)이 그대로 통과한다 — 특히 1)의 변경이 PR 과 무관한 쓰기 도구(코멘트 등, 승인에 `PRVersion` 이 `nil`)를 막지 않아야 한다.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/approval/approval.go:Check` — (a) `req.PRVersion != nil && currentPRVersion == nil` 분기 추가 → `ErrStale` ("현재 PR 버전을 확인할 수 없습니다"). 기존 `*req.PRVersion != *currentPRVersion` 분기는 그대로 둘 것. (b) 마지막 `UPDATE approval_requests SET status=$2, consumed_at=NOW() WHERE id=$1` 에 `AND status='approved'` 를 붙이고 `pool.Exec` 의 `pgconn.CommandTag.RowsAffected()` 가 0 이면 `ErrStale` 반환.
  - `internal/tools/integration_test.go` — 기존 `fakeBitbucket` 에 `prFetchFails bool` 추가하고 `/rest/api/1.0/projects/AI/repos/text2sql/pull-requests/7` 핸들러(현재 145행 근처)에서 참이면 503 반환. 새 테스트 2건: `TestMergeApprovalDeniedWhenPullRequestVersionUnknown` (승인 생성 → `Decide(true)` → `prFetchFails=true` → `approvalId` 로 재호출 → `CodeApprovalStale`, `f.bitbucket.merged == false`, `f.approvals.ByID(...).Status == approval.StatusApproved`), `TestApprovalIsConsumedOnceUnderConcurrency` (`f.approvals.Decide` 로 승인한 뒤 `sync.WaitGroup` + 닫히는 채널 배리어로 20개 고루틴이 `f.approvals.Check(ctx, id, "sub-hkjang", "bitbucket_comment_pull_request", args, nil)` 호출 → 성공 1건).
  - 픽스처는 이미 있는 `newFixture`/`f.principal()`/`cloneArgs`/`codeOf` 를 쓸 것. 새 목·대역을 만들지 말 것(프로덕션 배선 그대로 도는 기존 하네스다).
- 검증 명령:
  - `go build ./... && go vet ./...`
  - `TEST_DATABASE_URL='postgres://bbmcp:bbmcp@localhost:5432/bbmcp_test?sslmode=disable' go test ./internal/tools/ ./internal/approval/... -count=1 -p 1 -race -run 'Approval|Merge'`
  - 전체: `TEST_DATABASE_URL=... go test ./... -count=1 -p 1` (CI 와 같은 형태. `TEST_DATABASE_URL` 없으면 DB 테스트는 skip 되므로 **반드시 Postgres 를 띄우고** 돌릴 것 — 안 띄우면 이 과제의 테스트는 전부 skip 되어 아무것도 증명하지 못한다. 미확인: 이 워크트리에서 로컬 Postgres 가 떠 있는지 확인하지 못했다. 없으면 `docker run -d --name bbmcp-test -e POSTGRES_USER=bbmcp -e POSTGRES_PASSWORD=bbmcp -e POSTGRES_DB=bbmcp_test -p 5432:5432 postgres:16-alpine`.)
- 위험과 피할 것:
  - `Check` 의 **검사 순서를 바꾸지 말 것**. 인자 해시 검사(`req.ArgumentsHash != Hash(...)`)가 PR 버전 검사보다 먼저 와야 한다 — 해시가 같다는 것이 `Resource` 가 동일함(따라서 `res.PullRequest > 0 && res.Repository != ""` 조건이 생성 시점과 같음)을 보장하고, 그래서 1)이 PR 과 무관한 도구를 오탐으로 막지 않는다.
  - `internal/tools/executor.go:prVersion` / `prTargetBranch` 는 **건드리지 말 것**. 오류를 `nil`/`""` 로 삼키는 것이 보기엔 원인이지만, 거기서 오류를 올리면 브랜치 제한 경로(`CodeBranchRestricted`/`CodePermissionUnknown`)의 의미가 같이 바뀌어 파일 수와 영향 범위가 커진다. 이번 회차는 `approval.Check` 안에서만 닫는다.
  - `internal/database/migrations`, `internal/auth/*`, `internal/api/oauth*.go`, `.github/workflows/*` 는 건드리지 말 것 (보호 경로, 이번 과제와 무관).
  - `approval_requests.status` 는 `VARCHAR` 로 쓰이고 기존 `Decide` 가 `$2::VARCHAR` 캐스팅을 쓰고 있다. 조건부 UPDATE 에서도 리터럴 `'approved'`/`'consumed'` 또는 동일한 캐스팅을 써서 pgx 타입 추론 오류를 피할 것.
  - 동시성 테스트는 `-race` 로 돌릴 것. 수정 전 코드에서 실제로 1건 초과 성공하는지 한 번 확인하고(결함 증명), 그 다음 수정하라.
  - 수정 **전** 1)번 시나리오의 실제 코드는 확인했다: 머지 핸들러(`internal/tools/write.go` 268행 근처)가 PR 을 다시 조회하므로 `prFetchFails=true` 면 수정 전에도 머지 자체는 실패해 `CodeUpstream` 이 난다. 즉 "머지되지 않는다"만으로는 결함을 증명하지 못하니, **반드시 오류 코드(`APPROVAL_STALE` vs `UPSTREAM_ERROR`)와 승인 레코드 상태(`approved` vs `consumed`)를 함께 단정**하라. 수정 전에는 승인이 `consumed` 로 소진되고, 수정 후에는 `approved` 로 남는다 — 이것이 이 과제의 핵심 증거다.
  - `newFixture` 는 `TEST_DATABASE_URL` 이 없으면 `t.Skip` 하고, 매 테스트마다 `users, approval_requests, …` 를 `TRUNCATE` 한 뒤 `id=1 / keycloak_sub='sub-hkjang'` 사용자를 심는다(187~210행). 동시성 테스트에서 승인 주체 sub 은 `sub-hkjang` 으로 맞출 것.
- 차선 후보: `internal/mcp/jsonrpc.go` 의 JSON-RPC 프레이밍 단위 테스트 추가 (현재 이 패키지에 테스트 0건, DB 불필요, 가치 3 / 위험 1 / 작업량 S). 1순위가 성립하지 않으면(예: 조건부 UPDATE 가 이미 들어가 있거나 PR 버전 분기가 이미 닫혀 있으면) 이것을 하라.
