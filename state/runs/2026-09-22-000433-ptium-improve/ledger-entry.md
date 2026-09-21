## 2026-09-22
- 선택: XLSX에서 생략된 행의 실제 r 좌표를 슬라이드 출처에 반영 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: onScreen에서 유효한 행 r(TrimSpace→Atoi, 1..1048576)을 기존 placement에 전달해 생략 행이 있는 XLSX도 실제 원본 좌표를 인용하며, 누락·0·음수·문자열·오버플로·상한 초과는 기존 배열 index로 대체한다. 실제 ZIP→Read→deck.ParseSource의 24개 사례로 실패를 먼저 확인한 뒤 차트/표의 Sources·Items/Blocks, 숨김 경고, 빈 행, 17개 본문 행의 3장 이어쓰기, 최대 좌표를 검증했고 go test ./internal/docs, go test -race ./..., go vet ./..., git diff --check가 통과했다. 커밋 3c3493f; 요청 스킬 세 종은 도구/로컬에서 찾지 못해 미적용이며 실제 Excel/LibreOffice 출력과 DB 연결 검증은 미실시했다.
- 보류 아이디어:
  - XLSX 출처의 AAA 이상 열 이름을 올바르게 표기 (가치 2 / 위험 1 / 작업량 S)
  - XLSX 과도한 열 번호로 생기는 대규모 할당 제한 (가치 4 / 위험 3 / 작업량 M)
  - XLSX ZIP 전체 해제 크기와 불필요한 파트 읽기 제한 (가치 4 / 위험 3 / 작업량 M)
  - e2e call()의 headers={} 기본 신원 대체 함정 제거 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 배열 index만 출처에 기록하는 결함이 현재 코드와 신규 ZIP 기반 실패 재현에서 확인되어 지정된 placement 매핑만 수정했다.
