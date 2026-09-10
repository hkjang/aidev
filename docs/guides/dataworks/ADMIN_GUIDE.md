# Data Works 관리자 가이드

> 적용 버전: **v0.9.51**<br>
> 서비스 관리자 화면: `http://<host>:8080/dataworks/settings`<br>
> 일반 사용 방법은 [사용자 가이드](USER_GUIDE.md)를 참고하세요.

이 문서는 폐쇄망 설치, 최초 관리자 로그인, AI·MCP, Keycloak SSO, 키 정책과 데이터 상품 운영 절차를 실제 v0.9.51 구현 기준으로 설명합니다.

## 1. 운영 구성

Data Works 배포 이미지는 React SPA와 Go API를 하나의 바이너리로 포함합니다. 운영 저장소는 PostgreSQL을 사용하며, 브라우저는 `/dataworks/`, API는 같은 호스트의 `/admin/*`, `/v1/*`, `/mcp*` 경로를 사용합니다.

### 서비스 접속 경로

| 주소 | 용도와 동작 |
| --- | --- |
| `/dataworks/` | 새 React 기반 사용자·관리자 Workbench의 **표준 진입점** |
| `/dataworks` | `/dataworks/`로 `308 Permanent Redirect` |
| `/admin` | 기존(레거시) 관리자 콘솔 |
| `/` | UI 라우트가 아니며 기본 구성에서는 `404 Not Found` |

운영 안내에는 `http://<host>:8080/dataworks/`를 서비스 주소로 사용하세요. 대표 도메인의 루트(`/`)를 진입점으로 사용할 경우 리버스 프록시에서 `/`를 `/dataworks/`로 리다이렉트합니다. 프록시는 SPA와 로그인·API 기능이 함께 동작하도록 `/dataworks/*`, `/auth/*`, `/admin/*`, `/v1/*`, `/mcp*`, `/openapi.json`, `/swagger`를 동일한 Data Works 서비스로 전달해야 합니다.

GitHub Release의 운영 산출물은 다음 하나의 custom asset입니다.

```text
dataworks-v0.9.51.tar.gz
```

압축 파일을 적재하면 다음 이미지가 생성됩니다.

```text
dataworks:v0.9.51
```

## 2. 폐쇄망 설치

### 필수 환경변수 네 개

| 이름 | 설명 |
| --- | --- |
| `POSTGRES_DSN` | 운영 PostgreSQL 연결 문자열 |
| `BOOTSTRAP_ADMIN` | 최초 최고 관리자 이메일 |
| `BOOTSTRAP_ADMIN_PASSWORD` | 최초 최고 관리자 비밀번호 |
| `ENCRYPTION_KEY` | JWT 서명과 저장 비밀 암호화에 사용할 키 |

운영자가 컨테이너에 전달해야 하는 설정은 이 네 항목입니다. AI 공급자, Keycloak과 런타임 정책은 기동 후 관리자 화면에서 저장합니다.

`ENCRYPTION_KEY`는 32바이트 난수를 64자리 16진수로 표현하는 방식을 권장합니다.

```bash
openssl rand -hex 32
```

키는 백업 가능한 비밀 저장소에 보관하고 운영 중 임의로 변경하지 마세요. 변경하면 기존 암호화 비밀을 해독할 수 없고 로그인 세션에도 영향을 줍니다.

### 이미지 적재

```bash
gzip -t dataworks-v0.9.51.tar.gz
gunzip -c dataworks-v0.9.51.tar.gz | docker load
docker image inspect dataworks:v0.9.51
```

### 컨테이너 실행

PostgreSQL은 폐쇄망 안에서 컨테이너가 접근할 수 있어야 합니다.

```bash
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres.internal:5432/dataworks?sslmode=require' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='replace-with-a-strong-password' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.51
```

저장소의 `docker-compose.yml`을 함께 반입한 환경에서는 같은 네 값을 `.env`에 저장한 뒤 실행할 수 있습니다.

