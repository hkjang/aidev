# 과제서 — 2026-10-03 (우선 과제: verify-failed 수정)

- 과제: `cd web && npm test --silent` 가 깨끗한 작업 트리에서 exit 1 로 끝나는 원인 제거 — web 테스트 게이트를 자기충족적으로 만들기 (가치 4 / 위험 2 / 작업량 S) — **수정 과제**
- 왜: 마지막 두 회차의 검증이 똑같이 `cd web && npm test --silent (exit 1)` 로 끝났는데, 직전 두 커밋(3bf8799 sqlsafe Go, 814482a 릴리즈)은 `web/` 의 **버전 문자열 6곳만** 건드렸고 그것들은 모킹과 단정이 같은 파일 안에서 짝을 이뤄 자기일관적이다(아래 "내가 확인한 것" 참고) — 즉 web 단위 테스트는 VERSION 변화에 독립적이므로 **코드 회귀로 깨진 것이 아니다**. 이 저장소의 어떤 워크플로도 `cd web && npm test` 를 쓰지 않는다(`ci.yml:46-49`·`release.yml:72-75` 는 반드시 `npm ci --prefix web` 를 먼저 돌린다). 러너의 검증 명령만 `npm ci` 없이 `npm test` 를 돌리므로, 의존성이 없는 트리에서는 `vitest` 가 없어 테스트가 한 줄도 돌지 않고 exit 1 이 된다. 이 게이트가 설치 상태에 따라 "코드와 무관하게" 빨개지는 한 회차마다 같은 실패가 반복된다.

## 먼저 할 일 (이것이 분기점이다 — 추측하지 말고 출력부터 잡아라)
```
cd web && npm test --silent ; echo "EXIT=$?"
```
출력을 **그대로 기록**하라. 두 갈래 중 하나다.

- **분기 A (내 예측, 가능성 높음)**: `sh: 1: vitest: not found` / `npm error Missing script` 류 — 테스트가 하나도 실행되지 않는다. → 아래 "분기 A 작업" 을 하라.
- **분기 B**: 실제 테스트 단정 실패(`FAIL src/...`, `AssertionError`) — 그 실패가 이번 회차의 과제다. 아래 "분기 B 작업" 을 하라. 분기 A 작업은 하지 마라.

※ 나는 이 환경에서 `npm ci` 가 샌드박스에 막혀(승인 거부) **실행하지 못했다. 그래서 분기 A·B 중 어느 것인지 미확인이다.** `web/node_modules` 는 이 워크트리와 원본 체크아웃 양쪽 모두에 없다(확인). 구현자는 설치가 가능한 환경이므로 위 명령으로 1분 안에 판정할 수 있다.

## 분기 A 작업 — `npm test` 를 의존성 유무와 무관하게 성립시킨다
- `web/package.json` 에 `"pretest"` 를 추가해, `node_modules/.bin/vitest` 가 없을 때만 `npm ci` 를 돌리고(이미 있으면 즉시 통과 = CI 에서는 비용 0), 네트워크가 없어 설치가 불가하면 **원인을 적은 메시지로 실패**하게 한다(조용한 exit 1 금지).
- 구현 형태(권고): `web/scripts/ensure-deps.mjs` 한 파일 + `package.json` 의 `"pretest": "node ./scripts/ensure-deps.mjs"`. `existsSync('node_modules/.bin/vitest')` 면 `process.exit(0)`, 아니면 `spawnSync('npm', ['ci','--no-audit','--no-fund'], {stdio:'inherit'})` 의 종료코드를 그대로 전파.
- **금지**: `vitest run` 을 `--passWithNoTests` 로 바꾸거나 테스트를 건너뛰게 하는 변경, `ci.yml`·`release.yml` 의 단계 삭제·완화, `npm ci` 를 `npm install` 로 바꿔 package-lock 을 갱신하는 것(락파일은 릴리즈 계약 `scripts/validate-release-contract.sh:25-32` 가 버전까지 검사한다).
- `node_modules` 는 `.gitignore` 대상인지 먼저 확인하고, 생성된 `node_modules`·`web/dist` 는 커밋하지 마라.

