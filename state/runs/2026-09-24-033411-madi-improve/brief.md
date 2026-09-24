# 수정 과제서 (2026-09-24 정찰)

- 과제: 수정 과제 — 깨끗한 체크아웃에서 `go build ./...` 를 깨뜨리는 `all:dist` 임베드 자리표시자 부재를 고치고, 같은 회차에 반복 실패하는 릴리즈 단계 두 곳(govulncheck GO-2026-6348, apk lock 드리프트)을 함께 통합 (가치 5 / 위험 2 / 작업량 M)

## 왜
정찰 worktree(HEAD = `fd2c3b5`, 이번 브랜치 `auto/2026-09-24-0334`)에서 `web/dist` 디렉터리 자체가 없고 `.gitignore:2` 가 `web/dist/` 를 통째로 무시하므로, `web/embed.go:7` 의 `//go:embed all:dist` 가 대상 디렉터리를 못 찾아 러너 검증 `go build ./...` 가 exit 1 로 끝난다(이번 회차가 verify-failed 로 끝난 바로 그 명령). 자리표시자 하나를 추적하면 웹 빌드 없이도 컴파일이 성립해 러너·기여자·`git archive` 체크아웃이 모두 같은 결과를 내고, 함께 통합하는 grpc·apk lock 수정이 release.yml 이 `compatibility.yml`/ci 와 공유하는 "Reject reachable Go vulnerabilities"·"Build service image" 두 단계의 반복 실패를 없앤다.

## 지금 코드에서 직접 확인한 사실 (추측 아님)
- `ls web/dist` → `No such file or directory`. `git ls-files web/dist` → 빈 출력.
- `.gitignore` 2행 `web/dist/`.
- `web/embed.go:7` `//go:embed all:dist`, 8행 `var Assets embed.FS`.
- `web/vite.config.ts:36` `plugins: [react(), pdfjsAssets(), offlineAssets()]` — 자리표시자 유지 플러그인 없음.
- `go.mod:87` `google.golang.org/grpc v1.82.1 // indirect`, `go.mod:83/84` `golang.org/x/sys v0.47.0`·`golang.org/x/text v0.41.0` 모두 indirect 블록.
- `deploy/runtime-apk.lock:11/12/69` `ca-certificates-bundle=20260611-r0`, `ca-certificates=20260611-r0`, `tzdata=2026c-r0`.
- `.github/workflows/release.yml:46-53` 은 `npm ci` → `licenses.mjs --check` → `npm run build` → `govulncheck` 순서다. 즉 **워크플로 자체는 웹 빌드를 먼저 하므로 임베드 때문에 깨지지 않는다.** 릴리즈에서 깨지는 곳은 53행 govulncheck 와 110행 `scripts/release-image.sh`(이미지 빌드) 다. 두 원인을 섞어 보고하지 말 것.
- 미병합 참고 커밋이 이 worktree 에 모두 존재한다(`git log --all`): `bb4e155`(임베드 자리표시자 4파일), `f942218`(취약점·lock 5파일), 그리고 직전 회차의 `1de4d00`(lock 정적 계약 시험). 셋 다 `main@fd2c3b5` 의 조상이 아니다 — 구현 전 `git merge-base --is-ancestor <sha> HEAD` 로 다시 확인할 것.

## 수용 기준
1. 웹 빌드를 한 번도 돌리지 않은 상태에서 `go build ./...` 와 `go vet ./...` 가 exit 0. (수정 전에 같은 명령이 `web/embed.go:7:12: pattern all:dist: no matching files found` 로 실패하는 것을 먼저 기록할 것.)
2. `npm ci --prefix web && npm run build --prefix web` 뒤에도 `web/dist/.gitkeep` 이 0 바이트로 남아 있고 `git status --short web/dist` 가 빈 출력이다(Vite 가 dist 를 비우고 다시 만들어도 자리표시자가 살아남는다는 증거). 실제 dist 위에서 `go build ./...` 도 exit 0.
3. `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` 가 exit 0 이고 GO-2026-6348 이 사라진다. 수정 전 같은 명령이 `Found in grpc@v1.82.1 / Fixed in v1.83.1` 과 `internal/server/jobs.go` 의 `StartJobs` 도달 경로를 내는 것을 먼저 기록할 것.
4. `docker build --target runtime-base .` 가 exit 0. 수정 전에는 `ca-certificates-…-r0 breaks: world[ca-certificates=20260611-r0]` / `tzdata-… breaks: world[tzdata=2026c-r0]` 로 실패하는 것을 먼저 기록할 것. Docker 를 못 쓰면 이 항목은 "미검증" 으로 명시하고 5)·6) 으로 대체하되, lock 완화는 금지.
5. `node scripts/licenses.mjs` 재생성 결과가 커밋된 `web/public/licenses-manifest.json`·`web/public/licenses.txt` 와 byte 동일이고 `node scripts/licenses.mjs --check` 가 `{"ok":true,...}`. 손으로 고친 고지 파일은 반려 사유다.
6. `go test -count=1 ./tests/deployment-contract` 와 `node --test tests/natural-date.mjs tests/silent-sso.mjs` 통과.
7. `.github/workflows/**`, `Dockerfile`, `internal/server/**`, `scripts/**` 는 `fd2c3b5` 대비 diff 가 없다. 검증을 느슨하게 하는 변경(govulncheck 무시 설정, lock 핀 제거, 이미지 검사 완화)은 절대 금지.

