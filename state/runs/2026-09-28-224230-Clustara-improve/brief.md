- 과제: PSS Restricted 검사에 seccompProfile 항목 추가 (가치 3 / 위험 2 / 작업량 S)

- 왜: `internal/analyzer/security.go:357 restrictedProfileViolations` 는 Restricted 통제 중 세 가지(runAsNonRoot·allowPrivilegeEscalation·capabilities drop ALL)만 검사하고 `seccompProfile` 을 전혀 보지 않는다(저장소 전체에 analyzer 쪽 seccomp 코드 0건 — `grep -rn seccomp internal/` 는 `internal/proxy` 의 매니페스트 생성기와 buildmanifest 안내 문구만 나온다). PSS Restricted 는 파드 또는 컨테이너의 `securityContext.seccompProfile.type` 이 `RuntimeDefault` 또는 `Localhost` 여야 하므로, seccomp 미설정·`Unconfined` 워크로드는 지금 `Level="restricted"` 로 분류되고 — 그 등급은 `internal/proxy/admin_ui.go:13460` 의 `filter(p => p.level !== 'restricted')` 와 `internal/proxy/admin_k8s_dw.go:82` 의 `if p.Level != "restricted"` 가 **화면·웨어하우스에서 행 자체를 버리는 데 쓰는 값**이다. 즉 이 제품이 스스로 생성하는 매니페스트(`admin_k8s_agent.go:426`, `admin_k8s_service_backup.go:406` 등)는 `seccompProfile` 을 넣는데, 남의 클러스터에서 그것이 빠진 워크로드는 포스처 표에 아예 보이지 않고 `enforce_pss_restricted` Deny 게이트(`internal/analyzer/policy.go:185`)도 통과한다. 고치면 seccomp 미적용 워크로드가 Baseline 으로 내려와 표·DW·게이트에 드러난다.

- 수용 기준:
  1) 파드에도 컨테이너에도 `seccompProfile` 이 없는 파드는 `restrictedProfileViolations` 가 컨테이너별 위반 문자열을 내고, `classifyPodSecurity` 의 `Level` 이 `"restricted"` 가 아니라 `"baseline"` 이 된다(다른 조건이 모두 하드닝돼 있을 때).
  2) 선행 규칙이 정확히 지켜진다 — 파드 레벨 `spec.securityContext.seccompProfile.type: RuntimeDefault` 만 있고 컨테이너가 아무것도 선언하지 않으면 **위반 아님**; 컨테이너가 `Unconfined` 로 덮으면 파드가 `RuntimeDefault` 여도 **그 컨테이너만** 위반; 파드가 `Unconfined` 인데 컨테이너가 `RuntimeDefault` 면 위반 아님. `Localhost` 는 허용값이며 (`localhostProfile` 값 자체는 검사하지 않는다).
  3) 같은 헬퍼를 쓰는 `enforce_pss_restricted` Deny 룰(`policy.go:185`)이 seccomp 미설정 파드에 대해 `Violated=true` 가 되고 `Detail` 에 `seccompProfile` 이 들어간다 — 화면과 게이트가 같은 말을 한다는 기존 계약 유지.
  4) 신규 회귀 테스트가 **수정 전 코드에서 빨강**임을 확인한 기록이 있을 것(실패 출력 인용). 기존 `TestPSSRestrictedRuleChecksTheRestrictedControls` 와 `TestUnhardenedPodIsNotClassifiedRestricted` 는 `hardenedContainer()` 에 `seccompProfile` 을 추가한 뒤 그대로 초록이어야 한다.

- 건드릴 파일 (프로덕션 1개 + 테스트 2개):
  - `internal/analyzer/security.go:restrictedProfileViolations` — 파드 `securityContext.seccompProfile.type` 을 `podRunAsNonRoot` 처럼 루프 밖에서 한 번 읽고, `SecurityRelevantContainers(ps)` 루프 안에서 컨테이너 `securityContext.seccompProfile.type` 이 있으면 그것을, 없으면 파드 값을 유효 타입으로 삼아 `RuntimeDefault`/`Localhost` 가 아니면 위반을 추가한다. 메시지는 기존 한국어 스타일에 맞춘다(예: `cname+": seccompProfile 미설정"`, `cname+": seccompProfile=Unconfined"`). 함수 위 주석 "It is shared by the posture report and the enforce_pss_restricted guardrail" 는 그대로 유효하니 유지하고, 왜 이 항목이 빠져 있었는지·무엇이 달라지는지를 기존 주석 톤(무엇이 잘못 보였는가)으로 한두 문장 덧붙인다. 컨테이너 열거는 `SecurityRelevantContainers`(containers+initContainers+ephemeralContainers)를 **그대로** 쓴다 — 같은 함수의 다른 세 검사와 다르게 굴면 안 된다.
  - `internal/analyzer/policy_restricted_test.go:hardenedContainer` — `securityContext` 에 `"seccompProfile": map[string]any{"type": "RuntimeDefault"}` 추가. 헬퍼의 doc 주석이 "satisfies every Restricted control this package checks" 라고 선언하므로 이 갱신이 정직한 수정이다. **주석도 함께 손보지 말 것 — 문구가 이미 맞다.**
  - 신규 테스트 파일(예: `internal/analyzer/security_seccomp_test.go`) — 위 수용 기준 1~3의 표 기반 케이스. `AnalyzeSecurity` + `deployWithPodSpec`(`policy_restricted_test.go` 에 있음)와 `evalRule`(같은 패키지 헬퍼)을 쓰고, 손으로 만든 대역이 아니라 `store.K8sInventoryItem` 실제 타입으로 구성한다.

