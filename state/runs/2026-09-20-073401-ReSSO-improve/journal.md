# 회차 노트 2026-09-20-073401-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:34] base pinned — main@0cae785
- [러너 07:34] autonomy release — 

## 정찰 노트
- 고른 이유: 최근 다섯 회차가 전부 로그인 화면 쪽이라 백엔드 프로토콜 쪽의 확인된 결함(POST userinfo 라우트가 등록돼 있는데 `bearerToken`이 헤더만 읽음 — server.go:114-115, oidc.go:1162 직접 확인)을 골랐다. 연동 테스트로 수정 전 실패를 실제 런타임에서 보일 수 있고(운영자 규칙), M짜리 logout/hint 대조 둘은 한 세션에 안 들어간다.
- 확신 없는 곳: `bearerToken`이 userinfo 밖(introspection 등)에서도 쓰이는지 미확인 — 그래서 시그니처를 바꾸지 말라고 적었다. 라우트 워커 테스트(795-812행)가 빈 본문 POST userinfo에 기대하는 상태 코드는 읽지 않았다 — 구현자가 먼저 확인할 것.
- 조심할 것: `r.FormValue`는 쿼리도 읽는다(§2.3을 열어 버림) — 반드시 `r.PostForm`. `writeBearerError`는 401 고정이라 400에는 못 쓴다. 공유 stash 금지.
- 프로필은 09-19 것이 지금 코드와 어긋나지 않아(구조·명령·위험 구역 그대로) 다시 쓰지 않았다.
- [러너 07:36] scout done — UserInfo POST가 form-encoded `access_token`(RFC 6750 §2.2)을 받게 하고, 헤더와 본문에 둘 다 오면 400 `invalid_request`로 �

## 구현 노트
- 무엇·왜: `userInfo`(oidc.go)가 POST일 때 `r.PostForm.Get("access_token")`도 읽고, 헤더와 본문에 둘 다 오면 400 `invalid_request`로 끝낸다 — POST 라우트는 있었는데 토큰은 헤더에서만 읽어 RFC 6750 §2.2 방식의 RP가 유효 토큰으로 401 `invalid_token`을 받았다. 커밋 76f5974. `bearerToken`은 mcp.go·middleware.go도 쓰므로 시그니처를 두지 않고 userInfo 안에서만 합쳤다.
- 수정 전 실패 확인: 새 테스트 `TestIntegrationUserInfoReadsTheTokenFromAPostBody`를 먼저 넣고 돌려 첫 단언이 `401 invalid_token`으로 실패하는 것을 봤고(stash 없이 — 테스트만 먼저 추가), 핸들러 수정 뒤 통과.
- 확신 없는 곳: 둘 다 왔을 때 400을 답하는 시점이 `Verify` **앞**이라 헤더의 토큰이 만료·위조여도 400이다(토큰 판정보다 요청 형식 판정이 먼저) — §2 문구상 맞다고 보지만 "401을 먼저 줘야 한다"는 해석도 가능. `error_description` 문구는 새로 쓴 것.
- 일부러 하지 않은 것: 쿼리 방식(§2.3)·토큰 없음의 §3.1 error 없는 401(과제서 범위 밖)·`Content-Type`이 폼이 아닌 POST에 대한 400(아이디어에 둠). ParseForm 실패는 다른 세 핸들러와 같은 400 `invalid_request "invalid form body"`인데 이 경로는 테스트하지 않았다(Go의 ParseForm이 폼 본문에서 오류를 내는 입력을 만들기 어렵다).
- 다음 역할이 조심할 것: 새 테스트는 PostgreSQL 컨테이너(`RESSO_TEST_POSTGRES_DSN`)가 있어야 돌고 없으면 SKIP이다. `make test`는 exit 0·SKIP 0·29파일/161테스트로 확인했고 `webui/dist/index.html` 변경은 없었다. 문서는 `docs/compatibility.md` 한 행만이라 PDF 재생성 없음.
- [러너 07:44] brief accepted — 채택 — 과제서의 근거(POST 라우트 등록·bearerToken 헤더만)가 코드와 정확히 맞았고 수용 기준 셋을 그대로 구현했다; 토�
- [러너 07:45] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: diff 3파일(oidc.go·integration_test.go·compatibility.md)만이고 범위 이탈 없음. 새 테스트를 main의 oidc.go로 바꿔 돌려 401 invalid_token으로 실패 → HEAD에서 통과를 직접 봤고, 이웃 userinfo 테스트·라우트 스윕·gofmt·go vet도 통과(테스트 서비스 변수 4개 세워 SKIP 없음).
- 보안: 본문 토큰은 POST+r.PostForm만 읽어 쿼리(§2.3)는 안 열리고, 헤더 토큰과 같은 Verify 경로라 인가 범위 변화 없음. 접근 로그는 path만 찍어 토큰이 남지 않음. 차단 사유 없음.
- 구현자가 의심한 "둘 다 왔을 때 Verify 앞에서 400"은 §2·§3.1과 맞고 인증 전 응답이라 정보 누출 없음 — 결함 아님.
- 못 본 것: ParseForm 실패 경로(테스트 없음), Content-Type이 폼이 아닌 POST(JSON 본문 토큰)는 여전히 조용히 401 — 다음 회차 아이디어로 남김.
- 릴리즈 노트: "UserInfo POST가 본문 access_token(RFC 6750 §2.2)을 받음, 헤더+본문 동시 전송은 400 invalid_request" 한 줄이면 됨. 작업 트리의 webui/dist/index.html 변경은 커밋 밖이니 릴리즈 전에 되돌릴 것.
- [러너 07:46] review approved — 리뷰 승인 (risk=low)
- [러너 07:47] pr created — https://github.com/hkjang/ReSSO/pull/24
- [러너 07:55] ci passed — 검사 2개 모두 success
- [러너 07:55] merge done — 76f5974
- [러너 08:08] release published — v0.9.87
- [러너 08:09] gh-release created — GitHub Release v0.9.87
- [러너 08:13] assets verified — v0.9.87 자산 2개 (이전 v0.9.86: 2)
