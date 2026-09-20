## 주요 변경

- Langflow Ready 검사에서 전체 `GET /api/v1/admin/settings`의 `sessionGateway` 객체를 검증·복제하여 추가 필드까지 보존합니다.
- 임시 설정 저장 후 `finally`에서 원본을 복원하고, 조회·임시 저장·복원 실패를 전파합니다. 키가 없으면 쓰기 없이 이유를 알리고 검사를 생략합니다.
- PostgreSQL 17은 Docker 이미지/오프라인 번들에 **포함되지 않습니다**. 별도로 운영하는 PostgreSQL과 `AGENTHUB_POSTGRES_DSN` 연결이 필수입니다.
- `offline-bundle.json`과 `agenthub-offline-linux-amd64`가 런타임 선택, 검증 다운로드, 분할 archive 로드를 제공합니다.
- 게시 이미지의 SPDX JSON SBOM, `SHA256SUMS`, GitHub OIDC/Sigstore 증명은 기존 태그 릴리즈 워크플로가 생성합니다.

AgentHub 제어 이미지: `agenthub:v0.250.0` — 태그 푸시 후 릴리즈 워크플로에서 게시됩니다.

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

각 자산의 실제 원본 릴리즈와 분할 파일/체크섬은 CI가 생성하는 `offline-bundle.json`에 기록됩니다.

### 검증

- 전체 Go race 테스트, Go 빌드, 웹 npm ci/lint/build 통과.
- SSO Node 12건, sessionGateway Node 18건 및 Langflow 구문 검사 통과.
- 릴리즈 버전 일치, 카탈로그 validate/check-versions, Kubernetes 렌더링, Compose 및 오프라인 외부 PostgreSQL 필수 조건 검사 통과.
- 구현 단계에서 실제 Server.Handler·격리 PostgreSQL·Node subprocess의 5개 시나리오를 검증했습니다. 이번 릴리즈 세션은 DB 연결 없이 검증했으며 DB 의존 테스트는 생략됐습니다.
- 실제 Ready Langflow/클러스터 브라우저 검사는 실행하지 않았습니다.
