- 과제: 자산 관계 생성·삭제의 UUID 입력을 기존 canonicalUUID 계약으로 검증 (가치 3 / 위험 1 / 작업량 M)
- 왜: `createAssetRelation`은 source·target ID를 날것으로 INSERT하고 모든 SQL 오류를 409로 돌리며, `deleteAssetRelation`은 relationID를 검증 없이 UPDATE하여 PostgreSQL UUID와 SQLite TEXT의 입력 처리 차이에 노출되어 있다. 기존 UUID 검증을 SQL 전에 적용하면 잘못된 형식을 일관된 400/404로 거절하고 대문자 정규형은 같은 관계를 가리키게 된다.
- 수용 기준:
  1) 콘솔 `POST /api/v1/assets/{assetID}/relations`와 외부 `POST /api/v1/external/assets/{assetID}/relations`에서 source 경로 또는 `target_asset_id` 본문이 비UUID·`urn:uuid:`·중괄호·하이픈 없는 32자이면 각각 400 `INVALID_RELATION`; 관계 INSERT와 `relation.create` 감사 기록이 없어야 한다. 두 위치를 독립적으로 시험한다. 빈 target·빈 relation_type·잘못된 JSON의 기존 400도 유지한다.
  2) 두 prefix의 `DELETE .../assets/{assetID}/relations/{relationID}`에서 위 네 종류의 relationID는 404 `RELATION_NOT_FOUND`; 실제 관계의 `valid_to`와 `relation.delete` 감사 건수가 변하지 않아야 한다. 존재하지 않는 정규형 relationID 및 이미 종료된 관계는 기존 404를 유지한다. 이 회차는 DELETE의 부모 assetID 검증·소속 조건을 변경하지 않는다.
  3) 양쪽 인증 경로에서 하이픈 있는 36자 대문자 source·target·relationID를 허용하고 소문자로 정규화한다. 생성은 201 및 정규형 source/target 저장, 삭제는 200 및 해당 관계의 valid_to 설정, 재삭제는 404를 증명한다. 감사 after의 target_asset_id와 삭제 resource_id도 정규형으로 남는다. 정규형의 없는 자산 및 중복 관계는 기존 409 `RELATION_CONFLICT`를 유지한다. 같은 새 테스트가 SQLite·PostgreSQL에서 통과하고 기존 merge·split·자산 UUID 테스트가 그대로 통과해야 한다.
- 건드릴 파일:
  - `server/internal/httpapi/assets.go:createAssetRelation`(622행) — decode/필수값 검사 뒤 SQL 전에 `assetIDParam(request)`와 `canonicalUUID(input.TargetID)` 검사. 거절은 기존 코드 `INVALID_RELATION`의 400, 성공 시 sourceID와 정규화한 input.TargetID를 INSERT·감사에 사용. 입력 형식용 메시지는 정확히 적되 없는 자산까지 사전 조회하지 않는다.
  - `server/internal/httpapi/assets.go:deleteAssetRelation`(654행) — `canonicalUUID(chi.URLParam(request,"relationID"))` 실패 시 기존 404, 성공 시 정규형 id로 UPDATE·감사. `canonicalUUID`(687행)·`assetIDParam`(708행)의 구현/허용 형식은 변경하지 않는다.
  - `server/internal/httpapi/asset_relation_validation_test.go`(신규) — 실제 라우터·실제 Runtime·실제 인증을 사용하는 테이블 테스트. 프로덕션 파일 1개, 테스트 파일 1개가 기본 범위다.
