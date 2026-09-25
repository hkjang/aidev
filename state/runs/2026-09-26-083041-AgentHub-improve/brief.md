- 과제: guide-shots 의 모든 API 호출에 요청별 제한 시간을 두어 멈춘 컨트롤 플레인이 촬영을 영원히 붙잡지 못하게 한다 (가치 3 / 위험 2 / 작업량 M)
- 왜: `web/scripts/guide-shots.mjs` 의 `call` 은 `page.evaluate` 안에서 `fetch(path, { method, credentials, headers, body })` 를 기한 없이 기다린다 — `signal` 이 없고 Playwright 의 `page.evaluate` 에도 기본 제한 시간이 없다. 컨트롤 플레인이 연결은 받되 응답하지 않으면(재시작 중, 잠긴 DB) 스크립트가 그 한 요청에서 무기한 멈추고, 이미 `withGuideSettings` 본문에 들어간 뒤라면 finally 의 복원에 도달하지 못해 그 배포의 정책·DLP·세션 게이트웨이·방문 추적이 데모 값인 채로 남는다. 제한 시간을 두면 같은 상황이 "오류 → 복원 → 비정상 종료" 로 끝난다.
- 왜(두 번째 자리): `settle`(scripts/guide-shots.mjs:324)은 180초 바깥 deadline 을 갖지만 그 안의 `await get('/api/v1/tasks')` 한 건이 멈추면 `Date.now() < deadline` 검사에 영영 돌아오지 않는다. 바깥 deadline 은 대기 중인 단일 요청을 끊지 못한다 — 이번 수정이 이 구멍도 같이 막는다.

- 수용 기준:
  1) `call` 이 보낸 요청이 제한 시간 안에 응답하지 않으면 `call` 이 거부(reject)한다. 오류 메시지에 메서드·경로와 시간 초과라는 사실이 들어가 어느 호출이 끊겼는지 로그만 보고 안다.
  2) 시간 초과가 기존 경로로 그대로 흘러간다 — 백업 GET 에서 나면 `withGuideSettings` 가 쓰기 없이 `... backup failed: …` 로 막고, 본문(seed/capture/captureTracking)에서 나면 네 설정 복원이 모두 실행된 뒤 원본 오류가 재전파되며, 복원 PUT 에서 나면 `note('복원 …', false, …)` 로 기록되고 다음 설정 복원이 계속된다. 즉 `guide-settings-check.mjs` 는 한 줄도 바꾸지 않는다.
  3) `settle` 의 폴링 한 건이 시간 초과로 거부해도 seed 전체가 중단되지 않고, 자신의 180초 deadline 까지 폴링을 계속한 뒤 기존과 같은 `{ done: false, detail: … }` 를 돌려준다(워커 미부착 경로의 동작을 바꾸지 않기 위함).
  4) 테스트가 (a) 응답하지 않는 GET 이 백업 단계를 막고 쓰기가 0건임을, (b) 응답하지 않는 본문 요청 뒤에도 PUT 4건이 모두 나감을, (c) 정상 응답에는 시간 초과가 개입하지 않아 기존 64건이 그대로 통과함을 증명한다.

- 건드릴 파일:
  - `web/scripts/guide-shots.mjs`:`call`(88-101행 부근) — `page.evaluate` 의 인자 배열(`[method, path, body ?? null]`)에 제한 시간 ms 를 실어 보내고, 평가 함수 안에서 `fetch(path, { …, signal: AbortSignal.timeout(ms) })` 를 쓴다. **`page.evaluate` 는 함수를 직렬화해 브라우저에서 실행하므로 바깥 변수를 클로저로 참조할 수 없다 — ms 는 반드시 인자 배열로 넘어가야 한다.** 브라우저 안의 abort 는 `DOMException`(TimeoutError, message 가 "signal timed out" 류)으로 나오고 Playwright 가 이를 Error 로 다시 던지므로, `call` 쪽에서 try/catch 로 받아 `${method} ${path} 가 ${ms}ms 안에 응답하지 않음` 같은 메시지로 감싸 던지는 편이 로그에서 읽힌다(원인은 `{ cause }` 로 보존).
  - `web/scripts/guide-shots.mjs` — 제한 시간 상수. 기본값은 30000ms 를 권한다(이 스크립트의 `call` 은 전부 관리자 REST 호출이고 가장 느린 것도 `/api/v1/tasks` 목록이다. 촬영의 긴 대기는 `page.waitForTimeout`·`settle` 쪽이지 `call` 이 아니다). 테스트가 짧은 값으로 돌 수 있게 `process.env.AGENTHUB_GUIDE_REQUEST_TIMEOUT_MS` 로 덮어쓸 수 있게 할 것 — 아래 테스트 하니스가 `process.env` 를 주입한다.
  - `web/scripts/guide-shots.mjs`:`settle`(324-337행) — `const tasks = (await get('/api/v1/tasks')).body?.items ?? []` 를 try/catch 로 감싸 거부 시 `statuses` 를 유지한 채 2초 뒤 다시 폴링하게 한다. deadline·반환 형태·`busy` 집합은 그대로 둘 것.
  - `web/scripts/guide-settings-check.test.mjs` — 시간 초과 시나리오 추가.

