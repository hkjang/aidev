# 과제서 — 2026-10-17-052859-ReSSO-improve (수정 과제)

> **정찰의 한계를 먼저 밝힙니다.** 이 세션에서는 `Bash` 권한이 거의 전부 거부되었습니다
> (`go version` 조차 "This command requires approval"). `gh run list` 도, `WebFetch` 로
> GitHub Actions API 를 읽는 것도 거부되었습니다. 따라서 **실패한 단계와 그 로그를 직접 보지
> 못했습니다.** 아래 원인 후보는 전부 **파일을 읽어 추린 것**이며, 어느 것이 실제 실패인지는
> **미확인**입니다. 구현자는 **1단계(재현)를 반드시 먼저** 하고, 재현된 실패에 맞는 후보를
> 골라 고치세요. 재현 없이 후보 중 하나를 "고치는" 것은 금지입니다.

- 과제: 릴리즈 워크플로가 v0.9.104 태그에서 두 번 같은 이유로 실패하는 것을 재현해 원인을 고치기 (가치 5 / 위험 2 / 작업량 M)
- 왜: `edf2c2f release: ReSSO v0.9.104` 태그에서 `Offline image release` 워크플로가 두 번 같은 이유로 실패해 v0.9.104 의 오프라인 이미지 자산(`resso-v0.9.104.tar.gz`·`release-sha256.txt`)이 GitHub Release 에 붙지 않았습니다. 이 저장소의 산출물은 그 아카이브 하나이므로, 릴리즈는 기록상 존재하지만 운영자가 가져갈 것이 없는 상태입니다. 고치면 v0.9.104 가 실제로 설치 가능해지고, 같은 자리에서 다음 릴리즈가 또 멈추지 않습니다.

## 1단계 — 재현 (이것이 과제의 절반입니다)

실패한 런의 로그를 먼저 보세요. 그것이 이 과제서의 추측 전부를 대체합니다.

```
gh run list --workflow "Offline image release" --limit 10
gh run view <실패한 run id> --log-failed | tail -120
```

로그를 볼 수 없으면, 릴리즈 워크플로의 각 단계를 **태그 커밋에서** 순서대로 로컬 재현합니다.
릴리즈 잡은 CI verify 와 달리 **PostgreSQL 만** 준비하고 **`-race` 없이** 돌립니다 — 지난
회차가 로컬에서 통과시킨 `make test`(세 서비스 + `-race`)와 모양이 다릅니다. 이 차이가
"CI 는 통과했는데 릴리즈만 실패" 를 설명할 수 있는 첫 자리입니다.

```
git fetch --tags && git checkout v0.9.104      # 또는 edf2c2f
# release.yaml:55-58 "Test backend" 과 같은 모양: PostgreSQL 만, -race 없이
test_env="$(scripts/test-services.sh)" && eval "$test_env"
unset RESSO_TEST_LDAP_URL RESSO_TEST_LDAPS_URL RESSO_TEST_LDAP_CA
go test ./... && go vet ./...
# release.yaml:64-66 "Test frontend"
cd web && npm ci && npm run test && npm run build && cd ..
# release.yaml:67-76 "Build offline image" — 여기서 실패하면 Go/Node 베이스 이미지 핀 문제
docker build --platform linux/amd64 \
  --build-arg VERSION=v0.9.104 --build-arg COMMIT="$(git rev-parse HEAD)" \
  --build-arg "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)" --tag resso:v0.9.104 .
# release.yaml:77-107 "Test built offline image" 와 108-138 "Load the published archive back"
# 도 같은 스크립트를 그대로 복사해 돌립니다 (scripts/smoke-test.sh, scripts/offline-load.sh).
```

## 원인 후보 (읽어서 추린 것 — 전부 미확인, 재현이 하나를 지목하면 그것만 고칩니다)

