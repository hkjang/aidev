## 2026-09-29
- 선택: 스프레드시트 이어지는 슬라이드 제목의 '(계속)' 중복 방지 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: writeSheet의 무조건 접미사 추가를 기존 continued() 호출로 바꿔 CSV·TSV 파일명과 XLSX 시트명이 이미 '(계속)'으로 끝나면 뒷장 제목에 다시 붙이지 않게 했다(프로덕션 1파일, 테스트 1파일). 신규 테스트 6개 하위 사례는 실제 Read→deck.ParseSource로 세 장의 제목과 XLSX 출처 시트명 보존을 검증하며, 수정 전 실패→수정 후 통과→수정 되돌림 시 동일 실패를 확인했다. make test(Go race/vet, 웹 typecheck/build), make build, git diff --check 통과 후 산출물을 제거하고 6f41ea4로 커밋했다; PTIUM_TEST_DSN 부재로 DB 테스트는 Skip, 실서버·Docker·릴리즈는 미실행이며 웹 청크 크기 경고와 npm ci의 moderate 취약점 2건은 남아 있다.
- 실패 재현: sheetcontinuation_test.go:55: titles = ["분기 실적 (계속)" "분기 실적 (계속) (계속)" "분기 실적 (계속) (계속)"], want ["분기 실적 (계속)" "분기 실적 (계속)" "분기 실적 (계속)"] — csv/tsv/xlsx 세 하위 사례 모두 동일 실패.
- 보류 아이디어:
  - 마크다운 setext h2: YAML front matter 처리 방침 선행 필요 (2/3/S).
  - XLSX 해제 누적 바이트/CPU 예산: 사용자 경고·작업량 한계 계약 필요 (3/3/M).
  - 새 후보 — 마크다운 #뒤 공백 없는 문장의 제목 오인: 기존 간편 문법 호환성 확인 필요 (2/2/S).
  - 새 후보 — 마크다운 표의 이스케이프된 파이프 보존: 코드 span과 deck 이스케이프 계약까지 검증 필요 (3/3/M).
