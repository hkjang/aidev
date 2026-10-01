- 과제: `cd web && npm test` 가 npm 이 PATH 앞에 끼워 넣은 **상위 디렉터리의 node@20** 위에서 돌지 않게, 테스트 러너를 실행할 인터프리터를 스크립트가 직접 정하게 하기 (가치 4 / 위험 2 / 작업량 S)

- 왜: 이 회차 검증 명령 `cd web && npm test` 는 두 회차 연속 exit 1 이다. 원인은 저장소의 의존성이 아니라 **npm 의 PATH 조립 방식**이다 — `@npmcli/run-script/lib/set-path.js:22-28` 이 프로젝트 디렉터리에서 루트까지 올라가며 **모든** `node_modules/.bin` 을 실제 PATH 앞에 unshift 하고, 이 머신에는 `/home/hkjang/node_modules/node/bin/node`(node@20.19.2, `.bin/node` 심링크 포함)가 있다. `vitest` 의 shebang 은 `#!/usr/bin/env node` 이므로 스위트 전체가 node 20 에서 시작되고, node 20 에는 `worker_threads.markAsUncloneable` 이 없어 `jsdom`→`undici` import 가 파일 수만큼 터진다(2026-10-01 회차가 fork 워커 안에서 `process.execPath=/home/hkjang/node_modules/node/bin/node`, `process.version=v20.19.2` 로 직접 증명). 고치면 상위 디렉터리에 `node` 패키지를 가진 어떤 체크아웃에서도 웹 스위트가 의도한 인터프리터에서 돌고, 돌 수 없으면 120줄 벽 대신 한 줄로 실패한다.

- 수용 기준:
  1) 이 워크트리(`/home/hkjang/.cache/auto-improve-wt/Vendra`)에서 `cd web && npm ci --ignore-scripts && npm test` 가 **exit 0**, `Test Files 22 passed (22)` / `Tests 98 passed (98)`, `Unhandled Errors 0`. 상위 오염을 치우지 않은 채로 통과해야 한다(치워서 통과시키는 것은 이번 과제가 아니다).
  2) 느슨하게 만든 것이 0: `web/vitest.config.ts`·`web/src/**`·`package-lock.json`·`.github/workflows/ci.yml` **무변경**(`git diff --stat` 로 보이기), `exclude`/`skip`/`continue-on-error`/`--passWithNoTests` 추가 없음, 테스트 0건 skip, 의존성 버전 하향 없음.
  3) 적합한 인터프리터를 못 찾으면 **조용히 통과하지 않고** exit 1 로 끝나며, 실제 해석기 경로와 버전을 한 줄로 이름 붙인다(예: `web tests need Node >= 22; resolved /home/hkjang/node_modules/node/bin/node (v20.19.2)`). 이 분기를 실제로 한 번 터뜨려 보고 그 출력을 보고서에 붙일 것.
  4) `npm test -- --coverage` 처럼 추가 인자를 넘겨도 그대로 vitest 에 전달된다(한 번 실행해 확인).
  5) `npx tsc -b --noEmit`, `npx eslint src --max-warnings 0`, `npm run build` 가 전과 같이 통과.

