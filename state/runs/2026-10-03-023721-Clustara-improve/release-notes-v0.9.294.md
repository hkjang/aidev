## Baseline 을 깨뜨린 워크로드가 "Baseline 만족" 으로 분류돼 알림이 한 번도 나가지 않았습니다

Pod Security Standards 의 등급은 파드가 **깨뜨린 통제의 목록**이 아니라 **그 파드를 그래도 허용하는
가장 약한 정책**의 이름입니다. 그래서 Baseline 통제를 하나라도 깨뜨린 파드는 다른 것을 아무리 잘
지켰더라도 Privileged 정책만이 허용합니다.

`classifyPodSecurity` 는 Baseline 이 스스로 금지하는 통제(hostPath 볼륨, `hostPort`, Baseline 허용
목록 밖의 capability)를 Restricted 가 추가로 요구하는 통제와 **같은 버킷**에 모아 놓고 결과를
`Level="baseline"` 으로 적었습니다 — 만족하지 않는 프로파일을 만족한다고 말한 것입니다.

이 등급은 장식이 아닙니다. `k8s_notify.go` 는 `level == "privileged"` 인 워크로드만 알리므로, 호스트
파일시스템을 마운트한 DaemonSet 이나 `SYS_ADMIN` 을 쥔 컨테이너는 **알림이 한 번도 나가지 않았고**,
DW 에는 Baseline 을 만족하는 워크로드의 낮은 심각도로 적혔습니다. 이번 릴리즈는 등급을 PSS 의
의미대로 매깁니다.

### Pod Security 등급 · "그 파드를 허용하는 가장 약한 프로파일" 을 가리킵니다

버킷을 세 개로 분리했습니다. 통제별로 어느 버킷에 들어가는지는 PSS 가 그 통제를 **어느 프로파일에서
금지하는지**로 정해집니다.

| 등급 | 버킷 | 통제 |
| --- | --- | --- |
| `privileged` | `hostLevel` | `hostNetwork`·`hostPID`·`hostIPC`, 컨테이너 `privileged=true` |
| `privileged` | `failsBaseline` | hostPath 볼륨, `hostPort`, **Baseline 허용 목록 밖**의 추가 capability, 명시적 `seccompProfile.type=Unconfined` |
| `baseline` | `failsRestricted` | `runAsNonRoot` 미설정/false, `allowPrivilegeEscalation!=false`, `capabilities` drop ALL 아님, `seccompProfile` **미설정**, `runAsUser=0`, **Baseline 허용 목록 안**의 추가 capability |
| `restricted` | — | 위 어느 것도 걸리지 않음 |

경계를 가르는 두 줄이 PSS 원문과 어긋나기 쉬우므로 명시합니다.

- **capability**: Baseline 은 기본 컨테이너 집합의 재추가를 허용합니다 —
  `AUDIT_WRITE`·`CHOWN`·`DAC_OVERRIDE`·`FOWNER`·`FSETID`·`KILL`·`MKNOD`·`NET_BIND_SERVICE`·
  `SETFCAP`·`SETGID`·`SETPCAP`·`SETUID`·`SYS_CHROOT`. 그래서 `CHOWN` 이나 `SETUID` 를 더한
  워크로드는 **`baseline`** 이고 알림 대상이 아닙니다. `SYS_ADMIN`·`NET_ADMIN` 처럼 목록 밖이면
  Baseline 실패이므로 `privileged` 입니다.
- **seccompProfile**: Baseline 통제는 "명시적으로 `Unconfined` 로 두지 말 것" 입니다(허용값
  미설정·`RuntimeDefault`·`Localhost`). 따라서 `Unconfined` 를 **명시한** 파드는 `privileged`,
  아무것도 **설정하지 않은** 파드는 Restricted 만 깨뜨려 `baseline` 입니다. 컨테이너의 프로파일이
  파드의 것을 덮으므로, 판정은 다른 컨테이너 통제와 같은 우선순위를 따릅니다.

