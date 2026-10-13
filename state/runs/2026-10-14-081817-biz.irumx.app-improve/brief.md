- 과제: 로그인 복귀 경로 가드 safeNext 의 회귀 시험과 두 사본 일치 고정 (가치 4 / 위험 1 / 작업량 S)
- 왜: 로그인 후 돌아갈 경로를 걸러 주는 `safeNext` 가 서버(`packages/core/src/routes/sso.ts:57`)와 화면(`apps/web/src/lib/nav.ts:2`)에 같은 한 줄로 두 번 적혀 있는데(화면 쪽 주석이 "mirrors routes/sso.ts safeNext" 라고 스스로 밝힌다) 어느 쪽에도 시험이 없다. 한쪽만 손대도 알아차릴 방법이 없어 외부 출처(`//evil.com`, `/\evil.com`)를 내부 경로로 통과시키는 회귀가 조용히 들어올 수 있으므로, 통과/거절 목록을 시험으로 못 박아 두 사본이 같은 답을 내는 것을 증명한다.
- 수용 기준:
  1) `safeNext` 가 거절해 `'/'` 로 바꾸는 입력을 증명한다: `'//evil.com'`, `'  //evil.com'`(앞 공백 trim 후에도 거절), `'/\\evil.com'`(소스에서 `'/\\evil.com'` → 실제 문자 `/\evil.com`), `'/\\/evil.com'`, `'https://evil.com/x'`, `'javascript:alert(1)'`, `'evil.com'`, `'relative/path'`, `''`, `null`, 길이 300 이상인 `'/' + 'a'.repeat(320)`.
  2) 통과시키는 입력을 증명한다: `'/'`, `'/w/acme/tasks?tab=mine#top'`, `'  /w/acme  '` → `'/w/acme'`(trim 된 값이 그대로), 길이 299 인 경로는 그대로 통과(경계).
  3) 같은 케이스 표를 서버 사본과 화면 사본에 각각 돌려 **두 사본이 모든 케이스에서 동일한 결과를 낸다**는 것을 시험이 보인다(둘 중 하나만 바뀌면 시험이 깨진다).
- 건드릴 파일 (프로덕션 변경 없음, 시험만):
  - `packages/core/src/__tests__/domain.test.ts` — 파일 끝에 `describe('safeNext')` 추가. 기존 관례대로 상단에 `import { safeNext } from '../routes/sso.js'`(이 파일은 이미 `../routes/sales.js`·`../auth/tenant.js` 를 `.js` 확장자로 가져온다). 케이스 표를 `const CASES: [string | null, string][]` 로 두고 `it.each`/`for` 로 돌린다.
  - `apps/web/src/lib/nav.test.ts` — 신규. `import { safeNext } from './nav'` 로 같은 케이스 표를 복제해 돌린다(워크스페이스 경계를 넘는 import 는 하지 말 것 — tsc 가 워크스페이스별로 돌아 깨진다. 표를 공용 모듈로 빼지 말고 두 파일에 같은 표를 적는다. 구현이 일부러 중복이므로 시험도 중복이 맞다).
- 검증 명령:
  - `npm ci --no-audit --fund=false` (이 체크아웃에는 node_modules 가 없다 — 확인함: `node_modules/.bin/vitest` 부재)
  - `npm test -- packages/core/src/__tests__/domain.test.ts apps/web/src/lib/nav.test.ts`
  - `npm run check` (ESLint + 세 워크스페이스 tsc + 전체 vitest)
  - 증명 절차: 한쪽 사본의 `!next.startsWith('//')` 조건을 임시로 지워 해당 시험이 실패하는 것을 확인하고 되돌려 다시 통과시킬 것(시험이 실제로 가드를 보고 있다는 증거).
- 위험과 피할 것:
  - **`sso.ts` 의 프로덕션 코드를 고치지 말 것.** 이번 과제는 시험 추가뿐이다. 지금 가드 동작이 틀렸다고 단정하지 말고, 지금 동작을 기록하는 데서 멈춘다. auth/session·db/migrations·.github/workflows 는 건드리지 않는다.
  - 두 사본을 하나로 합치려 하지 말 것(운영자 지시: 같은 종류의 파서가 여러 경로에 있으면 통합하지 말고 좁히는 방향으로만). 이번엔 좁히는 것도 하지 않는다.
  - **탭/개선문자 케이스는 시험에 넣지 말 것.** 정찰에서 `'/\t/evil.com'`(실제 탭 문자)는 `safeNext` 를 통과함을 소스로 확인했지만, 서버의 최종 복귀는 `sso.ts:302` 의 `new URL(flow.next_path, ctx.env.publicUrl)` 뒤 `target.pathname + target.search` 만 쓰므로 외부 출처로 나가지 않는 것으로 읽힌다(브라우저 동작은 미확인). 지금 통과한다는 사실을 "올바른 동작" 으로 시험에 못 박으면 거짓 계약이 되고, 반대로 이번 회차에 가드를 좁히면 로그인 경로 변경이 되어 범위를 넘는다. 둘 다 하지 않는다.
  - `길이 < 300` 경계는 `<` 이므로 299 통과 / 300 거절이다. 반대로 적지 말 것.
- 차선 후보: 달력 날짜 검증 v.day 의 윤년·월말·범위 회귀 시험 보강 — `packages/core/src/util/validate.ts:216` 의 `v.day` 는 ISO 왕복과 2000~2100 범위를 검사하지만(정찰에서 소스 확인) `domain.test.ts` 에 경계 시험이 없다. 2024-02-29 통과 / 2023-02-29·2026-04-31·2026-02-30 거절 / `2026-1-1`·`20260101`·숫자 입력 형식 거절 / 1999-12-31·2101-01-01 범위 거절, 2000-01-01·2100-12-31 통과를 고정. 역시 프로덕션 변경 없음.
