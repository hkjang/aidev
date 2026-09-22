# 문서 안의 선언형 조회 표

문서 읽기 화면에서 현재 접근 가능한 문서 속성·할 일·명시적 관계를 표로 확인합니다. `madi-query` 코드 블록의 JSON 정의가 Markdown 원본이며 결과는 일시적인 읽기 화면입니다. SQL, JavaScript, `eval`, 임의 JSONPath, 외부 URL 호출과 셸 실행은 지원하지 않습니다.

## 작성과 실행

1. 문서 편집 도구의 **선언형 조회 표**를 엽니다.
2. 문서 속성·할 일·문서 관계 중 조회 대상과 표시할 열을 고릅니다. 대상 선택을 바꾸면 이전 대상의 열을 초기화합니다.
3. 필요하면 최상위 Front Matter 속성과 실행 시 입력받을 문구를 지정합니다. **조회 정의 넣기**는 JSON 정의만 기존 편집기에 삽입합니다. 저장·공동 편집·승인은 기존 문서 절차를 따릅니다.
4. 정본 문서의 읽기 화면에서 매개변수를 입력하고 **조회 실행**을 누릅니다. 읽기 화면 방문만으로 표를 실행하지 않습니다.
5. 표의 원문 링크에서 버전·해당 위치를 미리 봅니다. **새로 조회**를 눌러 새 문서와 새 행을 포함해 다시 계산합니다.

조회 결과는 Markdown 본문, 문서 버전, AI 답변, 공개 공유, 승인 스냅샷, Markdown 내보내기에 저장하지 않습니다. 인쇄에서는 동적 결과를 숨깁니다. 동기화 블록·AI 출력·템플릿/문서 미리보기에서도 정의만 표시하며 자동 실행하지 않습니다. 이미 사람이 복사한 정보의 외부 사본을 회수하는 기능은 아닙니다.

## 문법 예시

````markdown
```madi-query
{
  "version": 1,
  "source": "documents",
  "columns": [
    { "field": "title", "label": "문서" },
    { "field": "status", "label": "상태" },
    { "field": "property.점검일", "label": "점검일" }
  ],
  "properties": { "점검일": "date" },
  "parameters": {
    "검색어": { "type": "string", "label": "포함할 문구", "required": true }
  },
  "filters": [
    { "field": "title", "op": "contains", "value": { "parameter": "검색어" } }
  ],
  "order_by": [{ "field": "updated_at", "direction": "desc" }],
  "limit": 20
}
```
````

지원 필드는 다음과 같습니다. 원문 전체·암호·개인 AI 이력·API 키 속성 등은 조회 필드가 아닙니다.

| 대상 | 필드 |
| --- | --- |
| `documents` | `id`, `title`, `status`, `tags`, `owner_id`, `owner_name`, `created_at`, `updated_at`, `classification`, `kind`, `version`, 선언한 `property.이름` |
| `tasks` | `document_id`, `document_title`, `text`, `done`, `status`, `priority`, `assignee_id`, `assignee_name`, `due_date`, `version`, `line` |
| `relations` | `source_id`, `source_title`, `target_id`, `target_title`, `relation_type`, `depth` |

문서의 `properties`는 Front Matter **최상위 스칼라**만 읽습니다. 유형은 `string`, `number`, `boolean`, `date`입니다. YAML 별칭·중첩 객체·배열을 펼치거나 문자열을 다른 유형으로 추정 변환하지 않습니다. 형식이 맞지 않거나 한도를 넘은 속성은 빈 값과 제한 안내로 표시합니다. 날짜는 `YYYY-MM-DD` 또는 RFC3339를 받습니다. 숫자는 유한한 JavaScript 안전 정수 범위 내 수만 허용합니다(소수 지원).

