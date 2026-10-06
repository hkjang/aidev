# 과제서 (2026-10-06 정찰)

- 과제: `isSafeLink` 가 주석에 적힌 대로 사설·루프백 대역을 실제로 막게 하기 (가치 4 / 위험 2 / 작업량 S)

- 왜: `packages/core/src/util/text.ts:60` `isSafeLink` 의 주석은 "메타데이터 주소와 **명백한 사설 대역**을 막아, 회원 브라우저에서 렌더된 링크로 내부망을 탐색할 수 없게 한다"고 적혀 있지만, 코드가 실제로 거르는 호스트는 `169.254.169.254` 와 `metadata.google.internal` 두 개뿐입니다. `http://127.0.0.1:8796/...`, `http://192.168.0.1/`, `http://[::1]/`, `http://10.0.0.5/` 는 전부 통과합니다. 이 함수는 **사용자·공급사·관리자가 넣는 모든 주소의 단일 관문**(아래 호출부 10곳 전부가 이 함수를 지남)이라, 한 파일만 고치면 모든 입력 경로가 함께 막힙니다.

- 수용 기준:
  1) `isSafeLink` 가 다음을 `false` 로 돌린다 — 루프백(`127.0.0.0/8`, `localhost`, `*.localhost`, `[::1]`), 링크 로컬(`169.254.0.0/16`, `[fe80::…]`), 사설(`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), `0.0.0.0`/`0`, 유니크 로컬 IPv6(`fc00::/7`), 호스트가 빈 주소(`http:///foo`), 그리고 10진수·16진수로 쓴 같은 주소(`http://2130706433/`).
  2) 지금 통과해야 하는 주소는 계속 통과한다 — `https://www.notion.com/pricing`, `https://10.notion.com/`(사설 대역처럼 보이지만 호스트명), `http://localhost.example.com/`(`localhost` 접미사 아님). 즉 **정확히 `localhost` 또는 `.localhost` 로 끝나는 경우만** 막아야 하고, 점으로 구분된 라벨 접미사 비교로 해야 합니다.
  3) 새 단위 테스트가 위 1)·2) 목록을 표로 돌려, "주석이 말하는 것"과 "코드가 하는 것"이 같아졌음을 증명한다. 테스트는 프로덕션 함수 `isSafeLink` 를 그대로 import 해서 쓰고, 별도 대역(fake)·복사한 판정 로직을 만들지 마세요.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `packages/core/src/util/text.ts:60` `isSafeLink` — 호스트 판정을 추가. `ALLOWED_LINK_PROTOCOLS` 검사는 그대로 두고, 그 뒤 `url.hostname` 을 보는 부분을 넓히세요. 지금의 두 하드코딩(`169.254.169.254`, `metadata.google.internal`)은 새 규칙에 흡수되거나 그대로 남겨도 됩니다. **주석도 코드와 맞게 고치세요** (지금은 코드가 안 하는 일을 적고 있음).
  - `packages/core/src/__tests__/` 아래 새 파일 (예: `text.test.ts`) — vitest `include` 가 `packages/**/*.test.ts` 이므로 이 위치면 자동으로 잡힙니다. 기존 `domain.test.ts` 의 `describe`/`it`/`expect` 스타일을 따르세요.
  - 이 둘 말고는 건드리지 마세요. 호출부(아래)는 수정 불필요합니다.

- 호출부 (전부 `isSafeLink` 를 지남 — 확인함, 수정하지 말 것):
  - `packages/core/src/routes/me.ts:24` `checkUrl` → `:104` 공급사 제안 `website`, `:154` 정정 요청 `source_url` (회원 입력)
  - `packages/core/src/routes/vendor.ts:63` `url()` / `:67` `checkUrls` → `:153`, `:211`(`website`·`pricing_url`·`docs_url`·`calculator_url`), `:250`, `:296`, `:318`, `:447` (공급사 입력)
  - `packages/core/src/routes/admin/ads.ts:35` `checkAd` (서비스 안 경로 `/…` 는 `isSafeLink` 를 건너가므로 그 분기는 그대로)
  - `packages/core/src/routes/admin/catalog.ts:39`

- 검증 명령 (package.json 의 실제 스크립트):
  - `npm run test` (= `vitest run`) — 새 테스트가 도는 곳. **정찰은 이 명령을 실행하지 못했습니다(샌드박스 승인 거부). 구현자가 먼저 변경 없이 한 번 돌려 기준선을 확인하세요.**
  - `npm run lint` (eslint), `npm run typecheck` (core→server→web)
  - `npm run check` (= lint + typecheck + test) 가 CI 1단계와 같은 명령입니다 (`.github/workflows/check.yml`).
  - e2e(`node scripts/e2e.mjs`)는 PostgreSQL 16 이 `127.0.0.1:55443` 에 떠 있어야 합니다. **e2e 가 넣는 주소 payload 는 전부 `https://www.notion.com/...`·`https://www.atlassian.com/...` 형태라(확인함) 이 변경으로 깨지지 않아야 합니다.** 로컬에 DB가 없으면 돌리지 말고, 위 세 명령으로 마치세요.

- 위험과 피할 것:
  - **WHATWG `URL` 의 호스트 정규화를 가정하지 말고 직접 확인하세요.** 정찰은 `node -e` 실행 승인을 못 받아 **미확인**입니다. 구현 전에 이것부터 돌려 실제 `hostname` 값을 보고 판정을 짜세요:
    `node -e "for (const u of ['http://2130706433/','http://[::1]/','http://[::ffff:127.0.0.1]/','http://127.1/','http://0/','http://localhost/','http://10.notion.com/','http:///foo','http://[fe80::1]/']) { try { console.log(u, JSON.stringify(new URL(u).hostname)) } catch { console.log(u,'THROW') } }"`
    특히 IPv6 는 `hostname` 이 대괄호를 **포함**할 가능성이 큽니다(`"[::1]"`). 문자열 비교로 짜면 여기서 어긋납니다.
  - 과차단이 이번 과제의 유일한 실패 모드입니다. `10.notion.com`·`localhost.example.com`·`172.200.0.1`(사설 아님, 172.16~172.31 만 사설)을 막으면 공급사 콘솔에서 정상 주소가 거부됩니다. 수용 기준 2) 를 테스트로 먼저 고정하세요.
  - `db/migrations/`, `packages/core/src/auth/`, `.github/workflows/`, `packages/core/vendor/`(postgres 벤더 복사본)는 건드리지 마세요.
  - 이 변경은 "차단 목록을 넓히는" 것이므로 DNS 조회·네트워크 호출을 새로 넣지 마세요. `isSafeLink` 는 동기 순수 함수이고 workerd 에서도 같은 코드로 돕니다. 호스트명 뒤에 숨은 사설 IP(DNS rebinding)는 이 과제의 범위가 아니며, 그렇게 적어 두세요.

- 차선 후보: **`util/text.ts` 순수 함수 단위 테스트 보강** — `mailboxKey`(`+tag` 제거, `+` 로 시작하는 local-part 의 폴백 분기), `maskEmail`(점 없는 도메인, 한 글자 local), `koreanParticle`(한글 종성 유무, 숫자 `136780` 분기, 괄호로 끝나는 이름), `excerpt`(코드포인트 기준 자르기), `tidyBody`(`\r\n` 정규화, 빈 줄 4개 이상). 이 파일에는 지금 단위 테스트가 **하나도 없습니다**(`domain.test.ts` 가 `text.js` 를 import 하지 않음). 1순위와 건드릴 파일이 겹치므로 둘 중 하나만 하세요.
