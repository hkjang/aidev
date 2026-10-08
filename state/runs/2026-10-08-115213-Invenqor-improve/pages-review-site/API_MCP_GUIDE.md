# Invenqor 자산 API·MCP·키 관리 가이드

대상 Server 버전: v0.2.45 · 기준일: 2026-08-24

## 1. 목적과 설계 원칙

Invenqor는 자동화 도구와 AI Agent가 같은 자산 원장을 안전하게 사용할 수 있도록
REST API와 MCP(Model Context Protocol)를 Server의 단일 HTTPS TCP `7070`에서
제공합니다.

- 브라우저 관리 콘솔은 Session+CSRF를 사용합니다.
- 자동화 REST API와 MCP는 별도의 scoped API key를 Bearer로 사용합니다.
- 키 원문은 생성·회전 응답에서 한 번만 보이고 DB에는 SHA-256 해시만 저장합니다.
- Key scope는 사용자 RBAC와 분리된 최소권한 경계이며 즉시 추가·교체·삭제할 수
  있습니다.
- 읽기·변경 권한을 분리하고 MCP는 현재 읽기 전용 자산 도구만 제공합니다.
- MCP 처리는 stateless이므로 Kubernetes의 어느 Server Pod로 요청해도 됩니다.

MCP 구현은 최신 안정 규격 `2026-07-28`의 무상태 JSON-RPC 2.0과 Streamable
HTTP를 기본으로 하고, 기존 연계의 무중단 업그레이드를 위해 `2025-11-25`
initialize 방식도 같은 URL에서 지원합니다. 최신 방식은 handshake와 protocol
session ID가 없고 모든 요청이 버전·client capability를 자체 포함하므로 sticky
session이나 Pod별 session 저장소가 필요 없습니다. 공식 전송 규격의 Origin 검증과
헤더/본문 일치 검증을 적용하고, SSE가 필요 없는 현재 도구 집합은 POST JSON
응답을 사용합니다.

Agent 자동 등록의 운영 API도 같은 Session+CSRF 경계에 있습니다. 관리자는
`/api/v1/admin/settings/agent-enrollment`에서 URL-only Open, Token 보호,
비활성 모드와 IP/CIDR allowlist·신뢰 프록시를 관리하고 `/token` 하위 경로에서
등록 Token을 발급·회전·폐기합니다. 정책은 공용 DB에서 요청마다 검증되어 모든
Server Pod에 즉시 적용되며 Token 원문은 발급 응답에서 한 번만 반환됩니다.
등록 성공 즉시 접속 IP의 host 자산이 생성되고 첫 system inventory가 같은
자산으로 병합됩니다.

Keycloak은 `/api/v1/admin/settings/keycloak/auto-configure`에 Keycloak URL,
Realm, Client ID, Client Secret과 InvenQor 외부 URL을 보내면 Discovery/TLS
검증, Callback/Logout URI와 표준 claim 구성을 거쳐 활성화할 수 있습니다.

멀티 Pod 진단은 `GET /api/v1/admin/diagnostics/logs`에서 공용 DB의 구조화
API·Agent access log와 warning/error, Agent 등록 이벤트를 조회합니다.
정상 health probe와 정적 UI asset은 제외하고 모든 실패 응답은 기록합니다.
`audit.read`가 필요하며 `level`, `component`, `instance_id`, `q`, `limit` 필터를
지원합니다. Agent 응답과 로그의 `request_id`를 `q`로 검색하면 처리 Pod와 실패
단계를 연결할 수 있습니다. Secret과 원문 인벤토리는 저장하지 않습니다.

주요 소프트웨어는 Agent가 수집한 process·service·software.package 원천을
Server 내장 카탈로그로 host별 정규화한 `software_product` 자산입니다. 제품명만
반환하는 블랙박스가 아니라 설치·실행 상태, 버전, 확신도, `runs_on` host 관계와
판별 evidence를 함께 제공하므로 CMDB와 AI가 결론의 근거를 확인할 수 있습니다.
카탈로그 `2026.08.1`은 인프라, PC 생산성·협업, 런타임, Endpoint 관리·보안의
51개 주요 제품을 다룹니다. Chrome·Java의 generic process 단독 신호는 제품으로
승격하지 않는 보수적 규칙으로 오탐을 억제합니다.
원시 process는 보존되지만 일상 관리용 `scope=managed` 조회에서는 제외할 수
있습니다.

