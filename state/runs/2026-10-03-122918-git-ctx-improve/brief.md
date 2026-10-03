# 과제서 — 2026-10-03 (run 2026-10-03-122918-git-ctx-improve)

## 먼저: 이번 회차의 '실패' 에 대해 확인한 것과 확인 못 한 것

- 주어진 실패 문자열은 `fix-round: error: agent produced no result (TIMEOUT )` 다. 이것은 **에이전트의 벽시계 타임아웃**이고, 저장소 안의 테스트 실패가 아니다.
- **미확인**: GitHub Actions 의 실제 실패한 런을 보지 못했다. 이 환경에서 `gh run list` 는 권한 승인이 없어 실행되지 않았다(두 번 시도, 둘 다 거부). 따라서 "릴리즈 워크플로가 같은 이유로 두 번 실패했다" 의 구체적 런·단계·로그는 확인하지 못했다. 아래 과제는 **로컬에서 실측한 것만**을 근거로 한다.
- 저장소 상태는 건강하다(실측): `git status` 깨끗, HEAD `ed1376d release: v0.77.22`, `internal/version/version.go:10` 의 `Version = "0.77.22"`, `docs/release-notes-v0.77.22.md` 존재 — 즉 release.yml 의 "Validate tag and source version"(:78-103) 과 "Stage the verified release artifact"(:202-210, `install docs/release-notes-v…md`) 가 걸릴 자리는 아니다.
- 그래서 TIMEOUT 과 인과가 닿는 자리로 남는 것은 **검증의 벽시계 시간**이다. `release.yml:25` 의 `timeout-minutes: 45` 하나 안에서 `build` 잡이 직렬로 전부를 돈다: 전체 suite 평문(`:151`) → 전체 suite `-race`(`:152`) → 통합 `-run Integration`(`:166`) → `go install govulncheck` + 스캔(`:168-173`) → `docker buildx build`(`:178-188`) → 패키징·오프라인 이미지 적재 검증(`:190-200`). 비교로 `ci.yml` 은 같은 일을 `build`·`vulnerabilities`·`integration` 세 잡으로 병렬 분할한다.

## 과제

- 과제: `internal/app` 테스트의 벽시계 시간을 지배하는 "백그라운드 워커가 색인 작업을 집을 때까지의 대기" 를 줄이기 (가치 4 / 위험 2 / 작업량 M)

- 왜: `internal/app` 한 패키지가 전체 테스트 시간을 지배한다(이번 회차 실측 `ok git-ctx/internal/app 100.516s`). 그 100초의 거의 전부가 **단정이 아니라 고정 대기**다 — 공유 픽스처 `indexedAppOn`(`internal/app/driver_agreement_integration_test.go:119`)이 저장소를 등록한 뒤 `waitFor(t, 90*time.Second, …)`(:156-160)로 `index_jobs` 를 200ms 간격(`internal/app/live_integration_test.go:251-259`)으로 폴링하며 워커가 작업을 집어 `status='completed' AND files_processed>0` 이 되기를 기다린다. 이 픽스처를 쓰는 테스트가 2.3~2.5초에 촘촘히 몰려 있고(아래 실측 표), 약 40건 × ~2.3초 ≈ 90초다. 릴리즈의 `build` 잡은 이 suite 를 평문과 `-race` 로 **두 번** 돌고 그 뒤에 Docker 빌드까지 45분 안에 끝내야 하므로, 이 고정 대기를 줄이면 단정을 하나도 지우지 않고 릴리즈 예산이 넓어진다.

