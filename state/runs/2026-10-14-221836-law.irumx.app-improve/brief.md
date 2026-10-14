- 과제: 수정 과제 — PR #6 CI 실패를 **축소 경로로 확정**하고(전수 재현 금지), 그 뒤 `env.ts loadEnv` 전반 특성화 테스트 집합을 추가 (가치 3 / 위험 1 / 작업량 M)
- 왜: 우선 과제로 적재된 PR #6(`auto/2026-10-14-1728` / `6d6f810`, 내용은 테스트 파일 1개 459줄)의 `check` 실패는 **저장소 밖**이다 — 잡이 3초에 끝나고 실패 로그가 비어 `npm ci` 에도 닿지 못했으며, 같은 신호였던 PR #4 의 체크런 주석 원문이 `"The job was not started because recent account payments have failed or your spending limit needs to be increased"` 로 보존돼 있다(2026-10-14-110037 회차). 직전 수리 회차가 CI 순서를 로컬 docker 로 **끝까지 돌려 코드 무결을 이미 증명**했으므로(`…-175134-law.irumx.app-shepherd/fix-summary.md:4`: npm ci → check 7 files/193 tests → build → migrate 13건 → rls-probe exit 0 → e2e 27/27 → acceptance 26/26), 네 번째 전수 재현에 회차를 또 태우면 비용만 나간다. 그래서 확인은 싸게 끝내고, 남는 시간을 `loadEnv` 의 테스트 공백(현재 `env-mock-providers.test.ts` 7건이 `providers.domains` 한 축만 덮는다)에 쓴다.

- 수용 기준:
  1) **CI 원인 확정(싸게)**: `.github/workflows/check.yml` 을 읽고 3초·빈 로그를 만들 수 있는 저장소 쪽 원인이 없음을 확인한다(이 회차 정찰이 읽어 둠: `runs-on: ubuntu-latest`, 서비스 컨테이너 postgres:17-alpine, 첫 실행 step 이 `apt-get install postgresql-client-17` — 어느 step 이든 들어갔으면 3초를 넘는다. YAML 은 유효하고 `actions/checkout@v4`·`setup-node@v4`·`upload-artifact@v4` 모두 현행). `gh auth status` 가 인증돼 있으면 `gh run list --branch auto/2026-10-14-1728` / `gh api repos/hkjang/law.irumx.app/check-runs/<id>/annotations` 로 주석 원문을 받아 **그대로 인용**하고, 미인증이면 "미확인(세션 `gh` 미인증)" 으로 적고 위 보존 증거를 근거로 "저장소 밖 — Actions 결제/지출한도, 사람이 Billing & plans 에서 복구 후 같은 커밋 재실행" 으로 결론낸다. **워크플로·검증 스크립트를 느슨하게 고쳐 초록을 만드는 것은 금지** — 고칠 코드가 없으면 그렇게 적는다.
  2) 신규 `packages/core/src/__tests__/env-loading.test.ts` 가 `loadEnv`/`EnvError` 를 실제로 import 해 호출하며(대역 없음), 아래 축을 **현재 동작 그대로** 고정한다. `env.ts` 는 한 줄도 고치지 않는다(프로덕션 0파일).
  3) 테스트가 증명해야 할 것(소스 근거는 괄호의 줄 번호 — 실제로 열어 확인했다):
     - **DB URL**: `LAW_DATABASE_URL` 없으면 `EnvError`(:108), 그런데 `DATABASE_URL` 이 있으면 그것으로 통과한다(:107 의 `??` 별칭 — 지금 테스트 0건). 공백만 있는 값은 `str()` 이 트림해 부재로 본다(:216).
     - **키 3종**(:110-113, `resolveKey` :194): 운영에서 없으면 `EnvError` 메시지에 `LAW_SESSION_KEY is required` 가 들어가고, **dev 에서는 던지지 않고 결정적(랜덤 아님) 32바이트 파생 키**가 나온다(:210-213 — 두 번 호출하면 같은 바이트, `session`/`secrets`/`match` 는 서로 다름). 비(非)hex 또는 홀수 길이 → `must be 64 hex characters`(`util/crypto.ts:37` 의 `fromHex` 가 던지고 :200 이 바꿔 던진다), 짝수지만 32바이트가 아닌 값(예 62자) → `must decode to 32 bytes, got 31`(:202). 대문자 hex 는 허용(`fromHex` 정규식이 `a-fA-F`).
     - **서명 키**(:123-124, :152-153): 운영에서 없으면 `EnvError`, dev 에서는 저장소에 든 `DEV_SIGNING_KEY` 로 떨어지고 `signingKeyId === 'dev-insecure'`; 키를 주면 기본 `signingKeyId === 'law-1'`.
     - **publicUrl 파싱**(:115-121, :143·147): 기본값 `http://localhost:8810` → `scheme 'http'`·`devPort ':8810'`·`platformHost 'localhost'`; 끝의 `/` 는 여러 개여도 제거(`replace(/\/+$/,'')`); 대문자 호스트는 소문자화; `opsHost` 기본 `ops.<host>`, `siteSuffix` 기본 `<host>`, `cnameTarget` 기본 `sites.<host>`, `opsUrl === '<scheme>://<opsHost><devPort>'`; `https` URL 은 `devPort ''`. **엣지: 잘못된 URL 은 `EnvError` 가 아니라 `new URL()` 의 `TypeError` 로 터진다(:116)** — 바꾸지 말고 현재 동작으로 기록만(`expect(() => …).toThrow()` 로 두고 `EnvError` 가 아님을 못박아도 된다).
     - **dev 파생 기본값**: `trustProxy === !dev`(:150), `internalKey` 가 운영에서 `''`·dev 에서 `'dev-internal-key'`(:184), `build.channel` 이 `release`/`dev`(:189).
     - **`bool()` 표**(:221-226): `1`·`true`·`yes`·`on`·`TRUE`(대문자) → true, `0`·`false`·`no`·`off` → false, 그 밖(`yep`, `''`, 공백) → **fallback**.
     - **`enumOf()` 표**(:228-231): 목록 밖·빈 값은 조용히 fallback(`LAW_MAIL_DRIVER=sendgrid` → `console`, `LAW_MEDIA_DRIVER=gcs` → `fs`), 대문자 입력은 소문자화해 매칭(`RESEND` → `resend`).
  4) `env-mock-providers.test.ts` 의 `prodRaw()` 헬퍼와 같은 모양(HEX32 = `'a'.repeat(64)`)을 새 파일에 **복사해 쓴다** — 기존 파일을 고쳐 export 하지 말 것(그 파일은 건드리지 않는다).
  5) 특성화 테스트라 첫 실행에 전부 통과하는 것이 정상이다. 대신 **돌연변이로 경로를 증명**하라: `env.ts` 에서 한 줄씩 지워 보고(예 :107 의 `?? raw.DATABASE_URL` 제거, :150 의 `!dev`→`dev`, :224 의 `off` 목록 제거, :153 의 `'law-1'`→`'dev-insecure'`) 각각 몇 건이 실패하는지 적고 **복원 후 `git diff` 가 비었음을 확인**해 원장에 쓴다.

