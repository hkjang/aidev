## 승인한 리소스와 실행한 리소스가 달랐습니다

Action Center 의 실행기는 액션마다 **하나의 고정된 리소스 종류**를 addressing 합니다 — `delete_pod` 은 `DELETE /api/v1/namespaces/{ns}/pods/{resource_name}`, `cordon`·`uncordon` 은 `PATCH /api/v1/nodes/{resource_name}` 입니다. 그런데 요청에 적히는 `resource_kind` 는 아무도 검사하지 않는 자유 입력이었습니다. 검토·승인 화면이 무엇을 보여주든, 실행되는 대상은 이름 하나로만 정해졌습니다.

### ① "Deployment/web 에 delete_pod" 가 승인되고, Pod `web` 이 지워졌습니다

`resource_kind: Deployment`, `resource_name: web`, `action: delete_pod` 인 요청은 아무 데서도 막히지 않고 승인까지 간 다음, 실행 시점에 `/api/v1/namespaces/{ns}/pods/web` 로 갔습니다. 워크로드와 같은 이름의 Pod 가 있으면 **승인 화면에 한 번도 나온 적 없는 객체가 사라지고**, 감사 로그에는 그 삭제가 Deployment 이름으로 남습니다. 같은 형태의 요청이 `cordon` 이면 Pod 이름과 같은 이름의 **노드**를 차단합니다.

이제 세 지점에서 대조합니다.

- **영향도 산출기** 가 kind 불일치를 승인 사유로 올립니다 — 승인자가 화면에서 먼저 봅니다.
- **실행 디스패치** 가 Kubernetes API 로 아무것도 보내기 전에 거절합니다.
- **`kube.DeletePod`** 이 Scale·RolloutRestart 와 마찬가지로 kind 를 인자로 받아 자기 앞에서 확인합니다.

표기 차이는 그대로 통과합니다 — `Pod`·`pods`·`po`·`v1/Pod`, `deploy`·`sts`·`ds` 는 운영자와 Ops Agent 가 실제로 쓰는 이름이므로 모두 같은 것으로 봅니다. kind 가 비어 있으면 주장하는 바가 없으므로 판정하지 않습니다(기존 요청이 깨지지 않습니다). `scale` 대상이 DaemonSet 인 경우(= `/scale` 서브리소스가 없음)도 같은 자리에서 함께 걸립니다.

### ② 인벤토리에 없는 대상의 "현재 상태"가 지어낸 값이었습니다

두 요청 핸들러 모두 `GetK8sInventoryItem` 이 실패하면 **zero value 를 그대로** 영향도 산출기에 넘겼습니다. 없는 값이 "0" 으로 읽히면서, 관측한 적 없는 상태가 승인 기록에 사실처럼 적혔습니다.

- 아직 수집되지 않은(또는 이름이 틀린) 워크로드에 대한 scale 요청이 `replicas 0 → 5 (+5)` 라는 diff 로 남았습니다. 실제로 5개가 돌고 있어도 승인자는 "0 에서 올리는 요청" 으로 읽습니다.
- `delete_pod` 은 라벨이 하나도 없다는 이유(zero value 에는 라벨이 없습니다)로 **"standalone Pod 이라 자동 재생성되지 않습니다"** 라고 단정했습니다.

이제 대상을 인벤토리에서 찾지 못하면 현재 상태를 미확인으로 표시하고(`current_replicas`·`controller_owned` 는 비워 둡니다) 그 사실과 함께 승인 대상으로 넘깁니다. 인벤토리에 있는 정상 경로의 문구와 판정은 그대로입니다.

### 검증

신규 테스트 6개(action 4 + kube 1 + proxy 종단 1, 오탐 회귀 포함)를 **고치기 전 코드에 되돌려 붙여** 여섯 개가 각 결함을 정확히 지목하며 실패하는 것을 확인했습니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과합니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.271.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.271.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.271.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.271.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.271
```
