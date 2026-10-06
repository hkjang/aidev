## 2026-10-07
- 선택: 제안 관계 심사(`POST /api/v1/assets/relations/{relationID}/{decision}`)의 relationID 를 기존 canonicalUUID 계약으로 검증 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `openapi.yaml:1677` 이 `relationId` 를 `format: uuid` 로 선언하고 `:1685-1688` 이 200/400/404 만 약속하는데 `classification.go:425 reviewProposedRelation` 은 `chi.URLParam` 값을 날것으로 `UPDATE asset_relations ... WHERE id = $6` 에 넘겼다. `decision` 검사 직후·`decodeJSON`/`ExecContext`/`recordAdminAudit` 앞에서 `canonicalUUID` 로 통과시키고 실패 시 404 `PROPOSAL_NOT_FOUND` 로 반환하게 했다(프로덕션 1파일, 테스트 1파일 신규). 이후 SQL·감사에는 정규화된 id 만 쓴다. 검증: SQLite 전체 `go test ./... -count=1` ok(httpapi 28.852s), PostgreSQL 전체 `POSTGRES_CONTAINER=invenqor-impl-20261007-0427-full POSTGRES_PORT=55532 ./scripts/test-postgres.sh -count=1` ok(httpapi 74.122s), `go vet ./...`·`gofmt -l .` 빈 출력, `go build ./cmd/invenqor-server` 성공(생성된 바이너리는 커밋 전 삭제).
- 실패 재현: 과제서가 '미확인(추론)' 으로 남긴 PostgreSQL 동작을 수정 전에 직접 돌려 확인했고, 추론보다 나빴다 — 두 철자는 500 이고 나머지 두 철자는 **실제로 제안을 심사해 버렸다**:
  `classification_relation_review_test.go:80: review response = 500 "INTERNAL_ERROR", want 404 PROPOSAL_NOT_FOUND` (not_a_uuid_at_all, urn_prefixed / approve·reject 모두)
  `classification_relation_review_test.go:80: review response = 200 "", want 404 PROPOSAL_NOT_FOUND: {"status":"active"}` (brace_wrapped) 및 `:86: relation status = "active", want unchanged "proposed"` (unhyphenated)
  `classification_relation_review_test.go:132: audit resource_id = "35FA5F48-1827-4895-BC8F-CD5FC64C7917", want canonical "35fa5f48-..."` (대문자 36자: PostgreSQL 은 200 이지만 감사에 비정규형 저장)
  SQLite 수정 전: 네 철자는 모두 이미 404 라 통과했고 대문자만 실패 — `:113: upper-case approve = 404, want 200` (방언 차이 자체가 결함).
- 보류 아이디어:
  - 관계 삭제를 경로의 부모 assetID 에 속한 관계로 제한 (가치 4 / 위험 2 / M): `assets.go:665` 가 relationID 만으로 UPDATE 해 자산 A 의 URL 로 자산 B 의 관계를 끝낼 수 있다. incoming/outgoing 부모 정책 확정이 선행 — 세 회차 연속 차선.
  - `canonicalUUID` 미적용 나머지 id 경로 점검 (mcp.go·users.go·api_keys.go·agents.go) (가치 3 / 위험 2 / M): classification.go 는 이번에 메웠다. 남은 네 파일의 현황은 아직 미확인이라 먼저 세어 보고 쪼개야 한다.
  - 명시적 `confidence: 0` 이 1 로 승격되는 동작 (가치 2 / 위험 2 / S): `*float64` 로 '없음' 과 0 을 구분해야 하고 기존 테스트 기대값도 함께 바꿔야 한다.
  - 트랜잭션 안 `internalError` 의 SQLite 교착 (가치 4 / 위험 3 / M): 프로덕션 배선으로 오류를 터뜨릴 재현 수단이 여전히 없다. 일곱 회차 연속 미선택.
  - 목록 핸들러 `rows.Err()` 검사 (`assetHistory`·`assetRelations`) (가치 3 / 위험 1 / M): 반복 중 실패를 재현할 수단이 없어 '테스트가 먼저 실패해야 한다' 를 만족시킬 수 없다.
- 과제서: 채택 — 지목한 행·헬퍼·테스트 템플릿·검증 명령·제안 포트(55533)가 모두 그대로 맞았고, '미확인' 으로 남긴 PostgreSQL 의 500 과 대문자 폴딩을 수정 전에 재현해 실측으로 바꿨다. 덧붙여 중괄호형·32자가 500 이 아니라 **조용히 심사까지 성공했다**는 것이 새로 드러났다.
