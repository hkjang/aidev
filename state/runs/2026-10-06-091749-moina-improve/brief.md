- 과제: (수정 과제) 릴리즈 CI를 두 번 깨뜨린 유실 커밋 `98529c0` 복원 — `AuthContext.test.tsx`의 silent SSO 해제 단언이 passive effect를 기다리지 않아 image 잡의 `npm test`가 `expected true to be false`로 실패 (가치 5 / 위험 1 / 작업량 S)
- 왜: 지난 회차 CI는 `Image and browser smoke` 잡의 `Build linux/amd64 service image` 단계에서 `src/auth/AuthContext.test.tsx (4 tests | 1 failed)` → `AssertionError: expected true to be false // Object.is equality` → `Tests 1 failed | 249 passed (250)`로 죽었습니다(`ci-failure-238eff15f11b.txt`). 2026-09-29 회차가 이 결함을 이미 진단·수정했지만(`98529c0`) **그 커밋이 main 에 들어오지 못했습니다** — `git branch -a --contains 98529c0` 가 `auto/2026-09-29-2013` 와 그 origin 복사본만 내놓고 main 을 내놓지 않는 것으로 확인했습니다. 즉 워크플로가 느슨해서가 아니라 고친 테스트가 유실돼 같은 경합이 그대로 남아 CI 가 계속 깨집니다.

- 정찰이 **실행으로 확인한 것**(추측이 아님):
  - `git log --oneline --all -- frontend/src/auth/AuthContext.test.tsx` → `98529c0`, `09cd6a0` 두 커밋뿐.
  - `git branch -a --contains 98529c0` → `auto/2026-09-29-2013`, `remotes/origin/auto/2026-09-29-2013` **둘뿐 = main 에 없음**.
  - `git diff 98529c0^ HEAD -- frontend/src/auth/AuthContext.test.tsx` → **빈 출력**. 지금 main 의 테스트 파일은 수정 **이전** 상태와 완전히 동일하므로 `git checkout 98529c0 -- <파일>` 이 다른 수정을 되돌릴 위험이 없습니다.
  - `git show 98529c0 --stat` → `1 file changed, 7 insertions(+), 1 deletion(-)`, 테스트 파일 1개뿐(프로덕션 0개).
  - 현재 파일을 눈으로 확인: `:77`·`:92` 가 `waitFor` 없는 맨 `expect(silentSsoAttempted()).toBe(...)` 이고, 케이스 시작부(`:70~72`)에 `markSignedOut()` 이 없습니다.
  - 프로덕션 쪽 근거: `frontend/src/auth/AuthContext.tsx:76` 이 `useEffect(() => { if (user) clearSilentSsoState(); }, [user]);` — 깃발 해제가 **passive effect** 라 `findByText('user:mina')` 가 풀린 **뒤에** 내려갑니다. `markSignedOut()` 은 `:100` 의 `logout` 안에서 동기로 섭니다.
  - 실패 지점의 명령: `Dockerfile:13` 의 `RUN npm test && VITE_MOINA_VERSION="${VERSION}" npm run build`, `Makefile:62` 의 `image` 타깃(`docker build`)이 그것을 호출하고 `.github/workflows/ci.yml:113~114` 의 `Build linux/amd64 service image` 단계가 `make image` 를 돕니다.
  - 왜 `Source tests` 잡은 통과했는가: 같은 `ci.yml:72` 가 `npm test --prefix frontend` 를 돌지만 CI 는 image 잡에서만 깨졌습니다 — 부하 의존 경합이라는 2026-09-29 진단과 일치합니다(docker 빌드 레이어가 동시에 돌아 타이머가 먼저 이깁니다).
  - **미확인**: 이 worktree 에 `frontend/node_modules` 가 없고 `npm ci` 가 이 세션에서 승인되지 않아(네트워크) **정찰은 테스트를 한 번도 돌리지 못했습니다**. red 재현과 green 확인은 전부 구현자 몫입니다. `gh` 도 승인되지 않아 PR #42 의 CI 를 직접 조회하지 못했고 근거는 러너가 적재한 `ci-failure-238eff15f11b.txt` 입니다. "두 번 실패" 중 두 번째 run 의 로그는 보지 못했습니다(같은 원인이라고 러너가 적어 둔 것만 확인).
- 수용 기준:
  1) `frontend/src/auth/AuthContext.test.tsx` 가 `98529c0` 의 버전과 **byte-identical** 이다 — `git diff 98529c0 -- frontend/src/auth/AuthContext.test.tsx` 가 빈 출력.
  2) 프로덕션 파일 변경 **0개** — `git diff --stat` 이 테스트 파일 1개만 보여야 하고, `git diff --stat` 이 `98529c0` 의 stat(`1 file changed, 7 insertions(+), 1 deletion(-)`)과 일치한다.
  3) `cd frontend && npm test` 가 **36 파일 250 테스트 전부 통과**(exit 0). `npm run lint` 가 0 errors / 경고 상한 40 이내(직전 39), `npm run build` exit 0.
  4) **계약 미완화**를 출력으로 증명: `it.skip`·`test.retry`·fake timer·단언 삭제·`toBe(false)` → `toBeFalsy()` 류의 완화가 diff 에 **없다**(복원 diff 는 `markSignedOut()` 한 줄 + `waitFor` 두 곳 + 주석 4줄뿐). 워크플로 파일(`.github/workflows/*.yml`)·`Dockerfile`·`Makefile` 은 **손대지 않는다**.
  5) **되돌려 검증**: `frontend/src/auth/AuthContext.tsx` 의 `clearSilentSsoState()` 호출을 잠시 지우면 복원한 테스트가 **여전히 실패**해야 한다(= `waitFor` 가 계약을 가리지 않았다는 증거). 확인 후 복원하고 `git diff` 로 되돌림을 확인한다. (2026-09-29 회차가 같은 방법으로 확인했다고 원장에 적혀 있다 — 미재현.)
  6) CI 가 재현하는 경로 그대로인 **image 빌드**까지 확인: `make image` 가 exit 0(느림, 수 분). 이것이 실패했던 바로 그 명령(`Makefile:64`, 컨테이너 안에서 `npm test && npm run build`)이다.
