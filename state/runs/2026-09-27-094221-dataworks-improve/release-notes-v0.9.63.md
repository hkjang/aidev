## Data Works v0.9.63

### 주요 변경 사항
- **퍼블리시 게이트가 승인 이력의 `expires_at` 만 트림 없이 읽어 공백 섞인 승인이 만료로 뒤집히고 출시가 영구히 막히던 문제 수정 (v0.9.63)**: `bestApprovalStatus`(`internal/dataworks/domain.go`)는 같은 함수 안에서 `trace.Step`·`trace.Status` 는 `strings.TrimSpace` 후 비교하면서 `trace.ExpiresAt` 만 원문으로 `!= ""` 를 검사하고 그 원문을 `time.Parse(time.RFC3339Nano, …)` 에 넘겨, 앞뒤 공백이 섞인 먼 미래 만료일(`" 2030-01-01T00:00:00Z "`)을 가진 `approved` 행이 파싱 실패로 `expired` 가 되었다. 그 결과 승인이 실제로 유효한데도 `MissingApprovals`·`BlockedReasons` 에 올라가 해당 상품의 퍼블리시가 영구히 막혔다. `store.EntitlementActive`·런타임 `contractScopeActive`·액션 센터는 모두 한 번 트림한 값으로 파싱하므로 승인 만료일만 규칙이 달랐다. 한 번 `TrimSpace` 한 지역 값을 빈 값 검사와 파싱에 함께 쓰도록 맞췄고, 트림 후에도 해석할 수 없는 값은 종전대로 `expired` 로 닫는다(주석에 이유 명시). `ApprovalTrace` 는 값 전달이고 필드에 다시 대입하지 않으므로 저장된 원문과 API 응답 JSON 은 그대로다. `EvaluatePublishGateV2` 가 `EvaluatePublishGate` 를 호출하므로 두 게이트가 함께 낫는다. 검증: 표 테스트 7사례를 추가했다 — 공백 미래·공백 없는 미래·공백만·빈 값은 `approved` + 차단 없음, 공백 과거·공백이 든 `" invalid-time "`·만료 == now 는 `expired` + 차단이며, 사례마다 `EvaluatePublishGate` 와 `EvaluatePublishGateV2` 양쪽에서 `ApprovalStatus`·`MissingApprovals`·`BlockedReasons`·`Allowed` 를 단언하고 `ExpiresAt` 원문 불변까지 확인한다. 수정 전 실행에서 `padded_future_expiry`·`whitespace_only_expiry` 두 사례만 `expired` 로 실패하고 나머지 5사례는 통과함을 확인했다. `go build ./...`·`go vet ./...`·`go test ./...` 전체 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths), `gofmt -l` 클린. `docs/OPERATIONS.md` 4절에 승인 만료일 판정 규칙을 명시했다. `dataworks:v0.9.63` 이미지를 `dataworks-v0.9.63.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.63.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.63.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.63
```