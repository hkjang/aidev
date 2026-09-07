## Data Works v0.9.43

### 주요 변경 사항
- **한 API 키가 같은 상품에 엔타이틀먼트를 여러 개 가질 때 활성 행 대신 최근 행을 평가하던 문제 수정 (v0.9.43)**: `dw_api_entitlements` 는 `id` 만 PRIMARY KEY 라 admin POST 마다 새 행이 쌓여 한 API 키가 같은 상품에 엔타이틀먼트를 여러 개 가질 수 있는데(만료된 체험 옆에 발급한 갱신 계약, 감사용으로 남긴 `revoked` 행), `store.FindAPIEntitlement` 는 `ORDER BY updated_at DESC LIMIT 1` 로 마지막에 쓰인 행 하나만 읽었음. 그래서 활성 계약이 revoked/만료 행보다 먼저 쓰였다면 유효한 권한이 있는데도 `POST /v1/data-products/{key}/query` 가 403 `inactive_entitlement` 로 막혔고, 활성 계약이 둘이면 어느 쪽 `allowed_fields`·`rate_limit`·과금으로 처리될지 임의였음(신규 두 행은 `updated_at` 이 같은 값으로 저장될 수 있어 순서가 비결정적). 활성·미만료 행을 우선하고 그중 가장 최근 것을 고르도록 선택 규칙을 바꿨으며, 활성 행이 없으면 종전처럼 최근 행을 돌려줘 게이트가 `missing_entitlement` 가 아닌 `inactive_entitlement` 로 응답하도록 유지. 활성 판정은 `store.EntitlementActive`(파싱 불가 만료일은 만료로 취급) 하나로 모으고 런타임 게이트 `entitlementActive` 가 이를 위임하게 해 선택 규칙과 게이트 규칙이 어긋나지 않도록 정렬. 스토어 선택 테스트, `EntitlementActive` 표 테스트, HTTP 회귀 테스트를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 에 선택 규칙을 문서화. `dataworks:v0.9.43` 이미지를 `dataworks-v0.9.43.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.43.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.43.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.43
```