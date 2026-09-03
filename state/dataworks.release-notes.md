## Data Works v0.9.40

### 주요 변경 사항
- **데이터 상품 접근 창 타임스탬프 검증 부재와 admin·runtime 판정 불일치 수정 (v0.9.40)**: `POST /admin/dataworks/products/{key}/contract-scopes`와 `.../entitlements`가 `valid_from`·`valid_to`·`expires_at`를 형식 검사 없이 그대로 저장했지만(바로 옆 approval trace 핸들러는 RFC3339 검증을 수행), 런타임 게이트 `entitlementActive`·`contractScopeActive`는 파싱 실패를 inactive로 처리해 `"2026-12-31"` 같은 자연스러운 입력이 200으로 저장된 뒤 `/v1/data-products/{key}/query`가 전부 403(`inactive_entitlement`·`contract_scope_inactive`)이 되었음. 반대로 admin 쪽 `entitlementExpired`·`dataworks.expiredAt`은 파싱 실패를 "만료 아님"으로, action center는 파싱 불가 `valid_to`를 아예 건너뛰어 런타임이 거부하는 규칙이 화면에서는 정상으로 보이고 retirement 점수에서도 활성 사용으로 집계됐음. 쓰기 경로에 approval trace와 동일한 메시지·에러코드(`invalid_valid_to` 등) 검증을 추가하고 trim 후 저장하며, admin 판독기 3곳을 런타임과 같은 fail-closed로 맞춰 레거시 행이 만료로 노출되도록 정렬. HTTP 회귀 테스트 2건과 도메인 단위 테스트 1건을 추가해 재발을 차단. `dataworks:v0.9.40` 이미지를 `dataworks-v0.9.40.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.40.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.40.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.40
```
