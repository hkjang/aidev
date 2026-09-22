## 2026-09-22
- 선택: OpenAPI 계약 테스트에 실제 YAML 파싱 검증을 추가해 구문 오류 재발 차단 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 412a894에서 기존 경로 추출·예외 비교는 유지하고 디스크 문서 전체를 읽는 공통 함수에 yaml.v3 v3.0.1 파싱을 추가했다. 현재 문서 복제본의 정상 인용은 통과하고 과거 /login?sso=limited 인용 누락·닫히지 않은 flow mapping·paths 밖 components 구문 오류는 거부하며, 구현 전 및 파싱 입력을 빈 문서로 바꾼 변형에서 오류 사례 셋이 실패하고 원복 후 통과했다. go test -count=1 -run 'OpenAPI|UndocumentedRoute' ./internal/api, go vet ./..., go test -count=1 ./..., go build ./... 통과 및 go mod tidy 재실행 무변경 확인; DB DSN 없는 통합 skip은 실검증에 포함하지 않았고 프런트·실제 Keycloak은 미검증이다.
- 보류 아이디어: requestList의 API 오류 code 보존 (가치 2 / 위험 1 / 작업량 S)
  OpenAPI page_size 상한 불일치 정리 (가치 2 / 위험 2 / 작업량 S)
  internal/store 순수 헬퍼 5개 표 기반 테스트 (가치 2 / 위험 1 / 작업량 S)
  OpenAPI servers URL과 계약 테스트 접두사 불일치 검출 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 현재 코드의 파싱 공백과 695591e 오류 유형이 과제서와 일치해 지정 범위 및 수용 기준을 그대로 구현했다.
