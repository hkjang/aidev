# 과제서 (2026-10-02, moyro) — 우선 과제: verify-failed 수리

- 과제: webapp 테스트 전체를 죽이는 jsdom 30.0.1 + undici 8.10.0 조합을 동작하는 버전 쌍으로 고정 (가치 5 / 위험 2 / 작업량 S)

- 왜: 이전 회차(run `2026-10-02-053737-moyro-improve`)는 Go 전용 변경이었는데도 `npm test` 에서 죽었다. `verify.txt` 를 읽어 보면 Go 49 패키지는 전부 ok 이고, vitest 가 **jsdom 환경을 쓰는 테스트 파일 35개의 forks worker 를 하나도 띄우지 못한다**. 전부 같은 원인이다:
  `Caused by: TypeError: webidl.util.markAsUncloneable is not a function`
  `❯ new CacheStorage node_modules/undici/lib/web/cache/cachestorage.js:20:17`
  `❯ Object.<anonymous> node_modules/undici/index.js:179:25`
  `❯ Object.<anonymous> node_modules/jsdom/lib/api.js:12:33`
  즉 `jsdom` 을 require 하는 순간 `undici` 의 모듈 최상위 `new CacheStorage()` 가 터진다. 테스트 코드와 무관한 **의존성 조합 결함**이므로, 고치지 않으면 webapp 을 건드리지 않는 모든 회차가 계속 verify-failed 로 폐기되고 릴리즈 워크플로의 `Verify web dependencies and types` 단계(`npm test`)도 계속 깨진다.

- 재현(확인됨, 로그 근거):
  - 실패 검증 명령(러너): `cd webapp && ([ -d node_modules ] || npm ci --no-audit --no-fund) && npm run typecheck --silent && npm test --silent` (exit 1) — `verify.gate.json` 에 그대로 적혀 있다.
  - 실패한 것은 `npm test` 이다. typecheck 는 통과했고, jsdom 을 쓰지 않는 13개 테스트 파일(`approval-preview.test.ts`, `stickers.test.ts`, `tsconfig.test.ts`, `time.test.ts`, `citations.test.ts`, `ssoCallback.test.ts` 등)은 전부 ✓ 로 통과한다. 죽는 것은 `.tsx` 컴포넌트 테스트와 `transport.test.ts`/`registry.test.ts`/`silentSso.test.ts` 등 **jsdom 이 필요한 35개**뿐이다.
  - 내가 직접 확인한 버전(이 워크트리, 2026-10-02):
    - `node --version` → **v22.23.1**
    - `webapp/package.json` devDependencies: `jsdom: "30.0.1"`, `vitest: "4.1.11"`, `vite: "8.2.2"`
    - `webapp/package-lock.json:2238` `jsdom@30.0.1` → `dependencies.undici: "^8.9.0"`, `engines.node: "^22.22.2 || ^24.15.0 || >=26.0.0"`
    - `webapp/package-lock.json:4289` `undici@8.10.0`(유일한 최상위 해석), `engines.node: ">=22.19.0"`
    - `package.json`/`package-lock.json` 의 `version` 은 둘 다 `0.2.42` 로 동기 상태 → `npm ci` 자체의 sync 오류는 **아니다**.
    - 즉 **engines 는 전부 충족하는데도 런타임에서 깨진다.** 따라서 "Node 를 올려라" 는 답이 아니다. `jsdom@30.0.1` 이 `^8.9.0` 범위로 끌어온 `undici@8.10.0` 에서 `webidl.util.markAsUncloneable` 이 사라졌거나(또는 정의 경로가 바뀌었거나) `cachestorage.js` 가 기대하는 모양이 아닌 것이 원인이다. **미확인:** `undici@8.10.0` 소스 안에서 `markAsUncloneable` 이 어디서 정의되는지는 node_modules 를 깔 수 없어 이번 정찰에서 열어 보지 못했다 — 구현자가 첫 단계에서 반드시 눈으로 확인할 것.
  - CI 도 같은 조건이다: `.github/workflows/release.yml:85` 와 `.github/workflows/ci.yml:88` 모두 `node-version: 22` → setup-node 가 최신 22.x(=로컬과 같은 22.23.x)를 깔므로 CI 에서도 같은 이유로 깨진다.

- 수용 기준:
  1) `cd webapp && npm ci --no-audit --no-fund && npm test` 가 **exit 0** 이고, 로그에 `Unhandled Errors` / `Failed to start forks worker` / `markAsUncloneable` 가 **한 줄도 없다**. 수정 전에 먼저 돌려 35개 worker 실패(RED)를 직접 관찰한 뒤 고칠 것.
  2) jsdom 이 필요한 테스트 파일들이 **skip 이 아니라 실제로 실행되어** 통과한다. 최소한 `src/components/App.test.tsx`, `src/api/transport.test.ts`, `src/features/workspace/composer/useDraft.test.tsx`, `src/plugins/registry.test.ts` 4개가 vitest 출력에 ✓ 와 함께 테스트 개수로 찍히는 것을 확인한다(`npm test -- --reporter=verbose` 로 파일별 확인 가능).
  3) `npm run typecheck`(= `tsc --noEmit && tsc --noEmit -p tsconfig.node.json`) 와 `npm run build` 가 여전히 통과한다.
  4) `npm audit --omit=dev --audit-level=high` 와 `npm audit --audit-level=high` 가 통과한다 — 릴리즈 워크플로의 같은 단계(`release.yml:118-119`)가 이 두 줄을 `npm test` 보다 **먼저** 돌리므로, 버전을 내려 받다가 advisory 가 걸리면 릴리즈가 다른 이유로 또 깨진다.
  5) `webapp/package.json` 과 `webapp/package-lock.json` 의 `version` 이 **둘 다 `0.2.42`** 로 유지된다(릴리즈 마커). 잠금파일은 `npm install` 로 재생성하되 버전 필드가 바뀌지 않았는지 diff 로 확인할 것.
  6) 워크플로 파일(`.github/workflows/*`)과 러너 검증 명령은 **한 글자도 바꾸지 않는다.** `npm test` 의 대상 축소(`src` → 일부), `--maxWorkers` 조정으로 숨기기, `pool: 'threads'` 로 바꿔 오류를 가리기, 실패 테스트 skip 은 전부 금지.

