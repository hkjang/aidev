# 과제서 (수정 과제 — 지난 회차 verify-failed 대응)

- 과제: 웹 테스트 스위트가 jsdom 을 import 하지 못해 22개 테스트 파일이 **하나도 시작하지 못하는** 것 고치기 — `undici` 가 lockfile 에서 깨진 조합으로 고정된 것 (가치 5 / 위험 2 / 작업량 S)

- 왜: 지난 회차(2026-10-01-141216-Vendra-improve)의 verify 가 `cd web && npm test --silent` 에서 exit 1 로 끝났고, 그 로그는 테스트 실패가 아니라 **스위트가 뜨지 못한 것**이다 — Vitest v4.1.11 이 1초 만에 `Vitest caught 22 unhandled errors` 로 끝나며 22개 전부 `[vitest-pool]: Failed to start forks worker for test files <파일>` / `Caused by: TypeError: webidl.util.markAsUncloneable is not a function` 이고, 스택은 `node_modules/undici/lib/web/cache/cachestorage.js:20:17` ← `node_modules/undici/index.js:179:25` ← `node_modules/jsdom/lib/api.js:12:33` 이다. 즉 `jsdom` 을 require 하는 순간 `undici` 가 터지므로 jsdom 환경 테스트가 **한 건도 실행되지 않는다**. 고치면 18~22개 프런트엔드 테스트 파일이 다시 실제로 돌고, 같은 단계를 가진 CI 의 `web` 잡(`.github/workflows/ci.yml:72`)도 통과한다 — 지금은 거기서도 같은 이유로 막힌다.

## 확인된 사실 (이 회차에 실제로 읽은 것)

- `/mnt/c/.../2026-10-01-141216-Vendra-improve/verify.json` — 앞 네 단계(`go build ./...`, `go vet ./...`, `go test ./...`, `cd web && [ -d node_modules ] || npm ci --no-audit --no-fund`)는 전부 exit 0 이고 마지막 `cd web && npm test --silent` 만 exit 1, **1초**. Go 쪽은 무결하다.
- 같은 런의 `verify.txt:9` — `added 255 packages in 5s`. 즉 node_modules 는 lockfile 에서 **갓 설치된 것**이며, 낡은 설치가 남아 있던 문제가 아니다.
- `verify.txt:11-120` — `RUN v4.1.11`, 그 다음 바로 Unhandled Errors 블록. 테스트 통과/실패 줄이 하나도 없다. 로그에 이름이 보이는 파일: `src/pages/admin-workflow.test.tsx`, `src/pages/admin-tracking.test.tsx`, `src/api.test.ts`, `src/pages/supplier-invitation.test.tsx`, `src/pages/admin-mcp-oauth.test.tsx`, `src/silent-sso-boot.test.tsx` … (총 22개).
- `web/package-lock.json:3641-3650` — `node_modules/undici` = **8.10.0**, `dev: true`, `engines.node >= 22.19.0`.
- `web/package-lock.json:2515-2545` — `jsdom` = 30.0.1, 의존성에 `"undici": "^8.9.0"`, `engines.node: "^22.22.2 || ^24.15.0 || >=26.0.0"`. `undici` 는 lockfile 전체에 **한 항목뿐**(중복 설치 아님).
- `web/package.json` — `jsdom: ^30.0.1`, `vitest: ^4.1.11`, `undici` 직접 의존 없음, **`overrides` 블록 없음**.
- 로컬 node = **v22.23.1** (두 engines 범위 모두 만족). 즉 Node 를 올려서 해결되는 문제로 단정할 수 없다.
- `web/vitest.config.ts` — `environment: "jsdom"`, `globals: true`, `setupFiles: ["./src/test/setup.ts"]`, `pool` 미지정(Vitest 4 기본 `forks`). 그래서 실패가 fork 워커 시작 시점에 난다.
- `web/src/test/setup.ts` — 7줄(`@testing-library/jest-dom/vitest` import + `afterEach(cleanup)`)뿐. 여기에는 원인이 없다.
- `git log -- web` 최신 커밋은 `35fd306`(MCP OAuth 화면) — 수 주 전이다. **이 저장소의 소스 변경이 원인이 아니고 의존성 그래프가 원인이다.**

## 수용 기준

