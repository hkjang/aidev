- 과제: `npm test`(전체 = unit + api) 가 두 회차 연속 exit 1 — 먼저 실패 지점을 재현해 확정하고, api 묶음이 기동/통과하도록 고친다 (가치 5 / 위험 2 / 작업량 M)
- 왜: 릴리즈 검증이 `npm test --silent` 로 두 번 같은 이유로 실패했고, 직전 두 회차의 구현자는 모두 `npm run test:unit` + `npm run check` 만 돌렸다(개선 기록에 그대로 적혀 있다) — 즉 **아무도 `npm test` 를 실제로 돌려 본 적이 없다**. 이 회차의 목표는 기능 추가가 아니라 `npm test` 가 다시 초록이 되는 것이고, 고치면 다음 회차부터 api 그물(로그인·수집·정책·격리·제어 모듈)이 되살아난다.

## 먼저 할 일 (추측 금지 — 출력부터 본다)
워크트리에 `node_modules`·`dist` 가 **둘 다 없다**(이번 정찰에서 확인). 순서대로:
1. `npm install`
2. `npm run build`  ← `playwright.config.ts` 머리 주석이 "npm run build 가 선행돼야 한다" 고 적고 있다. 이 전제는 지금 **어느 스크립트에도 들어 있지 않다**(`package.json` 의 `test` 는 `playwright test` 뿐, `pretest` 없음).
3. `npm test 2>&1 | tail -60` — 실패가 (A) webServer 기동 실패인지 (B) 특정 시험의 단정 실패인지 먼저 가른다. 이 구분이 이번 과제의 전부다.

## 가설 (정찰이 읽기만으로 좁힌 것 — 어느 쪽인지는 **미확인**)
**(A) api webServer(`npx wrangler dev --local`) 가 아예 못 뜬다 — 1순위.**
- `wrangler.jsonc:9` 의 `"assets"` 블록에 **`directory` 가 없다**(`{ "binding": "ASSETS", "not_found_handling": "none", "html_handling": "auto-trailing-slash" }`). 개발·배포는 `@cloudflare/vite-plugin` 이 디렉터리를 채워 주지만, `playwright.config.ts` 의 webServer 는 **플러그인 없이 생(生) `npx wrangler dev`** 를 띄운다.
- 또 `main` 은 `src/worker/index.ts` 라 워커 자체는 `dist` 가 없어도 뜨지만, 정적 자산(`app.html` 등)은 `dist/client` 가 있어야 한다.
- 이 가설이 맞으면 증상은 "webServer 가 `http://127.0.0.1:8940/api/health` 를 못 열고 180초 뒤 타임아웃" 이고, **api 시험 전부가 한꺼번에** 실패한다. 두 회차가 코드와 무관하게 같은 이유로 실패한 것과 가장 잘 맞는다.
- 고칠 방향(느슨하게 만드는 것이 아니라, 문서에만 있던 전제를 명령에 넣는 것):
  - `package.json`: `"pretest": "npm run build"` 를 추가해 `npm test` 가 스스로 `dist` 를 만들게 한다. (`test:unit` 은 `pretest` 가 붙지 않도록 주의 — npm 은 `pretest:unit` 만 `test:unit` 에 붙이므로 `pretest` 는 `test` 에만 붙는다. 확인하고 쓸 것.)
  - 자산 디렉터리는 **`wrangler.jsonc` 를 건드리지 말고** `playwright.config.ts` 의 webServer 명령에 `--assets dist/client` 를 붙이는 쪽을 먼저 시도한다. `wrangler.jsonc` 에 `directory` 를 박으면 vite 플러그인·`scripts/deploy.mjs`·`scripts/verify-build.mjs`(워커 설정을 검사한다)에 번지고, 그쪽은 배포 경로다.
  - `--assets` 플래그 이름·필요 여부는 설치된 wrangler 버전(`npx wrangler dev --help`)으로 확인하고 쓸 것. 지금 이 플래그가 맞다는 보장은 없다(미확인).

**(B) `tests/api-c-policy.spec.ts:171` 의 새 전제가 깨진다 — 2순위.**
- e85afae 가 급증 시험 맨 앞에 넣은 줄: `expect((await a('POST', '/api/test/age-resources', { orgId: org, days: 7 })).body.changed).toBeGreaterThan(0);`
- 정찰이 읽기로 확인한 것: `resources` 에 `org_id`·`first_seen` 열이 있고(`src/worker/db/schema.ts:173-193`), `first_seen` 은 삽입 때만 쓰이고 이후 수집이 덮어쓰지 않으며(`src/worker/collect/resources.ts:197-210` — `ON CONFLICT ... DO UPDATE SET name, last_seen, gone` 뿐), `connectedOrg()`(`tests/helpers.ts:90-105`)가 `resources` 직업을 먼저 돌려 자원을 만든다. **따라서 논리적으로는 통과해야 한다** — 이 가설은 (A) 가 아닐 때만 파라.
- 그때 볼 자리: `src/worker/routes/test.ts:47-51`(`/test/age-resources` — `r.meta.changes` 가 로컬 D1 에서 UPDATE 에 대해 뭘 주는지), `src/shared/anomaly.ts:113-116`(`isNew` 판정), `src/worker/engine/run.ts:150-167`(`first_seen` 을 맵으로 넘기는 자리).
- 고칠 때 **시험의 단정을 지우거나 약하게 바꾸지 말 것**. `changed` 가 0 인 이유(자원이 없다/열이 다르다/`meta.changes` 가 없다)를 찾아 그 원인을 고친다.

