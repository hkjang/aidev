- 과제: 목록·AI HTTP 오류에서도 서버 error.code를 보존 (가치 2 / 위험 1 / 작업량 S)
- 왜: web/src/api/client.ts의 request는 서버 error.code를 ApiError에 넣지만 requestList와 streamAI의 HTTP 오류 분기는 버린다. 세 공개 요청 경로가 같은 서버 오류를 같은 status/code/message로 전달하면 호출자가 오류 원인을 일관되게 판단할 수 있다.
- 수용 기준:
  1) request·requestList·streamAI가 비정상 HTTP JSON 응답 {error:{code:"forbidden",message:"권한이 없습니다."}}에서 실제 ApiError 인스턴스와 status=403, code="forbidden", message를 모두 보존한다. 다른 코드 invalid_query/400도 같은 표에서 검증한다.
  2) code 없는 객체 오류·문자열 error·최상위 message·비JSON 본문에서도 기존 메시지 우선순위와 code=undefined를 유지한다. 정보가 없을 때 request는 "요청에 실패했습니다. (상태)", requestList는 "목록을 불러오지 못했습니다. (상태)", streamAI는 "AI 요청에 실패했습니다. (상태)"를 유지한다.
  3) 일반 요청의 data unwrap, 목록의 meta·중첩 data 평탄화, 정상 SSE 및 오류 종료 시 reader 해제는 기존 테스트를 계속 통과한다. 네트워크 실패와 AbortError, HTTP 200 뒤 SSE event:error의 계약은 바꾸지 않는다.
  4) 실제 로컬 HTTP 서버와 네이티브 fetch로 세 export를 호출하는 회귀 테스트가 수정 전 목록·AI의 code 단언에서 실패하고 수정 후 통과한다. code 전달만 되돌리면 같은 테스트가 다시 실패해야 한다. ApiError·클라이언트를 모킹하거나 손으로 만든 Response 대역을 결함 증거로 쓰지 않는다.
- 건드릴 파일:
  - web/src/api/client.ts: request·requestList·streamAI HTTP 오류 분기 — 기존 request의 error.code 추출 규칙을 작은 비공개 오류 생성 함수로 공유하거나 같은 동작으로 맞춘다. errorMessage는 기존 우선순위를 유지한다. 프로덕션 변경은 이 파일 하나.
  - web/src/api/client_http_errors.test.ts (신규): 실제 HTTP 서버 회귀 테스트. 참고로 실제 열어 본 web/src/api/client.test.ts에는 request 코드 보존·목록 평탄화·SSE 테스트가 있고, web/src/test/setup.ts는 jsdom 전역과 정리를 담당한다. 기존 테스트 파일 수정은 필요한 경우만.
- 검증 명령 (저장소 루트):
  - 의존성이 없으면 npm --prefix web ci
  - npm --prefix web test -- src/api/client_http_errors.test.ts src/api/client.test.ts
  - npm --prefix web test
  - npm --prefix web run lint
  - git diff --check
  - 참고로 정찰에서 실행한 go test -count=1 ./internal/api ./internal/store는 api 0.060s/store 0.006s 통과. DSN 미지정이므로 DB 통합 검증은 아니다. 프런트 npm 명령은 package.json·vite.config.ts·CI에서 확인했으나 이 작업 트리에 node_modules가 없어 정찰에서는 미실행.
- 위험과 피할 것: auth·migrations·workflows·의존성·SSE 파서·재시도·HTTP 파싱 정책 변경 금지. requestList를 request로 치환하면 meta가 사라질 수 있으므로 성공 경로를 통합하지 않는다. 실제 화면에서 code를 이용한 분기 개선 효과는 미확인으로, 사용자 화면 변경을 약속하지 않는다. 과거 SSE 정리 변경을 다시 구현하지 않는다.
- 차선 후보: OpenAPI /users page_size maximum 100을 실제 pageBounds 상한 200과 맞추기 — 실제 라우터 경계 검증을 먼저 하고 문서만 고친다. 이 차선의 HTTP 경계 호출은 정찰에서 미확인.

