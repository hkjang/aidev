# 회차 노트 2026-09-21-160431-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:04] base pinned — main@5bd5058
- [러너 16:04] autonomy release — 

## 정찰 노트
- AI SSE 오류 뒤 stream 취소/잠금 해제 누락을 실제 ReadableStream과 로컬 HTTP fetch로 재현하여 선택; 순수 헬퍼/문서 정리보다 관찰 가능한 효과가 있고 2파일로 제한된다.
- web/node_modules·DB DSN 없어 프런트/DB 통합 미실시; 실제 브라우저→Go→공급자 연결 종료와 비용 영향은 미확인. Go 전체 및 버전·캡처 검사는 통과.
- 오류 cleanup 실패가 원래 ApiError/AbortError를 가리지 않게 하고 SSE 파서/[DONE]·auth·migrations·workflows를 함께 바꾸지 말 것.
- 필수 세 스킬은 headcount/plugins의 로컬 원문으로 읽었으며 이전의 스킬 부재 가정은 이번 환경에서 해소됨. 보류 12개 재평가·유지, 새 후보 3개 추가.
- [러너 16:09] scout done — AI SSE 오류 종료 때 응답 스트림을 취소하고 reader 잠금을 해제 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- 56fcb81: streamAI 오류 경로에 best-effort cancel, 모든 읽기/마지막 버퍼 처리 종료에 finally releaseLock 추가. 원래 오류와 SSE 출력 계약 유지.
- 실제 Response·ReadableStream 회귀 7개: 수정 전 7실패→11개 통과, 구현 제거 시 동일 7실패. event/JSON error·cancel reject·read/AbortError·EOF·UTF-8 분할·DONE 뒤 버퍼 검증.
- npm --prefix web ci / test -- src/api/client.test.ts(11개) / run lint / test(20파일 88개) / run build 모두 exit 0. 실제 fetch+로컬 HTTP에서도 오류 뒤 서버 response close 관찰.
- 확신 없는 곳·검증 못 한 것: 브라우저→Go→외부 공급자의 종료 연쇄·비용 영향, 실제 AbortController로 전송 중 취소하는 E2E는 미검증(AbortError read 실패는 실제 stream으로 검증).
- 의도적 제외: SSE 파서·DONE 의미·OpsChat·Go·의존성·버전 변경 및 Go/DB 재검증은 범위 밖. build dist 제거, 코드 두 파일만 커밋.
- 다음 역할 주의: npm ci moderate 취약점 2건, ResourceListPage 테스트의 jsdom getComputedStyle 미구현 stderr가 있었으나 88개 전부 통과. 브라우저 E2E로 해석하지 말 것.
- [러너 16:13] brief accepted — 채택 — 실제 열린 스트림으로 취소 누락과 잠금 유지를 재현했으며 지정된 두 파일만 변경하여 수용 기준을 검증했다.
- [러너 16:13] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음. 지정 세 스킬 원문 적용; 두 파일 diff·호출부·Go 중계·권한·데이터 흐름·범위·가역성 검토에서 실제 결함 없음.
- client 테스트 11개 통과; 실제 stream의 main 취소 0회·잠금 유지 대비 HEAD 취소 1회·잠금 해제를 독립 확인.
- Node 실제 fetch+로컬 HTTP에서 공급자 오류 및 AbortController 취소의 원래 오류 보존·잠금 해제·서버 연결 종료 확인.
- 브라우저→Go→외부 공급자 E2E·비용 영향 미확인; 전체 프런트/Go/DB 검증은 재실행하지 않음. 릴리즈에서 비용 절감이나 브라우저 E2E 완료로 표현하지 말 것.
- [러너 16:14] review approved — 리뷰 승인 (risk=low)
- [러너 16:14] pr created — https://github.com/hkjang/jupiq/pull/21
