# 회차 노트 2026-09-29-003202-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:32] base pinned — main@a537cdb
- [러너 00:32] autonomy release — 

## 정찰 노트
- 고른 이유: `openapi.yaml:2234` 가 `POST /api/v1/assets/merge` 의 `"400": Primary or secondary assets missing` 을 공개 계약으로 적어 놓았는데 `mergeAssets`(assets.go:647) 에는 존재 확인 코드가 아예 없다 — 문서와 구현이 어긋난 자리를 문서 쪽 수정 없이 구현으로 맞출 수 있어 수용 기준이 명확하고 프로덕션 파일이 1개다. 페이지네이션·rows.Err() 후보는 web+server 를 함께 건드려 파일 수가 늘거나(전자) 실패 주입 수단이 없어 증명이 어려워(후자) 제쳤다.
- 추측으로 적은 것: 없는 secondary 는 200, 없는 primary 는 FK 로 500 이라는 실제 응답 코드를 **실측하지 못했다**. probe 테스트를 썼지만 `internal/httpapi` 첫 컴파일이 이 환경에서 10분 넘게 끝나지 않아 출력을 못 봤다(코드 기반 추론 + 두 방언 스키마의 `asset_changes.asset_id REFERENCES assets(id)` + `PRAGMA foreign_keys = ON` 확인까지만). 구현자는 반드시 수정 전 빨강을 먼저 볼 것.
- 이미 병합된 자산을 다시 secondary 로 주는 기존 동작도 미확인 — 존재 확인에 `deleted_at IS NULL` 을 넣지 말 것.
- 조심할 것: PostgreSQL 의 `assets.id` 는 UUID 타입이라 형식이 틀린 ID 는 존재 확인 쿼리 자체가 오류가 된다. 두 방언이 같은 입력에 같은 코드를 돌려주는지 end-to-end 로 확인할 것.
- [러너 00:41] scout timeout — 단계 제한 시간 초과
- [러너 00:41] scout done — `POST /api/v1/assets/merge` 가 primary·secondary 자산의 존재를 확인하지 않아, openapi 가 공개 계약으로 약속한 400 �

## 구현 노트
- `mergeAssets` 가 tx 를 연 직후 primary·모든 secondary 를 단건 `SELECT id FROM assets WHERE id=$1` 로 확인하고 없으면 아무 것도 쓰지 않고 400 `INVALID_MERGE`. 프로덕션 파일 1개(assets.go) + 새 테스트 1개.
- 정찰의 예측과 다른 실측: 없는 primary 는 500 이 **아니라** SQLite 폴백에서 응답이 영원히 안 온다. `internalError` → `recordDiagnostic` 이 새 커넥션을 요구하는데 열린 tx 가 `SetMaxOpenConns(1)`(storage/runtime.go:273)의 유일한 커넥션을 쥔다. 스택으로 확인했고 ledger 에 적었다. **이 교착은 tx 안에서 `internalError` 를 부르는 다른 핸들러(`splitAsset`, `mergeAssets` 의 나머지 오류 경로, ingest 등)에 그대로 남아 있다** — 이번 범위 밖이라 손대지 않았고 ideas.json 에 올렸다. 내가 새로 넣은 오류 경로만 `tx.Rollback()` 을 먼저 부르게 했다.
- 확신 없는 것: (1) 존재 확인 쿼리 자체가 실패하는 경로(`missingAssetID` 의 `err != nil`)는 테스트로 지나가지 못했다 — 실패 주입 수단이 없다. (2) `uuid.Parse` 는 대문자·중괄호·urn 형식을 통과시키므로 대문자 UUID 로 병합할 때 `after_json.secondary_ids` 가 글자 그대로 저장되는 기존 결함은 그대로다(ideas.json 에 남김). (3) 이미 병합된 자산을 다시 secondary 로 주는 경로는 여전히 통과하지만(행만 보므로) 그 동작 자체를 새로 테스트하지는 않았다.
- 일부러 하지 않은 것: `openapi.yaml` 수정(이미 400 을 약속함), `splitAsset`(별 과제), `IN` 절 한 번에 확인(방언 차이).
- 다음 역할이 조심할 것: 새 테스트 3개는 `newRuntime`/`testServer` 로 실제 라우터를 통과하므로 DB(SQLite 폴백 또는 `scripts/test-postgres.sh` 의 PostgreSQL)가 있어야 돈다. `internal/httpapi` 첫 컴파일이 이 환경에서 매우 느리다(캐시가 비면 10분+) — 타임아웃을 넉넉히. 기본 포트 55432 는 점유돼 있어 55491 을 썼다.
- [러너 00:56] brief accepted — 채택 — 근거(openapi 의 400 약속, `mergeAssets` 에 존재 확인이 아예 없음, FK·`PRAGMA foreign_keys`)가 모두 코드와 맞았고, 유일한 
- [러너 00:57] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인함: assets.go 를 a537cdb 판으로 되돌려 새 테스트 3개가 실제로 빨강임을 직접 봤다(primary 는 `panic: test timed out after 45s`, 나머지 둘은 `200 "" want 400`) — 원장의 실패 재현과 일치. 고친 판은 SQLite 와 실 PostgreSQL(포트 55493) 양쪽에서 3개 통과, `go test ./internal/httpapi/` 전체 통과. 범위는 파일 2개로 깨끗하고 인증·마이그레이션·비밀값을 건드리지 않았다.
- 못 본 것: `missingAssetID` 의 `err != nil` 경로는 실패 주입 수단이 없어 여전히 테스트가 지나가지 않는다(구현자와 같은 한계). 콘솔 UI 에서의 실제 병합 흐름은 돌려 보지 않았다.
- 승인이어도 남는 우려 (1): `uuid.Parse` 선검사가 방언 차이를 다 막지 못한다 — PostgreSQL 에서 `id='urn:uuid:<uuid>'` 는 22P02 로 죽어 500 이 되는데(실측) SQLite 는 400 이다. 회귀는 아니지만 테스트 주석의 "Both must answer the same code" 는 과한 서술이다. 대문자 UUID 도 PostgreSQL 은 찾고 SQLite 는 못 찾아 갈린다.
- 승인이어도 남는 우려 (2): 열린 tx 안에서 `internalError` 를 부르면 SQLite 폴백이 교착하는 결함이 mergeAssets 의 나머지 오류 경로(assets.go:724,737,745,761,769,773)·splitAsset·ingest 에 그대로 남아 있다. 유일한 커넥션을 잡아 서버 전체가 멎는 가용성 문제라 다음 회차 1순위로 볼 만하다.
- 릴리즈 노트: 없는 id·형식이 틀린 id 로 병합하던 클라이언트가 200 대신 400 INVALID_MERGE 를 받게 된다(openapi.yaml:2234 가 이미 약속한 계약이라 문서 변경 불필요).
- [러너 01:02] review approved — 리뷰 승인 (risk=low)
- [러너 01:02] pr created — https://github.com/hkjang/invenqor/pull/27
- [러너 01:09] ci passed — 검사 11개 모두 success
- [러너 01:09] merge done — fed2c84
- [러너 01:29] release published — v0.2.39
- [러너 01:29] gh-release created — GitHub Release v0.2.39
- [러너 01:29] manifest ok — ADMIN_GUIDE.md ADMIN_GUIDE.pdf API_MCP_GUIDE.md API_MCP_GUIDE.pdf compose.invenqor-0.2.39.yaml compose.offline.yaml EXECUTIVE_REPORT.md EXECUTIVE_REPORT.pdf invenqor-0.2.39.env.example invenqor-0.2.39
- [러너 01:30] assets uploaded — 29개
- [러너 01:30] assets verified — v0.2.39 자산 29개 (이전 v0.2.38: 29)
