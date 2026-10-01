# 과제서 (2026-10-01 정찰) — 오류 대응 회차

- 과제: 릴리즈 검증 `npm run lint`(web, `eslint .`, eslint 10.9.1) 실패 복구 — **원인을 실행으로 먼저 확정한 뒤** 고치기 (가치 4 / 위험 2 / 작업량 S)

- 왜: CI `web` 잡의 Lint 단계(`.github/workflows/ci.yml:60-61`, `defaults.run.working-directory: web`)가 exit 1 이라 릴리즈 워크플로가 같은 이유로 두 번 깨졌고, 그 뒤 단계(`npm test`·`npm run build`)는 실행조차 되지 않는다. lint 가 녹색이어야 지난 회차의 web 수정(`npm test` 복구)도 다시 머지될 수 있다.

## 0) 먼저 재현 — 이 단계를 건너뛰지 말 것

이번 정찰은 **lint 를 실제로 돌리지 못했다**(이 샌드박스에서 `npm ci` 가 허용되지 않았고, 워크트리·메인 체크아웃 어디에도 `web/node_modules` 가 없다 — Glob 으로 확인). 아래 원인 가설은 전부 **미확인**이다. 지난 회차(2026-10-01)에서 과제서의 1순위 가설이 실행으로 반증된 전례가 있으니, 먼저 출력을 받고 그 출력이 가리키는 파일을 고쳐라.

```
cd web && npm ci && npm run lint; echo "LINT_EXIT=$?"
```

- 출력의 **파일 경로 + 규칙 id** 를 그대로 원장에 남길 것. (`eslint . -f stylish` 기본)
- `npm ci` 없이 돌리면 `eslint: not found` / **exit 127** 이다. 127 과 1 을 섞지 말 것(프로필 검증 함정).

### 어느 트리에서 재현할 것인가 (확인함)
- 이 워크트리 HEAD = `98c5081`(v0.9.66, main), 작업 트리 clean.
- 지난 회차 커밋 `1d63b7c` "fix(web): run vitest with npm own node so jsdom tests start" 는 **이 브랜치에 없고** `auto/2026-10-01-1902` 에만 있다. 바꾼 파일: `docs/OPERATIONS.md`, `web/package.json`(+`engines.node: ">=22.19.0"`, `test`/`test:watch` 를 런처 경유로), **신규** `web/scripts/run-with-supported-node.mjs`(94줄), **신규** `web/src/test/test-script-interpreter.test.ts`(69줄).
- 따라서 **두 트리 모두에서 lint 를 돌려라**:
  1. clean main(`98c5081`) 에서 `npm run lint` — 여기서 이미 exit 1 이면 원인은 기존 소스이고 `1d63b7c` 와 무관하다.
  2. `git cherry-pick 1d63b7c` (또는 `git checkout auto/2026-10-01-1902 -- web docs`) 한 상태에서 `npm run lint` — 여기서만 깨지면 범인은 위 신규 2파일이다.
  이 두 결과의 차이가 과제 범위를 결정한다.

## 원인 가설 (전부 미확인 — 출력이 이기면 출력을 따를 것)

- **H1 (가장 유력): 새 `web/scripts/run-with-supported-node.mjs` 가 `eslint.config.js` 의 어떤 `files` 와도 안 맞는 `.mjs` 다.**
  `web/eslint.config.js` 는 `{ ignores: ['dist','coverage'] }` 와 `files: ['**/*.{ts,tsx}']` 블록 둘뿐이다. ESLint 10 은 기본 린트 대상에 `**/*.mjs` 를 포함하므로 이 파일은 "설정 없는 린트 대상"이 된다(ESLint 10 이 이를 오류로 보고하는지는 **미확인** — 출력으로 확인할 것).
  고치는 방향: `eslint.config.js` 에 `.mjs`(또는 `scripts/**`)용 **설정 객체를 추가**해 `js.configs.recommended` + Node 전역을 적용하고, 그 규칙에 맞게 스크립트를 고친다. **`ignores` 에 넣어 린트 대상에서 빼는 것은 금지**(그게 바로 "워크플로를 느슨하게 만들기"다).
