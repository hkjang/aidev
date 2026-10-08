## 2026-10-08
- 선택: 페이지 숫자 파서의 정수 overflow를 기본값으로 처리 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: fmtSscanf의 십진수 누적에 int 범위 검사를 넣어 큰 limit/offset이 양수로 감겨 다른 페이지를 반환하던 문제를 수정했다(프로덕션 1파일, 테스트 1파일, API 가이드 1파일; 커밋 fb7ecd8). 기존 숫자 문법·정상값·기본값은 유지하며, 실제 New → 인증/미들웨어/라우트 → PostgreSQL → 최종 HTTP 목록에서 수정 전 실패·수정 후 통과 및 범위 검사 제거 시 재실패를 확인했다. JIKIM_TEST_POSTGRES_DSN을 설정한 회귀 테스트(단위 12개·통합 6개), go test ./... -count=1 및 bash scripts/verify.sh 모두 exit 0(Go vet/포맷·프런트 59/59·lint/build·문서·Compose 포함); 임시 PostgreSQL 17과 스키마를 정리했으며 이미지·브라우저 E2E는 미실행, 기존 500kB 청크 경고는 남아 있다.
- 실패 재현: `response_test.go:158: returned 1 applications, want 3 in the expected order` / `response_test.go:158: returned 2 applications, want 3 in the expected order` — 수정 전 실제 API의 overflowing_limit/overflowing_offset에서 exit 1. 총 단위 3개·통합 4개 실패, 나머지는 처음부터 통과. 범위 검사만 제거해 같은 7개 실패를 다시 확인하고 복원했다.
- 보류 아이디어:
  - Momento 프록시의 인코딩된 URL 경로 보존 (3/2/S): RawPath 제거가 %2F 의미를 바꿀 가능성, 실제 collector 관찰로 재현부터 필요.
  - CSP 진단 페이지의 UTF-8 절단 방지 (2/1/S): page[:512] 경계가 다중 바이트 문자를 자를 수 있어 최종 관리자 응답 검증 필요.
  - 활성 추적 CSP 리포트 속도 제한/동일 출처 검증 (3/3/M): 별도 신뢰·제한 계약 필요, 본문 크기 상한과 다른 문제.
  - retryWebhookDelivery 왕복 테스트 공백 (2/1/M): 실제 PostgreSQL 기반 이력/재시도 검증을 별도 범위로 진행.
