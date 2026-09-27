## Data Works v0.9.65

### 주요 변경 사항
- **런타임이 영구히 403 으로 막는 거꾸로 된 계약 창(`valid_from` > `valid_to`)이 액션 센터에 전혀 뜨지 않던 문제 수정 (v0.9.65)**: `contractScopeActive`(`internal/proxy/dataworks_runtime.go`)는 `valid_from` 이 올 때까지 모든 조회를 막고 `contractScopeCanServe` 는 `valid_to` 가 지나면 닫으므로, `valid_from` 이 `valid_to` 보다 뒤인 거꾸로 된 창은 그 계약이 존재하는 내내 `403 contract_scope_inactive` 다. 쓰기 경로는 `dataWorksAccessWindowOrdered`(`internal/proxy/admin_dataworks.go`)로 `400 invalid_access_window` 를 내며 이미 막고 있지만 그 검사 이전에 저장된 행은 남아 있고, 액션 센터의 계약 루프는 `valid_to` 가 과거이거나 lookahead 안일 때 또는 두 값 중 하나가 파싱 불가일 때만 보고하므로 두 값 모두 읽히고 `valid_to` 가 lookahead 밖인 거꾸로 된 창은 운영 화면 어디에도 뜨지 않았고 첫 신호가 고객의 거부 신고였다. `valid_from` 분기를 switch 로 바꿔 파싱 실패(종전대로 high) 외에 `rawValidTo != "" && validToErr == nil && validFrom.After(validTo)` 이면 `contract_expiring`/high 로 보고하도록 더했다(`valid_to` 파싱 결과를 루프 안에서 재사용하도록 지역 변수만 끌어올렸고 기존 세 분기의 판정은 그대로다). 경계 `valid_from == valid_to` 는 쓰기 경로와 같이 통과시키고, 판정에만 트림한 지역 값을 쓰며 저장 원문·admin 응답·액션 JSON 은 재포맷하지 않는다. 새 action type·summary 키를 만들지 않아 계약당 액션 1건 보장과 기존 web 화면 동작은 그대로다. 검증: 실제 `store.SQLStore`(SQLite) + `NewServer(...).Routes()` HTTP 경로만 쓰는 표 테스트 9사례를 추가했다 — `inverted_far_future`·`inverted_past_end` 는 기본·`7d`·`13w` 세 lookahead 창 모두에서 액션 정확히 1건·`severity` high·`summary.expiring_contracts`=1 및 런타임 403 `contract_scope_inactive`, `ordered_window`·`empty_from`·`empty_to` 는 0건·런타임 200, `equal_bounds`·`draft_inverted`·`revoked_inverted` 는 0건이며, 사례마다 `GetContractScope` 와 `GET …/contract-scopes` 의 `valid_from`·`valid_to` 원문 불변까지 단언한다. 프로덕션 파일만 되돌린 변이에서 `inverted_far_future` 사례만 실패하고 나머지 8사례와 기존 ActionCenter·AccessWindow 테스트는 전부 통과함을 실행으로 확인했다. `go build ./...`·`go vet ./...`·`go test ./... -count=1` 전체 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths), 변경한 두 Go 파일은 `gofmt -l` 출력 없음. `docs/OPERATIONS.md` 5절에 거꾸로 된 계약 창 판정 규칙을 명시했다. `dataworks:v0.9.65` 이미지를 `dataworks-v0.9.65.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.65.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.65.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.65
```