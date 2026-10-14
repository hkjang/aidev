- 과제: 금액·정수·소수 입력이 '0x10'·'1e3' 같은 비십진 리터럴을 조용히 받는 것 차단 (가치 4 / 위험 2 / 작업량 S)
- 왜: `packages/core/src/util/validate.ts` 의 `v.money`/`v.int`/`v.decimal` 은 문자열을 `Number()` 로 바로 변환한다. JS 의 문자열→수 변환은 16진('0x10'→16)·2진('0b101'→5)·8진('0o17'→15)·지수('1e3'→1000) 리터럴을 모두 받아들이므로, 공급가·수금액·차변/대변 같은 금액 필드에 `"0x10"` 이 들어오면 검증 오류가 아니라 **16원**으로 조용히 통과해 불변 원장에 기록된다. 허용 문자열을 십진 리터럴로 좁히면 같은 입력이 기존 한국어 오류 메시지로 거절된다(새 메시지·새 분기 없음).
- 수용 기준:
  1) `parse({ supply_amount: v.money({ label: '공급가', min: 1 }) }, { supply_amount: '0x10' })` 가 `validation_failed`(`AppError`)로 던져진다. 수정 전에는 `{ supply_amount: 16 }` 을 반환한다 — 이 차이를 시험이 먼저 실패로 보여야 한다.
  2) `'1e3'`, `'0b101'`, `'0o17'`, `'Infinity'`(지금도 거절), `'1_000'`(지금도 NaN 거절)에 대해 money·int·decimal 세 검증기 모두 거절한다. 메시지는 기존 문구(`… 원 단위 정수여야 합니다.` / `… 정수여야 합니다.` / `… 숫자여야 합니다.`)가 그대로 나온다.
  3) 지금 통과하는 입력은 그대로 통과한다: 숫자 타입 `1000`/`0`, 문자열 `'1000'`, `'1,000,000'`, `' 1000 '`(money 는 쉼표·공백 제거), money 에 `'12.0'`→`12`, decimal 에 `'12.5'`→`12.5`, `v.int` 에 `' 12 '`→`12`, money 상한 `999999999999` 통과·`1000000000000` 거절, `min` 위반 메시지 유지.
  4) 시험이 가드를 실제로 보고 있음을 증명한다: 새로 넣은 십진 리터럴 정규식 검사를 임시로 제거하면 (1)(2)의 케이스가 다시 실패하고, 복원하면 전원 통과한다(실패 출력을 회차 기록에 남길 것).
- 건드릴 파일 (프로덕션 1개 + 시험 1개):
  - `packages/core/src/util/validate.ts` — 파일 상단(`UUID_RE`/`EMAIL_RE` 옆)에 십진 리터럴 정규식 하나와 작은 도움 함수를 두고, `v.money`·`v.int`·`v.decimal` 의 `const n = typeof value === 'string' … ? Number(…) : value` 한 줄씩만 그 함수 호출로 바꾼다. 권장 형태(문자열이 십진 리터럴이 아니면 **원래 문자열을 그대로 돌려줘** 기존 `typeof n !== 'number'` 분기가 기존 메시지로 거절하게 한다 — 새 메시지를 만들지 말 것):
    ```ts
    const DECIMAL_RE = /^[+-]?(\d+(\.\d*)?|\.\d+)$/
    function asNumber(value: unknown, stripSeparators: boolean): unknown {
      if (typeof value !== 'string') return value
      const cleaned = stripSeparators ? value.replace(/[,\s]/g, '') : value.trim()
      if (cleaned === '' || !DECIMAL_RE.test(cleaned)) return value
      return Number(cleaned)
    }
    ```
    `v.money`(validate.ts:236 부근)·`v.decimal`(:252 부근)은 지금 쉼표·공백을 제거하므로 `stripSeparators: true`, `v.int`(:97 부근)은 지금 제거하지 **않으므로** `false`(= trim 만) 로 호출한다. 각 검증기의 나머지 검사(`Number.isInteger`, min/max, `decimal` 의 `places` 반올림)는 손대지 말 것.
  - `packages/core/src/__tests__/domain.test.ts` — 끝에 `describe('입력 검증: 금액·정수·소수')` 를 추가. 시험은 라우트가 실제로 쓰는 진입점(`parse`, `parseOne`)과 실제 검증기 구성을 그대로 통과시켜야 한다. import 는 core 관례대로 `.js` 확장자: `import { parse, parseOne, v } from '../util/validate.js'`. 거절 확인은 `expect(() => parse(...)).toThrow()` 로 하되, 한 케이스에서는 오류 객체의 모양까지 확인할 것 — `errors.ts:54` 의 `AppError` 는 `code`(여기서는 `'validation_failed'`)와 `fieldErrors`(`{ [경로]: 메시지 }`, `errors.ts:80` 의 `invalid` 가 채움)를 공개 필드로 갖는다. 예: `try { parse({ supply_amount: … }, { supply_amount: '0x10' }) } catch (e) { expect((e as AppError).code).toBe('validation_failed'); expect((e as AppError).fieldErrors?.supply_amount).toContain('원 단위 정수') }` — `expect.fail()/assert` 로 "던지지 않음" 도 잡을 것. `AppError` 는 이미 `domain.test.ts:17` 에서 import 하고 있다.
