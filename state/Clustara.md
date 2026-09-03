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

## 2026-09-03
- 선택: 액션 영향도 산출기의 승인 게이트 결함 4개 수정 + `internal/action` 테스트 보강 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `internal/action/impact.go`(272줄, 테스트 4개)는 클러스터를 바꾸는 모든 액션이 승인 전에 통과하는 유일한 영향도 산출기이고 그 결과가 `DryRunDiff`로 저장돼 운영자는 그 텍스트만 보고 승인한다. 넷이 잘못돼 있었다 — ① preview 의 `num()` 은 float/int 만 읽는데 실행기의 `intFromParams` 는 JSON 문자열도 `Atoi` 로 파싱해서, `{"replicas":"5"}` 요청이 승인기록·감사로그에 `replicas 2 → 0 (-2)` 로 남고 실제로는 5로 스케일됐다(승인한 값 ≠ 실행된 값) ② `replicas` 부재·음수도 똑같이 "0으로 축소"로 렌더 — 실행기가 거절할 요청에 전면 중단 diff 가 남았다 ③ replica 0 축소에 승인 게이트가 없음: `scale` 은 `Classify` 가 유일하게 `RequiresApproval:false` 로 두는 액션인데 0 은 워크로드 전면 중단이고, 서비스 플랫폼 stop 경로는 같은 요청을 이미 `approval_required` 로 기록 중이라 `POST /admin/k8s/actions` 만 `pending` 으로 통과시키고 있었다 ④ `cordon`/`uncordon` 이 한 `case` 를 공유해 uncordon 승인자가 "cordon은 신규 스케줄만 차단합니다" 라는 반대 동작 설명을 읽었다. 덤으로 patch 미허용 필드 목록이 map 순회 순서라 같은 요청이 매번 다른 승인 문구를 만들던 것도 정렬했다. 검증: 신규 테스트 5개를 고치기 전 코드에 되돌려 붙여 다섯 개가 각 결함을 지목하며 실패함을 확인했고 `go build ./...`·`go vet ./...`·`go test ./...` 전부 통과(20 패키지). 저장소 관례대로 AppVersion·changelog·docs 버전 마커를 v0.9.265 로 올렸다(release gate 테스트가 강제).
- 보류 아이디어: ① `.github` 에 CI 워크플로 없음 — build/vet/test 게이트 추가 (가치 3 / 위험 1 / S) ② `detectScanner` 가 못 맞히면 `scanner="unknown"` 저장 후 trivy 파서 실행 — 저장값과 실제 파서 불일치 (가치 2 / 위험 1 / S) ③ Grype `Negligible` 심각도가 `Unknown` 으로 접혀 Low 미만 등급 정보 소실 (가치 2 / 위험 2 / S) ④ `collectBenchmarkResults` 의 Section 귀속 — 바깥 노드 라벨이 먼저 이겨 kube-bench Section 이 "1.1" 대신 상위 control 설명으로 채워짐 (가치 2 / 위험 2 / S) ⑤ `AssessImpact` 가 인벤토리에 없는 대상을 zero value 로 받아 "replicas 0 → N" 처럼 현재 상태를 아는 척함 — 미관측 대상임을 표시해야 함 (가치 3 / 위험 1 / S)
- 릴리즈: v0.9.265 (2026-09-03)
- 릴리즈: v0.9.265 (2026-09-03)

## 2026-09-03
- 선택: kubectl 의 last-applied 주석이 Secret 값을 저장·응답으로 실어나르던 경로 차단 + 마스킹 3곳 정합 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: 수집기(`internal/kube/client.go`)는 Secret 의 `data` 를 절대 저장하지 않고 `tls.key` 조차 버리는데, `metadata.annotations` 는 그대로 저장돼 `kubectl apply` 가 남긴 `kubectl.kubernetes.io/last-applied-configuration`(적용 객체 전체 = Secret 의 base64 `data`, 워크로드의 모든 env 값)이 그 정책을 우회하고 있었다. 그 주석은 `GET /admin/k8s/inventory` 가 가공 없이 돌려주고 Manifest Viewer 는 `"masked": true` 라고 말하면서 함께 내보냈다. 수집 경로 셋(실시간·에이전트 push·스냅샷 import)이 전부 `UpsertK8sInventory` 로 모이므로 저장 시점에 떨어내고, 기존 행 때문에 읽기 시점(`scanK8sInventory`)에서도 떨어낸다. 같은 테마로 둘 더 고쳤다 — Manifest Viewer 의 `maskStringMap` 이 주석을 키 이름으로만 판단해 값(토큰 붙은 webhook URL·DSN)을 그대로 복사하던 것을 Pod 지문 경로와 동일하게 `analyzer.MaskSensitive` 로 맞췄고, `DetectStackFieldDrift` 가 `DB_PASSWORD=...` 를 선언/실제 양쪽 평문으로 관리자 UI 에 렌더하던 것을 비교는 원본·출력은 마스킹으로 바꿨다(드리프트 검출은 그대로, 이름은 남김). 검증: 신규 테스트 5개를 고치기 전 코드에 되돌려 붙여 넷이 각 결함을 지목하며 실패함을 확인했고 `go build ./...`·`go vet ./...`·`go test ./...` 전부 통과(21 패키지). 저장소 관례대로 AppVersion·changelog·docs 버전 마커를 v0.9.266 으로 올렸다(release gate 테스트가 강제).
- 보류 아이디어: ① `.github` 에 CI 워크플로 없음 — build/vet/test 게이트 추가 (가치 3 / 위험 1 / S) ② `analyzer.MaskSensitive`(Pod 로그 마스킹)가 `Authorization: Basic <base64>` 를 못 잡음 — audit redactor 는 잡는데 로그 경로만 구멍 (가치 3 / 위험 1 / S) ③ `isSensitivePath` 가 substring 매칭이라 `imagePullSecrets`·`volumes[].secret.secretName` 같은 참조 이름까지 `***` 로 덮어 manifest 원장 diff 에 잡음이 생김 (가치 2 / 위험 2 / S) ④ `detectScanner` 가 못 맞히면 `scanner="unknown"` 저장 후 trivy 파서 실행 — 저장값과 실제 파서 불일치 (가치 2 / 위험 1 / S) ⑤ `AssessImpact` 가 인벤토리에 없는 대상을 zero value 로 받아 "replicas 0 → N" 처럼 현재 상태를 아는 척함 (가치 3 / 위험 1 / S)
- 릴리즈: v0.9.266 (2026-09-03)
- 릴리즈: v0.9.266 (2026-09-03)

