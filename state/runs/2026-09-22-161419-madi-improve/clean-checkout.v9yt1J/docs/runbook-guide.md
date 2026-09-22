# 격리 실행 · Runbook 가이드

madi의 실행형 운영 절차는 문서를 **사내 AWX / Ansible 또는 전용 Kubernetes Job**에 연결합니다. madi 서비스 호스트에서는 shell, `os/exec`, Docker socket, 임의 SSH 명령을 실행하지 않습니다. 기본은 꺼짐이며 환경변수는 기존 네 개에서 늘어나지 않습니다.

## 사용자 흐름

1. 관리자가 `서비스 관리 → 격리 실행 · Runbook`에서 기능과 허용 실행기를 설정합니다.
2. 문서 메뉴의 `격리 실행 · Runbook`에서 목적·선행 조건·검증·롤백 기준을 작성합니다.
3. 실행 / 검증 / 롤백 탭별로 허용 작업을 선택하고 자료형에 맞는 매개변수를 저장합니다. 단계 순서는 위·아래 버튼으로 변경합니다.
4. `계획 준비`는 문서 버전, 절차 버전, 실행기 설정 버전, 정확한 허용 명령과 매개변수, 실제 외부 격리 설정을 고정합니다. 아직 외부 작업을 시작하지 않습니다.
5. 서비스 검토·승인이 켜져 있으면 관리자가 **Runbook 전용 승인 정책**을 명시해야 합니다. 별도 검토자의 승인 후 요청자가 직접 `EXECUTE 실행-ID`를 입력합니다. 검토·승인이 꺼져 있으면 검토 과정 없이 동일한 직접 확인만 수행합니다.
6. 실행 화면에서 단계 상태·원격 작업 ID·실시간 출력을 확인합니다. 동일 계획의 이중 실행은 차단됩니다. 문서·정책·권한·실행기 변경 후에는 새 계획이 필요합니다.
7. 중지는 `CANCEL 실행-ID`를 입력합니다. 중지 요청을 보냈다는 사실과 원격 중지 완료는 다릅니다. 완료를 확인하지 못하면 `외부 실행 확인 필요`로 표시합니다.

검증 단계가 성공한 경우, 검증했던 문서와 절차 버전이 그대로일 때만 마지막 검증 시간이 갱신됩니다. 롤백은 별도의 고정 계획·확인·선택적 승인을 거치는 실제 작업이며 자동으로 실행하지 않습니다.

## AWX / Ansible 설정

관리 화면에 HTTPS API 기본 주소, 최소 권한 Bearer 토큰, 필요하면 사내 CA PEM을 입력합니다. TLS 검증을 끄는 옵션은 없습니다. HTTP 리다이렉트와 환경 프록시를 따라가지 않습니다.

허용 작업마다 Job Template ID와 제한 시간, 실행 가능 역할 또는 팀, 매개변수를 등록합니다. AWX 쪽에서 다음을 충족해야 합니다.

- 고정 프로젝트와 동기화된 SCM revision. 실행 시 프로젝트 업데이트 및 브랜치 변경은 끕니다.
- Job Template에 양수 timeout을 지정하며 madi의 단계 제한 시간 이하여야 합니다.
- 템플릿에 Execution Environment를 명시하고 그 이미지를 `registry/image@sha256:...`로 고정합니다.
- `extra_vars` 실행 시 입력을 허용합니다. 사용자가 선언한 변수 외에 `madi_execution_id`, `madi_execution_step` 추적 값이 함께 전달됩니다. 이 예약 변수는 사용자가 덮어쓸 수 없습니다.
- 실행 시 인벤토리·자격 증명·SCM 브랜치 변경, 대화형 비밀번호 입력은 사용하지 않습니다.
- 인벤토리, 원격 계정, Playbook 내용, Container Group / 실행 노드 격리, 대상 시스템 권한은 AWX 관리자가 검토해야 합니다. 입력 문자열을 shell 코드로 결합하는 Playbook을 허용하지 마세요.

