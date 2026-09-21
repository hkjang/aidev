## 2026-09-21
- 선택: Catalog·Content query 변경 시 이전 조건의 진행 조회를 무효화한다 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: query 적용 시 기존 request guard를 무효화하고 loading·이전 결과·완료 상태를 함께 초기화해 늦은 응답의 캐시 복구와 이전 행 노출을 막았다. 실제 Vue 컴포넌트·memory router·기능 API·apiGet/apiGetPage·Axios transport·envelope를 통과하는 신규 20개 테스트 중 수정 전 16개 실패를 확인했고, 수정 후 대상 48개와 npm run build(타입 검사·전체 193개·Vite·설정·offline·integrity 19개)가 통과하여 15e032f로 커밋했다. 요청된 technology 스킬/Skill 도구는 찾지 못해 적용했다고 주장하지 않으며 실제 서버/UAT는 미검증이다.
- 보류 아이디어:
  - AdvancedPolicyView 미저장 편집 보존 (가치 3 / 위험 2 / 작업량 M)
  - Catalog·Directory·Content 기존 테스트 wrapper 정리 (가치 2 / 위험 1 / 작업량 S)
  - 세 목록 화면 재조회 중 표 유지 (가치 2 / 위험 2 / 작업량 M)
  - 실제 HTTP envelope부터 Catalog 범위 밖 페이지 복구 검증 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 현재 코드와 근거가 일치했고, 실제 배선 테스트로 query 변경 뒤 이전 요청이 새 조회를 막거나 이전 결과를 남기는 문제를 양쪽 화면에서 재현했다.
