- 과제: PR #8 의 CI 실패를 저장소 안에서 끝까지 배제한 뒤, 모든 라우트의 입력 관문인 `util/validate.ts` 의 `v.*` 검증기에 단위 테스트를 입히기 (가치 3 / 위험 1 / 작업량 S)
- 왜: 우선 과제가 지목한 PR #8(`origin/auto/2026-10-15-1508`, 커밋 `49334c4`)은 `git diff --stat main origin/auto/2026-10-15-1508` 로 확인한 결과 **`scripts/perf.mjs` 한 파일 +10/-1** 뿐이고, 그 diff 는 `let failed = total > BUDGET_KB` 와 `BUDGET_KB` 기본값을 건드리지 않은 채 `WARN_AT_PCT = 95` 경고 한 줄만 더한다(diff 전문을 읽고 확인). 프로덕션 코드 0줄·워크플로 0줄이므로 repo 안에 CI 단계를 떨어뜨릴 변경이 없다 — 같은 결론을 2026-10-14·10-15 두 회차가 로컬 전체 게이트 통과로 이미 증명했고(계정 수준 Actions 즉시 실패, 5개 무관 저장소 13/13 동시), 이 세션에서도 `gh` 가 승인 거부되어 CI 주석을 읽을 수 없었다. 배제가 끝나면 남는 가장 값싸고 겹침 없는 과제는 `packages/core/src/util/validate.ts`(230줄, 라우트·도메인 **14개 파일**이 import — `routes/auth.ts`·`reviews.ts`·`catalog.ts`·`me.ts`·`consults.ts`·`vendor.ts`·`interviews.ts`·`client-errors.ts`·`admin/{catalog,moderation,ads,vendors,general}.ts`·`domain/reviews.ts`)가 **테스트 0건**인 것이다. `packages/core/src/__tests__/` 에는 `domain`·`gateway-role`·`shared-db-guard`·`text`·`vendor-postgres` 5파일만 있다(ls 로 확인). 이 파일이 모든 사용자 노출 검증 오류 문구와 모든 입력 경계를 혼자 정하는데, 조용히 틀려도 지금은 아무 테스트도 말해 주지 않는다.

## 0단계 — PR #8 배제 (코드 변경 없이, 먼저 할 것)
1. `git log --oneline main..origin/auto/2026-10-15-1508` → 커밋 1개(`49334c4`), `git diff --stat` → `scripts/perf.mjs | 11 ++++++++++-`.
2. PR #8 내용을 main 위에 **임시로** 얹어(`git cherry-pick 49334c4` 또는 `git checkout origin/auto/2026-10-15-1508 -- scripts/perf.mjs`) check.yml 단계를 그 순서대로 로컬 재현: `npm ci --no-audit --fund=false` → `npm run check` → `npm run build` → `node scripts/perf.mjs` → `node scripts/ci-env.mjs` → `DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55443/review REVIEW_APP_DB_PASSWORD=devapppass REVIEW_GATEWAY_DB_PASSWORD=devgatewaypass node scripts/migrate.mjs` → 서버 기동 → `node scripts/e2e.mjs` → `npx playwright install chromium`(`--with-deps` 는 sudo 를 요구해 로컬에서 실패한다) → `node scripts/seed-demo.mjs` + `node scripts/a11y.mjs`. 선행: `docker run` 으로 `127.0.0.1:55443` PostgreSQL, 그리고 **8796 포트를 다른 프로세스가 듣고 있지 않은지 확인**(이 샌드박스에서 지워진 워크트리의 고아 서버가 8796 을 듣고 있던 사례가 있다 — 그대로면 남의 서버를 검사해 가짜 통과가 난다).
3. 전부 통과하면 repo 쪽 결백이 증명된 것이다. **중복 PR 을 만들지 말 것** — `scripts/perf.mjs` 를 원래대로 되돌리고(`git checkout main -- scripts/perf.mjs`, `git status` 로 깨끗함 확인) 아래 본 과제로 넘어간다. 통과하지 못하는 단계가 나오면 **그것이 이번 회차의 과제**다(그때는 아래 본 과제를 버리고 그 실패를 고친다).
4. 금지: `check.yml` 의 단계 삭제·`continue-on-error`·`timeout-minutes` 완화·`npm ci`→`npm install` 교체·`INITIAL_JS_BUDGET_KB` 상향. 워크플로를 느슨하게 만들어 통과시키는 것은 금지다(운영자 지시). 실패 분석 때 주석 한 줄 수정까지는 허용.

