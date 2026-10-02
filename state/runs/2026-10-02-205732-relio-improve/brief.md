- 과제: 수정 과제 — 저장소 **루트에 위임 `package.json`** 을 두어 `npm run typecheck && npm run build` 가 cwd·설치 상태와 무관하게 통과하게 하고, `make test` 에 `npm run build` 를 넣어 빌드 전용 결함이 조용히 통과하지 못하게 하기 (가치 5 / 위험 2 / 작업량 S)

- 왜: 러너의 verify 는 `.github/workflows/ci.yml:33` 에서 **명령 문자열만** 가져오고 그 step 의 `working-directory: web`(:32)과 바로 앞 `Frontend install` 의 `npm ci`(:26-27)를 떨어뜨린다. 그런데 이 저장소 루트에는 `package.json` 이 **없고**(`ls` 로 확인), 이 워크트리에는 `web/node_modules` 도 **없다**(`ls` 로 확인) — 그래서 루트에서 돌리면 npm 이 상위로 올라가 엉뚱한 `package.json` 을 집어 `Missing script` 로 exit 1 이고, cwd 를 `web` 으로 옮겨도 `tsc` 가 없어 실패한다. 루트에 위임 스크립트를 두면 저장소 어디에서 돌려도 같은 명령이 **실제로 typecheck·build 를 수행**하게 되고(느슨해지는 것이 아니라 비로소 돌게 된다), 루트에서의 모든 `npm run` 이 홈 디렉터리로 탈출하는 구멍도 함께 닫힌다.

- 수용 기준:
  1) `web/node_modules` 가 **없는** 상태에서 **저장소 루트**에서 `npm run typecheck && npm run build` 가 exit 0 이고, 출력에 `tsc`/`vite build` 가 실제로 돈 흔적(vite 의 `built in …`)이 보인다. 이 명령 그대로를 재현 로그에 남길 것.
  2) 같은 명령이 `web/` 안에서도 그대로 exit 0 (기존 CI 관용이 깨지지 않음). 그리고 루트 `package.json` 을 잠시 다른 이름으로 옮기면 루트에서 다시 exit 1 이 재현되는 것(섭동 1종)을 눈으로 확인.
  3) `make test` 가 exit 0 이고, 그 레시피가 이제 `npm run build` 를 **포함**한다. 증명: `web/src/main.tsx` 에 `import './nonexistent-probe.css'` 를 한 줄 넣으면 **고치기 전** 레시피는 exit 0(눈이 멀었다), **고친 뒤** 레시피는 exit 1(rollup `error during build:`) — 두 수치를 모두 적고 주입을 되돌릴 것.
  4) `.github/workflows/` 변경 **0줄**(`git diff --numstat -- .github/workflows/` 가 빈 출력). 워크플로를 느슨하게 하는 변경은 금지.
  5) `go build ./...` · `go vet ./...` · `go test ./...` 전부 exit 0, `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh` 통과, 빌드 뒤 `cmp internal/webui/dist/README web/public/README` 가 동일(embed 앵커 보존), `git status --short` 에 의도한 파일만.

