## 용량 리포트의 메트릭 조회가 실패해도 "아직 수집 전인 정상 클러스터" 와 똑같이 비어 있었습니다

`GET /admin/k8s/capacity` 는 메트릭 표본 조회의 오류를 그냥 버렸습니다 — `metrics, _ := s.db.ListK8sMetricSamples(...)`.

그런데 할당 효율(SCALE-03/04 과소·과다 할당)과 노드 소진 예측(SCALE-05)은 **이 표본만으로**
계산됩니다. 조회가 실패하면 `report.allocation` 과 `report.projections` 는 빈 배열이 되고,
그 응답은 수집기가 아직 보고하지 않은 정상 클러스터의 응답과 **한 글자도 다르지 않았습니다** —
두 경우 모두 `scan` 이 `{status: "checked", resources: 2}` 였습니다.

UI 는 그 빈 배열을 `과소 할당 0 · 과다 할당 0 · 30일 내 소진 0` 과
"즉시 증설이 필요한 신호는 없습니다" 배너로 렌더했습니다. 저장소 고장이 안심 신호로 바뀐 것입니다.

v0.9.302 는 그 조회 실패를 v0.9.300 이 만든 `scan` 블록에 실어 보고합니다.

### 용량 API · 메트릭 조회 실패를 '표본 없음' 과 구별해 보고합니다

| 상황 | 수정 전 `scan` | v0.9.302 `scan` |
|------|----------------|-----------------|
| 정상, 표본 아직 없음 | `{status: checked, resources: 2}` | `{status: checked, resources: 2}` |
| 메트릭 조회 실패 | `{status: checked, resources: 2}` | `{status: partial, resources: 2, metrics_error: …, reason: …}` |
| 창 포화 + 메트릭 실패 | 포화 안내만 | `status: partial`, 두 `reason` 이 이어붙음 |

- `scan.status` — 메트릭 조회가 실패하면 `partial`.
- `scan.metrics_error` — 실패한 조회의 오류 메시지. `notify` 창의 `clusters_error`·`events_error`
  와 같은 결입니다.
- `scan.reason` — 두 표가 비어 있는 것은 '이상 없음' 이 아니므로 DB 상태를 확인하라는 안내.
  창 포화와 메트릭 실패는 서로 다른 손실이라 두 안내가 이어붙어, 하나가 다른 하나를 가리지 않습니다.

요청은 그대로 **200** 입니다. 인벤토리에서 계산하는 HPA 현황·노드 packing·GPU 는 영향이 없으므로,
요청을 깨는 대신 무엇이 빠졌는지를 적어 축소 보고합니다.

### 용량 화면이 그 경고를 실제로 보여줍니다

용량 화면은 이제 이 `scan` 블록을 다른 리포트와 같은 공유 `scanNotice` 경고로 렌더합니다.
v0.9.300 이 추가한 창 포화 보고도 이로써 비로소 운영자 화면에 닿습니다. 집계가 전수가 아닐 때
요약 배너는 더 이상 "증설 불필요" 로 결론 내리지 않습니다.

### 바뀌지 않은 것

`report`·`note` 키와 집계 로직, 메트릭 조회 행수(2000)와 조회 예산은 그대로입니다. 기존
`truncated` 분기의 키와 문구는 상수로만 뽑았을 뿐 값이 같습니다. 응답에 키가 늘어나므로 이 JSON 을
소비하는 스크립트는 알려지지 않은 키를 무시해야 합니다. DB 스키마·설정·의존성 변경은 없습니다.

### 검증

신규 테스트 2개(`internal/proxy/admin_k8s_capacity_metrics_test.go`)를 수정 전 코드에 먼저 붙여
빨강을 확인했습니다. 실제 SQLite `store.Open`+`Migrate`, `store.NewAsyncLogger`, `NewServer`,
`httptest.NewServer(server.Routes())`, 실제 `kube.InventoryFromObject` 로 프로덕션 배선을 지나며,
같은 dsn 의 두 번째 커넥션에서 `DROP TABLE k8s_metrics_samples` 로 오류를 주입해 인벤토리 테이블은
멀쩡한 부분 고장을 그대로 재현합니다(손으로 만든 대역·주입 객체 없음). UI 는 서버가 내보내는
`adminHTML` 을 꺼내 node `vm` 으로 `scanNotice`·`scanIncomplete` 를 실제 `scan` 페이로드로 실행해
확인했습니다.

릴리즈 게이트: `gofmt -l`(변경 파일 해당 없음) → `go build ./...` → `go vet ./...` →
`go test ./... -count=1`(테스트가 있는 20개 패키지 전부 통과, 실패 0). 버전 일치
(`AppVersion` ↔ `scripts/changelog.txt` ↔ `docs/K8S_OPERATIONS_HUB.md`)와 산출물 이름 게이트를
포함합니다. 실제 Kubernetes·PostgreSQL·브라우저 렌더링은 미검증입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.302.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.302.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.302.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.302.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.302
```