- 건드릴 파일 (프로덕션 2개):
  - `web/scripts/run-vitest.mjs` — **신규**. (a) `process.env.npm_node_execpath`(없으면 `process.env.NODE`)로 **npm 자신을 돌리고 있는 node** 를 얻는다 — `@npmcli/config/lib/set-envs.js:105` 의 `env.NODE = env.npm_node_execpath = config.execPath` 로 npm 이 항상 설정한다(이 머신의 npm v22.23.1 에서 소스 확인). (b) 그 경로의 `-v` 를 읽어 major < 22 면 수용 기준 3 의 한 줄로 exit 1. (c) 실행할 vitest 파일은 `web/node_modules/.bin/vitest`(스크립트 파일 경로로 node 에 **인자로** 넘기면 shebang 은 주석이 되어 무시되고, 심링크 실현 경로가 `vitest.mjs` 라 ESM 으로 로드된다)를 쓰거나, 더 엄격히 하려면 `createRequire(import.meta.url).resolve('vitest/package.json')` → `bin` 필드로 해석한다(후자는 vitest 의 `exports` 가 `./package.json` 를 내보내야 하므로 먼저 확인할 것). **절대 경로를 하드코딩하지 말 것.** (d) `spawnSync(target, [vitestBin, ...process.argv.slice(2)], {stdio:'inherit'})` 후 같은 코드로 종료. 이 래퍼 자체는 node 20 이 실행해도 되는 코드여야 한다(jsdom·undici 를 건드리지 않으므로 가능).
  - `web/package.json` — `"test": "node scripts/run-vitest.mjs run"`, `"test:watch": "node scripts/run-vitest.mjs"`(래퍼가 argv 를 그대로 넘기므로 `run` 유무로 갈린다), `"engines": {"node": ">=22"}` 추가. `dependencies`/`devDependencies` 는 한 글자도 건드리지 말 것.
  - 주석은 이 저장소 문체로: 「왜 `node` 를 믿으면 안 되는가 / 안 고치면 무엇이 잘못 전달되나」를 서술(integrations.go 의 인자 헬퍼 주석이 본보기).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm run env | grep -E '^(npm_node_execpath|NODE)='` — **맨 먼저**. 이 두 변수가 nvm 의 v22 를 가리키는지 확인하고, 안 가리키면 설계 전제가 깨진 것이니 차선 후보로 갈 것.
  - `cd web && npm ci --ignore-scripts` → `npm test`(고치기 **전에** 한 번 돌려 `webidl.util.markAsUncloneable is not a function` 벽과 exit 1 을 실제 출력으로 남길 것) → 고친 뒤 다시 `npm test`.
  - `cd web && npx tsc -b --noEmit`, `npx eslint src --max-warnings 0`, `npm run build`
  - Go 회귀(코드 무변경 확인용): `gofmt -l internal cmd`, `go vet ./internal/... ./cmd/...`, `go test ./internal/... ./cmd/... -count=1`(DSN 미설정이면 DB 통합은 skip — 그렇게 보고할 것)
  - `gate.py secrets` 를 diff 에 먼저.

- 실행 순서 (각 단계 끝에서 트리가 동작하는 상태여야 한다):
  1) `npm ci --ignore-scripts` → `npm test` 로 **고치기 전 실패를 먼저 출력으로 남긴다**(수용 기준 1의 대조군). 프로덕션 변경 0.
  2) `npm run env` 로 `npm_node_execpath`/`NODE` 확인 — 전제 검증 체크포인트. 비어 있으면 차선 후보로.
  3) `web/scripts/run-vitest.mjs` 추가(아직 `package.json` 은 그대로 → 트리 동작 유지). `node web/scripts/run-vitest.mjs run` 을 수동으로 한 번 돌려 동작 확인.
  4) `web/package.json` 의 `test`/`test:watch`/`engines` 교체 → `npm test` exit 0 확인.
  5) 가드 분기 확인(수용 기준 3) → tsc·eslint·build → Go 회귀 → `gate.py secrets`.

- 범위 밖(이번에 하지 말 것): lockfile·의존성, `.github/workflows/**`, `.nvmrc`, `web/src/**`, `vitest.config.ts`, 빌드·lint 스크립트 래핑, 머신 환경 정리.

- 위험과 피할 것:
  - **의존성·lockfile 을 건드리지 말 것.** 2026-10-01 회차가 「undici 가 lockfile 에서 깨졌다」는 전제로 과제서를 썼다가 기각됐다 — 상위 오염이 없는 체크아웃에서는 **같은 lockfile 로 22/22·98/98 이 통과한다**. jsdom 30 은 스스로 `engines: ^22.22.2 ...` 를 선언하므로 node 20 을 지원하게 끌어내리는 모든 변경은 「통과시키려 느슨하게 만드는 것」이다.
  - `.github/workflows/ci.yml` 은 보호 경로다. 이번에 손대지 말 것(CI 의 node 는 이미 깨끗한 22 이므로 래퍼는 그대로 통과한다). `.nvmrc` 로 patch 고정하는 건 별개 회차.
  - 머신의 `/home/hkjang/node_modules/node` 를 **지우지 말 것** — 저장소 밖 환경을 되돌릴 수 없게 바꾸는 것은 그 자체로 반려 사유이고, 지우면 이번 수정이 실제로 효과가 있는지 증명할 수 없다.
  - 호스트 `node` 가 아니라 **워커 안의 인터프리터**를 증거로 쓸 것. `process.version` 을 fork 워커 안에서 읽는 임시 프로브는 2026-10-01 회차가 쓴 방식이고, 쓰고 나면 지울 것(커밋에 남기지 말 것).
  - `npm run build`·`eslint` 도 같은 PATH 위에서 돈다(지금은 node 20 에서도 우연히 통과). 이번 범위는 **테스트 러너 한 자리**다. 빌드까지 같은 래퍼로 감싸려 들면 파일 수와 위험이 늘어난다 — 하지 말 것.
  - 2026-10-01 의 `fe87155`(engines + `pretest` 가드)는 **이 베이스(main@887bd84)에 없다**. 그 접근은 실패를 한 줄로 바꿨을 뿐 `npm test` 는 여전히 exit 1 이라 같은 verify-failed 로 끝났다 — 진단 문구는 가져오되 **거기서 멈추지 말 것**. 이번 수용 기준 1은 exit 0 이다.

- 차선 후보: `npm_node_execpath`/`NODE` 가 실제 실행에서 비어 있거나 node 20 을 가리키면(= 러너가 node@20 의 npm 으로 `npm test` 를 부른다면), 래퍼가 **PATH 에서 npm 이 끼워 넣은 `node_modules/.bin` 항목들을 제거한 뒤** 남은 PATH 에서 `node` 를 찾아 major >= 22 인 첫 번째를 쓰는 방식으로 바꾼다(실패 시 동일하게 한 줄 exit 1). 그것마저 성립하지 않으면 과제를 바꿔 「compare_suppliers 설명이 약속하는 계약·이슈 비교가 응답에 없음 — 설명을 실제 응답으로 좁히기 (3/1/S, `internal/httpapi/integrations.go:415` 한 줄 + DB 불필요 가드)」를 하고, 웹 실패는 **저장소 밖 환경 문제**로 보고서에 명시할 것.

- 견적 근거(한 세션 45분 안에 끝나는가):
  - 분해: ① 고치기 전 실패 재현 + `npm ci` 10분(`npm ci` 자체가 1~2분, 네트워크에 좌우) ② `npm run env` 전제 확인 2분 ③ 래퍼 작성·수동 실행 10분 ④ package.json 교체 + `npm test` 5분 ⑤ 가드 분기·인자 전달·tsc/eslint/build 10분 ⑥ Go 회귀 + gate 8분. 합 **45분**, 신뢰 범위 **35~70분**(열에 여덟은 이 안).
  - 범위를 넘기는 경우는 하나뿐이다: `npm_node_execpath` 가 실제 실행에서 v22 를 가리키지 않는 경우(차선 경로로 +20분). 그래서 2단계를 체크포인트로 앞에 두었다.
  - 이 견적이 가장 크게 기대는 가정: **구현자 세션에서 `npm` 실행 권한이 있다**(정찰 세션에서는 없었다). 없으면 이번 과제는 성립하지 않으니 차선 후보(compare_suppliers 설명 좁히기, Go 전용)로 갈 것.
