## Data Works v0.9.59

### 주요 변경 사항
- **액션 센터의 공백 포함 계약 만료일 오경보와 조회 창 밖 집계 수정 (v0.9.59)**: 계약 `valid_to`의 빈 값 검사와 `RFC3339Nano` 파싱에 한 번 `strings.TrimSpace`한 지역 값을 사용하도록 런타임 판정과 맞췄다. 앞뒤 공백이 있는 미래 계약이 잘못된 `high` 경고로 뜨거나 조회 기간 밖에서 집계되던 문제를 해결하고, 저장된 값과 JSON 원문은 그대로 유지한다. 실제 SQLite와 `NewServer(...).Routes()`를 사용하는 HTTP 회귀 9사례에서 동일 계약·API 키·접근권의 query 200/403, 기본·7d·13w 액션 센터 summary/actions와 severity, 원문 보존을 검증했다. 수정 전과 trim 변경만 되돌린 변이에서 미래 두 사례가 실패하고 복구 후 통과함을 확인했다. `docs/OPERATIONS.md`에 계약 만료일 판정과 원문 보존을 명시했다. `dataworks:v0.9.59` 이미지를 `dataworks-v0.9.59.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.59.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.59.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres.internal:5432/dataworks?sslmode=require' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.59
```
