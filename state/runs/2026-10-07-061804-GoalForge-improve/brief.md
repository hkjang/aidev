# 과제서 (2026-10-07-061804-GoalForge-improve)

## 먼저: 배정된 "우선 과제" 에 대한 판정 — 이 저장소에서 고칠 것이 아니다

- 러너가 적은 실패는 `2026-10-07-042740-GoalForge-improve/stages.json` 의
  `"pr": {"state":"create-failed","reason":"gh pr create 실패"}` 이고, 같은 run 의
  `journal.md:30` 도 `[러너 04:47] pr create-failed — gh pr create 실패` 한 줄뿐이다.
  **gh 의 stderr 원문은 두 파일 어디에도 남아 있지 않다 — 원인 문자열 미확인.**
- 이 `pr create` 는 GoalForge 의 GitHub Actions 단계가 **아니다**. 이 저장소의 워크플로는
  `.github/workflows/ci.yml` 과 `release.yml` 둘뿐이고, 둘 다 `gh pr create` 를 호출하지 않는다
  (`release.yml` 은 v* 태그에서 빌드·릴리즈 게시). 실패한 `pr create` 는 aidev 러너(하네스)의
  단계이며 그 스크립트는 이 저장소 밖(`/mnt/c/Users/USER/projects/aidev`)에 있다.
- 유력한 기전(확인한 사실): 이 작업 트리의 `remote.origin.pushurl` 이 `DISABLED` 다
  (`git remote -v` → `origin DISABLED (push)`). 브랜치가 origin 에 올라가지 못하면
  `gh pr create` 는 성립할 수 없다. **다만 러너가 PR 을 만드는 체크아웃이 이 작업 트리와
  같은 설정인지는 확인하지 못했다 — 미확인.**
- 따라서 구현자가 이 저장소에서 할 수 있는 수정은 없다. 하네스 설정/자격증명 문제이므로
  **사람(hkjang)에게 올릴 일**이고, 이 회차 과제서는 그 실패가 실제로 삼켜 버린 작업을
  되살리는 것으로 쓴다(아래). 워크플로 파일을 느슨하게 만드는 변경은 하지 말 것 —
  애초에 느슨하게 할 워크플로가 이 저장소에 없다.

## 이번 회차 과제

- 과제: `storage prune --older-than` 이 비유한 수·오버플로 기간을 조용히 삼키는 것을 고친다 (가치 3 / 위험 1 / 작업량 S)
- 왜: 직전 회차(2026-10-07-042740)가 이 결함을 고쳐 전 패키지 통과까지 확인했는데 `gh pr create`
  실패로 그 커밋이 main 에 들어가지 못했다 — 방금 `grep -n "IsInf\|IsNaN\|MaxInt64" cmd/goalforge/*.go`
  가 **무출력**이고 `cmd/goalforge/main.go:4384` 가 여전히 `return time.Duration(count * float64(unit)), nil`
  이라, 수정은 실제로 유실됐다. 결함 자체는 아키텍처 의존이라 위험하다: 같은 `"1e300w"` 가 amd64 에서는
  음수(`MinInt64`)가 되어 `window <= 0` 에 걸리지만 arm64 에서는 양수(`MaxInt64`)가 되어 통과하고,
  경계 시각이 1734 년이 되어 **삭제가 실제로 실행**된다(직전 회차가 binfmt aarch64 로 실측).
  CI 매트릭스에 macos 러너(arm64)가 있으므로 가설이 아니다.
- 수용 기준:
  1) `dayDuration("Infd")`, `dayDuration("NaNd")`, `dayDuration("1e300w")`, `dayDuration("-Infd")` 가
     **에러를 반환**한다(값이 아니라 거절). 새 어휘를 만들지 말고 그 분기에 이미 있는
     `fmt.Errorf("기간 %q 를 읽을 수 없습니다", value)` 를 그대로 재사용한다.
  2) 기존 `TestDurationAcceptsDaysAndWeeks` 의 6개 입력(`30d/1d/12w/720h/90m/1.5d`)과
     `TestUnreadableDurationSaysWhatItAccepts` 의 4개 입력이 **기대값 변경 없이** 그대로 통과한다.
  3) CLI end-to-end: `storage prune --older-than 1e300w` 가 프루닝 보고를 출력하지 않고
     "읽을 수 없습니다" 로 끝난다. `--older-than 0d` / 음수는 지금처럼 `storagePrune` 의
     `window <= 0` 이 `--older-than must be positive` 로 거절한다(이 분기는 건드리지 말 것).
  4) 테스트는 **아키텍처에 상관없이** 같은 결론이어야 한다. `time.Duration` 변환 결과의 부호나
     구체적 값을 기대값으로 쓰지 말 것 — amd64/arm64 에서 서로 다르다. "err != nil" 만 단언할 것.
