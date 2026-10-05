- 과제: notify scan 의 클러스터 목록 조회 실패를 "등록된 클러스터 없음" 폴백과 구별해 보고하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `notifyScanTargets`(`internal/proxy/k8s_notify.go:188`)는 `s.db.ListK8sClusters` 가 오류를 내면 `[]string{""}` 을 돌려주는데, 이것은 레지스트리가 빈 설치에서 쓰는 정상 폴백과 **완전히 같은 값**이다. 그래서 `cluster_id` 없는 함대 전체 cron 스캔(ADMIN_GUIDE 가 권하는 사용법)이 v0.9.295 에서 고친 "모든 클러스터가 한 창을 공유해 조용한 클러스터가 평가되지 않는" 옛 동작으로 조용히 퇴행해도, 응답은 `clusters_scanned:1` · 오류 키 없음으로 건강한 단일 창 스캔과 한 글자도 다르지 않다. 이 저장소가 최근 네 회차 내내 고쳐 온 "degraded 를 clean 으로 보고" 결함의 마지막 남은 자리다.
- 수용 기준:
  1) `cluster_id` 없이 POST 했을 때 클러스터 목록 조회가 실패하면 응답의 `window` 블록에 `clusters_error`(오류 문구)가 실리고, 스캔은 지금처럼 단일 공유 창으로 **계속 돈다**(500 아님, `sent` 계산 그대로).
  2) 레지스트리가 비어 있는 기존 폴백은 응답에 새 키가 붙지 않아 `TestNotifyScanFallsBackToASingleWindowWithoutRegisteredClusters`(`internal/proxy/k8s_notify_fanout_test.go:96`)와 나머지 notify 테스트 6파일이 무수정 통과한다.
  3) 새 테스트가 두 경로(오류 / 빈 레지스트리)의 실제 HTTP 응답이 서로 다르다는 것을 증명한다 — 수정 전에는 두 응답이 같아서 빨강이어야 하고, 손으로 만든 대역·주입 객체 없이 실 SQLite + `Server.Routes()` 로 한다.
- 건드릴 파일 (프로덕션 1개 + 테스트 1개):
  - `internal/proxy/k8s_notify.go:188 notifyScanTargets` — 반환을 `([]string, error)` 로 바꾼다(또는 두 번째 반환값으로 폴백 사유 문자열). 폴백 동작·주석의 계약은 그대로 두고 **오류를 삼키지 않게만** 바꾼다. 호출자는 `handleK8sNotifyScan` 한 곳뿐이다(`grep -n notifyScanTargets internal/` → 정의 1 + 호출 1, `k8s_notify.go:239`).
  - `internal/proxy/k8s_notify.go:239~335 handleK8sNotifyScan` — `window` 맵(현재 `events`/`revisions`/`events_truncated`/`revisions_truncated`/`clusters_scanned`/`clusters_truncated`, 선택적 `events_error`/`revisions_error`)에 선택적 `clusters_error` 를 `events_error` 와 같은 결로 추가한다. **감사 로그는 추가 작업 없음**: 함수 끝의 `audit` 맵이 `for k, v := range window { audit[k] = v }` 로 window 를 그대로 병합한다(확인함).
  - 운영자 안내: `k8s_notify.go:333` 의 `window_notice` 조건문은 이벤트·리비전 창에 대한 문구다. 그 문장을 늘리지 말고 `clusters_notice` 같은 별도 키로 "클러스터 목록을 읽지 못해 전 클러스터 스캔이 단일 공유 창으로 퇴행했다. 클러스터별로 다시 실행하고 DB 상태를 확인하라"는 뜻을 한국어로 적는 쪽을 권한다(UI 문구 한국어 관례). `window_notice` 를 비우면 `k8s_notify_window_test.go:191` 이 깨진다.
  - 신규 테스트 1파일 `internal/proxy/k8s_notify_registry_test.go` — 배선은 `internal/proxy/k8s_notify_window_test.go:142-170` 을 그대로 복사한다(`t.TempDir()` 에 명시적 `dsn` → `store.Open`+`Migrate` → `store.NewAsyncLogger`+`Start` → `NewServer(testConfig("http://upstream.invalid","secret"), db, logger, nil)` → `httptest.NewServer(server.Routes())`, httptest Mattermost webhook + `SetFlag` + `server.invalidateMattermostCache()`). `openTestStore(t)` 는 DSN 을 노출하지 않아 이 테스트에 쓸 수 없다. 오류 재현은 같은 파일 :183-190 과 똑같이 두 번째 커넥션(`sql.Open("sqlite", dsn)`)에서 `DROP TABLE k8s_clusters`. 실제 워크로드는 기존 `privilegedPod(t, ...)` 헬퍼를 쓰고 `cluster_id` 없이 `notifyScan(t, proxy.URL, "")` 로 호출한다(헬퍼 시그니처는 window_test 에서 확인할 것).
  - 문서·릴리즈 파일은 건드리지 않는다. `docs_reference_test.go`/`release_gate_test.go` 가 실제로 걸릴 때만 최소 갱신.
