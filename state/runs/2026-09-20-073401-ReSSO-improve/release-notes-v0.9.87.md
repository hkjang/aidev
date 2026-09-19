**UserInfo가 POST 본문의 토큰을 읽지 않아, 유효한 토큰에도 "토큰을 버려라"라고 답하고 있었습니다.** `POST /realms/{realm}/protocol/openid-connect/userinfo`는 라우트로 등록되어 있었지만 토큰은 `Authorization` 헤더에서만 읽었습니다. RFC 6750 §2.2대로 form-encoded 본문에 `access_token=…`을 싣는 RP·SDK는 방금 받은 멀쩡한 토큰으로도 **401 `invalid_token`**을 받았고, 그 답은 "이 토큰은 만료됐거나 위조됐으니 버리고 다시 받아라"라는 뜻이라 RP는 재발급을 반복하거나 사용자를 다시 로그인시키게 됩니다 — 토큰에는 아무 문제가 없었는데도요.

### 수정

- **POST UserInfo는 본문 `access_token`도 받습니다.** POST일 때만 `r.ParseForm()` 후 `r.PostForm.Get("access_token")`을 읽어 헤더 값과 합칩니다. GET은 본문을 읽지 않습니다.
- **헤더와 본문에 둘 다 오면(값이 같아도) 400 `invalid_request`입니다** — §2 "한 가지 방법만 써야 한다"대로 `WWW-Authenticate: Bearer error="invalid_request"`를 붙여 끝냅니다. 토큰이 나쁜 게 아니라 요청이 잘못된 것이므로, 401 `invalid_token`으로 토큰을 버리게 하지 않습니다. 이 판정은 토큰 검증 **앞**이라 헤더의 토큰이 만료·위조여도 400입니다.
- **쿼리 파라미터(§2.3)는 일부러 받지 않습니다** — URL의 토큰은 지나는 모든 접근 로그에 남습니다. GET·POST 어느 쪽이든 `?access_token=`만 보낸 요청은 전과 같이 401 `invalid_token`입니다.
- `bearerToken` 헬퍼는 MCP·미들웨어도 쓰므로 시그니처를 바꾸지 않고 `userInfo` 안에서만 합쳤습니다. `writeUserInfoUnavailable`(500) 분기와 realm 조회 순서는 그대로입니다.
- `docs/compatibility.md`의 UserInfo 행에 POST 본문·둘 다·쿼리에 무엇을 답하는지 적었습니다.

### 확인

- 새 연동 테스트 `TestIntegrationUserInfoReadsTheTokenFromAPostBody` — 헤더 GET 200 기준값 → 본문만 POST 200 + `sub`·`preferred_username` 일치 → 헤더+본문 400 `invalid_request` + `WWW-Authenticate` → 본문 garbage 401 `invalid_token` → GET 쿼리 401 → POST 쿼리 401. 토큰은 프로덕션 발급 경로(`IssueUserTokens`, 실제 저장소·서명 키)로 만들었고, **수정 전 핸들러에서 첫 단언(본문만 → 200)이 실제로 401 `invalid_token`으로 실패하는 것을 확인**했습니다.
- 정지된 realm이 모든 프로토콜 Endpoint를 거절하는 라우트 워커 테스트(빈 본문 POST userinfo는 realm 조회에서 먼저 끊겨 401 그대로)와 CSRF·API 키 라우트 테스트도 통과합니다.

### Upgrade notes

달라지는 곳은 UserInfo Endpoint 하나입니다: POST 본문의 `access_token`을 받고, 헤더와 본문에 둘 다 온 요청에는 401 대신 **400 `invalid_request`**로 답합니다. 헤더로만 보내던 RP는 전과 똑같이 동작하고, 응답 본문·저장되는 데이터·설정은 전과 같으며 **마이그레이션도 설정 변경도 없습니다.** 되돌릴 때 이전 이미지도 그대로 동작합니다 — 본문 방식 RP만 다시 401을 받게 됩니다.
