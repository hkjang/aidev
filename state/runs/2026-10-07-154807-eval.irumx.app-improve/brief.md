# 과제서 — 2026-10-07-154807-eval.irumx.app-improve

> 정찰 메모(먼저 읽을 것): 러너가 붙인 "우선 과제" 는 *릴리즈 워크플로가 같은 이유로 두 번 실패* 라고
> 적었지만, **이 저장소에는 워크플로 파일이 없습니다** — `.github/` 자체가 없음(확인: `ls -a`,
> `ls -R .github` → 없음). 실패 사유도 `error: agent produced no result ()` 로, 저장소 코드가 아니라
> **구현 에이전트가 아무 결과도 내지 않은** 러너 쪽 실패입니다. 즉 저장소에서 고칠 "실패한 단계의
> 스크립트·테스트" 가 존재하지 않습니다(워크플로를 느슨하게 만드는 일도 당연히 없음).
> 그러므로 이번 회차는 **지난 회차가 손도 못 댄 같은 과제를, 구현자가 멈출 자리를 모두 지워서 다시
> 냅니다.** 아래는 추측 없이 실제로 열어 본 코드로 쓴 것입니다.

- 과제: 예산·시험 수 한도 **동시 예약** 시험 (설계 수용 시험 12번) (가치 4 / 위험 1 / 작업량 S)
- 왜: `docs/spec-coverage.md:35` 12번이 🟡 — "조건부 INSERT 하나(활성 예약+정산 ≤ 한도). api-targets '예산'
  은 순차 시험". `src/worker/services/usage.ts:32 reserveSql` 은 한도 검사와 INSERT 를 한 문장에 넣어
  동시 예약을 막게 설계돼 있는데, 그 동시성을 실제로 밟는 시험이 하나도 없다. 조건을 나중에 누가
  `SELECT` 후 `INSERT` 로 풀어도 지금은 아무 시험도 깨지지 않는다 — 그 구멍을 닫는다.
- 수용 기준:
  1) `tests/api-targets.spec.ts` 에 동시 예약 시험 1개가 추가되고 `--project=api` 로 통과한다.
  2) 같은 업체 공간에 `POST .../experiments` 를 `Promise.all` 로 5건 동시에 보내면, **201 의 개수가
     한도에 들어가는 수와 정확히 같고** 나머지는 전부 `409 over_budget` 이다(201 이 한도를 넘는 수만큼
     나오면 실패해야 한다).
  3) 시험이 끝난 뒤 `GET /api/w/:ws/usage` 의 `trials`(그리고 `reserved`)가 **시작 전 값 + 성공한
     예약분** 과 정확히 같다 — 즉 "한도를 넘긴 예약이 하나도 들어가지 않았다" 를 합계로 증명한다.
  4) 프로덕션 코드 변경 0개. `npm run build` 의 `scripts/check-pack.ts`(판정기 역검증 40개)에 영향 없음.

- 건드릴 파일 (2개, 그 중 프로덕션은 주석 1줄뿐):
  - `tests/api-targets.spec.ts` — 기존 `test('예산: 월 한도를 넘는 실험은 예약되지 않는다…')`
    (158~173줄) **바로 아래**에 새 `test('예산 동시 예약: …')` 를 추가. 기존 시험은 건드리지 말 것.
  - (선택, 1줄) `src/worker/lib/util.ts:76` 주석의 `tests/guard.spec.ts` 참조 — 그 파일은 **없다**
    (`ls tests` 로 확인). 이번에 쓴 동시 예약 시험 경로로 바꿔 적어 두면 주석이 사실이 된다.
    부담되면 생략해도 수용 기준에 영향 없음.

