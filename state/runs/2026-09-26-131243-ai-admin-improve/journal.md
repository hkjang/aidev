# 회차 노트 2026-09-26-131243-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:12] base pinned — main@dc1b8c5
- [러너 13:12] autonomy release — 

## 정찰 노트
- 골랐다: 승인 코멘트의 trim·NULL·길이 계약 통일. 최근 채택된 과제 4회가 모두 "같은 값을 읽는 두 경로의 계약 불일치" 유형이었고, 이 결함은 그 유형이면서 코드 위치(workflow.go:282 vs :290·:292)를 직접 읽어 확인했다. `available_models` JSON(M, 미확인 4곳)과 chat Content-Type(3회 연속 미확인)은 판정 비용이 커서 제쳤고, 문서만 고치는 후보 2개는 가치 1이라 차선으로 내렸다.
- 확실한 것: workflow.go:226·231·282·290·292·139의 코드, migrations 001:169·186의 text 컬럼, `nullString`(server.go:643)·`signIn`(db_error_integration_test.go:100)·`signInReviewer`(key_target_integration_test.go:134) 헬퍼의 존재, `web/src`에 `decisionComment` 참조 없음(grep 0건).
- 추측으로 적은 것: 2000자 상한 숫자는 선례 없는 내 판단이다(기존 선례는 200·160·300). 202+approvalId를 만드는 고권한 scope 이름은 key_rotate_target 테스트에서 패턴만 보고 적었으니 구현자가 실제 scope 값을 확인할 것. 다단계(approval_levels=2) 경로는 테스트하지 않아도 된다고 판단했다.
- 조심할 것: 새 검증은 반드시 `tx.Begin` 앞. `:290`과 `:292` 둘 다 고칠 것 — 한쪽만 고치면 이 과제가 없애려는 불일치를 그대로 남긴다. 테스트는 DSN 없으면 SKIP이므로 `-v`로 PASS를 눈으로 확인할 것.
- [러너 13:16] scout done — 승인 결정 코멘트(`decision_comment`)의 trim·NULL·길이 계약을 `approval_action.comment`와 일치시키기 (가치 2 / 위험 

## 구현 노트
- `decideApproval`이 한 결정을 `approval_action`과 `approval_request`에 다른 값으로 쓰던 것을 고쳤다: `comment := strings.TrimSpace(...)` 한 번으로 정규화해 INSERT와 두 UPDATE(`:290` 다단계 중간, `:292` 최종)가 모두 `nullString(comment)`를 쓰고, rune 2000자 상한을 `tx.Begin` **앞**에서 400 `comment_too_long`으로 검사한다. 커밋 953c51d(3개 파일).
- 확신 없는 곳: ① `approval_levels=2`의 `:290` 경로는 **테스트로 돌리지 않았다** — 검토자 2명이 필요해 S 범위를 넘겼고, 과제서 지시대로 코드 동일성으로만 확보했다. 비평가가 볼 곳은 여기다. ② 상한 2000은 저장소에 선례가 없는 내 숫자다(기존: 200·160·300). `docs/api.md`·테스트·코드 세 곳에 일관되게 썼다. ③ 기존 행에 남은 `''` decision_comment를 `NULL`로 정리하는 데이터 마이그레이션은 하지 않았다 — 새 결정만 계약을 따른다.
- 일부러 하지 않은 것: 컬럼 타입(`text`)·마이그레이션 변경(요청 단계 검증으로 충분, 기존 긴 행이 CHECK를 깨뜨릴 위험), `executeApprovedOperation`·`errKeyStale` 분기(과제 무관), 웹(`web/src`에 `decisionComment` 참조 0건 — `npm test`·`internal/ui/dist` 재빌드 불필요), VERSION·CHANGELOG(릴리즈 단계 전용).
- 다음 역할이 조심할 것: `TestApprovalCommentContract`는 `TEST_POSTGRES_DSN` 없으면 **조용히 SKIP** 한다 — `-v`로 PASS를 눈으로 확인할 것. 테스트가 `DROP SCHEMA ai_admin/aiportal CASCADE`를 하므로 공유 DB 금지. 이번 검증은 전용 `postgres:16-alpine` 포트 55461을 썼다(컨테이너 `ai-admin-comment-pg`는 세션 종료 시 정리).
- 검증 실측: 수정 전 FAIL(3개 기준 모두 재현) → 수정 후 `go test -count=1 ./internal/server/ -run ApprovalComment -v` PASS; rune→byte 검사로 되돌리면 한글 2000자 사례가 다시 FAIL(인과 확인 후 복구); `go test -race -count=1 ./...` 전체 통과(internal/server 105.064s); `make lint`(1.2.27 일관)·`go build ./...` 통과.
- [러너 13:23] brief accepted — 채택 — 지정된 결함·코드 위치(`workflow.go:231`·`:282`·`:290`·`:292`)·재사용 셋업·헬퍼가 모두 현재 코드와 정확히 일치했�
- [러너 13:23] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 전용 postgres(포트 55473)로 `TestApprovalCommentContract` **PASS**(SKIP 아님), 전체 `go test -p 1 -count=1 ./...` 통과, `gofmt`/`go vet`/`go build`/`make lint` 통과. workflow.go만 main 판으로 되돌려 같은 테스트가 4개 단언에서 FAIL하는 것을 확인해 테스트가 변경을 실제로 고정함을 입증했다(이후 `git checkout HEAD --`로 복구, worktree clean).
- 못 본 것: `approval_levels=2` 중간 단계(workflow.go:299)를 실제로 돌리지 않았다 — 코드 독해로만 판정했고, 최종 단계와 정규화가 글자 그대로 같아 결함으로 보지 않았다. 웹 빌드·`npm test`는 `decisionComment`/comment 상한 참조가 없어 돌리지 않았다.
- 승인이어도 남는 우려 ①(릴리즈 노트용): 기존 행의 `''` decision_comment를 정리하지 않아 과거 결정은 `""`, 신규 결정은 `null`로 섞여 보인다 — "앞으로의 결정부터 적용"을 명시할 것.
- 우려 ②(다음 회차 후보): 다단계에서 2단계 검토자가 코멘트를 비우면 1단계 코멘트가 `decision_comment`에서 덮여 사라진다(기존 동작, 이번 회귀 아님·`approval_action`에 이력은 남음). 또 `web/src/pages/WorkflowPage.tsx:107` 텍스트에어리어에 `maxLength`가 없어 2000자 초과는 서버 400으로만 막힌다.
- 보안·법무 모두 차단 없음: 새 엔드포인트·인가 변경·비밀값 노출·새 의존성 없고, 개인정보를 새로 수집·전송하지 않으며 저장량을 늘리지 않는다. 데이터 마이그레이션이 없어 revert로 완전히 되돌아온다.
- [러너 13:26] review approved — 리뷰 승인 (risk=low)
- [러너 13:26] pr created — https://github.com/hkjang/ai-admin/pull/32
- [러너 13:34] ci passed — 검사 2개 모두 success
- [러너 13:35] merge done — 953c51d
