- 과제: 승인 레코드의 인자 스크럽 누락 키 보강 (`internal/approval/redactArgs`) (가치 4 / 위험 2 / 작업량 S)
- 왜: `internal/approval/approval.go:258 redactArgs` 는 키 이름에 `token`/`secret`/`password` 가 들어간 것만 `[redacted]` 로 바꾼다. `credential`, `authorization`, `apikey`, `api_key` 같은 이름으로 들어온 값은 원문 그대로 `approval_requests.arguments_redacted` 에 저장되고, `scan()` 이 그것을 `Request.Arguments` (JSON `arguments`) 로 되살려 `GET /api/admin/approvals` (`internal/api/admin.go:58`, `listApprovals` → `httpx.JSON(w, 200, list)`) 로 관리 콘솔에 그대로 나간다. 호출자 인자는 필터 없이 흘러온다 — `Executor.Invoke(ctx, p, name, args Args)` (`internal/tools/executor.go:99`, `Args = map[string]any`) 가 받은 맵이 스키마 필터를 거치지 않고 `newApprovalRequest` (executor.go:309) → `Approvals.Create` (executor.go:326) → `redactArgs` 로 그대로 전달된다. 고치면 "감사·저장에 스크럽 이전 원문을 넘기지 않는다" 는 운영 원칙이 승인 경로에서도 지켜지고, 같은 호출자 맵을 읽는 두 경로(`internal/audit/audit.go:redactKeys` 는 이미 `authorization`/`pat`/`apikey`/`api_key` 까지 막는다)의 차이가 좁혀진다.
- 수용 기준:
  1) `project`/`repository`/`pullRequest`/`text` 같은 정상 인자는 `arguments_redacted` 에 값이 그대로 남아 승인자가 무엇을 승인하는지 계속 볼 수 있다(기존 동작 유지, 2000자 초과 문자열 절단도 유지).
  2) 이름에 `credential`, `authorization`, `apikey`, `api_key`, `accesskey`, `bearer`, `passwd`, `privatekey` 가 들어간 인자는 DB 에 `[redacted]` 로만 저장된다 — 즉 `approvals.ByID(...).Arguments["apiKey"] == "[redacted]"`.
  3) 테스트가 증명해야 하는 것: **원문이 DB 를 거쳐 되돌아오지 않는다**는 것. 함수 단위 비교가 아니라 프로덕션 배선(`f.exec.Invoke` → 승인 생성 → `f.approvals.ByID`)을 통과한 레코드의 `Arguments` 맵을 검사해야 한다. 수정 전에는 이 단정이 원문 값("super-secret-pat" 등)으로 실패하는 것을 먼저 확인할 것.
- 건드릴 파일:
  - `internal/approval/approval.go:redactArgs` (258–274행) — 민감 키 판정을 `token|secret|password` 3개 하드코딩 조건문에서 패키지 수준 슬라이스(예: `var sensitiveKeyParts = []string{...}`) 순회로 바꾸고 위 목록을 추가. 함수 시그니처·`Create` 호출부는 그대로 둔다.
  - `internal/tools/integration_test.go` — 기존 프로덕션 배선 하네스에 테스트 1건 추가. `TestWriteToolRequiresApprovalBoundToArguments` (374행)의 1단계를 그대로 본떠서: `f.bitbucket.projectPerm["AI"]="PROJECT_WRITE"`, `f.bitbucket.repoPerm["AI/text2sql"]="REPO_WRITE"` 를 세팅하고 `tools.Args{"project":"AI","repository":"text2sql","pullRequest":7,"text":"검토 의견","apiKey":"…","authorization":"Bearer …","credential":"…"}` 로 `bitbucket_comment_pull_request` 를 `f.exec.Invoke` 한다. `errors.As(err, &te)` 로 `tools.CodeApprovalRequired` 와 `te.Approval` 을 받고, `f.approvals.ByID(f.ctx, te.Approval.ID)` 의 `Arguments` 를 검사한다(민감 키 3개는 `[redacted]`, `project`/`text` 는 원문). 핸들러를 실제로 실행할 필요가 없으니 `Decide` 단계는 넣지 않는다.
  - 프로덕션 파일 1개 + 테스트 1개. 더 늘리지 말 것.
