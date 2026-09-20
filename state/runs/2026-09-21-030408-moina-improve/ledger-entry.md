## 2026-09-21
- 선택: 미디어 multipart 본문 한도 초과를 400 대신 413 media_too_large로 분류 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: ParseMultipartForm의 *http.MaxBytesError만 errors.As로 구분해 기존 크기 오류 메시지와 413 media_too_large를 반환하고 OpenAPI 설명을 보강했습니다(1bd4408). 실제 multipart.Writer·httptest 10케이스 중 본문 초과 2케이스(ContentLength 알려짐/-1)가 수정 전 400/413 불일치로 실패한 뒤 수정 후 통과했으며, 지정 집중 테스트·go test -race ./...·go vet ./...·make fmt·make check 모두 통과했습니다. DB DSN이 없어 PostgreSQL integration은 skip이며 회사 세 스킬은 도구/로컬 경로에 없어 고유 절차·반환 형식 준수는 미확인입니다.
- 보류 아이디어:
  - Makefile test에 CI와 같은 -race 적용 (가치 2 / 위험 1 / 작업량 S)
  - 가이드 Keycloak OIDC·SMTP·방문 추적 화면 재캡처 (가치 3 / 위험 1 / 작업량 M)
  - 관리 설정 placeholder origin의 시각 회귀 정규화 (가치 2 / 위험 1 / 작업량 S)
  - 미디어 415 및 인증·DB 통합 거절 경로 회귀 테스트 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 큰 본문에서 400이 반환되는 현상을 수정 전 재현했고 지정한 최소 오류 분리와 검증을 완료했습니다. error.code는 기존 HTTP 계약의 최상위 code 필드로 검증했습니다.
