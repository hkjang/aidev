# 과제서 — 2026-09-19 (수정 과제)

- 과제: 릴리즈 워크플로가 두 번 연속 실패한 원인 진단·수정 — 워크플로는 그대로 두고 의존성/스크립트 쪽을 고친다 (가치 5 / 위험 2 / 작업량 M)
- 왜: 러너는 v0.7.15 이후 릴리즈 워크플로(`.github/workflows/release.yml`)가 같은 이유로 두 번 실패했다고 적재했지만, 이 정찰 세션은 `gh`·네트워크 승인이 없어 실패 로그를 **직접 보지 못했다(미확인)**. 로컬에서 확인한 것: `go test ./...` 전체 통과(cmd/igame, internal/api, realmguard, config, database, secretbox, tracking, web). 따라서 실패 지점은 Go 테스트가 아니라 **Node 쪽 게이트**일 가능성이 가장 높고, 과거 교훈(2026-09-10, "Audit locked Node dependencies" 2회 실패)과 구조가 같다 — CI(`ci.yml:43-44`)는 `--omit=dev`로만 막고 dev 권고는 요약에 적기만 하므로(`ci.yml:46-59`) PR은 초록이고 릴리즈(`release.yml:84-89`, dev 포함 `--audit-level=low`)에서 처음 빨개진다.
- 수용 기준:
  1) 구현자가 **먼저** `gh run list --repo hkjang/igame --workflow=release.yml --limit 5` 와 `gh run view <id> --log-failed` 로 실패한 step 이름과 오류 메시지를 확인하고 회차 노트에 그대로 적는다(추측 금지).
  2) 실패한 step 을 로컬에서 **워크플로와 같은 명령으로** 재현해 exit≠0 을 보고, 고친 뒤 같은 명령이 exit 0 이 되는 것을 확인한다. Audit 이면 `npm --prefix sdk/gamehub-js audit --audit-level=low` 와 `npm --prefix web audit --audit-level=low` 둘 다 `found 0 vulnerabilities`.
  3) 새 lockfile 로 `make deps`(npm ci) → `make lint` → `make test` 통과. web 테스트 수(직전 246~247)와 SDK 9 가 줄지 않는다. `make web-build`(오프라인 번들 검사 포함), `bash scripts/check-release-contract.sh` 통과.
  4) `release.yml`·`ci.yml` 의 게이트 문구(`--audit-level=low`, `fail-build: true`, `severity-cutoff: high`)는 한 글자도 완화하지 않는다. `npm audit fix --force` 금지 — 어떤 패키지가 왜 올라가는지 하나씩 적는다.
  5) 실패 step 이 audit 이 아니면(예: grype `Scan final runtime image` severity high, `govulncheck`, `Prove clean Docker load … go1.26.6` 고정 문자열, browser-smoke) 그 단계의 스크립트/Dockerfile 을 고치되 같은 원칙 — 워크플로 완화 금지, 로컬 재현 필수(`make docker-build` 뒤 `docker run anchore/grype` 또는 `govulncheck ./...`).
- 건드릴 파일 (실패 원인에 따라 택일):
  - Audit 인 경우: `web/package.json`·`web/package-lock.json`, `sdk/gamehub-js/package.json`·`sdk/gamehub-js/package-lock.json` — 권고가 걸린 패키지만 semver 범위 안에서 올린다(현재 web: vitest ^5.0.0, vite ^7.1.3, eslint ^9.34.0, typescript ~5.8.3; SDK: vitest ^5.0.0, vite ^7.1.3, typescript ^5.9.2). 두 트리가 같은 권고를 들고 있으면 양쪽 모두. `web/vitest` 로 RealmGuard kernel vector 가 바뀌면 `UPDATE_KERNEL_VECTORS=1` 재생성이 바이트 동일한지 확인(지난번 절차).
  - govulncheck 인 경우: `go.mod`·`go.sum` 의 해당 모듈 patch 올림(메이저 금지), `Dockerfile` 의 Go 이미지 태그와 `release.yml:122,133` 의 `1.26.6` 문자열이 일치해야 하므로 Go 툴체인 자체를 올리는 것은 이번 회차에서 피한다.
  - grype 인 경우: `Dockerfile` 런타임 베이스 이미지 digest/태그 갱신(`docs/operations.md` 의 언급도 함께).
  - 어느 경우든 `docs/operations.md` 또는 `README.md` 에 "릴리즈 전에 로컬에서 같은 audit 을 돌린다"는 한 줄이 없으면 추가(있으면 손대지 않음).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `npm --prefix sdk/gamehub-js audit --audit-level=low && npm --prefix web audit --audit-level=low`
  - `make deps && make lint && make test`
  - `go test -race ./cmd/... ./internal/... ./migrations/...`
  - `make web-build && bash scripts/check-release-contract.sh`
  - 가능하면 `make docker-build` (node:22-alpine 에서 두 트리 `npm ci` 가 도는지 — 지난 audit 수정 때 이것으로 vitest 4 의 npm 10.9.8 `edgesOut` 크래시를 걸러냈다)
- 위험과 피할 것:
  - 워크플로 완화 금지(운영자 규칙·이번 과제의 명시 조건). `|| true`, `continue-on-error`, `--omit=dev` 를 release.yml 에 넣지 말 것.
  - vitest 4.x 라인은 npm 10.9.8 에서 설치가 깨진다(2026-09-10 기록) — 4 로 내리지 말 것.
  - SDK 에는 vitest 의 vite peer 를 제약하는 것이 없어 업그레이드가 vite 8/rolldown 을 끌어올 수 있다 — SDK `package.json` 의 `vite ^7.1.3` 명시를 유지.
  - `.github/workflows/*`, `migrations/*`, `internal/api/auth.go` 는 이번 과제 범위 밖. 원인이 정말 워크플로 자체의 결함(예: 액션 SHA 삭제, 러너 이미지 변화)이면 고치지 말고 노트에 적고 차선 후보로 넘어간다.
  - 원인이 재현되지 않거나(로컬에서 이미 0 vulnerabilities) 단순 일시 장애(레지스트리 5xx, 러너 취소)로 확인되면 **코드를 바꾸지 말고** 노트에 근거를 적은 뒤 차선 후보를 한다 — 효과 없는 변경은 반려 사유다.
- 차선 후보: 동점일 때 rank 번호와 행 순서가 어긋날 수 있음 — `internal/api/catalog.go`(2곳)·`defense.go`(2곳)·`realmguard.go`(3곳)의 `row_number() OVER(ORDER BY score DESC)` 와 바깥 `ORDER BY score DESC LIMIT` 양쪽에 `u.id` 같은 결정적 tiebreaker 를 같게 넣고 `make test-db DSN=...` 로 PG 테스트 1개 추가 (가치 3 / 위험 1 / 작업량 S).
