# 과제서 — 2026-10-17-220917-postra-improve (정찰)

- 과제: 릴리즈 워크플로(`.github/workflows/release.yml`)가 v0.25.9 태그에서 두 번 연속 실패한 **실제 단계**를 로그로 고정하고, 그 단계의 결함을 고친다 (가치 5 / 위험 2 / 작업량 M)

- 왜: 러너가 이번 회차를 `verify-failed` 로 적재했고 "릴리즈 워크플로가 같은 이유로 두 번 실패" 라고 지정했다. 태그 `v0.25.9` 는 이미 `23479bc`(HEAD)에 붙어 있으므로 릴리즈 산출물(정적 바이너리·오프라인 이미지·SBOM·체크섬)이 배포되지 않은 상태이며, 재실행이 같은 지점에서 또 죽는다는 것은 환경 요동이 아니라 **재현 가능한 결함**이라는 뜻이다. 고치면 v0.25.9 가 실제로 공개되고 다음 태그도 막히지 않는다.

## 먼저 할 일 (정찰이 못 한 것 — 반드시 1단계로)
정찰 세션의 Bash 샌드박스가 `gh`·`go`·`npm` 실행을 거부해서 **실패 로그를 직접 보지 못했다. 아래 원인 후보는 전부 "미확인" 이다.** 구현자는 추측으로 시작하지 말고 먼저 로그를 가져와라:

```
gh run list --workflow=release.yml --limit 5
gh run view <failed-run-id> --log-failed | tail -120
```

로그가 지목한 단계가 아래 후보 중 하나면 그 항목의 처방을 쓰고, 아니면 로그를 근거로 그 단계를 고쳐라. **로그를 못 가져오는 경우에만** 아래 순서로 로컬 재현을 돌려 첫 실패를 찾아라(모두 이 저장소에서 실제로 도는 명령이다):

```
# 1) 프런트 자산 드리프트 (release.yml:40-52 와 동일)
cd web && npm ci --no-audit --no-fund && npm run build && cd ..
git status --porcelain --untracked-files=all -- internal/transport/spa/assets   # 빈 출력이어야 한다

# 2) 정적 바이너리 (release.yml:54-62 와 동일 플래그)
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags="-s -w -X postra/internal/platform/build.Version=v0.25.9 -X postra/internal/platform/build.Commit=$(git rev-parse HEAD) -X postra/internal/platform/build.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o dist/postra-0.25.9-linux-amd64 ./cmd/postra

# 3) 오프라인 이미지 (release.yml:64-72)
go run scripts/mkimage.go dist/postra-0.25.9-linux-amd64 dist/postra-0.25.9.tar postra:0.25.9 v0.25.9 $(git rev-parse HEAD)

# 4) Go SBOM (release.yml:74-77) — 과거에 git 레이아웃으로 깨진 전력이 있는 단계
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.10.0
cyclonedx-gomod app -json -output dist/postra-0.25.9-sbom.cdx.json -main cmd/postra .

# 5) 프런트 SBOM (release.yml:79-81)
cd web && npm sbom --sbom-format=cyclonedx --omit=dev --package-lock-only > ../dist/postra-0.25.9-frontend-sbom.cdx.json && cd ..
```
끝나면 `dist/` 를 지우고 `git status --porcelain --untracked-files=all` 이 깨끗한지 확인하라(릴리즈 산출물은 커밋하지 않는다).

## 원인 후보 (확인한 것 / 미확인 구분)

**확인한 것 (정찰이 파일을 읽어서 배제했다 — 다시 조사하지 마라):**
- Go 핀은 일관적이다. `3a10d84` 가 `ci.yml:15`·`contracts.yml:20`·`release.yml:14`·`Dockerfile`·`go.mod`(`go 1.26.0` / `toolchain go1.26.9`)를 **한 번에** 1.26.9 로 올렸다. 워크플로 간 Go 버전 불일치는 원인이 아니다.
- 릴리즈 노트 파일이 있다. `docs/releases/v0.25.9.md` 존재하고 태그 `v0.25.9` 는 그 파일을 포함한 `23479bc` 를 가리킨다 → `release.yml:96-101` 의 `--notes-file` 분기는 성립하고 `--generate-notes` 폴백도 실패 원인이 아니다.
- `scripts/mkimage.go` 의 인자 계약은 맞다. `main()` 은 `len(os.Args) < 4` 만 거부하고 version·commit 은 선택 인자다(`scripts/mkimage.go:17-33`) → `release.yml:66-71` 이 넘기는 5개 인자는 유효하다.
- `govulncheck` 는 `release.yml` 에 **없다**(`ci.yml:163-175` 전용). 따라서 릴리즈 워크플로 실패의 원인일 수 없다 — Go 핀 재조정으로 "고쳐 보려는" 시도를 반복하지 마라. `3a10d84` 가 이미 그 경로다.

