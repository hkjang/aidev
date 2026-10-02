- 과제: `cd web && npm test` 실패를 재현해 원인 지점을 고친다 (수정 과제 / 가치 5 / 위험 2 / 작업량 M)

- 왜: 지난 회차가 `cd web && npm test --silent` (exit 1) 로 verify-failed 로 끝났고 같은 이유로 두 번 실패했다. 지난 회차의 프로덕션 변경은 `internal/adapters/pop3/client.go` 5줄뿐이어서 Go 쪽 원인이 아니고, 프런트 검증이 막혀 있는 동안은 어떤 회차도 머지까지 갈 수 없다.

- **먼저 읽을 것 — 이번 정찰이 재현하지 못했다(정직한 한계)**
  - 이 세션에서 `npm ci` / `npm` 실행이 **권한 거부**됐고(`This command requires approval`), 워크트리에 `web/node_modules` 가 **없다**(실측: `ls node_modules` 실패). 따라서 **어느 스펙이 실패하는지는 미확인**이다. 추측을 사실로 적지 않기 위해 아래는 "확인한 것" 과 "가설" 을 분리해 적었다.
  - 구현자는 **가장 먼저 재현**하고, 실제 실패 출력(스펙 파일명·테스트명·assertion diff)을 원장에 그대로 붙여라. 재현 출력이 아래 가설과 다르면 **가설을 버리고 실제 출력을 따르라.**

- 수용 기준:
  1) `cd web && npm ci --no-audit --no-fund && npm test` 가 **통과**한다(exit 0). 실행 전 실패 출력과 실행 후 통과 출력을 둘 다 원장에 남긴다.
  2) `cd web && npm run typecheck` 도 통과한다(CI 의 frontend 잡이 typecheck → test 순서로 둘 다 돈다).
  3) 실패가 **프로덕션 결함**이었다면: 그 결함을 드러내는 assertion 이 남아 있고, 수정 전 그 테스트가 실패했음을 변이 검증(수정을 되돌리면 그 테스트만 실패)으로 보인다.
  4) 실패가 **테스트 쪽 결함**(날짜·타임존·순서 의존 등)이었다면: 테스트가 검증하던 **동작은 그대로 검증**하면서 환경 의존만 제거한다(예: `vi.useFakeTimers({toFake:['Date']})` + `vi.setSystemTime(...)` 로 시점을 고정). 테스트를 삭제·skip·약화하지 않는다.
  5) 프런트 **소스**(`web/src/**`)를 바꿨다면 `cd web && npm run build` 후 `git status --porcelain --untracked-files=all -- internal/transport/spa/assets` 가 **비어 있다**(CI 가 드리프트를 잡는다). 테스트 파일만 바꿨다면 자산은 바뀌지 않아야 한다.

- 재현·조사 순서 (이대로):
  1. `cd web && npm ci --no-audit --no-fund` — 워크트리에 node_modules 가 없으므로 **반드시 먼저**.
  2. `npm test 2>&1 | tail -80` 로 실패 스펙을 확정한다. 실패가 여러 개면 전부 적는다.
  3. 실패 스펙 하나를 단독 실행해 좁힌다: `npx vitest run src/<path>/<file>.test.tsx -t '<테스트명>'`.
  4. 그 스펙이 읽는 프로덕션 파일을 열어 **테스트가 틀렸는지 / 코드가 틀렸는지** 판정한다. 판정 근거를 원장에 적는다.

