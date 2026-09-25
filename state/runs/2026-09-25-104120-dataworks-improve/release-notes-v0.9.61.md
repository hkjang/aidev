## Data Works v0.9.61

### 주요 변경 사항
- **계약 `valid_from` 만 트림 없이 파싱해 공백이 섞인 계약이 이미 열린 창에서도 403 으로 막히던 문제 수정 (v0.9.61)**: `contractScopeActive`(`internal/proxy/dataworks_runtime.go`)는 `strings.TrimSpace(scope.ValidFrom)` 으로 빈 값만 검사한 뒤 트림하지 않은 원문을 `time.Parse` 에 넘겨, 앞뒤 공백이 섞인 `valid_from` 행은 창이 이미 열려 있고 `valid_to` 가 먼 미래여도 파싱 실패로 모든 조회가 `403 contract_scope_inactive` 가 되었다. 같은 함수의 `valid_to` 검사와 액션 센터, `store.EntitlementActive` 는 모두 한 번 트림한 값으로 파싱하므로 `valid_from` 만 규칙이 달랐다. 한 번 트림한 지역 변수를 빈 값 검사와 파싱에 함께 쓰도록 맞췄고, `scope` 는 값 전달이며 필드를 다시 대입하지 않으므로 저장된 원문과 API 응답 JSON 은 그대로다. 검증: 실제 `store.SQLStore` 와 `NewServer(...).Routes()` 를 거치는 HTTP 회귀 표 테스트 7사례(공백 과거·공백 없는 과거·빈 값·공백만 → 200, 공백 미래·공백 없는 미래·`not-a-date` → 403 `contract_scope_inactive`)를 추가하고 사례마다 `GetContractScope` 와 `GET …/contract-scopes` 의 원문 불변을 확인했으며, 트림을 되돌린 변이에서 공백 과거 사례만 403 으로 실패하고 나머지 6사례는 통과함을 실행으로 확인했다. `go build ./...`·`go vet ./...`·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths). `docs/OPERATIONS.md` 5절에 계약 시작일 판정과 원문 보존을 명시했다. `dataworks:v0.9.61` 이미지를 `dataworks-v0.9.61.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.61.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.61.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.61
```