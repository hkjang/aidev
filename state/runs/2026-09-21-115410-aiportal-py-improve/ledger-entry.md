## 2026-09-21
- 선택: 법령 별표·부칙의 긴 한 줄이 source 청크 길이 제한을 우회하는 오류 수정 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 긴 줄 앞의 버퍼를 먼저 확정·초기화하여 버퍼 유무와 관계없이 source content를 15,000 Python 문자 이내로 분할했다. 실제 XMLParserFactory→SourceDocumentBuilder→JSONOutputFormatter 회귀 15건은 수정 전 8 실패·7 통과에서 수정 후 모두 통과했고, 본문 문자·순서, seq 범위와 모든 해당 RAG 연결, 짧은 출력 계약을 검증했다. 전체 pytest 1071 passed(2.74s), 변경 Python 파일 pyflakes 및 git diff --check 통과; 요청된 technology 스킬 3종/Skill 도구는 발견하지 못하여 사용했다고 간주하지 않으며 운영 적재·뷰어·재색인은 검증 범위 밖이다.
- 보류 아이디어: 법령 일반 XML fixture·formatter 회귀 확대 (가치 3 / 위험 1 / 작업량 M)
- 보류 아이디어: CODEBASE_MAP·CURRENT_STATE_AUDIT의 tests·pytest.ini 부재 서술 정정 (가치 2 / 위험 1 / 작업량 S)
- 보류 아이디어: process_feedback_single policy token 부재 사전검사 (가치 2 / 위험 2 / 작업량 S)
- 보류 아이디어: executor 피드백 배치 shutdown 시 행 사이 중단 요청 (가치 2 / 위험 2 / 작업량 M)
- 과제서: 채택 — 현재 코드에서 별표·부칙 모두 지정 오류가 재현되어 분할 분기만 최소 수정하고 실제 변환 체인으로 연결을 검증했다.
