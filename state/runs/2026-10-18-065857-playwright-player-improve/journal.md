# 회차 노트 2026-10-18-065857-playwright-player-improve — playwright-player
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:59] base pinned — main@0de9aba
- [러너 06:59] autonomy release — 
- [러너 06:59] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 추가한 파일은 `.github/workflows/ci.yml` 하나다(커밋 24fa79a, 프로덕션 코드 0개). `push:[main]` + `pull_request`, node 22 + `npm ci` + `npx playwright install --with-deps chromium`, 잡 둘: `check`(= `npm run check`, 게이트) / `test`(= `SMOKE_REQUIRE_BROWSER=1 npm test`, `continue-on-error` 로 보고 전용).
- 확신 없는 곳: `test` 잡을 게이트로 올리지 않은 판단. 이 base 에서 같은 명령을 3회 돌려 150/0/0 (195s) · 150/0/0 (193s) · **149 passed 1 failed (168s, `durationMs:-203`)** 였다 — 통과하지만 1/3 확률로 기존 결함이 터진다. "지금 통과하는 것만 옮겨라" 와 "러너가 돌리는 테스트 명령을 옮겨라" 가 충돌하는 지점이고, 다르게 판단할 여지가 있다. `continue-on-error` 는 실패가 잡 결론을 빨갛게 만들지 않으므로 '조용한 초록' 에 가깝다는 비판이 가능하다(승격 조건을 파일 주석에 명시해 완화).
- 검증 못 한 것: GitHub Actions 러너에서의 실제 실행. 여기서는 (a) CI 가 돌릴 네 명령을 node_modules 없는 워크트리에서 같은 순서로 직접 실행, (b) `actionlint 1.7.12` 통과 + 사본에 오류를 주입해 린터가 실제로 검사함을 확인, (c) `act 0.2.89` 플래너로 `pull_request`/`push` 가 두 잡을 고르고 `workflow_dispatch` 는 아무 잡도 고르지 않음을 확인했다. `act --dryrun` 은 이 샌드박스에서 러너 이미지 선택/네트워크로 멈춰 포기하고 `act -l` 로 대체했다. 호스티드 러너에서 `--with-deps` 의 apt 단계와 setup-node 캐시는 미검증이다.
- 일부러 하지 않은 것: 음수 duration 결함 수정(이번 지시가 소스·테스트 변경 금지), 6스위트를 잡 매트릭스로 쪼개기(러너 명령을 그대로 옮기라는 범위 밖), docker build 잡, chromium 캐시, CHANGELOG(이 저장소에 파일 자체가 없다). 모두 ideas.json 에 적었다.
- 다음 역할이 조심할 것: 이 워크트리에는 `npm ci` 와 chromium 이 이미 깔려 있다(`~/.cache/ms-playwright`) — 재검증은 `SMOKE_REQUIRE_BROWSER=1 npm test` 로 3분이면 된다. 다만 **1/3 확률로 음수 duration 때문에 빨갛게 나온다**; 그 한 건만 실패하면 이 회차의 변경이 아니라 기존 결함이다. `npm audit` 이 이제 critical 1건(proxy-addr)을 보고한다 — 이번 회차는 lockfile 을 건드리지 않았다.
- [러너 07:17] verify passed — 검증 2개 통과 (auto)
- [러너 07:17] pr created — https://github.com/hkjang/playwright-player/pull/2
- [러너 07:17] guard held — .github/workflows/ci.yml 
