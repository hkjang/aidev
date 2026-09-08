## Data Works v0.9.46

### 주요 변경 사항
- **한 API 키가 같은 상품에 엔타이틀먼트를 여러 개 가질 때 죽은 계약을 가리키는 행만 평가하던 문제 수정 (v0.9.46)**: `dw_api_entitlements` 는 `id` 단위로 저장돼 한 API 키가 같은 상품에 여러 행을 가질 수 있고 각 행이 자기 `contract_key` 를 가리키는데, `POST /v1/data-products/{key}/query` 는 `store.FindAPIEntitlement` 가 고른 한 행만 평가했음. 그래서 우선순위 1순위 행이 이미 종료된 계약(Contract Scope)을 가리키고 다른 활성 엔타이틀먼트가 살아 있는 계약을 가리키면, 실제로 유효한 접근권이 있는데도 403 `contract_scope_inactive` 로 막혔음. 후보 전체를 활성 우선·최신 우선으로 돌려주는 `store.ListAPIEntitlementCandidates` 를 추가하고(`FindAPIEntitlement` 는 그 첫 원소로 재구현), 런타임이 후보를 훑어 "엔타이틀먼트 활성 + scope 가 조회 허용 + 계약이 이 상품 것이고 active·유효기간 안 + 민감 상품이면 purpose 존재" 를 모두 만족하는 첫 행을 쓰도록 정렬. 요청 본문에 좌우되는 `allowed_fields` 와 호출량을 소모하는 `rate_limit` 은 선택 기준에서 제외해 후보 순회가 분당 한도를 앞당겨 소진하지 않게 했고, 만족하는 후보가 없으면 종전대로 1순위 행으로 검사를 흘려보내 `inactive_entitlement`·`contract_scope_missing`·`contract_scope_inactive`·`missing_contract_purpose` 중 실제 원인이 그대로 응답되도록 유지. HTTP 회귀 테스트(활성 두 행 중 죽은 계약 쪽이 1순위여도 200 이고 살아 있는 계약 키로 처리, 계약이 전부 닫히면 여전히 403 `contract_scope_inactive`)와 store 후보 정렬 테스트를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 에 선택 규칙을 문서화. `dataworks:v0.9.46` 이미지를 `dataworks-v0.9.46.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.46.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.46.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.46
```