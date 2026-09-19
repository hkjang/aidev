# 과제서 2026-09-20 — weekly

- 과제: 관리자 메일 발송 현황이 세 큐를 모두 세고, 최근 발송 기록 목록을 보여 준다 (가치 4 / 위험 1 / 작업량 M-)
- 왜: `adminMailHealth`(internal/app/mail.go:1107)는 `report_mail_deliveries` 만 집계해, 팀원 작성 권고(`team_reminder_deliveries`)와 상황판 마감 알림(`schedule_reminder_deliveries`)이 전부 거부돼도 관리자 카드는 "실패 0건" 이라고 말합니다 — ADMIN_GUIDE 3.7 마지막 줄이 이미 "권고·알림 장애는 서버 로그와 큐 테이블에서 봅니다" 라고 자백하고 있어 운영자가 psql 을 열어야 합니다. 세 큐를 한 번에 집계하고 "무엇이 누구에게 언제 나갔고 왜 실패했는지" 최근 n건 목록(본문 없음)을 같은 카드에 두면, 릴레이 장애를 로그 없이 화면에서 봅니다.

## 수용 기준
1) `GET /api/v1/admin/mail/health` 의 `sent·queued·failed·writers·lastError·lastFailedAt` 이 세 표(`report_mail_deliveries`·`team_reminder_deliveries`·`schedule_reminder_deliveries`) 를 합산한 값이다. `writers` 는 세 표의 사용자 열(`user_id`·`recipient_user_id`·`user_id`) 을 합친 DISTINCT. `lastError` 는 세 표 가운데 가장 최근의 비어 있지 않은 `error_message`. 기존 응답 필드·이름·`days` 는 그대로(프런트 `MailHealth` 타입 호환). 종류별 건수를 더 주고 싶으면 `byKind: {report, teamReminder, scheduleReminder}` 처럼 **추가**만 하고 기존 필드는 바꾸지 말 것.
2) 새 `GET /api/v1/admin/mail/deliveries` (ADMIN 전용, `app.go:273` 옆에 `requireRole("ADMIN")` 으로 등록) 가 최근 `mailHealthDays` 일 안의 발송 기록을 `created_at DESC` 로 최대 50건 돌려준다. 항목: `kind`(`REPORT`|`TEAM_REMINDER`|`SCHEDULE_REMINDER`), `address`, `status`, `attempts`, `errorMessage`, `createdAt`, `sentAt`, `subject`(사람이 읽을 한 줄 — 제출 메일은 `weekly_reports.week_start` 로 "2026-08-17 주 제출 메일", 권고는 `week_start` 로 "2026-08-17 주 작성 권고", 알림은 `reminder_on` 으로 "2026-09-20 마감 알림"), `userName`(수신자 `users.display_name`). **본문·첨부·비밀번호는 절대 싣지 않는다** (표에 본문 열이 없으니 자연히 없지만 JOIN 으로 끌어오지 말 것). 응답은 `{data: {days, items}}` 이며 `total` 을 넣지 않는다 — `total` 을 넣으면 `paging-check.py` 가 쪽 넘김을 요구한다. 50건 상한은 openapi 설명에 "최근 50건, 그 이상은 큐 테이블" 로 적는다.
3) 관리자 화면 `주간보고 메일 발송` 카드(frontend/src/pages/AdminPage.tsx:119 부근, 기존 `mailHealth` 문단 아래)에 최근 발송 표(종류·수신자·제목·상태·시도·사유·시각)를 둔다. 기록이 없으면 표 대신 기존 "아직 발송한 주간보고가 없습니다" 문장을 유지. `frontend/src/types.ts:381` 에 `MailDelivery`·`MailDeliveryList` 타입 추가. 카드 도움말의 "제출 메일만 집계" 취지의 문장은 없음(프런트에는 없음, 가이드에만 있음) — ADMIN_GUIDE 3.7 마지막 문장을 "관리자 카드의 발송 현황이 세 큐를 모두 집계하고 최근 50건을 보여 준다" 로 고칠 것.
4) 시험이 증명할 것(`internal/app/mail_test.go` 의 `TestTheOperatorCanSeeWhatHappenedToTheMail`(979줄) 확장 또는 이웃에 새 시험, `// guards: adminMailHealth, adminMailDeliveries` 표시): (a) 제출 메일 1건 SENT + 팀 권고 1건 FAILED(`team_reminder_deliveries` 에 직접 INSERT — `weekly_automation_test.go:756` 의 INSERT 모양을 그대로 빌릴 것, `error_message='relay refused reminder'`) + 마감 알림 1건 QUEUED(`schedule_reminder_deliveries` 직접 INSERT) 를 두고 health 가 `sent>=1, failed>=1, queued>=1`, `lastError` 에 `relay refused reminder` 가 담기는 것 — **고치기 전 코드에서는 failed=0 으로 실패해야 함** (그 실패를 먼저 보고 회차 노트에 적을 것); (b) deliveries 목록에 세 종류(`kind`)가 모두 있고 `created_at` 내림차순이며 항목 어디에도 본문 문자열(제출한 보고서 제목 등)이 없음; (c) USER 역할은 두 경로 모두 403.

