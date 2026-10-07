- 과제: 유효한 긴 ID를 가진 부모에도 하위 작업을 추가할 수 있도록 신규 ID 길이 제한 (가치 3 / 위험 2 / 작업량 S)
- 왜: `lib/treeUtils.ts:addActivity`는 부모 ID에 `-${Date.now()}`를 붙이므로, 정상 업로드 가능한 120자 ID 부모 아래에 추가하면 134자 ID가 생성되어 `validateActivityImport`에서 거절된다. 부모 ID 길이에 영향받지 않는 짧은 신규 ID를 만들면 유효한 트리를 관리자 화면에서 계속 확장할 수 있다.
- 수용 기준:
  1) 120자 ID의 정상 부모를 저장하고 `PUT {action:'add', parentId:<부모 ID>}`를 보내면 200이며, 뒤이은 GET에 자식 1개가 추가되어 있다. 자식 ID는 비어 있지 않은 120자 이하 문자열이고 기존 항목 ID는 변하지 않는다.
  2) 새 자식은 `parentId`가 원래 부모 ID와 정확히 같고 `level`은 부모+1이다. 기존 입력 배열은 불변이며 시간 `미정`, 제목 `새 액티비티`, 상태 `대기`, 배열 끝 추가 계약을 유지한다. 최상위 추가(`parentId:null`)도 정상이다.
  3) 단위 테스트가 실제 `addActivity` 결과를 실제 `validateActivityImport`에 넘겨 긴 부모 아래 자식 생성 성공을 증명한다. 같은 자식 아래 다시 추가해도 검증을 통과해야 한다. 수정 전에는 `activities[1].id`의 120자 제한 오류로 실패하고 수정 후 통과한다.
  4) 고정된 같은 `Date.now()` 아래에서 두 번 연속 추가해도 서로 다른 ID가 만들어지고 결과 전체가 검증을 통과한다. 시간 모킹은 테스트 종료 시 복구하며 임의 sleep·확률적 타이밍에 기대지 않는다.
  5) API 회귀 테스트는 실제 로그인 세션/개발 서버와 격리 데이터 파일을 사용한다. PUT 응답과 후속 GET으로 증명하며 폴링에 보인다는 이유만으로 저장 성공을 판단하지 않는다.