- **H2: 새 `web/src/test/test-script-interpreter.test.ts` 가 tseslint 규칙에 걸린다.** 이 파일은 `**/*.{ts,tsx}` 블록이 잡으므로 `js.configs.recommended` + `tseslint.configs.recommended` 가 적용된다. 템플릿 리터럴 안의 `JSON.stringify`, `as { scripts: Record<string,string> }` 캐스트, `120_000` 등을 썼다. 걸리면 **규칙을 끄지 말고 코드를 고칠 것**.
- **H3: clean main 에서도 깨진다** — 그러면 eslint 10.9.1 / typescript-eslint 8.68.0 / typescript ^6.0.3 조합이 기존 소스에서 새 오류를 내는 것이다(lockfile 은 pin 되어 있으므로 버전 드리프트는 아니다 — 아래 "확인한 것" 참고). 출력이 가리키는 소스 파일을 고친다.
- **H4: lint 를 `web/` 가 아닌 저장소 루트에서 돌렸다.** 루트에 `package.json` 이 없음을 확인했으므로 루트에서 `npm run lint` 는 실패한다. 재현에서 `cd web` 을 한 뒤에도 깨지는지 반드시 확인하고, 만약 루트 실행이 원인이면 **저장소에 가짜 루트 `package.json` 을 만들지 말고** 원장에 그 사실을 적고 차선 후보로 넘어갈 것.

## 수용 기준
1. `cd web && npm ci && npm run lint` 가 **exit 0**, 경고 포함 출력이 수정 전후로 어떻게 달라졌는지 원장에 기록.
2. 수정 전 실패 출력(파일·규칙 id·줄)과 수정 후 통과를 **같은 명령으로** 재현해 원장에 남김. 프로덕션 파일만 되돌리면 같은 실패가 다시 나는 것까지 확인.
3. lint 를 통과시키려고 **느슨하게 만든 것이 하나도 없음**: `.github/workflows/ci.yml` 무변경, `eslint.config.js` 의 `ignores` 에 소스 경로 추가 없음, 새 `eslint-disable`/`eslint-disable-next-line` 없음, `--max-warnings`/`|| true`/`--no-error-on-unmatched-pattern` 같은 플래그 추가 없음, 기존 규칙 off 로 바꾸지 않음(`no-explicit-any: off` 는 기존 그대로 유지).
4. 뒤 단계도 함께 녹색: `cd web && npm test`(10파일 전후, 지난 회차 기준 9+1), `npm run build`, 그리고 빌드 뒤 `git status --short` 에 `web/dist` 항목 없음(`.gitkeep` 유지).
5. `1d63b7c` 의 `npm test` 복구 내용(런처 + `engines.node`)이 **살아 있음** — 되돌려서 lint 를 통과시키지 말 것. `npm test` 가 다시 exit 1 이 되면 실패다.

## 건드릴 파일 (최대 3개, 프로덕션)
- `web/eslint.config.js` — H1 이면 `.mjs`/`scripts` 용 설정 객체 **추가**(ignores 아님).
- `web/scripts/run-with-supported-node.mjs` — 새로 적용된 규칙에 맞게 코드 수정.
- `web/src/test/test-script-interpreter.test.ts` 또는 출력이 가리키는 소스 1파일 — H2/H3.
- 문서는 필요하면 `docs/OPERATIONS.md` 에 몇 줄(정본 PDF 없음 — 재생성 불필요).
- **예외 조항**: 재현 결과가 위와 다른 파일을 가리키면 그 파일을 고쳐라. 단 총 프로덕션 파일은 3개 안쪽으로 유지하고, 넘치면 lint 를 녹색으로 만드는 최소 조각만 이번 회차에 담아라.

