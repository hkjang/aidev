# 회차 노트 2026-10-04-164215-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:42] base pinned — main@2f18d04
- [러너 16:42] autonomy release — 

## 정찰 노트
- 고른 이유: 2026-10-02 의 ai.go 과제가 **머지되지 않았음을 확인했다**(`git log -12` 에 없고 HEAD 2f18d04 에서 `grep -rn "secret_material_rejected|errAISecretMaterial" internal` → ai.go:94 한 줄뿐). 당시 떨어진 사유는 변경이 아니라 프런트 vitest 가 Node 20 으로 돌던 환경이었고 그 원인은 `7a13e85`(v0.2.27)로 이미 닫혔다 — 사람 반려가 아니라 환경 실패이므로 재시도가 정당하고, 당시 구현자 판정도 "채택" 이었다.
- 다른 후보를 제친 이유: `_ =` 광맥은 소진, temperature 클램프는 쓰기 경로가 0~2 를 강제해 관찰 가능한 변화를 못 만들고(4회 연속 미실행), CSP 속도 제한·nonce 파싱·TOCTOU·remoteIP 는 새 계약이나 PostgreSQL·브라우저가 필요해 한 세션 밖이다. `~/node_modules` 심링크는 저장소 밖 전역 변경이라 rejected 로 닫았다.
- 확신 있는 것(열어서 확인): ai.go:55-96 이 다섯 원인을 한 code 로 보고한다, `internal/httpapi` 의 400 `writeError` 24곳 중 이런 자리는 여기뿐, 라우트 server.go:141·전역 체인 server.go:68·`requestToken`(205-216) 순서·`revokeServer`(auth_revoke_test.go:16-29) 하네스·docs 에 AI error code 표 없음·`invalid_ai_request` 미사용, base 에서 `go test ./internal/httpapi/` green(0.682s).
- 추측/미확인: 262144 초과 테스트 입력이 마지막 Secret 패턴(`[A-Za-z0-9+/_=-]{48,}`)에 먼저 걸리는지는 실제로 돌려 보지 않았다 — 과제서에 공백 섞은 입력으로 분기를 code 로 확인하라고 적어 두었다. `bash scripts/verify.sh` 는 느려서 이번 정찰에서 돌리지 않았다. 구현자가 **반드시** exit 0 까지 확인할 것 — 지난 회차가 정확히 그 자리에서 떨어졌다.
- 조심할 것: `secretMaterialPatterns`·262144 리터럴·메시지 문구는 한 글자도 바꾸지 말 것(`security_test.go:16` 이 문구를 못박았다). CHANGELOG·`web/` 은 건드리지 말 것.
- [러너 16:48] scout done — AI 채팅 입력 검증 실패를 전부 `secret_material_rejected` 로 보고하는 문제 분리 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `aiChat` 이 `ValidateAIInput` 의 다섯 가지 거절 전부를 `secret_material_rejected` 로 보고했다. 패키지 수준 sentinel `errAISecretMaterial` 을 추가해 Secret 매처 루프만 그것을 반환하게 하고, 핸들러가 `errors.Is` 로 `secret_material_rejected` / `invalid_ai_request` 를 가린다. 프로덕션 파일 1개(internal/httpapi/ai.go, +11줄) + 테스트 1개 신규. 커밋 `03cbae1`.
- 확신 없는 곳·검증 못 한 것: `invalid_ai_request` 는 **새 공개 계약 문자열**이다 — 저장소·docs·web/src 어디에도 AI error code 가 계약으로 적혀 있지 않고 프런트가 code 로 분기하지도 않아 깨질 소비자를 찾지 못했지만, 외부 클라이언트가 `secret_material_rejected` 로 분기하고 있었다면 이 다섯 경우에서 분기가 달라진다(그게 이 수정의 목적이다). 실제 LLM 백엔드를 붙인 정상 경로(SSE)·PostgreSQL 경로는 이 환경에서 못 돌렸고(store nil 테스트), 거절이 `s.store.AIConfig` 앞에서 반환된다는 사실에만 의존한다.
- 일부러 하지 않은 것: `docs/guides/api-guide.md` 에 error code 표를 만들지 않았다 — 지금 문서에 AI error code 가 한 줄도 없어 "갱신" 이 아니라 새 표 작성이고, 과제서가 "되어 있다면 한 줄 갱신" 이라고 한 조건에 맞지 않아 별도 회차 후보로 남겼다(ideas.json). `maximumAITokens`(21행)와 262144 리터럴(74행)의 중복 통합, `secretMaterialPatterns` 손질, CHANGELOG·버전·`web/` 도 건드리지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 DB 불필요(store nil, 1초 내). 262144 초과 서브테스트는 `strings.Repeat("가 ", 70000)` 로 **공백을 섞어** 마지막 Secret 패턴(`[A-Za-z0-9+/_=-]{48,}`)을 피한다 — 패딩을 ASCII 연속 문자열로 바꾸면 Secret 분기로 빠져 테스트 의도가 사라진다. `bash scripts/verify.sh` 는 이 트리에서 exit 0(`검증 완료: jikim v0.2.27`, 프런트 59/59)이고, 500kB 청크 경고는 기존 상태다. `npm ci` 가 `web/node_modules` 를 만들지만 커밋에는 들어가지 않았다(git status 확인: 변경 2개 파일뿐).
- [러너 16:51] brief accepted — 채택 — 근거와 배선 사실이 모두 코드와 정확히 맞았다(ai.go:94 가 저장소 유일한 자리, `invalid_ai_request` 미사용, 거절이 `s
- [러너 16:52] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 테스트 유효성을 직접 재현했다: main 의 ai.go 로 되돌려 돌리니 5개 서브테스트가 `code=secret_material_rejected, want "invalid_ai_request"` 로 실패하고 Secret 2개는 통과 — 원장의 `- 실패 재현:` 줄과 일치한다. 262144 초과 케이스의 실패 메시지가 크기 분기 문구라 Secret 패턴으로 새지 않음이 증명됐다(정찰이 남긴 미확인 항목 해소). 확인 후 복구해 트리는 깨끗하다.
- 구현자가 의심한 자리를 전부 열어봤다: `grep` 결과 `ValidateAIInput` 호출자는 ai.go:98 하나뿐, docs/·web/src/ 에 AI error code 소비자 없음, 라우트는 server.go:141 `withAuth` 로 테스트 체인과 동일. 다섯 에러 메시지가 전부 상수라 400 본문에 사용자 입력이 반향되지 않는다.
- 보안·법무 차단 사유 없음: 인가 경로·식별자·암호 비교·의존성 변경이 없고, 개인정보 신규 수집·전송·보관도 없다. 보안 매처·한도·문구는 무변경(security_test.go:16 통과).
- 못 본 것: `bash scripts/verify.sh` 와 프런트 `npm ci`·vitest 는 이 세션에서 돌리지 않았다(diff 가 `web/` 무관, 구현자 exit 0 보고 신뢰). 실제 LLM 백엔드 SSE 정상 경로·PostgreSQL 경로도 미검증이나 거절은 그 앞에서 반환된다.
- 승인 후 남는 우려(릴리즈·다음 회차): `invalid_ai_request` 는 문서 없는 새 공개 계약 문자열이다 — 릴리즈 노트에 코드 변경을 명시하고, `docs/guides/api-guide.md` 의 AI error code 표를 다음 회차에서 만들 것. 다섯 형식 오류가 여전히 한 code 를 공유한다는 점도 후속 후보다.
- [러너 16:54] review approved — 리뷰 승인 (risk=low)
- [러너 16:54] pr created — https://github.com/hkjang/jikim/pull/50
