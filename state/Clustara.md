# Clustara 자율 개선 기록

## 2026-09-02
- 선택: 스캔/SBOM 정규화기가 버리던 필드 복구 + 첫 단위 테스트 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `internal/analyzer/vulnerability.go`(576줄, 단위 테스트 0개)가 Trivy·Grype·trivy-operator 스캔과 CycloneDX·SPDX SBOM import의 유일한 파싱 경로인데, 아티팩트에 실제로 들어 있는 값 4가지를 못 읽고 있었다 — Grype CVSS(`cvss[].metrics.baseScore`가 두 단계 아래라 항상 0), SBOM `generated_at`(존재하지 않는 최상위 `created`를 보고 `creationInfo` 객체에 문자열 변환을 걸어 두 포맷 모두 빈 값), CycloneDX 1.4 generator(`metadata.tools` 배열 형태 미지원 + 빈 값을 그대로 return해 SPDX fallback 차단), Grype EPSS(객체 배열). 기존 Trivy V3Score 경로와 요청이 준 defaults 우선순위는 그대로 두고 누락분만 추가했으며, 신규 테스트 6개를 **고치기 전 코드에 되돌려 붙여** 4개가 정확히 해당 값을 지목하며 실패하는 것을 확인했다. 검증: `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(17 패키지). 저장소 관례대로 AppVersion·changelog·`docs/K8S_OPERATIONS_HUB.md` 버전 마커를 v0.9.262로 올렸다(release gate 테스트가 강제함).
- 보류 아이디어: ① `collectBenchmarkResults`의 Section 귀속 — 바깥 노드 라벨이 먼저 이겨서 kube-bench 결과의 Section이 "1.1" 대신 상위 control 설명으로 채워짐 (가치 2 / 위험 2 / S) ② Grype의 `Negligible` 심각도가 `Unknown`으로 접혀 Low보다 낮은 등급 정보가 사라짐 (가치 2 / 위험 2 / S) ③ `.github`에 FUNDING.yml만 있고 CI 워크플로가 없음 — build/vet/test 게이트 추가 (가치 3 / 위험 1 / S) ④ `detectScanner`가 아무것도 못 맞히면 `scanner="unknown"`으로 저장하면서 trivy 파서를 돌림 — 저장값과 실제 파서 불일치 (가치 2 / 위험 1 / S) ⑤ `internal/harbor`, `internal/servicecatalog` 테스트 0개 — 커버리지 공백 (가치 3 / 위험 1 / M)
- 릴리즈: v0.9.262 (2026-09-02)

## 2026-09-02
- 선택: Harbor manifest·정책 게이트 결함 수정 + `internal/harbor` 첫 단위 테스트 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `internal/harbor`(460줄, 단위 테스트 0개)는 운영자가 그대로 `kubectl apply` 하는 imagePullSecret/Deployment manifest 와 그 앞의 배포 정책 게이트를 만드는 유일한 경로인데 4가지 결함이 있었다 — ① note 가 하나의 double-quoted 스칼라인데 그 안의 robot 이름·registry 를 `yamlScalar` 로 따로 인용해서, Harbor 가 실제로 발급하는 `robot$project+name` 이름이면 따옴표가 note 안쪽에 박혀 **Secret 문서 전체가 YAML 파싱 실패** ② `expires_at` 을 RFC3339 로만 파싱하고 실패 시 조용히 무시 → 날짜·unix seconds(Harbor v2 robot API 의 실제 반환형)·`-1` 이 전부 "만료 없음"으로 읽혀 allow 통과 (이제 해석하고, 그래도 못 읽으면 `robot_expiry_unreadable` 승인 필요) ③ `DefaultSecretName("")` → `harbor--pull` (가드를 이어붙인 문자열에 걸어둬 미동작) ④ `RegistryHost("  ")` 이 공백을 host 로 반환. 검증: 신규 테스트 8개(생성 manifest 를 gopkg.in/yaml.v3 로 실제 디코드)를 고치기 전 코드에 되돌려 붙여 4개가 각 결함을 지목하며 실패함을 확인했고, `go build ./...`·`go vet ./...`·`go test ./...` 전부 통과. 저장소 관례대로 AppVersion·changelog·docs 버전 마커를 v0.9.263 으로 올렸다(release gate 테스트가 강제).
- 보류 아이디어: ① `.github`에 CI 워크플로 없음 — build/vet/test 게이트 추가 (가치 3 / 위험 1 / S) ② `detectScanner` 가 못 맞히면 `scanner="unknown"` 저장 후 trivy 파서 실행 — 저장값과 실제 파서 불일치 (가치 2 / 위험 1 / S) ③ Grype `Negligible` 심각도가 `Unknown` 으로 접혀 Low 미만 등급 정보 소실 (가치 2 / 위험 2 / S) ④ `collectBenchmarkResults` 의 Section 귀속 — 바깥 노드 라벨이 먼저 이겨 kube-bench Section 이 "1.1" 대신 상위 control 설명으로 채워짐 (가치 2 / 위험 2 / S) ⑤ `internal/servicecatalog` 테스트 0개 — 커버리지 공백 (가치 3 / 위험 1 / M)
- 릴리즈: v0.9.263 (2026-09-02)
- 릴리즈: v0.9.263 (2026-09-02)

## 2026-09-03
- 선택: 셀프서비스 카탈로그 검증 게이트의 구멍 4개 수정 + `internal/servicecatalog` 첫 단위 테스트 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `internal/servicecatalog`(158줄, 단위 테스트 0개)의 `ValidateInput` 은 셀프서비스 요청 본문의 `values`(전부 사용자 입력)가 운영자가 승인·apply 하는 manifest 로 바뀌는 사이의 유일한 게이트인데 4가지가 그냥 통과했다 — ① `port` 미검증(그대로 `containerPort:` 에 렌더 → 0·-1·70000 이 검증·정책 미리보기·승인을 다 통과하고 apply 시점에만 거절) ② 태그 없는 이미지가 `Contains(":latest")` 가드 우회(k8s 는 무태그를 latest 로 해석), 게다가 호출부 운영 digest 게이트도 `Contains(image, ":")` 전제 때문에 콜론 없는 이미지는 digest 요구 자체를 건너뜀 ③ 같은 substring 검사가 `tomcat:latest-jdk21` 같은 고정 태그와 digest 로 고정된 `repo:latest@sha256:...` 을 오탐 거절 ④ `0`·`0Gi` 가 유효 수량(storage 0 은 PVC 검증기가 거절, memory limit 0 은 첫 할당에서 OOM). 참조를 `[registry[:port]/]repo[:tag][@digest]` 로 분해하는 방식으로 바꾸고 port 범위·0 초과 수량을 추가했으며, 호출부(`admin_k8s_services.go`)의 운영 digest 게이트에서 콜론 전제를 제거했다. 검증: 신규 테스트 7개를 고치기 전 코드에 되돌려 붙여 4개가 각 결함을 지목하며 실패함을 확인했고 `go build ./...`·`go vet ./...`·`go test ./...` 전부 통과(21 패키지). 저장소 관례대로 AppVersion·changelog·docs 버전 마커를 v0.9.264 로 올렸다(release gate 테스트가 강제).
- 보류 아이디어: ① `.github` 에 CI 워크플로 없음 — build/vet/test 게이트 추가 (가치 3 / 위험 1 / S) ② `detectScanner` 가 못 맞히면 `scanner="unknown"` 저장 후 trivy 파서 실행 — 저장값과 실제 파서 불일치 (가치 2 / 위험 1 / S) ③ Grype `Negligible` 심각도가 `Unknown` 으로 접혀 Low 미만 등급 정보 소실 (가치 2 / 위험 2 / S) ④ `collectBenchmarkResults` 의 Section 귀속 — 바깥 노드 라벨이 먼저 이겨 kube-bench Section 이 "1.1" 대신 상위 control 설명으로 채워짐 (가치 2 / 위험 2 / S) ⑤ 카탈로그 caller 의 `num()` 헬퍼가 문자열 `"0"`·비숫자를 조용히 기본값으로 되돌림 — 잘못된 입력이 기본값으로 성공 (가치 2 / 위험 1 / S)
- 릴리즈: v0.9.264 (2026-09-03)
- 릴리즈: v0.9.264 (2026-09-03)
