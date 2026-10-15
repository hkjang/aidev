- 과제: PR 버전 조회 실패 시 버전 고정 없는 승인 요청을 만들지 않는다 (가치 4 / 위험 2 / 작업량 S)
- 왜: `internal/tools/executor.go:prVersion` 이 업스트림 오류를 삼켜 `nil` 을 돌려주므로, PR 읽기가 실패하는 동안 만들어진 승인 요청은 `pr_version = NULL` 로 저장된다. `internal/approval/approval.go:Check` 의 두 버전 비교는 `req.PRVersion != nil` 조건 아래에 있어서, 그렇게 만들어진 승인은 승인 이후 PR 에 새 커밋이 올라와도 영구히 통과한다 — 2026-10-03 에 닫은 "현재 버전 미확인" 쪽과 달리 "애초에 고정하지 못한" 쪽이 아직 열려 있고, 일시적인 Bitbucket 장애가 승인 보호를 조용히 무력화한다.
- 수용 기준:
  1) PR 읽기가 실패하는 동안 `bitbucket_comment_pull_request` (RiskWrite + 승인 필요 + PR 대상, 브랜치 인자 없음) 를 호출하면 `APPROVAL_REQUIRED` 가 아니라 `PERMISSION_UNKNOWN` 으로 거부되고, `approval_requests` 테이블에 행이 **0개** 생긴다(현재는 `pr_version = NULL` 행이 1개 생긴다).
  2) 업스트림이 복구된 뒤 같은 호출은 전과 같이 `PRVersion == 14` 로 고정된 승인 요청을 만들고, 승인 후 PR 버전이 15 로 올라가면 `APPROVAL_STALE` 로 거부된다(기존 보호 유지).
  3) 테스트가 증명할 것: (a) 실패 중 승인 행 0개 + 거부 코드, (b) 복구 후 고정된 버전으로 정상 승인 생성, (c) 기존 `TestCommentApprovalDeniedWhenPullRequestVersionUnknown`(승인 소비 시점 미확인 → `APPROVAL_STALE`, 승인은 `approved` 유지) 가 그대로 통과 — 소비 경로의 fail-closed 는 바꾸지 않는다.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/tools/executor.go:prVersion` — 시그니처를 `(*int, error)` 로 바꿔 `Provider.Adapter`/`credential`/`adapter.PullRequest` 오류를 삼키지 말고 전달한다.
  - `internal/tools/executor.go:checkApproval` — `raw` 분기 전에 PR 버전을 한 번만 조회한다(`res.PullRequest > 0 && res.Repository != ""` 일 때). (a) 승인 요청 **생성** 경로(`raw == ""`)에서 조회가 실패하면 `toolErr(CodePermissionUnknown, "PR 버전을 확인할 수 없어 승인 요청을 만들 수 없습니다")` 를 돌려주고 `Approvals.Create` 를 호출하지 않는다. (b) 승인 **소비** 경로에서는 오류 시 `currentVersion` 을 nil 로 두어 기존 `approval.Check` 의 fail-closed(`ErrStale`) 를 그대로 쓴다 — 여기 의미는 바꾸지 않는다.
  - `internal/tools/executor.go:newApprovalRequest` — 내부 `prVersion` 호출을 없애고 `checkApproval` 이 조회한 `version *int` 을 인자로 받는다(호출부 1곳).
  - `internal/tools/integration_test.go` — 기존 `newFixture`(실제 Registry→Executor→Resolver→PostgreSQL→HTTP Bitbucket 대역)와 이미 있는 `fakeBitbucket.prFetchFails` 로 테스트 1건 추가: `prFetchFails = true` 에서 comment 호출 → `PERMISSION_UNKNOWN` + `SELECT count(*) FROM approval_requests` 가 0, 그 뒤 `prFetchFails = false` 로 복구 → `PRVersion == 14` 고정 승인 생성 → `Decide` 후 `f.bitbucket.prVersion = 15` → `APPROVAL_STALE` 와 `len(f.bitbucket.comments) == 0`. 새 목·새 주입은 만들지 말 것(기존 테스트들이 승인 행 수를 세는 방식은 `TestMergeTargetLookupFailureDoesNotCreateApproval`(626행) 참고).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd /home/hkjang/.cache/auto-improve-wt/bbmcp`
  - `docker start bbmcp-test-pg` — 이번 정찰에서 `docker ps -a` 로 컨테이너가 **Exited (255) 2 days ago** 상태로 남아 있음을 확인했다(기동은 승인이 필요해 하지 못했으므로 구현자가 먼저 켜야 한다. 없으면 `docker run -d --name bbmcp-test-pg -e POSTGRES_USER=bbmcp -e POSTGRES_PASSWORD=bbmcp -e POSTGRES_DB=bbmcp_test -p 15532:5432 postgres:16-alpine`)
  - `export TEST_DATABASE_URL='postgres://bbmcp:bbmcp@127.0.0.1:15532/bbmcp_test?sslmode=disable'`
  - `go build ./... && go vet ./...`
  - `go test ./internal/tools -count=1 -p 1 -v` (수정 전 새 테스트만 FAIL → 수정 후 PASS 를 반드시 로그로 남길 것)
  - `go test ./... -count=1 -p 1 -v` — `--- SKIP` 0건을 눈으로 확인(TEST_DATABASE_URL 없으면 통합 테스트가 조용히 skip 되므로 exit 0 만 믿지 말 것)
- 위험과 피할 것:
  - `internal/approval/approval.go` 는 건드리지 말 것. `Check` 의 `req.PRVersion != nil && currentPRVersion == nil → ErrStale`(2026-10-03) 와 조건부 UPDATE 단일 소비(동시성 테스트)가 회귀 금지선이다.
  - merge 경로는 13b44b9 이후 승인 검사보다 먼저 `prTargetBranch` 가 fail-closed 로 막아서 `prFetchFails` 시 이미 `PERMISSION_UNKNOWN` 이다. merge 테스트의 기대 코드를 바꾸지 말고, 새 테스트는 브랜치 검사를 타지 않는 comment 경로로 쓸 것.
  - 보호 경로(`internal/auth/*`, `internal/api/oauth.go`, `migrations`, `.github/workflows`, `web/`)는 건드리지 말 것 — 스키마 변경 없이 끝나는 과제다.
  - `prVersion` 을 `(*int, error)` 로 바꿀 때 소비 경로에서 오류를 그대로 도구 오류로 올리면 안 된다. 그러면 `APPROVAL_STALE` 이 `PERMISSION_UNKNOWN` 으로 바뀌어 기존 테스트가 깨지고, 승인 재시도 보존 의미도 흐려진다.
  - 승인 행이 0개여야 한다는 단정은 해당 테스트의 `approval_requests` 를 다른 테스트가 TRUNCATE 하는 fixture 특성에 기댄다. `t.Parallel()` 금지, `-p 1` 유지.
- 이번 정찰에서 확인한 것 / 미확인: 확인 — `prVersion`(executor.go:338) 이 세 오류 경로 모두에서 `nil` 반환, `newApprovalRequest`(executor.go:316) 가 그 nil 을 `Approvals.Create` 의 `version` 으로 넘김, `Check`(approval.go:224~232) 의 버전 비교 2개가 모두 `req.PRVersion != nil` 아래, 테스트 하네스에 `fakeBitbucket.prFetchFails`(integration_test.go:58) 와 comment 경로 테스트(541행) 가 이미 있음. 미확인 — 이번 회차에는 `go vet`/`go test` 실행이 환경 승인으로 막혀 하나도 돌리지 못했다(샌드박스 거부). 즉 "실패 중 `pr_version = NULL` 행 1개 생성" 은 코드 판독에 의한 추론이며 런타임 재현이 아니다. 구현자는 수정 전에 먼저 red 로그를 남길 것.
- 차선 후보: `approval.redactArgs` 의 중첩 map/slice 미스크럽 (가치 3 / 위험 2 / 작업량 S) — `internal/approval/approval.go:271 redactArgs` 는 최상위 키만 검사하므로 `{"body":{"password":"…"}}` 같은 중첩 값이 `arguments_redacted` 에 원문으로 저장되고 `GET /api/admin/approvals` 로 나간다(2026-10-04 에 최상위 키만 닫았다). 같은 함수의 `s[:2000]` 바이트 절단도 한국어 인자의 마지막 글자를 깨뜨리므로 `internal/mcp/server.go:trimToRune` 과 같은 원칙으로 함께 고칠 수 있다(재귀 깊이 상한을 정해 둘 것).
