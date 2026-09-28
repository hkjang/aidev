## seccomp 을 설정하지 않은 워크로드가 "restricted" 로 분류돼 포스처 표·DW·Deny 게이트에서 전부 빠졌습니다

`restrictedProfileViolations` — 파드 시큐리티 포스처 표와 `enforce_pss_restricted` Deny 게이트가
공유하는 헬퍼 — 는 PSS Restricted 통제 중 runAsNonRoot·allowPrivilegeEscalation·capabilities
drop ALL 만 보고 `securityContext.seccompProfile` 을 전혀 읽지 않았습니다. Restricted 는 유효
seccomp 타입이 `RuntimeDefault` 또는 `Localhost` 여야 하는데, seccomp 를 설정하지 않은 워크로드는
헬퍼가 아는 세 통제를 모두 만족해 `restricted` 로 분류됐습니다. 그 등급은 화면과 DW 가 행을
**버리는 데** 쓰는 값이라, 하드닝이 가장 덜 된 워크로드가 정확히 아무에게도 보이지 않았습니다.
이번 릴리즈는 그 통제를 검사에 추가합니다.

### PSS Restricted · `seccompProfile` 을 검사합니다

`Level="restricted"` 는 파드 시큐리티 표(`admin_ui.go` 의 `filter(p => p.level !== 'restricted')`)와
DW 내보내기(`admin_k8s_dw.go` 의 `if p.Level != "restricted"`)가 행을 제외하는 기준입니다. seccomp
미설정·`Unconfined` 워크로드가 그 등급을 받았기 때문에 포스처 화면에도, 데이터 웨어하우스에도
나타나지 않았고 Deny 게이트도 그대로 통과했습니다. 같은 헬퍼를 화면과 게이트가 공유하는 설계
자체는 의도된 것이지만(둘이 서로 다른 판정을 하지 않도록), 그래서 통제 하나가 빠진 영향도 세
표면에 동시에 나타났습니다.

이제 파드의 `securityContext.seccompProfile.type` 을 `podRunAsNonRoot` 와 같은 자리에서 한 번 읽고,
`SecurityRelevantContainers(ps)` 루프 안에서 컨테이너가 선언한 타입이 있으면 그것을, 없으면 파드
값을 유효 타입으로 삼습니다 — runAsNonRoot 와 같은 선행 규칙입니다. 유효 타입이
`RuntimeDefault`·`Localhost` 가 아니면 위반을 문제가 된 컨테이너 이름과 함께 적습니다
(`seccompProfile 미설정` / `seccompProfile=<값>`). `localhostProfile` 값 자체는 PSS 규격이 요구하지
않으므로 검사하지 않습니다.

### 이번 릴리즈는 신호가 바뀝니다 — 올리기 전에 확인하십시오

의도된 변화이지만 조용하지 않습니다. 헬퍼를 Deny 게이트가 공유하므로:

- seccomp 을 설정하지 않은 워크로드가 `enforce_pss_restricted` 에 **새로 막힙니다.**
- 그런 파드가 restricted 에서 baseline 으로 내려가는 만큼 **포스처 점수가 떨어집니다**
  (`summarize` 가 Baseline 하나당 2점을 뺍니다).

적용 폭은 클러스터의 seccomp 미설정 비율에 그대로 비례합니다. 그 게이트를 Deny 로 켜 둔 환경은
올리기 전에 `/admin/k8s/security` 에서 새로 baseline 으로 내려오는 워크로드를 먼저 확인하십시오 —
비율이 높으면 워크로드에 `seccompProfile: RuntimeDefault` 를 적용한 뒤에 올리는 것이 안전합니다.

알림은 늘지 않습니다: `k8s_notify.go` 는 `level == "privileged"` 만 알립니다. DB 스키마 변경은
없으며, 중복 제거 윈도우·담당팀 채널 라우팅·딥링크 동작은 그대로입니다.

### 검증

신규 테스트 2개(표 기반 8케이스 + 컨테이너별 위반 귀속)를 수정 전 코드에 먼저 붙여
`TestRestrictedProfileChecksSeccompProfile` 이 `level "restricted", want "baseline"` 으로 실패하는
것을 확인했습니다. 손으로 만든 대역 없이 `AnalyzeSecurity`+`deployWithPodSpec`+`evalRule` 로 실제
`store.K8sInventoryItem` 을 씁니다. 파드↔컨테이너 선행 규칙은 컨테이너의 `Unconfined` 가 파드의
`RuntimeDefault` 를 덮는 쪽과 그 반대쪽을 모두 단언합니다. 인과 확인으로 seccomp 블록만 다시 빼서
같은 테스트가 다시 빨강이 되는 것도 보았습니다.

`go build ./...`, `go vet ./...`, `go test ./... -count=1`(20개 패키지 전부 통과)과 버전 일치·산출물
이름 릴리즈 게이트를 실행했습니다. 프로덕션 파일은 `internal/analyzer/security.go` 1개입니다. 실
Kubernetes 클러스터·PostgreSQL·브라우저 렌더링은 미검증입니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.292.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.292.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.292.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.292.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.292.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.292
```
