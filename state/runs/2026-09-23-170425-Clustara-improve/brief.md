# 과제서 (2026-09-23 정찰) — Clustara

- **과제: 알림이 전달되지 않는 상태의 `/admin/k8s/notify/scan` 이 6시간 dedup 창을 소비하고 `sent` 를 거짓으로 보고하는 문제 (가치 3 / 위험 1 / 작업량 S)**

## 왜
`internal/proxy/k8s_notify.go:106` 의 `notify` 클로저는 `ShouldSendK8sNotification`(dedup 기록 쓰기)과 `sent++` 를 **먼저** 하고, 실제 전송 `notifyMattermostTo`(`internal/proxy/mattermost.go:73`)는 `cfg.enabled == false` · `cfg.webhookURL == ""` · `cfg.events[category] == false` 이면 아무것도 하지 않고 조용히 반환한다 — 그래서 Mattermost 가 꺼져 있거나 webhook 이 아직 없거나 해당 카테고리가 mute 된 상태에서 스캔하면 ① 응답과 감사 로그가 보내지 않은 알림을 `sent=N` 으로 보고하고 ② 그 finding 의 dedup 기록이 6시간 창에 남아, 운영자가 곧바로 Mattermost 를 켜도 **같은 high/critical finding 이 최대 6시간 동안 조용히 누락**된다. `docs/ADMIN_GUIDE.md:369` 가 이 엔드포인트를 cron/`/loop` 으로 주기 호출하라고 안내하므로, 알림 구성 전에 cron 부터 돌리는 흔한 순서에서 정확히 이 침묵이 생긴다. 고치면 dedup 창은 실제로 전달된 알림에만 쓰이고 `sent` 는 실제 전송 건수를 뜻하게 된다.

## 수용 기준
1. `mattermost_enabled` 가 false 이거나 `mattermost_webhook_url` 이 비어 있거나 `mattermost_events` 가 해당 카테고리를 빼고 있으면, 스캔은 그 finding 에 대해 **dedup 기록을 쓰지 않는다** — 설정을 켠 직후의 스캔에서 같은 finding 이 정상적으로 전달된다.
2. 같은 상황의 응답 `sent` 는 0(전달된 건수)이며 감사 로그(`k8s.notify.scan`)의 `sent` 도 같은 값이다. 전달 가능한 카테고리의 finding 은 지금과 똑같이 전송·집계된다(기존 두 종단 테스트 `TestK8sNotifyScanRoutesEachFindingToItsOwnCluster`, 이어지는 probe 테스트가 그대로 통과해야 한다).
3. 테스트는 **실제 배선**으로 증명한다: `openTestStore` + `NewServer(testConfig(...))` + `httptest.NewServer(server.Routes())` + 실제 httptest webhook 으로 (a) Mattermost 꺼짐 상태 스캔 → `sent=0`·webhook 무수신, (b) 그 다음 `SetFlag`(`mattermost_enabled=true`,`mattermost_webhook_url=hook.URL`) + `server.invalidateMattermostCache()` 후 재스캔 → 같은 finding 이 실제로 webhook 에 도착하고 `sent>0`. 수정 전 코드에서 (b) 가 `sent=0`·무수신으로 실패함을 먼저 확인할 것. 카테고리 mute(`mattermost_events="k8s_failure"`) 로 `k8s_security` finding 만 막히는 경우도 한 케이스로 덮을 것.

