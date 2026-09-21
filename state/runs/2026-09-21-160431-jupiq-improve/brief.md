- 과제: AI SSE 오류 종료 때 응답 스트림을 취소하고 reader 잠금을 해제 (가치 3 / 위험 1 / 작업량 M)
- 왜: `web/src/api/client.ts:streamAI`는 SSE 오류를 만나 reject해도 응답 body를 취소하거나 reader 잠금을 해제하지 않아, 화면이 오류를 표시한 뒤에도 읽지 않는 연결이 남는다. 오류 종료 시 스트림 자원을 정리하면 새 질문을 시작할 때 이전 요청 연결이 남는 일을 막을 수 있다.
- 수용 기준: 1) EOF 전에 `event: error` 또는 JSON `error` 이벤트를 받으면 기존 ApiError(메시지·502)를 유지하면서 실제 ReadableStream의 underlying cancel이 호출되고 body.locked가 false가 된다. 2) 정상 EOF와 reader.read() 실패·AbortError에서도 reader 잠금이 해제되고, 취소 실패가 원래 오류를 덮어쓰지 않는다. 3) 테스트는 프로덕션 `streamAI`를 호출하고 실제 Response·ReadableStream 객체로 취소·잠금 해제를 관찰한다. 손으로 만든 reader 대역이나 소스 문자열 검사는 증거로 삼지 않는다. 기존 OpenAI delta·단순 delta·[DONE]·UTF-8 분할 청크·마지막 버퍼 출력 동작은 유지한다.
- 건드릴 파일: `web/src/api/client.ts:streamAI` — reader를 획득한 뒤 전체 읽기/버퍼 처리 구간에 수명 정리를 추가, 조기 실패 시 best-effort cancel 후 반드시 releaseLock; `web/src/api/client.test.ts` — 기존 SSE 성공 테스트에 더해 실제 열린 스트림의 오류 종료·cancel 실패·read 실패·정상 종료 회귀 검증. 참조만 할 파일: `web/src/pages/AiOpsPage.tsx:OpsChat/ask`(finally에서 controller.current=null), `internal/api/ai_handlers.go:aiChat/copyFlushedStream`(상류 body 중계), `web/vite.config.ts`, `web/src/test/setup.ts`.
- 검증 명령: 최초 의존성 준비 `npm --prefix web ci`; 단계별 `npm --prefix web test -- src/api/client.test.ts`; 최종 `npm --prefix web run lint`와 `npm --prefix web test`. Go 경로를 바꾸지 않으므로 Go 재검증은 필수 아님. 정찰은 `go test ./...`, `node scripts/check-screenshots.mjs`, `./scripts/check-version.sh`의 exit 0을 확인했다. web/node_modules가 없어 프런트 명령 실행 결과는 미확인이며 위 npm ci가 선행되어야 한다.
- 위험과 피할 것: auth·migrations·workflows·Go proxy·API 오류 계약·의존성은 변경하지 않는다. [DONE]을 즉시 종료 신호로 바꾸거나 SSE 파서를 일반화하지 않는다. OpsChat unmount 처리, request/requestList 오류 통합은 별도 후보이며 이번 범위가 아니다. cancel()의 reject가 기존 ApiError/AbortError를 가리지 않도록 하고 releaseLock만으로는 upstream 취소가 되지 않는 점에 유의한다. fetch가 AbortSignal을 받는 기존 배선 유지. 실제 브라우저에서 Go→외부 공급자까지 연결이 닫히는 전체 연쇄와 실제 공급자 비용 영향은 미확인이다.
- 차선 후보: OpenAPI page_size 계약을 현행 pageBounds 상한 200과 일치시키기 — 첫 과제가 이미 해결되었거나 런타임 재현과 다를 때만. `openapi/openapi.yaml` /users maximum 100과 /audit maximum 200, `internal/store/store.go:pageBounds` 200 확인. API 동작을 100으로 축소하지 말고 /users 문서를 현행 동작에 맞추며, `internal/api/scoped_rbac_integration_test.go`의 실제 Handler·DB 경로로 meta.page_size를 증명한다. 사용자 상세 limit(별도 100)은 그대로 둔다.

