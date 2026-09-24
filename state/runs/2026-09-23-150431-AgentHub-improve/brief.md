# 과제서 — 2026-09-23 AgentHub

- 과제: guide-shots 복원 루프가 PUT 한 건의 예외로 중단되지 않고 네 설정을 모두 되돌린다 (가치 3 / 위험 1 / 작업량 M)

- 왜: `web/scripts/guide-settings-check.mjs:43-47` 의 `finally` 는 네 전역 설정을 순서대로 `await call('PUT', …)` 하는데, 이 호출은 브라우저 안 `fetch`(guide-shots.mjs:85-96 의 `call` → `page.evaluate`)라서 컨트롤 플레인 재시작·페이지 종료 같은 흔한 사유로 **거부(reject)** 될 수 있다. 지금은 첫 PUT 이 던지면 루프가 그 자리에서 끝나 뒤의 정책·내용검사·게이트웨이·추적 중 남은 것들이 seed 가 덮어쓴 데모 값(정책 3규칙·DLP block·`agents.example.internal` 게이트웨이·추적 설정) 그대로 남고, 게다가 finally 의 예외가 본문(seed/capture/captureTracking)이 던진 원래 오류를 덮어써 운영자가 무엇이 처음 실패했는지 못 본다. 지난 회차(af47666)가 "백업이 완전할 때만 쓰기 시작한다" 를 보장했으므로, 남은 구멍은 나가는 길 한 곳뿐이다.

- 수용 기준:
  1) 복원 PUT 네 건 중 하나가 **reject** 해도 나머지 세 건이 모두 시도되고 실제로 요청이 나간다(요청 기록으로 확인).
  2) 개별 복원 실패(예외든 non-2xx 든)는 `note(\`복원 ${path}\`, false, …)` 로 남아 guide-shots 의 `problems` 에 들어가 `process.exitCode = 1` 이 된다 — 조용히 삼키지 않는다.
  3) 본문이 던진 오류(seed/capture/captureTracking 실패)는 복원 실패가 있어도 그대로 밖으로 전파된다. 본문이 성공했는데 복원만 실패한 경우에만 복원 실패가 오류로 전파된다(어느 경로가 실패했는지 메시지에 포함).
  4) 테스트가 증명할 것: (a) 첫 PUT 이 던질 때 PUT 4건이 모두 기록되고 복원 실패 note 가 남는다 (b) 본문 오류 + 복원 오류가 겹칠 때 밖으로 나오는 것은 본문 오류다 (c) 정상 경로의 PUT 순서·본문(`{value: …}` 래핑 포함)은 기존 테스트대로 하나도 바뀌지 않는다.

- 건드릴 파일:
  - `web/scripts/guide-settings-check.mjs:withGuideSettings` — `finally` 안 복원 루프에만 손댄다. 각 `call('PUT', …)` 을 개별 `try/catch` 로 감싸 예외를 note 로 바꾸고 계속 진행, 실패 목록을 모아 루프 뒤에서 처리. 백업(사전 검증) 블록 12-34 행은 **그대로 둔다**.
  - `web/scripts/guide-settings-check.test.mjs:run` — 현재 하니스의 `fetch` 는 28행에서 non-GET 을 무조건 `{status:200}` 으로 먼저 돌려보내 PUT 실패를 흉내낼 수 없다. `failMethod`(또는 동등한) 옵션을 더해 지정한 method/path 의 PUT 이 throw 하거나 503 을 주게 하고, 위 (a)(b) 시나리오를 추가. 기존 `originals()`·`blocked()`·정상 복원 테스트는 유지.
  - (선택) `web/scripts/guide-shots.mjs` 상단 주석 19행 "네 전역 설정을 있던 대로 되돌린다" 문장에 한 건이 실패해도 나머지를 되돌린다는 사실을 한 문장으로 반영. 코드 동작은 건드리지 말 것.

- 검증 명령:
  - `cd web && node --test scripts/guide-settings-check.test.mjs` (신규 포함 전부 통과)
  - `cd web && node --test scripts/guide-settings-check.test.mjs scripts/session-gateway-check.test.mjs scripts/runtime-settings-check.test.mjs scripts/silent-sso.test.mjs` (인접 회귀)
  - `cd web && node --check scripts/guide-shots.mjs`
  - `go test ./internal/api ./cmd/runtime-proxy` (변경 없음 확인용, 빠름)
  - 수정 **전** 코드에 새 테스트를 대고 (a)(b) 가 실제로 실패하는 것을 먼저 확인하고 그 사실을 회차 노트에 적을 것. 이 저장소의 지난 네 회차가 모두 그렇게 했고 심사도 그것을 본다.

- 위험과 피할 것:
  - **오류를 삼키지 말 것.** 전부 note 로만 바꾸고 아무것도 던지지 않으면 "복원이 통째로 실패했는데 스크립트는 성공" 이 되어 운영자 규칙(효과 없는/숨기는 변경 금지)에 걸린다. 본문 오류 우선, 본문 성공 시엔 복원 실패를 던진다.
  - 본문 오류를 `AggregateError` 나 래핑으로 감싸 메시지를 바꾸면 기존 테스트 78-92행의 `assert.equal(result.error?.message, \`${callbackError} failed\`)` 가 깨진다. 본문 오류는 **원본 그대로** 다시 던질 것.
  - 사전 백업 계약(네 값 모두 검증 뒤에만 쓰기 시작)을 손대지 말 것 — 51건이 여기에 걸려 있고 af47666 의 채택 근거다.
  - 복원 순서(policy → dlp → sessionGateway → tracking)와 `/api/v1/admin/settings/{key}` 의 `{value: …}` 래핑을 바꾸지 말 것. GET 은 policy=`document`, dlp=`settings`, settings=키별 최상위라는 비대칭이 실제 계약이다(프로필 "검증 함정" 참조).
  - 보호 경로 회피: `internal/api/auth.go`·`mcpoauth.go`·`internal/store/migrations`·`.github/workflows` 는 이번 과제에서 열 이유가 없다. `package.json` 스크립트 목록도 건드리지 말 것(별도 보류 항목).
  - 실제 브라우저 촬영·클러스터·DB live 는 이 환경에서 미확인이다. 증명은 실제 스크립트 소스 블록을 읽어 실행하는 기존 vm 하니스 + 실제 `withGuideSettings` 모듈로 할 것. 손으로 만든 별도 대역 모듈을 새로 만들어 그것만 검사하는 방식은 이 저장소에서 반려 사유였다.

- 차선 후보: guide-shots 의 API 호출에 요청별 제한 시간(AbortSignal)을 두어 응답 없는 GET 하나가 복원 진입을 무한정 막지 않게 한다 (가치 3 / 위험 2 / 작업량 M) — `guide-shots.mjs:85-96` 의 `page.evaluate` fetch 에 기한이 없고 `settle` 의 바깥 deadline 도 대기 중인 단일 GET 을 끊지 못한다. 1순위와 같은 파일군이라 함께 고르지 말 것.
