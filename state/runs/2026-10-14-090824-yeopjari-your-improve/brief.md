- 과제: 프로필의 비워 둘 수 있는 항목이 실제로 비워지지 않는다 — `v.optional` 이 빈 문자열을 "안 보낸 것"으로 삼킨다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `v.optional(inner)` 는 `value === ''` 일 때 inner 를 아예 돌리지 않고 `fallback`(여기서는 `undefined`)을 돌려주고, `parse` 는 `undefined` 인 키를 결과에서 뺀다(`util/validate.ts:225`, `:29`). `PATCH /profile` 은 그 결과를 `Object.entries(fields).filter(([, value]) => value !== undefined)` 로 현재 행에 덮어쓰므로(`routes/profile.ts:252-255`), 빈 문자열로 보낸 항목은 조용히 **예전 값 그대로** 남는다. 웹은 흡연·음주·MBTI 를 `clearable allowDeselect` Select 로 내주고 해제하면 `''` 를 보내며(`apps/web/src/components/editors.tsx:711,712,726,754,755,758`), 하는 일도 `job_title: jobTitle.trim()` 로 `''` 를 보낸다(`:382`) — 회원이 지우고 저장하면 성공 응답을 받지만 값이 되돌아온다. 같은 문제를 키(`height_cm`)에서는 `body.height_cm === null` 특례로 손으로 막아 두었다(`routes/profile.ts:234`) — 이미 알려진 결함을 한 칸만 때운 자리다.

- 수용 기준:
  1) `PATCH /profile` 에 `{"smoking":""}` (또는 `drinking`/`mbti`/`religion`/`job_title`/`intro`)을 보내면 그 칼럼이 실제로 `''` 가 되고, 이어지는 `GET /profile/me` 가 `''` 를 돌려준다.
  2) **키를 아예 보내지 않으면 값은 그대로 남는다**(PATCH 의 기존 의미 유지). `{"smoking":"none"}` 만 보낸 요청이 `intro`·`job_title` 을 지우지 않는다 — 이것이 회귀 위험의 본체이므로 검사로 같이 못 박을 것.
  3) 공개 중(state `active`/`review`)인 프로필에서 체크리스트가 요구하는 항목(예: `intro`)을 지우려 하면 지금처럼 `invalid` 로 거부된다 — 그 경로는 바뀌지 않는다(`routes/profile.ts:290-296`).
  4) e2e 가 실제 배선(HTTP → 라우트 → DB)으로 1)과 2)를 증명한다. 단위 테스트만으로 끝내지 말 것.

- 건드릴 파일 (프로덕션 2개 + 테스트 2개):
  - `packages/core/src/util/validate.ts` — `v` 에 `clearable<T>(inner: Validator<T>): Validator<T | undefined>` 를 더한다. `undefined`·`null` 이면 `undefined`(= 안 보낸 것), 그 밖에는 **빈 문자열도 inner 에 그대로 넘긴다**. `Validator` 의 `optional?: boolean` 표시는 저장소 어디에서도 읽지 않는다(확인함: `v.optional` 외의 `.optional` 참조 0건) — 새 래퍼에 붙여도 되고 안 붙여도 동작은 같다. `v.optional` 은 **손대지 말 것**(호출부 30곳 이상, `v.optional(v.bool(...))` 같은 자리가 `''` 를 받으면 오류로 바뀐다). 모듈 머리 주석 밀도에 맞춰 "왜 optional 과 따로 두는가"를 영어 주석으로 적을 것.
  - `packages/core/src/routes/profile.ts:209-215` — `job_title`, `intro`, `smoking`, `drinking`, `religion`, `mbti` 여섯 개만 `v.optional(...)` → `v.clearable(...)` 로 바꾼다. 안쪽 검증기는 이미 빈 문자열을 통과시킨다(확인함: `v.text({min:0})` 는 `''` 반환 `validate.ts:85`, `v.string({min:0,max:20})` 는 길이 검사를 건너뛰고 `''` 반환 `:60-61`, 지역 함수 `oneOfOrEmpty` 는 `''` 를 명시적으로 허용 `profile.ts:70-79`). **`region`·`job_family` 는 바꾸지 말 것** — 웹이 `region ?? ''` 로 보내고(`editors.tsx:378-381`) 이것들은 필수 항목이라 지금처럼 빈 문자열이 무시되는 쪽이 맞다. `height_cm` 의 `null` 특례도 이번에는 그대로 둘 것(지우지 말 것 — 범위가 커진다).
  - `packages/core/src/__tests__/rules.test.ts` (또는 새 `validate.test.ts`) — `parse` 수준 단위 검사: `v.clearable(v.text({min:0,max:500}))` 가 `''` → `''`, 키 없음 → 키 없음, `v.optional` 은 둘 다 키 없음(현재 동작을 고정). 이것은 보조 증거다.
  - `scripts/e2e.mjs` — **라우트는 `PUT /api/v1/profile` 이다**(PATCH 아님, 부분 갱신 의미만 PATCH 식). 응답 본문이 갱신된 프로필을 그대로 돌려주므로(`withConsent.body.religion` 처럼) 응답으로 바로 검사할 수 있다. `:585-590` 의 종교·동의 블록 바로 뒤가 자리다. 두 검사: ① `await capped.client.put('/api/v1/profile', { smoking: <값> })` 로 채운 뒤 `{ smoking: '' }` 로 다시 보내 `body.smoking === ''` 이고 `db\`SELECT smoking FROM your.profiles WHERE account_id = ${capped.id}\`` 도 `''` ② 같은 회원에 `{ smoking: <값> }` 만 보냈을 때 `body.job_title`·`body.intro` 가 직전 값과 같다(키를 안 보내면 그대로 남는다). 검사 이름은 주변과 같은 한국어 투로.