- 검증 명령:
  - `go build ./... && go vet ./...`
  - Postgres 를 띄운 뒤 `TEST_DATABASE_URL='postgres://bbmcp:bbmcp@localhost:5432/bbmcp_test?sslmode=disable' go test ./internal/tools -count=1 -run 'Approval' -v` → 그다음 전체 `TEST_DATABASE_URL=… go test ./... -count=1 -p 1`
  - **`-p 1` 필수**, `TEST_DATABASE_URL` 없으면 `newFixture` 가 조용히 `t.Skip` 하므로 "통과" 가 "전부 건너뜀" 일 수 있다. 출력에 `ok  github.com/hkjang/bbmcp/internal/tools` 가 skip 없이 찍히는지 확인할 것.
  - (이번 정찰 세션에서는 샌드박스 권한 때문에 위 명령을 **실행하지 못했다** — 명령 자체는 2026-10-03 회차에서 실제로 통과한 것이다.)
- 위험과 피할 것:
  - **`internal/audit/audit.go:redactKeys` 를 베껴 오지 말 것.** 두 경로의 계약이 다르다. 감사 쪽은 `diff`/`content`/`source`/`prompt`/`messages` 본문도 버리는데, 승인 쪽은 승인자가 무엇을 승인하는지 봐야 하므로 본문을 가리면 제품이 망가진다(이미 2000자 절단으로 처리 중). 또 `redactKeys` 에는 `"pat"` 이 있고 판정이 `strings.Contains` 라서 **`path` 가 걸린다** — `path` 는 실제 도구 인자다(`internal/tools/read.go:142,168`). 승인 쪽에 `pat` 을 부분일치로 넣으면 `path` 인자가 가려진다. `pat` 은 넣지 말거나 정확 일치로만 처리할 것. (감사 쪽의 `path` 과다 스크럽은 다른 파일·다른 계약이므로 **이번 회차 범위 밖** 이다. 아이디어 파일에 남겨 뒀다.)
  - 맨 `key` / 맨 `auth` 를 부분일치 목록에 넣지 말 것 — 지금은 충돌하는 도구 인자가 없지만 `projectKey`·`authMode` 류가 생기면 조용히 가려진다. `apikey`/`api_key`/`accesskey` 처럼 한정해서 쓸 것.
  - `Hash(toolName, args)` 는 **원문** 인자로 계산된다(approval.go:125). 스크럽은 저장용에만 적용되므로 해시 경로를 건드리면 승인-인자 결속이 깨진다. `redactArgs` 안에서만 작업할 것.
  - 보호 경로 회피: `internal/auth/*`, `internal/api/oauth.go`, `internal/database/migrations`, `Dockerfile`, `.github/workflows/*` 는 건드리지 말 것. 이 과제는 그 어느 것도 필요 없다.
  - 새 목(mock)을 만들지 말 것. `internal/tools/integration_test.go` 의 `fakeBitbucket` + `newFixture` 가 Executor·Resolver·Policy·Approval·Audit 실제 타입을 쓰는 기존 배선 하네스다.
  - 마이그레이션 불필요 — `arguments_redacted JSONB` 는 이미 있다(`0001_init.sql:137`). 기존 행을 소급 정리하려 하지 말 것.
- 차선 후보: `internal/mcp/jsonrpc.go` 프레이밍 단위 테스트 추가 (가치 3 / 위험 1 / 작업량 S) — `internal/mcp/` 에 테스트 파일이 0건이고(`jsonrpc.go` 74행, `server.go` 372행) DB 가 필요 없어 로컬에서 바로 돈다. 1순위가 성립하지 않을 때(예: `redactArgs` 가 이미 고쳐져 있을 때) 이것을 고를 것.
