# 회차 노트 2026-10-06-091749-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:17] base pinned — main@c9bfca2
- [러너 09:17] autonomy release — 

## 정찰 노트
- 우선 과제가 주어졌으므로 새 아이디어를 고르지 않았다. CI 실패의 원인은 워크플로가 아니라 **main 에 유실된 커밋 `98529c0`** 이다 — 2026-09-29 회차가 `AuthContext.test.tsx` 의 passive-effect 경합을 이미 고쳤는데 `git branch -a --contains 98529c0` 가 main 을 내놓지 않는다. 프로필이 "유실 커밋 복원 캠페인은 끝났다" 고 단정한 것이 이 미스의 뿌리라서 프로필을 고쳐 썼다.
- 복원이 안전하다는 근거는 실행으로 확인했다: `git diff 98529c0^ HEAD -- <파일>` 이 빈 출력이라 지금 main 은 수정 이전 상태와 동일하고, `git checkout 98529c0 -- <파일>` 한 줄이 다른 수정을 되돌리지 않는다. 테스트 파일 1개·프로덕션 0개.
- **확신 없는 곳**: 정찰은 테스트를 한 번도 돌리지 못했다 — `frontend/node_modules` 가 없고 이 세션에서 `npm ci`(네트워크)가 승인되지 않았다. `gh` 도 승인되지 않아 PR #42 CI 를 직접 못 봤고 근거는 러너가 적재한 `ci-failure-238eff15f11b.txt` 뿐이다("두 번 실패" 중 둘째 run 로그는 미확인). 되돌려 검증(`clearSilentSsoState()` 제거 시 실패)은 원장의 기록이고 이번에 재현하지 않았다.
- **구현자가 조심할 것**: 워크플로·Dockerfile·Makefile 을 손대 통과시키는 것은 금지다. `toBe(false)` 를 유지하고 `it.skip`·retry·fake timer 를 쓰지 말 것. 부하 의존 경합이라 `npm test` 1회 통과는 약한 증거다 — 원장에 남은 Scheduler `setImmediate` 5ms 지연 probe(테스트 전용 setup, 프로덕션 무변경)로 red→green 을 관측하고 probe 파일은 커밋 전에 지울 것.
- 최근 기각·차선 반복 유형(워크플로 건드리는 과제, 대역 테스트로 때우는 증명)은 피했다. 차선은 `admin.go` 25곳 이관이지만 CI 가 깨진 채로는 어떤 기능 과제도 릴리즈까지 가지 못하므로 1순위를 양보시켰다.
- [러너 09:24] scout done — (수정 과제) 릴리즈 CI를 두 번 깨뜨린 유실 커밋 `98529c0` 복원 — `AuthContext.test.tsx`의 silent SSO 해제 단언이 p

## 구현 노트
- 유실 커밋 `98529c0` 을 `git checkout 98529c0 -- frontend/src/auth/AuthContext.test.tsx` 한 줄로 복원했습니다(7b0f70d). `98529c0` 과 byte-identical, 프로덕션·워크플로·Dockerfile·Makefile 0개, 테스트 파일 1개뿐. 원인은 `AuthContext.tsx:76` 의 passive effect 해제가 `findByText` 보다 늦게 내려가고 그 사이를 RTL `asyncWrapper` 의 `setTimeout(0)` 한 틱만 메워 주던 것입니다.
- **확신 없는 곳·검증 못 한 것**: (가) PR #42 의 CI 를 직접 조회하지 못했습니다 — `gh` 미인증이라 근거는 러너가 적재한 `ci-failure-238eff15f11b.txt` 뿐이고 "두 번 실패" 중 둘째 run 로그는 못 봤습니다. (나) 로컬 `make image` 는 통과했지만 **원격 CI 러너의 부하 조건은 재현한 것이 아니라 probe 로 모형화**한 것입니다 — 경합이 사라졌다는 증명이 아니라 관측된 두 실패 모드가 장벽으로 막혔다는 증명입니다. (다) e2e·`make package`/`verify-package`·백엔드 DB integration 은 돌리지 않았습니다(이 diff 는 프런트 테스트 파일 1개라 무관, 백엔드는 `go build`·`go vet` 만).
- **일부러 하지 않은 것**: 워크플로·`Dockerfile`·`Makefile` 을 손대 통과시키는 것(보호 경로, 금지). `admin.go` 차선 후보(범위 밖). 같은 경합이 있을 수 있는 나머지 35개 테스트 파일 점검 — 이번 범위를 넘고 파일 수가 불어나므로 `ideas.json` 에 다음 회차 후보(4/1/M)로 적었습니다.
- **다음 역할이 조심할 것**: diff 는 `markSignedOut()` 한 줄 + `waitFor` 두 곳 + 주석 4줄뿐이고 `toBe(false)` 가 두 곳 모두 그대로입니다 — `waitFor` 는 단언을 지운 것이 아니라 effect 완료를 기다리는 장벽이고, `AuthContext.tsx` 의 `clearSilentSsoState()` 를 지우면 복원한 테스트가 정상 config 에서도 여전히 실패하는 것으로 확인했습니다(되돌려 검증 후 복원). 프런트 테스트에는 `frontend/node_modules` 가 필요하고 이 세션에서는 `npm ci` 가 승인됐습니다. probe 파일 3개(`src/test/__probe-setup.ts`·`src/auth/__red.test.tsx`·`vitest.probe.config.ts`)는 커밋 전에 지웠고 `git status --short` 가 테스트 파일 1개만 내놓는 것을 확인했습니다. vitest CLI 에는 `--setupFiles` 옵션이 없어 probe 는 `mergeConfig` 하는 별도 config 로 걸어야 합니다.
- [러너 09:30] brief accepted — 채택 — 과제서가 실행으로 확인했다고 적은 다섯 가지를 모두 재확인했고 전부 참이었습니다(`git branch -a --contains 98529c0`
- [러너 09:30] verify passed — 검증 7개 통과 (auto)
- [러너 09:30] pr created — https://github.com/hkjang/moina/pull/43
- [러너 09:30] guard held — frontend/src/auth/AuthContext.test.tsx 
