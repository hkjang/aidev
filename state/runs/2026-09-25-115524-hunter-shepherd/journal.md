# PR 처리기 노트 2026-09-25-115524-hunter-shepherd — hunter PR #10
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-25-104129-hunter-improve)
# 회차 노트 2026-09-25-104129-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:41] base pinned — main@50cc993
- [러너 10:41] autonomy release — 

## 정찰 노트
- 보류 아이디어 절반이 아직 차단 상태였다(mail*.go, trackingSnippetOrigins 모두 main@50cc993에 부재 — ls/grep 확인). csvSafe·handoff 상한·서로게이트 clip 세 건은 v1.13/v1.14로 머지되어 done. views 8개 상한 알림 후보는 list-tools.tsx:133에 이미 안내가 있어 rejected.
- 고른 이유: 2026-09-20 CSV 회차의 "공유 JSON 벡터로 Go/TS 두 파서 교차 검증"이 이 저장소에서 통한 패턴이고, tracking 쪽은 auth가 아니며 `trackingOrigin`이 순수 함수라 Go 테스트에 DB가 필요 없어 OIDC return_to 후보(auth 보호 경로)보다 위험이 낮다.
- 추측으로 적은 것: 브라우저 `URL`이 `0177.0.0.1`·`foo.123` 같은 호스트를 어떻게 정규화하는지 런타임으로 확인하지 못했다(Node 실행이 승인에 걸림). 구현자는 `node -e`로 후보를 먼저 찍어 진짜 차이 케이스를 확정한 뒤 범위를 정하고, 차이가 없으면 차선(전체 길이 산식 일치)으로 갈 것.
- 조심할 것: 서버 `trackingOrigin`을 느슨하게 고쳐 맞추지 말 것 — 숫자 끝 라벨 거절은 레거시 IPv4 별칭으로 앱 원점 차단을 우회하는 것을 막는 보안 규칙이다(코드 주석). 방향은 항상 TS→Go.
- [러너 10:46] scout done — CSP 위반의 "허용" 원클릭 버튼이 서버가 거절할 원점을 제안하는 문제 — TS `trackingViolationOrigin`/`validateTrack

## 구현 노트
- 왜: 미리보기 위반의 "허용 목록에 추가" 버튼과 허용 주소 입력이 브라우저 `URL` 파서만 믿어 서버 `trackingOrigin`이 거절할 원점을 제안·통과시켰다(저장 PUT 400). `normalizeTrackingOrigin` 하나를 두고 `trackingViolationOrigin`·`validateTrackingDraft`가 그것만 쓰게 했고, 전체 길이도 서버처럼 정규화·중복 제거 후 `len+1` 누적으로 맞췄다. 서버 `internal/app/tracking.go`는 한 줄도 바꾸지 않았다(방향은 TS→Go).
- 확신 없는 곳·검증 못 한 것: (1) 실제 브라우저 렌더·클릭은 확인하지 못했다 — `tracking.tsx`는 미접촉이고 `origin === null`이면 이미 버튼을 숨기는 기존 분기를 그대로 쓴다(tsc·vite build 통과, import 배선 확인). (2) `[::ffff:127.0.0.1]` 같은 IPv4-mapped IPv6는 TS가 `[::ffff:7f00:1]`, Go가 `127.0.0.1`로 정규화한다 — 둘 다 서버가 받아들여 400은 안 나지만 문자열이 달라 벡터에 넣지 않고 함수 주석에 적었다. (3) Node 22로 테스트했고 CI는 Node 26이다. UTS46 판정이 버전 간 달라질 수 있는 IDN 사례는 한글 호스트·punycode·`xn--a`(무효) 세 개뿐이다. (4) `HUNTER_TEST_DSN`이 없어 DB 경로 tracking 테스트 5개는 SKIP이다(벡터 테스트는 순수 함수라 skip 없이 PASS).
- 일부러 하지 않은 것: 서버 파서 완화(숫자 끝 라벨 거절은 레거시 IPv4 별칭 우회를 막는 보안 규칙), `tracking_violations.go` record()·revision 계약, 문서 재생성(설정·API 계약 변화가 없고 관리자 화면 문구도 그대로), 전체 `go test ./...`(DB 필요·수백 초, Go 변경은 테스트 파일뿐).
- 다음 역할이 조심할 것: 벡터 파일 `internal/app/testdata/tracking-origins.json` 하나를 Go(`os.ReadFile("testdata/...")`)와 웹(`new URL("../../internal/app/testdata/...")`) 두 테스트가 읽는다 — 복사본을 만들지 말고 옮길 때 두 경로를 같이 고쳐야 한다. 기존 웹 테스트의 1,536바이트 사례는 입력 라벨이 156바이트(서버가 애초에 거절할 값)여서 60/60/31로 바꿨다. 프런트를 또 바꾸면 `cp -a web/dist/. internal/webassets/dist/`를 다시 해야 한다(dist는 gitignore).
- [러너 11:02] brief accepted — 채택 — 과제서가 지목한 두 파서의 불일치가 런타임으로 26건 재현되어 제안한 범위·파일·벡터 위치를 그대로 따랐다(�
- [러너 11:02] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 테스트 유효성은 직접 확인했다 — main 의 구 TS 로직으로 102개 벡터를 재생하면 26건 불일치가 나고 현재 코드에서 전부 통과하므로 수정 전에는 통과하지 않는 진짜 회귀 테스트다. `go test -run TestTracking` 전부 PASS(DB 1건 SKIP), web 95/0/0, tsc·go vet·gofmt·prettier 통과.
- 무작위 3,342건 + 표적 94건으로 Go `trackingOrigin` 과 TS `normalizeTrackingOrigin` 을 차등 실행했다. TS 가 서버보다 엄격한 사례는 0건(관리자가 유효 원점을 못 넣는 방향의 결함 없음), 서버 파일 무변경, 마이그레이션 없음 → revert 안전.
- 남는 우려(차단 아님): idna CheckHyphens(V2) 계열이 미반영이라 `http://my--service.internal`·`api--v2.internal` 을 UI 가 유효로 보고 PUT 은 400 을 준다(`xn--` 는 서버가 면제). bidi 계열(`https://٢`)도 동일. main 과 같은 동작이라 회귀는 아니지만 벡터 75건에 이 규칙이 없다.
- 따라서 두 테스트 주석의 "PUT 이 거절할 원점을 결코 제안·통과시키지 않는다"는 과장이다. 릴리즈 노트에 "완전 일치"로 적지 말고 "legacy IPv4 별칭·라벨·포트·중복 합산 규칙 일치"로 한정할 것. 다음 회차 후보: CheckHyphens/bidi 벡터 2건 추가.
- 못 본 것: 실제 브라우저 렌더·"허용 목록에 추가" 클릭, 전체 `go test ./...`, Node 26(로컬은 Node 22 — UTS46 판정 차이 가능). 릴리즈 전 `cp -a web/dist/. internal/webassets/dist/` 재수행 필요(dist 는 gitignore).
- [러너 11:09] review approved — 리뷰 승인 (risk=low)
- [러너 11:09] pr created — https://github.com/hkjang/hunter/pull/10
- [러너 11:11] ci failed — 성공이 아닌 검사: verify=failure

