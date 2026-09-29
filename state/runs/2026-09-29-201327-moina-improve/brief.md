- 과제: `AuthContext.test.tsx:92`가 passive effect 완료를 기다리지 않아 간헐 실패(`expected true to be false`)하는 것을 고친다 (가치 4 / 위험 1 / 작업량 S)
- 왜: 지난 회차의 `make image` 첫 실행에서 `frontend/src/auth/AuthContext.test.tsx:92`가 `expected true to be false`로 249/250 실패했고 같은 조건 재실행은 250/250 통과했습니다(원장 기록, 원인 미확정). 이 단언이 읽는 값은 프로덕션의 **커밋 이후에 흐르는 passive effect** `useEffect(() => { if (user) clearSilentSsoState(); }, [user])`(`frontend/src/auth/AuthContext.tsx:76`)가 만들고, 테스트는 그것을 `await screen.findByText('user:mina')`(:91) 직후 **즉시** 읽습니다 — DOM 변화(findBy의 MutationObserver)와 effect flush 사이에 순서 보장이 없으므로 이 한 줄만 타이밍에 걸립니다. 고치면 무관한 프런트 flaky 때문에 CI·이미지 빌드가 빨개지는 일이 사라집니다.
- 수용 기준:
  1) **원인을 먼저 결정론적으로 증명한다.** 흔들리는 재현에 매달리지 말고 "되돌려 검증"(2026-09-28 회차에서 쓴 기법)을 쓰세요: 임시로 `AuthContext.tsx:76`의 effect를 `useEffect(() => { if (user) setTimeout(() => clearSilentSsoState(), 0); }, [user])`처럼 한 tick 늦춘 뒤 `vitest run src/auth/AuthContext.test.tsx`를 돌려 **:92만** `expected true to be false`로 실패하는 것을 출력으로 남깁니다(= 단언이 effect 완료 시점에 의존한다는 증명). 이 임시 변경은 반드시 되돌립니다.
  2) 테스트 한 줄을 `await waitFor(() => expect(silentSsoAttempted()).toBe(false));`로 바꾼 뒤, 1)의 임시 지연을 다시 넣은 상태에서 통과하고 지연을 되돌린 상태에서도 통과하는 것을 보입니다. 계약을 **완화하지 않습니다**: `toBe(false)` 그대로, `it.skip`·`test.retry`·단언 삭제·`vi.useFakeTimers` 금지.
  3) 프로덕션 파일 변경 0개 — 최종 diff에 `AuthContext.tsx`·`silentSso.ts`가 남으면 안 됩니다(`git diff --stat`으로 확인).
  4) 전체 프런트가 그대로 통과: lint 0 errors(경고 상한 40, 직전 39), vitest 36파일 250테스트 전부 통과, build exit 0.
- 건드릴 파일:
  - `frontend/src/auth/AuthContext.test.tsx:92` — 마지막 `it('스스로 로그아웃한 뒤에는 시도하지 않고, 세션이 다시 생기면 억제가 풀린다')`의 `expect(silentSsoAttempted()).toBe(false)`를 `waitFor`로 감쌈. `waitFor`는 이미 이 파일 1행에서 import 중입니다.
  - (선택) 같은 파일 :48 `expect(silentSsoAttempted()).toBe(true)`와 :77은 `beginSilentSso`/`markSignedOut`이 **동기로** 쓰는 값이라 같은 위험이 없습니다 — 근거 없이 함께 바꾸지 마세요.
- 검증 명령 (worktree에 `frontend/node_modules`가 없으므로 `npm ci` 선행):
  - `cd frontend && npm ci`
  - `npx vitest run src/auth/AuthContext.test.tsx` (1·2단계 red/green용)
  - `npm run lint && npm test && npm run build`
  - 백엔드 무변경이면 `make check`만 확인(백엔드 테스트·DB 불필요).
- 위험과 피할 것:
  - 프로덕션 `AuthContext.tsx`/`silentSso.ts`를 "고치는" 방향으로 번지지 말 것(계약은 옳고 문제는 단언 시점입니다). 인증 경로는 보호 구역입니다.
  - flaky 재현을 위해 같은 테스트를 수십~수백 회 돌리는 데 시간을 쓰지 마세요 — 2026-09-28 회차가 과잉 검증(39회)으로 TIMEOUT 됐습니다. 1)의 결정론적 증명이면 충분합니다.
  - e2e·시각 베이스라인·`ci.yml`·백엔드는 이번 범위가 아닙니다.
  - 미확인: 실제 CI에서의 실패 빈도와 vitest 반복 옵션 이름은 확인하지 않았습니다. `--repeats` 사용은 선택 사항이며 필수 아닙니다.
- 차선 후보: CI Go 테스트 실패(`Go formatting, tests and vet`) 요약을 `$GITHUB_STEP_SUMMARY`에 적기 (3/2/S) — 지난 회차가 e2e 요약을 복원했고 Go 단계는 여전히 annotation "exit code 1"만 남깁니다. `ci.yml` 한 곳 + 필요 시 스크립트 1개, 실제 실패 출력으로 tee/pipefail 동작을 확인할 것.
