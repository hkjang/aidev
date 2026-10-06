# Invenqor Server·Agent v0.2.45 릴리즈 노트

릴리즈 일자: 2026-10-07
호환 Agent: v0.2.45 (Linux·Windows)

이번 릴리즈는 결함 하나를 고치고, 새로 공개된 권고 하나를 닫습니다. 결함의
자리는 **제안된 자산 관계를 심사하는 API**
(`POST /api/v1/assets/relations/{relationId}/{decision}`)이고, 문제는
**`openapi.yaml` 이 `format: uuid` 로 선언한 모양이 아닌 `relationId` 를 받았을
때 PostgreSQL 과 SQLite fallback 이 서로 다른 답을 했다**는 점입니다. 공개
계약은 이 경로의 응답을 200·400·404 로만 약속하는데, 운영 기본값인
PostgreSQL 에서는 계약에 없는 500 이 되거나 **거절해야 할 요청으로 제안을 실제로
승인·거부하고 감사 기록까지 남겼습니다.** v0.2.40 이 `split` 에서, v0.2.41 이
`merge` 에서, v0.2.42 가 `{assetId}` 단건 다섯 곳에서, v0.2.43 이 관계 쓰기 두
곳에서 닫은 것과 같은 결함이 관계 *심사* 한 곳에 남아 있었습니다. 이제 다섯
번째 자리가 같은 방식으로 닫힙니다. 콘솔과 Agent 는 손대지 않았습니다.

## 1. 선언된 모양이 아닌 id 가 방언마다 다르게 끝났습니다

`openapi.yaml` 은 이 경로를 이렇게 선언합니다.

- `relationId` — `{in: path, name: relationId, required: true, schema: {type: string, format: uuid}}`
- 응답 — `200` 결정 기록, `400` 결정이 approve·reject 가 아님,
  `404` 제안이 없거나 이미 심사됨. **`500` 은 없습니다.**

그런데 `reviewProposedRelation` 은 `chi.URLParam(request, "relationID")` 의 값을
그대로 `UPDATE asset_relations ... WHERE id = $6` 에 넘겼습니다.
`asset_relations.id` 는 **PostgreSQL 에서 `UUID`, SQLite fallback 에서 `TEXT`**
이므로, 선언된 형태가 아닌 철자는 저장 엔진이 알아서 갈라 놓았습니다.

**수정 전 동작을 두 방언에서 직접 재어 보았습니다.**

| `relationId` 철자 | PostgreSQL | SQLite fallback |
|---|---|---|
| `not-a-uuid` | SQLSTATE 22P02 → **계약에 없는 500** | 404 |
| `urn:uuid:<id>` | SQLSTATE 22P02 → **계약에 없는 500** | 404 |
| `{<id>}` (중괄호) | **일치로 접어 넣어 실제로 심사** | 404 |
| 하이픈 없는 32자 | **일치로 접어 넣어 실제로 심사** | 404 |
| 대문자 36자 | **일치로 접어 넣어 실제로 심사** | 404 |

가운데 세 줄이 이 결함의 무게입니다. PostgreSQL 의 `uuid` 입력은 중괄호형과
하이픈 없는 32자를 **정식 철자로 받아들이므로**, 계약이 금지한 모양으로 들어온
요청이 `asset_relations.status` 를 `proposed` 에서 `active`·`rejected` 로 옮기고
`asset.relation.approve`·`asset.relation.reject` 감사 기록까지 남겼습니다.
같은 요청이 SQLite fallback 에서는 404 였으므로, **같은 호출이 어느 저장
엔진에서 돌고 있느냐에 따라 심사되거나 되지 않았습니다.**

마지막 줄은 반대로 갈라집니다. 대문자 36자는 `format: uuid` 가 **허용하는**
철자인데, PostgreSQL 만 일치시키고 SQLite 의 문자열 비교는 일치시키지
못했습니다. 즉 계약이 허용하는 id 가 fallback 에서 404 로 끝났습니다.

감사 기록에도 영향이 있었습니다. `audit_logs.resource_id` 는 두 방언 모두
`TEXT` 이므로, 호출자가 보낸 비정규형 철자가 그대로 저장되고 감사 출력으로 다시
나갔습니다.

## 2. UPDATE 전에 모양을 봅니다

