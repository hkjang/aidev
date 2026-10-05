# 과제서 (2026-10-06, run 2026-10-06-033756-cutover-improve)

- 과제: 존재하지 않는 `targetId` 의 delete / move / 상태변경 PUT 이 조용한 200 "저장 성공" 을 돌려주는 것을 404 로 막기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `app/api/activities/route.ts` PUT 의 세 분기(`:79` delete, `:81` move, `:87` targetId+newStatus)는 대상 id 가 데이터에 없어도 **아무 변화 없이** `currentData.activities` 를 그대로 다시 저장하고 새 `lastUpdated` 와 200 을 돌려준다 — `deleteActivity`(`lib/treeUtils.ts:91`)는 걸러낼 것이 없어 같은 배열을 반환하고, `moveActivity`(`:27`)는 `targetIndex === -1` 이면 복사본을 그대로 반환하고, 상태변경 `map`(`route.ts:88-90`)은 어떤 항목에도 맞지 않는다. 관리 화면의 `submitActivityChange`(`app/admin/page.tsx:76`)는 `res.ok` 만 보므로, 다른 관리자가 이미 지운 행을 🗑/▲▼/상태 버튼으로 건드리면 오류 배너 없이 "저장됐다" 로 보이고 `lastUpdated` 만 전진한다(그 전진은 다른 관리자의 `expectedLastUpdated` 를 쓸데없이 409 로 만든다). 404 로 바꾸면 10-01(`UNSUPPORTED_ACTION`)·10-05(`STALE_SNAPSHOT`)가 메운 "조용한 성공" 구멍의 마지막 한 칸이 닫히고, 관리자는 이미 배너 경로가 있으므로 서버 문구가 그대로 화면에 나온다.

- 수용 기준:
  1) `PUT {action:'delete', targetId:'<없는 id>'}` 가 404 와 `{ error, code:'TARGET_NOT_FOUND' }` 를 돌려준다. `error` 는 빈 문자열이 아닌 한국어 문구.
  2) `PUT {action:'move', targetId:'<없는 id>', direction:'up'}` 과 `PUT {targetId:'<없는 id>', newStatus:'완료'}` 도 같은 404.
  3) 세 경우 모두 **아무것도 쓰지 않는다** — 요청 전후로 GET 의 `activities` 와 `lastUpdated` 가 완전히 같다(`writeActivityData` 까지 가지 않으므로 메일 알림도 나가지 않는다).
  4) 기존 200 계약이 그대로다: `{dashboardTitle}` 단독 PUT(`e2e/admin-put-contract.spec.ts:90`), `{activities: [...]}`(`:99`), `{activities, targetId, expectedLastUpdated}`(관리 화면 `onUpdateActivity`, `app/admin/page.tsx:343`) — **마지막 것이 핵심**: 배열 분기 요청도 `targetId` 를 같이 보내므로, 존재 확인을 `Array.isArray(body.activities)` 분기보다 **앞으로 끌어올리면 안 된다**.
  5) 기존 400 계약도 그대로다: `{action:'move', targetId:'root', direction:'left'}` 와 `{action:'rename', targetId:'root'}` 는 여전히 400 `UNSUPPORTED_ACTION`(`e2e/admin-put-contract.spec.ts:80,84`) — 즉 분기 자체가 성립하지 않는 요청은 404 가 아니라 400 이어야 한다.
  6) 테스트가 증명할 것: "수정 전에는 200 이고 데이터가 안 바뀐다"(= 조용한 성공)가 수정 후 404 + 불변으로 바뀌는 것. 신규 e2e 는 실제 dev 서버(127.0.0.1:3100)에 로그인 세션의 `request.put` 으로 보내고, 성공/실패 판정에 1초 SWR 폴링을 쓰지 않는다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `app/api/activities/route.ts` — `unauthorizedResponse()`(`:20`) 옆에 `targetNotFoundResponse(id)` 헬퍼를 하나 두고, **세 분기 안에서** 각각 `updated.some(a => a.id === body.targetId)` 가 false 면 그 404 를 `return` 한다(검증·저장·`after()` 로 내려가지 않게 조기 반환). 분기 조건 문자열, `Array.isArray` 분기, `expectedLastUpdated` 409, `matched` 플래그, `validateActivityImport` 호출, `isAdminRequest`(`:49`) 는 손대지 않는다.
  - `e2e/admin-missing-target.spec.ts` (신규) — `e2e/admin-put-contract.spec.ts` 의 `beforeEach` 시드 + `expectRejectedAndUnchanged` 패턴(`:39-65`)을 그대로 베끼되 기대 status 를 404 로. 404 3건 + 기존 200 보호 1~2건(`{dashboardTitle}` 단독, `{activities, targetId, expectedLastUpdated}`)을 같이 담으면 수용 기준 4 가 테스트로 고정된다.
  - 선택: `docs/ADMIN_GUIDE.md` 의 5.6 절 API 표에 한 줄(10-05 회차가 같은 자리에 409 를 적었다). 문서까지 포함해도 3파일.

