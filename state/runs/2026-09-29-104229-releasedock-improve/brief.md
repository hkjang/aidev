- 과제: 전체 모드 실시간 로그가 서버의 30분 제한 뒤 마지막 커서에서 재연결되게 하기 (가치 3 / 위험 2 / 작업량 M)
- 왜: `web/src/pages/releases/ReleaseDetailPage.tsx:useReleaseLogs`(55~90행)는 모든 `end` 이벤트에서 EventSource를 닫고 끝내므로, 서버가 30분 상한으로 연결을 끝내면 진행 중 릴리즈의 로그도 멈춘다. 서버가 이미 제공하는 `reason=max_duration`과 `after` 계약을 사용하면 화면을 새로 열지 않고 다음 로그를 이어 볼 수 있다.
- 수용 기준:
  1) 실제 `/releases/release-1` 화면의 “실시간 로그” 탭을 열어 서버 형태의 log(id=41, 다음 id=57)를 받은 뒤 `end {"reason":"max_duration"}`를 보내면 기존 source가 닫히고 하나만 새로 열린다. 새 URL은 `?after=57`이고 `withCredentials=true`이며, 기존 로그를 보존하고 후속 id=58을 표시한다. 화면용 sequence(2)를 커서로 사용하면 실패해야 한다.
  2) 서버의 정상 완료 프레임은 `end {}`이다. 이 프레임에서는 연결을 닫고 재연결하지 않는다. 파싱 불가·알 수 없는 이유 역시 자동 재연결하지 않고, 명시적인 `reason === 'max_duration'`만 재연결한다. `{}`를 비정상 종료로 분류하는 단순 모드 헬퍼를 가져오지 않는다.
  3) 로그가 여러 줄 들어오거나 재렌더되어도 같은 source를 유지한다. 로그가 없을 때의 시간 제한은 after=0으로 연결하고, 반복 시간 제한은 최신 커서로 계속 이어간다. 일반 error 이벤트만으로 새 source를 만드는 자동 루프는 없다.
  4) “지우기”는 표시 목록만 지우고 커서를 되돌리지 않는다. 탭 이탈/언마운트 시 활성 source를 닫고 추가 연결을 만들지 않는다. 다른 릴리즈로 이동하면 이전 릴리즈 커서를 재사용하지 않는다(탭 재진입 시 과거 로그를 다시 읽는 기존 동작은 유지).
  5) 프로덕션 App→ReleaseDetailPage→LogPanel→useReleaseLogs 경로를 렌더한 회귀 테스트로 1~4를 검증한다. end 재연결을 제거하면 시간 제한 테스트가 실패하고, 모든 end에서 재연결하게 바꾸면 `{}` 테스트가 실패하며, cursor 갱신을 제거하면 after=57 검증이 실패해야 한다. 코드 문자열 검사나 복제한 훅 테스트로 대체하지 않는다.
- 건드릴 파일:
  - `web/src/pages/releases/ReleaseDetailPage.tsx:useReleaseLogs` — 서버 로그 id용 ref와 시간 제한 재연결 트리거를 추가하고 기존 effect의 cleanup으로 한 연결을 유지한다. `api.releaseLogStreamUrl(releaseId)` 뒤에 `?after=`를 붙이면 API 공통 파일을 변경할 필요가 없다. 기존 raw JSON/data envelope 및 plain-text 표시 호환성, 화면용 sequence와 4999행 상한은 유지한다. 커서는 MessageEvent.lastEventId 또는 실제 payload id에서 읽되 유효하지 않은 값으로 후퇴시키지 않는다.
  - `web/src/pages/releases/ReleaseDetailPage.stream.test.tsx`(신규) — `NewReleasePage.test.tsx:renderPage`의 실제 App/ThemeProvider/로그인 배선을 참고하고, `api.release`는 실제 `Release` 타입 값(예: DEPLOYING)으로 응답시킨다. 탭 role/name “실시간 로그”를 클릭해야 LogPanel이 마운트된다. 네트워크와 브라우저 EventSource 경계만 대역으로 둔다. `SimpleDeployPage.stream.test.tsx:TestEventSource`를 참고하되 전체 모드의 onopen/onmessage/onerror 프로퍼티 콜백도 실제로 호출되도록 만들거나 프로덕션 리스너를 동등한 addEventListener로 옮긴다. URL 빌더와 훅 자체는 모킹하지 않는다.
  - 프로덕션 1개 + 테스트 1개. backend/auth/migrations/workflows 변경 없음.
- 검증 명령:
  - 준비: `cd web && npm ci` (현재 node_modules 없음; 구현 단계에서 설치 필요).
  - 집중: `cd web && npm test -- --run src/pages/releases/ReleaseDetailPage.stream.test.tsx`
  - 회귀: `cd web && npm test -- --run`
  - 타입: `cd web && npx tsc -b --noEmit`
  - 빌드: `cd web && npm run build` (생성 dist/tsbuildinfo는 커밋 금지).
  - 정찰 실행 결과: `cd web && npm test -- --run`은 `vitest: not found`로 종료 127. `cd backend && go test ./internal/server -run 'Test.*Stream.*Cursor|TestSimpleRunLogStream' -count=1 -v`는 종료 0이나 6건 모두 TEST_POSTGRES_DSN 미설정으로 SKIP. 동작 재현·전체 테스트 통과를 주장하지 않는다. 위 웹 명령은 package.json/CI에서 확인했지만 설치 후 성공 여부는 미확인이다.
- 위험과 피할 것: `backend/internal/server/releases.go:streamReleaseLogs`에서 정상 종료는 `{}`, 시간 제한만 `{reason:"max_duration"}`인 것을 직접 확인했다(1557행 부근). 단순 모드 `streamEndedRun`은 status를 요구하므로 계약이 다르며 통합 금지. effect 의존성에 logs/매번 새 객체를 넣어 줄마다 재연결하면 과거 429 문제가 재발한다. 429/CLOSED 수동 복구 UI, onerror 표시 개선, rAF 배치, 저장 로그 API, 서버 상한 변경은 이번 범위 밖이다. VERSION·release 설정도 건드리지 않는다. 30분 실제 대기는 불필요하며 서버가 실제 보내는 두 프레임을 브라우저 경계에 전달해 분기를 검증한다. 실제 브라우저/DB 연결에서 이 결함을 재현한 것은 아니며 현재 근거는 양쪽 소스 계약이다.
- 차선 후보: 로그 스트림 사용자당 3개 제한의 429 거절 및 해제 후 재접속 HTTP 통합 테스트 — `simple_stream_test.go:newSimpleStreamFixture`와 실제 `s.Handler()`를 사용한다. 실제 PostgreSQL과 httptest.Server를 쓰고 활성 연결 확인/닫기 후 핸들러 반환을 동기화한다. 전역 64개까지 같은 회차에 확대하지 않는다. 1순위가 이미 해결돼 있을 때만 택한다.

구현 순서·예산: 렌더 픽스처/실패 재현 10분 → cursor/end 분기 15분 → 회귀·타입·빌드 10분 → cleanup/반증 확인 여유 10분, 총 45분(M). 20분 안에 렌더 배선이 안 잡히면 공통 훅 추출로 범위를 키우지 말고 현재 실패와 제약을 보고한다.
정찰 제약: 요청한 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`은 사용 가능한 Skill 도구 및 로컬 검색에서 발견하지 못했다. 해당 스킬의 원문·반환 형식은 미확인이고 적용하지 못했으며, 사용자 지정 형식과 시간/위험/차선 계획을 따랐다.
