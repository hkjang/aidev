# 회차 노트 2026-10-03-122918-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:29] base pinned — main@ed1376d
- [러너 12:29] autonomy release — 

## 정찰 노트
- GitHub 런을 못 봤다 — `gh run list` 가 두 번 거부됐다. 그래서 "릴리즈 워크플로 실패" 를 추측으로 메우지 않고, TIMEOUT 과 인과가 닿고 **로컬에서 실측되는** 것 하나만 골랐다: `internal/app` 100.5s 중 ~90s 가 단정이 아니라 `indexedAppOn`→`waitFor` 가 워커의 작업 집기를 기다리는 고정 대기다(단일 테스트 2.29s = 사실상 전부 픽스처). 릴리즈 build 잡은 이 suite 를 평문·`-race` 두 번 돌고 45분 안에 Docker 빌드까지 끝낸다.
- 다른 후보를 제친 이유: 저장소는 깨끗하고 v0.77.22 가 버전·릴리즈 노트까지 정합해 워크플로의 검증 단계가 걸릴 자리가 없었다. 보류 목록의 mcp 계열은 전부 가치 2~3 / S 로 "실패 대응" 과 무관하다.
- 확신 없는 곳(과제서에 미확인으로 표시): 워커의 유휴 폴링 간격이 `internal/worker/worker.go` 어디에 어떤 이름으로 있는지 — 확인한 상수는 `pausedRequeueDelay = 45s`(:315) 뿐이다. 구현자의 1단계가 그 특정이다. `index.pollingMinutes` 설정 카테고리가 이미 있다는 것만 `app_test.go:75` 로 확인했다.
- 구현자가 조심할 것: 워크플로 무수정(`git diff --stat .github/` 빈 출력으로 보일 것), 프로덕션 기본 간격 불변, 간격을 줄여 생기는 새 flake — `-race` 와 `-count=2` 를 반드시 돌릴 것. `t.Skip`·`-short`·단정 삭제로 시간을 줄이면 반려다.
- [러너 12:35] scout done — `internal/app` 테스트의 벽시계 시간을 지배하는 "백그라운드 워커가 색인 작업을 집을 때까지의 대기" 를 줄

## 구현 노트
- 무엇/왜: `internal/app` 테스트 100초의 대부분은 단정이 아니라 "워커가 큐를 들여다보기까지의 2초" 였다(`worker.New` 의 `poll: 2*time.Second`, `RunOnce` 는 틱당 1건). `config.Config.WorkerPollInterval`(테스트 전용, `WorkerIdentity` 선례) → `App.startBackground` → `Worker.SetPollInterval` 주입 경로만 더했다. 프로덕션 기본값 2초는 `DefaultPollInterval` 이라는 이름만 얻고 값 불변. 평문 116.894s→73.102s, `-v` 100.516s→51.455s.
- 확신 없는 곳 / 검증 못 한 것: (1) **postgres 경로 미검증** — `indexedAppOn(driver="postgres")` 와 `replica_test.go` 는 이 환경에 DSN 이 없어 돌지 않았다. replica 는 claim 경쟁 타이밍이 바뀌는 것이 싫어 **일부러 간격을 주지 않았다**; driver_agreement 의 postgres 변주는 같은 fixture라 간격을 받는다 — CI 의 `-run Integration` 잡에서 처음 돈다. (2) 25ms 가 느린 CI 러너에서 큐 폴링 부하(틱마다 `RecoverStaleJobs` UPDATE + `claim`)로 SQLite 락 경쟁을 키우지 않는다는 보장은 로컬 `-race -count=2` 통과(app 114.969s, 경쟁 0)까지만이다.
- 일부러 하지 않은 것: `.github/workflows/*` 무수정(`git diff --stat .github/` 빈 출력) — `timeout-minutes`·`-race`·`-short`·단계 제외 전부 금지 사항. `waitFor` 의 90초 상한·200ms 폴링도 그대로. 남은 20.6초를 지배하는 `internal/outboundnotification` 백오프는 "몇 번 만에 포기" 단정과 얽혀 다음 회차로 넘겼다(ideas.json).
- 다음 역할이 조심할 것: 되돌림 확인은 `internal/app/live_integration_test.go:255` 의 `testWorkerPoll` 한 줄만 `2*time.Second` 로 바꿔 보면 된다 — 그러면 전체가 104.478s 로 돌아오고 `TestTheConfiguredWorkerPollIntervalReachesTheWorker` 가 `worker_poll_test.go:70` 에서 실패한다. 이 테스트의 1초 마감은 "설정값이 워커까지 닿는가" 만 보도록 **집기 지연**(`index_jobs.started_at`)에 걸려 있고 색인 완료 시간에 걸려 있지 않다 — 색인이 느려져도 흔들리지 않게 한 의도다.
- [러너 12:51] brief accepted — 채택 — 과제·수용 기준·금지 사항이 현재 코드와 정확히 맞았다. 과제서가 "미확인" 으로 남긴 "실제 유휴 폴링 간격이 
- [러너 12:52] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 두 신규 테스트가 실제 경로를 지남을 되돌림 프로브로 확인: `app.go:368` 배선 제거 → `worker_poll_test.go:70` 3/3 실패(`timed out after 1s`), `worker.go:79` 의 `if d > 0` → `if true` → `poll_interval_test.go:30` 실패. 단정 없는 테스트가 아니다.
- 직접 돌린 것: build·gofmt·vet 깨끗, `./internal/app` 54.701s 통과, `-race ./internal/app ./internal/worker ./internal/config` 58.247s 통과(경쟁 0), 타이밍 민감 픽스처 `-count=2` 통과, verify-version-sync v0.77.22 정합, 범위 이탈 0(`.github/`·`go.mod`·`docs/`·`scripts/`·`internal/version`·`web/` 무수정).
- 못 본 것: CI `-run Integration`(postgres·pgvector·Vault·OpenSearch DSN 없음) — 25ms 가 `worker.go:426` 의 `FOR UPDATE SKIP LOCKED` claim 에 닿는 것은 거기서 처음 돈다. Docker 빌드·패키징·전체 저장소 suite 도 미실행.
- 승인 후 남는 우려 ①: `worker_poll_test.go:70` 의 1초 마감은 `New`+admin HTTP 3왕복 시간에 민감하다(로컬 ~41ms 라 마진 넓음). 아주 느린 러너에서 그 구간이 1초를 넘으면 배선이 깨져도 통과하는 **거짓 통과**가 된다 — 거짓 실패는 아니니 flake 는 아니지만, 이 테스트의 회귀 탐지력을 믿기 전에 볼 자리.
- 승인 후 남는 우려 ②(릴리즈 노트): 커밋 제목의 "let an installation configure" 는 넓게 읽힌다 — `FromEnv` 가 이 필드를 안 채워 운영자용 손잡이는 없다(주석은 정확함). 커밋의 절대 수치(116.894→73.102s, -race 114.969s)도 재현 안 됨(여기선 54.701s/58.247s) — "설정 가능한 폴링 간격"·고정 수치로 옮기지 말 것.
- [러너 12:57] review approved — 리뷰 승인 (risk=low)
- [러너 12:57] pr created — https://github.com/hkjang/git-ctx/pull/45
- [러너 13:03] ci passed — 검사 5개 모두 success
- [러너 13:03] merge done — b2bbf35

