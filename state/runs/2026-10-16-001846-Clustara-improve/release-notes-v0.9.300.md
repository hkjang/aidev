## 바쁜 클러스터의 용량 리포트가 "아무것도 안 도는 클러스터" 와 똑같이 비어 있었습니다

`GET /admin/k8s/capacity` 는 인벤토리를 **Kind 필터 없이** `updated_at` 내림차순 4000행으로 읽었습니다.
그런데 집계를 담당하는 `AnalyzeCapacity` 가 실제로 읽는 Kind 는 `HorizontalPodAutoscaler`·`Pod`·`Node`
세 개뿐입니다.

분석 대상이 아니면서 자주 갱신되는 ConfigMap·Secret·Event 는 바쁜 클러스터에서 가장 최신 행입니다.
그래서 그 4000행 창을 비분석 Kind 가 채우고 Node·Pod 를 밖으로 밀어냈습니다 — `report.node_packing`·
`report.gpu`·`report.projections` 가 **빈 배열**로 돌아왔고, 그 응답은 노드도 Pod 도 없는 클러스터의
응답과 구별되지 않았습니다. 응답에는 조회가 상한에 닿았다는 사실조차 들어 있지 않았으므로,
운영자가 볼 수 있는 증상은 "용량 데이터가 없다" 뿐이었습니다.

v0.9.300 은 조회를 분석 대상 세 Kind 로 좁히고, 창이 가득 찼는지를 응답에 적습니다.

### 용량 API · 분석 대상 Kind 로만 창을 쓰고, 포화를 보고합니다

| 입력 | 수정 전 응답 | v0.9.300 응답 |
|------|-------------|---------------|
| Node·Pod 보다 최신인 ConfigMap 이 조회 창을 채운 클러스터 | `node_packing: []`, `gpu: []`, `projections: []` (scan 키 없음) | 노드·GPU 행 정상 집계 + `scan.status: "checked"` |
| 용량 관련 리소스가 상한을 초과한 클러스터 | 잘린 집계, 잘렸다는 표시 없음 | 상한까지 집계 + `scan.status: "partial"` 과 `scan.reason` |
| 정말로 아무것도 없는 클러스터 | 빈 배열 | 빈 배열 + `scan.status: "checked"`, `scan.resources: 0` |

조회는 신규 `analyzer.CapacityRelevantKinds()`(HPA·Node·Pod)로 좁히고, 상한보다 한 행 많은
`capacityScanBudget + 1` 로 받아 초과분을 잘라낸 뒤 바로 옆 `/admin/k8s/security` 와 같은 결의
`scan` 블록을 함께 돌려줍니다.

- `scan.status` — 창이 전부를 담았으면 `checked`, 상한에 닿았으면 `partial`
- `scan.resources` — 집계에 실제로 들어간 용량 관련 리소스 수
- `scan.reason` — `partial` 일 때만, 노드 packing·GPU·증가 추세가 실제 수요보다 낮게 보일 수 있으니
  이 결과만으로 증설을 판단하지 말라는 안내

### 유지되는 것

UI 가 직접 렌더하는 `report`·`note` 키와 집계·금액 로직(비용 경로가 공유하는 `podRequestGPU` 포함)은
한 줄도 바뀌지 않았습니다. 기본 상한 4000 과 메트릭 조회 2000행, HPA 진단·과소/과다 할당·노드
packing·GPU·용량 예측의 계산 결과도 그대로입니다. 응답에 키가 늘어나므로 이 JSON 을 소비하는
스크립트는 알려지지 않은 키를 무시해야 합니다. DB 스키마·설정·의존성 변경은 없습니다.

### 검증

신규 테스트 3개(`internal/analyzer/capacity_kinds_test.go` 1개,
`internal/proxy/admin_k8s_capacity_scope_test.go` 2개)를 수정 전 코드에 먼저 붙여 빨강을 확인했습니다 —
실제 SQLite 저장소 + `store.NewAsyncLogger` + `NewServer` + `httptest.NewServer(server.Routes())` +
실제 `kube.InventoryFromObject`, `item.UpdatedAt` 직접 지정으로 시간 동률 없이 창 포화를 재현하고,
`capacityScanBudget` 을 2로 낮춰 수천 행 없이 잘림 경로를 덮습니다(손으로 만든 대역·주입 객체 없음).
릴리즈 검증으로 `go build ./...`, `go vet ./...`, `go test ./...`(테스트가 있는 20개 패키지 전부 통과)와
버전 일치·산출물 이름 게이트를 실행했습니다. 실제 Kubernetes·PostgreSQL·브라우저 렌더링은
미검증입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.300.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.300.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.300.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.300.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.300
```
