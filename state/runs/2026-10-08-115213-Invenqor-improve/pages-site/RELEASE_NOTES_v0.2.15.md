# Invenqor Server·Agent v0.2.15 릴리즈 노트

릴리즈 일자: 2026-08-24
호환 Agent: v0.2.15 (Linux x86_64·aarch64, Windows x86_64)

v0.2.15는 Agent를 URL 하나만으로 등록·운영하는 흐름을 더 엄격하고 진단 가능하게
만들고, Windows 자동 업데이트와 설치 소프트웨어 식별의 빈틈을 보완합니다. Server는
공용 PostgreSQL을 사용하는 여러 Pod에서 Keycloak Secret, API Key 변경, Agent
이벤트 상태가 일관되도록 동시성 경계를 강화했습니다. 자산 API와 MCP는 최신
`2026-07-28` stateless lifecycle과 기존 `2025-11-25` client를 한 endpoint에서
함께 지원합니다.

Kubernetes 배포용 공개 이미지는
`ghcr.io/hkjang/invenqor-server:0.2.15`이며 `linux/amd64`와 `linux/arm64`를
지원합니다. 오프라인 Server·PostgreSQL 묶음은
`invenqor-0.2.15.tar.gz`입니다. Agent는 운영체제·아키텍처별
`invenqor-agent-*` 패키지와 이를 한 번에 반입하는
`invenqor-agents-0.2.15.tar.gz`를 별도 제공하며, 모든 archive에는 같은 이름의
`.sha256`이 있습니다. Agent 묶음과 Release의
`sign-agent-update-manifest-v2.py`로 관리 콘솔용 단일 dual-signature v2 JSON을
오프라인 생성할 수 있습니다.

## 1. URL-only Agent 등록과 실패 진단

- `config.toml`의 `server.url`만으로 자동 등록하고, 발급된 장비 Token은 설정 파일이
  아니라 Server origin별 보호 상태 저장소에 보존합니다. 관리자가 자동 등록을
  허용하고 IP/CIDR 정책을 만족하면 provisioned token을 수동 입력할 필요가 없습니다.
- `server.url`은 scheme, host, 선택적 port만 포함하는 origin이어야 합니다. API path,
  query, fragment, 사용자 정보가 들어간 주소는 시작 단계에서 거부해 Ingress HTML이나
  잘못된 endpoint로 전송하는 오류를 예방합니다.
- 사설 주소의 HTTP는 명시 정책 아래 허용하지만 공용 주소는 HTTPS를 요구합니다.
  외부 `https://` Ingress가 내부 Service의 단일 `7070` port로 전달하는 구성이 기본
  경로입니다.
- 긴 한글 Server 오류도 UTF-8 문자 경계를 보존해 축약하므로 진단 출력 자체가
  panic하지 않습니다. Agent `--diagnose`, `--status`, `status.json`과 Server의
  request ID·멀티 Pod 진단 로그를 함께 사용하면 등록 거부 단계와 인식된 source IP를
  추적할 수 있습니다.

## 2. 서명 Agent 자동 업데이트 강화

- 자동 업데이트를 켜면 channel은 `stable` 또는 `beta`, 공개 키는 유효한 Base64
  Ed25519 32-byte key, 설치 경로는 절대 경로여야 합니다. 잘못된 설정은 다운로드
  이후가 아니라 기동 시 실패합니다.
- 새 게시물은 artifact 원문만이 아니라 version, channel, 운영체제, 아키텍처,
  정확한 크기, SHA-256, `allow_downgrade`를 canonical Ed25519 manifest v2로 함께
  서명합니다. 기존 v1 릴리즈는 정상 상향 업데이트에 한해 호환하되 unsigned
  rollback 권한으로는 사용하지 않습니다.
- 오프라인 signer는 이전 Agent용 artifact 서명과 metadata-bound manifest 서명을
  하나의 `.signature-bundle.json`으로 출력합니다. 관리 콘솔은 artifact와 이 JSON
  하나만 받고 서명된 식별 필드를 자동 고정합니다. 두 raw 서명 출력은 레거시
  자동화를 위한 선택 사항입니다.
- Server 공개키가 없으면 관리 콘솔과 게시 API가
  `UPDATE_SIGNING_KEY_MISSING`으로 fail-closed 거부합니다. 공개키가 있어도 두 서명,
  artifact 크기 또는 SHA-256이 다르면 게시 단계에서 거부되어 fleet에 노출되지
  않습니다.
