# 과제서 (2026-10-16 정찰)

- 과제: `fddc0b8` 이 넣은 **검증되지 않은 Go 1.27.2 핀**(`server/go.mod:3` + `Dockerfile:14` 의 다이제스트)이 CI·릴리즈를 매 실행 같은 지점에서 결정론적으로 깨뜨리는 것 찾아 고치기 (가치 5 / 위험 2 / 작업량 S)

- 왜: 직전 회차의 수리 커밋 `fddc0b8`("fix: raise Go toolchain to 1.27.2 and x/net to v0.60.0 to clear govulncheck")가 `server/go.mod` 의 `go` 지시자를 `1.27.2` 로, `Dockerfile:14` 를 `golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d` 로 올렸다. 이 두 값은 **이 저장소 안에서는 검증할 수 없는 외부 사실**(해당 Go 패치 릴리즈가 실제로 존재하는가 / 그 다이제스트가 실제 매니페스트인가)이고, 둘 중 하나라도 틀리면 `actions/setup-go`(ci.yml:46-49·82-85, release.yml 의 setup-go) 또는 `docker build`(ci.yml:118-125 "Build service image", release.yml "Build release image")가 **입력이 바뀌지 않는 한 매 실행 같은 메시지로 실패**한다 — 관측된 "같은 이유로 두 번 실패" 와 모양이 정확히 일치한다. 고치면 v0.2.50 릴리즈가 다시 진행되고, 다음 회차가 같은 자리에서 또 멈추지 않는다.

- 수용 기준:
  1) 실패하는 단계를 **출력으로** 확정한다. 아래 세 가지를 순서대로 돌려 어느 것이 빨간불인지 그 출력(명령·종료코드·에러 한 줄)을 결과 보고에 인용한다:
     a. `cd server && go build ./...` — 로컬 툴체인이 `go 1.27.2` 를 만족하지 못하면 `go: downloading go1.27.2 ...` 뒤에 "toolchain go1.27.2 not found"/"unknown toolchain" 류로 즉시 죽는다. 이것이 죽으면 **Go 1.27.2 가 존재하지 않는다**는 1차 증거다.
     b. `docker pull golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d` — `manifest unknown` 이면 `Dockerfile:14` 가 원인이다. (과거 회차가 실제 `docker build`·`docker pull postgres:16-alpine` 를 돌렸으므로 이 러너에는 Docker 네트워크가 있다.)
     c. `docker manifest inspect golang:1.27.2-alpine` — 태그 자체가 없으면 1.27.2 미출시가 확정된다. 있으면 여기서 나오는 **실제 index 다이제스트**가 정답 값이다.
     네트워크가 막혀 a~c 를 전부 못 돌리면 그 사실을 적고 (아래 "둘 다 아닐 때") 로 넘어간다.
  2) 확정된 원인을 **외부 사실에 맞추는 방향**으로 고친다 — 느슨하게 만들어 통과시키는 것은 금지(아래 위험 항목).
     - 1.27.2 가 없으면: `server/go.mod:3` 의 `go` 지시자와 `Dockerfile:14` 의 태그를 **실제로 존재하는 최신 1.27.x** 로 내리고, 다이제스트는 `docker manifest inspect` 가 돌려준 값을 그대로 쓴다. 그 버전에서 `govulncheck` 가 다시 빨간불이면 그 취약점을 **다른 수단**(해당 모듈 버전 상향)으로 막고, 무엇을 어떻게 막았는지 커밋 메시지에 적는다.
     - 1.27.2 는 있고 다이제스트만 틀렸으면: `Dockerfile:14` 의 `@sha256:` 한 값만 실제 값으로 교체한다(핀을 **지우지 말 것**).
  3) 고친 뒤 **릴리즈 워크플로와 같은 검증을 같은 순서로** 로컬에서 재현해 초록불을 확인하고 출력을 인용한다(아래 "검증 명령"). 최소한 `go build`/`go vet`/`govulncheck`/`docker build --platform linux/amd64`/`scripts/verify-release.sh` 는 반드시 포함한다.
  4) `.github/workflows/` diff 는 **0 줄**이다(워크플로를 고쳐서 통과시킨 것이 아님을 증명).
  5) 프로덕션 동작 변경 0: `server/internal/**`, `webapp/src/**` 에 손대지 않는다(툴체인 상향이 요구하는 컴파일 오류 수정이 생기면 그 파일만, 최소로, 보고에 명시).
  6) 결과 보고에 **"수정 전 RED 1줄 / 수정 후 GREEN 1줄"** 을 나란히 적는다(2026-10-04 회차가 단계 이름조차 못 밝혀 차선으로 끝난 것을 되풀이하지 않기 위해).

- 건드릴 파일 (프로덕션 3개 이내):
  - `server/go.mod:3` — `go` 지시자. 존재하는 패치 버전으로만.
  - `Dockerfile:14` — `FROM golang:<ver>-alpine@sha256:<실제 다이제스트> AS server-build`. 태그와 다이제스트를 **함께** 맞춘다.
  - `server/go.sum` (필요할 때만) — 2)에서 모듈 버전을 추가로 올렸을 때 `go mod tidy` 가 쓰는 결과만.
  - (참고용·수정 금지) `.github/workflows/ci.yml:46-49,82-85,107-125`, `.github/workflows/release.yml` 의 setup-go·Test Go modules·Scan reachable Go vulnerabilities·Build release image 단계.

