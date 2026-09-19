# 과제서 — 2026-09-19-213406 (수정 과제)

- 과제: PR #7(`auto/2026-09-19-1923`, 커밋 e8e0960 "web/dist/.gitkeep 추적") 의 CI 실패 원인을 로그로 특정해 고치고, 같은 단계를 로컬에서 재현 통과시킨다 (가치 4 / 위험 2 / 작업량 S)
- 왜: 지난 회차는 로컬 검증(verify.txt: `go test ./...` ok, `npm run build` ok, `dist/.gitkeep 0.00 kB` 재생성)은 통과했지만 GitHub CI(`.github/workflows/ci.yml`, PR 이벤트)가 실패해 verify-failed 로 끝났고, 이 수정이 main 에 못 들어가면 다음 회차마다 `go build ./...` 가 깨끗한 체크아웃에서 계속 실패한다. 이번 정찰 환경에서는 `gh` 가 승인 없이 실행되지 않아 **실패한 단계·로그를 직접 보지 못했다(미확인)** — 구현자가 첫 단계로 로그를 읽어 원인을 확정한 뒤 고쳐야 한다.

## 정찰이 확인한 사실
- PR #7 의 변경은 4파일뿐: `.gitignore`(+5/-1: `web/dist/` 제거 → `!web/dist/`, `web/dist/*`, `!web/dist/.gitkeep`), `README.md`(1줄), `web/dist/.gitkeep`(0바이트), `web/vite.config.ts`(`keepDistPlaceholder()` 플러그인, `generateBundle` 에서 `emitFile({type:"asset", fileName:".gitkeep", source:""})`).
- `ci.yml` 단계 순서: `npm ci`(web) → `node scripts/licenses.mjs --check` → `npm run build` → govulncheck → `npm ci`(tests) → playwright install → `node scripts/build-docs.mjs && node tests/docs.mjs` → `node --test tests/natural-date.mjs tests/silent-sso.mjs …` → `go test -race -failfast -count=1 -timeout 45m ./...`(브라우저 모듈 env 전부 켬) → `go vet ./...` → `createdb madi_browser_ci` → `bash scripts/verify-browser.sh`. 별도 job `offline-image`(Dockerfile 빌드·`scripts/verify-image.sh`).
- `.dockerignore` 에 `web/dist` 가 있어 Docker 컨텍스트에는 `.gitkeep` 이 들어가지 않고, Dockerfile 은 web 스테이지에서 `npm run build` 뒤 `COPY --from=web /build/web/dist ./web/dist` 하므로 Docker 쪽은 이 변경과 무관할 가능성이 높다(미확인 — 로그로 확인).
- `web.Assets`(`web/embed.go:7 //go:embed all:dist`) 는 `cmd/madi/main.go:52` 에서 `fs.Sub(web.Assets,"dist")` 로만 쓰이고, `internal/server/*_test.go`·`tests/*.mjs`·`scripts/*` 어디에도 `.gitkeep`/`.gitignore`/dist 파일 목록을 검사하는 코드는 없다(grep 결과 0건). 따라서 "`.gitkeep` 이 dist 에 있어서 테스트가 깨진다" 는 가설은 근거가 약하다.
- `tests/deployment-contract/deployment_test.go:83` 은 release.yml 문자열만 검사하며 PR #7 은 워크플로를 바꾸지 않았다.

## 수용 기준
1) PR #7 의 실패한 CI job/step 이름과 실패 메시지 첫 줄이 회차 노트(journal.md)에 인용되어 있다(`gh run list --branch auto/2026-09-19-1923 --limit 3` → `gh run view <id> --log-failed | head -80`). 원인이 PR #7 의 변경 때문인지, main 에서도 재현되는 기존 실패(플레이키 브라우저 시험·타임아웃 등)인지가 명시된다.
2) 원인이 PR #7 변경이면: 해당 파일을 고치고 같은 단계를 로컬에서 재현해 통과한다. 원인이 PR #7 과 무관한 기존 실패면: 그 실패 지점(테스트/스크립트)을 고치되 **워크플로 파일(`ci.yml`/`release.yml`)의 단계 제거·`continue-on-error`·타임아웃 연장·env 끄기는 금지**. 어느 쪽이든 커밋 메시지에 CI 실패 단계명을 적는다.
3) 수정 후에도 e8e0960 의 목적이 유지된다: `git clean -xfd web/dist && go build ./... && go vet ./...` exit 0, `cd web && npm run build` 뒤 `git status --short web/dist` 빈 출력.
4) 변경한 단계에 해당하는 검증(아래 표)을 로컬에서 실제로 돌려 출력이 journal 에 남는다. 전체 `go test -race ./...`·브라우저 시험 전체는 돌리지 않는다(러너 예산).

