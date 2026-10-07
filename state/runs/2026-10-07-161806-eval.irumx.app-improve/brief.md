- 과제: 시험 세션 캐시와 로컬 D1 비우기가 어긋나 두 번째 실행이 깨지는 것 고치기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `scripts/db-local.mjs` 는 `.wrangler/test-state`(로컬 D1)만 지우고 `tests/helpers.ts:71` 이 쓰는 `.wrangler/test-sessions/*.json`(로그인 쿠키 보관)은 남긴다. `src/worker/auth.ts:38` 의 `cookieCache: { enabled: true, maxAge: 5 * 60 }` 때문에 DB 를 비운 직후에도 남은 쿠키로 `/api/me` 가 사람을 돌려주고(세션 행 조회 없이 서명 쿠키에서) `workspaces` 만 빈 배열이 되어, `ownerApi`(`tests/helpers.ts:92-95`)의 `me.workspaces[0].id` 가 `Cannot read properties of undefined (reading 'id')` 로 터진다 — 같은 트리에서 5분 안에 시험을 다시 돌리면 거의 전부 깨지고, 원인이 코드 변경인지 환경인지 구분이 안 된다. 고치면 회차가 환경 때문에 헛돌지 않고, 터져도 메시지만 보고 원인을 안다.
- 수용 기준:
  1) `node scripts/db-local.mjs` 를 돌린 뒤 `.wrangler/test-sessions` 가 남아 있지 않다(또는 비어 있다).
  2) 같은 트리에서 전체 시험을 끝내고 떠 있는 `wrangler dev`(:8860)를 끈 뒤 **5분 안에** 다시 돌려도 `ownerApi` 단계에서 깨지지 않는다(고치기 전에는 깨진다 — 아래 재현 절차로 먼저 깨지는 것을 보고 나서 고칠 것).
  3) 세션 캐시만 남고 DB 가 빈 상태를 억지로 만들었을 때, `login(..., { cache: true })` 가 캐시를 버리고 새로 로그인한다(터지지 않는다). 터질 수밖에 없는 자리에서는 `Cannot read properties of undefined` 대신 "DB 를 비웠는데 세션 캐시가 남았다 — `.wrangler/test-sessions` 를 지우세요" 같은 한국어 메시지가 나온다.
- 건드릴 파일 (프로덕션 코드 0개, 시험 보조만 2개):
  - `scripts/db-local.mjs` — `rmSync(join(root, persist), …)` 바로 옆에 `.wrangler/test-sessions` 도 함께 지우는 줄 하나(경로는 `--persist-to` 와 무관하게 고정값 `.wrangler/test-sessions`. `tests/helpers.ts` 가 그 고정 경로를 쓰므로 거기서 상수를 꺼내 공유하려 하지 말고 각자 두는 쪽이 안전하다 — mjs 스크립트가 ts 를 import 하지 않는다).
  - `tests/helpers.ts:login` — 캐시 재사용 조건을 강화. 지금은 `me.user?.email === email` 만 본다. 쿠키 캐시가 응답하는 동안에도 통과하므로, "세션이 DB 에 실제로 있다"는 신호를 같이 봐야 한다. `me.workspaces?.length || me.operator` 가 참일 때만 캐시를 쓰고, 아니면 `ctx.dispose()` 후 새로 로그인(기존 fallback 경로 그대로 타게). `ops@eval.test` 는 업체 공간이 없고 `operator: true` 라서(`src/worker/routes/me.ts:19`) 이 조건에 걸리지 않는다 — 셋 다(`OWNER`·`OTHER`·`OPERATOR`) 정상 동작하는지 확인할 것.
  - `tests/helpers.ts:ownerApi` — `me.workspaces[0]` 가 없으면 위의 한국어 메시지로 `throw` (수용 기준 3의 뒷부분).
- 검증 명령 (이 저장소에서 실제로 도는 것):
  1) 재현(고치기 전): `npm run build` → `npx playwright test --project=api --grep "예산"` (끝까지 통과, `.wrangler/test-sessions/owner_eval_test.json` 생성 확인) → `pkill -f "wrangler dev"` → **5분 안에** 같은 명령 재실행 → `ownerApi` 에서 `reading 'id'` 로 깨지는 것을 로그에 남길 것. (`playwright.config.ts:26` 의 webServer 가 `node scripts/db-local.mjs --persist-to .wrangler/test-state && npx wrangler dev …` 라서 서버를 끄고 다시 돌릴 때만 DB 가 비워진다. `reuseExistingServer: true` 이므로 서버가 떠 있으면 재현되지 않는다 — 반드시 끄고 할 것.)
  2) 고친 뒤 같은 절차 반복 → 두 번째 실행도 통과.
  3) 전체: `npm run build && npm test` (수 분 걸린다).
- 위험과 피할 것:
  - **프로덕션 코드를 고치지 말 것.** 특히 `src/worker/auth.ts` 의 `cookieCache` 를 끄는 식으로 고치면 안 된다 — 운영 성능 설정이고, 시험 환경 문제를 운영 설정 변경으로 바꾸는 꼴이다. `migrations/`·`src/worker/security/*`·`src/worker/services/usage.ts` 는 이번 과제와 무관하니 열지 말 것.
  - 세션 캐시 자체를 없애지 말 것. 캐시는 로그인 코드 "15분에 5번" 제한을 피하려고 일부러 있다(`tests/helpers.ts:70` 주석). 캐시를 지우는 시점은 **DB 를 비우는 그 순간뿐**이어야 한다.
  - `wrangler dev` 는 `dist/` 의 빌드 결과를 띄운다(`@cloudflare/vite-plugin` 이 `.wrangler/deploy/config.json` 에 redirect 를 쓴다 — 이 부분은 지난 회차 기록에 따른 것으로 **이번 정찰에서 직접 확인하지 않음**). worker 소스를 고쳤으면 `npm run build` 를 다시 하고 서버를 재시작해야 반영된다. 이번 과제는 worker 소스를 건드리지 않으니 영향은 작지만, 재현 절차 중에 헷갈리기 쉽다.
  - 운영자 교훈: 변경이 최종 결과까지 실제로 바뀌는지 end-to-end 로 볼 것 — 여기서는 "두 번째 실행이 통과한다" 는 실제 실행 기록이 증거다. 소스 문자열 검사(`rmSync` 가 있는지)만으로 끝내지 말 것.
  - 설계 수용 시험 12번(예산 동시 예약) 은 **고르지 말 것** — 같은 과제로 두 회차(error · verify-failed)를 썼다. 같은 접근을 세 번째로 올리지 않는다.
- 차선 후보: `wrangler dev` 가 `dist/` 빌드 결과를 띄운다는 점을 문서에 한 줄 적기 (가치 2 / 위험 1 / 작업량 S) — `docs/architecture.md` 의 개발/시험 절 또는 `playwright.config.ts:26` 바로 위 주석. worker 소스를 고치고 `vite build` 를 다시 하지 않으면 시험이 옛 코드를 돌아 회귀를 심어도 통과한다는 함정. 1순위의 재현 절차가 성립하지 않으면(예: 쿠키 캐시 창을 못 잡으면) 이것으로 돌릴 것.
