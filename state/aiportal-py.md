# aiportal-py 자율 개선 기록

## 2026-09-02
- 선택: mask_sensitive_data NameError 수정 + pytest 테스트 기반 도입 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `util/sensitive_data_masker.py` 의 `mask_sensitive_data()` 가 dict/list 입력에서 존재하지 않는 `_mask_sensitive_data()` 를 호출해 NameError 로 실패하는 실제 버그를 재현·수정했다. 동시에 저장소에 전혀 없던 테스트 기반(`pytest.ini`, `tests/conftest.py`, `requirements-dev.txt`)을 만들고 외부 의존성 없는 순수 unit 테스트(sensitive_data_masker, parse_promptmanager, dpe_merge, text_renderer)와 저장소 전체 Python AST parse 테스트를 추가했다. `python -m pytest` 로 177 passed 확인, README/docs/TESTING.md 를 실제 상태에 맞게 갱신. 커밋 `8122984`.
- 보류 아이디어:
  - GitLab CI 에 deploy 앞단 test stage 추가 — 가치 4 / 위험 3 / L (러너가 shell 태그 기반이라 python 실행 환경 불확실, 실패 시 배포 차단 위험)
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - feedback loop 의 undefined `ENV` 수정 (감사 A-003) — 가치 4 / 위험 2 / S
  - `pipeline/law/library/xml_parser_core.py`, `json_formatter.py` 단위 테스트 (fixture 필요) — 가치 3 / 위험 1 / M
  - `util/extract_minor.py` 의 `except Exception: return e` (예외 객체 반환) 정리 및 테스트 — 가치 3 / 위험 2 / S

## 2026-09-02 (2회차)
- 선택: 정의되지 않은 이름으로 인한 런타임 오류 일괄 수정 + 정적 회귀 테스트 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: pyflakes 로 저장소를 훑어 NameError/UnboundLocalError 를 일으키는 실제 결함 전부(감사 A-003 undefined `ENV`, A-004 `Collection(collection)`, api.py 의 `payload` 선참조·typing import 누락, kai/cf workflow 의 `guard_rail_completions_*` import 누락, chunker 의 `traceback` import 및 `text_chunking(file)` 인자 누락, truncate_table 의 rows/collection_name 참조)를 수정했다. 함께 A-005(`close_all_db_connections.closeall()` → 함수 호출)와 A-109(finally 가 try 지역 `conn` 참조하는 8곳에 `conn = None` 초기화, `fetch_unprocessed_feedback` 의 중복 `conn.close()` 제거)도 정리했다. 회귀 방지로 운영 의존성 없이 AST 만 쓰는 `tests/unit/test_undefined_names.py`(pyflakes 기반, 미설치 시 skip)와 `tests/unit/test_cleanup_paths.py`(finally/종료 경로 검사)를 추가하고 requirements-dev 에 pyflakes 를 넣었다. `python -m pytest` 352 passed(기존 177 → 352), `python -m pyflakes .` 의 undefined name 0건 확인. docs/TESTING.md·CURRENT_STATE_AUDIT.md 를 실제 상태로 갱신. 커밋 `89cf479`.
- 보류 아이디어:
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M (기동 동작 변경이라 운영 검증 필요)
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S
  - `pipeline/law/library/xml_parser_core.py`, `json_formatter.py` 단위 테스트 (fixture 필요) — 가치 3 / 위험 1 / M
  - `util/extract_minor.py` 의 `except Exception: return e` (예외 객체 반환) 정리 및 테스트 — 가치 3 / 위험 2 / S