- 테스트 하니스에서 반드시 볼 것 (이 파일을 먼저 읽고 시작하라):
  - 이 테스트는 실제 소스를 잘라 실행한다: `const boundary = source.slice(source.indexOf('  const call ='), source.indexOf('  if (problems.length)'))`. **제한 시간 상수를 `const call =` 줄 위에 선언하면 잘린 조각 밖으로 나가 `ReferenceError` 가 된다.** 상수를 `call` 안이나 인자 배열 안에 두든지, 아니면 테스트의 슬라이스 시작 앵커를 새 상수 선언 줄로 바꾸든지 둘 중 하나를 의식적으로 고를 것(후자를 권한다 — 스크립트가 더 읽힌다).
  - vm 컨텍스트는 전역을 명시적으로만 준다(`...helper, structuredClone, process, document, page, fetch, note, seed, capture, captureTracking`). `AbortSignal`(및 `setTimeout` 을 쓴다면 그것도)을 `context` 에 더하지 않으면 잘린 조각이 `ReferenceError` 로 죽는다. **미확인**: 이번 세션에서 node 실행이 승인되지 않아 fresh vm 컨텍스트에 `AbortSignal` 이 없다는 것을 직접 확인하지 못했다. 구현자는 테스트를 한 번 돌려 실제로 필요한 전역만 추가하라.
  - `page: { evaluate: (fn, args) => fn(args) }` 이므로 평가 함수는 Node 안에서 그대로 돈다. 따라서 `fetch` 스텁이 `options.signal` 을 실제 fetch 처럼 존중해야 시간 초과가 재현된다 — 멈춘 요청은 `new Promise((_, reject) => options.signal.addEventListener('abort', () => reject(options.signal.reason)))` 로 흉내 낼 것(영원히 pending 인 Promise 를 그냥 돌려주면 테스트 자체가 멈춘다).
  - `run()` 의 기존 옵션(`failPath`/`failAt`/`failure`/`failWrite`/`failWriteAt`/`writeFailure`)과 같은 규칙으로 `hangPath`/`hangAt` 같은 옵션을 더하고, `process: { env: { … } }` 에 짧은 `AGENTHUB_GUIDE_REQUEST_TIMEOUT_MS`(예: 50) 를 실어 테스트가 빨리 끝나게 할 것.
  - 기존 도우미 `blocked(result, reason)` 이 "쓰기 0건 + 콜백 0건" 을 이미 검사하므로 백업 시간 초과 테스트는 그대로 재사용된다.

- 검증 명령 (저장소 루트 기준):
  - `cd web && node --test scripts/guide-settings-check.test.mjs` — 이번 수정의 주 증거. 이전 회차 기록상 64건이 통과하고 있었다(**미확인**: 이번 세션에서는 명령 승인이 거부되어 기준선을 직접 돌려보지 못했다. 구현자는 수정 **전** 에 한 번 돌려 기준선을 확인하고 시작하라).
  - `cd web && node --test scripts/guide-settings-check.test.mjs scripts/session-gateway-check.test.mjs scripts/runtime-settings-check.test.mjs scripts/silent-sso.test.mjs` — 인접 회귀(기록상 121건+).
  - `cd web && node --check scripts/guide-shots.mjs`
  - `go test ./internal/api ./cmd/runtime-proxy` — 변경 범위 밖이지만 값싼 확인.
  - **새 테스트가 수정 전 코드에서 옳은 이유로 실패하는 것을 먼저 확인할 것**(지금은 시간 초과가 없으므로 멈추거나 타임아웃으로 죽는다). 이 저장소의 지난 네 회차가 모두 이 순서를 밟아 채택됐다.

- 위험과 피할 것:
  - `web/scripts/guide-settings-check.mjs` 는 건드리지 말 것. 백업 사전 검증·복원 순서·`{ value: … }` 래핑은 최근 세 회차가 만든 계약이고, `call` 을 주입받는 구조라 이번 수정은 주입되는 함수만 바뀌면 된다.
  - 제한 시간을 짧게 잡지 말 것. `call` 이 아닌 곳(`page.goto`, `shoot` 의 700ms, `settle` 의 180초)의 대기를 제한 시간 대상으로 착각해 값을 낮추면 느린 배포에서 멀쩡한 촬영이 실패한다.
  - Go 코드·`internal/api`·마이그레이션·`.github/workflows`·런타임 base 이미지 소스(`internal/dlp`, `cmd/runtime-proxy`)는 이번 과제에서 손댈 이유가 없다 — 건드리면 BASE_VERSION 상향 문제가 따라온다.
  - 실물 브라우저 촬영은 이 환경에서 돌릴 수 없다(클러스터·Playwright chromium 미확인). 검증은 vm 하니스로 하고, 실행하지 않은 것은 실행하지 않았다고 적을 것.
  - 문자열 검색(grep)으로 "signal 이 들어갔다" 를 증거로 제출하지 말 것 — 실제로 거부되는 것을 테스트로 보여야 한다.

- 차선 후보: **복원 실패로 guide-shots 의 problems 요약이 출력되지 않는 것을 고친다 (가치 2 / 위험 1 / 작업량 S)**. `guide-shots.mjs` 는 `await withGuideSettings(...)` 뒤 `if (problems.length) { … }` 로 실패 목록을 찍는데(113행 부근), 본문이 성공하고 복원이 실패하면 `withGuideSettings` 가 `복원 실패: …` 를 던지므로 그 요약 블록이 건너뛰어지고 운영자는 스택만 본다 — `note(…, false, …)` 로 모은 목록이 정작 출력되지 않는다. 요약 출력을 `finally` 로 옮기거나 `withGuideSettings` 를 try/catch 로 감싸 요약을 찍은 뒤 재전파하면 된다. 종료 코드는 어느 쪽이든 0 이 아니므로 피해는 작다.
