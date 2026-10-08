## 2026-10-08
- 선택: 전체 내보내기의 세션 전용 인증과 완결 판정을 OpenAPI·API 문서에 맞추기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 실제 인증 정책은 session-only인데 공통 operation이 people:read와 전역 Bearer를 안내하는 원인을 확인해, openapi.go에 orbit_session 쿠키 스키마와 export 전용 보안 override·완결 설명을 추가하고 docs/API.md에 영어 안내 및 가상 계정의 정상/people 실패 JSON 예시를 넣었다(프로덕션 1파일, 총 3파일, 커밋 a5de472). TestOpenAPIExportContract는 New(nil, "test-version", "test", "test")의 실제 GET /openapi.json 응답을 검사하며 수정 전 실패→수정 후 통과→원본 openapi.go 복원 시 재실패→수정 복원 후 통과를 확인했다. go test -race -count=1 ./internal/server -run '^TestOpenAPIExportContract$' 및 go test -race -count=1 ./... PASS, go vet ./...·go build ./...·git diff --check 종료 0, gofmt 무출력, Python은 2 JSON examples parsed; 기존 DB 시험은 ORBIT_TEST_DATABASE_URL 미설정으로 SKIP이고 exportData/기존 시험 단언과 문서의 수동 대조만 했으며 실제 인증 통합·실 DB 내보내기를 실측했다고 주장하지 않는다.
- 실패 재현: `openapi_test.go:44: sessionCookie = map[], want map[in:cookie name:orbit_session type:apiKey]` / `openapi_test.go:55: export security = [], want [map[sessionCookie:[]]]`
- 보류 아이디어: orbit_list_memories의 DB limit 적용 — 성능 실측 후 호출자 범위로만 검토 (가치 2 / 위험 2 / 작업량 S)
- 보류 아이디어: decodeJSON의 뒤따르는 JSON·쓰레기 거부 — 전체 입력 수용 계약 변경이라 별도 검토 (가치 2 / 위험 2 / 작업량 S)
- 보류 아이디어: auth.go:userByAPIKey의 rows.Err 누락 — fail-closed이며 보호 경로 (가치 2 / 위험 3 / 작업량 S)
- 보류 아이디어: web/src/pages 통합 시험 기반 — UI 제약 검증과 별도 범위 필요 (가치 2 / 위험 2 / 작업량 M)
- 과제서: 채택 — 지정한 문서/인증 불일치와 exportData 계약이 현재 코드와 일치했고, nil store의 실제 라우터 시험도 실행으로 성립해 수용 기준을 모두 구현했다.
