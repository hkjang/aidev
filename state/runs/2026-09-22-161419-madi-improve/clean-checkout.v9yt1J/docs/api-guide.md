# REST API · MCP 연동 가이드

기본 REST 경로는 `/api/v1`입니다. JSON 필드명은 `snake_case`입니다. 성공 응답은 객체 또는 배열이며 오류는 `{"error":"메시지","request_id":"요청 ID"}` 형식입니다. 요청 ID는 응답에 있을 때 운영 로그 조회에 사용합니다.

실행 중인 서비스의 `/api/v1/openapi.json`에서 기계가 읽을 수 있는 OpenAPI 경로 목록과 계약을 조회할 수 있습니다.

외부 알림의 개인 구독·전송 이력, 이메일 수집과 공개 HMAC 수집 엔드포인트는 [알림·수집 가이드](notification-capture-guide.md)를 확인하세요. `/capture-hooks/{id}`는 Bearer 대신 채널별 HMAC 서명을 요구하며 수락 후 작업 큐에서 비공개 문서로 저장합니다.

## 인증과 권한

웹 화면은 동일 출처 쿠키 세션을 사용합니다. 외부 API·MCP 클라이언트는 개인 또는 서비스 계정 키를 `Authorization: Bearer <TOKEN>`에 넣습니다. 변경 요청에는 `X-Madi-Request: 1` 헤더를 추가합니다. 쿠키 기반 요청은 출처 검사도 적용됩니다.

키는 워크스페이스를 지정해야 하며 소유자의 권한을 넘을 수 없습니다. 문서 접근 범위, 키 워크스페이스, API 범위, 만료, IP 및 호출 제한이 함께 적용됩니다.

| 범위 | 용도 |
| --- | --- |
| `document:read` | 문서·버전·백링크 조회 |
| `document:write` | 문서 작성·수정·삭제 |
| `database:read` | 데이터베이스·행 조회 |
| `database:write` | 데이터베이스·행 변경 |
| `search:read` | 접근 가능한 검색 메타데이터와 제한된 스니펫. 전체 원문·그래프는 `document:read` 필요 |
| `ai:execute` | AI 스트리밍 요청 |

API 키로 개인 설정·키 재발급·서비스 관리자 권한을 획득할 수는 없습니다. 관리 API와 개인 키 발급은 권한을 가진 사용자 세션에서 수행합니다. 키 비밀은 발급·회전 응답에서 한 번만 반환됩니다. 회전하면 구 비밀은 즉시 폐기됩니다.

## 자주 사용하는 REST 경로

| 메서드 | 경로 | 설명 |
| --- | --- | --- |
| GET | `/public` | 서비스 이름·버전·로그인 선택지 |
| POST | `/auth/login` | 로컬 로그인 `{email,password}` |
| POST | `/auth/logout` | 로그아웃 |
| GET | `/auth/me` | 현재 사용자 |
| GET, PUT | `/profile` | 개인 정보·환경설정 |
| GET, POST | `/workspaces` | 워크스페이스 목록·생성 |
| GET, PUT | `/workspaces/{id}/members` | 멤버 조회·추가·역할 변경 |
| GET, POST | `/documents` | 크기가 제한된 문서 요약 목록·작성 (원문은 개별 GET) |
| GET, PUT, DELETE | `/documents/{id}` | 문서 조회·수정·휴지통 이동 |
| GET | `/documents/{id}/backlinks` | 백링크 |
| GET | `/documents/{id}/versions` | 버전 목록 |
| POST | `/documents/{id}/versions/{version}/restore` | 이전 버전 복구 |
| POST | `/documents/{id}/restore` | 휴지통 복구 |
| POST | `/documents/{id}/favorite` | 즐겨찾기 전환 |
| GET, POST | `/documents/{id}/comments` | 문서 댓글 조회·작성 |
| POST | `/documents/{id}/approval` | 활성화된 검토 절차에서 submit/approve/reject |
| GET | `/graph?workspace_id={id}` | 권한이 적용된 문서 연결 그래프 |
| GET, PUT | `/tasks` | Markdown 할 일 조회·완료 상태 변경 |
| GET, POST | `/databases` | 데이터베이스 목록·생성 |
| GET, PUT, DELETE | `/databases/{id}` | 데이터베이스·속성 관리 |
| GET, POST | `/databases/{id}/rows` | 행 조회·추가 |
| PUT, DELETE | `/databases/{id}/rows/{rowId}` | 행 수정·삭제 |
| POST | `/attachments?document_id={id}` | multipart `file` 첨부 |
| GET | `/attachments/{id}` | 권한 확인 후 다운로드 |
| GET | `/export?workspace_id={id}` | Markdown ZIP |
| POST | `/import?workspace_id={id}` | multipart `file`: `.md` 또는 `.zip` |
| GET, POST | `/keys` | 키 메타데이터 목록·발급 |
| PUT, DELETE | `/keys/{id}` | 키 정책 변경·폐기 |
| POST | `/keys/{id}/rotate` | 키 회전 |
| POST | `/ai/chat` | SSE AI 응답 |
| POST | `/mcp` | JSON-RPC MCP |

