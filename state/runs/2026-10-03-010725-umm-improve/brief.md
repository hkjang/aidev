# 과제서 — 2026-10-03-010725-umm-improve (umm)

- 과제: 수정 과제 — 저장소 루트에서 `npm test --silent` 가 체크아웃을 벗어나 바깥 package.json 을 읽고 exit 1 (가치 4 / 위험 2 / 작업량 S)

- 왜: 지정된 검증 명령 `npm test --silent` 를 이 base(715fd97)의 깨끗한 워크트리 루트에서 그대로 돌려 `npm error Missing script: "test"` / exit 1 을 재현했습니다. 프런트엔드가 `web/` 에만 있어 저장소 루트에 package.json 이 없고, npm 은 package.json 을 찾아 **체크아웃 밖으로** 올라가 이 기계에서는 `/home/hkjang/package.json` 에 붙습니다(그 파일에는 `test` 스크립트가 없습니다). 고치면 릴리즈를 두 번 막은 검증이 저장소 자신의 시험을 돌게 되고, 바깥 파일이 같은 이름의 스크립트를 갖고 있을 때 우리 것 대신 그것이 **실행**되는 경로도 함께 막힙니다.

- 수용 기준:
  1) 저장소 루트에서 `npm test --silent` 가 EXIT=0 이고, 그 실행이 실제로 이 저장소의 vitest 를 돈 것(파일 수·시험 수가 출력에 보임 — 숫자는 실행 결과에서 읽고 추정하지 말 것)임을 확인한다. `--silent` 는 오류를 숨기므로 진단은 `--silent` 없이 한 번 더 돌려 확인한다.
  2) 루트의 새 설정 파일만 치우면 같은 `Missing script: "test"` / exit 1 이 그대로 다시 나고, 되돌리면 EXIT=0 이 된다(이 파일이 고침의 원인이라는 증거).
  3) 위임이 실패를 삼키지 않는다: `web/src/lib/` 의 아무 시험 하나를 일부러 깨뜨린 뒤 루트 `npm test --silent` 가 **비0** 으로 끝나는 것을 확인하고 되돌린다. (EXIT=0 이 "시험이 없어서" 가 아니라는 증거)
  4) 기존 검증이 그대로 통과: `npm --prefix web run lint`·`typecheck`·`build`, `node web/scripts/check-i18n.mjs`, `scripts/check-version.sh`, `go vet ./...`, `go build ./cmd/...`, `go test ./... -count=1`.
  5) 릴리즈 경로 확인: `release.yml` 과 같은 `docker build --build-arg VERSION=$(cat VERSION) .` 가 EXIT=0 (검증용 이미지는 지울 것).

- 건드릴 파일 (프로덕션 코드 0개, 설정 파일 2~4개):
  - `package.json` (저장소 루트, 신규) — `private: true`, `version` 필드 **없음**(`scripts/check-version.sh` 가 지킬 값을 새로 만들지 않기 위해), `scripts` 에 CI 가 `working-directory: web` 로 돌리는 것들을 Makefile 이 이미 쓰는 `npm --prefix web …` 형태로 위임: `test`·`typecheck`·`lint`·`test:offline-queue`·`verify:pwa`·`build`. 이 파일은 `scripts/*.js`(CommonJS, `require()`) 의 모듈 타입도 함께 못 박습니다 — 지금은 그 타입이 체크아웃 밖 package.json 에 달려 있습니다.
  - `package-lock.json` (저장소 루트, 신규) — 의존성 0개. 루트에서의 `npm ci`·`npm audit` 도 저장소 안에 머물게 합니다.
  - **이미 쓰인 초안이 로컬에 있습니다**: `git show 88cdca1:package.json` / `git show 88cdca1:package-lock.json` (브랜치 `auto/2026-10-02-2300`). 사람이 반려한 것이 **아닙니다** — origin 에는 `auto/2026-10-02-*` 브랜치가 하나도 없고(origin 의 마지막 auto 브랜치는 `auto/2026-09-30-2012`), 그 세 회차는 검증 실패로 푸시되지 못했을 뿐 PR 도 리뷰도 없었습니다. 그대로 베껴도 되고 줄이거나 고쳐도 됩니다. 운영자 규칙 1번(사람이 반려한 접근 재제출 금지)에 걸리지 않습니다.
  - 아래 "두 번째 실패 모드" 가 **실제로 재현되면**: `web/package.json` 에 `engines.node` 선언 + `web/scripts/run-on-supported-node.mjs` (신규 래퍼). 선례는 `git show 2d8ad12`(브랜치 `auto/2026-10-02-2012`, 역시 origin 에 없음 = 미반려). 재현되지 않으면 **손대지 마세요** — 파일 수를 넷 안쪽으로 유지합니다.

