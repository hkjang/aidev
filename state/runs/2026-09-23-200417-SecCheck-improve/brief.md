# 과제서 — 2026-09-23-200417-SecCheck-improve (수정 과제)

- 과제: 수정 과제(7회째) — PR #16 의 CI 실패를 **오늘의 실제 출력으로** 재판정하고, 경로 A(저장소 안 원인)면 그것을 고치고, 경로 B(vulndb GO-2026-6452 외부 차단)면 커밋 0개로 확정 (가치 4 / 위험 1 / 작업량 S)

- 왜: 릴리즈가 막혀 있는데, 이 저장소에서 같은 자리(security-ci step 7 `Go vulnerability scan`)가 09-19~09-20 에 여섯 번 연속 빨강이었고 그 원인은 저장소 밖(golang/vulndb 의 GO-2026-6452 에 `fixed` 이벤트가 없어 excelize v2.11.0 이 계속 취약으로 판정됨)이었다. 이번 회차의 정찰은 **네트워크 접근이 거부되어**(아래 "미확인") 원격 CI·vulndb 를 조회하지 못했으므로, 구현자가 먼저 실제 출력으로 원인을 확정한 뒤에만 코드를 건드려야 한다. 원인이 저장소 안이면 고치고, 밖이면 아무것도 바꾸지 않는 것이 이번 회차의 올바른 결과다.

## 0단계 — 판정 (코드를 건드리기 전에 반드시 먼저, 세 명령 모두 실제 출력을 원장에 붙일 것)

(a) vulndb 에 `fixed` 가 생겼는가
```
curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.modified, [.affected[].ranges[].events]'
```
- 09-20 마지막 확인값: `"2026-09-16T18:00:43Z"`, `[[{"introduced":"0"}]]` (= `fixed` 없음). 이번 회차 **미확인**.

(b) PR #16 의 security-ci 가 **어느 step 에서** 죽었는가
```
curl -s "https://api.github.com/repos/hkjang/SecCheck/actions/runs?branch=auto/2026-09-23-1834" | jq '.workflow_runs[] | {id,name,head_sha,status,conclusion,created_at}'
curl -s "https://api.github.com/repos/hkjang/SecCheck/actions/runs/<RUN_ID>/jobs" | jq '.jobs[].steps[] | select(.conclusion=="failure") | {number,name}'
```
- 실패 step 이 **7 `Go vulnerability scan`** 이면 과거 6회와 같은 자리다.
- **7 이 아닌 다른 step 이면 경로 A** — 그 step 의 명령을 그대로 로컬에서 재현해서 고칠 것.

(c) main 자체가 이 게이트에서 빨강인가 (= 브랜치와 무관한 외부 차단인지)
```
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...        # origin/main 트리에서
```
- 09-20 마지막 확인값: exit 3, `Found in: github.com/xuri/excelize/v2@v2.11.0 / Fixed in: N/A / internal/store/seed.go:66:25: store.ExtractWorkbookDefaults calls excelize.File.GetRows`.
- **이번 회차에 로컬에서 확인한 사실**: `go.mod` 는 여전히 `github.com/xuri/excelize/v2 v2.11.0`, 호출부도 여전히 `internal/store/seed.go:66` 의 `f.GetRows(spec.Sheet)` 이다. 즉 (a) 가 여전히 `fixed` 없음이면 (c) 는 필연적으로 exit 3 이다.

### 판정 → 경로
- **경로 A** — 실패 step 이 7 이 아니거나, (a) 에 `fixed` 가 생겼는데도 여전히 빨강: 저장소 안 원인이다. 그 step 을 로컬에서 재현해 고치고 아래 "검증 명령" 을 돌린다.
- **경로 B** — 실패 step = 7 이고 (a) 에 `fixed` 없음이고 (c) 가 origin/main 에서도 exit 3: **저장소 안 해결 수단 없음. 커밋 0개로 끝낸다.** `go.mod`·워크플로·`replace`·vendor·excelize 호출 분리는 **전부 금지**(게이트 우회). 원장에 세 출력을 그대로 붙이고 운영자 결정 사항(golang/vulndb PR — 09-20 회차 디렉터리의 `vulndb-fix-proposal.md` 가 본문 초안)을 적는다.