- 검증 명령:
  - 저장소 루트에서 `cd server && go test ./internal/httpapi/ -run 'AssetRoutes|AssetCollections|Merge|Split|Relation' -count=1` — 정찰에서 기존 테스트 통과, httpapi 6.683초. 새 테스트 이름에도 Relation을 넣는다.
  - 저장소 루트에서 `POSTGRES_CONTAINER=invenqor-scout-20261005-063728 POSTGRES_PORT=55505 ./scripts/test-postgres.sh -run 'AssetRoutes|AssetCollections|Merge|Split|Relation' -count=1` — 정찰에서 postgres:17-alpine으로 통과, httpapi 11.853초. 기본 55432는 다른 컨테이너 사용 중. 이 스크립트는 같은 이름의 컨테이너를 삭제하므로 구현자는 자신만의 컨테이너 이름과 비어 있는 포트를 쓴다.
  - 최종 SQLite 전체: `cd server && go test ./... && go vet ./...` 및 `gofmt -l internal/httpapi/assets.go internal/httpapi/asset_relation_validation_test.go`(빈 출력). 최종 PostgreSQL 전체: 루트에서 `POSTGRES_CONTAINER=invenqor-builder-20261005-063728 POSTGRES_PORT=55505 ./scripts/test-postgres.sh -count=1`. 전체 suite·vet는 이번 정찰에서는 미실행이다.
- 위험과 피할 것: auth·migrations·workflows·webui/dist·릴리즈/버전/PDF 변경 금지. UUID 파서를 새로 만들거나 uuid.Parse만 써서 중괄호·32자·urn을 허용하지 않는다. 대문자 36자만 정규화 허용이다. 관계의 confidence·자기 연결·부모 소속·존재 조회·트랜잭션/오류 분류 전체 개편은 범위 밖. DELETE 부모 assetID 미사용은 별도 보류 아이디어이지 이 패치가 해결한다고 주장할 문제가 아니다. DB 실패를 만들려고 가짜 Runtime/DB를 주입하지 않는다. SQLite tx 안의 internalError 교착 구역도 건드리지 않는다.
- 차선 후보: 관계 생성 confidence의 [0,1] 범위 검증 (가치 3 / 위험 1 / S) — 1순위가 이미 해결되어 성립하지 않을 때만. 같은 createAssetRelation의 기존 0→1 기본값은 유지하면서 음수·1 초과를 400 INVALID_RELATION으로 거절하고 양쪽 실제 인증 경로에서 관계/감사 무변경을 검증한다. openapi.yaml의 콘솔·외부 schema가 모두 minimum 0/maximum 1을 선언하며 현재 핸들러와 초기 DB 스키마에는 범위 검사가 없다. 실제 잘못된 confidence 요청은 정찰에서 미실행이다.

범위·근거와 구현 순서

- 2026-10-05 main@12ada2f 기준. 초안을 먼저 저장한 뒤 라우팅·스키마·테스트 실행 결과를 반영했다. 저장소에 CLAUDE.md·AGENTS.md·전용 ROADMAP 파일은 검색 결과 없으며 README·docs/README·EXECUTIVE_REPORT의 단계별 로드맵·최근 git log 30개·CI·테스트 배선을 읽었다. 검색한 src/server/web/src에서 TODO/FIXME는 없었다.
- `server.go:233~239, 429~436`의 두 인증 경로는 동일한 핸들러를 공유한다. `openapi.yaml:1851~1910, 2173~2215, 2489~2490`은 UUID 형식, 생성 400/409 및 삭제 404를 선언한다. **OpenAPI가 INVALID_RELATION이라는 문자열까지 명시한다고 주장하지 말 것**: 오류 코드 문자열은 현재 핸들러에서 가져온다. 이번에는 응답 코드 집합을 늘리지 않아 OpenAPI 수정이 필수는 아니다.
- PostgreSQL `migrations/postgres/001_initial.sql:244`은 관계 id/source/target을 UUID로, SQLite 같은 위치는 TEXT로 선언한다. 새 잘못된 요청의 현재 응답·부작용은 정찰에서 직접 재현하지 않았으며 소스에서 예상한 것이다. 위 통과 결과는 기존 suite의 기준선이지 결함 재현 증거가 아니다.
- 헬퍼 재사용: `server_test.go:newRuntime/testServer`, `admin_test_helpers_test.go:authenticateInitialAdmin/performAuthenticatedJSON`, `software_inventory_test.go:insertSoftwareTestAsset`, `asset_id_param_test.go:assetIDSpellings/assetIDResponse`, `asset_merge_validation_test.go:countRows`. 외부 경로는 `external_api_test.go:createAPIKeySecret/performWithAPIKey/errorCode`로 relations.write 권한의 실제 키를 발급해 호출한다. 같은 패키지에 헬퍼 이름을 중복 정의하지 않는다.
- 생성 성공 응답의 id로 관계를 다시 읽어 검증한다. 거절 생성마다 독립적인 자산 쌍 또는 relation_type을 쓰고, 거절 삭제는 이미 종료된 행을 재사용하지 말고 활성 행으로 시작한다. 감사는 로그인·키 발급 기록을 포함한 전체 행수가 아니라 action='relation.create'/'relation.delete'를 범위로 세거나 전후 차이를 비교한다. 반환 코드만 비교하지 말고 source/target/valid_to도 직접 읽는다.

