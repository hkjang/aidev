# 과제서 — 2026-10-15-185856-studio.irumx.app-improve

- 과제: Cron 보관 만료 안내가 이미 만료된 프로젝트에 영구히 막힌다 — 안내 조회에 하한(`retentionUntil > now`)을 넣어 20건 창을 되찾기 (가치 4 / 위험 1 / 작업량 S)

- 왜: `src/worker/services/cron.ts` 의 `retention()` 은 "만료 7일 전 안내" 대상을 `retentionUntil < now + 7일` + `retentionNoticeAt IS NULL` + `legalHold=false` + `limit(20)` 으로만 고른 뒤, 루프 첫 줄(`cron.ts:113`)에서 `p.retentionUntil < now` 인 것을 `continue` 로 건너뛴다. 건너뛴 프로젝트는 `retentionNoticeAt` 이 끝까지 `NULL` 로 남고 상태·`retentionUntil` 도 바뀌지 않으므로 **다음 Cron 에서도 같은 조회에 계속 잡힌다**. 즉 납품 뒤 보관 기간이 지난 프로젝트가 20건 쌓이면(운영 1년이면 자연히 쌓인다) `limit(20)` 이 전부 그 찌꺼기로 채워져, 정작 7일 뒤에 만료될 프로젝트는 영원히 안내 메일을 못 받고 파일이 삭제된다. 같은 목록을 보여 주는 운영 화면 조회는 이미 상·하한을 모두 쓴다 — `src/worker/routes/admin.ts:50`: `and(eq(status,'delivered'), lt(retentionUntil, now + 7*DAY), gt(retentionUntil, now))`. 의도는 그쪽이 정본이고 Cron 쪽이 틀렸다.

- 수용 기준:
  1) 만료가 이미 지난(그리고 안내를 못 받은) delivered/closed 프로젝트가 21건 이상 있어도, 7일 안에 만료될 프로젝트의 소유자·검수자에게 `보관 만료 안내` 메일이 들어온다.
  2) 이미 만료된 프로젝트에는 안내 메일을 **보내지 않는다**(지금 동작 유지). 파일 삭제(`retention.purge` 감사 기록, `/download` 410)도 지금과 똑같이 된다 — 이 회차는 삭제 시점·정책을 바꾸지 않는다.
  3) 기존 `tests/api-retention.spec.ts` 의 AT-24 시험(안내 → 만료 → 삭제 → 법적 보존 제외)이 그대로 통과한다.
  4) 시험이 증명할 것: 고치기 **전**에는 찌꺼기 21건을 만든 뒤 새 프로젝트의 안내 메일이 오지 않아 실패하고, 고친 **뒤**에는 온다. (프로덕션 배선 통과 — 실제 `POST /__cron`(tests/api.ts 의 `cron()` 헬퍼) → `runCron` → outbox → 가짜 Resend)

- 건드릴 파일 (프로덕션 1개 + 시험 1개):
  - `src/worker/services/cron.ts:retention()` — `soon` 조회의 `where` 에 하한을 추가: `gt(schema.projects.retentionUntil, now)` (drizzle `gt` 를 import 에 추가). `cron.ts:113` 의 `if (!p.retentionUntil || p.retentionUntil < now) continue;` 가드는 **지우지 말고 그대로 둘 것**(이중 방어, 수용 기준 2 를 지킨다). 주석 한 줄로 "이미 만료된 것은 안내 대상이 아니다 — 창(limit 20)을 막지 않도록 SQL 에서 뺀다" 를 남기면 좋다.
  - `tests/api-retention.spec.ts` — 새 시험 1개 추가. 기존 `delivered(label)` 헬퍼를 재사용.
  - 다른 파일은 건드리지 말 것. 특히 `src/worker/routes/admin.ts:50` 은 이미 맞으므로 손대지 않는다(두 조회를 공통 함수로 묶으려 하지 말 것 — 운영 화면은 `status='delivered'` 만, Cron 은 `delivered|closed` + `legalHold` 제외로 계약이 다르다).

