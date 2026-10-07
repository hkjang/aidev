# 회차 노트 2026-10-08-033829-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:38] base pinned — main@d7e01c6
- [러너 03:38] autonomy release — 

## 정찰 노트
- 골랐다: REST 두 핸들러의 `err != nil || !exists` 분리 + 빠진 `looksLikeUUID` 가드. `grep -n 'err != nil || !exists'` 가 data.go:505·workflow.go:75 **딱 두 곳**만 집어내고, v0.7.10 이 mcp.go:337 의 같은 자리를 이미 같은 방식으로 갈라 사람 손 없이 릴리즈된 선례가 있다. 형제 핸들러 7곳이 쓰는 404/400 리터럴을 그대로 복사하므로 새 어휘가 0이다.
- 제친 이유: docs/API.md 문서화(다섯 회차 연속 차선)는 가치 신호가 약해 또 밀렸고 — API.md 를 직접 읽어 /personal/export 설명이 없음은 확정했다 — openapi.go 시험·list_memories limit·orbitAt 계열은 각각 공백 보강/성능 실측 부재로 기존 보류 사유가 그대로 유효했다.
- 추측으로 적은 것: 기준 3(DB 장애 → 500 + slog)을 **실측하지 않았다**. 코드 경로만 읽었다. 근거는 2026-10-05 가 mcp.go 의 동일 분리를 실 DB 로 증명한 것뿐이다. DSN 이 있으면 `breakPeopleUserIDLookup`(mcp_db_test.go) 방식으로 실측하되 `display_name` 이 아니라 `user_id` 를 깨뜨려야 한다(그 회차의 실측 교훈).
- 구현자가 조심할 것: 모양 가드를 `validateInteractionInput` **앞**으로 올리지 말 것(기준 4(c) 가 순서를 고정한다), 모양이 어긋난 personID 의 상태 코드를 404→400 으로 "개선" 하지 말 것, createMemory 의 모양 가드는 `in.PersonID != ""` 블록 **안**에 둘 것(빈 값은 "사람 없는 기억" 이라 통과해야 한다).
- 기준선: `go test -count=1 ./...` 초록(config·secure·server·scripts ok, DB 시험 SKIP), 작업 트리 깨끗, 코드 변경 없음.
- [러너 03:42] scout done — REST 두 핸들러(`createInteraction`·`createMemory`)가 사람 조회의 **DB 장애를 "사람을 찾을 수 없습니다"** 로 덮는 

## 구현 노트
- `createInteraction`(data.go)·`createMemory`(workflow.go)의 `SELECT EXISTS` 를 `err != nil` / `!exists` 두 분기로 가르고(DB 장애가 404/400 으로 덮여 slog 에 원인이 안 남던 것), 같은 묶음으로 `looksLikeUUID` 가드를 깔았다 — 가드 없이 분기만 가르면 22P02 가 500 으로 승격되어 회귀가 된다. 프로덕션 2파일, SQL·인자·공유 함수 무변경.
- 확신 없는 곳: 없음. 과제서가 "선택·실측 미확인" 으로 남긴 기준 3 까지 실 postgres(포트 55701)로 실측했다 — 두 핸들러 모두 500 + 로그에 22P02 확인, 되돌리면 404/400 으로 빨개지는 것까지 확인. `breakPeopleUserIDLookup` 은 `user_id` 를 깨뜨린다(display_name 은 플래너가 가지치기한다 — 2026-10-05 교훈 그대로 맞았다).
- 일부러 하지 않은 것: 메시지·상태 코드 "개선"(기준 1·2 가 바이트 동일을 요구), `looksLikeUUID` 호출 네 자리를 묶는 헬퍼 리팩터, 가드를 `validateInteractionInput` 앞으로 올리기(기준 4(c) 가 순서를 고정), `assertAPIError` 수정(공유 헬퍼라 새 `assertAPIErrorMessage` 를 따로 더했다).
- 다음 역할이 조심할 것: `data_person_db_test.go` 2개 하위 시험은 **DSN 이 있어야 돈다**(없으면 `openTestStore` 가 SKIP — CI 에서는 SKIP 된다). 나머지 신규 하위 시험 14개는 DB 불필요로 CI 에서 실제로 돈다. 그 DB 시험은 `slog` 기본 로거(전역)를 갈아 끼우므로 `t.Parallel` 금지 — `t.Cleanup` 으로 원복한다.
- 테스트에서 `1' OR '1'='1` 는 공백 때문에 경로에 그대로 못 넣는다(`httptest.NewRequest` 가 패닉). 경로만 `url.PathEscape` 하고 라우트 파라미터에는 날값을 넣었다 — 핸들러가 읽는 것은 후자다.
- [러너 03:50] brief accepted — 채택 — 지정한 두 자리와 형제 핸들러 리터럴·테스트 헬퍼(`personRequest`·`assertAPIError`·`callWithoutStore`)가 지금 코드와 정�
- [러너 03:50] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, 차단 없음). 원장에 `- 실패 재현:` 줄이 없어 직접 세웠다 — 프로덕션 2파일만 main 으로 되돌리면 DB 불필요 하위 시험 8개가 '질의까지 내려갔다'로, 실 postgres 로는 data_person_db_test.go:95 가 404/400→500 으로 빨개진다. 두 묶음 모두 바뀐 경로를 실제로 지난다.
- 구현 노트가 '실측했다'고 적은 기준 3 을 독립 재현했다: 격리 postgres:16-alpine 신규 포트 55733, 두 하위 시험 PASS, DSN 준 `go test -race ./...` 전체 초록, 끝난 뒤 people 은 BASE TABLE 로 원복·users/people 0행(뷰 오염 없음). gofmt·vet·DSN 없는 전체도 초록.
- 못 본 것: web 쪽 vitest/vite build 는 돌리지 않았다(Go 전용 변경이고 web/src 에 invalid_person·not_found 분기가 0개임은 grep 으로 확인). openapi.go·docs 는 이 두 코드에 대한 약속이 없어 문서 불일치가 생기지 않는다.
- 승인이어도 남는 우려(릴리즈 노트 후보): 중괄호·하이픈 없는 32자 16진수 person id 가 이전에는 postgres 캐스팅으로 통했으나 이제 404/400 이다. looksLikeUUID 의 문서화된 관례이고 같은 가드가 이미 10곳이며 프런트는 서버 발급 uuid 만 보내므로 실무 영향은 없지만, 외부에서 직접 REST 를 치는 사용자가 있다면 알아야 한다.
- 다음 회차: data_person_db_test.go:30 userRequest 가 data_test.go:105 personRequest 와 userID 선택만 다른 중복이다. 세 번째 변형이 나오면 personRequest 쪽으로 접을 것.
- [러너 03:54] review approved — 리뷰 승인 (risk=low)
- [러너 03:54] pr created — https://github.com/hkjang/orbit/pull/22
- [러너 03:56] ci passed — 검사 1개 모두 success
- [러너 03:56] merge done — 5cf61a2
