- 과제: PR #34(`5915beb`, resolveReport transaction화)의 CI `Source tests` 실패를 원인대로 고친다 (가치 4 / 위험 2 / 작업량 M)
- 왜: 2026-09-28 12:31 러너가 `ci failed — 성공이 아닌 검사: Source tests=failure`로 끝나 `5915beb`이 main에 들어가지 못했고, 같은 패턴으로 `972113f`(2026-09-25)·`ab36254`도 main에 못 들어온 채 사라졌습니다. 이 회차가 원인을 특정해 고치지 않으면 resolveReport 수정이 또 버려집니다.
- 수용 기준:
  1) CI `source` 잡이 실제로 실행하는 **네 단계 전부**를 로컬에서 재현해 실패를 눈으로 재현한다 — 특히 지난 회차가 건너뛴 둘: (a) `go run honnef.co/go/tools/cmd/staticcheck@2025.1.1 ./...`(비평 노트: "staticcheck는 미설치로 미실행"), (b) `MOINA_TEST_POSTGRES_DSN`을 **postgres:16-alpine**으로(지난 회차는 17로만 검증, CI `source`는 16 — `.github/workflows/ci.yml:21`).
  2) 실패의 **원인 한 줄**을 journal에 적고(어느 단계·어느 파일·어느 메시지), 그 원인을 고친다. 고친 뒤 같은 네 단계를 다시 돌려 전부 통과함을 출력으로 보인다.
  3) 워크플로를 느슨하게 만들어 통과시키지 않는다 — `ci.yml`의 단계 제거·`continue-on-error` 추가·테스트 제외·`t.Skip` 확대·staticcheck 지시어(`//lint:ignore`) 남용은 금지. 고칠 곳은 `admin.go` 또는 새 테스트 파일이다.
  4) `5915beb`이 고친 행동 계약(저장 실패 500 `storage_error`, 0행 404 `not_found`, 제재 기록과 같은 transaction)은 그대로 유지되고, 새 테스트 6케이스가 여전히 6/6 PASS·`--- SKIP` 0줄이다.

- 시작 지점(확인된 사실):
  - PR #34 = branch `auto/2026-09-28-1202` = commit `5915beb` (로컬에 있음: `git show 5915beb`). 이 worktree의 HEAD는 `d23f75d`(main)이므로 **먼저 `5915beb`을 가져와야** 한다: `git cherry-pick 5915beb` 또는 `git checkout -b fix auto/2026-09-28-1202`.
  - `5915beb`이 건드린 파일은 3개뿐: `api/openapi.yaml`(+3 description), `backend/internal/httpapi/admin.go`(resolveReport 27줄), `backend/internal/httpapi/admin_report_resolve_postgres_integration_test.go`(신규 234줄).
  - CI 로그는 이번 정찰에서 **읽지 못했다**(이 샌드박스에서 `gh` 실행이 권한 거부). 실패 단계 이름은 러너 요약의 `Source tests=failure` 한 줄뿐 — **어느 step인지는 미확인**. 구현자가 `gh run list --repo hkjang/moina` / `gh run view --log-failed`를 쓸 수 있으면 **그것을 먼저** 보고 아래 가설을 건너뛰어라.
  - `source` 잡의 step 순서(`.github/workflows/ci.yml:36~93`): Checkout → setup-go → setup-node → `make check` → (`make fmt`; `cd backend`; `go test -race ./...`; `go vet ./...`) → frontend(`npm ci`/lint/test/build) → `staticcheck@2025.1.1 ./...` → Dependency audit(`continue-on-error: true`, 실패해도 잡을 안 깨뜨림).
    - 따라서 **Dependency audit은 범인이 될 수 없다**. 남는 후보는 `make check`·Go 단계·frontend 단계·staticcheck 넷.

- 가설 순위(높은 것부터 — 전부 "미확인", 구현자가 재현으로 판정할 것):
  1. **staticcheck** — 지난 회차가 유일하게 "미실행"이라고 적은 단계이고, 프런트·`make check`·`go test`는 로컬에서 통과했다. 새 테스트 파일(234줄)이나 `resolveReport`의 새 코드에 지적이 날 가능성. 특히 `defer tx.Rollback(r.Context())`의 반환값 무시는 같은 파일 `softDeletePost`(admin.go:306 부근)와 같은 형태이므로 그쪽이 통과한다면 이 형태는 아니다 — 새 테스트 쪽을 먼저 보라.
  2. **PostgreSQL 16 vs 17** — 지난 회차 검증은 `postgres:17-alpine`으로만 했고 CI `source`는 `postgres:16-alpine`이다. 새 테스트의 `CREATE FUNCTION … RETURNS trigger LANGUAGE plpgsql`·`CREATE TRIGGER … EXECUTE FUNCTION`·`ANY($1::text[])`는 16에도 있는 문법이라 이 가설은 약하지만, 검증 비용이 싸므로 무조건 16으로 돌려 확인하라.
  3. **CI 병렬 실행에서의 공유 DB 경합** — CI는 `moina_ci` 하나를 `go test -race ./...` 전 패키지가 공유하고, 새 테스트는 `reports`·`moderation_actions`에 `CREATE TRIGGER`(ACCESS EXCLUSIVE 잠금)를 건다. 다른 패키지가 같은 테이블을 잡고 있으면 블록·타임아웃(`timeout-minutes: 20`)이 될 수 있다. 재현하려면 `go test -race -count=1 ./...`를 **빈 DB 하나에 전 패키지 동시로** 돌려라(개별 패키지만 돌리면 이 경합은 안 보인다).
  4. **테스트 잔여물** — 새 테스트는 트리거를 Cleanup에서 DROP 하지만, 앞선 실패로 중단되면 남는다. CI는 매번 새 DB이므로 CI 실패 원인은 아니다. 로컬 재현 시에는 매번 새 컨테이너를 쓰라(아래 명령).

