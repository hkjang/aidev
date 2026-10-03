# 회차 노트 2026-10-04-070314-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:03] base pinned — main@d326142
- [러너 07:03] autonomy release — 

## 정찰 노트
- 보류 1순위였던 "전 클러스터 공유 예산" 을 골랐다. 계약 불일치(L, 호출자 36곳)는 6파일 규칙 위반, PSS volume 화이트리스트는 2026-09-28·10-03·7fedbb3 이 연속으로 포스처 등급을 움직인 직후라 한 회차 쉬어야 하고, dedup 키 Kind(가치 2)는 5회 연속 밀린 항목이라 제쳤다.
- 코드에서 확인한 것: k8s_notify.go:195-237 의 단일 창 3회 조회, store/k8s.go:276 ListK8sClusters·242 UpsertK8sCluster 존재, 기존 notify 테스트 6파일에 UpsertK8sCluster 0건(= 빈 k8s_clusters 폴백이면 무수정 통과), ListK8sEvents 비테스트 호출자 36곳, docs/K8S_OPERATIONS_HUB.md 의 "### notify scan ·" 섹션 선례.
- 추측으로 적은 것(미확인): 팬아웃 뒤 쿼리 수 3×N 이 실환경 함대 규모에서 괜찮다는 가정 — 등록 클러스터 수의 실제 분포는 확인하지 못했다. 또한 분석기가 이어 붙인 다중 클러스터 입력에 그대로 돈다는 것은 v0.9.285~0.9.287 기록에 기댄 판단이며 이번에 실행으로 확인하지는 않았다.
- 구현자가 조심할 것: 스토어 상한(500/1000)과 notifyScan*Budget 값을 올리지 말 것, 클러스터 수에 조용한 상한을 두지 말 것(지금 고치는 결함과 같은 모양), analyzer 와 dedup 키는 손대지 말 것.
- 이 정찰에서 실행한 명령: go test ./internal/proxy -run 'NotifyScan|K8sNotify' -count=1 → ok clustara/internal/proxy 2.469s. 코드 변경 없음, 작업 트리 clean.
- [러너 07:07] scout done — notify scan 의 전 클러스터 실행이 조회 예산을 모든 클러스터에 공유해 조용한 클러스터가 한 줄도 평가되�

