- 과제: 단순 실행 상세에서 다른 실행으로 이동할 때 로그·커서·지연 응답을 실행 ID별로 격리하기 (가치 4 / 위험 1 / 작업량 S)
- 왜: `SimpleRunDetailPage`는 같은 라우트의 id가 바뀌어도 로그 상태를 재사용하고, `loadStoredLogs`의 이전 비동기 응답이 새 실행의 로그·커서를 덮어쓸 수 있다. 실행 ID가 바뀔 때 상세 상태의 수명을 분리하면 다른 실행 로그의 표시·복사와 이전 커서로 여는 스트림을 막아 배포 결과를 올바르게 판단할 수 있다.
- 수용 기준:
  1) 실제 App에서 A→B 상세 이동 시 A 로그·오류·잘림·끊김 안내를 B에 남기지 않는다. B의 상세/저장 로그를 기다리는 중에는 A 로그를 복사할 수 없고 A를 가리키는 로그 다운로드 링크도 남지 않는다.
  2) A의 지연된 저장 로그 성공/실패가 B 조회 중 또는 B 조회 완료 후 도착해도 B의 로그·오류·로딩·커서에 영향을 주지 않는다. B 조회 실패 시에도 A 로그를 대신 표시하지 않는다.
  3) A 스트림은 이동 시 닫힌다. B의 상세 상태가 RUNNING이고 B 저장 로그 조회가 끝난 뒤에만 B 스트림 하나를 열며 after는 B의 lastId(빈 로그는 0)이다. B 로딩 중 A 커서로 잠깐 열었다 닫는 스트림도 없어야 한다.
  4) 같은 id의 새로 고침·수동 재연결·max_duration 재연결에서는 컴포넌트를 다시 만들지 않는다. 기존 저장 로그 수집, 단일 연결, 마지막 커서 재개 테스트가 그대로 통과한다.
  5) 실제 App·BrowserRouter·프로덕션 컴포넌트 렌더 테스트로 지연 응답과 이동을 재현한다. 고유 메시지 A-only/B-only를 쓰고 화면뿐 아니라 실제 clipboard.writeText 결과에 A가 없는지 검증한다. API/전송 경계만 기존 방식으로 대체하며 소스 문자열 검사나 별도 모형 훅으로 대체하지 않는다.
- 건드릴 파일:
  - `web/src/pages/simple/SimpleRunDetailPage.tsx:SimpleRunDetailPage` — export 페이지는 useParams로 id를 얻는 얇은 래퍼로 두고, 현재 본문과 모든 훅을 같은 파일의 모듈 수준 내부 컴포넌트(예: RunDetailContent, id prop)로 옮겨 `<RunDetailContent key={id} id={id} />`로 마운트한다. 내부 컴포넌트를 렌더 함수 안에 정의하지 말 것. helper export들은 그대로 둔다. JSX 하위 박스 하나에만 key를 주면 상위 훅 상태는 분리되지 않으므로 해결이 아니다.
  - `web/src/pages/simple/SimpleRunDetailPage.stream.test.tsx:renderRunDetail/TestEventSource` — id별 simpleRun/simpleRunLogs 응답과 직접 resolve/reject하는 Promise로 라우트 전환 테스트 추가. 기본 renderRunDetail은 simpleRunLogs mock을 덮어쓰므로 필요한 응답을 주입할 수 있게 테스트 helper만 작게 조정한다.
- 검증 명령:
  - 최초 환경 준비: `cd web && npm ci` (현재 원본 작업 트리에는 node_modules 없음).
  - 집중 검증: `cd web && npm test -- --run src/pages/simple/SimpleRunDetailPage.stream.test.tsx src/pages/simple/SimpleRunDetailPage.test.ts`
  - 회귀 검증: `cd web && npm test -- --run`
  - 타입/빌드: `cd web && npx tsc -b --noEmit`, `cd web && npm run build`
  - diff: 저장소 루트에서 `git diff --check`; VERSION, web/dist 등 생성물은 변경 목록에 넣지 않는다.
  - 정찰 기준선 결과: 산출물 디렉터리 assets/web에 원본 웹을 수정 없이 복사해 npm ci 성공, ./node_modules/.bin/tsc -b --noEmit 종료 0 확인. 전체 npm test -- --run은 13파일/149테스트 통과(87.02초, Vitest 3.2.7). 선택 파일의 기존 스트림 테스트 11건과 helper 테스트 16건도 이 실행에 포함됐다. npm run build와 Go/DB 테스트는 정찰에서 실행하지 않았다. 새 결함의 실패 테스트는 정찰에서 작성/실행하지 않았으며, 위 영향은 확인한 코드에서 도출한 판단이다.