## 수리 노트
- 지적(CI verify 실패)은 맞았다. 구현 노트가 "확신 없는 곳 (3) Node 22/26 UTS46 차이"로 적어둔 바로 그 지점이 터졌다: Node 26의 ada 파서는 `http://xn--a.internal`을 검사 없이 통과시키고 ICU 빌드는 거절한다. Node 26.10.0을 직접 받아 같은 자리에서 재현했다.
- 고친 방법: TS에 RFC 3492 디코더를 넣어 `xn--` 라벨을 직접 풀고, 파서에는 두 런타임이 일치하는 유니코드→ASCII 방향만 남겼다. x/net/idna:403의 "ASCII로만 디코드되는 ACE 라벨 거절"(보안 사유) 규칙도 옮겼다. 서버는 무변경, 테스트·단언은 느슨하게 하지 않았고 벡터 3건을 오히려 추가했다.
- 근거: Go와의 차등 실행 3,400여 건에서 Node 22/26 결과가 완전히 동일해졌고, 비평가가 승인한 Node 22 프로파일과 불일치 집합이 정확히 일치한다(신규 0, Node 26 전용 7~14건 제거).
- 여전히 확신 없는 곳: 혼합 ACE 라벨(`xn--한글-989an41e`)은 idna가 받고 UI는 거절한다 — 기존 ICU 동작과 같아 유지했지만 서버보다 엄격한 유일한 방향이다. UTS46 CheckHyphens/bidi/카테고리 미반영 3~5건도 그대로다(다음 회차 후보).
- 못 한 것: 실제 브라우저 클릭, DB 테스트(`HUNTER_TEST_DSN` 없어 skip), 전체 `go test ./...`.

## 심사 노트
- 확인한 것: 구 TS 로직을 재구현해 벡터에 돌리니 27건 불일치 → 현재 0건(진짜 회귀 테스트). 표적 94 + 난수 4,000 차등 실행에서 "UI 통과·서버 거절" 방향이 main 353건 → 14건이고 **새로 생긴 위험 방향 0건**. 서버 tracking.go 무변경, 보호 파일·마이그레이션 없음, Dockerfile 이 web 을 직접 빌드해 릴리즈 경로 자동 반영.
- 실행: web test 95/0/0, tsc·vite build·prettier·gofmt·go vet 통과, go test -run TestTracking 통과(DB 5건 SKIP — 통과로 보지 않음).
- 못 본 것: Node 26(로컬 부재). 단 원래 실패했던 xn--a 류는 ASCII 호스트가 파서에 닿지 않는 경로로 바뀌어 런타임 의존이 제거됐고, 남은 의존은 비ASCII ToASCII·대괄호 IPv6 뿐이다. 실제 브라우저 클릭, 전체 go test ./... 도 미확인.
- 남긴 권고(차단 아님): 두 테스트 주석의 "PUT 이 거절할 원점을 결코 통과시키지 않는다"는 CheckHyphens/bidi 14건 때문에 과장 — 릴리즈 노트는 "레거시 IPv4 별칭·라벨·포트·중복 합산 규칙 일치"로 한정. 또 `%`-인코딩 UTF-8 호스트 2건은 이제 UI 만 거절(서버는 수용)하는 유일한 신규 엄격화로, 저장 실패·제안 경로 영향은 없으나 다음 회차에 방향을 정할 것.
- 근거: 결함은 없고 불일치를 단조 감소시키며 서버가 최종 판정자로 남아 보안 경계가 이동하지 않는다 → approve / merge (risk low).
