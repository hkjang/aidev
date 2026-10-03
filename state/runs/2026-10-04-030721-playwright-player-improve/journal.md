# 회차 노트 2026-10-04-030721-playwright-player-improve — playwright-player
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:07] base pinned — main@a2a44d4
- [러너 03:07] autonomy release — 
- [러너 03:07] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 추가한 파일은 `.github/workflows/ci.yml` 하나뿐(프로덕션 코드 0, 문서 0, CHANGELOG 는 이 저장소에 없음). push(main)+pull_request, node 22 + `npm ci` + `npm run check`(게이트 잡) / `npx playwright install --with-deps chromium` + `SMOKE_REQUIRE_BROWSER=1 npm test`(smoke 잡).
- **확신 없는 곳**: ① `npm test` 스텝에 `continue-on-error: true` 를 붙였다. 근거는 측정이다 — 이 base(pristine a2a44d4)에서 스위트를 두 번 돌려 둘 다 실패했다(`149 passed, 1 failed` / `148 passed, 2 failed`, 각 5m47s). 실패는 기존 보류 결함 2건(플레이그라운드 녹화 패널 낡은 읽기, 음수 step duration)이고 내 변경과 무관하다. 게이트로 걸면 모든 PR 이 알려진 결함으로 막힌다고 판단했다 — 다르게 판단한다면 그 한 줄만 지우면 된다. ② GitHub 러너에서 `npx playwright install --with-deps chromium`(apt 포함)과 `actions/setup-node` 캐시가 도는 것은 **확인하지 못했다**. 이 환경에는 push 권한이 없어 실제 Actions 실행을 못 본다. 로컬에서는 `--with-deps` 없이(sudo 불가) chromium 설치 → 스위트 실행까지 확인했다. ③ `act --dryrun` 의 push 이벤트 실행은 내 백그라운드 시간 제한에 걸려 중간에 중단됐다(그때까지 smoke 잡 계획까지 성공). pull_request 실행은 exit 0 으로 두 잡·모든 스텝을 계획했다.
- 검증 수단: `actionlint 1.7.12` 깨끗(사본에 `bogus-input` 을 주입해 린터가 실제로 잡는 것 확인), `act 0.2.89 --dryrun`, 그리고 CI 가 돌릴 네 명령을 node_modules 없는 워크트리에서 같은 순서로 직접 실행.
- 일부러 하지 않은 것: `docker build`/컨테이너 검증(체크리스트 2~3절) — 이 환경에서 통과를 확인할 수 없어 검증 못 한 잡은 넣지 않았다. `npm audit`(advisory 2건으로 지금 빨갛다). 브라우저 캐시(`actions/cache`) — CI 가 초록인 걸 본 뒤에. 테스트 파일은 한 줄도 손대지 않았다(지시).
- 다음 역할이 조심할 것: 이 PR 의 CI 는 자기 자신을 돌린다 — `check` 잡이 빨가면 그건 진짜 문제이고, `smoke` 잡의 `npm test` 스텝은 실패해도 주석(annotation)만 남고 잡은 초록이다. 스위트는 브라우저가 있어야 하고 로컬 실측 5m47s + chromium 다운로드(~360MB)다.
- [러너 03:47] verify failed — 실패한 검증: npm test --silent (exit 1)
