# PR 처리기 노트 2026-10-17-040852-ReSSO-shepherd — ReSSO PR #43
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-17-032104-ReSSO-improve)
# 회차 노트 2026-10-17-032104-ReSSO-improve — ReSSO
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:21] base pinned — main@2dc5657
- [러너 03:21] autonomy release — 

## 정찰 노트
- `httpserver/oidc.go:653-657` 주석이 "RotateRefreshToken … Those are separate changes and are still to make" 로 이번 과제를 저장소 스스로 지목했고, 이웃 두 경로(648 `InspectRefreshToken`·711 `userLookupFailed`)가 같은 논증으로 이미 500 이라 설계 논쟁 없이 일관성만 맞추면 된다 — 그래서 문서 과제(README 401 문구)를 제치고 이것을 골랐다(문서 회차 3연속을 피하는 뜻도 있다).
- 보류 아이디어 재평가: "make test 두 번째 Go 실행" 은 Makefile:1-11 의 pipefail 이 이미 첫 실행 상태를 잡아 가치가 낮아 기각, "UpdateUser 세션 종료 테스트" 는 기존 테스트가 있어 기각, "test-services CA 디렉터리" 는 147fce8 로 done.
- 과제서의 추측: ① `ALTER TABLE sso_sessions RENAME` 이 FK 의존에도 즉시 성공하는지 실행하지 않았다(대안을 과제서에 적어 둠) ② 수용 기준 5의 "끝난 세션 400 유지" 를 어떤 Endpoint 로 만들지 확인하지 않았다 ③ `sessionIsLive` 정의 본문은 읽지 않았다.
- 구현자 주의: `ErrFamilyNotRevoked` 는 `ErrTokenReuse` 를 함께 감싸므로 새 500 분기는 반드시 재사용 분기 **뒤**에 와야 한다. 반려된 5bed9dc(코드 교환 500)와 선을 긋기 위해 `handleAuthorizationCodeGrant` 는 건드리지 말 것.
- 프로필은 다시 쓰지 않았다(2026-10-09판, 8일 경과·14일 미만이고 읽어 본 범위에서 코드와 어긋나는 곳이 없었다). 바뀐 것은 버전 v0.9.102→v0.9.103, HEAD 2dc5657 뿐이다.
- [러너 03:26] scout done — refresh 그랜트가 `RotateRefreshToken` 의 저장소 장애를 400 `invalid_grant` 로 답해 살아 있는 Refresh Token 을 RP 가 버�

