## Data Works v0.9.44

### 주요 변경 사항
- **데이터 상품 조회 응답 키가 계약 표기를 따르지 않던 문제 수정과 빈 `allowed_fields` 계약 저장 차단 (v0.9.44)**: `POST /v1/data-products/{key}/query` 의 `contractResponseFields` 는 요청 필드를 계약의 `allowed_fields` 와 대소문자 무시로 대조하면서도 응답 `data` 의 키는 클라이언트가 보낸 표기를 그대로 사용했음. 그런데 같은 상품의 `BuildDynamicOpenAPIDocument` 는 `allowed_fields` 를 그대로 응답 프로퍼티 이름이자 `required` 목록으로 선언하므로, 계약이 `score` 인데 소비자가 `Score` 로 요청하면 상품 자신의 OpenAPI 문서에 없는 키가 내려가고 `required` 로 선언된 키는 빠져 응답 스키마를 검증하는 소비자가 깨졌음. 매칭된 계약 표기를 응답 키로 돌려주도록 정렬. 같은 경로에서 `POST /admin/dataworks/products/{key}/contract-scopes` 가 `allowed_fields` 없이도 200 으로 계약을 저장하는 것을 확인했는데, 그런 계약은 런타임에서 모든 조회를 403(`empty_contract_scope`·`forbidden_fields`)으로 막으면서 admin 목록에는 활성 계약으로 보였음. 접근 창·마스킹 정책 검증과 같은 관례로 `400 invalid_allowed_fields` 거부와 trim·대소문자 무시 중복 제거 정규화를 쓰기 경로에 추가. 표기 정규화 단위 테스트, 런타임 HTTP 회귀 테스트(`Score`·`RISK_BAND` 요청 → `score`·`risk_band` 응답), admin 쓰기 회귀 테스트를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 에 문서화. `dataworks:v0.9.44` 이미지를 `dataworks-v0.9.44.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.44.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.44.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.44
```