1. **`golang:1.26.9-alpine` 이미지 부재 (Build offline image, `Dockerfile:11`).** 직전 PR #43 의
   `dcfd26f` 가 `go.mod`의 `toolchain go1.26.9` 와 `Dockerfile:11` 의 빌더 베이스를 함께
   1.26.9 로 올렸습니다(둘이 1.26.9 로 일치하는 것은 이번에 읽어 확인). CI 의 lint/verify 잡은
   **Dockerfile 을 전혀 빌드하지 않으므로**, 그 태그가 Docker Hub 에 없거나 `actions/setup-go`
   가 해결하지 못하면 **릴리즈 워크플로만** 결정적으로 실패하고 매 재실행이 같은 이유로
   실패합니다. 증상: `manifest for golang:1.26.9-alpine not found` 또는
   `go.mod requires go >= 1.26.9 (running go 1.26.x)`. **미확인**: 1.26.9 태그의 실제 존재 여부
   (이 세션에서 `docker`·네트워크를 쓸 수 없었습니다).
   - 고치는 방향: 두 핀(`go.mod:5`, `Dockerfile:11`)을 **실제로 존재하는 같은 패치 버전**으로
     함께 맞춥니다. `cmd/resso/pins_test.go:96 TestToolchainPinsAgreeAcrossTheBuild` 가 그 일치를
     계속 대조하므로 한쪽만 바꾸면 테스트가 잡습니다. **되돌려서 통과시키는 것은 금지** — 1.26.9
     상향의 목적은 `net/http`·`crypto/tls` 의 패치된 stdlib 이고, 내려가면 govulncheck 의
     stdlib 취약점 10건이 돌아옵니다. 존재하는 더 높은 패치(1.26.10 등)로 올리는 것은 허용.
