- 과제: UserInfo POST의 폼 본문을 기존 프로토콜 POST와 같은 1MiB로 제한 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/httpserver/oidc.go:userInfo`는 최근 추가된 POST `ParseForm` 앞에 본문 제한이 없어, token·introspect·revoke·logout의 1MiB 제한과 달리 Go 기본 폼 제한인 10MiB까지 읽고 파싱한다. 이 경로에도 기존 제한을 적용하면 토큰 검증 전에 큰 폼을 처리하는 부담을 줄이고 초과 요청에 일관된 400 `invalid_request`를 답할 수 있다.
- 수용 기준: 1) 활성 Realm의 application/x-www-form-urlencoded POST에서 인코딩된 본문이 1,048,576바이트를 넘으면 400 JSON `error=invalid_request`로 거절하며 Content-Length가 없는 chunked 요청도 같다. 2) 실제 발급한 유효 토큰과 무시되는 padding 필드로 만든 정확히 1MiB 본문은 200이고, 기존 GET 헤더·POST 본문·POST 헤더 인증, 헤더+본문 중복 400, 쿼리만 401, 불량 토큰 401 계약이 유지된다. 3) 실제 PostgreSQL·서명 키·세션·`IssueUserTokens`·`New(...).Handler()`·httptest HTTP 서버를 거치는 회귀 테스트가 경계값과 초과값을 증명하고, 수정 전에는 초과 요청 거절 단언이 200으로 실패하는 것을 확인한다.
- 건드릴 파일: `internal/httpserver/oidc.go:userInfo` — POST 분기의 `r.ParseForm()` 바로 앞에 `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)` 적용; 기존 ParseForm 오류의 400 응답 재사용. `internal/httpserver/integration_test.go:TestIntegrationUserInfoReadsTheTokenFromAPostBody` 및 인접 신규 `TestIntegrationUserInfoLimitsPostBody` — 실제 토큰 발급 준비와 HTTP 요청 방식을 재사용해 경계/초과/chunked 회귀 검증. `docs/compatibility.md`의 UserInfo / JWKS 행 — POST 폼 1MiB 상한과 초과 시 400을 짧게 명시.
- 검증 명령: 아래 실행 계획의 명령을 사용한다. 정찰에서 기존 테스트 명령은 실제 실행해 PASS 및 SKIP 0을 확인했다. 새 테스트와 전체 검사 결과는 아직 미확인이다.
- 위험과 피할 것: `auth.go`, 세션 종료 로직, `internal/store/migrations/`, `.github/workflows/`, 프런트, 공용 bearerToken과 폼 파싱 방식은 건드리지 않는다. Realm 조회 순서와 기존 오류 코드를 유지하고 413으로 새 계약을 만들지 않는다. 전역 제한 미들웨어나 공용 파서 리팩터링, Content-Type 정책 변경, logout ParseForm 오류 처리는 범위 밖이다. `Content-Length`만 비교하면 chunked를 놓치므로 반드시 Reader 제한을 쓴다. 본문·토큰을 로그/감사/테스트 실패 출력에 통째로 남기지 않는다. 이전 회차의 이미 완료된 POST 토큰 지원을 다시 구현하지 않는다.
- 차선 후보: README의 통합 테스트 준비를 `scripts/test-services.sh`로 통일 — `README.md` 개발 및 검증 절의 별도 docker run(같은 resso-test-pg 이름, 다른 포트/비밀번호) 대신 `docs/user-federation.md`와 CI가 쓰는 명령을 안내한다. 서비스 스크립트·CI는 수정하지 않는다.

범위와 근거
- 기준 HEAD: `6eab1c2` (v0.9.87). 시작 및 정찰 말미 git status 깨끗함.
- 직접 확인: `oidc.go` token(421), introspect(859), revoke(982), oidcLogout(1068)에 이미 `MaxBytesReader(..., 1<<20)`가 있다. 기존 '프로토콜 다섯 곳에 상한' 아이디어는 사실과 달라 폐기하고 누락된 userInfo(739)만 선택했다.
- `server.go:Handler`, `middleware.go:commonMiddleware/oidcCORS`에는 전역 본문 제한이 없고 `cmd/resso/main.go`의 MaxHeaderBytes는 헤더 제한이다. Go 1.26.7 로컬 표준 라이브러리 `net/http/request.go:parsePostForm`에서 제한 없는 폼의 10MiB 기본 상한을 확인했다. 무제한 읽기라고 주장하지 않는다.
- 실제 대형 요청의 현재 200 응답은 정찰에서 재현하지 않았으며 코드에 근거한 예상이다. 구현 첫 단계에서 이 가정을 검증하고 다르면 과제서에 이유를 기록한 후 차선으로 전환한다.
- 저장소의 CLAUDE.md·AGENTS.md·별도 roadmap 및 TODO/FIXME 표식은 검색에서 발견하지 못했다. README, docs 호환표/운영 및 가이드 관련 절, 최근 git log 30개, Makefile, CI/릴리즈 workflow, 테스트 준비 함수를 확인했다.

대안 비교 (solution-exploration)
- 선택: userInfo 지역 Reader 제한. 기존 네 핸들러와 동일한 구성으로 변경·영향 범위가 가장 작고, 실제 초과 요청의 답이 바뀐다.
- 전역/공용 프로토콜 제한: 다음 엔드포인트 누락 방지에는 유리하지만 GET·JSON·프록시까지 영향을 줄 수 있고 45분 과제에 필요하지 않다.
- 역방향 프록시에만 제한: 코드 변경은 없지만 단일 바이너리 직접 배포의 계약을 보장하지 못한다. 외부 프록시 설정은 미확인이다.
- 현상 유지: 이미 10MiB 제한은 있지만 같은 서비스의 다른 폼 대비 10배 허용하는 이유가 코드에 없으므로 지역 수정의 가치가 더 높다.

실행 계획 (implementation-planning; 모두 미착수, 사람 승인 체크포인트 없음)
1. 회귀 입증: 기존 `TestIntegrationUserInfoReadsTheTokenFromAPostBody` 준비 코드를 참고한다. 올바른 access_token과 `&padding=` 뒤 ASCII로 인코딩된 본문 길이를 정확히 1MiB 및 1MiB+1로 만든다. 헤더 토큰+padding 본문도 확인한다(중복 토큰으로 거절되는 것을 크기 거절로 오인하지 않기). chunked는 실제 `http.Request.ContentLength=-1` 등으로 만들어 네트워크를 통과시킨다. 초과를 400으로 기대하는 새 테스트를 수정 전 실행해 실패를 기록하고, 이후 구현과 테스트를 한 묶음으로 유지한다. 체크포인트: 예상과 다른 실패라면 원인을 확인하기 전 변경하지 않는다.
2. 최소 수정 및 문서: userInfo POST ParseForm 앞 Reader 제한만 추가하고 위 수용 기준의 테스트를 통과시킨다. 정상 GET 및 POST 경로는 그대로인지 기존 테스트로 확인한다. 체크포인트: 로컬 회귀 검사 PASS 후 다음 단계로 진행.
3. 전체 검증: make lint와 서비스 설정 후 make test를 실행한다. 연동 SKIP를 성공으로 보지 않는다. 빌드가 변경한 webui/dist/index.html이 산출물에 섞이지 않게 확인한다. 체크포인트: 실패/미검증과 실제 변경 파일을 기록한 뒤 구현 결과를 넘긴다.

```bash
# 정찰에서 실행한 기존 테스트: PASS (테스트 0.90s, 패키지 1.945s), SKIP 0
RESSO_TEST_POSTGRES_DSN='postgres://resso:resso@127.0.0.1:55439/resso?sslmode=disable' go test -race ./internal/httpserver -run '^TestIntegrationUserInfoReadsTheTokenFromAPostBody$' -count=1 -v
# 구현자의 신규+기존 테스트 (신규 테스트 이름은 위 계획대로)
RESSO_TEST_POSTGRES_DSN='postgres://resso:resso@127.0.0.1:55439/resso?sslmode=disable' go test -race ./internal/httpserver -run '^TestIntegrationUserInfo(ReadsTheTokenFromAPostBody|LimitsPostBody)$' -count=1 -v
# 전체 검증: 저장소의 기존 공식 명령. 현재 서비스 포트는 docker port로 55439 확인.
eval "$(scripts/test-services.sh)"
make lint
make test
git diff --check
```

추정 근거 (estimating-and-contingency)
- Bottom-up: 경계/초과/전달 방식 테스트 10–14분, 최소 수정·문서 3–5분, 회귀 및 전체 검사·결과 기록 10–14분 = 기본 23–33분. 기존 UserInfo POST 과제와 같은 핸들러·DB 테스트 준비를 재사용하므로 유사 작업 규모 S와 일치하나 과거 실제 작업시간은 미확인이라 정량 유사 추정은 사용하지 않는다.
- 알려진 변동 예비시간 5–10분: chunked 구성, 기존 LDAPS 인증서/도구 환경 문제 확인. 합계 28–43분, 신뢰도 중간(정찰자 판단이며 통계적 확률 아님). 전제는 기존 컨테이너와 검사 도구를 재사용할 수 있다는 것.
- 관리 예비시간은 0분 배정(정찰 권한 밖). 45분을 넘길 환경 복구나 새 범위는 묵시적으로 추가하지 말고 미검증 사유를 기록한다. 새 의존성·마이그레이션·UI·배포는 추정에서 제외했다.
- 세 요청 스킬은 Skill 도구 미제공으로 `/mnt/c/Users/USER/projects/headcount/plugins/{pmo,technology}/skills/`의 SKILL.md를 직접 읽어 적용했다. PMO references/sources.md도 확인했으며 외부 가이드의 정량 모델을 인용하지 않고 저장소 증거로 추정했다.