- 검증 명령 (이 정찰 워크트리에는 `node_modules` 가 없어 **하나도 실행하지 못했다** — 구현 워크트리에서 `npm ci --legacy-peer-deps` 먼저):
  - `npx tsc --noEmit`
  - `node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts` (기대 `# pass 95 / # fail 0`. `npm run test:unit` 의 `"lib/**/*.test.ts"` 글롭은 이 환경에서 확장되지 않아 조용히 0건이 되므로 관문으로 쓰지 말 것 — 10-01·10-03·10-05 회차가 모두 재현)
  - 구현 전: `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npx playwright test e2e/admin-missing-target.spec.ts` → 404 기대 3건이 `Expected: 404 / Received: 200` 으로 실패하는 것을 **먼저 보고** 저널에 붙여넣기. 구현 후 같은 명령 통과.
  - `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e` (기존 32건 + 신규 ⇒ 35건 전후)
  - `rm -rf playwright-report test-results` 뒤 `npm run lint`, 그다음 `npm run build`, 마지막에 `git status --porcelain` 이 의도한 파일만 담는지.

- 위험과 피할 것:
  - **404 를 배열 분기 위로 올리지 말 것** (수용 기준 4). `onStatusChange`(`app/admin/page.tsx:332-337`)는 `{activities, targetId, newStatus, expectedLastUpdated}` 를, `onUpdateActivity`(`:343-347`)는 `{activities, targetId, expectedLastUpdated}` 를 보낸다 — 둘 다 `Array.isArray` 분기가 먼저 이기도록 설계된 것이고, 서버의 `targetId+newStatus` 분기는 UI 에서 사실상 죽은 코드다. 위로 올리면 `admin-save-failure.spec.ts`·`admin-stale-overwrite.spec.ts`·`admin-edit-stale.spec.ts` 가 함께 깨진다.
  - `lib/treeUtils.ts` 는 건드리지 말 것. `treeUtils.test.ts:24,55` 가 "없는 id 는 자기 자신만 / 아무것도 바뀌지 않는다" 를 이미 고정하고 있다 — 순수 함수의 무변화 계약은 옳고, 바뀌어야 하는 것은 HTTP 층의 응답뿐이다. 이렇게 하면 단위 테스트 기대값 변경이 0건이다.
  - 범위 밖으로 둘 것: 대상이 있는데 이미 맨 위/맨 아래인 `move`(무변화 200 — 화살표를 누른 사용자에게는 정상 동작), `action:'add'` 의 없는 `parentId`(`validateActivityImport` 가 이미 400 orphan 으로 잡는다 — 이전 회차 기록), `lastUpdated` 밀리초 해상도, 상태 전파 규칙 이전, 업로드 라우트.
  - 보호 경로 안 건드림: `app/api/auth/**`, `lib/adminSession.ts`, `proxy.ts`, `Dockerfile`, `amplify.yml` 모두 무관.
  - e2e 산출물(`playwright-report/`, `test-results/`)은 `eslint.config.mjs` 의 `globalIgnores` 에 **없다**(09-29 커밋 `0de2f1e` 는 main 에 머지되지 않았고 운영자 지시상 같은 접근 재제출 금지). e2e 뒤 lint 하려면 두 경로를 `rm -rf` 로 치울 것 — `.gitignore` 15-16행에 있어 커밋 위험은 없다.
  - 미확인으로 남긴 것: ① 404 와 400 중 어느 쪽이 더 맞는지는 판단이다 — 분기는 성립했고 대상만 없으므로 404 `TARGET_NOT_FOUND` 를 권한다. 구현 중 기존 테스트와 충돌하면 400 `TARGET_NOT_FOUND` 로 내려도 수용 기준 1~6 은 그대로 만족한다(status 기대만 맞춰서 바꿀 것). ② 404 때 관리 화면이 추가로 `mutate()` 를 해야 하는지는 보지 않았다 — 1초 폴링이 어차피 지운 행을 치우므로 클라이언트 변경 없이 가는 쪽을 권한다(프로덕션 1파일 유지). ③ `docs/ADMIN_GUIDE.md` 5.6 절의 실제 표 모양은 이번에 열지 않았다.

- 차선 후보: `parentId === id` 가 자기참조·순환 issues 2건을 내는 중복 제거 (2/1/S) — `lib/activityData.ts:260` 의 `continue` 는 부모 관계 루프만 건너뛰고 순환 검사(`:273-299`)가 같은 `activities[0].parentId` 를 한 번 더 신고한다. 단독 회차로 가능하지만 `lib/activityData.test.ts` 가 현재 동작(2건)을 고정해 두었으므로 기대값을 함께 바꿔야 한다(라인 번호는 10-03 회차 기록 기준, 이번 미재확인).
