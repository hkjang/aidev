# 회차 노트 2026-10-02-053732-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:37] base pinned — main@1d13889
- [러너 05:37] autonomy release — 

## 구현 노트
- b3fa353: 비밀번호 쓰기 뒤 `_ = DeleteUserSessions(...)`가 오류를 버려 204 + "모든 로그인 세션을 종료했습니다" 알림이 거짓이 되던 것을, `store.UpdatePasswordAndRevokeSessions`(UPDATE users + DELETE sessions 한 tx)로 묶었습니다. 프로덕션 3파일(store.go·auth.go·admin.go) + 테스트 1파일, OpenAPI 무변경(새 상태 코드 없음, route 120 유지).
- 확신 없는 곳·검증 못 한 것: (1) 새 테스트는 **DB가 있어야 돕니다** — `MOINA_TEST_POSTGRES_DSN` 없으면 t.Skip입니다. 검증은 postgres:16-alpine(16.15)에서만 했고 image 잡의 PostgreSQL 17은 미확인. (2) 테스트가 `sessions`에 BEFORE DELETE 트리거를 **전역으로** 만듭니다(sentinel 사용자 2명만 거부, t.Cleanup에서 DROP). 공유 DB 병렬 실행에서 안전하다고 보지만 CI source 동시 실행은 로컬에서만 재현했습니다. (3) 자기 비밀번호 변경 실패가 이제 500인데, 프런트는 `readableError`로 "비밀번호를 변경할 수 없습니다"를 띄웁니다 — 비밀번호가 실제로 안 바뀌므로 문구는 맞지만 UI는 실제로 띄워 보지 않았습니다(프런트 무변경이라 vitest 미실행).
- 일부러 하지 않은 것: 비활성화 경로(admin.go:178)의 `_ = DeleteUserSessions` — `SessionUser`가 `u.active`로 join하고 `authenticate`가 `!Active`를 401로 막아 거짓 보장이 아닙니다(ideas.json에 rejected 이유 기록). API 키는 비밀번호 변경 뒤에도 살아 있지만 자동 폐기는 통합을 깨므로 별도 과제로 남겼습니다. 운영자 로그 보강(SQLSTATE)도 범위에서 빼 pending으로 남겼습니다.
- 다음 역할이 조심할 것: 되돌려 검증은 "버그 상태를 그대로 단언"하는 방식으로 했고(204·세션 2개·새 비밀번호·알림 1건 → 5/5 통과) 그 임시 단언은 커밋에 없습니다. `UpdatePassword`는 삭제했으니 새 비밀번호 쓰기 경로를 추가할 때 세션 종료를 같은 tx에 두세요.
- [러너 05:54] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 프로덕션 3파일만 1d13889로 되돌려 새 테스트가 4/4 FAIL(`status=204 기대=500`)하고 HEAD에서 5/5 PASS함을 직접 재현 — 테스트는 바뀐 경로를 지납니다. PostgreSQL 17.11에서도 PASS(구현자 미검증 (1) 해소). go vet 0·gofmt 무출력·`make check` exit 0(route 120)·`go test -race ./internal/httpapi ./internal/store` PASS. UpdatePassword 호출자 전수 grep, BootstrapAdmin이 기존 hash를 다시 쓰지 않음, admin.go:178 판단(store.go:205가 `AND u.active`로 join) 확인. 판정 approve, 차단 없음.
- 못 본 것: 프런트 vitest·e2e 미실행(프런트 무변경이라 SettingsPages.tsx:203 catch 경로를 코드로만 확인), CI source(PG16) 동시 실행 실환경, image 잡.
- 남는 우려 1(릴리즈 노트 후보): `DELETE FROM sessions`가 계속 실패하면 비밀번호 변경·초기화가 아예 불가능해집니다(이전에는 세션만 남기고 성공). production에 sessions 참조 FK·트리거가 없어 DB 장애 때만 발생하지만 '정직한 500' 선택임을 적어 두는 게 좋습니다.
- 남는 우려 2(다음 회차): 테스트가 `sessions`에 전역 BEFORE DELETE 트리거를 만들어 CREATE/DROP 시 ACCESS EXCLUSIVE 락을 잡습니다. 정합성은 안전하나 공유 DB에서 sessions 접근을 순간 직렬화하므로 lock-wait flake가 보이면 여기를 먼저 보세요.
- 무관 flake: `go test ./...` 4회 중 1회 TestPostgreSQLFlowSQLQueryBudget(feed_postgres_integration_test.go:89) 500 storage_error. sessions·비밀번호 무관 경로이고 단독 5회·되돌린 상태 4회에서 미재현 — 공유 DB 경합으로 보고 이번 변경 탓으로 단정하지 않습니다.
- [러너 06:01] review approved — 리뷰 승인 (risk=low)
- [러너 06:01] pr created — https://github.com/hkjang/moina/pull/38
- [러너 06:13] ci passed — 검사 2개 모두 success
- [러너 06:13] merge done — b3fa353
- [러너 06:29] release published — v0.1.40
- [러너 06:34] assets verified — v0.1.40 자산 1개 (이전 v0.1.39: 1)
