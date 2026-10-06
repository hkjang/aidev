- 과제: 제안 관계 심사(`POST /api/v1/assets/relations/{relationID}/{decision}`)의 relationID 를 기존 canonicalUUID 계약으로 검증 (가치 3 / 위험 1 / 작업량 S)
- 왜: `openapi.yaml:1677` 이 이 경로의 `relationId` 를 `{type: string, format: uuid}` 로 선언하고 응답은 200/400/404 만 약속하는데(`:1685-1688`, 500 없음), `classification.go:421 reviewProposedRelation` 은 `chi.URLParam(request, "relationID")` 을 날것으로 `UPDATE asset_relations ... WHERE id = $6` 에 넘긴다. `asset_relations.id` 는 PostgreSQL 에서 UUID, SQLite 폴백에서 TEXT 이므로 형식이 틀린 값은 한쪽에서 SQLSTATE 22P02 → `internalError` 500, 다른 쪽에서 404 가 되어 같은 요청이 방언에 따라 다르게 끝난다. 고치면 이 저장소가 merge·split·상세/수정/삭제/history/관계 생성·삭제에 이미 적용한 `canonicalUUID` 계약의 마지막 자산 관계 경로가 메워진다.
- 수용 기준:
  1) 형식이 틀린 relationID(예: `not-a-uuid`, `urn:uuid:<36자>`, `{…}` 중괄호형, 하이픈 없는 32자)로 approve·reject 를 호출하면 두 방언 모두 **같은** 응답을 준다 — 404 `PROPOSAL_NOT_FOUND`(기존 '없는 제안' 응답과 동일). PostgreSQL 에서 500 이 사라진다.
  2) 거절된 요청은 `asset_relations` 행도 `audit_logs` 도 바꾸지 않는다(`recordAdminAudit` 가 호출되지 않는다).
  3) 선언된 형태인 **대문자 36자** relationID 는 소문자로 정규화되어 정상 심사되고, 감사 레코드의 대상 id 도 정규형(소문자)으로 남는다. 기존 동작(정상 approve/reject 200, 이미 심사된 제안 404, `decision` 이 approve/reject 아닐 때 400 `INVALID_REQUEST`)은 그대로 통과해야 한다.
  4) 테스트는 **수정 전에 실패**해야 한다. 수정 전 실제 동작(PostgreSQL 500 / SQLite 404, 대문자 요청의 방언 차이)을 돌려서 확인하고 그 출력을 verification.md 에 남길 것. — **주의: 수정 전 PostgreSQL 의 500 과 대문자 폴딩은 이번 정찰에서 재현하지 않았다(미확인, 추론).** 근거는 `asset_relations.id` 가 `migrations/postgres/001_initial.sql:245` 에서 `UUID PRIMARY KEY`, `migrations/sqlite/001_initial.sql:245` 에서 `TEXT PRIMARY KEY` 라는 것(직접 확인)과 앞선 다섯 회차에서 같은 방언 차이가 반복 실측된 것이다. 구현자가 먼저 돌려 확인할 것.
- 건드릴 파일 (프로덕션 1 + 테스트 1):
  - `server/internal/httpapi/classification.go:421 reviewProposedRelation` — `decision` 검사 직후, `decodeJSON`/`ExecContext`/`recordAdminAudit` **앞**에서 `canonicalUUID(chi.URLParam(request, "relationID"))` 로 통과시키고 실패 시 404 `PROPOSAL_NOT_FOUND` 로 즉시 반환. 이후 SQL·감사에는 정규화된 값을 쓴다. (현재 `relationID == ""` 검사는 400 `INVALID_REQUEST` 와 묶여 있는데, 빈 값은 chi 라우팅상 이 핸들러에 도달하지 않는다 — 기존 400 분기를 지우지 말고 `decision` 검사는 그대로 두는 쪽이 안전하다.)
  - `server/internal/httpapi/` 아래 새 테스트 파일(예: `classification_relation_review_test.go`) — 아래 "재사용할 헬퍼" 만 쓰고 **새 헬퍼를 중복 선언하지 말 것**.
  - **테스트 템플릿은 `internal/httpapi/classification_test.go:240 TestProposedRelationsAreReviewable` 를 그대로 베낄 것**(직접 열어 확인함). 거기에 필요한 모든 것이 있다: `newRuntime(t)`/`testServer(t, runtime)`/`authenticateInitialAdmin(t, server, runtime)` → `cookie, csrf`, `runtime.DB().Exec` 로 assets 2건 INSERT, 이어서 `INSERT INTO asset_relations(id, source_asset_id, relation_type, target_asset_id, source, confidence, derivation, status) VALUES($1,$2,'duplicate_of',$3,'inferred',0.6,'machine_identity','proposed')`(`:267-272`), 그리고 `performAuthenticatedJSON(t, server, http.MethodPost, "/api/v1/assets/relations/"+relationID+"/approve", map[string]any{"reason":"확인"}, cookie, csrf)`(`:307-311`). 이 경로는 **session 전용**이라 외부 API key 경로를 따로 테스트할 필요가 없다(라우팅은 `server.go:399-401` 콘솔 한 곳뿐 — 확인함).
  - 그 밖에 재사용할 헬퍼: `countRows`(`asset_merge_validation_test.go:41`), `errorCode`(`external_api_test.go:226`), `assetIDSpellings`(비정규형 철자 목록이 이미 있으면 재사용).
