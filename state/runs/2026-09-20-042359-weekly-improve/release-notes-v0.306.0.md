# v0.306.0 — 관리자 메일 카드가 세 큐를 모두 세고 최근 발송 기록을 보여 줍니다

## 실패 0건이 거짓말이었습니다

- 관리자 화면의 `주간보고 메일 발송` 카드는 `report_mail_deliveries` **하나만** 세었습니다. 릴레이는 세 큐를 지납니다 — 제출 메일, 팀원 작성 권고(`team_reminder_deliveries`), 상황판 마감 알림(`schedule_reminder_deliveries`). 권고와 알림이 전부 거부돼도 카드는 "실패 0건" 이라고 답했고, 관리자 가이드 3.7 이 "권고·알림 장애는 서버 로그와 큐 테이블에서" 라고 자백하고 있었습니다.
- 이제 세 표를 하나의 `UNION ALL` 서브쿼리(`mailDeliveriesUnion`) 위에서 셉니다. 집계·마지막 실패 사유·목록이 **같은 14일 창의 같은 행**을 읽으므로 숫자와 표가 어긋나지 않습니다.

## 종류별 건수와 최근 발송 기록

- `GET /api/v1/admin/mail/health` 는 기존 필드(`sent`·`queued`·`failed`·`writers`·`lastError`·`days`)를 그대로 두고 **`byKind{report, teamReminder, scheduleReminder}`** 만 더했습니다. `writers` 는 세 큐의 수신자를 합친 사람 수, `lastError` 는 세 큐 가운데 가장 최근의 실패 사유입니다.
- 새 **`GET /api/v1/admin/mail/deliveries`**(ADMIN 전용) 가 같은 창의 기록을 `createdAt` 내림차순으로 **최근 50건**까지 `{days, items}` 로 돌려줍니다. 각 행은 종류·수신자 표시 이름·주소·제목·상태·시도 횟수·실패 사유·생성·발송 시각이 전부이며, **보고서 본문과 제목은 싣지 않습니다** — `subject` 는 `week_start`/`reminder_on` 으로 만든 날짜 한 줄(`2026-08-17 주 제출 메일`, `2026-08-17 주 작성 권고`, `2026-09-20 마감 알림`)입니다. `total` 은 없습니다.
- 관리자 화면의 메일 카드는 현황 문단 아래에 종류·수신자·제목·상태·시도·사유·시각 표를 보여 주고, 기록이 없으면 기존 문장을 그대로 둡니다. 관리자 가이드 3.7 의 마지막 문장과 5.1 표가 이 사실을 말합니다.

## 검증

새 시험 `TestTheOperatorSeesEveryQueueTheRelayTouches` 는 제출 SENT 1건에 `team_reminder_deliveries` FAILED 1건·`schedule_reminder_deliveries` QUEUED 1건을 직접 넣고, **고치기 전 코드에서 `failed=0`·`queued=0`·`writers=1`·빈 `lastError`·deliveries 404 로 실패**하는 것을 먼저 확인한 뒤 통과했습니다 — 세 종류가 모두 있고 내림차순이며, 제출한 보고서 제목이 응답 어디에도 없고, `total` 이 없고, `days` 가 health 와 같고, USER 는 두 경로 모두 403 입니다. 실제 PostgreSQL 위에서 전체 Go 테스트와 `go vet`·`gofmt`, 가드 검사(`adminMailHealth`·`adminMailDeliveries` 도달), OpenAPI·쪽넘김·모달·버전 검사, 프런트엔드 160개 시험과 `tsc -b`·프로덕션 빌드를 통과했고, `mutation-check --test` 가 적용한 변이 넷 가운데 살아남은 둘은 모두 손대지 않은 옛 줄(마지막 사유 질의의 로그 한 줄만 좌우)입니다.

## 업그레이드

마이그레이션도, 새 환경변수도, 설정 키의 변경도 없습니다. health 응답은 필드가 하나 늘 뿐 기존 필드의 뜻이 같아 옛 화면도 그대로 읽습니다. 기존과 같이 PostgreSQL 을 백업한 뒤 v0.306.0 이미지를 올리면 됩니다.
