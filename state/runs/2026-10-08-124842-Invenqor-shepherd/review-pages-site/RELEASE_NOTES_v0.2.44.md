# Invenqor Server·Agent v0.2.44 릴리즈 노트

릴리즈 일자: 2026-10-06
호환 Agent: v0.2.44 (Linux·Windows)

이번 릴리즈는 결함 하나를 고칩니다. 자리는 **자산 관계 생성 API**
(`POST /api/v1/assets/{assetId}/relations` 와 API key 로 들어오는
`POST /api/v1/external/assets/{assetId}/relations`)이고, 문제는
**`openapi.yaml` 이 `minimum: 0, maximum: 1` 로 선언한 `confidence` 를 구현이
전혀 검사하지 않았다**는 점입니다. 두 경로의 요청 본문 스키마와
`GET .../relations` 의 응답 스키마가 모두 같은 범위를 약속하는데,
`createAssetRelation` 에는 `0 → 1` 승격만 있고 범위 검사가 없어 `-5` 나 `42` 가
그대로 저장되고 **그 뒤 조회가 자기 응답 스키마가 금지한 값을 돌려주었습니다.**
v0.2.40 부터 v0.2.43 까지가 같은 두 경로에서 id 의 *모양*을 계약에 맞춘 데
이어, 이번에는 본문 숫자의 *범위*를 맞춥니다. 콘솔과 Agent 는 손대지
않았습니다.

## 1. 계약이 선언한 범위를 아무도 지키지 않았습니다

`openapi.yaml` 은 이 값을 세 곳에서 같은 범위로 선언합니다.

- `createAssetRelation` 요청 본문 — `confidence: {type: number, minimum: 0, maximum: 1, default: 1}`
- `externalCreateAssetRelation` 요청 본문 — 같은 선언
- `AssetRelation` 응답 스키마 — `confidence: {type: number, minimum: 0, maximum: 1}`,
  그리고 `confidence` 는 `required` 목록에 있습니다. 이 스키마가
  `listAssetRelations` 와 `externalListAssetRelations` 의 응답 본문입니다.

그런데 쓰는 쪽에는 검사가 없었습니다. `createAssetRelation` 은 본문을 디코드한
뒤 `if input.Confidence == 0 { input.Confidence = 1 }` 하나만 거쳐
`INSERT INTO asset_relations` 로 값을 넘겼습니다. 저장 계층도 막지 않습니다 —
`asset_relations.confidence` 는 PostgreSQL 에서
`DOUBLE PRECISION NOT NULL DEFAULT 1.0`, SQLite fallback 에서
`REAL NOT NULL DEFAULT 1.0` 이고 **두 방언 어느 쪽에도 `CHECK` 가 없습니다.**
즉 계약·핸들러·스키마 세 층 가운데 범위를 아는 것은 계약뿐이었습니다.

**수정 전 동작을 두 방언에서 직접 재어 보았습니다.** 콘솔·외부 두 입구 ×
`-0.5`·`1.5`·`-1`·`42` 의 여덟 경우가 PostgreSQL 과 SQLite fallback 양쪽에서
**전부 201 로 관계 행과 `relation.create` 감사 기록을 남겼습니다.** 이 결함은
v0.2.40–v0.2.43 이 닫은 것들과 달리 방언 사이에서 갈라지지 않습니다 — 두
방언이 똑같이, 똑같은 방향으로 계약을 어겼습니다.

남는 결과는 읽는 쪽에 있습니다. 한 번 들어간 범위 밖 값은
`GET .../relations` 가 `minimum: 0, maximum: 1` 로 선언한 자리에 그대로 실려
나가므로, 생성된 스키마로 응답을 검증하는 클라이언트는 **서버가 자기 계약을
위반한 응답**을 받습니다. `confidence` 를 0 과 1 사이의 비율로 해석해 임계값과
비교하는 호출자에게는 `42` 가 언제나 임계값을 넘고 `-5` 가 언제나 못 넘습니다.

## 2. 쓰기 전에 범위를 봅니다

`targetID` 를 정규형으로 바꾼 직후, `INSERT` 와 `recordAdminAudit` 보다 앞에서
범위를 확인합니다.

- **범위 밖이면 400 `INVALID_RELATION` 입니다.** `confidence < 0` 또는
  `confidence > 1` 이면 거기서 돌아갑니다.
- **응답 코드 집합을 넓히지 않았습니다.** 두 생성 경로가 이미 400
  `INVALID_RELATION` 을 선언하고 있으므로, 구현을 문서에 맞춘 것이지 계약을
  바꾼 것이 아닙니다. `openapi.yaml` 은 손대지 않았습니다.
- **거절된 요청은 아무 것도 남기지 않습니다.** 검사가 `INSERT` 앞에 있으므로
  `asset_relations` 행도, `audit_logs` 의 `relation.create` 기록도 쓰이지
  않습니다.
- **경계값은 그대로 받습니다.** `0` 과 `1` 은 범위 안이므로 거절 대상이
  아니고, 필드를 생략하면 이전과 같이 `1` 이 됩니다.
- **콘솔과 외부 API 가 함께 고쳐졌습니다.** 두 경로가 같은 핸들러를
  공유하므로 프로덕션 파일 한 개(`server/internal/httpapi/assets.go`) 열 줄로
  두 입구가 같이 닫혔습니다. 라우팅은 손대지 않았습니다.

### 이번에 고치지 않은 것

