- 과제: 가이드 추적 캡처의 CSP 보고에 제한 시간을 적용해 전역 설정 복원 경계로 빠져나오게 한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `web/scripts/guide-shots.mjs:captureTracking`은 추적 설정을 켠 다음 제한 시간 없는 CSP 보고 fetch를 기다리므로 서버가 응답하지 않으면 네 전역 설정의 복원도 시작하지 못한다. 기존 REST 요청과 같은 제한 시간을 적용하면 보고 실패가 유한 시간 안에 전파되고 `withGuideSettings`가 원래 설정 복원을 모두 시도할 수 있다.
- 수용 기준: 1) `AGENTHUB_GUIDE_REQUEST_TIMEOUT_MS`의 동일한 양수 설정값(미설정 기본 30000ms)이 CSP 보고에도 전달되고, 응답 없는 보고를 중단한 뒤 policy·dlp·sessionGateway·tracking 원본 복원을 각각 시도한다. 복원 성공까지 네트워크 장애 상황에서 보장한다는 뜻은 아니다. 2) 정상 요청은 POST `/api/v1/tracking/csp-report`, `application/csp-report`, 현재 document-uri·blocked-uri·directive 본문, HTTP 204 성공 판정과 기존 촬영/정리 순서를 유지한다. `GUIDE_SKIP_SEED=1`은 기존대로 캡처만 수행한다. 3) 실제 호출부 → 실제 captureTracking → 실제 withGuideSettings를 실행하는 회귀가 정상 보고와 멈춘 보고를 검증한다. 타임아웃 오류는 POST 경로와 제한 시간을 식별하고, 복원 한 건이 실패해도 나머지를 시도하며 원래 보고 오류가 우선한다. CSP 요청의 signal 또는 호출부의 제한 시간 전달을 제거하면 회귀가 실패해야 한다.
- 건드릴 파일: `web/scripts/guide-shots.mjs:95 requestTimeoutMs, 134 captureTracking 호출부, 428 captureTracking` — 같은 제한 시간을 옵션으로 넘기고 page.evaluate의 직렬화 인자로 전달하여 CSP fetch에 AbortSignal.timeout을 적용한다. 기존 call의 오류 안내처럼 요청 경로와 시간을 식별할 수 있게 한다. `web/scripts/guide-settings-check.test.mjs:9 boundary, 36 run, 69 captureTracking 대역` — 실제 함수와 호출부가 함께 실행되는 선택적 하니스를 추가하고 보고 지연/정상/복원 실패를 관찰한다. 프로덕션 1개 + 테스트 1개이며 새 모듈은 필요 없다.
- 검증 명령: 저장소 루트에서 `node --check web/scripts/guide-shots.mjs`; `cd web && node --test scripts/guide-settings-check.test.mjs`; `cd web && node --test scripts/*.test.mjs`; `git diff --check`. 앞의 세 명령은 정찰에서 실제 통과했다(전체 Node 회귀 147건, 실패·skip 0). Go·웹 빌드·실제 브라우저/DB는 이번 정찰에서 실행하지 않았다. 변경이 두 스크립트에 한정되면 Node 검증을 중심으로 하고 CI 필수 검증은 그대로 따른다.
- 위험과 피할 것: auth·세션·migrations·workflows·서버 API·runtime 이미지 버전을 건드리지 않는다. tracking 위반 전체 DELETE, problems 요약 누락, 환경변수 범위 검증은 별도 후보이며 이번에 섞지 않는다. `withGuideSettings`의 백업 완료 전 쓰기 금지·복원 전체 시도·원래 작업 오류 우선 계약을 유지하고 복원 오류를 삼켜 성공으로 만들지 않는다. page.evaluate는 Node 클로저를 쓸 수 없으므로 timeout은 인자로 전달해야 한다. 현재 CSP 보고는 captureTracking 내부 try 앞에 있지만 외부 withGuideSettings 안에 있다. 내부 finally 이동까지 해야 복원된다고 오해하지 말 것. `runtime-images.json` 14개 이미지 sourcePaths에 web 경로가 없는 것을 재확인했다.
- 차선 후보: 복원 실패가 guide-shots의 problems 요약 출력을 건너뛰는 것을 고친다 (가치 2 / 위험 1 / S) — 130행 await가 throw하면 138행 요약을 건너뛴다. 첫 과제가 현재 코드에서 이미 해결됐거나 재현이 틀렸을 때만 전환하고, 원래 예외/실패 종료를 보존하며 실제 요약 실행을 검증한다.

