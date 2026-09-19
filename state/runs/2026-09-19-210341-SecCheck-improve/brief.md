# 과제서 — 2026-09-19-210341 SecCheck (수정 과제)

- 과제: PR #9(`auto/2026-09-19-1843`, 6424eb1 "Pin excelize past the negative shared-string index panic")의 security-ci 실패를 **오늘의 vulndb 상태로 다시 판정**하고, 그 판정에 따라 (a) 초록이면 CI 재실행으로 통과를 증명하거나 (b) 아직 막혀 있으면 "코드로는 풀 수 없는 외부 DB 결함" 을 원장에 확정한 뒤, 같은 게이트를 푸시 전에 로컬에서 보게 하는 `precheck.sh` 단계와 운영 문서 절을 더한다 (가치 4 / 위험 1 / 작업량 S)
- 왜: 지난 회차가 확정한 대로 실패 지점은 release.yml 이 아니라 ci.yml `Go vulnerability scan`(`govulncheck ./...`)이고, 원인은 Go vulndb 의 GO-2026-6452 보고서에 `fixed:` 가 없어 **어떤 excelize 버전으로도 통과하지 않는 것**이다(main@eb2a3b0 도 지금 돌리면 같은 단계에서 죽는다; PR #9 코드의 잘못이 아님). 이 상태에서 저장소 안에서 게이트를 초록으로 만드는 유일한 수단은 워크플로 완화(금지)이므로, 이번 회차는 "고칠 수 있는 것" 과 "기다려야 하는 것" 을 명확히 가르고 다음 회차가 같은 자리에서 또 헤매지 않게 하는 것이 목표다. vulndb 이슈(golang/vulndb#6510·#6501·#6513·#6532)가 그 사이에 닫혔을 수 있으므로 0단계 확인이 먼저다.

## 수용 기준
1) **0단계 (판정, 반드시 기록)**: 아래 두 명령의 출력을 journal.md 에 그대로 붙인다.
   - `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.modified, [.affected[] | {module: .package.name, ranges: .ranges}]'` → `ranges` 에 `fixed` 가 있는지.
   - `gh run list --repo hkjang/SecCheck --branch auto/2026-09-19-1843 --workflow=ci.yml --limit 3` → 실패 run 번호와, `gh run view <id> --json jobs --jq '.jobs[].steps[] | select(.conclusion=="failure") | .name'` 이 여전히 `Go vulnerability scan` 인지.
2) **(a) `fixed` 가 생겼으면**: 이 회차 브랜치에 `git cherry-pick 6424eb1`(go.mod·go.sum·`internal/web/import_upload_test.go` — 지난 회차의 실제 panic 수정, 잃으면 안 됨)을 올리고, 로컬에서 `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...` 가 exit 0 임을 보인다. PR #9 는 `gh run rerun <id>` 또는 `gh workflow run ci.yml --ref auto/2026-09-19-1843` 으로 재실행해 `gh run watch` 로 success 를 확인하고 번호를 원장에 적는다. 이 경우 3)·4) 는 그대로 수행하되 문서 절은 "이런 일이 있었다" 시제로 쓴다.
3) **(b) `fixed` 가 여전히 없으면**: 원장에 "코드 변경으로 해결 불가 — vulndb 결함, 워크플로 완화 금지, 운영자 결정 항목: golang/vulndb 에 `fixed: 2.11.0`(in-memory 경로) 또는 `v2.11.1-0.20260918021423-0434413565bf`(temp-file 경로) 를 더하는 PR 을 낼지" 를 **수정 과제** 로 확정하고, 그래도 `git cherry-pick 6424eb1` 은 올린다(게이트와 무관하게 실제 panic 을 막는 효과 있는 변경).
4) **`scripts/precheck.sh`**: `step "비밀정보 스캔"`(105행) 바로 앞에 `step "Go 취약점 스캔"` 을 둔다 — `command -v govulncheck` 가 있으면 `govulncheck ./... || fail "govulncheck 가 걸린 항목이 있습니다. CI 의 'Go vulnerability scan' 도 같은 이유로 멈춥니다."`, 없으면 비밀정보 스캔 단계와 같은 관례로 `printf '건너뜀: govulncheck 가 없어 … CI에서는 반드시 돌아갑니다.\n'`. 증명: govulncheck 가 설치된 셸에서 `bash scripts/precheck.sh` 출력에 단계가 보이고, (b) 상태라면 이 단계에서 CI 와 같은 GO-2026-6452 로 `fail` 하는 것을 보인다(그것이 정답 — 통과시키려고 단계를 빼지 말 것). `PATH` 에서 govulncheck 를 지운 셸에서는 건너뜀 문구가 나온다.
5) **`docs/operations.md` `## 릴리즈 게이트`(263행) 절 끝**에 소절 "govulncheck 가 `Fixed in: N/A` 로 막힐 때" 를 더한다: 위 curl 확인 명령, 무슨 뜻인지(보고서에 fixed 가 없으면 모든 버전이 영향 범위), 하지 말 것(워크플로에서 단계 제거·`continue-on-error`·`replace` 로 모듈 경로 바꾸기 — 게이트를 속이는 것), 할 것(실제 취약 경로가 있으면 upstream 수정 커밋으로 핀하고 회귀 테스트, vulndb 이슈에 fixed 버전 근거를 달거나 PR, 그때까지 릴리즈 보류). GO-2026-6452 사례를 한 줄로 든다. `go test ./internal/web -run 'Docs|Guide'` 가 operations.md 를 검사하는지 미확인 — 돌려서 초록인지 본다.
6) **워크플로 불변**: `git diff main -- .github/workflows/` 가 비어 있다.