- 검증 명령:
  - 좁게: `cd server && go test ./internal/httpapi/ -run 'Relation|Proposed|Classification' -count=1` — **이번 정찰에서 실제로 돌려 3.252초 ok 확인**. 수정 전 기준선이다.
  - 전체 SQLite: `cd server && go test ./... -count=1` (httpapi 30~52초 실측)
  - PostgreSQL(**필수** — 이 과제의 핵심 증거): 루트에서 `POSTGRES_CONTAINER=invenqor-impl-20261007-0427-review POSTGRES_PORT=55533 ./scripts/test-postgres.sh -run 'Relation|Proposed|Classification' -count=1`. **포트 실측(`ss -ltn`): 55531 은 점유 중, 55532·55533 은 비어 있다.** 기본 55432 도 이전 회차에서 점유였다. 컨테이너 이름은 **회차 고유**로 유지할 것 — 이 호스트에 `workkit-irumx-dev`·`yeopjari-*-dev`·`resso-test-*` 등 무관한 컨테이너가 떠 있고 스크립트는 지정한 이름을 먼저 삭제하므로 그 이름들을 절대 쓰지 말 것.
  - `cd server && go vet ./... && gofmt -l .`(빈 출력), `go build ./cmd/invenqor-server`
- 위험과 피할 것:
  - 응답 **코드 집합을 늘리지 말 것**. openapi 가 이 경로에 400·404 만 약속하므로 새 코드(422 등)를 쓰면 계약이 깨진다. 형식 오류는 404 `PROPOSAL_NOT_FOUND` 로 기존 '없는 제안' 과 합치는 것이 openapi 변경 없이 끝내는 길이다. **openapi.yaml 을 고치지 말 것.**
  - `canonicalUUID`(`assets.go:702`)를 **느슨하게 바꾸지 말 것**. 중괄호·32자·urn 거절은 merge·split 의 기존 테스트가 고정한 계약이다(2026-10-04 교훈).
  - 감사 기록의 대상 id 에는 정규화된 값만 넘길 것(원문 그대로 넘기면 감사 출력으로 비정규형이 새 나간다 — 운영자 반복 지시).
  - `auth`·`migrations`·`.github/workflows`·`web/`·`server/internal/webui/dist` 는 건드리지 않는다. 버전 범프·릴리즈 노트·PDF 재생성도 이 과제에 섞지 않는다.
  - 기본 `go test`(SQLite) 통과만으로 PostgreSQL 동작을 보장하지 못한다 — 다섯 회차 연속 실측으로 확인됨. 반드시 PostgreSQL 로 한 번 더 돌릴 것.
  - 가짜/주입 DB 로 증명하지 말 것. 실제 `newRuntime` + `server.Handler().ServeHTTP` 로만 검증한다.
- 차선 후보: 관계 삭제를 경로의 부모 assetID 에 속한 관계로 제한 (가치 4 / 위험 2 / M) — `assets.go:665 deleteAssetRelation` 가 `relationID` 만으로 UPDATE 해 자산 A 의 URL 로 자산 B 의 관계를 끝낼 수 있다. 다만 `assetRelations` 가 incoming·outgoing 을 모두 보여 주므로 "부모" 를 source 한쪽으로 볼지 양쪽으로 볼지 정책 확정이 선행이다. 1순위가 성립하지 않을 때만 고를 것.
