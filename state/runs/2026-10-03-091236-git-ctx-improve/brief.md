- 과제: CI/릴리즈의 `govulncheck` 단계가 보고하는 도달 가능한 취약 의존성을 닫아 PR #44 를 통과시키기 (가치 4 / 위험 2 / 작업량 S)
- 왜: PR #44 는 비평 승인까지 끝났는데 코드와 무관한 검사 하나, `Known vulnerabilities` 잡만 failure 라 머지가 막혀 있다(회차 노트 `[러너 08:37] ci failed — 성공이 아닌 검사: Known vulnerabilities=failure`). 같은 단계가 `release.yml` 에도 있어 릴리즈까지 같은 이유로 멈춘다 — 취약점 DB 는 매일 갱신되므로 코드를 안 바꾼 트리도 다음 날 실패한다. 취약 모듈을 올리면 두 워크플로가 같이 풀린다.

- 수용 기준:
  1) `govulncheck ./...` 가 `No vulnerabilities found` 로 exit 0 (수정 전에는 exit 3 + 특정 `GO-2026-nnnn` 보고 — **두 출력을 모두 기록**).
  2) 바뀐 파일은 `go.mod`·`go.sum` 뿐이고 `go 1.25.0`·`toolchain go1.26.6` 두 줄은 그대로. (툴체인을 올려야만 닫히는 **표준 라이브러리** 권고라면 예외 — 그때는 `toolchain` 한 줄만 올리고 `Dockerfile:1` 의 `FROM golang:1.26-bookworm` 과 상한이 맞는지 확인.)
  3) `go build -tags sqlite_fts5 ./...`·`go vet ./...`·`go test -tags sqlite_fts5 -count=1 ./...` 전부 통과(동작 변화 없음을 보이는 것이 이 과제의 테스트다 — 새 단위 테스트를 만들 대상이 아니다).
  4) `.github/workflows/ci.yml`·`release.yml` 의 govulncheck 단계는 **한 글자도 바뀌지 않는다**(`-show`/`|| true`/제외목록 추가 금지).
  5) 의존성만 올리는 릴리즈는 이 저장소 관례상 릴리즈 노트를 남긴다 — `docs/release-notes-v0.77.14.md` 를 본으로, 닫은 권고 ID·도달 경로·"재색인 불필요" 를 적는다.