## 건드릴 파일
- `go.mod`·`go.sum`·`internal/web/import_upload_test.go` — `git cherry-pick 6424eb1` 로만 (손으로 다시 쓰지 말 것)
- `scripts/precheck.sh` — `step "비밀정보 스캔"` 앞에 `step "Go 취약점 스캔"` 추가(4번)
- `docs/operations.md` — `## 릴리즈 게이트` 절에 소절 추가(5번). PDF: `bash scripts/build_docs_pdf.sh --list` 에 operations.md 가 있으면 precheck 의 "가이드 PDF" 단계가 신선도를 요구하므로 공용 md2pdf 로 다시 굽는다(있는지 미확인 — 목록으로 확인).
- journal.md — 0단계 출력과 판정

## 검증 명령
- `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '[.affected[].ranges]'`
- `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...` (exit 0 이면 (a), exit 3 이면 (b))
- `bash scripts/precheck.sh` (TEST_POSTGRES_DSN 은 임시 Postgres 16 컨테이너 — 지난 회차 관례 `:55432`; 없으면 go test 통합은 건너뜀)
- `go test ./internal/web -run 'Docs|Guide' -count=1`
- `git diff main -- .github/workflows/` → 출력 없음
- (a) 일 때만: `gh run rerun <id>` → `gh run watch <id> --exit-status`

## 위험과 피할 것
- **워크플로 완화 금지** — 단계 제거·`continue-on-error`·`|| true`·특정 ID 무시 스크립트 모두 반려 사유. `replace github.com/xuri/excelize/v2 => <포크 경로>` 로 모듈 경로를 바꿔 스캐너 눈을 피하는 것도 같은 부류이니 하지 말 것.
- precheck 의 새 단계는 (b) 상태에서 **정당하게 실패** 한다. 그 실패를 없애려고 단계를 조건부로 만들지 말 것 — 스크립트의 목적이 CI 와 같은 판정을 미리 보는 것.
- `internal/auth/*`·`internal/store/migrations/*` 는 건드리지 않는다(이번 과제와 무관).
- 파괴적 확인(`git checkout -- …`) 전에 `git add -A && git commit` 으로 WIP 커밋 — 이 저장소에서 편집을 날린 사례 3회.
- `pkill -f <경로>` 는 자기 셸을 죽인다(코드 144). 임시 Postgres 컨테이너는 끝나면 지운다.
- excelize 가 v2.11.0 뒤 태그를 냈는지 미확인 — `go list -m -versions github.com/xuri/excelize/v2` 로 보고, 태그가 있고 f98df08 을 포함하면 pseudo-version 대신 태그로 핀하되 같은 회귀 테스트로 확인.

## 차선 후보
- docs/admin-guide.md(대체된 옛 통합 가이드, 옛 `알림` 탭 인용) 제거해 ADMIN_GUIDE.md 하나만 정본으로 — 다른 문서·스크립트·docs_test 가 참조하지 않는지 `grep -rn admin-guide.md` 로 확인 뒤 삭제, manual/features PDF 목록에 들어 있으면 재생성 (가치 2 / 위험 1 / S). 1순위의 4)·5) 가 성립하지 않을 때(예: precheck 에 이미 같은 단계가 있는 것으로 드러남) 고른다.
