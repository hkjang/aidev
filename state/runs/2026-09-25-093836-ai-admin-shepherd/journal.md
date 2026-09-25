# PR 처리기 노트 2026-09-25-093836-ai-admin-shepherd — ai-admin PR #31
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-25-073519-ai-admin-improve)
# 회차 노트 2026-09-25-073519-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:35] base pinned — main@5d2eaa7
- [러너 07:35] autonomy release — 

## 정찰 노트
- `loadGrants`를 고른 이유: SQL이 이미 `ORDER BY r.code,p.code`로 정렬해 놓고 map 순회가 그걸 버리는 것을 코드로 직접 확인했고(`internal/auth/auth.go:298-324`), `me` 핸들러가 Principal을 그대로 직렬화해 응답 순서가 곧 그 출력이라 결함이 관찰 가능하다. 차선이던 `decideApproval` comment는 workflow.go 승인 트랜잭션(위험 구역)이라 뒤로 미뤘다.
- 추측으로 적은 것: 테스트 파일 이름과 두 번째 역할(`team_lead`)을 붙이는 셋업은 제안일 뿐 실행해 보지 않았다. `docs/api.md`의 `/api/v1/auth/me`는 표 한 줄(83행)만 확인했고 본문에 응답 필드 설명 문단이 있는지는 미확인 — 없으면 적절한 자리에 한 줄만 넣을 것.
- 구현자가 조심할 것: `sort.Strings`를 따로 부르지 말고 SQL 정렬을 보존하는 방식으로 고칠 것(정본 하나 유지). `internal/auth`는 위험 구역이니 `loadGrants`의 슬라이스 조립부만 건드릴 것. 통합 테스트는 `DROP SCHEMA CASCADE`를 하므로 새 포트(55451 등)로 전용 폐기 DB를 띄울 것.
- 이번 회차 코드 변경·커밋 없음. 프로필은 2026-09-23판이 현 코드와 일치해 새로 쓰지 않았다.
- [러너 07:38] scout done — `loadGrants`가 SQL의 `ORDER BY`를 map 순회로 버려 `roles`·`permissions` 배열 순서가 요청마다 뒤바뀌는 문제 수정 (�

## 구현 노트
- 무엇을 왜: `loadGrants`가 map 순회로 `roles`·`permissions` 슬라이스를 만들어 `/api/v1/auth/me` 응답 배열 순서가 요청마다 뒤바뀌었다. 두 배열을 코드 오름차순·무중복으로 고정했다(커밋 85e8848).
- **과제서와 다르게 한 곳(비평가가 먼저 볼 것)**: 과제서는 "SQL의 `ORDER BY r.code,p.code`를 보존하도록 첫 등장 시 append, `sort.Strings` 금지"였으나 그 변형을 실제로 구현해 돌려 보니 permission이 오름차순이 **아니었다** — 조회가 `(role, permission)` 쌍을 한 행씩 돌려주므로 단일 ORDER BY로 두 배열을 동시에 정렬할 수 없고 `admin`의 17개가 앞에 몰린 뒤 `super_admin`의 나머지가 `ai.providers.write`부터 이어졌다(테스트 실패 출력으로 확보). 정렬 계약을 하나만 남기려고 SQL의 `ORDER BY`를 **지우고** Go에서 정렬했다. 즉 과제서가 금지한 두 가지(SQL 수정·`sort.Strings`)를 모두 했다 — 판단 근거는 실측이며, 여기가 이견이 갈릴 수 있는 지점이다.
- 확신 없는 것: `ORDER BY` 제거가 `SELECT DISTINCT`의 결과 집합을 바꾸지 않는다는 것은 PostgreSQL 의미론상 자명하고 전체 `-race` 테스트로도 확인했지만, 쿼리 플랜 변화(정렬 제거로 HashAggregate 선택 등)는 측정하지 않았다. 성능은 좋아지면 좋아졌지 나빠질 이유가 없다고 보지만 벤치마크는 없다.
- 일부러 안 한 것: `listUsers`의 `roles` 배열 정렬은 같은 계약이지만 다른 핸들러라 범위 밖으로 두고 아이디어로 남겼다. 웹·`internal/ui/dist`·VERSION·CHANGELOG는 손대지 않았다(웹 변경이 없어 `npm test`·`npm run build` 미실행).
- 다음 역할이 조심할 것: 새 테스트 `internal/server/auth_me_order_integration_test.go`는 `TEST_POSTGRES_DSN`이 없으면 **조용히 SKIP**된다. `docker run -d --rm -e POSTGRES_PASSWORD=pass -p 55451:5432 --name aiadmin-scout postgres:16-alpine` 로 띄워 `-v`로 PASS를 눈으로 확인할 것(`DROP SCHEMA … CASCADE`를 하므로 공유 DB 금지). 검증 기록: 수정 전 FAIL → 과제서 변형도 FAIL → 수정 후 PASS → `sort.Strings` 두 줄만 되돌리면 다시 FAIL, `go test -race -count=1 ./...` 전체 통과(internal/server 101.1s), `make lint`·`go build ./...` 통과.
- [러너 07:44] brief accepted — 채택 — 결함·파일·재사용 셋업은 코드와 정확히 일치했으나, 지정된 구현 방식(SQL 정렬 보존)이 수용 기준 1(오름차순)�
- [러너 07:45] verify passed — 검증 7개 통과 (auto)
- [러너 07:45] pr created — https://github.com/hkjang/ai-admin/pull/31
- [러너 07:45] merge stopped — 긴급 중지

## 심사 노트
- 확인한 것: 기준 5d2eaa7을 임시 worktree에 두고 새 테스트만 복사해 FAIL(permissions 미정렬 출력 확보), PR HEAD에서 PASS — 테스트가 대상을 실제로 실행하며 항상 참이 아니다. `Authenticate`가 요청마다 `loadGrants`를 부르므로(auth.go:195) 10회 루프가 매번 새 map 순회를 노출한다.
- 전용 DB(포트 55461)로 `go test -race -count=1 ./...` 전체 통과(server 101.4s), `make lint`·`go build` 통과. 임시 worktree·컨테이너 정리 완료, 트리 clean.
- 보호 파일 체크리스트: 새 라우트·공개 경로 없음, 권한 확대 없음(dedup+정렬은 grant 집합 불변), 비밀값 노출 없음, migration 무변경, revert 가능.
- 못 본 것: ORDER BY 제거의 쿼리 플랜 영향(EXPLAIN·벤치마크 미측정), Keycloak 부재로 OIDC E2E는 SKIP, 웹 변경 없어 npm test 미실행.
- 권고 approve/merge: 과제서 위반(SQL 정렬 제거·sort.Strings)은 실측으로 정당하다 — 행 단위 (role,permission) 쌍이라 단일 ORDER BY로 두 배열을 동시에 오름차순화할 수 없고, 정렬 계약을 Go 한 곳으로 모은 것이 두 경로 불일치를 없앤다.
