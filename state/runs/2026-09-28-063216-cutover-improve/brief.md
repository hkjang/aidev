- 과제: `validateActivityImport` 단위 테스트 신규 추가 (`lib/activityData.test.ts`) (가치 3 / 위험 1 / 작업량 S)
- 왜: `validateActivityImport`(`lib/activityData.ts:120`)는 관리자 PUT(`app/api/activities/route.ts:81`)과 JSON 업로드(`app/api/activities/import/route.ts:77`)가 **공유하는 단 하나의 입력 검증 관문**이고, 여기를 잘못 통과하면 `data/activity.json` 에 깨진 트리(고아 노드·순환·중복 id)가 원자 쓰기로 그대로 저장된다. 그런데 단위 테스트가 0건이다 — `lib/**/*.test.ts` 8개(adminSession·treeUtils·mail 4·tracking 2) 중 activityData 는 없음(이번 회차 `ls` 로 확인). 프로덕션 코드 변경 0으로 이 함수의 계약을 고정하면, 앞으로 이 함수나 두 라우트를 건드리는 회차가 회귀를 즉시 본다.
- 수용 기준:
  1) `npm run test:unit` 이 기존 79건을 그대로 통과한 채 신규 테스트만큼 늘어난다(신규 파일 1개 → suites +1). 단독으로 `node --test lib/activityData.test.ts` 도 통과한다.
  2) **정상 입력 통과**: `parentId:null`+`level:1` 최상위 1건과 그 자식(`parentId:'root'`, `level:2`) 1건을 주면 `activities` 2건이 반환되고, `id`·`parentId` 의 앞뒤 공백이 trim 되며(`activityData.ts:223-224`), `dashboardTitle: '  제목  '` 이 `'제목'` 으로 trim 되어 돌아온다. `dashboardTitle` 을 주지 않으면 반환값이 `undefined` 다.
  3) **빈 배열 통과**: `{ activities: [] }` 는 throw 하지 않고 `activities: []` 를 돌려준다 — 09-27 회차의 "최상위 작업 추가" 진입점이 이 계약에 의존한다.
  4) **최상위 throw 2종**: 객체가 아닌 값(예: `[]`, `'x'`, `null`)은 `issues` 1건 `path:'$'`, `activities` 가 배열이 아니면 `issues` 1건 `path:'activities'`. 둘 다 `ActivityImportValidationError` 이고 `err.issues` 로 읽는다(`activityData.ts:39-47`).
  5) **필드별 실패의 `issues[].path` 가 정확함**을 각각 증명(각 케이스는 그 하나만 틀린 입력으로): id 중복 → `activities[1].id` 이고 메시지에 상대 인덱스(`activities[0]`)가 들어간다 / 부모 부재 → `activities[0].parentId` / 부모 대비 level 불일치 → `activities[1].level` / 자기 자신을 부모(`parentId === id`) → `activities[0].parentId` (이때 부모 부재·level 검사는 `continue` 로 건너뛰므로 issues 는 1건) / 최상위인데 `level !== 1` → `activities[0].level` / 2건 순환(a↔b) → `parentId` path 이고 `reportedCycles` 때문에 **순환 1건당 issues 1건만** 난다.
  6) **`'진행중'` 정규화**: `status:'진행중'` 2건을 주면 반환 `activities` 의 status 가 모두 `'진행'` 이고 `normalizedLegacyStatuses === 2` 다(카운트가 boolean/1 고정이 아님을 고정 — 이 숫자는 `components/ActivityJsonImporter.tsx:140-141` 이 사용자에게 "…N건을 '진행'으로 변환했습니다" 로 그대로 보여준다).
  7) **issues 상한 계약**: 오류 항목을 60개 만드는 입력에서 `issues.length === 51`(최대 50 + 안내 1)이고 마지막 항목의 `path === '$'` 이며 메시지에 생략 건수(10)가 들어간다(`activityData.ts:125-131`, 296-301).
