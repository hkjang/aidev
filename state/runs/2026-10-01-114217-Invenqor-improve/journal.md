# 회차 노트 2026-10-01-114217-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:42] base pinned — main@21954f0
- [러너 11:42] autonomy release — 

## 정찰 노트
- `splitAsset` 을 골랐다. 보류 1위였던 "tx 안 internalError 교착"(4/3/M)은 트랜잭션 안에서 오류를 터뜨릴 **프로덕션 경로를 찾지 못해** 제쳤다(손으로 만든 대역 금지 규칙과 충돌). split 은 openapi 가 400 만 약속하는데 방언에 따라 500 이 되고, 테스트가 0건이라 가치·증명 가능성이 둘 다 선다. 차선 `rows.Err()` 는 같은 재현 수단 문제가 있어 2순위로만 뒀다.
- 추측으로 적은 것: PostgreSQL 에서 형식 틀린 UUID 가 **실제로** 22P02 로 500 이 되는 것은 열 타입(`postgres/001_initial.sql:216-218` UUID)으로 추론한 것이고 돌려 보지 못했다. `scripts/test-postgres.sh` 로 먼저 실패를 재현해 확인할 것. 초기 admin 이 `assets.merge` 권한을 갖는다는 것도 merge 테스트가 통과 중이라는 간접 근거다.
- 조심할 것: 같은 패키지에 `countRows` 등 헬퍼가 이미 있으니 이름을 다시 정의하면 컴파일이 깨진다. `assets.go` 의 나머지 tx 내 `internalError` 지점은 이번에 손대지 말 것 — 범위가 M 으로 커지고 검증 수단이 없다.
- [러너 11:47] scout done — `POST /api/v1/assets/{assetId}/split` 이 형식이 틀린 UUID 를 PostgreSQL 에서 500 으로 터뜨리고, 이 엔드포인트에 테스

