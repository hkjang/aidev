# 회차 노트 2026-10-18-073854-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:38] base pinned — main@1c8421a
- [러너 07:38] autonomy release — 

## 구현 노트
- mcp.go 의 세 스캔 루프가 `_ = rows.Scan(...)` 으로 오류를 버리고도 append 해서 빈 구조체가 MCP 응답에 섞였다. 앞선 네 회차가 깔아 둔 `rowScanner` 꼴대로 순수 함수 셋으로 뽑았다(프로덕션 1파일, 커밋 c1d2e66).
- **확신 없는 곳·검증 못 한 것**: ① 실서버 대조는 `search_employees` **한 경로만** 했다. `list_available_seats`·`get_floor_map` 은 좌석·도면 업로드가 필요해 단위테스트로만 덮었다 — 세 헬퍼가 같은 꼴이라 유추했을 뿐이다. ② 실서버에서 일으킨 것은 cached-plan 오류뿐이라 `Next()` 가 바로 false 가 되는 경우다. **"행마다 Scan 이 실패해 빈 구조체가 들어가는" 쪽은 fakeRows 단위테스트로만 증명했고 실서버에서 재현하지 못했다** — 그 쪽 pgx 경로를 실제로 밟는 수단을 찾지 못했다. ③ E2E 스위트는 돌리지 않았다(응답 계약 무변경이라 생략).
- 일부러 하지 않은 것: `keys.go`·`detection.go`·`settings.go`·`grid.go`·`dashboard.go:317` 의 남은 같은 패턴(한 회차 한 파일 규칙). `mcpToolCall` 의 오류 문장·로깅 경로(이미 isError + 일반 문장으로 올바르다 — 새로 만들면 스키마 유출 위험만 는다). 프런트·문서(계약 무변경).
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(`go test ./internal/app` 11.5초). 세 scanMCP* 의 Scan 인수 순서는 열 순서 가드 테스트가 지키고 있으니 인수를 옮기면 반드시 테스트도 같이 볼 것. 실서버로 재현하려면 POST /mcp 에도 `X-CSRF-Token` 이 필요하고(auth.go:171 이 제외하는 것은 logout 뿐), cached plan 은 **결과 타입이 실제로 바뀌는** ALTER 로만 깨진다.
- [러너 07:46] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 저장소 밖 사본에서 돌연변이로 테스트 인과를 직접 재현했다 — 오류 검사를 옛 `_ =` 꼴로 되돌리면 6건, 같은 타입 열을 맞바꾸면 열 순서 가드 2건이 붉어진다. SQL 3개와 Scan 인수 순서를 한 열씩 대조하고 세 헬퍼의 배선(mcp.go:162·174·197)도 확인했다. go build·vet·gofmt·`go test ./... -count=1` 통과, 트리 clean.
- 구현자가 의심한 ①(list_available_seats·get_floor_map 실서버 미대조)은 스키마로 메웠다 — migrations.sql:123-137 의 좌표 4열은 double precision NOT NULL, 나머지는 NOT NULL 또는 COALESCE 라서 정상 운영에서 새 isError 가 날 경로가 없다. ②③ 은 재현하지 않았다(핸들러 수준 테스트·E2E 모두 미실행).
- 승인이어도 남는 우려: MCP 클라이언트의 가시적 동작 변화다 — 조회가 중간에 끊기면 부분 목록+isError:false 대신 호출 전체가 isError 가 되고, get_floor_map 은 한 행이 깨지면 도면 응답 전체가 사라진다(점진적 열화 아님). 릴리즈 노트에 한 줄 남길 것.
- 다음 회차: 핸들러 수준 증거(executeMCPTool→isError)가 여전히 비어 있다. `_ = rows.Scan` 은 저장소에서 0건이 됐지만 keys.go:28·detection.go:468·settings.go:35·grid.go:150·dashboard.go:317 의 다른 꼴은 남았다.
- security·legal 차단 없음: 인증·권한·마이그레이션·비밀값 무변경, 수집 항목·목적·접근 통제 무변경, 오류 원문은 mcp.go:76-89 이 일반 문장으로 가려 서버 로그에만 남긴다.
- [러너 07:49] review approved — 리뷰 승인 (risk=low)
- [러너 07:49] pr created — https://github.com/hkjang/seaton/pull/47
- [러너 07:54] ci passed — 검사 2개 모두 success
- [러너 07:54] merge done — c1d2e66
- [러너 08:03] release published — v1.4.19
- [러너 08:05] assets verified — v1.4.19 자산 1개 (이전 v1.4.18: 1)