- 건드릴 파일 (4개):
  - `package.json` — **신규, 저장소 루트**. 프런트엔드 명령의 위임 전용이며 제품 전체 빌드는 여전히 `Makefile` 이 정본임을 `description` 에 한 줄로 적을 것. 권장 모양:
    ```json
    {
      "name": "relio",
      "private": true,
      "description": "web/ 의 프런트엔드 명령을 저장소 루트에서도 그대로 쓰게 하는 위임 스크립트. 제품 빌드는 Makefile 이 정본이다.",
      "scripts": {
        "web:deps": "test -d web/node_modules || npm --prefix web ci --no-audit --no-fund",
        "typecheck": "npm run web:deps && npm --prefix web run typecheck",
        "build": "npm run web:deps && npm --prefix web run build",
        "test": "npm run web:deps && npm --prefix web test"
      }
    }
    ```
    `npm --prefix web ci` / `npm --prefix web run …` 는 **새 관용이 아니다** — `.github/workflows/release.yml:33-36` 이 이미 저장소 루트에서 그대로 쓰고 있어 `--prefix` 가 web 안에서 스크립트를 돌린다는 것이 이 저장소에서 이미 증명돼 있다. `web:deps` 를 무조건 `npm ci` 로 하지 않고 `test -d` 가드를 둔 이유는 `typecheck && build` 가 연달아 돌 때 설치가 두 번 일어나 네트워크 창이 두 배가 되기 때문이다(이 환경에 `npm ci` ETIMEDOUT 이력이 있다). 가드가 부담스러우면 무조건 `npm ci` 로 바꿔도 수용 기준은 모두 만족한다 — 다만 그 선택을 커밋 메시지에 적을 것.
    루트에 `ci` 라는 이름의 스크립트는 **만들지 말 것**: 루트에는 lockfile 이 없어 누가 `npm ci` 를 루트에서 돌리면 혼동만 커진다.
  - `web/package.json:10` — `test` 의 맨 `node` 를 `"$npm_node_execpath"` 로 바꿔 npm 자신의 node 로 고정. (근거: npm 은 조상 `node_modules/.bin` 을 PATH 앞에 세우고 이 홈에는 v20.19.2 짜리 `node` 심링크가 있어 `node --test 'test/**/*.test.ts'` 의 `**` 가 전개되지 않는다 — 지난 두 회차가 실측했고 그 커밋들은 머지되지 않아 main@494d00f 에는 **없다**. 이번 회차에는 npm 실행이 권한으로 막혀 재측정하지 못했다: **착수 전에 `cd web && npm ci && npm test` 를 먼저 돌려 실제로 실패하는지 확인하고**, 이미 통과한다면 이 파일은 건드리지 말 것.) `**` 글롭은 **그대로 둘 것** — 평면화하면 `test/sub/*.test.ts` 가 조용히 빠진다(지난 회차가 실측).
  - `Makefile:16` (`test` 타깃) — 줄 끝에 `&& npm run build` 를 추가해 `cd web && npm ci && npm run typecheck && npm test && npm run build` 가 되게 할 것. 순서를 바꾸거나 `npm ci`·`go test`·`go vet` 줄을 건드리지 말 것.
  - `README.md` — 개발 절에 "저장소 루트에서 `npm run typecheck`·`npm run build`·`npm test` 가 web/ 으로 위임된다" 는 한 문장. 소스에서 확인한 스크립트 이름만 적을 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `rm -rf web/node_modules && npm run typecheck && npm run build`  ← 루트에서. 핵심 재현.
  - `cd web && npm run typecheck && npm run build`  ← 기존 CI 관용이 그대로인지.
  - `make test`
  - `go build ./... && go vet ./... && go test ./...`
  - `./scripts/check-env-contract.sh && ./scripts/check-static-assets.sh`
  - `go test ./internal/webui/ -run TestEmbedAnchorIsRestoredByTheWebBuild -v`
  - `git diff --numstat -- .github/workflows/` (빈 출력이어야 함) · `git diff --check`