- 건드릴 파일:
  - `frontend/src/auth/AuthContext.test.tsx` — `98529c0` 의 두 hunk를 복원. (가) `'스스로 로그아웃한 뒤에는 …'` 케이스(현재 :69) 시작부에 `markSignedOut();` 과 주석 4줄을 넣고 `await screen.findByText('user:mina');`(현재 :73) 뒤에 `await waitFor(() => expect(silentSsoAttempted()).toBe(false));` 장벽을 추가. (나) 같은 케이스 마지막 줄(현재 :92) `expect(silentSsoAttempted()).toBe(false);` 를 `await waitFor(() => expect(silentSsoAttempted()).toBe(false));` 로 교체. `markSignedOut`·`waitFor` 는 이미 :1·:4 에서 import 돼 있어 import 변경은 없다.
  - **그 외 파일 없음.** 프로덕션 코드 파일 0개, 워크플로 0개.
  - 권장 절차: `git checkout 98529c0 -- frontend/src/auth/AuthContext.test.tsx` 한 줄이면 끝난다(아래 "위험" 의 사전 확인 뒤). 수동 편집보다 byte-identical 보장이 강하다.
- 검증 명령:
  - `cd frontend && npm ci && npm run lint && npm test && npm run build`
  - `git diff 98529c0 -- frontend/src/auth/AuthContext.test.tsx` (빈 출력이어야 함) · `git diff --stat`
  - `make check` (OpenAPI route 120개 유지 확인. 이 diff 와 무관하지만 관례)
  - `make image` (CI 의 실패 단계 재현. 느림)
  - 백엔드는 이 diff 와 무관하므로 `cd backend && go build ./... && go vet ./...` 정도로 충분하고 DB integration 전체 재실행은 불필요하다(응답·Go 코드 무변경).
- 위험과 피할 것:
  - **워크플로를 느슨하게 만들지 말 것.** `.github/workflows/ci.yml`·`release.yml`, `Dockerfile`, `Makefile:64`(`image` 타깃)은 보호 경로다. 재시도 추가·`continue-on-error`·`npm test` 제거·`--retry` 는 전부 금지다.
  - **테스트를 약하게 만들지 말 것.** `toBe(false)` 유지가 수용 기준이다. `waitFor` 는 단언을 지우는 것이 아니라 effect 완료를 기다리는 장벽이므로 계약 완화가 아니다 — 수용 기준 5)의 되돌려 검증이 그것을 증명한다.
  - `git checkout 98529c0 -- <파일>` 전에 **그 커밋의 부모와 현재 파일이 같은지** 확인할 것: `git diff 98529c0^ HEAD -- frontend/src/auth/AuthContext.test.tsx` 가 빈 출력이어야 복원이 다른 수정을 되돌리지 않는다. (정찰이 이것을 확인했다 — 아래 "확인한 것" 참고.) 비어 있지 않으면 두 hunk 를 수동 적용하고 `98529c0` 와의 diff 로 결과를 확인한다.
  - 같은 저장소의 교훈: `followTopic`(972113f)·e2e 요약(ab36254) 복원 회차와 동일한 유형이다. 둘 다 "원본과 byte-identical" 을 기준으로 삼아 통과했다.
  - **프런트 테스트는 반드시 실제로 돌릴 것.** 이 결함은 부하 의존 경합이라 `grep` 으로는 증명되지 않는다(운영자가 "소스 문자열 검사를 증거로 쓰지 말라" 를 반복했다). 1회 통과만으로는 경합 제거를 증명하기 어려우니 `npm test` 를 2~3회 돌려 250/250 이 반복되는지 보는 것이 좋다. 다만 2026-09-29 원장에 **결정론적 재현법**이 남아 있다: React Scheduler 의 `setImmediate` 만 5ms 늦추는 **테스트 전용 setup 파일**(프로덕션 무변경)을 얹으면 복원 전에는 깨지고 복원 후에는 통과했다. 그 probe 로 red→green 을 관측하는 것이 가장 강한 증거이며, probe 파일은 커밋 전에 반드시 지우고 `git status` 로 확인해야 한다(지난 회차도 그렇게 했다).
  - 로그인 5분/5회 레이트리밋은 백엔드 integration 얘기라 이번 과제와 무관하다.
- 차선 후보: `admin.go` 의 남은 `storage_error` 500 출구 25곳을 `writeStorageError` 로 이관 (가치 3 / 위험 1 / 작업량 M) — `admin_report_resolve_postgres_integration_test.go` 에 이미 로그 단언 배선이 있어 red→green 수단이 갖춰져 있다. 단, **CI 가 깨진 채로는 어떤 기능 과제도 릴리즈까지 못 간다**. 1순위가 성립하지 않는 경우(= `98529c0` 복원이 현재 파일과 충돌)에만 고르고, 그때도 CI 실패 원인을 먼저 보고할 것.
