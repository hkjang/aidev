# 과제서 (2026-10-06 정찰)

- 과제: 브라우저 오류 알림이 한 경로에서 서로 다른 오류를 삼킨다 — 중복 묶음 서명에 오류 종류·내용을 넣는다 (가치 4 / 위험 2 / 작업량 S)

- 왜: `packages/core/src/routes/client-errors.ts` 는 모든 브라우저 보고를 `new Error(...)` 로 만들어
  `alertOperator` 에 넘기는데, `AppError` 가 아니므로 `packages/core/src/http/pipeline.ts:164` 에서
  `code` 가 항상 `'internal'` 이 된다. 그래서 중복 묶음 서명이
  `` `${method} ${normalisePath(path)} ${code}` `` (pipeline.ts:166) = **`CLIENT /today internal`** 하나로,
  `kind` 가 error/unhandled/render 중 무엇이든, 메시지가 무엇이든 같은 경로의 모든 브라우저 실패가 한 행에 합쳐진다.
  `your_priv.record_error_alert` (db/migrations/0002_security.sql:458) 은 `signature` 를 PK 로 upsert 하고
  창(window) 안에서는 `send=false` 를 돌려주므로, `/today` 에서 흔한 오류 하나가 그 창 동안
  **같은 화면의 다른 모든 오류 메일을 막는다** (`last_message` 만 덮어쓰이고 occurrences 만 오른다).
  "화면이 members 에게 깨지면 아무도 말하지 않아도 알게 된다"(커밋 550bc5d)는 기능의 목적이 바로 이것 때문에 무너진다.
  종류·내용을 서명에 넣으면 서로 다른 고장이 각각 한 번씩 메일로 온다.

- 수용 기준:
  1) 같은 경로(`/today`)로 서로 다른 메시지의 브라우저 오류 두 건을 보고하면 `your_priv.error_alerts` 에
     **서로 다른 signature 행 2개**가 생긴다. 같은 메시지를 두 번 보고하면 행은 1개이고 `occurrences = 2` 다.
  2) `kind` 만 다른 두 보고(같은 경로·같은 메시지)도 서명이 다르다.
  3) 서버 자신의 실패(`AppError`/`Error` 가 라우트에서 터진 경우)의 서명은 **바이트 단위로 지금과 같다**.
     기존 `error_alerts` 행의 occurrences·창이 끊기지 않아야 하므로, 추가 조각은 기본값 `''` 이고
     기본값일 때 뒤에 공백조차 붙지 않아야 한다.
  4) 기존 e2e 검사 `signature LIKE 'CLIENT /chat/:id%'` (`scripts/e2e.mjs:800`) 가 계속 통과한다
     — 즉 추가 조각은 서명 **끝에** 붙이고, `CLIENT ` 접두사와 경로 자리를 바꾸지 않는다.
  5) 단위 테스트가 증명할 것: 서명을 만드는 순수 함수가 (a) 추가 조각이 없으면 예전 문자열을 그대로 내고,
     (b) 메시지·kind 가 다르면 다른 서명을 내고, (c) 같은 입력에는 같은 서명을 낸다(안정적 — 매 보고마다
     달라지면 중복 묶음이 아예 사라져 메일 폭탄이 된다).

- 건드릴 파일 (프로덕션 3개):
  - `packages/core/src/http/pipeline.ts`
    - `alertOperator(deps, requestId, path, method, err)` 에 **마지막 선택 인자** 하나 추가
      (예: `signatureExtra = ''`). 서명 계산을 순수 함수로 꺼내 export 하는 것을 권함:
      `export function alertSignature(method: string, path: string, code: string, extra = ''): string`
      — `extra === ''` 이면 지금과 똑같은 `` `${method} ${normalisePath(path)} ${code}` `` 를 돌려야 한다.
      본문 166행과 171행이 **같은 서명 문자열**을 쓰도록(두 곳에서 따로 조립하지 말 것 — 지금은
      `normalisePath(path)` 를 166·171·186 세 번 따로 부른다) 한 번 계산해 변수에 담는다.
    - 메일(`path:` 186행)과 `record_error_alert` 의 `path` 인자는 **지금처럼 폴딩된 경로만** 넘긴다.
      서명 조각을 path 칼럼에 섞지 말 것.
  - `packages/core/src/routes/client-errors.ts`
    - `scrub(input.message)` 결과를 한 번만 계산해 (지금은 `new Error()` 안에서 인라인) 변수로 두고,
      그것과 `input.kind` 로 안정적인 짧은 지문을 만들어 `alertOperator` 의 새 인자로 넘긴다.
      지문은 **고정 길이**(예: FNV-1a 32비트를 8자 hex)를 권함 — signature 가 PK 이므로
      메시지 원문을 그대로 넣으면 PK 길이와 카디널리티가 메시지를 따라 늘어난다.
      암호 해시는 필요 없다(`util/crypto.ts` 의 함수들은 async WebCrypto 라 여기선 과하다).
      넘기는 값은 이미 스크럽된 문자열에서만 만들 것 — **스크럽 이전 원문을 알림 경로로 넘기지 말 것.**
  - `packages/core/src/__tests__/rules.test.ts`
    - `alertSignature` 에 대한 `describe` 블록 추가. `normalisePath` 가 이미 이 파일에서
      `../http/pipeline.js` 에서 import 되어 테스트되고 있으므로 import 자리는 그대로 쓸 수 있다.
      client-errors 의 지문 함수도 export 해 같은 파일에서 (b)(c) 를 검증한다.
  - (선택, 가능하면 함께) `scripts/e2e.mjs` 798~805행 "브라우저 오류 수집" 절에
    같은 경로로 메시지가 다른 두 건을 보내 `COUNT(DISTINCT signature) = 2` 를 확인하는 check 한 줄 추가.
    이것이 실제 배선을 증명하는 유일한 검사다(단위 테스트는 순수 함수만 본다).

