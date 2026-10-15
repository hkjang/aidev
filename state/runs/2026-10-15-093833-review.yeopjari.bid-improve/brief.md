- 과제: 방문자 IP 추출이 **빈 헤더에서 폴백하지 않아** 모든 방문자가 쿼터 양동이 하나(`ip:unknown`)를 공유할 수 있는 것 — 살아 있는 경로(`pipeline.ts: clientAddress`)를 고치고 쓰이지 않는 쌍둥이(`util/text.ts: clientIpFrom`)를 지워 파서를 하나로 좁힌다 (가치 3 / 위험 2 / 작업량 S)

- 왜: `packages/core/src/http/pipeline.ts:479` 의 `clientAddress` 는 `headers.get('cf-connecting-ip') ?? headers.get('x-real-ip') ?? (headers.get('x-forwarded-for') ?? '').split(',')[0]?.trim() ?? ''` 라는 `??` 연쇄다. `??` 는 `null`에서만 넘어가므로 앞 헤더가 **빈 문자열**(또는 공백, 또는 `x-forwarded-for` 의 첫 원소가 빈 `", 1.2.3.4"`)이면 뒤 헤더를 보지 않고 `''` 를 반환한다. 그 값이 `hashClient` 를 거쳐 `ipHash` 가 되고, 모든 IP 쿼터가 `quota.ts:152` 의 `ipSubject(ipHash)` 를 쓰므로 영향받는 요청 전부가 **한 주체로 합쳐진다** — 로그인/가입 시도 제한(`routes/auth.ts:253`), 클라이언트 오류 보고(`routes/client-errors.ts:44`), 광고 노출·이벤트 집계(`routes/catalog.ts:575,597,619`), `routes/me.ts:238`. 한 명이 한도를 태우면 나머지 모든 익명 방문자가 429 를 받는다. 고치면 "첫 번째로 비어 있지 않은 헤더"가 주체가 되어 쿼터가 방문자 단위로 돌아오고, 같은 버그를 가진 죽은 복사본이 사라져 다음 사람이 엉뚱한 쪽을 고치는 일이 없어진다.

- 수용 기준:
  1) `cf-connecting-ip` 가 **빈 문자열이거나 공백뿐**일 때 `x-real-ip` → `x-forwarded-for` 순으로 넘어가고, 셋 다 비어 있을 때만 `''` 를 반환한다.
  2) `x-forwarded-for` 의 첫 원소가 비어 있을 때(`", 203.0.113.7"`) 빈 값이 아니라 `203.0.113.7` 을 집는다. (첫 원소가 비지 않았으면 지금처럼 첫 원소를 쓴다 — 여러 개 중 **첫 번째**를 고르는 기존 정책은 바꾸지 않는다.)
  3) `trustProxy === false` 면 헤더가 무엇이든 `''` 를 반환한다(현재 동작 유지 — 회귀 금지).
  4) `util/text.ts` 의 `clientIpFrom` 이 사라지고, 저장소 어디서도 참조되지 않는다(`grep -rn clientIpFrom --include=*.ts . --exclude-dir=node_modules` 가 0건). 파서는 `pipeline.ts` 쪽 하나만 남는다.
  5) 테스트가 증명하는 것: 위 1)~3)이 **실제 `Request` 객체**(`new Request('http://x/', { headers: … })`)로 호출한 결과로 확인되고, 고치기 전 코드에서는 1)·2)가 실패한다(구현자는 실패를 먼저 관측해 노트에 남길 것).

- 건드릴 파일 (프로덕션 2개 + 테스트 1개):
  - `packages/core/src/http/pipeline.ts:479 clientAddress` — `??` 연쇄를 "후보 헤더를 순서대로 보며 `trim()` 후 비어 있지 않은 첫 값을 쓴다" 로 바꾼다. **함수를 다른 파일로 옮기지 말고 그 자리에서 고칠 것.** 단위 테스트를 위해 `export` 를 붙이는 것까지만 허용(이 파일의 다른 로직·`resolvePrincipal`·GUC 설정·auth 분기는 손대지 말 것).
  - `packages/core/src/util/text.ts:240 clientIpFrom` — 함수와 그 주석(`SEC-05` 언급) 삭제. 삭제 전에 수용 기준 4)의 grep 으로 참조 0건을 다시 확인할 것. 참조가 하나라도 나오면 삭제하지 말고 그 자리에 `pipeline.ts` 와 같은 수정만 적용하고 과제서에 적어 둘 것.
  - `packages/core/src/__tests__/client-address.test.ts` (신규) — `clientAddress` 를 `../http/pipeline.js` 에서 import 해 1)~3)을 덮는다. 기존 5개 테스트 파일과 같은 vitest 스타일(`describe`/`it`/`expect`)을 따를 것.

