# 과제서 (2026-10-02 정찰)

- 과제: `web/package.json` 의 `test` 스크립트에서 Node 버전에 따라 전개되지 않는 `**` 글롭을 없애기 (가치 4 / 위험 1 / 작업량 S)

- 왜: 지난 회차의 검증이 `cd web && npm test --silent` 에서 exit 1 로 죽었고, 이번 회차에 이 워크트리에서 그대로 재현했다 — 출력은 테스트 실패가 아니라 **파일 발견 실패**다: `Could not find '/home/hkjang/.cache/auto-improve-wt/relio/web/test/**/*.test.ts'` (Node v22.23.1). 스크립트가 `node --test 'test/**/*.test.ts'` 로 글롭을 **따옴표로 묶어** Node 에게 넘기기 때문에, 글롭을 Node 가 처리해 주는 버전(24, CI 가 핀한 버전)에서만 돌고 그 아래에서는 테스트가 한 건도 실행되지 않은 채 빨개진다. 셸이 전개하도록 바꾸면 Node 버전과 무관하게 같은 4개 파일이 돌아, CI(`node-version: 24`)와 러너·개발자 로컬이 같은 결과를 낸다.

- 수용 기준:
  1) 의존성이 설치된 상태에서 `cd web && npm test --silent` 가 exit 0 이고, `web/test/` 의 4개 파일(dateTime·login·memberList·silentSso)에서 27건이 pass 한다. `Could not find … test/**/*.test.ts` 메시지가 사라진다. (현재 Node 는 v22.23.1 — 이 환경에서 통과를 눈으로 확인할 것)
  2) 스크립트가 `**` 를 따옴표로 묶어 Node 에 넘기지 않는다. 새 테스트 파일이 자동으로 포함되는지 증명할 것: `web/test/tmp_probe.test.ts` 에 반드시 실패하는 테스트 하나를 넣고 `npm test` 가 **비정상 종료**하는 것을 본 뒤 삭제하고 다시 27/27 통과를 확인한다(2026-09-29 회차가 쓴 기법).
  3) `.github/workflows/ci.yml` 과 `.github/workflows/release.yml` 은 **한 글자도** 바뀌지 않는다(`git diff --name-only` 가 `web/package.json` 만 보여야 한다). 워크플로의 `node-version: 24` 를 내리거나, 테스트를 빼거나, `|| true` 를 붙이거나, `login.test.ts` 를 제외하는 식의 "느슨하게 만들어 통과" 는 금지다.
  4) `make test`(go test → go vet → `cd web && npm ci && npm run typecheck && npm test`)가 npm test 단계를 **지나간다**. npm ci 가 네트워크 때문에 실패하는 환경이면 그 사실을 보고하고, 설치된 상태에서 `npm test` 부분만 따로 돌려 통과를 보일 것.