상세 계약은 `openapi.yaml`을 따릅니다.

## 2. Scope 카탈로그

| Scope | 허용 작업 |
|---|---|
| `assets.read` | 자산 목록·검색·상세 조회 |
| `assets.write` | 수동 자산 생성·수정 |
| `assets.delete` | 논리 삭제·복원 |
| `relations.read` | 자산 관계 조회 |
| `relations.write` | 자산 관계 생성·삭제 |
| `agents.read` | Agent 상태 조회 |
| `queries.execute` | 검증된 Query DSL 실행 |
| `mcp.access` | `/mcp` 연결과 도구 탐색 |

`mcp.access`만 부여하면 MCP 연결은 가능하지만 자산 도구는 표시되지 않습니다.
예를 들어 `mcp.access + assets.read` 키에는 `asset_search`, `asset_get`,
`software_inventory`가 노출됩니다. `asset_relations`는 `relations.read`가 추가로
필요합니다. 키를 만드는 관리자도 자신이 갖지 않은 권한을 scope로 위임할 수
없습니다. `api_keys.manage` 자체는 API key에 위임할 수 없는 관리 Session
전용 권한입니다.

## 3. 키 수명주기 API

모든 관리 요청은 관리자 Session Cookie, `X-CSRF-Token`,
`api_keys.manage` 권한이 필요합니다.

브라우저 콘솔은 로컬 로그인 또는 Keycloak callback에서 발급된 SameSite CSRF
cookie를 상태 변경 요청의 `X-CSRF-Token`에 자동 연결합니다. CLI 예시에서는
cookie jar와 로그인 응답의 CSRF 값을 명시적으로 보관해야 합니다.

| Method | 경로 | 기능 |
|---|---|---|
| `GET` | `/api/v1/admin/api-key-scopes` | 허용 scope 카탈로그 |
| `GET` | `/api/v1/admin/api-keys` | 키 목록; 원문 제외 |
| `GET` | `/api/v1/admin/api-keys/{id}` | 키 상세; 원문 제외 |
| `POST` | `/api/v1/admin/api-keys` | 키 생성 |
| `PATCH` | `/api/v1/admin/api-keys/{id}` | 이름·scope 전체 교체 |
| `POST` | `/api/v1/admin/api-keys/{id}/scopes` | scope 추가 |
| `DELETE` | `/api/v1/admin/api-keys/{id}/scopes/{scope}` | scope 개별 삭제 |
| `POST` | `/api/v1/admin/api-keys/{id}/rotate` | 무중단 회전 |
| `DELETE` | `/api/v1/admin/api-keys/{id}` | 즉시 폐기 |

### 3.1 생성

```bash
curl -b cookies.txt -H "X-CSRF-Token: $CSRF" \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"cmdb-readonly",
    "scopes":["assets.read","relations.read","mcp.access"],
    "expires_at":"2027-01-01T00:00:00Z"
  }' \
  https://invenqor.example.com:7070/api/v1/admin/api-keys
```

응답의 `secret`은 안전한 Secret Manager에 즉시 저장하십시오. 목록·상세 API는
`prefix`, scope, 만료·마지막 사용 시각만 제공하며 원문을 다시 보여주지 않습니다.

### 3.2 Scope 추가·변경·삭제

```bash
# 추가
curl -b cookies.txt -H "X-CSRF-Token: $CSRF" \
  -H 'Content-Type: application/json' \
  -d '{"scopes":["agents.read"]}' \
  https://invenqor.example.com:7070/api/v1/admin/api-keys/$KEY_ID/scopes

# 전체 교체: 빈 배열이면 유효한 인증정보이지만 어떤 업무 권한도 없음
curl -X PATCH -b cookies.txt -H "X-CSRF-Token: $CSRF" \
  -H 'Content-Type: application/json' \
  -d '{"scopes":["assets.read","mcp.access"]}' \
  https://invenqor.example.com:7070/api/v1/admin/api-keys/$KEY_ID

# 개별 삭제
curl -X DELETE -b cookies.txt -H "X-CSRF-Token: $CSRF" \
  https://invenqor.example.com:7070/api/v1/admin/api-keys/$KEY_ID/scopes/agents.read
```

