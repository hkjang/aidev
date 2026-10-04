## 2026-10-05
- 선택: 자산 관계 생성·삭제의 UUID 입력을 기존 canonicalUUID 계약으로 검증 (가치 3 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 생성 source·target과 삭제 relationID를 SQL 전에 기존 helper로 검증해 비UUID·urn·중괄호·32자를 각각 400 INVALID_RELATION/404 RELATION_NOT_FOUND로 거절하고, 대문자 36자는 저장·감사까지 소문자로 정규화했다(79f1889, 프로덕션 1파일·테스트 1파일). 실제 Runtime·라우터·세션/API key를 쓰는 새 테스트 4개로 두 입력 위치의 독립 거절, 관계/감사 무변경, 대문자 생성·삭제·재삭제, 빈 필드·잘못된 JSON 및 기존 409/404를 검증했고, 수정 전 두 DB의 실패와 수정 제거 시 SQLite 재실패도 확인했다. SQLite `go test ./... -count=1`(httpapi 30.384s), PostgreSQL `POSTGRES_CONTAINER=invenqor-impl-20261005-063728-uuid POSTGRES_PORT=55516 ./scripts/test-postgres.sh -count=1`(httpapi 68.456s), `go vet ./...`, 서버 빌드, 변경 파일 `gofmt -l`(빈 출력)이 모두 통과했으며 상세 명령·환경 재시도는 verification.md에 기록했다.
- 실패 재현: 수정 전 PostgreSQL 새 테스트 실제 출력 두 줄(콘솔 brace_wrapped 삭제; 같은 외부 경로도 실패):
  `asset_relation_validation_test.go:172: relation response = 200 "", want 404 RELATION_NOT_FOUND: {"deleted":true}`
  `asset_relation_validation_test.go:175: delete audits = 1, want unchanged 0`
- 보류 아이디어:
  - 트랜잭션 안 internalError의 SQLite 교착 (가치 4 / 위험 3 / M): 실제 배선으로 오류 재현 수단 필요.
  - 목록 핸들러 rows.Err() 검사 (가치 3 / 위험 1 / M): 실제 Runtime에서 반복 중 실패 재현 필요.
  - 관계 조회 페이지네이션 (가치 3 / 위험 2 / M): 공개 응답과 웹 소비자 계약까지 별도 검토.
  - 관계 삭제의 부모 assetID 소속 제한 (가치 4 / 위험 2 / M): incoming/outgoing 부모 정책부터 확정; 이번에는 변경하지 않음.
- 과제서: 채택 — 현재 핸들러·라우팅·스키마가 근거와 일치했고, 미재현이던 방언 차이와 감사 부작용을 수정 전 실제 요청으로 확인했다.
