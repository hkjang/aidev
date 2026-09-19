## 전 클러스터 보안 점검에서 Pod Security 딥링크가 빈 클러스터로 갔고, DW 리포트에서 보안 행이 사라졌으며, 두 클러스터의 privileged 알림이 하나로 합쳐졌습니다

`/admin/k8s/security`·`/admin/k8s/dw/sink`·`/admin/k8s/notify/scan` 은 모두 `cluster_id` 가 **선택** 파라미터입니다 — 비우면 등록된 모든 클러스터를 한 번에 분석합니다. 그런데 세 자리가 결과에 **요청 파라미터의** cluster_id 를 적었습니다. 전 클러스터 실행에서 그 값은 빈 문자열이므로, finding 이 어느 클러스터에서 나왔는지가 세 경로 모두에서 사라졌습니다. 이번 릴리즈는 그 결함 3종을 고칩니다. **UI 가 읽는 필드는 응답에 없었고, DW 행은 빈 클러스터로 적재됐으며, 알림은 첫 클러스터가 dedup 창을 잡아 나머지를 삼켰습니다.**

### ① Pod Security 표의 YAML·토폴로지 딥링크가 빈 클러스터로 갔습니다

관리 UI 의 Pod Security 표는 각 행의 `p.cluster_id` 로 YAML/토폴로지 딥링크를 만듭니다. 그런데 `analyzer.PodSecurityResult` 에는 ClusterID 필드가 없어 전 클러스터 보기에서 그 값이 항상 비었습니다 — v0.9.282 에서 고친 RBAC Diff 의 `rbacDiffEntry` 와 같은 모양의 결함입니다. `PodSecurityResult` 에 `cluster_id`(omitempty)를 더하고 `classifyPodSecurity` 가 인벤토리 항목의 클러스터를 채웁니다. Level/Violations 판정과 `summarize` 점수는 손대지 않았습니다.

### ② 전 클러스터 DW sink 에서 security_finding 행만 `cluster_id=""` 로 적재됐습니다

`k8sSecurityRows` 는 finding 이 v0.9.278 부터 가진 `f.ClusterID` 를 무시하고 요청 clusterID 를 모든 행에 적었습니다. `/admin/k8s/dw/sink` 를 `cluster_id` 없이 돌리면 change/event/workload_health 행은 실제 클러스터를 갖는데 security_finding 행만 비어, `/admin/k8s/dw/report?cluster_id=X` 필터가 보안 행을 하나도 돌려주지 않았습니다. 이제 RBAC·Images·Network·PodSecurity 행 모두 finding 자신의 클러스터를 쓰고, 비어 있을 때만 요청 clusterID 로 폴백합니다.

### ③ notify scan 이 두 클러스터의 동명 privileged 워크로드를 한 알림으로 합쳤습니다

`handleK8sNotifyScan` 의 dedup 키(`clusterID|podsec/ns/name`)·namespace owner 조회(NOTI-04)·딥링크가 전부 요청 clusterID 였습니다. 전 클러스터 스캔에서는 첫 클러스터의 알림이 6시간 dedup 창을 잡아 나머지 클러스터의 같은 이름 워크로드가 **알림 없이** 빠졌고, 보낸 하나의 링크·담당팀 채널도 빈 클러스터로 갔습니다. RCA·RBAC·PodSecurity 알림 모두 finding 자신의 클러스터(RCAFinding·SecFinding·PodSecurityResult 모두 보유)로 dedup·라우팅·링크를 만듭니다.

### 호환성

단일 클러스터 요청에서는 finding 의 클러스터와 요청 clusterID 가 같으므로 기존 dedup 기록·DW 행과 그대로 호환됩니다. 폴백은 finding 이 클러스터를 갖지 않을 때만 요청 값을 씁니다. **`cluster_id` 없이 돌리던 전 클러스터 notify 스캔을 운영 중이라면**, dedup 키가 `|podsec/…` 에서 `<cluster>|podsec/…` 로 바뀌므로 배포 직후 6시간 창 안에서도 클러스터별로 기존 finding 이 **한 번** 다시 알림됩니다 — 의도된 동작이며 그 뒤로는 평소 dedup 이 적용됩니다. `k8sCostRows` 의 clusterID 는 집계 단위라 그대로이고, UI 는 이미 `p.cluster_id` 를 읽고 있었으므로 변경이 없습니다.

### 검증

신규 테스트 3개(analyzer 1 + proxy 2). 두 번째 proxy 테스트는 실제 Pod 객체를 `kube.InventoryFromObject` 로 두 클러스터에 저장하고 Mattermost webhook 을 httptest 로 받아 `/admin/k8s/notify/scan` 을 호출하는 종단 테스트입니다. 세 테스트를 **고치기 전 코드에서 돌려** 각각 빈 `ClusterID`·빈 `cluster_id` 행·`sent=1`(두 클러스터가 하나로 dedup) 로 실패하는 것을 확인했습니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(20 패키지).

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.285.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.285.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.285.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
sha256sum -c clustara-v0.9.285.tar.gz.sha256
gunzip -c clustara-v0.9.285.tar.gz | docker load
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.285
```