```dotenv
POSTGRES_DSN=postgres://dataworks:change-me@postgres.internal:5432/dataworks?sslmode=require
BOOTSTRAP_ADMIN=admin@dataworks.local
BOOTSTRAP_ADMIN_PASSWORD=replace-with-a-strong-password
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

```bash
docker compose config --images
# dataworks:v0.9.51
docker compose up -d
```

### 기동 확인

```bash
curl -fsS http://127.0.0.1:8080/health
curl -fsS http://127.0.0.1:8080/ready
docker logs --tail=100 dataworks
```

정상 응답:

```json
{"status":"ok"}
{"status":"ready"}
```

PostgreSQL 마이그레이션은 기동 시 자동 실행됩니다. 외부 공개 환경에서는 TLS 종료와 접근 제어가 적용된 리버스 프록시 뒤에 배치하세요.

## 3. 최초 관리자 로그인

`BOOTSTRAP_ADMIN` 계정이 DB에 없으면 최초 기동 시 `super_admin`으로 생성됩니다. 이미 같은 이메일이 있으면 재생성하거나 비밀번호를 덮어쓰지 않습니다.

1. `http://<host>:8080/dataworks/`를 엽니다.
2. Bootstrap 이메일과 비밀번호로 로그인합니다.
3. 로그인 화면 또는 프로필 메뉴에서 `v0.9.51`를 확인합니다.
4. 왼쪽 아래 `관리자 설정`을 엽니다.

![로그인 화면](assets/screenshots/desktop/00-login.jpg)

![프로필 버전 정보](assets/screenshots/desktop/15-profile-menu.jpg)

> 문서의 화면은 비밀값이 없는 데모 데이터로 촬영했습니다. 실제 운영 키와 고객 정보가 아닙니다.

## 4. 관리자와 개인화 영역

- **관리자 설정** `/dataworks/settings`: AI 공급자, AI·MCP 정책, Keycloak과 전체 런타임 설정
- **내 작업 공간** `/dataworks/personal`: 현재 사용자의 사용량·비용·품질과 개인 신호
- **내 API 키** `/dataworks/personal/keys`: 현재 사용자가 소유한 개인 키

관리자 영역과 개인화 영역은 별도 메뉴와 권한으로 분리됩니다. 설정 쓰기는 서버가 역할과 설정 카테고리별 권한을 다시 검사합니다.

주요 내장 역할에는 `super_admin`, `admin`, `team_admin`, `team_manager`, `developer`, `viewer`, `service_account`, `ops_admin`, `ai_admin`, `security_admin`, `billing_admin`, `readonly_admin`이 있습니다. Bootstrap 계정은 모든 Scope를 가진 `super_admin`입니다.

### 역할 및 권한 관리

`관리자 설정 → 역할 및 권한`은 기본 역할과 사용자 정의 역할, 역할별 사용자 수와 실제 할당 계정을 한 화면에서 관리합니다.

- 기본 역할은 제품 권한 기준이므로 수정하거나 삭제할 수 없습니다.
- 사용자 정의 역할은 영문 소문자로 시작하는 식별자, 설명, 로그인 시작 화면과 필요한 Scope를 조합해 생성합니다.
- `admin:write`, `routing:write`, `mcp:admin`을 선택하면 각각 필요한 조회·사용 Scope가 자동 포함되며 서버도 같은 종속성을 검증합니다.
- 관리자는 자기 역할보다 낮은 등급만 설계·할당할 수 있고 `super_admin`만 관리자 동급 역할을 위임할 수 있습니다.
- 사용자에게 할당된 역할은 먼저 다른 역할로 이동해야 삭제할 수 있습니다.
- 사용자 역할 또는 사용자 정의 역할의 Scope를 변경하면 기존 로그인 세션이 즉시 종료되어 다음 로그인부터 새 권한이 적용됩니다.
- 마지막 활성 `super_admin`은 강등하거나 비활성화할 수 없습니다.

역할 변경은 확인 대화상자를 거쳐 적용되고 감사 로그에 남습니다. Keycloak 역할 매핑의 내부 역할 이름에도 기본 역할과 여기서 만든 사용자 정의 역할을 사용할 수 있습니다.

```http
GET    /admin/roles
POST   /admin/roles
DELETE /admin/roles?role={role}
GET    /admin/users
PATCH  /admin/users/{user_id}
```

## 5. AI 공급자 설정

`관리자 설정 → AI 및 MCP`에서 공급자를 등록합니다.