## 릴리즈 노트
- v0.77.23 를 이전 릴리즈와 똑같은 모양으로 냈다 — 커밋 `d198fb3` (`release: v0.77.23`, 한 문단 한국어 본문, 트레일러 없음), 주석 태그 `v0.77.23` (`git-ctx v0.77.23`). **푸시하지 않았다.** 바뀐 파일 9개는 v0.77.20~22 와 같은 집합·같은 줄 수다(version.go, openapi.yaml, deployment.yaml, index.html, index_en.html, offline-deployment.md, test-plan.md, completion-audit.md 추가 블록, 새 release-notes-v0.77.23.md).
- 자산은 빈 배열이고 `github_release` 는 false 다 — `.github/workflows/release.yml` 이 태그 푸시로 돌면서 Docker 이미지를 빌드하고 `package-offline-image.sh` 로 `git-ctx-v0.77.23.tar.gz`·`.sha256` 를 만들어 **스스로 draft Release 를 만들고 올린 뒤 검증하고 공개**한다. 이 기계에서 만들 것이 없다.
- 직접 돌려 통과한 것: `gofmt -l ./cmd ./internal`(빈 출력), `go build -tags sqlite_fts5 ./...`, `go vet ./...`, `go test -tags sqlite_fts5 ./...`(app 54.671s), `go test ./...`(app 53.189s), `go test -tags sqlite_fts5 -race ./...`, `-race -count=2 ./internal/app ./internal/worker`(app 114.716s, `DATA RACE` 0), `verify-version-sync.sh`(`version sync verified: v0.77.23`), `test/release/version-sync.test.sh`, `kubectl kustomize deploy/kubernetes/base`(`git-ctx:v0.77.23`·`:4747`), `node --check` 4개와 `test/web/*.test.js` 5개. 로그는 `relog/verify1.log`·`relog/verify2.log`.
- 못 돌린 것: PostgreSQL·pgvector·Vault 통합(DSN 없음), `govulncheck`(미설치·설치에 네트워크 필요), Docker 빌드·오프라인 아카이브 검증 — 전부 릴리스 워크플로가 하는 일이라 노트와 감사 블록에 "태그 푸시 후 CI 수행" 으로 적었다. 비평 노트가 지적한 postgres 미검증 구간(`claim` 의 잠금 경쟁에 25ms 가 닿는 자리)이 그 잡에서 처음 도는 것도 릴리스 노트 검증 절에 남겼다.
- 비평 노트의 "승인 후 남는 우려 ②" 를 릴리즈 노트에 반영했다. 처음 쓴 초안은 커밋 제목의 "let an installation configure" 를 그대로 받아 "설치본이 설정할 수 있게" 로 헤드라인을 잡고 `116.894s → 73.102s` 를 그대로 옮겼는데, 둘 다 우려 ② 가 하지 말라던 것이라 커밋·태그를 **amend 하고 태그를 다시 만들었다**(푸시 전이라 비용 없음). 고친 모양: 헤드라인은 "이름과 주입 지점을 줍니다" + "조정하는 환경변수나 관리 설정은 **없습니다**", 수치는 "개발 기계에서 약 40%(평문 약 117초 → 약 73초), 기계와 실행마다 흔들림" 으로 귀속해서 적었다. 감사 블록의 고정 수치 줄도 "수트 벽시계 단축" 으로 바꿨다.
- 다음 역할이 조심할 것: 태그를 푸시하면 `release.yml` 이 **스스로 Release 를 만든다** — 사람이 `gh release create` 를 먼저 하면 워크플로의 `lookup_release_by_tag` 와 겹친다. 그리고 `verify-version-sync.sh` 의 completion-audit 검사는 **가장 새 "릴리스 전 검증 결과:" 블록**만 보며, 그 블록 안에서 `Kustomize`·`Docker` 가 들어간 줄 두 개가 현재 버전을 담아야 한다 — 그 두 줄 말고 다른 줄에 `v<숫자>` 를 쓰면 검사가 깨진다.

- [러너 13:21] release published — v0.77.23
- [러너 13:31] assets verified — v0.77.23 자산 2개 (이전 v0.77.22: 2)