## 본 과제 — `validate.test.ts`
`packages/core/src/__tests__/text.test.ts` 의 관례를 그대로 따를 것(`Array<[입력, 기대]>` 표 + `for … it()`, 함수별 `describe`). `v.*` 는 모두 `run(value, path, errors)` 를 가진 Validator 를 돌려주므로, `parse`/`parseOne` 으로 호출하고 실패는 `errors.ts` 의 `invalid()` 가 던지는 예외로 받는 것이 프로덕션 배선에 가장 가깝다(`parseOne(v.int({min:1,label:'쪽'}), '0', 'page')` 형태). 던져진 오류의 필드 메시지까지 단정할 것 — 그 문구가 화면에 그대로 나간다.

덮을 것(코드를 읽고 뽑은 경계. 기대값 중 "손계산" 표시된 것은 **반드시 실행해 확정**하고, 어긋나면 코드 쪽이 정답이니 기대값을 고칠 것):
- `v.string`: 공백만 → `'…을 입력해 주세요.'`, `min` 미달/`max` 초과 문구, `pattern` 불일치 시 `patternMessage` 우선, 반환값이 `trim()` 된 것. **`max` 는 `text.length`(UTF-16 단위)로 센다** — `v.text` 는 `[...text].length`(코드포인트)로 센다. 같은 `max:4` 에 `'🙂🙂🙂'` 를 주면 `string` 은 6 으로 세어 거부하고 `text` 는 3 으로 세어 통과한다(손계산). 지금 동작을 그대로 기록하고 주석으로 불일치를 남길 것 — **이번 회차에 통일하지 말 것**(사용자 노출 한도가 바뀌는 변경이다).
- `v.text`: `\r\n`·`\r` → `\n` 정규화, `\u0000`·`\u001b` 제거, **`\t`(\u0009)와 `\n` 은 남는다**, `min: 0` + 공백만 → `''`(오류 없음), `min: 0` + 빈 문자열 → `''`.
- `v.int`: `'7'` → 7, `'  '` → 오류(`trim()` 이 빈 문자열이라 문자열이 그대로 넘어가 `typeof !== 'number'`), `7.5` → 오류, `min`/`max` 경계값은 통과, **`'0x10'` → `Number()` 가 16 으로 읽어 통과(손계산)**, `true` → 오류.
- `v.bool`: `'true'`/`'1'`/`'false'`/`'0'`/실제 boolean 만 통과, `'yes'`·`1`(숫자) → 오류.
- `v.uuid`: 대문자 UUID → **소문자로 반환**, nil UUID(`00000000-…`)도 통과(버전 검사 없음, 손계산), 길이 틀린 값 → `'… 식별자가 올바르지 않습니다.'`.
- `v.email`: `trim()` 후 판정, 255자 이상 → 형식 오류, 도메인에 점이 없는 `'a@b'` → 오류, **`'.a@b.com'` 은 통과**(local 문자 클래스에 `.` 가 들어 있어 선·후행 점을 막지 않는다, 손계산).
- `v.instant`: 주석은 "ISO-8601 instant" 라고 하지만 구현은 `new Date(value)` 라 **`'2026'` 같은 느슨한 입력도 통과**한다(손계산 — 2026-01-01T00:00:00Z). 현재 동작을 기록하고 주석으로 남길 것.
- `v.oneOf`: 값 ≤6 이면 목록을 나열한 문구, >6 이면 `'… 값이 목록에 없습니다.'`(경계인 6과 7 양쪽을 찍을 것).
- `v.array`: 비배열 → 오류, `min`/`max` 개수 문구, 원소 오류의 경로가 `path[0]` 형식, **원소 하나가 깨져도 `parse` 전체가 던진다**.
- `v.object`: 비객체·배열·`null` → 오류, 중첩 경로가 `path.key` 형식.
- `v.optional`: `undefined`·`null`·**`''`** 를 모두 '없음'으로 보고 `fallback` 반환 — `v.optional(v.string({max:10,label:'소개'}))` 에 `''` 를 주면 오류 없이 `undefined` 이고, `parse` 는 그 키를 **결과에서 아예 뺀다**(`if (result !== undefined) out[key] = result`).
- `parse`/`parseOne`/`fail`: `parse(shape, 'nope')`·`parse(shape, null)` → 입력을 `{}` 로 보고 필수 필드가 전부 오류, 같은 path 에 두 번 실패하면 **첫 메시지가 남는다**(`fail` 의 `if (!(path in errors))`).

