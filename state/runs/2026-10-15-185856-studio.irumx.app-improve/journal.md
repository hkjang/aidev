# 회차 노트 2026-10-15-185856-studio.irumx.app-improve — studio.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:59] base pinned — main@b00d53c
- [러너 18:59] autonomy release — 

## 정찰 노트
- 골랐다: Cron 보관 만료 안내의 영구 기아(`cron.ts:107-113`) — 운영 화면의 같은 조회(`admin.ts:50`)가 `gt(retentionUntil, now)` 를 이미 쓰고 있어 어느 쪽이 정본인지 추측 없이 판정할 수 있었다. 차선(seo 짝 시험)은 시험 전용·가치 2, MOV inline·단위 불일치는 가치 1~2 여서 제쳤다.
- 추측으로 적은 것: ① 찌꺼기 21건을 `delivered()` 로 만들 때 `npm test` 가 얼마나 느려지는지(재 보지 않았다) ② 관리자 라우트로 프로젝트를 더 싸게 delivered/closed 로 만들 길이 있는지(미확인 — 과제서에 미확인이라 적었다) ③ `orderBy` 없는 조회가 rowid 순으로 온다는 가정(실무상 그렇지만 보장은 아니다).
- 이 회차에 테스트를 돌리지 않았다(예산). 종전 기준선은 base `b00d53c` 에서 76 passed 다(앞 노트의 78 은 다른 base).
- 구현자가 조심할 것: 삭제 시점·보관 정책을 바꾸지 말 것, `cron.ts:113` 가드를 지우지 말 것, `admin.ts:50` 과 공통 함수로 묶지 말 것(두 조회의 계약이 다르다 — 운영자의 "통합하지 말고 좁히는 방향으로만" 과 같은 자리).
- 프로필은 0일 전 것이라 새로 쓰지 않았다. 이번에 확인한 자리: `services/{seo,files,outbox,cron}.ts`, `routes/public.ts`, `lib/spec.ts`, `shell.html`, `shared/domain.ts:165-186`, `tests/api-retention.spec.ts`, `tests/worker-test.env`.
- [러너 19:04] scout done — Cron 보관 만료 안내가 이미 만료된 프로젝트에 영구히 막힌다 — 안내 조회에 하한(`retentionUntil > now`)을 넣
