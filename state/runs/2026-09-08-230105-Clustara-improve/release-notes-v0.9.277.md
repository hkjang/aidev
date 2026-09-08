## 와일드카드 인증서로 정상 서비스되는 Ingress 가 "평문 노출" 로 보고됐습니다

같은 Ingress 를 두 화면이 서로 다르게, 그리고 둘 다 틀리게 읽고 있었습니다 — 외부 노출 점검(Exposure Center)과 Ingress/PVC 연결성 점검(K8S-23/24)입니다. 이번 릴리즈는 그 오탐·집계 결함 다섯 지점을 고칩니다. **없는 위험이 목록 맨 앞에 올라왔고, 진짜 위험 하나는 아예 검사되지 않았습니다.**

### ① `*.example.com` 인증서를 쓰는 host 가 "TLS 미적용(평문 노출)" +30점이었습니다

`analyzer.AnalyzeExposure` 는 host 가 TLS 로 덮이는지를 `spec.tls[].hosts` 와 **문자열 그대로** 대조했습니다. 그런데 Kubernetes 는 이 필드에 와일드카드를 허용하고, `*.example.com` 은 DNS 라벨 **하나**를 덮습니다. 그래서 와일드카드 인증서로 정상 서비스되는 `app.example.com` 이 평문 노출로 채점돼, 노출 위험 목록에서 진짜 평문 Ingress 와 나란히 올라왔습니다.

이제 라벨 하나를 덮는 규칙으로 대조하고, DNS 이름이므로 대소문자를 무시합니다. `a.b.example.com` 은 `*.example.com` 으로 덮이지 않으므로 종전대로 미커버로 남습니다(오탐 회귀 테스트로 고정).

### ② 와일드카드 host 3개짜리 Ingress 하나가 `Total=1` 인데 `Wildcard=3` 을 만들었습니다

`SummarizeExposure` 의 `Plaintext`·`Wildcard` 가 finding 이 아니라 `RiskReasons` 를 셌습니다. host 를 여러 개 가진 Ingress 하나가 요약에서는 여러 건으로 잡혀, 총계와 내역이 서로 맞지 않았습니다. 이제 finding(Ingress) 개수를 셉니다.

같이 끊은 결합이 하나 더 있습니다 — 집계가 사유 **문구**를 문자열로 찾고 있었기 때문에, 문구를 다듬는 순간 집계가 조용히 0이 됐습니다. 사유 문자열을 상수로 묶어 그 경로를 없앴습니다.

### ③ 하나의 Ingress 가 자기 자신과 host 충돌로 잡혔습니다

`IngressDuplicateHost` 가 rule 단위로 owners 에 append 해서, 같은 host 아래 path 를 rule 두 개로 나눠 적은 **하나의** Ingress 가 중복 host 로 보고됐습니다(`ingresses: default/solo, default/solo`). 이제 host 를 Ingress 단위로 한 번만 셉니다.

겸사겸사 이 finding 만 `cluster_id` 가 비어 있던 것과, map 순회라 실행마다 findings 순서가 달라지던 것을 고쳤습니다(host 정렬).

### ④ 일치하지 않는 모든 요청을 받는 `spec.defaultBackend` 를 아무도 검사하지 않았습니다

`IngressBackendMissing` 은 rule 의 backend 만 봤습니다. rule 에 일치하지 않는 **모든 요청**이 도달하는 defaultBackend 가 존재하지 않는 Service 를 가리켜도 검사를 통과했고, 노출 분석의 `TargetServices` 에서도 빠져 있었습니다. 이제 둘 다 defaultBackend 를 포함합니다.

또 같은 Service 를 여러 path 가 참조하면 동일한 finding 이 그 수만큼 반복돼 응답의 `count` 를 부풀렸습니다 — 이제 Service 당 한 번만 보고합니다. 존재하는 defaultBackend 는 깨끗하게 통과하는지도 회귀 테스트로 고정했습니다.

### ⑤ A 의 증적에 B 의 `ProvisioningFailed` 가 그대로 붙었습니다

PVC Pending 의 증적 조건이 (볼륨 실패 reason) **또는** (message 에 이름 포함) 이었습니다. 네임스페이스에 PVC 가 여러 개면 다른 PVC 가 낸 프로비저닝 실패가 이 PVC 의 근거로 인용됐습니다.

이제 이벤트의 **involved object** 로 대상을 확인합니다 — PVC 자신에 기록되는 provisioning/binding 이벤트와, 청구 이름을 message 에 적는 Pod 의 mount/attach 실패만 붙습니다.

### 검증

신규 테스트 11개(analyzer 10 + proxy 1) 중 9개를 **고치기 전 코드에 되돌려 붙여** 각 결함을 지목하며 실패하는 것을 확인했습니다. 나머지 2개는 와일드카드가 깊은 host 를 덮지 않는지·존재하는 defaultBackend 가 깨끗한지 지키는 오탐 회귀입니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.277.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.277.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.277.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.277.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.277.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.277
```
