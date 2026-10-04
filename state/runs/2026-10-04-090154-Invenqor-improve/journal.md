# 회차 노트 2026-10-04-090154-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:02] base pinned — main@735ede2
- [러너 09:02] autonomy release — 

## 정찰 노트
- 보류 아이디어 1순위였던 "canonicalUUID 를 쓰지 않는 나머지 id 입력 경로 점검" 을 실제로 조사해 가장 큰 조각(`/api/v1/assets/{assetId}` 계열 다섯 핸들러)으로 쪼개 과제화했다. split(10-01)·merge(10-03)가 같은 모양으로 성공했고 프로덕션 파일이 `assets.go` 한 개라 파일 수 규칙에 여유가 있다. `rows.Err()`·relations 페이지네이션은 실패 주입 수단이 없거나 web+openapi 까지 번져 제쳤다.
- 추측으로 적은 것: PostgreSQL 에서의 **실제** 응답 코드를 직접 돌려 보지 않았다(코드 읽기로 22P02 → `internalError` → 500 을 추론). 컬럼 타입(`assets.id`·`asset_changes.asset_id`·`asset_relations.source/target_asset_id` 모두 UUID)과 openapi 의 선언 코드 집합은 실제로 열어 확인했다.
- 구현자가 조심할 것: `history`·`relations` GET 을 빈 목록 200 으로 두는 것이 판단이다(openapi 가 그 두 경로에 404 를 선언하지 않았다) — 404 로 바꾸려면 `openapi.yaml`+`openapi_routes_test.go` 까지 번지니 하지 말 것. `server.go` 는 손대지 않는다(콘솔·외부 두 경로가 같은 핸들러를 공유해 자동으로 같이 고쳐진다).
- 고치기 **전에** `scripts/test-postgres.sh`(기본 포트 55432 점유 — 포트 지정)로 새 테스트를 한 번 돌려 실패 모양을 남길 것. 기본 `go test` 는 SQLite 폴백이라 이 결함을 못 본다.
- [러너 09:06] scout done — `/api/v1/assets/{assetId}` 계열 핸들러가 경로 파라미터를 날것으로 SQL 에 넘겨 PostgreSQL 에서 500 을 내고 비정규

## 구현 노트
- `assets.go` 에 `assetIDParam`(= `chi.URLParam("assetID")` → `canonicalUUID`)을 두고 `getAsset`·`updateAsset`·`setAssetDeleted` 는 404 `ASSET_NOT_FOUND`, `assetHistory`·`assetRelations` 는 빈 목록 200 으로 쿼리 전에 반환하게 했다. 프로덕션 파일 1개 + 신규 테스트 1개.
- 실측이 과제서보다 나쁜 결함을 드러냈다: PostgreSQL 에서 비정규형이 500 만 내는 게 아니라 중괄호·하이픈 없는 32자를 조용히 접어 **PATCH 가 이름을 바꾸고 DELETE 가 삭제 표시를 남겼다**(`asset_changes` 6행). 커밋 메시지와 원장에 그 출력을 그대로 남겼다.
- **확신 없는 곳**: 수용 기준 3) 의 괄호 주장("merge·split 이 비정규형을 정규화해 받아 준다")은 코드와 어긋난다 — `canonicalUUID` 는 `len != 36` 을 거절하고 merge·split 은 그 셋에 400 을 주는 기존 테스트가 있다. 그래서 대문자 36자만 정규화해 받고 나머지 셋은 두 방언 모두 404/빈 목록으로 일치시켰다. 비평가가 먼저 볼 지점은 이 판단이다.
- 일부러 하지 않은 것: `openapi.yaml`(응답 코드 집합 불변), `server.go`(두 경로가 핸들러 공유), `canonicalUUID` 자체의 수용 범위(merge·split 계약까지 번진다), `createAssetRelation`/`deleteAssetRelation`(범위 밖 — ideas.json 에 다음 조각으로 남겼고 이번 테스트 헬퍼를 재사용할 수 있다), `rows.Err()`(실패 주입 수단 없음), 버전 범프·릴리즈 노트·PDF.
- 다음 역할이 조심할 것: `asset_id_param_test.go` 의 세 테스트 중 두 개(`…NonCanonicalAssetID`)는 **SQLite 폴백에서는 고치기 전에도 통과한다** — 결함 증명은 `scripts/test-postgres.sh` 로만 보인다(포트 55494 로 돌렸다, 기본 55432 는 점유). 세 번째(`…AcceptUpperCaseAssetID`)는 반대로 SQLite 에서만 빨갛던 것이다. 두 방언 전체 패키지 ok 확인했다.
- [러너 09:13] brief accepted — 채택 — 지목한 파일·행·헬퍼·검증 명령이 모두 그대로 들어맞았고 '미확인' 으로 남겨 둔
- [러너 09:13] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인한 것: `assets.go` 를 base 로 되돌려 신규 테스트만 돌려 **두 방언 모두** 실패 모양을 직접 재현했다 — PostgreSQL 에서 중괄호·32자 철자가 실제로 삭제·복원을 수행(`{"deleted":true}`)하고 `asset_changes` 6행을 남기며 `urn:`·비UUID 는 500, SQLite 는 대문자 한 줄만 실패. 원장 기록과 일치한다. 수정 후 SQLite 전체 ok + `test-postgres.sh`(포트 55496) 전체 ok + vet·gofmt·build 통과를 직접 돌렸다. openapi 2067-2186·2489 를 열어 404/200-전용 선언과 핸들러 반환 코드가 맞는 것, `server.go` 미변경으로 `requirePermission`·`requireCSRF` 가 그대로인 것, 빈 목록 본문 모양이 정상 경로와 같은 것을 확인했다.
- 못 본 것: 콘솔(web) 런타임 동작 — `web/` 변경이 없고 콘솔은 API 가 돌려준 정규형 id 만 쓰므로 영향 없다고 판단해 브라우저로는 확인하지 않았다.
- 승인이어도 남는 우려: ① 릴리즈 노트에 "PostgreSQL 에서 통했던 중괄호형·32자 자산 id 가 이제 404" 를 외부 API key 클라이언트용 동작 변화로 한 줄 남길 것. ② `createAssetRelation`(`assets.go:643`)·`deleteAssetRelation`(`:655`) 의 날것 id 가 다음 조각으로 가장 익었다(이번 테스트 헬퍼 재사용 가능). ③ `assetRelations` 의 `$1` 2회 재사용·`rows.Err()`·페이지네이션은 선재 상태로 손대지 않았다.
- 차단 없음: 인가·CSRF 배선 불변, 새 개인정보·의존성·마이그레이션 없음, revert 로 완전히 되돌아온다.
- [러너 09:19] review approved — 리뷰 승인 (risk=low)
- [러너 09:19] pr created — https://github.com/hkjang/invenqor/pull/30
- [러너 09:23] ci passed — 검사 11개 모두 success
- [러너 09:24] merge done — 8503960
