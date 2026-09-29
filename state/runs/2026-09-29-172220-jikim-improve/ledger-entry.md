## 2026-09-29
- 선택: CSP 진단의 허용 여부를 지시어별 정책과 일치시키기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: Recorder.List가 script/connect/image 허용 목록을 합쳐 연결만 허용한 주소의 스크립트 차단도 허용됨으로 표시하던 문제를 지시어별 조회로 수정했다. 기존 실제 Server routes·Recorder·Config를 이용하는 HTTP 테스트 8개로 페이지 CSP 헤더와 리포트 POST→관리자 GET을 함께 검증하여 수정 전 4개 실패, 수정 후 통과, 원복 시 같은 실패를 확인했다. go test ./... -count=1, 관련 두 패키지 -race, bash scripts/verify.sh(Go test/vet·프런트 59개 테스트·lint/build·문서·Compose) 모두 통과했으며 커밋 ea62e53; PostgreSQL·브라우저 CSP 실행·이미지 E2E는 미검증이고 빌드에는 500kB 청크 경고가 있다.
- 실패 재현: --- FAIL: TestTrackingViolationAllowanceMatchesDirectivePolicy/connect-only_origin_cannot_load_scripts (0.00s) / tracking_test.go:445: directive=script-src origin=https://analytics.google.com allowed=true, want false
- 보류 아이디어:
  - 스니펫 nonce 속성 파싱 경계 검증 (2/2/M): 새 후보, 실제 HTML 파서·브라우저로 먼저 재현 필요.
  - 활성 추적 CSP 리포트 속도 제한 (3/3/M): 신뢰·제한 계약 필요.
  - retryWebhookDelivery 왕복 테스트 (2/1/M): PostgreSQL 통합 경로 필요.
  - baoKVWrite create/update TOCTOU (3/3/M): 트랜잭션·동시성 검증 필요.
