## 2026-09-21
- 선택: MCP search_suppliers가 공급업체 번호로도 검색되게 하기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: search_suppliers의 기존 검색 괄호에 supplier_number ILIKE 조건만 추가해 REST 목록·통합 검색과 번호 검색 동작을 맞췄다(commit a05402d). 실제 PostgreSQL·로그인 세션·App.Handler 회귀에서 이름/사업자번호를 번호와 분리하여 수정 전 MCP만 실패함을 확인했고, 수정 후 전체·부분·대소문자 번호, 이름·사업자번호, 부서·own·소프트 삭제 제외와 MCP 구조화/텍스트 응답을 검증했다. 서로 다른 전용 DB 세 개와 세 DSN으로 전체 Go 테스트(httpapi 23.440초), 지정 회귀(SKIP 없음), go vet·gofmt·diff 검사가 통과했으며 요청한 technology 스킬은 도구·리소스·로컬 경로에 없어 절차/반환 형식은 미확인이다.
- 보류 아이디어:
  - 사용자 가이드 MCP 도구표와 실제 tools/list 응답 비교 (가치 2 / 위험 1 / 작업량 S)
  - MCP search_suppliers와 analyze_spend의 limit 인자 반영 (가치 2 / 위험 2 / 작업량 S)
  - 작업 항목 상태 배치 상한 문서화·테스트 (가치 2 / 위험 1 / 작업량 S)
  - 관제탑 결재 역할 필터보다 먼저 적용되는 200건 상한 보완 (가치 3 / 위험 3 / 작업량 M)
- 과제서: 채택 — MCP에만 번호 조건이 없는 차이를 현재 코드와 수정 전 실제 DB/API 실패로 확인하여 지정 범위대로 구현했다.
