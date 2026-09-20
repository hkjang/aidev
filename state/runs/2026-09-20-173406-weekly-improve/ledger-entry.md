## 2026-09-20
- 선택: 공유 DSN을 직접 여는 통합 시험 12개를 기존 스크래치 DB 하네스로 격리 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 네 파일의 12개 시험이 createScratchDatabase 반환 DSN을 사용하고 실제 pool의 SELECT current_database()로 입력 공용 DB와 다른지 확인하도록 바꿨으며, 인증 시험의 실제 pool.Close() 장애 재현과 기존 fixture·skip·제품 주장을 유지했다. 이번 회차 전용 pgvector/pgvector:pg16에서 기존 배선이 검증문으로 12건 실패하는 것을 확인한 뒤 대상 -count=2가 24회 통과(skip 0), 실제 DB 전체 go test ./... -count=1(144.764초), DB 미설정 전체 시험(2.928초), go vet ./..., go build ./..., guard-check --changed main(5개) 및 diff 검사를 통과했고 종료 후 scratch·template DB 잔여 0개를 확인했다. 커밋 e909ca0이며 회사 스킬 technology:completion-verification·systematic-debugging·test-driven-development는 도구와 로컬에서 찾지 못해 원문 절차·반환 형식은 적용 미확인이다.
- 보류 아이디어:
  - weekly_u_* 청소 확대 — 이름 파서와 실제 DB 생존·정리 시험을 함께 검증 (가치 2 / 위험 2 / 작업량 S)
  - authz-check 실행 시간·동시 실행 금지 문서 보완 (가치 2 / 위험 1 / 작업량 S)
  - README·MCP 도입부 읽기 전용 설명 정정 (가치 2 / 위험 1 / 작업량 S)
  - 메일 발송 목록 동률 정렬에 큐 종류 기준 추가 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 현재 코드의 직접 접속 12곳을 확인했고 전용 PostgreSQL 컨테이너를 확보해 실제 DB 반복 실행과 격리 검증까지 완료했다.
