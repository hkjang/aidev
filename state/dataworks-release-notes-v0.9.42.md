## Data Works v0.9.42

### 주요 변경 사항
- **Entitlement `scope` 를 부분 문자열로 검사해 조회 권한이 없는 계약도 통과하던 문제 수정 (v0.9.42)**: `POST /v1/data-products/{key}/query` 런타임 게이트가 `strings.Contains(strings.ToLower(ent.Scope), "query")` 로 권한을 판정해, `no-query`·`query:denied`·`data_product:subquery` 처럼 조회를 허용하지 않는(또는 명시적으로 거부하는) 값이 그대로 통과해 상품 데이터가 반환됐음. 반대로 문서화된 와일드카드 `data_product:*` 는 `*` 와 정확히 일치하지도, "query" 를 포함하지도 않아 403 `scope_denied` 로 거부되어 정상 계약이 막혔음. scope 를 쉼표·세미콜론·파이프·공백으로 분리한 권한 목록으로 읽고 각 권한을 `*`·`query`·`data_product:*`·`data_product:query` 와 정확히 비교하도록 `entitlementAllowsQuery` 로 분리했으며, 빈 scope 는 필드 도입 이전 행을 위해 기존대로 무제한 유지. 권한 판정 단위 테스트와 HTTP 회귀 테스트(거부 마커 403, 와일드카드 200)를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 에 허용 권한 목록을 명시. `dataworks:v0.9.42` 이미지를 `dataworks-v0.9.42.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.42.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.42.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.42
```