- 검증 명령 (이 저장소에서 실제로 도는 것, 순서대로):
  1. `go test ./internal/analyzer -run 'PSS|Restricted|Security|Seccomp' -count=1` (현재 0.01s 수준)
  2. `gofmt -l internal/analyzer/security.go internal/analyzer/policy_restricted_test.go <신규 테스트 파일>` — 출력이 비어야 함
  3. `go build ./...` → `go vet ./...`
  4. `go test ./... -count=1` (전체 약 80s, `internal/proxy` 단독이 약 60s를 차지)
  5. 인과 확인: seccomp 검사 블록만 다시 빼서 신규 테스트가 다시 빨강이 되는 것을 보고 복원(이 저장소의 최근 4회차가 쓴 절차).

- 위험과 피할 것:
  - **`enforce_pss_restricted` 는 Deny 게이트**(`analyzer/policy.go:163-190`)와 공용이라, 이 변경은 게이트를 더 강하게 만든다. 이것은 의도된 방향이다 — 같은 줄의 기존 주석이 이 룰을 `deny_privileged_runtime` 의 별칭에서 실제 Restricted 검사로 끌어올린 전례를 남겨 뒀다. 그래도 커밋 메시지에 "seccomp 미설정 워크로드가 새로 Deny 대상이 된다" 를 명시할 것.
  - **점수·카운트가 움직인다**: `summarize`(`security.go:160-189`)가 `Baseline * 2` 를 감점하므로 seccomp 미설정 파드가 restricted→baseline 으로 내려오면 포스처 점수가 내려간다. 이건 정직한 신호 변화지만, 점수 기대값을 고정한 테스트가 있으면 같이 갱신해야 한다(`go test ./internal/analyzer` 로 드러난다).
  - **알림은 늘지 않는다 — 확인함**: `internal/proxy/k8s_notify.go:271` 이 `if p.Level != "privileged" { continue }` 이므로 restricted→baseline 이동은 Mattermost 알림을 만들지 않는다. 알림 경로를 같이 건드리지 말 것.
  - `classifyPodSecurity` 의 priv/baseline 버킷 분류(hostPath·hostPort·capability)는 **이번 범위 밖**이다(보류 아이디어의 3/3/M 항목). `restrictedProfileViolations` 만 고치고 `switch` 는 건드리지 말 것.
  - 문서 수정 불필요 — `docs/ADMIN_GUIDE.md:332,343` 은 등급 이름과 룰 목록만 적고 개별 Restricted 통제를 열거하지 않는다. `internal/proxy/docs_reference_test.go`·`release_gate_test.go` 가 문서의 API 경로·버전 마커를 검사하므로 **무관한 문서/버전 편집 금지**.
  - gofmt 는 손댄 파일에만. `.github/`·`store/sqlstore.go` DDL·인증 경로(`currentAccessClaims`·keycloak)는 건드리지 않는다.
  - 미확인: 실 Kubernetes·PostgreSQL·브라우저 렌더링은 이번 정찰에서 검증하지 않았다. `localhostProfile` 경로 존재 여부 검사를 요구할지는 PSS 규격상 불필요하다고 판단했으나 운영 요구는 미확인.

- 차선 후보: notify scan 의 events(500)·revisions(1000) 조회 상한이 여전히 무보고 (3/2/S) — `internal/proxy/k8s_notify.go:157-158`. v0.9.290 이 인벤토리에만 `truncated`/`truncation_notice` 를 붙였고 이벤트·리비전은 조용히 잘린다. 다만 이벤트는 kind 필터 축이 달라(`involved_kind`) 인벤토리와 같은 방식을 그대로 복사할 수 없다. 1순위가 성립하지 않으면 이쪽을 "상한 도달 보고만" 으로 좁혀서 할 것.
