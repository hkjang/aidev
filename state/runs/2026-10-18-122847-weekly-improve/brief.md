# 과제서 (정찰, 2026-10-18)

- 과제: 보고서를 **일시적으로** 읽지 못한 메일 발송을 영구 실패로 처리하지 않는다 (가치 3 / 위험 1 / 작업량 S)

- 왜: `sendNextQueuedMail`(internal/app/mail.go:647)은 큐에서 행을 claim 한 뒤 `report, err := a.loadReport(ctx, reportID)`(686행) 한 번으로 보고서를 읽고, **687행의 `if err != nil || report == nil`** 에서 어떤 오류든 곧바로 `markMailFailed(ctx, deliveryID, "보고서를 읽을 수 없습니다.")`(688행) 로 보내 `status='FAILED'` 를 박습니다. 그런데 `loadReport`(reports.go:163)가 돌려주는 오류는 두 종류입니다 — 행이 없으면 `errNotFound`(app.go:522), 그 밖에는 **DB 연결 끊김·타임아웃·일시 장애 같은 전달 오류**(reports.go:175 `return nil, err`, 179·189·195·205·211행의 Query/Scan/rows.Err 전파)입니다. 후자는 재시도하면 성공하는 상태인데도 **첫 시도 한 번에 재시도 예산(`mail.max_attempts`, 기본 5)을 건너뛰고 영구 FAILED** 가 되어, 발송 경로의 일시 오류는 `retryDelay` 로 최대 3시간을 기다려 주는 것과 대비됩니다(698-710행). 피해는 조용합니다: 작성자의 개인 설정 카드(`/api/v1/me/mail`)에는 "보고서를 읽을 수 없습니다." 가 붙은 FAILED 한 줄만 남고, 재큐잉 수단이 없으므로 **그 주의 보고 메일은 영구히 가지 않습니다**. 고치면 일시 장애는 예산 안에서 재시도되고, 진짜로 사라진 보고서(`errNotFound`)만 즉시 포기합니다.

