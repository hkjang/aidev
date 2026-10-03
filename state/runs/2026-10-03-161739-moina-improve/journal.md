# 회차 노트 2026-10-03-161739-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@8d6f8e3
- [러너 16:17] autonomy release — 

## 정찰 노트
- `972113f` 복원을 골랐다: 이미 TDD·검증을 통과한 커밋이 남은 마지막 유실분이고, 프로덕션 파일 1개(social.go 282줄)로 끝나며 HEAD와 hunk 교차가 없음을 `git diff --stat 972113f~1 HEAD`로 확인했다. 차선이던 "500 출구 구조화 로그"는 slog 관측 테스트 설계가 선행이라 45분에 위험하고, 접근성 catch·CI Go 요약은 실제 실패 재현(make image)이 선행이며 workflow 보호 경로에 가까워 제쳤다.
- 확신 없는 곳: 실제 `git cherry-pick`·`git apply --check`를 돌리지 못했다(코드 변경 금지 + 권한 거부). 무충돌은 hunk 위치 비교에 근거한 추론이다.
- 구현자 주의: social.go:271의 `decodeOptionalJSON`을 972113f 시점의 `decodeJSON` 게이트로 되돌리지 말 것. 둘은 충돌하지 않는다(ContentLength==0 → server.go:729가 즉시 true).
- e2e 전체에 `follow` 문자열이 0건이므로 2026-09-25 verify-failed(image 잡 API and browser smoke)는 이 변경과 인과가 없다. 같은 곳에서 또 걸리면 별건으로 보고할 것.
- [러너 16:21] scout done — 유실된 마지막 커밋 `972113f` 복원 — `followTopic`의 저장 실패를 404 `not_found`에서 500 `storage_error`로 분리 (가�

## 구현 노트
- `social.go:282`의 `err != nil || RowsAffected()==0` 합친 404를 두 분기로 분리했다 — Exec 오류는 500 `storage_error`("Topic을 Link할 수 없습니다"), 0행은 기존 404·기존 문구. 유실 커밋 `972113f` 복원이며 프로덕션 파일 1개 + 테스트 1개 + openapi `description` 1줄.
- TDD 순서를 지켰다: 테스트 파일만 먼저 복원해(`social.go`는 HEAD 그대로) red를 돌려 **저장 실패 1케이스만** FAIL(`:201 저장 오류가 감춰졌습니다: 404 {not_found …}`), 나머지 4케이스 PASS를 눈으로 확인한 뒤 고쳐 5/5 PASS.
- 확신 없는 곳·검증 못 한 것: 원격 CI(source/image 잡)는 못 돌렸다 — 로컬에서 CI source의 네 단계만 재현했고 `make image`·e2e·`make package`는 **미실행**이다(`e2e/`에 `follow` 문자열이 0건이라 이 변경이 e2e 경로를 지나지 않는다는 판단으로 생략했으며, 이 grep은 정찰 기록을 따랐고 이번에 재실행하지 않았다). frontend lint/vitest도 프런트 무변경이라 미실행이다.
- 일부러 하지 않은 것: `decodeOptionalJSON`(server.go:728)을 `972113f` 시점의 `decodeJSON` 게이트로 되돌리지 않았다 — `git diff 972113f -- social.go`가 그 한 줄만 보여 주며, 본문 없는 첫 케이스가 200으로 통과하는 것으로 두 커밋이 충돌하지 않음을 출력으로 확인했다. OpenAPI `responses` 목록도 관례대로 늘리지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 **DB가 있어야 돈다**(`MOINA_TEST_POSTGRES_DSN`, 없으면 `t.Skip`). 검증은 throwaway `postgres:16-alpine`에서 했고 `--- SKIP` 0줄·`TestPostgreSQL*` PASS 43→44로 실제 실행을 확인했다. 저장 실패는 sentinel `user_id`만 거부하는 테스트 전용 `BEFORE INSERT` 트리거로 만들고 `t.Cleanup`에서 `DROP TRIGGER`/`DROP FUNCTION`하므로 공유 DB에 잔존물이 없다.
- 2026-09-25에 이 커밋의 PR이 verify-failed한 원인은 image 잡 `API and browser smoke`였고 이 변경과 인과가 없다. 같은 곳에서 또 걸리면 **별건으로** 보고할 것 — 이 변경 탓으로 단정하지 말 것.
- [러너 16:25] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했습니다(`social.go:282`의 합친 조건, `unfollowTopic`의 500, 테스트가 쓰는 `N
- [러너 16:25] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 승인. 심사 범위는 `git diff 8d6f8e3..HEAD` 3파일이다(로컬 `main` ref가 41커밋 뒤처져 `main...HEAD`는 쓸 수 없다; origin/main == 8d6f8e3 == HEAD~1). red/green을 독립 재현했다 — base의 social.go만 되돌린 사본에서 `topic_follow_postgres_integration_test.go:201`의 `404 {not_found}` 1개 subtest만 FAIL, HEAD에서 5/5 PASS·SKIP 0. 신규 DB에서 `go test -race -count=1 ./...` 전 패키지 PASS, `make check` rc=0, `go vet` rc=0, 테스트 후 트리거·함수·row 잔존물 0건.
- **선행 결함(별건)**: 같은 DB를 재사용해 `go test -run TestPostgreSQL ./...`를 두 번째로 돌리면 `feed_postgres_integration_test.go:89` `TestPostgreSQLFlowSQLQueryBudget`가 `500 storage_error Flow 스냅샷` 으로 실패한다. 새 테스트 파일을 제거한 **base 코드로 3/3 재현**, 신규 DB에서는 base·HEAD 모두 PASS → 이 PR 탓이 아니다. 'source는 한 DB 공유' 경고와 맞물리니 다음 회차 후보.
- 남는 우려(차단 아님): 새 500 출구가 pgx 오류를 로그로 남기지 않는다(social.go의 기존 storage_error 전부가 동일, slog 0건) — 프로필의 '500 출구 구조화 로그' 과제 대상. openapi description의 '생략하면 50'은 명시적 `weight:0`에도 적용되고 400 invalid_weight는 문구에 없다(둘 다 선행 동작).
- 못 본 것: `make image`·e2e·`make package`·원격 CI(source/image 잡)와 frontend lint/vitest는 실행하지 않았다(프런트 무변경, e2e에 `follow` 0건이라는 선행 기록에 의존). 릴리즈 노트는 "Topic Link 저장 실패가 404가 아니라 500 storage_error로 보고된다"로 쓰면 되고, 마이그레이션·외부 상태 변경이 없어 revert 1건으로 복구된다.
- security·legal 차단 없음: 인가·식별자·비밀값·외부 요청·개인정보 수집/보존 변화 없음, 500 본문에 DB 오류 텍스트 없음, 프런트는 `DiscoveryPages.tsx:16,69`에서 readableError만 쓰므로 404→500이 UI를 깨지 않는다.
- [러너 16:32] review approved — 리뷰 승인 (risk=low)
- [러너 16:32] pr created — https://github.com/hkjang/moina/pull/39
- [러너 16:41] ci passed — 검사 2개 모두 success
- [러너 16:41] merge done — 3fe4190
- [러너 16:54] release published — v0.1.41
- [러너 16:59] assets verified — v0.1.41 자산 1개 (이전 v0.1.40: 1)
