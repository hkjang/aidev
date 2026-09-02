## 스캐너가 보낸 점수를 우리가 버리고 있었습니다

`internal/analyzer/vulnerability.go` 는 576줄인데 단위 테스트가 **한 줄도 없는** 파일이었습니다. 그런데 Trivy · Grype · trivy-operator 스캔과 CycloneDX · SPDX SBOM import 가 **전부** 이 파일 하나를 거쳐 들어옵니다. 실제 도구 출력 형태와 대조해 보니, 아티팩트에 분명히 들어 있는 값 **네 가지**를 파서가 못 읽고 있었습니다.

### ① Grype 로 import 한 모든 CVE 의 CVSS 가 0 이었습니다
Grype 는 점수를 `cvss[].metrics.baseScore` 로 **두 단계 아래**에 둡니다. `cvssScore` 는 한 단계만 내려가서 `V3Score`/`score` 만 찾았습니다 — `baseScore` 라는 키를 **아예 몰랐습니다.**

목록을 CVSS 로 정렬하는 운영자에게 Grype 결과는 **통째로 맨 아래**였습니다.

### ② SBOM 의 `generated_at` 은 두 포맷 모두 항상 비어 있었습니다
코드는 최상위 `created` 를 봤는데 그런 키는 **어느 포맷에도 없고**, 대안으로 둔 `strV(root["creationInfo"])` 는 **객체에 문자열 변환을 걸어** 언제나 빈 문자열이었습니다.

CycloneDX 는 `metadata.timestamp`, SPDX 는 `creationInfo.created` 에 둡니다. "이 SBOM 이 언제 만들어졌나" 는 오래된 SBOM 을 걸러내는 유일한 근거인데 **한 번도 저장된 적이 없었습니다.**

### ③ CycloneDX 1.4 는 generator 를 못 읽었고, 그 실패가 SPDX fallback 까지 막았습니다
1.5 부터 `metadata.tools.components` 로 바뀌었는데 그 형태만 지원해서, 1.4 의 **배열** 형태(syft 구버전)는 빈 값이었습니다. 게다가 빈 값을 그대로 **return** 해 버려서 SPDX 쪽 fallback 으로 넘어가지도 못했습니다.

### ④ Grype 의 EPSS 는 CVE 별 객체 **배열**이라 0 이었습니다
`floatV` 로 스칼라를 기대했으니 언제나 0 이 저장됐습니다.

### 없는 값을 지어내지는 않았습니다
네 가지 모두 "파일에 있는데 우리가 못 읽은" 것이고, 누락분만 더했습니다. 기존 **Trivy `V3Score` 경로의 동작은 그대로**이며 V2/V3 혼용 같은 확장은 하지 않았습니다. 요청이 명시로 준 `generated_at` · `generator` 가 여전히 **파일 값보다 우선**하는 순서도 테스트로 고정했습니다.

### 초록불을 믿지 않았습니다
신규 테스트 6개를 **고치기 전 코드에 되돌려 붙여** 확인했습니다 — 그중 넷이 위 값을 **정확히 지목하며** 실패했습니다. 이 파일의 첫 단위 테스트이기도 합니다.

검증: `go build ./...` · `go vet ./...` · `go test ./...` 17개 패키지 전부 통과.

---
- 이미지: `clustara:v0.9.262`
- 배포 압축본: `clustara-v0.9.262.tar.gz` (SHA256 검증 완료)