- 어떻게 (읽어 본 코드 기준, 그대로 따라도 됨):
  1) 도우미는 이미 다 있다 — `tests/helpers.ts` 의 `ownerApi`, `packProject`, `login(OPERATOR)`,
     `Api.raw/post/get`. 기존 예산 시험과 같은 업체 공간(`ownerApi('other@eval.test')`)을 쓸 것.
     실행 중인 실험을 만들지 않으므로(계획까지만) 다른 시험과 충돌하지 않는다.
  2) **절대 수를 쓰지 말고 기준선부터 읽을 것.** 같은 달 같은 업체 공간에서 앞선 시험이 이미 예약을
     남긴다(158줄 시험이 3시험을 예약한다). 그래서:
     `const u0 = await api.get(`/api/w/${ws}/usage`)` → `u0.trials`, `u0.reserved` 를 기준선으로 잡는다.
  3) **한 건이 몇 시험·몇 마이크로인지 예약 없이 미리 알 수 있다** —
     `POST ${p.base}/experiments/estimate` (`src/worker/routes/experiments.ts:30`)는
     `{ trials, reserveMicros, … }` 를 돌려주고 **아무것도 예약하지 않는다**.
     `{ dataset_version_id: p.dv, candidate: p.reference, repetitions: 1, case_keys: ['N02','N03','A01'] }`
     로 불러 `trials`(=3일 것, 확인은 시험이 하게 둘 것)를 얻는다.
  4) 운영자로 한도를 **딱 2건만 들어가게** 맞춘다:
     `ops.post(`/api/ops/workspaces/${ws}/limits`, { monthly_budget_micros: 1_000_000_000,
     monthly_trial_cap: u0.trials + est.trials * 2 + 1 })`.
     `p.reference` 는 `scripted` 대상이라 모델 비용이 0이다(`usage.ts:23` — `cfg.kind === 'builtin'` 일
     때만 비용을 쌓는다). 그래서 **시험 수 한도만 걸리게** 하는 것이 결정적이고, 금액은 넉넉히 둔다.
  5) 동시에 5건:
     `const rs = await Promise.all([...Array(5)].map((_, i) => api.raw('POST', `${p.base}/experiments`,
     { dataset_version_id: p.dv, candidate: p.reference, repetitions: 1,
       case_keys: ['N02','N03','A01'], name: `동시 ${i}` })))`
     - `idempotency_key` 를 **주지 말 것**. 주면 `services/experiments.ts:100` 의 중복 처리로 같은
       실험이 돌아와 아무것도 증명하지 못한다. 안 주면 5건 모두 별개로 만들어진다(확인함).
     - 201 개수 === 2, 나머지는 모두 409 이고 `(await r.json()).error.code === 'over_budget'`.
  6) 합계 확인: `const u1 = await api.get(`/api/w/${ws}/usage`)` →
     `expect(u1.trials).toBe(u0.trials + est.trials * 2)`.
  7) **끝에 반드시 한도를 되돌릴 것** — `try { … } finally { await ops.post(.../limits,
     { monthly_budget_micros: 1_000_000_000, monthly_trial_cap: 1_000_000 }) }`.
     되돌리지 않으면 뒤따르는 api 시험이 전부 `over_budget` 으로 깨진다(기존 172줄도 같은 이유로
     마지막에 되돌린다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 빠른 길(이미 `npm run build` 결과가 있고 `.wrangler` 가 살아 있을 때):
    `npx playwright test tests/api-targets.spec.ts --project=api -g "예산"`
    → 기존 예산 시험 + 새 동시 예약 시험 둘 다 돈다.
  - 코드를 바꾸기 전 **한 번 그대로 돌려 초록인지 먼저 볼 것**(배선이 살아 있는지 확인).
  - 최종: `npm run build && npm test` (빌드 수 분, 전체 Playwright). 빌드를 다시 했으면 떠 있는
    `wrangler dev`(:8860)를 끄고 다시 띄울 것 — `reuseExistingServer: true` 라 옛 서버가 떠 있으면
    **옛 코드를 시험한다**(profile 의 검증 함정).

- 위험과 피할 것:
  - `src/worker/services/usage.ts` 의 `reserveSql` 과 `src/worker/lib/util.ts` 의 `changedGuardSql`
    **SQL 을 고치지 말 것.** 이 과제는 지금 동작을 고정하는 시험이다. 가드가 0행을 오류로 바꾸는
    장치(batch 전체 롤백 → `malformed JSON` → 409 매핑, `services/experiments.ts:138-144`)를 건드리면
    낙관적 동시성이 조용히 무너진다.
  - `migrations/`, `src/worker/auth.ts`, `scripts/deploy.mjs`, `scripts/verify-build.mjs` 는 손대지 말 것.
  - 시험이 업체 공간 한도를 바꾼 뒤 되돌리지 않으면 뒤 시험이 전부 깨진다(위 7번).
  - 동시 5건을 `builtin` 대상으로 하지 말 것 — 가짜 모델을 실제로 부르게 되고 금액이 추정값이라
    경계 계산이 흔들린다. `p.reference`(scripted)로 시험 수 한도만 밟는다.
  - 201 개수가 2가 아니라 3 이상 나오면 **시험을 느슨하게 고치지 말고** 그대로 실패로 남기고
    `reserveSql` 의 조건이 정말 성립하는지 보고할 것 — 그게 이 시험의 목적이다.
  - 미확인: `estimate` 가 돌려주는 `trials` 가 정확히 3인지 직접 돌려 보지 않았다(코드상 사례 3개 ×
    반복 1 × 후보 1칸). 그래서 위 계산을 상수 3 대신 `est.trials` 로 쓰라고 적었다.

- 차선 후보: **모델 출력 속 비밀 모양 문자열 가림 범위 좁히기(수용 시험 23)** — `docs/spec-coverage.md`
  23번 🟡. 단, 이 서비스는 합성 자료의 표식 노출을 판정 근거로 쓰므로 가림은 **원문·보고서 출력
  경로로만** 좁혀야 하고, 넓히면 `scripts/check-pack.ts`(판정기 역검증 40개)가 빌드에서 막는다.
  1순위가 성립하지 않으면 이쪽을 잡되, 가림을 판정 경로에 넣지 말 것.