범위·선택 근거
- 확인 기준: main@5bd5058, VERSION 1.7.2. 이전 두 회차의 검증 타깃/정수 질의 거부는 이미 반영되어 재선택하지 않았다.
- 가장 작은 안(선택): streamAI 내부 cleanup + 실제 스트림 회귀 테스트. 소비자 모두 같은 함수 경로를 지나며 2파일에 한정된다.
- 화면에서만 controller.abort: 현재 OpsChat에는 도움이 되지만 다른 streamAI 호출자의 자원 소유권 문제가 남아 탈락.
- 공용 SSE 계층으로 재설계: 확장성은 있으나 현재 호출자가 하나이고 45분 범위를 넘기므로 탈락.
- 무변경/문서만 보완: 오류가 나면 upstream도 곧 EOF를 보낸다는 가정이 필요하다. 열린 응답으로 재현되어 선택하지 않았다.

재현 증거
- Node v22.23.1에서 실제 client.ts를 메모리에서 stripTypeScriptTypes로 읽어 실행했다. Vite 전용 import.meta.env 값만 대체했고 파서나 streamAI는 수정하지 않았다.
- 실제 ReadableStream이 `event: error\ndata: provider failed\n\n`을 enqueue하고 닫지 않게 했다. 실제 Response를 반환하는 fetch 경계만 바꿨다. 결과: `{"error":"provider failed","status":502,"cancelled":false,"locked":true}`.
- 별도 확인에서는 node:http 로컬 서버가 같은 이벤트를 쓰고 응답을 닫지 않게 했고, 대역 없는 실제 fetch로 streamAI를 호출했다. reject 후 100ms에도 서버 response close=false였다. 재현 종료 시 외부 AbortController와 server.closeAllConnections로 정리했다. 브라우저/실제 AI 공급자는 사용하지 않았다.

실행 순서와 체크포인트 (전부 pending; 사람 승인 대기 없음)
1. client.test.ts에 열린 실제 stream의 error event와 JSON error 회귀 테스트 추가. `npm --prefix web test -- src/api/client.test.ts`에서 현재 취소/잠금 단언이 실패하는지 확인. 이 단계는 의도한 red 체크포인트이며 그 외 테스트 실패면 원인부터 기록한다.
2. client.ts:streamAI만 수정하고 위 단일 파일 테스트를 다시 실행해 green 확인. 정상 EOF·read 오류·cancel 실패에도 원래 결과를 유지하는지 검증하고 다음 단계로 간다.
3. 실제 UTF-8 분할 청크와 기존 출력 계약을 확인한 뒤 `npm --prefix web run lint`, `npm --prefix web test` 통과를 기록. 가능하면 위 로컬 HTTP 재현에서 연결 종료도 다시 관찰하되 브라우저 E2E로 보고하지 않는다.
현실이 진단과 다르면 과제서/회차 노트에 근거를 남기고 차선 후보 조건을 판단한다. 완료 상태는 명령 결과를 확인한 뒤 갱신한다.

견적 근거 (pmo 스킬 적용)
- bottom-up: 의존성 준비·회귀 재현 5~8분, 최소 cleanup 7~10분, 경계 테스트 8~10분, 타입/전체 테스트·기록 5~7분 = 기본 25~35분. npm 설치 또는 stream/DOM 예외 차이에 대한 알려진 불확실성 contingency 5~10분을 별도로 두어 총 30~45분, 주관적 확신 중간(통계적 P80 추정 아님).
- 관리 예비비는 배정하지 않는다. 네트워크 장애·새 기능 확장으로 45분을 넘기면 임의로 범위를 늘리지 않고 미완료 이유를 기록한다.
- 과거 기록에는 비교 가능한 실제 소요시간이 없어 유사사례 방식의 수치 교차검증은 미확인. 추정 입력은 정찰이 읽은 2파일의 구현/테스트 범위와 기존 npm 스크립트다.
- 가정·분해·위험·추정 갱신을 기록하는 근거: [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g). 이 소스가 위 분 단위 견적이나 확신도를 보증하는 것은 아니다.

적용한 스킬 원문 (Skill 전용 도구 없음; 로컬 파일로 읽음)
- /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md
- /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md
