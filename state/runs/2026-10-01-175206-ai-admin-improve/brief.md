# 과제서 (2026-10-01 정찰) — 수정 과제 (verify-failed 대응)

- 과제: 웹 테스트가 Testing Library 기본 1초 `asyncUtilTimeout` 에 걸려 간헐 실패하는 것을 셋업에서 명시적 예산으로 고정 (가치 3 / 위험 1 / 작업량 S)

- 왜: `cd web && npm test` 가 두 회차 연속 같은 테스트에서 깨졌다 — `src/pages/AdminInteractions.test.tsx:41` 의 `await screen.findByText('샘플')` 이 "Unable to find an element with the text: 샘플" 로 실패했다(verify 로그: `2026-10-01-141220-ai-admin-improve/verify.txt:112-113`, `:382-388`). 같은 테스트는 2026-09-29 회차에서 **2774ms 로 통과**했고(`2026-09-29-045147-ai-admin-improve/verify.txt:48-49`) 실패 회차에서는 **1391ms 에 ×** 로 끝났다(`:40`) — 즉 단정이 틀린 것이 아니라 `findByText` 의 기본 대기 예산 1000ms 가 다 소진된 것이다. 저장소 어디에도 `configure()` 호출이 없어(`grep -rn "configure(" web/src` → 0건) 18개 테스트 파일의 모든 `findBy*`/`waitFor` 가 `vite.config.ts:testTimeout: 15_000` 와 무관하게 1초만 기다린다. 고치면 머신 부하에 따라 릴리즈가 막히는 이 실패 계열이 사라지고, 같은 함정이 있는 다른 31개 호출 지점(`App.test.tsx` 16건·`SystemSettingsPage.test.tsx` 9건·`AppShell.test.tsx` 4건·`AdminInteractions` 2건·`LoginPage` 1건)이 함께 안정된다.

- 수용 기준:
  1) `cd web && npm test` 가 통과하고, `npx vitest run src/pages/AdminInteractions.test.tsx` 를 **연속 5회** 돌려 5회 모두 PASS(SKIP 아님)한다.
  2) 결함이 "앱이 행을 그리기까지 필요한 순차 fetch 왕복이 1초 예산을 넘는다" 임을 **부하와 무관하게 결정적으로** 증명한다: `AdminInteractions.test.tsx` 의 `commonFetch` 스텁이 각 응답을 실제 HTTP 처럼 약간 지연시켜 돌려주게 하고(예: 각 요청 150ms 뒤 resolve — `await new Promise(r => setTimeout(r, 150))`), 셋업 수정 **전에는 그 테스트가 1초 예산 소진으로 FAIL**, **후에는 PASS** 함을 로그로 남긴다. 프로덕션 배선(실제 `App`·`AuthProvider`·`ThemeProvider`·react-query)은 그대로 쓰고 컴포넌트를 손으로 만든 대역으로 바꾸지 않는다.
  3) 테스트가 기다리는 **대상·단정은 그대로**다 — `findByText('샘플')`·`getByRole('button', {name: /수정/})`·`findByText('금칙어 관리 수정')` 세 단정이 모두 남아 있고, `it.skip`·`retry`·`test.fails`·단정 삭제·`vite.config.ts` 의 `testTimeout` 변경으로 통과시키지 않는다(검증을 느슨하게 만드는 것은 금지).
  4) `cd web && npm run build`(`tsc -b && vite build`) 가 통과한다 — `src/test/setup.ts` 는 `tsconfig.app.json` 의 타입 검사 범위에 들어갈 수 있으므로 import 와 타입이 성립해야 한다.