`decision` 검사 직후, `UPDATE` 와 감사 기록보다 앞에서 `canonicalUUID` 로
통과시킵니다. v0.2.40 – v0.2.43 이 쓴 것과 같은 함수입니다 — `uuid.Parse` 가
받아들이고 **길이가 정확히 36자**인 값만 정규형으로 돌려줍니다.

- **모양이 아니면 404 `PROPOSAL_NOT_FOUND` 입니다.** 없는 제안과 같은 답이므로
  호출자가 보는 응답 집합이 넓어지지 않습니다.
- **응답 코드 집합을 넓히지 않았습니다.** 404 는 이미 선언되어 있었고, 사라지는
  것은 선언에 없던 500 입니다. 구현을 문서에 맞춘 것이지 계약을 바꾼 것이
  아니므로 `openapi.yaml` 은 버전 문자열만 올라갑니다.
- **거절된 요청은 아무 것도 남기지 않습니다.** 검사가 `UPDATE` 앞에 있으므로
  `asset_relations.status` 도, `audit_logs` 의 심사 기록도 움직이지 않습니다.
- **정규화된 id 만 아래로 내려갑니다.** 이후 SQL 과 감사 기록에는 정규형이
  쓰이므로, 대문자로 들어온 요청도 두 방언에서 같이 200 이 되고 감사 기록은
  소문자 정규형을 남깁니다. 비정규형 철자가 감사 출력으로 새 나가지 않습니다.
- **프로덕션 파일 한 개의 짧은 변경입니다.**
  (`server/internal/httpapi/classification.go`) 라우팅·권한·CSRF 검증은 손대지
  않았습니다.

이 경로는 세션 인증 전용(`security: [{session: []}]`)이라 API key 로 들어오는
`/api/v1/external/...` 짝이 없습니다. 그래서 이번에는 입구가 하나입니다.

### 이번에 고치지 않은 것

**이미 심사된 관계는 되돌리지 않습니다.** 비정규형 id 로 과거에 승인·거부된
제안을 찾아 원래 상태로 돌리는 마이그레이션은 없습니다. 이번 변경은 새로
들어오는 심사만 막습니다 — 아래 "호환성" 의 점검 질의를 보십시오.

## 3. 프로덕션 의존성 감사를 막고 있던 권고 하나

제품 동작과 무관한 두 번째 변경입니다. CI 의 `npm audit --omit=dev
--audit-level=high` 가 `source-map-js` 1.2.1 에서 새로 공개된 권고
**GHSA-68fv-2mgg-jv7q** 를 찾아 실패했습니다. 이 crate 는 콘솔이 직접 쓰지
않고 `vite` → `postcss` 아래로 따라 들어옵니다 — `vite` 가 프로덕션
의존성이라 `--omit=dev` 가 걸러 주지 못합니다.

권고가 닫히는 가장 낮은 버전인 **1.2.2** 로 올렸습니다. `postcss` 의 요구가
`^1.2.1` 이라 **`package.json` 은 그대로이고 lockfile 한 항목만 바뀝니다.**
`npm audit --omit=dev --audit-level=high` 는 이제 취약점 0 을 보고합니다.

## 검증

- **회귀 테스트 2건을 더했습니다.** 대역 없이 실제 라우터와 실제 세션·CSRF 를
  지납니다. 거절 쪽은 `not-a-uuid`·`urn:uuid:`·중괄호형·하이픈 없는 32자 네
  철자를 `approve`·`reject` 양쪽으로 보내 404 `PROPOSAL_NOT_FOUND` 가 되는 것과
  **`asset_relations.status` 가 `proposed` 로 남아 있고 `asset_relation` 감사
  수가 그대로인 것**을 함께 확인합니다 — 심사해 놓고 404 를 돌려주던 동작이
  응답 코드만 보는 테스트를 통과하지 못하도록, 행을 다시 읽습니다. 수락 쪽은
  대문자 36자가 200 이 되고 상태가 `active` 로 옮겨지는 것에 그치지 않고
  **감사 기록의 `resource_id` 가 정규형인지** 확인합니다.
- **두 저장 엔진 모두에서 통과합니다.** `go test ./...`(SQLite fallback)와
  `scripts/test-postgres.sh`(postgres:17-alpine) 전 패키지가 통과합니다. 이
  결함은 방언 사이에서 갈라지므로 두 쪽에서 모두 돌려야 닫혔다고 말할 수
  있습니다.
- **Server.** `go vet` · `gofmt -l` 빈 출력 · `go build` 가 통과합니다. 콘솔
  정적 파일(`server/internal/webui/dist`)은 바뀌지 않았습니다.