단순 문자열은 관리자가 지정한 전체 일치 정규식, 선택 목록은 정확한 허용 값, 정수는 범위와 정수 여부, 참/거짓은 실제 boolean 값으로 검사합니다. Ansible 예약 변수와 Jinja 코드 표현 입력은 거부합니다. 템플릿의 숨겨진 `extra_vars` 원문은 검토 snapshot에 넣지 않고 해시만 보관합니다.

AWX의 launch 응답을 받지 못하면 실제 작업이 시작됐는지 알 수 없습니다. 이 경우 자동 재실행하지 않습니다. 관리자는 AWX의 시간·추적 ID·출력을 확인해야 합니다.

## Kubernetes 격리 조건

관리 화면에서 API HTTPS 주소, 최소 권한 토큰, CA PEM, 전용 namespace와 선택적 RuntimeClass를 지정합니다. 기본·시스템 namespace는 사용할 수 없습니다.

선택적 외부 실행 클러스터의 출발점은 [namespace·NetworkPolicy·Quota·최소 권한 RBAC 예시](../deploy/runbook-kubernetes.yaml)입니다. 대상 클러스터를 확인한 운영자만 적용해야 하며 madi 서비스 배포에 자동 적용하지 않습니다. ServiceAccount의 제한된 토큰을 발급해 관리자 화면에 입력하고 만료 전에 회전하세요. RuntimeClass를 설정한다면 해당 이름에 한정한 읽기 권한도 추가해야 합니다.

계획 준비와 실행 직전에 실제 API에서 다음을 확인합니다.

- Namespace Pod Security Admission `restricted`, 버전 `latest`.
- 실행 Pod에 적용되는 Ingress / Egress 양방향 deny-all. 전용 namespace의 다른 NetworkPolicy에도 허용 규칙이 없어야 합니다.
- Pod·Job 개수 및 CPU / 메모리 requests / limits를 제한하는 적용된 ResourceQuota.
- 선택한 RuntimeClass가 실제로 존재하는지 확인합니다.

생성되는 Job은 고정 image digest, 고정 절대경로 실행 파일, 독립 argv만 사용합니다. 매개변수 치환은 `${name}` 단독 인자에만 허용합니다. `sh -c`, `python -c` 등의 인라인 스크립트는 거부합니다. 이미지에 들어 있는 사전 검토된 절대경로 스크립트를 실행할 수 있습니다.

Job은 재시도 0회·동시 Pod 1개·non-root UID/GID 65532·읽기 전용 root filesystem·권한 상승 금지·capabilities 전체 제거·RuntimeDefault seccomp·서비스 계정 토큰 미마운트로 생성합니다. 호스트 네트워크/PID/IPC는 사용하지 않으며 `/tmp`의 16MiB 메모리 임시 볼륨만 허용합니다. 실제 생성된 Job과 Pod에 추가 컨테이너·환경 변수·볼륨이 주입되면 실행을 중지합니다.

NetworkPolicy 객체가 있다는 것만으로 CNI가 정책을 집행한다는 사실까지 증명할 수는 없습니다. **운영자는 실제 클러스터에서 통신 차단 시험을 수행해야 합니다.** RuntimeClass를 쓰면 gVisor/Kata 같은 해당 격리 런타임도 별도로 설치·검증합니다. 시스템 밖으로 통신해야 하는 Ansible 운영 작업은 검토한 AWX 인벤토리·실행 노드를 사용하세요.

Kubernetes 작업 이름은 실행 ID + 단계 번호로 고정됩니다. 생성 응답 손실 후에도 동일 Job의 사양·UID를 확인하여 재연결하며, 다른 이름의 새 작업을 만들지 않습니다. 삭제는 UID precondition과 Foreground 전파를 사용합니다.

## 장애와 불확실 실행

