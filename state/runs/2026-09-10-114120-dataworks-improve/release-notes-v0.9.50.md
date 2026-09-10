## Data Works v0.9.50

### 주요 변경 사항
- **API Entitlement 쓰기 경로의 `status` 허용값 검증 부재 수정 (v0.9.50)**: `POST /admin/dataworks/products/{key}/entitlements` 는 `expires_at` 형식만 검사하고 `status` 는 그대로 저장해, `enabled`·`actve` 같은 오타가 들어오면 `store.EntitlementActive` 가 `active` 만 인정하므로 발급 순간부터 고객 API 키가 `POST /v1/data-products/{key}/query` 에서 `403 inactive_entitlement` 를 받는데 admin 목록에는 정상 발급된 접근권으로 보였음(계약 쪽 `invalid_contract_status` 와 같은 유형의 fail-closed 오설정). 계약 쪽 `contractScopeStatusKnown` 과 대칭으로 `entitlementStatusKnown` 을 두고 `active`(기본)·`draft`·`suspended`·`revoked` 만 허용하며 그 외에는 `400 invalid_entitlement_status` 로 거부하고, 소문자·trim 정규화 후 빈 값은 `active` 로 채워 응답이 실제 저장 행과 일치하도록 정렬(감사용으로 남기는 `revoked` 행은 관례대로 계속 저장 가능). HTTP 회귀 테스트(오타 상태 거부·미저장, ` ACTIVE ` 정규화 저장, `revoked` 허용, 생략 시 기본값이 응답·저장 행 모두 `active`)와 `entitlementStatusKnown` 표 테스트를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 5절에 문서화. `dataworks:v0.9.50` 이미지를 `dataworks-v0.9.50.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.50.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.50.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.50
```