# 회차 노트 2026-09-24-162413-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:24] base pinned — main@6b52613
- [러너 16:24] autonomy release — 

## 정찰 노트
- 저장 결과 계열은 최근 3회 연속 채택·릴리스됐고 이 과제는 남은 흠(상태코드)이다 — 코드 4곳을 직접 열어 Get 전부 404·Delete 전부 400 을 확인했다. 검색(3/3/L)·세션 만료(3/2/M)는 권한/세션 위험 구역이거나 한 세션 밖이라 제쳤다.
- 추측으로 적은 것: 차선 후보의 `login.js` Retry-After 처리는 **미확인**. 세션 만료 알림이 기대는 `auth/me` 의 expires_at 도 이번에는 재확인하지 않았다.
- 구현자가 조심할 것 ①: `ErrNotFound` 문구를 바꾸면 saved.js 가 `cause.message` 를 그대로 띄우므로 화면 안내가 바뀐다 — 현행 문구를 유지할 것.
- 구현자가 조심할 것 ②: 같은 결함이 `history.go:55`·`http.go:1659` 에도 있지만 범위를 넓히지 말 것. 드라이버 원문 차단은 이미 `writeProblem` 한 곳에 있으니 다시 손대지 말고 회귀 테스트만 붙일 것.
- 테스트는 fake Repository 를 `resultsave.NewService` 에 넣어 실제 `New(logger, options)` 라우터를 지나게 할 것 — 핸들러 직접 호출 대역은 배선을 못 본다(운영자 지침).
- [러너 16:27] scout done — 저장 결과 조회·삭제 실패의 HTTP 상태를 원인과 맞추기 (없는 결과·권한 없음 404 / 잘못된 ID 400 / 저장소 �

## 구현 노트
- 저장 결과 조회·삭제의 실패를 원인별 상태코드로 갈랐습니다(커밋 6f5c1d7): resultsave 에 `ErrNotFound`·`ErrInvalidID` 센티널 → mariadb 저장소가 반환 → 핸들러의 `writeSavedResultProblem` 이 `errors.Is` 로 404/400/500. 안내 문구는 saved.js 가 그대로 띄우므로 한 글자도 바꾸지 않았습니다.
- 확신 없는 곳: 저장소 쪽 두 자리는 sqlmock 으로만 확인했습니다 — 실제 MariaDB 에서 `sql.ErrNoRows`·`RowsAffected()==0` 이 나는 상황(실물 권한 조건)은 이 환경에 DB 가 없어 돌리지 못했습니다. 브라우저 스모크도 돌리지 않았습니다(근거: saved.js 는 `cause.message` 만 쓰고 layout.js `api()` 는 401 만 특별 취급하므로 400→500 전환이 화면 문구를 바꾸지 않음 — 소스로 확인, 실행 확인 아님).
- 일부러 하지 않은 것: `history.go:55`·`http.go:1659` 의 같은 '모든 실패 404' 는 과제서가 범위 밖으로 못 박아 두었습니다. `safeProblemDetail` 자리는 손대지 않고 회귀 테스트만 붙였습니다.
- 과제서와 다른 점: 실제 라우트는 `/api/v1/query/saved/{id}`(과제서의 `/api/v1/saved-results/{id}` 아님). DELETE 는 `csrfProtection`(middleware.go:14)을 지나야 해 테스트가 세션의 실제 CSRF 토큰을 `X-DartFly-CSRF` 로 보냅니다 — 과제서에 없던 사항입니다.
- 다음 역할이 조심할 것: 새 테스트 2파일은 DB 없이 돕니다(sqlmock·httptest). `internal/server/resultsave_test.go` 의 `savedResultRequest` 는 요청마다 세션을 새로 만듭니다 — 세션 재사용을 전제로 고치면 CSRF 가 어긋납니다.
- [러너 16:32] brief accepted — 채택 — 지목한 코드 4곳이 모두 현재 코드와 일치했고 순환 import 없음도 맞았습니다. 다른 점은 경로 하나뿐입니다: 실�
- [러너 16:32] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 핸들러만 main 판으로 임시 복원해 돌려 새 테스트 8개 중 6개가 실패함을 실측했습니다 — 테스트는 진짜 판별적입니다. 복원 후 worktree clean, `go build`·`go test ./...`·`go vet`·`gofmt -l` 모두 통과.
- 구현자가 의심한 두 자리를 확인: sqlmock 인자 순서는 실제 SQL 과 일치, layout.js:102-104 이 401 만 특별 취급하므로 400→500 전환이 화면 문구를 안 바꿉니다. 실물 MariaDB 검증은 저도 못 했습니다(환경에 DB 없음) — `sql.ErrNoRows`·`RowsAffected()==0` 의 실제 발생은 여전히 미검증.
- 보안·법무 소견 없음: 인가 변경 없고, ErrNotFound 가 '없음'과 '남의 것'을 같은 404·같은 문구로 묶어 열거 오라클이 안 생깁니다. 새 500 도 safeProblemDetail 을 지납니다. 신규 개인정보·신규 의존성 없음.
- 남는 우려: safeProblemDetail 기본 안내가 "저장하지 못했습니다. 입력값을 확인하세요" 라서 조회 중 DB 장애 500 본문이 어색합니다(dberror.go:79, 기존 문제·다음 회차 후보). history.go:55·http.go:1659 의 같은 결함은 그대로 — 릴리즈 노트는 "저장 결과 조회·삭제"로 한정할 것.
- 미확인: 브라우저 스모크·livedb 미실행(화면 마크업 변경이 없어 필요한 종류가 아니라고 판단). 외부 클라이언트가 있다면 API 상태코드 계약 변경이라는 점은 릴리즈에서 언급할 가치가 있습니다.
- [러너 16:35] review approved — 리뷰 승인 (risk=low)
- [러너 16:35] pr created — https://github.com/hkjang/DartFly/pull/13
- [러너 16:40] ci passed — 검사 3개 모두 success
- [러너 16:40] merge done — 6f5c1d7
- [러너 16:44] release published — v2.74.0
- [러너 16:44] gh-release created — GitHub Release v2.74.0
- [러너 16:44] manifest ok — dartfly-v2.74.0.tar.gz dartfly-v2.74.0.tar.gz.sha256 
- [러너 16:44] assets uploaded — 2개
- [러너 16:44] assets verified — v2.74.0 자산 2개 (이전 v2.73.0: 2)