## 2026-09-03
- 선택: Pod 로그 마스크(`analyzer.MaskSensitive`)가 놓친 자격증명 8가지 수정 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `MaskSensitive` 는 Pod 로그·exec stdout/stderr·터미널 스트림·Pod env 지문·Manifest Viewer 주석 값이 나가기 전 통과하는 유일한 마스크인데, 바로 앞 틱(v0.9.266)이 주석 값을 이 함수로 넘기며 근거로 적은 "토큰 붙은 webhook URL·DSN·bearer 헤더" 중 둘을 실제로는 못 잡고 있었다. 실제 유입 형태를 하나씩 넣어 8가지를 확인했다 — ① DSN 비밀번호(`postgres://u:pw@h`) 무마스킹(key=value 규칙은 `DATABASE_URL` 을 모름; 이제 비밀번호만 가리고 host/db 는 남김) ② `Authorization: Basic` 규칙 부재(audit 리댁터는 처음부터 잡고 있었음 — 로그 경로만 구멍) ③ JSON 으로 덤프된 헤더는 이름과 콜론 사이 닫는 따옴표 때문에 미매치(audit 이 자기 쪽에서 이미 고친 것과 같은 처리) ④ Bearer 토큰 문자집합에 `+/=` 가 없어 base64 토큰을 앞부분만 가림 — `Bearer ***REDACTED***+Z/gh==` 로 "가렸다고 표시하면서" 뒷부분 노출 ⑤ `SECRET_KEY=`·`PRIVATE_KEY=` 통과(구분자가 키 이름 바로 뒤여야 하므로 `secret` 항목이 못 덮음) ⑥ 키 이름 없이 값만 찍힌 토큰(`sk-`·`gh[pousr]_`·`xox[abprs]-`·`glpat-`·`AIza`) 통과 — 기존엔 `AKIA` 만 이 형태를 다룸 ⑦ PEM 개인키 블록 통과. 표적 규칙 원칙은 유지(넓은 base64 휴리스틱 없음)하고 정상 로그가 한 글자도 안 바뀌는 것을 회귀 테스트로 고정했으며, 인덱스 `switch` 로 치환을 고르던 구조를 audit 과 같은 `{정규식, 치환}` 표로 바꿨다(가운데 규칙 삽입 시 뒤쪽 치환이 어긋나던 구조). 검증: 신규 테스트 10개(+오탐 회귀 1개)를 고치기 전 코드에 되돌려 붙여 열 개가 각 결함을 지목하며 실패함을 확인했고 `go build ./...`·`go vet ./...`·`go test ./...` 전부 통과(21 패키지). AppVersion·changelog·docs 버전 마커를 v0.9.267 로 올렸다.
- 보류 아이디어: ① `.github` 에 CI 워크플로 없음 — build/vet/test 게이트 추가 (가치 3 / 위험 1 / S) ② `AssessImpact` 가 인벤토리에 없는 대상을 zero value 로 받아 "replicas 0 → N" 처럼 현재 상태를 아는 척함 (가치 3 / 위험 1 / S) ③ `isSensitivePath` 가 substring 매칭이라 `imagePullSecrets`·`volumes[].secret.secretName` 같은 참조 이름까지 `***` 로 덮어 manifest 원장 diff 에 잡음 (가치 2 / 위험 2 / S) ④ `detectScanner` 가 못 맞히면 `scanner="unknown"` 저장 후 trivy 파서 실행 — 저장값과 실제 파서 불일치 (가치 2 / 위험 1 / S) ⑤ Grype `Negligible` 심각도가 `Unknown` 으로 접혀 Low 미만 등급 정보 소실 (가치 2 / 위험 2 / S)
- 릴리즈: v0.9.267 (2026-09-03)
- 릴리즈: v0.9.267 (2026-09-03)
