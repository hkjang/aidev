## AMD·Intel GPU가 있는 노드가 용량 리포트에서 사라졌습니다

`GET /admin/k8s/capacity`의 SCALE-08 GPU 표는 `nvidia.com/gpu`만 읽었습니다.
같은 인벤토리를 읽는 노드 모니터링은 NVIDIA·AMD·Intel을 모두 집계하므로,
AMD·Intel 운영자는 노드 모니터링에서 GPU를 확인하면서도 용량 리포트에서는 빈 표를 받았습니다.

v0.9.299는 용량 집계 안에서 `nvidia.com/gpu`·`amd.com/gpu`·`intel.com/gpu`의
노드 allocatable과 해당 클러스터·노드에 배치된 Pod requests를 각각 합산합니다.
혼합 공급자 노드의 수량도 한 행에 합쳐집니다.

### 용량 API · 세 공급자의 GPU 수량을 보고합니다

| 입력 | 수정 전 용량 표 | v0.9.299 allocatable / requested / idle |
|------|----------------|---------------------------------------|
| AMD 4개, Pod 요청 1개 | 행 누락 | 4 / 1 / 3 |
| Intel 2개, Pod 요청 1개 | 행 누락 | 2 / 1 / 1 |
| NVIDIA 8개, Pod 요청 2개 | 8 / 2 / 6 | 8 / 2 / 6 |

대상은 위 세 리소스 키를 사용하는 기존 Kubernetes 운영자입니다. 비용 집계의 GPU 단가와
공유 함수 `podRequestGPU`는 변경하지 않았습니다. MIG·추가 리소스 키, limits 대체,
init 컨테이너의 스케줄러 의미를 새로 지원하는 릴리즈는 아닙니다.

기존 일반·init 컨테이너 합산, CPU packing, 클러스터·노드 격리와 정렬,
미배치/미등록 노드 Pod 제외를 유지합니다. `idle_gpu`는 allocatable에서 requested를 뺀 값으로
음수가 될 수 있습니다. 노드 모니터링의 Available은 0으로 제한되므로 잔여값끼리 비교하지 말고
allocatable/requested를 비교해야 합니다. API 키·DB 스키마·설정·의존성 변경은 없습니다.

### 검증

- 실제 SQLite SQLStore → `kube.InventoryFromObject` → `Server.Routes` → HTTP JSON 테스트에서
  수정 전 AMD·Intel의 빈 행 실패를 확인했습니다. 수정 후 함대·AMD·Intel·NVIDIA·노드 없는
  클러스터·알 수 없는 클러스터 6범위와 노드 모니터링의 allocatable/requested 일치를 검증합니다.
- 단위 15케이스로 혼합 공급자/컨테이너, 문자열·JSON 숫자, init 합산, 빈 GPU, 미지원 키,
  limits 제외와 음수 Idle을 검증합니다. 구현 단계에서 NVIDIA-only 키로 되돌렸을 때 같은
  회귀 테스트가 재실패하는 것도 확인했습니다.
- 릴리즈 게이트: gofmt, `git diff --check`, `go build ./...`, 서버 단독 빌드,
  `go vet ./...`, `go test ./... -count=1`, 버전 일치와 산출물 이름 검사 모두 통과했습니다.
  전체 테스트는 테스트가 있는 19패키지를 통과했습니다(proxy 73.398s, store 17.003s).
- 기존 `scripts/release.sh`로 linux/amd64 이미지를 한 번 빌드하고 패키징했습니다. 배포 압축본의
  SHA256·이미지 digest·전체 blob 무결성을 확인했으며, 같은 이미지의 `/health`·`/ready`·
  `/admin`·용량 API가 HTTP 200으로 응답하고 관리자 페이지가 v0.9.299를 표시함을 확인했습니다.
  이미지 기동 검증은 임시 SQLite에서 실행했고 검증 컨테이너는 종료했습니다.
- 실제 Kubernetes/GPU 장치·PostgreSQL·브라우저 렌더링 및 팀 외부 사용자의 첫 사용 검증은
  실행하지 않았습니다. 자동 HTTP 검증 범위는 SQLite 저장소부터 응답 JSON까지입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.299.tar.gz | Docker 이미지 패키지, linux/amd64 |
| clustara-v0.9.299.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.299.md | 오프라인 배포 가이드 |

```bash
sha256sum -c clustara-v0.9.299.tar.gz.sha256
gunzip -c clustara-v0.9.299.tar.gz | docker load
```

기존 배포 설정에서 이미지 태그를 `clustara:v0.9.299`로 교체합니다. 운영 배포와 원격 게시·업로드는
이 로컬 릴리즈 세션에 포함하지 않았습니다. 운영 비밀 설정·백업·실제 클러스터 헬스체크 및
외부 모델을 호출하는 golden prompt 검증은 배포 환경에서 확인해야 합니다.

### 배포 후 확인과 롤백

이번 변경은 Tier 3 개선으로 릴리즈 노트와 운영 문서에 안내합니다. 배포 담당 운영자가 먼저
AMD·Intel 대표 클러스터 각 1곳에서 용량 API를 조회하고, 동일 인벤토리의 allocatable/requested와
맞는지 확인한 뒤 적용 범위를 넓힙니다. 첫 배포 후 24시간 이내 확인 목표는 두 클러스터 모두
해당 노드 행을 표시하고 수량 불일치가 0건인 것입니다. 초기 문의에서 GPU 행 누락과 잘못된
수량 보고를 확인합니다. 실제 이용 결과는 이번 세션에서 측정하지 않았습니다.

행 누락 또는 allocatable/requested 불일치가 1건이라도 생기거나 `/health`·`/ready`가 실패하면
배포 담당 운영자가 확대 적용을 중단하고 기존 설정으로 `clustara:v0.9.298` 이미지를 복구합니다.
이번 변경에는 DB 마이그레이션이 없으므로 코드 롤백을 위한 DB 되돌리기는 필요하지 않습니다.
