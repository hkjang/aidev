# 과제서 — 2026-09-19-223341 (수정 과제, 3회째 같은 게이트)

- 과제: PR #10(`auto/2026-09-19-2103`, 9b4b7aa) 의 security-ci 실패를 "오늘의 vulndb" 로 재판정하고, 코드로 풀 수 없음이 재확인되면 **같은 내용의 PR 을 또 만들지 말고** 원장에 '수정 과제: 외부 차단(vulndb GO-2026-6452 fixed 누락)' 으로 기록 + 차선 과제 수행 (가치 4 / 위험 1 / 작업량 S)
- 왜: 09-19 두 회차(PR #8 693b733, PR #9 6424eb1)가 모두 ci.yml `Go vulnerability scan`(`govulncheck ./...`) 한 단계에서 exit 3 으로 죽었고, 원인은 vulndb 보고서 GO-2026-6452 에 `fixed` 이벤트가 없어 excelize **모든 버전**이 영향 범위인 것(코드 잘못 아님)으로 두 번 확정됐다. PR #10 은 그 두 커밋(a08e347 excelize 핀 + 회귀 테스트, 9b4b7aa precheck govulncheck 단계 + operations.md 소절)을 담고 있어 같은 자리에서 같은 이유로 실패했을 가능성이 압도적이다(이번 정찰은 네트워크·govulncheck 실행이 승인 거부되어 **미확인** — 구현자가 0단계에서 확정할 것). 세 번째로 같은 PR 을 여는 것은 운영자 규칙("반려·실패한 접근 반복 금지") 위반이므로, 이번 회차는 판정을 고정하고 루프를 끊는 것이 목표다.
- 수용 기준:
  1) 0단계 판정이 원장에 명령 출력과 함께 기록됨: (a) `govulncheck ./...` 를 PR #10 트리에서 돌린 결과(exit code, `Fixed in:` 줄), (b) `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.modified, .affected[].ranges'` 의 출력, (c) PR #10 의 실패 step 이름(`gh run list --branch auto/2026-09-19-2103 --workflow=ci.yml` → `gh run view <id> --json jobs`; gh 미인증이면 공개 API `https://api.github.com/repos/hkjang/SecCheck/actions/runs?branch=auto/2026-09-19-2103` 로).
  2) **fixed 가 생겼으면**(경로 A): 현재 핀 `v2.11.1-0.20260918021423-0434413565bf` 가 fixed 이상인지 확인 → `govulncheck ./...` exit 0 → PR #10 을 rebase 없이 `gh run rerun <id>` 또는 빈 커밋 없이 그대로 두고 러너가 재검증하게 함. 새 코드 변경은 필요 없음. 원장에 '수정 과제 — vulndb 갱신으로 해소' 기록.
  3) **fixed 가 여전히 없으면**(경로 B, 예상): PR #10 과 같은 내용의 새 PR 을 만들지 않는다. `.github/workflows/*` 는 1바이트도 바꾸지 않는다(`git diff origin/main -- .github/workflows/` 가 비어 있어야 함). `go.mod` 에 `replace` 로 모듈 경로를 바꾸거나 excelize 를 vendor 하는 우회도 금지(operations.md 릴리즈 게이트 절이 이미 금지 목록에 적음). 원장에 '수정 과제: 외부 차단 — 저장소 안에서 초록으로 만들 수단 없음, 운영자 결정 필요(golang/vulndb 에 `fixed: 2.11.0` / pseudo-version 을 더하는 PR 제출 또는 vulndb 갱신 대기)' 로 기록하고, 아래 차선 과제를 수행한다.
  4) 차선 과제(경로 B 에서만): `docs/admin-guide.md`(소문자, 머리말에 "대체되었습니다", 옛 `알림` 탭 인용) 삭제 → 정본은 `docs/ADMIN_GUIDE.md` 하나. `grep -rn "admin-guide.md" docs scripts README.md web/src internal` 로 참조가 0건임을 먼저 확인(PDF 목록·docs_test·capture 스크립트가 이 파일을 읽지 않는지). `go test ./internal/web -run 'Docs|Guide' -count=1` 통과. 테스트가 이 파일의 존재를 요구하면 삭제하지 말고 원장에 사유를 적는다.
- 건드릴 파일:
  - (경로 A) 없음.
  - (경로 B) `docs/admin-guide.md` — 삭제. 참조가 있으면(README·features.md 링크 등) 그 링크를 `docs/ADMIN_GUIDE.md` 로 바꿈. 그 외 파일은 건드리지 않는다.
- 검증 명령(이 저장소에서 실제로 도는 것):
  - `govulncheck ./...` (`go install golang.org/x/vuln/cmd/govulncheck@latest`; ci.yml 49-52행과 동일)
  - `go test ./internal/web -run 'Docs|Guide' -count=1`
  - `git diff origin/main -- .github/workflows/` → 출력 없음
  - `bash scripts/precheck.sh` (TEST_POSTGRES_DSN 없으면 통합 테스트는 조용히 건너뜀; PR #10 브랜치의 precheck 는 govulncheck 단계가 있어 경로 B 에서는 그 단계에서 exit 1 이 **정답**임 — 단계를 빼지 말 것. main 트리의 precheck 에는 그 단계가 없음)
- 위험과 피할 것:
  - 워크플로 완화(단계 제거·`continue-on-error`·`|| true`·ID 무시·DB 스냅샷 고정) 절대 금지 — 러너 guard 가 `.github/workflows/*` 를 홀드한다.
  - 같은 접근(excelize 핀 cherry-pick 을 새 브랜치로 다시 올리기)을 세 번째로 반복하지 말 것 — 운영자 규칙 1항. PR #10 은 이미 그 변경을 담고 있고 vulndb 가 고쳐지는 순간 그대로 초록이 된다.
  - 파괴적 확인(`git checkout -- …`) 전 `git add -A && git commit` WIP 커밋 먼저(교훈 3회).
  - `pkill -f <경로>` 금지(자기 셸을 죽임).
  - 외부 저장소(golang/vulndb)에 이슈·PR 을 내는 것은 outward-facing 이라 이 회차에서 하지 않는다 — 운영자 결정 항목으로만 적는다.
- 차선 후보: 위 4) 가 성립하지 않을 때 — `docs_test` 에 가이드가 인용하는 `<row>.<key>` 설정 키가 마이그레이션 시드 행에 실제로 있는지 대조하는 테스트 추가(`internal/web/docs_test.go`, operations.md 의 이관 잔재를 자동으로 잡음; 가치 2 / 위험 1 / S). 이것도 CI 는 같은 게이트에서 빨강이므로 원장에 "외부 차단 중 선행 작업" 으로 적는다.