근거와 재현(정찰 완료)
- 기준 HEAD `6af55c3`(v0.261.0), 작업 트리 깨끗. 최근 git log 30건·README·docs/architecture.md·CI·테스트 구성 확인. CLAUDE.md/AGENTS.md/roadmap/TODO 파일은 저장소에서 발견하지 못했고 cmd/internal/web/src/web/scripts/scripts의 TODO/FIXME 검색도 결과 없음.
- guide-shots의 fetch 호출은 104행과 435행 두 곳이다. 104행에는 signal이 있지만 435행에는 없다. 기존 guide-settings-check.test.mjs의 `captureTracking`은 콜백 이름만 기록하고 실함수를 실행하지 않으므로 기존 147건 통과가 이 경로를 증명하지 않는다.
- 코드 변경 없이 Node vm에 `requestTimeoutMs`~`if (problems.length)` 직전의 실제 호출부와 파일 끝의 실제 `captureTracking`을 넣고, 실제 `withGuideSettings`를 import했다. 로컬 node:http 서버와 진짜 fetch를 사용하되 page.evaluate는 전달된 함수를 실행하는 어댑터, 촬영은 no-op으로 두었다. GET은 정상 백업값, PUT/DELETE는 성공 응답을 주고 CSP POST만 보류했다.
- 설정 20ms, CSP 서버 수신 후 120ms 관찰 결과 `reportHasSignal=false`, `settled=false`, 쓰기는 tracking 활성화 PUT 1건뿐이었다. CSP 응답을 204로 풀자 tracking 끄기/위반 DELETE 뒤 policy·dlp·sessionGateway·tracking 원본 복원 PUT 4건이 진행됐다. 이는 실제 브라우저/관리자 DB 검증이 아니라 스크립트 배선과 실제 HTTP 대기의 재현이다.

해법 비교와 선택
- 선택: 기존 CSP 전용 fetch를 유지하고 기존 제한 시간 값만 인자로 전달한다. 마크업 보고 Content-Type과 location.origin을 보존하며 프로덕션 1파일에 국한된다.
- 대안: 공통 call에 Content-Type 옵션을 추가해 post로 CSP 보고까지 통합한다. timeout/오류 처리 중복은 줄지만 공통 REST 어댑터와 body 생성 위치까지 바뀌어 이번 한 경로 수정에는 불필요한 계약 변화다.
- 대안: page.evaluate 또는 전체 촬영에 Promise.race만 씌운다. 실제 fetch를 취소하지 않아 복원과 늦은 요청이 겹칠 수 있으므로 채택하지 않는다. 현상 유지도 보고 장애가 설정 복원을 막아 채택하지 않는다.
- 핵심 가정: 기존 유효한 양수 timeout의 AbortSignal 동작을 CSP 요청에도 그대로 쓸 수 있다. 실제 Chromium 동작은 미확인이고 정찰 재현은 Node AbortSignal/fetch 기반이다.

구현 순서 및 점검(구현자는 완료 시 상태 갱신)
1. [대기] 테스트 파일에 실제 captureTracking 실행을 선택할 수 있는 하니스를 넣고 정상 204 경로·원본 복원·요청 본문을 검증한다. 기존 단순 대역 테스트는 유지한다. 증명: `cd web && node --test scripts/guide-settings-check.test.mjs`. 점검: 자동 테스트, 사람 확인 없음.
2. [대기] guide-shots의 호출부/직렬화 인자/실제 fetch를 연결하고 응답 없는 CSP 보고 회귀를 추가한다. 실제 AbortSignal과 fetch/로컬 HTTP 서버 또는 signal의 abort에만 반응하는 하니스를 쓴다. 서버/interval은 finally에서 정리하고 독립 watchdog으로 테스트 자체의 무한 대기를 막는다. captureTracking 대역에 직접 throw하는 것만으로 증명하지 않는다. 증명: 같은 Node 테스트와 `node --check web/scripts/guide-shots.mjs`; signal 제거 시 신규 회귀 실패 확인 후 원복. 점검: 자동, 사람 확인 없음.
3. [대기] 전체 Node 회귀와 `git diff --check` 통과를 확인하고 실제 브라우저/DB 미검증 여부를 구현 노트에 남긴다. 예상보다 공통화 범위가 커지면 작업을 넓히지 말고 과제서를 수정한다. 점검: 뒤이은 비평 단계가 diff·실행 증거를 검토한다.

작업량 추정과 여유
- bottom-up 판단: 하니스/정상 경로 8–10분 + timeout 배선/멈춤 회귀 10–15분 + 전체 회귀/검토 5–8분 = 기본 23–33분. 알려진 불확실성인 vm 슬라이스·연결 정리 조정에 별도 contingency 5분, 총 28–38분(중간 확신의 판단 범위, 통계적 신뢰구간 아님). 미지의 범위 추가용 management reserve는 배정하지 않고 다음 회차로 분리한다.
- 근거는 직접 읽은 2파일과 기존 stalls/복원 하니스다. 과거 동일 작업의 실측 시간은 없어 유사사례 추정치를 꾸미지 않았다. 새 의존성·DB·브라우저 설치·공통 요청 계층 재설계는 추정에서 제외한다. 1단계 후 하니스 재사용 가능성이 달라지면 재추정한다.
- 적용 스킬: 로컬 pmo/estimating-and-contingency, technology/implementation-planning, technology/solution-exploration의 SKILL.md를 직접 읽었다(세션에 전용 Skill 도구 없음). 분해·가정/범위·여유 분리, 단계별 증명/점검, 대안 비교를 반영했다. estimating 스킬의 references/sources.md도 확인했으며 이 소규모 작업의 분 단위 범위는 외부 문헌의 수치가 아니다.
