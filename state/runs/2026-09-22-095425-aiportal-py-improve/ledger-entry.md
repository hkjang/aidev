## 2026-09-22
- 선택: 긴 법령 별표·부칙의 분할 Markdown 들여쓰기 보존 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: _split_long_record의 첫/후속 Markdown 본문에 기존 _process_indentation을 재사용하고 content·분할 알고리즘은 유지했다(커밋 8a11ce7). 실제 XMLParserFactory→SourceDocumentBuilder→JSONOutputFormatter 회귀는 수정 전 들여쓰기 2건 실패·25건 통과에서 수정 후 27건 통과했고, 전체 1075 passed(2.56s), 지정 pyflakes와 git diff --check도 통과했으며 수정 전후 Markdown 외 source 전체 필드·seq_map·RAG 전체 필드 동일성을 확인했다. 요청된 technology 스킬 3종 및 Skill 도구는 발견하지 못해 적용했다고 간주하지 않으며, 운영 뷰어·Milvus 적재·발생 빈도는 미검증이다.
- 보류 아이디어: 법령 일반 XML fixture·formatter 회귀 확대 (가치 3 / 위험 1 / 작업량 M)
- 보류 아이디어: CODEBASE_MAP·CURRENT_STATE_AUDIT·DEVELOPMENT 테스트 부재 서술 정정 (가치 2 / 위험 1 / 작업량 S)
- 보류 아이디어: process_feedback_single policy token 부재 사전검사 (가치 2 / 위험 2 / 작업량 S)
- 보류 아이디어: executor 피드백 배치 shutdown 시 행 사이 중단 요청 (가치 2 / 위험 2 / 작업량 M)
- 과제서: 채택 — 현 코드에서 긴 별표·부칙의 들여쓰기 누락이 재현되어 지정 Markdown 생성 지점만 수정했다.
