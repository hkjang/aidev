## notify scan 이 분석하지 않는 kind 로 조회 창을 낭비해 알림이 조용히 사라졌습니다

`POST /admin/k8s/notify/scan` 은 인벤토리를 kind 제한 없이 `updated_at` 역순 2000행만 읽었습니다.
분석이 보지 않는 kind 가 자주 갱신되는 클러스터에서는 그 행들이 창을 채우고 privileged
워크로드·과도한 Role 이 창 밖으로 밀려났습니다. 이번 릴리즈는 조회 범위를 분석이 실제로 읽는
kind 로 좁히고, 상한을 넘겨 다 보지 못한 스캔을 보고합니다.

### notify scan · 분석이 읽는 kind 만 조회합니다

컨트롤러가 자주 고쳐 쓰는 ConfigMap 처럼 분석이 보지 않는 kind 가 창을 채우면, privileged
워크로드나 과도한 Role 이 2000행 밖으로 밀려나 알림이 나가지 않았습니다. 같은
`AnalyzeSecurity` 를 돌리는 `/admin/k8s/security` 는 v0.9.282 에 `SecurityRelevantKinds()` 로 이미
좁혀졌는데 알림 경로만 남아 있었고, 이쪽은 사람이 결과를 보지 않으므로 누락이 "알림이 그냥
오지 않는다" 로만 나타났습니다.

`AnalyzeRCA` 가 읽는 kind 를 `analyzer.RCARelevantKinds()`(`AnalyzeRCA`·`analyzeRolloutAndJobs`·
`analyzeNodeConditions` 의 kind switch 를 그대로 옮긴 Pod·Deployment·StatefulSet·DaemonSet·Job·
CronJob·Node)로 노출해, `SecurityRelevantKinds()` 와의 합집합만 조회합니다.

### notify scan · 상한을 초과한 스캔을 보고합니다

상한보다 한 행 더 요청해 잘림을 감지하고, 잘린 스캔은 응답과 감사 로그에 `truncated` 와 평가한
행 수 `resources` 를 적고 응답에 `truncation_notice` 를 덧붙입니다. `sent: 0` 이 "이상 없음"인지
"다 보지 못함"인지 구분되지 않던 문제입니다. `quiet_hours` 로 조기 반환하는 응답에는 보고할 스캔
결과가 없으므로 `truncated` 를 넣지 않습니다.

### 업그레이드 시 참고

조회 범위를 좁혔으므로, 그동안 창 밖으로 밀려나 조용하던 finding 이 이번 릴리즈의 첫 스캔에서
한꺼번에 나갈 수 있습니다 — 의도된 동작입니다. 6시간 중복 제거 윈도우·조용한 시간·담당팀 채널
라우팅·딥링크는 그대로이며, DB 스키마·설정 변경은 없습니다. `ListK8sEvents`(500행)·
`ListK8sRevisions`(1000행) 의 무보고 잘림은 이번 범위 밖입니다.

### 검증

신규 테스트 4개(kind 커버리지 단위 1 + 실제 SQLite·`kube.InventoryFromObject`·`Server.Routes`·
httptest Mattermost webhook 을 쓰는 종단 3개)를 수정 전 코드에 먼저 붙여
`TestNotifyScanDoesNotLoseWorkloadsToChurningKinds` 가 `sent: 0` 으로,
`TestNotifyScanReportsTruncation` 이 `truncated` 키 없이 실패하는 것을 확인했습니다. 이후
`go build ./...`, `go vet ./...`, `go test ./...`(전 패키지 ok)와 버전 일치·산출물 이름 릴리즈
게이트를 통과했습니다. 실 Kubernetes·PostgreSQL·실 Mattermost·브라우저 렌더링은 미검증입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.290.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.290.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.290.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.290.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.290.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.290
```
