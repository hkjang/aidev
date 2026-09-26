## Data Works v0.9.62

### 주요 변경 사항
- **런타임이 영구히 403 으로 막는 해석 불가 `valid_from` 계약이 액션 센터에 전혀 뜨지 않던 문제 수정 (v0.9.62)**: `contractScopeActive`(`internal/proxy/dataworks_runtime.go`)는 `valid_from` 을 `valid_to` 와 같은 규칙으로 읽어 해석할 수 없으면 그 계약의 모든 런타임 조회를 `403 contract_scope_inactive` 로 영구히 막는데, 액션 센터의 계약 루프(`internal/proxy/admin_dataworks.go`)는 `valid_to` 만 보고 `valid_to` 가 비어 있으면 `continue` 해서 이 "죽은 계약" 이 운영 화면 어디에도 뜨지 않았고 첫 신호가 고객의 거부 신고였음. 루프 본문을 scope 마다 "보고 여부 / severity" 를 먼저 계산하고 append 를 한 번만 하도록 정리해 계약당 항목 1건·`expiring_contracts` +1 을 보장한 뒤(두 값이 모두 해석 불가여도 하나), 트림 후 비어 있지 않은 `valid_from` 이 `time.Parse(RFC3339Nano)` 로 읽히지 않으면 보고 + severity `high` 를 켜는 조건을 더했다. 액션에 `valid_from` 원문도 실었고 `valid_to` 와 함께 저장 원문 그대로 둔다(판정에만 trim). 아직 열리지 않은 미래 `valid_from` 은 오류가 아니라 예정된 계약이므로 일부러 제외했고, 런타임 게이트(`contractScopeActive`·`contractScopeCanServe`)와 JSON 필드 이름은 손대지 않았다. 검증: 실제 `store.SQLStore` 와 `NewServer(...).Routes()` 를 거치는 HTTP 표 테스트 8사례를 추가했다 — `unparseable_from_no_end`·`unparseable_from_future_end`·`unparseable_both` 는 기본·7d·13w 세 창 모두에서 액션 정확히 1건·severity `high`·런타임 `403 contract_scope_inactive`, `revoked`·`draft` 는 0건, `padded_readable_from`·`empty_from`·`whitespace_only_from` 은 0건·런타임 200 이고, 사례마다 `GetContractScope` 와 `GET …/contract-scopes` 의 `valid_from`·`valid_to` 원문 불변을 함께 단언한다. 프로덕션 파일만 되돌린 변이에서 위 3사례만 실패하고 기존 ActionCenter 테스트 6건은 모두 통과함을 실행으로 확인했다. `go build ./...`·`go vet ./...`·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths), `gofmt -l` 클린. `docs/OPERATIONS.md` 5절에 해석 불가 `valid_from` 보고 규칙과 계약당 1건·원문 보존·미래 창 제외를 명시했다. `dataworks:v0.9.62` 이미지를 `dataworks-v0.9.62.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.62.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.62.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.62
```