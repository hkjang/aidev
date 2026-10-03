# 회차 노트 2026-10-03-181733-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:17] base pinned — main@735c26b
- [러너 18:17] autonomy release — 

## 정찰 노트
- `exportData` 의 미완결 JSON 을 골랐다 — 보류 목록에서 "결함이 실재하고 증명 수단이 이미 저장소에 있는" 유일한 항목이다(`breakPeopleRowStream`·`seedNamedPerson`·`seedRelationship` 재사용). ai.go SSE 파서 시험은 결함 없는 공백 보강이라, openapi 렌더·orbitAt contexts 는 여덟~열 회차 연속 차선이라 각각 제쳤고 뒤 둘은 이번에 `rejected` 로 내렸다. auth.go rows.Err() 는 보호 경로라 뺐다.
- 미확인(과제서에 추측으로 적은 것): export 질의(`people p JOIN relationships r … ORDER BY p.created_at`)에서 뷰 주입이 실제로 `rows.Err()` 에만 오류를 남기는지 DB 로 돌려 보지 못했다. 같은 기법이 v0.7.6·v0.7.7 에서 `users`·`people` 에 성립한 것에 기대고 있다. 안 빨개지면 `seedRelationship` 누락을 먼저 의심하고, 그래도 안 되면 차선(safeAIError 룬 자르기, DB 불필요)으로 갈 것.
- 구현자가 조심할 것: (1) 뷰에 `CASE` 가 있어 자동 갱신 가능 뷰가 아니므로 **씨앗을 먼저, 뷰를 나중에** — 순서를 바꾸면 INSERT 가 실패한다. (2) 행이 하나라도 스캔되면 nil Vault 경로(`dataKeyVersion`)에 닿으므로 정상 경로 하위 시험은 사람 0명인 별도 사용자로 부를 것. (3) 상태 코드는 200 그대로 — 헤더가 이미 나갔다.
- 기준선: `go test -count=1 ./...` 전부 ok(DSN 없음). 프로필은 1일 전이고 코드와 어긋나는 곳을 찾지 못해 다시 쓰지 않았다.
- [러너 18:22] scout done — `GET /api/v1/personal/export` 가 섹션 중간 실패 시 **파싱조차 불가능한 미완결 JSON** 을 200 으로 내보내는 것을, 

## 구현 노트
- `export.go` 섹션 루프의 오류 분기에 `array.close()` + `,"complete":false,"failed_section":%q}` 두 줄을 더해, 끊긴 내보내기 본문이 JSON 으로 남고 주석이 선언했던 계약이 실제로 성립하게 했다. 주석도 지금 코드가 하는 일로 고쳐 썼다. 상태 코드는 200 그대로(헤더가 이미 나갔다).
- 확신 없는 곳: 거의 없다. 정찰이 미확인으로 남긴 전제(export 질의에서도 뷰 주입이 `rows.Err()` 에만 오류를 남긴다)를 실제 postgres 로 확인했고, red→green→두 줄 되돌려 다시 red 까지 돌렸다. 다만 **섹션 2~4(`interactions`/`memories`/`links`)가 실패하는 경로는 시험하지 않았다** — 루프 본문이 같으므로 코드상 동일하지만 실측은 `people` 하나뿐이다.
- 일부러 안 한 것: 59~64줄 `user` 인코딩 실패 분기(쓰기 오류 = 클라이언트가 이미 끊김 → 더 써도 의미 없음), 섹션 함수 네 개·SQL·`jsonArray` 시그니처, `jsonArray.close()` 의 쓰기 오류 무시, openapi.go, docs(`/personal/export` 설명이 docs 에 아예 없어 추가는 다음 회차 후보로 `ideas.json` 에 남겼다).
- 다음 역할이 조심할 것: `internal/server/export_db_test.go` 는 `ORBIT_TEST_DATABASE_URL` 이 없으면 전부 SKIP 이다(CI 에 postgres 가 없으므로 CI 초록은 이 시험의 증거가 아니다). 이 파일은 `breakPeopleRowStream` 으로 `people` 테이블 DDL 을 바꾸므로 `t.Parallel()` 금지이고, 하위 시험 순서(씨앗 → 정상 경로 → 뷰 → 실패 경로)를 바꾸면 뷰가 자동 갱신 가능 뷰가 아니라 INSERT 가 깨진다. `failed_section` 은 추가 전용 필드이며 `complete===true` 를 보는 기존 소비자를 깨지 않는다.
- [러너 18:26] brief accepted — 채택 — 지정한 자리(export.go 75~87 의 구조, `breakPeopleRowStream`·`seedNamedPerson`·`seedRelationship` 재사용, nil Vault 때문에 0행이어�
- [러너 18:26] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 격리 postgres(55717)로 red→green 을 직접 재현했다: 고친 뒤 새 시험 3개 PASS, `export.go:86~87` 두 줄만 되돌리면 원장의 `실패 재현` 줄과 본문 접두사(`,"people":[`)까지 똑같이 FAIL. 정상 경로 하위 시험은 되돌린 코드에서도 PASS. DSN 준 `go test -race ./...`·`go vet`·`gofmt` 전부 깨끗하고 작업 트리는 원복해 뒀다.
- 구현자가 미검증으로 남긴 섹션 2~4 실패 경로는 코드로 확인했다 — 루프 본문이 섹션마다 동일하고 분기가 `section.name` 만 참조해 비대칭이 없다(실측은 여전히 `people` 하나뿐). 범위 이탈·마이그레이션·되돌리기 어려운 변경 없음, revert 는 커밋 하나로 끝난다.
- 보안: 경로는 `authenticate` 아래이고 네 질의 전부 `user_id=$1` 로만 긁으며 요청 식별자를 쓰지 않는다. 새 필드 `failed_section` 은 고정 네 값뿐 — 차단 사유 아님.
- 승인이어도 남는 우려(법무): 부분 내보내기는 평문 개인정보를 일부 반출하면서 `data.export` 감사 기록을 남기지 않고, 새 시험이 `audits != 0` 을 실패로 못 박아 그것을 의도된 동작으로 고정했다(export_db_test.go:124~134). 앞으로 부분 반출도 감사하려면 이 단언을 함께 고쳐야 한다 — 다음 회차 후보.
- 못 본 것: 웹 UI 실제 다운로드 동작, 쓰기 실패(클라이언트 단절)로 끊기는 부류(이 가드는 질의·복호화 실패만 구제한다), docs 의 `/personal/export` 설명(아예 없음).
- [러너 18:29] review approved — 리뷰 승인 (risk=low)
- [러너 18:29] pr created — https://github.com/hkjang/orbit/pull/18
- [러너 18:32] ci passed — 검사 1개 모두 success
- [러너 18:32] merge done — bc34c6d
- [러너 18:37] release published — v0.7.8
- [러너 18:38] assets verified — v0.7.8 자산 1개 (이전 v0.7.7: 1)