## 건드릴 파일 (이 9개 + 필요 시 docs 한 줄까지. 그 밖은 금지)
임베드 (`bb4e155` 와 같은 내용이면 cherry-pick 이 가장 안전):
- `.gitignore` — `web/dist/` 한 줄을 `!web/dist/` + `web/dist/*` + `!web/dist/.gitkeep` 로 바꾸고 이유 주석 한 줄.
- `web/dist/.gitkeep` — 0 바이트 새 파일(추적).
- `web/vite.config.ts` — 새 함수 `keepDistPlaceholder(): Plugin` 이 `generateBundle` 에서 `this.emitFile({type:"asset", fileName:".gitkeep", source:""})` 하고, 36행 `plugins:` 배열 끝에 추가. 기존 `offlineAssets()`·`pdfjsAssets()`·`build.rollupOptions.input` 은 무수정.
- `README.md` — 64행 근처 "웹 빌드를 실행합니다." 문장에, 자리표시자는 컴파일용이고 삭제 금지이며 **자리표시자만으로는 웹 UI 를 제공할 수 없다**는 구분을 덧붙인다.

취약점·lock (`f942218`):
- `go.mod` — `google.golang.org/grpc` v1.82.1 → v1.83.1, `golang.org/x/sys`·`golang.org/x/text` 는 `go mod tidy` 가 direct 블록으로 옮기는 그대로 둔다(손으로 편집 금지).
- `go.sum` — `go mod tidy` 결과만.
- `deploy/runtime-apk.lock` — `ca-certificates` **와** `ca-certificates-bundle` 을 **같은 버전으로 함께** 올리고 `tzdata` 를 현재 Alpine 3.24.1 저장소 버전으로 올린다. 한쪽만 올리면 origin 재고가 52→53 으로 어긋난다.
- `web/public/licenses-manifest.json`, `web/public/licenses.txt` — `node scripts/licenses.mjs` 재생성 결과만.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```sh
# 0) 수정 전 재현 — 출력을 그대로 기록
go build ./...                      # web/embed.go:7:12 pattern all:dist 실패 (exit 1)
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...   # GO-2026-6348 (exit 3)
docker build --target runtime-base .                    # apk breaks: world[...] (exit 1)

# 1) 수정 후
go build ./... && go vet ./...      # 웹 빌드 전에 exit 0 이어야 한다
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
node scripts/licenses.mjs && git diff --stat web/public   # 빈 출력이어야 함
node scripts/licenses.mjs --check
npm ci --prefix web && npm run build --prefix web
git status --short web/dist         # 빈 출력, .gitkeep 0바이트 유지
go build ./... && go vet ./...      # 실제 dist 위에서도 exit 0
go test -count=1 ./tests/deployment-contract
node --test tests/natural-date.mjs tests/silent-sso.mjs
docker build --target runtime-base .                     # 가능하면
git diff --check
git diff --stat fd2c3b5 -- .github Dockerfile scripts internal   # 빈 출력
```
마지막으로 최종 커밋을 `git archive HEAD | tar -x -C <새 디렉터리>` 로 풀어 **웹 빌드 없이** `go build ./... && go vet ./...` exit 0 을 확인할 것. 이것이 러너 검증과 같은 조건이다.

## 위험과 피할 것
- **인증/세션/SQL/`internal/server/backup_tables.go`/워크플로/Dockerfile 을 건드리지 말 것.** 이번 수정에 전혀 필요 없다.
- **검증을 느슨하게 해서 통과시키는 것은 금지.** govulncheck 무시 목록, lock 에서 핀 삭제, `verify-image.sh` 완화 모두 반려 사유다.
- `ca-certificates` 만 올리고 `ca-certificates-bundle` 을 두고 가는 실수가 09-19 회차에서 수 분짜리 이미지 빌드 뒤에야 "origins 53≠52" 로 드러났다. 두 값을 항상 함께 본다.
- 고지 파일(`licenses-manifest.json`/`licenses.txt`)은 반드시 스크립트 재생성으로 만든다. 손 편집은 `--check` 를 통과해도 다음 회차에 재생성 diff 로 터진다.
- `/tmp/verify-image.txt` 같은 고정 경로 출력은 이전 회차 잔재일 수 있다. 이미지 검증을 돌린다면 mtime 을 확인하고 이번 실행 출력만 근거로 쓸 것(09-22 회차가 실제로 오독할 뻔했다).
- 로컬 통과가 원격 CI/릴리즈 성공을 뜻하지 않는다. 보고에 "릴리즈 게시 성공" 이라고 쓰지 말 것.
- `1de4d00`(직전 회차의 `tests/deployment-contract/runtime_lock_test.go`)은 이 브랜치에 없다. 굳이 끌어오지 말고, 만약 끌어온다면 lock 을 올린 뒤 그 시험이 여전히 통과하는지 반드시 재실행할 것.

## 차선 후보
`.gitignore`/`web/dist/.gitkeep`/`web/vite.config.ts`/`README.md` 네 파일만 통합하는 **임베드 수정 단독**(가치 5 / 위험 1 / 작업량 S). govulncheck 나 Docker 를 이 환경에서 못 돌리면 이쪽만 완결하고, 수용 기준 1·2·6·7 과 `git archive` 재확인까지 끝낸 뒤 3·4·5 를 "미실행" 으로 명시해 보고할 것. 실패한 검증 명령(`go build ./...`)을 되돌리는 데는 이것만으로 충분하다.
