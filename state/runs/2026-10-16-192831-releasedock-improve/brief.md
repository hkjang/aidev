- 과제: 단순 실행 상세를 실행 ID 로 키를 준 내부 컴포넌트로 감싸 로그·커서·지연 응답을 실행마다 격리하기 (가치 4 / 위험 1 / 작업량 S)
- 왜: `SimpleRunDetailPage`(web/src/pages/simple/SimpleRunDetailPage.tsx)는 `/simple/runs/:id` 라우트 하나에 계속 마운트된 채로 `useParams()` 의 `id` 만 갈아타므로, 같은 페이지의 형제 패키지 링크(`:435` `to={`/simple/runs/${item.id}`}`)로 다른 실행으로 이동해도 `logs`·`logTruncated`·`logError`·`streamLost`·`lastIdRef` 가 전부 이전 실행의 값으로 남는다. 게다가 `loadStoredLogs`(`:204-223`)에는 요청 유효성 가드가 없어 A 의 느린 응답이 B 의 응답 뒤에 도착하면 `setLogs(stored.lines)`·`lastIdRef.current = stored.cursor` 가 B 의 화면을 A 의 로그로 덮고, 그 뒤 B 의 스트림은 A 의 커서에서 이어받아(`:239` `?after=${lastIdRef.current}`) B 의 줄을 건너뛰거나 되돌린다 — 읽는 사람은 다른 실행의 출력을 지금 실행의 로그로 보고 배포 성공/실패를 판단한다.
- 수용 기준:
  1) 형제 패키지 링크로 실행 A → B 로 이동하면 A 의 로그 줄이 화면에서 사라지고 B 의 저장 로그만 남는다(실제 App 라우트를 렌더해 `getByText`/`queryByText` 로 확인).
  2) A 의 저장 로그 응답이 B 로 이동한 **뒤에** 늦게 resolve 해도 화면에 A 의 줄이 들어오지 않고, B 의 라이브 스트림이 B 의 커서로 열린다(스트림 URL 의 `after=` 값으로 확인).
  3) A 에서 떠 있던 잘림 안내(`LOG_TRUNCATED_NOTICE` 기반 Alert)·로그 오류·끊김 경고(`다시 연결`)가 B 화면에 남아 있지 않다.
  4) 기존 24건 내외의 `SimpleRunDetailPage.stream.test.tsx` 가 전부 그대로 통과한다(스트림 재연결·`end` 프레임·`after` 커서 계약이 변하지 않았다는 증거).
- 건드릴 파일 (프로덕션 1 파일 + 테스트 1 파일):
  - `web/src/pages/simple/SimpleRunDetailPage.tsx` — 지금의 `SimpleRunDetailPage()` 본문(`:184` 부터 끝까지)을 `function RunDetail({ id }: { id: string })` 로 **이름만 바꿔** 옮기고(`useParams()` 줄 `:185` 는 제거하고 `id` 를 prop 으로 받는다), 그 자리에 얇은 래퍼를 새로 export 한다:
    `export function SimpleRunDetailPage() { const { id = '' } = useParams(); return <RunDetail key={id} id={id} />; }`
    `key={id}` 가 실행이 바뀔 때 React 로 하여금 언마운트/재마운트하게 해 모든 `useState`·`useRef`(`lastIdRef` 포함)와 `useAsync` 의 `data` 가 함께 새로 시작하고, 이전 인스턴스의 지연 응답은 언마운트된 컴포넌트의 setState 가 되어 화면에 닿지 못한다. 상태 reset effect 를 새로 쓰지 말 것 — 리셋 목록을 손으로 유지하는 방식(전체 모드가 v0.5.24 에 그렇게 고쳤다)은 상태를 하나 더할 때마다 빠뜨릴 자리를 만든다.
  - `web/src/pages/simple/SimpleRunDetailPage.stream.test.tsx` — 수용 기준 1~3 의 신규 테스트. 이 파일의 `renderRunDetail` 은 `api.simpleRunLogs` mock 을 **기본 빈 로그로 덮어쓴다**(정찰 확인 필요 항목 참조). 실행별로 다른 로그·지연 resolve 를 주려면 그 mock 을 run id 로 분기하도록 조정하거나, 지연 응답용 헬퍼를 따로 만들 것. `api.simpleRun` mock 은 A 의 `batchSiblings` 에 B 를 넣어 형제 링크가 그려지게 해야 한다(`uploadPackages` 는 `batchSiblings` 가 비면 아무것도 그리지 않는다, `:147`).
