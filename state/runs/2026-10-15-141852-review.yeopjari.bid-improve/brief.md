- 과제: `util/text.ts` 의 나머지 순수 함수에 단위 테스트를 입히기 — `koreanParticle`·`maskEmail`·`tidyBody`·`excerpt`·`mailboxKey`·`emailDomain`/`normalizeEmail`/`charCount` (가치 3 / 위험 1 / 작업량 S)

- 왜: `packages/core/src/__tests__/text.test.ts`(108줄)는 전부 `isSafeLink` 표 두 개뿐이고, 같은 파일의 나머지 export 는 한 줄도 덮여 있지 않다. 그중 `koreanParticle` 은 `util/validate.ts:10-11`(`eul`/`eun` → 모든 검증 오류 문구)과 `mail/templates.ts:71`, `routes/me.ts:26-27` 이 쓰는 **사용자 노출 문자열 생성기**이고, `maskEmail` 은 `auth/accounts.ts:144`·`routes/admin/general.ts:46,143` 이 쓰는 **개인정보 마스킹**이다 — 둘 다 조용히 틀려도 테스트가 아무것도 말해 주지 않는다. 이 저장소는 단위 테스트가 얇아 회귀를 `scripts/e2e.mjs`(실 PostgreSQL 필요)에 의존하는데, 이 함수들은 순수 함수라 DB·브라우저 없이 `npm run check` 하나로 영구 고정할 수 있다. 열린 PR #4·#5·#6 과 파일이 전혀 겹치지 않는다(아래 확인).

- 수용 기준:
  1) `packages/core/src/__tests__/text.test.ts` 에 위 함수들의 엣지케이스 검사가 추가되고 `npm run check` 가 통과한다. 테스트 수가 기준선 **118 tests / 6 files**(main@6ea982c) 보다 늘고 파일 수는 6 그대로다(새 파일을 만들지 말고 기존 `text.test.ts` 에 `describe` 를 덧붙인다).
  2) **프로덕션 코드 변경 0줄.** `git diff --stat` 에 `packages/core/src/__tests__/text.test.ts` 한 파일만 나온다. 테스트를 돌려 보고 "코드가 이상하다" 싶은 곳이 있어도 이번 회차에는 **현재 동작을 그대로 기록**하고(의도가 의심되면 테스트에 주석으로 남긴다) 고치지 않는다 — 고치려면 다음 회차의 별 과제다.
  3) 테스트가 증명해야 하는 것 — 아래는 코드를 읽고 손으로 계산한 기대값이다. **반드시 실행해 확정할 것**(정찰 샌드박스에서 `npm` 실행이 거부돼 미실행이다). 관측값이 다르면 관측값이 맞다 — 그때는 테스트를 관측값에 맞추고 과제서가 틀렸다고 적어 둘 것.
     - `koreanParticle(word, '이', '가')` — 받침 판정: `'서울'`→`'이'`(ㄹ 받침), `'클라우드'`→`'가'`(받침 없음). 괄호·따옴표 건너뛰기: `'제품(베타)'`→`'가'`(마지막 한글 `'타'` 로 판정, `)` 는 무시). 숫자 읽기: `'버전 3'`→`'이'`(삼), `'버전 2'`→`'가'`(이), `'버전 0'`→`'이'`(영). 한글·숫자가 없을 때: `'AWS'`→`'가'`, `''`→`'가'`(= `withoutFinal` 폴백). `text.ts:219-232` 의 `(code-0xac00)%28`·`'136780'.includes(char)` 두 규칙이 둘 다 걸리게 할 것.
     - `maskEmail` — `'hong@example.com'`→`'ho**@ex*****.com'`(주석에 적힌 모양). 한 글자 local: `'a@example.com'`→`'a*@ex*****.com'`(`Math.max(len-keep,1)` 때문에 별이 최소 1개). 두 글자 host: `'hong@ab.com'`→`'ho**@ab*.com'`. 점 없는 도메인: `'hong@localhost'`→`'ho**@lo*******'`(tld 없음). 주소가 없을 때: `null`·`undefined`·`''`→`''`(빈 문자열, `'***'` 가 아니다 — 주석이 이유를 적어 뒀다). `'@'` 가 없거나 맨 앞일 때: `'nope'`→`'***'`, `'@example.com'`→`'***'`.
     - `tidyBody` — `'a\r\nb'`→`'a\nb'`, `'a\rb'`→`'a\nb'`(단독 CR 도). 빈 줄 접기: `'a\n\n\n\n\nb'`→`'a\n\n\nb'`(4개 이상만 3개로), `'a\n\n\nb'`→`'a\n\n\nb'`(3개는 그대로 — 경계). 끝만 다듬음: `'a\n\n  '`→`'a'`, `'  a'`→`'  a'`(앞 공백은 남는다).
     - `excerpt` — 공백 접기: `'  a   b\n\nc  '`→`'a b c'`. 기본 `max` 가 120 이고 경계에서 말줄임이 붙지 않음: `'가'.repeat(120)` 은 그대로(길이 120), `'가'.repeat(121)` 은 `'가'×120 + '...'`. 코드포인트 경계: `excerpt('😀😀😀😀😀', 3)` → 이모지 3개 + `'...'`(UTF-16 반쪽이 나오지 않는다 — `[...flat]` 의 존재 이유).
     - `mailboxKey` — `'Name+Tag@Company.com'`→`'name@company.com'`(정규화 + 태그 제거, '한 메일함 한 번 인증' 규칙). local 이 태그뿐일 때 폴백: `'+tag@company.com'`→`'+tag@company.com'`(`split('+')[0]` 이 빈 문자열이라 원본 유지 — `text.ts:27` 의 `||`). `'@'` 가 여러 개: `'a@b@c.com'`→`'a@b@c.com'`(`lastIndexOf` 기준, 앞쪽 `@` 는 local 에 남는다). `'@company.com'`→`'@company.com'`(`at <= 0` 조기 반환).
     - 작은 것 셋: `normalizeEmail('  Name@Company.COM  ')`→`'name@company.com'`; `emailDomain(' a@B.COM ')`→`'b.com'`, `emailDomain('nope')`→`''`, `emailDomain('a@b@c.com')`→`'c.com'`; `charCount('😀')`→`1`(`'😀'.length` 는 2), `charCount('가나다')`→`3`.