- Manifest의 Server origin과 상대 download 경로도 적용 전에 검증합니다. 실행 전
  candidate self-check는 live 바이너리를 rename·교체하기 전에 수행하고 10초와
  stdout/stderr 각각 64 KiB로 제한하며, 원자 교체·backup 규칙을 유지합니다.
- `Content-Length`가 없는 Ingress의 chunked response도 manifest 크기를 상한으로
  streaming하며, 실제 byte 수·hash·서명을 마지막에 다시 검증합니다. 선언 크기를
  넘는 응답은 메모리나 디스크를 무제한 소비하지 않고 중단합니다.
- Windows service는 서명된 artifact를 staging만 해두지 않고 다음 안전한 주기에
  자동 적용한 뒤 fresh install 직후에도 동작하는 SCM crash recovery restart를
  요청합니다. Windows와 Linux의 진단 문구도 실제 activation 경로에 맞게
  분리했습니다.
- `install.ps1 -ServiceName` 사용자 지정 값은 이제 SCM `binPath`에 안전하게 인용되고
  보호된 marker에서 복구됩니다. service dispatcher·상태 진단·콘솔 update 재시작이
  모두 실제 이름을 사용하며, 기존 `--service`/`invenqor-agent` 설치도 호환됩니다.
  따옴표·slash·제어 문자를 이용한 argv injection은 Installer와 Agent 양쪽에서
  fail-closed로 거부합니다.

## 3. Windows·Linux 자산 품질

- Windows machine-wide와 user별 Uninstall registry 항목은 owner SID와 registry key를
  안정 식별자에 포함합니다. 같은 제품·버전이 여러 사용자 범위에 설치돼도 서로
  덮어쓰지 않습니다.
- 같은 이름의 RPM 여러 버전이 공존할 때 version, architecture, package instance를
  구분해 자산 ID 충돌을 방지합니다.
- container overlay나 bind 구성에서 같은 target에 겹친 mount는 실제 보이는 최상위
  layer 하나만 자산화해 디스크 중복을 줄입니다.
- 원시 process·service·package 증거를 Server의 내장 소프트웨어 catalog가 host별
  제품으로 자동 결합하는 v0.2.14 모델은 유지됩니다. 이번 안정 ID 보강으로 사용자별
  Windows 설치와 병렬 RPM이 제품 판별 근거에서 사라지지 않습니다.

## 4. Kubernetes 멀티 Pod 일관성

### 4.1 Keycloak Client Secret

Keycloak Client Secret은 Pod 로컬 파일이 아니라 공용 PostgreSQL 설정에 저장됩니다.
모든 Pod가 공유하는 정확히 같은 32-byte Master Key로 AES-256-GCM AEAD 암호화하며,
일반 설정 목록·이력에는 Secret key 자체를 노출하지 않습니다. Keycloak 설정은 전용
API와 화면에서만 변경할 수 있고 일반 설정 PATCH·rollback은
`409 DEDICATED_SETTING_ENDPOINT`로 거부됩니다.

v0.2.14의 `bootstrap.enc`에만 Secret이 있으면, 기존 state volume과 Master Key를
가진 첫 v0.2.15 Pod가 공용 DB에 값이 없을 때 insert-if-absent로 자동 이관합니다.
Rolling upgrade 동안 기존 volume을 먼저 제거하지 말고 두 번째 Pod에서도
`client_secret_configured=true`인지 확인해야 합니다.

### 4.2 API Key와 이벤트 상태

- Scope 추가·삭제는 PostgreSQL 현재 `scopes_json`에 대한 compare-and-swap과 제한된
  재시도를 사용합니다. 전체 scope 교체와 이름+scope PATCH는 하나의 조건부 UPDATE로
  처리해 부분 반영을 막습니다.
- Secret 회전은 현재 `key_hash`를 revision으로 사용합니다. 동시 회전에서 한 요청만
  일회성 Secret을 받고, 패자는 `409 API_KEY_CONFLICT`를 받아 사용할 수 없는 Secret을
  배포하지 않습니다. 충돌 후에는 최신 key를 다시 읽고 작업 의도를 재확인해야 합니다.
- `(Agent ID, Event ID)`가 같은 이벤트의 `processed`는 최종 상태입니다. 다른 Pod의
  늦은 실패 기록이 성공 이벤트를 `failed`로 되돌리지 않으며, 재전송은 중복 성공으로
  처리됩니다.
