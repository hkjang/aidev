# 회차 노트 2026-10-05-111721-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:17] base pinned — main@99c2c7d
- [러너 11:17] autonomy release — 

## 정찰 노트
- 보류 1순위였던 "없는 targetId 400"(2/2/S)을 차선으로 내리고, 같은 route.ts 분기에서 더 큰 손실을 내는 전체 배열 덮어쓰기를 골랐다 — 전자는 no-op 200 이라 무해하지만, 후자는 10-04 가 admin-load-error 를 띄우면서 "오래된 스냅샷으로 저장 가능" 상태가 눈에 보이게 됐고 그때 되돌림 범위가 무제한이다. 자기참조 중복·글롭·eslint 산출물은 모두 가치 2 이하라 제쳤다.
- 코드로 확인한 것: route.ts 전문, treeUtils.ts 전문, admin/page.tsx 1-40·60-180·255-360, lib/types.ts 13-17, e2e/dashboard-load-failure.spec.ts 1-40·57-59, e2e/admin-save-failure.spec.ts 25-48.
- 추측으로 적은 것(구현자가 먼저 확인할 것): ① submitActivityChange(76-103행)가 409 를 별도 분기 없이 서버 error 문구 그대로 배너에 띄운다고 적었으나 실제 문구가 사용자에게 충분한지는 미검증. ② ADMIN_GUIDE 의 동시 편집 관련 서술은 이번에 열지 않았다. ③ node_modules 가 없어 이번 회차에 테스트·lint 를 한 번도 돌리지 않았다 — 검증 명령은 과거 회차 기록 기준이다.
- 조심할 것: 409 검사를 opt-in 으로 두지 않으면 admin-save-failure.spec.ts 의 beforeEach/afterEach 가 깨져 여러 spec 이 연쇄로 죽는다. 그리고 onStatusChange 의 클라이언트 전파 규칙은 이번에 건드리지 말 것(별도 M 과제이며 ADMIN_GUIDE 4.3 과 일치한다).
- [러너 11:21] scout done — 관리 화면이 오래된 트리 스냅샷을 통째로 덮어써 다른 관리자의 변경을 조용히 되돌리는 것을 409 로 막�

## 구현 노트
- 바꾼 것: 관리 화면의 전체 배열 PUT 2곳(`onStatusChange`·`onUpdateActivity`)이 `expectedLastUpdated: data.lastUpdated` 를 함께 보내고, `route.ts` 의 `Array.isArray(body.activities)` 분기 **안에서만** 그 값이 저장된 `currentData.lastUpdated` 와 다르면 아무것도 쓰지 않고 409 `{error, code:'STALE_SNAPSHOT'}` 를 돌려준다. 다른 관리자의 변경이 조용히 되돌아가는 것을 막기 위함. 검사는 opt-in — 필드 없는 `{activities}` PUT 은 그대로 200.
- 확신 없는 곳: ① 이 검사는 `lastUpdated`(밀리초) 비교라 같은 밀리초 두 쓰기는 못 잡는다 — 과제서가 범위 밖으로 지정했고 고치지 않았다. ② 평시(1초 폴링이 도는 상태)에는 스냅샷이 거의 항상 최신이라 409 가 거의 나지 않는다. 실질 효과는 `admin-load-error`(GET 실패) 상태에 집중된다. ③ 업로드(`/api/activities/import`) 직후 저장이 409 가 되지 않는 것은 코드로만 확인했다(응답 `data` 에 `lastUpdated` 가 들어 있고 `onImported` 가 그것을 `mutate` 로 캐시에 넣는다) — **테스트로 고정하지 않았다**(ideas.json 에 후보로 남김). ④ `data` 가 undefined 인 최초 로드에는 필드를 보내지 않도록 조건 spread 를 썼지만, 그 상태에서 행이 렌더되지 않으므로 실제로 도는 경로는 아니다.
- 일부러 하지 않은 것: `submitActivityChange` 에 409 전용 분기(서버 `error` 문구가 이미 `admin-save-error` 배너에 그대로 들어간다 — 76-103행 확인), `onStatusChange` 의 클라이언트 전파 규칙 이전, 없는 `targetId` 400(차선 후보 그대로 보류), 업로드 라우트, `eslint.config.mjs`(09-29 반려된 접근).
- 다음 역할이 조심할 것: e2e 는 dev 서버(127.0.0.1:3100)와 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 가 필요하다. `npm run test:unit` 은 이 환경에서 글롭이 확장되지 않아 쓸 수 없고 `node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts`(# pass 95)로 돌려야 한다. `npm run lint` 는 e2e 산출물이 남아 있으면 257 errors 가 나므로 `rm -rf playwright-report test-results` 를 먼저 해야 한다. 신규 spec 의 `freezeGet` 은 GET 만 가로채고 PUT 은 `route.fallback()` 으로 실제 서버에 보낸다 — 최종 판정은 `page.unroute` 뒤의 GET 으로만 한다.
- [러너 11:29] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했고(286·329행의 전체 배열 PUT, `route.ts:63` 의 무조건 저장, `lib/types.ts:16` 의 `lastUp
- [러너 11:30] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 판정 **reject**. 수리가 먼저 볼 파일: `app/admin/page.tsx:95` — 성공 PUT 의 200 본문(새 `lastUpdated` 포함한 완전한 ActivitiesData)을 버리고 `void mutate()` 만 한다. 그래서 GET 실패 중에는 스냅샷이 전진하지 못하고, **다른 관리자가 없는데도** 자기 첫 저장 때문에 이후 상태 변경·행 편집이 전부 409 로 막힌다. 임시 spec 으로 실제 재현(GET 500 주입 → 저장1 200, 저장2 409, 값 저장 안 됨). 같은 파일 243행 `onImported` 의 `mutate(result, { revalidate: false })` 패턴을 쓰면 해소되고 진짜 충돌은 그대로 걸린다. 문구(route.ts:70)와 ADMIN_GUIDE:209 도 함께 맞출 것.
- 확인한 것: 변경 4파일 전문, route.ts·admin/page.tsx·fetchJson.ts·types.ts·import/route.ts·ActivityJsonImporter 관련부, 원장의 `- 실패 재현:` 줄. 전체 e2e 직접 실행 **32 passed** — 원장의 검증 주장은 사실이고 신규 spec 3건은 변경 경로를 정말 지난다(수정 전 409 기대가 200 으로 실패).
- 못 본 것: 단위 테스트·lint·build 는 재실행하지 않았다(러너 verify 가 tsc+build exit 0 기록, e2e 만 내가 직접 돌림). mail/tracking 계열은 이번 diff 밖이라 열지 않았다.
- 승인했더라도 남을 우려(릴리즈 노트에 쓸 것): 이 가드는 read→validate→write 창이 겹치는 동시 쓰기를 못 막는다. 직접 확인 — 평시 두 행을 잇따라 바꾸면 `200, 200` 이고 뒤 PUT 이 앞 PUT 을 조용히 덮어쓴다. 구현 노트의 '같은 밀리초' 보다 창이 넓다. '조용히 되돌리는 것을 막음' 을 무조건으로 적지 말 것.
- 보안·법무 차단 없음(`blocking: []`): 409 분기는 `isAdminRequest` 뒤, `expectedLastUpdated` 는 문자열 비교에만 쓰이고 `lastUpdated` 는 이미 비인증 GET 이 공개하는 값 — 비밀값 비교·오라클·신규 개인정보·의존성 없음. 마이그레이션이 없어 revert 로 완전히 복구된다.
- [러너 11:36] review rejected — 리뷰 거절: app/admin/page.tsx:95 성공한 PUT 의 200 응답 본문을 버리고 `void mutate()` 로 재요청만 한다. route.ts:110-120 이 돌려주는 본문은 새 `lastUpdated` 를 포함�

## 수리 노트
- 지적 3건 모두 맞았다. 직접 재현했다: GET 500 주입 상태에서 관리 화면 저장1 `200` → 저장2 `409`(다른 관리자 없음). 원인은 비평가가 지목한 그대로 `submitActivityChange` 가 200 본문을 버린 것이고, 틀렸다고 판단한 지적은 없다.
- 고친 방법: `app/admin/page.tsx` 에서 성공 응답을 `mutate(result, { revalidate: false })` 로 캐시에 넣어 내 쓰기만큼 스냅샷이 전진하게 했다(243행 `onImported` 와 같은 패턴, 본문 파싱 실패 시에만 기존 `mutate()` 로 폴백). `route.ts` 409 문구는 남을 지목하지 않고 새로 고침 경로를 제시하도록, ADMIN_GUIDE 4.5 는 '내 저장은 스스로 막지 않는다 / 읽기 실패 중에는 자동 갱신이 없으니 새로 고침' 으로 정정했다.
- 재현을 영구 회귀 테스트로 남겼다(`e2e/admin-stale-overwrite.spec.ts` 의 '읽기가 죽어 있어도 혼자 하는 연속 저장은 409 가 되지 않는다'). 수정 전 실패·수정 후 통과를 둘 다 확인했다. 기존 409 spec 의 문구 단언은 새 문구로 맞췄다(단언을 느슨하게 하지 않았다).
- 검증 전부 이 트리에서 실행: e2e 33 passed, 단위 95 pass/0 fail, tsc 통과, lint 무출력, build 성공.
- 확신 없는 곳: ① `mutate(data, {revalidate:false})` 가 SWR 의 `error` 를 지워 읽기 실패 배너가 한 틱(≤1초) 깜빡일 수 있다 — `onImported` 가 이미 같은 패턴이고 폴링이 곧 다시 띄운다. ② read→validate→write 창이 겹치는 진짜 동시 쓰기는 여전히 못 막는다(범위 밖, 가이드 4.5 에 한계로 명시).
- [러너 11:41] repair done — # 수리 요약 (시도 1)  - 지적 3건 모두 맞았다. `submitActivityChange` 가 200 응답 본문(새 `lastUpdated` 포함)을 버리고 `void mutate()` 만 해서, GET 이 500 인 동안 SWR �

## 비평 노트 (2차 — 수리 후)
- 판정 **approve**. 확인한 것: 실제 base `99c2c7d..HEAD` 의 4파일 전문(로컬 `main` 은 `cd9d1f3` 로 낡아 `main...HEAD` 는 쓰면 안 된다), route.ts·admin/page.tsx 전문, activityData.ts `normalizeStoredData`, import/route.ts·fetchJson.ts·ActivityTree.tsx 관련부, ADMIN_GUIDE 3.1·4.5·5.3·5.4. 검증 전부 이 트리에서 직접 실행: tsc exit 0, 단위 95 pass/0 fail, lint 무출력 exit 0, **e2e 33 passed**.
- 테스트가 정말 변경을 검증한다 — `git checkout 01c2d6d -- app/admin/page.tsx` 로 수리만 되돌려 새 회귀 테스트를 돌리자 `:113` 에서 `Expected: 200 / Received: 409` 로 실패했다(혼자 하는 두 번째 저장이 막히는 증상 그대로). 원장의 `- 실패 재현:` 줄도 구현 전 `2 failed / 1 passed` 와 일치한다. 되돌린 파일은 `git checkout HEAD --` 로 복원, 트리 깨끗.
- 수리 노트의 "확신 없는 곳" 둘 다 임시 probe spec 으로 실측했다(실행 후 삭제). ① 저장 성공이 SWR `error` 를 지워 **읽기 실패 배너가 약 1초 사라졌다가 되돌아온다**(250ms 간격 12회 표본: false×4 → true×8). SWR 2.4.1 은 `error` 가 있으면 interval 폴링을 쉬므로, 배너가 지워진 직후 폴링이 되살아나 다시 실패하며 스스로 복구된다 — 경계 있고 `onImported` 와 같은 기존 패턴이라 차단하지 않는다.
- ② **릴리즈 노트에 쓸 것**: 같은 콘솔에서 앞 PUT 응답이 닿기 전에 두 행을 연달아 바꾸면 `200, 200` 이고 한쪽 변경이 **조용히** 사라진다(실측 — 지연 없음: B 의 `완료` 소실 / PUT 500ms 지연: A 의 `진행` 소실). 가이드 4.5 마지막 불릿은 이것을 "두 관리자가 거의 동시에" 로만 적었다 — 한 사람의 빠른 연속 클릭도 같다. 선행 비평의 우려와 같은 자리이며, 이 PR 이 악화시킨 것은 아니다(이전에는 이것이 유일한 동작이었다). "조용히 되돌리는 것을 막음" 을 무조건으로 홍보하지 말 것.
- 승인해도 남는 작은 우려(차단 아님, 공격 경로 없음): (a) `data/activity.json` 을 손으로 만들어 `lastUpdated` 를 빼고 `docker cp` 하면 `normalizeStoredData:79` 가 매 읽기마다 새 시각을 만들어 가드된 PUT 이 영구 409 가 되어 상태 드롭다운·행 편집이 전부 막힌다 — 가이드 3.1 이 형식에 포함시키고 5.4 의 모든 복구 경로는 앱이 만든 파일을 쓰므로 문서화된 길에서는 닿지 않는다. (b) `expectedLastUpdated` 가 문자열이 아니면(null·숫자) 400 이 아니라 가드를 조용히 건너뛴다 — opt-in 설계상 의도이고 UI 는 항상 문자열이거나 생략한다. (c) `onStatusChange` 가 보내는 `targetId`·`newStatus` 는 배열 분기에서 쓰이지 않는다(기존 사정).
- 보안·법무 차단 없음(`blocking: []`): 409 분기는 `isAdminRequest` 뒤에만 있고, `expectedLastUpdated` 는 문자열 동등 비교에만 쓰이며 `lastUpdated` 는 이미 비인증 GET 이 공개하는 값이라 오라클이 되지 않는다. 신규 개인정보 수집·보존·전송 없음, 신규 의존성·암호 구현 없음, 마이그레이션이 없어 revert 로 완전 복구된다. 못 본 것: `npm run build` 는 재실행하지 않았다(러너 verify 의 exit 0 기록에 의존), mail/tracking 계열은 이번 diff 밖이라 열지 않았다.
- [러너 11:49] review approved — 리뷰 승인 (risk=low)
- [러너 11:49] pr created — https://github.com/hkjang/cutover/pull/13
- [러너 11:49] ci passed — 검사 없음 — 정책으로 허용
- [러너 11:49] merge done — 9da830b

## 릴리즈 노트
- `v1.15.0` 주석 태그를 `c304e91`(머지 커밋)에 달았다 — 이전 3개 태그와 같은 방식이다(태그 주석 한 줄 `vX.Y.0 - <한국어 제목>`, 별도 릴리즈 커밋 없음). 확인한 것: `git cat-file tag v1.14.0/v1.12.0`, 태그가 가리키는 커밋이 모두 머지 커밋, `git log -60` 에 릴리즈 커밋 양식이 **없음**.
- 버전 파일은 건드리지 않았다. `package.json` 의 `"version": "0.1.0"` 은 최초 커밋 이후 한 번도 바뀌지 않았고(`git log --follow -p -- package.json` 으로 확인) 버전은 태그와 GitHub Release 에만 존재한다. CHANGELOG.md·docs/RELEASE*.md·`.github/workflows` 는 저장소에 없다.
- 버전 결정: v1.12.0 → v1.13.0 → v1.14.0 이 전부 마이너 증가라 **1.15.0**. 패치로 내리지도, 메이저로 올리지도 않았다.
- 자산: README 132-144행의 절차 그대로 `docker build -t cutover:v1.15.0 .` → `docker save | gzip > cutover-v1.15.0.tar.gz`(84MB, `gzip -t` 통과). 이 빌드가 이미지 안에서 `npm run build` 를 다시 돌려 exit 0 — 이 트리에 `node_modules` 가 없어 릴리즈 단계에서 돌린 유일한 검증이고, tsc·lint·단위·e2e 는 수리·2차 비평이 이 트리에서 직접 돌린 기록(e2e 33 passed, 단위 95)을 인용했다. 로그: `docker-build.log`.
- 릴리즈 노트에서 고친 것: 초안에 e2e **32**건·한계를 "같은 밀리초" 로 적었는데 둘 다 틀렸다. 2차 비평 노트대로 **e2e 33건(신규 4건)** 으로, 한계는 **read→validate→write 창이 겹치는 경합(한 사람의 빠른 연속 클릭 포함)** 으로 고쳤다. 비평가가 요구한 대로 "조용히 되돌리는 것을 막음" 을 무조건으로 쓰지 않고 "목록이 실제로 뒤처진 저장을 막는다" 로 한정했다. 배너 1초 깜빡임도 알려진 동작에 적었다.
- 원격에 아무것도 보내지 않았다(`git push`·태그 푸시·`gh release`·`docker push` 없음). 태그는 로컬에만 있고 트리는 깨끗하다. GitHub Release 생성과 자산 업로드는 러너가 한다.
- 런치 등급은 **tier 2**(기존 관리자에게 의미 있는 버그 수정, 새 이야기는 아님) — 저장소 관례인 릴리즈 노트 + 가이드 갱신으로 충분하고 그 이상의 홍보 작업은 하지 않았다. 노트 끝에 배포 후 확인할 신호 두 개(다른 관리자와 함께 쓸 때 409 배너가 나오는지 / 혼자 쓸 때 나오지 않는지)를 적어 두었다.
