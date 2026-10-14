# PR 처리기 노트 2026-10-14-224159-yeopjari-your-shepherd — yeopjari-your PR #5
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-212848-yeopjari-your-improve)
# 회차 노트 2026-10-14-212848-yeopjari-your-improve — yeopjari-your
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:28] base pinned — main@735609c
- [러너 21:28] autonomy release — 

## 정찰 노트
- 우선 과제가 "CI 실패를 고쳐라" 였지만 원인은 저장소 밖이다 — PR #4 의 `check` 는 2초(07:24:11Z→07:24:13Z), 러너의 `ci-failure-ea6beee87790.txt` 의 `실패 단계:` 가 빈 칸이고, 같은 계정 `biz.irumx.app` 이 같은 날 11:48Z 에 3초 failure 다. 그래서 과제를 "PR #4 네 커밋 재출하 + CI 가 못 하는 검증을 로컬에서 전부 재현 + 원장에 계정 장애 명기" 로 좁혔다.
- 다른 후보를 제친 이유: `check.yml` 을 조이는 개선(2/3/S)은 Actions 가 뜨지 않는 동안 검증 수단이 0 이라 미뤘고, `v.int` 결함은 실제로 확인했지만(validate.ts:100) 우선 과제가 배정된 회차라 차선으로 뒀다.
- 추측으로 적은 것: cherry-pick 이 충돌 없이 올라간다는 전제(베이스가 PR #4 와 같은 `735609c` 라는 사실에서 추론, 실제로 cherry-pick 을 실행해 보지는 않았다 — 코드 변경 금지). 충돌하면 전제가 틀린 것이니 멈추고 차선으로 갈 것.
- 구현자가 조심할 것: 워크플로는 한 줄도 건드리지 말 것(완화 금지이고 검증 불가). 포트 18998·18997 점유는 네 회차 연속 발생했으니 `ss -ltnp` 로 먼저 보고 환경변수로만 우회할 것. `quota.ts` 의 `windowStart` 키는 운영 DB PK 다.
- 프로필은 2026-10-06 판을 그대로 둔다 — 이번에 확인한 `check.yml`·구조·검증 명령이 기재와 일치했다.
- [러너 21:33] scout done — PR #4 의 CI 실패를 규명하고(원인은 저장소 밖) 네 커밋을 로컬 전체 게이트로 재검증해 다시 출하한다 (가�

## 구현 노트
- 과제서대로 PR #4 의 네 커밋을 cherry-pick(exit 0, 충돌 없음) 해 재출하했다. 새 코드는 한 줄도 쓰지 않았다 — 결과 트리가 PR #4 head `8538b01` 과 바이트 동일하다(`git diff --stat 8538b01 HEAD` 빈 출력). 프로덕션 파일은 `quota.ts` 하나, `check.yml` 무변경.
- 검증은 `check.yml` 과 같은 순서로 전부 로컬 재현: check(53 passed) → build → ci-env → 마이그레이션 10 → :8788 → **e2e 164 passed / 0 failed / exit 0**. 인과는 되돌려 넣어 확인했다(단위 36000 vs 3600, e2e `retry_after=40993` vs `8593` — 차이 정확히 32400초). 되돌린 코드는 커밋하지 않았다(`git status` 빈 출력).
- **확신 없는 곳**: (1) CI 가 실제로 초록이 되는지는 끝까지 검증할 수 없었다 — Actions 계정 장애가 원인이라는 진단은 러너가 남긴 타임스탬프(2초/3초, 빈 실패 단계, 두 저장소)에 의존하고, 나는 `gh` 가 막혀 있어 로그를 직접 읽지 못했다. 이 진단이 틀렸다면 로컬 164/0 은 CI 실패를 설명하지 못한다. (2) 로컬 상자가 KST 라서 서버를 `TZ=UTC` 로 띄워 UTC 러너를 흉내 냈다 — 충실하다고 보지만 CI 와 완전히 같은 환경은 아니다. (3) `8538b01` 의 루프백 가드는 거절 두 경우(`https://yeopjari.bid`, 포트 없는 `http://127.0.0.1`)만 실행으로 확인했고 IPv6 `[::1]` 경로는 돌려 보지 않았다.
- **일부러 하지 않은 것**: `check.yml` 을 조이는 개선과 `v.int`(차선 후보) 둘 다 미착수 — 전자는 Actions 가 뜨지 않아 검증 수단이 0 이고, 후자는 1순위가 성립했기 때문이다. 둘 다 ideas.json 에 남겼다. `quota.ts` 의 `windowStart` PK 도 과제서대로 손대지 않았다(옮기면 운영 DB 카운터가 한 번에 끊긴다).
- **다음 역할이 조심할 것**: e2e 는 Docker `postgres:16-alpine` @127.0.0.1:55440(컨테이너명 `yeopjari-your-dev`) + `node scripts/ci-env.mjs` 로 생성되는 `.env.dev` + `:8788` 서버가 모두 있어야 돈다. 이번엔 포트 18998·18997 이 비어 있어 기본값으로 통과했으니 환경변수 우회는 쓰지 않았다 — 점유 시 `YOUR_YEOPJARI_URL`/`E2E_PUSH_ORIGIN` 로 **환경변수만** 바꿀 것(코드 기본값은 옛 하드코딩과 바이트 동일해야 한다). `CI 초록` 을 완료 조건으로 삼지 말 것.
- [러너 21:40] brief accepted — 채택 — 수용 기준 네 개가 모두 지금 코드와 맞았다. 정찰이 "추측으로 적은 것"으로 남긴 전제(cherry-pick 이 충돌 없이 올
- [러너 21:40] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 승인. 독립 재현: check 53 passed → build → dev-reset(마이그레이션 10) → e2e **164 passed / 0 failed**, 새 검사가 84행에서 실제로 PASS 한다. 구 구현을 /tmp 에 재구성해 돌려 보니 새 단위 테스트 5건 중 4건이 구 코드에서 FAIL 이고 day·month 양쪽 구-신 차이가 전 구간 32400초 하나뿐(1년 9000표본, 불일치 0) — 테스트가 vacuous 하지 않고 e2e 상한도 구 코드를 반드시 잡는다. quota.ts 의 1-based/0-based 월 인자와 한국 자정 정각 경계를 직접 계산해 확인했고, windowStart PK·check.yml·error_alerts 서명은 무변경이라 revert 가 순수 계산 되돌리기로 끝난다. 보안·법무 모두 차단 없음(새 경로·권한·개인정보·라이선스 변화 없음, Retry-After 는 당사자 자신의 한도 상태).
- 남는 우려 1 (다음 회차가 반드시 볼 것, 구현 노트의 안내가 반쯤 틀렸다): e2e.mjs:46 주석은 가짜 주소를 "서버에 알려 준 값에서 읽는다"고 하지만 e2e 는 자기 env 를 읽고, 서버는 **ci-env.mjs:35 가 `.env.dev` 에 하드코딩한 18998** 을 읽는다(dev-reset.sh 의 `. ./.env.dev` 가 export 를 덮어쓴다). 그래서 `YOUR_YEOPJARI_URL` 만 export 하는 포트 우회는 동작하지 않고 가짜 서버만 옮겨가 SSO 가 안 붙는다(throw 로 시끄럽게 실패, 거짓 통과 아님). 두 곳을 같이 바꿔야 한다. `E2E_PUSH_ORIGIN` 은 routes/push.ts:28 이 임의 루프백 포트를 받아 env 만으로 정상.
- 남는 우려 2: fakeOrigin 이 `[::1]` 을 통과시키지만 listen 은 IPv4 고정이고 push 정규식도 IPv6 를 안 받는다 — 구현자가 못 돌려 봤다고 한 바로 그 경로. 시끄럽게 실패하므로 결함은 아니나 허용목록에서 빼는 편이 정직하다.
- 남는 우려 3: e2e 검사는 요청 길이짜리 창(신고~업로드 또는 base~거절 사이에 한국 자정이 끼는 경우) 에서 하루 한 번 flake 가능하고, 하한이 `retryAfter > 0` 뿐이라 서버가 1 을 답해도 통과한다 — 다음에 조일 자리. 릴리즈 노트에는 "하루/한 달 한도 429 의 Retry-After 가 UTC 자정이 아니라 다음 한국 자정을 알린다(최대 9시간 과대 고지 해소)" 로 적으면 된다.
- 못 본 것: CI 가 실제로 초록이 되는지(gh 권한 없음 — 다만 check.yml 무변경이라 머지가 CI 를 악화시킬 수는 없다), 그리고 screenshots.mjs 는 돌리지 않았다(이 diff 에 UI 변경이 없다).
- [러너 21:48] review approved — 리뷰 승인 (risk=low)
- [러너 21:48] pr created — https://github.com/hkjang/yeopjari-your/pull/5
- [러너 21:48] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- CI 지적은 **틀렸다(코드 결함 아님)**: 체크런 주석이 "The job was not started because recent account payments have failed or your spending limit needs to be increased" 라고 적혀 있다(`gh api .../check-runs/115860836446/annotations`, `HOME=/home/hkjang` 로 기존 gh 인증 사용 — 앞선 두 역할이 "gh 권한 없음"으로 포기한 곳이다). 손대지 않은 main `735609c` 푸시 런에도 같은 주석이 붙고 00:29Z 이후 전 런이 3~5초 실패 — 저장소 밖 결제 문제다. 그래서 CI 용 코드 변경 없음(`check.yml` 무변경).
- 맞았던 지적은 비평 "남는 우려 1" 하나다: `scripts/e2e.mjs` 머리 주석의 "둘 다 환경변수로 우회 가능"이 거짓. `set -a; . ./.env.dev` 가 export 를 덮어쓰는 것을 실측(서버 18998 / e2e 19998)하고 주석을 "ci-env.mjs 도 같이 바꿔라, push 는 불필요"로 바로잡았다(`5547bbe`, 주석만).
- 검증: 고친 뒤 CI 와 같은 순서 전부 — check 53 passed → build → ci-env → 마이그레이션 10 → :8788 → e2e **164/0 exit 0**.
- 여전히 확신 없는 곳: (1) 결제가 풀리기 전까지 CI 초록은 누구도 확인할 수 없다 — 머지 판단을 CI 에 걸면 이 PR 은 영구 보류된다. (2) `fakeOrigin` 의 `[::1]` 과 자정 끼임 flake 는 그대로 남겼다(비평가가 결함 아님으로 분류). (3) 로컬은 KST 상자이고 서버만 TZ 를 흉내 내지 않은 기본값으로 돌렸다 — e2e 는 통과했지만 UTC 러너와 완전히 같은 환경은 아니다.

## 심사 노트
- 승인·머지 권고. CI 와 같은 순서를 전부 로컬 재현: npm ci → check 53 passed → build → ci-env → 마이그레이션 10 → :8788 → e2e **164 passed / 0 failed**.
- 테스트가 vacuous 하지 않음을 **양쪽에서** 증명했다 — `quota.ts:46` 을 옛 본문으로 되돌리면 단위 `rules.test.ts:130·137` 이 36000 vs 3600 으로 FAIL 하고, 서버를 옛 코드로 다시 빌드해 프로덕션 배선으로 돌리니 e2e `하루 한도 429의 Retry-After` 가 `retry_after=36412 한국자정까지=4012` 로 FAIL 한다(차이 정확히 32400초). 확인 후 `git checkout` 으로 되돌려 트리가 깨끗하고 `quota.ts` 가 바이트 동일함을 확인했다.
- CI 적색은 코드 결함이 아님을 **직접** 재확인했다(수리 노트가 맞다): check-run 115860836446 주석이 "recent account payments have failed…". `check.yml` 무변경이라 머지가 CI 를 악화시킬 수 없고, CI 를 완료 조건으로 삼으면 영구 보류되므로 로컬 전체 게이트를 근거로 삼았다. 강제 경로(`windowStart`·quota_counters PK)와 보호 파일(db/·auth/)은 diff 가 빈 출력 — 바뀐 것은 429 의 retry-after 헤더 값 하나뿐이라 revert 가 순수 계산 되돌리기로 끝난다. `fakeOrigin` 가드는 운영 주소·포트 없는 주소·유사 호스트를 실제로 돌려 거절을 확인했다.
- 못 본 것: 러너에서의 실제 초록(결제 전까지 불가), UTC 러너 환경(경계 테스트는 고정 Date 라 시스템 TZ 비의존), `screenshots.mjs`(UI 변경 없음).
- 다음 회차용 선재 결함(이 diff 밖, 반려 사유 아님): `quota.ts:146` purgeOldCounters 가 `window_start < now() - interval '3 days'` 로 지우는데 month 키는 그 달 1일 UTC 자정이라 유일한 월 한도(`routes/profile.ts:287` profile.nickname) 카운터가 매달 4일쯤 지워진다 — 정확해진 월 Retry-After 가 운영에서는 여전히 과대 고지다. 그 외 e2e 하한이 `retryAfter > 0` 뿐인 점, `fakeOrigin` 의 `[::1]` 허용.
