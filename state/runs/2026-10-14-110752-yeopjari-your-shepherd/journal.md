# PR 처리기 노트 2026-10-14-110752-yeopjari-your-shepherd — yeopjari-your PR #3
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-14-102826-yeopjari-your-improve)
# 회차 노트 2026-10-14-102826-yeopjari-your-improve — yeopjari-your
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:28] base pinned — main@735609c
- [러너 10:28] autonomy release — 

## 구현 노트
- quota.ts 의 `retryAfterSeconds` 만 고쳤다(프로덕션 파일 1개). 429 의 `retry-after` 가 항상 정확히 32400초(9시간) 과대였다 — 카운터 행의 PK 인 `windowStart`(한국 날짜를 UTC 자정에 박은 값)를 창의 경계로 썼기 때문. 키는 손대지 않았다: 옮기면 운영 DB 의 모든 quota_counters 행이 한 번에 끊긴다.
- 확신 없는 곳: `kstParts(at).m` 이 1-based 라는 전제로 `Date.UTC(p.y, p.m, 1)` 가 '다음 달 1일'이 된다 — 12월 롤오버를 단위 검사로 못 박았지만 읽을 때 한 번 더 봐 줄 것. 그리고 `windowStart` 자체(키)는 일부러 한국 자정과 어긋난 채로 남겼다 — 카운팅은 kstDateKey 가 바뀌는 순간 초기화되므로 맞지만, 키 값만 보면 9시간 어긋나 보인다.
- 일부러 안 한 것: 월 한도의 e2e 검사(한 달 창을 e2e 에서 트립시키려면 profile.nickname 월 한도를 써야 해 부수효과가 크다 — 단위 검사 2개로만 덮었다). e2e 포트 18998 하드코딩도 그대로 뒀다(이번 범위 밖, ideas.json 에 올림).
- 다음 역할 주의: 새 e2e 검사(`하루 한도 429의 Retry-After …`)는 Docker PostgreSQL 16 + :8788 서버가 있어야 돈다. `your_priv.quota_counters` 에 행을 심지만 window_start 는 직전 신고가 남긴 행에서 읽어 오므로 키 포맷을 테스트가 다시 쓰지 않는다.
- 이 기계에서 e2e 를 돌리려면: 다른 프로젝트의 고아 프로세스가 18998 을 쥐고 있다. `sed -i 's/18998/18996/g; s/18997/18995/g' scripts/e2e.mjs` + `.env.dev` 의 YOUR_YEOPJARI_URL 을 18996 으로 바꾼 뒤 돌리고 되돌렸다(커밋에는 포함되지 않음 — 확인함).
- [러너 10:40] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 판정 approve (risk low, 차단 없음). 산술을 직접 따라갔다: `Date.UTC(p.y, p.m, 1)` 은 `kstParts().m` 이 1-based 이고 Date.UTC 의 month 가 0-based 이므로 '다음 한국 달 1일'이 맞고 12월도 JS 가 2027-01 로 넘긴다 — 구현자가 의심한 자리는 문제 없다. 새 단위 검사 4개는 수정 전 코드를 복제해 돌려 전부 정확히 +32400초로 실패함을 확인했다(원장의 실패 재현과 증상 일치).
- 못 본 것: e2e 를 돌리지 못했다(Docker PG16 + :8788 필요). 새 e2e 검사가 429 를 받는지는 원장의 164/0 기록과 코드 독해에만 의존한다 — 독해로는 minute(2/10)보다 day(41/40)가 먼저 걸려 거짓 통과는 없다.
- **남는 우려(다음 회차)**: `scripts/e2e.mjs:528-532` 는 플레이크다. 서버 값은 consume 시각, 검사 상한은 응답 수신 시각으로 계산해 항상 `retryAfter >= untilKstMidnight` 이고 사이에 초 경계가 끼면 1 초과로 거짓 실패한다(회당 대략 Δ/1000 ≈ 0.5~2%). 업로드 **전에** 기준 시각을 잡거나 `+1` 여유를 두면 검출력 손실 없이 사라진다.
- 범위 밖 선재 결함(차단 아님, ideas 행): `quota.ts:146` purgeOldCounters 의 `window_start < now() - interval '3 days'` 가 month 창 카운터를 매달 4일경 삭제해 `profile.nickname` 월 한도가 실질적으로 초기화된다 — 이번에 정직해진 월 Retry-After 와 실제 초기화 시각이 여전히 어긋난다. 그리고 e2e.mjs:535 의 resetQuotas 는 540 과 중복(무해).
- 릴리즈 노트용: 사용자에게 보이는 변화는 "한도 초과 시 안내되는 재시도 시각이 9시간 과대였던 것을 바로잡음" 하나. PK(`window_start`)·`error_alerts.signature`·마이그레이션 모두 무접촉이라 revert 로 완전히 되돌아온다.
- [러너 10:45] review approved — 리뷰 승인 (risk=low)
- [러너 10:45] pr created — https://github.com/hkjang/yeopjari-your/pull/3
- [러너 10:46] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 

