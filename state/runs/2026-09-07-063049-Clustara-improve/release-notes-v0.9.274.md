## 스캔 원장이 실행된 적 없는 파서를 말했고, 벤더가 High 로 매긴 CVE 는 게이트를 그냥 지나갔습니다

취약점·CIS 스캔 import 는 CI 나 Trivy Operator 가 만든 JSON 을 그대로 받아, Admission 게이트·컴플라이언스 화면·DW export 가 근거로 삼는 원장으로 바꾸는 경로입니다. 이번 릴리즈는 그 정규화(`analyzer.NormalizeVulnerabilityScan`·`NormalizeKubeBench`)에서 다섯 지점을 고칩니다. 앞의 셋은 게이트가 내리는 판단 자체를 바꾸고, 뒤의 둘은 CIS 결과를 읽을 수 있게 만듭니다.

### ① `scanner="unknown"` 이라고 저장한 스캔을 실제로는 Trivy 리더가 읽었습니다

`detectScanner` 가 아무 형식에도 맞히지 못하면 `"unknown"` 을 돌려주는데, 그 값이 그대로 `switch` 의 `default` 로 떨어져 **Trivy 파서가 실행**되고 스캔 행의 `scanner` 컬럼에는 `unknown` 이 저장됐습니다. 스캔 목록·감사 로그·DW export 가 **한 번도 실행된 적 없는 파서 이름**을 말한 셈이고, 호출자가 `scanner: "snyk"` 처럼 지원하지 않는 이름을 명시한 경우도 똑같이 그 이름으로 저장되면서 Trivy 리더로 읽혔습니다.

이제 형식이 무엇이든 **실제로 돌아간 리더**(`trivy`)로 라벨링하고, 요청에 적힌 이름은 `summary.requested_scanner` 로 따로 남깁니다. 더 중요한 것은 그 다음입니다 — 읽히지 않은 아티팩트와 정말로 깨끗한 이미지는 **둘 다 findings 0건**이라 Admission 게이트가 구분할 수 없었습니다. 그래서 형식 미인식 + 0건이면 `summary.parse_notice` 로 "인식되지 않아 Trivy 리더로 읽었고 한 건도 찾지 못했다" 를 명시하고, 인식 여부는 항상 `summary.scanner_detected` 로 남깁니다.

### ② Grype 의 `Negligible` 과 RPM 권고의 `Important`·`Moderate` 가 전부 `Unknown` 으로 접혔습니다

`NormalizeSeverity` 의 매핑표에 이 세 등급이 없어서 `default: "Unknown"`(랭크 0)으로 떨어졌습니다. `Negligible` 은 Low 미만이라는 **스캐너가 매긴 등급**인데 "심각도 정보 없음" 과 같은 칸에 들어갔고, 더 나쁜 쪽으로 Red Hat·SUSE 계열 권고가 쓰는 `Important`(= High)·`Moderate`(= Medium)도 랭크 0이 되어 **Admission 게이트의 승인 임계(`SeverityRank >= 3`)에 걸리지 않았습니다** — 벤더가 High 로 매긴 CVE 가 게이트를 그냥 통과했다는 뜻입니다.

이제 `Important`→High, `Moderate`→Medium 이고 `Negligible` 은 자기 등급을 유지합니다. 게이트 임계값은 저장된 계약이므로 등급을 다시 번호 매기지 않았습니다 — `Negligible` 은 Low 와 같은 랭크 1을 공유하며(둘 다 모든 임계 아래), Medium 이 승인 게이트로 밀려 올라가는 일은 없습니다.

### ③ severity 카운트 맵이 새 등급 하나에 핸들러를 죽일 수 있었습니다

워크로드 롤업과 요약이 `map[string]any{"Critical":0, …}` 를 손으로 적어 두고 `m[sev].(int) + 1` 로 증가시켰기 때문에, 정규화가 표에 없는 등급을 하나라도 내보내면 **nil 에 대한 타입 단언 = 패닉**이었습니다(요약 API 500). 두 맵 모두 정규화기의 `analyzer.SeverityLevels()` 에서 키를 만들도록 바꿔 등급 목록이 한 곳에만 존재하게 했습니다. 취약점 화면의 severity 필터에도 `Negligible` 을 넣었습니다.

### ④ kube-bench 결과의 Section 이 전부 상위 control 의 설명 문구였습니다

kube-bench JSON 은 `Controls[{id,text}] > tests[{section:"1.1",desc}] > results[{test_number}]` 로 중첩되는데, 수집기가 상속받은 값을 **먼저** 채택해서(`firstNonEmptyV(section, x["section"], …)`) 가장 바깥 노드의 자유 문구("Master Node Security Configuration")가 먼저 잡히고 그 아래 모든 control 에 그대로 눌러앉았습니다. CIS Benchmark 화면의 Section 열이 `1.1` 대신 상위 control 텍스트를 표시했고, 섹션 단위로 결과를 묶을 수 없었습니다. 이제 노드 자신의 `section` 이 상속값을 이깁니다(섹션이 아예 없는 문서는 종전대로 가장 가까운 라벨을 물려받습니다).

### ⑤ `scored: false` 인 control 이 전부 scored 로 기록됐습니다

kube-bench 의 `scored` 는 JSON **bool** 인데 문자열로 읽어서(`strV(true)` = `""`) `!EqualFold("", "false")` 가 언제나 참이었습니다. CIS 점수에 반영되는 control 과 참고용 control 의 구분이 사라졌습니다. bool·문자열 양쪽을 읽고, 필드가 없으면 종전대로 scored 로 봅니다. 함께, 핸들러는 읽고 있는데 정규화기가 한 번도 설정한 적 없던 `BenchmarkVersion` 을 채워 cis-1.7 과 cis-1.23 결과를 구분할 수 있게 했습니다.

### 검증

신규 테스트 8개(analyzer 7 + proxy 1)를 고치기 전 코드에 되돌려 붙여 다섯 결함을 각각 지목하며 실패함을 확인했습니다 — 인식된 형식의 라벨과 section 없는 평평한 문서의 기존 동작을 지키는 오탐 회귀 테스트를 포함합니다. `go build ./...`·`go vet ./...`·`go test ./...` 전부 통과합니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.274.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.274.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.274.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 이미지 로드
gunzip -c clustara-v0.9.274.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.274
```
