## Data Works v0.9.49

### 주요 변경 사항
- **action center 가 `revoked`·`draft`·`suspended` 계약도 만료 임박으로 영구 집계하던 문제 수정 (v0.9.49)**: `GET /admin/dataworks/action-center` 의 `contract_expiring` 루프는 계약(Contract Scope)의 `valid_to` 만 보고 `scope.Status` 를 보지 않아, 런타임이 절대 서빙하지 않는 계약(`draft`·`suspended`·`revoked`)까지 "renew, narrow, or retire" 항목으로 보고했음. 종료한 계약의 `valid_to` 는 시간이 갈수록 과거로 멀어지기만 하므로 이 항목은 절대 사라지지 않고, 이미 만료된 창이라 심각도까지 `high` 로 고정돼 운영 화면과 `expiring_contracts` 카운터가 의도적으로 닫은 계약으로 영구히 부풀고 정말 갱신이 필요한 활성 계약을 덮었음(바로 아래 엔타이틀먼트 루프는 이미 `ent.Status != "active"` 를 걸러냄). 런타임의 활성 판정과 어긋나지 않도록 `contractScopeCanServe` 안에 있던 상태 검사를 `contractScopeStatusActive` 로 분리해 두 곳이 같은 규칙을 쓰게 했고, 이미 만료된 **활성** 계약은 종전대로 `high` 로 남겨 운영자가 원하는 신호를 유지(창 기준 필터가 아니라 상태 기준 필터만 추가). HTTP 회귀 테스트(90일 전 만료된 `revoked`·`draft`·`suspended` 3건 + 10일 뒤 만료 `active` 1건 → `expiring_contracts=1`, 액션은 활성 계약 하나)와 `contractScopeStatusActive` 표 테스트를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 5절에 문서화. `dataworks:v0.9.49` 이미지를 `dataworks-v0.9.49.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.49.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.49.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.49
```