변경은 DB에서 직접 검증하므로 별도 cache 만료를 기다리지 않고 다음 요청부터
모든 Server Pod에 적용됩니다. scope 추가·삭제·전체 교체는 DB의 현재 revision을
조건으로 한 원자 변경이어서 두 Pod가 동시에 수정해도 한쪽 변경이 조용히
덮어써지지 않습니다. 경쟁이 반복되면 HTTP `409 API_KEY_CONFLICT`를 반환하므로
키 상세를 새로 읽고 의도한 scope를 다시 적용하십시오.

### 3.3 무중단 회전과 폐기

```bash
curl -b cookies.txt -H "X-CSRF-Token: $CSRF" \
  -H 'Content-Type: application/json' \
  -d '{"grace_seconds":3600}' \
  https://invenqor.example.com:7070/api/v1/admin/api-keys/$KEY_ID/rotate
```

새 `secret`을 소비자에 배포하고 1시간 안에 구 키를 제거합니다. 유예는
0~604800초(7일)이며 0이면 구 키가 즉시 무효화됩니다. Scope는 논리 Key에
연결되므로 구·신 키 모두 최신 scope 변경을 즉시 따릅니다. 여러 Pod에서 같은
키를 동시에 회전하면 먼저 반영된 새 Secret만 반환되고 stale 요청은
`409 API_KEY_CONFLICT`로 끝나므로, 사용할 수 없는 Secret이 성공 응답으로
노출되지 않습니다.

```bash
curl -X DELETE -b cookies.txt -H "X-CSRF-Token: $CSRF" \
  https://invenqor.example.com:7070/api/v1/admin/api-keys/$KEY_ID
```

폐기 시 현재·유예 키가 모두 즉시 거부됩니다. 소유 사용자가 비활성화되거나
삭제되어도 해당 키는 인증되지 않습니다.

## 4. 자산 REST API

기본 헤더:

```http
Authorization: Bearer ivq_sk_...
Accept: application/json
```

외부 자동화 전용 경로는 다음과 같습니다.

| Method | 경로 | Scope |
|---|---|---|
| `GET` | `/api/v1/external/assets` | `assets.read` |
| `GET` | `/api/v1/external/assets/{id}` | `assets.read` |
| `POST` | `/api/v1/external/assets` | `assets.write` |
| `PATCH` | `/api/v1/external/assets/{id}` | `assets.write` |
| `DELETE` | `/api/v1/external/assets/{id}` | `assets.delete` |
| `POST` | `/api/v1/external/assets/{id}/restore` | `assets.delete` |
| `GET` | `/api/v1/external/assets/{id}/relations` | `relations.read` |
| `POST/DELETE` | `/api/v1/external/assets/{id}/relations...` | `relations.write` |
| `POST` | `/api/v1/external/query/validate` | `queries.execute` |
| `POST` | `/api/v1/external/query/execute` | `queries.execute` |

검색 예:

```bash
curl -H "Authorization: Bearer $INVENQOR_API_KEY" \
  'https://invenqor.example.com:7070/api/v1/external/assets?q=ubuntu&type=host&status=active&limit=50'
```

Query DSL 실행이 한 번에 돌려주는 자산 수는 기본 100건이고 `limit`으로 최대
500건까지 조정합니다. 응답은 실제로 적용된 `limit`·`offset`과 함께, 조건에 맞는
자산 전체 수인 `total`, 다음 행이 실제로 있는지를 나타내는 `has_more`, 이어서
요청할 `next_offset`을 돌려줍니다. `truncated`는 `has_more`와 같은 값이며 offset이
없던 시절의 호출자를 위해 유지합니다.

