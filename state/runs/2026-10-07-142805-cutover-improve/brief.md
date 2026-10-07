- 과제: 없는 `parentId` 로 하위를 추가하면 "activity.json 데이터 형식이 올바르지 않습니다" 400 이 나오는 것을 404 `TARGET_NOT_FOUND` 로 바로잡기 (가치 3 / 위험 1 / 작업량 S)

- 왜: 10-06 이 `delete`/`move`/`targetId+newStatus` 세 분기에 대상 존재 검사를 넣었지만 `action:'add'` 분기(`app/api/activities/route.ts:90`)만 빠졌다. 다른 관리자가 이미 지운 행의 ➕(하위 추가)를 누르면 `addActivity` 가 부모 없는 활동을 만들고, `validateActivityImport`(`lib/activityData.ts:262-264`)가 `부모 id "..."가 activities에 존재하지 않습니다` 를 이슈로 올려 `issues.length > 0` 에서 throw 하므로(`:308`) 라우트 catch 가 400 `INVALID_DATA` 를 돌려준다. 관리 화면 배너에는 `ActivityImportValidationError` 의 메시지 `activity.json 데이터 형식이 올바르지 않습니다.`(`lib/activityData.ts:43`)가 그대로 뜬다 — 실시간 전환 훈련 중에 JSON 업로드 파일 형식 문제로 읽히는 문구가 나오고, 실제 원인(그 행이 사라졌다)과 해야 할 일(목록 확인/새로고침)은 어디에도 없다. 같은 상황의 🗑/▲▼/상태 버튼은 이미 올바른 404 문구를 준다.
- **주의(과장 금지)**: 이것은 조용한 200 이 아니다. 지금도 저장은 **되지 않고** 오류는 난다. 이 과제의 값은 데이터 손상 방지가 아니라 **오류 분류(400→404)와 관리자가 읽는 문구의 정확성**, 그리고 네 분기 중 유일하게 어긋난 계약을 맞추는 것이다. 10-06 과 같은 "조용한 성공" 으로 적지 말 것.

- 수용 기준:
  1) `PUT /api/activities` 에 `{ action: 'add', parentId: '<데이터에 없는 id>' }` → **404**, 본문 `code: 'TARGET_NOT_FOUND'`, 문구에 사라진 대상 id 포함(기존 `targetNotFoundResponse` 그대로).
  2) 그 요청 뒤 `GET /api/activities` 의 `activities` 와 `lastUpdated` 가 **둘 다 그대로**다(아무것도 쓰지 않음 — 조기 반환이라 `writeActivityData` 와 `after()` 메일까지 내려가지 않는다).
  3) 기존 200 계약 보호: `{ action:'add', parentId: null }`(최상위 추가)과 `{ action:'add', parentId: '<실제 존재하는 id>' }`(하위 추가)는 **그대로 200** 이고 활동이 1개 늘어난다.
  4) 기존 400 계약 보호: `{ action:'add', parentId: 5 }`(숫자)는 지금처럼 `UNSUPPORTED_ACTION` **400** 이다 — `typeof body.parentId === 'string'` 검사가 먼저 실패해 add 분기를 아예 타지 않는다. `e2e/admin-put-contract.spec.ts:78` 이 이미 이것을 기대하므로 깨뜨리면 안 된다.
  5) 테스트가 증명할 것: 수정 **전에는** 1)이 404 대신 **400 `INVALID_DATA`** 로 실패하는 것을 빨간 출력으로 먼저 남기고(실패 재현), 수정 후 통과.