## 이번 회차에 로컬에서 확인한 사실 (구현자가 다시 읽지 않아도 되는 것)
- PR #16 = 브랜치 `auto/2026-09-23-1834`, 커밋 `f4df4c2` "Scope the MCP review report by its tool arguments alone" (추정: PR 번호↔브랜치 대응은 **미확인**, 0단계 (b) 에서 `head_sha` 로 확인할 것).
- `git diff --stat origin/main..origin/auto/2026-09-23-1834` = `internal/web/mcp.go` (+10/-1) 와 `internal/web/mcp_report_test.go` (신규 169줄) **둘뿐**. `go.mod`·`go.sum`·`.github/workflows/`·`internal/auth`·`internal/store/migrations` 는 **한 줄도 바뀌지 않았다**. → 이 diff 는 step 7(govulncheck)·step 5(secret scan)·step 8~14(이미지/trivy/SBOM/DAST) 중 무엇도 새로 깨뜨릴 수 없다. 깨질 수 있는 곳은 사실상 step 4 `Unit and database integration tests` 하나다.
- `release.yml:62 "Require a green security-ci run for this commit"` 가 security-ci 의 결론이 `success` 가 아니면 `exit 1` 한다. **즉 "릴리즈 워크플로가 두 번 실패" 는 릴리즈 자체의 결함이 아니라 security-ci 가 빨강이라는 사실의 그림자일 수 있다.** 릴리즈 워크플로 자체를 고치려 들기 전에 0단계 (b) 로 security-ci 결론부터 볼 것.
- `VERSION` = `1.0.146`, 최신 태그 = `v1.0.146` (로컬 기준). `ci.yml` 이 `1.0.146` 을 하드코딩(74·78·91행)하고 있으나 이는 main 과 일치하므로 지금 실패 원인이 아니다.

## 경로 A 일 때 가장 먼저 볼 곳 (step 4 가 실패했다면)
신규 `internal/web/mcp_report_test.go` 에서 CI(=`go test -race ./...`, 실 PostgreSQL 16)에서만 깨질 수 있는 자리 — **모두 추측이며 이번 회차에 실행으로 확인하지 않았다**:
1. `after_value->'arguments' ? 'department'` — JSONB 존재 연산자 `?` 를 pgx 확장 프로토콜로 보낸다. 플레이스홀더로 오해되면 쿼리 자체가 실패한다. 실패하면 `jsonb_exists(after_value->'arguments','department')` 로 바꾸는 것이 같은 뜻의 안전한 표기다.
2. `ORDER BY chain_sequence LIMIT 1` 로 "첫 MCP_TOOL_CALL 감사 행" 을 고르는 두 쿼리 — 같은 테스트 안에서 `callTool` 이 여러 번 호출되므로 행 순서 가정이 깨지면 엉뚱한 행을 읽는다.
3. `follow_up_due_date current_date+30`, `now()` — DB 시간대(CI 는 UTC)에 의존.
4. `by_department` 가 2행이라는 단언 — `createReview` 의 기본 부서(보안팀)와 `UPDATE ... department='인프라팀'` 두 개를 전제한다.
5. `-race` 아래에서만 나오는 경합 — 로컬에서 `-race` 없이 돌렸다면 재현되지 않는다.

## 수용 기준
1. 0단계 세 명령의 **실제 출력**(잘라내지 말 것)이 원장에 있고, 경로 A/B 중 어느 쪽인지 한 줄로 명시되어 있다.
2. 경로 B 면: `git log origin/main..HEAD --oneline` 이 **빈 출력**이고 `git status --porcelain` 이 빈 출력이다(커밋 0개). `git diff origin/main -- .github/workflows/ go.mod go.sum` 이 **0바이트**.
3. 경로 A 면: 고치기 **전에** 실패를 로컬에서 재현한 출력과, 고친 **뒤** 같은 명령이 통과하는 출력이 둘 다 원장에 있다(수정을 되돌리면 다시 실패하는 것까지 확인).
4. 어느 경로든 `.github/workflows/` 는 바뀌지 않았다. 게이트의 임계값·`exit-code`·`severity`·`continue-on-error`·`--audit-level` 을 건드린 흔적이 없다.

## 건드릴 파일
- 경로 B: **없음**(파일 0개). 회차 디렉터리 원장만.
- 경로 A(step 4 라면): `internal/web/mcp_report_test.go` — 위 1~5 중 실제로 깨진 자리만. 필요하면 `internal/web/mcp.go:mcpReviewReport`(현재 `url.Values{}` 에서 출발).
- 경로 A(다른 step 이라면): 그 step 의 명령이 가리키는 파일. `internal/auth/*`, `internal/store/migrations/*`, `.github/workflows/*` 는 건드리지 말 것.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
# DB 필요 — 전용 컨테이너로 DSN 을 실제로 세울 것 (없으면 통합 테스트가 조용히 SKIP 되어 PASS 로 보인다)
docker run -d --rm --name seccheck-recon -e POSTGRES_PASSWORD=seccheck_ci -e POSTGRES_DB=seccheck -p 5433:5432 postgres:16-alpine
export TEST_POSTGRES_DSN='postgres://postgres:seccheck_ci@127.0.0.1:5433/seccheck?sslmode=disable'

