# 과제서 — 2026-10-06-001803-Invenqor-improve

- 과제: 자산 관계 생성의 `confidence` 를 openapi 가 선언한 `[0,1]` 범위로 검증 (가치 3 / 위험 1 / 작업량 S)

- 왜: `openapi.yaml:1885`(콘솔 `createAssetRelation`)·`:2193`(외부 `externalCreateAssetRelation`) 이
  요청 본문의 `confidence` 를 `{type: number, minimum: 0, maximum: 1, default: 1}` 로 선언하지만,
  `server/internal/httpapi/assets.go:622 createAssetRelation` 은 `if input.Confidence == 0 { input.Confidence = 1 }`
  한 줄만 두고 범위를 보지 않는다. 두 방언의 `asset_relations.confidence` 에도 CHECK 가 없어
  (`server/migrations/postgres/001_initial.sql:252` = `DOUBLE PRECISION NOT NULL DEFAULT 1.0`,
  `server/migrations/sqlite/001_initial.sql:252` = `REAL NOT NULL DEFAULT 1.0`; 두 파일 직접 확인)
  `-5` 나 `42` 가 그대로 INSERT 되고, 그 뒤 `GET /relations` 가 자기 응답 스키마
  (`openapi.yaml:2578` 의 `confidence: {type: number, minimum: 0, maximum: 1}`)를 위반하는 값을
  돌려준다. 고치면 쓰기 시점에 400 으로 막혀 저장된 데이터가 공개 계약 안에 머문다.

- 수용 기준:
  1) `POST /api/v1/assets/{assetID}/relations` 와 `POST /api/v1/external/assets/{assetID}/relations`
     둘 다, 본문 `confidence` 가 `0` 미만이거나 `1` 초과면 **SQL 실행 전에** 400 `INVALID_RELATION`
     으로 거절한다. 두 openapi 경로가 이미 400 을 선언하므로 응답 코드 집합은 넓어지지 않는다.
  2) 거절된 요청은 부작용이 없다 — `asset_relations` 행이 늘지 않고 `admin_audit` 의
     `relation.create` 기록도 늘지 않는다 (`recordAdminAudit` 호출 전에 반환할 것).
  3) 테스트가 증명할 것: (a) `-0.5`·`1.5`·`-1` 같은 경계 밖 값이 콘솔·외부 두 경로에서 각각 독립적으로
     400 `INVALID_RELATION` 을 받는다, (b) 같은 요청 뒤 `asset_relations`·`admin_audit` 행 수가
     그대로다, (c) 경계값 `0`·`1`·`0.8` 과 `confidence` 를 **생략한** 요청은 여전히 201 이고
     저장된 `confidence` 가 기대값이다 — 특히 생략 시 `1`, 명시적 `0` 은 현재 동작(1 로 승격)이
     그대로 유지됨을 고정한다, (d) 기존 409 `RELATION_CONFLICT`·404·UUID 400 경로는 그대로다.

- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `server/internal/httpapi/assets.go:622 createAssetRelation` — `canonicalUUID(input.TargetID)`
    검사 **직후**, `if input.Confidence == 0 { input.Confidence = 1 }` 와 `ExecContext` **앞에**
    `input.Confidence < 0 || input.Confidence > 1` 이면 400 `INVALID_RELATION` 반환. 이 핸들러는
    `BeginTx` 를 쓰지 않으므로 SQLite 폴백 교착 구역(아래 위험 참조)을 비켜간다.
  - `server/internal/httpapi/asset_relation_validation_test.go` — 기존 파일에 테스트 함수 추가.
    같은 패키지 헬퍼를 **재사용**할 것: `relationValidationClient(t, external)`(:16, 실제 라우터·
    실제 세션/CSRF 또는 실제 API key 발급), `relationValidationAssets(t, server)`(:41),
    `assertRelationResponse`(:56), `countRows`(`asset_merge_validation_test.go:41`),
    `errorCode`(`external_api_test.go:226`). 주의: 기존 `relationValidationBody(targetID, relationType)`
    는 `map[string]string` 이라 숫자 `confidence` 를 낼 수 없다 — 본문은 새 작은 빌더나 원시 JSON
    문자열로 만들 것(기존 함수의 시그니처는 바꾸지 말 것, 다른 테스트가 쓴다).

