## 2026-09-06
- 선택: 오류 경로의 미할당 이름 참조 수정 (감사 A-105 부분 해결) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `api.py` 의 5개 라우트(`/newchatsession`, `/getrefreshtoken`, `/history_list`, `/delete_history_session`, `/history_session`)가 `except` 에서 `response.json()` 을 호출했는데, 서비스 호출 자체가 실패하면 `response` 가 대입되지 않아 `UnboundLocalError` 가 원래 오류를 덮고 HTTP 500 이 됐다(history 계열은 성공 경로가 dict 를 반환하므로 `response` 가 있어도 `.json()` 이 없어 항상 실패). 저장소 표준인 `{"code":499, "message": traceback.format_exc()}` 로 교체하고, `/status_completions` 의 스트리밍 generator 는 `last_state` 를 `try` 밖에서 초기화하고 오류 시 종료 프레임을 실제로 `yield` 하며 `GeneratorExit`/`CancelledError` 를 삼키지 않도록 `except Exception` 으로 좁혔다. 같은 패턴인 `kcb_pipeline_milvus.remove_legacy_file` 의 `collection_name` 과 `milvus_collection.sync_collection` 의 `response` 초기화(및 `list.append` 결과를 재대입해 `data` 가 `None` 이 되던 버그, 반환 arity 불일치)도 함께 고쳤다. 회귀 방지로 기존 `tests/unit/test_cleanup_paths.py` 에 `finally` 검사와 같은 스코프 분석을 쓰는 `except` 블록 검사와 검사기 자체 self-test 2건을 추가했다(수정 전 저장소에서 7건 검출 확인). `python -m pytest` 751 passed(기존 654), `python -m pyflakes .` undefined name 0건. docs 3종(CURRENT_STATE_AUDIT/API_REFERENCE/TESTING) 갱신. 커밋 `4587666`.
- 보류 아이디어: A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
- 보류 아이디어: A-105 후속 — 공통 오류 응답 model 도입과 traceback 노출 제거(A-106 연계) — 가치 4 / 위험 3 / L
- 보류 아이디어: A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거·최대 크기 — 가치 3 / 위험 2 / S
- 보류 아이디어: A-108 Confluence 그래프의 `retrieve → llm_response → END` 명시적 edge 추가 — 가치 3 / 위험 3 / S (LangGraph 실행 확인 필요)
