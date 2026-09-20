## 운영 홈의 동명 Pod CPU·메모리 태그가 다른 클러스터 값으로 덮였습니다

여러 클러스터에 같은 이름의 Pod가 있을 때 RCA finding에 다른 클러스터의 자원 설정이
표시됐습니다. 이번 릴리즈는 자원 태그를 연결할 때 클러스터를 함께 구분합니다.

### RCA 자원 태그 · 동명 Pod의 CPU·메모리를 클러스터별로 연결합니다

여러 클러스터에 namespace·kind·name이 같은 Pod가 있으면 운영 홈의 RCA 자원 태그가
다른 클러스터의 CPU·메모리 requests/limits로 덮였습니다. `AttachFindingResources`의
인벤토리 인덱스와 finding 조회에 모두 ClusterID를 포함한 기존 `rcaKey`를 사용해,
각 finding에 같은 클러스터의 자원 설정만 연결합니다.

빈 ClusterID는 빈 값끼리만 연결합니다. namespace·kind·name 구분, kind 대소문자 처리,
일치 항목이 없을 때의 nil 유지와 Deployment template 자원 추출은 기존 동작을 유지합니다.
DB 스키마·설정 변경은 없으며 NodePressure 영향 Pod 집계는 이번 수정 범위 밖입니다.

### 검증과 배포

- 신규 analyzer 회귀와 실제 SQLite·Server.Routes·HTTP 홈 회귀는 구현 단계에서 수정 전 실패와 수정 후 통과를 확인했습니다.
- 입력 역순·빈 ID·namespace/kind/name 구분·nil 유지·Deployment template을 검증합니다.
- 릴리즈 검증 통과: `go build ./...`, `go vet ./...`, `go test ./...`, 버전 일치·산출물 이름 릴리즈 게이트.
- 실 Kubernetes·PostgreSQL·브라우저 렌더링 및 운영 환경 배포 확인은 수행하지 않았습니다.
- 기존 `scripts/release.sh`로 linux/amd64 이미지 압축본, SHA256, 오프라인 가이드를 제공합니다.
- DB 마이그레이션이나 새 설정은 필요하지 않습니다. 롤백 시 이전 `clustara:v0.9.286` 이미지를 사용합니다.
