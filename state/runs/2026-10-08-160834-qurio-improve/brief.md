- 과제: 웹 SSE 종료·예외 경로의 reader 취소와 잠금 해제 보장 (가치 4 / 위험 2 / 작업량 S)
- 왜: `web/src/lib/api.ts:309`의 `parseServerSentEvents`는 정상 EOF·읽기 실패·콜백 예외 어느 경로에서도 reader 잠금을 해제하지 않으며, `parseStream`이 서버 error 이벤트를 예외로 바꾸면 열린 원본 스트림도 취소되지 않는다. 소비가 중단된 스트림을 정리하면 불필요한 응답 소비를 종료하고 호출자에게 기존 오류를 그대로 전달할 수 있다.
- 수용 기준:
  1) 실제 ReadableStream을 닫지 않은 채 SSE error 이벤트를 보내고 공개 `apiClient.streamChat`을 호출하면 기존 ApiError(status 502·message·details)가 유지되고 underlying source의 cancel이 정확히 한 번 호출되며 `body.locked === false`가 된다.
  2) onDelta/onEvent가 sentinel Error를 던져도 같은 Error 객체로 reject되고 열린 스트림은 취소·잠금 해제된다. cancel 자체가 reject해도 취소 오류가 원래 예외를 덮어쓰거나 unhandled rejection이 되지 않는다.
  3) 정상 EOF와 reader.read 실패에서도 잠금이 풀린다. 정상 델타 순서·CRLF 경계·마지막 빈 줄 없는 이벤트 처리는 그대로 유지한다. 이미 errored 상태인 ReadableStream에서 underlying cancel 훅이 호출될 것이라고 가정하지 않는다.
  4) 기존 `delivers the terminal done event before surfacing a streamed agent error` 테스트가 계속 통과한다. streamAgentResponse와 streamLegacyCodeRoom은 error를 기록하고 EOF까지 소비하므로 공통 파서가 error라는 이벤트 이름만 보고 조기 중단해서는 안 된다.
  5) 새 테스트는 공개 apiClient → 실제 파서 배선을 통과한다. fetch 응답 경계만 기존 테스트 관례대로 대체하고 Response/ReadableStream/reader는 진짜 객체를 쓴다. 정리 코드를 되돌리면 새 cancel/locked 단정이 실패해야 한다.
- 건드릴 파일:
  - `web/src/lib/api.ts:parseServerSentEvents`(309–376) — 읽기 루프와 마지막 dispatch를 try/catch/finally 수명 범위에 넣어 조기 예외 시 reader.cancel을 best-effort 처리하고 모든 경로에서 releaseLock을 보장한다. 원래 예외를 다시 던지며 정상 EOF에는 불필요한 취소를 하지 않는 형태를 권장한다.
  - `web/src/lib/api.test.ts:describe('apiClient')` — 정상 EOF·서버 error·콜백 예외·read 실패·cancel 실패 회귀 사례. 열린 스트림을 사용하는 테스트는 실패 시에도 자체 정리하되 assertion 전에 정리하여 결함을 가리지 않는다.
- 검증 명령:
  - 저장소 루트: `npm test --prefix web -- --run src/lib/api.test.ts`
  - `npm test --prefix web -- --run`
  - `npm run typecheck --prefix web`
  - 정찰에서 실제 실행한 재현: `node --experimental-transform-types /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-160834-qurio-improve/sse-probe.mjs` → base cbbf578에서 exit 1; 수정 후 exit 0이어야 한다. 프로브가 현재 작업 트리의 api.ts를 직접 import하므로 다른 체크아웃에서는 import 절대경로를 맞춘다.
- 위험과 피할 것: 프로덕션 1파일+테스트 1파일만. 이벤트 문법·JSON 디코딩·EOF 의미·agent error→done 순서, 인증/CSRF, AbortSignal 전달, Go 서버, migrations, workflows, 패키지 버전·락파일은 범위 밖이다. 파서 통합·라이브러리 추가·retry/timeout 확대를 하지 않는다. 정리 오류로 원래 예외를 가리지 않으며 감사·로그에 SSE 원문/details를 새로 출력하지 않는다.
- 차선 후보: SSE UTF-8 멀티바이트 네트워크 분할 회귀 테스트 (가치 3 / 위험 1 / 작업량 S) — 착수 base에서 1순위가 이미 해결돼 있을 때만 api.test.ts 한 파일에 추가. 기존 한국어 테스트는 문자열을 통째로 encode하므로 바이트 중간 분할을 검증하지 않는다. 한 번 인코딩한 Uint8Array를 한글 코드포인트 내부에서 나눠 보내 최종 onDelta 값이 원문과 같은지 확인한다.

