# PR 처리기 노트 2026-09-28-131531-moina-shepherd — moina PR #34
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-28-120215-moina-improve)
# 회차 노트 2026-09-28-120215-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:02] base pinned — main@d23f75d
- [러너 12:02] autonomy release — 

## 구현 노트
- 무엇/왜: `resolveReport`(admin.go)가 reports UPDATE 실패를 404 "신고를 찾을 수 없습니다"로 감추고 `moderation_actions` INSERT 오류를 `_, _ =`로 버려 "처리됨인데 제재 기록 없음"을 남기던 것을, 같은 파일 `softDeletePost`의 관례대로 한 transaction으로 묶고 500 `storage_error`로 분리했습니다(5915beb). 프로덕션 파일 1개.
- 확신 없는 곳·검증 못 한 것: (1) `defer tx.Rollback(r.Context())`는 `softDeletePost`와 같은 형태지만, 클라이언트가 끊어 request context가 이미 취소된 상태에서 rollback이 어떻게 처리되는지는 테스트하지 않았습니다(pgx가 커넥션을 폐기하므로 안전하다고 봤을 뿐). (2) 성공 경로가 pool 직접 Exec 2회에서 transaction 1회로 바뀌어 커넥션을 조금 더 오래 붙잡습니다 — 부하 영향은 측정하지 않았습니다. (3) `s.audit(...)`이 여전히 검토 메모 원문(최대 2000자)을 audit metadata로 넘깁니다 — 운영자 지침의 "감사에 원문 넘기지 말 것"에 걸릴 수 있으나 기존 동작이고 이번 범위 밖으로 뒀습니다.
- 일부러 하지 않은 것: `followTopic`(social.go:282)의 같은 형태 통합 404 — 2026-09-25 `972113f`가 verify-failed로 main에 못 들어온 원인이 기록에 없어 손대지 않았습니다. OpenAPI `responses` 목록도 늘리지 않고 `description` 한 줄씩만 더했습니다(3경로). 프런트·e2e 무변경.
- 다음 역할이 조심할 것: 새 테스트 `admin_report_resolve_postgres_integration_test.go`는 `MOINA_TEST_POSTGRES_DSN`이 없으면 조용히 `t.Skip` 합니다. 증거로 쓰려면 throwaway `postgres:17-alpine` DSN을 주고 `--- SKIP` 0줄을 확인하세요(이번 회차 확인값: 전 패키지 ok, 최상위 `TestPostgreSQL*` 37건 PASS). 테스트가 `reports`·`moderation_actions`에 테스트 전용 트리거를 만들고 Cleanup에서 DROP 하므로, 중간에 강제 종료되면 그 DB에 트리거가 남습니다(sentinel id만 거부하므로 다른 테스트에는 영향 없음).
- [러너 12:25] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: base를 `d23f75d`로 다시 잡고(로컬 `main` ref가 낡아 `git diff main...HEAD`는 무관한 79파일을 보여줍니다 — 실제 변경은 admin.go·openapi description 3줄·새 테스트뿐) 새 테스트를 throwaway `postgres:17-alpine`에 돌려 6/6 PASS·`--- SKIP` 0줄, 별도 worktree의 수정 전 코드에는 같은 테스트가 `404 기대=500`·`200 기대=500`으로 실패함을 직접 재현했습니다. 커밋 메시지의 실패 주장과 일치합니다. `go test -race ./...` 전 패키지 ok, `go vet`·`gofmt`·`make check`(120 route) 통과. staticcheck는 미설치로 미실행.
- 구현 노트의 미검증 3건은 모두 결함 아님으로 판단: rollback 형태는 softDeletePost(admin.go:306)와 동일하고 ctx 취소 시 pgx가 커넥션을 폐기해 잠금이 남지 않음, 잠금 순서가 reports→moderation_actions 한 방향뿐이라 교차 deadlock 없음, audit 원문 전달은 기존 동작. `moderation_actions.action`에 CHECK가 없어 dismiss가 상시 500이 되는 회귀도 배제했습니다.
- 못 본 것: 실제 부하에서의 커넥션 점유 영향, frontend/e2e(무변경이라 미실행), staticcheck.
- 승인이어도 남는 우려(릴리즈 노트·다음 회차): (a) `reports.resolution`·`moderation_actions.reason`은 retention.go sweep 대상이 아니어서 최대 2000자 운영자 메모가 보존 기한 없이 남습니다 — 구현 노트 (3)의 audit metadata 건과 같은 묶음으로 보존 정책 과제 후보. (b) 새 500 출구 4곳이 pg 오류를 버려 운영자 로그에 원인이 없습니다. (c) PATCH description이 400 사유 중 빈 메모·2000 rune 초과를 빼먹었습니다(alias 2곳은 언급).
- 되돌리기: migration·외부 상태 없음. 프로덕션 파일 1개 revert로 완전 복원.
- [러너 12:29] review approved — 리뷰 승인 (risk=low)
- [러너 12:29] pr created — https://github.com/hkjang/moina/pull/34
- [러너 12:31] ci failed — 성공이 아닌 검사: Source tests=failure
