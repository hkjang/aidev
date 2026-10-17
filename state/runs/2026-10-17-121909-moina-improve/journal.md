# 회차 노트 2026-10-17-121909-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:19] base pinned — main@c9bfca2
- [러너 12:19] autonomy release — 

## 정찰 노트
- `admin.go` 25곳 이관을 골랐다. 1순위로 보였던 "미머지 커밋 복원"은 기각했다 — 그 작업은 이미 `origin/auto/2026-10-08-0138` 에 올라가 사람의 머지를 기다리는 중이고(`origin/main` 은 여전히 `c9bfca2`, `merge-base --is-ancestor` NOT), 세 번째로 같은 복원을 올리는 것은 운영자가 금지한 "같은 접근의 반복"이다. `admin.go` 는 그 브랜치(파일: social.go 1개 + 테스트 3개)와 **겹치지 않는다**.
- 프런트 `findBy` 패턴 점검(가치 4)은 더 가치가 높지만 같은 `AuthContext.test.tsx` 를 건드려 머지 대기 브랜치와 충돌하므로 미뤘다. `settings/preferences/analytics` 는 테스트 배선이 없어 위험이 더 크다.
- 실행으로 확인한 것: 25곳의 줄번호와 handler 매핑, `admin.go` 에 합친 Commit 조건 0건(남은 하나는 `workflow.go:202`), 네 곳의 `if rows.Scan(...) != nil` 인라인 오류 버림, `admin` 역할이 `posts:manage` 를 가진 것(`migrations/001_initial.sql:58`)과 라우트가 그것을 요구하는 것(`server.go:256`), `admin_report_resolve_…_test.go` 의 admin 세션·sentinel 트리거·`lockedBuffer` 배선(:34~128).
- **추측(미실행)**: 테스트를 실제로 돌리지 않았다 — `TestPostgreSQL*` 44건, `adminUpdatePost` 의 트리거 거절이 `P0001`/`*pgconn.PgError` 로 오는 것, `posts` 테이블에 새 BEFORE UPDATE 트리거를 달아도 공유 DB 의 다른 패키지 테스트가 깨지지 않는 것은 모두 과거 회차 기록에서 유도한 기대값이다. 구현자가 red 에서 리터럴 단언으로 확정할 것.
- 구현자가 조심할 것: loop 안 네 곳에서 **nil 인 Query `err` 를 넘기지 말 것**(`cause_type:"<nil>"` 이 되면 이 회차의 목적이 무너진다). `admin.go` 에서는 제어 흐름을 분할할 필요가 없으므로 리터럴 multiset 비교에서 **증감 모두 0** 이 기대값이다.
- [러너 12:24] scout done — `admin.go` 의 `storage_error` 500 출구 25곳을 `writeStorageError` 로 이관한다 (가치 3 / 위험 1 / 작업량 M)
- [러너 12:37] resume closed — PR 이전 단계에서 중단 — 에이전트 단계는 재실행하지 않음
