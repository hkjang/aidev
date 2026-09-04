## Relio v1.11.14 — 큰 MCP 요청이 "Parse error" 로 돌아오던 문제

1MB 를 넘는 MCP 요청을 보내면 클라이언트가 올바르게 만든 JSON-RPC 메시지가 파싱 오류로 거절되던 문제를 고쳤습니다. 이제 크기 문제라는 사실이 응답에 그대로 드러납니다.

## 1. 원인

### 1-1. 상한까지만 읽고 멈추면 본문이 잘린다

MCP 엔드포인트는 본문을 이렇게 읽었습니다.

```go
body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
if err != nil {
    s.writeErrorStatus(w, http.StatusBadRequest, nil, -32700, "Parse error", nil)
    return
}
```

`io.LimitReader` 는 상한을 넘는 입력을 오류로 만들지 않습니다. 딱 그 지점까지 읽고 정상적으로 EOF 를 돌려줍니다. 그래서 1MB 를 넘는 본문은 오류 없이 **1MB 에서 잘린 채** 다음 단계로 넘어갔습니다.

### 1-2. 잘린 JSON 은 깨진 JSON 이다

잘려 나간 본문은 닫히지 않은 문자열과 중괄호로 끝나므로 `parseRequests` 가 반드시 실패합니다. 그 결과 클라이언트가 받는 답은 이것이었습니다.

```json
{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}
```

HTTP 상태는 400 입니다. 클라이언트 입장에서 이 답은 자기 직렬화가 잘못됐다는 뜻이므로, 정작 문제인 "본문이 상한을 넘었다" 는 사실은 어디에도 없습니다. 긴 노트 본문이나 큰 `arguments` 를 담은 `tools/call` 이 이유 없이 실패하는 것처럼 보이고, 같은 메시지를 그대로 재시도하면 똑같이 실패합니다.

REST 쪽은 이미 이 상황을 구분해서 답하고 있었습니다. `httpx.DecodeJSON` 은 `http.MaxBytesReader` 로 413 `request_too_large` 를 돌려주고, 멱등성 미들웨어는 상한보다 한 바이트 더 읽어 초과를 판별합니다. MCP 만 이 구분이 빠져 있었습니다.

## 2. 수정

상한보다 한 바이트 더 읽어, 잘린 본문과 원래 깨진 본문을 구분합니다.

```go
// MaxRequestBytes caps one MCP HTTP message.
const MaxRequestBytes = 1 << 20

func readRequestBody(r io.Reader) (body []byte, tooLarge bool, err error) {
	body, err = io.ReadAll(io.LimitReader(r, MaxRequestBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(body) > MaxRequestBytes {
		return nil, true, nil
	}
	return body, false, nil
}
```

초과한 본문은 413 과 함께 JSON-RPC 오류로 답하고, 지켜야 할 상한을 `data` 에 담아 보냅니다.

```json
{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"요청 본문이 너무 큽니다.","data":{"maxBytes":1048576}}}
```

읽기 자체가 실패한 경우(연결 끊김 등)는 지금까지처럼 400 `Parse error` 입니다. 상한 값 자체는 1MB 그대로이며, 달라진 것은 상한을 넘겼을 때 돌아오는 답뿐입니다.

## 3. 회귀 방지

- 상한과 정확히 같은 크기의 본문은 통과하고, 한 바이트만 넘어도 초과로 판정되는지.
- 상한을 넘긴 실제 `tools/call` 메시지가 초과로 판정되는지, 그리고 그것을 상한에서 자른 본문은 `parseRequests` 가 반드시 거절하는지 — 잘린 본문이 우연히 파싱되면 일부만 담긴 호출이 실행될 수 있으므로 함께 확인합니다.
- 읽기 오류는 초과가 아니라 오류로 전달되는지.
- 413 응답이 JSON-RPC 형식을 유지하며 `data.maxBytes` 로 상한을 알려주는지.

## 4. 적용

마이그레이션은 없습니다. 재시작하면 바로 적용되며, 1MB 이하의 요청은 동작이 전혀 달라지지 않습니다.