## 건드릴 파일
- `internal/app/mail.go:1094-1131` — `mailHealthView`·`adminMailHealth`: 집계 질의를 세 표 `UNION ALL` 서브쿼리(`SELECT user_id AS who, status, error_message, created_at FROM report_mail_deliveries UNION ALL SELECT recipient_user_id, … FROM team_reminder_deliveries UNION ALL SELECT user_id, … FROM schedule_reminder_deliveries`) 위에서 돌리기. `lastError` 질의도 같은 서브쿼리. 새 `mailDeliveryView`·`adminMailDeliveries` 핸들러를 바로 아래에.
- `internal/app/app.go:273` — 새 경로 등록 한 줄.
- `internal/app/mail_test.go:979` — 시험 확장(위 4).
- `docs/openapi.yaml:1641` — `/admin/mail/health` 설명("세 큐 합산")과 새 `/admin/mail/deliveries` 경로. `python3 scripts/openapi-check.py` 가 경로 수를 맞춰 보므로 빠뜨리면 즉시 걸린다.
- `frontend/src/types.ts:381`, `frontend/src/pages/AdminPage.tsx:87-89, 119` — `useEffect` 하나 더(`api<MailDeliveryList>('/api/v1/admin/mail/deliveries')`), 표 렌더링. 기존 표 CSS 클래스(다른 카드의 `<table>` 을 따라 할 것; Confluence 매핑 표가 같은 파일에 있음).
- `docs/ADMIN_GUIDE.md:156-174` — 3.7 의 마지막 항목 문장 교체 + 표에 `GET /api/v1/admin/mail/deliveries` 한 줄, 270줄 "메일 발송 현황" 표 갱신. HTML 재생성: `python3 scripts/render-docs.py ADMIN_GUIDE` (PDF 잡음이 생기면 되돌릴 것 — 릴리즈 회차가 굽는다; 미확인: 이 스크립트가 HTML 만 따로 내는 옵션이 있는지).

## 검증 명령 (이 저장소에서 실제로 도는 것)
- `gofmt -l internal/ && go vet ./...`
- `WEEKLY_TEST_POSTGRES_DSN=<공유 DSN> go test ./internal/app -run 'TestTheOperatorCanSeeWhatHappenedToTheMail|MailDeliver' -count=1`
- 전체: `go build ./... && go test ./... -count=1` (실제 DB 로 약 145초)
- `python3 scripts/guard-check.py --changed main` · `python3 scripts/openapi-check.py` · `python3 scripts/paging-check.py` · `python3 scripts/modal-close-check.py` · `./scripts/version-check.sh`
- `cd frontend && npm run lint && npm run build && npm test`
- 커밋 뒤에만: `python3 scripts/mutation-check.py --test TestTheOperatorCanSeeWhatHappenedToTheMail --budget 480` (끝나면 `git status` 가 깨끗한지 확인)

## 순서와 시간 (45분 상한)
1. 시험을 먼저 넓혀 health 가 `failed=0` 으로 틀리는 것을 본다 (5분).
2. `adminMailHealth` UNION + `adminMailDeliveries` + 경로 등록 + openapi → 이름 붙은 시험 통과 → **첫 커밋 (착수 20분 안)**.
3. 프런트 타입·표 + 가이드 문장 + HTML 재생성 → lint/build/vitest → 둘째 커밋.
4. 전체 `go test ./...` 와 검사 스크립트 → 셋째 커밋(있으면). mutation-check 는 마지막, 커밋 뒤.
근거: 2026-09-17 설정 결함 회차(S)가 착수 12분에 커밋, M 캠페인 셋은 시간 초과. 이 과제의 백엔드는 질의 두 개와 핸들러 하나라 그 사이. 범위 35~55분, 8/10 확률로 45분 안.

## 위험과 피할 것
- 마이그레이션 없음. 세 표 모두 이미 있고(021·025·030) 열 이름이 다르니 UNION 에서 별칭을 맞출 것. `team_reminder_deliveries` 에는 `origin` 열이 있고 다른 둘엔 없다 — 공통 열만 뽑는다.
- `mail.go`·`AdminPage.tsx` 의 메일 카드는 아직 main 에 안 합쳐진 mail 표준 키 브랜치(auto/2026-09-16-1242 로 추정, 미확인)가 손댄 자리다. 설정 키·라벨·`mail.security` select 는 건드리지 말고 health 문단 아래에만 덧붙여 머지 충돌 면적을 줄일 것.
- auth·migrations·workflows 는 건드리지 않는다. `requireRole("ADMIN")` 만 쓴다.
- 운영자 규칙: 값을 읽는 경로가 둘(health 집계 / 목록)이면 같은 `mailHealthDays` 창을 같은 서브쿼리에서 읽게 해 두 숫자가 어긋나지 않게 할 것. 표시만 바뀌는 변경이 아니라 시험이 실제 행으로 집계 값을 증명해야 한다(가짜 대역 금지 — 이 시험은 fakeRelay + 실제 DB 를 이미 쓴다).
- 목록에 `total` 을 넣지 말 것(paging-check). 50건 상한을 넘는 것을 보고 싶다는 요청이 나오면 다음 회차의 offset 과제.
- 감사 로그는 남기지 않는다(읽기 전용 조회). 주소는 개인정보지만 이미 health 시험·개인 설정 화면에 노출되는 값이고 관리자 전용이므로 그대로 둔다.

## 차선 후보
- db_integration_test 의 나머지 7개 시험과 `authfailure_test.go:48`·`participationfairness_test.go:28,101`·`searchranking_test.go:22` 가 공유 DSN 을 직접 여는 것을 `openDatabase(ctx, createScratchDatabase(t, dsn))` 로 옮기고, 함께 `harnessNameStamp`(httpharness_test.go:410) 에 `weekly_u_` 접두사를 더해 `emptyScratchDatabase` 가 남긴 DB 도 `sweepAbandonedDatabases` 가 치우게 한다 (가치 3 / 위험 1 / S). 12곳 치환 + LIKE 한 줄, `harnesssweep_test.go` 에 `weekly_u_` 사례 하나. 시험 시간 약 5초 증가. 1순위가 성립하지 않을 때(예: 이 트리의 메일 카드가 예상과 다를 때) 고를 것.
