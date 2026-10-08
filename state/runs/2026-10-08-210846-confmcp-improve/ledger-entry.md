## 2026-10-08
- 선택: replace_text 공백 보존과 승인 인자 결합 검증 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 검색 문자열의 앞뒤 공백이 제거되어 다른 위치가 치환되는 문제를 수정하고, replace_text의 find/body 공백 변경도 승인 해시에 반영했다. 실제 cmd/server·PostgreSQL·모의 Confluence를 통한 승인→쓰기→재조회 6사례에서 수정 전 실패, 수정 후 통과, 원래 코드 재빌드 후 재실패를 확인했다. DB 포함 전체 Go 테스트·build·vet, 전체 e2e(34.022초), npm check/build, Docker 빌드 및 이미지 내부 4개 HTTP 경로 점검이 통과했으며 4b31c74로 커밋했다.
- 실패 재현: `e2e_test.go:416: stored body = "<p> dog | cat |cat</p>", want "<p>cat| dog |cat</p>"`; `e2e_test.go:435: changed body whitespace accepted: code="", isError=false`
- 보류 아이디어:
  - MCP 세션 종료의 인증과 소유자 검증 (가치 4 / 위험 3 / 작업량 M): DELETE가 세션 ID만으로 상태를 바꾸므로 계약과 사용자 분리를 검증한다.
  - MCP 정수 인자의 소수·접미 문자열 거부 (가치 3 / 위험 2 / 작업량 S): toInt의 절삭·숫자 접두사 수용을 호출별 계약에 맞게 좁힌다.
  - MCP 요청 본문 크기 초과의 명시적 거부 (가치 3 / 위험 2 / 작업량 S): 8MiB LimitReader 뒤의 초과 데이터를 구별하는 테스트를 추가한다.
  - 문서의 존재하지 않는 e2e 실행 명령 수정 (가치 3 / 위험 1 / 작업량 S): README·관리자 가이드의 hack/e2e.sh 안내를 실제 실행 경로로 바꾼다.