- 건드릴 파일:
  - `packages/core/src/__tests__/text.test.ts` — 기존 `isSafeLink` 표 아래에 함수별 `describe` 를 덧붙인다. import 를 `import { charCount, emailDomain, excerpt, isSafeLink, koreanParticle, mailboxKey, maskEmail, normalizeEmail, tidyBody } from '../util/text.js'` 로 넓힌다(현재는 `isSafeLink` 만, 2번째 줄). 기존 표 작성 방식(`Array<[string, string]>` + `it.each` 대신 `for … it()`)을 그대로 따라 쓸 것 — 파일의 관례다.
  - 끝. **프로덕션 파일 0개** (파일 1개 변경).

- 검증 명령:
  - `npm run check` — eslint + 타입(core·server·web) + `vitest run`. 이것만으로 충분하다(DB·브라우저 불필요).
  - 실패 재현을 요구하는 과제가 아니다(동작 변경이 없다). 대신 **테스트가 실제로 무언가를 잡는지**를 이렇게 증명할 것: `text.ts` 의 `koreanParticle` 에서 `% 28 === 0` 을 `% 28 !== 0` 으로 잠깐 뒤집거나 `maskEmail` 의 `Math.min(2, …)` 를 `3` 으로 바꿔 `npm run check` 가 **실패**하는 것을 보이고, 즉시 되돌린 뒤 다시 통과시킨다(`git diff -- packages/core/src/util/text.ts` 가 비어 있음을 확인).
  - 시간이 있으면 `node scripts/e2e.mjs`(dev-reset 직후 기준선 **218 passed / 0 failed**)로 회귀 없음을 덧붙여도 좋지만, 이 과제는 프로덕션 코드를 건드리지 않으므로 필수는 아니다.

- 위험과 피할 것:
  - **열린 PR 과 파일을 겹치지 말 것.** `git diff --name-only main..<branch>` 로 이번 회차에 확인했다: `auto/2026-10-15-0938`(PR #5) = `__tests__/client-address.test.ts`·`http/pipeline.ts`·`util/text.ts`; `auto/2026-10-14-1413`(PR #4) = `__tests__/quota.test.ts`·`quota.ts`·`scripts/e2e.mjs`; `auto/2026-10-15-1108`(PR #6) = `.github/workflows/check.yml`·`README.md`·`scripts/ci-local.sh`. **`text.test.ts` 는 어느 쪽도 건드리지 않는다** — 그래서 이 과제가 지금 안전하다. 반대로 `util/text.ts` 자체를 고치면 PR #5 와 충돌한다.
  - **`clientIpFrom` 테스트를 쓰지 말 것.** `text.ts:240` 에 아직 있지만 참조가 0건이고 PR #5 가 그 함수를 삭제한다 — 테스트를 쓰면 PR #5 머지 시 빨간 테스트가 된다.
  - 보호 경로(`db/migrations/`·`packages/core/src/auth/*`·`http/pipeline.ts`·`routes/sso.ts`·`.github/workflows/`)를 이번 과제는 전혀 건드리지 않는다. 그대로 두면 된다.
  - `apps/web/src/lib/josa.ts` 에 `koreanParticle` 과 **같은 규칙의 프런트 사본**(`josa`/`wa`/`ro`)이 있다. 보고 싶더라도 **이번에 통합하지 말 것** — 운영자 지시: 같은 종류의 파서가 여러 경로에 있으면 통합하지 말고 좁히는 방향으로만. 이번 과제는 서버 쪽 `text.ts` 함수만 고정한다.
  - 테스트를 "소스 문자열 검사"로 쓰지 말 것(운영자 지시). 전부 실제 함수를 호출해 반환값을 단정할 것.
  - `vitest.config.ts` 의 `testTimeout: 20s` 는 crypto 헬퍼 때문이고 이 과제와 무관하다 — 설정을 건드리지 말 것.

- 차선 후보: **첫 방문 JS 예산 근접(223.7 / 230 KB, 여유 6.3 KB)을 경고로 드러내기** — `scripts/perf.mjs` 가 예산의 95% 를 넘으면 실패는 아니고 눈에 띄는 경고를 찍게 하고(또는 지연 로드 경계 한 곳 추가), 숫자를 README 에 적어 두기. 프로덕션 파일 0~1개, 열린 PR 과 겹치지 않는다. 단 `scripts/perf.mjs` 는 CI 단계라 **실패 판정 기준은 느슨하게 만들지 말 것**(워크플로를 느슨하게 만드는 변경은 금지). 3순위: `mailboxKey`·`domain/products.ts:slugify` 가 **참조 0건의 죽은 코드**라는 것(이번 회차에 `.ts`/`.sql`/`.mjs` 전역 grep 으로 확인 — `mailboxKey` 는 정의 1곳 외 0건) — '한 메일함 한 번 인증' 이 코드로 지켜지지 않는다는 뜻일 수 있어 가치는 높지만(4) 확인하려면 auth·SSO·마이그레이션을 읽어야 해 위험 4다. 조사만 별 회차로 떼어 낼 것.
