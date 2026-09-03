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

## 2026-09-03
- 선택: 외부 HTTP 호출 timeout 누락 일괄 수정 (감사 A-107) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `requests` 는 timeout 이 없으면 무한 대기하는데, FastAPI 라우트와 LangGraph 노드가 이 호출을 동기로 하므로 상대 서비스가 멈추면 worker 가 그대로 묶인다. 설정 모듈에 의존하지 않는 `util/http_timeouts.py`(DEFAULT 5/30s, LLM 5/300s, FILE 5/600s, 환경변수로 조정 가능)를 추가하고 timeout 이 없던 호출 53곳(api.py, service/*, util/*, pipeline/*, config/eval_class.py)에 용도별 값을 지정했다. AST 로 저장소 전체의 `requests`/`Session` 호출을 검사해 timeout 누락·`timeout=None` 재발을 막는 `tests/unit/test_http_timeouts.py` 를 추가했고, `python -m pytest` 450 passed(기존 352 → 450), `python -m pyflakes .` undefined name 0건 확인. docs 3종(CURRENT_STATE_AUDIT/TESTING/CONFIGURATION) 갱신. 커밋 `be6f163`.
- 보류 아이디어:
  - A-104 Milvus filter 표현식 직접 조립 → 안전한 expression builder + 입력 검증 — 가치 4 / 위험 3 / M
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S
  - A-107 후속: 재시도·circuit breaker·공용 requests.Session 도입 — 가치 3 / 위험 3 / M

## 2026-09-03 (2회차)
- 선택: Milvus filter 표현식 주입 방지용 expression builder (감사 A-104) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: session id, filename, page, doc id, space key 등 외부 입력이 f-string 으로 Milvus 표현식에 그대로 들어가, filename 이 `a' or session_id != '` 이면 다른 세션 파일까지 삭제되는 구조였다. field 이름을 식별자로 제한하는 `field()` 와 `\`/`'`/`"`/개행을 escape 하는 `quote()`, `eq/ne/lt/in_/and_` 를 가진 `util/milvus_expr.py` 를 추가하고 조립 지점 전부(util/search_module, milvus_collection, milvus_confluence, milvus_batch, pipeline/kai·kcb·law)를 교체했다. 요청 body 의 `req.key_field` 가 field 이름 자리에 들어가던 경로와, `f"...'{a}'" + f"and ..."` 로 이어 붙여 `'value'and` 처럼 공백이 빠지던 버그 2곳도 함께 고쳤다. 기존 코드가 모두 문자열 literal 비교였으므로 builder 도 값을 항상 문자열로 인용해 의미를 바꾸지 않는다. 단위 테스트(`test_milvus_expr.py`)와 저장소 전체 AST 회귀 테스트(`test_milvus_expr_usage.py`)를 추가해 `python -m pytest` 574 passed(기존 450), `python -m pyflakes .` undefined name 0건 확인. docs 4종(CURRENT_STATE_AUDIT/TESTING/MILVUS_SEARCH/SQL_SECURITY) 갱신. 커밋 `53ce1f7`.
- 보류 아이디어:
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - A-105 오류 응답 계약 통일 (except block 의 미할당 `response` 포함) — 가치 4 / 위험 3 / L
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S
  - A-104 후속: collection name allowlist 와 field별 타입 검증 — 가치 3 / 위험 2 / M

## 2026-09-04
- 선택: collection_script 의 import-time 컬렉션 생성·삭제 제거 (감사 A-006) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `collection_script/` 의 여러 스크립트가 모듈 최상위에서 Milvus 에 연결하고 컬렉션을 만들거나 지웠다. 특히 `api.py` 의 `/create_filestorage_collection` 이 라우트 안에서 `create_collection_FILE_STORAGE` 를 import 하므로 라우트 호출만으로 최상위 `create("FILE_STORAGE")` 가 실행됐고, `delete_collection.py` 는 import 시점에 `KCBLAW_VIEW_BOX` 를 drop 했다. 호출 시점에만 연결하는 `connect()` 와 파괴적 작업 확인용 `confirm()` 을 가진 `collection_script/common.py` 를 추가하고, 최상위 실행(FILE_STORAGE·CONFLUENCE·KAI·KCBLAWVIEW·delete_collection)을 argparse 기반 `main()` 과 `__main__` 가드로 옮겼다. 하드코딩된 Milvus 주소 3곳(192.168.120.99×2, 192.168.116.99)을 `connect()` 로 교체하고, KAI 스크립트에 중복 정의된 컬렉션 목록을 `configs.local_variable.COLLECTION_LIST_KAI` 로 통일했으며, `delete_collection` 은 컬렉션 이름을 CLI 인자로 받고 `--yes` 없이는 확인을 요구한다. `api.py` 가 쓰는 `create`/`create_law_detail`/`create_law_viewer` signature 는 유지했다. AST 만 쓰는 `tests/unit/test_collection_script_side_effects.py`(import 시점 부작용 호출·최상위 실행 구문·하드코딩 주소·삭제 확인 절차)를 추가했고, 옛 코드를 되돌려 넣어 실제로 실패하는지 확인했다. `python -m pytest` 611 passed(기존 574), `python -m pyflakes .` undefined name 0건. docs 5종(CURRENT_STATE_AUDIT/CODEBASE_MAP/INSTALL/TROUBLESHOOTING/TESTING) 갱신. 커밋 `8ae2a04`.
- 보류 아이디어:
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
  - A-101 워크플로 모듈 전역 `original_question` 제거 → GraphState 로 전달 (동시 요청 간 질문 오염) — 가치 4 / 위험 3 / M
  - A-105 오류 응답 계약 통일 (except block 의 미할당 `response` 포함) — 가치 4 / 위험 3 / L
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S
