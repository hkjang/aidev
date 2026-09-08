## 다른 클러스터의 `prod` 가 이 클러스터의 검사를 대신 통과시켰습니다

`/admin/k8s/connectivity` 와 `/admin/k8s/security` 는 `cluster_id` 가 **선택** 파라미터입니다. 전 클러스터 보기에서는 인벤토리와 이벤트가 여러 클러스터를 한 번에 담는데, 이 두 분석의 교차 참조는 전부 **네임스페이스 이름만** 맞췄습니다. 네임스페이스 이름은 클러스터를 건너면 같은 네임스페이스가 아닙니다 — 그래서 다른 클러스터의 동명 네임스페이스가 이쪽 검사를 대신 통과시키고 있었습니다. 이번 릴리즈는 그 교차 참조 결함 다섯 지점과, 같은 함수에서 발견된 endpoint 판정 결함 하나를 고칩니다. **있는 위험이 리포트에서 사라졌고, 없는 위험 하나는 정상 구성에 붙었습니다.**

### ① 비어 있는 Service 가 목록에서 사라졌습니다

`analyzeServices` 는 Service 의 selector 를 네임스페이스만 맞춰 Pod 와 대조했습니다. 그래서 `dr` 클러스터의 `prod/api` Pod 가 `prod` 클러스터 `prod/api` Service 의 endpoint 로 계산됐고, **실제로 트래픽을 받을 Pod 가 하나도 없는 Service** 가 전 클러스터 보기에서 정상으로 넘어갔습니다. 이제 Pod 를 클러스터+네임스페이스로 묶어 자기 클러스터 안에서만 찾습니다.

### ② 존재하지 않는 Ingress backend 가 조용히 통과했습니다

Ingress backend 조회도 같은 네임스페이스-이름 대조였습니다. 이 클러스터에 없는 backend Service 를 가리키는 Ingress 가, 다른 클러스터에 같은 이름의 Service 가 있다는 이유로 `IngressBackendMissing` 없이 지나갔습니다. backend 도 소유 클러스터 안에서만 찾습니다.

### ③ 액티브/스탠바이 DR 구성이 라우팅 충돌로 잡혔습니다

`IngressDuplicateHost` 는 반대 방향의 오탐이었습니다. 두 클러스터의 Ingress 가 같은 host 를 서비스하는 것은 DR 구성의 정상 형태지 충돌이 아닌데, 중복 host 로 보고됐습니다. 이제 host 는 클러스터 안에서만 비교합니다.

### ④ A 클러스터의 이벤트가 B 클러스터 PVC 의 증적으로 인용됐습니다

이벤트 피드도 전 클러스터를 담습니다. PVC Pending 증적이 네임스페이스 이름만 맞췄기 때문에, 다른 클러스터의 같은 네임스페이스에서 난 프로비저닝 오류가 이 청구의 근거로 그대로 붙었습니다. 이제 같은 클러스터의 이벤트만 증적이 됩니다.

### ⑤ 보호되지 않은 네임스페이스가 SEC-06 리포트에서 통째로 빠졌습니다 (fail-open)

영향이 가장 큰 지점입니다. NetworkPolicy 공백 점검(SEC-06)은 "네임스페이스 집합 − 정책이 있는 네임스페이스 집합" 이라는 **차집합**이라, 한 클러스터의 `prod` 에 NetworkPolicy 가 하나 있으면 다른 **모든** 클러스터의 `prod` 가 덮였습니다. 정책이 하나도 없는 네임스페이스가 리포트에 아예 나타나지 않았다는 뜻입니다.

이제 클러스터별로 판정합니다. 어느 클러스터인지 알 수 있도록 `SecFinding` 에 `cluster_id` 를 담고(빈 값은 `omitempty` 라 기존 응답 모양은 그대로), map 순회라 실행마다 달라지던 findings 순서를 정렬로 고정했습니다.

### ⑥ 완료된 Job Pod 만 남은 Service 가 "endpoint 있음" 이었습니다

같은 함수의 endpoint 판정이 종료된 Pod 까지 셌습니다. API 서버는 `Succeeded`/`Failed` Pod 를 GC 전까지 인벤토리에 남기지만 endpoints 컨트롤러는 그 Pod 를 뺍니다. 그래서 완료된 Job Pod 만 남은 Service 가 정상으로 통과했습니다. 아직 정상이 아닐 뿐인 Pod(`Pending`·`CrashLoopBackOff`)는 not-ready 주소로 게시되므로 종전대로 셉니다.

### 검증

신규 테스트 6개 중 5개를 **고치기 전 코드에 되돌려 붙여** 각 결함을 지목하며 실패하는 것을 확인했고, 6번째는 정상이 아닐 뿐인 Pod 가 계속 endpoint 로 남는지 지키는 오탐 회귀입니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(20 패키지).

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.278.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.278.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.278.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.278.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.278.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  -e GATEWAY_SECRET=$(openssl rand -hex 32) \
  clustara:v0.9.278
```
