## 정책 팩 가드레일 넷이 잘못된 대상에 걸리고, 걸려야 할 대상을 놓쳤습니다

`analyzer.EvaluatePolicies` 는 Admission 시뮬레이터(`POST /admin/k8s/policies/simulate`)·컴플라이언스 스캔(`GET /admin/k8s/policies/compliance`)·GitOps Stack 배포 게이트(`AnalyzeStackManifest`) 셋이 공유하는 **유일한** 룰 평가기이고, `Deny` 판정은 Stack 전체의 apply 를 막습니다. 룰을 하나씩 실제 매니페스트에 넣어 확인한 결과 넷이 잘못돼 있었습니다.

### ① 이미지를 pull 하지 않는 리소스가 전부 공급망 위반이 됐습니다

`disallow_unsigned_image`·`require_sbom`·`require_vuln_scan_attestation` 은 attestation 주석의 **부재**로 발화하는데 kind 범위가 없었습니다. 그래서 그런 주석을 가질 이유가 없는 Service·ConfigMap·Ingress·ClusterRole 이 모두 위반으로 잡혔습니다.

배포 경로에서는 Stack 안의 Service 하나 때문에 "이미지 서명 attestation 없음" 으로 **apply 전체가 차단**되고, 컴플라이언스 경로에서는 ClusterRole 하나마다 참조하지도 않는 이미지에 대한 findings 가 3건씩 쌓여 **실제 위반이 묻힙니다.** 룰 카탈로그와 내보내는 Kyverno/Rego 형태는 이미 대상을 "워크로드" 로 적고 있었으므로, 평가기만 그 문서와 어긋나 있었습니다. 이제 실제로 이미지를 pull 하는 리소스에만 발화합니다.

### ② 레지스트리 포트가 붙은 무태그 이미지가 mutable 태그 가드를 통과했습니다

`disallow_latest_tag` 가 `:` 가 하나라도 있으면 "태그가 있다" 로 읽었습니다. `registry.corp.local:5000/app` 은 kubelet 이 `:latest` 로 pull 하는데도 그대로 통과했습니다 — 포트가 붙은 사설 레지스트리는 이 제품이 대상으로 하는 폐쇄망의 **기본 형태**입니다. `[registry[:port]/]repo[:tag][@digest]` 로 분해하도록 바꿨고, digest 로 고정된 참조는 계속 통과합니다.

### ③ 디버그 컨테이너가 붙은 Pod 는 항상 limits 위반이었습니다

`require_resource_limits` 가 ephemeral container 까지 순회했는데, Kubernetes API 는 ephemeral container 에 `resources` 를 **허용하지 않습니다.** 만족시킬 방법이 없는 조건이라 승인된 디버그 세션이 붙어 있는 동안 그 Pod 는 계속 가드레일 위반으로 표시됐습니다 — `securityRelevantContainers` 의 주석이 "선언된 리소스를 재는 호출자는 이걸 쓰면 안 된다" 고 이미 적어 둔 바로 그 경우입니다.

### ④ 컨테이너의 `runAsNonRoot=false` 가 Pod 설정에 가려졌습니다

컨테이너 securityContext 는 Pod 것을 **덮어씁니다.** 따라서 Pod 가 `runAsNonRoot: true` 여도 컨테이너가 `false` 면 그 컨테이너는 root 로 뜨는데, `require_run_as_non_root` 는 Pod 값만 보고 준수로 판정했습니다.

### 초록불을 믿지 않았습니다

신규 테스트 4개를 **고치기 전 코드에 되돌려 붙여** 확인했습니다 — 넷이 각 결함을 **정확히 지목하며** 실패했습니다.

검증: `go build ./...` · `go vet ./...` · `go test ./...` 전부 통과(21 패키지).

---
- 이미지: `clustara:v0.9.268`
- 배포 압축본: `clustara-v0.9.268.tar.gz`
