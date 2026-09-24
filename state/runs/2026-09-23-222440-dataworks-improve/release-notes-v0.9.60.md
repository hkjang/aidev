## Data Works v0.9.60

### 주요 변경 사항
- **액션 센터 `expiring_within` 이 int64 범위를 넘으면 감긴 창으로 조용히 200 을 돌려주던 문제 수정 (v0.9.60)**: `parseExpiryHorizon` 의 `d`·`w` 분기가 `time.Duration(count)*unit` 을 범위 검사 없이 계산해 `106751992d` 는 약 20시간(`19h59m5.224192s`)으로, `200000000d`·`30000000w` 는 음수 창으로 감겼고, 감긴 값이 뒤따르는 양수 검사를 통과하면서 200 응답의 `expiring_within` 필드에 "실제 적용된 창"으로 실려 나갔다. 문서는 "양수가 아니거나 해석할 수 없는 값은 400" 이라고 계약한다. 곱하기 전에 `int64(count) > math.MaxInt64/int64(unit)` 을 검사해 기존 `400 invalid_expiring_within` 으로 거부하도록 고쳤고, 경계값 `106751d`·`15250w` 는 종전대로 통과한다. 문서에 없는 새 정책이 되는 상한 클램프는 도입하지 않았고, 원래도 거부하던 `time.ParseDuration` 분기(`2562048h`)는 테스트로 확인만 하고 건드리지 않았다. 검증: 실제 SQLite 와 `NewServer(...).Routes()` 를 거치는 HTTP 회귀 테스트에서 수정 전 `expiring_within=106751992d` 가 `"19h59m5.224192s"` 창으로 200 을 반환하는 것을 먼저 재현한 뒤, 수정 후 과대값 5사례가 400 이고 경계 2사례가 정상 창을 돌려줌을 확인했으며 `TestParseExpiryHorizon` 에 표 7사례를 추가했다. `go build ./...`·`go vet ./...`·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0, `gofmt -l` 클린. `docs/OPERATIONS.md` 5절에 만료 창 해석 범위를 두 줄 명시했다. `dataworks:v0.9.60` 이미지를 `dataworks-v0.9.60.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.60.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.60.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.60
```