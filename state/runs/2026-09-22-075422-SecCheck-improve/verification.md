# 검증 증거

모든 DB 명령에 이번 회차 전용 PostgreSQL 16의 TEST_POSTGRES_DSN을 설정하고 `test -n "$TEST_POSTGRES_DSN"`으로 확인했다. 실제 DB 테스트 SKIP 없음.

| 명령 | 결과 | 파일 |
| --- | --- | --- |
| go test ./internal/web -run '^TestMCPReviewReportArguments$' -count=1 -v (수정 전) | exit 1, 비문자열 30건 성공 집계·감사 이벤트 때문에 실패; 정상/권한 10 PASS | red.log |
| 같은 명령 (수정 후) | exit 0, 부모 포함 43 PASS/0 FAIL/0 SKIP | green.log |
| 같은 명령 (타입 검사만 제거) | exit 1, 동일 30건 재실패; 즉시 수정 복구 | reverted-red.log |
| go test ./internal/web -run 'TestMCPReviewReportArguments|TestReviewReportSummarisesAPeriod' -count=1 -v | exit 0, 44 PASS/0 FAIL/0 SKIP | regression.log |
| go test -race ./internal/web -run '^TestMCPReviewReportArguments$' -count=1 | exit 0, 3.128s | race.log |
| go test ./... -json | exit 0, 419 PASS/0 FAIL/0 테스트 SKIP; web 175.602s | all-tests.jsonl |
| go vet ./... | exit 0 | vet.log |
| go build ./... | exit 0 | build.log |
| git diff --check, gofmt 확인 | exit 0 | 명령 출력 |

전체 테스트에서 internal/testdb와 scripts/extract-defaults는 테스트 파일이 없어 package skip 이벤트가 나온다. 테스트 SKIP는 0건이다. 첫 fixture 시도는 생성 중 created_at 변경으로 다음 생성이 500이었으며 MCP 결함 재현으로 세지 않았다. 최종 fixture는 모든 행을 생성한 뒤 날짜를 조정했다.

프런트·PDF·전체 race·외부 취약점 게이트·원격 CI는 이번 범위에서 실행하지 않았다. 게이트 통과나 릴리즈 완료를 주장하지 않는다.