## 구현 노트
- 무엇/왜: `handleK8sNotifyScan` 이 `cluster_id` 없는 함대 스캔에서 인벤토리·이벤트·리비전 창을 전 클러스터가 공유하던 것을, 신규 `notifyScanTargets` 로 대상을 정해 클러스터마다 예산을 한 번씩 쓰도록 팬아웃했다. 응답·감사에 `clusters_scanned`·`clusters_truncated` 추가. 프로덕션 1파일(`internal/proxy/k8s_notify.go`) + docs 1 + 신규 테스트 1.
- 확신 없는 곳: ① 팬아웃이 `3 × 등록 클러스터 수` 개 쿼리가 되는데 실환경 등록 클러스터 수 분포는 확인하지 못했다(정찰의 "수십 대" 가정에 기댐). 상한은 일부러 두지 않았다. ② 조용한 시간 판정이 조회 **뒤**에 오므로 quiet 창 안에서도 3×N 쿼리를 돌고 버린다 — 고칠 수 있었지만 500/suppressed 우선순위가 바뀌어 이번 인과를 흐리므로 ideas.json 에 분리해 남겼다. ③ 분석기(`AnalyzeRCA`/`AnalyzeSecurity`)가 이어 붙인 다중 클러스터 입력에 정상 동작하는 것은 신규 테스트가 클러스터 2개로 실제 확인했다(추측 아님). ④ SQLite·httptest 통과이며 실 PostgreSQL·실 Kubernetes 검증은 없다.
- 일부러 안 한 것: 스토어 상한(500/1000)·`notifyScan*Budget` 값 미변경, 클러스터 수 상한 미도입, dedup 키·analyzer·`classifyPodSecurity` 미변경, 기존 notify 테스트 6파일 무수정(폴백 경로가 보존됨을 그것으로 증명).
- 다음 역할이 조심할 것: 신규 `k8s_notify_fanout_test.go` 는 `notifyScanBudget` 을 1 로 낮추고 `t.Cleanup` 으로 복원하므로 같은 패키지 병렬 실행과 섞으면 안 된다(기존 테스트와 같은 관례, `t.Parallel()` 미사용). 폴백 테스트는 `k8s_clusters` 가 비어 있다는 전제에 기대므로 `newNotifyScanServer` 에 클러스터 등록을 추가하면 깨진다. 검증은 `go test ./... -count=1` 전부 ok(proxy 97.9s), `gofmt -l` 깨끗, 인과 확인(`targets` 단일화 → 빨강 → 복원) 완료.
- [러너 07:15] brief accepted — 채택 — 과제서의 근거(`k8s_notify.go:195-237` 의 단일 창 3회 조회, `store/k8s.go:276 ListK8sClusters`, 기존 notify 테스트 6파일에 `Upser
- [러너 07:16] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인한 것: 인과를 직접 재현했다 — main 의 `k8s_notify.go` 를 HEAD 위에 덮으면 신규 테스트가 `sent:1 resources:1 truncated:true` 로 실패한다(조용한 클러스터의 privileged 파드가 평가되지 않는 바로 그 증상). 복원 후 트리 clean. `go build`·`go vet ./internal/proxy`·`gofmt -l`·`go test ./... -count=1` 전부 통과. 범위는 3파일로 깨끗하고 revert 잔여물은 6h 만에 만료되는 dedup 행뿐.
- 보안·법무 모두 비차단: 신규 경로·인증 변경 없음, 대상 ID 는 요청 입력이 아니라 `k8s_clusters` 레지스트리에서 오고, 두 ingest 경로가 미등록 클러스터를 이미 404 하며 `DeleteK8sCluster` 가 인벤토리·이벤트·리비전을 cascade 한다. 범위도 넓어지지 않았다(이전 함대 스캔도 필터 없이 전 클러스터를 읽었다). 새로 수집·저장·전송하는 개인정보 없음.
- 승인이어도 남는 우려: ① 이 변경이 이 엔드포인트의 유일한 입력 상한을 없앤다 — `analyzer/rca.go:70-71 workloadRelatedEvents` 가 items×events 라 함대 규모에 **O(N²)** 가 된다(클러스터 50대 ≈ 2.5e8 부분문자열 스캔, Spec 블롭 2000×N 동시 상주). WriteTimeout 10분 여유가 있어 파손은 아니지만 여유를 쓰는 것이다. 루프 안에서 클러스터별로 분석·전달하면 두 성질을 다 지킬 수 있다(분석기는 교차 클러스터 상관을 하지 않는다) — 저자 판단에 맡김. ② quiet hours 가 조회 **뒤**라 조용한 창에서도 3×N 쿼리를 버린다(이전 1회). ③ `ListK8sClusters` 실패가 응답에 아무 표시 없이 공유 창으로 조용히 강등된다 — 이번에 고친 결함과 같은 모양.
- 못 본 것: 실 PostgreSQL·실 Kubernetes·실제 등록 클러스터 수 분포. 위 규모 수치는 예산값 산술이고 측정이 아니다.
- 릴리즈 노트에 넣을 것: 배포 후 첫 함대 스캔은 그동안 한 번도 평가되지 않던 클러스터를 전부 평가하므로 묵은 privileged 워크로드·과도한 Role 이 한꺼번에 알림으로 터진다(각자 새 6h dedup 창). 고장이 아니라 수정이 동작하는 모습이지만 미리 알리지 않으면 회귀로 읽힌다.
- [러너 07:22] review approved — 리뷰 승인 (risk=medium)
- [러너 07:22] pr created — https://github.com/hkjang/clustara/pull/35
- [러너 07:23] ci passed — 검사 없음 — 정책으로 허용
- [러너 07:23] merge done — e875735
- [러너 07:30] release published — v0.9.295
- [러너 07:30] gh-release created — GitHub Release v0.9.295
- [러너 07:30] manifest ok — clustara-v0.9.295.tar.gz clustara-v0.9.295.tar.gz.sha256 README-offline-v0.9.295.md 
- [러너 07:30] assets uploaded — 3개
- [러너 07:30] assets verified — v0.9.295 자산 3개 (이전 v0.9.294: 3)
