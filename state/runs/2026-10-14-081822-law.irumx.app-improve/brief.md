# 과제서 — 2026-10-14-081822-law.irumx.app-improve

- 과제: 모의 도메인 공급자를 운영에서 거부 — `loadEnv` 가 `LAW_DOMAIN_PROVIDER=mock` 을 `allowMocks` 없이 받아들이는 구멍 막기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `packages/core/src/env.ts:163` 의 `domains: enumOf<DomainProvider>(raw.LAW_DOMAIN_PROVIDER, ['mock','pages','manual'], dev ? 'mock' : 'manual')` 는 `allowMocks` 를 보지 않는다. 다른 모의 공급자는 전부 막혀 있다 — `providers/pg.ts:63`, `providers/sms.ts:17,21`, `providers/identity.ts:23,27`, `modules/payments.ts:75`, `routes/console/payments.ts:48` 이 모두 `env.providers.allowMocks` 를 확인한다. 도메인만 빠져 있어서, 운영 환경에서 이 변수 하나가 `mock` 으로 들어가면 `providers/dns.ts:10` 의 소유권 확인(DOM-04, TXT 레코드)이 Cloudflare DoH 대신 `env.providers.mockBase`(기본값 `http://127.0.0.1:18910`)의 레코드 표를 읽는다 — 즉 타인 도메인 소유권 증명을 외부 DNS 없이 통과시킬 수 있는 경로가 열린다. 설정 한 줄로 보안 통제가 무력화되는 자리이므로, 기존 공급자들과 같은 방식으로 막는다.

- 수용 기준:
  1) `allowMocks === false` 인데 `LAW_DOMAIN_PROVIDER=mock` 이면 `loadEnv` 가 `EnvError` 를 던진다 (`resolveKey` 가 운영에서 개발 키를 거부하는 것과 같은 방식 — `env.ts:196-199`).
  2) `allowMocks === true`(개발, 또는 `LAW_ALLOW_MOCK_PROVIDERS=true`)에서는 `mock` 이 그대로 통과한다. `pages`·`manual` 은 두 경우 모두 통과한다. 기본값 계산(`dev ? 'mock' : 'manual'`)은 그대로 둔다.
  3) 새 테스트가 위 네 조합(mock×allowMocks 두 값, pages/manual)을 모두 실행해 증명한다. 소스 문자열 검사가 아니라 `loadEnv` 를 실제로 호출해서 증명할 것.

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `packages/core/src/env.ts:loadEnv` — `allowMocks` 계산(126행) 뒤, `providers.domains` 를 정하는 자리에서 `!allowMocks && domains === 'mock'` 이면 `throw new EnvError('LAW_DOMAIN_PROVIDER=mock requires LAW_ALLOW_MOCK_PROVIDERS=true')`. `enumOf` 호출을 const 로 빼서 검사한 뒤 객체에 넣는 정도로 최소 변경.
  - `packages/core/src/__tests__/env-mock-providers.test.ts` (신규) — 표 테스트. `loadEnv` 는 순수 함수(raw 객체를 받음)라 DB 없이 돈다. 운영 모양의 raw 를 만들려면 **반드시** 다음을 채워야 한다(안 채우면 다른 이유로 먼저 던진다):
    `LAW_DATABASE_URL`(아무 문자열), `LAW_SESSION_KEY`·`LAW_SECRETS_KEY`·`LAW_MATCH_HMAC_KEY`(각각 64자 hex, 예 `'a'.repeat(64)`), `LAW_SIGNING_KEY`(비어 있지 않은 문자열이면 됨 — `env.ts:124` 는 존재만 본다), `LAW_DEV` 는 넣지 않음(→ `dev=false`).
    개발 쪽 사례는 같은 raw 에 `LAW_DEV: 'true'` 만 더하면 된다.
  - `.env.example` / `engineering/operations.md` — **건드리지 말 것.** `.env.example:18` 은 `LAW_DOMAIN_PROVIDER=mock` 이지만 개발 전용이고 `LAW_DEV=true` 와 함께 쓰이므로 그대로 둬도 통과한다.

- 검증 명령:
  - `npx vitest run packages/core/src/__tests__/env-mock-providers.test.ts`
  - `npm run test` (= `vitest run`, 전체 단위 시험 — 현재 테스트 파일 5개)
  - `npm run typecheck`, `npm run lint`
  - (DB 필요) `npm run e2e` 는 `scripts/ci-env.mjs` 가 `LAW_DEV=true` 를 주므로 영향 없음 — 확인은 했지만 실행은 **미확인**(이 세션에서 DB를 띄우지 못했다).

- 위험과 피할 것:
  - **운영 배포는 이미 안전한 값이다 — 확인했음.** `wrangler.toml:46` `LAW_DOMAIN_PROVIDER = "pages"`, `wrangler.toml:57` `LAW_ALLOW_MOCK_PROVIDERS = "false"`. 따라서 이 변경으로 지금 배포가 기동 실패할 일은 없다. 그래도 `loadEnv` 가 던지면 워커가 아예 못 뜨는 자리이므로, 조건을 `!allowMocks && domains === 'mock'` 정확히 이것만으로 유지하고 다른 검증을 끼워 넣지 말 것.
  - `enumOf` 의 fallback 동작을 바꾸지 말 것. 오타 값은 지금 `manual`(안전)로 떨어지는데, 이것까지 던지게 만들면 기동 실패 위험만 커진다.
  - `providers/domains.ts`·`providers/dns.ts` 는 읽기만 하고 고치지 말 것. mock 분기 자체를 지우는 식의 확장은 개발·e2e 를 전부 깬다.
  - 보호 경로(`packages/core/src/auth/*`, `db/migrations/*`, `.github/workflows/*`)는 건드리지 않는다. 이번 과제는 그럴 필요가 없다.
  - 과거 교훈: 같은 종류의 검사가 여러 곳(pg/sms/identity)에 있지만 **통합하려 하지 말 것** — 각자 반환 계약이 다르다(`{ok:false,...}` 대 throw). 도메인만 좁게 막는다.

- 차선 후보: **redactSecrets 계약을 표 테스트로 고정** (가치 4 / 위험 1 / 작업량 S). `packages/core/src/http/pipeline.ts:496` 의 `redactSecrets` 는 `logFailure`(pipeline.ts:479-490)를 통해 서버 오류 로그로, 그리고 운영자 알림 경로로 나가는 유일한 비밀값 차단기인데 테스트가 0건이다. 현재 5개 치환 규칙 — ① 쿼리 파라미터 `api_key|key|token|access_token|secret|client_secret|password|signature|sig|code` ② `Bearer|Basic` + 8자 이상 ③ 접두사 `re_|sb_secret_|sbp_|cfat_|sk-|test_sk_|live_sk_` + 8자 이상 ④ `postgres(ql)://user:pw@` ⑤ 주민번호 `\d{6}-?[1-8]\d{6}`(SEC-07/AT-26) — 을 `packages/core/src/__tests__/redact-secrets.test.ts` 에 차단/허용 표로 고정. **패턴을 넓히지 말 것**: 지금 안 잡히는 입력은 기대값을 현재 동작대로 쓰고 주석으로 "현재 계약 밖" 이라 적는다. 프로덕션 파일 0개 변경.
