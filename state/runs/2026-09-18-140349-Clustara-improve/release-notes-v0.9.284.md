## 오퍼레이터가 관리하는 Pod 의 삭제 승인 기록에 "자동 복구 없음" 이 적혔고, 모든 노드의 drain 에 "데이터 유실" 차단이 붙었습니다

Action Center 의 `delete_pod`·`drain` 승인 기록에는 **영향도**가 함께 저장됩니다 — 이 Pod 를 지우면 누가 다시 만드는지, 이 노드를 비우면 몇 개의 Pod 가 evict 되고 그중 로컬 스토리지를 쓰는 것이 몇 개인지. 그런데 `internal/action/impact.go` 는 Pod 의 소유를 내장 컨트롤러가 찍는 **라벨**(`pod-template-hash`·`controller-revision-hash`·`job-name`)로만 추정했습니다. 수집기(`kube.inventoryFromObject`)는 처음부터 `metadata.ownerReferences` 를 Spec 에 저장하고 있었고, 노드 drain 화면(`/admin/k8s/node-drain` → `podOwner`)은 이미 그것을 읽고 있었습니다 — 즉 제품의 두 경로가 **같은 Pod 를 다르게 판정**하고 있었습니다. 이번 릴리즈는 그 판정 결함 3종을 고칩니다. **컨트롤러가 재생성하는 Pod 가 standalone 으로 기록됐고, evict 되는 DB 레플리카가 DaemonSet 으로 집계됐으며, drain 이 건드리지 않는 Pod 때문에 사실상 모든 노드에 차단 사유가 붙었습니다.**

### ① ownerReferences 로만 소유된 Pod 가 "standalone Pod 이라 자동 재생성되지 않습니다" 로 기록됐습니다

오퍼레이터 CR(SparkApplication 등)이 직접 만든 Pod, bare ReplicaSet 의 Pod, Node 가 소유한 static Pod 는 위 라벨을 하나도 갖지 않습니다. 그래서 컨트롤러가 곧바로 다시 만드는 Pod 의 delete_pod 승인 기록에 "standalone Pod 이라 자동 재생성되지 않습니다 / standalone Pod 삭제는 승인이 필요합니다(자동 복구 없음)" 이 적혔습니다. 승인을 더 요구하는 fail-safe 방향이지만, 승인자가 읽는 기록이 거짓이었습니다.

### ② drain 미리보기가 evict 되는 StatefulSet Pod 를 "DaemonSet N" 으로 셌습니다

StatefulSet Pod 와 DaemonSet Pod 는 라벨 모양이 같습니다 — 둘 다 `controller-revision-hash` 가 있고 `pod-template-hash` 가 없습니다. 라벨만 보던 판정은 drain 이 실제로 evict 하는 데이터베이스 레플리카를 drain 이 건드리지 않는 DaemonSet Pod 로 집계했습니다.

### ③ DaemonSet Pod 가 evict 총계와 local-storage 에 들어가 모든 노드에 "데이터 유실" 차단이 붙었습니다

`kubectl drain` 은 DaemonSet Pod 를 evict 하지 않습니다. 그런데 미리보기는 그것을 evict 총계에 넣고, DaemonSet 이 늘 마운트하는 hostPath(로그 수집기·node-exporter·CNI)를 local-storage 로 세어 어느 노드를 골라도 "local storage Pod 데이터 유실" 차단 사유가 걸렸습니다. 같은 제품의 노드 drain 영향 분석(`AnalyzeDrainImpact`)은 이미 DaemonSet 을 제외하고 있었습니다.

### 고친 것

- `analyzer.PodOwnerReference`(controller 참조 우선, 없으면 kind 가 있는 첫 참조) / `PodControllerKind`(ownerReferences → 없을 때만 라벨, `statefulset.kubernetes.io/pod-name` 으로 StatefulSet 을 DaemonSet 과 구분)로 판정을 **한 곳**에 모았고, proxy 의 `podOwner` 와 action 패키지가 같은 함수를 씁니다.
- drain 미리보기는 `AnalyzeDrainImpact` 와 같이 DaemonSet 을 제외한 Pod 만 evict·local-storage·namespace 로 셉니다. 요약문은 "DaemonSet Pod N개는 evict 대상이 아닙니다" 와 PDB 는 노드 drain 영향 분석에서 확인하라는 안내를 적습니다. 응답 `details` 의 키(`affected_pods`·`local_storage_pods`·`daemonset_pods`·`namespaces`)는 그대로입니다.

### 검증

신규 테스트 5개(analyzer 2 + action 2 + proxy 1). proxy 테스트는 실제 API 객체를 `kube.InventoryFromObject` 로 저장하고 `/admin/k8s/actions` 를 호출해 응답 impact 와 저장된 DryRunDiff 를 읽는 종단 테스트입니다. 고치기 전 코드에 되돌려 붙여 action 2·proxy 1 이 각 결함을 지목하며 실패함을 확인했고, `go build ./...`·`go vet ./...`·`go test ./...` 전부 통과(20 패키지).

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.284.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.284.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.284.md | 오프라인 배포 가이드 |

```bash
sha256sum -c clustara-v0.9.284.tar.gz.sha256
gunzip -c clustara-v0.9.284.tar.gz | docker load
```
