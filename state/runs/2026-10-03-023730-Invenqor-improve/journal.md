# 회차 노트 2026-10-03-023730-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:37] base pinned — main@97344b7
- [러너 02:37] autonomy release — 

## 정찰 노트
- `mergeAssets` 가 `splitAsset` 과 달리 `canonicalUUID`(assets.go:661, 이미 존재)를 쓰지 않아 같은 방언 발산이 남아 있는 것을 골랐다 — 직전 두 회차가 같은 모양의 과제를 release-ready 로 통과시켰고, 프로덕션 파일이 `assets.go` 한 개로 끝나며, 테스트 선례(`asset_split_validation_test.go:204`·`235`·`274`)가 그대로 베껴 쓸 형태로 있다.
- 제친 후보: tx 내 `internalError` SQLite 교착(4/3/M)은 세 회차 연속 재현 수단이 없어 또 미룸 — 대역 주입은 반려 사유다. `rows.Err()`(3/1/M)도 실패 주입 수단이 없어 차선으로만 남겼다.
- 추측으로 적은 것: PostgreSQL 에서의 실제 코드(대문자 200 / `urn:uuid:` 500)를 이번에 직접 돌려 보지 못했다. 근거는 `canonicalUUID` 주석에 split 회차가 실측해 남긴 방언 동작과 postgres 001 의 UUID 열 타입이다. 구현자는 고치기 전에 `scripts/test-postgres.sh` 로 실패를 한 번 재현해 기록할 것.
- 조심할 것: 검증을 반드시 `BeginTx`(assets.go:716) 전에 둘 것(tx 안 `internalError` 는 응답 없음 → 타임아웃). `input.PrimaryID`/`input.SecondaryIDs` 를 아래에서 다시 읽는 자리(759행 자기 병합 스킵, 770-772 metadata)를 전부 정규형 지역 변수로 바꿔야 `merged_into` 가 고쳐진다 — 한쪽만 고치면 겉으로는 통과하고 MCP 안내만 계속 깨진다.
- baseline: `go test ./internal/httpapi/ -run 'Merge|Split|MergedInto' -count=1` → ok 2.819s (초록). 프로필을 2026-10-03 로 갱신했다(split 무검증 기술이 낡아 있었다).
- [러너 02:42] scout done — `POST /api/v1/assets/merge` 가 openapi 가 선언한 정규형이 아닌 UUID 철자를 받아 두 방언이 같은 요청에 다르게 답