- 검증 명령:
  - `go test ./internal/proxy -run 'NotifyScan|K8sNotify' -count=1` — 이번 정찰에서 수정 전 기준선 `ok clustara/internal/proxy 2.297s` 확인됨.
  - `gofmt -l internal/proxy/k8s_notify.go internal/proxy/k8s_notify_registry_test.go` (출력 없어야 함)
  - `go build ./...` → `go vet ./...` → `go test ./... -count=1` (proxy 60~100s, 20 패키지)
  - 인과 확인: `clusters_error` 를 window 에 넣는 한 줄만 되돌려 새 테스트가 다시 빨강이 되는 것을 보고 복원.
- 위험과 피할 것:
  - **오류를 500 으로 올리지 말 것.** 인벤토리 fetch 실패는 지금 500 으로 중단하지만, 클러스터 목록 실패를 500 으로 만들면 cron 스캔이 아예 안 돌아 보안 알림이 끊긴다. 보고만 추가한다.
  - 빈 ClusterID 폴백의 **의미를 바꾸지 말 것**: 이 스키마에서 빈 ClusterID 는 와일드카드가 아닌 독립 식별자인데 `ListK8sEvents` 의 빈 clusterID 필터는 전체 조회다. 인벤토리 행의 cluster_id 로 대상을 합성하는 식의 "개선" 은 조회 범위를 바꾸므로 금지(함수 주석이 이미 그 이유를 적어 뒀다).
  - `suppressed:"quiet_hours"` 조기 반환이 `window` 를 아예 내보내지 않는 것은 **이번에 고치지 말 것** — 별도 보류 과제이고 우선순위 변경이 섞이면 기존 테스트 계약이 흔들린다.
  - `truncated`·`events_truncated`·`revisions_truncated` 의 "하나라도 걸리면 true" 의미와 `resources`/`events`/`revisions` 합계 의미를 유지한다.
  - 보호 경로 회피: `authorizeAdmin`·`mcp_oauth.go`·`keycloak*.go`·`internal/store/sqlstore.go` DDL/마이그레이션은 손대지 않는다. 테이블 삭제는 테스트 전용 임시 SQLite 파일에서만.
  - grep 결과를 증거로 쓰지 말 것. 수정 전 실제 HTTP 응답 두 개가 동일하다는 테스트 실패 출력을 회차 기록에 남길 것.
- 차선 후보: `/admin/k8s/inventory` 의 실효 조회 상한·창 포화 미표시 (가치 2 / 위험 1 / 작업량 S) — `handleK8sInventory` 는 items 만 돌려주고 `ListK8sInventory` 는 기본 200/최대 10000 으로 조용히 깎는다. 직전 회차가 events GET 에 쓴 `K8sEventQueryLimit` 공유 상수 패턴을 inventory 한 경로에만 복제한다(다른 호출자·revisions 일괄 변경 금지).