## 구현 노트
- `store/oidc.go:323` 의 `err != nil || !sessionActive` 를 갈라 질의 실패는 원래 오류 그대로, `!sessionActive` 만 `ErrNotFound`. `handleRefreshGrant` 는 재사용 분기 **뒤**에 `!ErrTokenReuse && !ErrNotFound` 분기를 두어 500 `server_error` + `resso_token_errors_total{grant_type="refresh_token"}` + ERROR `the refresh token could not be rotated`. 그 시점에 아무것도 써지지 않았으므로 호출자의 Token 은 살아 있다. 커밋 8d3b77a, 프로덕션 2개·테스트 1개·문서 1개.
- **확신 없는 곳**: 새 500 분기에 닿는 다른 실패 경로(`tx.Begin` 실패, `last_access` UPDATE 실패, `rotated_at` UPDATE 실패, 새 Token INSERT·Commit 실패)는 테스트로 재현하지 않았다 — 테스트가 지나는 것은 세션 생존 질의 하나뿐이다. 나머지는 코드 읽기로만 "nothing written" 을 판단했고, 특히 **Commit 실패는 실제로 써졌는지 알 수 없는 유일한 경우**다(그래도 400 보다 500 이 맞다는 판단).
- **검증 못 한 것**: `make lint` 는 **실패한다** — govulncheck 가 stdlib 취약점 10건(net/http·net/textproto·crypto/tls, go1.26.9 수정)을 보고한다. 깨끗한 트리에서 돌려 같은 10건을 확인했으니 이 변경의 회귀는 아니고, 툴체인 상향은 범위 밖으로 두었다. golangci-lint 0 issues·ESLint 0·`make test` exit 0(SKIP 0)은 통과.
- **일부러 하지 않은 것**: `handleAuthorizationCodeGrant`/`RedeemAuthorizationCode` 무변경(반려된 5bed9dc), `metrics.go` 무변경, 감사 항목 신설 없음(쓰인 것이 없어 기록할 사건이 없다).
- **다음 역할 주의**: 새 테스트는 실제 PostgreSQL 이 있어야 돈다. 환경이 깨져 있었다 — `resso-test-pg` 가 exited 이고 기본 포트 55439 를 **다른 프로젝트의 컨테이너**가 쥐고 있어, resso 컨테이너 셋만 지우고 빈 `/tmp/resso-test-certs` 를 치운 뒤 `RESSO_TEST_POSTGRES_PORT=55440` 으로 재생성했다. 검증 명령에는 그 export 가 필요하다. 다른 프로젝트 컨테이너는 건드리지 않았다.
- 수용 기준 5의 "끝난 세션" 은 로그아웃으로 증명할 수 없다(`RevokeSession` 이 refresh token 도 폐기해 재사용 분기로 간다) — 유휴 시간 초과로 만들었다. ERROR 줄 수 단언은 요청 로그의 500 ERROR 줄을 빼고 세야 한다.
- [러너 03:45] brief accepted — 채택 — 지정한 두 분기 위치·조건식·로그 문구·지표 라벨·문서 불릿·장애 주입 수단을 그대로 썼고, 미확인으로 남긴
- [러너 03:45] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 직접 확인: 두 프로덕션 파일만 main 판으로 임시 되돌려 새 테스트가 `want 500 server_error` 로 FAIL(원장의 실패 재현과 같은 증상), HEAD 로 복원하니 PASS — 테스트는 바뀐 경로를 실제로 지난다. 실제 PG(포트 55440)에서 `go test ./internal/httpserver ./internal/store` ok 90.1s/54.1s, gofmt 깨끗, 트리 복원 완료. 판정 approve(risk low, blocking 없음).
- 못 본 것: `make lint`/`make test` 전체는 재실행하지 않았다(원장의 govulncheck stdlib 10건은 main 에서도 같아 이번 회귀가 아님을 근거로 수용). 프런트엔드·릴리즈 워크플로는 변경이 없어 보지 않았다.
- 승인이어도 남는 우려 ①: `tx.Commit` 모호 실패(서버는 커밋, 연결만 끊김) 때만 `rotated_at` 이 실제로 써진다. 그때 RP 가 30초 grace 를 넘겨 재시도하면 family 폐기 + **일어나지 않은 REFRESH_TOKEN_REUSE 감사 항목**이 남는다 — 변경 전 400 에는 없던 새 경로다. 확률이 낮고 저장소가 `RollbackRefreshRotation` 실패 경로에서 같은 위험을 이미 수용하므로 차단하지 않았다.
- 승인이어도 남는 우려 ②: `docs/operations.md:84` 의 "회전 실패는 롤백되므로 같은 Token 이 그대로 200 으로 교환됩니다" 는 ①에서 무조건 참이 아니다 — **릴리즈 노트에 단정형으로 옮기지 말 것**. 또 취소(`context.Canceled`)는 이번에도 지표에서 걸러지지 않는다(이웃 두 500 과 일관, 유효 Token 필요라 증폭 경로 아님).
- 다음 회차: 원장 보류의 `toolchain go1.26.7 → go1.26.9` 가 실제로 `make lint` 를 막는 유일한 원인이다. 워크트리의 미커밋 `webui/dist/index.html` 도 이 변경과 무관하게 남아 있다.
- [러너 03:50] review approved — 리뷰 승인 (risk=low)
- [러너 03:50] pr created — https://github.com/hkjang/ReSSO/pull/43
- [러너 03:57] ci failed — 성공이 아닌 검사: verify=failure, lint=failure · 실패한 검사: ? 잡: lint 

