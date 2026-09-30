- 과제: notify scan 의 events·revisions 조회가 실패하거나 상한에 걸려도 "이상 없음" 과 똑같이 보고되는 것을 고치기 (가치 3 / 위험 2 / 작업량 S)

- 왜: `handleK8sNotifyScan`(`internal/proxy/k8s_notify.go:195-196`)이 `events, _ := s.db.ListK8sEvents(ctx, clusterID, 500)` 와 `revisions, _ := s.db.ListK8sRevisions(..., Limit: 1000)` 로 **오류를 `_` 로 버리고** 결과가 상한에 찼는지도 보지 않는다. DB 오류든 창 포화든 `AnalyzeRCA(items, events)` 는 빈/부분 이벤트로 돌아 `{"sent":0,"evaluated_rca":0,"truncated":false}` 를 내놓는데, 이것은 **정말 아무 문제 없는 클러스터의 응답과 한 글자도 다르지 않다**. v0.9.290 이 인벤토리에만 `truncated`·`truncation_notice` 를 붙였고(같은 함수 191-193행) events·revisions 는 그 계약 밖에 남았다. 사람이 결과를 보지 않는 경로라서 누락은 "알림이 그냥 안 온다" 로만 나타난다.

- 수용 기준:
  1) `ListK8sEvents` 또는 `ListK8sRevisions` 가 오류를 반환하면 스캔 응답과 감사 로그에 그 사실이 남는다(예: `events_error`/`revisions_error` 또는 기존 `truncation_notice` 와 같은 결의 안내 필드). 스캔 자체는 계속 진행해도 되지만(인벤토리 기반 security 분석은 여전히 유효), **조용히 성공으로 보고하면 안 된다**.
  2) events·revisions 결과가 요청 상한에 찼을 때 응답이 그것을 말한다(예: `events`/`revisions` 개수 + `truncated` 계열 플래그). 기존 `truncated`(인벤토리) 의 의미를 바꾸지 말 것 — `k8s_notify_scope_test.go:170,185` 가 그 키를 직접 단언한다.
  3) 테스트가 증명할 것: (a) 이벤트 조회가 포화된 상태에서 스캔이 깨끗한 클러스터와 **구별 가능한** 응답을 준다, (b) 정상·비포화 스캔은 새 플래그가 꺼진 채로 기존과 같은 응답을 준다(회귀 없음), (c) 기존 `truncated` 단언 2개가 그대로 통과한다.

- **구현자가 반드시 알아야 할 함정 (직접 소스에서 확인함)**: 인벤토리에 쓴 "상한+1 을 요청해 잘림을 감지" 트릭을 **events·revisions 에는 그대로 복사할 수 없다.** `internal/store/k8s.go:684-685` 가 `boundedLimit(limit, 100, 500)`, `internal/store/k8s.go:1213-1214` 가 `boundedLimit(f.Limit, 100, 1000)` 로 하드 상한을 걸어 두어, 501/1001 을 요청해도 스토어가 조용히 500/1000 으로 깎는다(`boundedLimit` 은 `k8s.go:1082`). 반면 인벤토리는 `boundedLimit(f.Limit, 200, 10000)`(`k8s.go:583`)이라 2001 요청이 통했고 기존 로직은 정상이다 — **인벤토리 경로는 멀쩡하니 건드리지 말 것.**
  → 따라서 포화 감지는 "요청 상한과 같은 개수가 돌아왔으면 잘렸을 수 있다"(`len(events) >= 500`) 로 하거나, 스캔이 상한보다 **작은** 값을 요청해 +1 여유를 만드는 방식으로 해야 한다. 스토어의 하드 상한(500/1000)을 올리는 것은 이번 범위 밖이다 — 그 상한은 40곳 넘는 호출자가 공유한다(아래 차선 후보 참고).