한도를 넘는 결과는 `offset`으로 끝까지 순회합니다. `has_more`가 `true`인 동안
직전 응답의 `next_offset`을 그대로 넣어 다시 실행하십시오.

```bash
curl -H "Authorization: Bearer $INVENQOR_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"last_seen_at < \"now - 720h\"","limit":500,"offset":500}' \
  'https://invenqor.example.com:7070/api/v1/external/query/execute'
```

API key 인증에는 Cookie와 CSRF를 사용하지 않습니다. 키를 URL query, 로그,
지원 티켓, source control에 넣지 마십시오.

### 4.1 주요 소프트웨어와 관리 자산 범위

브라우저 관리 콘솔은 Session 인증과 `assets.read` 권한으로 다음 집계 API를
사용합니다.

```http
GET /api/v1/assets/software-products
GET /api/v1/assets?scope=managed
GET /api/v1/dashboard/statistics?scope=managed
```

`/api/v1/assets/software-products` query:

| 이름 | 값 | 의미 |
|---|---|---|
| `q` | 문자열 | 제품명/key, 역할, 제조사, 버전, host, 서비스·프로세스·패키지명 부분 검색 |
| `role` | catalog role | `database`, `web_server`, `security` 등 역할 일치 |
| `vendor` | 제조사 | 응답 `filters.vendors[]`에 나온 제조사 정확 일치 |
| `runtime_state` | `running`, `stopped`, `unknown` | 실행 상태 일치 |
| `confidence` | `high`, `review` | 각각 0.80 이상, 0.80 미만 |
| `limit` | 1~200 | 기본 50 |
| `offset` | 0 이상 | 페이징 시작 위치 |

응답의 `summary`는 고유 제품 수, host별 인스턴스, host 수, 실행·중지·미확인,
설치 확인·프로세스 관찰, 높은 확신도·검토 권장과 상위 제품을 제공합니다.
`items[]`의 안정 계약은 다음과 같습니다.

| 필드 | 의미 |
|---|---|
| `id`, `asset_key`, `status`, `last_seen_at` | host별 자산 식별자, 활성 상태와 최종 관찰 시각 |
| `product_key`, `product_name`, `role`, `vendor` | 제품 정체성과 운영 역할 |
| `version`, `versions[]` | 설치 패키지로 확인한 대표 버전과 관찰된 버전 목록 |
| `install_state` | `installed`, `observed`, `unknown` |
| `runtime_state` | `running`, `stopped`, `unknown` |
| `host` | 자동 `runs_on` 관계의 `{id,name}` |
| `service_names`, `process_names`, `package_names` | 제품 판별에 사용된 정규화 신호 |
| `executable_paths` | 서비스 인자를 제거한 실행 경로 |
| `evidence[]` | `kind`, `name`, `source_asset_id`의 설명 가능한 근거 |
| `detection_method`, `catalog_version`, `confidence` | 판별 방식·카탈로그 버전과 0~1 근거 확신도 |
| `evidence_count`, `process_count` | 상세 반환 제한과 무관한 전체 근거·매핑 process 개수 |

`scope=managed`는 자산 목록과 Dashboard 집계에서 `type=process`만 제외합니다.
원시 프로세스를 삭제하거나 수집 중단하지 않으며, `scope`를 생략하면 기존 전체
자산 API 의미가 그대로 유지됩니다.

전용 주요 소프트웨어 요약 경로는 현재 관리 Session용입니다. API key 연계는
외부 자산 API에서 정규화 자산을 직접 조회합니다.

```bash
curl -H "Authorization: Bearer $INVENQOR_API_KEY" \
  'https://invenqor.example.com:7070/api/v1/external/assets?type=software_product&status=active&limit=100'
```

각 자산의 `attributes`에 위 제품 상태와 evidence가 들어 있고 관계 API에서
`runs_on` host를 조회할 수 있습니다. MCP에서는 전용 `software_inventory`를
우선 사용하면 제품 요약, host, 상태, 확신도와 evidence를 한 번에 읽을 수
있습니다. 자산 원본과 전체 관계가 필요하면 `asset_get`, `asset_relations`를 이어서
호출합니다. AI는 `confidence`를 보안 위험도나 취약점 점수로 오해하지 말고
제품 식별의 증거 강도로 취급해야 합니다.