## 검증 명령 (이 저장소에서 실제로 도는 것)
```
cd web && npm ci && npm run lint          # 1차 게이트 (exit 0 이어야 함)
cd web && npm test                        # jsdom 8파일 + node 2파일, 지난 회차 10파일/28테스트
cd web && npm run build && git status --short   # tsc -b && vite build, dist 흔적 없어야 함
go build ./... && go vet ./... && go test ./... -count=1   # proxy 약 40초, store 약 15초
go run ./cmd/api-surface-audit            # 누락 0 (550 routes / 612 OpenAPI paths)
git diff --check
```
Go 파일을 안 건드리면 Go 쪽은 `go build ./...` + `go vet ./...` 로 줄여도 되지만, 릴리즈 경로를 건드리는 회차이므로 전체를 한 번 돌리는 편이 안전하다.

## 위험과 피할 것
- **워크플로·설정을 느슨하게 만드는 모든 수단 금지**(수용 기준 3). 과거 운영자 지시: "릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것."
- `web/vite.config.ts` 의 `keepDistPlaceholder`·`emptyOutDir`, `web/embed.go` 의 `//go:embed all:dist`, `web/package-lock.json`(`npm install` 로 재생성 금지), `vitest.config.ts` 의 `include` 는 건드리지 말 것.
- `web/src/test/keep-dist-placeholder.test.ts` 의 단정을 약화하지 말 것. (지난 회차에 이 파일은 범인이 아니었다.)
- `gofmt -l internal/proxy/dataworks_runtime.go` 는 기존 CRLF 때문에 HEAD 에서도 파일명을 출력한다 — 줄 끝을 건드리지 말 것.
- grep 결과를 증거로 쓰지 말 것. lint 출력과 exit code 만 증거다.
- 보호 경로(auth/keycloak/mcp_oauth, store 마이그레이션)는 이번 과제와 무관하므로 열지 말 것.

## 차선 후보
1. **`npm run build`·`npm run lint` 도 `scripts/run-with-supported-node.mjs` 를 거치게 하기 (2/2/S)** — 재현 결과 lint 실패가 **구버전 Node 로 eslint 가 기동된 것**(PATH 앞의 `/home/hkjang/node_modules/.bin/node` → Node 20.19.2)이면 이게 1순위가 된다. 지난 회차는 "lint 는 Node 20 에서도 통과" 를 근거로 일부러 제외했으니, 그 전제가 깨졌는지 실행으로 확인할 것. 적용 시 `npm run build` 뒤 `git status --short` 깨끗함과 `go build ./...` 까지 확인.
2. **`keep-dist-placeholder.test.ts` 의 `logLevel:'silent'` 가 vite 오류를 삼키는 문제 (3/1/S)** — lint 가 실제로는 깨지지 않는 것으로 판명되면(H4) 이쪽으로. 실패했을 때만 로그 수준을 올려 재시도하는 형태로 고치고 단정은 약화하지 말 것.

## 이번 정찰이 실제로 확인한 것 / 확인 못 한 것
- 확인: 워크트리 HEAD·clean 상태, `.github/workflows/` 에 `ci.yml` 하나뿐(release.yml 없음), ci.yml 의 web 잡 4단계와 Node 24, `web/package.json` 의 `lint: "eslint ."`, `web/eslint.config.js` 전문, `web/package-lock.json` 의 eslint `10.9.1`·typescript-eslint `8.68.0`, 저장소 루트에 `package.json` 없음, `1d63b7c` 의 변경 파일 목록과 `web/src/test/test-script-interpreter.test.ts` 전문, `web/vitest.config.ts`·`web/tsconfig.app.json`, 어디에도 `web/node_modules` 없음.
- 확인 못 함(미확인): **lint 를 한 번도 돌리지 못했다**(`npm ci` 불허). 실패 메시지·규칙 id·실패 파일 전부 미확인. `web/scripts/run-with-supported-node.mjs` 본문도 읽지 못했다(권한). ESLint 10 이 설정 없는 `.mjs` 를 어떻게 처리하는지 미확인.
