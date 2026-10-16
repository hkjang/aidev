- 과제: 런타임 게이트웨이가 거절당한 DLP 보고를 로그로 남긴다 — 이번에는 세 버전 거울을 함께 올려서 (가치 3 / 위험 2 / 작업량 S)

- 왜: `cmd/runtime-proxy/dlp.go` 의 `send()` 는 관제면 응답의 `StatusCode` 를 보지 않고 `_ = response.Body.Close()` 로 끝낸다(현재 main 에서 직접 확인, 159-164행) — 관제면이 401/500 으로 보고를 버려도 Pod 로그에는 아무 흔적이 없고, 운영자는 `dlp.tool` 감사 행이 비어 있는 이유를 알 수 없다. 이 변경은 지난 회차에 커밋 6059496 으로 만들어졌지만 `BASE_VERSION` 만 0.27.0 으로 올리고 그 번호를 적는 다른 두 곳을 두고 가서 릴리즈를 깨뜨렸고, 그 뒤 0fd2810 의 `check_version_mirrors` 게이트가 바로 그 불완전 상향을 막게 되었다 — 이제 세 거울을 같이 올리면 안전하게 올릴 수 있다.

- 수용 기준:
  1) 관제면이 2xx 밖의 상태로 DLP 보고를 거절하면 Pod 로그에 그 상태 문자열이 한 줄 남는다(기존 `report failed` 와 다른 문구). 토큰·DLP 검출 원문·응답 본문은 로그에 들어가지 않는다.
  2) 200·202·204 처럼 2xx 안쪽 응답은 추가 줄을 남기지 않는다 — 검사가 특정 코드 목록이 아니라 **범위**임을 테스트가 고정한다.
  3) `send` 의 best-effort 계약이 그대로다: 반환형 없음, 재시도 없음, 블로킹 없음. 거절 관제면과 성공 관제면에서 `inspect` 의 반환값(치환 문자열 + `*dlp.Result`)이 동일함을 테스트가 `reflect.DeepEqual` 로 단정한다.
  4) `go test -count=1 ./...` 가 통과한다 — 즉 `BASE_VERSION` 0.27.0 과 **같은 번호**가 `compose.yaml` 빌드 인자와 `internal/buildinfo` 기본값에 들어가 있다(이 둘은 Go 스위트가 지킨다).
  5) `bash scripts/release-catalog-images.sh check-versions` 가 exit 0 이다 — 올리기 **전**에는 `base: inputs changed since v0.263.0 but BASE_VERSION is still 0.26.0` 로 거절되는 것을 먼저 보고, 올린 뒤 통과를 확인한다.

- 건드릴 파일 (프로덕션 4 + 테스트 1):
  - `cmd/runtime-proxy/dlp.go:send` — `response.Body.Close()` 뒤에 `if response.StatusCode < 200 || response.StatusCode > 299 { log.Printf("agenthub-dlp report rejected: %s", response.Status) }` 한 블록. **왜 이 자리인가**를 주석 산문으로 적을 것(이 패키지 관례). 상태 문자열만 적는다 — 응답 본문에는 방금 보고한 이벤트가 그대로 실려 있을 수 있고 `r.token` 은 애초에 로그 재료가 아니다.
  - `cmd/runtime-proxy/dlpreport_test.go` (신규) — httptest 관제면 + `log.SetOutput` 캡처. `scanlimit_test.go:21-40` 이 이 패키지의 모범이다(이번 회차에 실제로 읽음): `httptest.NewServer` 로 관제면을 띄우고 `inspect := scannerFor("rrn","block")` 뒤 `inspect.report = &reporter{url: control.URL, runtimeID: "rt-1", token: "s3cret", client: control.Client()}` 로 배선한 다음 실제 `inspect` 를 통과시킨다 — **분기 함수를 손으로 부르지 말 것**. `scannerFor` 는 `cmd/runtime-proxy/mcp_test.go:258` 에 있다. 저장소 전체 `_test.go` 에 `log.SetOutput` 은 **한 건도 없다**(이번 회차 grep 으로 재확인) — `capturePodLog` 같은 헬퍼를 이 파일 안에 새로 만들면 된다.
  - `BASE_VERSION` — `0.26.0` → `0.27.0`.
  - `compose.yaml:29` — `BASE_VERSION: 0.26.0` → `0.27.0`.
  - `internal/buildinfo/buildinfo.go:12` — `BaseVersion = "0.26.0-dev"` → `"0.27.0-dev"`. (`-dev` 접미사 유지. 게이트는 번호만 비교한다.)

