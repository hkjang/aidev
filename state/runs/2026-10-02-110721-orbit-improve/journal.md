# 회차 노트 2026-10-02-110721-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:07] base pinned — main@3aceddd
- [러너 11:07] autonomy release — 

## 정찰 노트
- `internal/` 의 `rows.Next()` 루프 28곳을 `rows.Err()` 검사와 맞대 보니 가드가 없는 곳은 `mcp.go:151` 과 `auth.go:155` 둘뿐이었다 — 후자는 보호 경로·401 fail-closed 라 빼고 전자를 골랐다. v0.7.6 이 REST 다섯 곳에 같은 가드를 넣어 머지됐고 증명 기법(`settings_db_test.go:39 breakUsersRowStream`)까지 저장소에 남아 있어, 가치·위험·증명 수단이 모두 확인된 유일한 후보였다.
- 차선(ai.go SSE 파서 회귀 시험)과 신규 후보(safeAIError 의 바이트 절단, export.go 의 미완결 JSON)는 결함을 증명하지 못했거나 사용자 영향이 작아 가치 2 로 뒀다. 일곱 회차 연속 차선이던 openapi.go 렌더 시험은 가치 신호 소진으로 `rejected` 로 내렸다.
- 과제서에서 추측으로 적은 것: ① `ALTER TABLE people RENAME` 이 FK(relationships·interactions·memories·person_links)와 함께 옮겨 간다는 전제 — 미확인이며, 막히면 `relationships`/`relationship_label` 로 물러서는 길을 적어 뒀다. ② 잘린 결과가 0행일 것이라는 예측 — v0.7.6 의 블로킹 Sort 교훈에 근거하지만 행 수에 의존하는 단정은 쓰지 말라고 못 박았다.
- 구현자가 조심할 것: 뷰 DDL 을 `t.Cleanup` 으로 반드시 원복(안 하면 같은 컨테이너의 다른 DB 시험이 전부 깨진다), `t.Parallel()` 금지, 사람 시드마다 `relationships` 행 필수, 스캔 오류를 덮지 않도록 `if err == nil { err = rows.Err() }` 로 쓸 것.
- 이번 정찰은 코드를 바꾸지 않았고 `go test ./...` 기준선 초록만 확인했다. 프로필은 v0.7.4 기준으로 낡아 있어 `profile.md` 를 새로 썼다.
- [러너 11:16] scout timeout — 단계 제한 시간 초과
- [러너 11:16] scout done — MCP `tools/call orbit_search_people` 이 `rows.Err()` 를 보지 않아 끊긴 사람 검색 결과를 성공으로 돌려주는 것을 고�

## 구현 노트
- `internal/server/mcp.go` 의 `orbit_search_people` 루프 뒤에 `if err == nil { err = rows.Err() }` 만 넣었다(주석 포함 8줄). 끊긴 행 스트림이 `isError` 없는 `structuredContent` 성공 응답으로 나가던 것을 기존 매핑(224~231줄)이 받게 한 것뿐 — SQL·시그니처·`escapeLike`·새 오류 코드 없음. 프로덕션 1파일, 테스트 신규 1파일(`mcp_db_test.go`), 커밋 1개.
- 확신 없는 곳: 잘린 결과가 **0행**인 것은 `ORDER BY r.importance DESC` 의 블로킹 Sort 에 달린 질의 계획 특성이라, 다른 postgres 버전·통계에서는 1행 이상 나올 수 있다. 그래서 시험은 행 수를 단정하지 않고 `isError` 만 본다. 또 `people` 을 뷰로 가리면 `relationships`/`interactions`/`memories`/`person_links` 의 FK 가 `people_probe` 를 따라가는데, 이번 실행에서 rename·cleanup 모두 성공하고 `pg_class` 로 `relkind='r'` 복구와 `people_probe` 부재를 확인했지만 FK 이동 자체를 `pg_constraint` 로 들여다보진 않았다.
- 일부러 하지 않은 것: `auth.go:155` 의 같은 누락(보호 경로·401 fail-closed, 과제서가 범위 밖으로 지정), 나머지 MCP 도구의 uuid 모양 가드, 차선 후보(ai.go SSE 파서 시험). 문서는 손대지 않았다 — 외부 계약(HTTP 200 + `isError:true`)이 바뀌지 않았다.
- 다음 역할이 조심할 것: `mcp_db_test.go` 의 두 시험은 **DSN 이 있어야 돈다**(`ORBIT_TEST_DATABASE_URL`). CI 에 postgres 가 없어 거기서는 SKIP 되므로, 검증하려면 격리 컨테이너에 DSN 을 직접 줘야 한다(이번 실행: `postgres:16-alpine`, 포트 55617 — 이미 중지·제거됨). 두 시험은 `people` 을 전역으로 가리므로 `t.Parallel()` 을 붙이면 같은 패키지의 다른 DB 시험이 전부 깨진다. `TestMCPSearchPeopleProbeRestoresSchema` 는 앞 시험의 `t.Cleanup` 이 돌았는지 보는 사후 검사이므로 파일 내 순차 실행에 의존한다.
- [러너 11:20] brief accepted — 채택 — 지정한 자리(`mcp.go` 151~160줄의 구조, 224~231줄의 매핑, `requestHasScope` 의 authInfo 부재 시 true, people 13컬럼)가 지금 코�
- [러너 11:21] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 가드 한 줄만 `_ = rows.Err()` 로 되돌려 격리 postgres(포트 55723, 검토 후 제거)에서 직접 재현했다 — `mcp_db_test.go:142: isError = <nil>(0행), want true` 로 빨개지고 원복하면 초록. 원장의 실패 재현 줄과 증상·줄번호가 일치하고, 시험은 실제로 바뀐 경로를 지난다.
- 구현자가 미확인으로 남긴 두 자리를 봤다: FK 동반 이동은 탐침 뷰 상태의 JOIN 질의와 users 삭제 CASCADE 가 정상 동작해 간접 확인(`pg_constraint` 직독은 안 함), 0행 예측은 그대로 재현. DSN 있는 `-race` 패키지 전체 초록, 끝난 뒤 users/people/relationships 0/0/0, people·users 모두 relkind='r'.
- 승인이어도 남는 우려(차단 아님 — 운영자 고의 오설정 필요, v0.7.6 의 breakUsersRowStream 과 같은 기존 관례): 이 시험은 DSN 이 가리키는 DB 의 `people`(email_cipher·phone_cipher·note_cipher 보유)에 RENAME + 13컬럼 손실성 뷰를 건다. RENAME 과 `t.Cleanup` 사이에서 죽으면 개인정보 테이블이 뷰 뒤에 가려진다. 다음 회차 후보: DSN 표식 없으면 탐침 시험 Skip.
- `TestMCPSearchPeopleProbeRestoresSchema` 는 `-run` 단독·`-shuffle` 에서 무조건 PASS 한다(거짓 실패는 불가). 이 PASS 를 원복 증거로 읽으려면 패키지 전체 실행이어야 한다.
- 릴리즈 노트용: 외부 계약(HTTP 200 + `isError:true`) 불변, 마이그레이션·VERSION·의존성·문서 변경 없음, 단일 커밋 revert 로 완전 복구. 남은 같은 구멍은 `auth.go:155` 하나(의도적 범위 밖).
- [러너 11:24] review approved — 리뷰 승인 (risk=low)
- [러너 11:24] pr created — https://github.com/hkjang/orbit/pull/17
