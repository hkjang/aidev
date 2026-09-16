## Data Works v0.9.55

### 주요 변경 사항
- **다른 상품이 이미 쓰는 `contract_key`·Entitlement `id` 로 등록하면 계약·접근권이 그 상품으로 조용히 옮겨 가던 문제 수정 (v0.9.55)**: `POST /admin/dataworks/products/{key}/contract-scopes` 는 `contract_key` 가 플랫폼 전체에서 유일한데도 라우트는 상품별이고 `UpsertContractScope` 는 충돌 시 `product_key` 까지 덮어쓰므로, 운영자가 고객 이름을 딴 키(`ct_bank`)를 두 번째 상품에도 쓰면 첫 상품의 계약이 200 으로 조용히 재부모화되고 그 계약을 가리키는 첫 상품의 모든 Entitlement 는 다음 호출부터 `403 contract_scope_missing` 을 받으면서 admin 목록에는 그대로 남았음(런타임 `usableEntitlement` 는 `contract.ProductKey != product.ProductKey` 면 건너뛴다). `POST …/entitlements` 도 호출자가 준 `id` 가 다른 상품의 행이면 같은 방식으로 접근권을 옮겼음. 두 경로 모두 기존 행이 다른 상품 소유면 `409 contract_key_taken` / `409 entitlement_id_taken` 으로 거부하고, 같은 상품에 다시 보내는 것은 종전대로 갱신(`store.GetAPIEntitlement` 추가). HTTP 회귀 테스트 2건(상품 A 의 `ct_bank`+API 키로 런타임 조회 200 → 상품 B 가 `ct_bank` 등록 시 409·행 불변·런타임 조회 여전히 200·B 목록에 없음·A 재등록은 갱신 / 상품 B 가 A 의 `ent_bank` id 로 발급 시 409·행 불변·A 에서 `suspended` 로 갱신 가능)를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 5절에 문서화. `dataworks:v0.9.55` 이미지를 `dataworks-v0.9.55.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.55.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.55.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.55
```