- 위험과 피할 것:
  - **`.github/workflows/` 를 한 줄도 건드리지 말 것.** 2026-09-06 에 릴리즈 경로를 깨뜨린 머지가 되돌림 PR + 자율화 강등으로 이어졌다. 이번 변경은 워크플로가 쓰는 경로(`npm --prefix web …`, `working-directory: web`)를 **그대로 두고** 루트에 선택지를 하나 더 얹는 것이다.
  - Docker 는 안전함을 확인했다: `Dockerfile:5` 가 `web/package.json`·`web/package-lock.json` 만 COPY 하고 `:7` 이 `web/` 을 COPY 하므로 루트 `package.json` 은 web-build 스테이지에 들어가지 않는다. go-build 의 `COPY . .`(`:17`)에는 들어가지만 Go 빌드와 무관하다. 그래도 `make docker` 는 이번 회차에서 돌리지 않아도 된다(장시간) — 대신 `Dockerfile` 을 수정하지 말 것.
  - `actions/setup-node` 의 `cache-dependency-path: web/package-lock.json` 이 `ci.yml:24`·`release.yml:26` 양쪽에 **명시**돼 있으므로 루트에 lockfile 없는 `package.json` 이 생겨도 캐시 키가 바뀌지 않는다(확인). 루트에 `package-lock.json` 을 만들지 말 것.
  - `npm run build` 는 `web/public/` 을 `internal/webui/dist/` 로 되돌려 쓰므로 embed 앵커(`internal/webui/dist/README`)가 `web/public/README` 와 같아야 한다 — 빌드 뒤 `cmp` 로 확인하고 체크아웃이 더러워지지 않았는지 `git status` 로 볼 것.
  - **원인을 재현하고 나서 고칠 것.** 이 저장소의 지난 두 회차는 재현 없이 적은 원인 진단이 전부 틀렸다(글롭 탓 → 실제로는 PATH 그림자, PATH 그림자 탓 → 실제로는 cwd). 이번 브리프의 (a) "루트에서 exit 1" 은 지난 회차 구현자의 실측 기록이고 이번 회차에서는 **재측정하지 못했다(미확인)** — 손대기 전에 루트에서 `npm run typecheck` 를 한 번 돌려 실제 오류 문장을 먼저 로그에 남길 것. 만약 루트에서 이미 exit 0 이면 1순위의 전제가 깨진 것이니 그 사실을 적고 수용 기준 3(`make test` + `npm run build`)만 수행할 것.
  - `tsc`·`vite` 는 shebang 으로 PATH 의 `node` 를 쓰므로 위 v20 그림자 아래에서 돈다. vite 7.3.6 은 Node ^20.19 이상을 요구하고 그림자가 20.19.2 라 **간신히** 충족한다(지난 회차에 `npm run build` exit 0 실측). 그래도 `npm run build` 가 node 버전으로 죽으면 글롭·스크립트가 아니라 **환경**이 원인이니 실제 오류를 그대로 보고할 것 — 추측으로 스크립트를 더 비틀지 말 것.
  - 건드리지 말 것: `internal/` Go 코드, `migrations/`, `internal/auth`·`oidc`, `web/vite.config.ts`, `web/tsconfig*.json`, `web/package.json` 의 `typecheck`·`build` 줄(둘 다 이미 exit 0 이라고 기록돼 있다 — 증명된 원인만 고친다).

- 추정 근거(S): 신규 파일 1개(10줄) + 기존 1줄 수정 2건 + 문서 1문장. 재현·섭동·검증이 작업량의 대부분이고 `npm ci`(69패키지)·`npm run build`(~1초) 가 직렬로 몇 차례 돈다. 예비: `npm ci` 네트워크 실패(ETIMEDOUT 이력) 시 재시도 1회, 그래도 안 되면 `web/node_modules` 가 이미 있는 상태에서 수용 기준 1을 재현하고 "설치 없는 상태 재현 미확인" 을 정직하게 적을 것. 제외 범위: 러너·홈 디렉터리 환경 정리, `release.yml` 에 프런트 테스트 추가, `esbuild` 의존성 선언.

- 차선 후보: **`Makefile:16` 에 `npm run build` 추가 + `web/package.json:10` 의 `$npm_node_execpath` 고정만** 하기(루트 `package.json` 없이). 루트에서 `npm run typecheck` 가 이미 exit 0 인 것으로 드러나면 1순위의 전제가 깨지므로 이 두 줄만으로 회차를 닫는다 — 빌드 전용 결함을 `make test` 가 보게 되는 것만으로도 지난 두 회차가 exit 0 을 증거로 내고 릴리즈에서 죽은 구멍이 닫힌다.