- 건드릴 파일: `lib/treeUtils.ts:addActivity` — 부모 ID에 독립적인 짧은 ID 생성으로 교체(프로덕션 **1파일**); `lib/treeUtils.test.ts` — 기존 node 헬퍼/describe 관례로 addActivity와 검증기의 연결 테스트, 루트/불변성/같은 시각 생성 보호; `e2e/tree-ops.spec.ts` — 기존 loginAsAdmin·activities·beforeEach/afterEach를 활용하여 긴 ID 부모의 추가 PUT/GET 회귀 테스트. 총 3파일, 신규 의존성 없음.
- 검증 명령:
  - 이번 정찰에서 실제 성공: `node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts` → **95 pass / 0 fail / 0 skipped**, Node v22.23.1. node_modules 없이 실행된다.
  - 구현 중 집중 검증: `node --test lib/treeUtils.test.ts lib/activityData.test.ts` (위 전체 실행에 두 파일 포함, 이 축약 명령 자체는 정찰 미실행).
  - 의존성이 없으면 구현 세션에서 `npm ci --legacy-peer-deps`; 코드 작성 전 AGENTS.md에 따라 설치된 `node_modules/next/dist/docs/`에서 Server/Client Components 관련 가이드를 찾아 읽는다. 현재 node_modules가 없어 정찰에서는 가이드 미열람.
  - 구현 후 `npx tsc --noEmit`, `npm run lint`, `npm run build` (공용 treeUtils의 클라이언트 번들 호환 확인에 build 필요). 이어 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npx playwright test e2e/tree-ops.spec.ts`, `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e`. E2E와 단위 테스트는 동시에 돌리지 않는다.
  - 마지막 `git diff --check`. lint는 E2E보다 먼저 실행한다. 이전 E2E 산출물이 이미 있으면 생성물임을 확인해 정리한 뒤 실행하고, ESLint ignore 설정을 이 과제에 섞지 않는다.
  - 정찰 검증 한계: 의존성 설치·tsc·lint·build·E2E는 이번에 실행하지 않았다. 위 npm/Playwright 명령은 실제 package.json·playwright.config.ts와 이전 성공 기록에 근거한다. 이전 회차 전체 E2E 기준선은 38건/13 spec(이번 재실행 미확인).
- 위험과 피할 것: `treeUtils`는 `app/admin/page.tsx`에서 클라이언트도 import한다. `node:crypto` 정적 import나 모듈 초기화 시 UUID 호출을 넣지 말고, 서버에서 호출되는 addActivity 안에서 `globalThis.crypto.randomUUID()`를 사용하는 최소 변경을 권장한다(Node 22에서 실제 가용/36자 확인). 현재 운영 addActivity 호출자는 activities PUT 하나이며 브라우저는 API만 호출한다. 기존 ID 재작성·ID 상한 완화·parentId 형식 변경·새 패키지·Date.now만의 짧은 ID로 교체는 금지한다. auth, proxy, 파일 I/O/백업, 메일, 업로드 API, 리비전/동시 쓰기 잠금, 상태 전파, Docker/Amplify/workflows, test:unit/ESLint 설정은 범위 밖이다. UUID 중복에 대한 별도 재시도 체계까지 넓히지 말고 기존 검증기를 유지한다. level 50 부모 아래 추가 거절은 별개 정책으로 그대로 둔다.
- 차선 후보: `validateActivityImport` 제한값 경계 테스트 보강(가치 2 / 위험 1 / S) — 신규 ID 형식에 의존하는 외부 계약이 실제로 발견되어 주 과제가 성립하지 않을 때만, `lib/activityData.test.ts`에 120/121자 ID·100/101자 dashboardTitle·5000/5001개 경계를 추가한다. 이미 반려된 ESLint ignore 과제나 상태 전파 통합으로 전환하지 않는다.

근거 및 재현

- 기준 커밋 `51ffd03`. `lib/treeUtils.ts:47-59`는 부모 ID 연결과 기본 필드 설정, `lib/activityData.ts:177-182`는 ID 상한, `app/api/activities/route.ts:90-91`는 add 분기이며 뒤에서 검증 오류를 400 INVALID_DATA로 반환한다.
- `components/ActivityTree.tsx:108`의 하위 추가 → `app/admin/page.tsx`의 onAddActivity → PUT → addActivity → validateActivityImport가 실제 사용자 경로다.
- 정찰의 인메모리 호출 결과: valid parent count=1, new child id length=134, parent unchanged=true, issues=[{path:'activities[1].id', message:'id는 120자 이하여야 합니다.'}]. HTTP 400은 읽은 라우트 분기에서 확인한 예상 결과이며 실제 HTTP 재현은 구현자가 수행한다.
- `lib/treeUtils.test.ts`는 현재 collectDescendantIds/deleteActivity 7건만 다루고 addActivity 테스트는 없다. 기존 테스트 import는 `.ts` 확장자를 쓴다.
- 실제 소스 검색에서 부모 ID 접두사를 해석하는 운영 코드는 발견하지 못했다. collectDescendantIds/deleteActivity/buildTree는 parentId를 쓰고 ADMIN_GUIDE 4.3도 ID 형식 무관을 명시한다. 저장소 밖 소비자의 ID 형식 의존은 미확인이다.

선택안 비교 (solution-exploration)

- 권장: addActivity 함수 안에서 부모와 무관한 UUID를 생성. 프로덕션 1파일이며 긴 ID와 같은 시각 ID 중복의 원인을 함께 없앤다. 기존 ID는 유지된다. 전제는 ID가 불투명한 식별자라는 현재 코드·문서 계약이다.
- 대안: 부모 ID 접두사를 잘라 시간/난수 접미사를 붙이기. 접두사 모양 일부 보존 가능하지만 길이 배분·충돌 처리 규칙이 늘며, 문서상 필요 없는 규칙이라 선택하지 않는다.
- 대안: ID 검증 상한을 늘리거나 긴 ID 부모에서 추가 버튼 비활성화. 전자는 누적 길이 문제를 미루고 파일 정책까지 바꾸며, 후자는 유효한 데이터의 편집 제한을 제품에 남기므로 선택하지 않는다.
- 현상 유지: 데이터 손상은 검증기가 막는다. 그러나 정상 입력에서 기능이 실패하는 결정적 재현이 있어 단순 오류 문구 중복 제거·추측성 UX보다 이번 수정 가치가 높다.

구현 순서 및 체크포인트 (implementation-planning; 구현 상태는 모두 미착수)

1. [ ] 기존 기준선 및 클라이언트 가이드 확인 후 `lib/treeUtils.test.ts`에 회귀 테스트 작성. 집중 단위 명령으로 기존 동작의 긴 ID 거절 실패를 확인한다. 체크포인트: 구현자 자체 검토, 예상 실패 원인이 ID 상한인지 확인하고 다음 단계 진행(별도 사람 승인 없음).
2. [ ] `lib/treeUtils.ts:addActivity`의 ID 생성만 변경하고 의도 주석을 남긴다. 집중 단위 및 전체 단위 명령으로 통과/기존 계약 유지 확인. 체크포인트: 실패 원인이 바뀌면 과제서를 정정하고 범위 확장하지 않는다.
3. [ ] `e2e/tree-ops.spec.ts`에 로그인 API 기반 회귀 테스트 추가. 120자 부모 seed PUT → add PUT 200 → GET 부모/자식 관계 검사. 기존 복구 hook을 유지한다. 타입·lint·build·집중 E2E·전체 E2E·diff 검사 후 결과 기록. 체크포인트: 실패를 숨긴 완료 선언 금지, 환경 실패와 코드 실패 구분(별도 사람 승인 없음).

작업량 추정 및 여유 (estimating-and-contingency)

- 포함: 환경/기준선 3~5분 + 단위 테스트/작은 수정 8~10분 + API 테스트 5~7분 + 빌드·전체 검증/기록 7~10분 = 기본 23~32분(bottom-up). 모두 순차 실행, 각 항목에 여유를 중복 산입하지 않았다.
- 알려진 불확실성의 contingency: 의존성 설치·브라우저/개발 서버 기동 편차 5~10분. 합계 28~42분을 실무 예상 범위로 둔다. 신뢰는 중간이며 표본 기반 확률 구간이 아니다. 45분 내 완료는 설치/빌드가 종전처럼 수 분 내 끝난다는 가정에 의존한다.
- 유사 사례 교차검토: 10-06의 프로덕션 1파일+API 테스트 1파일(+문서) 회차와 범위가 비슷하고, 이번에는 라우트 계약을 바꾸지 않는 대신 단위 회귀를 추가한다. 당시 실측 소요시간이 없으므로 독립적인 정량 예측으로 위장하지 않으며, 범위 기준 S 판정만 교차확인한다.
- management reserve: 이 무인 회차에는 별도 미지 범위 예산을 배정하지 않았다(0분). 외부 ID 의존이나 별도 저장 경합이 드러나면 이번 작업에 흡수하지 말고 기록한다. 의존성/기준선 확인 직후 남은 시간을 재추정하며 검증을 생략해 45분에 맞추지 않는다.

적용 스킬: 전용 Skill/skills.list/skills.read 도구가 제공되지 않아 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md` 정본을 직접 읽었다(뒤 두 경로도 동일 plugins 루트). pmo의 references/sources.md도 확인했으며 위 시간은 저장소 범위에 근거한 자체 추정으로 외부 기관의 통계로 주장하지 않는다.