- 건드릴 파일 (프로덕션 1파일):
  - `web/package.json:scripts.test` — `node --test 'test/**/*.test.ts'` → 셸이 전개하는 형태로. 1순위 권고: `node --test test/*.test.ts` (따옴표 없음). npm 은 스크립트를 `sh -c` 로 돌리므로 셸이 4개 파일을 전개해 Node 버전과 무관하다. 주의: `sh` 는 `**` 를 재귀 전개하지 않으므로 `test/**/*.test.ts` 를 따옴표만 떼서 쓰지 말 것 — 지금 테스트 4개는 모두 `web/test/` 바로 아래 평면으로 있다(확인함). 하위 디렉터리를 지금 지원할 필요는 없다.
  - 1순위가 어떤 이유로든 성립하지 않으면(예: 셸이 sh 가 아닌 환경 고려) 파일 4개를 명시 열거: `node --test test/dateTime.test.ts test/login.test.ts test/memberList.test.ts test/silentSso.test.ts`. 이 형태는 2026-10-02 회차가 이 Node 에서 27/27 통과를 실제로 확인했다. 다만 새 테스트 파일이 조용히 빠지므로 2순위다 — 고른다면 수용 기준 2) 를 "새 파일을 추가할 때 package.json 도 함께 고쳐야 한다" 로 바꾸지 말고, 1순위를 먼저 시도한 기록을 남길 것.
  - 다른 파일은 건드리지 말 것. README·engines·node 버전 핀 추가는 이번 범위 밖(파일 수를 1개로 유지).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci` — **이번 정찰에서 실행하지 못했다(미확인)**: 이 워크트리에는 `web/node_modules` 가 아예 없고, 정찰 세션에서는 설치 명령이 권한으로 막혔다. 설치 없이 `npm test` 를 돌리면 글롭을 고친 뒤에도 `login.test.ts` 의 `import { build } from 'esbuild'` 와 react 임포트가 `ERR_MODULE_NOT_FOUND` 로 죽는다(추론, 미확인). 구현자는 **먼저 `npm ci` 를 돌려** 설치가 되는지 확인하고, 안 되면 그 출력을 노트에 남길 것.
  - `cd web && npm test --silent` — 기대: 27 pass / 0 fail, exit 0.
  - `node --version` — 기대: v22.18 이상. 테스트 파일들은 Node 의 타입 스트리핑에 의존한다(`test/dateTime.test.ts:1-4` 주석이 "Node 22.18+ / 24" 라고 적어 둔다). 22.18 미만이면 `key: string` 에서 구문 오류가 나는데 그것은 이 과제의 범위가 아니다 — 그 경우는 글롭 수정만 보이고 환경 제약을 보고할 것.
  - `make test` — 전체. Go 쪽은 이번 변경과 무관하지만 한 번 돌려 둘 것.
  - `git diff --name-only` / `git diff --check`.

- 위험과 피할 것:
  - **보호 경로**: `.github/workflows/release.yml`·`ci.yml` 은 열어서 읽기만 할 것(이번 정찰이 읽었다: 둘 다 `actions/setup-node@v4` + `node-version: 24`, release 의 `Test source` 단계는 `npm --prefix web ci / audit / run typecheck / run build` 로 **npm test 를 아직 돌리지 않는다**). 릴리즈 워크플로에 npm test 를 **추가하는 것도 이번 과제가 아니다** — 2026-09-06 에 릴리즈 경로 변경이 두 번 깨져 되돌림 PR + 자율화 강등으로 이어졌다.
  - 바꾸는 것은 npm 스크립트 한 줄이지만 그 줄이 CI·릴리즈·`make test` 세 경로에서 모두 호출된다. 반드시 로컬에서 실제로 돌려 통과를 본 뒤 커밋할 것(소스 문자열 확인이나 `make -n` 만으로는 증거가 아니다).
  - `web/test/login.test.ts` 는 esbuild 로 실제 `src/pages/Login.tsx` 를 번들링해 React 서버 렌더링한다 — 느리고 임시 디렉터리를 만든다. 테스트 자체를 손대지 말 것.
  - `esbuild` 는 `web/package.json` 의 의존성에 **선언돼 있지 않다**(dependencies/devDependencies 모두에 없음 — 확인함). vite 의 전이 의존성에 얹혀 돌고 있다. 실제 결함이지만 이번 회차에서 고치지 말 것 — package-lock 재생성과 네트워크가 끼어 파일 수와 위험이 올라간다. 보류 아이디어로 남겼다.
  - 커밋 메시지는 저장소 관례대로 `type(scope): 한국어 요약` (예: `fix(web): npm test 의 파일 발견이 Node 버전에 의존하지 않게 한다`).

- 차선 후보: 감사 화면 Frame 부제를 실제 채널값으로 맞추기 — `web/src/pages/AdminPages.tsx:619` 의 부제가 'Login과 Key' 인데 :620 드롭다운은 `['WEB','API','MCP','ADMIN','LOGIN','SSO']` 로, `Key` 는 존재하지 않는 채널이고 `SSO` 가 빠져 있다(2026-09-30 기록 근거, 이번 회차 미확인). 드롭다운을 정본으로 문자열 하나만 수정. `AdminPages.tsx` 는 한 줄이 매우 길어 대규모 재포매팅 금지. 단 1순위(verify 실패 수정)가 성립하지 않는 경우에만 고를 것 — 1순위를 고치지 않으면 이 회차의 검증도 같은 이유로 또 실패한다.