- 위험과 피할 것: 범위는 실행 ID 변경 경계 하나다. 공통 useAsync·App 라우팅·api/client·SSE 프로토콜·백엔드·auth/migrations/workflows·배포 화면·전체 모드 로그는 변경하지 않는다. 같은 id에서 새로 고침을 연속 클릭했을 때의 요청 순서 경쟁, 네트워크 취소/AbortSignal 추가, 페이지네이션 최적화는 별도 과제다. key에 status/loading/streamAttempt를 넣으면 한 줄 도착 또는 재연결마다 초기화돼 이전 스트림 회귀가 재발한다. 지연된 과거 요청의 실제 네트워크 취소까지 요구하지 않으며 새 실행 상태를 침범하지 않으면 된다. 브라우저가 close 뒤 error를 발사한다는 가정으로 가짜 이벤트를 만들어 정당화하지 말 것.
- 차선 후보: `docs/simple-mode.md` 서두·`docs/architecture.md:심플 모드 경계`의 Harbor 미사용 단정을 선택적 복제 동작과 일치시키기 (가치 2 / 위험 1 / S). 1순위 경로가 현 코드에서 재현되지 않을 때만 전환한다. README는 이미 복제/앱 배포를 설명하고 simple-mode.md:188 이후에도 상세 설명이 있으므로 해당 서두와 architecture 문단만 정합화한다. 근거는 simple.go:executeSimpleRun의 ReplicationEnabled 조건과 runReplication의 loadHarborRegistry/startReplication 호출. 기능/보안 경계는 그대로 두고 `git diff --check`와 실제 소스 대조로 확인한다.

## 확인한 근거와 재현 설계
- main@da054c9, VERSION 0.5.31. App.tsx:41는 `/simple/runs/:id`에 같은 SimpleRunDetailPage를 마운트한다. SimpleRunDetailPage.tsx:182~277에 id별 key/reset 없이 로컬 상태와 ref가 있고 loadStoredLogs의 try/catch/finally 모두 과거 요청 유효성 검사를 하지 않는다.
- useAsync.ts:reload에는 requestSequence 방어가 있지만 기존 data를 지우지 않으므로 id 전환 렌더에서는 A의 상태를 볼 수 있다. SimpleRunDetailPage의 별도 로그 요청은 그 방어를 사용하지 않는다. 스트림 effect는 같은 렌더의 live/loadingLogs 값을 읽으므로 reset effect의 setState만으로 순간적인 잘못된 연결을 막았다고 주장하지 않는다.
- 사용자가 도달하는 경로는 SimpleRunDetailPage.tsx:435의 형제 패키지 RouterLink다. 테스트는 batchSiblings를 채워 실제 링크를 누르는 방식을 우선 사용한다. 대안은 이미 ReleaseDetailPage.stream.test.tsx:260에서 쓰는 act + history.pushState + PopStateEvent다. id별 데이터의 id도 라우트와 일치시킨다.
- 최소 테스트 시나리오: (a) A 로그 로드 후 B로 이동, B 응답 지연/실패 — 이전 복사·다운로드·커서 누수 없음; (b) A의 로그 요청을 보류한 채 B로 이동, B를 먼저 완료한 뒤 A를 성공/실패시킴 — B 표시와 복사 유지; (c) B도 조회 중인 상태에서 A 완료 — B 스트림 조기 생성 없음; (d) 같은 B에서 기존 수동/timeout 재연결 테스트 유지. mock 시간이 아닌 Promise 해결 순서로 경합을 결정한다.