- 건드릴 파일:
  - `lib/activityData.test.ts` — **신규, 유일한 변경 파일**. 기존 관례를 그대로 따를 것(`lib/treeUtils.test.ts:1-4` 확인함): `import assert from 'node:assert/strict'` / `import { describe, it } from 'node:test'` / `import { validateActivityImport, ActivityImportValidationError } from './activityData.ts'` (**`.ts` 확장자 포함** — 같은 폴더 기존 테스트 2개가 그렇게 쓴다) / `import type { Activity } from './types.ts'` 가 필요하면 동일 형태. 한국어 `describe`/`it` 제목.
  - 프로덕션 파일 0개.
- 검증 명령:
  - `npm ci --legacy-peer-deps` — **`node_modules` 가 없다(이번 정찰에서 확인). 수 분 소요.**
  - `npm run test:unit` ← 핵심. 단독: `node --test lib/activityData.test.ts`
  - `npx tsc --noEmit`, `npm run lint`
  - `npm run lint` 전에 `rm -rf playwright-report test-results` (eslint `globalIgnores` 에 아직 없어 trace 뷰어 번들을 검사해 수천 건이 난다 — `eslint.config.mjs` 에 `.next/**`,`out/**`,`build/**`,`next-env.d.ts` 만 있는 것을 확인함).
  - e2e·`npm run build` 는 프로덕션 변경이 0이므로 필수 아님. 시간이 남으면 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e` (기대 18건).
- 위험과 피할 것:
  - **`lib/activityData.ts` 를 수정하지 말 것.** 같은 파일에 원자 쓰기·백업(위험 구역)이 있다. 테스트를 쓰다 실제 버그를 발견하면 **고치지 말고** 현재 동작을 고정하는 테스트를 쓰고, 그 사실만 회차 요약에 적어라(범위 밖 — 다음 회차 과제).
  - `readActivityData`/`writeActivityData` 를 테스트에서 호출하지 말 것. 모듈은 로드 시점에 `ACTIVITY_DATA_FILE`/`process.cwd()/data/activity.json` 경로를 계산하므로(`activityData.ts:10-17`), 쓰기 함수를 부르면 실제 데이터 파일을 덮어쓴다. **import 자체는 디스크를 건드리지 않는다**(경로 계산만 — 확인함). 테스트는 순수 함수 `validateActivityImport` 만 호출하라.
  - `MAX_VALIDATION_ISSUES`(=50)는 export 되지 않으므로(`activityData.ts:19`) 50/51 을 테스트에 하드코딩하고, 오류를 확실히 51건 넘기는 입력을 쓰라.
  - 미묘한 지점: `'진행중'` 카운트 증가(`:211`)는 **valid 판정 이전**에 일어난다 — 같은 항목의 다른 필드가 틀려 throw 되는 경우에도 카운트가 올라간다. 이 동작을 "고치려 하지 말고", 6)의 테스트는 다른 필드가 모두 정상인 입력으로 짜라.
  - 운영자 지침대로 **대역·수기 스텁으로 계약을 세우지 말 것**: `validateActivityImport` 는 `unknown` 을 받는 순수 파서이므로, 라우트가 넘기는 것과 동일한 평문 객체를 실제 함수에 넣는 것이 곧 프로덕션 호출과 같다. 모킹은 필요 없다.
  - 잘못된 옛 기록 주의: "`import type … from './types'` 에 `.ts` 가 없으면 `node --test` 가 못 읽는다" 는 교훈은 **틀렸다**(Node 22 타입 스트리핑). 그 이유로 `activityData.ts:4` 의 import 를 고치지 말 것.
  - grep 결과("문자열이 있다")를 증거로 제출하지 말 것. 실패 재현은 반드시 **테스트를 먼저 써서 실패시켜** 보여라 — 다만 이번 과제는 프로덕션을 안 바꾸므로, TDD 증거는 "테스트가 의도한 `issues[].path` 를 잘못 쓰면 실패하고 맞게 쓰면 통과한다"까지면 된다(한 케이스에서 의도적으로 잘못된 기대값을 넣어 실패를 한 번 확인한 뒤 고쳐라).
- 차선 후보: eslint `globalIgnores` 에 `playwright-report/**`·`test-results/**` 추가 (`eslint.config.mjs:10-16`, 2줄. 현재 목록에 없는 것을 이번 회차에 확인). 1순위가 성립하지 않을 때만 — 단독으로는 가치 2 / 위험 1 / S.
