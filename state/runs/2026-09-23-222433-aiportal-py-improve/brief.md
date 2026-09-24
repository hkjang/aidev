# 과제서 (2026-09-23) — aiportal-py

- **과제**: 법령 뷰어 렌더러가 표 뒤 일반 텍스트를 `<ht0>` 전각 블록으로 삼키는 문제 수정 (가치 3 / 위험 2 / 작업량 S)

- **왜**: `util/text_renderer.py:wrap_preformatted` 는 box-drawing 표 블록을 모으는 도중 비-box 줄이 5줄 쌓이면(`if len(gap_buffer) >= 5:`) **표를 닫는 대신** 그 줄들을 `box_buffer` 에 흡수하고 빈 줄은 버린다(85~93행 `else` 분기). 그래서 `/read_law_markdown`(api.py:1477) 이 돌려주는 텍스트에서 별표 표 뒤에 이어지는 일반 조문·비고 문장이 `white-space: pre` 인 `<ht0>` 안으로 들어가 줄바꿈 없이 가로 스크롤로 렌더되고 그 사이 빈 줄이 사라진다. 흡수는 `##`/`**제N조**` 마커가 나올 때까지 5줄 단위로 반복되므로 마커가 드문 별표 본문에서 특히 크게 번진다. 고치면 뷰어에서 표는 표대로, 문장은 문장대로 보인다.

- **수용 기준**
  1. 표 줄 뒤에 `##`·`**제…조` 마커가 없는 일반 줄이 5줄 이상 이어지는 입력에서, `</ht0>` 가 **첫 일반 줄 앞**에서 닫히고 그 일반 줄들이 태그 바깥에 **원래 순서·원문 그대로**(입력의 빈 줄 포함) 남는다.
  2. 기존 `tests/unit/test_text_renderer.py` 10건이 모두 그대로 통과한다 — 특히 짧은 간격 흡수(`test_short_gap_keeps_single_block`), 섹션 마커 분리 2건, `test_no_content_is_lost`, `test_custom_style_is_applied`.
  3. 테스트가 증명할 것: 회귀 테스트를 **현행 코드에 먼저 돌려 실패(Red)** 시킨 뒤 수정 후 통과(Green). 최소 3가지 — (a) 표 3줄 + 마커 없는 일반 6줄 입력에서 `<ht0` 블록 안에 일반 줄이 하나도 없다, (b) 같은 입력에서 입력의 모든 줄이 출력에 존재하고 빈 줄 개수가 줄지 않는다, (c) 표 → 마커 없는 5줄 이상 일반 텍스트 → 다시 표 인 입력에서 `<ht0` 블록이 2개가 된다(흡수되어 1개가 되지 않는다). 검증은 대역 없이 `wrap_preformatted` 를 실제 import 해 반환 문자열로 한다(이 모듈은 외부 의존성 없이 import 가능하다 — AST/소스 문자열 검사로 대체하지 말 것).

- **건드릴 파일**
  - `util/text_renderer.py:wrap_preformatted` — 85~93행 `if len(gap_buffer) >= 5:` 블록만. 마커 유무와 무관하게 `_flush_box()` → `result.extend(gap_buffer)` → `gap_buffer = []` 로 바꾼다(= 5줄이 쌓이면 표가 끝난 것으로 판정). 흡수(`box_buffer.extend([g for g in gap_buffer if g.strip()])`)는 이 분기에서 제거한다.
  - `tests/unit/test_text_renderer.py` — 위 회귀 3건을 `TestWrapPreformatted` 에 추가. 기존 10건은 수정하지 말 것.
  - 필요하면 `docs/TESTING.md` 의 text_renderer 항목 1줄 갱신(문서는 최소로).

- **검증 명령** (worktree 루트에서)
  - `python3 -m pytest -q -p no:cacheprovider tests/unit/test_text_renderer.py`
  - `python3 -m pytest -q -p no:cacheprovider` (직전 기준선 1075 passed, 약 2.6초)
  - `python3 -m pyflakes util/text_renderer.py tests/unit/test_text_renderer.py`
  - `git diff --check`

- **위험과 피할 것**
  - `pipeline/law/library/text_renderer.py` 는 **별개의 미사용 사본**이다(저장소 어디에서도 import 되지 않음 — grep 확인). 같이 고치거나 두 파일을 통합하지 말 것. 다만 그 사본의 같은 자리는 `if len(gap_buffer) > 2: _flush_box(); result.extend(gap_buffer)` 로 되어 있어 이번 수정 방향의 근거가 된다.
  - `api.py:1477 /read_law_markdown` 라우트·응답 형식·bare except·`traceback.format_exc()` 노출은 A-105 범위다. 건드리지 말 것.
  - `_has_section_break_marker` / `_should_flush_gap` / 박스 줄 직전의 짧은 gap 흡수 분기(72~80행) / 임계값 `5` 는 그대로 둘 것. 짧은 간격 흡수는 표 안 캡션("합계") 계약이고 기존 테스트가 잡고 있다.
  - `_BOX_DRAWING_RE`, `_HTML_TAG_RE`, `PRE_STYLE` 문자 집합을 넓히지 말 것 — 이번 결함과 무관하고 렌더 대상만 늘린다.
  - 운영자 교훈: 소스 문자열·AST 검사를 증거로 삼지 말 것. 이 모듈은 실제 호출이 가능하므로 반환값으로 증명한다.
  - **미확인**: 이번 정찰 세션에서는 `python3` 실행이 승인되지 않아 pytest 와 실측 출력을 직접 돌려보지 못했다. 위 동작 서술은 `util/text_renderer.py` 70~101행의 정적 추적 결과다. 구현자는 **먼저 현행 코드에서 회귀 테스트가 실제로 실패하는지 확인**하고, 재현되지 않으면 차선으로 넘어갈 것.
  - **미확인**: 실제 법령 별표 데이터에서 이 패턴이 얼마나 자주 나오는지(빈도)는 확인하지 못했다. 뷰어 실물 렌더 확인도 범위 밖이다.

- **차선 후보**: `docs/CURRENT_STATE_AUDIT.md:135` 의 사실과 다른 서술 정정 (가치 2 / 위험 1 / 작업량 S) — "저장소에 `tests/`, pytest 설정, CI 테스트 job이 없습니다" 라고 적혀 있으나 `tests/unit/` 에 25개 테스트 파일과 루트 `pytest.ini` 가 실제로 존재한다(CI 테스트 job 부재만 사실 — `.gitlab-ci.yml` 은 deploy stage 하나). `docs/CODEBASE_MAP.md`·`docs/DEVELOPMENT.md` 의 같은 취지 서술도 함께 현행화하고, TESTING.md 는 이미 구현된 테스트를 적고 있으므로 그 쪽에 맞춘다. 코드 변경 없음, 검증은 `git diff --check` 와 문서에 인용된 경로의 실재 확인.
