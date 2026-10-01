# 회차 노트 2026-10-01-210621-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:06] base pinned — main@98c5081
- [러너 21:06] autonomy release — 

## 정찰 노트
- 오류 대응 회차라 후보 선정 여지가 없었다. 과제는 CI `web` 잡의 `npm run lint`(= `eslint .`, eslint 10.9.1) exit 1 복구 하나이고, 다른 보류 아이디어는 전부 그대로 뒀다.
- **이번 정찰은 lint 를 한 번도 돌리지 못했다** — 이 샌드박스에서 `npm ci` 가 허용되지 않았고 워크트리·메인 어디에도 `web/node_modules` 가 없다. 과제서의 원인 가설(H1 새 `.mjs` 가 flat config 의 어떤 `files` 와도 안 맞음 / H2 새 `test-script-interpreter.test.ts` 의 tseslint 위반 / H3 clean main 자체 실패 / H4 루트에서 실행)은 전부 **추측**이다.
- 구현자는 과제서 0절을 반드시 먼저 하라: clean main(`98c5081`)과 `1d63b7c` cherry-pick 한 트리 **양쪽**에서 `cd web && npm ci && npm run lint` 를 돌려 어느 쪽이 깨지는지부터 가른다. 지난 회차처럼 과제서 1순위 가설이 반증될 수 있다.
- 조심할 것: `eslint.config.js` 의 `ignores` 에 소스 경로를 더하거나 `eslint-disable`·`--max-warnings`·`|| true` 로 통과시키는 것은 금지(수용 기준 3). `1d63b7c` 의 `npm test` 복구(런처 + `engines.node`)를 되돌려 lint 를 통과시키는 것도 실패로 친다.
- 프로필은 0일 전 것이고 이번에 읽은 것(ci.yml 4단계·eslint 설정·lockfile 버전·HEAD)과 어긋나는 데가 없어 새로 쓰지 않았다.
- [러너 21:11] scout done — 릴리즈 검증 `npm run lint`(web, `eslint .`, eslint 10.9.1) 실패 복구 — **원인을 실행으로 먼저 확정한 뒤** 고치기 

## 구현 노트
- 과제서 0절대로 먼저 재현했고 H1~H3 은 반증됐다: `cd web && npm ci && npm run lint` 는 clean main 과 `1d63b7c` 적용 트리 **양쪽에서 exit 0**(57파일/errors 0/warnings 0). ESLint 10 은 설정 없는 `.mjs` 를 오류로 보지 않고 **규칙 0개로 린트**한다(`calculateConfigForFile` rules: `.mjs` 0개 vs `.ts` 108개). exit 1 이 나는 유일한 경로는 H4 — 저장소 루트 실행(`npm error Missing script: "lint"`, npm 이 `/home/hkjang/package.json` 을 찾는다). 지시대로 가짜 루트 package.json 은 만들지 않았다.
- 대신 재현이 드러낸 실제 결함을 고쳤다: `eslint.config.js` 에 `**/*.{js,mjs}` 설정 객체 **추가**(ignores 아님) → 릴리즈 검증이 거쳐 가는 `scripts/run-with-supported-node.mjs` 가 64규칙으로 실제 검사된다. 수용 기준 5 를 위해 `1d63b7c` 를 cherry-pick(이 브랜치엔 없었고 `npm test` 가 실제로 exit 1 이었다 — 실행 확인).
- **확신 없는 곳**: (1) 러너가 `npm run lint` 를 정확히 어디서 돌렸는지는 추론이다. 루트 실행이 exit 1 임은 실행으로 확인했지만, 러너 로그를 볼 수 없어 그것이 그 실패였다는 직접 증거는 없다. (2) CI 의 Node 24 에서 lint 를 돌려보지 못했다(여기는 22.23.1, npm run 안에서는 PATH 가로채기로 20.19.2). (3) `globals` 패키지 대신 `console`/`process`/`URL` 만 인라인 선언했으므로 앞으로 다른 Node 전역(`Buffer`, `__dirname` 등)을 쓰는 스크립트를 추가하면 `no-undef` 가 난다 — 그때 전역을 추가하면 된다(규칙을 끄지 말 것).
- 일부러 안 한 것: 루트 `package.json` 추가(금지), `ci.yml`·`package-lock.json`·`vite.config.ts`·`vitest.config.ts`·`keep-dist-placeholder.test.ts` 변경, build/lint 를 런처 경유로 바꾸기(이번 재현에서 둘 다 Node 20 에서도 통과해 근거 없음).
- 다음 역할이 조심할 것: `web/src/test/eslint-config.test.ts` 는 `npm ci` 가 끝난 상태여야 돌고(실제 ESLint API 사용), `// @vitest-environment node` 다. `npm ci` 가 이번에 **high 취약점 2건**을 보고했다(지난 회차 0건) — 별도 후보로 ideas.json 에 올렸다.
- [러너 21:20] brief fallback — 차선 — 과제서의 가설 H1·H2·H3 은 전부 실행으로 반증됐고(`cd web && npm run lint` 는 두 트리 모두 exit 0), 재현되는 exit 1 은 �
- [러너 21:21] verify failed — 실패한 검증: npm run lint   # CI에서 가져옴 (exit 1)