## 건드릴 파일
- `internal/proxy/mattermost.go` — 전달 가능 여부 판정을 함수 하나로 추출(예: `func (s *Server) mattermostDeliverable(ctx context.Context, category string) bool`)하고 **`notifyMattermostTo` 의 기존 guard(`!cfg.enabled || cfg.webhookURL == "" || !cfg.events[category]`)도 그 함수를 쓰게** 할 것 — 같은 값을 두 경로가 다르게 읽으면 안 된다(운영자 지시 사항). 판정 로직·기본값(`mattermost_events` 미설정 시 전 카테고리 on)은 바꾸지 말 것.
- `internal/proxy/k8s_notify.go:handleK8sNotifyScan` 의 `notify` 클로저 — `ShouldSendK8sNotification` 호출 **앞에** 전달 가능 여부를 확인하고 아니면 즉시 반환. 클러스터 fallback(`cluster = firstNonEmpty(cluster, clusterID)`)·dedup 키·owner 라우팅·딥링크는 v0.9.285 에서 정착한 그대로 유지.
- (선택, 범위 최소) 응답에 무엇이 막혔는지 알리고 싶으면 기존 `quiet_hours` 억제와 같은 모양의 필드 추가만 고려하되, 필드를 늘리지 않고 끝나면 그대로 두는 편이 낫다.
- `internal/proxy/k8s_notify_scan_test.go` — 위 (a)/(b)/mute 회귀 추가. 기존 두 테스트의 fixture 패턴(`kube.InventoryFromObject` → `item.ID/ClusterID` 지정 → `UpsertK8sInventory`, `UpsertK8sNamespaceOwnership`, `postJSON(... "/admin/k8s/notify/scan" ...)`)을 그대로 재사용할 것.

## 검증 명령
```
go test ./internal/proxy -run 'K8sNotify|Mattermost|QuietHours' -count=1   # 정찰에서 baseline 통과 확인(0.65s)
go test ./internal/proxy ./internal/store -count=1                          # proxy 전체는 약 60s
go build ./... && go vet ./... && go test ./...                             # 최종 게이트(약 80s)
gofmt -l internal/proxy/mattermost.go internal/proxy/k8s_notify.go internal/proxy/k8s_notify_scan_test.go
```

## 위험과 피할 것
- **`mattermostConfig` 스냅샷 캐시는 15초 TTL**(`mattermostSnapshotTTL`)이다. 테스트에서 플래그를 바꾼 뒤 반드시 `server.invalidateMattermostCache()` 를 부를 것 — 안 부르면 (b) 단계가 캐시 때문에 실패해 원인 오진으로 이어진다.
- 전송은 `go func` 비동기다. webhook 수신은 채널 + `time.After` 로 기다릴 것(기존 테스트 패턴). "보내지 않았다" 를 증명하는 쪽은 `select { case <-received: t.Fatal ... default: }` 로.
- `notifyMattermost`/`notifyMattermostTo` 는 k8s 밖의 cost·secret·approval·provider 등 다수 호출자가 있다. **동작을 바꾸지 말고** 조건만 함수로 추출할 것(동작이 같아야 다른 경로가 안 깨진다).
- webhook POST 가 네트워크 오류로 실패해도 dedup 은 이미 쓰인다 — 이번 범위 밖(비동기 전송 결과를 dedup 에 되먹이는 것은 구조 변경이다). 손대지 말고 남길 것.
- 보호 경로(`server.go:currentAccessClaims`·`mcp_oauth.go`·`keycloak*.go`·`store/sqlstore.go` DDL·`.github/`)는 건드리지 말 것. `ShouldSendK8sNotification` 의 store 구현·스키마도 바꿀 필요가 없다.
- 버전·`scripts/changelog.txt`·docs 마커는 손대지 말 것(`internal/proxy/release_gate_test.go`·`docs_reference_test.go` 가 docs 의 API 경로/버전을 검사한다 — 무관한 문서 편집은 회피).
- 과거 교훈: 손으로 만든 대역이 아니라 실제 `Server.Routes()`·실제 SQLite·실제 httptest webhook 을 통과시켜 증명할 것.

## 차선 후보
**notify scan 의 Pod Security dedup 키에 Kind 추가 (가치 2 / 위험 1 / 작업량 S)** — `k8s_notify.go:145` 가 `"podsec/"+p.Namespace+"/"+p.Name` 이라 같은 클러스터·같은 네임스페이스의 동명 Deployment 와 Pod(또는 Job/CronJob)가 한 알림으로 합쳐진다. RBAC 키(`"rbac/"+f.Namespace+"/"+f.ResourceName+"/"+f.Rule`)도 Kind 가 없다. 키를 바꾸면 기존 6시간 기록과 한 번 어긋나 일회성 중복 알림이 날 수 있다는 점을 커밋 메시지에 남길 것.