- 확인한 것 (이번 정찰이 실제로 열어 본 것):
  - `web/package.json`: `"test": "vitest run"`, `"typecheck": "tsc --noEmit"`. vitest 5.0.0 / vite 8.3.0 / react 19.3.0 / jsdom 30.0.1, `engines.node >= 22.12`.
  - `web/vite.config.ts`: `test: {environment:'jsdom', setupFiles:['./src/test-setup.ts'], exclude:['tests/e2e/**','node_modules/**']}`. `src/test-setup.ts` 는 `import '@testing-library/jest-dom/vitest'` **한 줄뿐** — 전역 타임존·시스템 시각 고정이 **없다**.
  - vitest 스펙은 `web/src/**` 에 **48개**(`find src -name '*.test.ts*'`).
  - `.github/workflows/release.yml` 은 `npm test` 를 **돌리지 않는다**(`npm ci` → `npm run build` → assets 드리프트 검사만). `npm test` 를 돌리는 것은 `.github/workflows/ci.yml` 의 `frontend` 잡(`npm run typecheck` 다음 `npm test`)과 `Makefile: frontend-test`. → 회차 노트의 "릴리즈 워크플로" 표현은 러너의 verify 단계를 뜻할 가능성이 크다. **워크플로 YAML 을 손댈 이유를 먼저 증명하지 말고, 실패 스펙부터 확정하라.**
  - `Makefile`: `frontend-test` = `npm run typecheck` + `npm test` 로 **의존성 설치를 하지 않는다**. `npm ci` 는 `frontend` 타깃에만 있다. 깨끗한 워크트리에서 `make frontend-test` / `cd web && npm test` 는 설치 없이는 성립하지 않는다.
  - `git log -- web/`: 가장 최근 프런트 변경은 `d97577a`(v0.25.0) — `src/features/jobs/jobs.test.tsx`(신규 84줄), `src/features/inbox/sent-view.test.tsx`(신규 70줄), `InboxPage.tsx`, `SearchTools.tsx`, `views.ts`, `jobs/index.tsx`, `AIContext.tsx`, `MessagePage.tsx`, `responses.ts`, `types.ts`, `sent/index.tsx`, `contracts.generated.ts`. `1d382aa`(v0.25.1) 은 `package.json`/`package-lock.json` 의 **버전 범프만**.
  - 날짜·타임존 의존 후보를 훑어 **대부분 안전함을 확인**했다: `jobs.test.tsx` 는 `created_at` 을 넣지만 jobs 디렉터리에 `Intl`/`toLocale*`/`date-fns`/`formatDistance` 사용이 **0건**(grep)이라 상대시각 드리프트가 없다. `SnoozeMessage.test.tsx`·`work.test.tsx` 는 `vi.setSystemTime` 으로 시점을 고정한다. `deadlines.test.ts`·`snooze.test.ts`·`actions.test.tsx` 는 `now` 를 인수로 **주입**한다. `SearchTools.test.tsx:53` 은 양쪽 모두 `new Date(2099,0,2,10,30)` 로컬 생성이라 타임존 중립이다.

- 가설 (순서대로 확인할 것 — 전부 **미확인**):
  1. **스펙이 실제로 실패한다** — 가장 가능성이 높다. exit **1** 은 vitest 가 실행되어 테스트가 떨어진 코드이고, node_modules 부재라면 보통 `sh: vitest: not found` 로 **127** 이 난다. 후보 순위: `d97577a` 가 추가·수정한 파일(`jobs.test.tsx`, `sent-view.test.tsx`, `inbox/refresh.test.tsx`, `inbox/SearchTools.test.tsx`, `messages/*`, `sent/sent.test.tsx`) → 그 다음 `settings/connection-diagnostic.test.tsx`(301줄, 진단 UI).
  2. **환경·설치 문제** — 러너 verify 가 `npm ci` 없이 `npm test` 를 돌렸다. 재현 1단계에서 `npm ci` 후 `npm test` 가 **그냥 통과**하면 이것이 답이다. 이때의 올바른 수정은 YAML 완화가 아니라 **`Makefile: frontend-test` 가 설치를 보장하게** 하는 것(`frontend` 처럼 `npm ci --no-audit --no-fund` 를 앞에 두거나 `node_modules` 가드) — 느슨하게 만드는 변경이 아니고 CI 동작도 바꾸지 않는다. 그리고 원장에 "프로덕션 결함이 아니었다" 를 **정직하게** 적어라.
  3. **jsdom 30 / vitest 5 / react 19.3 조합의 전역 경고가 실패로 승격** — `test-setup.ts` 에 그런 설정이 없으므로 가능성 낮음. 재현 출력이 특정 스펙을 지목하지 않을 때만 본다.