- 검증 명령 (이 저장소에서 실제로 도는 것, 릴리즈 순서):
  - `cd server && go build ./...`
  - `cd server && go vet ./...`
  - `cd server && go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`
  - `MOYRO_TEST_POSTGRES_DSN=<DSN> go test -race -p 1 -count=1 ./...` (server/ 에서. DSN 은 `moyro-pg-improve`/55433 컨테이너. `Exited` 면 `docker start` 선행. DSN 은 자식 프로세스 환경에만 주고 출력/문서에 남기지 않는다. DSN 없으면 DB 테스트가 전부 skip 되어 빠른 ok 를 통과로 오인한다 — httpapi 가 0.2초면 안 돈 것이다.)
  - `cd plugin-sdk/go && go test -race ./...` 그리고 `go vet ./...`
  - `docker build --platform linux/amd64 --provenance=false --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 --build-arg VERSION=v0.2.50 --build-arg COMMIT=c0ab918 --tag moyro:v0.2.50 .`
  - `docker save --output /tmp/moyro-release-image.tar moyro:v0.2.50` → `gzip -n -9 < /tmp/moyro-release-image.tar > moyro-v0.2.50.tar.gz` → `EXPECTED_COMMIT=<c0ab918 의 full sha> bash scripts/verify-release.sh moyro:v0.2.50 moyro-v0.2.50.tar.gz`
  - `bash scripts/check-source-sizes.sh`, `node scripts/verify-pages.mjs`
  - `gofmt -l` (변경 파일), `git diff --check`, `git diff --stat .github/` (0 이어야 한다)

- 위험과 피할 것:
  - **금지(게이트 완화)**: 다이제스트 핀을 지우고 `golang:1.27-alpine` 같은 떠 있는 태그로 바꾸기, `govulncheck`/`npm audit` 단계에 `|| true`·`continue-on-error`·`--audit-level` 완화·`-show`/제외 플래그 추가, 단계 삭제·조건부화, 타임아웃 연장. 운영자 규칙상 워크플로를 느슨하게 해 통과시키는 것은 이번 회차의 실패로 취급된다.
  - `npm audit` 과 `govulncheck` 는 **라이브 취약점 DB**다 — 시점에 따라 결과가 달라진다. 이번 실패가 여기서 났다면 "두 번 같은 이유" 와 모양이 맞는지(같은 CVE·같은 모듈인지) 먼저 확인하고, 아니면 그쪽으로 끌려가지 말 것.
  - 보호 경로: `server/internal/{auth,session,principal,oidcauth,native_mcp_oauth}`, `server/internal/store/migrations`, `.github/workflows/` — 이번 과제는 전부 건드릴 필요가 없다.
  - `webapp/tsconfig.json` 에 `types` 가 없어 `@types/node` 가 끼면 `useDraft.test.tsx:196` 이 TS2345 로 깨진다(2026-09-09 교훈). 웹을 건드리지 않은 이번 회차에서 이 오류가 보이면 **자기 변경 탓이 아니다** — 포맷/tsconfig 변경을 섞지 말 것.
  - `gh` 는 이 저장소의 과거 다섯 회차에서 전부 **미인증**이었다. `gh run view` 로 로그를 보려 시간을 쓰지 말고(이번 정찰도 확인 못 했다) 로컬 재현으로 단계를 특정하라.
  - 미확인(정찰이 확인하지 못한 것, 추측으로 적은 것): ① Go 1.27.2 가 실제 출시되었는지 — 이 세션에는 네트워크 도구(`curl`/WebFetch)·`go version` 실행 권한이 **없었다**. ② `Dockerfile:14` 의 다이제스트가 실제 매니페스트인지 — 같은 이유로 미확인. ③ 실패한 단계 이름 — `gh` 미인증으로 미확인. ④ 확인된 사실: `server/go.sum:347-348` 에 `golang.org/x/net v0.60.0` 의 h1/go.mod 해시가 **둘 다 있다**(그 버전은 존재한다), `plugin-sdk/go/go.mod` 는 `go 1.22` 로 그대로다, `1.27` 문자열이 들어간 파일은 `server/go.mod` 와 `Dockerfile` **두 개뿐**이다(grep).
  - **둘 다 아닐 때**(a·b·c 가 전부 초록): 추측으로 아무 파일이나 고치지 말고, 릴리즈 순서를 그대로 로컬 재현해(위 검증 명령 전체) 처음 빨간불이 나는 단계를 찾아 **그 단계만** 고쳐라. 전부 초록이면 2026-10-04 회차처럼 "트리에 깨지는 게이트가 없다" 를 출력과 함께 보고하고 차선 후보로 넘어가라.

- 차선 후보: `postsByIDsReactions`(`server/internal/httpapi/compat_wave_handlers_late.go:304-356`)가 DB 장애를 200 `{}`("반응 없음")으로 보고하는 것 차단 — `ChannelForPost` 오류 `continue`, `IsMember` 오류 무시, `ListForPost` 오류 `continue` 중 **`ListForPost` 장애만** 기존 500 계열로 분리(가치 3 / 위험 2 / S). 10-08 의 `rows.Err()` 회차와 같은 계열이지만 함수가 다르고, 실제 DB 장애 주입(격리 스키마 안에서 뷰로 0 나누기 / `ALTER TYPE`)으로 RED 를 만들 수 있다.
