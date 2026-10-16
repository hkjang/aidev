## CrashLoopBackOff 가 도는 클러스터의 원인 분석이 "아무 문제 없는 클러스터" 와 똑같이 비어 있었습니다

`GET /admin/k8s/rca` 는 인벤토리를 **Kind 필터 없이** `updated_at` 내림차순 1000행으로 읽었습니다.
그런데 원인 후보를 만드는 `AnalyzeRCA` 가 실제로 읽는 Kind 는 `RCARelevantKinds()` 가 이름을 적어 둔
`CronJob`·`DaemonSet`·`Deployment`·`Job`·`Node`·`Pod`·`StatefulSet` 일곱 개뿐입니다.

분석 대상이 아니면서 자주 갱신되는 ConfigMap·Secret·Endpoints 는 바쁜 클러스터에서 가장 최신 행입니다.
그래서 그 1000행 창을 비분석 Kind 가 채우고 Pod 를 밖으로 밀어냈습니다 — CrashLoopBackOff·OOMKilled
Pod 가 실제로 돌고 있어도 응답은 `candidates: []`·`count: 0` 이었고, 그 응답은 아무 문제도 없는
건강한 클러스터의 응답과 **한 글자도 다르지 않았습니다**. 응답에는 조회가 상한에 닿았다는 사실조차
들어 있지 않았으므로, 운영자가 볼 수 있는 증상은 "원인 후보가 없다" 뿐이었습니다.

v0.9.301 은 조회를 분석 대상 일곱 Kind 로 좁히고, 창이 가득 찼는지를 응답에 적습니다.

### 원인 분석 API · 창을 분석 대상 Kind 에만 씁니다

| 입력 | 수정 전 응답 | v0.9.301 응답 |
|------|-------------|---------------|
| 창을 채운 ConfigMap + CrashLoopBackOff Pod | `candidates: []`, `count: 0` | 해당 Pod 가 `candidates` 에 나타남 |
| 분석 대상이 상한을 초과 | 표시 없음 | `scan.status: "partial"` + `reason` |
| 창이 전부를 담음 | 표시 없음 | `scan.status: "checked"` + `scan.resources` |

- `scan.status` — 창이 전부를 담았으면 `checked`, 상한에 닿았으면 `partial`.
- `scan.resources` — 후보 산출에 실제로 들어간 RCA 관련 리소스 수(상한 이하).
- `scan.reason` — `partial` 일 때만, 창 밖으로 밀려난 워크로드의 장애는 후보에 나타나지 않으므로
  후보가 없다는 것을 정상으로 판단하지 말라는 안내.

`scan` 은 바로 옆 `/admin/k8s/capacity`(v0.9.300)·`/admin/k8s/security` 가 이미 쓰는 키·같은 결
그대로입니다.

### 호환성

UI 가 직접 렌더하는 `candidates`·`count`·`note` 키와 후보 판정 로직은 한 줄도 바뀌지 않았습니다.
조회 예산은 기존 상한 그대로 **1000** 이라 조회량이 늘지 않으며, 같은 핸들러의 Event 500행·spec
리비전 1000행·메트릭 4000행 조회도 그대로입니다. 응답에 키가 늘어나므로 이 JSON 을 소비하는
스크립트는 알려지지 않은 키를 무시해야 합니다. DB 스키마·설정·의존성 변경은 없습니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.301.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.301.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.301.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.301.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.301.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.301
```
