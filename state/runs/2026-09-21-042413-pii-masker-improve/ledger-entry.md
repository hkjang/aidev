## 2026-09-21
- 선택: 재기동 후 미완료 job의 잔여 결과 파일 다운로드 차단 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 완료 기록이 있는 작업만 결과 경로를 복구하고 HTTP 조회·이력·결과의 공개 조건을 completed && OutputPath != ""로 맞췄으며, 미완료 작업과 결과 없는 작업의 오래된 다운로드 URL을 제거했다. 디스크 fixture→app.New→실제 HTTP 리스너 통합 및 저장소 회귀 테스트로 수정 전 실패와 수정 후 통과를 확인했고, 두 번째 로드·시간 유지·잔여 파일 보존·완료 GET/HEAD/Range·404 보안 헤더를 검증했다. gofmt·git diff --check, go test -count=1 ./internal/jobs ./internal/httpapi, go test -count=1 ./..., go vet ./..., go build ./..., go test -race -count=3 ./internal/jobs ./internal/httpapi 모두 통과했으며 요청한 technology 스킬 3종은 도구/로컬 검색에서 찾지 못해 형식 미확인이다.
- 보류 아이디어:
  - internal/config Load 경유 환경변수 정규화·기본값 테스트 (2/1/S)
  - gorilla/mux 405 응답 Allow 헤더 (1/2/S)
  - 동기 슬롯 대기열 메모리 상한 (3/3/M)
  - 업스트림 진단 문자열 UTF-8 절단 경계 보존 (2/1/S)
- 과제서: 채택 — 상태와 무관한 결과 경로 복구와 HTTP 공개가 현재 코드에 남아 있었고 새 회귀 테스트로 재현했다.