## 5. MCP 연결

MCP URL은 다음 하나입니다.

```text
https://invenqor.example.com:7070/mcp
```

클라이언트에는 `Authorization: Bearer <API key>`를 Secret 환경변수 또는
지원되는 secure header 설정으로 전달합니다. 일반적인 설정 개념:

```json
{
  "transport": "streamable-http",
  "url": "https://invenqor.example.com:7070/mcp",
  "headers": {
    "Authorization": "Bearer ${INVENQOR_API_KEY}"
  }
}
```

### 5.1 최신 `2026-07-28` 무상태 연결

최신 client는 initialize 없이 `server/discover`로 Server capability를 확인하거나
바로 도구를 호출합니다. 모든 POST에는 다음 계약이 적용됩니다.

| 위치 | 값 | 규칙 |
|---|---|---|
| Header | `MCP-Protocol-Version: 2026-07-28` | 모든 최신 요청에서 필수 |
| Header | `Mcp-Method` | JSON-RPC `method`와 정확히 일치 |
| Header | `Mcp-Name` | `tools/call`에서 필수이며 `params.name`과 일치. `tools/list`, `server/discover`에서는 생략 |
| `params._meta` | `io.modelcontextprotocol/protocolVersion` | Header와 같은 `2026-07-28` |
| `params._meta` | `io.modelcontextprotocol/clientCapabilities` | 객체. 기능이 없으면 `{}` |
| `params._meta` | `io.modelcontextprotocol/clientInfo` | client 이름·버전. 권장 값 |

Gateway는 body를 열지 않고 `Mcp-Method`와 `Mcp-Name`으로 routing·rate limit을
적용할 수 있습니다. Server는 Header가 없거나 body와 다르면 HTTP 400과 JSON-RPC
`-32020`을 반환해 잘못 라우팅된 요청을 실행하지 않습니다.

Capability 탐색 예:

```bash
curl -H "Authorization: Bearer $INVENQOR_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -H 'MCP-Protocol-Version: 2026-07-28' \
  -H 'Mcp-Method: server/discover' \
  -d '{
    "jsonrpc":"2.0","id":1,"method":"server/discover",
    "params":{"_meta":{
      "io.modelcontextprotocol/protocolVersion":"2026-07-28",
      "io.modelcontextprotocol/clientCapabilities":{},
      "io.modelcontextprotocol/clientInfo":{
        "name":"asset-assistant","version":"1.0"
      }
    }}
  }' \
  https://invenqor.example.com:7070/mcp
```

응답의 `supportedVersions`는 `2026-07-28`이며, 모든 최신 성공 결과는
`resultType: complete`와
`_meta.io.modelcontextprotocol/serverInfo`를 포함합니다. `server/discover`와
`tools/list`는 scope별 결과가 사용자 간 공유되지 않도록 `ttlMs: 0`,
`cacheScope: private`을 반환합니다.

### 5.2 기존 `2025-11-25` client 호환

기존 client는 이전과 같이 initialize를 사용합니다. Server는 `2025-11-25`로
협상하고 후속 요청을 처리하므로 기존 설정을 즉시 바꿀 필요는 없습니다.

```bash
curl -H "Authorization: Bearer $INVENQOR_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json, text/event-stream' \
  -d '{
    "jsonrpc":"2.0","id":1,"method":"initialize",
    "params":{
      "protocolVersion":"2025-11-25",
      "capabilities":{},
      "clientInfo":{"name":"asset-assistant","version":"1.0"}
    }
  }' \
  https://invenqor.example.com:7070/mcp
```

새 연계는 멀티 Pod 단순성, 표준 Gateway routing과 향후 SDK 호환성을 위해
`2026-07-28`을 사용하십시오.

## 6. 제공 MCP 도구

