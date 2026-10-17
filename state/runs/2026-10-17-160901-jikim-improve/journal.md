# 회차 노트 2026-10-17-160901-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:09] base pinned — main@69b62df
- [러너 16:09] autonomy release — 

## 정찰 노트
- 우선 과제가 지정한 '실패' 는 저장소 CI·릴리즈 워크플로 실패가 아니라 **지난 회차가 PR 전에 끊겨 과제가 미구현으로 남은 것**이다(base main@69b62df, 트리 clean, v0.2.39 릴리즈 커밋까지 정상 반영). 그래서 워크플로 파일은 건드릴 것이 없고, 끊긴 그 과제(auth/token/create 의 chunked 본문 누락)를 지금 코드로 다시 검증해 같은 범위로 과제서에 적었다. 워크플로를 느슨하게 만드는 변경은 없다.
- 다른 후보를 제친 이유: transit 계약 동등성 표(2/1/S)는 프로덕션 변경이 없고, baoKVMutateVersions 0·음수(2/2/S)는 실제 OpenBao 판정 근거가 아직 없다. 1순위는 **같은 저장소의 다른 세 호출부가 이미 `!= 0` 으로 옳게 적혀 있어** 계약 판단이 필요 없는 한 줄 결함이다.
- 실행으로 확인한 것: Content-Length 가 있는 `{"ttl":"99999h"}` 는 400 `invalid ttl`, `ContentLength = -1` 로 바꾼 같은 본문은 ttl·JSON 검사를 모두 통과해 `store.CreateSession` 까지 내려가 nil store panic. 하네스(`quietServer` + `sessionResolver` + `openBaoRoutes` mux)가 store 없이 실제 라우트를 통과하는 것도 확인. 임시 테스트는 삭제했고 트리는 clean.
- 추측으로 적은 것: `sessionCreator` seam 추가 형태와 `decodeBaoJSONBody(allowEmpty)` 헬퍼 형태는 권장안이며 컴파일 검증은 하지 않았다. compatibility.md 의 문장 삽입 위치(51-52행 근처)도 행 번호만 확인했다.
- 구현자가 조심할 것: `r.ContentLength != 0` 로만 바꾸면 **빈 chunked 본문이 400** 이 된다(`Decode` 가 `io.EOF`) — 본문 없는 요청이 기본 12시간으로 성공하는 현행 계약을 깨지 말 것. 권한 거절(admin·manager)은 디코드 **전**에 남겨야 한다.
- [러너 16:14] scout done — OpenBao `auth/token/create` 가 길이를 알 수 없는(chunked) 요청 본문을 조용히 버려 `ttl`·`display_name` 이 무시되고 �

## 구현 노트
- `openbao.go:241` 의 `r.ContentLength > 0` 게이트가 chunked 본문(`ContentLength = -1`)을 읽지 않아 `ttl`·`display_name` 이 무시되고 깨진 JSON 도 통과했다. 게이트를 `decodeOptionalBaoJSON` 호출로 바꾸고 `decodeBaoJSON` 을 `decodeBaoJSONBody(…, allowEmpty bool)` 공용 코어로 뽑았다(`io.EOF` 만 통과 → 본문 없는 요청의 기본 12시간 계약 유지). `server.go` 에 `sessionCreator` seam 1개 추가.
- 확신 없는 곳: `decodeOptionalBaoJSON` 은 지금 `baoTokenCreate` 한 곳만 쓴다 — 다른 OpenBao 핸들러에 맞는지는 **확인하지 않았고** 확장하지 않았다. `sessionCreator` seam 의 실제 store 경로(nil 일 때 `store.CreateSession`)는 DB 가 필요해 테스트로 통과시키지 못했다; 컴파일·기존 전체 테스트 통과로만 확인했다. 실제 OpenBao 서버가 chunked 본문을 어떻게 다루는지는 대조하지 않고, 같은 저장소의 다른 세 호출부(`!= 0`)와의 일관성만 근거로 삼았다.
- 일부러 하지 않은 것: `resource_handlers.go:521`·`oidc.go:68`·`ai.go:228` 은 이미 `!= 0` 이라 손대지 않음. `DisallowUnknownFields` 미추가(호환성 계약 변경). `maxJSONBody`·`UseNumber`·기존 두 에러 메시지·권한 검사 순서 불변. 버전·CHANGELOG 미변경.
- 다음 역할이 조심할 것: 새 테스트는 DB 없이 돈다(`quietServer` + `sessionResolver`/`sessionCreator` 주입 + `openBaoRoutes` 로 실제 라우트·`withBaoAuth` 통과). chunked 재현은 `request.ContentLength = -1` 한 줄에 달려 있으니 지우면 테스트가 결함을 못 본다. `bash scripts/verify.sh` exit 0, 프런트 59/59.
- [러너 16:18] brief accepted — 채택 — 행 번호(openbao.go:241·883-896, server.go:28-49·221-226, store/users.go:52, compatibility.md 51-52 근처)·권장 수정 형태(`decodeBaoJSONB
- [러너 16:19] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 게이트 한 줄만 옛 형태로 되돌려(헬퍼 유지) chunked 6개가 원장과 같은 줄·같은 본문으로 실패하는 것을 직접 재현하고 복원했다(트리 clean). 테스트는 `openBaoRoutes`+`withBaoAuth` 실제 라우트를 타고 거절마다 생성기 0회를 단정하므로 수정 전엔 통과 불가. `go test ./... -count=1`·`go vet`·`gofmt -l` 재실행 모두 깨끗.
- 구현자가 의심한 자리: `sessionCreator` nil 경로는 server.go:229-234 ↔ store/users.go:52 시그니처와 openbao.go:253 인자 6개를 글자 단위로 대조해 불변 확인. seam 은 unexported 라 외부 주입 불가.
- 승인이어도 남는 우려(릴리즈 노트용): 공백만 있는 본문(`"   "`)이 `io.EOF` 로 판정돼 **Content-Length 경로에서도** 400 → 200+기본 12시간으로 바뀐다(실측). 무해하지만 compatibility.md 새 줄의 "본문이 아예 없는 요청만" 과 글자대로 어긋나고(`{}`·`null` 도 통과), 2MB 초과 chunked 본문이 조용한 성공 대신 400 이 된다.
- 못 본 것: `scripts/verify.sh` 전체·프런트·실제 OpenBao 서버의 chunked 처리 대조는 이 세션에서 재실행하지 않았다(구현 노트의 exit 0 보고에 의존).
- 보안·법무 차단 없음: 새 인가 경로·식별자·비밀값 노출·의존성·개인정보 수집이 없고, 토큰 수명 12배 문제를 닫는 쪽이라 방향은 강화다. 마이그레이션 없어 revert 안전.
- [러너 16:21] review approved — 리뷰 승인 (risk=low)
- [러너 16:22] pr created — https://github.com/hkjang/jikim/pull/65
- [러너 16:25] ci passed — 검사 2개 모두 success
- [러너 16:25] merge done — ebd6c59
- [러너 16:33] release published — v0.2.40
- [러너 16:36] assets verified — v0.2.40 자산 2개 (이전 v0.2.39: 2)
