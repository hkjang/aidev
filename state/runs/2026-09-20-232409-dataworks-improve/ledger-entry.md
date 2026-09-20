## 2026-09-20
- 선택: 액션 센터가 공백 포함 계약 만료일을 런타임과 같게 읽도록 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 액션 센터의 valid_to 빈 값 검사와 RFC3339Nano 파싱에 한 번 TrimSpace한 지역 값을 사용해 미래 계약의 잘못된 high 경고와 조회 창 밖 집계를 없앴고 저장·JSON 원문은 유지했다. 실제 SQLite + NewServer(...).Routes() HTTP 회귀 9사례에서 동일 계약·API 키·접근권의 query 200/403과 기본·7d·13w 액션 센터 summary/actions, severity 및 원문 보존을 검증했으며 수정 전과 trim 변경만 되돌린 변이에서 미래 두 사례가 예상대로 실패하고 복구 후 통과했다. 지정 proxy 테스트(1.590초), go build ./..., go vet ./..., go test ./...(proxy 33.952초, 일부 패키지 캐시), go run ./cmd/api-surface-audit(550 routes/612 OpenAPI paths, CLI·SDK 각 5, gap 0), gofmt 및 diff 검사 모두 통과했고 1e9045a로 커밋했다; web 변경이 없어 웹 검증은 미실행이다.
- 보류 아이디어:
  - parseExpiryHorizon의 d/w 곱셈 오버플로 거부 (가치 2 / 위험 1 / 작업량 S)
  - 상품이 없는 참조 행을 고아 전용 운영 경고로 분류 (가치 2 / 위험 2 / 작업량 M)
  - 가이드 캡처 스펙의 하드코딩 버전 문자열 한 곳에서 읽기 (가치 2 / 위험 1 / 작업량 S)
  - 실제 MCP 클라이언트 흐름 e2e를 스텁 Keycloak + Playwright로 편입 (가치 3 / 위험 2 / 작업량 M)
- 과제서: 채택 — 원문 파싱과 런타임 trim 파싱의 불일치가 현재 코드 및 HTTP 재현과 일치했고 수용 기준을 모두 검증했다.