- 건드릴 파일 (프로덕션 2개 이하로 끝낼 것):
  - `backend/internal/httpapi/admin.go:378~` `resolveReport` — 원인이 프로덕션 코드일 때만.
  - `backend/internal/httpapi/admin_report_resolve_postgres_integration_test.go` — 원인이 테스트일 때(가장 가능성 높음).
  - `.github/workflows/ci.yml` — **읽기만**. 원인이 워크플로 자체의 버그(예: 잘못된 env·순서)임이 출력으로 증명되지 않는 한 수정하지 말 것.
  - 원인이 위 셋 중 어디에도 없고 `5915beb` 이전에도 실패한다면(= main도 빨갛다면) 그 사실을 journal에 적고 main 기준의 실패를 고쳐라 — 그때는 `5915beb` 재적용을 범위에서 빼라.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  ```
  # 0) 대상 커밋 확보
  git cherry-pick 5915beb        # 또는 git checkout -b fix auto/2026-09-28-1202

  # 1) CI와 같은 PostgreSQL 16으로 새 DB (docker 사용 가능 확인됨)
  docker run -d --rm --name moina-ci16 -e POSTGRES_DB=moina_ci \
    -e POSTGRES_USER=moina -e POSTGRES_PASSWORD=moina-ci-password \
    -p 55432:5432 postgres:16-alpine
  export MOINA_TEST_POSTGRES_DSN='postgres://moina:moina-ci-password@127.0.0.1:55432/moina_ci?sslmode=disable'

  # 2) CI source 잡의 Go 단계 그대로
  make check
  make fmt
  cd backend && go test -race ./... && go vet ./...

  # 3) 지난 회차가 건너뛴 단계 — 여기서 실패할 가능성이 가장 높다
  cd backend && go run honnef.co/go/tools/cmd/staticcheck@2025.1.1 ./...

  # 4) 프런트 단계 (node_modules 없으면 npm ci 먼저)
  npm ci --prefix frontend && npm run lint --prefix frontend && npm test --prefix frontend
  VITE_MOINA_VERSION="$(tr -d '[:space:]' < VERSION)" npm run build --prefix frontend

  # 5) 새 테스트만 다시 (증거용: --- SKIP 0줄, 6/6 PASS)
  cd backend && go test -race -count=1 -run TestPostgreSQLAdminResolveReport -v ./internal/httpapi/
  ```
  - `docker run` 뒤에는 `pg_isready`가 될 때까지 기다려라. 끝나면 `docker rm -f moina-ci16`.
  - `--- SKIP` 0줄을 반드시 확인하라. DSN 없으면 새 테스트가 조용히 통과처럼 보인다(테스트 파일 첫 `t.Skip`).

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들지 말 것**(운영자 지침: 릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과를 확인할 것). 임계값·`continue-on-error`·테스트 제외·재시도 추가는 반려 사유다.
  - **grep 증거 금지** — "문자열이 있다"가 아니라 명령 출력으로 실패와 통과를 보여라.
  - 손으로 만든 대역 금지 — 새 테스트는 실제 `New(repository, secrets, …)` + `server.Handler()` 배선을 지난다. 그 방식을 유지하라.
  - `store/migrations`·`auth.go`·`oidc.go`·`mcp_oauth.go`는 건드리지 말 것. `api/openapi.yaml`의 `responses` 목록을 늘리지 말 것(관례: `description` 한 줄만).
  - `docker` 컨테이너를 남기지 말고, 기존 컨테이너(`ptium-chart-pg` 등 다른 프로젝트 것이 돌고 있음)를 건드리지 말 것 — 반드시 새 이름·새 포트를 쓸 것.
  - `git stash` 금지(다른 세션과 스택 공유). 필요하면 WIP 커밋을 쓸 것.

- 차선 후보: `followTopic`(social.go:282)의 `if err != nil || tag.RowsAffected() == 0` 통합 404를 500 `storage_error`와 404로 분리 (가치 2 / 위험 2 / 작업량 S) — 이 저장소에 남은 마지막 통합 404이고, `972113f`가 이미 같은 수정을 했으나 verify-failed로 사라졌다. 1순위가 "main 자체가 빨갛다"로 무너질 때만 고를 것.