- 두 번째 실패 모드 (미확인 — 반드시 먼저 확인할 것):
  루트 위임이 생기면 `npm test --silent` 는 `npm --prefix web test` → `vitest run` 을 돕니다. 이 워크트리에는 `web/node_modules` 가 없어(이번 정찰에서 `cd web && npm test` → `sh: 1: vitest: not found`, **exit 127**) 그 뒤를 확인하지 못했습니다. 그러니 **먼저 CI 와 같은 `npm ci` 를 `web/` 에서 돌린 뒤** 루트 `npm test --silent` 의 실제 종료 코드를 보세요.
  이 환경에서는 npm 이 lifecycle 스크립트의 PATH 에 패키지 디렉터리부터 **파일시스템 루트까지** 모든 조상의 `node_modules/.bin` 을 앞에 붙이고, `/home/hkjang/node_modules` 에 `node@20.19.2` 가 있어 인터프리터가 가려집니다(2026-10-02-2012 회차가 undici 를 계측해 `version=v20.19.2` 로 직접 증명). 그러면 jsdom 30 → undici 8 이 로드 시점에 `TypeError: webidl.util.markAsUncloneable is not a function` 으로 18개 파일 전부 죽고 **exit 1** 입니다. 직접 실행한 node 는 v22.23.1 이고 `web/package.json` 에는 지금 `engines` 선언이 없습니다(이번에 열어 확인). 이 오류가 나오면 루트 위임만으로는 지정된 검증이 통과하지 않으므로 `engines.node` + 래퍼까지 이번 회차에 넣으세요 — 그래도 파일 4개입니다.
  주의: 지정된 실패가 `exit 1` 이라는 사실만으로는 러너가 루트에서 돌렸는지 `web/` 에서 돌렸는지 가려지지 않습니다(루트=Missing script exit 1, 의존성이 설치된 `web/`=undici exit 1, 둘 다 1). **두 자리에서 모두 EXIT=0 이 되는 것**을 확인하세요.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd /home/hkjang/.cache/auto-improve-wt/umm && npm test --silent` ← 지정된 검증. `echo $?` 로 EXIT 확인.
  - `cd .../umm && npm test` (진단용, `--silent` 없이)
  - `cd .../umm/web && npm ci` → `cd .../umm/web && npm test`
  - `npm --prefix web run typecheck` / `npm --prefix web run lint` / `npm --prefix web run build` / `npm --prefix web run test:offline-queue`
  - `node web/scripts/check-i18n.mjs` · `scripts/check-version.sh`
  - `go vet ./...` · `go build ./cmd/...` · `go test ./... -count=1` (POSTGRES_DSN 없으면 DB 통합은 SKIP — 통과 표시만으로 돌았다고 쓰지 말 것)
  - `docker build --build-arg VERSION=$(cat VERSION) .` (릴리즈 경로, 이미지 삭제)

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들어 통과시키는 것 금지.** `.github/workflows/ci.yml`·`release.yml` 을 **바꾸지 마세요.** 이번 정찰에서 읽어 보면 `release.yml` 에는 `npm` 이 한 줄도 없고(`grep -n npm` 결과 0건), `ci.yml` 의 npm 단계(114~137행)는 `working-directory: web` 로 돌아 CI 에서는 이미 통과합니다 — **GitHub 워크플로는 멀쩡하고**, 깨진 것은 러너가 저장소 루트에서 같은 명령을 돌릴 때입니다. `npm audit --audit-level=high` 의 임계를 낮추거나 `--production` 을 붙이는 것도 금지.
  - 루트 package.json 에 `version` 을 넣지 마세요 — `scripts/check-version.sh` 는 `VERSION` 과 `web/package.json` 을 명시 경로로 보는데, 새 버전 값이 생기면 지켜야 할 곳이 하나 늘고 릴리즈가 거기서 깨집니다.
  - `workspaces` 를 쓰지 마세요 — `web/node_modules` 설치 위치와 lockfile 모양이 바뀌어 CI 의 `cache: npm`(ci.yml:42)·`npm ci` 와 Dockerfile 빌드까지 영향을 받습니다. 이번 과제는 위임 한 겹입니다.
  - `Makefile`·`Dockerfile` 은 손대지 마세요(보류 아이디어 "Node 하한 드리프트 검사" 는 별 과제).
  - `/home/hkjang/package.json` 은 `@hkjang/openpro`·`playwright` 같은 이 저장소가 선언하지 않는 의존성을 갖고 있습니다 — 그 트리에 의존하는 설정을 만들지 마세요. 그 파일을 고치는 것도 금지(저장소 밖, 되돌릴 수 없는 환경 변경).
  - 증거로 grep 결과를 내지 마세요 — 종료 코드와 실행 출력으로 보이세요. 래퍼를 넣는 경우에는 실패하는 시험 하나로 "실패를 삼키지 않는다" 를 꼭 확인하세요(수용 기준 3).
  - 보호 경로(`internal/auth/`, `migrations/`, `.github/workflows/`)는 이번에 건드릴 이유가 없습니다.

- 차선 후보: `web/package.json` 의 `engines.node: ">=22.22.2"` + `web/scripts/run-on-supported-node.mjs` 래퍼 **단독** — 러너가 `npm test --silent` 를 `web/` 안에서 돌린 것으로 밝혀져 루트 위임이 과제와 무관한 경우. (3순위: `verify-offline-queue.mjs` 가 `.ts` 를 직접 import 해 Node 의 타입 스트리핑에 의존하는 계약을 `verify-pwa.mjs` 와 맞추기 — 가치 2 / 위험 1 / S)
