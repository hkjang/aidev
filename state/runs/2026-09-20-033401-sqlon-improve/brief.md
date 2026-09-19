# 과제서 — 2026-09-20 sqlon

- 과제: REST 변경 관리·DBA 콘솔의 행위자가 항상 "dba" 로 기록되는 결함 수정 + `adminAudit` 에 actor 필드 추가 (가치 4 / 위험 1 / 작업량 S)

- 왜: main(57f99b7) 에서 `/api/*` 핸들러는 어떤 미들웨어도 인증 사용자를 요청 컨텍스트에 싣지 않는다(`withUser` 호출은 `server.go:220` 의 `/mcp` 경로와 `admin.go:992` `fleetProfilesForRequest` 두 곳뿐). 그래서 `POST /api/changes/{id}/approve|execute|rollback`(admin.go:444·480·504) 의 `userFrom(r.Context())` 가 항상 nil 이라 actor 가 "dba" 로 고정되고, `critical`(승인 2건 필요, `change.ApprovalRequirement`) 계획은 두 번째 승인자가 `approver "dba" has already approved` 로 거부되어 REST 로는 승인을 끝낼 수 없다. 같은 이유로 `adminAudit`(admin.go:1130) 은 `remote` 주소만 남겨 누가 무엇을 바꿨는지 감사에서 알 수 없다. 고치면 승인 흐름이 실제로 동작하고 REST 감사가 MCP 감사(actor 있음)와 같은 형식이 된다.

- **지름길(먼저 시도할 것)**: 같은 수정이 로컬 브랜치 `auto/2026-09-17-0933` 의 커밋 `a47973b`(fix(mcp): REST 변경 관리 승인·감사 행위자를 인증 사용자로 기록합니다)에 이미 있고, 그 커밋의 부모가 정확히 main 57f99b7 이라 `git cherry-pick a47973b` 가 깨끗이 적용될 것이다(미확인: 실제 cherry-pick 은 해 보지 않았음). 변경 내용: admin.go 47줄, dbaapi.go 3줄, changes_api_test.go +70(두 dba 순차 승인 테스트), CHANGELOG. cherry-pick 뒤 `go test ./internal/mcp/` 가 통과하면 그 위에 아래 2단계(adminAudit actor)만 더 얹으면 된다. cherry-pick 이 안 되거나 테스트가 깨지면 아래 설계대로 손으로 만든다.

- 수용 기준:
  1) 메타 DB 모드(`newAuthServer` 방식)에서 dba 역할 사용자 둘(예: dan·erin)이 `risk:"critical"` 계획을 `/api/changes` 로 만들고 submit 한 뒤 차례로 `POST /api/changes/{id}/approve` 하면 — 첫 승인 후 상태 `review_required` + `approvals[0].actor=="dan"`, 같은 사용자가 다시 승인하면 400 "already approved", 두 번째 사용자가 승인하면 `approved` + approvals 2건. 미인증은 401, `user` 역할은 403.
  2) 위 흐름에서 `adminAudit` 이 남기는 JSONL 항목(`appendAudit`, `tool:"admin:…"`)에 `actor` 필드가 인증 사용자 이름으로 들어간다(메타 DB 모드). 단독 모드(마스터 토큰)에서는 actor 없음 또는 빈 값 — 기존 `TestAdminAPILifecycle`/`TestAdminTokenEnforcedOnMutations` 가 그대로 통과해야 한다.
  3) 테스트는 **프로덕션 배선**(실제 `mux` + 세션 쿠키 로그인 + `requireDBA`/`requireAdmin` 경로)을 통과해야 한다. 컨텍스트에 사용자를 손으로 넣은 요청이나 소스 문자열 검사로 증명하지 말 것. 감사 JSONL 은 `newFixtureServer` 가 주는 dir 아래 실제 파일을 읽어 단언한다(기존 admin_test 의 감사 검증 방식을 따를 것 — 파일 경로는 admin_test.go/audit.go 에서 확인, 미확인).

