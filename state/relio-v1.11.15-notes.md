## Relio v1.11.15 — MCP 목록 도구가 "더 있다"고 알려주고도 다음 장을 못 받던 문제

`search_customers` 와 `list_opportunities` 가 응답에 `hasMore` 와 `nextCursor` 를 담아 뒤에 더 있다고 알려주면서 정작 그 커서를 받을 방법을 주지 않던 문제를 고쳤습니다. 기본 50건, 최대 200건을 넘는 계정에서는 첫 페이지 뒤가 MCP 로는 아예 보이지 않았습니다.

## 1. 원인

### 1-1. 돌려주는 커서를 다시 받을 자리가 없었다

두 도구의 결과는 전체 집합이 아니라 한 페이지입니다. 뒤에 있는 질의는 `LIMIT`/`OFFSET` 을 적용하고, 응답은 `hasMore` 와 `nextCursor` 를 함께 담습니다. 그런데 어느 쪽도 `cursor` 인자를 스키마에 선언하지 않았고, 핸들러는 커서 자리에 빈 문자열을 넘겼습니다.

```go
case "search_customers":
	v, err = s.CRM.ListCustomers(ctx, p, strArg(a, "query"), "", "", intArg(a, "limit", 50))
```

두 스키마 모두 `additionalProperties` 가 `false` 이므로, 클라이언트가 알아서 `cursor` 를 끼워 보낼 수도 없었습니다. 스키마를 엄격하게 지키는 클라이언트는 선언되지 않은 인자를 거절하고, 설령 보냈더라도 서버가 읽지 않았습니다. 에이전트 입장에서는 "더 있다"는 안내를 받고도 요청할 방법이 없는 상태였습니다.

### 1-2. 선언한 인자가 질의까지 가지 못했다

`list_opportunities` 는 인자를 핸들러 안에서 곧바로 필터 구조체로 옮겼습니다. 선언과 전달이 서로 다른 곳에 흩어져 있으면, 스키마에 광고한 인자가 질의로 가는 길에 조용히 빠져도 아무 신호가 없습니다. 실제로 `cursor` 가 그렇게 빠져 있었습니다.

`status` 도 같은 자리에서 문제가 됐습니다. 질의는 저장된 값과 그대로 비교하는데 저장된 값은 대문자(`OPEN`, `WON`, `LOST`)입니다. 도구 설명을 산문으로 읽고 `"open"` 을 보낸 모델은 오류가 아니라 빈 목록을 받았습니다.

### 1-3. 알 수 없는 커서를 1페이지로 되돌렸다

커서 해석은 실패를 전부 offset 0 으로 삼켰습니다.

```go
func pageOffset(cursor string) int {
	if cursor == "" {
		return 0
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	...
	return n
}
```

이 서버가 발급하지 않은 커서는 조용히 첫 페이지가 됩니다. 그런데 첫 페이지 응답에는 다시 `nextCursor` 가 붙으므로, 페이지를 따라 걷는 호출자 — 특히 MCP 에이전트 — 는 "첫 페이지 + 더 있음"을 계속 받으며 끝에 도달하지 못합니다.

## 2. 수정

두 도구가 `cursor` 를 인자로 받고, 인자에서 서비스 필터로 옮기는 일을 한곳에 모았습니다.

```go
var cursorProp = str("다음 페이지 커서 · 직전 응답의 nextCursor 값을 그대로 전달합니다. 첫 페이지는 비웁니다.")

func opportunityFilterArgs(a map[string]any) crm.OpportunityFilter {
	return crm.OpportunityFilter{
		Query:      strArg(a, "query"),
		CustomerID: strArg(a, "customerId"),
		Status:     strings.ToUpper(strings.TrimSpace(strArg(a, "status"))),
		Cursor:     strArg(a, "cursor"),
		Limit:      intArg(a, "limit", 50),
	}
}
```

`customerSearchArgs` 와 `opportunityFilterArgs` 를 스키마 바로 옆에 두어, 선언한 인자가 질의로 가는 길에 누락될 수 없게 했습니다. `status` 는 같은 함수에서 대문자로 정규화하므로 `"open"` 도 실제 결과를 돌려줍니다. 두 도구의 설명에도 `hasMore` 가 `true` 면 `nextCursor` 를 `cursor` 로 넘기라는 안내를 넣었습니다.

`pageOffset` 은 `parseCursor` 로 바뀌어, 이 서버가 발급하지 않은 커서를 거절합니다.

```go
var errInvalidCursor = errors.New("cursor is not valid: pass back the nextCursor value from the previous page")

func parseCursor(cursor string) (int, error) {
	...
}
```

오류 문구는 `serviceError` 의 substring 분류를 그대로 통과해 REST 에서 400 `invalid_request` 가 됩니다. 커서 형식 자체와 기본 50건·최대 200건 상한은 그대로이며, 달라진 것은 이제 다음 페이지를 요청할 수 있다는 점과 잘못된 커서가 조용히 1페이지로 되돌아가지 않는다는 점입니다.

## 3. 회귀 방지

- 발급한 커서를 그대로 다시 넣으면 같은 offset 이 나오는지(왕복), 빈 커서는 offset 0 으로 통과하는지.
- base64 가 아닌 값, 접두사가 없거나 다른 값, 음수, 숫자가 아닌 값, 구분자가 더 붙은 값 등 위조 커서 7종이 모두 거절되고, 거절된 커서가 offset 을 함께 보고하지 않는지.
- 잘못된 커서 문구가 `serviceError` 의 404·403·409 분류어를 건드리지 않고, 대신 돌려보낼 값의 이름(`nextCursor`)을 담고 있는지.
- 두 도구가 `cursor` 를 문자열로 선언하고 `additionalProperties:false` 를 유지하는지, 스키마가 광고한 모든 인자가 실제로 필터에 전달되는지.
- `status` 가 대문자로 정규화되는지, `limit` 을 생략하면 기본 50건이 되는지.

## 4. 적용

마이그레이션은 없습니다. 재시작하면 바로 적용됩니다. 첫 페이지만 읽던 기존 호출은 동작이 달라지지 않으며, 잘못된 커서를 보내던 호출만 조용한 1페이지 대신 400 을 받습니다.
