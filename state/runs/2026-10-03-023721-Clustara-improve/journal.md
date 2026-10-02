# 회차 노트 2026-10-03-023721-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:37] base pinned — main@758bd7e
- [러너 02:37] autonomy release — 

## 구현 노트

- 무엇을 왜: `classifyPodSecurity` 가 Baseline 이 금지하는 통제(hostPath·hostPort·허용목록 밖 capability)를 Restricted 전용 실패와 같은 버킷에 모아 `Level="baseline"` 으로 적어, `k8s_notify.go:308` 의 `level=="privileged"` 필터에 걸려 호스트 마운트 DaemonSet·SYS_ADMIN 컨테이너 알림이 한 번도 안 갔다. 버킷을 셋으로 쪼개고 등급 매핑을 PSS 의미("그 파드를 허용하는 가장 약한 프로파일")로 고쳤다. `runAsUser=0` 은 Baseline→Restricted 버킷으로 옮겨야 한다 — 안 옮기면 root 워크로드 전부가 privileged 로 올라가 알림이 폭주한다. 프로덕션 파일 1개(internal/analyzer/security.go).
- **확신 없는 곳·검증 못 한 것**: ① 실환경 알림 증가 폭을 모른다 — hostPath 는 로그·모니터링 DaemonSet 에서 흔하므로 첫 스캔에서 알림이 몰릴 수 있다(워크로드당 6시간 dedup 1건). 폐쇄망 실 클러스터로 재어 보지 못했다. ② `admin_ui.go` 의 Pod Security 표/KPI 문구는 손대지 않았다 — 등급별 개수가 privileged 쪽으로 이동하는데 화면 설명 문구가 그 의미를 설명하는지 브라우저로 못 봤다. ③ 포스처 점수(`summarize`: privileged 8점·baseline 2점)가 더 떨어지는 폭도 미측정. ④ 실 PostgreSQL·실 Kubernetes·ClickHouse·브라우저는 전부 미검증(SQLite+httptest 만).
- 일부러 안 한 것: `runtimesecurity.go:ScorePodSecurity`(hostPath→baseline) 는 그대로 뒀다 — 위험 점수 휴리스틱이라 PSS 분류와 계약이 다르고, 운영자 지시가 "계약 다른 파서 통합 금지" 다. 결과로 `/admin/k8s/security` 와 `/admin/k8s/runtime-security` 가 hostPath 파드의 프로파일을 다르게 표시한다(문서·ideas.json 에 명시). `k8s_notify.go` 의 알림 문구("Privileged 워크로드")도 안 고쳤다 — 등급 의미상 맞고 위반 목록이 함께 나가며, 이번 회차에 이 파일을 쉬게 하려 했다.
- 다음 역할이 조심할 것: `internal/proxy/k8s_notify_podsec_level_test.go` 는 t.TempDir SQLite + httptest Mattermost webhook 이 필요하다(DB 파일 쓰기·로컬 포트). `DaemonSet` 을 쓴 이유는 `AnalyzeRCA` 를 조용히 유지하려는 것 — `Deployment` 로 바꾸면 status 없는 객체가 RCA 알림을 만들어 `sent` 카운트가 오염된다(처음 작성 때 실제로 겪었다). 등급 매핑을 되돌리는 변경은 analyzer 표 테스트 3케이스 + proxy 종단 1건이 즉시 빨강이 된다.
- [러너 03:06] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인한 것: `git diff main...HEAD` 전문, `classifyPodSecurity`/`restrictedProfileViolations` 전체, Level 소비자 4곳(k8s_notify.go:308 · admin_k8s_dw.go:82 · summarize 점수 · admin_secops.go:122), 원장의 `- 실패 재현:` 줄(실재·증상 일치), build·vet·analyzer 전체·신규 proxy 2개·release/docs 게이트 모두 통과. 작업 트리는 안 건드렸고 `go test -overlay` 로 main 과 HEAD 를 같은 입력에 돌려 비교했습니다.
- 거절 사유(수리가 먼저 볼 파일 `internal/analyzer/security.go:327-330`): capability 검사가 Restricted 의 허용 목록(NET_BIND_SERVICE 하나)을 쓰는데 이번 변경이 그 위반을 `failsBaseline` 로 옮겨, PSS Baseline 이 허용하는 기본 capability(CHOWN·SETUID·SETGID·KILL·DAC_OVERRIDE 등 12개)를 추가한 워크로드가 privileged 로 올라갑니다. overlay 재현: `add:[CHOWN]` 만 다른 하드닝 Pod 가 main=baseline → HEAD=privileged, 점수 90→72. 구현자가 runAsUser=0 에서 피한 오탐 페이지가 capability 쪽으로 들어왔습니다.
- 둘째 사유(`security.go:399-413`, `security_pss_level_test.go:98`): 명시적 `seccompProfile=Unconfined` 는 PSS 에서 Baseline 실패인데(미설정만 Restricted 전용) 둘을 한 버킷에 넣어 Unconfined 파드는 여전히 baseline·무알림입니다. 새 테스트가 그 틀린 의미를 정답으로 고정합니다. 문서(`K8S_OPERATIONS_HUB.md:7-37`)도 두 지점을 코드와 다르게 설명합니다.
- 못 본 것: 실 Kubernetes·실 PostgreSQL·ClickHouse·브라우저(admin_ui 의 Pod Security 표·KPI 문구는 읽기만 했고 렌더링 미확인), 실 클러스터에서의 알림 증가 폭. 구현 노트의 미검증 항목 ①②③은 여전히 미검증입니다.
- 승인으로 갈 경우라도 남는 우려: hostProcess·sysctls·/proc mount·AppArmor·SELinux 등 나머지 Baseline 통제는 미검사라 "등급 = 가장 약한 허용 프로파일" 계약이 아직 부분적입니다. 차단 부서는 없습니다(공격 경로·개인정보·라이선스 변경 없음, DW fact 는 추가 전용이라 revert 안전).
- [러너 03:11] review rejected — 리뷰 거절: internal/analyzer/security.go:327-330 capability 검사는 Restricted 의 허용 목록(NET_BIND_SERVICE 하나)을 쓰는데, 이번 변경이 그 위반을 `failsBaseline` 버킷으�

## 수리 노트
- 지적 3개 전부 맞았습니다. capability 는 Restricted 허용 목록으로 Baseline 을 판정했고(=`add:[CHOWN]` 하드닝 Pod 가 privileged), seccomp 는 명시적 `Unconfined`(Baseline 실패)와 미설정(Restricted 전용)을 한 버킷에 넣어 Unconfined Pod 가 baseline·무알림으로 남았습니다. 비평가가 적은 overlay 증상을 그대로 재현했고, 수리 후 overlay 로 되돌려 신규 단언이 전부 다시 빨강이 되는 것까지 확인했습니다(CHOWN `sent:1`, Unconfined `sent:0`). 틀린 지적은 없었습니다.
- 고친 방법: `baselineCapabilities`(PSS Baseline 기본 13개) 집합을 추가해 목록 안 add → `failsRestricted`, 목록 밖 → `failsBaseline`. 공용 헬퍼는 `containerProfileViolations`(통제별 `failsBaseline` 태그)로 바꾸고 `restrictedProfileViolations` 가 평탄화 래퍼로 남아 Deny 게이트가 보는 문자열·순서는 불변 — 프로파일은 누적이라 게이트는 양쪽을 다 봐야 하기 때문입니다.
- 테스트는 지우지 않았습니다. `security_pss_level_test.go` 의 Unconfined 기대값을 privileged 로 고치고 4케이스(allow-list 안 capability, seccomp 미설정, pod 상속, container 덮어쓰기)를 추가. main 에 이미 있던 `security_seccomp_test.go` 가 `wantLevel` 을 `violates` 에서 파생해 "모든 seccomp 실패=baseline" 을 단정하고 있었으므로 케이스별 명시로 바꿨습니다(Restricted 위반·detail 단언은 유지). proxy 종단 3케이스 추가.
- 여전히 확신 없는 곳: ① 실 클러스터 알림 증가 폭 미측정 — 범위가 hostPath·hostPort·목록 밖 capability·명시적 Unconfined 네 가지로 줄었으니 이전보다 좁지만 hostPath DaemonSet 은 그대로 올라옵니다. ② `admin_ui.go` Pod Security 표·KPI 문구 브라우저 미확인. ③ hostProcess·sysctls·/proc mount·AppArmor·SELinux 등 나머지 Baseline 통제 미검사(의도적 범위 외) — 등급 계약은 아직 부분적입니다. ④ seccomp `default` 분기(API 가 허용하지 않는 제3의 값)는 Restricted 전용으로 두었습니다 — PSS 가 이름 붙이지 않은 값을 privileged 로 올리지 않으려는 보수적 선택입니다.
- [러너 03:20] repair done — # 수리 요약 (7fedbb3)  비평의 세 지적 모두 맞았고 전부 고쳤습니다. (1) capability 검사가 Restricted 허용 목록(NET_BIND_SERVICE)으로 Baseline 을 판정해 `add:[CHOWN]` 

