## Pod 에 `runAsUser: 0` 을 적고 컨테이너는 비워 둔 워크로드가 "root 아님" 으로 보고됐습니다

`spec.securityContext.runAsUser: 0` 를 **Pod 레벨**에 적고 컨테이너는 아무것도 적지 않는 것이 워크로드를 root 로 돌리는 가장 흔한 형태입니다 — Pod 값은 덮어쓰지 않은 모든 컨테이너의 기본값이기 때문입니다. 그런데 세 보안 화면이 **컨테이너 securityContext 만** 읽어 그 워크로드를 root 아님으로 보고했습니다. 이번 릴리즈는 그 판정 결함 4종을 고칩니다. **있는 위험이 보안 화면에서 사라졌고, 없는 위험 하나는 정상 구성에 붙었으며, 제품의 두 부분이 같은 Pod 를 두고 반대 판정을 내놓고 있었습니다.**

### ① SEC-01 Pod Security 가 `runAsUser=0` 위반을 아예 만들지 않았습니다

`classifyPodSecurity` 는 컨테이너의 `securityContext.runAsUser` 만 봤습니다. 그래서 Pod 레벨에 0 을 적고 컨테이너는 선언하지 않은 — 즉 **모든 컨테이너가 실제로 UID 0 으로 뜨는** — Pod 에 baseline 위반이 하나도 붙지 않았습니다.

이제 컨테이너 → Pod 우선순위로 실제 UID 를 해석해 상속된 root 도 위반으로 적고, 상속인 경우 사유에 `runAsUser=0 (Pod securityContext 상속)` 으로 출처를 밝힙니다.

### ② 반대 방향: 컨테이너가 실제 UID 로 덮어쓴 Pod 가 root 로 채점됐습니다

Runtime Security Profile(CLU-OCP-03)의 `podSecurityInput` 은 Pod 레벨을 읽기는 했지만 **무조건** 적용했습니다. 그래서 Pod 가 0 을 적어도 컨테이너가 non-root UID 로 덮어쓴 정상 구성 — 실제로는 root 로 돌지 않는 워크로드 — 이 root 로 올라왔습니다.

이제 우선순위가 **양방향**으로 적용됩니다. 명시적 `null` 은 미설정으로 봅니다: 이미지의 `USER` 가 적용되므로 spec 은 어느 쪽도 주장하지 않습니다.

### ③ 지금 붙어 있는 privileged 디버그 컨테이너가 "위험 설정 없음" 으로 채점됐습니다

같은 화면이 `containers` 만 순회했습니다. 그래서 privileged init 컨테이너, 그 컨테이너가 추가한 capability, 그리고 **디버그 세션이 방금 붙인 privileged ephemeral 컨테이너**가 전부 채점에서 빠졌습니다. 같은 Pod 를 정책 엔진과 SEC-01 은 이미 위반으로 적고 있었으므로, 제품의 두 부분이 한 Pod 를 두고 반대 판정을 내놓고 있었습니다.

이제 포스처·정책 엔진이 이미 쓰던 `analyzer.SecurityRelevantContainers`(regular + init + ephemeral)를 순회합니다.

### ④ Workspace 건강도가 런타임 보안 화면과 같은 Pod 를 두고 다르게 답했습니다

`podHasRuntimeSecurityRisk` 는 Pod 레벨 securityContext 를 통째로 무시하고 역시 `containers` 만 봤습니다. 그래서 두 화면이 같은 Pod 에 서로 다른 답을 내놓았습니다.

판정을 한 곳으로 모았습니다 — `analyzer.EffectiveRunAsUser` 가 컨테이너 → Pod 우선순위를 양방향으로 적용하고, `analyzer.PodRunsAsRoot` 가 세 호출자 모두에게 한 번만 답합니다.

### 검증

신규 테스트 8개 중 6개를 **고치기 전 코드에 되돌려 붙여** 각 결함을 지목하며 실패하는 것을 확인했습니다. 나머지 2개는 컨테이너가 Pod 의 0 을 덮어쓴 경우와 Pod 의 non-root 를 컨테이너가 덮어쓴 경우를 지키는 오탐 회귀입니다. `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(20 패키지).

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.280.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.280.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.280.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.280.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.280.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  -e GATEWAY_SECRET=$(openssl rand -hex 32) \
  clustara:v0.9.280
```
