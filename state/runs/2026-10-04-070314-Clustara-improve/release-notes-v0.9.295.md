## 시끄러운 클러스터 하나가 조용한 클러스터의 조회 예산을 다 써서, 그 클러스터의 privileged 워크로드는 평가 자체가 되지 않았습니다

`POST /admin/k8s/notify/scan` 을 `cluster_id` 없이 돌리는 것이 함대 전체를 커버하는 사용법이고,
`docs/ADMIN_GUIDE.md` 도 그 한 줄을 cron 에 걸어 한 작업으로 모든 클러스터를 덮으라고 권합니다.
그런데 그 실행은 인벤토리 2000행(`updated_at` 역순)·이벤트 500행(`last_seen` 역순)·리비전
1000행(`observed_at` 역순)을 **등록된 모든 클러스터가 나눠 쓰는 단일 창**으로 읽었습니다.
컨트롤러가 행을 자주 고쳐 쓰는 클러스터 하나가 그 창을 혼자 채우면, 조용한 클러스터의
privileged 워크로드와 과도한 Role 은 **평가 자체가 되지 않았고** 알림은 그냥 오지 않았습니다.

응답의 `truncated`·`events_truncated`·`revisions_truncated` 는 "뭔가 잘렸다" 만 말하고 **어느
클러스터가 차례를 잃었는지** 는 말하지 않았습니다. 이 엔드포인트의 출력은 사람이 읽지 않으므로,
운영자가 볼 수 있는 증상은 "알림이 그냥 오지 않는다" 하나뿐이었습니다. 이번 릴리즈는 클러스터
하나하나에 자기 예산을 주고, 예산이 찬 클러스터를 이름으로 보고합니다.

### notify scan · 전 클러스터 실행에서 클러스터마다 자기 조회 예산을 씁니다

스캔은 대상 클러스터 목록을 먼저 정하고(`cluster_id` 가 있으면 그 하나, 비어 있으면
`ListK8sClusters` 로 등록된 클러스터 전부) **클러스터마다 같은 예산을 한 번씩** 씁니다.
응답·감사 로그에 두 필드가 추가됩니다.

- `clusters_scanned` — 이번 실행에서 각자 예산을 받은 클러스터 수.
- `clusters_truncated` — 세 창(인벤토리·이벤트·리비전) 중 하나라도 찬 클러스터 ID 목록.

기존 `truncated`·`events_truncated`·`revisions_truncated` 는 "하나라도 걸리면 true" 라는 함대
단위 의미를, `resources`·`events`·`revisions` 는 합계라는 의미를 그대로 유지하며
`window_notice`·`truncation_notice` 의 조건도 같습니다 — 이 키들을 읽는 호출자는 영향을 받지
않습니다. `events_error`·`revisions_error` 는 여러 클러스터를 돌 때 처음 실패한 것을
`"<cluster_id>: <오류>"` 로 적고, 대상이 하나일 때는 접두사 없이 지금까지와 같은 문자열을
적습니다. 인벤토리 조회 실패는 전과 같이 500 으로 끝내되 팬아웃 중이면 어느 클러스터에서
깨졌는지 메시지에 넣습니다.

`k8s_clusters` 가 비어 있으면(등록 전, 또는 조회 실패) 함대로 삼을 목록이 없으므로 지금까지와
**완전히 같은** 단일 창 1회 조회로 폴백하며 빈 `cluster_id` 를 그대로 넘깁니다 — 이 스키마에서
빈 `ClusterID` 는 wildcard 가 아니라 독립 식별자이므로 인벤토리 행의 `cluster_id` 값으로 대상을
만들어내지 않습니다. 팬아웃은 `3 × 등록 클러스터 수` 개의 인덱스 LIMIT 쿼리가 되고(이전에는
3개), 클러스터 수에 상한은 두지 않았습니다 — 조용히 버리는 상한은 지금 고친 결함과 같은
모양이기 때문입니다. 스토어 상한(이벤트 500·리비전 1000)과 `notifyScan*Budget` 값, 분석 호출
(`AnalyzeRCA`→`EnrichWithConfigChanges`→`AnalyzeSecurity`)·중복 제거 키·담당팀 라우팅·딥링크·조용한
시간 판정, DB 스키마·설정은 모두 그대로입니다.

### 올리기 전에

등록된 클러스터가 여럿이고 그중 하나가 창을 채우고 있었다면, **이 릴리즈 직후 첫 스캔에서
그동안 평가되지 않았던 클러스터의 워크로드가 한꺼번에 알림으로 나갈 수 있습니다**(dedup 창
6시간, 워크로드당 1건). 지금까지 가려져 있던 실재하는 위반이지만 양은 처음 한 번 많을 수
있으므로, 올린 뒤 첫 스캔은 `/admin/k8s/security` 를 먼저 확인할 수 있는 시간에 돌리십시오.
`clusters_scanned` 가 `k8s_clusters` 의 등록 수와 같은지, `clusters_truncated` 가 비었는지로
스캔이 함대를 온전히 덮었는지 판정할 수 있습니다.

### 검증

신규 종단 테스트 2개(실제 SQLite `openTestStore` + `store.NewAsyncLogger` + `NewServer` +
`httptest.NewServer(server.Routes())`, 실제 `kube.InventoryFromObject` privileged Pod 3개,
`UpsertK8sCluster` 등록 2건, httptest Mattermost webhook — 손으로 만든 대역 없음)를 수정 전
코드에 먼저 붙여 `the churning cluster must not spend the quiet cluster's query budget: map[...
resources:1 sent:1 truncated:true ...]` 와 `an empty cluster registry means one shared window, as
before` 로 빨강을 확인했습니다. 인과 확인으로 `targets` 를 `[]string{clusterID}` 로만 되돌려 같은
테스트가 다시 빨강이 되는 것도 보았습니다. 기존 notify 테스트 6파일은 `UpsertK8sCluster` 를 쓰지
않아 빈 레지스트리 폴백 경로로 무수정 통과합니다. `go build ./...`, `go vet ./...`,
`go test ./... -count=1`(20 패키지 전부 ok, 실패 0) 및 버전 일치·산출물 이름 릴리즈 게이트를
실행했습니다. 실 Kubernetes·PostgreSQL·실 Mattermost·브라우저 렌더링은 미검증이며, 프로덕션
파일은 `internal/proxy/k8s_notify.go` 1개입니다. DB 스키마·설정 변경은 없습니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.295.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.295.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.295.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.295.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.295.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.295
```
