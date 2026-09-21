## 2026-09-21
- 선택: 심의 리포트 날짜 필터를 REST·Excel·MCP 공통으로 검증 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 350d36c에서 reportFilter에 YYYY-MM-DD 고정 형식·실제 달력일·연도 1~9999·기간 순서 검증을 추가하고 두 호출자가 공통 오류를 전달하도록 해 REST JSON/Excel은 422 VALIDATION_FAILED와 필드 details, MCP는 원문 없는 날짜 안내의 result.isError를 반환한다. 전용 PostgreSQL 16과 실제 NewServer·로그인 세션·HTTP로 수정 전 오류를 재현하고 수정 되돌림으로 재실패도 확인했으며, 정상 7개 조건에서 JSON 전체와 MCP structuredContent 및 실제 XLSX 요약 집계가 일치하고 display_day_start의 종료일 포함 의미가 유지됨을 검증했다. 지정 회귀 명령은 PASS 83/FAIL 0/SKIP 0(부모·하위 포함), 제한 race PASS(5.547s), TEST_POSTGRES_DSN 포함 go test ./... 전 패키지 PASS(web 159.641s), go vet ./... exit 0이며 원격 CI·취약점 게이트·릴리즈는 검증하지 않았다.
- 보류 아이디어:
  - docs/user-guide.md 제거해 USER_GUIDE.md 하나만 정본으로 (가치 2 / 위험 1 / 작업량 S)
  - 리포트 월 시작 프리셋의 로컬 자정→UTC 날짜 이동 수정 (가치 3 / 위험 2 / 작업량 M)
  - 심의번호의 생성일 의존성과 과거 자료 이관 시 번호 충돌 조사 (가치 3 / 위험 3 / 작업량 M)
  - MCP 리포트 날짜 인자의 JSON 타입 검증 (가치 3 / 위험 2 / 작업량 M)
- 과제서: 채택 — 날짜 검증 누락과 두 호출자 배선이 현재 코드와 일치했고 전용 DB에서 500 및 비정규 날짜·역전 기간의 잘못된 허용을 실제로 재현했다.
