## 2026-09-21
- 선택: AI SSE 오류 종료 때 응답 스트림을 취소하고 reader 잠금을 해제 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: streamAI가 오류를 던질 때 reader 정리가 없어 연결과 잠금이 남는 것을 실제 Response·ReadableStream으로 재현하고, 오류 시 best-effort cancel과 finally releaseLock을 추가했다(56fcb81). 회귀 7개가 수정 전 실패·수정 후 통과·수정 제거 시 재실패했고, ApiError 메시지·502, read 오류·AbortError 원본, 취소 실패, 정상 EOF, UTF-8 분할·DONE 이후 마지막 버퍼 출력을 검증했다. npm --prefix web ci, test -- src/api/client.test.ts(11개), run lint, test(20파일 88개), run build가 exit 0이며 실제 fetch 로컬 HTTP에서 응답 연결 종료를 관찰했다; 실제 브라우저→Go→공급자 E2E는 미검증이고 npm 설치의 moderate 2건 및 별도 ResourceListPage 테스트의 jsdom getComputedStyle 경고는 범위 밖으로 유지했다.
- 보류 아이디어: OpenAPI page_size 상한 불일치 정리 (가치 2 / 위험 2 / 작업량 S)
  internal/store 순수 헬퍼 5개 표 기반 테스트 (가치 2 / 위험 1 / 작업량 S)
  OpenAPI 계약 테스트가 실제 YAML 구문 오류도 검출하도록 보강 (가치 3 / 위험 2 / 작업량 M)
  requestList도 request와 같은 API 오류 code를 보존 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 실제 열린 스트림으로 취소 누락과 잠금 유지를 재현했으며 지정된 두 파일만 변경하여 수용 기준을 검증했다.
