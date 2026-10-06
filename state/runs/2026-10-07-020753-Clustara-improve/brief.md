- 과제: quiet hours 조기 반환이 이미 계산한 조회 진단(clusters_error·events_error·revisions_error·truncated)을 전부 버리는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/proxy/k8s_notify.go:318-325` 의 quiet 분기는 `{"sent":0,"suppressed":"quiet_hours","quiet_hours":...}` + `clock` 만 돌려주고 즉시 `return` 한다. 바로 위 루프(`:259-304`)가 **이미 실행해 대가를 치른** 조회에서 얻은 `clustersError`/`eventsError`/`revisionsError`/`truncated`/`clustersTruncated`/`events`/`revisions` 는 한 글자도 나가지 않는다(`window` 맵 자체가 `:327` — quiet return 아래에서 만들어진다). 게다가 `s.auditAdmin(r, "k8s.notify.scan", ...)` 는 `:411` 에 있어 quiet 경로에서는 **감사 기록도 남지 않는다** — 조용한 시간의 스캔은 응답에도 감사에도 흔적이 없다. 그래서 ADMIN_GUIDE 가 cron 으로 걸라고 권하는 함대 스캔이 quiet 창(보통 야간, cron 이 가장 많이 도는 시간대) 안에서 DB 고장(`k8s_clusters` 조회 실패 → v0.9.295 가 없앤 단일 공유 창으로 퇴행, 이벤트/리비전 조회 실패)을 만나도 건강한 설치와 **완전히 같은 응답**을 낸다. v0.9.293(`events_error`/`revisions_error`)·v0.9.295(`clusters_truncated`)·v0.9.297(`clusters_error`)이 세 회차에 걸쳐 쌓은 진단이 하루 중 quiet 창 동안 통째로 사라진다. 고치면 조용한 시간에 도는 cron 도 "알림이 없는 것"과 "조회가 깨진 것"을 구별해 보고한다.

- 수용 기준:
  1) quiet hours 중 스캔이라도 `clustersError`/`eventsError`/`revisionsError` 중 하나라도 있으면 응답에 `clusters_error`/`events_error`/`revisions_error` 와 비-quiet 경로가 쓰는 **같은 문구의** `clusters_notice`/`window_notice` 가 들어간다. `truncated` 가 참이면 `truncation_notice` 도 같다.
  2) `sent:0` 과 `suppressed:"quiet_hours"` 와 `quiet_hours` 와 `clock`(`timezone`/`timezone_notice`)의 기존 의미·키 이름은 그대로다. 기존 quiet 테스트 3곳(`k8s_notify_config_test.go:128,135`, `k8s_notify_timezone_test.go:65,104`)은 `out["suppressed"]`/`out["timezone"]` 만 보므로 무수정 통과해야 한다.
  3) 조회가 전부 정상이면 quiet 응답에 `*_error`·`*_notice` 키가 없다(건강한 설치 응답을 소음으로 채우지 않는다).
  4) 테스트가 증명할 것: **수정 전에는** 레지스트리가 깨진 설치의 quiet 응답이 정상 설치의 quiet 응답과 `fmt.Sprint` 수준에서 구별되지 않았다는 것, **수정 후에는** 다르고 `clusters_error` 가 비어 있지 않다는 것.

- 건드릴 파일 (프로덕션 1개 + 신규 테스트 1개):
  - `internal/proxy/k8s_notify.go:handleK8sNotifyScan` — `window` 맵 생성 블록(`:327-350`, `events`/`revisions`/`events_truncated`/`revisions_truncated`/`clusters_scanned`/`clusters_truncated` + `events_error`/`revisions_error`/`clusters_error` + `clusters_notice`/`window_notice`)을 **quiet 판정(`:318`) 위로 그대로 옮기고**, quiet 분기의 `out` 에 `clock` 과 **같은 방식으로** `window` 를 병합한다. `truncated` 와 `truncation_notice`(`:429` 의 문구)도 함께 싣는다. `window` 생성은 순수 맵 조립이라 I/O 순서가 바뀌지 않는다 — **조회를 옮기지 말 것**. 아래 비-quiet 경로의 `for k, v := range window` 병합 두 곳(`:412` 감사, `:426` 응답)은 그대로 둔다.
  - (선택, 같은 함수) quiet 경로에도 `s.auditAdmin(r, "k8s.notify.scan", ...)` 를 남기는 것이 이치에 맞으나 **이번 범위 밖**으로 둔다 — 감사 엔트리 수가 늘면 다른 테스트의 전제를 건드릴 수 있다. 필요하면 다음 회차.
  - 신규 `internal/proxy/k8s_notify_quiet_window_test.go` — 아래 "테스트 레시피" 참조.

- 테스트 레시피 (전부 같은 패키지에 **이미 있는** 헬퍼, 손으로 만든 대역 금지):
  - `newNotifyScanServerAt(t)` (`k8s_notify_registry_test.go:24`) → `(*store.SQLStore, dsn string, *httptest.Server)`. 실 SQLite + `NewServer` + `httptest.NewServer(server.Routes())`.
  - `privilegedPod(t, "c1", "edge", "agent", "2026-09-01T00:00:00Z")` (`k8s_notify_scope_test.go:60`) → `db.UpsertK8sInventory` 로 넣는다.
  - quiet 창 강제: `quietWindowAround(time.Now().Hour())` (`k8s_notify_timezone_test.go` 가 쓰는 헬퍼) 를 `POST {proxy.URL}/admin/k8s/notify/config` 의 `quiet_hours` 로 보낸다. `newNotifyScanServerAt` 는 `notifyScanFixture` 가 아니므로 `postNotifyConfig` 대신 직접 POST 하거나, 플래그를 스토어에 직접 쓰는 기존 방식을 따른다 — **구현자가 실제 시그니처를 보고 고를 것**(미확인: `newNotifyScanServerAt` 에 플래그 설정 헬퍼가 붙어 있는지는 열어 보지 않았다).
  - 오류 주입: `k8s_notify_registry_test.go:82-95` 의 레시피를 그대로 쓴다 — 같은 dsn 에 `sql.Open("sqlite", dsn)` 로 두 번째 커넥션을 열고 `DROP TABLE k8s_clusters`. 인벤토리는 건드리지 않으므로 보안 분석은 정상인 **부분 실패**가 된다.
  - 비교: 정상 설치 quiet 응답 A 와 깨진 설치 quiet 응답 B 를 `notifyScan(t, proxy.URL, "")` (`k8s_notify_scope_test.go:75`) 로 받아 `fmt.Sprint(A) == fmt.Sprint(B)` 이면 실패시킨다. 이것이 v0.9.297 테스트와 같은 형태이며, 그 테스트가 비-quiet 경로에서 통과하는 동안 quiet 경로에서는 실패한다는 점이 이번 결함의 핵심이다.
  - **수정 전에 먼저 빨강을 확인**하고, 수정 후 `window` 병합 한 줄만 다시 빼서 같은 테스트가 다시 빨강이 되는지(인과 확인) 보고 복원할 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/proxy -run 'NotifyScan|K8sNotify' -count=1` — 직전 회차 기준 ok 약 2.6s
  - `gofmt -l internal/proxy/k8s_notify.go internal/proxy/k8s_notify_quiet_window_test.go` (출력 없어야 함)
  - `go build ./...` → `go vet ./...` → `go test ./... -count=1` (20 패키지, proxy 약 90s · store 약 27s — 느리니 마지막에 한 번)