- 검증 명령 (이 순서대로, 이 저장소에서 실제로 도는 것):
  1) `npm ci --no-audit --fund=false` (워크트리에 node_modules 가 없다. ESLint 지원 종료 경고 1건은 정상)
  2) `npm test -- packages/core/src/__tests__/domain.test.ts` (먼저 실패를 확인 → 수정 → 통과)
  3) `npm run check` (= `eslint .` + 세 workspace tsc + 전체 vitest; 직전 회차 기준 113개 시험)
  4) `npm run build`
  - `npm run e2e`/`ui`/`screens`/`a11y` 는 실제 PostgreSQL·fake OAuth·Playwright 가 필요해 이번 회차에서 돌리지 않는다.
- 위험과 피할 것:
  - **넓히지 말고 좁히기만 할 것.** `v.int` 에 쉼표 제거를 새로 넣거나 money 의 제거 규칙을 바꾸면 같은 값이 경로마다 다르게 읽힌다(운영자 지시: 같은 종류의 파서는 통합·확장하지 말고 좁히는 방향으로만).
  - 같은 파일의 `v.day`·`v.string`·`v.text`·`v.email`·`v.instant` 는 건드리지 말 것. `v.day` 는 이미 형식+ISO 왕복+2000~2100 범위로 올바르게 동작함을 이번 정찰에서 읽어 확인했다.
  - `packages/core/src/routes/finance.ts`·`sales.ts`·`approvals.ts` 등 호출부는 **읽기만** 할 것(금액 필드가 어디서 들어오는지 확인용). 라우트·마이그레이션·`auth/*`·`routes/sso.ts`·`.github/workflows` 는 이번 범위 밖.
  - `v.optional(v.money(...), 0)` 형태가 많다(`finance.ts:668` 차변/대변). 기본값 경로가 그대로인지 시험에 한 케이스 넣을 것.
  - 음수: money 는 기본 `min = 0` 이라 `'-1'` 은 지금도 min 위반으로 거절된다. 정규식에 `[+-]?` 를 넣는 것은 int/decimal(할인율 등 min 지정)의 기존 동작을 유지하기 위한 것이며, 부호 자체의 허용 범위를 바꾸려 하지 말 것.
  - **미확인**: 이번 정찰에서는 node 실행이 허용되지 않아 `Number('0x10') === 16` 을 이 저장소에서 직접 실행해 보지는 못했다(ECMAScript 문자열→수 변환 규칙에 따른 판단). 구현자는 수용 기준 (1) 의 사전 실패 출력으로 결함이 실제로 재현되는지 **먼저** 확인하고, 재현되지 않으면 차선 후보로 넘어갈 것.
  - 소스 문자열 검사나 손으로 만든 대역을 증거로 삼지 말 것 — 위 (1)(2)처럼 실제 `parse`/`parseOne` 반환값·예외로 증명하고, 가드 제거 변이로 시험의 유효성까지 보일 것.
- 차선 후보: 달력 날짜 검증 `v.day` 의 윤년·월말·범위 회귀 시험 보강 (가치 3 / 위험 1 / 작업량 S) — 프로덕션 변경 0파일. `domain.test.ts` 에 `parseOne(v.day({ label: '지출일' }), …)` 로 통과(`2024-02-29`, `2000-01-01`, `2100-12-31`)·거절(`2023-02-29`, `2026-04-31`, `2026-02-30`, `2026-1-1`, `20260101`, `1999-12-31`, `2101-01-01`, 숫자·null) 표를 못 박고, `d.toISOString().slice(0,10) !== value` 조건을 임시 제거해 시험이 가드를 보는지 확인하면 끝난다.
