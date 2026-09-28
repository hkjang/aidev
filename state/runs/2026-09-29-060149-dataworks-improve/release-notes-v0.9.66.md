## Data Works v0.9.66

### 주요 변경 사항
- **액션 센터의 목록 조회 장애가 HTTP 200·경고 0건으로 표시되던 문제 수정 (v0.9.66)**: 적합도·계약·권한·Watermark·비용·폐기 후보의 6개 직접 목록 조회 오류를 각 조회 오류별 HTTP 500으로 반환한다. 실제 SQLite·store.SQLStore·NewServer.Routes() HTTP 회귀 테스트 6사례에서 테이블을 일시 rename하면 500·오류 코드·summary/actions 부재를, 복구하면 200·빈 actions·9개 0 집계를 확인한다. 구현 단계에서 수정 전 실패→수정 후 통과→프로덕션 코드 원복 시 같은 실패를 확인했다. 퍼블리시 게이트 평가 및 내부 선택적 조회 오류 처리는 이번 범위 밖이다. 스키마 변경은 없다. `dataworks:v0.9.66` 이미지를 `dataworks-v0.9.66.tar.gz` 단일 오프라인 GitHub Release asset으로 제공한다.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.66.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
gunzip -c dataworks-v0.9.66.tar.gz | docker load
docker compose up -d
```
기존 운영 환경변수를 유지하고 이번 태그가 지정된 docker-compose.yml을 사용하세요.

### 운영 확인 및 롤백
배포 담당자는 소규모 환경에서 /health·/ready·/dataworks/와 액션 센터를 확인한 후 확대합니다. 정상 저장소에서 액션 센터가 500을 반환하거나 상태 확인이 실패하면 확대를 중단하고 v0.9.65 이미지로 복귀합니다. 배포 후 24시간 동안 대상 운영 환경의 확인 완료율 100%와 장애를 0건 집계로 오인한 사례 0건을 확인합니다.

### 검증 범위
운영 PostgreSQL·실제 Keycloak·브라우저 E2E·외부 사용자 최초 사용 검증은 미실시입니다. 실제 배포와 관측은 후속 배포 담당자의 작업입니다.

검증 완료: Go build·vet·전체 테스트(proxy 51.564s, store 22.622s), 릴리즈 버전 일치 검사, API 감사 누락 0(550 routes/612 OpenAPI paths), npm ci(취약점 0)·lint·27 tests·build. Docker linux/amd64 이미지의 /health·/ready·/dataworks/와 gzip·manifest 태그 확인 통과. 로컬 Go 1.26.7/Node 22.23.1, Docker 빌드는 저장소의 Go 1.25/Node 24 설정을 사용했습니다. 기존 release.sh는 VERSION만 전달하므로 이미지의 Commit·BuildTime은 기존 기본값 unknown입니다.
