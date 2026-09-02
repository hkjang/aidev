# Clustara 자율 개선 기록

## 2026-09-02
- 선택: 스캔/SBOM 정규화기가 버리던 필드 복구 + 첫 단위 테스트 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `internal/analyzer/vulnerability.go`(576줄, 단위 테스트 0개)가 Trivy·Grype·trivy-operator 스캔과 CycloneDX·SPDX SBOM import의 유일한 파싱 경로인데, 아티팩트에 실제로 들어 있는 값 4가지를 못 읽고 있었다 — Grype CVSS(`cvss[].metrics.baseScore`가 두 단계 아래라 항상 0), SBOM `generated_at`(존재하지 않는 최상위 `created`를 보고 `creationInfo` 객체에 문자열 변환을 걸어 두 포맷 모두 빈 값), CycloneDX 1.4 generator(`metadata.tools` 배열 형태 미지원 + 빈 값을 그대로 return해 SPDX fallback 차단), Grype EPSS(객체 배열). 기존 Trivy V3Score 경로와 요청이 준 defaults 우선순위는 그대로 두고 누락분만 추가했으며, 신규 테스트 6개를 **고치기 전 코드에 되돌려 붙여** 4개가 정확히 해당 값을 지목하며 실패하는 것을 확인했다. 검증: `go build ./...`, `go vet ./...`, `go test ./...` 전부 통과(17 패키지). 저장소 관례대로 AppVersion·changelog·`docs/K8S_OPERATIONS_HUB.md` 버전 마커를 v0.9.262로 올렸다(release gate 테스트가 강제함).
- 보류 아이디어: ① `collectBenchmarkResults`의 Section 귀속 — 바깥 노드 라벨이 먼저 이겨서 kube-bench 결과의 Section이 "1.1" 대신 상위 control 설명으로 채워짐 (가치 2 / 위험 2 / S) ② Grype의 `Negligible` 심각도가 `Unknown`으로 접혀 Low보다 낮은 등급 정보가 사라짐 (가치 2 / 위험 2 / S) ③ `.github`에 FUNDING.yml만 있고 CI 워크플로가 없음 — build/vet/test 게이트 추가 (가치 3 / 위험 1 / S) ④ `detectScanner`가 아무것도 못 맞히면 `scanner="unknown"`으로 저장하면서 trivy 파서를 돌림 — 저장값과 실제 파서 불일치 (가치 2 / 위험 1 / S) ⑤ `internal/harbor`, `internal/servicecatalog` 테스트 0개 — 커버리지 공백 (가치 3 / 위험 1 / M)
- 릴리즈: v0.9.262 (2026-09-02)
