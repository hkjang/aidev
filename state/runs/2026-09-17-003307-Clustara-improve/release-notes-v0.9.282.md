## `tls-cert` 하나만 읽던 Role 이 모든 Secret 을 읽게 됐는데, RBAC Diff 는 "변경 없음" 이었습니다

SEC-08 RBAC Diff 는 Role/ClusterRole 의 최근 두 리비전을 비교해 **추가된 권한**을 보여 주는 화면입니다. 그런데 `RBACDiffExpansions` 는 각 규칙을 `apiGroup|resource|verb` 문자열로 펼쳐 **집합 차이**를 냈고, 그 표현이 볼 수 없는 것이 셋이었습니다. 이번 릴리즈는 그 판정 결함 3종과 UI·핸들러 결함 2종을 고칩니다. **있는 확대가 화면에서 사라졌고, 하드닝이 위험 확대로 보고됐으며, 전 클러스터 보기의 모든 행이 엉뚱한 클러스터로 연결됐습니다.**

### ① `resourceNames` 를 푼 확대가 항목을 만들지 않았습니다

`resourceNames` 는 문자열에 들어가지 않았습니다. 그래서 Secret `tls-cert` 하나만 `get` 하던 규칙이 네임스페이스의 **모든 Secret** `get` 으로 넓어져도 양쪽이 같은 `|secrets|get` 이라 아무 항목도 생기지 않았습니다 — 이 화면이 잡으라고 있는 바로 그 확대입니다.

이제 이전 리비전이 각 grant 를 이미 **덮고 있는지**(`rbacCovers`)를 API 서버의 매칭 규칙으로 판정합니다: 모든 슬롯의 `*`, `*/subresource`, 끝 `*` URL prefix, 그리고 resourceNames 는 객체 단위 제한. 이름으로 한정된 확대는 네 번째 `|name` 세그먼트로 표시됩니다(응답 `note` 와 UI 표 머리글 `apiGroup|resource|verb[|resourceName]` 에 명시).

### ② wildcard 를 좁힌 하드닝이 위험 확대로 올라왔습니다

wildcard 를 문자 그대로 비교했기 때문에 `resources: ["*"]` → `["secrets"]`, `verbs: ["*"]` → `["get","list"]`, `apiGroups: ["*"]` → `["apps"]`, `*/scale` → `deployments/scale` 같은 **축소**가 전부 "새로 추가된 risky 권한" 으로 보고됐습니다. 하드닝을 권한 상승이라 말하는 화면은 무시하도록 학습시킵니다. 이제 이전 wildcard 가 이미 덮던 권한은 세지 않습니다.

### ③ `nonResourceURLs` 에 `*` 를 얻은 ClusterRole 이 변경 없음이었습니다

비리소스 URL 규칙은 아예 읽지 않았습니다. 이제 URL 을 resource 슬롯에 담아 같은 커버리지 판정을 거칩니다(끝 `*` 는 prefix). `IsRiskyPermission` 은 `|name` 형식을 받고, 포스처 검사(`rbac-wildcard` high)와 맞춰 **apiGroups `*` 도 risky** 로 봅니다.

### ④ 전 클러스터 보기의 YAML 딥링크가 전부 페이지가 선택한 클러스터로 갔습니다

UI 가 `e.cluster_id` 를 읽는데 응답에 그 필드가 없었습니다. `rbacDiffEntry` 에 `cluster_id` 를 `omitempty` 로 추가해 각 행이 자신의 클러스터로 연결됩니다.

### ⑤ 바쁜 클러스터에서 안정적인 Role 이 인벤토리 창 밖으로 밀려 표시 없이 빠졌습니다

핸들러가 4000행을 kind 구분 없이 `updated_at` 순으로 받아 메모리에서 Role 만 남겼습니다. 변동이 잦은 워크로드가 창을 채우면 Role 은 창에 들어오지 못했고, 그 사실은 응답에 나타나지 않았습니다. 포스처 핸들러의 `SecurityRelevantKinds` 와 같은 방식으로 `Kinds: [Role, ClusterRole]` 로 좁혔습니다.

### 검증

신규 테스트 6개(analyzer 5 + proxy 1) 중 4개를 **고치기 전 코드에 되돌려 붙여** 각 결함을 지목하며 실패하는 것을 확인했습니다(풀린 resourceNames 항목 없음, 축소 5종이 확대로 보고, 비리소스 wildcard 항목 없음, apiGroups `*` 가 risky 아님). 5번째는 `*/scale`·다른 리소스의 group wildcard 가 과잉 적용되지 않는지 지키는 오탐 회귀, 6번째는 핸들러 종단 + `cluster_id` 입니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(20 패키지, 릴리즈 게이트 포함).

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.282.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.282.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.282.md | 오프라인 배포 가이드 |

```bash
sha256sum -c clustara-v0.9.282.tar.gz.sha256
gunzip -c clustara-v0.9.282.tar.gz | docker load   # Loaded image: clustara:v0.9.282
```
