- 과제: 수정 과제(4회째 같은 게이트) — PR #11 security-ci 실패를 오늘의 vulndb 로 재판정하되, 여전히 외부 차단이면 **같은 내용의 PR 을 또 만들지 말고** 원장에 "외부 차단" 으로 기록하고 차선 과제(docs_test 설정 키 대조)만 수행 (가치 4 / 위험 1 / 작업량 S)
- 왜: PR #8·#9·#10·#11 이 모두 ci.yml `test-build-scan` 의 step 7 `Go vulnerability scan`(`govulncheck ./...`, `.github/workflows/ci.yml:49-52`) 에서 GO-2026-6452 로 죽었고, 원인은 vulndb 보고서에 `fixed` 이벤트가 없어 **excelize 어느 버전도 영향 범위** 인 외부 결함이다(09-19 21시·22시 회차가 각각 curl·govulncheck 로 확인; 이 정찰 세션은 네트워크·go 실행이 거부돼 오늘 상태는 **미확인**). PR #11(`auto/2026-09-19-2233`, docs/admin-guide.md 삭제)은 main@eb2a3b0(excelize v2.11.0) 기반이라 코드 잘못이 아니며, main 자체도 지금 돌리면 같은 단계에서 빨강이다. 세 회차 연속으로 "재판정 + 차선 과제 + 새 PR" 을 반복해 verify-failed 가 쌓였으므로 이번에는 반복을 끊는 것이 목표다.
- 수용 기준:
  1) 0단계 판정이 명령 출력으로 기록됨: `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.modified, [.affected[].ranges[].events]'` 와 `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`(main 트리) 의 exit code·`Fixed in:` 줄. `go list -m -versions github.com/xuri/excelize/v2` 로 v2.11.0 뒤 태그 유무.
  2) **경로 A(fixed 가 생겼거나 새 태그가 있음)**: fixed 가 `2.11.0` 이면 go.mod 변경 없이 `govulncheck ./...` exit 0 을 확인하고 끝(PR #10·#11 은 CI 재실행만으로 초록). fixed 가 f98df08 이후 pseudo-version 이거나 새 태그면 `git cherry-pick 6424eb1`(excelize 핀 + `TestAHostileWorkbookIsRejectedRatherThanCrashingTheImport`) 또는 새 태그로 `go get` + `go mod tidy` 후 `govulncheck ./...` exit 0, `go test ./internal/web -run Hostile -count=1` ok.
  3) **경로 B(여전히 fixed 없음)**: go.mod·워크플로·precheck 를 건드리지 않는다(PR #9 의 핀, PR #10 의 precheck 단계·operations.md 소절이 이미 열려 있음 — 중복 금지). 원장·회차 노트에 "수정 과제: 외부 차단(vulndb GO-2026-6452 fixed 누락) 4회째 — 저장소 안 해결 수단 없음, 운영자 결정: (1) golang/vulndb 에 fixed 이벤트 PR 제출 (2) 열린 PR #9/#10/#11 은 vulndb 갱신 시 CI 재실행으로 초록 — #9 는 #10 이 포함하므로 닫아도 됨" 을 적고 차선 과제로 넘어간다.
  4) 차선 과제(경로 B 일 때만; A 면 생략 가능): `internal/web/docs_test.go` 에 테스트 하나 추가 — `docs/ADMIN_GUIDE.md`·`docs/operations.md`·`docs/USER_GUIDE.md` 본문에서 `` `<row>.<key>` `` 꼴(`mail.enabled`, `oidc.auto_login`, `notification.digest_hour` 같이 row ∈ {general, mail, oidc, analytics, mcp, notification, …} 인 백틱 토큰)을 정규식으로 모아, 마이그레이션 시드(`internal/store/migrations/*.sql` 의 `settings` 행 JSON 키 — 기존 docs_test 가 3-2 표 대조에 쓰는 시드 파서를 재사용)에 **없는** row·key 를 실패 메시지에 파일:행 과 함께 나열해야 한다. 테스트는 현재 트리(main + 이 브랜치)에서 통과해야 하고, 가이드에 `notification.digest_hour` 를 한 줄 넣은 트리에서 실패해야 한다(부정 검증, 결과를 노트에 적을 것). 시드에 없는 row 가 의도적으로 문서에 남은 경우(036 이관 서술의 옛 `notification` 행)는 테스트가 허용 목록으로 명시하고 그 이유를 주석에 쓴다.
  5) `bash scripts/precheck.sh`(DSN 있으면 포함) 통과, `git diff origin/main -- .github/workflows/ go.mod` 가 경로 B 에서 0바이트.