- 건드릴 파일 (프로덕션 1개):
  - `app/api/activities/route.ts:90` add 분기 — `updated = addActivity(...)` 앞에 `parentId` 가 `null` 이 **아닐 때만** `updated.some(a => a.id === body.parentId)` 를 보고 false 면 `return targetNotFoundResponse(body.parentId)`. 아래 세 분기(`:93,:100,:103`)와 같은 한 줄 모양을 쓰고 헬퍼(`:30`)는 새로 만들지 말 것. `parentId === null` 은 반드시 통과시켜야 한다(최상위 추가).
  - `e2e/admin-missing-target.spec.ts` — 10-06 이 만든 파일. 같은 `expectNotFoundAndUnchanged` 헬퍼(`:61` 부근)를 재사용해 add 케이스를 더하고, 수용 기준 3)의 200 보호 2건도 함께 둘 것.
  - `docs/ADMIN_GUIDE.md` 5.6 절 `PUT /api/activities` 행(303행 부근, 10-05 가 409·10-06 이 404 를 적어 둔 자리) — 404 가 add 에도 적용된다는 문구 한 줄. 선택이지만 앞 두 회차가 같은 자리를 갱신했다.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 준비: `npm ci --legacy-peer-deps` (이 워크트리에 `node_modules` 없음. 수 분. audit 가 high/critical 을 보고하지만 이번 범위 밖)
  - 단위: `node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts` → 기준선 **95 pass / 0 fail**(base 51ffd03 에서 정찰이 확인). `npm run test:unit` 은 글롭이 확장되지 않아 그 자체로 exit 1 이다 — **관문으로 쓰지 말 것**.
  - `npx tsc --noEmit`, `npm run lint`, `npm run build` (각 exit 0)
  - 빨간 테스트: `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npx playwright test e2e/admin-missing-target.spec.ts`
  - 전체: `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e` → 기준선 **38 passed**(base 51ffd03, spec 13개) + 신규
  - `npm run lint` 는 E2E **전에** 돌리고, E2E 뒤에는 `rm -rf playwright-report test-results` 로 정리할 것(두 경로는 `.gitignore` 에 있고 ESLint 는 이것을 무시하지 않는다). `eslint.config.mjs` 에 `globalIgnores` 를 추가하는 09-29 접근은 **반려됐으므로 재제출 금지**.

- 위험과 피할 것:
  - 검사를 `Array.isArray(body.activities)` 분기(`:76`) **위로 올리지 말 것**. `onStatusChange`/`onUpdateActivity`(`app/admin/page.tsx:332-347` 부근)는 배열과 `targetId` 를 함께 보내므로 정상 저장이 404 가 된다 — 10-06 이 같은 자리에서 명시적으로 피한 함정이다.
  - `lib/treeUtils.ts` 와 `addActivity` 자체는 **건드리지 말 것**. 단위 테스트 기대값 변경이 0건으로 유지된다. (참고: base 51ffd03 에는 10-07 의 UUID ID 생성 수정이 **아직 머지되지 않았다** — review-pending. `addActivity:49` 는 여전히 `${parentId}-${Date.now()}` 다. 그 과제와 충돌하지 않도록 ID 생성에 손대지 말 것.)
  - `validateActivityImport` 의 부모 존재 검사(`:262-264`)를 약화시키지 말 것 — 업로드 경로(`app/api/activities/import/route.ts`)가 같은 검증기를 쓰므로 유령 행이 파일로 들어온다.
  - 보호 경로 회피: `app/api/auth/**`·`lib/adminSession.ts`·`proxy.ts`·`lib/mail/*`·`Dockerfile`·`amplify.yml` 은 이 과제와 무관하다. `route.ts:62` 의 `isAdminRequest` 도 손대지 말 것.
  - 폴링(1초)으로 저장 성공을 판정하지 말 것 — PUT 응답 코드와 후속 GET 으로 볼 것. UI 경로 테스트가 필요하면 같은 browser context 의 `page.request` 를 쓸 것.

- 차선 후보: `validateActivityImport` 제한값 경계 테스트 보강 (가치 2 / 위험 1 / 작업량 S) — `lib/activityData.test.ts` 에 id 120/121자, dashboardTitle 100/101자, 개수 5000/5001 경계 테스트가 없다(정찰이 파일 전체 검색으로 확인). 프로덕션 변경 0건, 테스트 1파일. 1순위가 기존 e2e 와 충돌해 성립하지 않을 때 이것을 고를 것.
