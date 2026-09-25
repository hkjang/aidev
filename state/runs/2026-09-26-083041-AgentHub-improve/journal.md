# 회차 노트 2026-09-26-083041-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:30] base pinned — main@fd58af7
- [러너 08:30] autonomy release — 

## 정찰 노트
- guide-shots 의 요청별 제한 시간을 골랐다: `call` 의 fetch 에 signal 이 없고 `settle`(324행)의 180초 deadline 이 대기 중인 단일 GET 을 끊지 못하는 것을 소스에서 직접 확인했으며, 실패가 기존 백업/복원 경로로 그대로 흘러가 guide-settings-check.mjs 를 건드릴 필요가 없다. CI 포함(보호 경로인 워크플로를 건드림)과 tracking 위반 전체 삭제(DISPOSABLE 전제라 피해 제한)는 그래서 제쳤다.
- 추측으로 적은 것: fresh vm 컨텍스트에 `AbortSignal` 이 없다는 것(하니스가 structuredClone·fetch 를 명시 주입하는 패턴에서 추론), 제한 시간 기본값 30000ms 의 적정성, 브라우저 abort 가 Playwright 를 통해 나오는 오류 문자열의 정확한 모양.
- **이번 세션은 Bash 명령 승인이 거부되어 테스트를 한 번도 돌리지 못했다.** 기준선 64건 통과는 이전 회차 기록이지 이번 확인이 아니다 — 구현자는 수정 전에 `cd web && node --test scripts/guide-settings-check.test.mjs` 로 기준선부터 잡을 것.
- 조심할 것: 테스트가 실제 소스를 `'  const call ='` 앵커부터 잘라 쓰므로 그 위에 상수를 선언하면 ReferenceError 가 난다. 멈춘 요청 스텁은 pending Promise 가 아니라 signal 의 abort 로 reject 해야 한다(아니면 테스트 자체가 멈춘다).
- [러너 08:34] scout done — guide-shots 의 모든 API 호출에 요청별 제한 시간을 두어 멈춘 컨트롤 플레인이 촬영을 영원히 붙잡지 못하게

## 구현 노트
- `call` 의 `page.evaluate` fetch 에 `AbortSignal.timeout` 을 인자 배열로 실어 보내고(기본 30000ms, `AGENTHUB_GUIDE_REQUEST_TIMEOUT_MS`), `settle` 의 폴링 GET 을 try/catch 로 감쌌다. `guide-settings-check.mjs` 는 건드리지 않았다 — 실패가 기존 백업/복원 경로로 그대로 흘러간다. 커밋 611a18a.
- 과제서의 권고 하나를 의도적으로 바꿨다: Playwright 가 되던진 오류의 **문자열**을 밖에서 읽는 대신, 브라우저 안에서 `error.name === 'TimeoutError' || 'AbortError'` 를 보고 `{ timedOut: true }` 마커를 돌려준다. Playwright 의 오류 직렬화 모양에 기대지 않는 쪽이 안전하다고 봤다. 대가로 `{ cause }` 원인 객체는 보존하지 않는다(메시지에 메서드·경로·ms 는 들어간다).
- **확신 없는 곳**: (1) 실물 Chromium 에서 `AbortSignal.timeout` 이 실제로 이 경로를 태우는 것은 확인 못 했다 — 이 환경에 클러스터도 Playwright chromium 도 없어 vm 하니스로만 증명했다. 브라우저의 abort 가 `AbortError`(구현에 따라 `TimeoutError`)로 나온다는 전제에 기대며, 둘 다 받도록 넓혀 뒀다. (2) 기본값 30초가 느린 배포에서 충분한지는 판단이지 측정이 아니다. (3) `response.text()` 도 try 안으로 넣어 본문 읽는 중 abort 도 같은 경로로 가게 했는데, 이 분기는 하니스가 재현하지 않는다.
- 하니스에서 실제로 걸린 것 — 정찰이 미확인으로 남긴 부분: vm 컨텍스트에 `AbortSignal` 을 넣어야 하는 것은 맞았고, **추가로** Node 의 `AbortSignal.timeout` 타이머가 unref 라 다른 일이 없으면 이벤트 루프가 먼저 빠져나간다. 멈춘 요청 스텁이 `setInterval` 로 루프를 붙잡아 준다(테스트 전용 장치이며 프로덕션과 무관 — 브라우저에는 해당 없음).
- 일부러 안 한 것: `page.goto`·`shoot`·`settle` 의 대기에는 손대지 않았다(과제서의 경고대로 `call` 과 성격이 다르다). 차선 후보였던 problems 요약 출력 수정도 범위 밖으로 뒀다 — 다만 그 수정은 `if (problems.length)` 줄을 옮기게 되고, 그 줄이 테스트 슬라이스의 **끝 앵커**라 함께 고쳐야 한다(ideas.json 에 적어 뒀다).
- 다음 역할이 조심할 것: 이 테스트는 DB 도 브라우저도 필요 없다(`cd web && node --test scripts/guide-settings-check.test.mjs`, 73건). 슬라이스 시작 앵커를 `'  const call ='` 에서 `'  const requestTimeoutMs ='` 로 옮겼으므로 그 상수를 지우거나 이름을 바꾸면 슬라이스가 깨진다. Go·마이그레이션·워크플로·런타임 base 이미지 소스는 손대지 않아 BASE_VERSION 상향은 불필요하다.
- [러너 08:41] brief accepted — 채택 — `call` 의 signal 부재와 `settle` 의 바깥 deadline 이 대기 중인 단일 GET 을 끊지 못한다는 근거가 현재 코드와 정확히 �
