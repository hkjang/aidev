## 2026-09-21
- 선택: 업스트림 진단 문자열을 UTF-8 경계에서 절단 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 공통 truncateString이 바이트 예산 안에서 완전한 UTF-8 접두부만 남기도록 수정하고 기존 TrimSpace·무제한·ASCII·말줄임 규칙을 유지했다. 단위 경계 테이블과 app.New→실제 HTTP 통합에서 연결 점검/마스킹 오류 detail 및 성공 debug.body의 정확한 접두부·바이트 길이·U+FFFD 부재, completed/applied_regions=0/원본 PNG 파일 파트를 검증했으며 수정 전 실패와 수정 후 통과를 확인했다. go test -count=1 ./internal/upstage ./internal/httpapi, go test -count=1 ./..., go vet ./..., go build ./..., gofmt 및 git diff --check 모두 통과했고 7511062로 커밋했다; 요청된 technology 스킬 3종은 도구·로컬 검색에서 미발견으로 절차·반환 형식 미확인이다.
- 보류 아이디어:
  - internal/config Load 경유 환경변수 정규화·기본값 테스트 (가치 2 / 위험 1 / 작업량 S)
  - gorilla/mux 405 응답 Allow 헤더 (가치 1 / 위험 2 / 작업량 S)
  - 동기 슬롯 대기열 메모리 상한 (가치 3 / 위험 3 / 작업량 M)
  - 업로드 파일명 유니코드 정규화(NFC) (가치 2 / 위험 2 / 작업량 S)
- 과제서: 채택 — 바이트 인덱스 절단이 현재 코드에 남아 있었고 실제 HTTP 세 경로의 새 회귀 테스트로 진단 손상을 재현했다.
