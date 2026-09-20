# 회차 노트 2026-09-20-085355-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:54] base pinned — main@f44496f
- [러너 08:54] autonomy release — 

## 정찰 노트
- [정찰 09:00] 과제: `rotateKey` 소유자 비활성 검사(발급 경로 v1.2.21 계약의 회전 확장). 고른 이유: 코드로 결함을 직접 확인(keys.go:181·workflow.go:419 모두 app_user.status 미조회)했고, 기존 `key_target_integration_test.go` 셋업·`errKeyTargetUnavailable`·409 매핑을 그대로 재사용해 S 크기로 끝난다. Keycloak e2e(CI에서만 검증)·합성 시드(M)는 예산·검증 가시성 때문에 제침.
- 확신 없는 곳: workflow.go 회전 분기(419~)는 grep로 `app_user` 조회가 없음을 확인했을 뿐 전체를 읽지 않음 — 구현자가 잠금 순서·롤백 위치를 직접 확인할 것. `locked` 상태 계정도 `status != 'active'`로 같이 거부되는지는 발급 경로 테스트가 `disabled`만 덮고 있어 미확인.
- 조심할 것: 새 검사는 승인 요청 생성 전의 명시적 `tx.Rollback`(keys.go:249) 앞에 넣어야 승인 요청이 남지 않는다. 프로필은 하루 전 것이고 v1.2.22로 릴리즈된 것 외에 어긋남이 없어 다시 쓰지 않음.
- [러너 08:56] scout done — `rotateKey`가 소유자 계정이 비활성인 키를 그대로 회전하던 문제 수정 — v1.2.21의 `user_not_active`·`key_target_un

## 구현 노트
- [구현 09:10] `rotateKey`(keys.go)에 소유자 `app_user.status` 검사를 FOR UPDATE 잠금 안·승인 요청용 Rollback 앞에 추가(409 `user_not_active`, 조회 실패 500 `key_rotate_failed`). workflow.go 회전 승인 분기는 과제서와 달리 소유자 검사가 **이미 있었고** `errKeyStale`(409 `key_conflict`)로 오분류하던 것을 `errKeyTargetUnavailable`로 바꿨고 그 메시지를 발급·회전 공용 표현으로 다듬었다. 커밋 d8a9ad0, 변경 4개.
- 확신 없는 곳: `key_target_unavailable` 메시지 문구 변경("받을(소유할) 계정 … 반려하거나 계정 상태를 복구한 뒤 다시 승인") — 문자열을 assert하는 테스트·웹 코드는 grep로 없음을 확인했으나 웹 화면이 코드만 매핑하는지 끝까지 보지는 않았다. 소유자가 호출자 본인인 경우도 항상 검사하는데(인증된 세션이라 항상 active) 조회 1회가 늘어난다 — 단순함을 택했다.
- 일부러 하지 않은 것: `npm test`(웹 변경 없음), 단위 분류 표 테스트(통합 테스트가 disabled·locked 둘을 직접 덮음), 회전 분기의 나머지 `errKeyStale` 세분화(ideas.json에 보류).
- 조심할 것: `TestKeyRotationChecksTheOwnerAccount`는 `TEST_POSTGRES_DSN` 없으면 skip — 이번엔 55432를 다른 세션 컨테이너(hunter-ci-pg)가 점유해 55433으로 띄웠다. 되돌림 검증 중 `git checkout -- workflow.go`로 제 변경까지 지워져 재적용했으므로 diff에 두 변경(sentinel·메시지)이 모두 있는지 비평가가 한 번 확인하면 좋다(`git show d8a9ad0 -- internal/server/workflow.go`).
- [러너 09:03] brief accepted — 채택 — 결함·파일·재사용 셋업 모두 코드와 일치했고, 유일한 어긋남(workflow 회전 분기가 "검사 없음"이 아니라 "검사하
- [러너 09:03] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- [비평 09:08] approve, risk low, blocking 없음. 전용 Postgres(55461)로 새 통합 테스트를 HEAD에서 PASS시키고, keys.go·workflow.go를 main으로 되돌려 105행(201≠409)·workflow.go만 되돌려 184행(key_conflict≠key_target_unavailable)에서 FAIL하는 것을 직접 확인 — 구현자가 걱정한 "두 변경이 다 diff에 있나"는 sentinel·메시지 둘 다 있고 테스트가 둘 다 고정한다. gofmt·vet 깨끗, ui/dist 미변경, web/src에 영향 문구 없음.
- 못 본 것: Keycloak e2e(로컬 skip), API 키 자격으로 인증된 호출자가 비활성 소유자 키를 회전하는 경로(인증 단계에서 막힌다고 가정).
- 남는 우려(기존과 동일): app_user 행은 잠기지 않아 상태 조회~commit 사이의 비활성화 경쟁 창이 발급·회전 즉시 경로에 남음 — 다음 회차 후보. 릴리즈 노트에는 회전 소유자 검사와 회전 승인 실행의 key_conflict→key_target_unavailable 재분류를 모두 적을 것.
- [러너 09:06] review approved — 리뷰 승인 (risk=low)
- [러너 09:06] pr created — https://github.com/hkjang/ai-admin/pull/27