- **콘솔.** `npm test` · `npm run build` 가 통과하고, 빌드 결과가 저장소에
  체크인된 `server/internal/webui/dist` 와 같습니다.
  `npm audit --omit=dev --audit-level=high` 가 취약점 0 입니다.
- **OpenAPI.** `@redocly/cli lint openapi.yaml` 이 통과합니다. 계약은 버전
  문자열만 올라갔습니다.
- **Agent.** Rust 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈 커밋에서
  `cargo fmt --all -- --check` · `cargo clippy --all-targets -D warnings` ·
  `cargo test --all-targets` 가 통과하고, 세 타깃
  (Linux x86_64·aarch64, Windows x86_64) 릴리즈 빌드가 만들어집니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.** 열을 더하거나 바꾸지 않고,
  `UPDATE` 앞에서 id 의 모양을 한 번 더 볼 뿐입니다.
- **OpenAPI 문서가 바뀌지 않았습니다.** `format: uuid` 와 404 는 이미 적혀
  있었습니다. 구현이 뒤늦게 그것을 지킵니다.
- **정규형 id 를 쓰던 호출자에게는 아무 변화가 없습니다.** 관리 콘솔의 심사
  화면은 목록 응답이 돌려준 id 를 그대로 보내므로 이전과 똑같이 동작합니다.
  달라지는 것은 **계약이 선언하지 않은 모양으로 `relationId` 를 보내던
  요청**뿐입니다 — PostgreSQL 에서 500 이나 200 을 받던 그 요청이 이제 두 방언
  모두에서 404 `PROPOSAL_NOT_FOUND` 를 받습니다. 대문자 36자는 반대로, 두 방언
  모두에서 200 이 됩니다.
- **비정규형 id 로 이미 심사된 제안은 남아 있습니다.** 과거의 심사를 되돌리는
  마이그레이션은 없습니다. 영향을 확인하려면 다음을 돌려 보십시오.

  ```sql
  SELECT r.id, r.status, a.created_at, a.actor_id
    FROM asset_relations AS r
    JOIN audit_logs AS a
      ON a.resource_type = 'asset_relation'
     AND a.resource_id <> r.id::text
     AND lower(a.resource_id) = lower(r.id::text)
   WHERE a.action IN ('asset.relation.approve', 'asset.relation.reject');
  ```

  이 질의는 대문자 철자로 남은 기록을 찾습니다. 중괄호형·하이픈 없는 철자는
  `resource_id` 에 그 모양대로 남아 있으므로
  `WHERE resource_id LIKE '{%'` 와 `WHERE length(resource_id) = 32` 로 함께
  찾아보십시오. 행이 나오면 그 관계의 `status` 가 의도한 결정인지 확인해야
  합니다. 0행이면 할 일이 없습니다.
- **자동 분류가 만드는 제안은 이번 변경에 닿지 않습니다.** `classify` 는 자기가
  만든 `uuid` 를 쓰므로 제안 생성과 목록 조회의 동작은 그대로입니다.
- **관계 생성·삭제·조회의 계약은 그대로입니다.** `POST .../relations` 의
  `confidence` 범위는 v0.2.44 가, id 모양은 v0.2.43 이 정한 대로 답합니다.
  이번 변경은 심사 한 곳에만 닿습니다.
- **`merge`·`split`·단건 다섯 곳·관계 쓰기 두 곳의 id 검증은 그대로입니다.**
  v0.2.40 – v0.2.44 가 정한 대로 동작합니다.
- **콘솔 번들이 바뀌지 않았습니다.** `source-map-js` 는 빌드 도구 아래의
  의존성이라 산출물에 들어가지 않습니다. 임베디드 정적 파일은 v0.2.44 와
  같습니다.
- **알려진 제한.** 트랜잭션 안에서 오류를 보고하면 SQLite fallback 이
  교착하는 문제는 그대로 남아 있습니다. 목록 핸들러 가운데
  `assetHistory` · `assetRelations` 에 `rows.Err()` 검사가 없는 것도
  그대로입니다. 두 항목 모두 운영 기본값인 PostgreSQL 에는 해당하지 않거나
  500 으로 끝나며, 재현 수단이 없어 아직 손대지 않았습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜은 이전과
  같고 버전 문자열만 올라갑니다. 기존 Agent 는 서두르지 않아도 Server 와의
  호환에 영향이 없습니다.
