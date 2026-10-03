- 과제: notify scan 의 전 클러스터 실행이 조회 예산을 모든 클러스터에 공유해 조용한 클러스터가 한 줄도 평가되지 않는 것 고치기 (가치 3 / 위험 2 / 작업량 M)

- 왜: `cluster_id` 를 비우고 돌리는 스캔(운영자가 cron 으로 하나만 걸어 함대 전체를 커버하는 바로 그 사용법 — `docs/ADMIN_GUIDE.md:369` 가 "cron/`/loop`으로 주기 호출하세요" 로 권하는 방식)은 인벤토리 2000행·이벤트 500행·리비전 1000행을 **전 클러스터가 나눠 쓰는 단일 창**으로 읽는다(`k8s_notify.go:195-237`, 각각 `updated_at`/`last_seen`/`observed_at` 역순). 자주 갱신되는 한 클러스터가 창을 채우면 다른 클러스터의 privileged 워크로드·과도한 Role 은 평가 자체가 되지 않고, 응답의 `truncated`/`events_truncated` 는 "잘렸다" 만 말하고 **어느 클러스터가 빠졌는지** 는 말하지 않는다. 클러스터별로 각자의 예산을 주면 한 클러스터의 churn 이 다른 클러스터의 알림을 삼키지 않고, 어느 클러스터의 창이 찼는지도 응답에 남는다.

- 수용 기준:
  1) 클러스터 2개(`UpsertK8sCluster` 로 등록)가 있고 A 의 인벤토리 행이 예산을 다 쓰는 상황에서, B 의 privileged 워크로드가 알림으로 나간다. 수정 전에는 같은 테스트가 A 의 것만 보내고(`sent:1`) B 를 놓친다.
  2) 응답·감사 로그가 `clusters_scanned`(이번에 각자 예산을 받은 클러스터 수)와 `clusters_truncated`(세 창 중 하나라도 찬 클러스터 ID 목록)를 보고한다. 기존 `truncated`·`events_truncated`·`revisions_truncated`·`resources`·`events`·`revisions`·`window_notice` 의 의미(하나라도 걸리면 true / 합계)는 그대로 둔다 — 기존 테스트는 키 단위로 읽으므로 필드를 더해도 깨지지 않는다.
  3) `k8s_clusters` 가 비어 있으면(= 기존 notify 테스트 6개 파일의 상태. `grep UpsertK8sCluster internal/proxy/k8s_notify*_test.go` → 0건으로 확인) 지금과 **완전히 같은** 단일 창 1회 조회로 폴백한다. 즉 `k8s_notify_test.go`·`k8s_notify_scan_test.go`·`k8s_notify_scope_test.go`·`k8s_notify_window_test.go`·`k8s_notify_timezone_test.go`·`k8s_notify_podsec_level_test.go` 를 **수정하지 않고** 통과해야 한다. 이 폴백은 `cluster_id` 를 명시한 요청에도 해당한다(목표 1개 = 지금과 같은 1회 조회).

- 건드릴 파일 (프로덕션 2개):
  - `internal/proxy/k8s_notify.go` — ① 신규 헬퍼 `(s *Server) notifyScanTargets(ctx, clusterID string) []string`: `clusterID != ""` 면 `[]string{clusterID}`; 비면 `s.db.ListK8sClusters(ctx)`(= `internal/store/k8s.go:276`, `K8sCluster.ID`) 로 등록된 클러스터 ID 를 모으고, 오류이거나 결과가 비면 `[]string{""}` 를 돌려 현재 동작을 보존한다. ② `handleK8sNotifyScan`(`k8s_notify.go:183`) 의 조회 블록 — 지금 한 번씩 부르는 `ListK8sInventory`(`notifyScanBudget+1`, `Kinds: notifyScanKinds()`)·`ListK8sEvents`(`notifyScanEventBudget`)·`ListK8sRevisions`(`notifyScanRevisionBudget`)를 목표 클러스터마다 돌려 결과를 이어 붙이고, 클러스터별 잘림/포화를 모아 `truncated`(any)·`clusters_scanned`·`clusters_truncated` 로 집계한다. `events_error`·`revisions_error` 는 여러 목표 중 처음 실패한 것을 `"<clusterID>: <err>"` 형태로 적는다(인벤토리 오류는 지금처럼 500 으로 끝내되, 전 클러스터 팬아웃에서는 어느 클러스터에서 깨졌는지 메시지에 넣는다). 분석 호출(`AnalyzeRCA`→`EnrichWithConfigChanges`→`AnalyzeSecurity`)과 `notify` 클로저·dedup 키는 **그대로** — 분석기와 dedup 은 이미 finding 자신의 `ClusterID` 를 쓰므로(v0.9.285~0.9.287) 이어 붙인 입력에 그대로 돈다.
  - `docs/K8S_OPERATIONS_HUB.md` — `### notify scan · ...` 형식으로 섹션 하나 추가(기존 선례: 63행 "이벤트·리비전 조회 실패와 창 포화", 146행 kind 범위, 199행 전 클러스터 cluster_id). 무엇이 왜 틀렸고 응답에 어떤 필드가 생겼는지, DB 스키마·설정 변경 없음을 적는다. **다른 문서·버전 마커는 건드리지 말 것**(`internal/proxy/docs_reference_test.go`·`release_gate_test.go` 가 검사한다).
  - 신규 테스트 `internal/proxy/k8s_notify_fanout_test.go` — 표준 배선(`openTestStore(t)` + `store.NewAsyncLogger` + `NewServer(testConfig(...))` + `httptest.NewServer(server.Routes())` + `postJSON`)에 실제 `kube.InventoryFromObject` 로 만든 privileged 워크로드 2개(클러스터 A/B)와 `UpsertK8sCluster` 등록 2건. `notifyScanBudget` 을 1 로 낮춰(`defer` 로 복원) A 의 행 하나가 공유 창을 다 쓰게 만든 뒤 B 의 알림이 나가는지 httptest webhook 으로 확인한다. 손으로 만든 대역 금지.

