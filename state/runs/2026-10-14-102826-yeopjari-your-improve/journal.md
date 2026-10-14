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
