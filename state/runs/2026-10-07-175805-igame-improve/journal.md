# 회차 노트 2026-10-07-175805-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:58] base pinned — main@4e7911e
- [러너 17:58] autonomy release — 

## 정찰 노트
- 두 회차 연속 차선으로 밀렸던 페이지 tiebreak 를 골랐다. 이전에 M 이던 것을 admin.go 한 파일(세 ORDER BY)로 좁혀 S 로 내렸고, 재사용할 테스트 하비스(admin_total_pg_test.go 의 adminListFixture·patchUser)가 이미 있다는 것을 열어서 확인했다. 차선(ENCRYPTION_KEY 거부 회귀)은 테스트 전용이라 가치가 낮고, 바로 앞 회차가 같은 성격(secretbox 거부 회귀)이어서 뒤로 뒀다.
- serviceLocation(8회차)과 loadAPIKeyPolicyContext(5회차)는 이전 회차의 자기 지시대로 rejected 로 내렸다 — 다음 정찰이 다시 집어 들지 말 것.
- 추측으로 적은 것: ① 동일 created_at + 페이지 사이 UPDATE 로 실제 PG17 Red 가 난다는 것은 **미검증**이다(이 환경에 PG 가 떠 있는지 확인하지 못했다). 과제서에 대안 레시피와 "Red 실패 시 견고성으로 좁혀라" 를 명시했다. ② admin.go:571/709/746 의 행 번호는 이번에 직접 읽어 확인했다. ③ catalog.go:389-392(playWindows)와 README 42/44행은 재확인하지 않은 이전 근거다.
- 구현자가 조심할 것: gameSelect 상수 본문을 바꾸면 공개 게임 목록까지 번진다 — adminListGames 의 ORDER BY 절만 바꿀 것. pagedTotal/감사 CSV 공유 술어는 2026-10-04 의 계약이므로 손대지 말 것. DSN 없는 go test 는 DB 검증이 아니다.
- [러너 18:03] scout done — 관리자 목록 페이지네이션에 유일한 tiebreak 을 넣어 OFFSET 경계에서 행이 겹치거나 사라지지 않게 하기 (가

## 구현 노트
- `listUsers`·`listAuditLogs`·`adminListGames` 의 `ORDER BY created_at DESC` 에 유일 `id` 를 덧붙였다(프로덕션 1파일 = admin.go, + 테스트 1 + docs 1줄). `gameSelect` 상수·`usersFilter`/`auditLogFilter`·`pagedTotal`·감사 CSV 는 손대지 않았다.
- 과제서와 다르게 한 두 곳(비평가가 먼저 볼 곳): ① Red 레시피 1 의 "아직 보지 못한 행 PATCH" 로는 Red 가 안 났다(첫 시도 PASS). UPDATE 는 힙 튜플을 **끝**으로 옮기므로 **이미 본 행**을 PATCH 해야 중복/누락이 난다 — psql probe 로 확인하고 테스트를 그렇게 고쳤다. ② games 를 레시피 2(limit=1 순회)로 쓰면 **flaky** 였다(단독 녹색/합동 Red = top-N heapsort bound·통계 타이밍 의존). 버리고 users 와 같은 힙 이동 방식(동일 created_at 6행 + 실제 `PUT /api/v1/admin/games/{id}`)으로 다시 써 5/5 안정 Red 를 확인했다.
- 확신 없는 곳: audit 테스트만 아직 bound 의존 레시피(limit=1)를 쓴다 — 프로덕션에 audit UPDATE 경로가 없어 힙 이동을 못 쓴다. 이 환경(PG17-alpine, 기본 work_mem)에서 5/5 안정 Red 였지만 **다른 PG 설정에서 Green 으로 바뀔 가능성은 배제하지 못했다**. 그래도 Green 쪽(수정 후)은 안정적이므로 CI 를 깨뜨리는 방향의 위험은 아니다. 감사 행은 한 문장으로 직접 INSERT 한다(프로덕션이 트랜잭션당 1행만 쓰므로 불가피) — 읽기 경로는 실제 엔드포인트다.
- 일부러 하지 않은 것: `extended.go:581`·`catalog.go:130`·`catalog.go:990` 의 같은 tiebreak (프로덕션 1파일 유지, ideas.json 후속 항목). 새 인덱스·마이그레이션·환경변수 없음. 버전/릴리즈 관련 파일 무수정.
- 다음 역할이 조심할 것: 신규 3개 테스트는 **PG 가 있어야 돈다**(IGAME_TEST_DSN 없으면 조용히 skip — DSN 없는 `go test ./...` 는 검증이 아니다). 검증용 PG 는 README 절차로 `igame-test-db` 컨테이너를 띄워 썼다; 세션 종료 시 `docker rm -f igame-test-db` 로 지울 것. 실행 후 `api_test_`/`migrate_` 잔여 스키마 0개 확인했다.
- 도달성 표현 주의: "데이터 유실 버그" 가 아니라 페이지 경계 결정론 수정이다. 단 `adminListGames` 는 갓 설치한 상태에서 이미 동점이라(마이그레이션이 카탈로그를 한 문장으로 seed) 9행 중 3행 누락을 실측했다 — 과제서 추정보다 도달성이 높다.
- [러너 18:17] brief accepted — 채택 — 근거가 지금 코드와 정확히 일치했고(세 ORDER BY, `gameSelect` 의 `g` 별칭과 `g.id` 선택, `audit_logs.id` bigserial) 지정한 �
- [러너 18:17] verify passed — 검증 4개 통과 (policy)

## 비평 노트
- 일회용 PG17-alpine(포트 15433, 전용 pgcrypto 스키마)을 띄워 러너가 못 돌린 Go 를 직접 검증했다: 새 세 테스트가 HEAD 에서 9/9 PASS, 테스트 파일만 main 에 올린 임시 worktree 에서 9/9 FAIL — 구현 노트가 의심한 audit 의 limit=1 bound 레시피도 3/3 안정 Red 였다. `make test-db` 전체 PASS, `go vet` 무출력, 잔여 스키마 0개. 임시 worktree·컨테이너는 제거했다.
- reject 사유는 프로덕션 코드가 아니라 docs/api.md:10 한 줄이다. 수리가 먼저 볼 파일은 **docs/api.md:10** 뿐이고 admin.go 는 손댈 것이 없다.
- 첫째, 그 줄은 `## 공통 규칙`(모든 limit/offset 목록에 적용)에 있는데 catalog.go:130(`ORDER BY g.name`, games.name 은 UNIQUE 아님)·catalog.go:990·extended.go:581 은 아직 유일 키가 없다. 둘째, "offset 을 끝까지 넘기면 각 행이 정확히 한 번씩" 은 고친 세 목록에서도 거짓이다 — 페이지 사이 INSERT/DELETE 는 여전히 창을 흔든다(테스트가 Green 인 것은 교란을 UPDATE 로 넣었기 때문).
- 보지 못한 것: web/SDK 쪽 영향(이번 diff 에 없음), 큰 감사 테이블에서의 깊은 OFFSET 실측. 보안·법무 차단 사유 없음(정적 SQL 추가, 새 입력·권한·개인정보 없음).
- 릴리즈 노트가 알아야 할 것: 수정 범위는 관리자 users/audit/games 세 목록이며 공개 게임 목록·내 세션·점수 모더레이션은 다음 회차 몫이다. revert 는 깨끗하다(마이그레이션·외부 상태 없음).
- [러너 18:23] review rejected — 리뷰 거절: docs/api.md:10 이 새 보증을 `## 공통 규칙`(모든 목록에 적용되는 preamble, 8행 "목록은 limit/offset 을 사용합니다" 바로 아래)에 넣었으나 코드가 �

## 수리 노트
- 지적 두 개 모두 맞았다: catalog.go:130 은 `ORDER BY g.name` 인데 migrations/001_initial.sql 에서 `name` 은 UNIQUE 가 아니고(slug 만), catalog.go:990·extended.go:581 도 유일 키가 없다 — `## 공통 규칙` 자리의 문장이 거짓이었다. 둘째 지적(스냅샷이 달라 "정확히 한 번씩" 은 세 목록에서도 거짓)도 맞다. 틀린 지적은 없었다.
- 비평가가 준 두 선택지 중 **문서 한정** 을 골랐다(세 쿼리에 tiebreak 추가는 프로덕션 범위를 넓히고 의미 있으려면 테스트 3개가 더 필요해 이 회차 목적을 벗어난다). docs/api.md:10 을 `/api/v1/admin/{users,audit,games}` 로 한정 + "다른 목록은 아직 없다" 명시 + 보증을 "동점 행의 페이지 경계 결정론" 으로 낮추고 페이지 사이 INSERT/DELETE 는 모든 목록에서 창을 흔든다고 적었다.
- 비평가가 "admin.go 는 손댈 것 없다" 고 했지만 admin.go:576 주석의 "Every paged list here sorts to a unique column" 이 같은 과장이어서 한 문장 범위를 좁혔다(동작 변화 없음). 프로덕션 SQL·테스트는 무수정.
- 검증: 일회용 PG17(15441) 로 `make test-db` 전체 PASS, `-run WhenTimestampsTie` 3/3 PASS, `go vet` 무출력, 잔여 스키마 0, 컨테이너 제거.
- 확신 없는 곳: 세 공개/모더레이션 목록의 tiebreak 미해결은 의도적 잔여물이다 — 문서가 이제 그것을 약속하지 않지만, 다음 회차에서 실제로 집어 들 가치는 여전히 있다(특히 동명 게임이 있는 설치의 `GET /api/v1/games`).
- [러너 18:27] repair done — # 수리 요약 (시도 2)  - 문제: docs/api.md:10 이 `## 공통 규칙`(모든 limit/offset 목록) 자리에서 두 가지를 과장했다 — ① 유일 키 tiebreak 은 admin.go 의 세 목록�
