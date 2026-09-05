## 실클러스터 write 경로가 잘못된 대상으로 요청을 만들었습니다

`internal/kube` 의 executor 는 Action Center 승인 뒤 **실제로 클러스터를 바꾸는** 곳이고, stack applier 는 GitOps Stack 의 server-side apply 경로입니다. 둘 다 요청을 URL 문자열로 조립하는데, 그 조립이 다섯 군데에서 틀려 있었습니다.

### ① 대상 이름이 비면 collection URL 이 만들어졌습니다

`DeletePod` 는 `/api/v1/namespaces/{ns}/pods/{name}` 을 조립합니다. name 이 비면 남는 것은 `/api/v1/namespaces/{ns}/pods/` — Kubernetes API 가 **deletecollection 으로 처리하는 바로 그 URL** 입니다. 즉 이름 없는 `delete_pod` 한 건이 네임스페이스의 Pod 를 **전부** 지웁니다.

그리고 그런 요청이 실제로 만들어질 수 있었습니다. 개발자 셀프서비스 요청의 검증기인 `analyzer.PlanDevRequest` 는 `in.ResourceName != ""` 로만 확인하는데, 핸들러는 `strings.TrimSpace(in.ResourceName)` 를 저장합니다. 공백만 있는 `resource_name` 이 검증을 통과해 **대상이 빈 액션 요청**으로 적재됐습니다.

이제 검증기는 trim 후 비교하고, executor 는 요청 경로의 검증을 믿지 않고 자기 앞에서 빈 namespace·name·node 를 거절합니다 — 요청은 전송되지 않습니다.

### ② kubectl 단축 이름 `sts`·`ds` 가 거절됐습니다

`normalizeWorkloadKind` 가 switch 앞에서 복수형 `s` 를 무조건 떼어내는 바람에 `sts`→`st`, `ds`→`d` 가 됐고, 바로 아래의 `case "sts"`·`case "ds"` 는 **한 번도 실행되지 않는 코드**였습니다. `ResourceKind` 는 자유 입력이고 단축 이름은 운영자와 Ops Agent 가 실제로 쓰는 표기입니다 (`deploy` 는 `s` 로 끝나지 않아 우연히 동작하고 있었습니다).

### ③ DaemonSet scale 이 승인 뒤에야 실패했습니다

apps/v1 DaemonSet 에는 `/scale` 서브리소스가 없습니다 — 노드마다 1개라는 것이 정의 그 자체입니다. 그런데 `workloadResourcePlural` 이 이를 scale 대상으로 받아들여, 요청 → 영향도 평가 → 승인을 **모두 통과한 다음** 실행 시점에 404 로 끝났습니다. 이제 앞단에서 이유와 함께 거절합니다.

### ④ cluster-scoped kind 가 네임스페이스 경로로 apply 됐습니다

`resolveStackTargets` 는 모든 문서에 stack 의 namespace 를 채웁니다. 그래서 `clusterScopedKinds` 목록에 없는 cluster-scoped kind 는 `/namespaces/{stack}/...` 로 조립돼 실패할 수밖에 없고, Stack 안의 그런 문서 하나가 전체 apply 를 **부분 적용**으로 만듭니다.

`IngressClass`·`PodSecurityPolicy` 는 `pluralizeKind` 의 불규칙 목록에는 이미 들어 있으면서 이 목록에는 빠져 있었습니다. 함께 `ValidatingWebhookConfiguration`·`MutatingWebhookConfiguration`·`APIService`·`RuntimeClass`·`CSIDriver`·`VolumeSnapshotClass`·`ValidatingAdmissionPolicy`·`ValidatingAdmissionPolicyBinding` 도 추가했습니다.

### ⑤ write 경로만 경로 세그먼트를 이스케이프하지 않았습니다

같은 패키지의 읽기 경로(`podLogRequest`·`podExecURL`)는 `url.PathEscape` 를 씁니다. 그런데 apiVersion·kind·name 을 저장된 manifest 에서 그대로 받아 조립하는 `apiResourcePath` 만 날것으로 이어붙였습니다. 값 안의 슬래시 하나가 `force=true` server-side apply 를 **운영자가 검토하지 않은 리소스**로 돌릴 수 있었습니다.

### 초록불을 믿지 않았습니다

신규 테스트 5개를 **고치기 전 코드에 되돌려 붙여** 확인했습니다 — 다섯 개가 각 결함을 **정확히 지목하며** 실패했습니다.

검증: `go build ./...` · `go vet ./...` · `go test ./...` 전부 통과.

---
- 이미지: `clustara:v0.9.270`
- 배포 압축본: `clustara-v0.9.270.tar.gz`