- 시험 작성 지침 (여기서 헤맬 수 있으니 자세히):
  - 시험 서버는 `TEST_DAY_MS=200`(`tests/worker-test.env`) — '하루' 가 200ms 다. `api-retention.spec.ts` 의 `beforeAll` 이 `retentionDays: 30` 으로 바꾸므로 보관 30일 = **6초**, 만료 7일 전 = **1.4초 전**. 기존 시험이 쓰는 타이밍(`approvedAt + 4_900` 에 안내, `+6_600` 에 삭제)을 그대로 따르면 안전하다.
  - 찌꺼기 만들기: `delivered('stale-N')` 을 21회 돌린 뒤 7초 이상 기다리면(또는 생성에 그만큼 걸리면) 전부 `retentionUntil < now` 인 delivered 상태가 된다. 이들은 `retentionNoticeAt IS NULL` 로 남아 안내 창을 채운다. **주의: 21건은 `limit(20)` 을 넘겨야 하므로 20건으로는 증명되지 않는다.**
  - `delivered()` 전체 흐름(견적→결제→스토리보드 승인→시안 공유→최종 승인)이 21번 돌면 `npm test` 가 눈에 띄게 느려질 수 있다. 먼저 한 번 재 보고, 전체 시험 시간이 두 배(약 2분) 이상으로 늘면 `tests/api.ts` 에 더 싼 경로가 있는지(관리자 라우트로 프로젝트 상태를 `closed` 로 바꾸고 `retentionUntil` 이 설정되는 길이 있는지 — **미확인**) 먼저 확인하고, 없으면 느린 채로 두되 시험 1개로 끝낼 것. 같은 파일에 `test.describe.configure({ mode: 'serial' })` 가 필요할 수 있다(찌꺼기가 같은 D1 을 공유한다).
  - 찌꺼기를 만든 **뒤에** 안내 받을 프로젝트(`delivered('starved')`)를 만들어야 한다 — `soon` 조회에 `orderBy` 가 없어 SQLite 가 대개 rowid 순(=오래된 것 먼저)으로 돌려주므로, 찌꺼기가 먼저 들어가 있어야 결함이 재현된다.
  - 메일 확인은 `waitMail(ce, /보관 만료 안내/)`(기존 시험과 동일). 고치기 전 실패를 확인할 때는 `waitMail` 이 타임아웃으로 실패하는 것이 재현 증거다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  - `npm run build` — 느리다(글꼴 서브셋 → tsc 3종 → vite → `scripts/verify-build.mjs`). 끝에 "빌드 점검 통과" 가 나와야 한다. **`npm test` 전에 반드시 필요**(webServer 가 `dist` 를 `wrangler dev :8848` 로 띄운다).
  - `npm test` — Playwright 76개 + 신규. 종전 기준선은 **76 passed** 다(base `b00d53c`).
  - 좁혀서 돌릴 때: `npx playwright test tests/api-retention.spec.ts --project=api`
  - 타입만: `npm run check`

- 위험과 피할 것:
  - **삭제 정책을 바꾸지 말 것.** "만료가 지나면 삭제" 는 그대로다. 안내를 못 받은 프로젝트의 삭제를 7일 미루는 식으로 `purged` 쪽 조회를 건드리면 보관 정책·개인정보 처리방침과 어긋나고 위험이 커진다 — 이번 회차 범위 밖이다.
  - `migrations/` 는 손대지 않는다(덧붙이기만 하는 저장소이고 이 과제는 스키마가 필요 없다). `.github/workflows/`, `src/worker/auth.ts`, `src/worker/access.ts`, `src/worker/services/payments.ts` 도 건드리지 말 것.
  - 감사 기록(`audit`)의 `detail` 에 메일 주소·제목 같은 원문을 새로 넣지 말 것 — 식별자만(기존 `retention.purge` 는 `{files, retentionUntil}` 만 넣는다).
  - `admin.ts:50` 이 `7 * DAY`(실제 하루)를 쓰고 Cron 은 `dayMs(env)` 를 쓰는 불일치가 있다. 운영에서는 둘이 같으므로 **이번에 고치지 말 것**(별건, ideas.json 에 남겨 둠).
  - 시험이 같은 D1 을 쓰는 다른 spec 과 병렬로 돌아 찌꺼기 수가 흔들릴 수 있다. 21건을 "이 시험이 직접 만든 것" 으로 충분히 넘기고, 다른 spec 의 상태에 의존하지 말 것.

- 차선 후보: `seo.ts`·`shell.html` 짝 고정 시험 (가치 2 / 위험 1 / 작업량 S, 시험 전용) — `/app`·`/admin`·`/login`·`/invite` 의 `robots=noindex, nofollow`, 색인 가능한 쪽만 자기 `canonical`(아닌 쪽은 `${origin}/`), `/work/없는-슬러그` 가 404 + noindex, `noscript#seo` 의 `esc()` 가 `<`·`&` 를 이스케이프하는 것을 HTTP 로 고정한다. 확인된 코드 자리: `src/worker/services/seo.ts:35 pageMeta()`(`PRIVATE` 정규식 `cron.ts:25`… 가 아니라 `seo.ts:25`), `src/worker/index.ts:153`. 프로덕션 코드는 건드리지 않는다.
