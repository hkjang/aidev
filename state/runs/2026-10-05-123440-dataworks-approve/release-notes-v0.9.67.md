## Data Works v0.9.67

### 주요 변경 사항
- **릴리즈 검증의 web 게이트가 저장소 루트에서 영구히 실패하던 문제 수정 (v0.9.67)**: `.github/workflows/ci.yml` 의 web 잡은 `defaults.run.working-directory: web` 에 의존하므로, 같은 명령을 저장소 루트에서 돌리면 루트에 `package.json` 이 없어 npm 이 저장소 밖 조상 `package.json` 을 프로젝트 루트로 잡았고(`npm prefix` 가 `/home/<user>` 를 보고) 릴리즈 검증은 `npm error Missing script: "lint"`·exit 1 로 끝났다 — 설치 누락의 exit 127 과는 다른 증상이며, 루트의 `npm run <script>` 가 저장소 밖 스크립트를 실행할 수도 있었다. 루트 `package.json` 의 lint/test/build 가 `scripts/web-run.mjs` 로 `web/` 의 같은 스크립트에 위임만 하도록 더해, 루트 경로와 `web/` 경로가 하나의 eslint·vitest·vite build 를 돌린다(`|| true`·`--max-warnings`·`--passWithNoTests` 등 완화 플래그 없음, `workspaces` 선언 없음 — hoisting 이 바뀌면 CI 의 `web/package-lock.json`·`cache-dependency-path` 가정이 깨진다). npm 이 run-script PATH 에 모든 조상 `node_modules/.bin` 을 앞세우므로 위임 스크립트 자체가 가로채인 node 로 돌 수 있어, 자식 npm 에는 `process.execPath` 대신 `npm_node_execpath` 를 넘긴다. `run-with-supported-node.mjs` 와 `engines.node` 하한도 함께 되살렸고(`resolveLocalBin` 은 fresh worktree 의 ENOENT 를 던지지 않고 부재로 보고하며, bin 이 실제로 없을 때만 선택한 인터프리터로 `npm ci` 를 한 번 돌리고 설치 실패는 자기 상태코드로 끝낸다), 인터프리터 후보를 설치된 node 들($NVM_DIR 버전 디렉터리·`/usr/local/bin/node`·`/usr/bin/node`)까지 넓히되 조상 `node_modules/.bin` 은 절대 쓰지 않으며 하한을 넘는 것 중 **가장 낮은** 버전을 고른다 — `engines.node` 는 하한 선언이고 최신을 고르면 `src/features/auth/silent-sso.test.ts` 가 Node 25.0.0·25.9.0 에서 깨진다(22.23.1·23.11.1 통과). 그 비호환은 기존 문제로 그대로 두었다. `root-npm-scripts.test.ts` 는 `npm test --silent` 가 자식에 물려주는 `npm_config_loglevel=silent` 때문에 관찰 대상 배너가 사라져 `npm test` 로는 통과하고 릴리즈 게이트의 `npm test --silent` 로는 실패했으므로, 물려받은 로그 수준을 지우고 자식 것을 직접 고정하도록 바꿨다(단정 자체는 불변). 하위 프로세스를 띄우는 두 스위트는 vitest 기본 5초 제한에 걸려 CI 에서 `Test timed out in 5000ms` 로 깨졌으므로 벽시계 상한을 `spawnSync` 쪽 `NPM_TIMEOUT_MS`(180초) 하나로 모으고 테스트 제한시간은 그보다 넉넉히 두었다. `ci.yml`·`web/package-lock.json`·`web/eslint.config.js`·`web/vite.config.ts`·`web/vitest.config.ts`·`keep-dist-placeholder.test.ts` 는 건드리지 않았고 skip·todo·continue-on-error 추가도 없다. web 테스트 파일 9 → 11, 테스트 27 → 36. 릴리즈 검증 실행: 저장소 루트에서 `npm run lint`(eslint 무경고)·`npm test --silent`(11파일 36사례 전부 통과)·`npm run build`(tsc -b && vite build, 2769 모듈) 통과, `go build ./...`·`go vet ./...`·`go test ./... -count=1` 전체 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths). 런타임 동작 변경과 스키마 변경은 없다. `dataworks:v0.9.67` 이미지를 `dataworks-v0.9.67.tar.gz` 단일 오프라인 GitHub Release asset으로 제공한다.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.67.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.67.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.67
```