문서 목록에는 `workspace_id`, `q`, `tag`, `trash=1`, `favorite=1` 쿼리를 사용할 수 있습니다. 문서 수정에는 읽을 때 받은 `version`을 함께 전송하세요. 충돌 시 `409`가 반환됩니다. 토큰을 가진다고 모든 표의 경로를 사용할 수 있는 것은 아니며 각 경로의 세션·범위 조건이 적용됩니다.

```sh
curl --fail-with-body 'https://madi.example.internal/api/v1/documents?workspace_id=WORKSPACE_UUID' \
  -H "Authorization: Bearer $MADI_API_TOKEN"

curl --fail-with-body 'https://madi.example.internal/api/v1/documents' \
  -H "Authorization: Bearer $MADI_API_TOKEN" \
  -H 'Content-Type: application/json' -H 'X-Madi-Request: 1' \
  --data '{"workspace_id":"WORKSPACE_UUID","title":"운영 점검","markdown":"# 운영 점검\n\n- [ ] 상태 확인"}'
```

예시의 `MADI_API_TOKEN`은 호출하는 클라이언트의 비밀 주입 예시이며 madi 서비스 실행 환경변수에 추가하는 항목이 아닙니다.

## 스트리밍 AI

```sh
curl --no-buffer --fail-with-body 'https://madi.example.internal/api/v1/ai/chat' \
  -H "Authorization: Bearer $MADI_API_TOKEN" \
  -H 'Content-Type: application/json' -H 'X-Madi-Request: 1' \
  --data '{"prompt":"현재 문서의 핵심을 정리해 주세요","document_id":"DOCUMENT_UUID","workspace_id":"WORKSPACE_UUID"}'
```

응답은 `text/event-stream`이며 데이터 이벤트를 순서대로 처리합니다.

```text
data: {"sources":[{"id":"DOCUMENT_UUID","title":"운영 점검"}]}

data: {"text":"문서의 주요 내용은 "}

data: {"text":"다음과 같습니다."}

data: [DONE]
```

오류는 `{"error":"메시지"}` 이벤트로 전달될 수 있습니다. 네트워크 청크 경계와 이벤트 경계는 다르므로 줄·이벤트 버퍼를 유지하는 SSE 파서를 사용하세요. 클라이언트가 취소하면 연결을 종료합니다. 최대 출력 토큰은 관리자 설정에서 1~262,144로 정하며 제공자 모델의 실제 한도가 우선합니다.

## MCP

접속 주소는 `https://madi.example.internal/mcp` 또는 `https://madi.example.internal/api/v1/mcp`입니다. Bearer 키가 필수이며 HTTP POST JSON-RPC의 상태 없는 JSON 응답 방식을 사용합니다. stdio 서버 또는 별도의 legacy SSE 세션 서버는 아닙니다. 클라이언트에서 원격 HTTP MCP와 Authorization 헤더를 설정하세요.

지원 프로토콜 버전은 `2025-11-25`, `2025-06-18`, `2025-03-26`입니다. `initialize`, `notifications/initialized`, `tools/list`, `tools/call`을 제공합니다. 클라이언트가 요구하는 인증·전송 방식과 호환되는지 연결 시험을 먼저 수행하세요.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {},
    "clientInfo": {"name": "company-agent", "version": "1.0"}
  }
}
```

사용 가능한 도구와 입력 스키마는 `tools/list`에서 조회합니다.

| 도구 | 역할 |
| --- | --- |
| `search_documents`, `search_workspace` | 통합 검색의 메타데이터·최대 600자 스니펫. 전체 원문은 별도 `get_document` |
| `get_document` | 문서 조회 |
| `create_document`, `update_document`, `delete_document` | 문서 변경 |
| `get_backlinks`, `get_graph` | 문서 연결 탐색 |
| `list_databases`, `query_database` | 데이터베이스 조회 |
| `create_database_row` | 데이터베이스 행 추가 |

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "search_documents",
    "arguments": {"query": "운영", "workspace_id": "WORKSPACE_UUID"}
  }
}
```