- 수용 기준:
  1. `go test -tags sqlite_fts5 -count=1 ./internal/app` 의 보고 시간이 수정 전 실측 100.5s 대비 **30% 이상** 줄어든다(수정 전·후 같은 명령의 출력 두 줄을 나란히 제시할 것).
  2. 테스트 개수와 단정이 그대로다: 같은 명령을 `-v` 로 돌려 `--- PASS` 줄 수가 수정 전과 같고(수정 전 실측은 `app_v.log` 에 보존되어 있다), `git diff` 에 **삭제된 단정·새 `t.Skip`·새 `testing.Short()` 분기·`-short` 사용·워크플로 수정이 하나도 없다**.
  3. 줄어든 이유가 "워커의 작업 집기 지연" 임을 증거로 보인다 — 줄인 값(워커 폴링/깨우기 간격)을 되돌리면 시간이 다시 100초대로 돌아오는 것을 같은 명령으로 보일 것.
  4. `go test -tags sqlite_fts5 -race -count=1 ./internal/app` 가 통과한다(간격을 줄이면 경쟁이 드러날 수 있으므로 반드시 돌릴 것).
  5. `go test -tags sqlite_fts5 -count=1 ./...` 전 패키지 통과, `go vet ./...`, `go build -tags sqlite_fts5 ./...`, `gofmt -l ./cmd ./internal` 빈 출력.

