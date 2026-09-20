## 2026-09-21
- 선택: 명시적 숫자 날짜의 달력 유효성을 공통 파서에서 검증 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: ParseTimeExpressions의 raw 숫자 날짜를 기존 ymd와 time.Parse로 검증하여 불가능한 날짜만 제외하고 유효한 윤일·월말·혼합 입력의 expression과 YYYYMMDD를 유지했다(3b9fcb1). 파서 표 테스트와 TempDir JSON을 catalog.Load로 읽은 실제 Server의 /mcp tools/call 테스트로 resolve_time/get_schema_context/build_sql_skeleton의 5개 날짜 타입·4개 표기에서 수정 전 실패를 확인하고 수정 후 통과했다. 지정 부분 테스트, go build ./..., go vet ./..., go test ./... 및 git diff --check 통과; 요청된 technology 부서 스킬/Skill 도구는 검색에서 찾지 못해 원문 절차·반환 형식 적용은 미확인이며 로컬 superpowers 디버깅·TDD 스킬을 대체 참고했다.
- 보류 아이디어:
  - 개발자 가이드 Go 버전·의존성·MCP 도구 수 갱신 (3/1/S): 차선 유지, 이번 범위 밖.
  - ValidateReadOnlySQL 정규식 사전 컴파일 (2/1/S): 효과 측정 벤치마크 필요.
  - CI 워크플로 추가 (3/1/S): 보호 경로로 보류.
  - cmd/* 플래그·env 파싱 테스트 (2/1/M): 별도 과제로 유지; 나머지 기존 항목도 ideas.json에 보존.
- 과제서: 채택 — 현재 코드의 검증 없는 raw 날짜 추가 및 세 도구의 공통 파서 배선이 정찰 근거와 일치했다.