- 검증 명령 (web/ 안에서, 루트 `npm test` 금지):
  - `cd web && npm ci`  (의존성 없으면 선행)
  - `cd web && npm test -- --run`  (전체 149건/13파일이 기준선. 약 90초)
  - `cd web && npx tsc -b --noEmit`  (npm test 는 타입 검사를 하지 않는다)
  - `cd web && npm run build`
  - 백엔드 0 파일 변경이면 Go 테스트는 돌리지 않아도 된다. 돌린다면 `cd backend && go test ./... -count=1` 이며 `TEST_POSTGRES_DSN` 없이는 DB 테스트가 SKIP 되므로 통과로 보고하지 말 것.
- 위험과 피할 것:
  - `key={id}` 재마운트는 **의도한 동작 변화**를 하나 만든다: 이전에는 `run.loading && !run.data` 가드(`:302`) 때문에 이동 중 A 의 상세가 계속 보였는데, 이제 `PageLoading`("실행 기록을 불러오는 중입니다")이 잠깐 보인다. 이것을 깨진 테스트로 오해하지 말고, 기존 테스트가 이동 직후 바로 B 의 내용을 찾는다면 `await waitFor`/`findBy*` 로 바꿀 것. 반대로 "이동 중에도 A 의 상세가 보여야 한다" 는 테스트가 있다면 그 자리를 먼저 보고 과제를 재검토할 것(정찰에서 미확인).
  - 스트림 effect(`:237-277`)의 의존성 배열과 `end`/`error` 프레임 처리, `streamAttempt` 배선은 **한 줄도 바꾸지 말 것**. 매 렌더 새 객체를 의존성에 넣으면 재연결 폭주로 사용자당 3스트림 한도를 때린다(코드 주석 `:229-235` 가 그 사고를 기록해 두었다).
  - `appendStreamedLine`·`collectStoredLogs`·`nextLogCursor`·`streamDisconnected`·`streamEndedRun` 같은 export 된 순수 헬퍼의 시그니처를 바꾸지 말 것 — `SimpleRunDetailPage.test.ts` 와 전체 모드(`ReleaseDetailPage` 가 `streamDisconnected` 를 import)가 쓴다.
  - `SimpleDeployPage`·`ReleaseDetailPage`·백엔드·`useAsync.ts` 는 건드리지 말 것. `useAsync` 를 "dep 바뀌면 data 초기화" 로 고치는 것은 전역 영향이고 전체 모드 릴리즈 로그 테스트가 그 유지 동작에 걸려 있다(v0.5.24 기록). 보호 경로(auth/migrations/workflows)는 접근하지 않는다.
  - 소스 문자열 검사나 직접 만든 props 주입으로 완료를 주장하지 말 것 — 실제 App 라우트를 렌더하고 API/EventSource 전송 경계만 대체하는 이 저장소의 기존 테스트 방식을 따를 것.
  - MUI 의 `disabled` 는 링크의 `href` 를 지우지 않는다. 링크 활성 여부를 봐야 하면 `aria-disabled`/`tabindex` 도 확인할 것(v0.5.31 기록).
- 차선 후보: `docs/architecture.md:72` 의 "Harbor 미사용" 단정을 README·`docs/simple-mode.md` 후반의 선택적 Harbor 복제 서술과 맞추기 (가치 2 / 위험 1 / S) — `executeSimpleRun`/`runReplication` 의 활성 조건을 먼저 확인하고 문서만 고친다. 그다음은 `RELEASE_LOG_TRUNCATED_NOTICE`(ReleaseDetailPage.tsx:67) 복사 안내에 "로그 내려받기" 를 언급하기 (가치 1 / 위험 1 / S).
