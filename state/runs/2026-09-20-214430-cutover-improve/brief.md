- 과제: 관리자 활동 저장의 모든 진입점에서 실패를 표시하고 편집 내용을 보존하며 삭제 확인을 한 번으로 통일 (가치 4 / 위험 1 / 작업량 M)
- 왜: app/admin/page.tsx는 상황판 제목 저장 2곳과 트리 조작 5곳 모두 PUT 응답 상태를 무시하고, ActivityTree.handleSave는 서버 결과 전에 편집을 닫아 400·401·통신 실패가 사용자에게 숨겨진다. 저장 실패를 명확히 알리고 수정 중인 입력을 유지하면 통제관이 미반영 작업을 알아차리고 재시도할 수 있으며 하위 삭제 시 중복·잘못된 제목의 확인창도 없어진다.
- 수용 기준:
  1) 상황판 제목 Enter/저장 버튼, 활동 제목·시간 저장, 상태 변경, 삭제, 위/아래 이동, 하위/같은 레벨 추가 모두 공통 응답 처리로 연결한다. 400/500 등 비성공 응답에는 role="alert" 영역에 서버 error와 가능한 details(path/message)를 표시하고 비JSON 응답과 fetch reject에는 일반 오류 문구를 표시한다. 실패 후 저장 성공 처리·명시적 mutate·편집 종료가 실행되지 않는다(기존 SWR 정기 GET은 유지).
  2) 어떤 활동 PUT이든 401이면 기존 handleUnauthorized를 호출해 표시용 쿠키 제거·관리자 로그인 화면·재로그인 안내가 나타난다. 서버 인증 구현은 변경하지 않는다. 400 및 통신 오류에서는 제목/시간 입력값과 편집 상태를 보존하며, 정상 재시도 시 저장 결과가 화면과 GET에 반영되고 이전 오류가 해제된다.
  3) 삭제 확인은 실제 대상 node.title을 아는 ActivityTree에서 한 번만 한다. 취소하면 PUT이 없고 데이터가 유지된다. 확인하면 한 번의 PUT으로 해당 노드와 parentId 하위만 삭제된다. 부모 AdminPage 콜백의 중복 confirm을 제거한다.
  4) 실제 브라우저→기존 API를 통과하는 e2e로 증명한다: 101자 상황판 제목(Enter와 버튼 각각) 및 빈 활동 제목/시간을 저장해 실제 400·오류 상세·입력 보존·GET 불변 확인 후 정상 값으로 재저장한다. 별도 케이스에서 관리자 로그인 완료 후 같은 브라우저 context.request.delete('/api/auth/session')로 실제 쿠키를 만료시키고 각 저장 진입점(제목 2개, 트리 5종)을 조작해 실제 PUT 401 및 로그인 화면 복귀를 확인한다. 인증되지 않은 별도 request fixture로 로그아웃하면 브라우저 쿠키는 안 지워지므로 주의한다.
  5) 통신 실패는 로그인·초기 로딩 후 context.setOffline(true)로 실제 브라우저 네트워크를 끊고 저장을 눌러 오류와 편집 보존을 확인한 뒤 online 복구·재시도로 검증한다. 삭제는 임의 ID 3단 트리에서 dialog 횟수/대상 제목/취소 시 PUT 0회/확인 시 PUT 1회와 GET 잔존 목록을 검증한다. e2e/tree-ops.spec.ts 기존 상태 전파 회귀 2건도 유지한다. fetch 대역·route.fulfill 가짜 401·소스 문자열 검사는 성공 근거로 쓰지 않는다.
  6) docs/ADMIN_GUIDE.md 4.2의 '확인창 두 번'과 4.4의 '100자 초과 시 편집만 닫힘'을 새 동작으로 고치고 저장 오류·재로그인 안내를 추가한다. PDF는 이번 범위 밖이며 불일치를 기록한다.
- 건드릴 파일:
  - app/admin/page.tsx: AdminPage, handleUnauthorized, 제목 onKeyDown/onClick, onStatusChange/onUpdateActivity/onDeleteActivity/onMoveActivity/onAddActivity — 컴포넌트 안의 작은 공통 PUT 처리 함수와 오류 상태(조건부 return보다 위에 Hook 선언), 성공 여부 반환, 모든 7개 fetch 경로 배선. 제목 저장 두 곳도 하나의 핸들러를 공유한다. 오류 표시 방식은 components/ActivityJsonImporter.tsx:handleUpload 및 components/MailSettings.tsx의 기존 error/details 처리를 참고하되 이 컴포넌트들은 수정하지 않는다.
  - components/ActivityTree.tsx: Props.onUpdateActivity, handleSave — 비동기 성공 여부를 기다린 뒤에만 setIsEditing(false); 재귀 전달 타입도 맞춘다. 삭제 confirm은 이곳에 유지. 필요한 편집/저장/삭제 아이콘에 접근 가능한 이름을 붙여 e2e가 아이콘 CSS나 DOM 순번에 의존하지 않게 한다.
  - e2e/admin-mutations.spec.ts (신규 제안): 기존 e2e/tree-ops.spec.ts의 loginAsAdmin·activities·beforeEach/afterEach 및 e2e/activity-import.spec.ts 패턴 참고. 격리 데이터 원본(activities와 dashboardTitle)을 보관·복구하고 실패 시에도 offline 복구와 재로그인 뒤 정리한다.
  - docs/ADMIN_GUIDE.md: 4.2·4.4 및 필요한 오류 안내 문단만 갱신.