- 건드릴 파일 (프로덕션 2개 이내 목표):
  - `internal/worker/worker.go` — 워커 루프가 다음 작업을 집기까지의 대기 간격. **미확인**: 이 파일에서 확인한 상수는 `pausedRequeueDelay = 45 * time.Second`(`:315`) 뿐이고, 실제 유휴 폴링 간격이 어디에 어떤 이름으로 있는지는 열어 보지 않았다. 구현자의 1단계는 `rg -n 'time\.(Sleep|NewTicker|After)|Duration' internal/worker/worker.go` 로 그 자리를 특정하는 것. 고치는 모양은 **값을 하드코딩으로 줄이는 것이 아니라** 이미 설정으로 흐르는 값(`index.pollingMinutes` 가 설정 카테고리로 존재한다 — `internal/app/app_test.go:75` 가 그 검증을 단정한다)이나 `App`/`Worker` 생성 시 주입 가능한 필드로 만들어 **테스트 픽스처가 짧은 값을 주는** 것이다. 프로덕션 기본값은 바꾸지 말 것.
  - `internal/app/driver_agreement_integration_test.go:119 indexedAppOn` — `New(ctx, config.Config{…})` 에 그 짧은 간격을 넘기도록 한 줄. `waitFor` 의 90초 상한과 200ms 폴링은 그대로 둘 것(상한을 줄이면 느린 CI 에서 새 flake 가 된다).
  - 필요하면 `internal/app/live_integration_test.go:251 waitFor` 는 **건드리지 말 것** — 이미 200ms 로 충분히 촘촘하다. 병목은 폴링 쪽이 아니라 워커 쪽이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test -tags sqlite_fts5 -count=1 ./internal/app` (수정 전 실측 100.516s)
  - `go test -tags sqlite_fts5 -count=1 -v ./internal/app` ( `--- PASS` 줄 수 비교용; 수정 전 로그는 이 디렉터리의 `app_v.log` )
  - `go test -tags sqlite_fts5 -race -count=1 ./internal/app`
  - `go test -tags sqlite_fts5 -count=1 ./...` (수 분)
  - `go vet ./...` / `go build -tags sqlite_fts5 ./...` / `gofmt -l ./cmd ./internal`
  - `bash scripts/verify-version-sync.sh`
  - 단일 테스트로 고정 대기를 바로 보는 법: `go test -tags sqlite_fts5 -count=1 -v -run 'TestReadFileReturnsTheWholeFile$' ./internal/app` — 이번 회차 실측 `--- PASS … (2.29s)` / `ok … 2.294s`, 즉 그 2.3초가 전부 픽스처 대기다.

- 수정 전 실측 (이번 회차, `app_v.log`; 1초 이상인 것 전부. 합계 ≈ 89.6s / 100.5s):
  `TestPlatformDegradationIntegration 10.34s`, `TestPlatformChainIntegration 10.32s`, `TestOneUnindexableRepositoryDoesNotStarveTheOthers 6.46s`, `TestBothPlatformsAnswerTheSameToolsIntegration 5.07s`, `TestDocumentSourceAnswersAreCorrectIntegration 4.30s`, `TestDocumentSourceChainIntegration 4.28s`, `TestIncrementalPushChainIntegration 4.26s`, `TestARunbookInKoreanIsFoundIntegration 4.25s`, `TestOpenSearchProjectionChainIntegration 4.24s`, `TestTheProbesAnswerWhileTheDatabaseIsBusy 4.02s`, `TestAFailedLiveReadSaysWhyItFailed 3.96s`, `TestVectorStatusProbeSeparatesModelFailureFromVectorFailure 3.21s`, `TestTheIndexFallbackSaysItIsNotTheWholeFile 3.19s`, `TestAnAnswerFromAnOldIndexSaysHowOldItIs 2.55s`, `TestAccessControlChainIntegration 2.52s`, `TestReadFileReturnsTheWholeFile 2.47s`, `TestAWordThatOnlyAppearsInAHeadingIsFound 2.46s`, `TestEveryToolThatTakesALibraryRecordsIt 2.46s`, `TestBitbucketChainIntegration 2.45s`, `TestReadingAFileRecordsWhichRepositoryItCameFrom 2.45s`, `TestAWordThatOnlyAppearsInADocCommentIsFound 2.44s`, `TestFindFileNamesTheToolThatCanReadAnUnindexedFile 2.44s`. 2.4~2.5s 에 열 건 넘게 몰린 이 양자화가 "고정 대기" 의 증거다.

- 위험과 피할 것:
  - **`.github/workflows/*` 를 한 글자도 바꾸지 말 것.** `timeout-minutes` 올리기·`-race` 빼기·`-short` 넣기·단계 제외는 전부 "느슨하게 만들어 통과시키기" 이고 이번 회차의 금지 사항이다. 끝에 `git diff --stat .github/` 가 빈 출력인 것을 확인해 보고할 것.
  - 워커의 **프로덕션 기본 간격을 바꾸지 말 것**. 운영 폴링을 빠르게 하면 온프레미스 소스(GitLab/Bitbucket) 호출량이 늘고, 과거 교훈에 "rate-limit 헤더를 지연으로 읽은" 사고가 있다(`ci.yml:1-4` 주석). 테스트가 주입하는 경로만 추가할 것.
  - 간격을 줄이면 **경쟁 조건이 드러날 수 있다.** `-race ./internal/app` 를 반드시 돌리고, 한 번 통과를 믿지 말고 `-count=2` 로도 한 번 볼 것. 새 flake 를 만드는 것이 이 과제의 가장 큰 실패 모드다.
  - `testing.Short()` 가드(`internal/app/read_file_fidelity_test.go:29-31`)는 이미 있다 — 이것을 넓혀 테스트를 건너뛰게 만드는 것은 커버리지 축소이지 최적화가 아니다.
  - 보호 경로를 피할 것: `internal/auth`·`internal/store` 의 migration·백업 복원·`internal/version`.
  - `App.Close()`/`t.Cleanup` 경로를 건드리면 픽스처 누수가 다른 테스트로 번진다 — 생성 시 주입 한 가지만 더하고 생애 주기는 그대로 둘 것.

- 차선 후보: **`fencesContent` 의 도구 이름 허용목록을 `format.go` 의 `contentFence` 사용처와 계약으로 묶기** (가치 3 / 위험 2 / S). PR #44 비평이 "승인해도 남는 우려 (1)" 로 남긴 것이고 PR #44 는 `20b05c9` 로 머지됐으므로 이제 가능하다. `internal/mcp/budget.go` 의 `fencesContent(tool)` 가 `ReadFile`·`GetSymbolContext` 만 참인데, 세 번째 포매터가 `internal/mcp/format.go:207` 의 `contentFence` 를 쓰기 시작하면 아무 테스트도 깨지지 않고 그 도구만 조용히 구동작으로 남는다. "펜스를 쓰는 도구" 를 한 군데서 선언하게 묶고, 그 선언을 벗어난 조합을 깨뜨리는 테스트를 더한다. 검증은 `go test -tags sqlite_fts5 -count=1 ./internal/mcp` (단일 패키지 ~1.2s). 픽스처 주의: `internal/mcp/server_test.go:22` 는 공유 in-memory SQLite 이름을 쓰므로 새 테스트에 `t.Parallel()` 금지.