2. **`Test backend` 단계의 테스트 실패 (`release.yaml:55-58`).** 직전 PR 이 더한
   `internal/httpserver/integration_test.go` 의 `TestIntegrationRefreshSurvivesASessionLookupItCannotRun`
   은 `ALTER TABLE sso_sessions RENAME TO sso_sessions_hidden` 로 장애를 만듭니다. 패키지별
   테스트는 `openHTTPIntegrationStore`(`integration_test.go:245-302`)가 **테스트마다 고유 스키마**
   (`resso_http_test_<uuid>` + `search_path`)를 만들고 끝에 `DROP SCHEMA ... CASCADE` 하므로
   패키지 간 간섭은 **구조적으로 없습니다**(이번에 읽어 확인). 따라서 이 후보는 `-race` 없는
   실행·다른 PostgreSQL 마이너(릴리즈 서비스는 `postgres:17-alpine`, `release.yaml:23)·러너의
   느린 I/O 로 드러나는 타이밍 차이에 한정됩니다. 재현에서 실제로 실패하는 테스트가 나오면
   그 테스트의 전제를 고치세요(장애 주입 범위를 좁히거나 복구를 단언 전으로 당기는 쪽).
3. **`cmd/resso/pins_test.go` 의 핀 대조 테스트 하나가 태그 커밋에서만 깨짐.** 이 파일의
   여섯 테스트는 저장소의 여러 파일에 같은 사실이 중복된 것을 대조합니다. 릴리즈 커밋
   `edf2c2f` 는 CHANGELOG 와 compose 파일을 함께 스탬프하는데, 그 중 한 곳이 어긋나면 태그
   커밋에서만 깨집니다. 이번에 읽어 **배제한** 것: `TestComposeFilesRunTheReleaseThisTreeDescribes`
   (`compose.offline.yaml:5`·`compose.maintenance.yaml:5` 모두 `resso:${RESSO_VERSION:-v0.9.104}`
   로 CHANGELOG 최신과 일치), `TestImageVersionDefaultCannotPassForARelease`
   (`Dockerfile:18`·`:34` 모두 `v0.0.0-dev`), `TestToolchainPinsAgreeAcrossTheBuild` 의 Go 쪽
   (1.26.9 ↔ 1.26.9). **미확인**: Node 핀 셋(`Dockerfile:3` `node:22.23.0-alpine` ↔
   `ci.yaml:32,49` ↔ `release.yaml:61` 모두 22.23.0 으로 보이지만 전수 대조는 안 했습니다),
   `TestLinterVersionIsTheSameLocallyAndInCI`(Makefile `GOLANGCI_VERSION` ↔ `ci.yaml:23` v2.13.1),
   `TestThePostgresVersionIsTheSameEverywhereItIsNamed`(`scripts/test-services.sh` ↔
   `release.yaml:23` ↔ `ci.yaml` ↔ README).
4. **`Publish the release with the image archive` (`release.yaml:143-172`) 의 마지막 단언.**
   `test "$attached" = "release-sha256.txt,resso-v0.9.104.tar.gz"` 는 릴리즈에 **정확히 그 두
   자산만** 있을 것을 요구합니다. 사람이 손으로 만든 릴리즈에 다른 자산이 하나라도 붙어
   있으면, 업로드가 모두 성공한 뒤 이 줄에서 실패하고 재실행도 같은 자리에서 같은 이유로
   실패합니다. 고치는 방향: 단언을 **두 자산의 존재 확인**으로 바꿉니다(부분집합 검사).
   이것은 느슨하게 만드는 것이 아니라 단언의 대상을 바로잡는 것입니다 — 자산이 **붙었는지**가
   이 단계의 계약이고, 릴리즈에 다른 파일이 더 있는지는 이 워크플로가 통제하는 사실이 아닙니다.
   **다만 재현이 이 단계를 지목할 때만** 손대세요.

## 수용 기준

1. 실패한 릴리즈 런의 **실제 실패 단계와 메시지**를 회차 디렉터리에 기록한다(로그 인용 또는
   로컬 재현 출력). 추측이 아니라 관찰된 문구가 원장에 남는다.
2. 수정 **전**에 같은 실패를 로컬에서 재현해 그 출력을 남기고, 수정 **후** 같은 명령이 통과한다.
3. 릴리즈 워크플로의 검증을 **느슨하게 하지 않았다**: 삭제·`continue-on-error`·`|| true`·
   테스트 스킵·단언 제거가 diff 에 없다. 후보 4처럼 단언 자체를 바로잡는 경우에는, 바뀐 단언이
   여전히 "두 자산이 모두 붙었다" 를 증명한다는 것을 테스트나 주석으로 남긴다.
4. 후보 1 이면: `go.mod:5` 의 `toolchain` 과 `Dockerfile:11` 의 베이스가 **존재하는 같은 패치
   버전**이고 `go test ./cmd/resso -run '^TestToolchainPinsAgreeAcrossTheBuild$' -v` 가 PASS 하며,
   `govulncheck ./cmd/... ./internal/...` 의 stdlib 취약점이 1.26.9 때와 같거나 적다(0건 유지).
5. 릴리즈 잡과 같은 모양의 백엔드 검증이 통과한다: PostgreSQL 만 켜고 LDAP 변수 없이
   `go test ./... && go vet ./...` exit 0, 그리고 그 실행에 **SKIP 이 있다면 그것이 LDAP 게이트
   뿐**임을 `-run 'TestIntegration|TestDirectory' -v` 로 확인한다.
6. CHANGELOG 에 이번 수정이 한 줄로 기록된다(이 저장소 관례: 한국어, 왜를 포함).

## 건드릴 파일 (후보에 따라 — **최대 3개**로 끝내세요)

- 후보 1: `go.mod:5`(toolchain) + `Dockerfile:11`(golang 베이스) — 두 핀을 존재하는 같은 패치로.
  필요하면 `CHANGELOG.md` 한 줄. 프로덕션 코드 0개.
- 후보 2: 실패한 테스트 1개(`internal/httpserver/integration_test.go` 의 해당 함수) — 장애 주입
  범위/복구 시점만. **프로덕션 코드(`internal/httpserver/oidc.go`·`internal/store/oidc.go`)는
  건드리지 마세요** — 8d3b77a 의 500 `server_error` 계약은 이번 과제의 대상이 아닙니다.
- 후보 3: 어긋난 핀을 가진 **그 한 파일**만.
- 후보 4: `.github/workflows/release.yaml:171-172` 의 단언 한 줄 + 위 주석.

## 검증 명령 (이 저장소에서 실제로 도는 것)

```
# 서비스 준비 — 저장소 루트, 같은 셸
test_env="$(scripts/test-services.sh)" && eval "$test_env"

# 릴리즈 잡과 같은 모양 (PostgreSQL 만, -race 없이)
go test ./... && go vet ./...
go test ./cmd/resso -count=1 -v            # pins_test.go 여섯 개

# 전체 (지난 회차들이 기준선으로 쓴 것)
go test -race ./internal/store -count=1
go test -race ./internal/httpserver -count=1
make test
PATH="$(go env GOPATH)/bin:$PATH" golangci-lint run ./cmd/... ./internal/...
PATH="$(go env GOPATH)/bin:$PATH" GOTOOLCHAIN=auto govulncheck ./cmd/... ./internal/...
git diff --check
```

이미지까지 바꿨다면 (후보 1):
```
docker build --platform linux/amd64 --build-arg VERSION=v0.9.105-test \
  --build-arg COMMIT="$(git rev-parse HEAD)" \
  --build-arg "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)" --tag resso:v0.9.105-test .
docker image inspect --format '{{.Os}}/{{.Architecture}}' resso:v0.9.105-test   # linux/amd64
docker image rm resso:v0.9.105-test
```

## 위험과 피할 것

- **워크플로를 느슨하게 만들어 통과시키는 것은 금지** (러너의 명시 지시). `continue-on-error`,
  `|| true`, 단계 삭제, 테스트 스킵, 이미지 로드 왕복 검증(`release.yaml:120-138`) 축소 전부 해당.
  그 단계들의 주석이 왜 그것이 거기 있는지 적어 두었으니 먼저 읽으세요.
- **`make lint` 는 PATH 를 스스로 잡지만, 툴체인을 올린 직후에는 `GOPATH/bin` 의 `goX.Y.Z` 심이
  `make lint` 를 깨뜨린 선례가 있습니다**(운영자 메모). Go 만 바뀐 diff 는 `golangci-lint` 를
  패키지 목록과 함께 직접 돌리세요.
- **govulncheck 실행 파일이 구형 Go 로 빌드돼 분석에 실패한 선례**가 있습니다(2026-10-08 회차).
  도구를 빌드한 Go 와 분석 대상 Go 를 구분하고, 필요하면 같은 버전을 현재 툴체인으로 재빌드하세요.
- **서비스 환경변수가 없으면 통합 테스트는 SKIP 이고 `go test` 는 exit 0** 입니다. "통과" 를
  주장하기 전에 SKIP 수를 확인하세요. `scripts/test-services.sh --stop` 은 컨테이너와 인증서를
  **모두 삭제**합니다 — 검증 끝에 자동 실행하지 마세요.
- `npm` 이 조상 `node_modules/.bin` 의 Node 20 을 선점해 jsdom 이 터진 선례가 있습니다
  (의존성 문제가 아닙니다). `web/package.json` 의 `npm_node_execpath` PATH 고정을 유지하세요.
- 보호 경로: `internal/httpserver/auth.go`·`oidc.go`·`middleware.go`, `internal/store/migrations`.
  이번 과제는 이들을 건드리지 않고 끝나야 합니다.
- 사람이 반려한 접근(코드 교환 저장소 장애를 500 으로, `5bed9dc`)은 재제출 금지.

## 차선 후보

재현이 끝내 아무 실패도 보여 주지 않고(= 릴리즈 실패가 러너의 환경·권한 문제였고 코드가
아니면), 그 사실을 원장에 적고 다음을 하세요: **`Makefile:77` 과 `scripts/test-services.sh`
머리말의 단독 `eval "$(scripts/test-services.sh)"` 예제 정정** (가치 2 / 위험 1 / 작업량 S).
`c1b71bb` 가 `docs/user-federation.md` 에서 고친 것과 **같은 실패 은닉**이 두 곳에 남아
있습니다 — 준비 스크립트가 exit 1 이어도 `eval ""` 이 성공해 안내받은 사람이 서비스 없이
테스트를 돌립니다. `assignment && eval && <명령>` 형태로 바꾸고, 문구 변경이므로
`go test ./cmd/resso -count=1` 과 `bash -n scripts/test-services.sh` 로 검증합니다.
