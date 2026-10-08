## 주요 변경

- 가이드 스크린샷 생성 중 CSP 보고 요청에도 `AGENTHUB_GUIDE_REQUEST_TIMEOUT_MS`(기본 30000ms)를 적용했습니다. 응답이 멈추면 요청을 중단하고 정책·내용 검사·세션 게이트웨이·방문 추적의 원래 설정 네 가지를 모두 복원 시도합니다.
- 실제 호출부·`captureTracking`·`withGuideSettings`를 연결한 회귀 7건을 추가했습니다. 정상 보고, 제한 시간 전달, 요청 중단 후 복원 시도와 복원 실패 시 원래 보고 오류 우선 처리를 검증합니다.
- PostgreSQL 17은 Docker 이미지/오프라인 번들에 **포함되지 않습니다**. 별도로 운영하는 PostgreSQL과 `AGENTHUB_POSTGRES_DSN` 연결이 필수입니다.
- `offline-bundle.json`과 `agenthub-offline-linux-amd64`가 런타임 선택(`--runtime`, `--all-runtimes`, `--no-runtimes`), 검증 다운로드, 분할 archive 로드를 제공합니다.
- `runtime-images.json` 카탈로그가 런타임 종류, 버전, 빌드 입력, 의존성, archive와 상태 점검을 연결합니다.
- 태그 릴리즈 워크플로에서 게시 이미지의 SPDX JSON SBOM, `SHA256SUMS`, GitHub OIDC/Sigstore attestation bundle을 생성합니다. 제어 이미지의 버전·커밋과 게시 런타임의 상태도 검사합니다.

AgentHub 제어 이미지: `agenthub:v0.262.0` — 태그 릴리즈 워크플로에서 게시됩니다.

### 런타임 이미지

| 런타임 | 이미지 | 상태 | 용도 |
| --- | --- | --- | --- |
| Runtime Base | `agenthub-base:v0.26.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | OpenCode, Hermes, Qwen Paw 런타임을 쓰는 사이트에 필요합니다. |
| Langflow | `agenthub-langflow:v0.2.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | Langflow Agent를 쓰는 사이트만 필요합니다. |
| Qwen Code | `agenthub-qwencode:v0.2.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | 터미널 코딩 에이전트를 쓰는 사이트만 필요합니다. |
| JupyterLab | `agenthub-jupyter:v0.1.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | 노트북 작업대(Qwen Code 포함)를 쓰는 사이트만 필요합니다. |
| Goose | `agenthub-goose:v0.1.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | Goose Agent(ACP 실행)를 쓰는 사이트만 필요합니다. |
| HolmesGPT | `agenthub-holmes:v0.2.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | 장애 조사(조사 실행)를 쓰는 사이트만 필요합니다. |
| BrowserCode | `agenthub-browsercode:v0.2.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | 브라우저를 직접 모는 에이전트를 쓰는 사이트만 필요합니다. |
| Pi | `agenthub-pi:v0.1.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | Pi 코딩 에이전트를 쓰는 사이트만 필요합니다. |
| Prime Agent | `agenthub-primeagent:v0.1.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | Prime Agent 런타임을 쓰는 사이트만 필요합니다. |
| Orca | `agenthub-orca:v0.5.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | 멀티 에이전트 실행 패브릭을 쓰는 사이트만 필요합니다. |
| OpenHands | `agenthub-openhands:v1.43.1` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | OpenHands Agent Server를 쓰는 사이트만 필요합니다. |
| OpenCodeReview | `agenthub-opencodereview:v0.1.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | 코드 리뷰 에이전트를 쓰는 사이트만 필요합니다. |
| Node-RED | `agenthub-nodered:v0.1.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | Node-RED Agent를 쓰는 사이트만 필요합니다. |
| n8n | `agenthub-n8n:v0.2.0` | 버전 유지; 자산 재사용 여부는 태그 CI에서 검증 | n8n Agent를 쓰는 사이트만 필요합니다. |

각 자산의 실제 원본 릴리즈와 분할 파일/체크섬은 태그 CI가 생성하는 `offline-bundle.json`에 기록됩니다.

### 검증

- Go 전체 race 테스트, 웹 의존성 설치·lint·SSO 테스트·프로덕션 빌드 통과.
- Node 스크립트 회귀 154건 통과(실패·skip 0), `guide-shots.mjs` 구문 검사 통과.
- 이미지 카탈로그·소스/버전 검사, Kubernetes 렌더링, 개발 및 오프라인 Compose 설정 검사 통과.
- 실제 Chromium·관리자 HTTP/DB를 사용하는 가이드 촬영과 운영 배포는 이번 로컬 릴리즈에서 실행하지 않았습니다. 게시 이미지 빌드·상태 검사·자산 검증은 태그 CI에서 수행합니다.
- `npm ci`의 기존 high 취약점 경고 2건은 남아 있으며, 이번 릴리즈에서 의존성을 변경하지 않았습니다.