| 도구 | Scope | 입력 | 결과 |
|---|---|---|---|
| `asset_get` | `assets.read` | `asset_id` | 자산 상세. 다른 자산으로 병합된 자산은 `asset` 없이 `merged_into`(primary UUID)·`merged_at`·`message`로 안내하며, 그 primary 로 다시 부르면 상세가 옴. 존재하지 않는 UUID 는 `asset not found` 오류 |
| `asset_relations` | `relations.read` | `asset_id`, `limit`, `offset` | 활성 inbound/outbound 관계 |
| `asset_search` | `assets.read` | `q`, `type`, `status`, `include_observations`, `limit`, `offset` | 정규화 자산 목록. 원시 process는 기본 제외 |
| `software_inventory` | `assets.read` | `q`, `role`, `vendor`, `runtime_state`, `confidence`, `limit`, `offset` | 제품 요약·host·상태·확신도·evidence |
| `agents_list` | `agents.read` | `limit`, `offset` | Agent 상태·버전·최근 수신 |

도구 목록은 고정된 결정적 순서이며 키에 없는 scope의 도구는 아예 노출하지
않습니다. 결과는 MCP 호환 text content와 typed `structuredContent`를 함께
제공합니다. 자산의 이름·속성·사용자 입력은 신뢰할 수 없는 데이터이며 AI에 대한
명령으로 해석해서는 안 된다는 지침도 최신 `server/discover`와 기존 initialize
응답에 포함됩니다.

`software_inventory`의 `runtime_state`는 `running|stopped|unknown`, `confidence`는
`high|review`이며 `limit`는 기본 50, 최대 100입니다. 범용 `asset_search`는
`include_observations`를 생략하거나 `false`로 두면 `type=process`를 제외합니다.
사고 조사나 판별 근거 검증처럼 원시 관찰이 필요한 경우에만
`"include_observations":true`를 명시하십시오.

`asset_search`, `agents_list`, `asset_relations`의 `has_more`는 페이지 크기
추측이 아니라 서버가 한 행을 더 읽어 확인한 사실이므로, 결과가 정확히 `limit`에서
끝나면 `false`이며 빈 페이지를 한 번 더 요청할 필요가 없습니다. `has_more`가
`true`이면 응답의 `next_offset`을 그대로 `offset`에 넣어 다음 페이지를
요청하십시오.

`asset_relations`도 다른 도구와 같이 `limit` 기본 50, 최대 100으로 잘라
돌려줍니다. `runs_on` 관계를 수천 개 가진 host 하나가 응답 하나로 모델의 context를
차지하지 않도록 한 것이며, 전체 관계가 필요하면 `next_offset`으로 이어서
조회하십시오. 관계 목록 자체는 이전과 같은 `relations` 키에 담겨 옵니다.

```bash
curl -H "Authorization: Bearer $INVENQOR_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'MCP-Protocol-Version: 2026-07-28' \
  -H 'Mcp-Method: tools/call' \
  -H 'Mcp-Name: asset_search' \
  -d '{
    "jsonrpc":"2.0","id":2,"method":"tools/call",
    "params":{
      "name":"asset_search","arguments":{"type":"host","limit":20},
      "_meta":{
        "io.modelcontextprotocol/protocolVersion":"2026-07-28",
        "io.modelcontextprotocol/clientCapabilities":{},
        "io.modelcontextprotocol/clientInfo":{
          "name":"asset-assistant","version":"1.0"
        }
      }
    }
  }' \
  https://invenqor.example.com:7070/mcp
```

주요 소프트웨어 인벤토리 조회 예:

```bash
curl -H "Authorization: Bearer $INVENQOR_API_KEY" \
  -H 'Content-Type: application/json' \
  -H 'MCP-Protocol-Version: 2026-07-28' \
  -H 'Mcp-Method: tools/call' \
  -H 'Mcp-Name: software_inventory' \
  -d '{
    "jsonrpc":"2.0","id":3,"method":"tools/call",
    "params":{
      "name":"software_inventory","arguments":{
        "runtime_state":"running","confidence":"high","limit":50
      },
      "_meta":{
        "io.modelcontextprotocol/protocolVersion":"2026-07-28",
        "io.modelcontextprotocol/clientCapabilities":{},
        "io.modelcontextprotocol/clientInfo":{
          "name":"asset-assistant","version":"1.0"
        }
      }
    }
  }' \
  https://invenqor.example.com:7070/mcp
```

