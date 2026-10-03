# 회차 노트 2026-10-04-043721-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:37] base pinned — main@af45b2c
- [러너 04:37] autonomy release — 

## 구현 노트
- `loginThrottleFor` 가 남은 차단 시간을 계정 기준 `min(created_at)` 하나로만 계산해, 주소 제한(`auth.max_login_attempts_per_ip`)으로 차단된 호출자에게 `Retry-After: 1` 과 "1분 후에" 를 돌려주고 있었습니다. 두 계수기의 `min` 을 각각 읽어 실제로 차단한 쪽(둘 다면 늦게 풀리는 쪽)을 쓰고, 감사 `detail` 에 `addressFailures` 를 함께 남깁니다.
- 확신 없는 곳: `windowRemaining` 은 PostgreSQL 의 `now()` 로 읽은 시각을 Go 의 `time.Until` 과 비교합니다 — 기존 코드와 같은 방식이지만 DB 와 앱의 시계가 어긋난 배포에서는 남은 시간이 그만큼 어긋납니다. 고치지 않았습니다(범위 밖이고 기존 동작입니다).
- 확신 없는 곳: 감사 `detail` 에 필드를 하나 더했습니다. 전체 시험·openapi-check 는 통과했지만 감사 `detail` 의 모양을 읽는 외부 소비자가 있다면 그쪽은 확인하지 못했습니다.
- 일부러 하지 않은 것: 계정·주소 중 어느 쪽이 차단했는지를 **사용자 문구** 로는 말하지 않았습니다(어느 계정이 몇 번 틀렸는지 알려 주는 쪽으로 새기 쉽습니다). 운영자용 감사 기록에만 담았습니다.
- 다음 역할이 조심할 것: 새 시험 2개는 실제 PostgreSQL 이 있어야 돕니다(`WEEKLY_TEST_POSTGRES_DSN`, 이번엔 `docker start weekly-test-pg` 로 15434 를 올렸습니다). 둘 다 `auth.max_login_attempts_per_ip` 를 켜고 실패를 쌓은 뒤 성공 비밀번호가 429 를 받는 경로라, 실패 누적의 `loginFailureDelay` 때문에 각 1.8~2.5초가 걸립니다.
- 프런트 파일은 한 줄도 바꾸지 않아 `npm` 검증은 돌리지 않았고, `docs/OPERATIONS.md` 는 렌더 HTML 이 없어 `render-docs.py` 도 돌리지 않았습니다.
- [러너 04:51] verify passed — 검증 7개 통과 (auto)
- [러너 05:05] review timeout — 단계 제한 시간 초과
- [러너 05:05] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 05:05] pr created — https://github.com/hkjang/weekly/pull/29