- 위험과 피할 것:
  - quiet 판정 자체(`inQuietHours`, `notifyLocation`, `parseQuietHours`)와 **조회 순서**는 건드리지 말 것. 보류 아이디어 "notify scan 이 조용한 시간에도 팬아웃 조회를 먼저 돌고 버린다"(조회를 quiet 판정 **뒤로** 미루기)는 이 과제와 **정반대 방향**이다 — 동시 채택 금지. 이번 과제는 "이미 한 조회의 결과를 버리지 말자"이지 "조회를 하지 말자"가 아니다.
  - 인벤토리 조회 오류의 500 조기 반환(`:267-270`)은 이번에 손대지 말 것 — 부분 성공 응답 의미를 먼저 정의해야 하는 별도 M 과제다.
  - `truncated`/`events_truncated`/`revisions_truncated` 의 "하나라도 걸리면 true" 계약과 `resources`/`events`/`revisions` 의 합계 의미를 바꾸지 말 것.
  - 보호 경로(`server.go` 인증, `mcp_oauth.go`, `store/sqlstore.go` DDL, 마이그레이션, 워크플로)는 전혀 건드리지 않는다. `docs_reference_test.go`/`release_gate_test.go` 가 보는 버전·릴리즈 파일도 수정하지 않는다.
  - notice 문구는 **기존 상수 문자열을 재사용**하라(새로 쓰지 말 것). 비-quiet 경로와 문구가 갈리면 운영자가 두 경로를 다른 고장으로 읽는다.

- 차선 후보: `/admin/k8s/inventory` 의 실효 조회 상한·창 포화 미표시 (가치 2 / 위험 1 / 작업량 S) — `handleK8sInventory` 가 `items` 만 반환하는데 `ListK8sInventory` 는 기본200/max10000 으로 조용히 깎는다. v0.9.296 의 events GET 패턴(`K8sEventQueryLimit` 공유 상수 + 응답에 `limit`/`window_full`/포화 시 `window_notice`)을 **inventory 한 경로에만** 복제한다. 다른 호출자·revisions 일괄 변경 금지.
