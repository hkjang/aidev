## 2026-09-29
- 선택: AI 스트림에서 조회할 수 없는 사람만 SSE 시작 전 404로 반환 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: people JOIN relationships 첫 조회의 ErrNoRows만 AI 전용 sentinel로 구분해 없는 사람·타인 소유 사람·관계 행 없는 사람을 동일한 404 not_found JSON으로 반환한다. 실제 격리 PostgreSQL(임의 포트 49761), store.Open, streamAI와 HTTP 제공자 시험 서버로 9개 회귀 시험을 검증했고, 정상/빈 ID SSE·잘못된 UUID 400·비활성 503·키/설정 누락 500을 유지하며 거부 요청의 제공자 호출이 0임을 확인했다. 수정 전 실패→수정 후 통과→수정 복원 전 재실패→최종 통과를 확인했으며 DB 포함 go test -race -count=1 ./..., 지정 집중 시험, go vet ./..., go build ./..., gofmt 무출력 및 설정 전체 컬럼 복원 확인 완료(프로덕션 1파일·테스트 1파일).
- 실패 재현: missing_person 및 other_users_person — `ai_db_test.go:120: status = 500, want 404; body = {"error":{"code":"internal_error","message":"요청을 처리하지 못했습니다."}}` (두 시험 모두 동일 출력, ai-red.log; missing_relationship도 같은 실패)
- 보류 아이디어: rows.Err 누락 여섯 목록 경로의 실제 중간 실패 재현 (3/2/M)
- 보류 아이디어: orbitAt contexts 집계 실DB 회귀 (2/1/S)
- 보류 아이디어: decodeJSON 뒤 추가 JSON·쓰레기 입력 계약 강화 (3/2/S)
- 보류 아이디어: AI 제공자 SSE 여러 data 줄·프레임 경계 시험 (2/1/S)
- 과제서: 채택 — 첫 사람 조회와 기억 키 조회가 모두 ErrNoRows를 반환함을 실제 DB로 확인했으며 지정한 첫 조회 전용 분류와 모든 수용 기준을 구현했다.
