# Invenqor Server v0.2.2 릴리즈 노트

릴리즈 일자: 2026-07-29

## 핵심 변경

- 쉘이 없는 Distroless 컨테이너에서도 초기 관리자를 자동 생성할 수 있도록
  `INVENQOR_BOOTSTRAP_ADMIN`과 `INVENQOR_BOOTSTRAP_ADMIN_PASSWORD`를
  지원합니다.
- 요청된 호환 이름 `bootstrap_admin`, `bootstrap_admin_password`와
  Compose용 `BOOTSTRAP_ADMIN`, `BOOTSTRAP_ADMIN_PASSWORD`도 지원합니다.
- Kubernetes Secret을 환경변수 값으로 노출하지 않도록
  `INVENQOR_BOOTSTRAP_ADMIN_PASSWORD_FILE`을 지원합니다.
- 최초 생성은 PostgreSQL의 공유 claim을 원자적으로 소비합니다. 여러 Server
  Pod가 동시에 시작해도 관리자 한 명만 생성됩니다.
- 사용자가 이미 존재하면 환경변수로 기존 계정이나 비밀번호를 변경하지 않습니다.
- 기존 일회성 bootstrap token 방식은 환경변수를 사용하지 않는 설치를 위해
  그대로 유지합니다.

## 보안과 운영 특성

- 초기 비밀번호는 정책 검증 뒤 Argon2id hash로만 저장합니다.
- 평문 비밀번호는 로그와 감사 이벤트에 기록하지 않으며, 시작 처리 직후 프로세스
  설정에서도 제거합니다.
- 직접 전달한 환경변수는 `docker inspect`에서 보일 수 있습니다. 계정 생성 후
  제거하거나 Kubernetes에서는 Secret file mount 방식을 사용하십시오.
- Server 이미지는 비루트 Distroless 구조를 유지합니다. 이미지 내부에
  `/bin/bash`나 `/bin/sh`는 포함하지 않습니다.

## 배포 파일

- 오프라인 Docker 번들: `invenqor-0.2.2.tar.gz`
- Helm chart: `invenqor-0.2.2.tgz`
- Offline Compose, OpenAPI 명세, 사용자·관리자·임원·Server·API/MCP 가이드의
  Markdown 및 PDF
- Agent는 별도 저장소의
  [Invenqor Agents v0.2.1](https://github.com/hkjang/invenqor-agents/releases/tag/v0.2.1)
  릴리즈를 사용합니다.

## 검증

- Go 전체 테스트와 race detector, `go vet`
- 실제 Docker 이미지의 환경변수 및 password file 초기 관리자 생성·로그인
- 동일 PostgreSQL에 연결한 Server 2개 동시 기동과 단일 관리자 생성
- 재기동 시 기존 비밀번호 불변 확인
- Helm lint·template 및 Kubernetes client dry-run
- Online/Offline Compose 설정 검증
- Trivy filesystem/image HIGH·CRITICAL 취약점 및 secret scan