- 수용 기준:
  1) 0단계에서 check.yml 의 9단계가 PR #8 내용으로 로컬에서 전부 통과하고, 그 관측값(테스트 수, perf KB, e2e passed/failed)이 보고에 적힌다. 워크플로 파일은 완화되지 않았다(`git diff .github/` 가 비어 있거나 주석 전용).
  2) `packages/core/src/__tests__/validate.test.ts` 가 새로 생기고, `npm run check` 가 통과한다. **프로덕션 코드 0줄 변경** — `git diff --stat` 에 `packages/core/src/util/validate.ts` 가 없다.
  3) 테스트가 실효 있음을 **변이 주입**으로 증명한다: ① `v.string` 의 `text.length > opts.max` → `>=` ② `v.optional` 의 `value === ''` 조건 제거 ③ `fail` 의 `if (!(path in errors))` 제거 — 각각 새 테스트가 실패하는 것을 출력으로 보이고 즉시 되돌린 뒤(`git diff -- packages/core/src/util/validate.ts` 빈 것 확인) 다시 통과시킨다.
- 건드릴 파일:
  - `packages/core/src/__tests__/validate.test.ts` — 신규. 위 목록대로 함수별 `describe`.
  - (0단계 중 임시로만) `scripts/perf.mjs` — 반드시 되돌릴 것.
  - 프로덕션 파일 0개.
- 검증 명령:
  - `npm run check` (= `eslint .` + core·server·web 타입 + `vitest run`). **기준선: main 에서 `Test Files 6 passed (6)` / `Tests 118 passed (118)`** — 앞 두 회차가 실행으로 관측한 값이다. 이번 회차 뒤에는 파일 7, 테스트 118+N 이 되어야 한다.
  - `npx vitest run packages/core/src/__tests__/validate.test.ts` — 작성 중 빠른 반복용.
  - 0단계는 위 9단계. `node scripts/e2e.mjs` 기준선은 `bash scripts/dev-reset.sh` 직후 **218 passed / 0 failed**, `node scripts/perf.mjs` 는 **223.7 KB initial JS (gzip), budget 230 KB** (여러 회차 관측값, 이번에 재확인할 것).
- 위험과 피할 것:
  - 열린 PR 과 파일이 겹치면 안 된다 — #4 `quota.ts`·`quota.test.ts`·`scripts/e2e.mjs`, #5 `pipeline.ts`·`util/text.ts`·`client-address.test.ts`, #6 `check.yml`·`README.md`·`scripts/ci-local.sh`, #7 `__tests__/text.test.ts`, #8 `scripts/perf.mjs`. `validate.test.ts` 는 어느 것과도 겹치지 않는다(새 파일). **`__tests__/text.test.ts` 를 건드리지 말 것** — PR #7 과 충돌한다. `validate.ts` 가 `koreanParticle` 을 쓰지만 조사 자체는 PR #7 이 이미 덮으므로 여기서 조사 표를 다시 만들지 말고, 메시지 전문으로만 단정할 것.
  - 보호 경로를 피할 것: `db/migrations/`·`auth/*`·`http/pipeline.ts`·`routes/sso.ts`·`.github/workflows/`.
  - 테스트를 쓰다 보면 위 세 자리(`string` 의 UTF-16 계수, `instant` 의 느슨한 파싱, `email` 의 선행 점)가 결함처럼 보일 것이다. **이번 회차에 고치지 말 것** — 사용자 노출 한도·수락 입력 범위가 바뀌는 동작 변경이고, 고치려면 e2e 회귀까지 봐야 한다. 현재 동작을 테스트로 고정하고 주석 + 보류 아이디어로만 남길 것.
  - 손계산으로 적은 기대값(`'0x10'`→16, `'🙂🙂🙂'` 6 vs 3, `'.a@b.com'` 통과, `'2026'` 통과, nil UUID 통과)은 **미확인**이다. 실행해 확정하고, 다르면 코드가 정답이다.
  - `vitest.config.ts` 의 `testTimeout` 은 20s(crypto 헬퍼 때문) — 순수 함수 테스트라 영향 없지만 설정을 건드리지 말 것.
- 차선 후보: `packages/core/src/domain/products.ts` 의 `slugify` 제거 — 정의 외 참조 0건의 죽은 코드이고 한글 제품명에서 빈 슬러그를 만든다. 삭제 전·후로 전역 `grep -rn slugify` 0건을 확인할 것(`clientIpFrom` 제거 때 쓴 방법). 어느 열린 PR 과도 겹치지 않는다.
