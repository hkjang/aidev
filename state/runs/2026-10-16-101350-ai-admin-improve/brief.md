- 과제: PR #44·#45·#46 의 CI 실패 — ① 실제 체크런 로그를 확보하고 ② 아직 아무도 돌려 본 적 없는 "콜드 Go 모듈 캐시" 조건으로 lint·test 의 첫 Go 명령을 재현해, **확정된 원인만** 고치기 (가치 4 / 위험 2 / 작업량 M)

- 왜: 세 PR(#44 `auto/2026-10-16-0228`, #45 `auto/2026-10-16-0558`, #46 `auto/2026-10-16-0830` = `1e7d704`, diff 는 `Makefile` + `docs/operations.md` 2파일뿐)의 변경 범위가 서로 무관한데 CI 의 **lint·test 두 job 이 모두** 실패했고, 2026-10-08 의 PR #43·릴리즈 v1.2.37 은 같은 워크플로로 통과했다 — 즉 저장소 내용이 아니라 10-08~10-16 사이에 바뀐 **공통 요소**가 원인일 가능성이 높다. 지난 세 회차가 in-repo 재현(lint 세 명령 / `go test -race ./...` / web npm 3단계 / `docker build`)을 **전부 통과**시켰으므로 같은 재현을 네 번째로 반복하는 것은 금지다. 이번 회차는 (a) 증거(로그)를 실제로 가져오는 길과 (b) 아직 한 번도 시도되지 않은 조건(워밍업된 `GOMODCACHE` 없이 돌리기 — 지금까지 모든 로컬 실행은 모듈 캐시가 이미 채워진 상태였다)을 연다.

- 수용 기준:
  1. **로그 확보 시도가 기록으로 남는다.** 다음을 순서대로 시도하고 각 명령의 출력(성공/실패 모두)을 `logs/` 에 남긴다: `gh auth status`; `git config --get remote.origin.url`(토큰이 박혀 있는지); `env | grep -i -E 'gh_token|github_token'`; 있으면 `gh run list --repo hkjang/ai-admin --branch auto/2026-10-16-0830 --limit 5` → `gh run view <id> --log-failed`. 자동화가 PR #44~#46 을 실제로 올렸으므로 어딘가에 push 자격증명이 있을 수 있다(이전 회차는 "gh 미인증"으로 단정하고 더 찾지 않았다 — 이번엔 위 네 가지를 실제로 돌려서 확인할 것).
  2. **콜드 모듈 캐시 재현을 돌렸고 결과가 로그로 남는다.** 빈 디렉터리를 만들어 `GOMODCACHE` 로 지정하고 CI 가 돌리는 **두 job 의 첫 Go 명령**을 각각 돌린다:
     - lint: `GOMODCACHE=$(mktemp -d) GOFLAGS=-mod=readonly go vet ./...`
     - test: `GOMODCACHE=$(mktemp -d) GOFLAGS=-mod=readonly go test -race ./... ` (DSN 있으면 같이)
     - 덧붙여 `GOMODCACHE=$(mktemp -d) go mod download` 와 `go mod verify`, 그리고 `go mod tidy -diff`(exit code 기록).
     이 조건에서 `missing go.sum entry` / 모듈 다운로드 실패 / `toolchain go1.26.6` 다운로드 실패 중 하나가 나오면 **그것이 원인**이다(두 job 모두 첫 Go 명령에서 죽는 것과 일치한다).
  3. **원인이 확정된 경우에만 고친다.** 확정 원인이 go.sum/go.mod 면 `go mod tidy` 로 생긴 차이만 커밋하고(프로덕션 Go 코드 0파일), 같은 콜드 캐시 명령을 다시 돌려 exit 0 을 보인다. 워크플로의 검사를 빼거나 `continue-on-error`·`|| true`·타임아웃 연장 같은 **느슨하게 만드는 변경은 금지**.
  4. **①②가 모두 아무 실패도 내지 않으면**(세 회차와 같은 결과) 워크플로를 추측으로 고치지 말고, 차선(아래)으로 넘어가되 **근거를 먼저 확보**한다: `curl -sS https://raw.githubusercontent.com/actions/checkout/v4.4.0/action.yml` 과 `.../checkout/v5/action.yml`(setup-go v5↔v6, setup-node v4↔v5 도 같이)에서 `runs.using:` 값을 뽑아 **현재 핀이 구 런타임(node20)인지**를 출력으로 보인다. node20 → node24 이행이 확인될 때만 세 워크플로의 핀을 올린다(아래 "건드릴 파일"). 확인되지 않으면 **아무것도 바꾸지 말고** 결과를 원장에 "외부 원인·사람 확인 필요"로 적는다.
  5. 무엇을 하든 마지막에 `make lint`(gofmt·go vet·`scripts/verify-version.sh` → `Version metadata is consistent: 1.2.37`)·`go build ./...`·`git diff --check` 가 exit 0 이고, 전용 PostgreSQL 컨테이너를 제거했으며 작업 트리가 깨끗한 것을 확인한다.

- 건드릴 파일 (최대 4개, 분기별로 둘 중 하나만):
  - 수용 기준 3 분기: `go.sum`(+ 필요 시 `go.mod`) — `go mod tidy` 가 만든 차이만. 손으로 편집하지 말 것.
  - 수용 기준 4 분기(근거 확인된 경우에만): `.github/workflows/ci.yml` — `actions/checkout`(현재 `11d5960…` # v4.4.0, 17·49행), `actions/setup-go`(`40f1582…` # v5.6.0, 18·50행), `actions/setup-node`(`49933ea…` # v4.4.0, 54행) 의 **커밋 SHA 핀 + 주석 버전**을 최신 메이저로 교체. SHA 는 `git ls-remote --tags https://github.com/actions/checkout` 로 실제 태그에서 뽑아 적고(추측 금지), `# vX.Y.Z` 주석을 같은 태그로 맞출 것. 같은 세 액션을 쓰는 `.github/workflows/release.yml`(22행 checkout) 과 `.github/workflows/pages.yml`(25행 checkout) 도 **checkout 만** 같은 SHA 로 맞춘다. `pages.yml` 의 `configure-pages`·`upload-pages-artifact`·`deploy-pages` 와 `release.yml` 의 `softprops/action-gh-release` 는 **이번에 건드리지 말 것**(Pages/릴리즈 경로는 이번 실패와 무관하고 되돌리기 어렵다).
  - 어느 분기든 `workflow.go`·`providers.go`·`internal/**` 프로덕션 Go 코드와 `web/**`·`Dockerfile`·`internal/ui/dist` 는 건드리지 않는다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 전용 폐기 DB: `docker run -d --rm --name ai-admin-ci-pg -e POSTGRES_USER=ai_admin -e POSTGRES_PASSWORD=test-password -e POSTGRES_DB=ai_admin_test -p 127.0.0.1:55611:5432 postgres:16-alpine` (끝나면 `docker rm -f`; 통합 테스트가 `ai_admin`·`aiportal` 스키마를 DROP CASCADE 한다)
  - `TEST_POSTGRES_DSN='postgres://ai_admin:test-password@127.0.0.1:55611/ai_admin_test?sslmode=disable' make ci-test` (지난 회차가 `Makefile:ci-test` 로 추가해 둔 CI test job 4단계 재현 — 이미 존재한다. `internal/server` 약 153~155초, 전체 약 2분 50초)
  - 콜드 캐시: `COLD=$(mktemp -d) && GOMODCACHE=$COLD GOFLAGS=-mod=readonly go vet ./...` / 같은 변수로 `go test -race ./...` / `GOMODCACHE=$COLD go mod download` / `go mod verify` / `go mod tidy -diff`
  - `make lint`, `go build ./...`, `git diff --check`

- 위험과 피할 것:
  - **같은 재현을 네 번째로 반복하지 말 것.** 워밍업된 모듈 캐시 상태의 `make lint`/`go test -race ./...`/`npm ci && npm test && npm run build`/`docker build` 는 세 회차가 전부 통과시켰다(2026-10-16 회차 3건). 새로 얻는 정보는 콜드 캐시 조건과 실제 로그뿐이다.
  - `.github/workflows/**` 는 보호 경로다. 분기 4 의 조건(런타임 값 출력으로 확인)이 충족되지 않으면 **한 줄도 바꾸지 말 것**. 액션 SHA 를 기억이나 추측으로 적는 것은 금지 — `git ls-remote` 출력에 있는 값만 쓴다.
  - 통과시키기 위해 검사를 빼거나 완화하는 변경(스텝 삭제, `continue-on-error`, `--timeout` 늘리기, `cache: false` 로 돌리기)은 금지. `cache: false` 는 원인 확정 전에는 "느슨하게 만들기"에 해당한다.
  - `scripts/verify-version.sh` 는 `README.md`·`docs/*.md` 안의 모든 `vX.Y.Z` 토큰이 현재 버전 하나와 일치할 것을 요구한다. 문서를 건드린다면 새 버전 토큰을 넣지 말 것(`expect_current_v_tokens`).
  - `VERSION`/`CHANGELOG.md`/`web/package*.json` 은 릴리즈 단계 전용이다. 이번 회차에 버전을 올리지 말 것.
  - 미머지 브랜치(`auto/2026-10-04-1142` 의 `f66d25c`, `auto/2026-10-01-1412` 의 `b5146f2`)의 접근을 다시 구현하지 말 것. `jsonObjectHasNUL`·`storablePreferenceObject` 이름 금지.

- 차선 후보: **GitHub Actions 핀을 최신 메이저로 올리기**(수용 기준 4 — 근거 확인이 전제). 그것마저 근거가 안 서면 두 번째 차선으로 **chat 의 `stream` 타입 오류와 두 토큰 한도 동시 지정(`stream_invalid`/`max_tokens_conflict`/`max_tokens_invalid`, `providers.go:611-634` 부근)의 400 계약을 기존 `internal/server/chat_request_json_integration_test.go` fixture 안에 서브테스트로 고정**(프로덕션 0파일, 새 통합 테스트 파일을 만들지 말 것 — 패키지 시간이 늘어난다). 착수 전 `grep -rn max_tokens_conflict internal/server` 로 이미 덮여 있지 않은지 확인할 것(미확인).

- 견적(근거·범위·예비):
  - 분해와 근거: ① 로그 확보 시도 5~10분(명령 4개) · ② 콜드 캐시 재현 — `go vet` 콜드 1~3분 + `go test -race ./...` 콜드 약 3~4분(직전 회차 실측 전체 2분 30초 + 모듈 다운로드) · ③ 원인 확정 시 수정과 재검증 10~15분 · ⑤ 최종 검증 `make lint`+`go build`+`make ci-test` 약 3~5분(지난 회차 실측 `make ci-test` 2m48.992s, `internal/server` 153~155초).
  - 범위: 40~60분(10번 중 8번). 최선 25분(①에서 로그가 바로 나오고 원인이 명확할 때), 최악 70분(콜드 재현이 모듈 다운로드로 느리고 분기 4 의 근거 확인까지 가는 경우).
  - 가정: 구현자 세션에 네트워크와 docker 가 있고(지난 세 회차 실측), 전용 PostgreSQL 컨테이너를 새로 띄울 수 있다. 이 정찰 세션에는 둘 다 권한으로 막혀 있었다.
  - 컨틴전시(과제 안): ②가 통과할 확률이 높으므로(같은 조건의 다른 재현이 3회 연속 통과) 분기 4 와 두 번째 차선을 미리 적어 두었다 — 추가 시간 15분.
  - 관리예비(과제 밖, 사람 몫): 원인이 계정/러너 수준(Actions 쿼터·빌링·러너 이미지 롤아웃)이면 저장소 안에서 고칠 수 없다. 이 경우 구현자는 시간을 더 쓰지 말고 원장에 "외부 원인" 으로 올릴 것.

- 미확인(추측으로 적지 않기 위해 명시):
  - PR #46 의 실패한 job 이름·단계·로그를 **보지 못했다**(이 정찰 세션은 `gh`·네트워크 명령이 권한으로 막혀 있었다). "lint·test 두 job 모두 실패"는 이전 두 회차의 기록을 이어받은 것이다.
  - 콜드 모듈 캐시에서 실패하는지는 **미확인**(이 세션에서 `go env`·`go mod` 명령도 권한으로 막혔다). 가설일 뿐이며, 통과하면 그대로 기록하고 고치지 말 것.
  - `actions/checkout@v4.4.0` 가 node20 런타임이고 v5 가 node24 라는 것은 **일반 지식이며 이 저장소에서 확인하지 않았다** — 수용 기준 4 의 `curl` 로 직접 확인할 것.