- 건드릴 파일:
  - `go.mod`·`go.sum` — `go get <module>@<fixed>` + `go mod tidy` 결과만.
  - `docs/release-notes-v0.77.<next>.md` — 신규(위 5번). 선례: `docs/release-notes-v0.77.14.md`.
  - (표준 라이브러리 권고인 경우에만) `go.mod` 의 `toolchain` 줄, 그리고 `Dockerfile:1`.
  - 프로덕션 Go 파일은 0개가 정상이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go install golang.org/x/vuln/cmd/govulncheck@v1.7.0` → `govulncheck ./...`  ← CI(`ci.yml:85-88`)·릴리즈(`release.yml:168-173`)와 **같은 두 줄**. 모듈 캐시에 `golang.org/x/vuln@v1.7.0` 의 `.zip` 이 이미 있다(`$GOMODCACHE/cache/download/golang.org/x/vuln/@v/v1.7.0.zip`). 바이너리도 `/home/hkjang/go/bin/govulncheck` 에 있으나 **버전 미확인** — 판정은 v1.7.0 로 할 것.
  - `go build -tags sqlite_fts5 ./...` / `go vet ./...` / `gofmt -l ./cmd ./internal`(빈 출력)
  - `go test -tags sqlite_fts5 -count=1 ./...` (수 분, `./internal/app` 혼자 ~100초) / `go test -tags sqlite_fts5 -race -count=1 ./internal/mcp`
  - `bash scripts/verify-version-sync.sh` (릴리즈 노트를 새로 쓰면)

- 조사 순서 (이 순서를 지키면 추측이 안 들어간다):
  1) **먼저 실제 권고 ID를 확보한다.** `gh run view --log --job <Known vulnerabilities job id>` 또는 PR #44 체크 로그. 그게 막히면 `govulncheck ./...` 를 직접 돌려 받는다. ID 없이 버전만 올리지 말 것 — 아무 효과 없는 diff 가 된다.
  2) 권고가 가리킨 모듈을 `go get <module>@<그 권고가 fixed 로 적은 최소 버전>` 로 올린다. 메이저 업그레이드 금지, 필요한 최소 패치만.
  3) `govulncheck ./...` 재실행 — 다른 권고가 줄줄이 나오면 **모듈 수준 0건까지** 올린다(v0.77.14 가 그렇게 했다).

- 지금 코드에서 확인한 사실 (추측 아님):
  - `ci.yml:72-88` 잡 `vulnerabilities` / name `Known vulnerabilities`, `release.yml:168-173` 단계 `Verify reachable dependencies` — 둘 다 `go install golang.org/x/vuln/cmd/govulncheck@v1.7.0` + `govulncheck ./...` 똑같은 쌍이다.
  - 두 호출 모두 **`-tags sqlite_fts5` 가 없다**(같은 워크플로의 빌드·테스트는 `ci.yml:64-70` 처럼 태그를 쓴다). 이번 실패와는 무관하지만, 태그 없이 돌면 FTS5 분기는 분석 범위 밖이라는 뜻이다 — **이번 회차에 고치지 말 것**(워크플로 수정이고 범위 밖이다. ideas.json 에 따로 올려 두었다).
  - 선례가 정확히 같은 모양으로 끝났다: v0.77.14 는 `grpc@v1.82.1` 의 **GO-2026-6348**(HTTP/2 DATA 프레임 조각화 OOM, `search.Query` → `transport.NewHTTP2Client` 로 도달)을 `grpc@v1.83.2` 로 닫고 `go.mod`·`go.sum` 만 바꿨다. 지금 `go.mod` 는 이미 grpc v1.83.2·x/net v0.58.0 이므로 **그 권고는 아니다**.
  - 모듈 캐시에 go.mod 보다 새 버전이 이미 내려와 있는 직접/간접 의존성(= 오프라인에서도 올릴 가능성이 있는 후보, `.zip` 유무는 미확인): `github.com/go-jose/go-jose/v4` v4.1.4→**v4.1.5**, `golang.org/x/net` v0.58.0→**v0.59.0**, `github.com/jackc/pgx/v5` v5.9.2→**v5.10.0/v5.11.0**, `golang.org/x/oauth2` v0.36.0→**v0.37.0**, `github.com/coreos/go-oidc/v3` v3.20.0→**v3.21.0**, `google.golang.org/grpc` v1.83.2→**v1.84.0**, `google.golang.org/protobuf` v1.36.11→**v1.36.12**. 캐시에 있다는 것은 과거에 누가 내려받았다는 뜻일 뿐 **이번 권고의 대상이라는 증거가 아니다** — 1)의 ID 확보가 먼저다.
  - **미확인**: 실제 권고 ID·대상 모듈·도달 경로. 이 세션에서는 `govulncheck`·`go run`·`gh`·웹 도구가 모두 권한 차단이고 네트워크 확인도 못 했다. 그래서 위 후보는 순위가 아니라 목록이다.

- 위험과 피할 것:
  - **워크플로를 느슨하게 만드는 어떤 변경도 금지** — 이번 회차의 명시적 금지 사항이고, 운영자가 되풀이해 말한 "릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것" 에 걸린다.
  - PR #44 의 `internal/mcp/budget.go` 변경은 **건드리지 말 것**. 이미 비평 승인을 받았고(`fencesContext`→`fencesContent(tool)` 설계), 이번 실패와 인과가 없다. 같은 브랜치에서 의존성 커밋만 얹는다.
  - `go mod tidy` 가 `go`/`toolchain` 줄을 바꾸거나 간접 의존성을 대량으로 흔들면 되돌리고 `go get` 대상을 좁힐 것. diff 가 go.mod·go.sum 밖으로 번지면 과제를 쪼갠 것이다.
  - 메이저 업그레이드(v4→v5 류) 금지. `go-jose`·`go-oidc`·`oauth2` 는 `internal/auth` 의 OIDC·세션 경로가 쓰므로, 올렸으면 `go test -tags sqlite_fts5 -count=1 ./internal/auth ./internal/app` 를 반드시 돌릴 것(보호 경로다).
  - govulncheck 는 DB 가 매일 바뀐다 — "어제 통과" 는 증거가 아니다. 수정 전/후 출력을 같은 세션에서 연달아 찍어 둘 것.

- 차선 후보: 추적된 디버그 산출물 `c.txt`·`server.log` 제거 + `.gitignore` 등록 (가치 2 / 위험 1 / S). 1순위가 성립하지 않는 경우는 하나뿐이다 — 권고 ID를 끝내 확보할 수 없을 때. 그때는 **추측으로 버전을 올리지 말고** 1순위를 "진단 불가" 로 적고 이 위생 작업만 내라(지우기 전에 두 파일 내용을 열어 비밀값 유무를 확인할 것).
