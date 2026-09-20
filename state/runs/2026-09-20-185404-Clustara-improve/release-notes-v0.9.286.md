## 다른 클러스터의 이벤트가 RCA 증적에 섞이고, 동명 Pod 의 probe 장애 알림이 빠졌습니다

여러 클러스터를 함께 분석할 때 namespace·kind·name 만으로 이벤트와 최근 리비전을 묶어
다른 클러스터의 증적·변경이 연결됐습니다. 이번 릴리즈는 이벤트·리비전 연결과 probe/DNS
중복 제거에 ClusterID 를 포함합니다.

### RCA·배포 분석 · 동명 리소스의 이벤트와 최근 변경을 클러스터별로 연결합니다

여러 클러스터에 같은 namespace·kind·name 의 리소스가 있으면 RCA 이벤트 키에 ClusterID 가
없어 다른 클러스터의 증적이 섞이고 Deployment/Job 의 severity 가 잘못 올라갔습니다.
probe/DNS 이벤트의 중복 제거도 클러스터를 구분하지 않아 두 번째 클러스터의 finding 이
사라졌고, 최근 리비전 선택은 다른 클러스터의 더 최신 변경으로 덮였습니다.

이제 공유 `rcaKey` 에 ClusterID 를 포함해 RCA·Deployment/Job 이벤트 연결, probe/DNS 중복
제거, 설정 변경 보강과 배포 후 오류의 최근 리비전 선택을 클러스터별로 수행합니다.
workload 이름이 메시지에 들어 있는 이벤트를 찾는 fallback 과 배포 후 Warning 연결도
같은 클러스터로 제한합니다. 빈 ClusterID 는 빈 값끼리만 연결하며 다른 클러스터에 매칭하지
않습니다. 같은 클러스터의 반복 이벤트·재스캔 알림 중복 억제와 기존 시간·심각도 정책은 유지합니다.

자원 태그 연결(`AttachFindingResources`)과 NodePressure 영향 Pod 집계는 이번 수정 범위에
포함되지 않습니다. DB 스키마·설정 변경은 없습니다.

### 검증과 운영 영향

신규 회귀 6개(analyzer 5·proxy 1)는 구현 단계에서 수정 전 증적 혼합·잘못된 severity 상승·
finding 누락·알림 sent=1 로 실패한 것을 확인했습니다. 빈 ClusterID 독립성, 같은 클러스터
반복 이벤트, 재스캔 알림 억제, lookback·created 제외·배포 이전/Normal 제외를 검증합니다.
릴리즈 검증 `go build ./...`, `go vet ./...`, `go test ./...`(버전 일치·산출물 이름 게이트 포함)가 모두 통과했습니다.
실 Kubernetes·PostgreSQL·Mattermost 와 원격 모델 golden regression 은 실행하지 않았습니다.
원격 서비스가 필요한 검증 환경을 사용하지 않았으며, proxy 종단 회귀는 SQLite·httptest 를 씁니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.286.tar.gz | Docker 이미지 clustara:v0.9.286 (linux/amd64) |
| clustara-v0.9.286.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.286.md | 오프라인 배포 가이드 |

기존 `scripts/release.sh` 로 생성합니다. 체크섬 확인 후 이미지를 적재합니다.

```bash
sha256sum -c clustara-v0.9.286.tar.gz.sha256
gunzip -c clustara-v0.9.286.tar.gz | docker load
```