## 구현 노트
- 무엇을: `mergeAssets` 가 `decodeJSON` 직후·`BeginTx` 전에 `canonicalUUID` 로 `primary_id`·`secondary_ids` 를 거르고, 그 아래 쿼리·자기 병합 스킵·`asset_changes` metadata·`audit` 기록이 전부 지역 변수 `primaryID`/`secondaryIDs` 만 읽게 했다. `input.*` 를 다시 읽는 자리는 남아 있지 않다(grep 아닌 컴파일·테스트로 확인). 프로덕션 파일 1개.
- 확신 없는 곳: `missingAssetID` 안의 `uuid.Parse` 모양 검사를 지웠다 — 호출자가 하나뿐이고 그 호출자가 정규화를 끝내므로 닿지 않는 분기지만, 앞으로 다른 호출자가 생기면 전제가 깨진다(주석으로 명시해 뒀다). 그 외에 검증 못 한 것은 없다: PostgreSQL·SQLite 양쪽 전체 suite 를 돌렸다.
- 일부러 안 한 것: `mergeAssets` 의 tx 안 `internalError` 호출들(교착 위험)은 과제서 범위 밖이라 그대로 뒀다. openapi·버전·릴리즈 노트·PDF 도 손대지 않았다.
- 다음 역할이 조심할 것: 새 테스트 4건 중 비정규형 거절 6개 서브테스트는 **SQLite 에서는 고치기 전에도 통과했다** — 이 변경의 회귀를 실제로 지키는 것은 `scripts/test-postgres.sh` 쪽이다(`POSTGRES_PORT=55497 POSTGRES_CONTAINER=invenqor-pgtest-merge`, 기본 55432 는 점유). `nonCanonicalSpellings` 는 `asset_split_validation_test.go` 에, `countRows`/`assertMergeWroteNothing`/`assetStatus` 는 `asset_merge_validation_test.go` 에, `sourceOwner`/`insertTestAssetSource` 는 split 쪽에, `mcpAssetGetReplyFor` 는 `mcp_asset_get_merged_test.go` 에 있다 — 같은 이름을 다시 정의하지 말 것.
- [러너 02:48] brief accepted — 채택 — 지목한 파일·행·헬퍼·검증 명령이 모두 들어맞았고, '미확인' 으로 남겨 둔 PostgreSQL 의
- [러너 02:49] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 판정 approve. 구현 노트에 `- 실패 재현:` 줄이 없어 직접 재현했다 — `assets.go` 만 `c2d700c^` 로 되돌려 SQLite(대문자 2건 '400 want 200' 실패)와 PostgreSQL(4건 전부 실패: urn → 500 INTERNAL_ERROR, 중괄호·하이픈없는32자 → 200 실제 병합, 대문자 → `asset_get` 'asset not found')에서 커밋 메시지의 주장과 글자까지 일치하는 실패를 얻었다. 수정 후 양쪽 통과, gofmt·vet·build 통과, 작업 트리 원상복구.
- 못 본 것: 콘솔(web)·Rust 쪽은 이 회차가 손대지 않아 돌리지 않았고, `go test ./...` 전체는 PostgreSQL 쪽만 `-run Merge|Split|MergedInto` 로 좁혀 돌렸다(SQLite 는 전체가 아니라 같은 좁은 집합).
- 승인이어도 남는 우려 — **릴리즈 노트에 양방향 동작 변화를 적을 것**: PostgreSQL 에서 지금까지 200 이던 `{중괄호}`·하이픈 없는 32자 철자가 이제 400 이고(openapi `format: uuid` 안쪽이지만 그 철자로 호출하던 클라이언트는 깨진다), SQLite 폴백에서 400 이던 대문자 UUID 가 이제 200 으로 병합된다.
- 다음 회차용: `mergeAssets` 루프 안 `s.internalError` 4곳(assets.go:763·777·786·801 부근)의 tx 교착 경로는 그대로이고, assets.go:751 의 `missing != ""` 분기도 `tx.Rollback()` 선행이 없다(현재 `writeAPIError` 는 진단 기록을 안 해 무해). 병합 루프 `rows.Err()` 도 여전히 무검증.
- `missingAssetID` 의 모양 검사 제거는 호출자가 assets.go:738 하나뿐임을 확인해 지금은 안전하나, 두 번째 호출자가 생기면 방언 발산이 조용히 돌아오는 전제다(주석에만 남아 있음).
- [러너 02:53] review approved — 리뷰 승인 (risk=low)
- [러너 02:53] pr created — https://github.com/hkjang/invenqor/pull/29
- [러너 02:59] ci passed — 검사 11개 모두 success
- [러너 02:59] merge done — c2d700c
- [러너 03:19] release published — v0.2.41
- [러너 03:19] gh-release created — GitHub Release v0.2.41
- [러너 03:19] manifest ok — ADMIN_GUIDE.md ADMIN_GUIDE.pdf API_MCP_GUIDE.md API_MCP_GUIDE.pdf EXECUTIVE_REPORT.md EXECUTIVE_REPORT.pdf RELEASE_NOTES_v0.2.41.md SERVER_INSTALLATION.md SERVER_INSTALLATION.pdf USER_GUIDE.md USER_GU
- [러너 03:20] assets uploaded — 29개
- [러너 03:20] assets verified — v0.2.41 자산 29개 (이전 v0.2.40: 29)