- 검증 명령:
  - `npm run check` — eslint + 타입(core·server·web) + `vitest run`. **기준선 118 tests / 7 files**(main@6ea982c, `quota.test.ts` 없음). 신규 테스트만큼 늘어야 한다.
  - `node scripts/e2e.mjs` — `bash scripts/dev-reset.sh` 직후 **218 passed / 0 failed** 가 기준선. 이 과제는 e2e 수치를 바꾸지 않아야 한다(아래 참조).
  - 준비가 필요하면: docker postgres:16-alpine(`127.0.0.1:55443`) → `npm ci` → `node scripts/ci-env.mjs` → `npm run build -w @review/server` → `bash scripts/dev-reset.sh`.

- 위험과 피할 것:
  - **`packages/core/src/quota.ts` 를 절대 건드리지 말 것.** 미머지 PR #4(`auto/2026-10-14-1413`)가 바로 그 파일의 `retryAfterSeconds`/`nextWindowReset` 을 고친다. 여기서 같이 손대면 충돌하고, 같은 수정을 세 번째로 올리는 중복이 된다.
  - **e2e 로는 이 결함을 증명할 수 없다**: `packages/core/src/env.ts:113` 이 `trustProxy: bool(raw.REVIEW_TRUST_PROXY, !dev)` 라 로컬·CI(dev)에서는 기본 `false` 이고 `clientAddress` 가 항상 `''` 를 돌려준다. 즉 e2e 는 이 코드를 아예 실행하지 않는다. e2e 를 통과시키려고 `REVIEW_TRUST_PROXY` 를 켜거나 `scripts/e2e.mjs` 의 서버 기동을 바꾸지 말 것 — 범위 밖이고 다른 검사의 전제를 흔든다. e2e 는 **회귀 확인용**으로만 돌린다(218/0 유지).
  - 파서를 "통합" 하지 말 것 — 운영자 지시: 같은 종류의 파서가 여러 경로에 있으면 좁히는 방향으로만 고친다. 여기서 허용되는 좁히기는 **죽은 복사본 삭제**뿐이다. `pipeline.ts` 가 `util/text.ts` 의 함수를 import 하도록 바꾸는 방향은 금지.
  - `ipHash` 는 감사·쿼터 주체다. `hashClient` 의 호출 방식이나 `env.keys.session` 사용을 바꾸지 말 것 — 바꾸면 배포 시점에 진행 중인 모든 IP 쿼터 카운터의 주체가 갈린다.
  - 미확인으로 남긴 것: ① `hashClient('' , key)` 가 `''` 를 돌려주는지 digest 를 돌려주는지 확인 못 했다(`crypto.ts` 에 없었다 — `grep -rn "hashClient" --include=*.ts packages` 로 찾을 것). 어느 쪽이든 "모든 방문자가 한 주체로 합쳐진다" 는 결론은 같다(빈 값이면 `ip:unknown`, 상수 digest 면 그 digest 하나). ② Cloudflare 가 `cf-connecting-ip` 를 **빈 문자열로** 보내는 일이 실제로 있는지는 확인하지 못했다 — 이 과제는 "빈 헤더를 없는 헤더로 취급한다" 는 방어적 정정이고, 2)의 `x-forwarded-for` 첫 원소 빈 값은 프록시 설정에서 흔하다.
  - 보호 경로(`db/migrations/`, `auth/*`, `.github/workflows/`, `scripts/deploy-pages.sh`)는 건드리지 않는다. 이 과제는 그럴 이유가 없다.
  - CI(`check` 잡)는 2026-10-14 01:09Z 부터 **저장소 밖의 이유**(계정 Actions 결제/쿼터로 추정)로 단계 실행 없이 3초 만에 실패한다 — 5개 무관한 저장소가 동시에 같은 증상. CI 가 빨간 것을 이번 변경의 탓으로 오인하지 말고, 로컬 `npm run check` + `node scripts/e2e.mjs` 출력으로 증명할 것.

- 차선 후보: **`util/text.ts` 나머지 순수 함수 단위 테스트 보강** (가치 3 / 위험 1 / 작업량 S, 프로덕션 파일 0개) — `packages/core/src/__tests__/text.test.ts`(108줄)는 `isSafeLink` 중심이고 다음이 전혀 안 덮여 있다: `mailboxKey`(`+tag` 제거, local-part 가 `+tag` 뿐일 때의 폴백, `a@b@c.com`), `maskEmail`(`null`/`undefined`, 점 없는 도메인, 한 글자 local, `@` 로 시작), `koreanParticle`(종성 있음/없음, 끝이 숫자 `1·2`, 끝이 `)`·`」`, 한글 없는 단어), `excerpt`(코드포인트 경계·공백 접기·`max` 경계 정확히 120), `tidyBody`(`\r\n`→`\n`, 빈 줄 4개 이상 접기, `trimEnd`), `emailDomain`/`normalizeEmail`/`charCount`. `mailboxKey` 는 "한 메일함 한 번 인증" 제품 규칙을 지탱하는 함수라 계약을 테스트로 고정할 가치가 있다. 검증은 `npm run check` 하나로 끝난다(DB·브라우저 불필요).
