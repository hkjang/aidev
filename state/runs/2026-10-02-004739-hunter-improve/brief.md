- 과제: `web/package.json` 의 `test` 스크립트가 지금 Node 에서 제거된 `--experimental-strip-types` 를 넘겨 테스트가 한 건도 돌지 않고 exit 9 로 죽는 것 수정 (가치 5 / 위험 1 / 작업량 S)

- 왜: `web/package.json:10` 의 `"test": "node --experimental-strip-types --test tests/*.test.mjs"` 가 현재 Node 에서 `node: bad option: --experimental-strip-types` 로 즉시 죽어(종료 코드 9) web 테스트 19파일이 **한 건도 실행되지 않는다** — 이번 회차 verify 실패(`cd web && npm test --silent (exit 9)`)와 릴리즈 워크플로 2연속 실패가 모두 이 한 줄이다. CI Verify 의 `npm ci && npm test && npm run build`(`.github/workflows/ci.yml`, `node-version: '26'`)도 같은 스크립트를 쓰므로 고치지 않으면 main 의 모든 PR·태그가 계속 막힌다. Node 는 22.18/23.6 이후 `.ts` 타입 스트리핑이 기본 동작이므로 플래그만 빼면 같은 일을 한다.

- 수용 기준:
  1) 저장소 web 디렉터리에서 `npm ci && npm test` 가 종료 코드 0 으로 끝나고, 테스트 19파일이 실제로 실행된 PASS 수(이전 회차 기록상 105 전후)가 출력된다. "0 테스트 실행 후 성공" 은 불합격 — 반드시 통과 개수를 로그로 확인하고 보고서에 적을 것.
  2) `tests/*.test.mjs` 가 `../src/*.ts` 를 그대로 import 하는 현재 구조를 유지한다(빌드 산출물·컴파일 단계·새 devDependency 추가 금지). `tests/list-view.test.mjs:9` 의 `from "../src/list-view.ts"` 같은 import 는 한 글자도 바꾸지 않는다.
  3) CI 가 실제로 쓰는 순서 그대로 `npm ci && npm test && npm run build` 가 모두 통과한다(세 번째가 `tsc --noEmit && vite build`).
  4) 프로덕션 TS/Go 코드는 손대지 않는다. 변경 파일은 `web/package.json` 1개가 목표이며, 그 이상이면 왜인지 보고서에 적는다.

- 건드릴 파일:
  - `web/package.json:10` — `scripts.test` 에서 `--experimental-strip-types` 를 제거해 `"test": "node --test tests/*.test.mjs"` 로 한다. 다른 스크립트(`build`/`typecheck`/`dev`/`preview`)·`version` 필드·의존성은 건드리지 않는다.
  - (필요할 때만) `AGENTS.md` 7장 / `docs/` 안의 테스트 실행 안내 — 현재 저장소 전체 grep 결과 `--experimental-strip-types` 문자열은 `web/package.json:10` **한 곳뿐**이므로 문서 수정은 아마 불필요하다. 확인만 하고 불필요하면 손대지 말 것.

- 검증 명령 (저장소 루트에서, 전부 실제로 돌리고 출력을 보고서에 붙일 것):
  ```sh
  node --version                      # 어떤 Node 에서 검증했는지 반드시 기록
  npm --prefix web ci
  npm --prefix web test               # 종료 코드 0 + pass 개수 확인
  npm --prefix web run typecheck
  npm --prefix web run build
  git diff --check
  ```
  - 통과 후에만: `mkdir -p internal/webassets/dist && cp -a web/dist/. internal/webassets/dist/` → `go vet ./...` → `go build ./cmd/hunter` (Go 코드 무변경이므로 선택 사항이지만, 릴리즈 경로를 건드리는 변경이라 한 번 확인해 두면 좋다).
  - `go test -race ./...` 는 PostgreSQL(`HUNTER_TEST_DSN`)이 필요하고 Go 변경이 없으므로 생략 가능. 생략했으면 생략했다고 쓸 것.

- 위험과 피할 것:
  - **미확인(가장 중요)**: 정찰 세션에 `node` 직접 실행 권한이 없어 `node --test tests/*.test.mjs`(플래그 없이)가 이 환경에서 실제로 `.ts` 를 import 하는지 **확인하지 못했다**. 확인한 것은 (a) `npm test` 가 `node: bad option: --experimental-strip-types` 로 exit 9 라는 재현, (b) 로컬 `node --version` 이 `v22.23.1` 이라는 출력, (c) 저장소에 그 플래그가 한 곳뿐이라는 것, (d) `web/src` 에 enum/namespace 같은 **비(非)지울 수 있는 TS 구문이 없어** 기본 타입 스트리핑으로 충분하다는 grep 결과다. 구현자는 먼저 `node --test web/tests/list-view.test.mjs` 를 한 번 돌려 플래그 없이 되는지 **직접 확인한 뒤** package.json 을 고쳐라.
  - 플래그 없이도 안 되면(= 그 Node 가 타입 스트리핑을 기본 지원하지 않음) `--experimental-strip-types` 제거가 답이 아니다. 그때는 `node --help` 로 이 Node 가 제공하는 타입 스트리핑 플래그 이름(예: `--experimental-transform-types`, 또는 `--no-experimental-strip-types` 의 반대쪽)을 확인해 **실제로 존재하는 플래그**로 바꿔라. 추측으로 플래그 이름을 적지 말고 `node --help` 출력을 근거로 삼을 것.
  - **워크플로를 느슨하게 만들어 통과시키는 것은 금지.** `ci.yml` 의 `npm test` 를 빼거나 `|| true` 를 붙이거나 `node-version: '26'` 을 낮추는 것은 반려 사유다. `.github/workflows/*` 는 건드리지 말 것 — 원인은 워크플로가 아니라 package.json 이다.
  - 테스트 파일을 `.mts`/`.ts` 로 개명하거나 tsx·ts-node·vitest 같은 러너를 새로 들이지 말 것(작업량이 S 를 넘고 CI·Dockerfile 파급이 생긴다).
  - `web/node_modules` 는 이 워크트리에 설치돼 있지 않다(`web/node_modules/.bin` 없음). `npm ci` 를 먼저 돌려야 `npm run build`/`typecheck` 가 된다. 단, 순수 테스트는 node_modules 없이도 돌 수 있으니 테스트 한 파일 확인은 `npm ci` 전에 해도 된다.
  - `third_party/pentagi` 원본 312파일, `internal/app` 의 auth/finding_bulk, `VERSION`·버전 필드는 이번 범위 밖이다. `web/package.json` 의 `"version": "1.18.0"` 도 그대로 둘 것(릴리즈 단계가 다룬다).
  - 과거 교훈: 이 저장소는 "로컬에서 통과했다" 와 "CI(Node 26)에서 통과했다" 가 다르다. 이번 변경은 바로 그 차이에서 터진 것이므로, 보고서에 **어떤 Node 버전에서 무엇을 돌렸는지** 명시하고 Node 26 CI 동등성은 미확인으로 남겨라(원격 CI 를 돌릴 수 없다면).

