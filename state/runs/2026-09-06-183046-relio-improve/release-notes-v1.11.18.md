## Relio v1.11.18 — 질의 파라미터의 숫자가 보낸 것과 다르게 읽히던 문제

`limit=1e3` 을 보내면 1건이 돌아왔습니다. `0x10` 은 16, `030` 은 8진수 24, `50abc` 는 50 이었습니다. 이 서버는 OpenAPI 에서 그 키들을 `type: integer` 에 minimum/maximum 까지 붙여 공표하는데, 실제로는 읽을 수 있는 앞부분까지만 소비하고 나머지를 조용히 버렸습니다. 이번 릴리즈는 정수 질의 파라미터를 엄격하게 파싱하고, 값이 범위를 벗어났을 때 **필터 자체가 꺼지던** 반대 방향의 문제도 함께 고쳤습니다.

## 1. 원인

### 1-1. `fmt.Sscan` 은 읽을 수 있는 만큼만 읽고 나머지에 대해 아무 말도 하지 않는다

`httpx.IntQuery` 는 이렇게 파싱했습니다.

```go
var n int
if _, err := fmt.Sscan(v, &n); err != nil || n < min || n > max {
	return fallback
}
```

`Sscan` 은 입력 전체가 숫자여야 한다고 요구하지 않습니다. 앞에서부터 정수로 읽히는 데까지 읽고 성공을 돌려주며, 남은 부분은 오류가 아닙니다. Go 의 정수 스캔은 `0x`·`0` 접두사도 진법으로 해석합니다. 그래서

| 보낸 값 | 읽힌 값 |
| --- | --- |
| `1e3` | 1 |
| `0x10` | 16 |
| `030` | 24 (8진수) |
| `50abc` | 50 |

가 되었습니다. 어느 것도 오류가 아니었으므로 fallback 으로 떨어지지도 않았고, 클라이언트는 자기가 보낸 것과 다른 숫자로 계산된 응답을 정상 응답으로 받았습니다. 이 함수는 핸들러 30곳이 함께 쓰는 자리라서, `limit`·`days`·`page` 같은 키 전부가 같은 방식으로 읽혔습니다.

### 1-2. `expiringDays` 는 범위를 벗어난 값이 필터를 꺼버렸다

계약 목록의 종료일 조건은 질의 안에서 이렇게 생겼습니다.

```sql
AND ($5=0 OR c.end_date BETWEEN current_date AND current_date+$5)
```

`0` 은 "종료일로 좁히지 않는다"는 뜻입니다. 그런데 fallback 도 0 이었습니다.

```go
if expiringDays < 0 || expiringDays > 3650 {
	expiringDays = 0
}
```

좁혀 달라는 요청이 상한을 넘기는 순간 **좁히지 말라는 요청으로 바뀌었습니다.** `get_expiring_contracts` 에 days=5000 을 준 에이전트는 범위 안 계약 전부를 — 종료일이 아예 없는 계약까지 — "만료 예정" 이라는 이름으로 돌려받았습니다. 값이 클수록 결과가 넓어지는 것도 아니고, 상한 근처에서 갑자기 전체 목록으로 뒤집히는 형태였습니다.

## 2. 수정

### 2-1. 엄격한 파싱

`strconv.Atoi` 로 바꿔 값 전체가 10진 정수일 때만 받아들입니다.

```go
func intQuery(r *http.Request, key string) (int, bool) {
	v := strings.TrimSpace(r.URL.Query().Get(key))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}
```

앞뒤 공백은 계속 허용합니다. 질의 문자열은 `+` 를 공백으로 실어 나르므로 `?days=+30` 은 서버에 `" 30"` 으로 도착합니다. 클라이언트가 정상적으로 보낸 값을 새 파싱이 거절해서는 안 됩니다. `030` 은 이제 8진수 24 가 아니라 30 입니다.

### 2-2. 좁히는 필터는 fallback 대신 상한으로 줄인다

fallback 이 필터를 끄는 키를 위해 `ClampQuery` 를 두었습니다.

```go
// ClampQuery is IntQuery for a filter whose fallback switches the filter off.
// Falling back there answers a request to narrow with everything the caller did
// not ask for, so an out-of-range value is pulled to the nearest bound instead
// and only an unreadable one yields fallback.
```

읽을 수 있는 값은 가까운 경계로 당겨집니다(5000 → 3650). 읽을 수 없는 값은 당길 방향이 없으므로 그대로 fallback 입니다.

같은 상한을 서비스 쪽 `boundExpiringDays` 에도 두었습니다. REST 핸들러는 `ClampQuery` 를 지나지만 MCP 의 `get_expiring_contracts` 는 서비스를 직접 부르므로, 두 경로가 같은 기준을 쓰려면 서비스에도 있어야 합니다. OpenAPI 설명에도 "최대값보다 큰 값은 최대값으로 줄여 적용합니다" 를 적어, 공표한 계약과 실제 동작이 어긋나지 않게 했습니다.

## 3. 회귀 방지

테스트 6개를 추가했습니다.

- 정수가 아닌 값 8가지(`1e3`, `0x10`, `50abc`, `50 999`, `1_000`, `abc`, `3.5`, 빈 값)가 전부 fallback 인지. fallback 을 7로 잡아, 어떤 경우도 "요청한 값과 우연히 같아서" 통과할 수 없게 했습니다.
- 질의 문자열이 실을 수 있는 모든 형태(`30`, `" 30"`, `+30`, `030`, 경계값 `1`·`365`)를 보낸 그대로 읽는지. 엄격해진 파싱이 정상 요청을 막지 않는지 보는 쪽입니다.
- 범위 밖 값(`0`, `-3`, `201`, `int` 를 넘는 자릿수)이 fallback 인지.
- `ClampQuery` 가 상한 위의 값을 상한으로 줄여 필터를 켠 채로 두는지(5000 → 3650).
- 읽을 수 없는 값은 `ClampQuery` 에서도 fallback 인지.
- 서비스 쪽 `boundExpiringDays` 가 같은 상한을 적용하는지.

옛 구현으로 되돌리면 이 중 6건이 실제로 실패하는 것을 확인했습니다 — `1e3`→1, `0x10`→16, `030`→24, `boundExpiringDays(5000)`→0 으로 출력합니다.

## 4. 적용

마이그레이션은 없습니다. 정상적인 10진 정수를 보내던 클라이언트는 동작이 전혀 바뀌지 않습니다. 달라지는 것은 두 가지뿐입니다. 정수가 아닌 값은 잘려 읽히는 대신 파라미터의 fallback 이 적용되고, `GET /contracts` 의 `expiringDays` 가 3650 을 넘으면 필터가 꺼지는 대신 3650 일로 좁혀집니다.