## 수용 기준
1) 워크트리에서 `npm install && npm test` 가 **exit 0** 으로 끝난다(unit + api 두 프로젝트 모두 실행되고 0 skipped 가 아니다 — 리포터 요약에 api 시험 개수가 찍혀야 한다).
2) 실패 원인을 한 줄 주석/커밋 본문에 남긴다. 이 저장소 관례대로 날짜·증상으로(예: `anomaly.ts:70`, `unit-policy.spec.ts:178` 식).
3) 시험·검증을 느슨하게 만들지 않았음이 디프로 보인다: 지워진 `expect` 가 없고, `playwright.config.ts` 의 projects·timeout 을 늘려 통과시키지 않았다(webServer 기동 전제를 명시하는 변경은 허용).
4) `npm run check`(tsc 4개)와 `npm run test:unit` 이 그대로 통과한다(기준선 49 passed — 직전 회차가 51 이라 했지만 그 커밋은 워크트리에 없다; **지금 HEAD 는 e85afae** 이므로 실제 수를 먼저 기록할 것).

## 건드릴 파일 (3개 안쪽으로 끝낼 것)
- `package.json`:scripts — `pretest` 로 빌드 전제를 넣는다(가설 A).
- `playwright.config.ts`:webServer — `wrangler dev` 명령에 자산 디렉터리를 준다(가설 A). projects·timeout 은 건드리지 않는다.
- `tests/api-c-policy.spec.ts` 또는 `src/worker/routes/test.ts` — 가설 B 로 판명될 때만, 원인 쪽을 고친다.
- 위 셋으로 안 끝나면 **과제를 쪼개고 이번에는 (A) 만** 담을 것.

## 검증 명령
```
npm install
npm run build          # 폰트 서브셋 + tsc 4개 + vite build + verify-build
npm test               # 느리다(가짜 서버 :8941~:8943 + wrangler dev :8940, timeout 180초, workers 1)
npm run check
npm run test:unit      # 빠른 되돌림 확인
```
실패 묶음만 좁혀 돌릴 때: `npx playwright test --project=api tests/api-c-policy.spec.ts`

## 위험과 피할 것
- **`wrangler.jsonc` 의 배포 설정(`assets`·`routes`·`d1_databases`)을 바꾸지 말 것** — `scripts/verify-build.mjs` 가 워커 설정(ops 전용 D1·KV·Queue·DO, 사용자 지정 도메인, workers.dev 끔, `TEST_*` 변수 없음)을 검사하므로 여기를 건드리면 빌드가 다른 이유로 깨진다. 자산 경로는 시험 쪽 명령에서 준다.
- `migrations/` 는 손대지 말 것. 0004(`0004_resolve_new_resource_spikes.sql`)는 정찰이 스키마와 대조해 열 이름이 다 맞음을 확인했고(`anomalies.connection_id`·`resource`·`detected_at`·`resolved_by`, `resources.first_seen`), `scripts/db-local.mjs` 가 매 실행에서 DB 를 지우고 다시 적용하므로 이 실패의 원인이 아니다. 되돌린 열 정리 이력(a47c108)이 있는 구역이다.
- `src/worker/auth.ts`·`ledger.ts`·`routes/guard.ts`·`packages/guard` 는 이 과제와 무관하다 — 열지 말 것.
- 계측 공식(`src/worker/collect/analytics.ts`)을 손대지 말 것. 최근 3회차가 전부 여기서 되돌려졌다(a6eb4ea → d30940e → a47c108).
- `npm test` 는 느리고 가짜 서버 3개 + `wrangler dev` 를 띄운다. 이전 포트가 남아 있으면 `reuseExistingServer: true` 때문에 **낡은 서버에 붙어 엉뚱하게 통과/실패**할 수 있다 — 돌리기 전에 8940~8943 포트가 비어 있는지 확인할 것.
- 통과를 주장하기 전에 **실제 `npm test` 출력(마지막 요약 줄)을 커밋 본문이나 보고에 붙일 것**. 소스만 읽고 "고쳐졌다" 고 하지 말 것 — 직전 두 회차가 정확히 그렇게 실패했다.

## 차선 후보
`npm test` 가 기동 자체는 되고 api-c 급증 시험 하나만 실패하는 것으로 판명되면, 그 시험의 전제(`/api/test/age-resources` → `changed > 0`)가 왜 0 인지만 파서 `src/worker/routes/test.ts:47-51` 쪽을 고친다(시험 단정은 유지). 그마저 아니면: **`do:*` 집계 가짜 자원이 `resources.key` 와 영원히 안 맞아 서비스 귀속·알림 제목이 깨진다** (`src/worker/collect/analytics.ts:185` 가 넣는 `do:*` 를 `run.ts:179` 의 `resourceSvc.get('do:*')` 가 못 찾아 `anomalies.service_id` 가 null, `run.ts:322` 알림 제목에 `do:*` 가 그대로 나온다 — 라벨만 '전체 Durable Objects' 로 바꾸고 계측 공식은 건드리지 않는다, 가치 3 / 위험 2 / 작업량 S).
