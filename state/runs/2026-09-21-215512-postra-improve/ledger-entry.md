## 2026-09-21
- 선택: POP3 본문 수신에 MaxMessageBytes 상한 적용 — 긴 단일 행과 실제 TCP·동기화 배선 검증 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 수정 전 실제 TCP에서 RETR/TOP의 상한 초과 성공 반환과 개행 없는 긴 행의 타임아웃을 재현한 뒤, 세션에 상한을 보관하고 고정 크기 버퍼의 ReadByte로 dot-unstuffing 이후 CRLF 포함 반환 바이트를 검사하며 초과 즉시 연결을 닫도록 수정했다(04b15be). 실제 Dialer 경계 테스트와 newTestApp→CreateAccount→StartSync→dialInbound→TCP RETR 통합 테스트로 정상 원문·해시·UIDL 유무 중복 방지 및 과대 본문 미저장·failed 집계를 검증했고, app→adapter 상한 전달 제거 변이도 실패함을 확인한 뒤 원복했다. 지정 POP3 race 3회·application 집중 race·make lint(gosec Issues 0)·go build/vet ./...·go test -race ./...(일부 캐시, application 58.748s)·계약 -check·git diff --check 전부 통과했으며, 요청 technology 스킬 3종/Skill 도구는 발견하지 못해 원문 절차·반환 형식 적용을 주장하지 않는다.
- 보류 아이디어:
  - CI gofmt 검사 추가 (가치 2 / 위험 1 / 작업량 S) — workflows는 범위 밖.
  - gosec 잡 -stdout 추가 (가치 2 / 위험 1 / 작업량 S) — SARIF 외 잡 로그 가시성 개선.
  - POP3 USER/PASS AuthError·TLS 프로토콜 테스트 확대 (가치 2 / 위험 1 / 작업량 M) — 이번 크기 제한 회귀 외 행렬은 보류.
  - POP3 명령 대기 중 context 취소 전파 (가치 3 / 위험 2 / 작업량 M) — 별도 실제 TCP 재현 후 판단.
- 과제서: 채택 — 현재 베이스의 Dial/retrBody가 상한을 무시하며 실제 TCP에서 작은 상한으로 결함을 재현했고 정해진 범위 안에서 수용 기준을 구현했다.
