## Data Works v0.9.41

### 주요 변경 사항
- **계약 `rate_limit`(분당 호출 한도)이 런타임에서 전혀 강제되지 않던 문제 수정 (v0.9.41)**: `dw_contract_scopes.rate_limit`은 저장되고 런타임 응답의 `rate_limit` 필드로 반환되며 `BuildDynamicOpenAPIDocument`가 상품 OpenAPI에 `429 Rate limit exceeded`로 광고까지 했지만, `POST /v1/data-products/{key}/query`는 그 값을 읽기만 할 뿐 검사하지 않아 분당 60회로 계약한 고객도 무제한 호출이 가능했고 `dw_usage_metering.over_limit_calls`는 설계만 되고 영원히 0이었음. 계약 키별 고정 1분 창(벽시계 분 경계 정렬) 카운터를 `internal/proxy/dataworks_ratelimit.go`에 추가해 허용된 호출에는 `X-DataWorks-RateLimit-{Limit,Used,Reset}` 헤더를, 초과 호출에는 `429 contract_rate_limited`와 `Retry-After`를 반환하도록 정렬. 거부된 호출은 창을 소모하지 않고 `rate_limit_exceeded:<contract>`로 감사 로그에 남으며 `failed_calls`가 아니라 `over_limit_calls`로 집계되도록 `IncrementUsageMetering`에 `overLimit` 파라미터를 추가했고, 런타임이 "무제한"으로 읽는 음수 `rate_limit`은 쓰기 경로에서 `400 invalid_rate_limit`으로 거부. 리미터 창 롤오버·`Retry-After` 올림 단위 테스트와 HTTP 회귀 테스트 2건을 추가해 재발을 차단. `dataworks:v0.9.41` 이미지를 `dataworks-v0.9.41.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.41.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.41.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.41
```
