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
