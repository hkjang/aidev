## 주요 변경

- Momento·Matomo 제공자 URL을 정규화한 출처(`scheme://host[:port]`)를 최대 300룬으로 제한합니다. 301룬 이상이면 추적 비활성·다른 제공자 선택·프록시 모드에서도 설정 검증이 거절됩니다.
- 제공자 출처가 CSP의 세 지시문에 반복되어 헤더가 과도하게 커지는 입력을 제한합니다. URL 전체 길이를 제한하지 않으며, 긴 경로·쿼리와 기존 불완전 입력 처리 조건은 유지합니다. 전체 CSP 헤더의 바이트 상한을 보장하는 변경은 아닙니다.
- 이미 저장된 과대 제공자 출처는 자동 정리하지 않습니다. 해당 값이 있으면 다른 설정을 저장할 때도 검증 오류가 날 수 있으므로 제공자 출처를 300룬 이내로 수정해야 합니다.
- PostgreSQL 17은 Docker 이미지/오프라인 번들에 **포함되지 않습니다**. 별도로 운영하는 PostgreSQL과 `AGENTHUB_POSTGRES_DSN` 연결이 필수입니다.
- `offline-bundle.json`과 `agenthub-offline-linux-amd64`가 런타임 선택(`--runtime`, `--all-runtimes`, `--no-runtimes`), 검증 다운로드, 분할 archive 로드를 제공합니다.
- `runtime-images.json` 카탈로그가 런타임 종류, 버전, 빌드 입력, 의존성, archive와 상태 점검을 빠짐없이 연결합니다.
- 게시 이미지마다 SPDX JSON SBOM을 제공하고, `SHA256SUMS`는 GitHub OIDC/Sigstore attestation bundle로 증명합니다.
- 제어 이미지의 `version --json` 결과(버전/commit)와 게시 런타임의 command/HTTP 상태를 이미지 빌드 직후 검사합니다.

AgentHub 제어 이미지: `agenthub:v0.259.0` — 이번 릴리즈에 게시됩니다.

### 런타임 이미지

| 런타임 | 이미지 | 상태 | 용도 |
| --- | --- | --- | --- |
| Runtime Base | `agenthub-base:v0.26.0` | 버전 변경 없음 | OpenCode, Hermes, Qwen Paw 런타임을 쓰는 사이트에 필요합니다. |
| Langflow | `agenthub-langflow:v0.2.0` | 버전 변경 없음 | Langflow Agent를 쓰는 사이트만 필요합니다. |
| Qwen Code | `agenthub-qwencode:v0.2.0` | 버전 변경 없음 | 터미널 코딩 에이전트를 쓰는 사이트만 필요합니다. |
| JupyterLab | `agenthub-jupyter:v0.1.0` | 버전 변경 없음 | 노트북 작업대(Qwen Code 포함)를 쓰는 사이트만 필요합니다. |
| Goose | `agenthub-goose:v0.1.0` | 버전 변경 없음 | Goose Agent(ACP 실행)를 쓰는 사이트만 필요합니다. |
| HolmesGPT | `agenthub-holmes:v0.2.0` | 버전 변경 없음 | 장애 조사(조사 실행)를 쓰는 사이트만 필요합니다. |
| BrowserCode | `agenthub-browsercode:v0.2.0` | 버전 변경 없음 | 브라우저를 직접 모는 에이전트를 쓰는 사이트만 필요합니다. |
| Pi | `agenthub-pi:v0.1.0` | 버전 변경 없음 | Pi 코딩 에이전트를 쓰는 사이트만 필요합니다. |
| Prime Agent | `agenthub-primeagent:v0.1.0` | 버전 변경 없음 | Prime Agent 런타임을 쓰는 사이트만 필요합니다. |
| Orca | `agenthub-orca:v0.5.0` | 버전 변경 없음 | 멀티 에이전트 실행 패브릭을 쓰는 사이트만 필요합니다. |
| OpenHands | `agenthub-openhands:v1.43.1` | 버전 변경 없음 | OpenHands Agent Server를 쓰는 사이트만 필요합니다. |
| OpenCodeReview | `agenthub-opencodereview:v0.1.0` | 버전 변경 없음 | 코드 리뷰 에이전트를 쓰는 사이트만 필요합니다. |
| Node-RED | `agenthub-nodered:v0.1.0` | 버전 변경 없음 | Node-RED Agent를 쓰는 사이트만 필요합니다. |
| n8n | `agenthub-n8n:v0.2.0` | 버전 변경 없음 | n8n Agent를 쓰는 사이트만 필요합니다. |

런타임 archive의 재사용 원본과 누락 자산의 재게시 여부는 태그 워크플로가 실제 GitHub 자산 목록으로 결정합니다. 각 자산의 실제 원본 릴리즈와 분할 파일/체크섬은 `offline-bundle.json`에 기록됩니다.

### 검증

- Go 전체 race 테스트, 프런트엔드 lint·SSO 테스트·빌드, 릴리즈 버전 일치·카탈로그 검사, Kubernetes 및 Compose 구문·배포 이미지 검증 통과.
- ASCII·한국어 출처 300/301룬 경계와 모드별 거절, 긴 경로·쿼리·포트 및 입력 불변 회귀 테스트 포함.
- 실제 PostgreSQL 연결 테스트와 운영 HTTP/DB 저장 검증은 DSN이 없어 수행하지 않았습니다. 이미지 빌드·상태 점검·SBOM·서명·게시 검증은 외부 러너의 태그 푸시 후 워크플로에서 수행합니다.