- 수용 기준:
  1) 보고서 적재가 `errNotFound` 가 **아닌** 오류로 실패하고 `attempts < mail.max_attempts` 이면 행은 `QUEUED` 로 남고, `attempts` 는 1 증가하며, `error_message` 에 사유가 남고 `next_attempt_at` 이 미래로 밀린다(= 발송 실패의 defer 경로와 같은 모양, 707-709행).
  2) 장애가 걷힌 뒤 같은 행이 다시 due 해지면 발송이 **성공** 한다(가짜 릴레이가 메시지 1통을 받고 행이 `SENT`).
  3) `errNotFound`(보고서가 실제로 없음)는 지금처럼 **즉시** `FAILED` 이고 재시도하지 않는다 — 회귀 시험으로 고정할 것.
  4) `attempts >= settings.MaxAttempts` 인 적재 오류는 포기하고 `FAILED` 로 끝난다(행이 영원히 재시도되지 않음 — 발송 경로 698행과 같은 판정).
  5) 시험은 손으로 주입한 대역·소스 문자열 검사 없이 `newTestServer`/프로덕션 배선(`server.submitted(...)` → 큐 → `sendNextQueuedMail`)과 실제 PostgreSQL 을 지나고, 결과를 `/api/v1/me/mail` 응답으로도 읽는다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/app/mail.go:686-690` — `sendNextQueuedMail` 의 적재 실패 분기. `errors.Is(err, errNotFound) || report == nil` 만 `markMailFailed` 로 보내고, 그 밖의 오류는 **발송 실패와 같은 defer 경로**(`attempts >= settings.MaxAttempts` 면 `markMailFailed`, 아니면 `UPDATE ... error_message=$2, next_attempt_at = now() + retryDelay(attempts)`)로 보낼 것. 697-712행의 블록을 작은 헬퍼(예: `func (a *App) deferMailRetry(ctx, deliveryID int64, attempts int, settings mailSettings, cause error) bool`)로 뽑아 두 자리에서 같이 쓰는 것이 가장 좁다. 반환값은 지금 계약을 유지할 것: 발송 실패는 `false`(스윕 중단), 적재 실패는 **`true` 를 유지**(릴레이 탓이 아니므로 다음 행으로 넘어가도 된다 — 지금도 688-689행이 `true` 다). 사용자에게 보일 사유는 relay 원문이 아니므로 기존 문구 "보고서를 읽을 수 없습니다." 를 그대로 쓰고, 상세 오류는 `a.logger.Error`/`Warn` 에만 남길 것(감사·사용자 문구에 DB 오류 원문·식별자 외 값을 넣지 말 것).
  - `internal/app/mailreloadretry_test.go` (신규) — 아래 재현법. 같은 패키지, `t.Parallel()` 금지.
  - 기존 시험은 바꾸지 말 것. 특히 `mail_test.go:751 TestADeliveryThatRunsOutOfTriesSaysSoAndKeepsTheLastReason` 는 수용 기준 4의 짝이니 그대로 통과해야 한다.

- 재현법 (설계. 실측하지 않았음 — 아래 순서를 먼저 확인하고, 어긋나면 대안을 쓸 것):
  1. `server := newTestServer(t)` → `relay := startFakeRelay(t)` 를 만들되 **릴레이 설정은 아직 하지 않는다**. `mail.enabled` 가 꺼져 있으면 `settings.unusable()`(mail.go:87) 때문에 워커가 행을 claim 하지 않으므로, 먼저 작성자(`server.createUser`) + `PUT /api/v1/me/mail {"address":…,"onSubmit":true}` + `server.submitted(writer, "2026-08-24", …)` 로 **QUEUED 행만 만들어 둔다**(`mail_test.go:546 TestSubmittingAReportSendsItToTheAddressTheWriterChose` 와 `:762` 의 셋업을 그대로 따를 것).
  2. 적재만 실패시킨다: `ALTER TABLE report_comments RENAME TO report_comments_blocked` (loadReport 의 두 번째 질의, reports.go:193). `t.Cleanup` 으로 반드시 되돌릴 것 — **공유 DB 에 남으면 다른 시험이 통째로 깨진다**(10-06·10-16 회차 교훈). 제출을 1단계에서 이미 끝냈으므로 rename 이 제출 경로를 건드리지 않는다. `report_comments` 가 `/api/v1/me/mail` 이나 워커의 다른 질의에 쓰이면 `report_items`(reports.go:179) 쪽으로 바꿔도 되지만, 그쪽은 제출 경로와 겹칠 수 있으니 rename 시점이 제출 **후** 임을 지킬 것.
  3. `server.configureRelay(relay)`(mail_test.go:460 부근) 로 릴레이를 켜고 `mail.max_attempts` 를 3 으로 둔다(기본 5 도 무관). 그 뒤 `server.app.sendNextQueuedMail(server.ctx())` 를 직접 부르거나 백그라운드 워커를 기다린다 — **둘이 같은 행을 다툰다**(mail_test.go:478 `awaitDelivery` 주석의 경고). `awaitDelivery("QUEUED")` 는 초기 상태와 구분이 안 되므로 쓰지 말고, `attempts >= 1` 이 될 때까지 폴링하는 작은 헬퍼를 이 시험 안에 두고 그때의 `status/error_message/next_attempt_at` 를 단정할 것.
  4. 고치기 전: `status=FAILED, attempts=1` (수용 기준 1 실패). 고친 뒤: `QUEUED, attempts=1, next_attempt_at > now()`.
  5. 복구: rename 을 되돌리고 `UPDATE report_mail_deliveries SET next_attempt_at=now()` 로 다시 due 하게 만든 뒤 한 번 더 태워 `SENT` + `relay.received()` 1통을 확인한다(수용 기준 2).
  6. 수용 기준 3·4: `errNotFound` 쪽은 **행만 지우는 것으로 재현** 할 수 있는지 먼저 보라 — `weekly_reports` 행을 지우면 `report_mail_deliveries` 가 cascade 로 같이 지워질 수 있다(미확인). 그러면 `report_mail_deliveries` 에 존재하지 않는 `report_id` 를 직접 INSERT 하는 방법(FK 가 있으면 불가 — 먼저 확인)이나, 기준 4 처럼 `mail.max_attempts=1` + 적재 오류로 `FAILED` 를 확인하는 쪽으로 좁혀도 된다. **기준 4 는 반드시 넣을 것**(행이 영원히 QUEUED 로 남지 않음을 증명하는 자리).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `docker start weekly-test-pg` **먼저**. 이게 없으면 DB 시험이 전부 SKIP 되고 `exit 0` 이 통과로 오독된다(2026-10-13 경부터 Exited(255), 10-16 회차에 `docker start` 로 복구됨). DSN 자격은 `docker inspect` 로 자식 환경에만 넘길 것.
  - `gofmt -l internal/app` (빈 출력) · `go build ./...` · `go vet ./...`
  - `go test ./internal/app -count=1 -run 'Mail|Delivery|Relay|Retry'` (지정 범위)
  - `go test ./... -count=1` (internal/app 160~215초)
  - `python3 scripts/openapi-check.py` (119경로) · `python3 scripts/paging-check.py` (10목록) · `python3 scripts/guard-check.py --changed 848f6b9`
  - `git diff --check`
  - 고친 뒤 **그 한 줄만 되돌려 같은 실패가 다시 나는 것**까지 확인하고 원복할 것.
  - TMPDIR 을 `/mnt/c` 아래로 두지 말 것(`attachmentwrite_test.go` 가 파일 모드로 실패). `scripts/mutation-check.py`·`authz-check.py` 는 소스를 제자리 변경하므로 이번 회차에 돌리지 말 것.

- 위험과 피할 것:
  - 프런트·`internal/app/migrations/`·auth/session/OIDC/MCP OAuth 는 한 줄도 건드리지 말 것. 이 과제는 프런트 변경이 없으므로 `npm` 검증도 불필요하다(응답 스키마가 바뀌지 않는다 — `mailDeliveryView` 에 새 필드를 **더하지 말 것**).
  - `mail_test.go:1030 TestTheOperatorSeesEveryQueueTheRelayTouches` 는 공유 DB 잔여 상태로 환경적으로 깨진다. 새 시험이 남긴 rename·delivery 행이 그 시험을 깨뜨리지 않는지 확인하고, 의심되면 일회용 DB(`docker exec` 로 새 데이터베이스 생성)에서 전체를 한 번 더 돌릴 것.
  - 재시도 간격을 바꾸지 말 것(`retryDelay` 는 `TestTheRetryBudgetOutlivesAnOrdinaryOutage`(mail_test.go:662)가 고정하고 있다).
  - 범위를 넓히지 말 것: 실패한 발송의 **재큐잉 기능**(관리자 카드)은 별개 과제이고 중복 발송 방지 결정이 선행돼야 한다. `loadReport` 자체를 고치는 쪽으로 가지 말 것(읽는 쪽이 많다 — 좁히는 방향으로만).

- 차선 후보: **`adminUsers` 가 알 수 없는 역할 필터를 조용히 버리지 않게 한다 (가치 2 / 위험 1 / 작업량 S)** — `admin.go:481-486` 에서 `validRole(role)` 이 실패한 값을 그냥 건너뛰므로, `role=` 에 적힌 이름이 하나라도 어긋나면 그 역할의 후보가 **말없이 빠지고**, 전부 어긋나면 `roles` 가 비어 `role = ANY(...)` 조건 자체가 사라져(496-499행) 검토 책임자 picker(`frontend/src/pages/AdminPage.tsx:207` 이 `role=TEAM_LEADER,ORG_MANAGER,ADMIN&limit=500` 로 호출)에 **전체 디렉터리가 올라온다**. 이번 회차에 481-486·496-499행을 직접 읽어 확인했고(HTTP 재현은 미실행), 좁은 수정은 "알 수 없는 역할이 하나라도 있으면 400 INVALID_ROLE" 하나뿐이다. 빈 `role=` 의 현재 동작(필터 없음)은 **바꾸지 말 것** — 일반 사용자 표 조회가 그 경로를 쓴다.