go test ./internal/web -run 'MCP|Report' -count=1 -v      # 해당 테스트만
go test -race ./internal/web -run 'MCP|Report' -count=1   # CI 와 같은 -race
go test ./... -count=1                                     # 전 패키지 (internal/web 약 3분)
go vet ./...
govulncheck ./...                                          # step 7 재현 (경로 판정용)
bash scripts/precheck.sh                                   # gofmt·vet·Go 테스트·프런트·그림·PDF·gitleaks
```
- `go test` 출력에 `SKIP` 이 몇 개인지 반드시 적을 것. DSN 없는 PASS 를 DB 검증으로 오인하는 것이 이 저장소의 상습 실수다.
- `scripts/precheck.sh` 의 프런트엔드 단계는 `web/node_modules` 가 없으면 건너뛴다. 프런트를 건드렸다면 `npm ci --prefix web && npm --prefix web test --silent && npx tsc --noEmit` 을 따로 돌릴 것.

## 위험과 피할 것
- **워크플로 완화 금지.** `govulncheck` 를 빼거나 `|| true` 를 붙이거나 `-severity` 를 낮추거나 trivy 의 `exit-code: 1` 을 0 으로 바꾸는 것은 전부 금지다. 이번 회차 지시에 명시되어 있다.
- **excelize 우회 금지.** `go.mod` 의 `replace`, vendor 디렉터리, `internal/store/seed.go:66` 의 `GetRows` 호출을 reflection·간접 호출로 숨겨 govulncheck 의 도달 분석을 피하는 것은 취약점을 고치는 게 아니라 스캐너를 속이는 것이다. 과거 회차에서 이미 금지로 분류되었다.
- `go list -m -versions github.com/xuri/excelize/v2` 로 v2.11.0 뒤 태그가 새로 나왔는지는 볼 것 — **새 태그가 실제로 있고 그것이 GO-2026-6452 를 고친다면 그것은 우회가 아니라 정당한 수정이다.** 09-20 기준으로는 뒤 태그가 없었고, 이번 회차에는 미확인.
- 보호 경로: `internal/auth/*`, `internal/web/core_handlers.go` 의 인증 함수, `internal/store/migrations/*`, `.github/workflows/*`.
- 과거 교훈: (1) DSN 없는 SKIP 을 통과로 착각, (2) 미병합 PR 의 코드가 main 에 있다고 가정(PR #13 의 `routes.test.ts` 는 지금도 main 에 없다), (3) 감사 컬럼명은 `details` 가 아니라 `audit_logs.after_value`, (4) 감사 테이블명은 `audit_events` 가 아니라 `audit_logs`.

## 차선 후보
**MCP `numberValue` 가 범위 밖 `limit` 을 조용히 기본값 50 으로 되돌리는 것 고치기** (가치 2 / 위험 1 / S) — `internal/web/mcp.go:numberValue`. 도구 스키마는 `limit` 을 1~100 integer 로 선언하는데 `limit:1000` 이나 문자열 `"5"` 가 오류 없이 50 이 된다(`list_reviews`·`search_controls` 둘 다). PR #16 이 만든 `mcp_report_test.go` 의 `client.callTool` 헬퍼로 바로 검증 가능 — 단 그 헬퍼는 **아직 main 에 없다**(PR #16 미병합). 경로 B 가 확정되어 손댈 코드가 없을 때에만, 그리고 이 과제도 같은 빨간 게이트에 걸린다는 점을 알고 고를 것.

## 미확인 (정찰이 확인하지 못한 것 — 사실로 쓰지 말 것)
- 이번 회차 정찰 세션은 `curl`·`WebFetch` 가 **모두 권한 거부**되어 네트워크에 나가지 못했다. 따라서 **PR #16 의 실제 실패 step, 오늘의 vulndb GO-2026-6452 상태, release.yml 이 실제로 두 번 실패했는지, 실패한 run id 는 전부 미확인**이다. 위의 09-20 값들은 과거 원장에서 인용한 것이지 오늘 확인한 것이 아니다.
- PR 번호 16 ↔ 브랜치 `auto/2026-09-23-1834` 대응은 브랜치 목록과 회차 순서로 추정한 것이다. 0단계 (b) 에서 `head_sha` 로 확인할 것.
- `go test`·`govulncheck`·`precheck.sh` 를 이번 회차에 실행하지 않았다(정찰은 코드를 바꾸지 않고 판정만 남긴다). 초록·빨강 어느 쪽도 주장하지 않는다.
