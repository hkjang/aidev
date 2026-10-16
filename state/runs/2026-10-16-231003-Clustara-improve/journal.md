# 회차 노트 2026-10-16-231003-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:10] base pinned — main@afe75fe
- [러너 23:10] autonomy release — 

## 정찰 노트
- RCA 핸들러를 골랐다: `RCARelevantKinds()`(rca.go:52)가 "예산 있는 호출자는 이걸 쓰라"고 주석까지 달고 존재하는데 `handleK8sRCA`(admin_k8s.go:525)만 안 쓴다 — 결함과 수정 모양이 capacity(v0.9.300)·security 선례로 완전히 결정돼 있고 프로덕션 1파일이다.
- 제친 후보: metrics_error(차선, 같은 함수 연속 변경 회피 + `metrics, _ :=` 가 10곳 넘어 범위 경계가 덜 분명), inventory 창 포화(7회 연속 차선·운영자 1순위 금지), home failure_candidates(조회 범위 미확인·M), notify 계열(4회 연속 같은 경로라 냉각).
- 확인한 것: Kind 분기 3곳(rca.go:65, workload.go:20, workload.go:41-46)이 7개 목록과 일치, `K8sInventoryFilter.Kinds` 존재, 기존 rca 테스트 2곳이 struct 디코드라 `scan` 키 추가에 안전, 기준선 `-run 'RCA|Rollout|NodeCondition'` 통과.
- 추측으로 적은 것(미확인): 실제 클러스터에서 ConfigMap/Endpoints 가 1000행을 실제로 채우는 빈도, `/admin/k8s/home` 의 조회 범위, 전체 `go test ./...` 는 이번에 안 돌렸다.
- 구현자 주의: `RCARelevantKinds()` 목록 자체를 넓히지 말 것(다른 과제), events 500·revisions 1000·metrics 4000 과 `lerr == nil` 가드는 그대로, `rcaScanBudget` 은 1000 유지해 조회량을 늘리지 말 것.
- [러너 23:14] scout done — RCA API의 인벤토리 조회를 분석 대상 Kind로 한정하고 창 포화를 보고하기 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 무엇/왜: `handleK8sRCA`(`internal/proxy/admin_k8s.go`)의 인벤토리 조회를 `analyzer.RCARelevantKinds()` 로 좁히고 `Limit: rcaScanBudget + 1`(예산 1000 유지, 조회량 불변)로 받아 초과분을 잘라 `scan{status,resources,reason}` 을 응답에 추가했다. 분석 대상이 아닌 ConfigMap·Secret·Endpoints 가 `updated_at DESC` 창을 채우면 CrashLoopBackOff Pod 가 밀려나 `candidates: []` 가 되고 포화 사실이 응답에 전혀 없던 문제다. 키·문구 결은 `handleK8sCapacity`(v0.9.300)·`handleK8sSecurity` 선례 그대로.
- 확신 없는 곳·검증 못 한 것: (1) 실제 운영 클러스터에서 비분석 Kind 가 1000행 창을 실제로 채우는 빈도는 확인하지 못했다 — 테스트는 `rcaScanBudget` 을 2로 낮춰 같은 구조를 축소 재현한 것이다. (2) `scan` 키 추가가 UI(`admin_ui.go`)에 보이지 않는다는 것은 UI 가 `candidates` 만 읽는다는 코드 확인에 근거하며 브라우저로 띄워보지는 않았다. (3) PostgreSQL 경로는 미확인(테스트는 SQLite). (4) 한국어 `reason` 문구는 내가 새로 쓴 것이고 운영자 검토를 받지 않았다.
- 일부러 하지 않은 것: 같은 핸들러의 `ListK8sEvents(..., 500)`·`ListK8sRevisions(..., 1000)`·`ListK8sMetricSamples(..., 4000)` 와 `lerr == nil` 가드는 그대로 뒀다(별도 보류 아이디어). `RCARelevantKinds()` 목록 자체도 건드리지 않았다 — 7개가 `rca.go:65`·`workload.go:20`·`workload.go:41-46` 분기와 일치함을 읽어 확인했다. `/admin/k8s/connectivity`(`:702`, 여전히 Kinds 없는 `Limit: 2000`)와 `/admin/k8s/home` 의 `failure_candidates` 는 범위 밖(한 회차 엔드포인트 하나).
- 다음 역할이 조심할 것: 신규 `admin_k8s_rca_scope_test.go` 는 실 SQLite 파일 DB(`openTestStore`)와 httptest 서버가 필요하고, `rcaScanBudget` 을 전역으로 바꿔 쓰므로 `t.Parallel()` 을 붙이면 안 된다(`capacityScanBudget` 과 같은 제약). 포화 재현은 `item.UpdatedAt` 의 상대 순서에 의존하니 fixture 시각을 동률로 만들지 말 것.
- [러너 23:19] brief accepted — 채택 — 과제서의 근거(`admin_k8s.go:525` 의 Kind 미필터 `Limit: 1000`, `RCARelevantKinds()` 7개와 `rca.go:65`·`workload.go` 분기의 일치, `r
- [러너 23:21] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인: `RCARelevantKinds()` 7개 = AnalyzeRCA 의 items 분기 3곳(rca.go:65, workload.go:20-31, :44-53) 합집합이고, 핸들러에서 items 소비처는 AnalyzeRCA 하나뿐 — 좁히기로 사라지는 finding 없음. `Kinds` 는 lower(?) 바인드(store/k8s.go:502)로 주입 없음. 인가·비밀값·개인정보 변화 없음(Secret 행을 더 이상 읽지 않아 노출면은 감소). 보안·법무 차단 사유 없음.
- 테스트 유효성: `Kinds` 가 빠지면 budget+1=3 으로 ConfigMap 2행이 창을 채워 CrashLoopBackOff 단언과 `scan.status=="checked"` 단언이 함께 빨강 → 변경 경로를 실제로 지난다. 원장 `실패 재현` 출력도 같은 증상. 재실행 `go test ./internal/proxy -run RCA -count=1` ok 0.460s, verify.json build/vet/test ./... 모두 exit 0. t.Parallel() 사용 테스트 0건이라 전역 `rcaScanBudget` 변조는 안전.
- 못 본 것: 실제 Kubernetes·PostgreSQL·브라우저 UI, 실 클러스터에서 비분석 Kind 가 1000행을 채우는 빈도(테스트는 budget=2 축소 재현).
- 승인이어도 남는 우려 ①: `scan` 은 JSON 에만 있고 `scanNotice`/`scanIncomplete`(admin_ui.go:5642,5664)가 RCA 호출(:10695,:12102)에 미배선 — capacity 와 같은 상태라 거절 사유는 아니지만 세 엔드포인트 공통 UI 부채.
- 우려 ②(릴리즈): `status:"checked"` 는 인벤토리 창만 뜻하는데 events 500·revisions 1000·metrics 4000·`metrics` 오류 무시(admin_k8s.go:563)는 응답에 없어 "완전함" 으로 읽힐 수 있다 — docs/K8S_OPERATIONS_HUB.md 에 capacity 선례 결로 `scan` 키와 이 범위 한계를 기록 권장.
- [러너 23:24] review approved — 리뷰 승인 (risk=low)
- [러너 23:25] pr created — https://github.com/hkjang/clustara/pull/41
- [러너 23:25] ci passed — 검사 없음 — 정책으로 허용
- [러너 23:25] merge done — 20a0165
- [러너 23:33] release published — v0.9.301
- [러너 23:33] gh-release created — GitHub Release v0.9.301
- [러너 23:33] manifest ok — clustara-v0.9.301.tar.gz clustara-v0.9.301.tar.gz.sha256 README-offline-v0.9.301.md 
- [러너 23:34] assets uploaded — 3개
- [러너 23:34] assets verified — v0.9.301 자산 3개 (이전 v0.9.300: 3)