필터는 최대 8개의 AND 조건입니다. `eq`, `ne`, `in`, `exists`, 문자열 `contains`, 숫자·날짜의 `lt/lte/gt/gte`를 지원합니다. `in`은 같은 유형의 리터럴 1~20개이며 매개변수 배열은 받지 않습니다. `contains`는 문자열에서 대소문자 구분 부분 일치, 태그 배열에서는 정확한 태그 일치입니다. `exists`는 값이 없는 조건으로 null/빈 문자열을 제외합니다. 없는 값은 `ne`를 포함한 비교에 일치하지 않습니다. 정렬은 2개까지이며 누락 값은 뒤로 갑니다.

선택적인 `space_id`는 해당 공간의 문서만, `document_ids`는 같은 워크스페이스의 최대 50개 지정 문서만 대상으로 삼습니다. 관계 조회의 기본 출발점은 조회 정의가 있는 문서이며 `document_ids`로 출발점을 지정할 수도 있습니다. `direction`은 `outgoing` 또는 `incoming`, `depth`는 1~3입니다. 경로의 양끝뿐 아니라 **모든 중간 문서**를 현재 ACL로 검사하고, 접근할 수 없는 노드 뒤로 탐색하지 않습니다. 관계 결과의 정렬/제한은 발견한 안전한 경로 후보 안에서 적용됩니다.

## 범위와 현재성

| 항목 | 한도 |
| --- | --- |
| 정의 JSON | 5 KiB, 중첩 12단계, 정확한 필드 이름·중복 키 거부 |
| 한 문서의 정의 | 처음 16개 실제 Markdown 코드 블록 |
| 정의를 해석할 부모 문서 | 1 MiB |
| 후보 | 최대 2,000개, 최근 수정 문서 우선 검사 |
| 읽어 분석할 원문·메타데이터 | 합계 8 MiB, 할 일/속성 본문 문서당 256 KiB |
| 속성 Front Matter | 32 KiB, 선언 속성/매개변수 각각 8개 |
| 표 | 최대 12열·100행·1 MiB 응답 행 데이터 |
| 동시 실행 | 서비스 프로세스당 4개, 포화 시 429 |
| 시간 예산 | 요청 2초·SQL 문장 1.9초, 초과/취소 시 결과 폐기 |
| 결과 검증 토큰 | 현재 계정·세션/키에 바인딩된 암호화 토큰, 10분 |

후보/바이트/행 한도, 큰 본문 제외, 잘린 필드, 해석하지 못한 속성을 결과에 표시합니다. 태그는 원형을 보존하는 최대 32개(각 200바이트), 제목은 최대 500문자, 할 일 문구는 최대 500문자로 제한합니다. 이 제한 안의 정렬은 **전체 조직 자료에 대한 전역 순위·전체 결과를 뜻하지 않습니다**. 후보 탐색과 CommonMark 해석 시간은 부하·본문 구조의 영향을 받으므로 2초 예산을 최악 지연 SLA로 표현하지 않습니다.

화면이 보일 때 3초 주기로 반환한 행과 사용한 경로의 현재 문서 ACL·버전/해시·할 일 속성·관계·보호 정책을 재검사합니다. 권한 회수, 삭제, 원문 또는 표시 속성 변경, 기능 비활성화, 세션 만료 시 결과를 지웁니다. 화면을 숨기거나 문서/사용자/매개변수를 바꾸어도 결과를 폐기합니다. 이는 새로운 후보를 자동 추가하는 실시간 전체 조회가 아닙니다. 더 최신의 전체 후보는 명시적으로 다시 실행하세요. 네트워크 지연과 브라우저 스케줄링 때문에 철회 표시가 정확히 3초 이내라는 SLA는 아닙니다.

## API와 MCP

