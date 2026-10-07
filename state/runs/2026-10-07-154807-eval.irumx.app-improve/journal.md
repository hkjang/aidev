## 구현 노트

- 무엇·왜: `tests/api-targets.spec.ts` 에 '예산 동시 예약' 시험 1개 추가 — `reserveSql` 의 조건부 INSERT 하나(활성 예약+정산 ≤ 한도)가 동시 예약을 막는다는 설계를 고정한다. 프로덕션 로직 변경 0개(`lib/util.ts` 주석 1줄에서 없는 파일 `tests/guard.spec.ts` 참조만 실제 경로로 수정).
- **과제서의 `api.raw` + `Promise.all` 은 동시성을 전혀 밟지 못한다** — Playwright 요청 컨텍스트가 요청을 하나씩 보낸다. 회귀를 심어도 201 이 정확히 2개 나와 통과했다. `api.ctx.storageState()` 의 쿠키를 꺼내 Node 전역 `fetch` 5개로 보내야 재현된다. 다음 역할이 이 줄을 `api.raw` 로 '정리' 하면 시험이 조용히 무의미해진다.
- **검증 함정(세 번 헛돌았다)**: `wrangler.jsonc` 의 main 은 `src/worker/index.ts` 지만 `@cloudflare/vite-plugin` 이 `.wrangler/deploy/config.json` → `dist/irumx_eval/wrangler.json` redirect 를 써 둔다. 즉 **worker 소스를 고쳐도 `vite build` 를 다시 하지 않으면 시험은 옛 코드를 돈다.**
- **빨강 증거**: `reserveSql` 의 시험 수 조건을 빼고 같은 검사를 JS 로 먼저 읽게(SELECT 후 INSERT) 바꾸니 새 시험만 `201,201,201,201,201 / Expected: 2` 로 실패하고, **기존 순차 예산 시험(158줄)은 통과**했다. 조건을 통째로 뺀 변형에서는 둘 다 실패했다. 변형은 모두 `git checkout` 으로 되돌렸다(`git status` 로 확인).
- 확신 없는 곳: 201 이 2개라는 경계는 `est.trials`(estimate 결과, 실측 3) × 2 + 1 한도에 달려 있다 — 사례 3개 중 하나라도 묶음에서 빠지면 수가 달라진다. 절대 수를 쓰지 않았으니 깨지지는 않지만 `STARTER` 를 바꾸면 다시 볼 것.
- 일부러 안 한 것: `reserveSql`·`changedGuardSql` SQL 은 손대지 않았다(현 동작을 고정하는 시험이다). CI 워크플로 추가(보호 경로·운영자 판단), 비밀 가림 좁히기(차선 후보)는 `ideas.json` 에 남겼다.
- 다음 역할이 조심할 것: 이 시험은 D1·wrangler dev 가 떠 있어야 돈다. `db-local.mjs` 가 `.wrangler/test-sessions` 를 지우지 않아 같은 트리에서 두 번 돌리면 세션 캐시 때문에 `me.workspaces` 가 undefined 로 깨진다 — 재현 전에 `rm -rf .wrangler/test-sessions`. `npm run build && npm test` 결과: 빌드 점검 통과, unit+api 47 passed. desktop/mobile 4건은 이 환경에 Chromium 이 없어서 실패했고(내 변경과 무관) 따로 설치해 다시 돌렸다.
- [러너 16:05] brief accepted — 채택 — 과제서의 근거(reserveSql 조건부 INSERT · 시험 공백 · estimate/usage 경로)는 코드와 그대로 맞았다. 다만 과제서가 적�
- [러너 16:06] verify failed — 실패한 검증: npm test --silent (exit 1)
