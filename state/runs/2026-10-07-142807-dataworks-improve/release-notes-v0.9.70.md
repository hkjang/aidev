## Data Works v0.9.70

### 주요 변경 사항
- **KPI·분석·검토 큐 GET 세 곳이 카탈로그 조회 실패를 "빈 팩토리 / 대기 0건"으로 숨기던 문제 수정 (v0.9.70)**: `handleDataWorksHome`·`handleDataWorksAnalytics`(`internal/proxy/admin_dataworks.go`)는 `FactoryDashboard`·`ListDataAssets`·`ListDataProducts` 의 오류를 전부 `_` 로 버려, 읽을 수 없는 테이블을 `total_assets`·`total_products`·`published_products`·`high_risk`·`ideas_total` 이 모두 0 인 "성공한 대시보드"로 내보냈다. `handleDataWorksReviews` 는 더 고약해서 `review_pending` 조회는 이미 오류 시 `500 reviews_failed` 를 내는데 바로 다음 줄의 `risk_review_pending` 만 오류를 버려, 같은 응답에서 검토 큐는 채워진 채 리스크 검토 큐만 `null` 이 되어(실측 응답 `"risk_review_pending":null`) "리스크 검토 대기 없음"으로 읽혔다. 같은 파일·같은 핸들러의 형제 조회(`/assets` 의 `assets_failed`, `/funnel` 의 `funnel_failed`, `/reviews` 의 첫 `ListDataProducts`)가 이미 같은 실패에 500 을 내고 있어 두 경로가 같은 입력을 다르게 읽고 있었다. 세 지점이 오류를 반환하도록 바꾸고 기존 명명 관례(`<리소스>_failed`)대로 `dashboard_failed`·`analytics_failed` 를 쓰고 reviews 는 기존 코드 `reviews_failed` 를 재사용했다(새 라우트·새 요약 키 없음, OpenAPI 는 오류 코드를 열거하지 않아 감사 수치 불변). 정상 조회된 빈 카탈로그는 종전대로 200 + 0건이며 느슨해진 판정은 없고 게이트를 닫는 방향만이다. 검증: 실제 `store.SQLStore`(SQLite)와 `NewServer(...).Routes()` 프로덕션 라우트만 쓰는 HTTP 회귀 테스트 1스위트 6사례를 추가했다 — 상품 2건(status `review`/`risk_review`)과 자산 1건을 `POST /admin/dataworks/assets`·`POST /admin/dataworks/products` 로 심은 뒤 `data_products`·`data_assets`·`product_ideas` 를 각각 일시 rename 해 home·analytics 가 500 + 지정 오류 코드를 내는지 단언한다. `/reviews` 는 두 조회가 같은 테이블을 읽어 통째 rename 으로는 두 읽기를 구분할 수 없으므로(첫 조회가 먼저 500), `PRAGMA table_info` 로 컬럼을 읽어 `description`(스키마상 NOT NULL, `COALESCE` 없이 `string` 스캔)을 `status='risk_review'` 행에서만 NULL 로 돌리는 뷰로 치환해 리스크 큐만 읽기 실패시키고, 레버 자체를 `GET /admin/dataworks/products?status=review` 는 200 · `?status=risk_review` 는 500 으로 가드해 형제 조회의 기존 500 에 올라타 통과하는 것이 아님을 확인한다. 각 사례 cleanup 에서 테이블·뷰를 복구한 뒤 세 엔드포인트의 기준선(home `total_products`=2·`total_assets`=1, analytics `total_products`=2, `review_pending`=1·`risk_review_pending`=1)으로 돌아오는지도 단언한다. 수정 전 6사례 전부 실패를 재현했고(`risk_review` 사례는 200 + `"risk_review_pending":null`, home 사례는 200 + `total_products`·`published_products`·`high_risk` 모두 0), 프로덕션 파일 `internal/proxy/admin_dataworks.go` 만 되돌린 변이에서 이 6사례만 실패하고 proxy 패키지의 나머지 전부가 통과함을 실행으로 확인했다. `store.FactoryDashboard` 내부가 네 집계 오류를 `_ =` 로 삼키는 것과 `admin_dataworks(_ops)` 의 남은 목록 조회 5곳, 퍼블리시 게이트의 선택적 SLA·비용 조회 오류 처리는 이번 범위 밖이다. 스키마 변경은 없고 web 워크벤치는 이 세 엔드포인트를 호출하지 않으며 레거시 admin UI 는 `/home`·`/analytics` 를 `try/catch` 로 감싸 오류 사유를 렌더하므로 소비자 변경도 없다. `docs/OPERATIONS.md` 3절에 세 오류 코드의 운영 대응 문단을 더했다. PostgreSQL 에서의 동일 조회 실패는 이 환경에 DB 가 없어 미검증(SQLite 만)이다. 릴리즈 검증 실행: 저장소 루트에서 `npm run lint`(eslint 무경고)·`npm test --silent`(11파일 36사례 전부 통과)·`npm run build`(tsc -b && vite build, 2769 모듈) 통과, `go build ./...`·`go vet ./...`·`go build ./cmd/dataworks`·`go test ./... -count=1` 전체 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths, 4항목 전부 `[]`). `dataworks:v0.9.70` 이미지를 `dataworks-v0.9.70.tar.gz` 단일 오프라인 GitHub Release asset으로 제공한다.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.70.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.70.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.70
```