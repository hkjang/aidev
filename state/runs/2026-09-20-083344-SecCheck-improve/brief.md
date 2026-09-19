# 과제서 — 2026-09-20 (SecCheck, 수정 과제 6회째 같은 게이트)

- 과제: 수정 과제(6회째) — PR #13 security-ci 실패를 오늘의 vulndb 로 재판정하고, 경로 A(fixed 생김)면 PR #13 재실행만으로 초록임을 확인, 경로 B(여전히 없음)면 **커밋 0개**로 끝내고 운영자에게 넘길 vulndb 수정 제안문을 회차 디렉터리에만 남긴다 (가치 4 / 위험 1 / 작업량 S)

- 왜: 러너는 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 배정했지만, 이 저장소에서 09-19 이후 PR #8~#12 다섯 번의 판정이 모두 같은 결론이었다 — 실패 지점은 `release.yml` 이 아니라 `.github/workflows/ci.yml:49-52` step 7 `Go vulnerability scan`(`govulncheck ./...`)이고, 원인은 vulndb 보고서 GO-2026-6452(excelize)에 `fixed` 이벤트가 없어 **origin/main(eb2a3b0, excelize v2.11.0)을 포함한 어떤 브랜치·어떤 excelize 버전으로도 초록이 되지 않는 외부 차단**이다. PR #13 은 정찰 환경에서 원격 조회가 거부되어 **미확인**이지만, 브랜치 목록상 #12(`auto/2026-09-19-2333`) 다음의 유일한 새 브랜치 `auto/2026-09-20-0734`(07c19c9 "Check every screen path against the routes the server registers", origin/main 대비 커밋 1개, `web/src/lib/routes.test.ts`·`web/test/sourceScan.ts` 만 추가)로 추정한다 — 이 변경은 Go 코드·go.mod·워크플로를 건드리지 않으므로 step 7 이 코드 잘못으로 죽었을 가능성은 없다. 고칠 코드가 없는데 새 PR 을 열면 여섯 번째 빨강이 하나 더 생길 뿐이다.

- 수용 기준:
  1) **0단계 판정 셋을 실제 명령 출력으로 원장에 남긴다** (정찰 환경은 curl·WebFetch·govulncheck 실행이 전부 승인 거부되어 아래는 모두 미확인 — 구현자가 직접 확인):
     (a) `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.modified, [.affected[].ranges[].events]'` — `fixed` 이벤트 유무. 09-20 07:42 기준 `modified 2026-09-16T18:00:43Z`, `[[{"introduced":"0"}]]`.
     (b) `curl -s "https://api.github.com/repos/hkjang/SecCheck/pulls/13" | jq '.head.ref,.head.sha,.state'` 로 PR #13 브랜치를 확정하고, `curl -s "https://api.github.com/repos/hkjang/SecCheck/actions/runs?branch=<head.ref>&per_page=3" | jq '.workflow_runs[]|[.id,.name,.head_sha,.conclusion]'` → 실패 run 의 `/runs/<id>/jobs` 에서 실패 step 이름. 예상: job `test-build-scan`, step 7 `Go vulnerability scan` 하나뿐. `release.yml` 의 실패 run 은 없을 것(`/actions/workflows/release.yml/runs?per_page=3` 로 함께 확인).
     (c) `git fetch origin && git diff --stat origin/main..origin/<head.ref>` 가 `web/` 아래 파일만 보이는지, 그리고 origin/main 트리에서 `govulncheck ./...` 가 exit 3 / `Fixed in: N/A` 인지(go1.26.x + `go install golang.org/x/vuln/cmd/govulncheck@latest`, CI 와 같은 명령).
  2) **경로 B(fixed 없음 — 예상)**: `git status --porcelain` 빈 출력, `git log origin/main..HEAD --oneline` 빈 출력(커밋 0개), `git diff origin/main -- .github/workflows/ go.mod go.sum` 0바이트. 원장에는 '수정 과제 — 외부 차단(vulndb GO-2026-6452 fixed 누락) 6회째, 변경없음' 으로 기록. 추가로 회차 디렉터리에 `vulndb-fix-proposal.md` 를 써서 운영자가 그대로 쓸 수 있는 golang/vulndb PR 본문(대상 파일 `data/reports/GO-2026-6452.yaml`, 더할 이벤트 `fixed: 2.11.0` 의 근거 = upstream 커밋 93f0b3c 가 v2.11.0 에 포함, temp-file 경로는 f98df08 → `2.11.1-0.20260918021423-0434413565bf`, 관련 열린 이슈 #6510·#6501·#6513·#6532)을 남긴다 — 저장소 밖 문서이고 제출은 운영자 결정.
  3) **경로 A(fixed 가 생겼을 때만)**: origin/main 과 PR #13 브랜치 둘 다에서 `govulncheck ./...` exit 0 을 확인하고, PR #13 의 실패 run 을 재실행하면 초록이 될 것임을 원장에 적는다(gh 인증이 있으면 `gh run rerun <id> --failed`, 없으면 재실행은 운영자 몫). fixed 버전이 v2.11.0 보다 높으면(예: pseudo-version) 그때만 `go.mod` 를 그 버전으로 올리고 `go mod tidy` 후 `go test ./internal/web -run TestAHostileWorkbook -count=1`(이 테스트는 PR #9/#10 브랜치에만 있음 — main 에 없으면 `go build ./... && go vet ./...` 로 대체)과 `govulncheck ./...` exit 0 을 증거로 남긴다. 이 경우에도 `.github/workflows/` 는 불변.
  4) 어느 경로든 워크플로 완화(단계 제거·`continue-on-error`·`|| true`·ID 무시·`replace`·vendor·excelize 호출 분리)는 하지 않았음을 `git diff origin/main -- .github/workflows/` 0바이트로 증명.

