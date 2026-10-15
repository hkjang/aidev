# 수리 요약 — PR #7 (고치지 않음: 코드 결함이 아님)

- **CI 'check' 실패는 코드 때문이 아니다.** 잡이 **시작조차 못 했다**: `gh api .../jobs/116265124977` 가 `"steps":[]`, `started_at == created_at`, `runner_name:""`, 3초. 런 주석이 원인을 그대로 말한다 — *"The job was not started because recent account payments have failed or your spending limit needs to be increased."* 그래서 `--log-failed` 가 빈 출력이었다(과제서의 로그 칸이 비어 있던 이유).
- **브랜치 무관임을 교차 확인했다.** 2026-10-14 01:14 이후 런 5개(브랜치 5개, PR #4·#5·#6·#7)가 전부 같은 주석으로 실패하고, 마지막 성공은 그 이전 main(38486630885)이다. 다른 두 런(38670557933·38494638722)에서도 같은 billing 주석을 확인했다.
- **같은 검증을 로컬에서 돌려 브랜치가 초록임을 확인했다**(`npm ci` 후): `npm run check` → lint·typecheck(core·server·web) 무경고, **187 passed / 6 files**. `npm run build` 성공, `node scripts/perf.mjs` → 223.7 KB / 예산 230 KB 통과. CI 1~3단계와 동일하다.
- **고치지 않았고 커밋도 없다.** 규칙상 워크플로를 손대 통과시키는 것은 금지이고, 결함이 저장소 안에 없다. 필요한 조치는 GitHub 계정의 Billing & plans(결제 수단·지출 한도) 뿐 — 사람이 해야 한다.
- 돌리지 않은 것: `scripts/e2e.mjs`·`a11y.mjs`(공유 `yeopjari-review-dev` 컨테이너가 5시간째 떠 있어 `dev-reset.sh` 의 DROP DATABASE 가 동시 실행 중인 다른 회차를 깰 수 있다). diff 는 `text.test.ts` 237줄 추가뿐이고 프로덕션 코드 0줄이라 회귀 경로가 없다.
