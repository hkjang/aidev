**UserInfo POST의 폼 본문에도 기존 프로토콜 POST와 같은 1MiB 상한을 적용합니다.** 이전 버전은 UserInfo에 이 제한이 빠져 있어, 유효한 토큰을 보내면 1MiB를 넘는 폼도 200으로 처리했습니다. 이제 인코딩된 폼 본문이 1,048,576바이트를 넘으면 **400 JSON `invalid_request`**로 거절합니다.

### 수정

- **폼을 읽기 전에 1MiB로 제한합니다.** `userInfo`의 POST `ParseForm` 바로 앞에 `http.MaxBytesReader(w, r.Body, 1<<20)`를 적용해 기존 폼 파싱 오류 응답으로 처리합니다.
- 토큰을 본문에 보내든 `Authorization` 헤더에 보내든 POST 폼 본문에는 같은 상한이 적용됩니다. 고정 길이와 chunked 전송 모두 같고, 정확히 1MiB인 유효한 요청은 계속 처리합니다.
- `docs/compatibility.md`의 UserInfo 행에 본문 상한을 명시했습니다.

### 확인

- 실제 PostgreSQL·서명 키·세션·`IssueUserTokens`·프로덕션 Handler와 HTTP 테스트 서버로 토큰 위치(본문/헤더) × 전송 방식(고정 길이/chunked) × 크기(정확히 1MiB/1MiB+1)의 여덟 경우를 검사합니다.
- 구현 단계에서 수정 전과 제한 재제거 시 초과 네 경우가 모두 200으로 실패하고, 수정 후 모두 통과하는 것을 확인했습니다. 기존 GET·POST 인증, 중복 전달, 쿼리 토큰, 불량 토큰 계약도 검증했습니다.

- 릴리즈 준비에서도 `make lint`, `make test`(Go race 전 패키지·연동 SKIP 0·`go vet`·프런트 29파일/161테스트), `make build VERSION=v0.9.88`, 버전 일치 검사와 `git diff --check`가 통과했습니다.

### Upgrade notes

UserInfo POST를 호출하는 RP·SDK는 **인코딩된 폼 전체를 1MiB 이하**로 보내야 합니다. 초과 요청의 `invalid_request`는 본문을 줄여 다시 보내라는 뜻이며, 새 토큰 발급으로 해결되지 않습니다. GET과 상한 이내의 기존 인증 계약은 유지됩니다. 마이그레이션이나 설정 변경은 없습니다. 이전 `v0.9.87` 이미지로 롤백할 수 있지만, 되돌린 동안에는 초과 본문도 다시 허용됩니다.

배포 담당자는 먼저 제한된 환경에서 정상 UserInfo 호출과 초과 요청 거절을 확인한 뒤 확대하세요. 정상 요청 한 건이라도 새로 실패하면 확대를 중단하고 이전 이미지로 되돌립니다. 배포 후 첫날은 UserInfo 400 증가와 RP 문의를 확인해 본문 크기 문제인지 점검하세요.