**미확인 (가능성 순서, 전부 로그로 확인해야 함):**
1. **`gh release create` 가 이미 존재하는 릴리즈/자산에 걸린다** (`release.yml:102-109`). 1차 실행이 뒷단계에서 죽었거나 릴리즈를 만든 뒤 자산 업로드에서 죽었다면, 재실행은 매번 `release already exists` 로 같은 지점에서 죽는다 — "두 번 같은 이유" 와 모양이 정확히 맞는다. 처방: 같은 태그를 멱등하게 다루도록 고친다(예: 기존 릴리즈가 있으면 `gh release upload --clobber` 로 자산만 올리고, 없으면 `create`). **이것은 게이트 완화가 아니다** — 검증 단계를 끄는 것이 아니라 발행 단계를 재실행 가능하게 만드는 것이다. 단, 로그가 이 단계를 지목할 때만 손대라.
2. **프런트 자산 드리프트** (`release.yml:46-52`). 같은 검사가 `ci.yml:48-54` 에도 있어 CI 가 통과했다면 배제되지만, `check-latest: true`(release.yml:25) 가 Node 22 의 최신 패치를 가져오므로 커밋된 자산을 만든 Node 와 다를 수 있다. 처방: 드리프트가 재현되면 **자산을 재생성해 커밋**한다(검사를 느슨하게 하지 않는다). 교훈: `web/` 아래 새 파일을 만들면 Tailwind v4 가 스캔해 해시가 바뀐다(과거 33건 드리프트).
3. **Go SBOM 단계** (`release.yml:74-77`). `cyclonedx-gomod v1.10.0` 은 오래된 도구이고 Go 1.26.9 로 빌드·실행된다. 이 저장소에는 이 도구가 git 레이아웃 때문에 메인 모듈 버전을 못 읽어 실패한 전력이 있다(과거 실측: worktree 실패 / full clone·shallow 통과). 처방: 로그가 이 단계면 `-main` 경로·모듈 버전 해석을 실제로 재현한 뒤 고친다. **`actions/checkout` 의 fetch-depth 를 바꾸는 것은 과거 실측을 근거로 재사용 금지** — 재현한 로그가 그걸 지목할 때만.
4. **프런트 SBOM 단계** (`release.yml:79-81`). `working-directory: web` 에서 `> ../dist/...` 로 쓴다. `dist/` 는 `release.yml:56` 에서 만들어지므로 순서는 맞다 — 다만 `npm sbom` 이 실패해도 리다이렉션이 빈 파일을 남겨 뒤의 `sha256sum` 이 아니라 여기서 죽는지 로그로 구분하라.

- 수용 기준:
  1) 실패한 릴리즈 실행의 로그(또는 로컬 재현 출력)에서 **죽은 단계 이름과 종료 메시지 원문**이 회차 노트에 인용돼 있다. "아마 X 였다" 는 불가.
  2) 그 단계의 실패가 수정 **전** 로컬에서 재현되고(명령과 출력을 남긴다), 수정 **후** 같은 명령이 통과한다. 재현이 불가능하면(순수 GitHub 측 상태) 그 사실과 근거를 적고, 수정은 멱등성 결함 같은 **코드/스크립트의 실제 결함**에만 적용한다.
  3) 검증을 끄거나 느슨하게 하지 않았음을 diff 로 보일 수 있다 — `continue-on-error`, `|| true`, `exit 0`, 자산 드리프트 검사 삭제, `--audit-level` 완화, 핀 제거가 **0건**. (이것은 러너의 명시 금지 사항이다.)
  4) 수정 후 `go build ./...`, `go vet ./...`, `make lint-format`, `go run ./cmd/postra-contracts -check`, `git diff --check` 가 모두 exit 0 이고 `git status --porcelain --untracked-files=all -- internal/transport/spa/assets` 가 비어 있다(자산을 의도적으로 재생성한 경우는 예외 — 그때는 재생성 사실을 노트에 적는다).

