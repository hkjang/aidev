## 2026-09-21
- 선택: DOCX 신규 가져오기에서 확정 제목과 같은 첫 H1만 본문에서 제거하기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 기존 Markdown 플래그를 titleInBody로 일반화하고 DOCX 신규 가져오기에만 기존 정확한 일치 규칙을 적용하여 본문·검색 텍스트·최초 revision의 중복 제목을 제거했습니다. 실제 docx.Build 출력의 단위 2개와 인증 HTTP 4개(보존 하위 사례 6개)에서 변경 전 실패·변경 후 성공·DOCX 활성화만 되돌린 재실패를 확인했고, 다른 제목/H2/뒤쪽 H1/파일명 정규화/240자 절단/끼워 넣기 보존과 제목만 있는 문서의 빈 문단을 검증했습니다. 전용 PostgreSQL 16에서 go test -count=1 -v ./internal/httpapi(SKIP 0), MUNI_TEST_DSN을 설정한 go test ./..., go vet ./..., gofmt -l internal/httpapi, scripts/check-webui-placeholder.sh가 통과했고 커밋은 7d406c5입니다.
- 보류 아이디어:
  - Markdown 표 마지막 셀의 이스케이프된 끝 파이프 보존 (3/2/S): 차선으로 유지, 동적 재현 미실행.
  - CI Playwright e2e 추가 (4/2/M): workflows와 계정 시드 준비는 범위 밖.
  - 워크스페이스 ZIP의 실제 한도 초과 때만 안내 (2/1/S): 정확히 2000건인 경계 별도 검증 필요.
  - PostgreSQL 백업·복구 안내와 외부 DB 예제 정합성 (3/2/S): 운영 절차 별도 검토 필요.
- 과제서: 채택 — 현재 코드와 실제 writer의 인증 HTTP 실패가 과제서의 원인을 확인하여 지정한 최소 수정을 적용했습니다.
