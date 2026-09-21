## 2026-09-22
- 선택: 미선정 — 필수 technology 스킬 부재로 구현 선행 조건 차단 (가치·위험·작업량 미평가)
- 결과: 변경없음
- 요약: technology:completion-verification, technology:systematic-debugging, technology:test-driven-development를 제공하는 Skill/skills.list/skills.read 도구와 해당 로컬 원문을 찾지 못했다. 다른 패키지의 유사 스킬로 대체하지 않고 필수 스킬 부재 시 중단하라는 개발 지침에 따라 코드 변경·테스트·린트·빌드·커밋을 시작하지 않았으며, git status --short는 비어 있었다. 기존 후보 12개의 평가·상태를 유지하고 기록 날짜와 차단 메모만 갱신했으며 신규 후보 선정은 수행하지 않았다.
- 보류 아이디어:
  - TestRetentionReportsEachUnattendedPass 타이밍 플레이크 원인 조사 (가치 2 / 위험 1 / 작업량 S)
  - 관리자 가이드 4.1 역할 표를 라우터 순회 테스트로 고정 (가치 2 / 위험 2 / 작업량 M)
  - Maintenance.Run의 30분 running 복구 UPDATE 실패 로깅 (가치 1 / 위험 1 / 작업량 S)
  - visitor-search의 ID·부서 일치가 기간 무관인 점을 문구에 나누어 적거나 서버 조회에 기간을 넣는다 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 구현 과제가 아닌 정찰 차단 기록으로 수용하며, 필수 스킬 제공 후 정찰 재실행이 필요하므로 차선 후보를 임의로 선정하지 않았다.