필수 또는 주요 필드:

- 공급자 이름
- OpenAI 호환 Base URL
- API 키
- 모델 패턴: 예 `qwen-*,local-*`
- 호출 제한 시간
- 활성 상태

API 키는 암호화되어 DB에 저장되며 다시 평문으로 표시되지 않습니다. 기존 공급자를 수정할 때 API 키를 비우면 저장된 키를 유지합니다.

폐쇄망에서는 내부 vLLM, Qwen 또는 다른 OpenAI 호환 서버 주소를 Base URL로 지정합니다. 저장 후 실제 `/v1/models`와 테스트 호출로 모델 라우팅을 검증하세요.

![AI 및 MCP 설정](assets/screenshots/desktop/12-admin-ai-mcp.jpg)

## 6. AI 스트리밍과 토큰 정책

같은 화면에서 다음 설정을 관리합니다.

| 설정 | 의미 |
| --- | --- |
| `ai.default_stream` | Chat Completions에서 `stream` 생략 시 기본값, 기본 `true` |
| `limits.max_output_tokens` | 일반 응답 출력 상한, `0`은 관리 상한 비활성 |
| `limits.agent_max_tokens` | 내부 Agent 출력 상한 |
| `mcp.agentic_model` | Agentic MCP에 사용할 모델 |
| `mcp.max_tokens` | MCP 각 LLM 턴의 출력 토큰 |
| `mcp.max_agent_steps` | Agentic MCP 최대 LLM 턴 |
| `mcp.max_tools` | 모델에 노출할 최대 MCP 도구 수 |
| `mcp.force_tool_first` | 첫 턴 도구 호출 강제 여부 |

토큰 관련 서비스 절대 상한은 `262144`입니다.

- Chat Completions: `max_tokens`, `max_completion_tokens`
- Responses API: `max_output_tokens`
- 관리 상한이 양수이면 요청 값은 상한 이하로 조정됩니다.
- 호출자가 `stream:false`를 명시하면 기본 스트리밍 설정이 이를 덮어쓰지 않습니다.
- 연결 모델 자체 한도가 더 낮으면 모델 제한이 우선합니다.

## 7. Keycloak SSO/OIDC

`관리자 설정 → Keycloak SSO`에서 Issuer URL, Client ID와 Client Secret만 입력하면 OIDC Discovery와 PKCE 로그인 흐름을 자동 구성합니다.

### Keycloak 사전 설정

Keycloak에 confidential OIDC client를 만들고 다음 Redirect URI를 허용합니다.

```text
https://<service-host>/auth/keycloak/callback
```

HTTP 내부망 시험 환경에서는 실제 서비스 origin의 HTTP URI를 사용할 수 있지만 운영 환경은 HTTPS를 권장합니다.

### Data Works 설정

1. Issuer URL을 입력합니다. 예: `https://keycloak.internal/realms/dataworks`
2. Client ID와 Client Secret을 입력합니다.
3. Redirect URI가 서비스 공개 주소와 일치하는지 확인합니다.
4. `Keycloak SSO 활성화`를 켭니다.
5. 비상 접근 정책에 따라 `로컬 로그인 허용`을 선택합니다.
6. `SSO 설정 저장`을 누릅니다.
7. `연결 진단`에서 Issuer와 RSA 서명 키 발견을 확인합니다.
8. 로그아웃 후 로그인 화면의 `Keycloak SSO로 계속`을 시험합니다.

Client Secret은 암호화 저장됩니다. 수정 화면에서 Secret을 비우면 기존 값을 유지합니다.

![Keycloak SSO 설정](assets/screenshots/desktop/13-admin-keycloak-sso.jpg)

### 고급 역할 매핑

- OIDC Scope 기본값: `openid profile email`
- 기본 내부 역할
- 역할 Claim, 기본 `realm_access.roles`
- 그룹 Claim, 기본 `groups`
- Keycloak 역할→Data Works 역할 JSON

역할 매핑은 권한 상승 방지 검사를 적용합니다. 일반 관리자는 자신과 동급 이상의 새 관리자 또는 `super_admin` 매핑을 임의로 만들 수 없습니다. SSO를 켜기 전에 Bootstrap 로컬 계정으로 복구할 수 있는지 확인하고, 로컬 로그인을 끄기 전 별도 관리자 SSO 계정을 시험하세요.

