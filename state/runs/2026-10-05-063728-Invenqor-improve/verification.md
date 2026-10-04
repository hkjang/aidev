# UUID 관계 검증 구현 증거

커밋: 79f1889. 변경: server/internal/httpapi/assets.go와 asset_relation_validation_test.go 두 파일. 작성자는 기존 hkjang, 트레일러 없음. 릴리즈·push 없음.

## 적용 스킬과 원인

전용 Skill 도구가 노출되지 않아 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/ 아래 completion-verification, systematic-debugging, test-driven-development의 SKILL.md를 읽었다. 새 테스트 작성 → 실제 실패 확인 → 최소 수정 → 통과 → 수정만 제거한 재실패 → 복구 → 전체 검증 순서로 수행했다.

원인: 두 핸들러가 UUID 원문을 DB에 전달하고 감사에도 원문을 전달한다. PostgreSQL은 중괄호/32자를 허용하여 실제 쓰기가 성공하지만 SQLite TEXT/FK는 거절한다. PostgreSQL은 대문자를 DB에서만 정규화하므로 감사에는 대문자가 남고, SQLite는 대문자를 다른 값으로 읽는다. 실제 존재하는 자산/관계의 철자를 바꾸어 요청하여 누락 자산이나 인증 실패가 원인이 아님을 구분했다. 동일 테스트에서 helper를 적용하면 두 DB가 같아지고, 수정만 제거하면 다시 실패한다.

## 새 테스트

- TestRelationCreateRejectsInvalidInputWithoutWrites: 양 인증 경로, source/target 독립 4종 거절, 빈 target/type와 잘못된 JSON, 관계 및 relation.create 감사 건수 전후 비교.
- TestRelationDeleteValidatesIDBeforeWriting: 매번 새 활성 관계, 4종 거절·없는 정규형 404, 대문자 200, source/target/valid_to와 relation.delete 감사 비교.
- TestRelationUpperCaseIDsAreCanonicalInRowsAndAudit: 대문자 source/target 생성 201, 응답 id로 저장 행 확인, 감사 after.target_asset_id 정규형, 대문자 relationID 삭제 200, 재삭제 404 및 valid_to 보존, 삭제 감사 resource_id 정규형.
- TestRelationCreatePreservesConflicts: 없는 정규형 source/target와 중복 관계 409, 기존 관계 및 생성 감사 보존.

두 인증 경로 모두 newRuntime/testServer/authenticateInitialAdmin을 사용하고, 외부 경로는 createAPIKeySecret으로 relations.write 권한 키를 실제 발급한다. malformed JSON을 포함하여 raw HTTP 요청을 Server.Handler().ServeHTTP로 보낸다. Fake Runtime/DB/인증 객체를 주입하지 않았다.

## 수행 명령 및 결과

Go 명령 cwd는 server, PostgreSQL 스크립트 cwd는 저장소 루트. 모든 PostgreSQL 실행은 POSTGRES_CONTAINER=invenqor-impl-20261005-063728-uuid POSTGRES_PORT=55516을 사용했다. 시작 전 해당 컨테이너가 없고 포트 bind가 가능한 것을 확인했다. 컨테이너는 스크립트 EXIT trap으로 제거됐다.

1. 기준선: `go test ./internal/httpapi/ -run 'AssetRoutes|AssetCollections|Merge|Split|Relation' -count=1` → ok httpapi 9.390s.
2. 기준선: 위 컨테이너/포트 환경의 `./scripts/test-postgres.sh -run 'AssetRoutes|AssetCollections|Merge|Split|Relation' -count=1` → ok httpapi 14.411s.
3. 수정 전: `go test ./internal/httpapi/ -run '^TestRelation' -count=1` → FAIL httpapi 7.404s, red-sqlite.log. 비정규형 생성 409≠400, 대문자 생성 409≠201·삭제 404≠200.
4. 수정 전: `./scripts/test-postgres.sh -run '^TestRelation' -count=1` → FAIL httpapi 3.487s, red-postgres.log. 비UUID/urn 생성 409≠400·삭제 500≠404, 중괄호/32자 생성 201≠400·삭제 200≠404 및 실제 쓰기/감사 증가, 대문자 감사 식별자 불일치. 첫 시도는 PostgreSQL did not become ready로 테스트 시작 전에 끝났고 같은 명령 재실행에서 이 실패를 관측했다.
5. 수정 후: 1번 명령 → ok httpapi 22.085s. 2번 명령 → ok httpapi 21.327s. 기존 merge/split/asset UUID 회귀 포함.
6. 인과 확인: assets.go만 HEAD 내용으로 잠시 되돌리고 3번 명령 → FAIL httpapi 3.458s(revert-sqlite.log); finally에서 검증된 수정 내용으로 복구.
7. 최종: `go test ./... -count=1` → 전체 통과, httpapi 30.384s(full-sqlite.log).
8. 최종: `./scripts/test-postgres.sh -count=1` → 전체 통과, httpapi 68.456s(full-postgres.log).
9. 최종: `go vet ./...` → exit 0, 출력 없음(vet.log).
10. 최종: `go build -o <회차 결과 디렉터리>/invenqor-server-check ./cmd/invenqor-server` → exit 0, 생성 바이너리 제거.
11. 최종: `gofmt -l internal/httpapi/assets.go internal/httpapi/asset_relation_validation_test.go` → exit 0, 출력 없음. `git diff --check` → exit 0.

초기 테스트 명령을 루트에서 잘못 호출하여 go.mod not found가 한 번 있었으며, 이후 위 server cwd 실행의 실제 assertion 실패만 결함 재현으로 사용했다. 검증 결과에 테스트 시작 전 환경/호출 실패를 제품 결함으로 포함하지 않았다.

## 범위와 남은 미확인

DELETE 부모 assetID와 관계 소속, confidence 범위, 존재 사전조회, 트랜잭션/SQL 오류 분류는 변경하지 않았다. UUID helper 구현/허용 형식도 그대로다. OpenAPI는 상태코드 집합이 유지되므로 수정하지 않았으며 INVALID_RELATION 문자열의 근거는 기존 핸들러다. Web/Rust, 실제 배포, 릴리즈는 이 서버 핸들러 과제 범위 밖으로 실행하지 않았다. 과제의 양 DB·양 인증 경로 수용 기준에는 남은 미확인이 없다.