MCP는 임의 SQL, shell, 원격 명령, 자산 변경 도구를 제공하지 않습니다. 변경
작업은 명시적 REST scope와 조직 승인 절차를 사용합니다.

## 7. 보안·감사·운영

- 모든 키 생성, 이름/scope 변경, 회전과 폐기는 감사 로그에 actor, 대상 ID,
  변경 전후 값과 request ID를 남깁니다. Secret 원문은 감사 로그에 넣지 않습니다.
- MCP는 `Origin`이 있으면 요청 Host와 일치하는지 확인해 DNS rebinding을
  차단합니다. 운영 TLS Proxy도 원래 Host를 보존해야 합니다.
- Key 별 Server rate limit 외에 Ingress/API Gateway에서 전역 rate limit,
  허용 IP, 최대 body와 timeout을 적용하십시오.
- 각 연계 시스템마다 별도 Key를 만들고 공유 Key를 금지합니다.
- 만료일은 90일 이하, 회전 유예는 실제 배포시간에 필요한 최소값을 권장합니다.
- Secret 탐지에 걸린 Key는 유예 없이 폐기하고 새 Key를 발급합니다.
- PostgreSQL과 감사 로그를 백업하되 Secret Manager의 원문 Key와 함께
  평문으로 내보내지 않습니다.

멀티 파드에서 키·scope·회전 상태는 PostgreSQL을 단일 원장으로 사용합니다.
최신 MCP 요청은 protocol version과 client capability가 매 요청의 `_meta`에 있고
Server는 `Mcp-Session-Id`를 발급하지 않습니다. Pod별 메모리 session이나 sticky
routing을 요구하지 않으므로 rolling update 중에도 같은 MCP client가 어느 Pod로든
요청할 수 있습니다.

## 8. 장애 코드

| 상태 | 코드 | 조치 |
|---|---|---|
| 401 | `INVALID_API_KEY` | 오타, 만료, 폐기, 소유 사용자 상태 확인 |
| 403 | `FORBIDDEN` | 필요한 scope 추가 또는 별도 Key 발급 |
| 403 | `MCP_ORIGIN_REJECTED` | Proxy Host/Origin 전달 정책 확인 |
| 403 | `SCOPE_ESCALATION` | 관리자의 RBAC 권한 범위 안에서만 위임 |
| 409 | `API_KEY_CONFLICT` | 키 상세를 새로 읽고 동시 scope 변경 또는 회전을 재시도 |
| 429 | `API_RATE_LIMITED` | 호출 빈도 감소, Gateway 정책 확인 |
| 400 | JSON-RPC `-32020` | 최신 요청의 protocol/method/name Header와 body 값을 일치시킴 |
| 400 | JSON-RPC `-32602` | 요청별 `_meta`에 `clientCapabilities` 객체를 포함하고 metadata 형식을 확인 |
| 200 | JSON-RPC `-32602` | 도구 이름·입력 형식을 확인 |
| 400 | JSON-RPC `-32022` | 오류 data의 지원 버전 목록에 맞춰 protocol version을 재선택 |

MCP protocol 오류는 JSON-RPC error로, 도구 입력·실행 오류는 `isError: true`
tool result로 반환해 AI client가 안전하게 수정 요청을 만들 수 있게 합니다.

## 9. 참고 규격

- [MCP 2026-07-28 릴리즈](https://blog.modelcontextprotocol.io/posts/2026-07-28/)
- [Stateless MCP와 버전 협상](https://modelcontextprotocol.io/seps/2575-stateless-mcp)
- [HTTP Header 표준화](https://modelcontextprotocol.io/seps/2243-http-standardization)
- [List 결과 Cache Hint](https://modelcontextprotocol.io/seps/2549-TTL-for-list-results)
- [MCP 2025-11-25 Streamable HTTP](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)
