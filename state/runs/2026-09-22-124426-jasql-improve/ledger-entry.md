## 2026-09-22
- 선택: jasql-eval 상세 출력에서 컬럼·SQL 등 진단이 있는 케이스를 MISS로 표시 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 기존 세 bool 판정에 Missing 진단 유무를 추가하여 컬럼·SQL 오류도 MISS로 출력하고, 문서에 상세 상태와 요약 종료 기준의 차이를 명시했다(caf7a00). t.TempDir JSON 메타데이터·골든셋과 한 번 빌드한 실제 CLI의 7개 fixture × verbose/비verbose 테스트에서 수정 전 컬럼·SQL 케이스만 실패하고 수정 후 모두 통과했으며 정상·선택 검사 생략·테이블/지표/조인 실패·JSON 요약·기존 종료 코드를 검증했다. go test ./cmd/jasql-eval -count=1, go build ./..., go vet ./..., go test ./..., git diff --check 통과(기존 패키지 전체 테스트는 캐시); 실 Oracle exec:/rows: 재현과 요청된 technology 스킬 원문 적용은 미확인이다.
- 보류 아이디어:
  - 개발자 가이드 Go 버전·의존성·MCP 도구 수 갱신 (3/1/S): 차선 유지.
  - ValidateReadOnlySQL 거부 키워드 정규식 사전 컴파일 (2/1/S): 효과 벤치마크 필요.
  - CI 워크플로 추가 (3/1/S): 이번 보호 경로 범위 밖.
  - cmd/* 플래그·env 파싱 테스트 (2/1/M): eval 출력 테스트 외 공백 유지; 나머지 기존 항목은 ideas.json 보존.
- 과제서: 채택 — 현재 CLI 표시 조건과 공통 Missing 생산 경로가 정찰 근거와 일치하여 카탈로그·요약·종료 정책 변경 없이 구현했다.