## 분기 B 작업 — 실제로 실패하는 테스트를 고친다
- 실패 파일·테스트명·단정 메시지를 그대로 과제 기록에 남기고, **테스트의 단정을 느슨하게 하는 방향이 아니라 프로덕션 코드(또는 모킹이 실제 배선과 어긋난 지점)를 고치는 방향**으로 수정하라.
- 버전 문자열 쪽이 터졌다면 원인은 하드코딩이다: `web/src/App.test.tsx:36,69,81,96` 과 `web/e2e/application.spec.ts:126,227,242` 가 `1.4.10` 을 손으로 들고 있고, 이 6곳은 `scripts/validate-release-contract.sh` 가 **검사하지 않는 유일한 버전 보유 위치**다(같은 스크립트는 `VERSION`·`web/package.json`·`package-lock.json`·`vite.config.ts`·`docs/index.html`·`docs/gallery.html`·릴리즈 마이그레이션은 검사한다). 이 경우 `src/lib/version.ts` 의 `APP_VERSION`(= `vite.config.ts:5` 의 `__APP_VERSION__`, `web/package.json` 의 version 에서 파생) 하나로 모아 드리프트가 구조적으로 불가능하게 만들어라.
- 주의: `LoginPage.tsx:58`·`AppShellLayout.tsx:303`·`WorkspacePages.tsx:520`·`AdminPages.tsx:1599` 는 `useState(APP_VERSION)` 를 **초기값**으로만 쓰고 `/version` 응답으로 덮어쓴다. `waitFor` 단정은 덮어쓴 값을 본다 — 초기값과 최종값을 혼동하지 마라.

## 수용 기준
1. 깨끗한 체크아웃(= `web/node_modules` 없음)에서 **러너와 똑같은 명령** `cd web && npm test --silent` 가 exit 0 으로 끝나고, 27개 테스트 파일/134개 테스트가 실제로 실행된 것이 출력에 보인다(0 실행으로 통과하면 실패로 간주).
2. `node_modules` 가 이미 있는 상태에서 같은 명령을 다시 돌려도 exit 0 이고, 추가 설치가 일어나지 않는다(분기 A 의 pretest 가 재설치하지 않음).
3. 회귀 증명: 수정 전에는 같은 상태에서 그 명령이 exit 1 임을 **출력으로** 보이고(red), 수정 후 exit 0(green), 수정 파일을 되돌려 다시 red 를 확인한다.
4. 어떤 테스트 단정·워크플로 단계도 느슨해지지 않았다: `.github/workflows/ci.yml`·`release.yml` 의 `npm` 단계는 한 줄도 바뀌지 않고, `scripts/ci_workflow_test.go`(모든 워크플로 `go test` 가 `-count=1` 인지 고정)가 계속 통과한다.
5. 기존 게이트 전부 통과: `cd web && npm run typecheck`, `cd web && npm run build`, `bash scripts/validate-release-contract.sh`, `go test ./... -count=1`, `go vet ./...`, `make check-go-format`.

