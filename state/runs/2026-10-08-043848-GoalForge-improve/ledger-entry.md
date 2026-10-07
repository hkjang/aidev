## 2026-10-08
- 선택: 실패한 웹훅 전송이 반복 억제 창을 소비하지 않게 한다 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: Post가 전송 전에 남기는 예약을 별도 참조로 식별하고 실패 반환 시 자기 예약만 해제하여, HTTP 500·요청 생성 오류·취소된 context 이후 다음 Post가 다시 전송할 수 있게 했다. 전송 중 중복 억제, 성공 시 전송 시작 기준의 창, redaction·미설정 URL·0 창 동작은 유지하며 프로덕션 2개와 테스트 2개만 변경했다(커밋 8f8a0f3). go test ./internal/notify -count=1, -count=2, go test -race ./internal/notify -count=1 및 go test ./... -count=1 -json은 exit 0(32개 패키지 통과, 기존 pushurl=DISABLED 관련 테스트 4개 skip, procctl/testscript는 no test files), go vet ./...·go build ./...·gofmt -l ./internal/notify·git diff --check도 exit 0이며 포맷 출력 없음; 실패 해제를 끄면 최초 회귀가 다시 실패하고 소유권 검사 없이 삭제하면 오래된 예약 해제 테스트가 실패함을 확인 후 복원했다.
- 실패 재현: 수정 전 go test ./internal/notify -count=2에서 `notify_test.go:113: requests after retry = 1, want 2` (server_error) / `notify_test.go:113: requests after retry = 0, want 1` (invalid_URL 및 canceled_context). 테스트 격리 후에도 두 반복 모두 같은 실패였고, 수정 후 같은 명령은 통과했다.
- 보류 아이디어: ① AutoApproveMerges 일일 한도 — 승인 집계·감사 정책 선행 (4/3/M).
  ② git 테스트 저장소 부트스트랩 헬퍼 공유 — remote 이름 단위 환경 프로브 유지, skip 확대 금지 (3/2/M).
  ③ 옛 커밋의 자동 병합 승인 철회 — 철회 시점·감사 설계 필요 (3/3/M).
  ④ MCP activity_report의 since 예시 7d→168h 교정 — 실제 tools/list→tools/call 계약 테스트로 고정할 차선 후보 (2/1/S).
- 과제서: 채택 — 지정한 원인이 현재 코드와 일치했고, 양수 창에서 실패→성공→억제·동시 요청·가짜 시계의 오래된 예약 보호를 검증하여 지정된 네 파일 안에서 완료했다.
