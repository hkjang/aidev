- 과제: GitHub Actions CI 워크플로 추가 — build·vet·test 를 PR 마다 자동 실행 (가치 5 / 위험 1 / 작업량 S)
- 왜: 저장소 루트에 `.github` 디렉터리 자체가 없어(이번 회차 `ls -a` 로 직접 확인) 러너가 올리는 PR 이 전부 "CI 검사 없음" 으로 막히고, 그 탓에 2026-09-20 이후 구현 5 회차가 전부 "성공" 판정을 받고도 pinned base `e9f3fb2` 에 하나도 들어오지 못했다(timeparse.go:114 의 map-range 가 이 HEAD 에 그대로 남아 있는 것으로 확인). `origin` 이 실제 GitHub 저장소(https://github.com/hkjang/jasql.git)이므로 러너가 이미 로컬에서 돌리는 build·vet·test 를 워크플로로 옮기면 이후 모든 회차의 머지 경로가 열린다 — 이번 한 번의 변경이 앞으로의 모든 회차를 살린다.

- 수용 기준:
  1) `.github/workflows/ci.yml` 이 `push`(main) 과 `pull_request` 에서 돌고, `actions/checkout` + `actions/setup-go` 로 Go 를 깔아 `go build ./...` → `go vet ./...` → `go test ./...` 를 이 순서로 실행한다. `setup-go` 는 `go-version-file: go.mod` 를 쓰거나 go.mod 의 `1.25.0` 과 일치하는 버전을 명시한다(하드코딩하면 go.mod 와 어긋날 때 조용히 깨지므로 `go-version-file` 권장).
  2) 워크플로가 외부 서비스 없이 완결된다: `-tags oracle` 을 쓰지 않고(기본 stub 드라이버), Oracle Instant Client·PostgreSQL 서비스 컨테이너를 붙이지 않으며, `JASQL_TEST_PG` 를 설정하지 않는다(설정하지 않으면 PG 통합 테스트는 skip).
  3) 증명은 문자열 검사가 아니라 실행 출력으로 한다: 로컬에서 `go build ./...`, `go vet ./...`, `go test ./...` 세 명령을 실제로 돌려 통과 출력을 회차 노트에 붙이고, 가능하면 `actionlint` 또는 `python3 -c "import yaml,sys;yaml.safe_load(open('.github/workflows/ci.yml'))"` 로 YAML 이 파싱되는 것을 실행으로 확인한다(워크플로 파일이 존재한다는 사실만으로는 동작 증거가 아니다).

- 건드릴 파일:
  - `.github/workflows/ci.yml` (신규) — 위 3 단계 잡 하나. 캐시는 `setup-go` 기본 캐시로 충분하니 따로 `actions/cache` 를 붙이지 말 것.
  - (선택) `docs/development.md` — "CI" 한 줄 추가. 같은 파일의 Go 버전·의존성 서술이 낡았지만(차선 후보 참고) **이번 과제에서 같이 고치지 말 것**. 범위 누적으로 실패하는 전형이다.

- 검증 명령:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`  ← 느림: 2026-09-22 측정으로 `internal/catalog` 약 57s, `internal/mcp` 약 10s. 워크플로 `timeout-minutes` 는 여유 있게(10 이상) 둘 것.
  - `gofmt -l ./internal ./cmd` ← 참고용. **CI 실패 조건으로 넣지 말 것**(아래 위험 참조).

- 위험과 피할 것:
  - `.github/workflows` 는 보호 경로다. **워크플로 파일 하나만** 추가하고 Go 소스·go.mod·go.sum·Dockerfile·scripts 는 건드리지 말 것. 사람 승인이 필요한 변경이므로 diff 가 작을수록 통과한다.
  - `-tags oracle` 을 켜면 godror 가 cgo + Oracle Instant Client 헤더를 요구해 CI 가 빌드 단계에서 실패한다. `internal/oracle/driver_oracle.go` / `driver_stub.go` 의 빌드 태그 분리를 이번 회차에 확인했다. 기본(태그 없음) 경로만 돌릴 것.
  - **gofmt 게이트를 넣지 말 것**: 2026-09-24 정찰이 `internal/oracle/profile.go`, `internal/oracle/oracle_test.go` 의 gofmt 드리프트를 보고했다(이번 회차에는 `gofmt` 실행이 샌드박스 승인에 막혀 **재확인하지 못했다 — 미확인**). 드리프트가 남아 있으면 첫 CI 실행부터 빨갛게 되고, 그걸 고치려고 소스를 포맷하면 보호 경로 PR 에 무관한 diff 가 섞인다. 포맷 정리는 별도 회차로.
  - 운영자 규칙 준수: 워크플로에 시크릿·토큰·외부 배포(도커 푸시, 릴리즈 업로드)를 넣지 말 것. 이번 과제는 검사만 한다.
  - 매트릭스(여러 OS/Go 버전)로 넓히지 말 것. `ubuntu-latest` 단일 잡이면 충분하고, 매트릭스는 실행 시간만 늘린다.
  - 이번 회차에 `go build`/`go vet`/`go test`/`gofmt` 를 **직접 실행하지 못했다(샌드박스 승인 거부) — 통과 여부는 2026-09-22 프로필 기록에 의존한 미확인 정보다.** 구현자는 워크플로를 쓰기 전에 세 명령을 먼저 로컬에서 돌려 green 인지 확인하고, 만약 지금 HEAD 에서 실패하는 테스트가 있으면 **워크플로를 그 실패를 감추도록 고치지 말고**(`continue-on-error` 금지) 회차 노트에 사실대로 적은 뒤 차선 후보로 전환할 것.

- 차선 후보: `docs/development.md` 의 개발자 가이드를 소스와 동기화 (가치 3 / 위험 1 / 작업량 S) — 문서의 "Go 1.24+" vs go.mod `1.25.0`(이번 확인), "외부 의존성 0 / go.sum 없음" vs go.mod 의 godror v0.51.0·pgx/v5 v5.10.0·x/crypto v0.53.0 + go.sum 존재(이번 확인), "MCP 도구 24 개" vs `internal/mcp/server.go` 의 `tools()` 등록 목록(2026-09-22 프로필은 42 개라고 기록 — 구현자가 직접 세어 확정할 것). `TestServeStdio` 가 도구 개수를 단정하고 있으니 그 테스트가 쓰는 수를 정본으로 삼으면 된다.
