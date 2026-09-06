## 2026-09-06
- 선택: 추천 질문 캐시 무한 append → 원자적 교체 (감사 A-206) (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `api.py` 의 `QuestionStore.auto_refresh_question` 이 30분마다 수집 결과를 기존 리스트에 `append` 만 해서 프로세스가 오래 살수록 캐시가 무한히 커지고 같은 질문이 갱신 횟수만큼 중복 저장됐다. 더 심각하게는 `question_recommend` 가 실패 시 `{"questions": "질문 생성 실패"}` 라는 **문자열**을 돌려주는데 루프가 이를 그대로 순회해 한 글자짜리 질문이 캐시에 쌓였고, 피드백 질문 병합 단계에서는 `str + list` 로 `TypeError` 가 나 백그라운드 태스크가 조용히 죽어 캐시가 영구히 멈췄다. 원자적 교체·중복 제거·최대 크기(`QUESTION_CACHE_MAX`, 기본 500)·`updated_at`·`sample()` 을 가진 `util/question_cache.py` 와 응답에서 리스트일 때만 질문을 꺼내는 `extract_question_list()` 를 추가하고, 갱신 루프는 앱별 수집을 `_collect_for_app()` 으로 분리해 앱 하나가 실패해도 나머지 앱과 다음 주기가 계속되게 했으며 수집 결과가 비면 기존 캐시를 유지한다. `question_store.questions` 는 스냅샷 property 로 남겨 읽기 경로 계약을 유지했다. `api.py` 는 kiwipiepy 미설치로 import 할 수 없어 캐시 모듈은 단위 테스트로, 갱신 루프는 AST 검사(append 금지·`cache.replace()` 존재·bare except 금지·헬퍼 import)로 검증하는 `tests/unit/test_question_cache.py` 를 추가했고, 옛 `api.py` 를 되돌려 넣어 정적 테스트 4건이 실제로 실패하는지 확인했다. `python -m pytest` 691 passed(기존 654), `python -m pyflakes .` undefined name 0건. docs 4종(CURRENT_STATE_AUDIT/API_REFERENCE/CONFIGURATION/TESTING) 갱신. 커밋 `0ab2740`.
- 보류 아이디어: A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
- 보류 아이디어: A-105 후속 — 공통 오류 응답 model 도입과 traceback 노출 제거(A-106 연계) — 가치 4 / 위험 3 / L
- 보류 아이디어: A-102/A-206 후속 — 추천 질문 캐시를 프로세스 간 공유(외부 cache 또는 sticky routing) — 가치 3 / 위험 3 / M
- 보류 아이디어: `util/extract_minor.py` 의 `except Exception: return e` (예외 객체 반환) 정리 및 테스트 — 가치 3 / 위험 2 / S
- 보류 아이디어: A-108 Confluence 그래프의 `retrieve → llm_response → END` 명시적 edge 추가 — 가치 3 / 위험 3 / S (LangGraph 실행 확인 필요)