관리 API:

```http
GET/PUT /admin/sso/keycloak/config
POST    /admin/sso/keycloak/test
GET     /auth/sso/status
```

## 8. 전체 런타임 설정

`관리자 설정 → 전체 설정`에서 설정 키와 설명을 검색하고 카테고리로 필터링합니다.

- 현재 값과 값 출처를 확인합니다.
- `저장`으로 DB 기반 관리자 override를 적용합니다.
- 되돌리기 버튼으로 관리자 override를 삭제하고 기본값을 복원합니다.
- 비밀 설정은 암호화하고 응답에서 마스킹합니다.
- 읽기 전용, 재시작 필요와 역할별 쓰기 가능 여부가 표시됩니다.

```http
GET    /admin/settings/effective
PUT    /admin/settings/by-key/{key}
DELETE /admin/settings/by-key/{key}
```

![전체 런타임 설정](assets/screenshots/desktop/14-admin-runtime-settings.jpg)

## 9. 개인 API 키 정책 운영

계정 로그인 사용자는 `/dataworks/personal/keys`에서 자신의 역할 범위 안에 개인 키를 발급할 수 있습니다. 비밀값은 발급·회전 직후 한 번만 표시됩니다.

발급 후 변경 가능한 정책은 정확히 다음 8개입니다.

1. `scopes`
2. `allowed_ips`
3. `allowed_models`
4. `denied_models`
5. `allowed_providers`
6. `denied_providers`
7. `budget_limit_krw`
8. `expires_at`

서버는 최종 정책이 호출자 정책의 부분집합인지 검사합니다. 상위 키나 사용자 역할보다 넓은 Scope, 모델, 공급자, IP, 예산 또는 만료 범위로 확대할 수 없습니다.

회전은 정책을 그대로 보존한 새 키를 원자적으로 만들고 기존 키를 폐기합니다. 폐기와 다른 사용자의 키 ID 조회는 소유권 검사를 적용합니다.

```http
GET    /me/keys
POST   /me/keys
PATCH  /me/keys/{id}
POST   /me/keys/{id}/rotate
DELETE /me/keys/{id}
```

![발급 후 권한 조정](assets/screenshots/desktop/11-personal-api-keys.jpg)

## 10. MCP 운영

Data Works에는 서로 다른 두 MCP 엔드포인트가 있습니다.

| 엔드포인트 | 용도 |
| --- | --- |
| `/mcp` | 등록된 외부 MCP 서버의 도구·프롬프트·리소스 집약 |
| `/mcp/gateway` | Data Works 자체 기능을 MCP 도구로 제공 |

React `AI 및 MCP` 설정에서는 Agentic MCP 모델과 토큰·턴·도구 수 정책을 관리합니다. 외부 MCP 업스트림 등록, Bearer 인증, allowlist·차단 정책, 도구별 역할 Scope, Route Explain, 연결 시험과 호출 관측은 기존 `/admin` 콘솔의 MCP 메뉴에서 관리합니다.

MCP 업스트림 인증 토큰은 암호화 저장됩니다. 개인 키에는 `mcp:use` Scope가 필요하고, MCP 관리에는 `mcp:admin` 등 관리자 권한이 필요합니다.

주요 API:

```http
GET/POST   /admin/mcp/upstreams
GET/PUT/DELETE /admin/mcp/upstreams/{id}
GET/POST   /admin/mcp/policies
GET/DELETE /admin/mcp/policies/{server}
GET/POST   /admin/mcp/tool-scopes
POST       /admin/mcp/route/explain
POST       /admin/mcp/test
```

사용자 연결 템플릿과 진단:

```http
GET  /me/onboarding-pack?client=mcp|cursor|roo|cline|openai-sdk
POST /me/connection-doctor
```

## 11. 데이터 상품 운영 흐름

React Workbench는 현황 탐색과 출시 게이트 외에도 쓰기 권한이 있는 사용자에게 자산 등록·수정·준비도 재평가·안전 삭제, 상품 생성·수정·상태 전환·초안/보관 삭제, Canvas 저장, 승인 추적 추가·갱신과 append-only 계약 버전 생성을 제공합니다. AI 상품 공장 실행 생성·재생과 일부 고급 운영은 REST API 또는 기존 관리자 콘솔을 사용합니다.