1. `cd web && npm ci --ignore-scripts && npm test` 가 exit 0 으로 끝나고, 출력에 `Unhandled Errors` 블록도 `Failed to start forks worker` 도 **없으며**, 22개 테스트 파일이 전부 실제로 실행된 집계(`Test Files  22 passed`)가 보인다. 이 출력 전문을 회차 노트에 붙일 것.
2. 고치기 **전에** 같은 명령으로 `webidl.util.markAsUncloneable is not a function` 을 **로컬에서 직접 재현**하고 그 출력을 기록할 것 (지난 런 로그를 베끼지 말 것 — 이 저장소의 관례는 실패를 먼저 보는 것이다).
3. 수정은 의존성 해석 한 곳에 머문다: `web/package.json` 의 `overrides`(또는 jsdom 버전) 한 자리 + `web/package-lock.json` 재생성. **프로덕션 소스(`web/src/**`)와 `web/vitest.config.ts` 는 건드리지 않는다** — 테스트를 끄거나 `environment` 를 바꾸거나 `pool` 을 바꿔 우회하는 것은 이 과제의 답이 아니다.
4. `npx tsc -b --noEmit`, `npx eslint src --max-warnings 0`, `npm run build` 가 전부 통과한다(CI `web` 잡이 `npm test` 앞뒤로 돌리는 것들이며, 의존성 변경이 이것들을 깨지 않았음을 보여야 한다).
5. Go 쪽 회귀 없음: `go vet ./internal/... ./cmd/...`, `gofmt -l internal cmd`(무출력), `go test ./internal/... ./cmd/... -count=1`.
6. `web/package.json` 에 바꾼 자리 옆(또는 그 아래)에 **왜 이 핀이 필요한가**를 이 저장소 문체(「왜 이렇게 되어 있나 / 고치지 않으면 무엇이 잘못 전달되나」)로 남긴다 — JSON 에는 주석을 못 쓰므로 `docs/` 또는 커밋 메시지 본문에 적을 것. 핀을 언제 풀 수 있는지(어떤 버전이 고쳐지면)도 적을 것.

## 건드릴 파일 (프로덕션 2개)

- `web/package.json` — `devDependencies` 위에 `overrides` 를 추가해 `undici` 를 동작하는 버전으로 고정(예: `"overrides": { "undici": "<동작 확인된 버전>" }`). **미확인**: 어느 버전이 동작하는지는 이 세션에서 npm 실행 권한이 없어 확정하지 못했다 — 구현자가 아래 「진단 절차」로 직접 정할 것. 대안은 `jsdom` 을 깨진 `undici` 를 끌어오지 않는 버전으로 올리는 것이지만, jsdom 의 범위가 `^8.9.0` 이라 그것만으로는 8.10.0 이 다시 떠오를 수 있으니 `overrides` 가 더 확실하다.
- `web/package-lock.json` — `npm install`(또는 `npm ci` 뒤 재생성)로 다시 만들어 커밋. **손으로 편집하지 말 것.**
- (필요하면) `docs/operations/` 아래 한 줄 — 핀 이유와 해제 조건.

## 진단 절차 (구현자가 먼저 할 것)

1. `cd web && npm ci --ignore-scripts && npm test` → 실패 재현, 출력 저장.
2. 실제 설치된 undici 의 모양을 보기: `node -p "require('./node_modules/undici/package.json').version"`, `grep -rn "markAsUncloneable" node_modules/undici/lib/ | head`, `sed -n '1,30p' node_modules/undici/lib/web/cache/cachestorage.js`. `webidl.util` 가 어디서 만들어지고 왜 그 키가 없는지(혹은 `node:util` 의 어느 API 에 기대는지)를 확인할 것 — 추측하지 말고 설치된 코드를 읽을 것.
3. `node -p "typeof require('node:util').markAsUncloneable"` 로 Node 쪽 API 유무를 확인(로컬 node v22.23.1). 이것이 `undefined` 면 원인은 Node 버전이고, 그때는 `overrides` 가 아니라 **CI 와 로컬이 같은 Node major 를 쓰는지**(`.github/workflows/ci.yml:66` 의 `node-version: 22`) 를 함께 봐야 한다. 그 경우에도 워크플로를 느슨하게 만들지 말고 `node-version` 을 정확히 고정하는 쪽으로 고칠 것.
4. 동작하는 undici 버전을 `npm i -D undici@<ver>` 로 임시 시험하지 말고, `overrides` 를 넣은 뒤 `rm -rf node_modules && npm install` → `npm test` 로 **clean install 에서** 통과하는 것을 확인할 것. lockfile 만 고쳐 놓고 캐시된 node_modules 로 통과를 주장하면 CI 에서 다시 깨진다.
5. 통과한 뒤 `git diff --stat` 이 `web/package.json` + `web/package-lock.json`(+ 문서) 만인지 확인.

