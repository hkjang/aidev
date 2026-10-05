## 2026-10-06
- 선택: 자산 관계 생성의 `confidence` 를 openapi 가 선언한 `[0,1]` 범위로 검증 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `openapi.yaml:1885`(콘솔)·`:2193`(외부) 가 요청 본문 `confidence` 를 `minimum: 0, maximum: 1` 로
  선언하고 `:2578` 응답 스키마도 같은 범위를 약속하지만 `assets.go:622 createAssetRelation` 에는
  `==0 -> 1` 승격만 있고 범위 검사가 없었고, `asset_relations.confidence` 는 두 방언 모두 CHECK 가
  없어(`postgres/001_initial.sql:252` DOUBLE PRECISION, `sqlite/…:252` REAL — 직접 확인) `-5`·`42` 가
  그대로 저장돼 조회가 자기 응답 스키마를 위반하는 값을 돌려줬다. `canonicalUUID(input.TargetID)`
  직후·`ExecContext` 와 `recordAdminAudit` 앞에서 범위 밖 값을 400 `INVALID_RELATION` 으로 거절하게
  했다(프로덕션 1파일 10줄, 테스트 1파일). 두 경로가 이미 400 을 선언하므로 응답 코드 집합은
  그대로다. 검증: SQLite `go test ./... -count=1` 전체 ok(httpapi 51.629s), PostgreSQL
  `POSTGRES_CONTAINER=invenqor-impl-20261006-0018-conf2 POSTGRES_PORT=55522 ./scripts/test-postgres.sh -run 'Relation' -count=1`
  ok(httpapi 8.605s), `go vet ./...`·`gofmt -l .` 빈 출력, `go build ./cmd/invenqor-server` 성공.
  기존 관계 테스트(409 `RELATION_CONFLICT`·404·UUID 400) 전부 그대로 통과.
- 실패 재현: 과제서가 '미확인' 으로 남긴 PostgreSQL 동작을 수정 전에 직접 돌려 확인했고, 두 방언
  모두 여덟 경우(콘솔·외부 × `-0.5`·`1.5`·`-1`·`42`) 전부 201 로 행과 감사를 남겼다. PostgreSQL 실측:
  `asset_relation_validation_test.go:277: relation response = 201 "", want 400 INVALID_RELATION: {"id":"2ff8a96a-624d-4dbd-9998-d7e7e7036402"}`
  `asset_relation_validation_test.go:282: create audits after rejected create = 1, want unchanged 0`
  SQLite 도 같은 줄로 실패(`:277 … = 201 "", want 400 INVALID_RELATION`). 경계값 테스트는 현재 동작을
  고정하는 쪽이라 수정 전에도 통과했다.
- 보류 아이디어:
  - 관계 삭제를 경로의 부모 assetID 에 속한 관계로 제한 (가치 4 / 위험 2 / M): `assets.go:675` 가
    `relationID` 만으로 UPDATE 해 자산 A 의 URL 로 자산 B 의 관계를 끝낼 수 있다. incoming/outgoing
    부모 정책 확정이 선행 — 이번 1순위가 성립해 차선으로 남겼다.
  - 명시적 `confidence: 0` 이 1 로 승격되는 동작 (가치 2 / 위험 2 / S): 이번 범위 검증이 머지됐으니
    이제 `*float64` 로 '없음' 과 0 을 구분하는 별도 회차가 가능하다. 현재 동작은 테스트로 고정됨.
  - 트랜잭션 안 `internalError` 의 SQLite 교착 (가치 4 / 위험 3 / M): 프로덕션 배선으로 오류를 터뜨릴
    재현 수단이 여전히 없다. 다섯 회차 연속 미선택.
  - 목록 핸들러 `rows.Err()` 검사 (가치 3 / 위험 1 / M): `assetHistory`·`assetRelations` 에 여전히 없으나
    반복 중 실패를 재현할 수단이 없어 수용 기준을 쓸 수 없다.
  - `classification.go:425 reviewProposedRelation` 의 relationID 원문 UPDATE (가치 3 / 위험 1 / S):
    `canonicalUUID` 미적용 경로 중 한 곳만 떼면 S 로 끝난다 — 다음 회차 가장 익은 후보.
- 과제서: 채택 — 지목한 행·헬퍼·검증 명령이 모두 그대로 맞았고, '미확인' 으로 남긴 PostgreSQL 의
  범위 밖 201 을 수정 전에 직접 재현해 추론을 실측으로 바꿨다. 제안 포트 55521 도 비어 있었다.
