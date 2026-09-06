## 가드레일이 "restricted" 라고 부른 워크로드가 사실은 가장 안 굳혀진 워크로드였습니다

Pod Security Standards 는 누적입니다 — Restricted 는 Baseline 을 포함해 그 위에 얹힙니다. 이번 릴리즈는 그 누적 관계가 코드에서 끊겨 있던 지점 네 곳을 고칩니다. 네 결함 모두 방향이 같습니다: **하드닝되지 않은 워크로드가 "준수" 로 보고됐습니다.**

### ① 등급 판정이 자기가 방금 계산한 restricted 위반을 무시했습니다

`classifyPodSecurity` 의 등급 판정은 privileged·baseline 위반만 봤습니다 — `switch { case len(priv)>0 … case len(baseline)>0 … default: "restricted" }`. host namespace 를 안 쓰고 `privileged: true` 만 아니면 곧바로 최고 등급이 되는데, root 로 뜨고 기본 capability 를 다 들고 있는 것은 baseline 위반이 아닙니다. 그래서 **securityContext 가 아예 없는 평범한 Pod 가 전부 `restricted`** 로 분류됐습니다.

이 라벨은 소비자가 보는 유일한 값입니다. Pod Security 표는 `level !== 'restricted'` 로 걸러 그리고, 데이터 웨어하우스 export 도 restricted 행을 건너뛰며, 요약은 그것을 목표 상태로 셉니다. 즉 **하드닝이 가장 안 된 워크로드가 화면에서 사라졌고**, 이미 계산돼 `Violations` 에 들어 있던 "runAsNonRoot 미설정 · allowPrivilegeEscalation!=false · capabilities drop ALL 아님" 이 아무 데서도 표시되지 않았습니다. 클러스터 전체가 unhardened 여도 화면에는 "restricted 미만 워크로드 없음" 이 떴습니다.

이제 restricted 위반이 있으면 등급은 `baseline` 입니다. 위반이 없는 워크로드는 그대로 `restricted` 입니다.

### ② `enforce_pss_restricted` 는 `deny_privileged_runtime` 의 복사본이었습니다

두 룰이 `case` 를 공유해서, "PSS Restricted 강제" 라는 이름의 **Deny 게이트**가 정작 Restricted 프로파일 항목은 하나도 검사하지 않았습니다 — host namespace·hostPath·privileged/privesc 만 봤고, 그건 이름 그대로 더 약한 `deny_privileged_runtime` 이 이미 하는 일입니다.

그래서 securityContext 가 통째로 없는 Pod(= root, 기본 capability, 권한 상승 허용)가 이 게이트를 그냥 통과했고, **같은 Pod 를 보안 포스처 리포트는 바로 그 세 가지 Restricted 위반으로 적고 있었습니다** — 제품의 두 부분이 같은 Pod 에 정반대 판정을 내리고 있었던 셈입니다. 이제 두 곳이 같은 헬퍼(`restrictedProfileViolations`)를 쓰므로 화면과 게이트가 어긋날 수 없고, 위반 문구에 어떤 항목이 걸렸는지 함께 남습니다. 컨테이너가 없는 리소스(Service 등)에는 여전히 발화하지 않습니다.

### ③ 컨테이너의 `runAsNonRoot=false` 가 Pod 설정에 가려졌습니다

포스처 리포트가 두 값을 `!컨테이너 && !Pod` 로 읽어서, Pod 가 `runAsNonRoot: true` 면 컨테이너가 명시적으로 `false` 여도(= 그 컨테이너는 root 로 뜹니다) 위반이 아니었습니다. 컨테이너 securityContext 는 Pod 것을 **덮어씁니다**. v0.9.268 에서 `require_run_as_non_root` 룰에 대해 고친 것과 같은 결함이 포스처 경로에 남아 있었습니다.

### ④ `require_resource_limits` 가 CPU limit 만 있으면 통과시켰습니다

검사가 `len(limits) == 0` 하나뿐이라, `limits: {cpu: "500m"}` 처럼 **메모리 상한이 없는** 컨테이너가 준수로 판정됐습니다 — 메모리 무제한은 노드를 먹고 **다른 Pod 를 evict 시키는**, 이 가드레일이 막으려는 바로 그 실패 형태이자 실무에서 가장 흔한 형태입니다. 내보내는 Kyverno 패턴은 이미 `memory: "?*"` 와 `cpu: "?*"` 를 둘 다 요구하고 룰 설명도 "CPU·메모리 limits 누락 탐지" 였으므로, 세 표현 중 구현만 어긋나 있었습니다.

값이 비어 있는 키(`memory: ""`, `memory: null`)도 limit 이 아닙니다(Kyverno 의 `"?*"` 가 거절하는 형태). 위반 문구는 빠진 키를 이름으로 지목합니다(`app: resources.limits.memory 미설정`). YAML 숫자로 쓴 수량(`cpu: 1`)은 그대로 수량으로 인정합니다. 내보내는 Rego/Kyverno 본문도 함께 맞췄습니다.

### 검증

신규 테스트 6개를 **고치기 전 코드에 되돌려 붙여** 네 결함을 각각 지목하며 실패하는 것을 확인했습니다(하드닝된 Pod 가 계속 `restricted` 인지, 컨테이너가 없는 리소스에 발화하지 않는지 지키는 오탐 회귀 포함). `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.273.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.273.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.273.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.273.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.273.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.273
```