1. [pending] 위 좁은 SQLite·PostgreSQL 명령으로 구현 환경의 기준선을 확인한다. 자동 체크포인트: 실패가 환경 때문인지 기존 코드 때문인지 기록하며 사람 승인 없이 진행한다.
2. [pending] assets.go의 두 핸들러를 수정하고 좁은 SQLite 명령으로 기존 테스트 회귀가 없는지 확인한다. 자동 체크포인트: 빌드와 기존 테스트가 통과한 상태를 유지한다.
3. [pending] 새 Relation 테스트에 양쪽 인증 경로·형식 거절·대문자·무변경·기존 409/404 동작을 추가한다. 두 방언의 좁은 명령으로 검증한다. 구현자는 가능하면 분리된 이전 코드 실행으로 새 테스트가 결함을 잡는지 먼저 증명하고 결과를 기록한다. 자동 체크포인트: 실패 시 기대치가 계약과 맞는지 확인하고 과제서를 수정한다.
4. [pending] 전체 SQLite·PostgreSQL suite, vet, gofmt를 실행하고 실제 수행 결과와 남은 미확인을 기록한다. 별도 사람 승인 체크포인트는 없다. 정찰이 구현 단계를 완료한 것으로 표시하지 않는다.

대안 비교와 추정 근거

- 선택: 두 쓰기 핸들러의 SQL 직전에서 기존 helper 재사용 — 원인과 오류 계약이 명확하고 기존 배선·DB·인증을 그대로 검증할 수 있다.
- 대안: 공용 라우터 UUID middleware — 호출자별 400/404/빈 목록 계약을 섞기 쉬워 적용 범위가 커진다. 여러 UUID 경로의 계약을 먼저 정리하는 별도 회차에서만 적합하다.
- 대안: DB 오류 코드 분류 — 잘못된 입력의 409/500 일부는 정리할 수 있지만 PostgreSQL이 성공시키는 비정규형과 SQLite 대문자 불일치를 해결하지 못한다. 이 문제의 대안으로는 제외한다.
- 현상 유지 — 코드 변경 비용은 없으나 이미 있는 정규형 정책의 빈틈과 방언 차이를 남기므로 선택하지 않는다.
- 추정(bottom-up, 정찰자 판단): 핸들러 5~8분 + 두 인증 경로의 회귀 테스트 12~17분 + 두 DB 및 정적 검증 8~10분 = 기본 25~35분. 알려진 환경 변동(컨테이너/포트·테스트 fixture) contingency 최대 10분을 별도로 두어 25~45분, 확신 중간(통계적 신뢰구간 아님). 관리 예비비는 배정하지 않았으며 새로운 요구는 별도 회차로 넘긴다. 과거 UUID 회차는 구조상 유사하나 소요시간 기록이 없어 수치 유추 검증은 미확인이다. 핵심 가정은 기존 helper와 실행 환경을 그대로 쓸 수 있다는 것이다.
- 적용한 스킬: 전용 Skill 도구가 노출되지 않아 로컬 원문을 읽었다. `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`(및 references/sources.md), `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. 대안 비교·범위 제외·단계별 proof/checkpoint·분해 추정과 예비 구분을 적용했다. 외부 기관 추정률이나 문헌 수치를 인용한 것은 없다.
