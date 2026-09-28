## 2026-09-29
- 선택: 액션 센터가 운영 목록 조회 오류를 정상 0건으로 숨기지 않도록 반환 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 액션 센터가 적합도·계약·권한·Watermark·비용·폐기 후보 목록 조회 오류를 버리고 HTTP 200과 0건 집계를 반환하던 경로를 각 조회 오류별 HTTP 500으로 바꿨다. 실제 store.SQLStore(SQLite)·NewServer.Routes()를 사용하는 HTTP 회귀 테스트 6사례로 각 테이블을 일시 rename하여 수정 전 실패→수정 후 통과→프로덕션 코드 원복 시 동일 실패를 확인했고, 매번 테이블 복구 후 정상 빈 목록은 200·빈 actions·9개 0 집계를 유지함을 확인했다. go test ./internal/proxy -run ActionCenter -count=1 통과(3.997s), go build ./...·go vet ./...·go test ./... -count=1 전체 통과(proxy 40.347s, store 15.172s), go run ./cmd/api-surface-audit 누락 0(550 routes/612 OpenAPI paths), npm ci(취약점 0)·npm run lint·npm test(9파일/27테스트)·npm run build 통과 및 gofmt/git diff --check 클린; 프로덕션 1파일+테스트 1파일+OPERATIONS 문서 1파일, 커밋 75b9d6d, 빌드 산출물 미포함. 실행 환경은 Go 1.26.7/Node 22.23.1(CI 1.25/24와 다름); PostgreSQL·브라우저 e2e·외부 연동은 미검증이며 퍼블리시 게이트 내부 오류 처리는 별도 후보로 유지했다.
- 실패 재현: `--- FAIL: TestDataWorksActionCenterRejectsUnavailableInventories/dw_contract_scopes` — `admin_dataworks_action_center_test.go:682: unavailable dw_contract_scopes: status = 200, want 500: {"actions":[],"expiring_within":"720h0m0s","summary":{"approval_pending":0,"blocked_launches":0,"expiring_access":0,"expiring_contracts":0,"inactive_access":0,"low_fit_scores":0,"negative_margin":0,"retirement_candidates":0,"stale_watermarks":0}}`
- 보류 아이디어:
  - 타임스탬프 네 읽기 경로의 계약 테스트 고정 (가치 2 / 위험 1 / 작업량 S)
  - 액션 센터의 퍼블리시 게이트 평가 오류 누락 방지 (가치 3 / 위험 2 / 작업량 S)
  - contract_expiring의 만료·설정 오류 원인 구분 (가치 2 / 위험 2 / 작업량 M)
  - 고아 Contract Scope·Entitlement 전용 운영 경고 (가치 2 / 위험 2 / 작업량 M)