## 대안 비교와 선택 근거 (solution-exploration)
- 선택: 동일 파일에 id-keyed 내부 상세 컴포넌트. 새로운 의존성 없이 run.data·로그·커서·로딩 수명을 함께 묶는다. 핵심 가정은 실행 간 로컬 상태를 유지해야 하는 UX 계약이 없다는 것이며 현재 화면과 테스트에서 그런 계약은 발견하지 못했다.
- 대안: 요청 세대 번호 + route identity로 모든 setState/커서/스트림을 보호. 같은 실행 내 중복 요청에도 확장 가능하지만 reset·finally·live 가드를 빠뜨릴 면적이 넓어 이번 ID 경계 과제에는 더 비싸다.
- 대안: App에서 key를 공급하거나 공통 useAsync에서 data를 초기화. 다른 페이지/권한 라우트에도 영향을 주므로 범위를 벗어난다.
- 유지: 변경하지 않으면 지연된 A 응답이 B 상태를 덮는 경로가 남는다. 문구만 고쳐서는 결과 혼합을 해결하지 못한다.
- 기존 보류의 공용 로그 helper 개명은 동작 이득이 없고, DB 통합 테스트는 DB/비동기 해제 준비가 필요하며, rAF 배칭은 개선량이 미측정이다. 이 과제는 사용자 오판을 막으면서 프로덕션 한 파일로 끝난다.

## 구현 순서·증거·체크포인트 (implementation-planning)
1. [pending] 현재 집중 검증 명령으로 기준선 확인 → 위 두 파일만 변경해 id 경계와 실제 렌더 회귀 테스트를 한 묶음으로 완성 → 같은 집중 명령 통과. 구현 전 임시 실패 재현 결과는 기록하되 빌드가 깨진 스텁을 중간 완료로 남기지 않는다. 체크포인트: 자동 검증, 사람 승인 없음.
2. [pending] 전체 npm test, 타입 검사, build, diff 검사 순서로 마무리 → 1 프로덕션+1 테스트 파일 범위와 수용 기준 확인. 체크포인트: 자동 검증, 사람 승인 없음. 소스와 계획이 다르면 먼저 brief를 수정하고 차선 전환 이유를 기록한다.

## 추정 근거·여유 (estimating-and-contingency)
- 방법: bottom-up — 기존 테스트 배선/기준선 4~6분, id 경계 분리 3~5분, 지연 응답·라우트 테스트 10~13분, 전체 검증/정리 5~7분 = 기본 22~31분.
- 알려진 변동의 예비시간(contingency): 테스트 helper의 기본 mock 덮어쓰기와 Promise flush 조정에 4~6분, 합계 26~37분. 신뢰도는 중간의 판단 추정이며 통계적 신뢰구간이 아니다. npm 설치가 가능한 환경을 전제하며 네트워크 대기 시간은 미확인이다.
- 교차 확인(analogous): 2026-09-30 릴리즈 이동 결함도 프로덕션 1+테스트 1파일로 끝났다. 이번에는 지연된 저장 조회를 포함해 검증량이 더 크다. 과거 실제 소요 시간은 제공되지 않아 시간 비율로 환산하지 않았다.
- 관리 예비시간(management reserve)은 위 합계에 숨겨 넣지 않는다. 45분 회차의 나머지는 실행자가 범위 밖 변경에 자동 소비하지 말고, 예상 밖 계약이 나오면 차선 후보로 전환하거나 재추정한다. 사람에게 질문하지 않는다.
- 포함: UI 상태 경계·동작 테스트·타입/빌드. 제외: 배포/릴리즈 버전·의존성 업그레이드·DB 검증·취소 API·동일 id 중복 새로 고침 경합. 분리 후 첫 라우트 테스트 통과 시 추정을 갱신한다.
- 적용 스킬: 로컬 headcount의 pmo/estimating-and-contingency, technology/implementation-planning, technology/solution-exploration SKILL.md를 직접 읽었다. 전용 Skill 도구는 세션에 없었으며 외부 비용 산정 수치를 가져오지는 않았다.
