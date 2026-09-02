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
