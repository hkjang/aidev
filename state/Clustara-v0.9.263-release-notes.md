## Harbor 로봇 계정 이름이 들어가면 Secret manifest 가 깨졌습니다

`internal/harbor` 는 460줄이지만 단위 테스트가 **0개**인 패키지였습니다. 그런데 운영자가 그대로 `kubectl apply` 하는 imagePullSecret · Deployment manifest 와, 그 앞에 서는 배포 정책 게이트를 **모두 이 패키지 하나가** 만듭니다. 실제 Harbor 가 돌려주는 값의 형태와 대조해 보니 결함 **네 가지**가 나왔습니다.

### ① imagePullSecret preview 가 유효하지 않은 YAML 이었습니다
note 는 하나의 double-quoted 스칼라인데, 그 안에 넣는 robot 이름·registry 를 `yamlScalar` 로 **각각 따로** 인용하고 있었습니다.

Harbor 가 발급하는 robot 이름은 `robot$project+name` 형태라 `$` · `+` 때문에 **항상** 인용 대상이 되고, 그 따옴표가 note 자신의 따옴표 **안쪽**에 박히면서 **Secret 문서 전체가 파싱에 실패**했습니다 — 즉 실제 Harbor 이름에서는 preview 가 거의 항상 붙여넣을 수 없는 값이었습니다. 이제 note 전체를 한 번만 인용합니다.

### ② 읽을 수 없는 robot 만료 시각이 "만료 없음" 으로 읽혔습니다
`expires_at` 은 검증 없이 사용자 입력 그대로 저장되는데, 정책 엔진은 **RFC3339 만** 파싱하고 실패하면 조용히 넘어갔습니다.

`2026-12-31` 같은 날짜, **Harbor v2 robot API 가 실제로 돌려주는 unix seconds**, 비만료 sentinel `-1` 이 전부 "만료 판정 없음" 이 되어 그대로 **allow 로 통과**했습니다. 이제 날짜 · unix seconds · `-1` 을 정식으로 해석하고, 그래도 해석 불가면 `robot_expiry_unreadable` 로 **승인 필요**를 냅니다. 이미 만료된 robot 이 deny 와 "7일 이내 만료" warn 을 **동시에** 내던 것도 정리했습니다.

### ③ `DefaultSecretName("")` 이 `harbor--pull` 이었습니다
빈 project 를 걸러내려던 가드가 **이어붙인 문자열**에 걸려 있어서 한 번도 동작하지 않았습니다.

### ④ `RegistryHost("  ")` 이 공백 문자열을 host 로 반환했습니다
그 값이 dockerconfig 키와 registry 이름에 그대로 들어갔습니다.

### 초록불을 믿지 않았습니다
신규 테스트 8개는 생성한 manifest 를 **실제 YAML 파서(`gopkg.in/yaml.v3`)로 디코드**합니다. 그 테스트를 **고치기 전 코드에 되돌려 붙여** 확인했습니다 — 넷이 위 결함을 **각각 정확히 지목하며** 실패했습니다. 이 패키지의 첫 단위 테스트이기도 합니다.

검증: `go build ./...` · `go vet ./...` · `go test ./...` 전부 통과.

---
- 이미지: `clustara:v0.9.263`
- 배포 압축본: `clustara-v0.9.263.tar.gz`
