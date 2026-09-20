## 2026-09-21
- 선택: JSON 본문 크기 초과를 두 디코딩 단계 모두에서 413으로 응답하기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 정상 JSON 뒤 공백으로 2MiB를 넘기면 두 번째 Decode의 MaxBytesError가 400으로 분류되는 문제를 overlay 및 새 테스트로 재현하고, 기존 스트리밍 흐름에 413/request_too_large 분기 5줄을 추가했다. 실제 MaxBytesReader·json.Decoder를 거치는 18개 케이스로 정확한 한도·알 수 없는 길이·기존 400 메시지·requestId를 확인하고, 실제 createCustomer 2개 케이스로 CRM·DB 없이 초과를 거절함을 검증했으며 분기 제거 시 재실패도 확인했다. go test ./internal/platform/httpx ./internal/server, go test -race ./..., go vet ./..., go build ./..., 지정 파일 gofmt -l 및 git diff --check 모두 통과; 커밋 b497df5, 인증 미들웨어·배포 검증은 미실시.
- 보류 아이디어: 감사 화면 Frame 부제를 실제 채널값으로 맞추기 (가치 1 / 위험 1 / S)
- 보류 아이디어: security.allowed_origins 시드 행 제거 (가치 2 / 위험 1 / S)
- 보류 아이디어: ClientIP 신뢰 프록시 설정과 전달 IP 처리 (가치 4 / 위험 3 / M)
- 보류 아이디어: DealsAtRisk·Coaching 열린 딜 커서 페이징 (가치 3 / 위험 2 / M)
- 과제서: 채택 — 기준 코드와 overlay 재현 결과가 과제서와 일치하여 지정된 공용 오류 분류와 회귀 테스트만 구현했다.
