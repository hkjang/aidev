## Data Works v0.9.45

### 주요 변경 사항
- **API Entitlement 만료 예고 부재와 action center 만료 예정 창 하드코딩 수정 (v0.9.45)**: `GET /admin/dataworks/action-center` 는 계약(Contract Scope)에만 30일 고정 예고를 주고 API Entitlement 에는 예고가 전혀 없어, 고객 API 키는 `expires_at` 당일 그대로 죽고 화면에는 그 뒤에야 `inactive_access` 로 나타나 운영자가 갱신을 준비할 신호가 하나도 없었음. 활성·미만료 엔타이틀먼트가 창 안에 만료되면 `expiring_access`/`entitlement_expiring` 액션으로 보고하도록 추가하고, 분기 단위 갱신 주기를 위해 30일로 고정돼 있던 창을 `?expiring_within=13w|45d|72h` 로 지정할 수 있게 해 응답이 적용된 창을 `expiring_within` 으로 되돌려 주도록 정렬. 해석 불가·비양수 값을 기본값으로 조용히 되돌리면 운영자가 물어본 창과 다른 답을 주게 되므로 `400 invalid_expiring_within` 으로 거부. 내장 admin UI 는 새 액션을 기존 "접근 만료" 탭에 합산·필터하고 두 접근 액션 모두에 만료일을 표시하며, workbench 검토 화면의 summary→action type 매핑이 5개뿐이라 나머지 카운터 버튼이 빈 목록을 보여주던 문제도 함께 해소. `parseExpiryHorizon` 표 테스트와 HTTP 회귀 테스트(기본 30일에서 20일 뒤 만료 감지·200일짜리 미감지, `13w` 로 60일 계약까지 감지, `7d` 로 미감지, 잘못된 파라미터 400)를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 와 README 에 문서화. `dataworks:v0.9.45` 이미지를 `dataworks-v0.9.45.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.45.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.45.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.45
```