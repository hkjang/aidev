## 2026-10-05
- 선택: ParseCriterion과 CLI·MCP의 완료 조건 저장 계약을 실제 경로로 고정한다 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 신규 테스트 파일 3개(프로덕션 변경 0개)로 ParseCriterion의 공백·대소문자·구분자·잘못된 구문 계약을 고정하고, 같은 원문을 실제 CLI run dispatch와 MCP Serve tools/call goal_set으로 저장한 뒤 실제 SQLite CurrentGoal의 Type/ExpectedValue/RequiredKind를 확인했다. 두 표면 모두 변경 사유를 제공하고 유효 조건 뒤에 빈 kind 또는 미등록 kind를 섞어 실패시킨 뒤 기존 활성 목표의 ID·Version·Criteria를 포함한 전체 값 보존을 확인했으며, 미등록 kind를 허용하는 구문 파서와 이를 거절하는 저장 정책의 책임을 유지했다. go test ./internal/model -count=1, go test ./internal/model ./internal/mcp ./cmd/goalforge -count=1, go test ./internal/observer ./internal/store/sqlite -count=1, go test ./... -count=1 -json 모두 exit 0(전체 32개 테스트 패키지 통과, 기존 push 제한으로 테스트 4개 skip), go vet ./... 및 go build ./... exit 0, gofmt -l ./cmd ./internal 무출력, go mod tidy 후 go.mod/go.sum 드리프트 없음을 확인했다.
- 실패 재현: 못 함 — 확인된 테스트 공백을 보호하는 과제이며 신규 테스트는 기존 프로덕션 코드에서 첫 실행부터 통과했다(`ok github.com/goalforge/goalforge/internal/model 0.002s`; 이후 MCP 3.279s, CLI 7.165s). 결함 수정이나 RED→GREEN 재현을 주장하지 않으며, 정책을 바꾸거나 인위적 실패를 만들지 않았다.
- 보류 아이디어: ① AutoApproveMerges 일일 한도 — 집계·감사 정책 설계 필요 (4/3/M).
  ② git 테스트 부트스트랩 헬퍼 공유 — 환경 프로브 동작을 함께 확인할 별도 과제 (3/2/M).
  ③ 옛 커밋의 자동 병합 승인 철회 — 철회 시점·감사 설계 필요 (3/3/M).
  ④ procctl·testscript 직접 테스트 — OS별 실행·신호 안정성 검증 필요 (2/1/M).
- 과제서: 채택 — 기존 하네스 변경 없이 테스트 파일 3개만으로 지정된 실제 입력·저장·실패 보존 경로를 모두 검증했다.