- 건드릴 파일:
  - 경로 A: `go.mod`·`go.sum`(cherry-pick 6424eb1 또는 새 태그), `internal/web/import_upload_test.go`(cherry-pick 에 포함) — 손으로 쓰지 않음.
  - 경로 B: `internal/web/docs_test.go` — 기존 시드 파서·가이드 읽기 헬퍼 옆에 `TestEverySettingKeyTheGuidesCiteExistsInTheSeed`(이름은 기존 관례대로 서술문) 추가. 필요하면 `docs/operations.md`·`docs/ADMIN_GUIDE.md` 의 실제 잔재 한두 줄 수정(잔재를 고치면 PDF 신선도 검사 때문에 `ADMIN_GUIDE.pdf` 재생성 필요 — operations.md 는 PDF 목록에 없음(지난 회차 확인)).
  - 원장(`journal.md`) — 수정 과제 결과.
- 검증 명령:
  - `govulncheck ./...` (exit code 와 `Fixed in:` 줄을 그대로 노트에)
  - `go test ./internal/web -run 'Docs|Guide|Setting' -count=1 -v`
  - `bash scripts/precheck.sh` (임시 Postgres 를 띄우면 `TEST_POSTGRES_DSN` 포함; 지난 회차 방식 `docker run -d -p 55432:5432 -e POSTGRES_PASSWORD=… postgres:16`)
  - `git diff origin/main -- .github/workflows/` → 비어 있어야 함
- 위험과 피할 것:
  - **워크플로 완화 금지**(단계 제거·continue-on-error·`|| true`·ID 무시·`replace`·vendor). 지난 세 회차가 이미 확정한 것을 다시 "고치려" 하지 말 것.
  - 이 worktree 의 로컬 `main`(c32eec7)은 stale — 비교 기준은 항상 `origin/main`(eb2a3b0). `git fetch origin` 먼저.
  - 열린 PR #10(precheck.sh·operations.md·go.mod)·#11(docs/admin-guide.md 삭제)과 같은 파일을 건드리면 병합 충돌 — precheck.sh·go.mod(경로 B)·admin-guide.md 는 손대지 말 것.
  - `docs_test.go` 는 가이드↔코드 양방향 대조라 새 테스트가 기존 표 파서를 깨뜨리지 않게 별도 함수로 두고, 정규식은 백틱 안 토큰만 잡을 것(본문의 `a.b` 문장 오탐 주의 — `mail.enabled` 처럼 소문자·밑줄만, 점 하나).
  - 파괴적 확인(`git checkout --`) 전 `git add -A && git commit` 으로 WIP 커밋 먼저 — 이 저장소에서 편집을 날린 사례 3회.
  - 정찰 세션은 curl·gh·go·govulncheck 실행이 모두 거부돼 **오늘의 vulndb 상태·PR #11 실패 step 이름·excelize 태그는 미확인** — 0단계에서 반드시 직접 확인하고 그 출력으로 경로를 고를 것.
- 차선 후보: `docs/user-guide.md`(소문자, "대체되었습니다" 배너의 잔재 — 22:33 회차 비평이 지적) 를 PR #11 과 같은 방식(참조 0건 `grep -rniI user-guide` 확인 → `git rm` → precheck 그림·PDF 검사 무영향 확인)으로 제거. 4) 의 docs_test 가 기존 파서 구조상 45분 안에 안 되면 이것으로.
