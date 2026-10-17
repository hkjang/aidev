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

- 릴리즈: v0.1.1 (2026-10-08, run 2026-10-08-210846-confmcp-improve)
## 2026-10-17
- 선택: DELETE /mcp 세션 종료에 인증과 소유자 검증 추가 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `internal/mcp/server.go:Handle` 의 DELETE 분기가 `PrincipalFor` 를 호출하지 않아 인증 없이 세션 ID 만으로 `closed_at` 이 설정됐고, `closeSession` 의 UPDATE 가 소유자를 보지 않아 타인의 세션도 종료할 수 있었다(관리 콘솔 활성 세션 집계 왜곡). GET·initialize 와 같은 인증을 요구하고 UPDATE 를 `WHERE id=$1 AND (user_id=$2 OR (user_id IS NULL AND username=$3))` 로 좁혔으며, 열거 오라클 방지를 위해 없는/타인 세션 모두 204 로 통일했다. 실제 기동한 confmcp + PostgreSQL + 모의 Confluence 에 HTTP 로 붙는 e2e `TestMCPSessionCloseRequiresOwner` 로 미인증·타인·본인 세 경우의 상태 코드와 `closed_at` 변화를 관리자 API(`GET /api/admin/sessions?limit=200`)로 확인했고, build·vet·DB 포함 전체 단위 테스트·전체 e2e(19건, 36.2초)가 통과해 904b775 로 커밋했다. 프로덕션 파일 1개 + e2e 1개.
- 실패 재현: `e2e_test.go:198: unauthenticated DELETE: status 204 challenge ""` (수정 전). 인증 분기만 고치고 SQL 소유자 조건을 되돌리면 두 번째 절반도 따로 재현된다: `e2e_test.go:208: another user closed alice's session`. 직접 curl 탐침으로도 확인 — 인증 헤더 없는 DELETE 뒤 `closedAt after unauthenticated DELETE: 2026-10-17T10:35:54.516706+09:00`.
- 보류 아이디어: GET /mcp 스트림이 touchSession 을 호출하지 않아 장기 SSE 세션의 last_seen_at 이 멈추고 관리 콘솔 활성 세션 집계가 작게 나온다 (3/1/S) · MCP 정수 인자의 소수·접미 문자열 거부 — args.go toInt 를 호출별 계약으로 나눠 조여야 한다 (3/2/S) · MCP 요청 본문 8MiB 초과의 명시적 거부 (3/2/S) · README.md:165·docs/guide-admin.html:758 의 존재하지 않는 hack/e2e.sh 안내 수정 (3/1/S) · internal/mcp 단위 테스트 부재 — 배선을 바꾸지 않는 범위로 좁혀야 한다 (3/1/M).
- 과제서: 채택 — 과제서의 근거(server.go:45 DELETE 분기에 PrincipalFor 없음, server.go:382-391 의 UPDATE 가 소유자 미확인)가 현재 코드와 정확히 일치했고 수용 기준 3건을 그대로 증명했다.