- 건드릴 파일 (총 **3개 이내**로 끝낼 것):
  - `web/src/<실패 스펙>` 과 그 스펙이 읽는 프로덕션 파일 1~2개 — 재현으로 확정된 것만.
  - (가설 2일 때만) `Makefile:frontend-test` — 설치 보장 한 줄.
  - 범위가 3개를 넘거나 실패 스펙이 서로 무관한 2개 이상이면 **가장 많은 스펙을 되살리는 하나만** 이번 회차에 담고 나머지는 보류 아이디어로 남겨라.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci --no-audit --no-fund` (최초 1회, 수 분)
  - `cd web && npm run typecheck`
  - `cd web && npm test` — 수용 기준의 핵심. 좁힐 때는 `npx vitest run src/<path>/<file>.test.tsx`
  - 프런트 소스를 바꿨다면: `cd web && npm run build` 그리고 `git status --porcelain --untracked-files=all -- internal/transport/spa/assets` 가 비어 있음
  - Go 쪽 회귀 없음 확인: `go build ./... && go vet ./...`, `go test -race -count=1 ./...`
  - `make lint-format`, `go run ./cmd/postra-contracts -check`, `git diff --check`

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들어 통과시키는 것은 금지** — `npm test` 스크립트에 `--passWithNoTests`/`--reporter=silent`/`--bail` 추가, 스펙 `skip`·`todo`·삭제, `vite.config.ts` 의 `test.exclude` 에 실패 스펙 추가, `ci.yml` 의 `npm test` 줄 제거·`continue-on-error` 추가 — 전부 반려 사유다.
  - `web/src/api/contracts.generated.ts` 는 **생성물**이다. 손으로 고치지 말고 `go run ./cmd/postra-contracts -check` 로 드리프트만 확인하라.
  - 보호 경로: `.github/workflows/**`, `internal/transport/spa/assets`(생성물), auth/session/OIDC/MCP OAuth, DB migrations. 가설 2가 아니면 워크플로·Makefile 을 건드리지 마라.
  - `internal/adapters/pop3/client.go` 의 `retrBody` 누적 상한은 **미병합 PR #22 / 04b15be 영역**이다. 이번 과제와 무관하니 손대지 마라.
  - 교훈: 브랜치 성공과 main 반영을 혼동하지 말 것(`git merge-base --is-ancestor`). Go 전체 통과만으로 PostgreSQL(`POSTRA_TEST_PG`)·브라우저 e2e(Chromium)가 돌았다고 적지 말 것 — 이번 과제에서 e2e(`npm run test:e2e`)는 **범위 밖**이다.
  - 커밋 메시지는 영어 conventional commit + 본문에 사유 + `Co-Authored-By: Claude …`.

- 차선 후보: **`sync.max_message_bytes` 의 0·음수 = 무제한 규약을 설정 카탈로그·관리자 문서에 명시** (가치 2 / 위험 1 / S) — `internal/application/sync.go:543-552 ingestOne` 의 `maxBytes > 0` 검사가 근거이고, POP3 본문 상한이 실제로는 전량 버퍼링 **뒤에** 걸린다는 점을 문서가 숨기지 않게 적는다. 10회 연속 차선이었던 항목이므로, 1순위가 끝내 재현되지 않을 때만 고른다. (주의: 프런트 검증이 막혀 있는 동안은 어떤 과제도 verify 를 통과하지 못할 수 있다 — 1순위를 먼저 끝내는 것이 거의 항상 옳다.)