범위·근거:
- base main@93943d7. 최신 메일 status 검증(a6cdb74)은 완료되어 제외했다.
- 실제 열람: client.ts:ApiError·errorMessage·parseResponse·request·requestList·streamAI, client.test.ts, useList.ts, vite.config.ts, test/setup.ts, package.json.
- 정찰 재현은 Node 22.23.1의 node:http 서버를 127.0.0.1의 임의 포트에 띄워 동일한 JSON 403을 반환하고 네이티브 fetch를 사용했다. 원본 client.ts를 메모리에서 타입 제거하고 Vite 전용 import.meta.env.VITE_API_BASE_URL만 서버 주소로 치환했다. 로직 수정·fetch 대역 없이 세 export를 실행했다.
- 관찰값: request → ApiError/403/forbidden, requestList → ApiError/403/undefined, streamAI → ApiError/403/undefined. 셋 모두 같은 서버 메시지를 보존했다. 이는 클라이언트 경계의 실행 증거이며 실제 Go 서버·브라우저 E2E를 실행했다는 뜻은 아니다.

구현 순서·체크포인트 (현재 모두 pending, 사람 승인 체크포인트 없음):
1. 신규 테스트에 node:http의 createServer를 사용해 실제 HTTP 오류를 반환한다. 테스트 파일 격리를 유지하고 vi.stubEnv로 VITE_API_BASE_URL을 임의 포트 /api/v1로 지정한 뒤 client를 동적 import한다(필요 시 vi.resetModules). fetch를 stub하지 않는다. afterAll에서 서버를 닫고 env를 원복한다. 기존 jsdom setup은 유지하고 node 환경으로 바꿔 window 참조를 깨뜨리지 않는다. 위 집중 테스트 명령으로 기존 정상 테스트 통과 및 신규 code 단언 실패를 확인한다. 빌드는 유지되며 이 실패는 의도한 회귀 증거다.
2. client.ts의 HTTP 오류 생성만 맞추고 같은 명령으로 녹색을 확인한다. 세 경로의 기본 메시지는 각 호출부에서 넘긴다. code 전달 되돌림 검증은 임시로 실행 후 반드시 원복한다.
3. 전체 npm test·lint·diff 검사를 실행하고 결과와 미실행 항목을 기록한다. 테스트 서버/환경 오염이 생기면 테스트 격리를 고치며 제품 범위를 늘리지 않는다. 예상과 다르면 이 과제서에 원인을 기록하고 범위를 재판단한다.

해결안 비교:
- 선택: HTTP 오류 생성만 공유/정렬 — 프로덕션 1파일, 성공·스트림 계약을 보존하며 확인된 누락 2곳을 동시에 해결한다.
- 목록만 수정: 가장 작지만 같은 누락이 AI HTTP 경로에 남으므로 제외했다.
- 요청 전체 통합: 장래 확장은 쉬울 수 있으나 목록 meta·평탄화·SSE 계약이 달라 위험과 작업량이 불필요하게 커진다.
- 무변경/예방 테스트만: 현재 화면은 주로 메시지를 표시하므로 긴급 장애는 아니다. 그래도 실제 누락이 재현되어 순수 헬퍼 예방 테스트·문서 캡처보다 근거가 강하다. 새 라이브러리는 필요 없다.

작업량 근거·예비시간:
- bottom-up 실무 추정: 실제 HTTP 회귀 10~15분 + 오류 전달 수정 3~5분 + 전체 테스트·타입 검사 5~8분 = 기본 18~28분. 알려진 변동(jsdom/env 격리 및 npm 준비) 예비시간 5~10분, 총 23~38분을 예상한다. 통계적 보장이 아닌 중간 확신의 범위다.
- 과거 기록은 클라이언트 SSE 회귀 추가가 성공했다는 비교 근거만 제공하고 소요시간 표본은 없다. 따라서 유사사례 추정치를 숫자로 꾸며 두 방법이 일치한다고 주장하지 않는다.
- 가정: Node 22 및 npm 레지스트리/캐시 사용 가능, HTTP 로컬 listen 가능. 정찰의 Node HTTP 재현은 성공했지만 Vitest 환경은 미확인이다. 45분 초과 시 새 기능을 넣지 말고 환경 장애를 명시한다. 관리 예비시간(미정의 범위)은 이번 과제에 배정하지 않으며 제품 범위 확장은 별도 회차로 넘긴다.
- 적용 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration의 로컬 SKILL.md를 읽었다. 전용 Skill 도구가 노출되지 않아 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills 아래 원문으로 절차를 적용했다.
