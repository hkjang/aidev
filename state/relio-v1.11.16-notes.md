## Relio v1.11.16 — OpenAPI 문서가 "어떤 값을 보내야 하는지" 를 말하지 않던 문제

`GET /api/openapi.json` 은 REST·MCP 클라이언트가 가진 유일한 계약입니다. 그런데 이 문서에는 경로와 요약만 있었고, 그 경로에 무엇을 채워 넣어야 하는지는 한 글자도 없었습니다. 이번 릴리즈는 경로 파라미터와 질의 파라미터를 문서에 넣고, 문서와 실제 핸들러가 어긋나면 빌드가 실패하도록 했습니다.

## 1. 원인

### 1-1. `{id}` 를 채울 자리가 선언되어 있지 않았다

OpenAPI 는 경로의 모든 템플릿 표현식(`{id}`)에 대응하는 path parameter 를 선언하도록 **요구**합니다. 그런데 `{id}` 를 가진 약 50개 경로 중 파라미터를 선언한 것은 하나도 없었습니다.

```go
"/customers/{id}": readUpdateDelete("고객 조회, 수정 및 삭제", "customer:read", "customer:write", "customer:delete"),
```

이 상태의 문서는 사양 위반입니다. 클라이언트 생성기는 `{id}` 에 넣을 인자가 없는 메서드를 만들거나 아예 실패하고, Redocly 같은 린터는 문서를 거절합니다. 문서만 읽는 쪽에서는 그 자리에 고객 ID 가 들어간다는 사실조차 알 방법이 없었습니다.

### 1-2. 질의 문자열이 전부 빠져 있었다

`/customers` 는 페이징 작업 이후로 `cursor` 와 `sort` 를 받고, `/opportunities` 는 다섯 가지 조건으로 거르며, `limit` 상한은 엔드포인트에 따라 200 또는 500 입니다. 인증된 GET 은 모두 `fields` 로 응답을 줄일 수 있습니다. 어느 것도 문서에 없었습니다.

특히 `sort` 는 화이트리스트 방식이라, 목록에 없는 값은 오류가 아니라 **조용히 무시되고** 기본 정렬이 적용됩니다.

```go
orders := map[string]string{"name": "c.name ASC,c.id", "-name": "c.name DESC,c.id", ...}
order := orders[filter.Sort]
if order == "" {
	order = "c.updated_at DESC,c.id"
}
```

즉 `-annualRevenue` 라는 값이 있다는 사실을 Go 소스를 읽어서 알아내지 못한 클라이언트는, 정렬을 요청했는데 최근 수정 순을 받고도 그 사실을 알 수 없었습니다.

### 1-3. 문서와 코드를 비교하는 장치가 경로까지만이었다

v1.11.9 에서 넣은 계약 테스트는 라우터의 경로 표와 문서의 경로를 양방향으로 비교합니다. 파라미터는 비교 대상이 아니었으므로, 새 필터를 추가하고 문서에 적지 않아도 아무 일도 일어나지 않았습니다.

## 2. 수정

경로 파라미터는 **템플릿에서 직접 파생**합니다. 손으로 관리하는 목록은 다음에 추가되는 경로가 또 빠뜨릴 수 있기 때문입니다.

```go
// pathParameters reads the `{name}` segments straight out of the template, so a
// path added later declares its parameters without anyone remembering to.
func pathParameters(path string) []any {
	...
}
```

질의 파라미터는 `queryParameters` 표에 적었습니다. enum 은 DB 의 CHECK 제약에서, 정수 범위와 기본값은 그 값을 읽는 `httpx.IntQuery` 호출에서 그대로 가져왔습니다.

```go
"GET /customers": {
	text("q", "고객명 또는 사업자번호 부분 일치 검색어입니다."),
	text("customerType", "고객 유형으로 좁힙니다."),
	text("grade", "고객 등급으로 좁힙니다."),
	cursor(),
	sortBy("name", "-name", "annualRevenue", "-annualRevenue", "createdAt", "-createdAt", "updatedAt", "-updatedAt"),
	limit(50, 200),
},
```

`fields` 는 `components.parameters` 의 공유 항목이고 인증된 GET 마다 `$ref` 로 붙습니다. 이 투영은 핸들러가 아니라 `requireAuth` 가 핸들러 실행 전에 적용하므로, 엔드포인트별 사정이 아니라 미들웨어의 성질이기 때문입니다.

`httpx.IntQuery` 는 범위를 벗어난 값을 오류로 만들지 않고 기본값으로 되돌립니다. 그리고 기본값 자체가 범위 밖일 때가 있는데(`year` 의 0, `version` 의 -1) 이는 "필터 없음" 을 뜻하므로 schema 의 `default` 로 쓰지 않고 설명에만 적었습니다.

## 3. 회귀 방지

- 경로의 `{var}` 와 선언된 path parameter 가 양방향으로 일치하는지, 그리고 모든 path parameter 가 `required: true` 인지.
- 문서의 질의 파라미터와 그 경로를 실제로 처리하는 핸들러가 읽는 질의 키가 양방향으로 일치하는지. 핸들러 목록은 라우터 등록문을 go/ast 로 읽어 만들고, 질의 키는 각 핸들러 본문의 `r.URL.Query().Get`·`q.Get`·`httpx.IntQuery` 호출에서 뽑습니다.
- `fields` 가 인증된 GET 에만, 그리고 인증된 GET 전부에 붙어 있는지. 라우트가 `requireAuth` 로 감싸였는지도 소스에서 판정합니다.
- 문서의 `sort` enum 이 `internal/crm` 의 정렬 화이트리스트와 값·개수까지 일치하는지.

각 검사는 문서를 실제로 어긋나게 만들어(커서 삭제, 없는 파라미터 추가, path parameter 제거, 공개 GET 에 `fields` 노출, 인증 GET 에서 `fields` 누락, sort 값 변조) 실패하는 것을 확인했습니다.

## 4. 적용

마이그레이션은 없습니다. 서버 동작과 응답은 바뀌지 않았고, `/api/openapi.json` 이 돌려주는 문서만 넓어집니다. 기존 호출은 그대로 동작합니다.