## 수리 노트
- CI 지적은 **틀렸다**(코드 결함 아님): 잡이 깃허브 결제/지출 한도 때문에 시작조차 못 했다 — `gh run view 38497445507` 주석 "The job was not started because recent account payments have failed…", steps 빈 배열, 2초. 결제가 풀리기 전엔 어떤 커밋도 녹색이 안 된다.
- 대신 CI 가 돌렸어야 할 전부를 로컬에서 돌렸다: check(53) · build · **e2e 164 passed / 0 failed** (Docker PG16 + :8788). 앞 회차 구현자·비평가 둘 다 못 돌린 새 Retry-After e2e 검사가 실제 서버에서 통과한다.
- 비평가의 플레이크 지적은 **맞았고 더 심했다**: 상한을 응답 뒤 시각으로 잡아 왕복 50~450ms 가정 시 거짓 실패 24.94%(20만 회 시뮬). 기준 시각을 업로드 전으로 옮기고 다음 한국 자정을 서버와 같은 `floor+1` 방식으로 구해 0/200000 로 없앴고, 수정 전 9시간 과대 버그는 10000/10000 그대로 검출한다(단언 완화 아님). 커밋 9865101, `scripts/e2e.mjs` 한 군데.
- 확신 없는 곳: 기준 시각과 서버 소비 시각이 한국 자정을 **가로질러** 갈라지는 1e-8 급 경우는 여전히 거짓 실패한다 — 고치려면 자정 직전 회피 로직이 들어가서 일부러 두었다. 포트 충돌 회피(18998→18996)는 `.env.dev`(gitignore)와 임시 sed 로만 했고 커밋에는 없다 — 확인함.

## 심사 노트
- 확인: quota.ts 산술 직접 검증(day=kstDayStart+24h, month=Date.UTC(y,1-based m,1) → 12월 2027-01 롤오버 실측) — 구현자가 의심한 자리 문제 없음. origin/main 의 옛 함수를 격리 스크립트로 복제해 돌려 새 단언 4개가 전부 정확히 +32400초로 실패함을 확인(minute 1개는 동작 불변 특성화). lint+typecheck+vitest 53/53 통과.
- 확인: e2e 단언 20만 회 시뮬 — 9시간 버그 200000/200000 검출, 거짓 실패 24.86%(2bbf98c) → 1/200000(9865101). 비평가·수리자 수치 재현. 잔존 플레이크는 1e-8 이 아니라 ~2.9e-6 (무시 가능).
- 못 봄: e2e 전체 미실행 — 18998 을 타 프로젝트 고아 프로세스가 점유, :8788 서버는 삭제된 워크트리의 낡은 바이너리. 공용 기계 손대지 않기로 하고 배선 독해(consume → errors.ts:88 → respond.ts:69 retry-after)로 대체.
- 비차단 기록: purgeOldCounters(quota.ts:146)의 3일 삭제가 month Retry-After 를 여전히 최대 ~27일 과대하게 만듦(선재, 회귀 아님). e2e.mjs:521-525 의 `?? null` 은 신고 검사 실패 시 NOT NULL PK 위반으로 스크립트 전체를 중단시킴(테스트 견고성 흠).
- 권고 merge 근거: PK(windowStart)·마이그레이션·auth·RLS 무접촉, 프로덕션 변경은 함수 1개·소비자 1곳, revert 로 완전 복구. 보안·법무 차단 소견 없음. CI 빨간색은 깃허브 결제 실패로 잡 미시작 — 코드 결함 아니며 결제 전까지 어떤 커밋도 녹색 불가(운영자 확인 필요).