## 구현 노트
- `splitAsset` 의 `decodeJSON` 직후·`BeginTx` 전에 `uuid.Parse` 로 `assetID` 경로 파라미터와 `source_ids` 각 원소의 모양을 검사해 400 `INVALID_SPLIT`·`INVALID_SOURCE` 로 끝낸다. 프로덕션 파일은 `assets.go` 한 개, 그 외는 신규 테스트 `asset_split_validation_test.go` 4건(실제 라우터 통과, 기존 헬퍼만 사용 — 새 대역 없음).
- 정찰이 추측으로 남긴 22P02 는 **실측으로 확인됐다**: 고치기 전 PostgreSQL 에서 두 입력 모두 `500 "INTERNAL_ERROR"`. SQLite 에서는 malformed `source_ids` 테스트가 고치기 전에도 **통과**했다 — 방언 차이가 그대로 드러났고, 그래서 이 과제의 실패 재현은 `scripts/test-postgres.sh` 없이는 반쪽이다.
- 확신 없는 곳: 오류 코드 선택. malformed `source_ids` 를 `INVALID_SOURCE` 로 둔 것은 SQLite 가 이미 답하던 코드를 보존해 폴백의 동작 변화를 0 으로 만들려는 선택이지만, "형식이 틀림" 을 "남의 자산에 속함" 과 같은 코드로 묶는다. 과제서가 둘 중 하나를 고르라 했고 openapi 설명(`Invalid split or source does not belong…`)이 둘을 한 문장에 담아 어느 쪽도 계약 위반은 아니다. 비평가가 `INVALID_SPLIT` 이 낫다고 보면 테스트 한 줄과 핸들러 한 줄만 바뀐다.
- `uuid.Parse` 는 모양만 보고 **정규화하지 않는다** — 대문자 UUID·중괄호 형식이 통과해 `asset_changes.after_json` 에 보낸 글자 그대로 저장된다. `mergeAssets` 의 기존 `merged_into` 결함(ideas.json, 2/1/S)과 같은 성질이며 이번에 고치지 않았다.
- 일부러 하지 않은 것: (1) 트랜잭션 안 `internalError` → SQLite 교착(`splitAsset` 809·820·846·850 등) — 검증을 tx 밖에 두어 **회피**했을 뿐 고치지 않았다. 과제서 지시이고 재현 수단이 여전히 없다. (2) `assets.id` 존재 확인 — 범위 밖. (3) `openapi.yaml` — 응답 코드 집합이 그대로라 `openapi_routes_test.go` 가 요구하지 않는다. (4) 버전 범프·릴리즈 노트·PDF.
- 다음 역할이 조심할 것: 새 테스트 4건 중 malformed `source_ids` 건은 **SQLite 로 돌리면 고치기 전에도 통과**하므로 회귀 가치를 확인하려면 docker 로 `scripts/test-postgres.sh` 를 돌려야 한다. 기본 포트 55432 는 이 머신에서 점유돼 있어 `POSTGRES_PORT=55493` 로 돌렸다. `go build` 가 `server/invenqor-server` 바이너리를 남기므로(gitignore 됨) 커밋 전에 지웠다.
- 검증 전부 통과: `go test ./...`(SQLite), `go vet ./...`, `gofmt -l .` 빈 출력, `go build ./cmd/invenqor-server`, `scripts/test-postgres.sh`(전체 패키지 ok). 기존 merge 3건·`mcp_asset_get_merged_test.go` 2건 무영향. 커밋 `22ef83e`, 작업 트리 clean.
- [러너 11:53] brief accepted — 채택 — 근거(openapi 의 400 단독 약속, `splitAsset` 에 형식 검증이 전무, UUID/TEXT 방언 차이, split
- [러너 11:53] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인한 것: 실제 `postgres:17-alpine` 과 SQLite 두 방언에 프로덕션 라우터로 새 테스트 4건을 직접 돌렸다(둘 다 통과) + `uuid.Parse` 의 허용 형식과 PostgreSQL `uuid` 입력의 허용 형식을 psql 로 대조했다. 보지 않은 것: `./...` 전체 완주, web·Rust(이번 diff 밖).
- **reject** — `uuid.Parse` 는 검증기가 아니다(godoc 명시). `urn:uuid:<정상uuid>` 는 Parse 를 통과하고 PostgreSQL 이 거절해 **고친 뒤에도** `assetID`·`source_ids` 양쪽에서 500 INTERNAL_ERROR 가 재현된다(SQLite 는 400). 커밋이 없앴다고 한 증상이 그대로 남았다.
- 더 나쁜 쪽: 중괄호 `{<정상uuid>}` assetID 는 PostgreSQL 에서 **201 로 split 이 실제 수행**되고 SQLite 에서는 400 이다 — 쓰기의 성공/실패가 방언에 따라 갈리며 `assets.go:792-797` 주석의 "두 방언이 같은 코드" 단언과 정면으로 어긋난다.
- 수리가 먼저 볼 파일: `server/internal/httpapi/assets.go:798-808`(정규형 요구 또는 `parsed.String()` 으로 정규화 후 SQL 에 넘기기 — 구현자가 남긴 '대문자·중괄호가 `after_json` 에 글자 그대로 저장' 문제도 같이 닫힌다), 그리고 `asset_split_validation_test.go` 에 비정규 형식 케이스 추가.
- 남는 우려(수리와 별개): malformed `source_ids` → `INVALID_SOURCE` 는 계약 위반 아님(거절 사유 아님). `TestSplitAssetRejectsSourceIDsThatAreNotUUIDs` 는 SQLite 로는 수정 전에도 통과하므로 CI 가 SQLite 기본이면 이 회귀는 CI 가 못 잡는다. 보안·법무는 깨끗(CSRF+`assets.merge` 로 가려짐, 새 개인정보·의존성·마이그레이션 없음).
- [러너 11:58] review rejected — 리뷰 거절: server/internal/httpapi/assets.go:798,802 uuid.Parse 는 모양 검사기가 아니다 — godoc 이 직접 "Parse should not be used to validate strings" 라고 적는다. urn:uuid: �

## 수리 노트
- 맞았던 지적: 3건 전부. 실제 `postgres:17-alpine` 으로 재현 — `urn:uuid:<id>` 는 `assetID`·`source_ids` 양쪽에서 500 INTERNAL_ERROR, `{<실제 id>}`·하이픈 없는 32자는 **201 로 split 이 실제 수행**되는데 SQLite 는 셋 다 400 이었다. 틀린 지적은 없었다. 다만 비평가가 "중괄호는 PG 에서 201" 이라 한 것은 중괄호가 **실제 존재하는** id 를 감쌀 때만 그렇다 — 처음 쓴 표 기반 테스트는 무작위 uuid 를 감싸서 이 발산을 놓쳤고, 실제 id 를 다시 쓰도록 고쳐서야 201 이 재현됐다.
- 고친 방법: `canonicalUUID`(assets.go:648) 를 추가해 openapi `AssetId`/`source_ids` 가 선언하는 `format: uuid` — 36자 하이픈 형식 — 만 받고(`len(value)==36` + `uuid.Parse`) `parsed.String()` 으로 소문자 정규형을 돌려준다. `splitAsset` 은 이 정규형만 SQL·`after_json`·audit 에 넘기므로 두 방언이 같은 답을 내고 대문자 id 가 글자 그대로 저장되던 문제도 닫혔다. 형식 판단 근거를 내 취향이 아니라 `openapi.yaml:2489` 의 `format: uuid` 에 뒀다.
- 테스트: 비정규 형식을 **실존 id** 에 적용하는 표 기반 2건(urn:·중괄호·하이픈 없는·not-a-uuid × assetID/source_ids) + 대문자 정규화 1건. 고치기 전 두 방언 합쳐 7건 실패 → 고친 뒤 전부 통과. 기존 테스트는 삭제·완화 없음.
- 확신 없는 곳: (1) 대문자 정규형을 **받아들이고 접는** 선택 — `format: uuid` 는 대소문자를 가리지 않으니 400 으로 거절하면 PG 에서 되던 요청이 깨지므로 접는 쪽을 골랐으나, 거절이 더 엄격해 좋다고 볼 수도 있다. (2) `missingAssetID`(merge, assets.go:689) 는 **똑같은 `uuid.Parse` 결함이 그대로 남아** `urn:uuid:` 로 500 이 날 것이다 — 이번 변경(base 21954f0) 밖의 기성 코드라 범위 밖으로 두었고 `canonicalUUID` 를 한 줄 적용하면 닫히지만 merge 쪽 테스트가 따로 필요하다. 다음 회차 후보.
- 범위: `assets.go`·`asset_split_validation_test.go` 두 파일만 수정. tx 안 `internalError` → SQLite 교착은 여전히 회피만 하고 고치지 않았다(검증이 tx 밖). `server/invenqor-server` 빌드 산출물은 지웠다.
- [러너 12:09] repair done — # fix-summary  비평가의 지적 3건은 모두 사실이었고, 실제 postgres:17-alpine 으로 재현했다 — `urn:uuid:<id>` 는 assetID·source_ids 양쪽에서 **500 INTERNAL_ERROR**, `{<실

## 비평 노트
- (수리 후 재심) 확인한 것: 신규 테스트를 수정 전 트리(22ef83e)에 복사해 돌려 **실패를 직접 재현**했다 — SQLite 4건, 실제 postgres:17-alpine 7건(`urn:uuid:` → 500, 중괄호·하이픈 없는 형식 → **201 로 split 수행**, after_json 에 대문자 그대로). 고친 뒤 두 방언 모두 `go test ./...` 전체 통과 + gofmt/vet 통과. 테스트는 프로덕션 라우터를 지나고 단언이 실질적이다(`assets WHERE source='manual'` 체크도 픽스처가 'agent' 라 비지 않는다).
- `canonicalUUID`(assets.go:661) 의 `len==36` 가드를 google/uuid v1.6.0 `Parse` 소스와 대조했다 — 36 분기만 하이픈 4곳과 hex 16쌍을 모두 검사하므로 통과 집합이 PostgreSQL `uuid` 입력의 정규형과 정확히 일치한다. 통과하는데 PG 가 거절하는 36자는 없다. 검증이 `BeginTx` 앞이라 tx 누수도 없다. **approve, risk low, 차단 없음.**
- 남는 우려: `missingAssetID`(assets.go:689)는 맨 `uuid.Parse` 그대로 → **merge 는 `urn:uuid:` 로 PG 에서 아직 500**, 중괄호·하이픈 없는 형식은 PG 만 merge 를 수행한다. 다음 회차 1순위(`canonicalUUID` 한 줄 + merge 테스트). tx 안 `s.internalError` → SQLite 교착은 splitAsset 849·865·886·890 에 여전히 남아 있다(회피만 했음).
- 릴리즈 노트에 꼭 적을 것: **조이는 변경**이다 — PG 에서 201 로 성공하던 비정규 철자가 이제 400 이고, SQLite 는 형식 틀린 assetID 코드가 `INVALID_SOURCE`→`INVALID_SPLIT` 로 바뀌며(400 유지, openapi 가 코드를 열거하지 않아 계약 위반 아님) 대문자 정규형 id 는 접혀서 성공한다.
- 보지 않은 것: web·Rust(이번 diff 밖). 범위는 `assets.go` + 신규 테스트 두 파일뿐이고 버전·PDF·`webui/dist`·`openapi.yaml` 은 손대지 않았다 — 관례대로 맞다.
- [러너 12:15] review approved — 리뷰 승인 (risk=low)
- [러너 12:16] pr created — https://github.com/hkjang/invenqor/pull/28
- [러너 12:20] ci passed — 검사 11개 모두 success
- [러너 12:20] merge done — 52ae5a9
- [러너 12:44] release published — v0.2.40
- [러너 12:44] gh-release created — GitHub Release v0.2.40
- [러너 12:44] manifest ok — ADMIN_GUIDE.md ADMIN_GUIDE.pdf API_MCP_GUIDE.md API_MCP_GUIDE.pdf EXECUTIVE_REPORT.md EXECUTIVE_REPORT.pdf RELEASE_NOTES_v0.2.40.md SERVER_INSTALLATION.md SERVER_INSTALLATION.pdf USER_GUIDE.md USER_GU
- [러너 12:45] assets uploaded — 29개
- [러너 12:45] assets verified — v0.2.40 자산 29개 (이전 v0.2.39: 29)