## 검증 명령 (이 저장소에서 실제로 도는 것)

```
cd web && rm -rf node_modules && npm ci --ignore-scripts
cd web && npm test                 # ← 이번 회차의 본 검증. Unhandled Errors 0, Test Files 22 passed
cd web && npx tsc -b --noEmit
cd web && npx eslint src --max-warnings 0
cd web && npm run build
gofmt -l internal cmd              # 무출력
go vet ./internal/... ./cmd/...
go test ./internal/... ./cmd/... -count=1
```
DB 통합 테스트까지 돌리려면 전용 `postgres:16-alpine` 과 세 DSN(`VENDRA_TEST_DSN`, `VENDRA_TEST_MIGRATE_DSN`, `VENDRA_TEST_UPGRADE_DSN`, 뒤 둘은 빈 DB)이 필요하다. 이번 과제는 웹 의존성만 건드리므로 Go 통합 회귀는 필수가 아니지만, DSN 을 unset 으로 두고 통과한 것을 「DB 검증」으로 주장하지 말 것.

## 위험과 피할 것

- **워크플로를 느슨하게 만들어 통과시키는 것은 금지다.** `.github/workflows/ci.yml:72` 의 `npm test` 를 지우거나 `|| true` 를 붙이거나 `continue-on-error` 를 넣지 말 것. `vitest.config.ts` 에 `exclude` 를 넣어 깨진 파일을 빼는 것도 같은 금지에 걸린다 — 지금 깨진 것은 **전부**다.
- `web/src/**` 의 테스트나 컴포넌트를 고쳐서 해결하려 들지 말 것. 실패는 테스트 코드에 닿기 전에, jsdom import 시점에 난다.
- `package-lock.json` 을 손으로 편집하지 말 것(integrity 해시가 틀어지면 `npm ci` 가 다른 방식으로 깨지고, 그 깨짐은 CI 에서만 보인다).
- 의존성 **메이저 업그레이드 금지**가 이 회차의 규칙이다. `vitest` 4 → 5, `jsdom` 30 → 31 같은 점프는 하지 말고, 깨진 전이 의존성 하나를 고정하는 **최소 변경**으로 끝낼 것. `react`·`vite`·`typescript` 는 손대지 말 것.
- 교훈(2026-09-10 사람 반려): 반려된 접근을 되풀이하지 말 것. 또 「릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것」 — `npm run build` 를 반드시 돌릴 것(`make build` 도 웹 빌드를 거친다).
- `gate.py secrets` 를 diff 에 먼저 돌릴 것. lockfile 에는 긴 integrity 문자열이 많아 비밀정보 게이트가 오탐을 낼 수 있으니 실패하면 그 줄을 확인해 보고하되, 게이트를 끄지 말 것.
- Vitest 는 `/mnt/c` 에서 돌지 않는다. 반드시 Linux 파일시스템의 워크트리(`/home/hkjang/.cache/auto-improve-wt/Vendra/web`)에서 돌릴 것.
- `npm test --silent` 의 `--silent` 는 npm 의 loglevel 이고 vitest 에는 전달되지 않는다 — 이것은 원인이 아니다. 혼동해서 스크립트를 바꾸지 말 것.

## 차선 후보

`compare_suppliers` 설명이 약속하는 「계약 및 이슈」 비교를 실제 응답으로 좁히기 (가치 3 / 위험 1 / 작업량 S) — `internal/httpapi/integrations.go:415` 의 설명은 「비용, 평가, 위험, 계약 및 이슈를 비교합니다」인데 응답은 `supplierSummaryRows`(id/number/name/status/grade/riskLevel/score/annualSpend)뿐이라 계약·이슈 필드가 없다. 모델은 설명을 믿고 이 도구로 「이슈 비교」를 물은 뒤 없는 것을 「이슈 없음」으로 사람에게 옮긴다. 설명 한 줄을 실제 응답으로 좁히고, 설명이 약속하는 필드가 `supplierSummaryRows` 에 실제로 있는지 대조하는 DB 불필요 가드를 더한다(`TestMCPNumericSchemaMaximumsMatchEnforcedCeilings` 가 본보기). **단, 이번 회차는 우선 과제가 배정되어 있으므로 1순위가 성립하지 않을 때만 고를 것** — 웹 테스트가 깨진 동안은 어떤 과제도 검증을 통과할 수 없다.