실행 중 로그인 사용자 비활성화, 문서 ACL 회수, 실행 역할·팀·설정 변경, 정책 변경, 제한 시간 초과, 명시 취소가 확인되면 새 단계를 시작하지 않고 추적 중인 원격 작업의 중지를 시도합니다. API 네트워크가 끊어진 경우 원격 중지를 보장할 수 없으므로 `unknown` 상태로 남깁니다. 서버 재시작 후에도 저장된 Job ID로만 복구·취소하며 AWX의 불확실 launch를 반복하지 않습니다.

같은 문서에 unknown 실행이 남으면 새 실행이 차단됩니다. `서비스 관리 → 격리 실행 · Runbook → 외부 실행 확인 필요`에서 관리자가 다음을 확인합니다.

1. 외부 시스템에서 해당 실행이 시작되지 않았거나 이미 중지된 것을 직접 확인합니다.
2. 확인 근거를 20자 이상 기록하고 명시적 확인란 및 `RESOLVE 실행-ID`를 입력합니다.
3. 수동 종결은 취소로 기록되며 감사로그를 남깁니다. **자동 검증·성공 기록이 아니고 원격 명령을 실행하지 않습니다.**

복원된 백업의 실행·승인은 재사용하지 않습니다. 활성 실행은 불확실 상태로 보관하며 실행 기능을 끕니다. 복원 자체만으로 외부 실행이나 취소를 수행하지 않으며 운영자가 현재 외부 환경을 확인해야 합니다. 과거 실행기 자격 증명 버전은 기존 작업 취소를 위해 암호화된 상태로 보관합니다.

## 한도와 API

- 단계별 최대 1,800초, 계획 전체는 여유 시간을 포함해 3,600초 이하, 유형별 최대 20단계.
- 사용자별 동시 대기·실행 10개. 계획 사전 검증은 서버 프로세스별 사용자당 1분에 10회.
- 단계 출력 2MB, 실행 출력 5MB. 화면은 최근 50만 자를 표시하며 전체 원격 출력은 실행기에서 확인합니다.
- SSE는 현재 로그인 세션·문서 ACL·실행 권한을 반복 검증합니다. API 키·플러그인으로 실행 확인 또는 원격 출력 조회를 대신할 수 없습니다.
- 사용자 API: `/api/v1/documents/{id}/runbook`, `/runbook/prepare`, `/runbook/executions`; `/api/v1/runbook/executions/{id}` 및 `/approval`, `/execute`, `/cancel`, `/events`.
- 관리자 API: `/api/v1/admin/runbook/settings`, `/runners`, `/runners/{id}/test`, `/runners/{id}/versions`, `/uncertain`, `/executions/{id}/resolve`.

서비스 이미지는 외부 실행기 이미지나 AWX/Kubernetes를 포함하지 않습니다. 폐쇄망에서는 실행기·런타임·고정 digest 이미지와 인증서를 조직 내부에 별도 반입해야 합니다. Go 서버에는 Node 런타임이나 클러스터 CLI를 추가하지 않습니다.

검증은 TLS 모의 AWX/Kubernetes API, 실제 PostgreSQL, 동시성 검사 및 브라우저 흐름으로 수행합니다. 조직의 실제 AWX/클러스터 설치를 원격 운영 시험했다고 주장하지 않습니다. 운영 전 해당 환경에서 최소 권한·격리·중지·백업 복원 수용 시험이 필요합니다.

## 실제 화면

[관리자 실행기 설정](screenshots/runbook-admin.png) · [운영 절차 편집](screenshots/runbook-document.png) · [실행 계획과 실시간 출력](screenshots/runbook-execution.png) · [모바일 실행 화면](screenshots/runbook-mobile.png)

프로토콜·격리 근거: [AWX API](https://docs.ansible.com/projects/awx/en/latest/rest_api/api_ref.html), [AWX Job Templates](https://docs.ansible.com/projects/awx/en/24.6.1/userguide/job_templates.html), [Kubernetes Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/), [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/), [NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/).