- 건드릴 파일 (프로덕션 0개 / 신규 테스트 1개):
  - `packages/core/src/__tests__/env-loading.test.ts` — 신규. 위 7개 축을 `describe` 로 나눈 표 테스트.
  - 읽기만: `packages/core/src/env.ts`(`loadEnv`·`resolveKey`·`str`·`bool`·`enumOf`), `packages/core/src/util/crypto.ts:35 fromHex`, `.github/workflows/check.yml`.

- 검증 명령:
  - `npm run test` (= `vitest run`, DB 불필요). 좁히려면 `npx vitest run packages/core/src/__tests__/env-loading.test.ts`.
  - `npm run check` (lint + typecheck + test) — 기준선은 **lint 0 errors / 646 warnings**(신규 파일 경고 0 이어야 함), typecheck 3 워크스페이스, vitest **6 files 103 tests**(main@c62b3a0 기준 — `redact-secrets`·`background-redact`·`validate-boundaries` 는 아직 미머지라 이 트리에 없다).
  - 참고: 이번 정찰 세션은 샌드박스 승인 제한으로 `npx vitest run` 과 `gh` 를 **실행하지 못했다 — 기준선 숫자는 원장 인용이고 이 세션에서 재확인하지 않았다(미확인)**. 구현자는 먼저 손대기 전 `npm run test` 를 한 번 돌려 실제 기준선을 적어 둘 것.

- 위험과 피할 것:
  - `.github/workflows/check.yml`, `scripts/acceptance.mjs`, `scripts/e2e.mjs` 를 **완화하는 방향으로 고치지 말 것**(명시 금지). 결제 차단은 코드로 못 고친다.
  - `env.ts` 의 조건을 넓히거나 좁히지 말 것 — 여기서 던지면 워커가 기동하지 못한다. 특히 :116 의 `new URL()` 을 `EnvError` 로 감싸고 싶어지지만 **이번 회차 범위 밖**(동작 변경이다). 기록만 하고 아이디어로 남겨라.
  - `env-mock-providers.test.ts`(기존 7건)는 수정·이동·리네임 금지. 미머지 브랜치 3개가 각각 `packages/core/src/__tests__/` 에 파일을 추가하므로 파일명 `env-loading.test.ts` 를 바꾸지 말 것(충돌 회피).
  - dev 파생 키 테스트에서 **실제 바이트 값을 하드코딩하지 말고** 길이·결정성·세 용도 간 차이만 단언하라(:210 의 시드 문자열을 테스트에 복사하면 구현을 복사한 테스트가 된다).
  - DB·브라우저가 필요한 `e2e`/`acceptance`/`ui-flows`/`test:rls` 는 이번 과제에 불필요 — 돌리지 말 것(시간).

- 차선 후보: `v` 검증기의 남은 멤버 경계 테스트 중 **`v.oneOf`·`v.bool`·`v.json` 세 개만** (`packages/core/src/util/validate.ts`, 신규 `packages/core/src/__tests__/validate-enum-json.test.ts`). `validate-boundaries.test.ts` 는 PR #6 에 미머지로 묶여 있으니 **그 파일명을 쓰지 말고** 다루는 멤버도 겹치지 않게(`string`·`text`·`int`·`optional` 제외) 할 것.
