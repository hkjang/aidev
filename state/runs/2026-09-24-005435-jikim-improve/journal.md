# 회차 노트 2026-09-24-005435-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:54] base pinned — main@bb652c9
- [러너 00:54] autonomy release — 

## 정찰 노트
- 골랐다: Vite 개발 서버에 `/.well-known/oauth-protected-resource` 프록시 추가. 보류 항목 중 유일하게 근거가 정적으로 확실하고(Go 라우트 `server.go:150-151` 존재 vs `vite.config.ts` 6개 항목에 없음) 재사용 가능한 실제-HTTP 하네스(`web/src/lib/vite-proxy.test.ts`)가 이미 있다.
- 제친 이유: settings 파생 필드 왕복은 다음 GET이 값을 덮어써 관찰 가능한 변화가 없어 운영자의 "효과 없는 수정 금지"에 걸린다(가치 2→1로 내림). audit 보존·baoKVWrite TOCTOU는 계약 확정이 필요한 M. 메일 관련 두 건은 HEAD에 기능이 없어 rejected.
- 추측으로 적은 것: 개발 서버가 그 경로에 HTML을 준다는 것을 **직접 보지 못했다**(`web/node_modules` 없음). 2026-09-22의 동일 구조 재현이 근거다. 구현자는 수정 전 실패를 먼저 관찰할 것.
- 조심할 것: 프록시 키를 `/.well-known` 로 넓히지 말 것(devtools 자동 요청이 백엔드로 샌다). `changeOrigin`·`rewrite` 금지 — `mcp_oauth.go` 의 `Host` 폴백 계약이 흔들린다. Go 쪽은 손대지 않는다.
- 프로필은 이전 것이 대체로 맞지만 "요청한 스킬/Skill 도구를 찾지 못함"이 이번에 틀린 것으로 확인되어(세 스킬 모두 로드됨) 정정해 새로 썼다.
- [러너 00:59] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `vite.config.ts`에 `/.well-known/oauth-protected-resource` 프록시 한 줄을 넣었다. Go는 이 경로와 `…/mcp`를 서빙하는데 개발 서버는 넘기지 않아 관리 화면이 안내하는 메타데이터 주소와 `/mcp` 401의 `resource_metadata`가 개발 환경에서 404였다. CONTRIBUTING 개발 안내에 주소 한 문장 추가.
- 정찰 추측 두 곳이 틀렸음: (1) 개발 서버는 HTML이 아니라 **404**를 준다. (2) Vite 8의 문자열 단축 프록시는 `changeOrigin: true`를 함의해(node_modules/vite `proxyMiddleware`) Host가 백엔드로 바뀐다 — 기존 `/mcp` 항목도 마찬가지라 새로 생긴 문제는 아니지만, `mcp_oauth.go`의 Host 폴백은 개발에서 `127.0.0.1:8080`을 본다. 그래서 절대값 대신 "메타데이터와 `/mcp`가 같은 Host로 도착한다"를 테스트로 고정했다.
- **확신 없는 곳**: Go 핸들러와 Vite 프록시를 한 프로세스에서 이은 end-to-end는 못 했다(교차 언어). 근거는 Go 테스트(핸들러가 두 경로를 서빙)와 새 vitest(개발 서버가 그 경로를 그대로 백엔드로 전달) 두 쪽이고, 이음매는 양쪽에 동일한 경로 문자열뿐이다. 실제 Keycloak·MCP 클라이언트로 개발 서버에 붙는 확인도 불가.
- 일부러 안 한 것: 프록시 키를 `/.well-known`으로 넓히지 않았다(devtools `appspecific` 요청이 백엔드로 새지 않는지 테스트로 고정). `changeOrigin`·`rewrite`도 넣지 않았다. Go 라우트는 손대지 않았다.
- 다음 역할 주의: `web/src/lib/vite-proxy.test.ts`는 `@vitest-environment node`로 실제 Vite 개발 서버를 띄운다(포트 0, 임시 cacheDir). 첫 실행은 `npm ci` 필요. DB는 필요 없다. 테스트 수 53→59.
- [러너 01:04] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 프록시 한 줄을 임시로 빼고 돌려 새 테스트 5개가 실제로 실패하는 것을 눈으로 봤고(되돌린 뒤 15/15 통과, 트리 clean), Go 가 bare 경로와 `/mcp` 접미 경로를 모두 등록하는 것(`server.go:150-151`)과 경로 상수 일치(`mcp_oauth.go:36`)를 확인했다. 범위는 3파일뿐이고 CHANGELOG·version 은 관례대로 안 건드렸다.
- 못 본 것: Go+Vite 를 한 프로세스로 이은 end-to-end, 실제 MCP 클라이언트/Keycloak. 구현자와 같은 한계다.
- 승인이어도 남는 우려(릴리즈 노트감): 문자열 단축 프록시가 `changeOrigin` 을 함의해 개발 환경에서 `mcpResource` 가 광고하는 resource 는 5173 오리진이 아니라 `http://127.0.0.1:8080/mcp` 다. 개발 서버로 MCP SSO 를 시험하는 사람은 이걸 알아야 한다.
- 다음 회차 주의: Vite 문자열 프록시 키는 접두사 매칭이라 `/.well-known/oauth-protected-resource-foo` 도 백엔드로 샌다(개발 한정, 공격 경로 없음). `/.well-known` 아래를 더 늘릴 때 다시 볼 것.
- 판정: approve / risk low / blocking 없음 (security·legal 모두 차단 사유 없음 — 개발 서버 설정, 인가 표면·개인정보·의존성 변화 없음).
- [러너 01:06] review approved — 리뷰 승인 (risk=low)
- [러너 01:06] pr created — https://github.com/hkjang/jikim/pull/42
- [러너 01:09] ci passed — 검사 2개 모두 success
- [러너 01:09] merge done — 8c7a4fe
- [러너 01:17] release published — v0.2.19
- [러너 01:21] assets verified — v0.2.19 자산 2개 (이전 v0.2.18: 2)
