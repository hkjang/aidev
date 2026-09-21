## 2026-09-21
- 선택: 사용자 목록 동일 created_at 행의 페이지 순서 고정 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: listUsers 목록 SELECT에 id DESC 보조 정렬을 추가하고 API 문서에 생성 시각 내림차순·동률 ID 내림차순 계약을 명시했다(54afcc5, 지정된 3개 파일만 변경). 전용 PostgreSQL 16과 New(...).Handler()를 통과하는 새 테스트에서 동일 timestamp 고정 UUID 6개를 섞어 삽입하고 다른 timestamp 2개 및 seed 관리자를 포함해 q 유무·pageSize 2/3별 전체 기대 ID 순서, 중복·누락 없음, page/pageSize/total 및 마지막 빈 페이지를 검증했으며 수정 전 네 사례 실패와 수정 후 PASS를 확인했다. TEST_POSTGRES_DSN을 설정한 go test -count=1 ./...(server 12.082초), 지정 두 테스트의 -v PASS(SKIP 아님), go test -race -count=1 ./...(server 78.828초), make lint, go build ./... 모두 통과했으며 웹·실제 Keycloak은 미검증이고 요청된 technology 세 스킬/Skill 도구는 찾지 못해 고유 절차는 적용하지 못했다.
- 보류 아이디어:
  - loadGrants roles·permissions 정렬 보존 (가치 2 / 위험 1 / 작업량 S)
  - 합성 레거시 스키마·데모 시드의 재사용 SQL 제공 (가치 3 / 위험 2 / 작업량 M)
  - 감사 CSV 문서의 전체 이벤트 보장 표현을 최대 50,000건으로 한정 (가치 1 / 위험 1 / 작업량 S)
  - 사용자 목록 roles 배열의 순서 명시 (가치 1 / 위험 1 / 작업량 S)
- 과제서: 채택 — 현 코드의 보조 정렬 누락이 확인되었고 전용 폐기 PostgreSQL을 확보해 지정된 수용 기준과 검증을 완료했다.