- 건드릴 파일 (프로덕션 1개):
  - `internal/proxy/k8s_notify.go:195-196` — 두 조회의 오류를 받아서 보관하고, 결과 포화 여부를 계산.
  - `internal/proxy/k8s_notify.go:277`(`s.auditAdmin(... "k8s.notify.scan" ...)`)와 `279-283`(응답 `out` 맵), `293-295`(`truncation_notice` 블록) — 새 필드/안내를 같은 곳에 추가. 감사 로그와 응답이 **같은 사실**을 말하게 할 것(지금도 둘이 같은 키를 쓴다).
  - 신규 테스트 파일 1개(예: `internal/proxy/k8s_notify_window_test.go`).

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  1) `go test ./internal/proxy -run 'NotifyScan|K8sNotify' -count=1` — 이번 정찰에서 실행해 `ok clustara/internal/proxy 1.333s` 확인함.
  2) `gofmt -l <손댄 파일>` (출력 없어야 함)
  3) `go build ./...` → `go vet ./...`
  4) `go test ./... -count=1` (전체 약 80s, proxy 단독 약 60~75s)

- 테스트 배선 (이 저장소 표준 — 손으로 만든 대역 금지):
  `openTestStore(t)`(t.TempDir SQLite) + `store.NewAsyncLogger` + `NewServer(testConfig(...))` + `httptest.NewServer(server.Routes())` + `postJSON`. 기존 예시는 `internal/proxy/k8s_notify_scope_test.go`(포화/truncated 단언)와 `k8s_notify_timezone_test.go` 를 그대로 볼 것.
  - 이벤트 포화는 실제 `store.K8sEvent` 를 500개 이상 넣어 재현하는 것이 정직하지만 느리다. 스토어 상한을 낮출 수 있는 훅이 없으므로, 스캔이 요청하는 값을 `notifyScanBudget` 처럼 **패키지 var 로 빼서** 테스트가 작게 줄이는 방식이 기존 관례와 맞는다(`k8s_notify.go:143` 가 선례).
  - 조회 오류 재현 경로는 **미확인** — `s.db` 가 구체 타입 `*store.SQLStore` 라 오류 주입 지점이 없을 수 있다. 그 경우 오류 경로는 닫힌 DB 등으로 재현하거나, 재현이 비싸면 수용 기준 1) 은 코드로만 처리하고 테스트는 2)·3) 에 집중할 것(무리해서 대역을 만들지 말 것 — 운영자 지시).

- 위험과 피할 것:
  - `mattermostConfig` 스냅샷 15초 캐시 — 테스트에서 플래그를 바꾸면 `server.invalidateMattermostCache()` 필수.
  - Mattermost 전송은 `go func` 비동기 — httptest webhook 수신은 채널+`time.After`, 미전송 증명은 `select ... default`.
  - 빈 `ClusterID` 는 wildcard 가 아니라 독립 식별자다. 전 클러스터 스캔(`cluster_id` 없음)은 500 이벤트를 모든 클러스터가 나눠 쓰므로 포화가 가장 잘 난다 — 재현은 여기서 하는 게 쉽다.
  - `internal/proxy/docs_reference_test.go`·`release_gate_test.go` 가 docs 의 API 경로·버전 마커를 검사한다. 응답 필드를 문서에 적는다면 `docs/ADMIN_GUIDE.md` 의 notify 절에만, 무관한 버전 마커는 건드리지 말 것.
  - 보호 경로(auth/migrations/workflows) 와 `internal/store/` DDL·상한은 이번 회차에서 **건드리지 않는다.**
  - 알림 개수·dedup 동작은 바뀌면 안 된다. 이번 변경은 **보고만** 추가하는 것이다.

- 차선 후보: **events 상한 500 이 호출자 요청을 조용히 깎는 계약 불일치** — `admin_k8s_dw.go:295`·`admin_k8s_resource_advisor.go:41` 이 2000 을, `admin_k8s_pods.go`·`admin_k8s_reports.go` 등 10곳 넘게 1000 을 요청하지만 `boundedLimit(limit, 100, 500)` 이 전부 500 으로 깎는다(revisions 는 2000 요청 → 1000). 호출자는 자기가 요청한 창을 받았다고 믿고 분석한다. 가치는 더 높지만 호출자가 40곳이 넘어 한 회차 6파일 규칙을 넘는다 — 하려면 **DW fact 적재 한 경로만** 잘라서 하고, 상한을 올리는 쪽이 아니라 "받은 개수를 호출자가 알 수 있게" 하는 쪽으로 좁힐 것.
