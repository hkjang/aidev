## 2026-10-09
- 선택: 용량 API에서 AMD·Intel GPU 노드와 요청 수량 누락 수정 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: NVIDIA만 읽던 nodePackingAndGPU 내부에서 nvidia.com/gpu·amd.com/gpu·intel.com/gpu의 노드 allocatable과 Pod requests를 합산하여 AMD·Intel 행 누락을 고쳤고, 비용 공유 podRequestGPU·CPU packing·음수 Idle 계약은 보존했다. 실제 SQLite SQLStore→kube.InventoryFromObject→Server.Routes→HTTP JSON 테스트(6범위) 및 단위 15케이스로 공급자 수량·혼합 합계·문자열/JSON 숫자·미지원 키·빈 GPU·클러스터/노드 격리·정렬·미배치/미등록 노드 제외를 확인하고 같은 fixture의 노드 모니터링 allocatable/requested와 대조했으며, NVIDIA-only 키 목록으로 되돌리면 같은 테스트가 재실패함을 확인한 뒤 복원했다. 지정 좁은 go test(analyzer 0.008s, proxy 1.791s), gofmt -l(출력 없음), go build ./..., 서버 단독 go build -o <회차 경로>/clustara-build-check ./cmd/clustara, go vet ./..., go test ./... -count=1(테스트 있는 19패키지 통과, proxy 75.762s, store 19.815s; cmd/clustara-agent는 기존 테스트 없음)을 통과했고 산출물 제거 후 프로덕션 1파일·테스트 2파일·문서 1파일을 08cfc7f로 커밋했다.
- 실패 재현: `--- FAIL: TestK8sCapacityGPUProvidersThroughRoutes/amd (0.00s)` / `admin_k8s_capacity_gpu_test.go:113: capacity GPU rows = [], want [{ClusterID:amd Node:worker-1 Allocatable:4 Requested:1 Idle:3}]` (같은 실행에서 Intel도 빈 배열로 실패)
- 보류 아이디어: quiet 경로의 감사 기록 누락 (가치 2 / 위험 2 / 작업량 S) — 최근 알림 경로 연속 변경을 피하고 별도 회차로 유지.
- 보류 아이디어: /admin/k8s/inventory 실효 조회 상한·창 포화 미표시 (가치 2 / 위험 1 / 작업량 S) — 이번 용량 합산과 별개 엔드포인트.
- 보류 아이디어: 용량 API 메트릭 조회 오류를 정상 무표본 응답과 구별 (가치 3 / 위험 2 / 작업량 S) — 현 핸들러의 오류 폐기를 재확인, 별도 HTTP 오류 재현 필요.
- 보류 아이디어: 용량 API 인벤토리 조회를 분석 대상 Kind로 한정 (가치 3 / 위험 2 / 작업량 M) — Kinds 없는 Limit4000 조회 유지, 포화 재현은 별도 과제.
- 과제서: 채택 — NVIDIA 전용 용량 집계와 세 공급자 노드 모니터링의 차이가 현재 코드 및 수정 전 실제 HTTP 실패로 확인되어 지정한 4파일 범위로 구현했다.
