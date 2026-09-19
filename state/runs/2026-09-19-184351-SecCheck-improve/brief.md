# 과제서 — 2026-09-19 (수정 과제)

- 과제: PR #8(`auto/2026-09-19-0143`, 693b733)의 security-ci 실패 원인을 실패 로그에서 확정하고 원인을 고쳐 같은 게이트를 로컬에서 통과시키기 (가치 4 / 위험 2 / 작업량 S~M)
- 왜: 자동 배정이 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 하지만, 정찰 환경에서는 `gh`·`curl`·WebFetch 가 모두 승인 거부되어 **실패한 워크플로·단계·로그를 직접 보지 못했다(미확인)**. PR #8 의 diff 는 9파일(테스트 1개·Settings.tsx·capture_all.js·가이드 2줄·PNG/PDF 4개)로 작고 이전 회차가 DSN 포함 precheck.sh 전체 통과를 확인했으므로, 실패는 precheck 가 돌리지 않고 CI 만 돌리는 단계(아래 표)에 있을 가능성이 높다. 구현자는 **추측으로 고치지 말고 로그부터 읽어야 한다.**

## 0단계 — 원인 확정 (반드시 먼저, 5분)
```bash
git fetch origin && git checkout auto/2026-09-19-0143
gh pr view 8 --json statusCheckRollup,headRefName,url
gh run list --workflow=ci.yml --branch auto/2026-09-19-0143 --limit 5
gh run view <run-id> --log-failed | head -200
# 릴리즈 워크플로도 실제로 실패했는지 확인(과제 문구의 전제 검증 — 09-19 이전 회차는 release.yml 6건이 전부 success 였음)
gh run list --workflow=release.yml --limit 5
```
실패한 step 이름을 회차 노트에 그대로 적을 것. **PR #8 브랜치에서 실패했더라도 main(eb2a3b0)에서도 같은 단계가 실패하는지** `gh run list --workflow=ci.yml --branch main --limit 3` 로 확인 — main 도 실패하면 PR 의 잘못이 아니라 환경 게이트(새 CVE)다.

## 후보 원인 — precheck.sh 는 돌리지 않고 ci.yml 만 돌리는 단계 (가능성 순)
| 순위 | ci.yml 단계 | 로컬 재현 명령 | 고치는 방향 |
|---|---|---|---|
| 1 | `Go vulnerability scan` — `govulncheck@latest ./...` (go.mod 는 `go 1.26.6`) | `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...` | 표준 라이브러리 취약점이면 go.mod `go` 지시어·toolchain 을 패치 릴리즈(1.26.x)로 올림, 모듈이면 `go get <모듈>@<고친 버전> && go mod tidy`. 교훈: 게이트를 푸는 게 아니라 취약점을 고친다 |
| 2 | `Container vulnerability gate`(trivy CRITICAL/HIGH, ignore-unfixed) · `Scan SBOM`(grype) | `docker build --build-arg VERSION=1.0.146 -t seccheck:v1.0.146 . && trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 seccheck:v1.0.146` | 2026-09-13 교훈과 같은 자리: Dockerfile 이 이미 `apt-get upgrade` 를 한다면 베이스 이미지 태그를 최신 digest 로, Go 바이너리 쪽이면 1번과 같이 의존성 올리기 |
| 3 | `Frontend build and dependency gate` — `npm audit --prefix web --audit-level=high` | `cd web && npm ci && npm audit --audit-level=high` | 취약 패키지의 고친 버전으로 `npm update <pkg>` 또는 package.json 범위 조정 후 `package-lock.json` 재생성. 메이저 업그레이드 금지 |
| 4 | `Unit and database integration tests` — `go test -race ./...` (precheck 는 `-race` 없음) | `TEST_POSTGRES_DSN=… go test -race -run 'TestTheTestMailGoesToTheAddressTheScreenNames\|TestMailSettings' ./internal/web/` 그리고 `go test -race ./...` | PR 이 더한 `internal/web/mail_test.go:TestTheTestMailGoesToTheAddressTheScreenNames` 가 죽은 릴레이(127.0.0.1:1, timeout 1초)를 치는데 -race 로 느려져 타이밍이 어긋나거나 데이터 레이스가 보고될 수 있음. 레이스면 원인 코드(internal/mail 또는 핸들러)를 고치고, 테스트를 지우거나 skip 하지 말 것 |
| 5 | `DAST baseline`(ZAP) / 컨테이너 smoke | 로컬 재현 어려움 — 로그의 규칙 ID 를 `.zap/rules.tsv` 와 대조 | PR 이 화면에 `type="email"` 입력 하나를 더했을 뿐이라 가능성 낮음 |

