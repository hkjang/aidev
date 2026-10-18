# 회차 노트 2026-10-18-122847-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:28] base pinned — main@848f6b9
- [러너 12:28] autonomy release — 

## 정찰 노트
- 같은 계열("질의 오류를 버린 Scan")을 여섯 번 연속 고쳤고 남은 후보(imports.go:691)도 같은 모양이라 일부러 비켜, 오류를 **버리지는 않지만 잘못 분류하는** 결함을 골랐습니다: mail.go:687 이 loadReport 의 errNotFound 와 일시적 DB 오류를 한 분기에 묶어, 재시도 예산(기본 5)을 건너뛰고 첫 시도에 영구 FAILED 를 박습니다(재큐잉 입구가 없어 그 주 메일이 영구 소실). 차선(adminUsers 역할 필터)은 가치 2 이고 프런트가 올바른 값만 보내므로 뒤로 뒀습니다.
- 직접 읽어 확인한 것: mail.go:647-712(claim·적재·발송·defer)·780-785(markMailFailed)·242-400(헤더 안전), reports.go:163-213(loadReport 의 errNotFound 와 전달 오류 두 갈래), app.go:522(errNotFound), mail_test.go 의 fakeRelay·configureRelay·awaitDelivery·751행 give-up 시험.
- 추측으로 적은 것(구현자가 먼저 실측할 것): ① `report_comments` rename 으로 적재만 실패시키는 재현 순서 — 제출을 rename 전에 끝내야 하고, 그 테이블이 /api/v1/me/mail 경로에 쓰이면 report_items 로 바꿔야 합니다. ② errNotFound 경로를 HTTP 로 재현할 수 있는지(weekly_reports 삭제가 delivery 행을 cascade 로 가져갈 가능성 — 미확인). 안 되면 수용 기준 4(max_attempts=1 에서 포기)로 대체하세요.
- 조심할 것: 백그라운드 워커가 같은 행을 다투므로 `awaitDelivery("QUEUED")` 는 초기 상태와 구분이 안 됩니다 — attempts>=1 폴링으로 단정하세요. rename 은 반드시 t.Cleanup 으로 복원(공유 DB), `docker start weekly-test-pg` 가 선행돼야 하며 DSN 없는 exit 0 을 통과로 읽지 마세요.
- 프로필은 2일 전(2026-10-16) 것이 지금 코드와 어긋나지 않아 새로 쓰지 않았습니다. 다만 두 줄을 더해 읽으세요: `_ = ....Scan(` 는 internal/app 비시험 코드에 10곳뿐(일괄 금지는 과함), `rows.Err()` 누락은 keys.go:31·handover.go:531·admin_analytics.go:245/401/460·mail.go:849 에 있습니다.
- [러너 12:35] scout done — 보고서를 **일시적으로** 읽지 못한 메일 발송을 영구 실패로 처리하지 않는다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `sendNextQueuedMail`(mail.go:686)의 `if err != nil || report == nil` 을 쪼개, `errNotFound` 만 즉시 `FAILED` 로 보내고 그 밖의 적재 오류(연결 끊김·타임아웃)는 발송 실패와 같은 defer 경로로 보냅니다. 697-712행의 defer 블록은 `deferMailRetry` 헬퍼로 뽑아 두 자리에서 같이 씁니다. 프로덕션 1파일 + 신규 시험 1파일. 커밋 cd468ed.
- **과제서의 조건식은 그대로 쓰면 수정이 되지 않습니다.** `errors.Is(err, errNotFound) || report == nil` 는 `loadReport` 가 **모든 오류 경로에서 `report` 를 nil 로** 돌려주기 때문에 전달 오류까지 전부 즉시 실패로 빠집니다. 처음 그대로 썼다가 시험이 고친 뒤에도 같은 실패를 내서 발견했고, `err != nil && !errors.Is(err, errNotFound)` 를 먼저 묻는 순서로 바꿨습니다. 비평가는 이 순서를 먼저 보세요.
- 확신 없는 곳: 발송 실패 분기의 반환값을 `spent := attempts >= settings.MaxAttempts` 한 변수로 모았습니다 — 동작은 이전과 같다고 보지만(예산 소진 `true`, 그 밖 `false`), 로그 수준이 `Error`/`Warn` 두 갈래로 갈리는 자리를 if/else 로 바꿨으니 함께 읽어 주세요. `deferMailRetry` 는 `markMailFailed` 와 defer 를 한 함수에 담고 있어 이름보다 넓게 행동합니다.
- 일부러 하지 않은 것: `loadReport` 자체는 손대지 않았습니다(읽는 쪽이 많음). 같은 파일의 `sendNextQueuedReminder`·`sendNextQueuedScheduleReminder` 두 큐도 읽지 않았고 같은 모양일 수 있습니다 — ideas.json 에 신규 후보로 적었습니다. `mailDeliveryView` 에 새 필드를 더하지 않았으므로 응답 스키마·프런트는 그대로이고 `npm` 검증은 돌리지 않았습니다.
- 다음 역할이 조심할 것: 새 시험 3개는 **실제 PostgreSQL 이 있어야 돕니다**(DSN 없으면 SKIP). `weekly-test-pg` 는 이번 회차에 이미 Up(37시간) 이었습니다. 적재 실패 재현은 `report_comments` rename 인데 `newTestServer` 가 **시험마다 scratch DB** 를 만들므로 공유 DB 오염 위험은 없고(과제서의 걱정은 불필요했습니다), 그래도 `t.Cleanup` 으로 되돌립니다. 기준 3 은 `report_mail_deliveries_report_id_fkey` 를 떼고 보고서를 지웁니다 — FK 이름이 바뀌면 그 시험이 `t.Fatalf` 로 바로 말합니다.
- 검증: gofmt(빈 출력)·`go build ./...`·`go vet ./...`, `-run 'Mail|Delivery|Relay|Retry'`(5.738s), 전체 `go test ./... -count=1`(internal/app 162.092s, EXIT=0), openapi-check 119경로·paging-check 10목록·guard-check --changed 848f6b9(18개 도달, 새 시험 `deferMailRetry` 40%/60%)·`git diff --check`. 고친 조건 한 줄을 `if false` 로 되돌려 같은 실패 재발까지 확인하고 원복했습니다.
- [러너 12:45] brief accepted — 채택 — 근거(686-690행의 묶인 분기, `errNotFound` 와 전달 오류의 구분, 재큐잉 수단 없음)가 현재 코드와 실제 동작에 그대�
- [러너 12:48] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(위험 low, 차단 없음). 실패 재현을 직접 되돌려 확인: mail.go:700 을 `if false` 로 바꾸면 원장과 같은 출력으로 실패하고 원복하면 통과 — 신규 3개 `-count=8`·Mail|Delivery|Relay|Retry|Reminder·gofmt/build/vet 통과, 트리 복원 확인. `spent` 통합도 줄 대조로 이전 동작과 동일(Warn 로그 위치만 이동).
- 남는 우려 ①: mailreloadretry_test.go:188 이 `configureRelay` 뒤에 max_attempts=1 을 넣어, 좁지만 배경 워커가 기본 예산 5로 claim 하면 QUEUED 로 남아 Fatalf 가 날 창이 있습니다. 다음에 이 파일을 열면 설정을 configureRelay 앞으로 옮길 것.
- 남는 우려 ②: 전달 오류 분기가 `return true` 라 defer UPDATE 까지 실패하는 부분 장애에서 워커 내부 루프가 조일 수 있으나 수정 전도 동일한 모양이므로 회귀 아님. 전달 오류로 예산을 소진해 포기할 때 'gave up' 로그가 없는 것은 관측 공백.
- 릴리즈 노트: 사용자 문구·응답 스키마 불변, 마이그레이션·프런트 무관, revert 안전. 같은 계열의 weekly_automation.go:820·schedulereminder.go:289 는 테이블이 달라 이번 헬퍼로 못 덮었고 읽히지도 않았습니다 — 다음 회차 후보로 유지.
- 못 본 것: `loadReport` 의 `includedMaterialsFor`(워커 컨텍스트에 principal 없음) 와 프런트 카드 렌더는 보지 않았습니다(이번 변경이 건드리지 않음).
- [러너 12:53] review approved — 리뷰 승인 (risk=low)
- [러너 12:53] pr created — https://github.com/hkjang/weekly/pull/35
- [러너 13:18] ci passed — 검사 1개 모두 success
- [러너 13:18] merge done — cd468ed
- [러너 13:47] release published — v0.321.0
- [러너 13:48] assets verified — v0.321.0 자산 1개 (이전 v0.320.0: 1)