- 검증 명령:
  - 정찰에서 실제 실행: `npm run test:unit` → Node v22.23.1, 72 tests / 24 suites / 72 pass / 0 fail. `git status --porcelain` → 빈 출력.
  - 구현 환경 준비: 현재 node_modules 없음. 구현자는 `npm ci --legacy-peer-deps`(Dockerfile의 설치 방식)를 실행한 뒤 AGENTS.md가 요구하는 `node_modules/next/dist/docs/`의 관련 클라이언트 컴포넌트·데이터 갱신 가이드를 먼저 읽는다. 정찰은 코드 작성 없이 읽기만 했으며 로컬 Next 가이드는 부재로 미확인.
  - 구현 후: `npm run lint`, `npx tsc --noEmit`, `npm run test:unit`, `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e -- e2e/admin-mutations.spec.ts e2e/tree-ops.spec.ts`, 이어 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome npm run test:e2e`, `npm run build`. package.json·playwright.config.ts에서 확인한 저장소 명령이며 시스템 Chrome 존재 확인. lint/tsc/e2e/build는 이번 정찰에서 실행하지 않았으므로 통과 미확인.
- 위험과 피할 것: auth/**·lib/adminSession.ts·proxy.ts·Dockerfile·저장/백업 함수·메일 발송·상태 전파 규칙을 바꾸지 않는다. 이전 parentId 수정은 main@530dfd3에 반영되어 재구현 불필요. 실패를 성공으로 취급하지 않되 2xx PUT 후 SWR 재검증 실패를 '저장 실패'로 혼동하지 않도록 경계를 둔다. 편집 Promise의 예외가 이벤트 핸들러 밖으로 새지 않게 한다. 정기 SWR GET 때문에 '실패 후 모든 GET이 0회' 같은 검증은 잘못이다. 실패 trace가 남으면 기존 eslint 설정이 playwright-report 번들을 검사할 수 있다(이전 회차 실측); 이를 회피하려고 제품 코드나 전역 설정을 바꾸지 말고 산출물 문제를 분리한다. 의존성 대규모 갱신·SSO·SMTP 타이머·동시 편집 충돌 처리는 제외.
- 차선 후보: validateActivityImport 실제 함수의 경계·정규화·오류 목록 단위 테스트 — 1순위가 이미 해결되었거나 현재 코드 구조와 달라 성립하지 않을 때만 선택. lib/activityData.test.ts 신규, 대상 함수/ActivityImportValidationError/MAX_ACTIVITY_COUNT는 lib/activityData.ts에 있다. 현재 `import type ... from './types'`는 Node 22에서 제거되므로 확장자 수정은 필요 없다(동적 import 및 빈 activities 검증 호출 성공을 정찰에서 확인). 입출력·타입 경계·trim 후 중복·부모 부재·level·순환·진행중 정규화·50개 상세+생략 안내를 실제 함수로 검증하고 readActivityData/writeActivityData는 호출하지 않는다.

추정 및 실행 순서: M, 총 45분 상한 = 실제 400/401 회귀 재현 및 e2e 작성 10분 → 공통 PUT/편집 완료 계약/confirm 정리 15분 → 문서·전체 검증 10분 → 설치·브라우저·lint 산출물 문제 예비 10분. 설치/빌드 시간은 환경에 따라 달라 미확인. 초과 시 새 기능을 늘리지 말고 남은 검증을 명시한다.
선택 근거: 사용자에게 바로 드러나는 저장 실패를 해결하고 보호 경로를 피할 수 있다. 단위 테스트 공백/DX 정리는 차순위이며 인증 변경은 미반영 선행 작업과 하위 호환 위험, 상태 전파 통합은 의미 변경 위험이 더 크다.
스킬 제한: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration을 제공하는 Skill/skills.list/skills.read 도구가 현재 도구 목록에 없고 로컬 스킬 검색에서도 찾지 못했다. 해당 스킬의 반환 형식은 미확인으로, 준수했다고 주장하지 않는다. 위 추정·대안·수용 기준은 사용자 프롬프트에 따라 작성했다.