**명시적 `confidence: 0` 은 여전히 `1` 로 승격됩니다.** `openapi.yaml` 의
`default: 1` 은 값이 *없을* 때만 적용되는 규칙이므로, 호출자가 `0` 을 적어
보냈는데 `1` 이 저장되는 지금 동작은 엄밀히는 틀렸습니다. 다만 "없음" 과 `0`
을 구분하려면 입력 구조체의 `float64` 를 `*float64` 로 바꿔야 하고 그것은 이
결함과 별개의 변경이라, 이번 범위에서 제외했습니다. 대신 **지금 동작을
테스트로 고정**해 두어 조용히 달라지지 않게 했습니다. 0 과 1 중 어느 쪽을
뜻했는지가 중요한 호출자는 당장은 `0` 대신 아주 작은 값을 쓰거나, 이 항목이
닫히기를 기다려야 합니다.

**이미 저장된 범위 밖 값은 고치지 않습니다.** 마이그레이션도, 열의 `CHECK` 도
더하지 않았습니다. 이번 변경은 새로 들어오는 쓰기만 막습니다 — 아래 "호환성"
의 점검 질의를 보십시오.

## 검증

- **회귀 테스트 2건을 더했습니다.** 대역 없이 실제 라우터와 실제 세션·API key
  를 지나 콘솔·외부 두 입구를 각각 통과합니다. 거절 쪽은 `-0.5`·`1.5`·`-1`·`42`
  가 400 `INVALID_RELATION` 이 되는 것과 **`asset_relations` 행 수·
  `relation.create` 감사 수가 그대로인 것**을 함께 셉니다. 수락 쪽은 생략·`0`·
  `1`·`0.8` 네 경우가 201 이 되는 것에 그치지 않고 **저장된 `confidence` 를
  행에서 다시 읽어** 각각 `1`·`1`·`1`·`0.8` 인지 확인합니다.
- **두 저장 엔진 모두에서 통과합니다.** `go test ./...`(SQLite fallback)와
  `scripts/test-postgres.sh`(postgres:17-alpine) 전 패키지가 통과합니다. 이
  결함은 방언을 가리지 않으므로 두 쪽에서 똑같이 재현되고 똑같이 닫힙니다.
- **Server.** `go vet` · `gofmt -l` 빈 출력 · `go build` 가 통과합니다. 콘솔
  정적 파일(`server/internal/webui/dist`)은 바뀌지 않았습니다.
- **OpenAPI.** `@redocly/cli lint openapi.yaml` 이 통과합니다. 계약은 버전
  문자열만 올라갔습니다.
- **Agent.** Rust 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈 커밋에서
  `cargo fmt --all -- --check` · `cargo clippy --all-targets -D warnings` ·
  `cargo test --all-targets` 가 통과합니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.** 열을 더하거나 바꾸지 않고,
  `INSERT` 앞에서 숫자의 범위를 한 번 더 볼 뿐입니다.
- **OpenAPI 문서가 바뀌지 않았습니다.** `minimum: 0, maximum: 1` 은 이미
  적혀 있었고 400 도 이미 선언되어 있었습니다. 구현이 뒤늦게 그것을 지킵니다.
- **범위 안의 값을 쓰던 호출자에게는 아무 변화가 없습니다.** 생략·`0`·`1` 과
  그 사이의 모든 값이 이전과 똑같이 동작하고, 201 응답의 모양도 권한과 CSRF
  검증도 그대로입니다. 달라지는 것은 **계약이 선언하지 않은 범위로
  `confidence` 를 보내던 요청**뿐입니다 — 201 을 받던 그 요청이 이제 400
  `INVALID_RELATION` 을 받습니다.
- **이미 쌓인 범위 밖 값은 남아 있습니다.** 과거에 들어간 값을 고치는
  마이그레이션은 없습니다. 영향을 확인하려면 다음을 돌려 보십시오.

  ```sql
  SELECT id, source_asset_id, target_asset_id, confidence
    FROM asset_relations
   WHERE confidence < 0 OR confidence > 1;
  ```

  행이 나오면 그 관계를 다시 만들거나 `confidence` 를 범위 안으로 직접 고쳐야
  합니다. 0행이면 할 일이 없습니다.
- **자동 분류와 ingest 가 만드는 관계는 이번 변경에 닿지 않습니다.** 그 경로는
  호출자 입력을 받지 않고 `0.8` 등 범위 안의 값을 직접 정하므로, `classify` ·
  `ingest` · `softwarecatalog` 의 동작은 그대로입니다.
- **관계 읽기·삭제의 계약은 그대로입니다.** `GET .../relations` 와
  `DELETE .../relations/{relationId}` 는 v0.2.42 · v0.2.43 이 정한 대로
  답합니다. 이번 변경은 생성 한 곳에만 닿습니다.
- **`merge`·`split`·단건 다섯 곳·관계 쓰기 두 곳의 id 검증은 그대로입니다.**
  v0.2.40 – v0.2.43 이 정한 대로 동작합니다.
- **알려진 제한.** 트랜잭션 안에서 오류를 보고하면 SQLite fallback 이
  교착하는 문제는 그대로 남아 있습니다. 목록 핸들러 가운데
  `assetHistory` · `assetRelations` 에 `rows.Err()` 검사가 없는 것도
  그대로입니다. 두 항목 모두 운영 기본값인 PostgreSQL 에는 해당하지 않거나
  500 으로 끝나며, 재현 수단이 없어 아직 손대지 않았습니다.
- **콘솔이 바뀌지 않았습니다.** 임베디드 정적 파일은 v0.2.43 과 같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과
  같고 버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의
  호환에 영향이 없습니다.
