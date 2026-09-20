# 회차 노트 2026-09-21-073416-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:34] base pinned — main@c964d8d
- [러너 07:34] autonomy release — 

## 정찰 노트
- 선택: RCA 자원 태그의 클러스터별 조인. 실제 홈 API→UI 경로와 이름만 쓰는 양쪽 키를 확인했고, 인증/정책 변경보다 작고 낮은 위험이라 우선했다.
- 검증: build/vet 및 analyzer·proxy 테스트 통과(proxy 61.874초). 신규 재현 테스트·전체 테스트·실클러스터는 미실행; 결함은 코드 경로로 판단했다.
- 주의: 기존 TestAttachFindingResources의 inventory ClusterID 누락을 바로잡고 빈 ID wildcard·owner fallback·NodePressure 동시 수정을 피한다. 신규 아이디어는 quiet_hours 범위 검증과 홈 조회 상한 안내다.
- 회사 스킬 3종·Skill 도구는 미발견하여 전용 절차 미확인. 요청문 형식으로 brief/ideas/profile 작성; 과거 반려의 구체 내용은 미확인이다.
- [러너 07:38] scout done — 운영 홈 RCA 자원 태그를 클러스터별로 정확히 연결 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- AttachFindingResources 인덱스·조회 양쪽을 cluster 포함 rcaKey로 변경해 운영 홈의 동명 Pod 자원 태그 혼합을 해결했다. 지정 3개 파일만 수정, 커밋 7a66433.
- 재현: 신규 analyzer 및 실제 SQLite·Server.Routes·HTTP 홈 회귀가 수정 전 동일 태그 오염으로 실패, 수정 후 통과했다. 입력 역순·빈 ID·kind 대소문자·namespace/name 구분·nil·Deployment template도 검증했다.
- 검증: go test ./internal/analyzer ./internal/proxy(proxy 61.521초) → go build ./... → go vet ./... → go test ./... 모두 통과; 수정 파일 gofmt 및 git diff --check 통과.
- 확신 없는 곳·검증 못 한 것: 실 Kubernetes·PostgreSQL·브라우저 렌더링 미검증. 회사 technology 스킬 3종/Skill 도구는 미발견하여 전용 절차·반환 형식 미확인; 로컬 superpowers 디버깅·TDD 절차 참고.
- 일부러 하지 않은 것: NodePressure·owner fallback·수량 파서·UI·auth·정책·스키마·릴리즈 변경은 과제 범위 밖이라 제외했다.
- 다음 역할 주의: HTTP 회귀는 임시 SQLite를 실제로 사용하며 외부 Kubernetes는 필요 없다. 배열/DB 반환 순서에 의존하지 않고 cluster_id별 네 자원 필드를 검증한다.
- [러너 07:43] brief accepted — 채택 — 현재 코드의 ClusterID 누락과 홈 API의 잘못된 자원 태그를 실제 회귀로 확인하여 지정된 3개 파일만 수정했다.
- [러너 07:44] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 변경 3개 파일과 수집·DB·홈 API·UI 경로를 확인했고 실제 차단 결함은 찾지 못했다.
- 신규 analyzer·SQLite HTTP 회귀를 -count=1로 실행해 통과, diff --check 통과. 기존 키로는 서로 다른 클러스터 기대값을 동시에 만족할 수 없음을 확인했다.
- 수정 전 재실행·전체 테스트·build·vet 및 실 Kubernetes·PostgreSQL·브라우저 검증은 이번 리뷰에서 하지 않았다.
- 스키마·인증·외부 상태 변경 없이 revert 가능. NodePressure·홈 조회 상한은 별도 과제로 유지한다.
- [러너 07:45] review approved — 리뷰 승인 (risk=low)
- [러너 07:45] pr created — https://github.com/hkjang/clustara/pull/26
- [러너 07:46] ci passed — 검사 없음 — 정책으로 허용
- [러너 07:46] merge done — 7a66433

## 릴리즈 노트
- v0.9.287 로컬 릴리즈 준비 완료: 8b05683, 기존과 같은 경량 태그. 원격 푸시·업로드 없음.
- go build ./..., CLI 빌드, go vet ./..., go test ./... 및 버전·산출물 이름 게이트와 diff 검사 통과.
- scripts/release.sh로 linux/amd64 자산 3종 생성; 지정 assets 경로에 복사 후 SHA256·manifest·가이드 검증. release.json과 release-notes.md 확정.
- 실 Kubernetes·PostgreSQL·브라우저·운영 배포 점검 미실행. 요청 회사 스킬·Skill 도구 미발견.
- [러너 07:51] release published — v0.9.287
- [러너 07:51] gh-release created — GitHub Release v0.9.287
- [러너 07:51] manifest ok — clustara-v0.9.287.tar.gz clustara-v0.9.287.tar.gz.sha256 README-offline-v0.9.287.md 
- [러너 07:51] assets uploaded — 3개
- [러너 07:51] assets verified — v0.9.287 자산 3개 (이전 v0.9.286: 3)