- 자산 상태 갱신은 `(effective created_at, Server received_at, event_id)`를
  단조 증가 순서로 사용합니다. 같은 Agent 시각과 DB 수신 시각이 겹쳐도 Event ID로
  결정적으로 순서를 정하고, 수신 시각보다 10분 넘게 미래인 Agent·record 시각은
  수신 시각으로 clamp해 잘못된 장비 시계가 이후 정상 이벤트를 막지 않습니다.

이 경계들은 Pod 메모리 lock이나 sticky session에 의존하지 않습니다. PostgreSQL
DSN이 지정되면 연결·migration 실패 시 Server 기동과 readiness가 fail-closed로
실패하며 Pod별 SQLite로 우회하지 않습니다. SQLite는 DSN을 지정하지 않은 단일
Server 개발·복구 모드이며 멀티 Pod 운영에는 PostgreSQL이 필요합니다.

### 4.3 공용 이벤트 spool과 update 저장소

Kubernetes Chart는 Pod별 RWO state와 분리된 event spool RWX PVC를 기본으로
마운트하고 update artifact에도 별도 RWX PVC를 사용합니다. DB 장애 중 어느 Pod가
Agent 이벤트를 수락해도 StatefulSet scale-down이나 Pod 장애 후 생존 Pod가 이어서
재처리할 수 있습니다.

- Event는 owner-only 임시 파일에 기록하고 fsync·close한 뒤 hard-link
  no-clobber 방식으로 최종 segment를 원자 게시합니다. Replayer는 완성된 파일만
  보며 같은 이벤트의 동시 retry가 기존 segment를 덮어쓰지 않습니다.
- 모든 Pod가 공유하는 `.replay.lock`의 OS advisory lock으로 한 replayer만
  동작합니다. 프로세스가 종료되면 커널이 lock을 해제하므로 별도 leader Pod나
  sticky routing이 필요 없습니다.
- Server는 root 소유, `fsGroup: 65532`, mode `0770`인 CSI mount의 쓰기 권한을
  검증해 사용할 수 있습니다. 다른 사용자에게 읽기 권한이 있는 저장소는 비밀과
  이벤트 보호를 위해 fail-closed 거부합니다.
- 공용 update 저장소도 cross-process lock과 staged publication을 사용해 여러 Pod의
  동시 게시·삭제가 artifact와 manifest의 절반 상태를 노출하지 않게 합니다.

## 5. MCP 2026-07-28와 API 계약

동일한 `POST /mcp` endpoint가 두 lifecycle을 지원합니다.

- 최신 client: `MCP-Protocol-Version: 2026-07-28`, `Mcp-Method`, named request의
  `Mcp-Name`, protocol `_meta`를 사용한 stateless request
- 기존 client: `initialize`로 협상하는 `2025-11-25` lifecycle과 legacy notification

최신 lifecycle은 `server/discover`, `tools/list`, `tools/call`을 제공하고 도구 노출과
실행 권한을 매 요청의 API Key scope로 다시 확인합니다. routing header와 body가
다르거나 지원하지 않는 protocol version, 여러 JSON message, trailing JSON은
명시적인 JSON-RPC 오류로 거부합니다. `2026-07-28`에서 제거된 `ping`과 현재 Server가
제공하지 않는 resource/task method를 성공으로 가장하지 않습니다. 상세 예제와
Base64 header sentinel 규칙은 [API·MCP 가이드](API_MCP_GUIDE.md)에 있습니다.

OpenAPI 문서는 새 MCP header와 충돌 응답을 포함하며 CI에서 Redocly lint를
통과해야 합니다.

## 6. 관리 콘솔·공급망 품질

- Ingress나 reverse proxy가 JSON 대신 HTML/text 오류를 반환해도 HTTP status와 원문
  요약을 보존한 오류로 표시합니다.
- API Key 이름 변경은 PATCH 응답 하나로 화면 상태를 갱신하며 이전 오류를 남기지
  않습니다.
- 설정, 사용자·역할, 프로필, 자산, 감사·Server 로그, navigation에 접근 가능한 이름,
  현재 상태, keyboard focus 표시를 추가했습니다.
- 모든 Pod의 API·Agent 요청을 공용 DB에 구조화 access log로 남겨 method, path,
  status, 처리 시간, request ID를 **Server 로그**에서 검색합니다. 정상 health probe와
  정적 asset은 제외하고 실패 응답은 항상 보존하며 Secret은 redaction합니다.
