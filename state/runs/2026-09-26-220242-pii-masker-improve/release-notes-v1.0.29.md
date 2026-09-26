## 업스트림 리다이렉트가 HTTPS에서 HTTP로 강등되어도 그대로 따라가던 문제 수정

`httpClient()`의 `CheckRedirect`는 리다이렉트 대상을 `checkURLAllowed`로만 검사했습니다. 이 검사는 호스트만 보고 스킴은 보지 않으므로, `https://`로 나간 추론 요청이 같은 호스트(또는 허용 목록에 있는 호스트)의 `http://`로 리다이렉트되면 차단되지 않고 따라갔습니다. `performParseRequest`가 요청 본문을 `*bytes.Buffer`로 넘기기 때문에 Go가 `GetBody`를 채우고 307·308 리다이렉트에서 멀티파트 본문을 그대로 재전송하므로, 마스킹하려고 업로드한 원문 PII와 `Authorization` 토큰이 평문으로 네트워크에 다시 실릴 수 있었습니다.

- 새 헬퍼 `checkRedirectScheme`가 최초 요청(`via[0].URL`)의 스킴이 `https`면 리다이렉트 대상도 `https`여야 통과시킵니다. 대소문자는 구분하지 않으므로 `HTTPS://`로 시작한 체인도 같게 다룹니다. 기존 `checkURLAllowed` 호스트 검사는 그 뒤에 그대로 남겨 두었습니다.
- 검사는 **최초 요청이 `https`일 때만** 적용됩니다. 처음부터 `http` 엔드포인트를 설정한 경우(내장 mock 업스트림, 또는 운영자가 의도적으로 지정한 평문 엔드포인트)는 동작이 달라지지 않습니다.
- 거부 사유는 `blockedHostError`와 섞지 않고 별도 오류(`refusing to follow a redirect from https to ...`)로 두어, 진단에서 "차단된 호스트"로 오인되지 않게 했습니다. `classifyRequestError`에서는 `network_error`로 분류됩니다.
- 프로덕션 배선인 `client.httpClient().CheckRedirect` 클로저를 직접 호출하는 테이블 테스트 5건으로 검증합니다. `https`→`https` 허용, 같은 호스트의 `https`→`http` 거부, `http`→`http` 허용, `http`→`https` 허용, 대문자 `HTTPS://`의 강등도 거부를 확인하고, 거부 시 `errors.As(err, &blockedHostError{})`가 false임까지 단언합니다. 기존 `TestParseDocumentFollowsRedirectToAllowedHost`·`TestParseDocumentRejectsRedirectToDisallowedHost`는 수정 없이 통과합니다.
- `PII_MASKER_ALLOW_HOSTS` 설명에 이 규칙과 그 이유를 README에 남겼습니다.
