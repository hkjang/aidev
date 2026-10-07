- 과제: 예산·시험 수 한도 **동시 예약** 시험 (설계 수용 시험 12번) (가치 4 / 위험 2 / 작업량 S)
- 왜: `docs/spec-coverage.md` 28장 12번(“예산 동시 예약”)이 🟡 이고 메모가 "조건부 INSERT 하나(활성 예약+정산 ≤ 한도). api-targets ‘예산’ 은 순차 시험" 이다 — 한도를 지키는 유일한 장치(`src/worker/services/usage.ts:reserveSql` 의 조건부 INSERT + `changedGuardSql(1)`)가 **동시 요청에서도** 한도를 넘지 않는지 증명하는 시험이 없다. 동시 예약 시험을 넣으면 한도 초과 과금(운영자 전체 한도 `PLATFORM_MONTHLY_CAP_MICROS` 포함)을 막는 근거가 생기고 대응표 12번을 ✅ 로 올릴 수 있다.
- 수용 기준:
  1) 업체 공간 한도를 "실험 한 개만 들어갈 크기"로 맞춘 뒤 **같은 요청 5개를 `Promise.all` 로 동시에** `POST {base}/experiments` 로 보내면, 201 은 한도가 허용하는 수(1개)뿐이고 나머지는 모두 409 + `error.code === 'over_budget'` 이다. (상태 코드 합이 5인지도 확인 — 500 이나 다른 코드가 섞이면 실패)
  2) 그 직후 `GET /api/w/:ws/usage` 에서 `reserved + settled <= budget` 이고 `trials <= trialCap` 이다 (이 응답에 `reserved·settled·trials·budget·trialCap` 이 그대로 있다 — `src/worker/routes/workspaces.ts` 의 `r.get('/w/:ws/usage')`).
  3) 같은 방식으로 **시험 수 한도**(`monthly_trial_cap`)만 걸리게 한 경우도 확인한다: 비용 0인 시연 대상(`p.reference`)으로 동시 5개를 보내 예약된 `trials` 합이 `trialCap` 을 넘지 않는다.
  4) 시험이 끝날 때 한도를 원래대로 되돌린다(`monthly_budget_micros: 1_000_000_000`, `monthly_trial_cap: 1_000_000`) — 기존 ‘예산’ 시험이 하는 것과 같게. 되돌리지 않으면 뒤 시험이 전부 깨진다.
  5) `docs/spec-coverage.md` 12번 행을 ✅ 로 바꾸고 근거를 새 시험 이름으로 적는다. (동시성 증명이 기대와 다르게 나오면 ✅ 로 올리지 말고 실제로 관찰한 것을 메모에 적을 것)
- 건드릴 파일:
  - `tests/api-targets.spec.ts` — 기존 `test('예산: 월 한도를 넘는 실험은 예약되지 않는다…')`(158행 부근) **바로 뒤에** 새 test 추가. 그 시험과 똑같이 `ownerApi('other@eval.test')` + `packProject` + `login(OPERATOR, { cache: true })` 로 한도를 바꾼다. 요청은 `api.raw('POST', …)` 를 써야 상태 코드를 직접 볼 수 있다(`api.post` 는 기대 상태와 다르면 throw 한다). `STARTER` 사례 묶음과 `p.reference`/`p.builtin` 은 이미 이 파일에 있다.
  - `docs/spec-coverage.md` — 28장 표 12번 행만.
  - 프로덕션 코드는 바꾸지 않는다(0개). 시험이 실제로 한도 초과를 드러내면 고치지 말고 과제서 결과에 적고 멈춘다(별도 회차 과제).
- 검증 명령:
  - `npm run build` (글꼴·타입검사·판정기 역검증 40개·vite build·결과 점검 — 몇 분 걸린다. 시험은 **빌드 결과**를 wrangler dev 로 띄우므로 먼저 해야 한다)
  - `npx playwright test --project=api tests/api-targets.spec.ts`
  - 마지막에 `npm test` 로 다른 시험(특히 `api-security` 의 ‘다른 업체 공간’, `api-zz-backup`)이 한도 변경 때문에 깨지지 않는지 확인.
- 위험과 피할 것:
  - `other@eval.test` 업체 공간은 `api-security` 도 쓴다. 한도를 낮춘 채로 시험을 끝내지 말 것(수용 기준 4). `playwright.config.ts` 는 `workers: 1` 이라 파일 안에서 순서대로 돈다.
  - 만들기만 한 실험의 예약은 `status='active'` 로 남아 뒤 실험의 예산을 먹는다 — 한도 복구로 해결하고, 새 실험을 `start` 하지 말 것(시험 시간이 길어진다).
  - `auth`·`migrations/*`·CI 설정은 건드리지 않는다. `.github` 는 이 저장소에 아예 없다 — 새로 만들지 말 것.
  - `WRITE_RATE_USER` 는 1분 120번(`wrangler.jsonc:47`)이라 동시 5개는 걸리지 않는다. 20개 이상 동시로 올리지 말 것.
  - 로컬 D1 은 쓰기를 직렬화하므로 이 시험은 “진짜 병렬 경쟁”이 아니라 **동시 도착 요청이 합계 한도를 넘지 않는가**를 본다. 대응표 메모에 그 한계를 한 줄로 솔직히 적을 것(과장 금지).
  - 빌드를 다시 했으면 떠 있는 `wrangler dev`(:8860)를 끄고 다시 띄워야 새 자산이 보인다(README).
- 차선 후보: **큐 중복 전달(수용 시험 10번) 시험** — 같은 시험 id 메시지를 두 번 처리해도 결과가 하나뿐임을 증명. `src/worker/domain/engine.ts:processTrial` 의 임대(lease)·세대(generation) 가드가 이미 있으나 "같은 메시지를 직접 두 번 보내는 시험은 없음"(대응표 10번). 다만 큐 메시지를 두 번 넣는 경로가 시험에서 열려 있는지 **미확인** — 열려 있지 않으면 1순위로 돌아올 것.