- 검토한 대안과 결론 (solution-exploration):
  1) **플래그 제거** — `node --test tests/*.test.mjs`. 움직이는 부품 추가 없음, 1파일. 전제: 이 Node 가 `.ts` 타입 스트리핑을 기본 지원한다(Node 22.18/23.6+).
  2) `--experimental-strip-types` → 실재하는 다른 플래그로 교체. 1)이 안 될 때만. 전제: `node --help` 에 대체 플래그가 있다.
  3) tsx/vitest 같은 러너 도입 — devDependency·Dockerfile·CI 파급이 생기고 S 를 넘는다. **기각**.
  4) 테스트를 `.ts`→컴파일 후 실행으로 전환 — 빌드 단계 추가, 테스트가 소스를 직접 import 하는 현재 장점을 잃는다. **기각**.
  5) 아무것도 안 함 / CI 를 느슨하게 — main 의 모든 PR·태그가 계속 막히고 반려 사유다. **기각**.
  → **1) 채택**, 2)를 비상 경로로. 가장 많이 기대고 있는 전제는 "이 Node 가 플래그 없이 `.ts` 를 import 한다" 이며, 깨지면 여기서 깨진다. 그래서 수용 기준 1)에 "실행된 PASS 수 확인" 을 넣었다.

- 실행 순서와 체크포인트 (implementation-planning):
  1. `node --test web/tests/list-view.test.mjs` — 플래그 없이 되는지 확인. **체크포인트**: 실패하면 2)번 대안으로 갈아타고 `node --help` 출력을 근거로 남긴다. 성공해야 다음 단계.
  2. `web/package.json:10` 에서 플래그 제거. 증거: `npm --prefix web test` 종료 코드 0 + pass 개수.
  3. `npm --prefix web ci && npm --prefix web run typecheck && npm --prefix web run build`. 증거: 각 종료 코드 0.
  4. `git diff --check` 와 변경 파일 목록 확인. 증거: `git status --short` 가 `web/package.json` 만 보여야 한다.
  - 각 단계 끝에서 저장소는 동작 상태다(1단계는 읽기만, 2단계부터 테스트가 돈다).
  - **범위 밖**: `.github/workflows/*`, `Dockerfile`, `scripts/release.sh`, `VERSION`·버전 필드, `third_party/pentagi`, Go 코드, 테스트 내용 추가·수정, 새 의존성.

- 작업량 근거와 예비 (estimating):
  - 분해: 확인 1회(5분) + package.json 1줄(2분) + `npm ci`·test·typecheck·build(15~25분, `npm ci` 가 대부분) + 보고(5분).
  - 범위: **30~45분**, 10회 중 8회는 이 안. 방법은 유사 비교(지난 회차들의 web 1파일 변경 회차가 같은 검증 묶음을 돌려 이 정도였음) 1종이며 과거 실측 rate 가 아니라 추정이다.
  - 상한 시나리오: 1단계가 실패해 대안 2)로 가면 `node --help` 조사·플래그 선택으로 +20분(= 예비). 이것은 알려진 불확실성에 대한 contingency 이며, "Node 가 아예 `.ts` 를 못 읽는 경우"(러너 전환 필요 = L)는 이 과제의 예비가 아니라 **회차를 중단하고 보고할 사유**다.
  - 포함: 검증 명령 실행과 보고. **제외**: 원격 CI·Docker 빌드·릴리즈 실행, Go DB 테스트, 문서 재생성.

- 차선 후보: bulk due_date 의 해제·형식·오류 래핑 순수 테스트 — `internal/app/finding_bulk.go:67-70` 이 `due_date` 를 `validateFindingOpsResource` 에 위임하는 경로를 `internal/app/finding_bulk_validate_test.go`(DSN 불필요 하네스 이미 있음)에서 nil/빈값 해제·잘못된 JSON 타입·400 래핑으로 고정 (가치 3 / 위험 1 / 작업량 S). 단, 1순위가 성립하면 **반드시 1순위만** 하라 — web 테스트가 전혀 돌지 않는 상태에서 다른 과제를 얹으면 검증 자체가 불가능하다.