MCP에도 REST와 같은 사용자 ACL, 워크스페이스 및 키 범위를 적용합니다. 검색 결과에 없는 문서를 ID로 직접 지정해도 권한 검사를 우회할 수 없습니다.

두 MCP 검색 도구는 `/search` 통합 검색을 사용하며 결과는 `results`, `has_more`, `next_offset`, `available_types`를 가진 객체입니다. 각 검색 결과는 제목·종류·URL·메타데이터와 최대 600문자 스니펫만 포함합니다. 문서 전체 원문은 `document:read` 권한으로 `get_document`를 호출해야 합니다. 데이터베이스·행 검색 결과에는 `database:read`도 필요하며 개인 AI 대화나 사용자 목록은 키 검색에 노출하지 않습니다.

### 문서 목록과 원문 구분

`GET /spaces/{id}/documents`도 같은 요약 형식이며 `document:read`가 필요합니다. `database:read` 키는 공간 목록을 탐색할 수 있지만 공간 안의 문서를 열람할 수 없습니다.

`GET /documents`는 `q` 유무와 관계없이 `document:read`를 요구합니다. 기본 100개, `limit` 최대 2,000개, `offset` 페이지를 지원하지만 응답은 탐색용 `DocumentSummary` 배열입니다. `markdown`과 `block_metadata`는 포함하지 않습니다. 카드 미리보기는 최대 160문자의 `excerpt`를 사용하세요.

태그와 alias는 각각 UTF-8 200바이트·JSON 인코딩 256바이트 이하인 실제 값 최대 32개만 표시합니다. 문자열을 잘라 새로운 alias로 만들지 않으며 생략 시 `tags_truncated`/`aliases_truncated`가 `true`입니다. 문서의 Markdown 원문과 모든 태그·alias·블록 속성은 `GET /documents/{id}`에서 그대로 조회합니다. 따라서 목록 데이터를 문서 정본으로 저장하지 마세요. 검색어 원문은 공용 감사 로그에 남기지 않습니다.

## 관리자와 운영 API

정보보호 설정은 `/admin/information-protection`의 `GET`과 `{revision,settings}`를 받는 `PUT`으로 관리합니다. 정본 저장에서 정책 위반은 `422`와 `code`, 검출 종류·개수로 응답하며 실제 검출 값은 반환하지 않습니다. 공개 공유는 소유자 세션의 `/documents/{id}/public-shares`에서 명시적으로 생성합니다. 방문자 API `/public-shares/{share}`는 쿠키·Bearer 대신 `X-Madi-Share-Token` 헤더가 필수이고, 암호가 있으면 `/unlock`에서 받은 `access_token`을 `X-Madi-Share-Access`에 추가합니다. 비밀을 URL query에 넣지 마세요. [상세 가이드](information-protection-guide.md)와 `/api/v1/openapi.json`의 헤더·본문 계약을 참고하세요.

관리자는 `/admin/stats`, `/admin/users`, `/admin/settings`, `/admin/audit`, `/admin/settings/history`, `/admin/backup`을 사용합니다. `GET /admin/backup`으로 만든 논리 ZIP은 `POST /admin/restore`에서 복원할 수 있습니다. 복원 요청은 관리자 세션과 multipart `file`, `confirmation=RESTORE`를 요구하며 현재 데이터를 교체하고 세션을 폐기합니다. 같은 madi 버전과 ENCRYPTION_KEY가 필요합니다. ZIP 256MB, 압축 해제 1GB, 10,000개 항목이 상한입니다.

관리자 설정의 OIDC·AI 비밀은 저장 시 암호화되며 조회 결과에는 구성 여부만 표시됩니다. 프로세스 점검은 API 접두사가 없는 `/healthz`, `/readyz`, `/metrics`입니다.

정확한 요청 스키마와 상태 코드는 현재 버전의 [API 계약](../BUILD_PLAN.md) 및 서버 구현을 함께 확인하세요. 이 문서에서 소개하는 REST API는 현재의 `v1` 인터페이스이며 별도의 API 호환성 장기 보장 정책이 수립된 것은 아닙니다.
