## Data Works v0.9.48

### 주요 변경 사항
- **Contract Scope 접근 창 순서와 미지원 `status` 검증 부재 수정 (v0.9.48)**: `POST /admin/dataworks/products/{key}/contract-scopes` 는 `valid_from`·`valid_to` 의 RFC3339 형식만 보고 두 값의 순서는 보지 않아 `valid_from` 이 `valid_to` 보다 뒤인 뒤집힌 창을 200 으로 저장했는데, 런타임 `contractScopeActive` 는 현재 시각이 창 안에 있을 때만 참이므로 그렇게 저장된 계약은 존재하는 내내 모든 조회를 403 `contract_scope_inactive` 로 막았음. `status` 도 store 가 빈 값만 `active` 로 채울 뿐 허용값 검증이 없어 `actve`·`enabled` 같은 오타가 그대로 저장돼 계약이 즉시 죽었고, 두 경우 모두 admin 목록에는 정상적인 고객 계약으로 보였음(같은 핸들러가 이미 형식·`rate_limit`·`allowed_fields`·`masking_policy` 는 거부함). 뒤집힌 창은 `400 invalid_access_window`, 미지원 상태는 `400 invalid_contract_status`(허용: `active`·`draft`·`suspended`·`revoked`)로 거부하고 `status` 를 소문자·trim 정규화한 뒤 빈 값은 `active` 로 채워 응답이 실제 저장 행과 일치하도록 정렬. HTTP 회귀 테스트(뒤집힌 창 거부·미저장, 오타 상태 거부·미저장, 정규화 저장, 생략 시 기본값)와 `contractScopeStatusKnown` 표 테스트를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 5절에 문서화. `dataworks:v0.9.48` 이미지를 `dataworks-v0.9.48.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.48.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.48.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.48
```