근거와 확인 범위

- 기준 main cbbf578, v1.4.14. git status --short 무출력. 저장소 코드는 수정하지 않았다.
- 실제 읽은 함수: parseServerSentEvents(309), parseStream(396), streamAgentResponse(422), apiClient.streamChat(478), streamLegacyCodeRoom(505). 기존 테스트 5–47, 82–98, 134–155와 web/src/test/setup.ts도 읽었다.
- Node v22.23.1의 실제 Response/ReadableStream으로 공개 streamChat을 실행했다. normal: cancel=0/locked=true/델타 ['안녕']; server-error: cancel=0/locked=true/'provider stopped'; callback-error와 read-error: cancel=0/locked=true/원래 Error 객체 유지. 새 기대 조건을 검사하는 프로브는 exit 1.
- 실제 브라우저 네트워크 연결의 잔존 시간·서버 CPU/메모리 영향은 미확인. 증명한 것은 잠금 잔존과 열린 source의 취소 누락이며, 절감량은 주장하지 않는다.
- web/node_modules가 없어 정찰의 저장소 밖 쓰기 제약상 npm ci·Vitest·typecheck를 실행하지 않았다. 위 npm 명령은 package.json/vite.config.ts에서 실제 스크립트와 대상 경로를 확인했다. 구현 단계에서는 pretest가 npm ci를 실행한다. Go·DB·E2E도 정찰 미실행.

구현 순서와 확인 지점 (전부 미착수; 사람 승인 지점 없음)

1. api.test.ts에 공개 API 회귀 테스트를 작성한다. proof: 표적 npm 명령으로 기존 사례는 녹색이고 새 cancel/locked 단정이 실패하는지 확인. 실패 원인이 다르면 계획을 갱신하고 범위를 넓히지 않는다.
2. api.ts:parseServerSentEvents의 reader 수명만 수정한다. proof: 같은 표적 명령 녹색, Error identity·cancel reject 사례 통과. 이 확인 뒤 다음 단계로 간다.
3. 전체 웹 스위트와 typecheck를 실행한다. proof: 위 npm 명령 두 개 및 diff가 두 파일에만 있는지 확인. agent error→done 테스트가 깨지면 변경을 좁힌다. 임시 프로브는 커밋하지 않는다.

대안 비교와 선택 가정

- 선택한 공통 파서 수명 관리: 프로덕션 함수 하나로 확인된 실패 경로를 닫는다.
- 호출자마다 AbortController 추가: 여러 호출 경로와 외부 signal 소유권까지 바뀌므로 이번 규모에 맞지 않는다.
- 테스트만 추가하는 차선: 위험은 더 낮으나 현재 확인된 취소 누락을 남기므로 후순위. 현상 유지도 가능하지만 오류 종료의 정리 계약은 계속 보장되지 않는다.
- 가장 중요한 가정: 공개 호출자가 예외 이후 Response body를 다른 소비자에게 넘기지 않는다. 현재 읽은 chat/agent/legacy 호출부는 response를 로컬로만 갖는다. 재사용 경로를 찾으면 취소 정책부터 재평가한다.

작업량과 예비 시간 (정찰자의 추정이며 확약 아님)

- Bottom-up: 회귀 테스트 8–12분 + 수명 관리 5–8분 + 검증/리뷰 8–12분 = 기본 21–32분. 첫 npm ci·cancel reject 사례의 알려진 불확실성에 contingency 5–8분을 별도로 잡아 총 26–40분, 주관적 확신 중간.
- 유사 사례 교차 확인: 최근 웹 회차와 같은 총 2파일 규모지만 이번은 프로덕션 수명 관리라 테스트 조합이 더 많다. 과거의 실제 분 단위 소요 기록은 없어 독립적인 수치 유추는 미확인이다.
- Management reserve 0분: 새 기능·다른 파서 문제를 45분에 흡수하지 않고 ideas에 남긴다. 표적 테스트 이후 45분 초과가 예상되면 범위를 재평가한다.
- 적용 스킬: /home/hkjang/.claude/plugins/marketplaces/headcount/plugins 아래 pmo/skills/estimating-and-contingency/SKILL.md, technology/skills/implementation-planning/SKILL.md, technology/skills/solution-exploration/SKILL.md. 전용 Skill 도구가 없어 파일로 읽었다. pmo references/sources.md도 확인했으며 외부 원가 산식·통계적 신뢰구간은 사용하지 않았다.
