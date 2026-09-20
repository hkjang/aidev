## 2026-09-21
- 선택: Langflow e2e가 runtime-settings의 실제 응답 구조로 백업하고 원래 프로필을 복원하도록 수정 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: GET 응답의 settings.profiles를 2xx·JSON·객체·배열 검증 후 깊게 복제하고 finally에서 {profiles: 백업배열}만 복원하는 전용 모듈을 실제 e2e에 연결했으며, 복원 실패와 검사·복원 이중 오류를 전파한다(커밋 a439f88). 기존 before.profiles 방식에서 원본 프로필 두 개가 사라지는 실패를 먼저 재현했고, 수정 뒤 실제 Server.Handler+격리 PostgreSQL 16+관리자 세션/CSRF+Node의 5개 시나리오에서 정상·예외·일부 PUT 실패의 DB/GET 보존과 무인증 GET 401의 PUT 0회를 확인했다. 신규 Node 27건+기존 30건, Langflow 구문 검사, Go API/runtime-proxy, DB 연결 전체 Go race(-p 1), 웹 npm ci/lint/build가 통과했으며 Ready Langflow 브라우저 검사는 미실행이다.
- 보류 아이디어:
  - guide-shots 전역 설정 백업 실패 시 seed 중단 (가치 4 / 위험 1 / 작업량 M)
  - guide-shots 복원 한 건이 네트워크 오류여도 나머지 설정 복원을 계속함 (가치 3 / 위험 1 / 작업량 M)
  - 게이트웨이 reporter.send non-2xx 보고 실패 로그 (가치 2 / 위험 2 / 작업량 S)
  - sessionGateway Node 장애 회귀 테스트를 기본 검증 명령에 포함 (가치 2 / 위험 2 / 작업량 S)
- 과제서: 채택 — 실제 API의 settings 래퍼와 기존 스크립트의 최상위 profiles 접근 불일치를 확인했고 지정한 범위 안에서 수정·실제 DB 회귀 검증을 완료했다.
