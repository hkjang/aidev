# 회차 노트 2026-09-20-185404-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:54] base pinned — main@10621cc
- [러너 18:54] autonomy release — 

## 정찰 노트
- RCA 이벤트·리비전 클러스터 격리를 선택: 장애 누락·잘못된 severity 상승을 한 과제로 해결하며 인증/PSS/스키마 변경이 없다. notify kind 충돌보다 영향이 크다.
- rcaKey는 workload.go도 공유한다. TestEnrichWithConfigChanges의 누락 ClusterID는 fixture에서 보정하고 빈 클러스터 wildcard로 회피하지 말 것.
- 실클러스터 재현·회사 스킬 절차는 미확인; 코드 근거와 예상 35분+예비 10분을 brief에 기록. 자원 태그/NodePressure 조인은 별도 후보로 범위를 제한했다.
- 프로필의 DB 설명을 기본 SQLite·PostgreSQL 지원으로 정정; 18개 아이디어를 유지/재평가했고 신규 3개 및 이전 해결 1개를 기록했다.
- [러너 18:59] scout done — RCA 이벤트·리비전 연결과 중복 제거를 클러스터별로 격리 (가치 4 / 위험 1 / 작업량 M)

## 구현 노트
- 158c620: RCA 이벤트·리비전 키와 직접 이벤트 필터를 클러스터별로 격리해 다른 클러스터 증적·severity 상승 및 probe 장애 누락을 수정했다.
- 지정한 5개 파일만 변경; 기존 config fixture 두 리비전에 c1을 채웠고 빈 ClusterID는 빈 값끼리만 매칭한다.
- 신규 회귀 6개는 수정 전 모두 실패(analyzer 5·proxy sent=1); 수정 후 단위/종단 테스트와 각 클러스터 링크·채널·재스캔 중복 억제 통과.
- 검증: go test ./internal/analyzer ./internal/proxy(proxy 58.627s), go build ./..., go vet ./..., 최종 go test ./..., 수정 파일 gofmt -l 및 git diff --check 모두 통과.
- 확신 없는 곳·검증 못 한 것: 실 Kubernetes·실 PostgreSQL·실 Mattermost는 미검증; 회사 스킬 3종/Skill 도구는 카탈로그와 로컬 검색에서 미발견하여 절차·반환 형식 미확인.
- 일부러 제외: 자원 태그/NodePressure 조인, notify inventory 범위·영구 dedup 키, severity·시간·이름 추정 정책, 인증·스키마·버전·릴리즈는 과제 범위 밖.
- 다음 역할 주의: proxy 종단 회귀는 t.TempDir SQLite와 httptest webhook을 사용하며 외부 DB/클러스터가 필요 없다. ideas.json 기존 18개를 유지하고 선택 과제만 done으로 갱신했다.
- [러너 19:04] brief accepted — 채택 — 공유 키와 직접 이벤트 필터의 클러스터 누락이 현 코드에서 재현되어 지정한 5개 파일만 수정했다.
- [러너 19:05] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- approve / low / blocking=[]: 변경 5개 파일과 공유 키·직접 필터·수집/저장/알림 경로를 확인했고 새 결함·범위 이탈·비가역 변경은 발견하지 못했다.
- 신규 회귀 6개의 실행·단언과 수정 전 실패 조건을 코드로 대조; analyzer/proxy 테스트 통과(proxy 58.502s), diff --check 통과. 수정 전 실행 및 전체 build/vet/test 재실행은 하지 않았다.
- 회사 스킬 3종/Skill 도구 미발견으로 해당 절차는 미확인; 실 Kubernetes·PostgreSQL·Mattermost는 미검증.
- 기존 자원 태그/NodePressure 조인 누락은 남는다. 릴리즈 설명은 RCA 이벤트·리비전 격리로 한정하고 후속 과제로 유지할 것.
- [러너 19:07] review approved — 리뷰 승인 (risk=low)
- [러너 19:07] pr created — https://github.com/hkjang/clustara/pull/25
- [러너 19:07] ci passed — 검사 없음 — 정책으로 허용
- [러너 19:07] merge done — 158c620
- [러너 19:13] release published — v0.9.286
- [러너 19:13] gh-release created — GitHub Release v0.9.286
- [러너 19:13] manifest ok — clustara-v0.9.286.tar.gz clustara-v0.9.286.tar.gz.sha256 README-offline-v0.9.286.md 
- [러너 19:13] assets uploaded — 3개
- [러너 19:13] assets verified — v0.9.286 자산 3개 (이전 v0.9.285: 3)
