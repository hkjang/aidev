# fix-summary (수리 1회차 → 커밋 d3fad3a)

- **문제(재현함):** 거절된 `aef822c` 의 5줄 조기 flush 가 `──── + 데이터 6행 + ────` 형식(각 행에 `│` 없음)의 별표 표를 `<ht0>` 1줄 + 평문 6줄 + `<ht0>` 1줄로 쪼갰다. 실제 `wrap_preformatted` 호출로 HEAD 2블록 vs HEAD~1 1블록 확인. 데이터 행이 `white-space: pre` 밖으로 나가 열 정렬이 무너진다.
- **고친 방법:** 비평가 권고대로 merge 분기를 되살리는 대신 **5줄 조기 flush 자체를 제거**했다. gap 은 길이 제한 없이 쌓이고, 뒤에 box 줄이 나오면 종전 72–80행이 한 표로 이어 붙이며, 그대로 끝나면 EOF 의 `_flush_box()` + `result.extend(gap_buffer)` 가 공백 줄까지 그대로 표 밖에 내보낸다(= A-118 수정). 임계값 5 · `_should_flush_gap` · 정규식은 삭제.
- **테스트:** 회귀를 계약으로 못 박던 `test_long_gap_between_two_tables_splits_them` 을 구분 신호 기준으로 재설계 — `..._keeps_one_block`(`┌─┐` 표)·`test_ruled_table_without_pipes_stays_one_block`(`───` 표, 회귀 방어)·`..._with_section_marker_still_splits`(섹션 표시 있으면 분할) 3건. 단언을 느슨하게 하지 않았다. 문서 2건도 새 동작에 맞춰 정정.
- **검증:** `python3 -m pytest -q -p no:cacheprovider` → **1081 passed**. 되돌림 확인: 원본 `a005eb9` 에 새 테스트 → A-118 3건 실패, 거절본 `aef822c` → 회귀 2건 실패. 무작위 20000건 차분 대조(vs `a005eb9`): NEW 가 OLD 보다 더 쪼개는 사례 0, OLD 가 보존한 내용을 NEW 가 잃는 사례 0, 섹션 표시 없는 입력의 차이는 전부 "표 뒤 본문이 문서 끝까지 이어지는" A-118 케이스뿐. pyflakes·`git diff --check` 통과.
- **남은 한계:** 표/후속 본문 구분 신호는 여전히 `##`·`**제n조**` 뿐 — 섹션 표시 없이 본문 뒤 또 표가 오면 한 블록으로 합쳐진다(수정 전과 동일, 새 회귀 아님). 실물 뷰어 화면·운영 법령 데이터 빈도는 이번에도 미확인.
