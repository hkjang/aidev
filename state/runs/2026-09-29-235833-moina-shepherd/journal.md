# PR 처리기 노트 2026-09-29-235833-moina-shepherd — moina PR #37
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-29-201327-moina-improve)
# 회차 노트 2026-09-29-201327-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:13] base pinned — main@1d13889
- [러너 20:13] autonomy release — 

## 정찰 노트
- AuthContext 플레이크를 골랐다: 지난 회차 이미지 빌드를 실제로 빨갛게 만든 유일한 실패이고, 소스에서 원인 구조(`AuthContext.tsx:76`의 passive effect가 만드는 값을 `AuthContext.test.tsx:91` findByText 직후 즉시 읽음)를 눈으로 확인했으며 프로덕션 파일 0개·테스트 한 줄로 끝난다.
- 제친 후보: 접근성 catch 덮어쓰기(실제 Axe 실패 재현에 이미지+Playwright가 필요해 45분 초과 위험), 972113f 복원(이전 verify-failed가 image 잡이었고 같은 곳에서 또 걸릴 수 있음), Go 실패 요약(가치는 있으나 진단 출력이라 후순위 — 차선으로 남김).
- 추측으로 남긴 것: flake를 실제로 재현하지 못했다(worktree에 `frontend/node_modules`가 없어 `npm ci` 비용을 쓰지 않음). 실패 문구·줄 번호는 원장과 소스로 일치 확인했고, 인과는 구현자가 과제서 1)의 결정론적 "되돌려 검증"으로 증명해야 한다.
- 구현자 주의: effect를 늦추는 임시 변경을 반드시 되돌릴 것(최종 diff는 테스트 파일 1개). 재현을 위한 수백 회 반복 실행 금지 — 과잉 검증이 지난 TIMEOUT의 원인이었다.
- 프로필은 0일 전 것이 현재 코드와 맞아 새로 쓰지 않았다.
- [러너 20:16] scout done — `AuthContext.test.tsx:92`가 passive effect 완료를 기다리지 않아 간헐 실패(`expected true to be false`)하는 것을 고친다 

## 구현 노트
- `AuthContext.test.tsx`만 고쳤습니다(프로덕션 0개, 98529c0). 원인은 effect 자체가 아니라 RTL `asyncWrapper`가 주는 **`setTimeout(0)` 한 틱**과 React Scheduler의 `setImmediate` 사이 작업원 경합입니다 — MutationObserver로 DOM 변경 순간 깃발이 아직 `true`임을 직접 관측해 확정했습니다.
- 과제서의 증명법(effect를 한 tick 늦추면 :92만 실패)은 **실행으로 반증**됐습니다(:92 통과, :77이 깨짐). 과제서의 ":77은 안전" 판단도 틀려서 :77도 함께 고쳤습니다 — 범위를 한 줄 넘었지만 같은 결함의 나머지 절반입니다.
- 확신 없는 것: 실제 CI 실패 빈도는 여전히 모릅니다. 원래 flake를 자연 조건에서 재현하지는 못했고(과잉 반복 금지 지침에 따라 시도하지 않음), `setImmediate` 5ms 지연 probe로 **동일한 오류 메시지**를 결정론적으로 만든 것이 증거입니다. 이 probe는 자연 경합보다 강해서 act/barrier로 못 막는 경우도 있었습니다 — 즉 :77 수정의 필요성은 probe 기준으로는 확실하나 자연 조건에서의 발생 빈도는 미측정입니다.
- 일부러 안 한 것: :48·:57·:66 등 다른 단언은 근거가 없어 두었습니다. 백엔드·e2e·`ci.yml`은 무변경이라 `make check`만 돌렸습니다(백엔드 테스트·DB 미실행).
- 다음 역할이 조심할 것: 임시 probe 파일 3개(`__probe.test.tsx`, `__red.test.tsx`, `src/test/__slow-scheduler.ts`)와 `__probe.vitest.config.ts`는 전부 삭제했고 `git status`로 확인했습니다. 이 테스트는 DB 불필요, `frontend/npm ci`만 있으면 돕니다.
- [러너 20:26] brief accepted — 채택 — 과제서가 지목한 파일·줄·수정 방향(:92를 `waitFor`로)은 옳았고 그대로 구현했습니다. 다만 과제서의 **증명 기법
- [러너 20:26] verify passed — 검증 7개 통과 (auto)
- [러너 20:26] pr created — https://github.com/hkjang/moina/pull/37
- [러너 20:26] guard held — frontend/src/auth/AuthContext.test.tsx 