### 11.1 데이터 자산

```http
GET/POST/DELETE /admin/dataworks/assets
GET/POST /admin/dataworks/assets/readiness
POST     /admin/dataworks/assets/{asset_key}/readiness/check
GET      /admin/dataworks/assets/{asset_key}/lineage
```

화면에서 자산의 키, 이름, 도메인, 담당자, 컬럼 요약, 민감도와 갱신 주기를 등록·수정하고 개별 또는 전체 준비도를 재평가합니다. 삭제는 확인 절차를 거치며, 상품이 원천 자산으로 참조 중이면 `409 Conflict`로 차단됩니다. 민감 상품 출시 전에 준비도 70 이상인지 확인합니다.

### 11.2 아이디어와 상품 정의

```http
POST /admin/dataworks/factory/ideas
POST /admin/dataworks/factory/definitions
POST /admin/dataworks/scoring/evaluate
POST /admin/dataworks/similarity/check
GET/POST/DELETE /admin/dataworks/products
POST /admin/dataworks/products/{product_key}/{submit|approve|reject|archive}
```

React 상품 공장은 현재 실행 관찰 화면이며 새 실행 생성 UI가 아닙니다. 대신 `데이터 상품` 목록에서 상품을 생성·수정하고, 상품 작업 공간에서 제출·승인·반려·보관 상태 전환을 제어합니다. 화면의 삭제 동작은 `draft` 또는 `archived` 상태에서만 제공되며, 삭제한 상품 키는 보존된 과거 승인·계약 이력의 오연결을 막기 위해 재사용할 수 없습니다.

### 11.3 Product Canvas

```http
GET/POST /admin/dataworks/products/{product_key}/canvas
POST     /admin/dataworks/products/{product_key}/canvas/generate
```

고객 문제, 구매자, 사용 사례, 제공 데이터, 차별점, 가격 모델, 위험 참고, PoC 성공 기준과 예상 수익을 관리합니다. 상품 작업 공간의 `블루프린트` 탭에서 Canvas를 편집해 저장할 수 있습니다.

### 11.4 위험 검토와 승인

```http
POST /admin/dataworks/risk/check
GET  /admin/dataworks/reviews
POST /admin/dataworks/reviews/{product_key}/approve
POST /admin/dataworks/reviews/{product_key}/reject
GET/POST /admin/dataworks/products/{product_key}/approvals
```

필수 승인 step은 `data_owner`, `legal`, `compliance`입니다. 승인 `expires_at`이 지나면 출시 증적으로 인정되지 않습니다. React 검토 센터는 필터와 상품 이동을 제공하고, 연결된 상품 작업 공간의 `승인` 탭에서 승인 추적 항목을 추가하거나 기존 결정·증적·만료 정보를 갱신할 수 있습니다.

### 11.5 Evidence Pack

```http
GET/POST /admin/dataworks/products/{product_key}/evidence-pack
```

상품, Canvas, 원천 자산, 준비도, 위험 검토, 승인, PoC, API 계약, 계약 버전과 출시 게이트 결과를 감사 가능한 JSON으로 묶습니다. 본문 없이 `POST`하면 현재 저장 정보를 기준으로 생성합니다.

### 11.6 Publish Gate와 출시

`risk_score >= 70` 또는 `restricted`, `personal_credit`, `pseudonymized` 등 민감 프로필에는 엄격 게이트를 적용합니다.

- 모든 원천 자산 준비도 70 이상
- 데이터 오너·법무·준법 승인 또는 면제
- Evidence Pack 존재
- 승인 증적 미만료

```http
GET  /admin/dataworks/products/{product_key}/publish-gate
POST /admin/dataworks/products/{product_key}/publish
```

조건 미충족 시 `409 Conflict`와 차단 사유를 반환합니다.

![상품 출시 게이트](assets/screenshots/desktop/product/00-overview-release-gate.jpg)

## 12. 계약, Entitlement와 상품 API

