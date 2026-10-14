- 과제: redactSecrets 계약을 표 테스트로 고정 (가치 4 / 위험 1 / 작업량 S)
- 왜: `packages/core/src/http/pipeline.ts:496` 의 `redactSecrets` 는 서버 오류 메시지가 밖으로 나가기 전에 비밀값을 지우는 **유일한** 차단기인데 테스트가 0건이다. 호출자는 확인된 2곳 — `logFailure`(`pipeline.ts:487`, stderr 로그)와 `alertOperator`(`pipeline.ts:145`): 후자는 redact 한 메시지를 `law_priv.record_error_alert(...)` 로 **DB 에 영구 저장**하고 `serviceErrorMail` 로 **운영자 이메일까지 보낸다**(pipeline.ts:150-162 에서 확인). 치환 규칙 5개를 차단/허용 표로 고정하면, 다음 회차에서 누군가 정규식을 손대도 주민번호(SEC-07/AT-26)나 토큰이 로그로 새는 회귀를 단위 시험이 즉시 잡는다.
- 수용 기준:
  1) 새 파일 `packages/core/src/__tests__/redact-secrets.test.ts` 가 `redactSecrets` 를 **실제로 import 해서 호출**하고(소스 문자열 검사 금지), `npm run test` 에서 통과한다.
  2) 표 테스트가 규칙 5개를 각각 **차단되는 입력**과 **차단되지 않아야 하는 입력**으로 고정한다 — ① 쿼리 파라미터(`api_key|api-key|apikey|key|token|access_token|secret|client_secret|password|signature|sig|code`) ② `Bearer`/`Basic` + 8자 이상 ③ 접두사 키(`re_ sb_secret_ sbp_ cfat_ sk- test_sk_ live_sk_`) + 8자 이상 ④ postgres/postgresql DSN 의 비밀번호 ⑤ 주민번호 `\d{6}-?[1-8]\d{6}`.
  3) 테스트는 "현재 동작"을 고정할 뿐 패턴을 넓히지 않는다. 지금 잡히지 않는 입력(예: 8자 미만 `Bearer abc`, 성별코드 0/9 인 13자리, 쿼리가 아닌 JSON `"token":"…"`, 공백 뒤 `Authorization: sk-…` 아닌 평문 토큰)은 **통과(원문 유지) 기대값**으로 적고 `// 현재 계약 밖 (contract boundary, intentionally not redacted)` 주석을 단다.
  4) 치환 결과 문자열이 `[숨김]` / `[주민번호 숨김]` 이라는 것, 그리고 ①④ 는 **키 이름·DSN 접두사를 남기고 값만** 지운다는 것(`$1` 보존)을 명시적으로 단정한다.
- 건드릴 파일:
  - `packages/core/src/__tests__/redact-secrets.test.ts` (신규) — `import { describe, expect, it } from 'vitest'` + `import { redactSecrets } from '../http/pipeline.js'` (`.js` 확장자 표기는 같은 디렉터리의 `text.test.ts:2`, `env-mock-providers.test.ts:2` 에서 확인한 관례), `describe` + `it.each` 또는 `for (const c of cases)` 표 구조. 파일 상단 주석은 영어 + 요구사항 번호(SEC-07/AT-26) — 이 저장소 관례.
  - **프로덕션 파일 0개.** `pipeline.ts` 를 고치지 말 것 — 정규식이 좁아 보여도 이번 회차 과제는 "지금 계약을 기록" 이지 "탐지 범위 확대" 가 아니다.
- 검증 명령:
  - `npm run test` (= `vitest run`, DB 불필요) — 또는 좁히려면 `npx vitest run packages/core/src/__tests__/redact-secrets.test.ts`
  - 마무리로 `npm run check` (= lint + typecheck + test). 직전 회차 기록상 기준선은 vitest 6 files / 103 tests 통과 — **이 정찰 세션에서는 샌드박스 권한 때문에 테스트를 돌려 보지 못했다(기준선 미확인)**. 구현자는 수정 전에 한 번 돌려 기준선을 먼저 확인할 것.
- 위험과 피할 것:
  - **정규식을 넓히지 말 것.** 운영자 반복 지시: 같은 종류의 검사/파서가 여러 경로에 있을 때 한쪽만 넓히면 다른 쪽이 깨진다 — 여기서는 `redactSecrets` 한 곳뿐이지만, 넓히면 정상 메시지까지 `[숨김]` 이 되어 오류 진단이 불가능해진다.
  - **테스트 입력에 진짜 비밀값·진짜 주민번호를 쓰지 말 것.** 명백한 가짜(`sk-TESTTESTTEST`, `900101-1234567` 같은 형식만 맞는 값)를 쓰고, 테스트 이름에 그 값을 넣지 말 것(운영자 지시: 검출 대상 문자열을 기록으로 흘리지 않는다).
  - `db/migrations/*`, `packages/core/src/auth/*`, `.github/workflows/*` 는 건드리지 않는다. `pipeline.ts:145` 주변(오류 응답 본문 생성)도 읽기만 할 것.
  - 보안 테스트가 통과한다고 해서 유출이 없다는 증명은 아니다 — `logFailure` 밖의 경로(예: `env.dev` 일 때의 `console.error(err.stack)`, pipeline.ts:488)는 redact 를 통과하지 않는다. 이번 과제의 수용 기준에 넣지 말고, 발견 사항으로만 `journal`/PR 설명에 한 줄 적어 다음 회차 후보로 남길 것.
- 차선 후보: `util/validate.ts` 경계 테스트 (가치 3 / 위험 1 / 작업량 M) — 모든 라우트 입력이 지나가는 검증기인데 테스트 0건. `optional` 의 `''` 취급, `text` 의 `min:0`, `array` 의 인덱스별 오류 경로, `int` 의 문자열 강제변환, `koreanParticle` 조사 메시지를 고정. 역시 프로덕션 0파일.