- 검증 명령:
  - `go test -count=1 ./cmd/runtime-proxy` — 이번 회차에 base(bfe7b8a)에서 실제로 돌려 `ok … 0.386s` 로 통과함을 확인했다(수정 전 기준선).
  - `go test -count=1 ./...` — 세 거울이 맞는지 보는 쪽이 여기 있다. 이번 회차에 실제 위치를 확인했다: `internal/api/imagesources_test.go:403`(compose 빌드 인자)과 `internal/runtime/spawner_baseimage_test.go:82`(`buildinfo.BaseVersion` 과 `agenthub-base` 기본 이미지 태그). 즉 `./cmd/runtime-proxy` 만 돌려서는 거울 누락이 **보이지 않는다** — 지난번 사고의 기전이다. `internal/buildinfo/workflows_test.go:273-283` 은 `BASE_VERSION` 의 현재 값을 파일에서 읽어 가상의 상향을 구성하므로 번호를 올려도 그대로 돈다.
  - `go test -race -p 1 -count=1 ./cmd/... ./internal/...` — `-p 1` 은 과거 병렬 오염 때문. `internal/buildinfo` 가 9p gitdir 탓에 로컬에서만 17~18초 걸리는 것은 정상이다(측정으로 확인된 환경 비용, 저장소 결함 아님)
  - `bash scripts/release-catalog-images.sh check-versions` 와 `... validate` — **올리기 전/후 둘 다** 돌려 게이트가 막았다가 통과하는 것을 기록할 것
  - `go build ./...`, `go vet ./cmd/runtime-proxy ./internal/buildinfo`, `gofmt -l cmd/runtime-proxy internal/buildinfo`, `git diff --check`
  - `docker compose config --quiet` — compose.yaml 을 건드리므로

- 위험과 피할 것:
  - **세 거울을 반드시 한 커밋에 같이 올릴 것.** 이것이 지난 회차의 실패 지점이다. `BASE_VERSION` 만 올리면 Go 스위트가 떨어지고(`compose build args do not pin BASE_VERSION to 0.27.0` / `buildinfo for BASE_VERSION = "0.26.0", want "0.27.0"`), 아무것도 올리지 않으면 `check-versions` 가 막는다. Go 스위트는 올린 **뒤에** 다시 돌릴 것 — 지난번에 깨진 이유가 바로 그 순서였다.
  - `cmd/runtime-proxy` 는 base 이미지 sourcePaths 다 — 그래서 상향이 필요하다. 반대로 `scripts/` 와 `web/` 은 어느 sourcePaths 에도 없다(이전 회차들이 확인).
  - `check_versions` 는 `buildDependencies` 전파를 쓰지 않는다(`create_plan` 전용, 스크립트 272-304행). 지목되는 이미지는 `base` 하나뿐이니 `JUPYTER_VERSION` 등 다른 `*_VERSION` 은 **손대지 말 것**.
  - `send` 를 재시도·블로킹·에러 반환으로 바꾸지 말 것 — best-effort 주석(dlp.go:139-143)이 명시적으로 그렇게 적혀 있고, 바꾸면 보고 실패가 도구 호출을 실패시킨다.
  - 거절을 **세어 요약하지 말 것** — 별개의 보류 아이디어(2/3/M)이고 reporter 에 상태·시계를 들여야 해서 이 회차 범위가 아니다.
  - 보호 경로를 건드리지 않는다: `auth.go`·`mcpoauth.go`·세션 게이트웨이·`internal/store/migrations`·`.github/workflows`.
  - 지난 회차 코드(브랜치 `auto/2026-10-16-0018`, 커밋 6059496)를 **그대로 cherry-pick 하지 말 것** — 운영자 지시는 릴리즈를 깨뜨린 접근을 같은 방식으로 다시 올리지 말라는 것이다. 같은 한 줄을 새로 쓰고 세 거울을 함께 올려, 이번에는 게이트와 Go 스위트가 둘 다 통과하는 상태로 만들 것.

- 차선 후보: 배포 매니페스트와 README 가 박아 둔 `agenthub-base` 태그를 검사로 묶는다 — `deploy/kubernetes/ollama-bridge.yaml:19` 과 `README.md:173` 이 `agenthub-base:v0.26.0` 을 직접 적는데 아무 테스트도 보지 않아, `BASE_VERSION` 이 올라가면 오프라인/minikube 설치가 이번 릴리즈에 안 실린 태그를 끌어온다. 주의: 저장소에 픽스처용 가짜 태그가 11곳 있어(`agenthub-base:v0.4.0` 등) "모든 태그가 최신" 은 **틀린 규칙**이다 — `deploy/` 같은 배포 산출물 경로로 좁히고, 옛 커밋 재생에서 거짓 실패가 나지 않는지 `b742060` 로 먼저 확인할 것. (1순위를 하면 이 태그 두 곳이 바로 낡으므로, 1순위 안에서 **값만** 같이 고치는 것도 선택지다 — 다만 검사를 새로 만드는 일은 이 회차에 넣지 말 것.)
