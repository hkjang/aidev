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

## 구현 노트
- `cron.ts:retention()` 의 안내 조회에 하한 `gt(retentionUntil, now)` 만 더했다(프로덕션 1파일). 루프의 `p.retentionUntil < now` 가드는 그대로 남겼고 `admin.ts`·삭제 조회·마이그레이션은 손대지 않았다. 변경 파일 2개(프로덕션 1 + 시험 1).
- 확신 없는 곳: 신규 시험은 `soon` 조회에 `orderBy` 가 없어 SQLite 가 rowid 순으로 돌려준다는 가정에 기댄다 — 찌꺼기 21건을 '안내 받을 프로젝트' 보다 먼저 만들어 재현한다. D1 이 이 순서를 바꾸면 재현이 약해진다(결함 자체는 하한이 없으면 그대로 존재한다). 시험이 D1 상태를 공유하므로 이 spec 뒤에 도는 spec 은 찌꺼기 21건을 보게 된다 — `workers: 1` 이고 `cron()` 을 부르는 spec 은 api-jobs·api-payments(모두 이 파일보다 앞) 뿐이어서 실제 간섭은 없었다(77 통과로 확인).
- 일부러 하지 않은 것: 두 조회를 공통 함수로 묶지 않았다(운영 화면은 `status='delivered'` 만, Cron 은 `delivered|closed` + legalHold 제외로 계약이 다르다). `admin.ts:50` 의 `7 * DAY` vs `dayMs(env)` 단위 불일치는 운영에서 같으므로 별건으로 남겼다(ideas.json).
- 다음 역할이 조심할 것: ① 이 시험은 빌드된 `dist` + `wrangler dev :8848` + 가짜 Resend :8849 가 있어야 돈다. ② 이 환경에서는 `npx` 가 cwd 조상의 `/home/hkjang/node_modules/.bin/node`(v20)를 집어 `wrangler` 가 "Node.js v22 필요" 로 죽는다 — 그 심링크를 v22 로 임시 재지정해 돌리고 **원래 대상으로 복구해 두었다**. ③ Playwright 브라우저는 `$HOME`(이 세션에선 run 디렉터리) 쪽 설치가 깨져 있어 `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 로 돌렸다. ④ `pgrep/pkill -f` 로 `wrangler dev --port 8848` 을 찾으면 자기 셸을 죽인다(패턴 쪼개 쓸 것).
- 신규 시험은 21.7s 걸린다(찌꺼기 21건 + 6.6s·4.9s 대기). 전체는 1.2m.
- 로컬 시험 D1(`.wrangler/test-state`)은 전체 시험을 돌린 **뒤 상태**로 남아 있다(찌꺼기 21건 포함). 다음 역할은 `node scripts/db-local.mjs --persist-to .wrangler/test-state` 로 먼저 비우고 `wrangler dev` 를 **새로** 띄울 것 — 돌아가는 wrangler 밑에서 DB 를 비우면 모든 요청이 `internal` 로 죽는다(한 번 당했다).
- [러너 19:33] brief accepted — 채택 — 과제서의 근거(파일·행 번호·수용 기준)가 지금 코드와 정확히 맞았고, 지시한 변경(하한 추가, 가드 유지, `admin
- [러너 19:33] verify failed — 실패한 검증: npm test --silent (exit 1)
