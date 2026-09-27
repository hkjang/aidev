# 회차 노트 2026-09-28-015228-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:52] base pinned — main@51f44c8
- [러너 01:52] autonomy release — 

## 정찰 노트
- 골랐다: CSP 위반 보고의 문자열 절단. 4회 연속 web/scripts(guide-shots)였던 흐름을 끊고 Go 프로덕션 코드로 돌아왔다 — 차선인 problems 요약(2/1/S)은 운영자만 보는 출력 순서 문제이고, `internal/api/server.go:130` 이 CSP 보고를 세션·CSRF 없이 받는다는 사실이 가치를 더 높게 만들었다. 2026-09-27 에 채택된 dlp.go 절단과 같은 계열이지만 **다른 경로**라, 한쪽만 고쳐 둔 자리를 메우는 일이다.
- 프로덕션 파일은 `internal/tracking/violations.go` 하나. `internal/tracking` 은 runtime-images.json base sourcePaths 에 없으므로 BASE_VERSION 상향이 없다(매니페스트로 확인) — 릴리즈 경로를 건드리지 않는다.
- 추측으로 남긴 것: 깨진 룬이 `writeJSON` 을 거쳐 U+FFFD 로 나가는 것을 실행으로 보지 않았다(성질상 확실하나 구현자가 수정 전 실패 출력으로 남길 것). 상한 값(origin 300 / directive 64 / page 200 룬)은 내 제안이며 근거를 상수 주석으로 적으라고 썼다.
- 조심할 것: 절단은 맵 키(violations.go:66) 계산 **앞**에 두어야 Count 집계가 깨지지 않는다. `allowTrackingOrigin`(api/tracking.go:283)의 감사 details origin 은 같은 계열이지만 다른 경로 — 이번 범위 밖으로 두고 아이디어 파일에 남겼다.
- 프로필은 HEAD·VERSION 이 어긋나 새로 썼다(profile.md, 2026-09-28).
- [러너 01:57] scout done — 인증 없이 들어오는 CSP 위반 보고의 문자열을 룬 경계로 잘라 관리자 화면에 Pod/브라우저가 정한 크기가 �

## 구현 노트
- 무엇/왜: `internal/tracking/violations.go` 에 `maxOriginRunes=300`/`maxDirectiveRunes=64`/`maxPageRunes=200` 과 `cutRunes` 를 두고, `Record` 에서 정규화 뒤·맵 키 앞에 origin·directive·page 를 룬 경계로 자른다. `page[:200]` 바이트 절단이 한글 주소의 마지막 글자를 반 토막 낸 채 남겼고 origin·directive 에는 상한이 아예 없었다. 프로덕션 파일 1개.
- 확신 없는 곳: 상한 값 세 개는 정찰 제안대로이고 근거를 상수 주석에 적었지만 운영 데이터로 검증한 값은 아니다. 잘린 origin 은 `Allowed=false` 로 남는데(테스트로 고정) 그러면 관리자가 그 행의 한 번 클릭 허용을 눌러 잘린 문자열을 설정에 넣을 수 있다 — 이 후속은 손대지 않고 아이디어 파일에 적었다. 관리자 콘솔(`web/src`)이 이 값을 어떻게 렌더링하는지는 여전히 미확인이고 프런트는 건드리지 않았다.
- 검증 못 한 것: DSN 이 없어 `go test -race -p 1 ./cmd/... ./internal/...` 는 DB live 테스트를 건너뛴 상태로 통과했다(이번 변경은 DB 를 지나지 않는다). 실물 브라우저는 미실행.
- 일부러 안 한 것: 라우팅/인증(`server.go`·`auth.go`), `evictOldest`·`List`·`matchesWildcard`·`MaxViolations`, `allowTrackingOrigin` 의 감사 details origin(다른 경로), 잘못된 UTF-8 **입력**의 정규화 — 핸들러 경로는 `encoding/json` 이 이미 유효 UTF-8 로 강제하므로 절단만 룬 경계로 맞췄다.
- 정찰의 미확인 하나는 닫았다: `encoding/json` 은 잘린 바이트 하나당 `�` 를 내보낸다(직접 실행으로 확인).
- 다음 역할이 조심할 것: `TestReportsThatDifferOnlyPastTheLimitAreOneEntry` 가 절단 순서를 지키는 테스트다 — 절단을 맵 키 뒤로 옮기면 `2 entries` 로 실패한다(되돌려 확인). 수용 기준 3 을 과제서 문구대로("같은 보고 두 번") 다시 쓰면 순서를 못 잡으니 이 형태를 유지할 것.
- [러너 02:03] brief accepted — 채택 — 근거가 코드와 정확히 일치했다(`server.go:130` 의 무인증 등록, `tracking.go:41` 의 16KiB 만이 방어, `violations.go:61` 의 바
- [러너 02:04] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인한 것: main 의 violations.go 로 되돌려 실행해 새 api 테스트가 실제로 실패함을 봤다(origin 4016 룬, page 끝에 `\xea` — 커밋 주장과 증상 일치). cutRunes 는 바이트 길이를 룬 상한과 비교하는 빠른 경로까지 안전하고, 절단은 맵 키 계산 앞에 있으며, tracking 패키지에 남은 다른 바이트 절단은 없다(grep). go vet·gofmt 깨끗, 범위 이탈 없음(파일 3개).
- 승인이어도 남는 우려 1: api 테스트의 상한 단언이 세 값 모두 300 룬으로 느슨하다 — maxDirectiveRunes 를 올리는 회귀는 그 테스트를 통과한다(패키지 테스트가 상수로 막아 주므로 결함은 아님). tracking 패키지 테스트 두 개는 비공개 상수를 참조해 수정 전 코드에서 컴파일이 안 되니, 절단 순서 검증은 구현자가 적은 수동 방법만 유효하다.
- 승인이어도 남는 우려 2: origin 절단이 List 의 Allowed 판정 앞이라, matchesWildcard(violations.go:174)의 prefix+suffix 매칭과 GA4/GTM 의 `https://*.google-analytics.com`(tracking.go:292) 때문에 앞 300 룬이 그 접미사로 끝나게 채운 보고가 화면에서 '허용됨' 으로 오표기될 수 있다. 얻는 것이 메모리 목록의 한 행뿐이라 차단하지 않았다.
- 다음 회차 후보(이번 범위 밖의 기존 결함): 인증 없는 보고의 문자열이 관리자 한 번 클릭(AdminSettings.tsx:220 → api/tracking.go:253)으로 AllowedHosts 에 들어가는데 검증이 `HasPrefix "http"` 뿐이다 — 와일드카드 출처를 보고해 CSP 를 넓히도록 유도할 수 있다.
- 못 본 것: 실물 브라우저 e2e, DB live(이번 경로는 DB 미경유), 관리자 콘솔 실제 렌더링. 릴리즈는 BASE_VERSION 상향 불필요(base sourcePaths 에 internal/tracking 없음, 매니페스트 확인).
- [러너 02:08] review approved — 리뷰 승인 (risk=low)
- [러너 02:08] pr created — https://github.com/hkjang/AgentHub/pull/37
- [러너 02:10] ci passed — 검사 1개 모두 success
- [러너 02:10] merge done — d985b81
- [러너 02:19] release published — v0.254.0
- [러너 02:25] assets verified — v0.254.0 자산 8개 (이전 v0.253.0: 8)