- 검증 명령 (이 저장소에서 실제로 도는 것, 이 정찰에서 마지막 것 실행 확인 — `ok clustara/internal/proxy 2.469s`):
  1) `go test ./internal/proxy -run 'NotifyScan|K8sNotify' -count=1`
  2) `gofmt -l internal/proxy/k8s_notify.go internal/proxy/k8s_notify_fanout_test.go`(출력 없어야 함)
  3) `go build ./...` → `go vet ./...`
  4) `go test ./... -count=1`(전체 약 80s, proxy 단독 60~75s)
  5) 인과 확인: 팬아웃 루프만 단일 조회로 되돌려 신규 테스트가 다시 빨강이 되는 것을 보고 복원한다.

- 위험과 피할 것:
  - **스토어 상한(`boundedLimit`)·`notifyScanEventBudget`(500)·`notifyScanRevisionBudget`(1000) 값을 올리지 말 것.** 이벤트 max 500(`store/k8s.go:685`)·리비전 max 1000(`:1214`)은 `ListK8sEvents` 호출자 36곳이 공유하는 하드 클램프이고, 그 계약 불일치는 별개의 L 과제다(보류 유지). 이번 과제는 "클러스터마다 그 예산을 한 번씩 쓴다" 일 뿐이다.
  - **클러스터 수에 상한을 두어 조용히 버리지 말 것** — 지금 고치는 결함과 같은 모양이 된다. 팬아웃은 `3 × 등록 클러스터 수` 개의 인덱스 LIMIT 쿼리가 된다(현 코드는 3개). 이것이 이 과제가 기대는 가장 큰 가정이다: 폐쇄망 운영 허브의 등록 클러스터는 수십 대 규모라는 것. 상한이 꼭 필요하다고 판단되면 패키지 `var` 로 빼고 **넘친 클러스터를 응답에 명시**해야 한다.
  - `internal/analyzer/*` 와 `classifyPodSecurity`·`restrictedProfileViolations` 는 건드리지 말 것 — 2026-09-28·10-03 이 연속으로 포스처 등급과 Deny 대상을 움직였고, 실환경 영향 평가 전에 또 넓히면 원인 분리가 안 된다.
  - dedup 키에 Kind 를 넣는 변경(별개 보류 아이디어, 가치 2)을 끼워 넣지 말 것 — 알림 억제 동작이 같이 바뀌면 이번 변경의 인과를 증명할 수 없다.
  - 테스트 함정: `mattermostConfig` 스냅샷은 15초 캐시 → 플래그를 바꾼 뒤 `server.invalidateMattermostCache()` 필수. Mattermost 전송은 `go func` 비동기 → webhook 수신은 채널+`time.After`, 미전송 증명은 `select ... default`. 빈 `ClusterID` 는 wildcard 가 아니라 독립 식별자이므로 폴백에서는 지금과 똑같이 `""` 를 그대로 넘긴다.
  - 보호 경로(auth/migrations/workflows) 는 건드리지 않는다. DDL·마이그레이션 변경 없음.

- 차선 후보: `/admin/k8s/events`(`internal/proxy/admin_k8s.go:478`)가 `limit` 쿼리 파라미터를 그대로 `ListK8sEvents` 에 넘기지만 스토어가 500 으로 하드 클램프해, `limit=1000` 을 요청한 호출자는 받지 못한 창을 받았다고 믿는다 — 응답에 실효 limit/잘림을 적는다(가치 2 / 위험 1 / S, 프로덕션 1파일).
