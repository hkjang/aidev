# 회차 노트 2026-10-17-102858-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:29] base pinned — main@75bab43
- [러너 10:29] autonomy release — 

## 구현 노트
- 무엇/왜: `lib/treeUtils.ts` `buildTree` 가 부모 행을 못 찾은 행을 버려, 복구 절차(ADMIN_GUIDE 5.4 `docker cp`)로 직접 넣은 파일의 작업이 세 화면 어디에도 안 나왔다(콘솔에 없으니 지울 수도 없음). 그런 행과 순환으로 어느 최상위에서도 닿지 않는 행을 최상위로 올리고, `id` 없는 행만 건너뛴다(그 한 행이 `TypeError` 로 화면을 무너뜨렸다). 프로덕션 1파일, 커밋 1639123.
- 확신 없는 곳·검증 못 한 것: 순환 데이터를 브라우저에서 직접 열어 보지는 않았다(React 무한 재귀가 트리 단계에서 끊긴 것만 단위로 확인). 중복 id 는 두 번 그려지던 것이 한 번만 그려지는 쪽으로 바뀌었고 단위 1건으로만 고정했다 — 중복을 관리자에게 알리는 설계는 안 했다. 최상위로 올린 행의 `level` 은 일부러 손대지 않아 들여쓰기 없이 상위 자리에 그대로 나온다.
- 일부러 하지 않은 것: `readActivityData` 에 읽기 검증 추가(살아 있는 훈련을 전면 읽기 실패로 바꿀 위험). 상태 전파·통계 집계 변경(ADMIN_GUIDE 4.3·USER_GUIDE 3.1 의 문서화된 계약). API·저장·인증·메일·추적·배포·의존성은 무수정.
- 다음 역할이 조심할 것: 신규 `e2e/dashboard-unvalidated-file.spec.ts` 는 격리된 `test-results/e2e-data/activity.json` 을 **직접 덮어쓰고** `afterAll` 에서 되돌린다(`workers:1` 순차 전제). 단위·E2E 는 순차로 돌리고, lint 전에 `rm -rf test-results playwright-report` 가 필요하다. 인과는 수정만 되돌려 E2E 2건 red 재현으로 확인했다(e2e-revert-red.log).
- [러너 10:42] verify passed — 검증 1개 통과 (policy)
