# 회차 노트 2026-09-20-121401-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:14] base pinned — main@7fc2212
- [러너 12:14] autonomy release — 

## 정찰 노트
- 고른 이유: 메일 PR(auto/2026-09-16-0332)과 출력 축소 PR(auto/2026-09-19-1527)이 아직 미머지라 settings·SettingsPage·scripts·web/e2e를 만지는 후보는 전부 충돌 위험이 있어 제쳤고, `login_rate_limit.go` 하나로 끝나는 잠금 밀어내기 문제를 골랐다(무작위 map 순회로 잠긴 항목이 지워지는 것을 코드에서 직접 확인).
- 추측한 곳: 상한 10,000·5회/5분이 문서 어디에도 없다는 것은 grep 한 번으로만 확인(미확인에 가까움). OpenBao의 `version=0` 처리(차선 후보)는 확인하지 않았다.
- 구현자가 조심할 것: 배선 테스트는 상한을 필드로 낮춰야 수정 전 실패가 결정적이다(상한 10,000 그대로면 e^-1 확률로 현재 코드도 통과). auth 핸들러의 failed/succeeded 위치는 건드리지 말 것.
- 프로필은 2026-09-19 것이 코드와 어긋나지 않아 다시 쓰지 않았다.
- [러너 12:17] scout done — loginRateLimiter가 10,000개 초과 시 잠긴 계정의 항목을 임의로 지워 잠금이 밀려나는 문제 정리 (가치 3 / 위험

## 구현 노트
- 바꾼 것: `login_rate_limit.go:failed`의 초과 축출을 `evict()`로 빼서 만료 → 미잠금(방금 key 제외) → 잠긴 항목만 남으면 `expiresAt` 오름차순으로 결정적으로 지우게 했고, 상한을 `loginFailureCapacity`/`capacity` 필드(기본 10,000)로 뺐다. auth 핸들러·`blocked`·`succeeded`·상한 값·5회/5분은 손대지 않았다. 커밋 `9a55a41`.
- 검증: 배선 테스트(`login_rate_limit_flood_test.go`, 실제 limiter+`server.login`, capacity 100·2,000회)와 단위 테스트(기본 상한 10,000개 잠긴 항목+`failed("new")`)를 먼저 써서 수정 전 실패(401 / 이른 항목 생존)를 보고 고쳤다. `-count=30` 반복, 기존 limiter 테스트 4개, gofmt·vet, `./scripts/verify.sh` 전체 exit 0.
- 확신 없는 곳: ② 단계(미잠금 항목)는 여전히 map 순서라 미잠금 항목 중 무엇이 지워질지는 무작위다 — 잠금 보존이 목표라 과제서대로 두었고 ideas.json에 별도 항목으로 남겼다. 초과 시 매 호출 O(n) 만료 순회는 기존과 같고 ③은 O(n log n)이지만 초과 상황(공격)에서만 발생한다 — 벤치마크는 하지 않았다.
- 일부러 하지 않은 것: CHANGELOG(릴리즈 커밋에서만 갱신되는 관례), 문서(상한 10,000이 어디에도 적혀 있지 않음), 미머지 PR이 만지는 settings·scripts·web/e2e 파일.
- 다음 역할이 조심할 것: 새 테스트는 DB 불필요(hook 기반). 배선 테스트는 `httptest`의 고정 RemoteAddr(192.0.2.1)에 의존하므로 `remoteIP`가 X-Forwarded-For를 보게 바뀌면 같이 봐야 한다. 단위 테스트는 기본 상한이 10,000이라는 것도 함께 고정한다(`capacity` 기본값을 바꾸면 그 assertion부터 깨진다).
- [러너 12:22] brief accepted — 채택 — 근거가 코드와 정확히 맞았고(map 순회 축출, 잠긴 항목 포함), 정찰이 계산한 대로 capacity 100·2,000회 흘림에서 수�
- [러너 12:22] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: `evict()` 3단계(만료→미잠금→잠금 expiresAt 오름차순) 논리와 `blocked()`의 `count >= limit` 판정 일치, range 중 delete 안전성, `keep` 키 보존, `auth_handlers.go:71`·`openbao.go:186` 배선 불변. 옛 축출 로직을 임시로 되돌려 새 테스트 2개를 돌리니 3/3 실패(401·locked-0 생존), 수정본에서는 통과 — 테스트가 변경을 실제로 고정한다. vet·gofmt 깨끗, 범위 이탈 없음, 마이그레이션·설정 변경 없음.
- 못 본 것: 상한 근처에서 `failed()`가 매 호출 O(n) 순회를 mutex 아래에서 하는 비용(기존과 동일)은 측정하지 않았다.
- 남는 우려(승인): 잠금 보존은 절대가 아니다 — 같은 5분 창 안에 10,000개 계정을 각 5회(≈50,000건) 흘리면 표가 잠금으로만 차고, 먼저 잠긴 대상이 expiresAt 최소라 여전히 밀려난다. 기존(≈17,000건에 50%)보다 약 3배 비싸졌을 뿐이니 릴리즈 노트는 "잠금이 밀려나지 않는다"가 아니라 "미잠금 항목보다 먼저 지워지지 않는다"로 써야 한다. 커밋 제목 "from the same address"는 표가 전역이라 어느 주소에서 흘려도 같다(수정은 둘 다 덮음).
- 다음 회차: 잠금만 남은 표에서의 정책(축출 대신 새 키 429, 또는 IP 단위 상한)은 ideas.json 후보로 적절하다.
- [러너 12:24] review approved — 리뷰 승인 (risk=low)
- [러너 12:24] pr created — https://github.com/hkjang/jikim/pull/37
- [러너 12:28] ci passed — 검사 2개 모두 success
- [러너 12:28] merge done — 9a55a41
- [러너 12:36] release published — v0.2.16
- [러너 12:40] assets verified — v0.2.16 자산 2개 (이전 v0.2.15: 2)