- 검증 명령:
  ```bash
  npm run check                 # lint + typecheck + vitest (가볍다)
  npx vitest run packages/core/src/__tests__/rules.test.ts
  # 실제 배선 확인 (Docker PostgreSQL 16 필요, CI 가 push 마다 도는 것과 같은 것):
  docker run -d --name yeopjari-your-dev -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=your \
    -p 127.0.0.1:55440:5432 postgres:16-alpine
  npm run build && bash scripts/dev-reset.sh && node scripts/e2e.mjs
  ```
  e2e 는 157개 검사이고 서버(:8788)가 떠 있어야 한다. 최소한 `npm run check` 와
  e2e 의 "브라우저 오류 수집" 절이 통과하는 것을 보고 끝낼 것.

- 위험과 피할 것:
  - **기존 서명을 깨지 말 것.** `alertOperator` 는 `pipeline.ts` 안의 서버 실패 경로 전부가 쓴다.
    추가 인자의 기본값이 `''` 일 때 결과 문자열이 **한 글자도 달라지면** 운영 DB 의 모든
    `error_alerts` 행이 고아가 되어 억제돼 있던 오류가 한꺼번에 메일로 쏟아진다. 수용 기준 3)이 이것이다.
  - **지문이 안정적이어야 한다.** 타임스탬프·requestId·랜덤을 섞으면 중복 묶음이 사라져 반대 방향으로
    고장난다(오류 1건 = 메일 1통). 수용 기준 5c).
  - `pipeline.ts:483` 의 `logFailure` 도 `normalisePath(path)`·`redactSecrets(message)` 를 쓰지만
    **서명을 만들지 않는다**(stdout 한 줄 로그). 계약이 다르니 합치지 말고 그대로 두라 — 확인했다.
  - `error_alerts` 는 signature 가 PK 다. 지문 때문에 행이 늘어나지만
    `packages/core/src/routes/internal.ts:253` 의 정리 작업이
    `last_seen_at < now() - interval '30 days'` 인 행을 지운다(확인했다). 그래도 고정 8자 hex 지문으로
    카디널리티를 "서로 다른 오류 수"에 묶어 두는 편이 안전하다 — 운영자 콘솔이
    `admin.ts:449` 에서 이 테이블을 목록으로 보여주므로 행이 메시지마다 흩어지면 화면이 쓸모없어진다.
  - 보호 경로를 건드리지 말 것: `db/migrations/**` (스키마 변경 불필요 — `signature`·`method` 는 `text` 다),
    `packages/core/src/auth/**`, `.github/workflows/**`.
  - `packages/core/src/vendor/postgres` (패치된 드라이버 사본)은 손대지 말 것.

- 차선 후보: **`packages/core/src/util/validate.ts` 와 `util/text.ts` 의 단위 테스트 공백 메우기**
  — 이 저장소의 단위 테스트는 `rules.test.ts`·`vendor-postgres.test.ts` **두 개뿐**이고, 모든 쓰기
  라우트가 지나가는 `parse`/`v.*` 와 `mailboxKey`/`maskEmail`/`koreanParticle`/`charCount` 에는 테스트가
  없다. 지금 실제로 확인한 값어치 있는 엣지: `v.text({min:0})` 의 빈 문자열 통과 경로,
  `v.string` 은 길이를 UTF-16 단위로 세는데 `v.text` 는 코드포인트(`[...text].length`)로 센다,
  `v.int` 의 문자열 강제변환, `v.optional` 의 `''`→fallback, `v.array` 가 항목 실패 시 부분 배열을
  돌려주는 것. `mailboxKey('+tag@x.com')` 같은 경계도 포함. 파일 1~2개, 위험 1.
