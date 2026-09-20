## 2026-09-21
- 선택: XLSX 인라인 서식 문자열의 텍스트 조각을 합쳐 셀 내용 손실 막기 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: worksheet가 is>r>t를 읽고 gridOf가 직접 텍스트가 비었을 때만 조각을 빈 구분자로 합쳐 인라인 rich text 누락을 고쳤으며, 셀 전체 TrimSpace와 숫자 파서는 그대로 유지했다. 실제 XLSX ZIP→Read→deck.ParseSource 테스트에서 inline 네 경우의 실패를 먼저 확인했고, 수정 뒤 plain/shared 대조군과 차트 라벨·1200 숫자·세 열 표·공백·엔티티·rPh 제외·숨김·출처 검증, go test ./internal/docs, go test -race ./..., go vet ./..., git diff --check가 통과했다. 커밋 8739ec8; 실제 Excel/LibreOffice 출력 및 DB 연결 검증은 미실시이며 요청한 technology:completion-verification·technology:systematic-debugging·technology:test-driven-development는 도구 및 로컬 스킬 목록에 없어 적용하지 못했다.
- 보류 아이디어:
  - XLSX 생략 행의 실제 r 좌표를 출처에 반영 (3/2/M)
  - XLSX 과도한 열 번호로 생기는 대규모 할당 제한 (4/3/M)
  - e2e call()의 headers={} 기본 신원 대체 함정 제거 (2/1/S)
  - 메일 연속 댓글을 30초 지연으로 묶기 (3/3/M)
- 과제서: 채택 — is>t만 읽는 결함이 현재 코드와 실행 재현에서 확인되어 지정한 두 지점만 국소 수정했다.