- 검증 명령:
  - `npm run check` (lint + typecheck + `vitest run`) — 가볍다. 반드시 통과.
  - `npm run build`
  - e2e(이 저장소에서 배선을 증명하는 유일한 수단): `bash scripts/dev-reset.sh` 로 Docker `postgres:16-alpine`(127.0.0.1:55440) + 마이그레이션 + `:8788` 서버를 띄운 뒤 `BASE=http://127.0.0.1:8788 DATABASE_URL=... node scripts/e2e.mjs`. **고치기 전에 새 검사를 먼저 넣어 FAIL 을 눈으로 확인하고**(수용 기준 1 이 실패해야 한다) 그 다음 고칠 것.
  - 주의: `scripts/e2e.mjs` 가 가짜 서버 포트 18998·18997 을 하드코딩해 같은 기계의 다른 세션과 충돌한 적이 있다(EADDRINUSE). 막히면 그 포트를 쥔 프로세스를 확인할 것 — 이번 과제에서 포트를 환경변수로 바꾸지는 말 것(범위 외).

- 위험과 피할 것:
  - `v.optional` 의 의미를 바꾸는 "통합" 수정은 금지. 호출부가 `auth.ts`(captcha_token·current_password·알림 bool), `discover.ts`, `safety.ts`, `safety-plans.ts` 에 퍼져 있고 `''` 를 "안 보낸 것"으로 취급하는 데 의존한다. **좁히는 방향으로만** 고칠 것.
  - `religion` 은 민감정보 경로를 지난다(`your_priv.consents` 기록, `if (!consent) next.religion = ''`, 동의 없이 값이 들어오면 거부 `profile.ts:262-268`). 지우기는 동의 흐름과 무관해야 한다 — 동의 기록 코드는 건드리지 말 것.
  - `db/migrations/**`, `packages/core/src/auth/**`, `packages/core/src/vendor/postgres/**`, RLS 정책(`your.profiles` 정책·`your_priv.can_see`)은 이번 과제와 무관하다 — 열지 말 것.
  - 프로필 칼럼은 `NOT NULL DEFAULT ''` 계열이다(`db/migrations/0001_schema.sql` 의 `your.profiles`) — 빈 문자열을 쓰는 것은 스키마와 맞고 마이그레이션이 필요 없다. `NULL` 을 쓰려 하지 말 것.
  - 바뀐 값이 **최종 출력까지** 가는지 end-to-end 로 확인할 것: `GET /profile/me` 응답(`routes/profile.ts:155` 부근의 직렬화)과 DB 행 둘 다.

- 차선 후보: `util/validate.ts` 단위 테스트 공백 메우기 (가치 3 / 위험 1 / 작업량 S). 단위 테스트가 `rules.test.ts`·`vendor-postgres.test.ts`·`gateway-role.test.ts`·`shared-db-guard.test.ts` 뿐인데 `parse`/`v.*` 는 모든 쓰기 라우트가 지난다. 확인한 엣지: `v.string` 은 길이를 UTF-16 단위로 / `v.text` 는 코드포인트(`[...text].length`)로 센다 · `v.text({min:0})` 의 빈 문자열 통과 · `v.int('0x10')`→16, `v.int('1e3')`→1000 (Number() 강제변환) · `v.optional` 의 `''`→fallback · `v.array` 가 항목 실패 시 짧은 배열을 반환(다만 `errors` 가 채워져 `parse` 는 throw).