```http
GET/POST /admin/dataworks/products/{product_key}/contract-versions
GET/POST /admin/dataworks/products/{product_key}/contract-scopes
GET/POST /admin/dataworks/products/{product_key}/entitlements
GET/POST /admin/dataworks/products/{product_key}/sla
GET/POST /admin/dataworks/products/{product_key}/watermarks
GET/POST /admin/dataworks/products/{product_key}/costs
POST     /v1/data-products/{product_key}/query
```

권장 순서:

1. 상품 작업 공간의 `계약` 탭에서 계약 정의를 새 버전으로 추가하고, 고객별 허용 필드, 호출 한도, 기간과 목적을 관리합니다.
2. Entitlement로 고객 API 키를 계약 Scope에 연결합니다.
3. SLA, 최신성 Watermark와 비용·마진을 설정합니다.
4. 런타임 상품 API를 호출해 출시, 계약 범위와 만료 검사를 확인합니다.

Product Workspace의 API·고객·사용량·수익·버전·활동 이력 탭은 현재 안내 화면입니다. 위 API가 React 탭에서 편집 가능하다고 가정하지 마세요.

계약 버전은 감사 가능성을 유지하기 위한 append-only 이력입니다. React UI와 관리 API는 새 버전 생성을 지원하지만 기존 계약 버전이나 과거 이력의 수정·삭제는 지원하지 않습니다.

## 13. OpenAPI와 개발자 문서

- 서비스 전체 명세: `/openapi.json`
- Swagger UI: `/swagger`
- 상품별 OpenAPI: `/admin/dataworks/products/{product_key}/openapi`
- 상품 런타임: `/v1/data-products/{product_key}/query`

서비스 전체 OpenAPI에는 개인 키 정책 변경과 회전 응답도 문서화되어 있습니다. 프로필 메뉴에서 전체 OpenAPI와 Swagger를 바로 열 수 있습니다.

## 14. Factory Run 재현성과 평가

```http
GET      /admin/dataworks/factory/runs
POST     /admin/dataworks/factory/runs/{id}/replay
POST     /admin/dataworks/factory/runs/{id}/evaluate
GET/POST /admin/dataworks/prompt-templates
```

재실행은 원본 `input_hash`와 부모 실행 계보를 보존합니다. 평가는 정확도, 유용성, 위험 통제와 출력 품질을 기록합니다. 현재 React 상품 공장 화면은 실행 생성·재생·평가 버튼이 아닌 조회 화면입니다.

## 15. 운영 점검표

### 일일

- `/health`, `/ready` 확인
- 관제실의 출시 차단, 승인 대기, 만료와 마진 경고 확인
- AI 공급자 오류와 MCP 도구 오류 확인
- 키 만료와 비정상 사용량 확인

### 변경 전

- PostgreSQL 백업
- 현재 이미지 태그와 서비스 버전 기록
- Keycloak 로컬 비상 로그인 확인
- 관리자 런타임 설정과 암호화 키 보관 상태 확인

### 변경 후

- 로그인과 프로필 버전 확인
- `/openapi.json`과 `/swagger` 확인
- 개인 키 발급→정책 변경→회전→폐기 시험
- Keycloak 연결 진단과 SSO 로그인 시험
- AI 스트리밍과 출력 토큰 상한 시험
- MCP `initialize`, `tools/list`와 정책 시험
- 상품 출시 게이트 차단·허용 경로 시험

## 16. 업그레이드와 롤백

새 릴리즈의 archive를 적재한 뒤 기존 네 환경변수를 유지하고 이미지 태그만 변경합니다. 기동 시 DB 마이그레이션이 실행되므로 먼저 PostgreSQL을 백업하세요.

롤백은 이전 이미지로 컨테이너를 다시 실행하는 방식이지만, DB 스키마 하위 호환성이 보장되는지 릴리즈 노트를 먼저 확인해야 합니다. 상세 절차는 [릴리즈 가이드](RELEASE_GUIDE.md)를 따릅니다.

## 관련 문서

- [사용자 가이드](USER_GUIDE.md)
- [운영 가이드](OPERATIONS.md)
- [안전 및 보안 가이드](SAFETY_GUIDE.md)
- [PostgreSQL 가이드](POSTGRES_GUIDE.md)
- [릴리즈 가이드](RELEASE_GUIDE.md)
