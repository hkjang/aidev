- 과제: 주간보고 간단 입력이 "진행 70%" 같은 진척률 부분을 읽지 못해 상태도 진척률도 버려지는 것 고치기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `parseSimpleLines`(packages/tools/src/weekly/model.ts:166)의 `STATUS_WORDS`(:151)는 `/^(진행|진행중|...)$/` 처럼 **완전 일치**만 보므로, 실무에서 흔한 `업무명 / 진행 70% / 내용` 입력은 상태 판정에 실패해 `status = null` 로 남고("상태를 알 수 없어 비워 두었습니다"), `70%` 는 `thisWeek` 문자열 뒤에 그대로 붙는다. 모델에는 `progress: number | null` 필드와 표 열(`DEFAULT_TEMPLATE` 의 `progress`, :88)이 이미 있는데 간단 입력 경로만 그 값을 채우지 못해, 사용자가 적어 준 숫자가 어느 칸에도 들어가지 않는다.
- 수용 기준:
  1) `weekly.parseSimpleLines('신규 화면 / 진행 70% / 와이어프레임 공유')` 가 `status: 'doing'`, `progress: 70`, `thisWeek: '와이어프레임 공유'` 를 돌려준다 — `70%` 가 `thisWeek` 에 중복으로 남지 않는다.
  2) 접두사 없는 `50%` 부분은 `progress: 50` 만 채우고 **상태는 추론하지 않는다**(`status` 가 null 이면 기존 안내 노트가 그대로 나온다). 100% 를 완료로 바꾸지 않는다.
  3) 진척률로 읽을 수 없는 부분(`120%`, `7.5%`, `%`, `70% 이상`)은 진척률로 읽지 않고 지금처럼 `thisWeek` 로 간다 — 그래서 파서가 `checkReport` 의 `진척률은 0~100 사이로 적어 주세요` 오류(model.ts:267)를 만들어 낼 수 없다.
  4) 읽은 방식이 `notes` 에 한 줄 남는다(기존 `'${part}' → 상태 진행` 과 같은 꼴, 예: `'진행 70%' → 진척률 70%`). 파일 상단 계약("입력에 없는 문장·숫자·결과는 더하지 않는다", model.ts:6~7)을 지킬 것 — 입력에 있는 숫자를 제자리에 넣는 것까지만.
  5) 기존 단언이 하나도 깨지지 않는다: `weekly: simple input` 의 `SAMPLE_WEEKLY_LINES` 파싱 단언(weekly-handover.test.ts:36~46, `SAMPLE_WEEKLY_LINES` 는 packages/tools/src 의 샘플 상수 — 먼저 열어 % 가 들어 있는지 확인하고, 들어 있다면 그 단언을 새 기대값으로 **의도적으로** 갱신하되 이유를 회차 노트에 적을 것), `상태를 알 수 없어…`(:48), 진척률 범위 검사.
- 건드릴 파일:
  - `packages/tools/src/weekly/model.ts:166 parseSimpleLines` — 상태 단어 분기 **바로 뒤**(= `/예정$/`·`/대기$|요청|필요/` 분기보다 **앞**)에 진척률 분기 한 개를 넣는다. 정규식은 앵커로 좁게: 예 `/^(?:진척|진행|달성)?\s*(\d{1,3})\s*%$/` + 값 0~100 검사 + `task.progress === null` 일 때만 대입. 접두사가 `진행` 이고 `task.status === null` 이면 `'doing'` 까지 세운다(이미 STATUS_WORDS 가 `진행`→doing 으로 보는 것과 같은 읽기, 노트에 남긴다).
  - `packages/tools/src/__tests__/weekly-handover.test.ts` — `describe('weekly: simple input')` 안에 수용 기준 1~4 를 덮는 `it` 추가. 손으로 만든 객체가 아니라 **실제 입력 문자열을 `parseSimpleLines` 에 통과시켜** 증명할 것(운영자 규칙).
  - 프로덕션 파일은 이 한 개로 끝낸다(총 2파일).
- 검증 명령:
  - `npm ci` (워크트리에 node_modules 가 없다 — 안 하면 `vitest: not found`)
  - `npx vitest run weekly-handover.test.ts` (고치기 전 빨간불을 눈으로 확인하고 출력을 회차 노트에 적을 것)
  - `npm run check` (eslint + 4개 패키지 typecheck + vitest 전체 — 직전 기준선은 11파일 135 passed)
  - 선택: `npm run build`. `e2e`·`tools`·`ui`·`a11y` 는 돌리지 말 것 — Docker PostgreSQL·Playwright·가짜 서버(:18997/:18998/:8797)가 필요하고 `@workkit/tools` 는 서버 코드가 import 하지 않는다.
- 위험과 피할 것:
  - `DEFAULT_TEMPLATE` 의 `progress` 열은 기본 `enabled: false` 다(model.ts:88). **이번 회차에 기본값을 켜지 말 것** — 표 열 기본 구성 변경은 별 과제다. 파싱된 값은 업무 편집 화면과 사용자가 열을 켤 때 보인다.
  - `apps/web/src/tools/weekly/WeeklyTool.tsx` 는 건드리지 말 것(:151 이 `parseSimpleLines` 를 그대로 쓴다 — 순수 함수만 고치면 화면은 따라온다).
  - `packages/tools/src/table/*`·`compare/*` 는 손대지 말 것: 최근 여섯 회차가 모두 그 자리를 고쳐 **미머지 PR 이 쌓여 있고**(이 워크트리 base aed4f5d 에는 `toTable` 의 접미사 고유화 수정조차 없다) 충돌 구역이다.
  - `.github/workflows/**` 는 열지 말 것. main 의 `check` 실패는 저장소 밖 원인(Actions 결제·지출 한도로 잡 미시작)으로 이미 두 번 판정됐다 — 워크플로 수정·테스트 단언 완화는 기각된 접근이다.
  - 상태 단어 분기 **앞**에 진척률 분기를 넣으면 `진행` 단독 부분을 가로채 기존 단언이 깨진다 — 순서를 지킬 것.
- 차선 후보: `sanitizePreset`(packages/tools/src/project/sanitize.ts:136~137)이 `Number.isInteger` 만 보고 `0`·음수·`1e9` 같은 헤더 줄 번호를 통과시키는 것 좁게 막고 sanitize 전용 테스트를 더하기 (가치 1 / 위험 1 / 작업량 S — 지금 `oldHeaderLine`/`newHeaderLine` 을 읽는 코드가 없어 사용자 영향은 0, 테스트 공백 보강 가치만 있다)