- 건드릴 파일 (프로덕션 1 + 테스트 2):
  - `cmd/goalforge/main.go:dayDuration` (4373–4392) — `strconv.ParseFloat` 성공 후,
    `time.Duration(count * float64(unit))` **앞에** 거절을 넣는다:
    `math.IsInf(count, 0) || math.IsNaN(count) || math.Abs(count)*float64(unit) > math.MaxInt64`
    → 같은 `기간 %q 를 읽을 수 없습니다` 에러. `import "math"` 추가 필요.
    `d`/`w` 접미사 분기 안에서만. 뒤의 `time.ParseDuration` 경로는 그 자체로 범위를 검사하므로 건드리지 말 것.
  - `cmd/goalforge/duration_test.go` — 비유한 수·오버플로 입력을 거절하는 테스트 추가
    (기존 두 테스트의 기대값은 그대로 둘 것).
  - `cmd/goalforge/main_e2e_test.go` — `storage prune --older-than 1e300w` 가 프루닝 보고 대신
    거절로 끝나는 것을 실제 CLI dispatch 로 확인. **주의**: `runCLIWithError` 는 전역 stdout 을
    바꾸므로 `t.Parallel()` 금지(프로필의 관례).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./cmd/goalforge -count=1` (직전 측정 2~7초)
  - `go test ./... -count=1` (기준선 exit 0, 32 ok. 알려진 skip 4건:
    TestCLIFullLifecycle, TestRestoreVerifiesRecordsAndSettlesOutsideWork,
    TestPublishReconcilesInsteadOfRepeating, TestPushBranchPublishesToLocalRemote — push 제한 때문이며 회귀 아님)
  - `go build ./...`, `go vet ./...`, `gofmt -l ./cmd ./internal`(무출력), `go mod tidy` 후 드리프트 없음
  - RED 먼저: 수정 전에 새 테스트를 돌려 빨강을 눈으로 확인하고, 고친 뒤 `math` 가드만 지워
    다시 빨강이 되는지로 인과를 고정할 것.
- 위험과 피할 것:
  - 다른 기간 파서(`internal/notify/suppress.go`, `internal/api/setup.go`)를 **통합하지 말 것**.
    운영자 지시: 같은 종류의 파서는 좁히는 방향으로만 고친다. 이번 회차는 `dayDuration` 한 곳만.
  - `s.Prune` SQL, `--apply`/`--vacuum` 의미, 기본값 `"30d"`, 플래그 설명 문구는 건드리지 말 것.
  - `window <= 0` 거절을 `dayDuration` 안으로 옮기지 말 것(음수·0 의 책임은 호출부에 그대로 둔다).
  - 보호 경로(.github/workflows, auth, migrations)는 이번 과제에서 전혀 필요 없다. 열지 말 것.
  - 유실된 직전 커밋을 git 에서 되살리려 하지 말 것 — 다른 작업 트리/브랜치에 있을 수 있으나
    확인하지 못했다. 위 지시대로 새로 쓰는 편이 빠르고 검증 가능하다.
- 차선 후보: 같은 `CheckReadiness` 결과를 네 소비자(doctor/plan/mcp/api)가 같은 ready 결론으로
  번역하는지 end-to-end 로 고정 (2/1/S). `internal/diagnostics/readiness.go` 의 결과를
  `cmd/goalforge/main.go` doctor·plan 과 `internal/mcp/tools.go`·`internal/api` 가 각각 번역하며,
  plan 은 FAIL·WARN 을 모두 WARN 으로 올리고 mcp 는 FAIL 개수로만 ready 를 정한다고 이전 회차가
  기록했다 — **네 소비자 코드는 이번에도 열지 못했다(미확인). 행 번호부터 확인할 것.**
  통합하지 말고 "같은 설정이 두 표면에서 같은 ready 결론" 만 고정하는 범위로.