- `GET /api/v1/documents/{id}/queries`: 현재 정의와 `hash`, `document_version` 조회.
- `POST /api/v1/documents/{id}/queries/execute`: `{ "document_version": 3, "query_hash": "…", "parameters": { "검색어": "운영" } }` 실행. 원문에 같은 hash의 유효한 정의가 있어야 하며 임의 정의를 요청 본문으로 받을 수 없습니다.
- `POST /api/v1/documents/{id}/queries/check`: `{ "validation_token": "…" }`로 표시 결과의 현재성 확인. 행 데이터를 다시 반환하지 않습니다.
- MCP `list_document_queries`, `execute_document_query`: 위 REST 경로를 그대로 거칩니다.

모두 `document:read`가 필요하고 API 키의 현재 워크스페이스·만료·스코프·IP 제한을 적용합니다. 읽기 전용 POST도 일반 세션에서 CSRF 검증을 거칩니다. 서비스 관리자는 다른 사용자의 private 문서를 우회하지 못합니다. 서비스 계정은 부여된 키와 현재 문서 권한으로만 사용할 수 있으며 플러그인 브리지에는 이 조회 도구를 노출하지 않습니다.

관리자는 기존 기능 관리의 `document-queries`로 서비스를 차단할 수 있습니다. 서비스·워크스페이스·사용자 false 설정은 상위 true로 우회되지 않습니다. 원문 정의를 지우거나 바꾸지 않고 실행만 차단합니다. 별도 저장 테이블·환경변수·외부 서비스·DB 확장은 필요하지 않습니다.

## 검증

```sh
MADI_TEST_POSTGRES_DSN='postgres://tester@127.0.0.1:5432/madi_test?sslmode=disable' \
  go test -race ./internal/server -run '^(TestDocumentQuery|TestPostgresDocumentQuery)' -count=1 -v

# web 빌드 및 Playwright Chromium 설치 후
MADI_TEST_POSTGRES_DSN='postgres://tester@127.0.0.1:5432/madi_test?sslmode=disable' \
MADI_BROWSER_DOCUMENT_QUERY=1 \
  go test ./internal/server -run '^TestBrowserDocumentQuery$' -count=1 -v
```

2026-09-09 Go 1.26.8/PostgreSQL 18.6의 임의 격리 스키마에서 첫 종합 PG race 16.053초 PASS: 문법/중복 키/SQL 거부, 실제 Front Matter·CommonMark 할 일, private·중간 경로 ACL, 원문/버전 불변, 키/MCP/세션 바인딩, 정책·키 회수, 후보 2,100→2,000/100행, 큰 본문 제외, 4슬롯 포화, 실제 SQL 잠금 1.91초 종료, 잠금 후 세션 만료를 확인했습니다. 추가 취소 회귀는 실제 SQL 대기를 확인한 뒤 HTTP 취소와 슬롯 반환을 3.237초 시험에서 검증했습니다. 합성 fixture의 실제 프로토콜 검증이며 운영 규모·최악 지연 보증은 아닙니다.

같은 날 대소문자 키의 모호함·잘린 필드 진단·관계 결과 버전 정합성 보강 후 E8 및 기존 키/MCP 교차 PG race 22.405초 PASS입니다. 실제 브라우저 `main-CnzNgKXJ.js`에서는 명시 실행/매개변수/비공개 제외/원문 무변경/출처 미리보기 초점 복귀/인쇄 결과 제외/390px/빈 결과 복구/권한 회수/네이티브 선택 변경/정의만 삽입·저장을 21.470초에 확인했습니다.

최종 `main-C5hG03ts.js`에서 동일 브라우저 회귀와 결과 셀의 실제 계산 글꼴 16px 검증을 17.577초에 통과했습니다. 정상 화면만 남긴 [조회 정의](screenshots/document-query-definition.png), [실행 결과](screenshots/document-query-results.png), [모바일 표](screenshots/document-query-mobile.png), [권한 회수 안내](screenshots/document-query-revoked.png), [조회 삽입 도구](screenshots/document-query-builder.png)를 참고하세요. 초기 실패 진단 화면은 공개 캡처에 포함하지 않습니다.
