## Data Works v0.9.47

### 주요 변경 사항
- **런타임이 실제로 수행하지 않는 마스킹 문구가 민감 상품 Publish Gate 를 통과시키던 문제 수정 (v0.9.47)**: `applyMasking` 이 구현한 정책은 `redact`·`hash` 뿐이고 그 외 값은 원본을 그대로 돌려주는데, `POST /admin/dataworks/products/{key}/contract-scopes` 는 `masking_policy` 를 검증 없이 저장했고 민감 상품 Publish Gate 는 비어 있지 않고 `none` 이 아니면 마스킹 구성으로 집계해, `"개인 단위 원천값 제외"` 같은 서술형 문구가 게시를 통과시키면서 조회 응답은 원본 값을 그대로 내보냈음. 같은 판독기는 반대 방향으로도 fail-open 이었는데, 마스킹은 계약별로 적용되는데 계약이 여럿일 때 그중 하나만 마스킹하면 상품 전체가 통과해 나머지 고객은 게시 후에도 원본 값을 받았음. 쓰기 경로에서 `none`(빈 값 포함)·`redact`·`hash` 만 허용하고(`400 invalid_masking_policy`, 소문자·trim 정규화) 게이트는 "아직 조회를 처리할 수 있는 계약이 모두" 런타임이 강제하는 정책을 가질 때만 참이 되게 정렬. `revoked` 상태나 이미 닫힌 창(파싱 불가 `valid_to` 포함)은 다시는 조회를 처리할 수 없으므로 제외해 옛 계약이 영구히 게시를 막지 않게 했고, 아직 시작되지 않은 창은 나중에 서빙하므로 포함한다(`contractScopeCanServe` 로 분리하고 `contractScopeActive` 가 이를 재사용). HTTP 회귀 테스트 3건(서술형 거부·정규화 저장, 레거시 행의 게이트 판정, 계약 둘 중 하나만 마스킹 시 차단 + 종료·미래 계약 처리)과 정책 분류·`contractScopeCanServe` 표 테스트, 허용 목록과 `applyMasking` 동기화 테스트를 추가해 재발을 차단하고 `docs/OPERATIONS.md` 5절에 문서화. `dataworks:v0.9.47` 이미지를 `dataworks-v0.9.47.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.47.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.47.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.47
```