## 수용 기준
1) 회차 노트에 **실패한 워크플로 이름·run id·step 이름·핵심 오류 줄**이 인용되어 있다(추측 아님).
2) 그 단계와 같은 명령을 로컬에서 재현해 고치기 전 실패 → 고친 뒤 통과를 둘 다 보였다.
3) `.github/workflows/ci.yml`·`release.yml` 의 게이트(severity·exit-code·audit-level·-race)는 한 글자도 완화하지 않았다(diff 로 증명). 단계를 더하는 것은 허용.
4) `TEST_POSTGRES_DSN=… bash scripts/precheck.sh` 전체 통과 + 고친 단계의 CI 명령(`go test -race ./...` 포함) 통과.
5) 원인이 main 에도 있는 환경 게이트(새 CVE)였다면 수정 커밋은 PR #8 브랜치 위에 얹어 같은 PR 에서 CI 가 green 이 되게 한다. 원인이 PR #8 의 코드였다면 그 코드를 고친다.

## 건드릴 파일 (원인에 따라 하나만)
- 후보 1: `go.mod`(`go`/`toolchain` 지시어 또는 require 버전), `go.sum`
- 후보 2: `Dockerfile`(베이스 이미지 digest / apt-get upgrade 줄) — 13e2cc0·c24438e 가 지난번에 한 방식 참고
- 후보 3: `web/package.json`, `web/package-lock.json`
- 후보 4: `internal/web/mail_test.go`(타이밍 여유) 또는 레이스의 근원 코드 `internal/mail/mail.go`·`internal/web/core_handlers.go`
- 어느 경우든 회차 노트(journal.md)에 원인·단계·명령을 기록 — 원장에 '수정 과제' 로 남긴다.

## 검증 명령
```bash
# Postgres (없으면): docker run -d --name pg-sc -e POSTGRES_PASSWORD=pw -e POSTGRES_DB=seccheck -p 55432:5432 postgres:16-alpine
export TEST_POSTGRES_DSN='postgres://postgres:pw@127.0.0.1:55432/seccheck?sslmode=disable'
go test -race ./... && go vet ./...
go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
npm ci --prefix web --no-audit --no-fund && npm --prefix web test --silent && npm --prefix web run build && npm audit --prefix web --audit-level=high
docker build --build-arg VERSION=1.0.146 -t seccheck:v1.0.146 . && trivy image --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1 seccheck:v1.0.146
bash scripts/precheck.sh
git diff main -- .github/workflows/   # 비어 있어야 함(완화 금지)
```

## 위험과 피할 것
- 워크플로 완화 금지(운영자 규칙·과제 문구 모두). `ignore-unfixed` 를 더 넓히거나 severity 를 낮추거나 `continue-on-error` 를 넣지 말 것.
- 의존성 메이저 업그레이드 금지 — 패치/마이너 범위에서 고친 버전이 없으면 그 사실을 노트에 적고 멈출 것.
- `internal/auth/*`·`internal/store/migrations/*` 는 건드릴 이유가 없음(보호 경로).
- 파괴적 확인(`git checkout -- 파일`) 전에 `git add -A && git commit` 으로 WIP 커밋 먼저(교훈 3회).
- PR #8 브랜치 위에서 작업할 것 — main 에 없는 0bcbbb8·693b733 를 잃지 않도록. `git stash` 금지(공유 스택).
- 테스트를 skip/삭제해 통과시키는 것은 "게이트 완화" 와 같다.
- ci.yml 의 `docker build --build-arg VERSION=1.0.146` 은 VERSION 파일(1.0.146)과 같음 — 이번 회차는 버전을 올리지 말 것(릴리즈 단계가 함).

## 차선 후보
- 원인이 현장에서 이미 사라졌거나(예: 재실행으로 green) 이 환경에서 재현 불가로 확정되면: docs/admin-guide.md(대체된 옛 통합 가이드, 옛 `알림` 탭을 아직 인용) 제거 + 참조 정리 + PDF 목록 확인 (가치 2 / 위험 1 / S). 정본을 하나로 유지하라는 운영자 규칙과 맞는다.