- 건드릴 파일: 경로 B 에서는 **저장소 파일 없음**. 읽기만: `.github/workflows/ci.yml:49-52`(`Go vulnerability scan` 단계), `go.mod:9`(`github.com/xuri/excelize/v2 v2.11.0`), `internal/store/seed.go:66`(`ExtractWorkbookDefaults` → `excelize.File.GetRows`, govulncheck 가 보고하는 호출 지점), `origin/auto/2026-09-20-0734:web/src/lib/routes.test.ts`(PR #13 내용 — `readFileSync(join(repoRoot,'internal/web/server.go'))`, Go 와 무관). 경로 A 에서만 `go.mod`·`go.sum`. 회차 디렉터리에 `vulndb-fix-proposal.md`(경로 B).

- 검증 명령:
  - `govulncheck ./...` (CI step 7 과 동일; 경로 B 면 exit 3 이 "정답" — 그것을 바꾸려 하지 말 것)
  - `git status --porcelain && git log origin/main..HEAD --oneline && git diff origin/main -- .github/workflows/ go.mod go.sum` (경로 B: 셋 다 빈 출력)
  - 경로 A 에서 go.mod 를 올렸을 때만: `go build ./... && go vet ./... && govulncheck ./...` exit 0, `TEST_POSTGRES_DSN` 이 있으면 `go test ./internal/store ./internal/web -count=1`

- 위험과 피할 것:
  - PR #8~#12 와 같은 판정을 여섯 번째 되풀이하는 회차다. **새 커밋·새 PR 을 만들지 말 것** — 09-20 첫 회차(PR #12 판정)가 커밋 0개로 끝났는데 러너가 다음 회차에 일반 과제(#13)를 배정해 같은 step 에서 또 빨강이 났다. 차선 과제로 코드 변경을 끼워 넣으면 그 PR 도 같은 자리에서 죽는다.
  - go.mod 는 열린 PR #9/#10 과 충돌(excelize pseudo-version 핀). 경로 A 에서만, 그리고 vulndb 의 fixed 버전이 v2.11.0 보다 높을 때만 손댄다.
  - `.github/workflows/*` 완화 금지(러너 guard·운영자 지시). `internal/auth/*`·`migrations/*` 는 이 과제와 무관 — 열지 말 것.
  - 로그 본문은 403 이므로 실패 step 은 `/runs/<id>/jobs` 의 step conclusion 으로 판정하고 `govulncheck ./...` 로컬 재현으로 확정한다.
  - vulndb 제안문은 회차 디렉터리에만 쓰고 실제 제출(outward-facing)은 하지 않는다.
  - 파괴적 확인 전 `git add -A && git commit`(WIP) 규칙은 이번엔 바꿀 파일이 없어 해당 없음 — 경로 A 에서 go.mod 를 만지면 적용.

- 차선 후보: (게이트가 풀린 뒤에만) `api.ts` 의 `download(path)` 호출을 PR #13 의 라우트 스캐너(`web/test/sourceScan.ts`)에 넣어 `ui.tsx:98` 로 변수 경로가 들어오는 호출부의 리터럴을 모으기 — 가치 2 / 위험 1 / S, 순수 프런트 vitest 라 Go 게이트와 무관하지만 PR 은 여전히 step 7 에서 빨강이므로 vulndb fixed 가 생기기 전에는 열지 말 것.
