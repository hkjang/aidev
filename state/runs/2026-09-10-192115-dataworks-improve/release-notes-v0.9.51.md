## Data Works v0.9.51

### 주요 변경 사항
- **action center 의 엔타이틀먼트 활성 판정이 런타임 규칙과 어긋나던 문제 수정 (v0.9.51)**: `GET /admin/dataworks/action-center` 는 `ent.Status != "active" || entitlementExpired(...)` 로 상태 문자열을 대소문자·공백까지 그대로 비교했지만, 런타임 조회 게이트는 `store.EntitlementActive`(status 는 `EqualFold`+trim, `expires_at` 은 trim 후 파싱하고 해석 불가면 만료)를 쓴다. 그래서 쓰기 경로가 status 를 정규화하기(v0.9.50) 전에 저장된 `"Active"`·`" active "` 행은 고객 API 키가 `POST /v1/data-products/{key}/query` 에서 정상적으로 서빙받는데도 운영 화면에는 `inactive_access` 로 떠서 운영자가 멀쩡한 접근권을 회수하도록 유도했고, 같은 행이 만료 예고(`entitlement_expiring`) 루프에는 도달하지 못해 갱신 신호까지 잃었음(계약 쪽 `contractScopeStatusActive` 와 같은 유형의 판정 불일치). 판정을 같은 패키지의 런타임 래퍼 `entitlementActive` 에 위임해 두 경로가 하나의 규칙을 쓰게 하고, 호출부가 사라진 `entitlementExpired` 를 삭제(진짜로 닫힌 `revoked` 행은 종전대로 `inactive_access` 로 남음). HTTP 회귀 테스트(레거시 `"Active"`·`" active "` 행 + `revoked` 행 → `inactive_access` 는 `revoked` 하나뿐, 20일 뒤 만료되는 공백 포함 행이 `expiring_access` 로 잡힘)를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 의 "만료 예정 계약·권한 확인" 절에 문서화. `dataworks:v0.9.51` 이미지를 `dataworks-v0.9.51.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.51.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.51.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.51
```