- 인증 전 시스템 정보는 로그인 화면용 제품 버전만 반환합니다. DB 모드, bind 주소와
  Agent 등록 정책은 `settings.read`가 필요한 관리자 시스템 정보로 분리했습니다.
- 프로덕션 dependency의 `nanoid`를 보안 수정 버전으로 올렸고 npm production audit을
  CI gate로 추가했습니다.
- GitHub Actions는 Node 24 runtime을 사용하는 공식 major로 갱신했으며 UI test,
  dependency audit, OpenAPI lint를 CI에서 실행합니다.

## 7. 릴리즈 검증 범위

릴리즈 후보에는 다음 자동 검증 경로가 포함됩니다. 릴리즈 판정은 같은 commit의
로컬 검증과 GitHub main·native Windows CI가 모두 성공한 뒤 확정합니다.

- Rust `fmt`, 모든 target `clippy -D warnings`, 단위 테스트와 Windows target build
- Go 전체 단위·통합 테스트, `go vet`, Server build
- React/Vitest, production build와 embedded UI 동기화, production dependency audit
- OpenAPI lint, Helm lint/template와 GitHub Actions workflow 검증
- 실제 chunked HTTP update staging, 잘못된 크기·hash·서명 거부와 signer 출력 보호
- PostgreSQL API Key 동시 scope·회전, OIDC 공유 Secret 이관, 완료 이벤트 상태 보존,
  미래 시각 clamp와 동일 시각 Event ID 순서
- CentOS 7, Red Hat UBI 8/9, Ubuntu 22.04/24.04 LTS, Alpine에서 최신 musl Agent의
  실제 수집·URL-only 등록·PostgreSQL Server 전달
- 배포용 Linux archive를 RHEL 8 계열 systemd 환경에 설치해 service daemon,
  update path unit, 설정 권한, 상태·진단과 Server 전달을 확인하는 packaged E2E
- PostgreSQL 장애 중 두 Server Pod가 공용 RWX spool에 수락하고, 한 Pod 제거 뒤
  생존 Pod가 replay하며 재확장 후에도 공용 update 게시가 유지되는 멀티 Pod E2E
- GitHub `windows-latest`에서 실제 배포 ZIP을 사용자 지정 SCM 서비스명으로 설치해
  자동 등록, 네이티브 수집, 운영체제 표시, 주요 소프트웨어 정규화, 상태·진단,
  service marker와 제거 경로를 확인하는 E2E

## 8. 업그레이드 순서

1. PostgreSQL, 공용 Master Key Secret, 공용 event spool·update RWX PVC와 기존
   Pod별 Server state volume을 같은 시점으로 백업합니다.
2. 동일 Master Key와 기존 state volume을 연결한 Server Pod 하나를 v0.2.15로 올립니다.
3. migration, Keycloak Secret 이관, readiness와 로그인·Agent ingest를 확인합니다.
4. 나머지 Pod를 올리고 다른 Pod에서도 Keycloak 설정 구성 여부, API Key scope와 Server
   로그가 동일한지 확인합니다.
5. Agent update manifest를 제한 rollout으로 발행하고 Windows service의 적용·재시작과
   Linux package/service 상태를 확인한 뒤 확대합니다.

Master Key가 서로 다른 Pod를 혼용하거나, 공용 DB 이관을 확인하기 전에 v0.2.14
`bootstrap.enc` volume을 폐기하면 안 됩니다. 기존 Agent 등록 Token과 자산 ID는
유지되므로 재등록이나 상태 디렉터리 삭제는 필요하지 않습니다.

## 9. 배포 파일

- Server 오프라인 image 묶음: `invenqor-0.2.15.tar.gz`와 `.sha256`
- Helm Chart: `invenqor-0.2.15.tgz`와 `.sha256`
- 개별 Agent: `invenqor-agent-linux-x86_64.tar.gz`,
  `invenqor-agent-linux-aarch64.tar.gz`, `invenqor-agent-windows-x86_64.zip`과
  각 `.sha256`
- Agent 전체 오프라인 묶음: `invenqor-agents-0.2.15.tar.gz`와 `.sha256`
- 오프라인 update signer: `sign-agent-update-manifest-v2.py`(Agent 전체 묶음에도 포함)
- 운영 자료: `compose.offline.yaml`, `openapi.yaml`, 역할별 사용자·관리자·API/MCP·
  Server 설치·임원 보고 문서의 Markdown/PDF
