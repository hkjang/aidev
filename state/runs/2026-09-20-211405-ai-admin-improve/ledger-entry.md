## 2026-09-20
- 선택: 감사 CSV의 선행 탭·CR 셀을 기존 수식 셀과 동일하게 중화 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: safeCSVCell의 선행 문자 집합에 탭·CR만 추가하고 API 문서에 작은따옴표 하나를 붙이는 계약을 명시했다(커밋 42c62bc, 변경 4개). 표 테스트 19개와 실제 PostgreSQL 16 → New.Handler → 인증된 GET 내보내기 통합 사례 11개로 HTTP 200·CSV Content-Type·BOM·13열·정상 셀 보존 및 actor/action/resource_type/request_id/stage/reason 중화, DB 원문·JSON 응답 불변을 확인했으며 수정 전 실패와 탭·CR 처리만 제거한 실패를 모두 확인했다. 전용 폐기 DB에서 go test -count=1 ./...(server 12.4초), 지정 -v 회귀 테스트(PASS, SKIP 아님), go test -race -count=1 ./...(server 79.7초), make lint, go build ./... 통과; 실제 스프레드시트와 Keycloak E2E는 미검증, 웹 변경이 없어 웹 테스트·번들 빌드는 생략했고 요청된 technology 스킬/Skill 도구를 찾지 못해 고유 절차는 적용했다고 주장하지 않는다.
- 보류 아이디어:
  - 사용자 목록 동일 created_at의 페이지 순서 고정 (가치 2 / 위험 1 / 작업량 S)
  - loadGrants roles·permissions 정렬 보존 (가치 2 / 위험 1 / 작업량 S)
  - 합성 레거시 스키마·데모 시드의 재사용 SQL 제공 (가치 3 / 위험 2 / 작업량 M)
  - 감사 CSV 문서의 전체 이벤트 보장 표현을 최대 50,000건으로 한정 (가치 1 / 위험 1 / 작업량 S)
- 과제서: 채택 — 선행 탭·CR 누락과 기존 테스트 구성 모두 현재 코드와 일치하여 지정된 네 파일만 변경했다.
