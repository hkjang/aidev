# 회차 노트 2026-09-28-224230-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:42] base pinned — main@97ac287
- [러너 22:42] autonomy release — 

## 정찰 노트
- 최근 5회차가 전부 `internal/proxy/k8s_notify.go` 안에서 돌았다. 남은 notify 후보(dedup 키 Kind = 가치 2, events/revisions 상한 = 이벤트 필터 축이 달라 설계 선행)보다, 검사 자체에 통제 하나가 통째로 빠진 `restrictedProfileViolations` 의 seccompProfile 누락이 가치가 높고 파급이 좁아(프로덕션 파일 1개) 그쪽을 골랐다. podsec dedup 키는 3회 연속 1순위로 적혔다가 3회 모두 밀렸으므로 ideas.json 에 "다음 정찰은 1순위로 적지 말 것" 을 남겼다.
- 코드에서 직접 확인한 것: analyzer 패키지에 seccomp 코드 0건, `admin_ui.go:13460`·`admin_k8s_dw.go:82` 가 level==restricted 행을 버림, `k8s_notify.go:271` 이 privileged 만 알림(→ 이 변경으로 알림은 안 늘어남), `summarize` 의 Baseline*2 감점, `hardenedContainer()` 에 seccompProfile 없음, `go test ./internal/analyzer` 현재 초록.
- 추측으로 적은 것(미확인): 실제 운영 클러스터의 seccomp 미설정 비율 — 높으면 Deny 게이트가 갑자기 많이 막고 포스처 점수가 눈에 띄게 떨어진다. `localhostProfile` 값까지 검사할지도 운영 요구 미확인(PSS 규격상 불필요하다고 판단해 범위에서 뺌).
- 구현자가 조심할 것: `classifyPodSecurity` 의 priv/baseline `switch` 는 같이 건드리지 말 것(별도 3/3/M 과제). 파드→컨테이너 선행 규칙은 양방향 케이스를 모두 테스트할 것(컨테이너 Unconfined 가 파드 RuntimeDefault 를 덮는 쪽, 그 반대쪽 둘 다).
- 프로필은 5일 전 것이 지금 코드와 어긋나지 않아 새로 쓰지 않았다.
- [러너 22:46] scout done — PSS Restricted 검사에 seccompProfile 항목 추가 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- `restrictedProfileViolations`(security.go)에 seccompProfile 통제를 추가했다 — 파드 type 을 루프 밖에서 한 번 읽고, 컨테이너가 선언한 type 이 있으면 그것이 이기는 runAsNonRoot 와 같은 선행 규칙. `RuntimeDefault`/`Localhost` 외에는 위반(미설정/`=<값>` 두 메시지). 프로덕션 파일 1개, 커밋 1bfa039.
- 확신 없는 곳·검증 못 한 것: (a) 실 운영 클러스터의 seccomp 미설정 비율 — 높으면 `enforce_pss_restricted` Deny 가 갑자기 많이 막고 포스처 점수가 눈에 띄게 떨어진다(`summarize` 의 Baseline*2). 이 저장소에서는 확인 불가. (b) 실 Kubernetes·PostgreSQL·브라우저 렌더링 미검증(SQLite+httptest 만). (c) 과제서에 없던 파일 하나를 더 고쳤다 — `security_test.go:TestAnalyzeSecurityPodSecurityLevels` 의 "hardened" fixture. 이것은 과제서가 예고한 "등급 기대값을 고정한 테스트" 이며, 고치지 않으면 `expected 1 privileged + 1 restricted, got {... Restricted:0 ...}` 로 빨강이었다.
- 일부러 하지 않은 것: `localhostProfile` 값 검사(PSS 규격상 type 만 요구), `classifyPodSecurity` 의 priv/baseline switch(별도 3/3/M), 문서·버전·알림 경로 수정(알림은 `k8s_notify.go:271` 이 privileged 만 보므로 안 늘어난다 — 전체 테스트 통과로 확인).
- 다음 역할이 조심할 것: 이 변경은 포스처 표시와 Deny 게이트를 동시에 움직인다. 새 테스트는 DB 불필요(analyzer 순수 함수, 0.02s). 인과 확인으로 seccomp 블록만 빼서 다시 빨강 → 복원까지 했다. `go test ./... -count=1` 20 패키지 전부 ok(proxy 73.5s).
- [러너 22:50] brief accepted — 채택 — analyzer 에 seccomp 코드가 0건이라는 근거와 `hardenedContainer()` 에 seccompProfile 이 없다는 근거가 현 코드와 정확히 맞�
- [러너 22:51] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인: diff 전체(프로덕션 1 + 테스트 3), `restrictedProfileViolations`·`classifyPodSecurity`·`policy.go:163-188` 문맥, 소비자(`admin_ui.go:13460`, `admin_k8s_dw.go:82`), 헬퍼 nil 안전성, docs 의 PSS 서술. 내가 직접 돌린 게이트: `go build`·`go vet`·`go test ./... -count=1` 전부 초록, 수정 파일 gofmt 깨끗. 원장의 `- 실패 재현:` 줄이 있고 증상과 일치하며, 새 테스트 첫 케이스가 violates=true 를 요구하므로 수정 전 코드에서 반드시 빨강이다. → **approve, 차단 없음**.
- 못 본 것: 실 Kubernetes·PostgreSQL·Keycloak·브라우저, 실 운영 클러스터의 seccomp 미설정 비율.
- 승인이어도 남는 우려 ①(릴리즈 노트용): 주석(`security.go:358-363`)의 "every pod nobody has hardened yet" 은 틀렸다. `security.go:383` 의 allowPrivilegeEscalation 검사에 파드 fallback 이 없어, 기존에 restricted 로 잘못 표시된 것은 컨테이너 레벨 APE=false+drop ALL+runAsNonRoot 를 이미 갖춘 **거의 하드닝된** 파드뿐이다. 코드는 옳고, 이 사실은 구현 노트의 "Deny 가 갑자기 많이 막음" 걱정을 오히려 줄인다 — 릴리즈 노트는 주석이 아니라 이 실제 대상 집합으로 쓰라.
- 우려 ②: `policy_iac.go:118-122` 의 Kyverno/Rego export 에 seccompProfile 이 없어, 내보낸 ClusterPolicy 는 Clustara 게이트가 이제 막는 파드를 통과시킨다(기존 격차가 한 칸 더 벌어짐). 우려 ③: PSS 는 `spec.os.name: windows` 를 Seccomp 등에서 면제하지만 이 헬퍼는 무조건 적용한다(기존 3개 통제도 동일). 둘 다 다음 회차 후보.
- 우려 ④(사소): 파드 레벨 Unconfined 가 컨테이너 전부를 이름으로 지목한다 — 같은 파일의 "(Pod 설정을 덮어씀)"·"(Pod securityContext 상속)" 표기와 어긋난다.
- [러너 22:57] review approved — 리뷰 승인 (risk=low)
- [러너 22:57] pr created — https://github.com/hkjang/clustara/pull/32
- [러너 22:57] ci passed — 검사 없음 — 정책으로 허용
- [러너 22:57] merge done — 1bfa039
- [러너 23:10] release published — v0.9.292
- [러너 23:10] gh-release created — GitHub Release v0.9.292
- [러너 23:10] manifest ok — clustara-v0.9.292.tar.gz clustara-v0.9.292.tar.gz.sha256 README-offline-v0.9.292.md 
- [러너 23:10] assets uploaded — 3개
- [러너 23:10] assets verified — v0.9.292 자산 3개 (이전 v0.9.291: 3)
