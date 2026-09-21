## 2026-09-22
- 선택: MCP 심의 리포트의 잘못된 JSON 타입을 필터 생략으로 처리하지 않기 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 837d1e8에서 기존 역할 검사 뒤에 from/to/department의 존재 여부와 string 타입 검사만 추가하여 명시적 null을 포함한 비문자열을 HTTP 200·JSON-RPC -32602로 거부하고 원문·result·structuredContent 및 성공 감사 기록을 남기지 않는다. 실 PostgreSQL 16·newHarness/NewServer·로그인·실 HTTP에서 두 프로토콜의 잘못된 타입 30건이 수정 전 전체 4행 집계와 성공 감사 이벤트를 반환함을 재현했고, 수정 후 통과 및 수정 제거 시 재실패를 확인했다. 정상/누락/빈값/공백/TrimSpace·REST 전체 JSON 일치·실제 행 제외·REQUESTER 권한을 포함한 회귀 44 PASS/0 FAIL/0 SKIP, 제한 race PASS(3.128s), 전용 DSN 전체 Go 테스트 419 PASS/0 FAIL/0 테스트 SKIP(web 175.602s), vet/build exit 0; 원격 CI·vulndb·프런트 검증은 하지 않았다.
- 보류 아이디어:
  - MCP 리포트 HTTP query의 도구 인자 보충 차단 (가치 3 / 위험 2 / 작업량 M) — 현 코드 경로 확인, 별도 HTTP 재현 필요.
  - 리포트 필터 변경 시 이전 응답이 최신 집계를 덮지 않도록 처리 (가치 3 / 위험 2 / 작업량 M) — 응답 취소/순서 검사 없음, 브라우저 재현 필요.
  - 서비스 timezone에 맞춘 월 시작 프리셋 (가치 3 / 위험 2 / 작업량 M) — 여러 화면 공통 계약 검증 필요.
  - 심의번호 생성일 의존성과 과거 이관 충돌 조사 (가치 3 / 위험 3 / 작업량 M) — fixture 생성 사이 created_at 변경 시 다음 생성 500 관찰, 본 과제와 분리.
- 과제서: 채택 — 역할 검사 뒤 타입 소실 경로와 실제 무필터 성공 응답을 확인했으며, 실제 감사 테이블명은 audit_events가 아닌 audit_logs여서 해당 테이블로 검증했다.