- 건드릴 파일 (여섯 개 안쪽, 로그가 지목한 하나의 단계만):
  - `.github/workflows/release.yml` — 로그가 지목한 단계 한 곳만. 보호 경로이므로 최소 변경, 주석으로 근거(실패 run id/메시지)를 남긴다.
  - `internal/transport/spa/assets/*` — 후보 2가 재현될 때만, `make frontend` 로 재생성한 결과를 그대로 커밋.
  - `scripts/mkimage.go` — 후보가 이 스크립트로 좁혀질 때만.
  - `docs/releases/v0.25.9.md` 또는 `README.md` — 릴리즈 절차에 걸린 함정을 한 절로 기록(선택, 1순위가 끝난 뒤에만).
  - 그 밖의 프로덕션 Go/TS 코드는 **건드리지 않는다**. 이번 과제는 릴리즈 파이프라인 결함이다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gh run list --workflow=release.yml --limit 5` / `gh run view <id> --log-failed`
  - 위 "로컬 재현" 1~5번 블록 중 실패한 단계 + 그 뒤 단계 전부
  - `go build ./...` · `go vet ./...` · `make lint-format` · `go run ./cmd/postra-contracts -check` · `git diff --check`
  - Go 코드를 건드렸다면 `go test -race -count=1 -timeout=600s ./internal/application ./internal/adapters/imap ./internal/adapters/pop3`(약 92s / 5s / 5s). 워크플로·자산만 바꿨다면 Go 테스트는 범위 밖이라고 적어라.
  - 자산을 재생성했다면 `make frontend-check` 와 `git status --porcelain --untracked-files=all -- internal/transport/spa/assets`.

- 위험과 피할 것:
  - **게이트 완화 금지** (러너 명시). `npm audit --omit=dev --audit-level=high`(ci.yml:44), `govulncheck ./...`(ci.yml:175), 자산 드리프트 검사(ci.yml:48 / release.yml:46), 버전 핀 — 어느 것도 약화시키지 마라.
  - `.github/workflows/*` 는 보호 경로다. `ci.yml`·`contracts.yml` 은 이번 실패가 거기라고 로그가 말하지 않는 한 **열지 마라**. 특히 `fetch-depth` 변경을 과거 worktree 실측을 근거로 재사용하지 마라(그 기록은 CI 변경의 근거로 쓸 수 없다고 이미 판정됐다).
  - Go 버전 재조정으로 고치려는 시도 금지 — `3a10d84` 가 이미 그 경로이고 `release.yml` 에는 govulncheck 가 없다.
  - 태그를 다시 밀거나 지우지 마라(`v0.25.9` 는 `23479bc` 에 있다). 릴리즈 재발행이 필요하면 워크플로를 멱등하게 고치고 러너/사람이 재실행하게 둔다.
  - `web/` 아래에 **새 파일을 만들지 마라** — Tailwind v4 가 주석까지 스캔해 전량 자산 드리프트를 만든 전력이 있다.
  - 어댑터·진단 코드(`internal/adapters/{imap,pop3}`, `internal/application/sync_diagnostics.go`)는 이번 과제와 무관하다. 같이 건드리면 리뷰가 커진다.

- 차선 후보: **greeting 실패에 "첫 응답이 IMAP 환영 메시지가 아니다" 운영 힌트 추가** (가치 3 / 위험 2 / 작업량 S) — `internal/application/sync_diagnostics.go` 의 `syncHint` 에 `stage=greeting` + `protocol=imap` + `class=other` 분기를 넣는다. 지금 최종 Summary 는 `서버 환영 메시지 수신 단계: 알 수 없는 오류로 중단되었습니다` 로 끝나 포트·프로토콜 오설정을 지목하지 못한다(`623dee1` 가 거부는 넣었지만 힌트는 안 넣었다). 주의: 같은 조합이 **과대 길이 greeting**(`errUnframed`)에도 걸리므로 문구를 두 경우 모두에 맞게 쓰거나 분기를 더 좁혀야 한다. 기존 greeting 힌트(implicit TLS·동시 접속) 뒤에 둘 것. 릴리즈 파이프라인이 막혀 있으면 이 과제는 릴리즈되지 않으니 1순위가 성립하지 않을 때만 고르라.

## 정찰이 실행하지 못한 것 (정직한 한계)
이 세션의 Bash 샌드박스가 `go`, `npm`, `gh` 실행을 모두 거부했다(`go version` 조차 승인 요구). 따라서 **이번 회차에는 어떤 테스트·빌드·워크플로 조회도 실행하지 못했다.** 위의 "확인한 것" 은 전부 파일 읽기(`release.yml`, `ci.yml`, `go.mod`, `scripts/mkimage.go`, `git log`/`git show --stat`/`git tag`)로만 확정한 것이고, 실패 단계 자체는 미확인이다. 구현자는 1단계 로그 확보를 건너뛰지 마라.