## 수리 노트
- 맞았던 지적: 비평 노트의 "toolchain go1.26.7 → go1.26.9 가 make lint 를 막는 유일한 원인". lint(govulncheck stdlib 10건)와 verify(Trivy stdlib v1.26.7)가 **같은 한 가지** 원인이었다 — 구현자가 "범위 밖"으로 둔 그 한 줄이 CI 두 검사를 동시에 막고 있었다.
- 틀렸던 것은 없다. 다만 go.mod 만 올리면 `TestToolchainPinsAgreeAcrossTheBuild`(cmd/resso/pins_test.go:96)가 Dockerfile 불일치로 FAIL 한다 — 저장소가 이 실수를 미리 테스트로 막아 두었으므로 두 줄을 함께 올렸다. 커밋 dcfd26f, 프로덕션 코드 0줄.
- 고친 방법: `toolchain go1.26.9` + `FROM golang:1.26.9-alpine`. 8d3b77a 의 refresh 회전 분기·테스트·문서는 한 글자도 건드리지 않았다. 검증은 fix-summary.md 에 명령과 결과를 적었다(핵심: govulncheck 0, trivy 0/exit 0, go test -race ./... 전부 ok SKIP 0).
- **확신 없는 곳**: ① `npm run lint`/`npm test` 는 이 워크트리에 node_modules 가 없어 못 돌렸다(변경이 go.mod·Dockerfile 두 줄이라 영향 없다고 판단) ② Trivy DB 는 로컬이 방금 내려받은 판이라 CI 러너의 판과 다를 수 있다 ③ Go 1.26.7→1.26.9 는 패치 릴리즈이고 전체 테스트가 통과하지만 런타임 거동 차이를 따로 조사하지는 않았다.
- 환경 메모: 이 샌드박스의 `$(go env GOPATH)/bin/go1.26.9` 는 `$HOME/sdk` 를 기대하는 낡은 dl 래퍼라 `make lint` 가 그대로는 "go1.26.9: not downloaded" 로 죽는다. 저장소 문제가 아니다 — golangci-lint·govulncheck 를 절대 경로로 따로 돌려 우회했다.

## 심사 노트
- 직접 확인: 실제 PG(55440)에서 새 테스트 PASS, 두 프로덕션 파일만 origin/main 으로 되돌리니 `want 500 server_error`·ERROR 0줄·지표 0 으로 FAIL — 테스트가 바뀐 경로를 실제로 지난다(트리 복원 확인). `go test -count=1 ./internal/store ./internal/httpserver ./cmd/resso` 전부 ok, gofmt 깨끗, golangci-lint 0 issues, govulncheck 0 (CI 를 막던 stdlib 10건이 go1.26.9 로 사라졌다). 1.26.7 잔존 참조는 CHANGELOG 과거 기록뿐이고 CI·release 는 `go-version-file: go.mod` 라 두 줄로 충분하다.
- 보호 파일 점검: 새 공개 경로·새 기본 켜짐 기능·권한 확대 없음(새 분기는 Token 을 발급하지 않고 `!sessionActive` 는 여전히 400). 로그에 raw token·verifier 없음(테스트가 단언), 지표 라벨은 상수. 마이그레이션 변경 없음 — 테스트의 `ALTER TABLE RENAME` 은 테스트 전용 스키마 안이고 복구된다. revert 로 완전히 되돌아온다.
- 못 본 것: Trivy 는 이미지 빌드가 필요해 재실행하지 않았다(베이스가 go.mod 와 같은 1.26.9 로 핀되고 govulncheck 가 0 인 것을 근거로 수용). `npm run lint`/`npm test` 미실행 — 프런트엔드 변경 0.
- 남는 우려(차단 아님): `tx.Commit` 모호 실패(서버는 커밋, 연결만 끊김) 때만 `rotated_at` 이 실제로 써지므로 oidc.go:758 주석의 "Nothing was written" 과 docs/operations.md:84 의 "같은 Token 이 그대로 200 으로 교환됩니다" 가 그 한 경우에 무조건 참은 아니다. 30초 grace 안의 재시도는 200 이라 응답 문구("retry after a short delay")가 완화하고, `RollbackRefreshRotation` 실패가 이미 같은 위험을 수용 중이다 — **릴리즈 노트에 단정형으로 옮기지 말 것**.
- 권고 근거: 결함 없음, 가용성 쪽으로만 바뀌고 거부 경로는 그대로이며 CI 두 검사의 단일 원인이 최소 변경으로 해소됐다 → approve/merge, risk medium(토큰 경로를 건드린다).