- 검증 명령 (이 저장소에서 실제로 돈 것):
  - `cd server && go test ./internal/httpapi/ -run 'Relation' -count=1` — 2026-10-06 실측 `ok … 2.072s`.
  - `cd server && go test ./... -count=1 && go vet ./... && gofmt -l .`(빈 출력) `&& go build ./cmd/invenqor-server`.
  - PostgreSQL(필수 — 기본 `go test` 는 SQLite 폴백이라 방언 차이를 못 본다):
    루트에서 `POSTGRES_CONTAINER=invenqor-impl-20261006-0018-conf POSTGRES_PORT=55521 ./scripts/test-postgres.sh -run 'Relation' -count=1`.
    컨테이너 이름·포트는 회차마다 새로 — 스크립트는 지정한 이름의 컨테이너를 먼저 삭제한다.
    **미확인**: 55521 포트가 비어 있는지는 확인하지 않았다. 점유 시 다른 번호로.

- 위험과 피할 것:
  - **`openapi.yaml` 은 손대지 말 것.** `minimum: 0, maximum: 1` 이 이미 선언돼 있고 두 경로가 이미
    400 을 선언한다 — 코드를 문서에 맞추는 변경이지 그 반대가 아니다. 버전 범프·릴리즈 노트·PDF
    재생성도 하지 말 것(이 저장소 관례상 별도 작업).
  - **마이그레이션에 CHECK 를 추가하지 말 것.** 기존 행에 범위 밖 값이 있으면 마이그레이션이
    실패하고, 두 방언 SQL 을 동시에 건드리는 것은 보호 구역이다. 검증은 핸들러에서만.
  - **명시적 `confidence: 0` 을 `1` 로 올리는 현재 동작은 이번 범위 밖**이다. openapi 의 `default: 1`
    은 "값이 없을 때" 를 뜻하므로 생략 시 1 은 옳고, 명시적 0 만 어긋난다. 고치려면 `*float64` 로
    "없음" 과 0 을 구분해야 하고 그건 별도 과제다. 기준 3)(c) 는 현재 동작을 **고정**하는 쪽으로 쓸 것.
  - **트랜잭션 안에서 `internalError` 를 부르지 말 것** — SQLite 단일 커넥션에서 교착한다는 기존
    교훈. 이번 과제의 검증은 `ExecContext` 전·`BeginTx` 밖이라 해당 없지만, 코드를 옮기지 말 것.
  - 같은 패키지 테스트 헬퍼 **중복 선언 금지**(위에 나열한 것을 쓸 것). `server.go:225-232`(외부)·
    `:424-428`(콘솔) 이 같은 핸들러를 공유하므로 `server.go` 와 `web/` 은 건드리지 않아도 두 경로가
    함께 고쳐진다. 웹 콘솔은 관계를 **삭제만** 하고(`web/src/operationsPages.tsx:1289`) 생성 POST 를
    보내지 않아 확인했으므로 콘솔 회귀 위험이 낮다.
  - **미확인**: 수정 전 PostgreSQL 이 범위 밖 값에 실제로 201 을 주는지 직접 요청해 보지 않았다
    (스키마에 CHECK 가 없음을 근거로 추론). 구현자는 수정 **전에** 새 테스트를 두 DB 에서 돌려
    실패를 실측 기록할 것 — 지난 네 회차 모두 이 단계에서 과제서의 예측이 보정됐다.

- 차선 후보: **관계 삭제를 경로의 부모 `assetID` 에 속한 관계로 제한** (가치 4 / 위험 2 / M).
  `assets.go:665 deleteAssetRelation` 은 `relationID` 만으로 UPDATE 하고 경로의 `assetID` 를 전혀
  쓰지 않아, 자산 A 의 URL 로 자산 B 의 관계를 끝낼 수 있다. 다만 `assetRelations`(:574) 가
  incoming·outgoing 을 모두 보여 주므로 "부모" 를 `source_asset_id` 한쪽으로 볼지 양쪽으로 볼지
  정책을 먼저 확정해야 한다 — 1순위가 성립하지 않을 때만, 그 정책을 과제서에 적고 시작할 것.