## 건드릴 파일(원인별 — 로그로 확정 후 한 갈래만)
- A. `npm run build`/`vite` 단계 실패 → `web/vite.config.ts:keepDistPlaceholder` (예: Vite 가 CI 의 `NODE_ENV`/캐시 조건에서 `.gitkeep` 이름의 asset 을 거부하는 경우 `closeBundle` 에서 `fs.writeFileSync(path.join(outDir,".gitkeep"),"")` 로 대체).
- B. `node scripts/licenses.mjs --check` 또는 `tests/docs.mjs` 실패 → 로그를 그대로 읽고 해당 스크립트가 무엇을 비교하는지 확인(정찰 grep 에선 dist/.gitignore 를 보지 않음 — 이 갈래면 PR 과 무관한 기존 실패일 가능성 큼).
- C. `go test -race … ./...` 단계 실패 → `-failfast` 라 첫 실패 테스트 하나만 보인다. 그 테스트 이름으로 `MADI_TEST_POSTGRES_DSN=… go test -race -count=1 -run '^<이름>$' ./internal/server` 로 로컬 재현. `.gitkeep` 과 무관한 테스트면 기존 플레이키/회귀이며, main(fd2c3b5)에서 같은 테스트를 한 번 더 돌려 기준을 잡는다.
- D. `offline-image` job 실패 → `Dockerfile`/`scripts/verify-image.sh`/`.dockerignore`. `.dockerignore` 의 `web/dist` 때문에 `COPY . .` 단계에서 `web/dist` 가 비어 `go build` 가 embed 오류를 내는 순서인지 확인(Dockerfile 15행이 그 뒤에 dist 를 복사하므로 정상이어야 함 — 미확인).
- 어느 갈래든 `.github/workflows/*.yml` 은 수정하지 않는다.

## 검증 명령(이 저장소에서 실제로 도는 것)
- `gh run list --branch auto/2026-09-19-1923 --limit 3` / `gh run view <id> --log-failed | head -120` (원인 확정, 첫 단계)
- `git clean -xfd web/dist && go build ./... && go vet ./...`
- `cd web && npm ci && npm run build && git status --short web/dist` (빈 출력이어야 함)
- `node scripts/licenses.mjs --check`
- `node --test tests/natural-date.mjs tests/silent-sso.mjs tests/usability/model.test.mjs tests/usability/recorder.test.mjs tests/pdf-font-sources.mjs tests/publish-verified-screenshots.test.mjs`
- `go test -count=1 ./tests/deployment-contract/ ./web/ ./cmd/...`
- 갈래 C 일 때만: 임시 PostgreSQL 17 컨테이너로 `MADI_TEST_POSTGRES_DSN=postgres://… go test -race -count=1 -run '^<실패 테스트명>$' ./internal/server`

## 위험과 피할 것
- 워크플로를 느슨하게 만들어 통과시키는 것 금지(단계 제거·continue-on-error·타임아웃·env 조정·`-failfast` 제거 모두 해당).
- 전체 `go test -race ./...`(45m)·`scripts/verify-browser.sh` 전체를 로컬에서 돌리지 말 것 — 지난 두 회차가 러너 예산으로 실패했다. `-run` 패턴으로 좁힌다.
- `.gitignore` 의 재포함 3줄 순서(`!web/dist/` → `web/dist/*` → `!web/dist/.gitkeep`)는 3행 generic `dist/` 규칙 때문에 필요했다 — 되돌리면 e8e0960 의 목적이 사라진다.
- `tests/docs.mjs` 가 다시 그리는 `docs/manuals/product-*.png` 스크린샷은 무관한 바이너리 diff 이므로 커밋 전 되돌릴 것.
- auth(`identity_saml.go`·`integrations_oidc.go`)·`backup_tables.go`·SQL 은 이번 과제와 무관 — 건드리지 않는다.
- 실패가 PR 과 무관한 기존 플레이키 시험이라 판단될 때도 "재실행하면 된다" 로 끝내지 말고 그 시험의 실패 메시지를 근거로 안정화(대기 조건·정리 순서) 한 가지를 고칠 것. 근거 없이 `t.Skip` 을 넣는 것은 금지.

## 차선 후보
- CI 로그를 끝내 얻지 못하면(gh 인증 불가): 갈래 A 를 선제 적용 — `keepDistPlaceholder` 를 `emitFile` 대신 `closeBundle` + `fs.writeFileSync` 로 바꿔 Vite/Rollup 의 asset 파일명 규칙에 의존하지 않게 하고, 위 로컬 검증 전부 통과시킨 뒤 회차 노트에 "CI 로그 미확인, 추정 수정" 이라고 명시한다.