`runAsUser=0` 은 이 변경에서 Baseline 버킷에서 Restricted 버킷으로 옮겼습니다 — Baseline 은 root
실행을 금지하지 않고 Restricted 가 `runAsNonRoot` 로 금지하며, root 는 하드닝하지 않은 워크로드의
기본 상태이므로 Baseline 실패로 셌다면 전 함대가 `privileged` 로 올라가 알림이 폭주합니다.

`Violations` 문자열과 `restrictedProfileViolations`(`enforce_pss_restricted` Deny 게이트 공용 헬퍼)의
판정 내용은 바뀌지 않았습니다 — 프로파일은 누적이므로 Baseline 이 금지하는 것은 Restricted 실패이기도
해서 Deny 게이트가 보는 목록은 그대로입니다.

### 올리기 전에 확인하십시오 — 이번 릴리즈는 신호가 바뀝니다

hostPath·`hostPort`·허용 목록 밖 capability·명시적 `seccompProfile=Unconfined` 워크로드가 `baseline`
에서 `privileged` 로 올라갑니다. 그 결과:

1. 그 워크로드가 **새로 알림을 받습니다** (dedup 창 6시간, 워크로드당 1건).
2. 포스처 점수가 더 떨어집니다 (`summarize` 는 Privileged 8점, Baseline 2점).
3. DW 의 `pod-security-*` fact 이름과 심각도가 그만큼 올라갑니다.

hostPath 는 로그·모니터링 DaemonSet 에서 흔하므로, 올리기 전에 `/admin/k8s/security` 에서 새로
privileged 로 올라오는 워크로드를 먼저 확인하십시오. 위 네 가지가 범위 전체입니다 — `CHOWN` 류
capability 추가나 seccomp 미설정만으로는 등급이 올라가지 않습니다.

`/admin/k8s/runtime-security`(CLU-OCP-03 `ScorePodSecurity`)는 PSS 분류가 아니라 위험 점수
휴리스틱이라는 다른 계약이므로 이번 변경에서 건드리지 않았고, 두 화면은 hostPath 파드의 프로파일을
다르게 표시합니다.

### 검증

신규 테스트 3개 — analyzer 등급 표 8케이스, 그리고 실제 SQLite·`kube.InventoryFromObject`·
`Server.Routes`·httptest Mattermost webhook 을 쓰는 proxy 종단 2개(손으로 만든 대역 없음) — 를 수정 전
코드에 먼저 붙여 빨강을 확인했습니다:

```
--- FAIL: TestPodSecurityLevelNamesTheWeakestAdmittingProfile
    security_pss_level_test.go:114: hostPath volume fails Baseline's volume-type restriction:
        level "baseline", want "privileged" (violations [hostPath volume])
    k8s_notify_podsec_level_test.go:63: a DaemonSet mounting the host root fails Pod Security
        Baseline and must be notified: map[... sent:0 ...]
```

인과 확인으로 등급 매핑 `switch` 한 블록만 되돌려 같은 테스트가 다시 빨강이 되는 것도 보았습니다.
`go build ./...`, `go vet ./...`, `go test ./...`(20 패키지 전부 ok, 실패 0) 및 버전 일치·산출물 이름
릴리즈 게이트를 통과합니다.

실 Kubernetes·PostgreSQL·실 Mattermost·브라우저 렌더링은 미검증입니다. 프로덕션 파일은
`internal/analyzer/security.go` 1개이며, DB 스키마·설정 변경은 없습니다.

### 배포 파일

| 파일 | 설명 |
|------|------|
| clustara-v0.9.294.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| clustara-v0.9.294.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.9.294.md | 오프라인 배포 가이드 |

### 빠른 시작

```bash
# 무결성 확인
sha256sum -c clustara-v0.9.294.tar.gz.sha256

# 이미지 로드
gunzip -c clustara-v0.9.294.tar.gz | docker load

# 실행
docker run -d --name clustara --restart=always \
  -p 9090:9090 \
  -v /opt/clustara/data:/data \
  -e UPSTREAM_BASE_URL=https://api.openai.com \
  -e UPSTREAM_API_KEY=sk-... \
  -e ADMIN_TOKEN=change-me \
  clustara:v0.9.294
```