- 건드릴 파일 (프로덕션 파일 0개):
  - `web/src/test/setup.ts` — 파일 상단에 `import { configure } from '@testing-library/react'`(`@testing-library/react` 가 dom 의 `configure` 를 재수출한다; 안 되면 `@testing-library/dom`) 후 `configure({ asyncUtilTimeout: 5_000 })` 를 호출. 왜 5초인지 주석으로 남길 것(근거: `vite.config.ts` 의 `testTimeout: 15_000` 보다 작고, 실패 회차에서 통과한 테스트들이 최대 2193ms 였다 — `verify.txt:102`). 기존 `localStorage`·`matchMedia`·`ResizeObserver`·`afterEach(cleanup)` 블록은 손대지 말 것.
  - `web/src/pages/AdminInteractions.test.tsx:22-38` — `commonFetch` 가 돌려주는 Promise 에 고정 지연을 넣어 기준 2의 red→green 을 결정적으로 만들기. 단정 3줄(`:41`,`:42`,`:43`)은 그대로.
  - (선택) `docs/` 에 쓸 내용 없음. **`internal/ui/dist` 재빌드는 필요 없다** — 바뀌는 것이 vitest 셋업과 테스트 파일뿐이고 SPA 번들에 들어가는 `web/src` 앱 코드는 건드리지 않기 때문이다. 앱 코드를 고치게 되면 `make build` 로 dist 를 반드시 교체해야 한다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd web && npm ci` (워크트리에 `node_modules` 가 없다 — 정찰 환경에서는 npm 실행이 막혀 있어 직접 돌리지 못했다)
  - `cd web && npm test` ← 실패했던 그 검증. `npm test --silent` 와 동일하게 `vitest run` 이다(`web/package.json:scripts.test`).
  - `cd web && npx vitest run src/pages/AdminInteractions.test.tsx --reporter=verbose` (연속 5회, PASS·SKIP 아님 확인)
  - `cd web && npm run build`
  - Go 쪽은 건드리지 않지만 회귀 확인용으로 `make lint`(gofmt·go vet·`scripts/verify-version.sh` — VERSION 1.2.31) 와 `go build ./...` 는 돌려 둘 것. `go test` 는 이 과제와 무관하다.

- 위험과 피할 것:
  - **검증을 느슨하게 만들지 말 것**: `it.skip`/`describe.skip`, `vitest --retry`, `test.fails`, 실패하는 단정 삭제, `vite.config.ts` 의 `testTimeout` 상향, `npm test` 를 CI(`.github/workflows/ci.yml:87` `npm ci && npm test && npm run build`)에서 빼는 것 모두 금지. 바꾸는 것은 "비동기 쿼리가 기다릴 수 있는 시간" 하나이고, 무엇을 기다리는지는 그대로다.
  - `.github/workflows/*` 는 건드리지 말 것 — `ci.yml` 의 web 단계는 이미 `npm ci && npm test && npm run build` 로 올바르고, `release.yml` 에는 web 테스트 단계가 아예 없다(확인함). 깨진 것은 워크플로 정의가 아니라 테스트의 대기 예산이다.
  - `web/src/test/setup.ts` 의 `afterEach` 는 `cleanup()`·`vi.restoreAllMocks()`·`localStorage.clear()`·CSRF 쿠키 삭제를 함께 하므로 순서를 바꾸면 다른 17개 파일이 서로 오염된다 — 추가만 하고 재배치하지 말 것.
  - 타임아웃을 과하게 올리면(예: 15초) 진짜 실패가 15초씩 매달려 전체 수행 시간이 늘어난다(현재 전체 12.74s). 5초 선을 넘기지 말 것.
  - `AdminInteractions.test.tsx` 에 지연을 넣을 때 `vi.useFakeTimers()` 를 쓰지 말 것 — `userEvent` 와 react-query 가 실제 타이머에 얽혀 다른 실패 양상을 만든다. 실제 `setTimeout` 지연으로 충분하다.
  - 정찰 미확인: 이 환경에서는 `npm`·`npm ci`·심볼릭 링크 생성이 모두 승인 차단되어 **웹 테스트를 직접 돌려 재현하지 못했다**. 근거는 전부 지난 두 회차의 `verify.txt` 로그와 저장소 파일 읽기다. 구현자는 먼저 `npm ci` 후 기준 2의 지연 스텁으로 red 를 **직접 확인**한 다음 고칠 것. 만약 지연 스텁으로도 red 가 나지 않고 다른 원인(예: `/admin/resources/banned-words` 요청 URL 이 바뀌어 스텁이 404 를 돌려주는 것)이 드러나면, 그 원인을 고치고 이 과제서의 진단을 버릴 것 — 로그의 DOM 덤프는 7000자 제한으로 헤더까지만 찍혀 본문이 로딩 중이었는지 빈 표였는지는 미확인이다.

- 차선 후보: 비스트리밍 chat 본문 절단 시 감사 `reason` 계약을 테스트로 고정 (2/1/S) — `relayChatBody`(`internal/server/providers.go:721-751`)가 `client_write_failed`/`upstream_read_failed` 를 구분하는 것을 `chat_truncation_integration_test.go` 셋업을 재사용해 `result=failure`·`details.reason=upstream_read_failed`·`complete=false` 로 고정한다(프로덕션 파일 0개). 단, 1순위가 성립하면 **1순위를 먼저** 할 것 — 웹 테스트가 깨진 채로는 어떤 회차도 릴리즈까지 가지 못한다.