## 비평 노트
- 확인한 것: `git diff main...HEAD` 전문, `classifyPodSecurity`/`containerProfileViolations`/`restrictedProfileViolations`, Level 소비자 전수(k8s_notify.go:307 · admin_k8s_dw.go:81 · summarize:162 · admin_secops.go:121,139 · admin_ui.go:13414/13460/13479 · policy.go:186), PSS 원문 대조(Baseline 허용 add 13개 정확히 일치). `go test -overlay` 로 main 의 security.go 를 끼워 신규 테스트가 실제로 빨강인지 확인: analyzer 8케이스 `baseline, want privileged`, proxy 2건 `sent:0`. 수리 전 커밋 845060a 오버레이에서는 `TestNotifyScanStaysQuietForRestrictedOnlyFailures` 가 `CHOWN ... sent:1` 로 실패 — 1차 비평의 오탐 회귀도 핀으로 고정됐습니다. build·vet·`go test ./... -count=1` 전부 통과, docs/release 게이트 통과, 손댄 4개 파일 gofmt 클린. 판정 approve, 차단 부서 없음(공격 경로·개인정보·라이선스 변경 없음, DW fact 는 추가 전용이라 revert 안전).
- 승인이어도 남는 우려 ①: pod 레벨 `seccompProfile=Unconfined` 를 컨테이너가 `RuntimeDefault` 로 덮으면 `security.go:443-446` 이 위반 없음 → `restricted` 로 판정합니다. 실제 PSA 는 `spec.securityContext.seccompProfile.type` 을 필드 단위로 검사하므로 Baseline·Restricted 모두 거부합니다. main 과 동일한 기존 결함이라 거절하지 않았지만 `security_pss_level_test.go` 마지막 케이스와 docs 의 "컨테이너의 프로파일이 파드의 것을 덮으므로" 문장이 이 차이를 정답으로 고정했습니다 — 다음 회차가 PSS 계약을 넓힐 때 먼저 볼 자리입니다. 런타임 노출은 없습니다(컨테이너는 실제로 RuntimeDefault 로 뜸).
- 남은 우려 ②: hostProcess·sysctls·/proc mount type·AppArmor·SELinux, hostPath 외 Baseline 볼륨 타입 제한 미검사 → "등급 = 가장 약한 허용 프로파일" 계약은 여전히 부분적인데 문서는 그 미검사 목록을 적지 않았습니다. ③ `CAP_` 접두 표기(`CAP_CHOWN`)나 빈 문자열 capability 는 허용 목록에 없어 `privileged` 로 올라갑니다(`security.go:343-350`) — 매니페스트 관례상 드물지만 오탐 알림 경로입니다.
- 릴리즈 노트에 넣을 것: baseline→privileged 이동으로 (1) 해당 워크로드가 신규 알림(6시간 dedup, 1건/워크로드 — 로그·모니터링 DaemonSet 에 몰릴 수 있고 실 클러스터 증가 폭은 여전히 미측정), (2) 포스처 점수가 워크로드당 2→8점으로 더 떨어짐, (3) DW `security_finding.rule` 이 `pod-security-baseline`→`pod-security-privileged`·severity medium→high 로 바뀌어 rule 별 집계 대시보드에 단차, (4) `admin_secops.go:121` 추천 문구가 hostPath DaemonSet 에도 "privileged workload 제거 또는 예외 승인" 으로 뜸(등급상 맞지만 호스트 마운트를 설명하지는 않음).
- 못 본 것: 실 Kubernetes·실 PostgreSQL·ClickHouse·Keycloak·브라우저(SQLite+httptest 만). `admin_ui.go` Pod Security 표·KPI 는 세 등급을 일반적으로 처리해 하드코딩 파손은 없음을 코드로 확인했지만 렌더링은 미확인 — 구현 노트의 미검증 ①②③ 중 ②는 코드 수준에서만 해소, ①③ 은 그대로 남습니다.
- [러너 03:26] review approved — 리뷰 승인 (risk=medium)
- [러너 03:26] pr created — https://github.com/hkjang/clustara/pull/34
- [러너 03:27] ci passed — 검사 없음 — 정책으로 허용
- [러너 03:27] merge done — 7fedbb3
- [러너 03:34] release published — v0.9.294
- [러너 03:34] gh-release created — GitHub Release v0.9.294
- [러너 03:34] manifest ok — clustara-v0.9.294.tar.gz clustara-v0.9.294.tar.gz.sha256 README-offline-v0.9.294.md 
- [러너 03:34] assets uploaded — 3개
- [러너 03:34] assets verified — v0.9.294 자산 3개 (이전 v0.9.293: 3)