## 건드릴 파일 (3개 이내로 유지 — 늘어나면 쪼개라)
- `web/package.json` — `scripts` 에 `pretest` 한 줄 추가. `version`·`dependencies`·`devDependencies` 는 건드리지 말 것(릴리즈 계약이 검사한다).
- `web/scripts/ensure-deps.mjs` (신규) — 의존성 유무 판정 + 필요 시 `npm ci`, 불가하면 원인 메시지로 실패.
- (분기 B 인 경우) 실패한 테스트가 가리키는 프로덕션 파일 1개 + 그 테스트 파일.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
cd web && npm test --silent ; echo EXIT=$?          # 러너와 동일한 실패 검증 — 이것이 통과해야 한다
cd web && npm ci && npm test -- --run               # ci.yml:46-49 / release.yml:72-75 와 동일한 경로
cd web && npm run typecheck && npm run build
bash scripts/validate-release-contract.sh
go test ./... -count=1 && go vet ./... && make check-go-format
```
통합 테스트(`-tags=integration`)는 이번 과제 범위가 아니다 — web 만 건드리므로 폐기 PostgreSQL 기동은 선택이다. 돌린다면 55520 이상 빈 포트를 쓰고 컨테이너는 반드시 제거하라(55432/55433/55439 점유 사례 반복).

## 위험과 피할 것
- **워크플로를 느슨하게 만들어 통과시키는 것은 금지**(이번 회차의 명시적 금지). `release.yml`·`ci.yml` 은 읽기만 하라.
- `npm ci` 를 `npm install` 로 바꾸지 마라 — `package-lock.json` 이 갱신되면 `scripts/validate-release-contract.sh:25-32` 의 버전 계약이 깨진다.
- `pretest` 가 전역 상태(전역 npm 설정, `~/.npmrc`, 상위 디렉터리)를 건드리지 않게 하라 — 운영자 반려 사유다. 쓰기는 `web/node_modules` 안으로만.
- `internal/webui/dist` 의 임베드 자산은 커밋하지 마라(`npm run build` 가 `web/dist` 를 만든다; 릴리즈 경로가 복사한다).
- `web/src/App.test.tsx` 의 버전 리터럴을 "정리" 하겠다고 분기 A 와 섞지 마라 — 두 문제는 별건이고 파일 수가 늘면 사람 손을 다시 탄다.
- 보호 경로(`internal/httpapi`·`oidcauth`·`mcpoauth`·`migrations`)는 건드리지 않는다.

## 차선 후보
- **`cteColumnList` 의 역방향 WITH 추적 루프 메모이제이션 (4/2/S)** — 전 회차가 실측한 선재 quadratic: `SELECT f0 (x) AS a0, f1 (x) AS a1, …` 모양에서 330KB 질의가 2.07s, 1MiB payload 한도로 ~19s 가 DB 권한 없이 validate 엔드포인트에 도달한다(양 방언). `internal/domain/sqlsafe/sqlsafe.go` 의 `cteColumnList` 가 매 토큰마다 WITH 목록을 역방향으로 재탐색하는데, 같은 파일의 `fromItemContext`/`postgresTokenIndex` 가 이미 쓰는 memo 방식으로 재사용하면 된다. 새 red 테스트 필요(기존 `TestAnalyzeDialectScalesLinearlyOverChainedPostgresCalls` 는 체인 호출 모양만 덮고 콤마 모양은 덮지 않는다). **판정은 1바이트도 바뀌어서는 안 되며**, 이 저장소의 확립된 증명 방식(현실 질의 70건+ × 2방언 판정을 base 와 수정본에서 덤프해 diff = 0)을 반드시 쓸 것.
- 3순위: `legacyapi` 통합 테스트 5파일의 `defer pool.Close()` → `t.Cleanup(pool.Close)` (3/2/M). `runtimeapi`·`intelligenceapi`·`platformapi` 는 미머지 a8f0775 와 겹치므로 피할 것.

## 내가 실제로 열어 본 것 (사실/미확인 구분)
- 확인: `web/package.json`(`"test": "vitest run"`, `pretest` 없음), `web/vite.config.ts`(`test.environment=jsdom`, `fileParallelism:false`, `maxWorkers:1`, `setupFiles:./src/test/setup.ts`, `include: src/**/*.{test,spec}.{ts,tsx}`), `.github/workflows/ci.yml:46-55`·`release.yml:72-81`(둘 다 `npm ci --prefix web` → `lint` → `typecheck` → `npm test --prefix web -- --run` → `build`), `scripts/release.sh`(web 을 건드리지 않음), `scripts/validate-release-contract.sh`(버전 계약 범위), `814482a` 의 `web/src/App.test.tsx` diff(1.4.9→1.4.10 네 줄, 모킹과 단정이 같은 파일 안에서 짝), `src/lib/version.ts`, `__APP_VERSION__`/`APP_VERSION` 사용처 전부, web 테스트 파일 27개 목록, `web/node_modules` 부재(이 워크트리·원본 체크아웃 모두).
- **미확인**: 실제 실패 출력. 이 세션에서 `npm ci` 가 승인 거부로 막혀 테스트를 한 번도 돌리지 못했다. 따라서 "vitest 부재가 원인" 은 **근거 있는 추정이며 증거는 아니다** — 구현자는 위 "먼저 할 일" 로 반드시 먼저 확정하라. 2026-10-01 회차에도 보고된 CI 실패가 재현되지 않아 그 단계의 실재 결함을 대신 고쳤다(판정: 차선, 회차 성공) — 같은 상황이면 같은 방식으로, 재현 가능한 실패를 잡아 고치고 못 한 것은 못 했다고 적어라.