- 건드릴 파일 (2개, 프로덕션 코드 0개):
  - `webapp/package.json` — devDependencies 의 `jsdom` 핀을 동작하는 버전으로 내리거나, 최상위 `overrides` 에 `"undici"` 핀을 추가한다. 기존 `overrides` 블록(`{"react-is": "$react-is"}`, 46-48행)이 이미 있으니 키만 추가하면 된다.
  - `webapp/package-lock.json` — 위 변경에 맞춰 `npm install` 로 재생성(수작업 편집 금지).

- 접근(권장 순서):
  1. `cd webapp && npm ci --no-audit --no-fund` 후 **먼저 RED 를 재현**한다(`npm test`).
  2. `node -e "require('jsdom')"` 로 테스트 러너 없이 같은 TypeError 가 나는지 확인해 vitest 와 무관한 의존성 문제임을 분리한다. 그 다음 `node_modules/undici/lib/web/cache/cachestorage.js` 20행 근처와 그 파일이 `webidl` 을 가져오는 경로(`require(...)`)를 열어 `markAsUncloneable` 이 **어디서 정의되어야 하는지** 눈으로 확인한다. 이것이 어느 쪽(undici 또는 jsdom)을 핀해야 하는지 정하는 근거다.
  3. 1순위 수정: `overrides` 에 `"undici": "8.9.x"` 형태로 **동작하는 undici 를 고정**한다(jsdom 30.0.1 의 요구 범위 `^8.9.0` 안이라 jsdom 을 내릴 필요가 없고 변경이 가장 작다). `npm install` → `npm ci` → `npm test` 로 GREEN 확인.
  4. 3이 안 되면 2순위: `jsdom` 을 undici 8 을 요구하지 않는 직전 메이저/마이너(예: `29.x`)로 내린다. 이때 jsdom 의 `engines` 가 Node 22 를 지원하는지 반드시 확인한다.
  5. 어느 쪽이든 **왜 그 버전인지**를 `webapp/package.json` 의 `overrides` 옆 주석이 불가능하므로 커밋 메시지에 한 줄로 남긴다(`fix(webapp): pin undici to <ver> so jsdom 30 can construct CacheStorage`).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd webapp && npm ci --no-audit --no-fund`
  - `cd webapp && npm test`  ← **핵심 GREEN 증거. 출력 전문을 원장에 붙일 것**
  - `cd webapp && npm run typecheck`
  - `cd webapp && npm run build`
  - `cd webapp && npm audit --omit=dev --audit-level=high && npm audit --audit-level=high`  (릴리즈 워크플로와 같은 두 줄)
  - 러너 검증 재현: `cd webapp && ([ -d node_modules ] || npm ci --no-audit --no-fund) && npm run typecheck --silent && npm test --silent` 가 exit 0
  - 회귀 없음 확인(서버 무변경이므로 가볍게): `cd server && go build ./... && go vet ./...`
  - `bash scripts/check-source-sizes.sh`, `node scripts/verify-pages.mjs`

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들어 통과시키는 것은 금지**(이 회차의 명시 조건). `release.yml` / `ci.yml` 은 읽기 전용으로 취급한다.
  - `webapp/dist`, `webapp/node_modules`, `webapp/tsconfig.tsbuildinfo` 는 커밋하지 않는다(CLAUDE.md).
  - `src/tsconfig.test.ts` 라는 테스트가 존재하며 tsconfig 내용을 단언한다. `webapp/tsconfig.json` 은 이번 과제에서 **건드리지 말 것** — 2026-09-09 교훈의 `"types"` 누락은 이미 고쳐져 있다(`tsconfig.json:5` 에 `"types": ["vite/client"]` 존재). 그 교훈을 다시 적용하려 들면 엉뚱한 회귀가 난다.
  - `package.json`/`package-lock.json` 의 `0.2.42` 버전 필드를 건드리면 릴리즈 마커가 깨진다.
  - `@types/node` 는 `tsconfig.node.json`(e2e·설정 파일 전용)에서만 쓰인다. 타입 범위를 넓히지 말 것.
  - `npm audit` 가 새 advisory 로 깨지면 그것은 **별개 과제**다. 이번 PR 에 합치지 말고 원장에 적어 둘 것.
  - 서버 Go 코드는 이번 회차에서 **전혀 건드리지 않는다**. 이 과제는 순수 webapp 의존성 수리다.

- 차선 후보: `jsdom` 을 `29.x` 로 내려 undici 8 의존 자체를 끊기 (가치 5 / 위험 3 / 작업량 S) — 1순위의 `undici` override 가 jsdom 30 의 다른 내부 API 와 충돌하면 이쪽으로 간다. 그것마저 막히면, 이번 회차의 보류 아이디어 중 Go 쪽 S 과제인 **`createIncomingWebhook` 의 `channels.Get` 오류를 404 `channel_missing` 에서 500 으로 분리**(가치 3 / 위험 1 / S, `server/internal/httpapi/handlers.go:3100` 부근)로 돌아간다 — 단, webapp 이 깨진 채로는 verify 가 다시 실패하므로 **webapp 수리 없이 다른 과제를 고르지 말 것**.