- 건드릴 파일:
  - `internal/mcp/admin.go:requireDBA`(1106) — `fleetProfilesForRequest`(985) 와 같은 방식으로 인증 사용자를 실은 `*http.Request` 를 돌려주도록 서명 변경(예: `(*http.Request, bool)`), 호출부 11곳 갱신(`grep -n "requireDBA(w, r)"`).
  - `internal/mcp/dbaapi.go` — requireDBA 호출부 1곳 갱신.
  - `internal/mcp/admin.go:requireAdmin`(1065) — 같은 방식으로 사용자를 컨텍스트에 실어 돌려주기. 호출부 29곳(admin.go 19, authapi.go 6, dbapi.go 4). **주의**: requireDBA 가 단독 모드에서 `s.requireAdmin(w, r)` 로 위임하므로 서명을 맞출 것. 29곳이 부담이면 requireAdmin 은 그대로 두고 `adminAudit` 이 컨텍스트에 사용자가 있을 때만 actor 를 쓰는 것으로 이번 회차를 끝내도 수용 기준 1)·2) 는 충족된다(2) 는 DBA 경로에서 검증). 그 경우 journal 에 "requireAdmin 경로는 다음 회차" 라고 적을 것.
  - `internal/mcp/admin.go:adminAudit`(1130) — `if u := userFrom(r.Context()); u != nil { entry["actor"] = u.Username }`. MCP 감사가 쓰는 필드명이 `actor` 인지 `audit.go`/`server.go` 에서 확인해 같은 이름을 쓸 것(미확인).
  - `internal/mcp/changes_api_test.go` — 두 dba 순차 승인 + actor 단언 + 401/403 테스트(a47973b 에 이미 있음). `newAuthServer`(authhttp_test.go:17) 를 본떠 dba 계정 둘을 `svc.CreateLocalUser(..., meta.RoleDBA, ...)` 로 만들고 `/auth/login` 세션 쿠키(`withCookie`)로 요청. `newFixtureServer` 가 `s.Changes` 를 이미 채운다(changes_api_test 가 그것으로 동작함).
  - `CHANGELOG.md` Unreleased 에 한 줄.

- 검증 명령:
  - `go test ./internal/mcp/ -run 'TestChange|TestAdmin|TestAuth' -count=1`
  - `go vet ./... && go test ./...` (전체 약 1~2분, 미확인)
  - `gofmt -l internal/mcp/admin.go internal/mcp/dbaapi.go internal/mcp/changes_api_test.go` — 손댄 파일만(저장소의 다른 Go 파일 ~90개가 CRLF 라 전체 `gofmt -l` 은 의미 없음).

- 위험과 피할 것:
  - `internal/mcp/auth.go` 의 `authenticate`/`guard`/세션 로직은 건드리지 말 것 — 인증 판정을 바꾸는 게 아니라 이미 판정된 사용자를 컨텍스트에 싣는 일만 한다.
  - `canUseProfileID`(authapi.go:701) 는 `u == nil` 을 "신뢰"로 본다. requireDBA/requireAdmin 이 사용자를 싣기 시작하면 그 뒤에 오는 `canUseProfileID(r.Context(), userFrom(r.Context()), …)`(admin.go:723·731·760 등) 가 갑자기 권한 검사를 시작할 수 있다 — 이는 의도된 강화지만 기존 테스트가 깨질 수 있으니 깨지면 원인을 적고(테스트가 admin 이 아닌 계정으로 남의 프로파일을 읽던 것인지) 판단할 것. 같은 값을 읽는 경로를 모두 end-to-end 로 확인하라는 운영자 지시에 해당한다.
  - 단독 모드(메타 DB 없음)에서 `userFrom` 은 nil 이어야 하며 행동이 바뀌면 안 된다.
  - 감사 항목에 요청 본문·SQL 등 원문을 넣지 말고 사용자 이름(식별자)만 넣을 것(운영자 지시).
  - 마이그레이션·워크플로·auth 세션 저장 방식은 무관하므로 손대지 말 것.

- 차선 후보: docs/auth.md·rest-api.md·security.md 채우기 (3/1/S) — 세 파일이 모두 0바이트이고 README 가 docs/auth.md 로 링크한다. `internal/mcp/auth.go` 상단 주석·`authapi.go`·`openapi.go`·`docs/admin_guide.md` §3 를 근거로 인증 모델(로컬 계정·Keycloak OIDC·MCP 키 `ssk_`·마스터 토큰)·REST 인증 헤더·변경 관리 승인(critical=2명)·감사 로그 위치를 정리. API 메서드는 반드시 소스(`mux.HandleFunc("METHOD /path"`)에서 확인해 적을 것(운영자 지시). 코드 변경 없음.
