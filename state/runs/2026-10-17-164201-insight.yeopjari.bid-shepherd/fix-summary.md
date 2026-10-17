# 수리 결과: 고칠 결함 없음 — 같은 3초 잡을 다시 받았다 (커밋 없음)

- **이번에 받은 CI 는 새 실패가 아니라 15:36 의 그 잡 그대로다.** 과제서의 `runs/39010546706/job/117331978771` 은 15:43 회차가 이미 분석한 `ci-b55ab2de5485.json` 과 **동일한 check_run id** 이고, `started_at 06:36:37Z → completed_at 06:36:40Z`(3초), 단계 로그 0줄, `annotations_count: 2`(파일 아님, 잡 수준). **b55ab2d 를 다시 돌려본 CI 는 아직 없다** — 상태가 낡은 것이지 두 번 실패한 것이 아니다.
- **3초에는 이 변경이 닿는 단계가 한 줄도 돌지 않는다.** `npm ci` 만 로컬에서 12초, 그 앞에 `services.postgres` health(5s×20)가 있다. lint·typecheck·test·build 실패는 3초가 물리적으로 불가능하므로 원인은 러너 기동·컨테이너 초기화·Actions 분/결제 같은 저장소 인프라다.
- **b55ab2d 그대로 CI 의 코드 게이트를 다시 재현해 전부 통과했다:** `npm ci --no-audit --fund=false`(12초) · `npm run check`(eslint + core/server/web typecheck + vitest **7파일 113개 통과**) · `npm run build`(web + server dist). 트리 깨끗(`git status` 빈 출력), HEAD 는 b55ab2d 그대로.
- **테스트가 대상을 실제로 실행하는 것도 직접 확인했다:** `apps/web/src/api/client.ts` 만 `b55ab2d^` 로 되돌리니 4개 중 2개가 바로 그 이유로 실패(`expected +0 to be 1` = reportDegraded 미호출, `expected TypeError: fetch failed to be an instance of ApiError`), 복구하면 4/4 통과. 단언을 느슨하게 하거나 지운 곳 없다.
- **못 한 것·권고:** 이 역할에 GitHub 자격증명이 없어(`gh` 미인증, `git fetch` 인증 실패, 비공개 저장소라 공개 API 404) **annotation 2개 본문을 못 읽었다** — 위 원인은 타이밍·로그 부재·로컬 재현에서 나온 추론이다. 조치는 코드 수정이 아니라 **잡 재실행**이고, 재실행이 또 3초에 죽으면 Actions 분·결제 한도·러너 가용성을 봐야 한다. e2e·스크린샷은 15:43 회차가 격리 환경에서 180/0·85장으로 이미 통과시켰으므로 이번에 다시 돌리지 않았다.
